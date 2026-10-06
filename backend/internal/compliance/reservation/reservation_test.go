package reservation

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestReservationManager_ConcurrentRaceSafety(t *testing.T) {
	cache := NewPositionCache()
	tenantID := uuid.New()
	accountID := uuid.New()
	securityID := uuid.New()

	// Initial Portfolio NAV = $10,000,000. Cap = 5% ($500,000 max).
	// Initial settled position = $300,000 (3% weight). Available capacity = $200,000 (2% weight).
	cache.SetAccountNAV(tenantID, accountID, decimal.NewFromInt(10000000))
	cache.SetSettledPosition(tenantID, accountID, securityID, decimal.NewFromInt(3000), decimal.NewFromInt(300000))

	mgr := NewReservationManager(cache, 5*time.Minute)

	// Launch 1,000 concurrent goroutines, each attempting to buy $150,000 (1.5% weight)
	// EXACTLY ONE order must succeed (taking weight to 4.5%); subsequent orders must be REJECTED with ErrCapacityExceeded (projected weight 6.0% > 5.0%)
	numGoroutines := 1000
	var wg sync.WaitGroup
	var successfulReservations int64
	var rejectedReservations int64
	var leases []*ReservationLease
	var leaseMu sync.Mutex

	maxCap := decimal.RequireFromString("0.0500")
	orderPrice := decimal.NewFromInt(100)
	orderQty := decimal.NewFromInt(1500) // $150,000

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lease, err := mgr.AcquireReservation(context.Background(), tenantID, accountID, securityID, uuid.New(), "BUY", orderQty, orderPrice, maxCap)
			if err == nil {
				leaseMu.Lock()
				leases = append(leases, lease)
				successfulReservations++
				leaseMu.Unlock()
			} else {
				leaseMu.Lock()
				rejectedReservations++
				leaseMu.Unlock()
			}
		}()
	}

	wg.Wait()

	if successfulReservations != 1 {
		t.Fatalf("Race Condition Detected! Expected exactly 1 successful reservation, got %d (rejected: %d)", successfulReservations, rejectedReservations)
	}

	if rejectedReservations != int64(numGoroutines-1) {
		t.Errorf("Expected %d rejected reservations, got %d", numGoroutines-1, rejectedReservations)
	}

	// Verify position projected state
	pos := cache.GetPosition(tenantID, accountID, securityID)
	if !pos.TotalProjectedValue().Equal(decimal.NewFromInt(450000)) {
		t.Errorf("Expected projected position value $450,000, got %s", pos.TotalProjectedValue().String())
	}

	// Release the reservation and verify capacity is restored
	err := mgr.ReleaseReservation(leases[0].LeaseID)
	if err != nil {
		t.Fatalf("ReleaseReservation failed: %v", err)
	}

	posAfter := cache.GetPosition(tenantID, accountID, securityID)
	if !posAfter.TotalProjectedValue().Equal(decimal.NewFromInt(300000)) {
		t.Errorf("Expected restored position value $300,000, got %s", posAfter.TotalProjectedValue().String())
	}
}

func TestReservationManager_HydrateFromDatabase(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to open sqlmock database: %v", err)
	}
	defer db.Close()

	tenantID := uuid.New()
	accountID := uuid.New()
	securityID1 := uuid.New()
	securityID2 := uuid.New()
	orderID1 := uuid.New()
	orderID2 := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "tenant_id", "account_id", "security_id", "order_side", "ordered_quantity", "price"}).
		AddRow(orderID1, tenantID, accountID, securityID1, "BUY", 100.0, 50.0).
		AddRow(orderID2, tenantID, accountID, securityID2, "SELL", 200.0, 25.0)

	mock.ExpectQuery(`SELECT id, tenant_id, account_id, security_id, order_side, ordered_quantity, COALESCE\(execution_price, average_price, 0\) AS price FROM oms\.trade_order WHERE order_status IN \('NEW', 'PENDING', 'PARTIALLY_FILLED'\) AND valid_to IS NULL`).
		WillReturnRows(rows)

	cache := NewPositionCache()
	mgr := NewReservationManager(cache, 5*time.Minute)

	hydrated, err := mgr.HydrateFromDatabase(context.Background(), db)
	if err != nil {
		t.Fatalf("HydrateFromDatabase failed: %v", err)
	}

	if hydrated != 2 {
		t.Fatalf("Expected 2 hydrated orders, got %d", hydrated)
	}

	pos1 := cache.GetPosition(tenantID, accountID, securityID1)
	if !pos1.ReservedBuyValue.Equal(decimal.NewFromInt(5000)) {
		t.Errorf("Expected reserved buy value 5000 for sec1, got %s", pos1.ReservedBuyValue.String())
	}

	pos2 := cache.GetPosition(tenantID, accountID, securityID2)
	if !pos2.ReservedSellValue.Equal(decimal.NewFromInt(5000)) {
		t.Errorf("Expected reserved sell value 5000 for sec2, got %s", pos2.ReservedSellValue.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled sqlmock expectations: %s", err)
	}
}

func TestReservationManager_ParkedOrderSweeperImmunity(t *testing.T) {
	cache := NewPositionCache()
	tenantID := uuid.New()
	accountID := uuid.New()
	securityID := uuid.New()

	cache.SetAccountNAV(tenantID, accountID, decimal.NewFromInt(10000000))
	mgr := NewReservationManager(cache, 5*time.Minute) // Standard 5m default TTL

	now := time.Now().UTC()
	// Acquire reservation for order requiring 4-eyes approval
	lease, err := mgr.AcquireReservation(context.Background(), tenantID, accountID, securityID, uuid.New(), "BUY", decimal.NewFromInt(1000), decimal.NewFromInt(100), decimal.Zero)
	if err != nil {
		t.Fatalf("AcquireReservation failed: %v", err)
	}

	// 1. Park order in Temporal for 15 minutes
	approvalTTL := 15 * time.Minute
	err = mgr.ParkReservation(lease.LeaseID, approvalTTL)
	if err != nil {
		t.Fatalf("ParkReservation failed: %v", err)
	}

	// 2. Sweeper runs at minute 6 (past normal 5m default lease TTL)
	swept := mgr.SweepExpiredLeases(now.Add(6*time.Minute), nil)
	if swept != 0 {
		t.Fatalf("Parked reservation was erroneously swept at minute 6! Swept count: %d", swept)
	}

	// Verify reserved capacity is STILL intact
	pos := cache.GetPosition(tenantID, accountID, securityID)
	if !pos.ReservedBuyValue.Equal(decimal.NewFromInt(100000)) {
		t.Errorf("Expected reserved value 100,000 to be preserved, got %s", pos.ReservedBuyValue.String())
	}

	// 3. Sweeper runs at minute 12 (still within 15m approval window)
	swept = mgr.SweepExpiredLeases(now.Add(12*time.Minute), nil)
	if swept != 0 {
		t.Fatalf("Parked reservation was erroneously swept at minute 12! Swept count: %d", swept)
	}

	// 4. Compliance officer approves at minute 12 -> Unpark reservation
	err = mgr.UnparkReservation(lease.LeaseID)
	if err != nil {
		t.Fatalf("UnparkReservation failed: %v", err)
	}

	// 5. Final release upon trade execution fill
	err = mgr.ReleaseReservation(lease.LeaseID)
	if err != nil {
		t.Fatalf("ReleaseReservation failed: %v", err)
	}

	posAfter := cache.GetPosition(tenantID, accountID, securityID)
	if !posAfter.ReservedBuyValue.IsZero() {
		t.Errorf("Expected reserved buy value 0 after final release, got %s", posAfter.ReservedBuyValue.String())
	}
}

func TestReservationManager_LeaseSweeper(t *testing.T) {
	cache := NewPositionCache()
	tenantID := uuid.New()
	accountID := uuid.New()
	securityID := uuid.New()

	cache.SetAccountNAV(tenantID, accountID, decimal.NewFromInt(1000000))
	mgr := NewReservationManager(cache, 1*time.Second) // 1s TTL

	now := time.Now().UTC()
	lease, err := mgr.AcquireReservation(context.Background(), tenantID, accountID, securityID, uuid.New(), "BUY", decimal.NewFromInt(100), decimal.NewFromInt(100), decimal.Zero)
	if err != nil {
		t.Fatalf("AcquireReservation failed: %v", err)
	}

	// Immediate sweep: lease is not expired
	swept := mgr.SweepExpiredLeases(now, nil)
	if swept != 0 {
		t.Errorf("Expected 0 expired leases, got %d", swept)
	}

	// Advance time past expiry
	swept = mgr.SweepExpiredLeases(now.Add(2*time.Second), func(l *ReservationLease) {
		if l.LeaseID != lease.LeaseID {
			t.Errorf("Sweeper passed wrong lease: got %s, want %s", l.LeaseID, lease.LeaseID)
		}
	})

	if swept != 1 {
		t.Fatalf("Expected 1 lease to be swept, got %d", swept)
	}

	// Verify reservation was released in cache
	pos := cache.GetPosition(tenantID, accountID, securityID)
	if !pos.ReservedBuyValue.IsZero() {
		t.Errorf("Expected reserved buy value 0 after sweep, got %s", pos.ReservedBuyValue.String())
	}
}

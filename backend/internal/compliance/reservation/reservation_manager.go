package reservation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrReservationNotFound = errors.New("reservation: lease not found or already released")
	ErrCapacityExceeded    = errors.New("reservation: capacity limit exceeded")
)

// ReservationLease represents a temporary lock on capacity for an in-flight order
type ReservationLease struct {
	LeaseID     uuid.UUID       `json:"leaseId"`
	OrderID     uuid.UUID       `json:"orderId"`
	TenantID    uuid.UUID       `json:"tenantId"`
	AccountID   uuid.UUID       `json:"accountId"`
	SecurityID  uuid.UUID       `json:"securityId"`
	Side        string          `json:"side"` // BUY, SELL, SHORT
	Quantity    decimal.Decimal `json:"quantity"`
	MarketValue decimal.Decimal `json:"marketValue"`
	IsParked    bool            `json:"isParked"`
	CreatedAt   time.Time       `json:"createdAt"`
	ExpiresAt   time.Time       `json:"expiresAt"`
}

// ReservationManager manages in-flight trade capacity reservations
type ReservationManager struct {
	mu           sync.RWMutex
	cache        *PositionCache
	activeLeases map[uuid.UUID]*ReservationLease // LeaseID -> Lease
	defaultTTL   time.Duration
}

// NewReservationManager creates a new reservation manager
func NewReservationManager(cache *PositionCache, defaultTTL time.Duration) *ReservationManager {
	if defaultTTL <= 0 {
		defaultTTL = 5 * time.Minute
	}
	return &ReservationManager{
		cache:        cache,
		activeLeases: make(map[uuid.UUID]*ReservationLease),
		defaultTTL:   defaultTTL,
	}
}

// AcquireReservation atomically reserves order capacity in the position cache
func (m *ReservationManager) AcquireReservation(ctx context.Context, tenantID, accountID, securityID, orderID uuid.UUID, side string, qty, price decimal.Decimal, maxAllowedWeight decimal.Decimal) (*ReservationLease, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	marketValue := qty.Mul(price)
	k := cacheKey(tenantID, accountID, securityID)

	m.cache.mu.Lock()
	pos, exists := m.cache.positions[k]
	if !exists {
		pos = &PositionState{
			TenantID:   tenantID,
			AccountID:  accountID,
			SecurityID: securityID,
		}
		m.cache.positions[k] = pos
	}

	// Check Projected Concentration if limit is provided
	if !maxAllowedWeight.IsZero() {
		nav := m.cache.accountNAV[navKey(tenantID, accountID)]
		if !nav.IsZero() {
			projectedVal := pos.SettledMarketValue.Add(pos.ReservedBuyValue).Add(marketValue)
			projectedWeight := projectedVal.Div(nav)
			if projectedWeight.GreaterThan(maxAllowedWeight) {
				m.cache.mu.Unlock()
				return nil, fmt.Errorf("%w: projected weight %s exceeds max %s", ErrCapacityExceeded, projectedWeight.StringFixed(4), maxAllowedWeight.StringFixed(4))
			}
		}
	}

	// Apply reservation delta
	if side == "BUY" {
		pos.ReservedBuyValue = pos.ReservedBuyValue.Add(marketValue)
		pos.ReservedBuyQuantity = pos.ReservedBuyQuantity.Add(qty)
	} else if side == "SELL" || side == "SHORT" {
		pos.ReservedSellValue = pos.ReservedSellValue.Add(marketValue)
	}
	pos.LastUpdated = time.Now().UTC()
	m.cache.mu.Unlock()

	lease := &ReservationLease{
		LeaseID:     uuid.New(),
		OrderID:     orderID,
		TenantID:    tenantID,
		AccountID:   accountID,
		SecurityID:  securityID,
		Side:        side,
		Quantity:    qty,
		MarketValue: marketValue,
		IsParked:    false,
		CreatedAt:   time.Now().UTC(),
		ExpiresAt:   time.Now().UTC().Add(m.defaultTTL),
	}

	m.activeLeases[lease.LeaseID] = lease
	return lease, nil
}

// ParkReservation marks a lease as parked in Temporal and extends its expiry by approvalTTL + margin
func (m *ReservationManager) ParkReservation(leaseID uuid.UUID, approvalTTL time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	lease, exists := m.activeLeases[leaseID]
	if !exists {
		return ErrReservationNotFound
	}

	if approvalTTL <= 0 {
		approvalTTL = 15 * time.Minute
	}

	lease.IsParked = true
	// Approval TTL + 5 minute safety margin
	lease.ExpiresAt = time.Now().UTC().Add(approvalTTL + 5*time.Minute)
	return nil
}

// UnparkReservation transitions a parked reservation back to active upon human approval
func (m *ReservationManager) UnparkReservation(leaseID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	lease, exists := m.activeLeases[leaseID]
	if !exists {
		return ErrReservationNotFound
	}

	lease.IsParked = false
	lease.ExpiresAt = time.Now().UTC().Add(m.defaultTTL)
	return nil
}

// ReleaseReservation atomically releases the reservation lease
func (m *ReservationManager) ReleaseReservation(leaseID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	lease, exists := m.activeLeases[leaseID]
	if !exists {
		return ErrReservationNotFound
	}
	delete(m.activeLeases, leaseID)

	m.cache.mu.Lock()
	k := cacheKey(lease.TenantID, lease.AccountID, lease.SecurityID)
	if pos, posExists := m.cache.positions[k]; posExists {
		if lease.Side == "BUY" {
			pos.ReservedBuyValue = pos.ReservedBuyValue.Sub(lease.MarketValue)
			if pos.ReservedBuyValue.IsNegative() {
				pos.ReservedBuyValue = decimal.Zero
			}
			pos.ReservedBuyQuantity = pos.ReservedBuyQuantity.Sub(lease.Quantity)
			if pos.ReservedBuyQuantity.IsNegative() {
				pos.ReservedBuyQuantity = decimal.Zero
			}
		} else if lease.Side == "SELL" || lease.Side == "SHORT" {
			pos.ReservedSellValue = pos.ReservedSellValue.Sub(lease.MarketValue)
			if pos.ReservedSellValue.IsNegative() {
				pos.ReservedSellValue = decimal.Zero
			}
		}
		pos.LastUpdated = time.Now().UTC()
	}
	m.cache.mu.Unlock()

	return nil
}

// HydrateFromDatabase reconstructs pending reservations on pod startup / rebalance
func (m *ReservationManager) HydrateFromDatabase(ctx context.Context, db *sql.DB) (int, error) {
	query := `
		SELECT id, tenant_id, account_id, security_id, order_side, ordered_quantity, COALESCE(execution_price, average_price, 0) AS price
		FROM oms.trade_order
		WHERE order_status IN ('NEW', 'PENDING', 'PARTIALLY_FILLED') AND valid_to IS NULL
	`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("query pending orders: %w", err)
	}
	defer rows.Close()

	hydrated := 0
	for rows.Next() {
		var orderID, tenantID, accountID, securityID uuid.UUID
		var side string
		var qty, price float64

		if err := rows.Scan(&orderID, &tenantID, &accountID, &securityID, &side, &qty, &price); err != nil {
			return hydrated, fmt.Errorf("scan pending order: %w", err)
		}

		dQty := decimal.NewFromFloat(qty)
		dPrice := decimal.NewFromFloat(price)
		_, err := m.AcquireReservation(ctx, tenantID, accountID, securityID, orderID, side, dQty, dPrice, decimal.Zero)
		if err == nil {
			hydrated++
		}
	}

	return hydrated, rows.Err()
}

// SweepExpiredLeases checks for expired leases and auto-releases them
func (m *ReservationManager) SweepExpiredLeases(now time.Time, onExpire func(lease *ReservationLease)) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	expiredCount := 0
	var expiredIDs []uuid.UUID

	for id, lease := range m.activeLeases {
		if now.After(lease.ExpiresAt) {
			expiredIDs = append(expiredIDs, id)
			if onExpire != nil {
				onExpire(lease)
			}
		}
	}

	for _, id := range expiredIDs {
		lease := m.activeLeases[id]
		delete(m.activeLeases, id)

		m.cache.mu.Lock()
		k := cacheKey(lease.TenantID, lease.AccountID, lease.SecurityID)
		if pos, posExists := m.cache.positions[k]; posExists {
			if lease.Side == "BUY" {
				pos.ReservedBuyValue = pos.ReservedBuyValue.Sub(lease.MarketValue)
				pos.ReservedBuyQuantity = pos.ReservedBuyQuantity.Sub(lease.Quantity)
			}
		}
		m.cache.mu.Unlock()
		expiredCount++
	}

	return expiredCount
}

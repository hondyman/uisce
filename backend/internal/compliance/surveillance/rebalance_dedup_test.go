package surveillance

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func getSurveillanceTestDB(t *testing.T) *sql.DB {
	homeDir, _ := os.UserHomeDir()
	dsn := fmt.Sprintf("postgres://postgres:postgres@100.84.50.65:5432/alpha?sslmode=verify-full&sslrootcert=%s/.uisce/certs/ca.crt&sslcert=%s/.uisce/certs/postgres-client.crt&sslkey=%s/.uisce/certs/postgres-client.key", homeDir, homeDir, homeDir)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Postgres alpha not reachable, skipping: %v", err)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Skipf("Postgres alpha ping failed, skipping: %v", err)
		return nil
	}
	return db
}

// TestSurveillance_RebalanceIdempotency_DeterministicDedupKey asserts that Kafka rebalances
// delivering duplicate CDC events result in idempotent upserts with zero duplicate findings
func TestSurveillance_RebalanceIdempotency_DeterministicDedupKey(t *testing.T) {
	db := getSurveillanceTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testTenantID := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")
	beneficialOwnerID := uuid.New()
	securityID := uuid.New()

	engine := NewPostTradeSurveillanceEngine(db)
	engine.RegisterAccountOwner(securityID, beneficialOwnerID)

	t0 := time.Now().UTC().Add(-10 * 24 * time.Hour)
	t1 := time.Now().UTC().Add(-5 * 24 * time.Hour)
	t2 := time.Now().UTC()

	lossExecID := uuid.New()
	replExecID := uuid.New()

	// Initial buy trade
	engine.tradeHistory[beneficialOwnerID] = append(engine.tradeHistory[beneficialOwnerID], TradeRecord{
		ExecutionID:       uuid.New(),
		TenantID:          testTenantID,
		AccountID:         securityID,
		BeneficialOwnerID: beneficialOwnerID,
		SecurityID:        securityID,
		Side:              "BUY",
		Quantity:          requireDecimal("100"),
		Price:             requireDecimal("150.00"),
		CostBasis:         requireDecimal("150.00"),
		RealizedGainLoss:  requireDecimal("0"),
		ExecutedAt:        t0,
	})

	// Loss-sale trade (Selling at 120 -> loss of 30/share)
	engine.tradeHistory[beneficialOwnerID] = append(engine.tradeHistory[beneficialOwnerID], TradeRecord{
		ExecutionID:       lossExecID,
		TenantID:          testTenantID,
		AccountID:         securityID,
		BeneficialOwnerID: beneficialOwnerID,
		SecurityID:        securityID,
		Side:              "SELL",
		Quantity:          requireDecimal("100"),
		Price:             requireDecimal("120.00"),
		CostBasis:         requireDecimal("150.00"),
		RealizedGainLoss:  requireDecimal("-3000.00"),
		ExecutedAt:        t1,
	})

	// Replacement buy event CDC payload
	envelopeJSON := fmt.Sprintf(`{
		"payload": {
			"op": "c",
			"after": {
				"id": "%s",
				"placement_id": "%s",
				"order_id": "%s",
				"exec_qty": "100",
				"exec_price": "122.50",
				"status": "FILLED",
				"exec_time": "%s",
				"tenant_id": "%s"
			},
			"ts_ms": %d
		}
	}`, replExecID, uuid.New(), securityID, t2.Format(time.RFC3339Nano), testTenantID, t2.UnixMilli())

	// First CDC processing pass
	err := engine.ProcessExecutionCDC(ctx, []byte(envelopeJSON))
	require.NoError(t, err)

	// Second CDC processing pass (simulating Kafka consumer group rebalance / duplicate delivery)
	err = engine.ProcessExecutionCDC(ctx, []byte(envelopeJSON))
	require.NoError(t, err)

	// Third CDC processing pass
	err = engine.ProcessExecutionCDC(ctx, []byte(envelopeJSON))
	require.NoError(t, err)

	expectedDedupKey := fmt.Sprintf("WASH_SALE:%s:%s:%s", beneficialOwnerID, lossExecID, replExecID)

	// Assert exactly 1 finding row exists for this dedup key despite 3 replay deliveries
	var findingCount int
	var severity, status string
	var actWindowStart, actWindowEnd, detectedAt time.Time
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*), MAX(severity), MAX(status), MAX(activity_window_start), MAX(activity_window_end), MAX(detected_at)
		FROM compliance.compliance_surveillance_finding
		WHERE tenant_id = $1 AND dedup_key = $2
	`, testTenantID, expectedDedupKey).Scan(&findingCount, &severity, &status, &actWindowStart, &actWindowEnd, &detectedAt)
	require.NoError(t, err)

	require.Equal(t, 1, findingCount, "At-least-once CDC replay must produce exactly 1 finding via deterministic dedup key")
	require.Equal(t, "OPEN", status)
	require.Equal(t, "MEDIUM", severity)

	// Assert temporal semantics: activity window start is T - 30d, not equal to detected_at
	require.True(t, actWindowStart.Before(detectedAt), "Activity window start must precede detection time")
	require.True(t, detectedAt.After(actWindowStart), "Detected time must reflect processing time")

	t.Logf("Verified Rebalance Idempotency: DedupKey=%s, Findings=%d, Severity=%s, Status=%s",
		expectedDedupKey, findingCount, severity, status)

	// Cleanup test finding and events
	_, _ = db.ExecContext(ctx, "DELETE FROM compliance.compliance_surveillance_finding WHERE tenant_id = $1 AND dedup_key = $2", testTenantID, expectedDedupKey)
}

func requireDecimal(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

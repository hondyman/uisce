package blotter

import (
	"context"
	"encoding/json"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/compliance"
	"github.com/hondyman/uisce/backend/internal/compliance/canonical"
	"github.com/stretchr/testify/require"
)

type HubStreamEnvelope struct {
	Type string                `json:"type"`
	Data EvaluationEventRecord `json:"data"`
}

// TestBlotter_SoakTest_OrderFlowMemoryAndReconnectRecovery executes continuous pre-trade order flow,
// simulates abrupt WebSocket client disconnects mid-flight, validates gap-fill convergence,
// and monitors memory consumption to ensure zero memory leaks and zero message loss.
func TestBlotter_SoakTest_OrderFlowMemoryAndReconnectRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping soak test in -short mode")
	}

	db := getAlphaTestDB(t)
	if db == nil {
		t.Skip("alpha database not available")
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	loader := compliance.NewMultiTenantRuleLoader(db)
	goldTenant, err := loader.GetGoldCopyTenantID(ctx)
	require.NoError(t, err)

	soakTenantID := uuid.New()
	_, err = db.ExecContext(ctx, `
		INSERT INTO public.tenants (
			id, name, display_name, status, plan, is_active, is_suspended, is_deleted, gold_copy, created_at, updated_at
		) VALUES (
			$1, 'soak-tenant', 'Soak Test Tenant', 'active', 'enterprise', true, false, false, false, now(), now()
		) ON CONFLICT (id) DO NOTHING
	`, soakTenantID)
	require.NoError(t, err)

	// Fetch sample rule
	var ruleID uuid.UUID
	var ruleCode, contentHash, citation string
	var ruleVersion int
	err = db.QueryRowContext(ctx, `
		SELECT r.id, r.rule_code, r.current_version, v.content_hash, v.citation
		FROM compliance.compliance_rule r
		JOIN compliance.compliance_rule_version v ON r.id = v.rule_id AND r.current_version = v.version
		WHERE r.tenant_id = $1
		LIMIT 1
	`, goldTenant).Scan(&ruleID, &ruleCode, &ruleVersion, &contentHash, &citation)
	require.NoError(t, err)

	hub := NewWebSocketHub()
	go hub.Run()
	svc := NewService(db, hub)

	// Track client state
	type ClientStore struct {
		mu              sync.Mutex
		receivedIDs     map[uuid.UUID]bool
		lastEvaluatedAt time.Time
		reconnectCount  int64
		activeClient    *Client
	}

	clientStore := &ClientStore{
		receivedIDs: make(map[uuid.UUID]bool),
	}

	createClient := func() *Client {
		c := &Client{
			hub:      hub,
			send:     make(chan []byte, 1000),
			tenantID: soakTenantID,
		}
		hub.register <- c
		return c
	}

	clientStore.activeClient = createClient()

	var (
		totalGenerated   int64
		totalBroadcasted int64
		stopProducer     atomic.Bool
		wg               sync.WaitGroup
	)

	// Collect initial memory stats
	var mStart, mEnd runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&mStart)

	// Producer Goroutine: Generates continuous pre-trade evaluation flow
	wg.Add(1)
	go func() {
		defer wg.Done()
		actions := []string{"PASSED", "WARNED", "BLOCKED", "APPROVAL_PENDING"}

		for !stopProducer.Load() {
			orderID := uuid.New()
			lineageID := uuid.New()
			now := time.Now().UTC()
			action := actions[rand.Intn(len(actions))]
			passed := action == "PASSED"

			inputParams := map[string]interface{}{"ticker": "AAPL", "qty": 10000}
			metricSnapshots := map[string]interface{}{"exposure": 0.052}

			evalHash, hashErr := canonical.ComputeEvaluationHash(canonical.EvaluationHashInput{
				LineageID:       lineageID,
				TenantID:        soakTenantID,
				RuleID:          ruleID,
				RuleVersion:     ruleVersion,
				RuleContentHash: contentHash,
				ActionTaken:     action,
				Passed:          passed,
				InputParams:     inputParams,
				MetricSnapshots: metricSnapshots,
			})
			if hashErr != nil {
				continue
			}

			inputBytes, _ := json.Marshal(inputParams)
			metricBytes, _ := json.Marshal(metricSnapshots)

			_, insErr := db.ExecContext(ctx, `
				INSERT INTO compliance.compliance_evaluation_event (
					id, lineage_id, tenant_id, order_id, rule_id, rule_version,
					passed, action_taken, latency_micros, rule_content_hash, evaluation_hash,
					input_params, metric_snapshots, evaluated_at, created_at
				) VALUES (
					gen_random_uuid(), $1, $2, $3, $4, $5,
					$6, $7, 250, $8, $9,
					$10::jsonb, $11::jsonb, $12, $12
				)
			`, lineageID, soakTenantID, orderID, ruleID, ruleVersion, passed, action, contentHash, evalHash, string(inputBytes), string(metricBytes), now)

			if insErr == nil {
				atomic.AddInt64(&totalGenerated, 1)

				rec := EvaluationEventRecord{
					LineageID:       lineageID,
					TenantID:        soakTenantID,
					OrderID:         &orderID,
					RuleID:          ruleID,
					RuleCode:        ruleCode,
					RuleVersion:     ruleVersion,
					Passed:          passed,
					ActionTaken:     action,
					LatencyMicros:   250,
					RuleContentHash: contentHash,
					EvaluationHash:  evalHash,
					EvaluatedAt:     now,
				}
				hub.BroadcastEvaluation(soakTenantID, rec)
				atomic.AddInt64(&totalBroadcasted, 1)
			}

			time.Sleep(10 * time.Millisecond) // ~100 events/sec sustained rate
		}
	}()

	// Consumer Goroutine with Simulated Periodic WebSocket Drops & Gap-Fill Recovery
	wg.Add(1)
	go func() {
		defer wg.Done()
		reconnectTicker := time.NewTicker(2 * time.Second)
		defer reconnectTicker.Stop()

		for {
			clientStore.mu.Lock()
			currClient := clientStore.activeClient
			clientStore.mu.Unlock()

			if currClient == nil {
				time.Sleep(50 * time.Millisecond)
				continue
			}

			select {
			case <-ctx.Done():
				return
			case rawMsg, ok := <-currClient.send:
				if ok {
					var env HubStreamEnvelope
					if err := json.Unmarshal(rawMsg, &env); err == nil {
						clientStore.mu.Lock()
						clientStore.receivedIDs[env.Data.LineageID] = true
						if env.Data.EvaluatedAt.After(clientStore.lastEvaluatedAt) {
							clientStore.lastEvaluatedAt = env.Data.EvaluatedAt
						}
						clientStore.mu.Unlock()
					}
				}
			case <-reconnectTicker.C:
				if stopProducer.Load() {
					return
				}
				// 1. Simulate abrupt socket disconnect
				hub.unregister <- currClient
				time.Sleep(100 * time.Millisecond)

				// 2. Re-establish connection
				newClient := createClient()
				clientStore.mu.Lock()
				clientStore.activeClient = newClient
				clientStore.reconnectCount++
				since := clientStore.lastEvaluatedAt
				// Apply 10-second overlap buffer
				overlapTime := since.Add(-10 * time.Second)
				clientStore.mu.Unlock()

				// 3. Trigger Gap-Fill from REST endpoint
				res, err := svc.ListEvaluations(ctx, ListFilter{
					TenantID: soakTenantID,
					From:     &overlapTime,
					Page:     1,
					PageSize: 200,
				})
				if err == nil && res != nil {
					clientStore.mu.Lock()
					for _, item := range res.Data {
						clientStore.receivedIDs[item.LineageID] = true
						if item.EvaluatedAt.After(clientStore.lastEvaluatedAt) {
							clientStore.lastEvaluatedAt = item.EvaluatedAt
						}
					}
					clientStore.mu.Unlock()
				}
			}
		}
	}()

	// Run soak flow for 10 seconds (~1,000 evaluations across 5 drop/reconnect cycles)
	time.Sleep(10 * time.Second)
	stopProducer.Store(true)
	wg.Wait()

	// Final gap-fill to ensure 100% convergence across tail evaluations
	clientStore.mu.Lock()
	tailOverlap := clientStore.lastEvaluatedAt.Add(-10 * time.Second)
	clientStore.mu.Unlock()

	finalFill, err := svc.ListEvaluations(ctx, ListFilter{
		TenantID: soakTenantID,
		From:     &tailOverlap,
		Page:     1,
		PageSize: 200,
	})
	require.NoError(t, err)

	clientStore.mu.Lock()
	for _, item := range finalFill.Data {
		clientStore.receivedIDs[item.LineageID] = true
	}
	receivedCount := len(clientStore.receivedIDs)
	reconnects := clientStore.reconnectCount
	clientStore.mu.Unlock()

	runtime.GC()
	runtime.ReadMemStats(&mEnd)
	heapAllocGrowthMB := float64(mEnd.HeapAlloc-mStart.HeapAlloc) / (1024 * 1024)

	t.Logf("=== Soak Test Results ===")
	t.Logf("Total Generated Events   : %d", atomic.LoadInt64(&totalGenerated))
	t.Logf("Total Received by Client : %d", receivedCount)
	t.Logf("Reconnect Cycles Induced : %d", reconnects)
	t.Logf("Heap Allocation Delta    : %.2f MB", heapAllocGrowthMB)

	// Assert complete convergence with zero message loss
	require.Equal(t, int(atomic.LoadInt64(&totalGenerated)), receivedCount, "Client received count must match total generated events")
	require.Less(t, heapAllocGrowthMB, 25.0, "Heap memory growth must remain flat/bounded under continuous soak flow")

	// Cleanup test tenant
	_, _ = db.ExecContext(ctx, "DELETE FROM compliance.compliance_evaluation_event WHERE tenant_id = $1", soakTenantID)
	_, _ = db.ExecContext(ctx, "DELETE FROM public.tenants WHERE id = $1", soakTenantID)
}

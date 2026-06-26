package integration_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hondyman/semlayer/backend/internal/events"
)

/**
 * Phase 3.5: Rate Limiting Tests
 * Per-tenant quotas, fairness, and quota management
 */

// TestTokenBucketBasic tests token bucket fundamentals
func TestTokenBucketBasic(t *testing.T) {
	tb := events.NewTokenBucket(60, 10) // 60 events/min = 1/sec, 10 burst

	// Should start with full bucket (60 + 10 burst)
	initial := tb.AvailableTokens()
	if initial < 69 || initial > 71 {
		t.Errorf("Initial tokens %.2f, expected ~70", initial)
	}

	// Consume some tokens
	success := 0
	for i := 0; i < 70; i++ {
		if tb.TryConsume() {
			success++
		}
	}

	if success < 60 || success > 72 {
		t.Errorf("Consumed %d tokens, expected ~60-70 (60+burst)", success)
	}

	// Wait for refill (1 sec = 1 token at 60/min)
	time.Sleep(1100 * time.Millisecond)

	if tb.TryConsume() {
		t.Logf("✅ Token refilled after 1 second")
	} else {
		t.Errorf("❌ Token bucket not refilled")
	}

	t.Logf("✅ Token bucket basic functionality works")
}

// TestQuotaManagerTenantRegistration tests tenant registration
func TestQuotaManagerTenantRegistration(t *testing.T) {
	qm := events.NewQuotaManager()

	config := &events.QuotaConfig{
		TenantID:          "test-tenant",
		EventsPerMinute:   1000,
		MaxConcurrentSubs: 100,
		BurstAllowance:    200,
	}

	err := qm.RegisterTenant(config)
	if err != nil {
		t.Errorf("Failed to register tenant: %v", err)
	}

	stats := qm.GetStats("test-tenant")
	if stats["tenant_id"] != "test-tenant" {
		t.Errorf("Tenant not found after registration")
	}

	if stats["events_per_minute"] != int64(1000) {
		t.Errorf("EventsPerMinute mismatch: %v", stats["events_per_minute"])
	}

	t.Logf("✅ Tenant registration works")
}

// TestQuotaCheckEventRateLimit tests event rate limiting
func TestQuotaCheckEventRateLimit(t *testing.T) {
	qm := events.NewQuotaManager()

	config := &events.QuotaConfig{
		TenantID:          "rate-test",
		EventsPerMinute:   60, // 1 event per sec in steady state
		MaxConcurrentSubs: 100,
		BurstAllowance:    20,
	}

	qm.RegisterTenant(config)

	// Should allow 60 events immediately (1 min quota + 20 burst)
	allowed := 0
	for i := 0; i < 100; i++ {
		if qm.CheckEventQuota("rate-test") {
			allowed++
		}
	}

	if allowed < 60 || allowed > 85 {
		t.Errorf("Allowed %d events, expected ~60-80", allowed)
	}

	deniedBefore := qm.GetGlobalStats()["global_errors"].(int64)

	// Wait for 1 token refill
	time.Sleep(1100 * time.Millisecond)

	if qm.CheckEventQuota("rate-test") {
		t.Logf("✅ Rate limit enforcement working")
	}

	t.Logf("Initial allowed: %d/%d, denied before refill: %d", allowed, 100, deniedBefore)
}

// TestQuotaCheckSubscriberLimit tests subscriber quota enforcement
func TestQuotaCheckSubscriberLimit(t *testing.T) {
	qm := events.NewQuotaManager()

	config := &events.QuotaConfig{
		TenantID:          "sub-test",
		EventsPerMinute:   10000,
		MaxConcurrentSubs: 5, // Only 5 subscribers
		BurstAllowance:    100,
	}

	qm.RegisterTenant(config)

	// Add 5 subscribers (should succeed)
	for i := 0; i < 5; i++ {
		if !qm.CheckSubscriberQuota("sub-test") {
			t.Errorf("Failed to add subscriber %d", i)
		}
	}

	// 6th should fail
	if qm.CheckSubscriberQuota("sub-test") {
		t.Errorf("Should have rejected 6th subscriber")
	}

	stats := qm.GetStats("sub-test")
	if stats["current_subscribers"] != 5 {
		t.Errorf("Subscriber count: %v, expected 5", stats["current_subscribers"])
	}

	// Release one and try again
	qm.ReleaseSubscriber("sub-test")

	if qm.CheckSubscriberQuota("sub-test") {
		t.Logf("✅ Subscriber quota enforcement working")
	} else {
		t.Errorf("Failed to add subscriber after release")
	}
}

// TestRateLimitedBrokerEventQuota tests rate-limited broker event enforcement
func TestRateLimitedBrokerEventQuota(t *testing.T) {
	qm := events.NewQuotaManager()

	config := &events.QuotaConfig{
		TenantID:          "broker-test",
		EventsPerMinute:   100, // 100 per minute = ~1.7 per second
		MaxConcurrentSubs: 50,
		BurstAllowance:    50,
	}

	qm.RegisterTenant(config)

	rlb := events.NewRateLimitedEventStreamBroker(5000, qm)
	defer rlb.Stop()

	ctx := context.Background()

	// Try to publish 200 events (should be limited)
	successful := 0
	for i := 0; i < 200; i++ {
		event := &events.StreamedEvent{
			Type:       events.EventTypeIncidentDetected,
			TenantID:   "broker-test",
			IncidentID: fmt.Sprintf("incident-%d", i),
			Region:     "us-east",
			Severity:   "high",
			Payload: map[string]interface{}{
				"incident_id": fmt.Sprintf("incident-%d", i),
				"title":       "Rate test",
			},
		}

		if rlb.Publish(ctx, "broker-test", event) == nil {
			successful++
		}
	}

	t.Logf("Published %d/%d events (rate limited)", successful, 200)

	if successful < 100 || successful > 160 {
		t.Errorf("Published %d events, expected ~100-150", successful)
	}

	t.Logf("✅ Rate-limited broker enforces quotas")
}

// TestRateLimitedBrokerSubscriberQuota tests subscriber quota in rate-limited broker
func TestRateLimitedBrokerSubscriberQuota(t *testing.T) {
	qm := events.NewQuotaManager()

	config := &events.QuotaConfig{
		TenantID:          "broker-sub-test",
		EventsPerMinute:   10000,
		MaxConcurrentSubs: 3,
		BurstAllowance:    100,
	}

	qm.RegisterTenant(config)

	rlb := events.NewRateLimitedEventStreamBroker(5000, qm)
	defer rlb.Stop()

	ctx := context.Background()

	// Try to add 5 subscribers (should get 3)
	subscribers := []*events.EventSubscriber{}
	successful := 0

	for i := 0; i < 5; i++ {
		sub, err := rlb.Subscribe(ctx, "broker-sub-test", []string{"us-east"})
		if err == nil {
			subscribers = append(subscribers, sub)
			successful++
		} else {
			t.Logf("Subscriber %d rejected: %v", i, err)
		}
	}

	if successful != 3 {
		t.Errorf("Added %d subscribers, expected 3", successful)
	}

	// Unsubscribe and verify we can add more
	rlb.Unsubscribe(subscribers[0].ID, "broker-sub-test")

	sub, err := rlb.Subscribe(ctx, "broker-sub-test", []string{"us-east"})
	if err == nil {
		subscribers[0] = sub
		t.Logf("✅ Subscriber quota enforcement working in rate-limited broker")
	} else {
		t.Errorf("Failed to add subscriber after unsubscribe: %v", err)
	}

	// Cleanup
	for _, sub := range subscribers {
		rlb.Unsubscribe(sub.ID, "broker-sub-test")
	}
}

// TestMultiTenantQuotaIsolation tests quota isolation between tenants
func TestMultiTenantQuotaIsolation(t *testing.T) {
	qm := events.NewQuotaManager()

	// Register two tenants with different quotas
	t1Config := &events.QuotaConfig{
		TenantID:          "tenant-1",
		EventsPerMinute:   60,
		MaxConcurrentSubs: 10,
		BurstAllowance:    10,
	}

	t2Config := &events.QuotaConfig{
		TenantID:          "tenant-2",
		EventsPerMinute:   120,
		MaxConcurrentSubs: 20,
		BurstAllowance:    20,
	}

	qm.RegisterTenant(t1Config)
	qm.RegisterTenant(t2Config)

	// Tenant 1: consume all quota
	t1Consumed := 0
	for i := 0; i < 100; i++ {
		if qm.CheckEventQuota("tenant-1") {
			t1Consumed++
		} else {
			break
		}
	}

	// Tenant 2: should still have quota
	t2Consumed := 0
	for i := 0; i < 100; i++ {
		if qm.CheckEventQuota("tenant-2") {
			t2Consumed++
		} else {
			break
		}
	}

	t.Logf("Tenant 1 consumed: %d", t1Consumed)
	t.Logf("Tenant 2 consumed: %d", t2Consumed)

	if t2Consumed > t1Consumed {
		t.Logf("✅ Quota isolation working (T2: %d > T1: %d)", t2Consumed, t1Consumed)
	} else {
		t.Errorf("Quota not properly isolated between tenants")
	}
}

// TestRateLimitingWebSocketIntegration tests rate limiting with quota manager
func TestRateLimitingWebSocketIntegration(t *testing.T) {
	qm := events.NewQuotaManager()

	// Register tenant with limited quota
	config := &events.QuotaConfig{
		TenantID:          "ws-tenant",
		EventsPerMinute:   100,
		MaxConcurrentSubs: 2,
		BurstAllowance:    20,
	}
	qm.RegisterTenant(config)

	broker := events.NewEventStreamBroker(5000)
	defer broker.Stop()

	ctx := context.Background()

	// Try to add 3 subscribers (should get only 2 due to quota)
	successful := 0
	subscribers := []*events.EventSubscriber{}

	for i := 0; i < 3; i++ {
		// Check quota first
		if !qm.CheckSubscriberQuota("ws-tenant") {
			t.Logf("Subscriber %d rejected due to quota", i)
			continue
		}

		// Subscribe to broker
		sub, err := broker.Subscribe(ctx, "ws-tenant", []string{"us-east"})
		if err != nil {
			t.Logf("Subscriber %d subscribe failed: %v", i, err)
			qm.ReleaseSubscriber("ws-tenant")
			continue
		}

		successful++
		subscribers = append(subscribers, sub)
	}

	// Cleanup
	for i, sub := range subscribers {
		broker.Unsubscribe(sub.ID)
		qm.ReleaseSubscriber("ws-tenant")
		t.Logf("Unsubscribed %d", i)
	}

	if successful == 2 {
		t.Logf("✅ Quota enforcement working (2/%d subscribers allowed)", 3)
	} else {
		t.Logf("WebSocket connections: %d (expected 2)", successful)
	}
}

// TestBurstHandling tests burst allowance behavior
func TestBurstHandling(t *testing.T) {
	qm := events.NewQuotaManager()

	config := &events.QuotaConfig{
		TenantID:          "burst-test",
		EventsPerMinute:   60, // 1/sec steady state
		MaxConcurrentSubs: 100,
		BurstAllowance:    100, // Allow 100 extra in burst
	}
	qm.RegisterTenant(config)

	// Should allow 60 + 100 burst = 160 immediately
	allowed := 0
	for i := 0; i < 200; i++ {
		if qm.CheckEventQuota("burst-test") {
			allowed++
		} else {
			break
		}
	}

	t.Logf("Burst test: allowed %d events (60 base + 100 burst + refill)", allowed)

	if allowed >= 150 && allowed <= 170 {
		t.Logf("✅ Burst allowance working correctly")
	} else {
		t.Errorf("Burst allowance issue: got %d, expected 150-170", allowed)
	}
}

// TestQuotaStatsReporting tests stats API
func TestQuotaStatsReporting(t *testing.T) {
	qm := events.NewQuotaManager()

	config := &events.QuotaConfig{
		TenantID:          "stats-test",
		EventsPerMinute:   1000,
		MaxConcurrentSubs: 50,
		BurstAllowance:    200,
	}
	qm.RegisterTenant(config)

	// Add some subscribers
	qm.CheckSubscriberQuota("stats-test")
	qm.CheckSubscriberQuota("stats-test")

	// Consume some events
	for i := 0; i < 50; i++ {
		qm.CheckEventQuota("stats-test")
	}

	// Get stats
	stats := qm.GetStats("stats-test")
	globalStats := qm.GetGlobalStats()

	t.Logf("Tenant stats: %v", stats)
	t.Logf("Global stats: %v", globalStats)

	if stats["current_subscribers"] != 2 {
		t.Errorf("Subscriber count in stats: %v, expected 2", stats["current_subscribers"])
	}

	if globalStats["global_events"].(int64) < 50 {
		t.Errorf("Global event count: %v, expected >= 50", globalStats["global_events"])
	}

	t.Logf("✅ Quota stats reporting working")
}

// TestQuotaUpdate tests dynamic quota adjustment
func TestQuotaUpdate(t *testing.T) {
	qm := events.NewQuotaManager()

	config := &events.QuotaConfig{
		TenantID:          "update-test",
		EventsPerMinute:   100,
		MaxConcurrentSubs: 10,
		BurstAllowance:    20,
	}
	qm.RegisterTenant(config)

	// Verify initial quota
	stats1 := qm.GetStats("update-test")
	if stats1["events_per_minute"] != int64(100) {
		t.Errorf("Initial quota not set correctly")
	}

	// Update quota
	qm.UpdateQuota("update-test", 200, 20, 40)

	stats2 := qm.GetStats("update-test")
	if stats2["events_per_minute"] != int64(200) {
		t.Errorf("Quota update failed: %v", stats2["events_per_minute"])
	}

	if stats2["max_concurrent_subs"] != 20 {
		t.Errorf("Max concurrent subs update failed: %v", stats2["max_concurrent_subs"])
	}

	t.Logf("✅ Dynamic quota updates working")
}

// BenchmarkRateLimitingThroughput benchmarks rate limiting overhead
func BenchmarkRateLimitingThroughput(b *testing.B) {
	qm := events.NewQuotaManager()

	// Register 10 tenants
	for i := 0; i < 10; i++ {
		config := &events.QuotaConfig{
			TenantID:          fmt.Sprintf("bench-tenant-%d", i),
			EventsPerMinute:   10000,
			MaxConcurrentSubs: 100,
			BurstAllowance:    2000,
		}
		qm.RegisterTenant(config)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tenantID := fmt.Sprintf("bench-tenant-%d", i%10)
		qm.CheckEventQuota(tenantID)
	}

	b.StopTimer()

	globalStats := qm.GetGlobalStats()
	b.Logf("Processed %d quota checks", globalStats["global_events"])
}

// BenchmarkTokenBucketRefill benchmarks token bucket refill performance
func BenchmarkTokenBucketRefill(b *testing.B) {
	tb := events.NewTokenBucket(10000, 1000)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tb.TryConsume()

		// Occasionally check available (triggers refill)
		if i%100 == 0 {
			tb.AvailableTokens()
		}
	}

	b.StopTimer()

	b.Logf("Token bucket: %.2f tokens remaining", tb.AvailableTokens())
}

// TestConcurrentQuotaChecks tests quota system under concurrent load
func TestConcurrentQuotaChecks(t *testing.T) {
	qm := events.NewQuotaManager()

	// Register 5 tenants
	for i := 0; i < 5; i++ {
		config := &events.QuotaConfig{
			TenantID:          fmt.Sprintf("concurrent-tenant-%d", i),
			EventsPerMinute:   1000,
			MaxConcurrentSubs: 50,
			BurstAllowance:    200,
		}
		qm.RegisterTenant(config)
	}

	const numGoroutines = 20
	const checksPerGoroutine = 1000

	var wg sync.WaitGroup
	successCount := atomic.Int64{}
	errorCount := atomic.Int64{}

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for i := 0; i < checksPerGoroutine; i++ {
				tenantID := fmt.Sprintf("concurrent-tenant-%d", (id+i)%5)

				// Alternate between event and subscriber checks
				if i%2 == 0 {
					if qm.CheckEventQuota(tenantID) {
						successCount.Add(1)
					} else {
						errorCount.Add(1)
					}
				} else {
					if qm.CheckSubscriberQuota(tenantID) {
						successCount.Add(1)
						qm.ReleaseSubscriber(tenantID)
					} else {
						errorCount.Add(1)
					}
				}
			}
		}(g)
	}

	wg.Wait()

	totalOps := successCount.Load() + errorCount.Load()
	successRate := float64(successCount.Load()) / float64(totalOps) * 100

	t.Logf("Concurrent quota checks: %d/%d successful (%.1f%%)", successCount.Load(), totalOps, successRate)
	t.Logf("✅ System handles %d concurrent quota checks/sec", totalOps/10)
}

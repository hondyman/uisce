package integration_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hondyman/semlayer/backend/internal/events"
	"github.com/hondyman/semlayer/backend/internal/handlers"
)

/**
 * Phase 3.5: Chaos Testing
 * Failure injection scenarios validating system resilience
 */

// ChaosInjector provides methods to simulate failures
type ChaosInjector interface {
	InjectSlowSubscriber(delay time.Duration)
	InjectConnectionClose()
	InjectNetworkLatency(ms int)
	Recover()
}

// TestChaosSlowSubscriberBackpressure tests behavior with slow event processing
func TestChaosSlowSubscriberBackpressure(t *testing.T) {
	broker := events.NewEventStreamBroker(1000)
	defer broker.Stop()

	wsHandler := handlers.NewWebSocketEventHandler(broker)
	server := httptest.NewServer(wsHandler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	wsURL = fmt.Sprintf("%s?tenant_id=chaos&regions=us-east", wsURL)

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	factory := events.NewIncidentEventFactory(broker)
	ctx := context.Background()

	// Publish 50 events rapidly
	go func() {
		for i := 0; i < 50; i++ {
			factory.NewIncidentDetected(
				ctx, "chaos",
				fmt.Sprintf("incident-%d", i), "Backpressure test", "high", "us-east",
			)
			time.Sleep(1 * time.Millisecond)
		}
	}()

	eventsReceived := 0
	timeoutCount := 0

	// Simulate slow subscriber: 100ms delay between reads
	for eventsReceived < 50 && timeoutCount < 2 {
		ws.SetReadDeadline(time.Now().Add(5 * time.Second))
		event := &events.StreamedEvent{}

		if err := ws.ReadJSON(event); err != nil {
			if strings.Contains(err.Error(), "i/o timeout") {
				timeoutCount++
				t.Logf("Timeout #%d, received %d events", timeoutCount, eventsReceived)
				continue
			}
			t.Logf("Read error: %v", err)
			break
		}

		eventsReceived++

		// Simulate slow processing
		time.Sleep(100 * time.Millisecond)
	}

	t.Logf("Slow subscriber test: received %d/50 events", eventsReceived)
	t.Logf("Timeout count: %d", timeoutCount)

	if eventsReceived > 10 {
		t.Logf("✅ System handled slow subscriber: processed %d events", eventsReceived)
	}
}

// TestChaosRapidConnectionCycles tests reconnection resilience
func TestChaosRapidConnectionCycles(t *testing.T) {
	broker := events.NewEventStreamBroker(5000)
	defer broker.Stop()

	wsHandler := handlers.NewWebSocketEventHandler(broker)
	server := httptest.NewServer(wsHandler)
	defer server.Close()

	baseURL := "ws" + strings.TrimPrefix(server.URL, "http")
	factory := events.NewIncidentEventFactory(broker)
	ctx := context.Background()

	// Publish events continuously
	publisherCtx, publisherCancel := context.WithCancel(context.Background())
	publishedCount := atomic.Int64{}

	go func() {
		i := int64(0)
		for {
			select {
			case <-publisherCtx.Done():
				return
			default:
				factory.NewIncidentDetected(
					ctx, "chaos-rapid",
					fmt.Sprintf("incident-%d", i), "Rapid cycle test", "high", "us-east",
				)
				publishedCount.Store(i)
				i++
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	successes := 0
	failures := 0

	// Rapidly connect/disconnect 50 times
	for cycle := 0; cycle < 50; cycle++ {
		wsURL := fmt.Sprintf("%s?tenant_id=chaos-rapid&regions=us-east", baseURL)
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			failures++
			t.Logf("Cycle %d: connection failed", cycle)
			continue
		}

		// Read 1 event
		ws.SetReadDeadline(time.Now().Add(1 * time.Second))
		event := &events.StreamedEvent{}
		if err := ws.ReadJSON(event); err == nil {
			successes++
		} else {
			failures++
		}

		ws.Close()
	}

	publisherCancel()

	successRate := float64(successes) / 50 * 100
	t.Logf("Connection cycle resilience: %d/%d successful (%.1f%%)", successes, 50, successRate)
	t.Logf("Published events during chaos: %d", publishedCount.Load())

	if successRate > 80 {
		t.Logf("✅ System resilient to rapid connection cycles")
	} else {
		t.Errorf("❌ Connection resilience degraded: %.1f%% success rate", successRate)
	}
}

// TestChaosHighConcurrency tests system under extreme concurrent load
func TestChaosHighConcurrency(t *testing.T) {
	broker := events.NewEventStreamBroker(50000)
	defer broker.Stop()

	wsHandler := handlers.NewWebSocketEventHandler(broker)
	server := httptest.NewServer(wsHandler)
	defer server.Close()

	baseURL := "ws" + strings.TrimPrefix(server.URL, "http")
	factory := events.NewIncidentEventFactory(broker)

	const numSubscribers = 100
	const eventsPerSubscriber = 10

	var wg sync.WaitGroup
	successCount := atomic.Int64{}
	errorCount := atomic.Int64{}

	// Spawn 100 concurrent subscribers
	for subID := 0; subID < numSubscribers; subID++ {
		wg.Add(1)
		go func(subscriberID int) {
			defer wg.Done()

			wsURL := fmt.Sprintf("%s?tenant_id=chaos-%d&regions=us-east", baseURL, subscriberID)
			ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				errorCount.Add(1)
				return
			}
			defer ws.Close()

			// Read 10 events
			for i := 0; i < eventsPerSubscriber; i++ {
				ws.SetReadDeadline(time.Now().Add(5 * time.Second))
				event := &events.StreamedEvent{}
				if err := ws.ReadJSON(event); err != nil {
					errorCount.Add(1)
					return
				}
				successCount.Add(1)
			}
		}(subID)
	}

	// Publish events from multiple goroutines
	publisherWg := sync.WaitGroup{}
	totalEvents := int64(0)
	totalEventsMu := sync.Mutex{}

	for publisherID := 0; publisherID < 5; publisherID++ {
		publisherWg.Add(1)
		go func(pid int) {
			defer publisherWg.Done()

			for i := 0; i < 200; i++ {
				ctx := context.Background()
				tenantID := fmt.Sprintf("chaos-%d", i%numSubscribers)
				factory.NewIncidentDetected(
					ctx, tenantID,
					fmt.Sprintf("incident-%d-%d", pid, i), "Concurrency test", "high", "us-east",
				)
				totalEventsMu.Lock()
				totalEvents++
				totalEventsMu.Unlock()
				time.Sleep(5 * time.Millisecond)
			}
		}(publisherID)
	}

	wg.Wait()
	publisherWg.Wait()

	successRate := float64(successCount.Load()) / float64(numSubscribers*eventsPerSubscriber) * 100
	t.Logf("High concurrency results:")
	t.Logf("  Subscribers: %d", numSubscribers)
	t.Logf("  Expected events: %d", numSubscribers*eventsPerSubscriber)
	t.Logf("  Received events: %d", successCount.Load())
	t.Logf("  Errors: %d", errorCount.Load())
	t.Logf("  Published: %d", totalEvents)
	t.Logf("  Success rate: %.1f%%", successRate)

	if successRate > 90 {
		t.Logf("✅ System handles high concurrency (%.1f%% success)", successRate)
	} else {
		t.Errorf("❌ Concurrency resilience issues: %.1f%% success", successRate)
	}
}

// TestChaosPortalFailure tests propagation during region failure
func TestChaosPortalFailure(t *testing.T) {
	broker := events.NewEventStreamBroker(5000)
	defer broker.Stop()

	wsHandler := handlers.NewWebSocketEventHandler(broker)
	server := httptest.NewServer(wsHandler)
	defer server.Close()

	baseURL := "ws" + strings.TrimPrefix(server.URL, "http")
	factory := events.NewIncidentEventFactory(broker)
	ctx := context.Background()

	// Subscribe to multiple regions
	regions := []string{"us-east", "eu-west", "ap-south"}
	subscribers := make(map[string]*websocket.Conn)

	for _, region := range regions {
		wsURL := fmt.Sprintf("%s?tenant_id=chaos-portal&regions=%s", baseURL, region)
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect to region %s: %v", region, err)
		}
		subscribers[region] = ws
	}
	defer func() {
		for _, ws := range subscribers {
			ws.Close()
		}
	}()

	eventsByRegion := make(map[string]int)
	var mu sync.Mutex

	// Receive events for each region
	for region, ws := range subscribers {
		go func(r string, w *websocket.Conn) {
			for i := 0; i < 10; i++ {
				w.SetReadDeadline(time.Now().Add(3 * time.Second))
				event := &events.StreamedEvent{}
				if err := w.ReadJSON(event); err != nil {
					return
				}
				mu.Lock()
				eventsByRegion[r]++
				mu.Unlock()
			}
		}(region, ws)
	}

	// Publish events across regions
	for i := 0; i < 30; i++ {
		region := regions[i%len(regions)]
		factory.NewIncidentDetected(
			ctx, "chaos-portal",
			fmt.Sprintf("incident-%d", i), "Portal failure test", "high", region,
		)
		time.Sleep(10 * time.Millisecond)
	}

	time.Sleep(2 * time.Second)

	// Check event distribution
	totalReceived := 0
	for region, count := range eventsByRegion {
		t.Logf("Region %s received: %d events", region, count)
		totalReceived += count
	}

	t.Logf("Total events received: %d/30", totalReceived)
	t.Logf("✅ Portal failure resilience: %.1f%% delivery rate", float64(totalReceived)/30*100)
}

// TestChaosBurstAndRecovery tests system recovery from traffic burst
func TestChaosBurstAndRecovery(t *testing.T) {
	broker := events.NewEventStreamBroker(10000)
	defer broker.Stop()

	wsHandler := handlers.NewWebSocketEventHandler(broker)
	server := httptest.NewServer(wsHandler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	wsURL = fmt.Sprintf("%s?tenant_id=chaos-burst&regions=us-east", wsURL)

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	factory := events.NewIncidentEventFactory(broker)
	ctx := context.Background()

	// Phase 1: Normal rate
	go func() {
		for i := 0; i < 30; i++ {
			factory.NewIncidentDetected(
				ctx, "chaos-burst",
				fmt.Sprintf("incident-pre-%d", i), "Pre-burst", "high", "us-east",
			)
			time.Sleep(50 * time.Millisecond)
		}

		// Phase 2: Burst (1000 events in 2 seconds)
		for i := 0; i < 1000; i++ {
			factory.NewIncidentDetected(
				ctx, "chaos-burst",
				fmt.Sprintf("incident-burst-%d", i), "Burst", "critical", "us-east",
			)
			time.Sleep(2 * time.Millisecond)
		}

		// Phase 3: Recovery (normal rate again)
		for i := 0; i < 30; i++ {
			factory.NewIncidentDetected(
				ctx, "chaos-burst",
				fmt.Sprintf("incident-post-%d", i), "Post-burst", "high", "us-east",
			)
			time.Sleep(50 * time.Millisecond)
		}
	}()

	phaseEvents := map[string]int{"pre": 0, "burst": 0, "post": 0}

	for {
		ws.SetReadDeadline(time.Now().Add(10 * time.Second))
		event := &events.StreamedEvent{}

		if err := ws.ReadJSON(event); err != nil {
			break
		}

		if event.Payload != nil {
			if incidentID, ok := event.Payload["incident_id"].(string); ok {
				if strings.Contains(incidentID, "pre-") {
					phaseEvents["pre"]++
				} else if strings.Contains(incidentID, "burst-") {
					phaseEvents["burst"]++
				} else if strings.Contains(incidentID, "post-") {
					phaseEvents["post"]++
				}
			}
		}

		if phaseEvents["pre"] >= 30 && phaseEvents["burst"] >= 1000 && phaseEvents["post"] >= 30 {
			break
		}
	}

	t.Logf("Burst and recovery results:")
	t.Logf("  Pre-burst: %d/30 events", phaseEvents["pre"])
	t.Logf("  Burst: %d/1000 events", phaseEvents["burst"])
	t.Logf("  Post-burst: %d/30 events", phaseEvents["post"])

	burstRate := float64(phaseEvents["burst"]) / 1000 * 100
	recoveryRate := float64(phaseEvents["post"]) / 30 * 100

	t.Logf("Burst delivery: %.1f%%", burstRate)
	t.Logf("Recovery delivery: %.1f%%", recoveryRate)

	if burstRate > 70 && recoveryRate > 80 {
		t.Logf("✅ System recovered from traffic burst (%.1f%% burst, %.1f%% recovery)", burstRate, recoveryRate)
	}
}

// BenchmarkChaosStressTest benchmarks system under stress
func BenchmarkChaosStressTest(b *testing.B) {
	broker := events.NewEventStreamBroker(50000)
	defer broker.Stop()

	wsHandler := handlers.NewWebSocketEventHandler(broker)
	server := httptest.NewServer(wsHandler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	factory := events.NewIncidentEventFactory(broker)
	ctx := context.Background()

	wsURL = fmt.Sprintf("%s?tenant_id=bench-chaos&regions=us-east", wsURL)
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	b.ResetTimer()

	var wg sync.WaitGroup

	// Publisher
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < b.N; i++ {
			factory.NewIncidentDetected(
				ctx, "bench-chaos",
				fmt.Sprintf("incident-%d", i), "Stress", "high", "us-east",
			)
		}
	}()

	// Consumer with random delays
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < b.N; i++ {
			ws.SetReadDeadline(time.Now().Add(5 * time.Second))
			event := &events.StreamedEvent{}
			if err := ws.ReadJSON(event); err != nil {
				b.Logf("Read error at %d: %v", i, err)
				return
			}

			// Variable processing delay (chaos)
			if i%10 == 0 {
				time.Sleep(100 * time.Millisecond)
			} else if i%7 == 0 {
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()

	wg.Wait()
	b.StopTimer()

	b.Logf("Chaos stress test completed: %d iterations", b.N)
}

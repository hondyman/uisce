package integration_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hondyman/semlayer/backend/internal/events"
	"github.com/hondyman/semlayer/backend/internal/handlers"
)

// TestE2EIncidentLifecycle tests complete incident flow from detection to resolution
func TestE2EIncidentLifecycle(t *testing.T) {
	broker := events.NewEventStreamBroker(1000)
	defer broker.Stop()

	wsHandler := handlers.NewWebSocketEventHandler(broker)
	server := httptest.NewServer(wsHandler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	wsURL = fmt.Sprintf("%s?tenant_id=test-tenant&regions=us-east", wsURL)

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	ws.SetReadDeadline(time.Now().Add(10 * time.Second))

	factory := events.NewIncidentEventFactory(broker)
	incidentID := "test-incident-001"

	// Step 1: Incident Detected
	go func() {
		time.Sleep(100 * time.Millisecond)
		factory.NewIncidentDetected(
			context.Background(), "test-tenant", incidentID,
			"Database connection timeout", "critical", "us-east",
		)
	}()

	event1 := &events.StreamedEvent{}
	if err := ws.ReadJSON(event1); err != nil {
		t.Fatalf("Failed to receive incident detected: %v", err)
	}

	if event1.Type != events.EventTypeIncidentDetected {
		t.Errorf("Expected incident.detected, got %s", event1.Type)
	}

	// Step 2: RCA Started
	go func() {
		time.Sleep(100 * time.Millisecond)
		factory.RCAStarted(context.Background(), "test-tenant", incidentID, "us-east")
	}()

	event2 := &events.StreamedEvent{}
	if err := ws.ReadJSON(event2); err != nil {
		t.Fatalf("Failed to receive RCA started: %v", err)
	}

	if event2.Type != events.EventTypeRCAStarted {
		t.Errorf("Expected rca.started, got %s", event2.Type)
	}

	// Step 3: RCA Completed
	go func() {
		time.Sleep(100 * time.Millisecond)
		factory.RCACompleted(
			context.Background(), "test-tenant", incidentID, "us-east",
			map[string]interface{}{"root_causes": []string{"pool exhausted"}},
		)
	}()

	event3 := &events.StreamedEvent{}
	if err := ws.ReadJSON(event3); err != nil {
		t.Fatalf("Failed to receive RCA completed: %v", err)
	}

	if event3.Type != events.EventTypeRCAResultsAvailable {
		t.Errorf("Expected rca.results, got %s", event3.Type)
	}

	// Step 4: Action Completed
	go func() {
		time.Sleep(100 * time.Millisecond)
		factory.ActionCompleted(
			context.Background(), "test-tenant", incidentID,
			"scale-action", "scale", "us-east",
			map[string]interface{}{"success": true},
		)
	}()

	event4 := &events.StreamedEvent{}
	if err := ws.ReadJSON(event4); err != nil {
		t.Fatalf("Failed to receive action completed: %v", err)
	}

	if event4.Type != events.EventTypeActionCompleted {
		t.Errorf("Expected action.completed, got %s", event4.Type)
	}

	// Step 5: Incident Resolved
	go func() {
		time.Sleep(100 * time.Millisecond)
		factory.IncidentResolved(
			context.Background(), "test-tenant", incidentID, "us-east",
			map[string]interface{}{"resolution": "action executed successfully"},
		)
	}()

	event5 := &events.StreamedEvent{}
	if err := ws.ReadJSON(event5); err != nil {
		t.Fatalf("Failed to receive incident resolved: %v", err)
	}

	if event5.Type != events.EventTypeIncidentResolved {
		t.Errorf("Expected incident.resolved, got %s", event5.Type)
	}

	t.Logf("✅ Complete incident lifecycle flow validated")
}

// TestE2EMultiRegionPropagation tests cross-region propagation detection
func TestE2EMultiRegionPropagation(t *testing.T) {
	broker := events.NewEventStreamBroker(1000)
	defer broker.Stop()

	wsHandler := handlers.NewWebSocketEventHandler(broker)
	server := httptest.NewServer(wsHandler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	wsURL = fmt.Sprintf("%s?tenant_id=test-tenant", wsURL)

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	ws.SetReadDeadline(time.Now().Add(10 * time.Second))

	factory := events.NewIncidentEventFactory(broker)

	// Incident detected
	incidentID := "test-incident-002"
	go func() {
		time.Sleep(100 * time.Millisecond)
		factory.NewIncidentDetected(
			context.Background(), "test-tenant", incidentID,
			"Service timeout", "critical", "us-east",
		)
	}()

	event1 := &events.StreamedEvent{}
	if err := ws.ReadJSON(event1); err != nil {
		t.Fatalf("Failed to receive incident: %v", err)
	}

	// Propagation detected
	go func() {
		time.Sleep(100 * time.Millisecond)
		factory.PropagationDetected(
			context.Background(), "test-tenant", incidentID,
			"us-east", []string{"eu-west", "ap-south"}, 0.85,
		)
	}()

	event2 := &events.StreamedEvent{}
	if err := ws.ReadJSON(event2); err != nil {
		t.Fatalf("Failed to receive propagation: %v", err)
	}

	if event2.Type != events.EventTypePropagationDetected {
		t.Errorf("Expected propagation.detected, got %s", event2.Type)
	}

	if toRegions, ok := event2.Payload["to_regions"].([]interface{}); !ok || len(toRegions) < 2 {
		t.Error("Expected to_regions in payload")
	}

	t.Logf("✅ Multi-region propagation flow validated")
}

// TestE2ERegionIsolation tests region-scoped event delivery
func TestE2ERegionIsolation(t *testing.T) {
	broker := events.NewEventStreamBroker(1000)
	defer broker.Stop()

	wsHandler := handlers.NewWebSocketEventHandler(broker)
	server := httptest.NewServer(wsHandler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Subscriber 1: us-east only
	ws1, _, err := websocket.DefaultDialer.Dial(wsURL+"?tenant_id=tenant-1&regions=us-east", nil)
	if err != nil {
		t.Fatalf("Failed to connect ws1: %v", err)
	}
	defer ws1.Close()

	// Subscriber 2: eu-west only
	ws2, _, err := websocket.DefaultDialer.Dial(wsURL+"?tenant_id=tenant-1&regions=eu-west", nil)
	if err != nil {
		t.Fatalf("Failed to connect ws2: %v", err)
	}
	defer ws2.Close()

	ws1.SetReadDeadline(time.Now().Add(5 * time.Second))
	ws2.SetReadDeadline(time.Now().Add(5 * time.Second))

	factory := events.NewIncidentEventFactory(broker)

	// Publish us-east incident
	go func() {
		time.Sleep(150 * time.Millisecond)
		factory.NewIncidentDetected(
			context.Background(), "tenant-1", "inc-us",
			"US incident", "critical", "us-east",
		)
	}()

	event1 := &events.StreamedEvent{}
	if err := ws1.ReadJSON(event1); err != nil {
		t.Fatalf("ws1 failed to receive: %v", err)
	}

	if event1.Region != "us-east" {
		t.Errorf("Expected region us-east, got %s", event1.Region)
	}

	// ws2 should not receive us-east event
	ws2.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := ws2.ReadJSON(event1); err == nil && event1.Region == "us-east" {
		t.Error("ws2 should not receive us-east event")
	}

	t.Logf("✅ Region isolation validated")
}

// TestE2ETenantIsolation tests tenant-scoped event delivery
func TestE2ETenantIsolation(t *testing.T) {
	broker := events.NewEventStreamBroker(1000)
	defer broker.Stop()

	wsHandler := handlers.NewWebSocketEventHandler(broker)
	server := httptest.NewServer(wsHandler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Tenant A
	wsA, _, err := websocket.DefaultDialer.Dial(wsURL+"?tenant_id=tenant-a", nil)
	if err != nil {
		t.Fatalf("Failed to connect tenant-a: %v", err)
	}
	defer wsA.Close()

	// Tenant B
	wsB, _, err := websocket.DefaultDialer.Dial(wsURL+"?tenant_id=tenant-b", nil)
	if err != nil {
		t.Fatalf("Failed to connect tenant-b: %v", err)
	}
	defer wsB.Close()

	wsA.SetReadDeadline(time.Now().Add(5 * time.Second))
	wsB.SetReadDeadline(time.Now().Add(5 * time.Second))

	factory := events.NewIncidentEventFactory(broker)

	// Publish for tenant-a
	go func() {
		time.Sleep(100 * time.Millisecond)
		factory.NewIncidentDetected(
			context.Background(), "tenant-a", "inc-a",
			"Tenant A incident", "critical", "us-east",
		)
	}()

	eventA := &events.StreamedEvent{}
	if err := wsA.ReadJSON(eventA); err != nil {
		t.Fatalf("Tenant A failed to receive: %v", err)
	}

	if eventA.TenantID != "tenant-a" {
		t.Errorf("Expected tenant-a, got %s", eventA.TenantID)
	}

	// Tenant B should not receive
	wsB.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := wsB.ReadJSON(eventA); err == nil && eventA.TenantID == "tenant-a" {
		t.Error("Tenant B should not receive tenant-a events")
	}

	t.Logf("✅ Tenant isolation validated")
}

// TestE2EFailoverFlow tests region failover scenario
func TestE2EFailoverFlow(t *testing.T) {
	broker := events.NewEventStreamBroker(1000)
	defer broker.Stop()

	wsHandler := handlers.NewWebSocketEventHandler(broker)
	server := httptest.NewServer(wsHandler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	wsURL = fmt.Sprintf("%s?tenant_id=test-tenant", wsURL)

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	ws.SetReadDeadline(time.Now().Add(10 * time.Second))

	factory := events.NewIncidentEventFactory(broker)

	// Incident detected
	go func() {
		time.Sleep(100 * time.Millisecond)
		factory.NewIncidentDetected(
			context.Background(), "test-tenant", "inc-failover",
			"Primary region failure", "critical", "us-east",
		)
	}()

	event1 := &events.StreamedEvent{}
	if err := ws.ReadJSON(event1); err != nil {
		t.Fatalf("Failed to receive incident: %v", err)
	}

	// Failover triggered
	go func() {
		time.Sleep(100 * time.Millisecond)
		factory.RegionFailover(
			context.Background(), "test-tenant", "us-east", "us-west",
		)
	}()

	event2 := &events.StreamedEvent{}
	if err := ws.ReadJSON(event2); err != nil {
		t.Fatalf("Failed to receive failover: %v", err)
	}

	if event2.Type != events.EventTypeRegionFailover {
		t.Errorf("Expected region.failover, got %s", event2.Type)
	}

	if fromRegion, ok := event2.Payload["from_region"].(string); !ok || fromRegion != "us-east" {
		t.Error("Expected from_region in failover")
	}

	t.Logf("✅ Failover flow validated")
}

// TestE2EHighVolume tests handling high-frequency events
func TestE2EHighVolume(t *testing.T) {
	broker := events.NewEventStreamBroker(5000)
	defer broker.Stop()

	wsHandler := handlers.NewWebSocketEventHandler(broker)
	server := httptest.NewServer(wsHandler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	wsURL = fmt.Sprintf("%s?tenant_id=test-tenant&regions=us-east", wsURL)

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	ws.SetReadDeadline(time.Now().Add(15 * time.Second))

	factory := events.NewIncidentEventFactory(broker)

	// Publish 50 rapid incidents
	numEvents := 50
	eventChan := make(chan struct{}, 10)

	for i := 0; i < numEvents; i++ {
		go func(idx int) {
			eventChan <- struct{}{}
			defer func() { <-eventChan }()

			factory.NewIncidentDetected(
				context.Background(), "test-tenant",
				fmt.Sprintf("incident-%d", idx), "Rapid", "high", "us-east",
			)
		}(i)
	}

	// Collect received events
	receivedCount := 0
	for i := 0; i < numEvents; i++ {
		event := &events.StreamedEvent{}
		if err := ws.ReadJSON(event); err != nil {
			t.Logf("Failed to receive event %d: %v", i, err)
			break
		}
		receivedCount++
	}

	if receivedCount < numEvents-5 {
		t.Errorf("Expected ~%d events, received %d", numEvents, receivedCount)
	}

	t.Logf("✅ High-volume event handling validated: %d events received", receivedCount)
}

// BenchmarkE2EPipelineThroughput benchmarks the complete incident pipeline
func BenchmarkE2EPipelineThroughput(b *testing.B) {
	broker := events.NewEventStreamBroker(10000)
	defer broker.Stop()

	wsHandler := handlers.NewWebSocketEventHandler(broker)
	server := httptest.NewServer(wsHandler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	wsURL = fmt.Sprintf("%s?tenant_id=bench&regions=us-east", wsURL)

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	factory := events.NewIncidentEventFactory(broker)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		factory.NewIncidentDetected(
			ctx, "bench",
			fmt.Sprintf("incident-%d", i), "Benchmark", "high", "us-east",
		)

		event := &events.StreamedEvent{}
		ws.SetReadDeadline(time.Now().Add(2 * time.Second))
		if err := ws.ReadJSON(event); err != nil {
			b.Logf("Read timeout at iteration %d", i)
			return
		}
	}
}

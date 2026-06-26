package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hondyman/semlayer/backend/internal/events"
	"github.com/hondyman/semlayer/backend/internal/handlers"
)

/**
 * Phase 3.4: WebSocket Integration Tests
 * Tests for real-time event streaming from broker to frontend
 */

// TestWebSocketEventStreaming tests basic WebSocket connection and event streaming
func TestWebSocketEventStreaming(t *testing.T) {
	// Create broker
	broker := events.NewEventStreamBroker(100)
	defer broker.Stop()

	// Create WebSocket handler
	wsHandler := handlers.NewWebSocketEventHandler(broker)

	// Create test server
	server := httptest.NewServer(wsHandler)
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	wsURL = fmt.Sprintf("%s?tenant_id=test-tenant", wsURL)

	// Connect
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	// Set read timeout
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Publish test event
	factory := events.NewIncidentEventFactory(broker)
	go func() {
		time.Sleep(100 * time.Millisecond)
		if err := factory.NewIncidentDetected(context.Background(), "test-tenant", "incident-1", "Test Incident", "critical", "us-east"); err != nil {
			t.Logf("Failed to publish event: %v", err)
		}
	}()

	// Receive event
	var received events.StreamedEvent
	if err := ws.ReadJSON(&received); err != nil {
		t.Fatalf("Failed to read event: %v", err)
	}

	// Verify event
	if received.TenantID != "test-tenant" {
		t.Errorf("Expected tenant_id 'test-tenant', got '%s'", received.TenantID)
	}
	if received.Type != events.EventTypeIncidentDetected {
		t.Errorf("Expected event type '%s', got '%s'", events.EventTypeIncidentDetected, received.Type)
	}
	if received.IncidentID != "incident-1" {
		t.Errorf("Expected incident_id 'incident-1', got '%s'", received.IncidentID)
	}
}

// TestWebSocketRegionFiltering tests that subscribers only receive events for their regions
func TestWebSocketRegionFiltering(t *testing.T) {
	broker := events.NewEventStreamBroker(100)
	defer broker.Stop()

	wsHandler := handlers.NewWebSocketEventHandler(broker)
	server := httptest.NewServer(wsHandler)
	defer server.Close()

	// Connect subscriber to specific regions
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	wsURL = fmt.Sprintf("%s?tenant_id=test-tenant&regions=us-east,us-west", wsURL)

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	ws.SetReadDeadline(time.Now().Add(5 * time.Second))

	factory := events.NewIncidentEventFactory(broker)

	// Publish event in subscribed region
	go func() {
		time.Sleep(100 * time.Millisecond)
		if err := factory.NewIncidentDetected(context.Background(), "test-tenant", "incident-1", "Test", "critical", "us-east"); err != nil {
			t.Logf("Failed to publish event: %v", err)
		}
	}()

	// Should receive event
	var received events.StreamedEvent
	if err := ws.ReadJSON(&received); err != nil {
		t.Fatalf("Failed to read event: %v", err)
	}

	if received.Region != "us-east" {
		t.Errorf("Expected region 'us-east', got '%s'", received.Region)
	}

	// Publish event in non-subscribed region
	go func() {
		time.Sleep(100 * time.Millisecond)
		if err := factory.NewIncidentDetected(context.Background(), "test-tenant", "incident-2", "Test", "critical", "eu-west"); err != nil {
			t.Logf("Failed to publish event: %v", err)
		}
	}()

	// Should not receive this event (timeout)
	ws.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := ws.ReadJSON(&received); err == nil {
		if received.Region == "eu-west" {
			t.Errorf("Received event from non-subscribed region: %s", received.Region)
		}
	}
}

// TestWebSocketMultipleTenants tests subscription isolation across tenants
func TestWebSocketMultipleTenants(t *testing.T) {
	broker := events.NewEventStreamBroker(100)
	defer broker.Stop()

	wsHandler := handlers.NewWebSocketEventHandler(broker)
	server := httptest.NewServer(wsHandler)
	defer server.Close()

	// Connect two subscribers from different tenants
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ws1, _, err := websocket.DefaultDialer.Dial(wsURL+"?tenant_id=tenant-1", nil)
	if err != nil {
		t.Fatalf("Failed to connect tenant-1: %v", err)
	}
	defer ws1.Close()

	ws2, _, err := websocket.DefaultDialer.Dial(wsURL+"?tenant_id=tenant-2", nil)
	if err != nil {
		t.Fatalf("Failed to connect tenant-2: %v", err)
	}
	defer ws2.Close()

	ws1.SetReadDeadline(time.Now().Add(5 * time.Second))
	ws2.SetReadDeadline(time.Now().Add(5 * time.Second))

	factory := events.NewIncidentEventFactory(broker)

	// Publish event for tenant-1
	go func() {
		time.Sleep(100 * time.Millisecond)
		if err := factory.NewIncidentDetected(context.Background(), "tenant-1", "incident-1", "Test", "critical", "us-east"); err != nil {
			t.Logf("Failed to publish event: %v", err)
		}
	}()

	// Tenant-1 should receive
	var received events.StreamedEvent
	if err := ws1.ReadJSON(&received); err != nil {
		t.Fatalf("Tenant-1 failed to read event: %v", err)
	}

	if received.TenantID != "tenant-1" {
		t.Errorf("Tenant-1 received wrong tenant event: %s", received.TenantID)
	}

	// Tenant-2 should not receive (timeout)
	ws2.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := ws2.ReadJSON(&received); err == nil {
		if received.TenantID == "tenant-1" {
			t.Errorf("Tenant-2 received tenant-1 event")
		}
	}
}

// TestWebSocketEventBatching tests EventAggregator batching
func TestWebSocketEventBatching(t *testing.T) {
	broker := events.NewEventStreamBroker(100)
	defer broker.Stop()

	// Create aggregator
	aggregator := events.NewEventAggregator(broker, 10, 500*time.Millisecond)

	factory := events.NewIncidentEventFactory(broker)

	// Publish 15 events rapidly
	for i := 0; i < 15; i++ {
		go func(idx int) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := factory.NewIncidentDetected(ctx, "test-tenant", fmt.Sprintf("incident-%d", idx), "Test", "critical", "us-east"); err != nil {
				t.Logf("Failed to publish event %d: %v", idx, err)
			}
		}(i)
	}

	// Should receive first batch of 10
	batches := make(chan []*events.StreamedEvent)
	go func() {
		ch, err := aggregator.Subscribe(context.Background(), "test-tenant", []string{})
		if err != nil {
			t.Fatalf("subscribe failed: %v", err)
		}
		for batch := range ch {
			batches <- batch
			if len(batch) >= 10 {
				close(batches)
				break
			}
		}
	}()

	batch1 := <-batches
	if len(batch1) < 10 {
		t.Errorf("Expected batch size >= 10, got %d", len(batch1))
	}
}

// TestWebSocketDisconnectHandling tests proper cleanup on disconnect
func TestWebSocketDisconnectHandling(t *testing.T) {
	broker := events.NewEventStreamBroker(100)
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

	// Get subscriber count
	initialSubscribers := len(broker.GetSubscribers())

	// Close connection
	ws.Close()

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)

	// Verify subscriber was removed
	finalSubscribers := len(broker.GetSubscribers())
	if finalSubscribers >= initialSubscribers {
		t.Errorf("Subscriber not cleaned up. Initial: %d, Final: %d", initialSubscribers, finalSubscribers)
	}
}

// TestWebSocketHealthCheck tests health check endpoint
func TestWebSocketHealthCheck(t *testing.T) {
	broker := events.NewEventStreamBroker(100)
	defer broker.Stop()

	healthHandler := handlers.NewHealthCheckEventHandler(broker)
	server := httptest.NewServer(healthHandler)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if status, ok := body["status"].(string); !ok || status != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", body["status"])
	}
}

// TestWebSocketBackpressure tests handling of slow subscribers
func TestWebSocketBackpressure(t *testing.T) {
	broker := events.NewEventStreamBroker(100)
	defer broker.Stop()

	factory := events.NewIncidentEventFactory(broker)

	// Subscribe with slow consumer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	subscriber, err := broker.Subscribe(ctx, "test-tenant", []string{})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Publish many events
	go func() {
		for i := 0; i < 50; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			if err := factory.NewIncidentDetected(ctx, "test-tenant", fmt.Sprintf("incident-%d", i), "Test", "critical", "us-east"); err != nil {
				t.Logf("Failed to publish event: %v", err)
			}
			cancel()
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Consume slowly (5ms per event)
	receivedCount := 0
	for {
		select {
		case <-subscriber.EventChan:
			receivedCount++
			time.Sleep(5 * time.Millisecond)
		case <-subscriber.Done:
			goto done
		case <-ctx.Done():
			goto done
		}
	}

done:
	// Should have received some events despite slow consumption
	if receivedCount == 0 {
		t.Error("No events received despite slow consumption")
	}
}

// TestWebSocketEventFactory tests all event factory methods
func TestWebSocketEventFactory(t *testing.T) {
	broker := events.NewEventStreamBroker(100)
	defer broker.Stop()

	factory := events.NewIncidentEventFactory(broker)

	tests := []struct {
		name         string
		publishFunc  func() error
		expectedType events.EventType
	}{
		{
			name: "IncidentDetected",
			publishFunc: func() error {
				return factory.NewIncidentDetected(
					context.Background(), "tenant", "inc-1", "Title", "critical", "us-east",
				)
			},
			expectedType: events.EventTypeIncidentDetected,
		},
		{
			name: "RCAStarted",
			publishFunc: func() error {
				return factory.RCAStarted(context.Background(), "tenant", "inc-1", "us-east")
			},
			expectedType: events.EventTypeRCAStarted,
		},
		{
			name: "ActionStarted",
			publishFunc: func() error {
				return factory.ActionStarted(context.Background(), "tenant", "inc-1", "action-1", "scale", "us-east")
			},
			expectedType: events.EventTypeActionStarted,
		},
		{
			name: "PropagationDetected",
			publishFunc: func() error {
				return factory.PropagationDetected(
					context.Background(), "tenant", "inc-1", "us-east", []string{"eu-west"}, 0.85,
				)
			},
			expectedType: events.EventTypePropagationDetected,
		},
		{
			name: "RegionFailover",
			publishFunc: func() error {
				return factory.RegionFailover(context.Background(), "tenant", "us-east", "us-west")
			},
			expectedType: events.EventTypeRegionFailover,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Subscribe
			subscriber, err := broker.Subscribe(context.Background(), "tenant", []string{})
			if err != nil {
				t.Fatalf("Failed to subscribe: %v", err)
			}
			defer broker.Unsubscribe(subscriber.ID)

			// Publish
			if err := tt.publishFunc(); err != nil {
				t.Fatalf("Failed to publish: %v", err)
			}

			// Receive with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			select {
			case event := <-subscriber.EventChan:
				if event.Type != tt.expectedType {
					t.Errorf("Expected type %s, got %s", tt.expectedType, event.Type)
				}
			case <-ctx.Done():
				t.Errorf("Timeout waiting for event")
			}
		})
	}
}

// TestWebSocketConcurrentEventProcessing tests handling many concurrent events
func TestWebSocketConcurrentEventProcessing(t *testing.T) {
	broker := events.NewEventStreamBroker(100)
	defer broker.Stop()

	// Create multiple subscribers
	numSubscribers := 50
	subscribers := make([]*events.EventSubscriber, numSubscribers)

	for i := 0; i < numSubscribers; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		sub, err := broker.Subscribe(ctx, "test-tenant", []string{})
		if err != nil {
			cancel()
			t.Fatalf("Failed to subscribe: %v", err)
		}
		cancel()
		subscribers[i] = sub
	}

	factory := events.NewIncidentEventFactory(broker)

	// Publish events concurrently
	numEvents := 100
	eventsChan := make(chan *events.StreamedEvent, numEvents)

	for i := 0; i < numSubscribers; i++ {
		go func(sub *events.EventSubscriber) {
			for event := range sub.EventChan {
				eventsChan <- event
			}
		}(subscribers[i])
	}

	// Publish events
	go func() {
		for i := 0; i < numEvents; i++ {
			factory.NewIncidentDetected(
				context.Background(), "test-tenant",
				fmt.Sprintf("incident-%d", i), "Test", "critical", "us-east",
			)
			time.Sleep(1 * time.Millisecond)
		}
	}()

	// Collect events
	receivedEvents := 0
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for {
		select {
		case <-eventsChan:
			receivedEvents++
			if receivedEvents >= numEvents {
				goto done
			}
		case <-ctx.Done():
			goto done
		}
	}

done:
	if receivedEvents == 0 {
		t.Error("No events received in concurrent test")
	}
}

// BenchmarkWebSocketEventThroughput benchmarks event streaming throughput
func BenchmarkWebSocketEventThroughput(b *testing.B) {
	broker := events.NewEventStreamBroker(100)
	defer broker.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	subscriber, _ := broker.Subscribe(ctx, "test-tenant", []string{})
	factory := events.NewIncidentEventFactory(broker)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		factory.NewIncidentDetected(
			context.Background(), "test-tenant",
			fmt.Sprintf("incident-%d", i), "Test", "critical", "us-east",
		)

		select {
		case <-subscriber.EventChan:
		case <-ctx.Done():
			return
		}
	}
}

// BenchmarkWebSocketSubscriberScaling benchmarks scaling with many subscribers
func BenchmarkWebSocketSubscriberScaling(b *testing.B) {
	broker := events.NewEventStreamBroker(100)
	defer broker.Stop()

	numSubscribers := 1000
	subscribers := make([]*events.EventSubscriber, numSubscribers)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := 0; i < numSubscribers; i++ {
		sub, _ := broker.Subscribe(ctx, "test-tenant", []string{})
		subscribers[i] = sub
	}

	factory := events.NewIncidentEventFactory(broker)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		factory.NewIncidentDetected(
			context.Background(), "test-tenant",
			fmt.Sprintf("incident-%d", i), "Test", "critical", "us-east",
		)
	}
}

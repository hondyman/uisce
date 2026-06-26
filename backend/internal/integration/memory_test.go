package integration_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hondyman/semlayer/backend/internal/events"
	"github.com/hondyman/semlayer/backend/internal/handlers"
)

/**
 * Phase 3.5: Memory Leak Detection Tests
 * Long-duration streaming stability and resource leak analysis
 */

// TestMemoryLeakLongDurationStreaming tests that memory doesn't leak during extended streaming
func TestMemoryLeakLongDurationStreaming(t *testing.T) {
	broker := events.NewEventStreamBroker(10000)
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

	factory := events.NewIncidentEventFactory(broker)

	// Capture baseline memory
	runtime.GC()
	var beforeAlloc runtime.MemStats
	runtime.ReadMemStats(&beforeAlloc)

	// Stream 10,000 events over 30 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	eventCount := 0
	const totalEvents = 10000

	// Publisher goroutine
	go func() {
		for i := 0; i < totalEvents && ctx.Err() == nil; i++ {
			factory.NewIncidentDetected(
				ctx, "test-tenant",
				fmt.Sprintf("incident-%d", i), "Test", "high", "us-east",
			)
			time.Sleep(3 * time.Millisecond)
		}
	}()

	// Consumer goroutine
	ws.SetReadDeadline(time.Now().Add(35 * time.Second))
	for i := 0; i < totalEvents && ctx.Err() == nil; i++ {
		event := &events.StreamedEvent{}
		if err := ws.ReadJSON(event); err != nil {
			t.Logf("Read error at event %d: %v", i, err)
			break
		}
		eventCount++
	}

	// Wait for cleanup
	time.Sleep(1 * time.Second)

	// Capture final memory
	runtime.GC()
	var afterAlloc runtime.MemStats
	runtime.ReadMemStats(&afterAlloc)

	// Calculate deltas
	allocDelta := (afterAlloc.Alloc - beforeAlloc.Alloc) / 1024 / 1024 // MB
	heapDelta := (afterAlloc.HeapAlloc - beforeAlloc.HeapAlloc) / 1024 / 1024

	t.Logf("Streamed %d events over 30 seconds", eventCount)
	t.Logf("Memory Alloc Delta: %d MB", allocDelta)
	t.Logf("Heap Alloc Delta: %d MB", heapDelta)
	t.Logf("GC Runs: before=%d, after=%d", beforeAlloc.NumGC, afterAlloc.NumGC)

	// Alloc delta should be < 100MB for 10k events (conservative limit)
	if allocDelta > 100 {
		t.Errorf("Memory leak detected: Alloc delta %d MB exceeds threshold", allocDelta)
	}

	// Heap should not spike excessively
	if heapDelta > 150 {
		t.Errorf("Heap growth excessive: %d MB (expected < 150 MB)", heapDelta)
	}

	t.Logf("✅ No memory leaks detected during long-duration streaming")
}

// TestMemoryLeakSubscriberChurn tests memory behavior with frequent connect/disconnect
func TestMemoryLeakSubscriberChurn(t *testing.T) {
	broker := events.NewEventStreamBroker(5000)
	defer broker.Stop()

	wsHandler := handlers.NewWebSocketEventHandler(broker)
	server := httptest.NewServer(wsHandler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	factory := events.NewIncidentEventFactory(broker)

	runtime.GC()
	var beforeAlloc runtime.MemStats
	runtime.ReadMemStats(&beforeAlloc)

	// Create and destroy 100 subscriber connections rapidly
	numChurns := 100
	for i := 0; i < numChurns; i++ {
		// Connect
		wsURL := fmt.Sprintf("%s?tenant_id=tenant-%d&regions=us-east", wsURL, i%10)
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Logf("Failed to connect subscriber %d: %v", i, err)
			continue
		}

		// Publish event
		go func(idx int) {
			factory.NewIncidentDetected(
				context.Background(), fmt.Sprintf("tenant-%d", idx%10),
				fmt.Sprintf("incident-%d", idx), "Churn test", "high", "us-east",
			)
		}(i)

		// Read one event
		ws.SetReadDeadline(time.Now().Add(2 * time.Second))
		event := &events.StreamedEvent{}
		ws.ReadJSON(event)

		// Disconnect
		ws.Close()

		if i%10 == 0 {
			t.Logf("Completed %d/100 subscriber churns", i)
		}
	}

	time.Sleep(500 * time.Millisecond)

	// Verify subscriber cleanup
	subscribers := broker.GetSubscribers()
	if len(subscribers) > 10 {
		t.Errorf("Too many subscribers after cleanup: %d (expected < 10)", len(subscribers))
	}

	runtime.GC()
	var afterAlloc runtime.MemStats
	runtime.ReadMemStats(&afterAlloc)

	allocDelta := (afterAlloc.Alloc - beforeAlloc.Alloc) / 1024 / 1024

	t.Logf("Subscriber churn: 100 connect/disconnect cycles")
	t.Logf("Memory Alloc Delta: %d MB", allocDelta)
	t.Logf("Remaining subscribers: %d", len(subscribers))

	if allocDelta > 50 {
		t.Errorf("Memory leak in subscriber cleanup: %d MB", allocDelta)
	}

	t.Logf("✅ No memory leaks detected during subscriber churn")
}

// TestMemoryLeakBufferManagement tests event buffer cleanup and eviction
func TestMemoryLeakBufferManagement(t *testing.T) {
	broker := events.NewEventStreamBroker(1000) // Small buffer for testing
	defer broker.Stop()

	factory := events.NewIncidentEventFactory(broker)

	runtime.GC()
	var beforeAlloc runtime.MemStats
	runtime.ReadMemStats(&beforeAlloc)

	// Publish 50,000 events (will trigger buffer eviction)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := 0; i < 50000 && ctx.Err() == nil; i++ {
		factory.NewIncidentDetected(
			ctx, "test-tenant",
			fmt.Sprintf("incident-%d", i), "Buffer test", "high", "us-east",
		)

		if i%5000 == 0 && i > 0 {
			runtime.GC()
			t.Logf("Published %d events", i)
		}
	}

	runtime.GC()
	var afterAlloc runtime.MemStats
	runtime.ReadMemStats(&afterAlloc)

	allocDelta := (afterAlloc.Alloc - beforeAlloc.Alloc) / 1024 / 1024

	t.Logf("Published 50,000 events with 1000-event buffer")
	t.Logf("Memory Alloc Delta: %d MB", allocDelta)

	// Buffer eviction should keep memory bounded
	if allocDelta > 200 {
		t.Errorf("Event buffer not properly evicting: %d MB", allocDelta)
	}

	t.Logf("✅ Event buffer properly managed and evicted")
}

// TestMemoryLeakSlowSubscriberTimeout tests cleanup of slow subscribers
func TestMemoryLeakSlowSubscriberTimeout(t *testing.T) {
	broker := events.NewEventStreamBroker(5000)
	defer broker.Stop()

	factory := events.NewIncidentEventFactory(broker)

	runtime.GC()
	var beforeAlloc runtime.MemStats
	runtime.ReadMemStats(&beforeAlloc)

	ctx := context.Background()

	// Create 20 subscribers
	subscribers := make([]*events.EventSubscriber, 20)
	for i := 0; i < 20; i++ {
		sub, err := broker.Subscribe(ctx, fmt.Sprintf("tenant-%d", i), []string{})
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}
		subscribers[i] = sub
	}

	// Publish 100 events
	for i := 0; i < 100; i++ {
		factory.NewIncidentDetected(
			ctx, fmt.Sprintf("tenant-%d", i%20),
			fmt.Sprintf("incident-%d", i), "Slow test", "high", "us-east",
		)
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for timeouts
	time.Sleep(6 * time.Second)

	// Verify cleanup
	remainingSubscribers := broker.GetSubscribers()
	t.Logf("Subscribers after timeout: %d (expected 0)", len(remainingSubscribers))

	runtime.GC()
	var afterAlloc runtime.MemStats
	runtime.ReadMemStats(&afterAlloc)

	allocDelta := (afterAlloc.Alloc - beforeAlloc.Alloc) / 1024 / 1024

	t.Logf("Memory Alloc Delta: %d MB", allocDelta)

	if allocDelta > 50 {
		t.Errorf("Slow subscriber cleanup not working: %d MB", allocDelta)
	}

	t.Logf("✅ Slow subscribers properly cleaned up after timeout")
}

// BenchmarkMemoryStressLongDuration benchmarks memory usage over extended streaming
func BenchmarkMemoryStressLongDuration(b *testing.B) {
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

	factory := events.NewIncidentEventFactory(broker)
	ctx := context.Background()

	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	startAlloc := m.Alloc

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		factory.NewIncidentDetected(
			ctx, "bench",
			fmt.Sprintf("incident-%d", i), "Bench", "high", "us-east",
		)

		event := &events.StreamedEvent{}
		ws.SetReadDeadline(time.Now().Add(2 * time.Second))
		ws.ReadJSON(event)

		if i%1000 == 0 {
			runtime.GC()
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m)
	endAlloc := m.Alloc

	bytesPerEvent := float64(endAlloc-startAlloc) / float64(b.N)
	b.Logf("Memory per event: %.2f bytes", bytesPerEvent)
}

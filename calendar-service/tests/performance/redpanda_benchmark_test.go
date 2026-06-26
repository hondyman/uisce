package performance

import (
	"calendar-service/internal/redpanda"
	"context"
	"encoding/json"
	"testing"
)

type fastMockListener struct{}

func (f *fastMockListener) OnEventCreated(ctx context.Context, userID, eventID string) {}
func (f *fastMockListener) OnEventUpdated(ctx context.Context, userID, eventID string) {}
func (f *fastMockListener) OnEventDeleted(ctx context.Context, userID, eventID string) {}

// BenchmarkCDCProcessing measures the overhead of parsing and routing a CDC event
func BenchmarkCDCProcessing(b *testing.B) {
	// Mocks for dependencies
	logger := createTestLogger()
	mockListener := &fastMockListener{}

	processor, _ := redpanda.NewCDCProcessor(
		[]string{"localhost:9092"},
		[]string{"cdc_calendar.public.internal_events"},
		nil, // temporal
		nil, // cache
		nil, // hasura
		nil, // availability
		mockListener,
		nil, // metrics
		logger,
	)

	event := redpanda.CDCEvent{
		Op:    "c",
		Table: "internal_events",
		After: json.RawMessage(`{"id": "00000000-0000-0000-0000-000000000001", "user_id": "00000000-0000-0000-0000-000000000002", "tenant_id": "00000000-0000-0000-0000-000000000003"}`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = processor.HandleInternalEventChange(context.Background(), event)
	}
}

// BenchmarkBatchThroughput simulates parallel event processing
func BenchmarkBatchThroughput(b *testing.B) {
	logger := createTestLogger()
	mockListener := &fastMockListener{}

	processor, _ := redpanda.NewCDCProcessor(
		[]string{"localhost:9092"},
		[]string{"cdc_calendar.public.internal_events"},
		nil, nil, nil, nil,
		mockListener,
		nil, // metrics
		logger,
	)

	ctx := context.Background()
	event := redpanda.CDCEvent{
		Op:    "c",
		Table: "internal_events",
		After: json.RawMessage(`{"id": "00000000-0000-0000-0000-000000000001", "user_id": "00000000-0000-0000-0000-000000000002"}`),
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = processor.HandleInternalEventChange(ctx, event)
		}
	})
}

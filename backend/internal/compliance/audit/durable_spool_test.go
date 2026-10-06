package audit

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockReplayBroker struct {
	mu       sync.Mutex
	messages [][]byte
}

func (m *mockReplayBroker) Publish(ctx context.Context, topic string, key string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, payload)
	return nil
}

func (m *mockReplayBroker) Subscribe(ctx context.Context, topic string, groupID string, handler func(key string, payload []byte) error) error {
	return nil
}

func TestDurableSpool_HotPathAndCrashRecovery(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "uisce_spool_test_*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mockBroker := &mockReplayBroker{}
	spool, err := NewDurableSpool(tempDir, mockBroker, "compliance.evaluations")
	if err != nil {
		t.Fatalf("NewDurableSpool failed: %v", err)
	}

	ctx := context.Background()
	eventCount := 50
	lineages := make([]uuid.UUID, eventCount)

	for i := 0; i < eventCount; i++ {
		lineageID := uuid.New()
		lineages[i] = lineageID
		ev := EvaluationEventPayload{
			ID:             uuid.New(),
			LineageID:      lineageID,
			TenantID:       uuid.New(),
			RuleID:         uuid.New(),
			RuleVersion:    1,
			Passed:         true,
			ActionTaken:    "APPROVED",
			LatencyMicros:  50,
			EvaluationHash: "test-hash",
			EvaluatedAt:    time.Now().UTC(),
		}

		if err := spool.WriteHotPath(ctx, ev); err != nil {
			t.Fatalf("WriteHotPath failed at %d: %v", i, err)
		}
	}

	if spool.SpooledCount() != int64(eventCount) {
		t.Errorf("Expected %d spooled events, got %d", eventCount, spool.SpooledCount())
	}

	// Simulate pod crash (close spool without background flush)
	activeFileName := spool.activeFile.Name()
	spool.Close()

	// Verify segment file exists on disk
	if _, err := os.Stat(activeFileName); os.IsNotExist(err) {
		t.Fatalf("WAL segment file was not persisted on disk!")
	}

	// Create new spool on same directory simulating restart
	recoveryBroker := &mockReplayBroker{}
	recoveredSpool, err := NewDurableSpool(tempDir, recoveryBroker, "compliance.evaluations")
	if err != nil {
		t.Fatalf("NewDurableSpool recovery failed: %v", err)
	}
	defer recoveredSpool.Close()

	// Verify all crashed events were replayed to broker
	recoveryBroker.mu.Lock()
	replayedCount := len(recoveryBroker.messages)
	recoveryBroker.mu.Unlock()

	if replayedCount != eventCount {
		t.Fatalf("Crash Recovery failed: expected %d replayed events, got %d", eventCount, replayedCount)
	}

	// Verify old segment file was cleaned up after successful replay
	if _, err := os.Stat(activeFileName); !os.IsNotExist(err) {
		t.Errorf("Replayed segment file was not unlinked after successful recovery replay")
	}
}

func BenchmarkDurableSpool_LocalWALWriteHotPath(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "uisce_spool_bench_*")
	if err != nil {
		b.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	spool, err := NewDurableSpool(tempDir, nil, "")
	if err != nil {
		b.Fatalf("NewDurableSpool failed: %v", err)
	}
	defer spool.Close()

	ev := EvaluationEventPayload{
		ID:             uuid.New(),
		LineageID:      uuid.New(),
		TenantID:       uuid.New(),
		RuleID:         uuid.New(),
		RuleVersion:    1,
		Passed:         true,
		ActionTaken:    "APPROVED",
		LatencyMicros:  50,
		EvaluationHash: "7e9fee6e5078f24064230c69c35b908469f0b4e6d3e74c83e00fbd84a03f65b1",
		EvaluatedAt:    time.Now().UTC(),
	}

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = spool.WriteHotPath(ctx, ev)
	}
}

package audit

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// MockBroker simulates Redpanda / Kafka broker with durable log buffer
type MockBroker struct {
	mu       sync.Mutex
	messages []BrokerMessage
}

type BrokerMessage struct {
	Topic   string
	Key     string
	Payload []byte
}

func (m *MockBroker) Publish(ctx context.Context, topic string, key string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, BrokerMessage{
		Topic:   topic,
		Key:     key,
		Payload: payload,
	})
	return nil
}

func (m *MockBroker) Subscribe(ctx context.Context, topic string, groupID string, handler func(key string, payload []byte) error) error {
	m.mu.Lock()
	copied := make([]BrokerMessage, len(m.messages))
	copy(copied, m.messages)
	m.mu.Unlock()

	for _, msg := range copied {
		if msg.Topic == topic {
			if err := handler(msg.Key, msg.Payload); err != nil {
				return err
			}
		}
	}
	return nil
}

func TestDurableAuditEmitter_PodKillZeroLoss(t *testing.T) {
	broker := &MockBroker{}
	emitter := NewDurableAuditEmitter(broker, "compliance.evaluations")

	tenantID := uuid.New()
	ruleID := uuid.New()

	// 1. Emit 500 audit events through Gateway Pod 1
	var ackedEvents []EvaluationEventPayload
	for i := 0; i < 500; i++ {
		ev := EvaluationEventPayload{
			ID:             uuid.New(),
			LineageID:      uuid.New(),
			TenantID:       tenantID,
			RuleID:         ruleID,
			RuleVersion:    1,
			Passed:         true,
			ActionTaken:    "APPROVED",
			LatencyMicros:  280,
			EvaluationHash: "mock_hash_" + uuid.New().String(),
			EvaluatedAt:    time.Now().UTC(),
		}

		err := emitter.EmitEvaluation(context.Background(), ev)
		if err != nil {
			t.Fatalf("Failed to emit event %d: %v", i, err)
		}
		ackedEvents = append(ackedEvents, ev)

		// 2. Simulate sudden Pod-Kill / crash at event 250
		if i == 250 {
			t.Logf("Simulating gateway pod crash at event %d...", i)
			// Pod is abruptly destroyed; new pod starts and continues producing
			emitter = NewDurableAuditEmitter(broker, "compliance.evaluations")
		}
	}

	// 3. Hot Tier Consumer starts and drains Redpanda topic
	var consumedEvents []EvaluationEventPayload
	err := broker.Subscribe(context.Background(), "compliance.evaluations", "hot-tier-consumer-group", func(key string, payload []byte) error {
		var ev EvaluationEventPayload
		if err := json.Unmarshal(payload, &ev); err != nil {
			return err
		}
		consumedEvents = append(consumedEvents, ev)
		return nil
	})
	if err != nil {
		t.Fatalf("Consumer subscribe failed: %v", err)
	}

	// 4. Assert ZERO Lost Records
	if len(consumedEvents) != len(ackedEvents) {
		t.Fatalf("CRITICAL LOSS: %d events acknowledged, but only %d consumed by Hot Tier!", len(ackedEvents), len(consumedEvents))
	}

	// Verify exact 1-to-1 match by LineageID
	consumedMap := make(map[uuid.UUID]bool)
	for _, ev := range consumedEvents {
		consumedMap[ev.LineageID] = true
	}

	for _, acked := range ackedEvents {
		if !consumedMap[acked.LineageID] {
			t.Errorf("Lost lineage event %s across pod-kill boundary!", acked.LineageID)
		}
	}

	t.Logf("Pod-Kill Zero Loss Verified: 500/500 events safely persisted through crash!")
}

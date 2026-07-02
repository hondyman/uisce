package audit

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// mockAuditPublisher is a hand-rolled AuditPublisher used only by these tests.
// We avoid generating a mock via the test workflow because the package
// contract is small and the mock is read-only.
type mockAuditPublisher struct {
	// recorded events — one slot per Publish method.
	AIQueryGeneratedCalls   []AIQueryGeneratedEvent
	AISemanticResolvedCalls []AISemanticResolvedEvent
	AIColumnMaskedCalls     []AIColumnMaskedEvent
	AIABACEvaluatedCalls    []AIABACEvaluatedEvent
	AIABACDeniedCalls       []AIABACDeniedEvent
	AILineageResolvedCalls  []AILineageResolvedEvent
	CatalogBOMutatedCalls   []CatalogBOMutatedEvent

	// settable per-call error to drive the failure-path tests.
	NextError error
}

func (m *mockAuditPublisher) PublishJobRun(ctx context.Context, e JobRunCompletedEvent) error {
	return m.NextError
}
func (m *mockAuditPublisher) PublishDAGRun(ctx context.Context, e interface{}) error {
	return m.NextError
}
func (m *mockAuditPublisher) PublishChangeSet(ctx context.Context, e ChangeSetCreatedEvent) error {
	return m.NextError
}
func (m *mockAuditPublisher) PublishSemanticSnapshot(ctx context.Context, e SemanticSnapshotEvent) error {
	return m.NextError
}
func (m *mockAuditPublisher) PublishOrchestrationEvent(ctx context.Context, e OrchestrationWorkflowEvent) error {
	return m.NextError
}
func (m *mockAuditPublisher) PublishComplianceViolation(ctx context.Context, e ComplianceViolationEvent) error {
	return m.NextError
}
func (m *mockAuditPublisher) PublishAIQueryAudit(ctx context.Context, e AIQueryExecutionEvent) error {
	return m.NextError
}
func (m *mockAuditPublisher) PublishAIQueryGenerated(ctx context.Context, e AIQueryGeneratedEvent) error {
	m.AIQueryGeneratedCalls = append(m.AIQueryGeneratedCalls, e)
	return m.NextError
}
func (m *mockAuditPublisher) PublishAISemanticResolved(ctx context.Context, e AISemanticResolvedEvent) error {
	m.AISemanticResolvedCalls = append(m.AISemanticResolvedCalls, e)
	return m.NextError
}
func (m *mockAuditPublisher) PublishAIColumnMasked(ctx context.Context, e AIColumnMaskedEvent) error {
	m.AIColumnMaskedCalls = append(m.AIColumnMaskedCalls, e)
	return m.NextError
}
func (m *mockAuditPublisher) PublishAIABACEvaluated(ctx context.Context, e AIABACEvaluatedEvent) error {
	m.AIABACEvaluatedCalls = append(m.AIABACEvaluatedCalls, e)
	return m.NextError
}
func (m *mockAuditPublisher) PublishAIABACDenied(ctx context.Context, e AIABACDeniedEvent) error {
	// Mirror production publisher behaviour: Cardinal Rule 7 stamps the
	// emitted-sync flag at serialisation time, before any return path.
	if m.NextError == nil {
		e.EmittedSync = true
	}
	m.AIABACDeniedCalls = append(m.AIABACDeniedCalls, e)
	return m.NextError
}
func (m *mockAuditPublisher) PublishAILineageResolved(ctx context.Context, e AILineageResolvedEvent) error {
	m.AILineageResolvedCalls = append(m.AILineageResolvedCalls, e)
	return m.NextError
}
func (m *mockAuditPublisher) PublishCatalogBOMutated(ctx context.Context, e CatalogBOMutatedEvent) error {
	m.CatalogBOMutatedCalls = append(m.CatalogBOMutatedCalls, e)
	return m.NextError
}
func (m *mockAuditPublisher) Close() error {
	return nil
}

// withFixedClock replaces the Recorder's now() for deterministic timestamps.
func withFixedClock(r *Recorder, ts time.Time) {
	r.now = func() time.Time { return ts }
}

func TestRecorder_RecordAIQueryGenerated_stamps_default_time(t *testing.T) {
	mock := &mockAuditPublisher{}
	rec := NewRecorder(RecorderConfig{Publisher: mock, Logger: zaptest.NewLogger(t)})
	fixed := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	withFixedClock(rec, fixed)

	evt := AIQueryGeneratedEvent{
		QueryID:     "q-1",
		TenantID:    "tenant-acme",
		UserID:      "u-1",
		InputPrompt: "demo",
	}
	if err := rec.RecordAIQueryGenerated(context.Background(), evt); err != nil {
		t.Fatalf("RecordAIQueryGenerated: %v", err)
	}
	if len(mock.AIQueryGeneratedCalls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.AIQueryGeneratedCalls))
	}
	got := mock.AIQueryGeneratedCalls[0]
	if !got.GeneratedAt.Equal(fixed) {
		t.Errorf("GeneratedAt not stamped: want %v got %v", fixed, got.GeneratedAt)
	}
}

func TestRecorder_RecordAIABACDenied_propagates_failure_for_cardinal_rule_7(t *testing.T) {
	mock := &mockAuditPublisher{NextError: errors.New("redpanda unreachable")}
	rec := NewRecorder(RecorderConfig{Publisher: mock, Logger: zap.NewNop()})

	err := rec.RecordAIABACDenied(context.Background(), AIABACDeniedEvent{
		QueryID:        "q-2",
		TenantID:       "tenant-acme",
		UserID:         "u-1",
		ProfileID:      "profile-restricted",
		DenialReason:   "PII not in profile",
		DeniedResource: "orders.ssn",
	})
	if err == nil {
		t.Fatalf("Cardinal Rule 7 violation: error must propagate from RecordAIABACDenied")
	}
	if !errors.Is(err, mock.NextError) {
		t.Errorf("expected wrapped publisher error, got: %v", err)
	}
	if len(mock.AIABACDeniedCalls) != 1 {
		t.Fatalf("expected denial recorded once, got %d", len(mock.AIABACDeniedCalls))
	}
	if mock.AIABACDeniedCalls[0].EmittedSync {
		t.Errorf("EmittedSync must remain false when publish failed")
	}
}

func TestRecorder_RecordAIABACDenied_sets_EmittedSync_true_on_success(t *testing.T) {
	mock := &mockAuditPublisher{}
	rec := NewRecorder(RecorderConfig{Publisher: mock, Logger: zap.NewNop()})

	if err := rec.RecordAIABACDenied(context.Background(), AIABACDeniedEvent{
		QueryID:        "q-3",
		TenantID:       "tenant-acme",
		UserID:         "u-1",
		DenialReason:   "test",
		DeniedResource: "x.y",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.AIABACDeniedCalls[0].EmittedSync {
		t.Errorf("EmittedSync must be true after successful publish (Cardinal Rule 7)")
	}
}

func TestRecorder_NilPublisher_returns_Sentinel(t *testing.T) {
	rec := NewNopRecorder()
	err := rec.RecordAIABACDenied(context.Background(), AIABACDeniedEvent{
		QueryID: "q-nil", TenantID: "t", UserID: "u", DeniedResource: "x", DenialReason: "y",
	})
	if !errors.Is(err, ErrAuditPublisherUnavailable) {
		t.Errorf("expected ErrAuditPublisherUnavailable, got: %v", err)
	}
}

func TestRecorder_NilPublisher_other_methods_short_circuit(t *testing.T) {
	rec := NewNopRecorder()
	methods := []func() error{
		func() error { return rec.RecordAIQueryGenerated(context.Background(), AIQueryGeneratedEvent{}) },
		func() error { return rec.RecordAISemanticResolved(context.Background(), AISemanticResolvedEvent{}) },
		func() error { return rec.RecordAIColumnMasked(context.Background(), AIColumnMaskedEvent{}) },
		func() error { return rec.RecordAIABACEvaluated(context.Background(), AIABACEvaluatedEvent{}) },
		func() error { return rec.RecordAILineageResolved(context.Background(), AILineageResolvedEvent{}) },
		func() error { return rec.RecordCatalogBOMutated(context.Background(), CatalogBOMutatedEvent{}) },
	}
	for i, m := range methods {
		if err := m(); err != nil {
			t.Errorf("method #%d expected nil short-circuit, got: %v", i, err)
		}
	}
}

func TestRecorder_failed_publish_returns_error(t *testing.T) {
	mock := &mockAuditPublisher{NextError: errors.New("broker down")}
	rec := NewRecorder(RecorderConfig{Publisher: mock, Logger: zap.NewNop()})

	evt := AIQueryGeneratedEvent{QueryID: "x", TenantID: "t"}
	if err := rec.RecordAIQueryGenerated(context.Background(), evt); err == nil {
		t.Errorf("expected error to propagate, got nil")
	}
}

func TestRecorder_CatalogBOMutated_passes_invalidation_keys_through(t *testing.T) {
	mock := &mockAuditPublisher{}
	rec := NewRecorder(RecorderConfig{Publisher: mock, Logger: zap.NewNop()})

	keys := []string{"md:v1:t:acme:bo:by-name:ds1:orders"}
	if err := rec.RecordCatalogBOMutated(context.Background(), CatalogBOMutatedEvent{
		MutationID:          "m-1",
		TenantID:            "t",
		BusinessObjectID:    "bo-orders",
		MutationType:        "update",
		InvalidateCacheKeys: keys,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := mock.CatalogBOMutatedCalls[0]
	if len(got.InvalidateCacheKeys) != 1 || got.InvalidateCacheKeys[0] != keys[0] {
		t.Errorf("invalidation keys not preserved on publisher: got %+v", got.InvalidateCacheKeys)
	}
}

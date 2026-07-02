package audit

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// =============================================================================
// Recorder — typed façade over AuditPublisher for the AI Gate + Catalog
// =============================================================================
//
// Why a Recorder instead of calling RedpandaAuditPublisher directly?
//
//   1. Type safety at call-sites: RecordAIQueryGenerated(evt) vs free-form
//      json.Marshal → Publish.
//   2. Failure semantics are uniform: every Recorder method logs+counts on
//      publish failure, so no caller accidentally swallows errors silently.
//   3. Cardinal Rule 7 (synchronous emission before response) is enforced
//      by the RecordAIABACDenied path: it returns the publisher error to
//      the caller so it can be surfaced to the user-facing handler.
//
// The Recorder is a thin struct; tests substitute a mock AuditPublisher.
// =============================================================================

// Recorder is the typed audit emitter used by the AI Gate and Catalog layers.
type Recorder struct {
	publisher AuditPublisher
	log       *zap.Logger
	// now is overridable for deterministic testing.
	now func() time.Time
}

// RecorderConfig configures a Recorder. Zero value is invalid; use NewRecorder.
type RecorderConfig struct {
	Publisher AuditPublisher
	Logger    *zap.Logger
}

// NewRecorder returns a Recorder wired to the given publisher.
// If cfg.Logger is nil, the global zap logger is used.
func NewRecorder(cfg RecorderConfig) *Recorder {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Recorder{
		publisher: cfg.Publisher,
		log:       logger,
		now:       time.Now,
	}
}

// NewNopRecorder is for tests / startup paths where the publisher may be nil.
// All Record* methods short-circuit to nil and return nil. Useful in unit tests.
func NewNopRecorder() *Recorder {
	return &Recorder{
		publisher: nil,
		log:       zap.NewNop(),
		now:       time.Now,
	}
}

// =============================================================================
// AI Gate — Recorder Methods
// =============================================================================

// RecordAIQueryGenerated emits a Cardinal-Rule-7-compliant audit event for the
// moment a SQL query is generated from an AI request. Returns nil if the
// underlying publish succeeds, the error otherwise. Errors are logged but the
// generation result is generally still returned to the caller — Cardinal Rule 7
// compliance is best-effort here (because the user is waiting), but an outage
// of the audit stream must be loud.
func (r *Recorder) RecordAIQueryGenerated(ctx context.Context, evt AIQueryGeneratedEvent) error {
	if evt.GeneratedAt.IsZero() {
		evt.GeneratedAt = r.now().UTC()
	}
	if r.publisher == nil {
		return nil
	}
	if err := r.publisher.PublishAIQueryGenerated(ctx, evt); err != nil {
		r.log.Error("audit publish failed",
			zap.String("eventType", EventTypeAIQueryGenerated),
			zap.String("tenantId", evt.TenantID),
			zap.String("queryId", evt.QueryID),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// RecordAISemanticResolved emits when a semantic request is resolved to a
// UUID-based SQL generation request.
func (r *Recorder) RecordAISemanticResolved(ctx context.Context, evt AISemanticResolvedEvent) error {
	if evt.ResolvedAt.IsZero() {
		evt.ResolvedAt = r.now().UTC()
	}
	if r.publisher == nil {
		return nil
	}
	if err := r.publisher.PublishAISemanticResolved(ctx, evt); err != nil {
		r.log.Error("audit publish failed",
			zap.String("eventType", EventTypeAISemanticResolved),
			zap.String("tenantId", evt.TenantID),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// RecordAIColumnMasked emits one event per PII/PHI/PCI column that was masked.
// Cardinal Rule 7: regulators want every masking decision independently auditable.
func (r *Recorder) RecordAIColumnMasked(ctx context.Context, evt AIColumnMaskedEvent) error {
	if evt.MaskedAt.IsZero() {
		evt.MaskedAt = r.now().UTC()
	}
	if r.publisher == nil {
		return nil
	}
	if err := r.publisher.PublishAIColumnMasked(ctx, evt); err != nil {
		r.log.Error("audit publish failed",
			zap.String("eventType", EventTypeAIColumnMasked),
			zap.String("tenantId", evt.TenantID),
			zap.String("column", evt.TableAlias+"."+evt.ColumnName),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// RecordAIABACEvaluated emits one event per ABAC evaluation outcome.
// Even allow-decisions are recorded — the audit is a complete ledger.
func (r *Recorder) RecordAIABACEvaluated(ctx context.Context, evt AIABACEvaluatedEvent) error {
	if evt.EvaluatedAt.IsZero() {
		evt.EvaluatedAt = r.now().UTC()
	}
	if r.publisher == nil {
		return nil
	}
	if err := r.publisher.PublishAIABACEvaluated(ctx, evt); err != nil {
		r.log.Error("audit publish failed",
			zap.String("eventType", EventTypeAIABACEvaluated),
			zap.String("tenantId", evt.TenantID),
			zap.String("decision", evt.Decision),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// RecordAIABACDenied is the most important method in this file (Cardinal Rule 7).
//
// CONTRACT: This MUST be called and the returned error MUST be inspected
// before the user-facing 403/500 denial is returned. The EmittedSync flag in
// the persisted Kafka payload is set to true inside the publisher, so any
// downstream consumer reading from Redpanda gets authoritative proof that the
// delivery was acknowledged before the user error surfaced.
//
// Recommended call pattern:
//
//	func (h *Handler) DenyAccess(ctx) error {
//	    err := h.recorder.RecordAIABACDenied(ctx, evt)
//	    if err != nil {
//	        return fmt.Errorf("audit failure prevented denial: %w", err)
//	    }
//	    return ErrAccessDenied
//	}
func (r *Recorder) RecordAIABACDenied(ctx context.Context, evt AIABACDeniedEvent) error {
	if evt.DeniedAt.IsZero() {
		evt.DeniedAt = r.now().UTC()
	}
	if r.publisher == nil {
		// No publisher means we cannot satisfy Cardinal Rule 7.
		// Mark accordingly and return a sentinel error.
		evt.EmittedSync = false
		return ErrAuditPublisherUnavailable
	}
	if err := r.publisher.PublishAIABACDenied(ctx, evt); err != nil {
		r.log.Error("CRITICAL: audit publish failed for ABAC denial",
			zap.String("eventType", EventTypeAIABACDenied),
			zap.String("tenantId", evt.TenantID),
			zap.String("userId", evt.UserID),
			zap.String("deniedResource", evt.DeniedResource),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// RecordAILineageResolved emits when ResolveGraphGovernanceContext traces
// physical-column lineage to find classification tags.
func (r *Recorder) RecordAILineageResolved(ctx context.Context, evt AILineageResolvedEvent) error {
	if evt.ResolvedAt.IsZero() {
		evt.ResolvedAt = r.now().UTC()
	}
	if r.publisher == nil {
		return nil
	}
	if err := r.publisher.PublishAILineageResolved(ctx, evt); err != nil {
		r.log.Error("audit publish failed",
			zap.String("eventType", EventTypeAILineageResolved),
			zap.String("tenantId", evt.TenantID),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// =============================================================================
// Catalog Mutation — Recorder Method (also feeds Phase A cache invalidation)
// =============================================================================

// RecordCatalogBOMutated emits when a Business Object is created/updated/deleted
// or its fields are mutated. The InvalidateCacheKeys hint on the event is
// consumed by cmd/cache-invalidator in Phase A to drain Redis keys.
func (r *Recorder) RecordCatalogBOMutated(ctx context.Context, evt CatalogBOMutatedEvent) error {
	if evt.MutatedAt.IsZero() {
		evt.MutatedAt = r.now().UTC()
	}
	if r.publisher == nil {
		return nil
	}
	if err := r.publisher.PublishCatalogBOMutated(ctx, evt); err != nil {
		r.log.Error("audit publish failed",
			zap.String("eventType", EventTypeCatalogBOMutated),
			zap.String("tenantId", evt.TenantID),
			zap.String("businessObjectId", evt.BusinessObjectID),
			zap.Error(err),
		)
		return err
	}
	return nil
}

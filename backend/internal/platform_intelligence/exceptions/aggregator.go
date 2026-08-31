package exceptions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/audit"
	"github.com/jmoiron/sqlx"
)

type ExceptionType string

const (
	ExceptionSLOBreach           ExceptionType = "slo_breach"
	ExceptionSemanticDrift       ExceptionType = "semantic_drift"
	ExceptionSecurityAnomaly     ExceptionType = "security_anomaly"
	ExceptionDataQuality         ExceptionType = "data_quality"
	ExceptionResidencyViolation  ExceptionType = "residency_violation"
	ExceptionAccessibility       ExceptionType = "accessibility_violation"
	ExceptionAPIInconsistency    ExceptionType = "api_inconsistency"
	ExceptionPreAggInconsistency ExceptionType = "preagg_inconsistency"
	ExceptionTenantAnomaly       ExceptionType = "tenant_anomaly"
	ExceptionPIIExposure         ExceptionType = "pii_exposure"
)

// ExceptionStatus tracks the lifecycle of a platform exception.
type ExceptionStatus string

const (
	StatusOpen           ExceptionStatus = "open"
	StatusAcknowledged   ExceptionStatus = "acknowledged"
	StatusAutoFixPending ExceptionStatus = "auto_fix_pending"
	StatusAutoFixed      ExceptionStatus = "auto_fixed"
	StatusResolved       ExceptionStatus = "resolved"
	StatusClosed         ExceptionStatus = "closed"
	StatusIgnored        ExceptionStatus = "ignored"
)

// AutofixAttempt records a single remediation attempt made against an exception.
type AutofixAttempt struct {
	AttemptedAt time.Time `json:"attempted_at"`
	Action      string    `json:"action"`
	Success     bool      `json:"success"`
	Verified    bool      `json:"verified"`
	Detail      string    `json:"detail,omitempty"`
}

// Exception is a single platform exception record, persisted in
// platform_exceptions and deduplicated by Fingerprint.
type Exception struct {
	ID              uuid.UUID        `json:"id" db:"id"`
	TenantID        uuid.UUID        `json:"tenant_id" db:"tenant_id"`
	Type            ExceptionType    `json:"type" db:"type"`
	Severity        string           `json:"severity" db:"severity"` // critical, high, medium, low
	Source          string           `json:"source" db:"source"`     // page_id, api_id, tenant_id, etc.
	Description     string           `json:"description" db:"description"`
	Evidence        []string         `json:"evidence" db:"-"`
	EvidenceRaw     json.RawMessage  `json:"-" db:"evidence"`
	Fingerprint     string           `json:"fingerprint" db:"fingerprint"`
	OccurrenceCount int              `json:"occurrence_count" db:"occurrence_count"`
	FirstSeen       time.Time        `json:"first_seen" db:"first_seen"`
	LastSeen        time.Time        `json:"last_seen" db:"last_seen"`
	Status          ExceptionStatus  `json:"status" db:"status"`
	ResolvedAt      *time.Time       `json:"resolved_at,omitempty" db:"resolved_at"`
	ResolvedBy      *string          `json:"resolved_by,omitempty" db:"resolved_by"`
	ClosedByAI      bool             `json:"closed_by_ai" db:"closed_by_ai"`
	AutofixAttempts []AutofixAttempt `json:"autofix_attempts" db:"-"`
	AutofixRaw      json.RawMessage  `json:"-" db:"autofix_attempts"`

	// Deprecated fields kept for API/back-compat with the old mock shape.
	DetectedAt time.Time `json:"detected_at" db:"-"`
	Resolved   bool      `json:"resolved" db:"-"`
}

// afterScan populates the JSON-backed convenience fields after a sqlx Get/Select.
func (e *Exception) afterScan() {
	if len(e.EvidenceRaw) > 0 {
		_ = json.Unmarshal(e.EvidenceRaw, &e.Evidence)
	}
	if len(e.AutofixRaw) > 0 {
		_ = json.Unmarshal(e.AutofixRaw, &e.AutofixAttempts)
	}
	e.DetectedAt = e.FirstSeen
	e.Resolved = e.Status == StatusResolved || e.Status == StatusClosed || e.Status == StatusAutoFixed
}

type ExceptionSummary struct {
	TotalExceptions    int                   `json:"total_exceptions"`
	CriticalCount      int                   `json:"critical_count"`
	HighCount          int                   `json:"high_count"`
	MediumCount        int                   `json:"medium_count"`
	LowCount           int                   `json:"low_count"`
	ByType             map[ExceptionType]int `json:"by_type"`
	RecentExceptions   []Exception           `json:"recent_exceptions"`
	TopAffectedTenants []string              `json:"top_affected_tenants"`
	TopAffectedPages   []string              `json:"top_affected_pages"`
	TopAffectedAPIs    []string              `json:"top_affected_apis"`
}

// AutofixPolicy is a per-tenant, per-exception-type (optionally per-user)
// opt-in for automated remediation. There is deliberately no global switch.
type AutofixPolicy struct {
	ID               uuid.UUID     `json:"id" db:"id"`
	TenantID         uuid.UUID     `json:"tenant_id" db:"tenant_id"`
	UserID           *uuid.UUID    `json:"user_id,omitempty" db:"user_id"`
	ExceptionType    ExceptionType `json:"exception_type" db:"exception_type"`
	Enabled          bool          `json:"enabled" db:"enabled"`
	RequiresApproval bool          `json:"requires_approval" db:"requires_approval"`
	UpdatedBy        string        `json:"updated_by" db:"updated_by"`
	UpdatedAt        time.Time     `json:"updated_at" db:"updated_at"`
}

type ExceptionAggregator struct {
	db       *sqlx.DB
	auditPub audit.AuditPublisher
}

// NewExceptionAggregator constructs the DB-backed exception hub. db may be
// nil in tests/dev environments that don't have Postgres wired yet; all
// methods degrade to empty results rather than panicking.
func NewExceptionAggregator(db *sqlx.DB) *ExceptionAggregator {
	return &ExceptionAggregator{db: db}
}

// WithAuditPublisher wires lifecycle events (created/autofix_attempted/
// auto_fixed/escalated/closed) into the Kafka topic IcebergSinkConsumer
// batches into the tier-1 per-tenant audit store. Optional — nil publisher
// means lifecycle events are simply not archived (e.g. in tests).
func (ea *ExceptionAggregator) WithAuditPublisher(pub audit.AuditPublisher) *ExceptionAggregator {
	ea.auditPub = pub
	return ea
}

func (ea *ExceptionAggregator) emitLifecycle(ctx context.Context, e Exception, stage, detail string) {
	if ea.auditPub == nil {
		return
	}
	_ = ea.auditPub.PublishExceptionLifecycle(ctx, audit.ExceptionLifecycleEvent{
		ExceptionID: e.ID.String(),
		TenantID:    e.TenantID.String(),
		Type:        string(e.Type),
		Stage:       stage,
		Status:      string(e.Status),
		Severity:    e.Severity,
		Source:      e.Source,
		Detail:      detail,
		OccurredAt:  time.Now(),
	})
}

// ComputeFingerprint derives the stable dedup key for an exception:
// tenant_id + type + source + a normalized, sorted rendering of the
// evidence key fields. Same underlying problem detected twice (e.g. by a
// re-run of a check) must produce the same fingerprint so Publish upserts
// instead of inserting a duplicate row.
func ComputeFingerprint(tenantID uuid.UUID, excType ExceptionType, source string, evidence []string) string {
	normalized := make([]string, len(evidence))
	copy(normalized, evidence)
	for i, s := range normalized {
		normalized[i] = strings.ToLower(strings.TrimSpace(s))
	}
	// Sort for order-independence so evidence assembled in a different
	// order by the same detector still fingerprints identically.
	for i := 0; i < len(normalized); i++ {
		for j := i + 1; j < len(normalized); j++ {
			if normalized[j] < normalized[i] {
				normalized[i], normalized[j] = normalized[j], normalized[i]
			}
		}
	}
	h := sha256.New()
	h.Write([]byte(tenantID.String()))
	h.Write([]byte("|"))
	h.Write([]byte(excType))
	h.Write([]byte("|"))
	h.Write([]byte(source))
	h.Write([]byte("|"))
	h.Write([]byte(strings.Join(normalized, "\x1f")))
	return hex.EncodeToString(h.Sum(nil))
}

// Publish is the single ingestion entrypoint every detector calls once it
// has produced a signal. It upserts by fingerprint: if an open (or
// acknowledged / auto_fix_pending) row with the same fingerprint already
// exists, its occurrence_count/last_seen are bumped instead of inserting a
// new row — this is the dedup mechanism that stops a repeat detector run
// from creating a duplicate exception.
func (ea *ExceptionAggregator) Publish(ctx context.Context, exc Exception) (*Exception, error) {
	if ea.db == nil {
		return &exc, nil
	}
	if exc.ID == uuid.Nil {
		exc.ID = uuid.New()
	}
	if exc.TenantID == uuid.Nil {
		return nil, fmt.Errorf("exceptions: Publish requires a tenant_id")
	}
	if exc.Fingerprint == "" {
		exc.Fingerprint = ComputeFingerprint(exc.TenantID, exc.Type, exc.Source, exc.Evidence)
	}
	if exc.Status == "" {
		exc.Status = StatusOpen
	}
	evidenceJSON, err := json.Marshal(exc.Evidence)
	if err != nil {
		return nil, fmt.Errorf("marshal evidence: %w", err)
	}
	now := time.Now()

	var out Exception
	row := ea.db.QueryRowxContext(ctx, `
		INSERT INTO platform_exceptions (
			id, tenant_id, type, severity, source, description, evidence,
			fingerprint, occurrence_count, first_seen, last_seen, status,
			autofix_attempts
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, 1, $9, $9, $10,
			'[]'::jsonb
		)
		ON CONFLICT (tenant_id, fingerprint)
		WHERE status IN ('open', 'acknowledged', 'auto_fix_pending')
		DO UPDATE SET
			occurrence_count = platform_exceptions.occurrence_count + 1,
			last_seen = EXCLUDED.last_seen,
			severity = EXCLUDED.severity,
			description = EXCLUDED.description,
			evidence = EXCLUDED.evidence,
			updated_at = now()
		RETURNING id, tenant_id, type, severity, source, description, evidence,
			fingerprint, occurrence_count, first_seen, last_seen, status,
			resolved_at, resolved_by, closed_by_ai, autofix_attempts
	`, exc.ID, exc.TenantID, exc.Type, exc.Severity, exc.Source, exc.Description, evidenceJSON,
		exc.Fingerprint, now, exc.Status)

	if err := row.StructScan(&out); err != nil {
		return nil, fmt.Errorf("publish exception: %w", err)
	}
	out.afterScan()

	if out.OccurrenceCount <= 1 {
		ea.emitLifecycle(ctx, out, "created", "")
	}
	return &out, nil
}

const exceptionColumns = `id, tenant_id, type, severity, source, description, evidence,
	fingerprint, occurrence_count, first_seen, last_seen, status,
	resolved_at, resolved_by, closed_by_ai, autofix_attempts`

func (ea *ExceptionAggregator) GetAllExceptions(ctx context.Context) ([]Exception, error) {
	if ea.db == nil {
		return []Exception{}, nil
	}
	var rows []Exception
	err := ea.db.SelectContext(ctx, &rows, `
		SELECT `+exceptionColumns+`
		FROM platform_exceptions
		ORDER BY last_seen DESC
		LIMIT 500
	`)
	if err != nil {
		return nil, fmt.Errorf("get all exceptions: %w", err)
	}
	for i := range rows {
		rows[i].afterScan()
	}
	return rows, nil
}

func (ea *ExceptionAggregator) GetSummary(ctx context.Context) (*ExceptionSummary, error) {
	exceptionsList, err := ea.GetAllExceptions(ctx)
	if err != nil {
		return nil, err
	}

	summary := &ExceptionSummary{
		TotalExceptions: len(exceptionsList),
		ByType:          map[ExceptionType]int{},
	}
	tenantSet := map[string]struct{}{}
	for _, ex := range exceptionsList {
		switch ex.Severity {
		case "critical":
			summary.CriticalCount++
		case "high":
			summary.HighCount++
		case "medium":
			summary.MediumCount++
		case "low":
			summary.LowCount++
		}
		summary.ByType[ex.Type]++
		tenantSet[ex.TenantID.String()] = struct{}{}
	}
	for t := range tenantSet {
		summary.TopAffectedTenants = append(summary.TopAffectedTenants, t)
	}
	if len(exceptionsList) > 5 {
		summary.RecentExceptions = exceptionsList[:5]
	} else {
		summary.RecentExceptions = exceptionsList
	}
	return summary, nil
}

func (ea *ExceptionAggregator) GetByType(ctx context.Context, exceptionType ExceptionType) ([]Exception, error) {
	if ea.db == nil {
		return []Exception{}, nil
	}
	var rows []Exception
	err := ea.db.SelectContext(ctx, &rows, `
		SELECT `+exceptionColumns+`
		FROM platform_exceptions
		WHERE type = $1
		ORDER BY last_seen DESC
		LIMIT 500
	`, exceptionType)
	if err != nil {
		return nil, fmt.Errorf("get exceptions by type: %w", err)
	}
	for i := range rows {
		rows[i].afterScan()
	}
	return rows, nil
}

// GetByID fetches a single exception, used by the rerun/close/ai-suggestion endpoints.
func (ea *ExceptionAggregator) GetByID(ctx context.Context, id uuid.UUID) (*Exception, error) {
	if ea.db == nil {
		return nil, fmt.Errorf("exceptions: no database configured")
	}
	var e Exception
	err := ea.db.GetContext(ctx, &e, `SELECT `+exceptionColumns+` FROM platform_exceptions WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	e.afterScan()
	return &e, nil
}

// AppendAutofixAttempt records an attempt and updates status in one write.
func (ea *ExceptionAggregator) AppendAutofixAttempt(ctx context.Context, id uuid.UUID, attempt AutofixAttempt, newStatus ExceptionStatus) error {
	if ea.db == nil {
		return nil
	}
	existing, err := ea.GetByID(ctx, id)
	if err != nil {
		return err
	}
	attempts := append(existing.AutofixAttempts, attempt)
	attemptsJSON, err := json.Marshal(attempts)
	if err != nil {
		return err
	}
	_, err = ea.db.ExecContext(ctx, `
		UPDATE platform_exceptions
		SET autofix_attempts = $2, status = $3, updated_at = now()
		WHERE id = $1
	`, id, attemptsJSON, newStatus)
	if err != nil {
		return err
	}
	existing.Status = newStatus
	ea.emitLifecycle(ctx, *existing, attempt.Action, attempt.Detail)
	return nil
}

// CloseExceptionOptions carries the terminal state for a resolved/closed exception.
type CloseExceptionOptions struct {
	Status     ExceptionStatus
	ResolvedBy string
	ClosedByAI bool
}

// Close marks an exception resolved/auto_fixed/closed. Called by the
// remediation workflow's verify-before-close step (only after verifyResolved
// succeeds) or by a human via POST /exceptions/{id}/close.
func (ea *ExceptionAggregator) Close(ctx context.Context, id uuid.UUID, opts CloseExceptionOptions) error {
	if ea.db == nil {
		return nil
	}
	_, err := ea.db.ExecContext(ctx, `
		UPDATE platform_exceptions
		SET status = $2, resolved_at = now(), resolved_by = $3, closed_by_ai = $4, updated_at = now()
		WHERE id = $1
	`, id, opts.Status, opts.ResolvedBy, opts.ClosedByAI)
	if err != nil {
		return err
	}
	if e, getErr := ea.GetByID(ctx, id); getErr == nil {
		e.Status = opts.Status
		stage := "closed"
		if opts.Status == StatusAutoFixed {
			stage = "auto_fixed"
		}
		ea.emitLifecycle(ctx, *e, stage, opts.ResolvedBy)
	}
	return nil
}

// ============================================================================
// Autofix policy resolution
// ============================================================================

// ResolveAutofixPolicy checks for a per-user override first, falling back to
// the tenant-wide default for that exception type. Returns disabled if
// neither row exists (safe default: no auto-fix unless explicitly enabled).
func (ea *ExceptionAggregator) ResolveAutofixPolicy(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, excType ExceptionType) (*AutofixPolicy, error) {
	if ea.db == nil {
		return &AutofixPolicy{TenantID: tenantID, ExceptionType: excType, Enabled: false, RequiresApproval: true}, nil
	}
	if userID != nil {
		var p AutofixPolicy
		err := ea.db.GetContext(ctx, &p, `
			SELECT id, tenant_id, user_id, exception_type, enabled, requires_approval, updated_by, updated_at
			FROM exception_autofix_policy
			WHERE tenant_id = $1 AND user_id = $2 AND exception_type = $3
		`, tenantID, *userID, excType)
		if err == nil {
			return &p, nil
		}
	}
	var p AutofixPolicy
	err := ea.db.GetContext(ctx, &p, `
		SELECT id, tenant_id, user_id, exception_type, enabled, requires_approval, updated_by, updated_at
		FROM exception_autofix_policy
		WHERE tenant_id = $1 AND user_id IS NULL AND exception_type = $2
	`, tenantID, excType)
	if err != nil {
		// No policy configured yet: default closed.
		return &AutofixPolicy{TenantID: tenantID, ExceptionType: excType, Enabled: false, RequiresApproval: true}, nil
	}
	return &p, nil
}

// ListAutofixPolicies returns every configured tenant-level default policy
// for the UI's per-type toggle panel.
func (ea *ExceptionAggregator) ListAutofixPolicies(ctx context.Context, tenantID uuid.UUID) ([]AutofixPolicy, error) {
	if ea.db == nil {
		return []AutofixPolicy{}, nil
	}
	var rows []AutofixPolicy
	err := ea.db.SelectContext(ctx, &rows, `
		SELECT id, tenant_id, user_id, exception_type, enabled, requires_approval, updated_by, updated_at
		FROM exception_autofix_policy
		WHERE tenant_id = $1 AND user_id IS NULL
		ORDER BY exception_type
	`, tenantID)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// SetAutofixPolicy upserts a tenant-default (userID nil) or per-user override
// policy row. This is the only place autofix is turned on — always scoped
// to one exception type, never a global toggle.
func (ea *ExceptionAggregator) SetAutofixPolicy(ctx context.Context, p AutofixPolicy) error {
	if ea.db == nil {
		return nil
	}
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	if p.UserID != nil {
		_, err := ea.db.ExecContext(ctx, `
			INSERT INTO exception_autofix_policy (id, tenant_id, user_id, exception_type, enabled, requires_approval, updated_by, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, now())
			ON CONFLICT (tenant_id, user_id, exception_type) WHERE user_id IS NOT NULL
			DO UPDATE SET enabled = EXCLUDED.enabled, requires_approval = EXCLUDED.requires_approval,
				updated_by = EXCLUDED.updated_by, updated_at = now()
		`, p.ID, p.TenantID, *p.UserID, p.ExceptionType, p.Enabled, p.RequiresApproval, p.UpdatedBy)
		return err
	}
	_, err := ea.db.ExecContext(ctx, `
		INSERT INTO exception_autofix_policy (id, tenant_id, user_id, exception_type, enabled, requires_approval, updated_by, updated_at)
		VALUES ($1, $2, NULL, $3, $4, $5, $6, now())
		ON CONFLICT (tenant_id, exception_type) WHERE user_id IS NULL
		DO UPDATE SET enabled = EXCLUDED.enabled, requires_approval = EXCLUDED.requires_approval,
			updated_by = EXCLUDED.updated_by, updated_at = now()
	`, p.ID, p.TenantID, p.ExceptionType, p.Enabled, p.RequiresApproval, p.UpdatedBy)
	return err
}

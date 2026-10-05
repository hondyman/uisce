package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/compliance/canonical"
)

// SnapshotMismatch represents a cryptographic divergence between stored and computed rule version hash
type SnapshotMismatch struct {
	RuleID       uuid.UUID `json:"rule_id"`
	RuleCode     string    `json:"rule_code"`
	Version      int       `json:"version"`
	TenantID     uuid.UUID `json:"tenant_id"`
	StoredHash   string    `json:"stored_hash"`
	ComputedHash string    `json:"computed_hash"`
	DetectedAt   time.Time `json:"detected_at"`
}

// ReconciliationReport summarizes the results of the rule snapshot hash verification sweep
type ReconciliationReport struct {
	TotalScanned int                `json:"total_scanned"`
	Matched      int                `json:"matched"`
	Mismatched   int                `json:"mismatched"`
	Mismatches   []SnapshotMismatch `json:"mismatches,omitempty"`
	DurationMs   int64              `json:"duration_ms"`
}

// Global metric counter for prometheus integration
var RuleSnapshotHashMismatchCount uint64

// RuleSnapshotReconciler iterates all compliance_rule_version snapshots,
// recomputes their RFC 8785 canonical content hash in Go, and alerts on mismatch.
type RuleSnapshotReconciler struct {
	db *sql.DB
}

// NewRuleSnapshotReconciler creates a new RuleSnapshotReconciler instance
func NewRuleSnapshotReconciler(db *sql.DB) *RuleSnapshotReconciler {
	return &RuleSnapshotReconciler{db: db}
}

// ReconcileAll runs a complete sweep across all stored rule versions in compliance_rule_version
func (r *RuleSnapshotReconciler) ReconcileAll(ctx context.Context) (*ReconciliationReport, error) {
	start := time.Now()

	rows, err := r.db.QueryContext(ctx, `
		SELECT 
			v.rule_id,
			COALESCE(r.rule_code, 'CUSTOM_RULE'),
			r.tenant_id,
			v.version,
			v.resolved_ast::text,
			v.parameter_thresholds::text,
			COALESCE(v.citation, ''),
			v.content_hash
		FROM compliance.compliance_rule_version v
		JOIN compliance.compliance_rule r ON v.rule_id = r.id
		WHERE r.valid_to IS NULL
		ORDER BY v.rule_id, v.version
	`)
	if err != nil {
		return nil, fmt.Errorf("query rule versions for reconciliation: %w", err)
	}
	defer rows.Close()

	report := &ReconciliationReport{
		Mismatches: make([]SnapshotMismatch, 0),
	}

	for rows.Next() {
		var (
			ruleIDStr, ruleCode, tenantIDStr, citation, storedHash string
			astStr, paramStr                                      string
			version                                               int
		)

		if err := rows.Scan(
			&ruleIDStr, &ruleCode, &tenantIDStr, &version,
			&astStr, &paramStr, &citation, &storedHash,
		); err != nil {
			return nil, fmt.Errorf("scan rule version row: %w", err)
		}

		ruleID, _ := uuid.Parse(ruleIDStr)
		tenantID, _ := uuid.Parse(tenantIDStr)

		// Recompute hash using Go as the single cryptographic authority
		computedHash, err := canonical.ComputeRuleContentHashFromRaw([]byte(astStr), []byte(paramStr), citation)
		if err != nil {
			return nil, fmt.Errorf("recompute hash for rule %s v%d: %w", ruleCode, version, err)
		}

		report.TotalScanned++

		if computedHash != storedHash {
			report.Mismatched++
			atomic.AddUint64(&RuleSnapshotHashMismatchCount, 1)

			mismatch := SnapshotMismatch{
				RuleID:       ruleID,
				RuleCode:     ruleCode,
				Version:      version,
				TenantID:     tenantID,
				StoredHash:   storedHash,
				ComputedHash: computedHash,
				DetectedAt:   time.Now(),
			}
			report.Mismatches = append(report.Mismatches, mismatch)

			// Record critical notification into database
			r.alertMismatch(ctx, mismatch)
		} else {
			report.Matched++
		}
	}

	report.DurationMs = time.Since(start).Milliseconds()
	return report, nil
}

func (r *RuleSnapshotReconciler) alertMismatch(ctx context.Context, m SnapshotMismatch) {
	payloadJSON, _ := json.Marshal(map[string]interface{}{
		"severity":      "CRITICAL",
		"rule_id":       m.RuleID,
		"rule_code":     m.RuleCode,
		"version":       m.Version,
		"stored_hash":   m.StoredHash,
		"computed_hash": m.ComputedHash,
		"detected_at":   m.DetectedAt.Format(time.RFC3339),
		"reason":        "Database snapshot content_hash does not match Go canonical RFC 8785 recomputation",
	})

	_, _ = r.db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_notification (
			id, tenant_id, kind, title, payload, is_read, created_at
		) VALUES (
			gen_random_uuid(), $1, 'SYSTEM', $2, $3::jsonb, false, now()
		)
	`, m.TenantID, fmt.Sprintf("CRITICAL: Rule Version Hash Mismatch on %s (v%d)", m.RuleCode, m.Version), string(payloadJSON))
}

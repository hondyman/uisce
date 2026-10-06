package blotter

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/compliance/canonical"
	"github.com/shopspring/decimal"
)

// Service provides high-level operations for compliance decision blotter and explainability.
type Service struct {
	db  *sql.DB
	hub *WebSocketHub
}

// NewService creates a new blotter Service instance.
func NewService(db *sql.DB, hub *WebSocketHub) *Service {
	return &Service{
		db:  db,
		hub: hub,
	}
}

// ListEvaluations retrieves paginated compliance evaluation events with filtering.
func (s *Service) ListEvaluations(ctx context.Context, filter ListFilter) (*PaginatedResponse[EvaluationEventRecord], error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 50
	}
	if filter.PageSize > 200 {
		filter.PageSize = 200
	}

	whereClauses := []string{"e.tenant_id = $1"}
	args := []any{filter.TenantID}
	argIdx := 2

	if filter.OrderID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("e.order_id = $%d", argIdx))
		args = append(args, *filter.OrderID)
		argIdx++
	}

	if filter.LineageID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("e.lineage_id = $%d", argIdx))
		args = append(args, *filter.LineageID)
		argIdx++
	}

	if filter.RuleCode != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("r.rule_code = $%d", argIdx))
		args = append(args, filter.RuleCode)
		argIdx++
	}

	if filter.ActionTaken != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("e.action_taken = $%d", argIdx))
		args = append(args, filter.ActionTaken)
		argIdx++
	}

	if filter.Passed != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("e.passed = $%d", argIdx))
		args = append(args, *filter.Passed)
		argIdx++
	}

	if filter.From != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("e.evaluated_at >= $%d", argIdx))
		args = append(args, *filter.From)
		argIdx++
	}

	if filter.To != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("e.evaluated_at <= $%d", argIdx))
		args = append(args, *filter.To)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// 1. Total count
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM compliance.compliance_evaluation_event e
		JOIN compliance.compliance_rule r ON e.rule_id = r.id
		WHERE %s
	`, whereSQL)

	var totalCount int64
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("count evaluations: %w", err)
	}

	// 2. Fetch page
	offset := (filter.Page - 1) * filter.PageSize
	query := fmt.Sprintf(`
		SELECT 
			e.id, e.lineage_id, e.tenant_id, e.order_id, e.rule_id,
			r.rule_code, r.name, e.rule_version, r.rule_phase, r.severity,
			e.passed, e.action_taken, e.latency_micros, e.rule_content_hash,
			e.evaluation_hash, e.input_params, e.metric_snapshots,
			e.evaluated_at, e.created_at
		FROM compliance.compliance_evaluation_event e
		JOIN compliance.compliance_rule r ON e.rule_id = r.id
		WHERE %s
		ORDER BY e.evaluated_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, filter.PageSize, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query evaluations: %w", err)
	}
	defer rows.Close()

	items := make([]EvaluationEventRecord, 0, filter.PageSize)
	for rows.Next() {
		var item EvaluationEventRecord
		var inputParamsBytes, metricSnapshotsBytes []byte
		var orderID sql.NullString

		if err := rows.Scan(
			&item.ID, &item.LineageID, &item.TenantID, &orderID, &item.RuleID,
			&item.RuleCode, &item.RuleName, &item.RuleVersion, &item.RulePhase, &item.Severity,
			&item.Passed, &item.ActionTaken, &item.LatencyMicros, &item.RuleContentHash,
			&item.EvaluationHash, &inputParamsBytes, &metricSnapshotsBytes,
			&item.EvaluatedAt, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan evaluation row: %w", err)
		}

		if orderID.Valid {
			u, err := uuid.Parse(orderID.String)
			if err == nil {
				item.OrderID = &u
			}
		}

		item.InputParams = json.RawMessage(inputParamsBytes)
		item.MetricSnapshots = json.RawMessage(metricSnapshotsBytes)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	totalPages := int((totalCount + int64(filter.PageSize) - 1) / int64(filter.PageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	return &PaginatedResponse[EvaluationEventRecord]{
		Data:       items,
		TotalCount: totalCount,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetEvidenceBundleByLineageID retrieves the complete decision evidence bundle for a given lineage ID.
func (s *Service) GetEvidenceBundleByLineageID(ctx context.Context, tenantID uuid.UUID, lineageID uuid.UUID) (*DecisionEvidenceBundle, error) {
	query := `
		SELECT 
			e.id, e.lineage_id, e.tenant_id, e.order_id, e.rule_id,
			r.rule_code, r.name, e.rule_version, r.rule_phase, r.severity,
			e.passed, e.action_taken, e.latency_micros, e.rule_content_hash,
			e.evaluation_hash, e.input_params, e.metric_snapshots,
			e.evaluated_at, e.created_at
		FROM compliance.compliance_evaluation_event e
		JOIN compliance.compliance_rule r ON e.rule_id = r.id
		WHERE e.lineage_id = $1 AND e.tenant_id = $2
		ORDER BY e.evaluated_at DESC
		LIMIT 1
	`

	var item EvaluationEventRecord
	var inputParamsBytes, metricSnapshotsBytes []byte
	var orderID sql.NullString

	err := s.db.QueryRowContext(ctx, query, lineageID, tenantID).Scan(
		&item.ID, &item.LineageID, &item.TenantID, &orderID, &item.RuleID,
		&item.RuleCode, &item.RuleName, &item.RuleVersion, &item.RulePhase, &item.Severity,
		&item.Passed, &item.ActionTaken, &item.LatencyMicros, &item.RuleContentHash,
		&item.EvaluationHash, &inputParamsBytes, &metricSnapshotsBytes,
		&item.EvaluatedAt, &item.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("evaluation with lineage_id %s not found", lineageID)
		}
		return nil, fmt.Errorf("query evaluation by lineage: %w", err)
	}

	if orderID.Valid {
		u, err := uuid.Parse(orderID.String)
		if err == nil {
			item.OrderID = &u
		}
	}
	item.InputParams = json.RawMessage(inputParamsBytes)
	item.MetricSnapshots = json.RawMessage(metricSnapshotsBytes)

	return s.buildEvidenceBundle(ctx, item)
}

// GetEvidenceBundleByID retrieves the complete decision evidence bundle for a given evaluation ID.
func (s *Service) GetEvidenceBundleByID(ctx context.Context, tenantID uuid.UUID, evalID uuid.UUID) (*DecisionEvidenceBundle, error) {
	query := `
		SELECT 
			e.id, e.lineage_id, e.tenant_id, e.order_id, e.rule_id,
			r.rule_code, r.name, e.rule_version, r.rule_phase, r.severity,
			e.passed, e.action_taken, e.latency_micros, e.rule_content_hash,
			e.evaluation_hash, e.input_params, e.metric_snapshots,
			e.evaluated_at, e.created_at
		FROM compliance.compliance_evaluation_event e
		JOIN compliance.compliance_rule r ON e.rule_id = r.id
		WHERE e.id = $1 AND e.tenant_id = $2
		LIMIT 1
	`

	var item EvaluationEventRecord
	var inputParamsBytes, metricSnapshotsBytes []byte
	var orderID sql.NullString

	err := s.db.QueryRowContext(ctx, query, evalID, tenantID).Scan(
		&item.ID, &item.LineageID, &item.TenantID, &orderID, &item.RuleID,
		&item.RuleCode, &item.RuleName, &item.RuleVersion, &item.RulePhase, &item.Severity,
		&item.Passed, &item.ActionTaken, &item.LatencyMicros, &item.RuleContentHash,
		&item.EvaluationHash, &inputParamsBytes, &metricSnapshotsBytes,
		&item.EvaluatedAt, &item.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("evaluation with id %s not found", evalID)
		}
		return nil, fmt.Errorf("query evaluation by id: %w", err)
	}

	if orderID.Valid {
		u, err := uuid.Parse(orderID.String)
		if err == nil {
			item.OrderID = &u
		}
	}
	item.InputParams = json.RawMessage(inputParamsBytes)
	item.MetricSnapshots = json.RawMessage(metricSnapshotsBytes)

	return s.buildEvidenceBundle(ctx, item)
}

// buildEvidenceBundle fetches immutable rule version snapshot and synthesizes complete proof and natural explanation.
func (s *Service) buildEvidenceBundle(ctx context.Context, eval EvaluationEventRecord) (*DecisionEvidenceBundle, error) {
	// Query immutable rule version snapshot
	var snap RuleSnapshotInfo
	var astBytes, paramsBytes []byte

	snapQuery := `
		SELECT rule_id, version, content_hash, resolved_ast, parameter_thresholds, citation
		FROM compliance.compliance_rule_version
		WHERE rule_id = $1 AND version = $2
	`
	err := s.db.QueryRowContext(ctx, snapQuery, eval.RuleID, eval.RuleVersion).Scan(
		&snap.RuleID, &snap.Version, &snap.ContentHash, &astBytes, &paramsBytes, &snap.Citation,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: snapshot missing for rule %s version %d", ErrProvenanceVerificationFailed, eval.RuleID, eval.RuleVersion)
		}
		return nil, fmt.Errorf("query rule version snapshot: %w", err)
	}
	snap.ResolvedAST = json.RawMessage(astBytes)
	snap.ParameterThresholds = json.RawMessage(paramsBytes)

	// 2. Recompute RFC 8785 canonical hash on the snapshot logic
	recomputedHash, err := canonical.ComputeRuleContentHashFromRaw(astBytes, paramsBytes, snap.Citation)
	if err != nil {
		return nil, fmt.Errorf("%w: canonicalization error: %v", ErrProvenanceVerificationFailed, err)
	}
	if recomputedHash != snap.ContentHash {
		return nil, fmt.Errorf("%w: recomputed RFC 8785 hash %s does not match snapshot stored hash %s (logic tampering detected)", ErrProvenanceVerificationFailed, recomputedHash, snap.ContentHash)
	}

	// 3. Assert evaluation event's recorded content hash matches snapshot hash
	if eval.RuleContentHash != "" && eval.RuleContentHash != snap.ContentHash {
		return nil, fmt.Errorf("%w: event rule_content_hash %s does not match snapshot content_hash %s for rule %s v%d", ErrProvenanceVerificationFailed, eval.RuleContentHash, snap.ContentHash, eval.RuleID, eval.RuleVersion)
	}

	contentHashMatches := true

	// Parse metrics and parameters
	metrics := extractEvaluatedMetrics(eval.MetricSnapshots, snap.ParameterThresholds, eval.InputParams)

	// Generate Natural-Language Decision Explanation
	explanation := generateNaturalLanguageExplanation(eval, snap, metrics)

	bundle := &DecisionEvidenceBundle{
		Evaluation:   eval,
		RuleSnapshot: snap,
		Metrics:      metrics,
		NaturalLanguageExplanation: explanation,
		IntegrityProof: IntegrityProof{
			ContentHashMatches:    contentHashMatches,
			StoredContentHash:     snap.ContentHash,
			RecomputedContentHash: recomputedHash,
			EvaluationHash:        eval.EvaluationHash,
			Authority:             "Go RFC 8785 JSON Canonicalization (v1 schema)",
		},
	}

	return bundle, nil
}

// extractEvaluatedMetrics parses JSON metric snapshots and maps them against thresholds.
func extractEvaluatedMetrics(metricSnapshotsJSON, thresholdsJSON, inputParamsJSON json.RawMessage) []EvaluatedMetricItem {
	var metricsMap map[string]any
	_ = json.Unmarshal(metricSnapshotsJSON, &metricsMap)

	var thresholdsMap map[string]any
	_ = json.Unmarshal(thresholdsJSON, &thresholdsMap)

	var inputParamsMap map[string]any
	_ = json.Unmarshal(inputParamsJSON, &inputParamsMap)

	items := make([]EvaluatedMetricItem, 0, len(metricsMap))
	for k, v := range metricsMap {
		item := EvaluatedMetricItem{
			MetricPath:    k,
			ObservedValue: v,
		}

		// Check if there is a matching threshold
		// e.g. pos.issuer_pct vs issuer_limit_pct or direct match
		for tKey, tVal := range thresholdsMap {
			if strings.Contains(strings.ToLower(k), "issuer") && strings.Contains(strings.ToLower(tKey), "issuer") ||
				strings.Contains(strings.ToLower(k), "weight") && strings.Contains(strings.ToLower(tKey), "limit") ||
				k == tKey {
				item.ThresholdValue = tVal
				item.Operator = "LTE"

				// Attempt numerical comparison
				vDec, err1 := decimal.NewFromString(fmt.Sprintf("%v", v))
				tDec, err2 := decimal.NewFromString(fmt.Sprintf("%v", tVal))
				if err1 == nil && err2 == nil {
					diff := vDec.Sub(tDec)
					if diff.GreaterThan(decimal.Zero) {
						item.Breached = true
						item.Margin = fmt.Sprintf("+%s (exceeded limit)", diff.StringFixed(4))
					} else {
						item.Breached = false
						item.Margin = fmt.Sprintf("%s (within limit)", diff.StringFixed(4))
					}
				}
				break
			}
		}

		items = append(items, item)
	}

	return items
}

// generateNaturalLanguageExplanation builds an auditor-grade, clear textual explanation.
func generateNaturalLanguageExplanation(eval EvaluationEventRecord, snap RuleSnapshotInfo, metrics []EvaluatedMetricItem) string {
	var sb strings.Builder

	orderRef := "Pre-trade order"
	if eval.OrderID != nil {
		orderRef = fmt.Sprintf("Order %s", eval.OrderID.String()[:8])
	}

	switch eval.ActionTaken {
	case "APPROVED":
		sb.WriteString(fmt.Sprintf("%s PASSED %s (%s v%d). ", orderRef, eval.RuleName, eval.RuleCode, eval.RuleVersion))
		sb.WriteString("All portfolio metrics and constraint limits remained fully within regulatory parameters. ")
	case "BLOCKED":
		sb.WriteString(fmt.Sprintf("%s was BLOCKED by %s (%s v%d). ", orderRef, eval.RuleName, eval.RuleCode, eval.RuleVersion))
		sb.WriteString(fmt.Sprintf("Execution would result in a hard violation under severity level %s. ", eval.Severity))
	case "WARNED":
		sb.WriteString(fmt.Sprintf("%s generated a COMPLIANCE WARNING on %s (%s v%d). ", orderRef, eval.RuleName, eval.RuleCode, eval.RuleVersion))
		sb.WriteString("Execution is permitted with advisory tracking and supervisor notification. ")
	case "APPROVAL_PENDING":
		sb.WriteString(fmt.Sprintf("%s requires COMPLIANCE APPROVAL for %s (%s v%d). ", orderRef, eval.RuleName, eval.RuleCode, eval.RuleVersion))
		sb.WriteString("Trade routing is held in escrow awaiting compliance officer authorization. ")
	default:
		sb.WriteString(fmt.Sprintf("%s evaluated %s (%s v%d) with status %s. ", orderRef, eval.RuleName, eval.RuleCode, eval.RuleVersion, eval.ActionTaken))
	}

	if len(metrics) > 0 {
		sb.WriteString("Observed metric values: ")
		metricDescs := make([]string, 0, len(metrics))
		for _, m := range metrics {
			if m.ThresholdValue != nil {
				metricDescs = append(metricDescs, fmt.Sprintf("%s = %v (Threshold: %v, %s)", m.MetricPath, m.ObservedValue, m.ThresholdValue, m.Margin))
			} else {
				metricDescs = append(metricDescs, fmt.Sprintf("%s = %v", m.MetricPath, m.ObservedValue))
			}
		}
		sb.WriteString(strings.Join(metricDescs, "; ") + ". ")
	}

	if snap.Citation != "" {
		sb.WriteString(fmt.Sprintf("Governing regulatory authority / legal citation: \"%s\". ", snap.Citation))
	}

	sb.WriteString(fmt.Sprintf("Evaluated in %dµs with cryptographic content anchor %s.", eval.LatencyMicros, truncateHash(eval.RuleContentHash)))

	return sb.String()
}

func truncateHash(h string) string {
	if len(h) <= 12 {
		return h
	}
	return h[:6] + "..." + h[len(h)-6:]
}

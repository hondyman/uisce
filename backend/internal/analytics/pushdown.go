package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/rules/vm"
)

// --- Column resolution (semantic term → physical column, allowlisted) ---

var sqlIdentRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// safeColumn validates against a strict identifier allowlist, then quotes.
// Catalog resolution is the source; this is the second layer against a
// polluted catalog.
func safeColumn(name string) (string, error) {
	if !sqlIdentRe.MatchString(name) {
		return "", fmt.Errorf("column %q fails identifier allowlist", name)
	}
	return pq.QuoteIdentifier(name), nil
}

// safeTable validates schema-qualified names ("schema.table" or "table").
func safeTable(name string) (string, error) {
	parts := strings.Split(name, ".")
	if len(parts) > 2 {
		return "", fmt.Errorf("table %q: too many qualifiers", name)
	}
	q := make([]string, len(parts))
	for i, p := range parts {
		c, err := safeColumn(p) // same charset rules apply
		if err != nil {
			return "", fmt.Errorf("table %q: %w", name, err)
		}
		q[i] = c
	}
	return strings.Join(q, "."), nil
}

// boColumns loads the semantic term → physical column map for one BO by
// joining business_object_fields through MAPS_TO edges.
func (s *ValidationRuleService) boColumns(ctx context.Context, tenantID, gold string, visible []string, boName string) (map[string]string, error) {
	var rows []struct {
		FieldName  string `db:"field_name"`
		ColumnName string `db:"column_name"`
	}
	err := s.db.SelectContext(ctx, &rows, `
		SELECT bf.field_name,
		       COALESCE(NULLIF(col.node_name, ''), split_part(col.qualified_path, '/', 2)) AS column_name
		FROM business_object_fields bf
		JOIN business_objects bo ON bo.id = bf.bo_id
		JOIN catalog_edge ce ON ce.source_node_id = bf.term_node_id
		JOIN catalog_edge_type et ON et.id = ce.edge_type_id
		JOIN catalog_node col ON col.id = ce.target_node_id
		WHERE (bo.bo_key = $1 OR bo.bo_name = $1)
		  AND bo.tenant_id = ANY($2::uuid[])
		  AND et.edge_type_name = 'MAPS_TO'
	`, boName, pq.Array(visible))
	if err != nil {
		return nil, fmt.Errorf("resolve column mappings for BO %q: %w", boName, err)
	}
	m := make(map[string]string, len(rows))
	for _, r := range rows {
		m[r.FieldName] = r.ColumnName
	}
	return m, nil
}

func resolverFor(cols map[string]string) (vm.ColumnResolver, error) {
	quoted := make(map[string]string, len(cols))
	for term, col := range cols {
		q, err := safeColumn(col)
		if err != nil {
			return nil, fmt.Errorf("term %q: %w", term, err)
		}
		quoted[term] = q
	}
	return func(field string) (string, error) {
		if q, ok := quoted[field]; ok {
			return q, nil
		}
		return "", fmt.Errorf("semantic term %q has no MAPS_TO column binding", field)
	}, nil
}

// --- Compiled rule + tenant enforcement ---

// compiledRule carries the pushdown predicate plus the structural guard:
// the executor refuses to run anything not compiled through
// CompileRuleForPushdown, which binds tenant_id as $1 unconditionally.
type compiledRule struct {
	rule        *EvaluableRule
	passSQL     string // TRUE = record passes
	args        []any
	tenantBound bool
	table       string
}

// CompileRuleForPushdown compiles one snapshot rule against a BO's column map.
// The tenant predicate is emitted here, by the compiler — not by callers.
func CompileRuleForPushdown(rule *EvaluableRule, table, tenantID string, cols map[string]string) (*compiledRule, error) {
	if rule.AST == nil {
		return nil, fmt.Errorf("rule %q has no parsed AST", rule.RuleKey)
	}
	tbl, err := safeTable(table)
	if err != nil {
		return nil, err
	}
	resolve, err := resolverFor(cols)
	if err != nil {
		return nil, err
	}
	b := &vm.ParamBinder{}
	tenantRef := b.Bind(tenantID) // ALWAYS $1 — structural Layer 2
	if tenantRef != "$1" {
		return nil, fmt.Errorf("tenant parameter must be $1, got %s", tenantRef)
	}
	passSQL, err := vm.CompileToSQL(*rule.AST, resolve, b)
	if err != nil {
		return nil, fmt.Errorf("rule %q: %w", rule.RuleKey, err)
	}
	return &compiledRule{
		rule:        rule,
		passSQL:     passSQL,
		args:        b.Args(),
		tenantBound: true,
		table:       tbl,
	}, nil
}

// --- Executor ---

// PushdownConnResolver supplies the tenant's data-plane connection
// (Layer 1). Implemented over platform.TenantDBManager.GetConnection.
type PushdownConnResolver interface {
	DataPlaneConn(ctx context.Context, tenantID string) (*sql.DB, error)
}

type PushdownOptions struct {
	MaxDetailRows int // sampled violation record IDs persisted; 0 = counts only
}

type RulePushdownResult struct {
	RuleID     string `json:"rule_id"`
	RuleKey    string `json:"rule_key"`
	RuleName   string `json:"rule_name"`
	Severity   string `json:"severity"`
	TotalRows  int64  `json:"total_rows"`
	Violations int64  `json:"violations"`
	RuleErrors int64  `json:"rule_errors"`
	Persisted  int    `json:"persisted"`
	CompileErr string `json:"compile_error,omitempty"`
}

type PushdownResult struct {
	SnapshotID string               `json:"snapshot_id"`
	BOName     string               `json:"bo_name"`
	TenantID   string               `json:"tenant_id"`
	Results    []RulePushdownResult `json:"results"`
	DurationMs int64                `json:"duration_ms"`
}

// PersistViolation writes a violation to the database (convenience wrapper).
func (s *ValidationRuleService) PersistViolation(ctx context.Context, v ViolationRecord) error {
	return PersistViolation(ctx, s.db, v)
}

// RunPushdown validates a full-table (or materialized view) against a rule
// snapshot inside the tenant's data plane. Violation telemetry persists to
// alpha (public.validation_rule_violations); raw data never leaves the plane.
func (s *ValidationRuleService) RunPushdown(ctx context.Context, connRes PushdownConnResolver, snap *RuleSnapshot, table string, pkColumn string, opts PushdownOptions) (*PushdownResult, error) {
	start := time.Now()
	if snap == nil || len(snap.Rules) == 0 {
		return nil, fmt.Errorf("empty rule snapshot")
	}
	// Layer 1: structural isolation — data plane or nothing.
	conn, err := connRes.DataPlaneConn(ctx, snap.TenantID)
	if err != nil {
		return nil, fmt.Errorf("resolve data-plane connection for tenant %s: %w", snap.TenantID, err)
	}
	pk, err := safeColumn(pkColumn)
	if err != nil {
		return nil, fmt.Errorf("pk column: %w", err)
	}

	res := &PushdownResult{
		SnapshotID: snap.SnapshotID,
		BOName:     snap.BOName,
		TenantID:   snap.TenantID,
		Results:    make([]RulePushdownResult, 0, len(snap.Rules)),
	}

	for i := range snap.Rules {
		rule := &snap.Rules[i]
		out := RulePushdownResult{
			RuleID:   rule.RuleID,
			RuleKey:  rule.RuleKey,
			RuleName: rule.RuleName,
			Severity: rule.Severity,
		}
		if rule.AST == nil {
			out.CompileErr = "rule has no parsed AST"
			res.Results = append(res.Results, out)
			continue
		}
		cr, cerr := CompileRuleForPushdown(rule, table, snap.TenantID, snap.ColumnMap)
		if cerr != nil {
			out.CompileErr = cerr.Error()
			res.Results = append(res.Results, out)
			continue
		}
		// Layer 2 guard: refuse anything without the compiler-emitted tenant predicate.
		if !cr.tenantBound {
			out.CompileErr = "compiled rule missing tenant predicate (refused)"
			res.Results = append(res.Results, out)
			continue
		}

		counts, err := runPushdownCounts(ctx, conn, cr)
		if err != nil {
			return nil, fmt.Errorf("pushdown counts for rule %q: %w", rule.RuleKey, err)
		}
		out.TotalRows, out.Violations, out.RuleErrors = counts[0], counts[1], counts[2]

		// Throttled detail persistence (Task 2 scope, embedded here):
		// sample at most MaxDetailRows offending record IDs.
		if opts.MaxDetailRows > 0 && out.Violations > 0 {
			ids, derr := runPushdownDetail(ctx, conn, cr, pk, opts.MaxDetailRows)
			if derr != nil {
				return nil, fmt.Errorf("pushdown detail for rule %q: %w", rule.RuleKey, derr)
			}
			for _, id := range ids {
				v := models.ViolationRecord{
					TenantID:     snap.TenantID,
					RuleID:       rule.RuleID,
					RuleKey:      rule.RuleKey,
					RuleVersion:  rule.RuleVersion,
					RuleName:     rule.RuleName,
					BOKey:        rule.BOName,
					Severity:     rule.Severity,
					RecordID:     id,
					Message:      fmt.Sprintf("pushdown audit: rule %q violated (sampled 1 of %d)", rule.RuleName, out.Violations),
					Fields:       rule.FieldRefs,
					WriteBlocked: rule.Severity == models.ValidationRuleSeverityBlock,
				}
				if perr := s.PersistViolation(ctx, v); perr != nil { // alpha sink
					return nil, fmt.Errorf("persist violation for rule %q record %q: %w", rule.RuleKey, id, perr)
				}
				out.Persisted++
			}
		}
		res.Results = append(res.Results, out)
	}
	res.DurationMs = time.Since(start).Milliseconds()
	return res, nil
}

func runPushdownCounts(ctx context.Context, conn *sql.DB, cr *compiledRule) ([3]int64, error) {
	var out [3]int64
	// (pass_sql) IS NULL is only ever TRUE for expression rules — condition
	// nodes COALESCE internally, matching Go's condition-never-errors semantics.
	q := fmt.Sprintf(`
		SELECT COUNT(*) AS total,
		       COUNT(*) FILTER (WHERE NOT COALESCE((%s), FALSE) AND (%s) IS NOT NULL) AS violations,
		       COUNT(*) FILTER (WHERE (%s) IS NULL) AS rule_errors
		FROM %s WHERE tenant_id = $1`,
		cr.passSQL, cr.passSQL, cr.passSQL, cr.table)
	err := conn.QueryRowContext(ctx, q, cr.args...).Scan(&out[0], &out[1], &out[2])
	return out, err
}

func runPushdownDetail(ctx context.Context, conn *sql.DB, cr *compiledRule, pk string, limit int) ([]string, error) {
	q := fmt.Sprintf(`
		SELECT %s FROM %s
		WHERE tenant_id = $1 AND NOT COALESCE((%s), FALSE) AND (%s) IS NOT NULL
		LIMIT $2`,
		pk, cr.table, cr.passSQL, cr.passSQL)
	args := append(append([]any{}, cr.args...), limit)
	rows, err := conn.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

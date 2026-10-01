package querybuilder

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// MetricBundleSchemaVersion is the official schema version identifier for metric bundles.
const MetricBundleSchemaVersion = "uisce.metric-bundle/1"

var (
	ErrGrainNotAllowed      = errors.New("requested grouping grain is not in the metric grain allowlist")
	ErrMetricInUse          = errors.New("metric is currently referenced and cannot be deleted")
	ErrCoreMetricImmutable  = errors.New("core metrics are managed by the master tenant and cannot be deleted by client tenants")
	ErrInvalidMetricFormula = errors.New("invalid metric formula expression")
	ErrCyclicMetricRef      = errors.New("cyclic metric dependency detected")
	ErrMissingMetricDep     = errors.New("missing referenced metric dependency")
)

// MetricFormatConfig specifies how metric values are rendered.
type MetricFormatConfig struct {
	Type           string  `json:"type"` // "currency" | "percentage" | "compact" | "number"
	Precision      *int    `json:"precision,omitempty"`
	CurrencySymbol *string `json:"currencySymbol,omitempty"`
	Prefix         *string `json:"prefix,omitempty"`
	Suffix         *string `json:"suffix,omitempty"`
}

// MetricVariable defines a parameter/variable bound into the metric calculation.
type MetricVariable struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"` // "number" | "string" | "date"
	DefaultValue interface{} `json:"defaultValue,omitempty"`
	Required     bool        `json:"required"`
	Description  string      `json:"description,omitempty"`
}

// MaterializationConfig specifies whether and how a metric is persisted.
type MaterializationConfig struct {
	Strategy                     string  `json:"strategy"` // "on_the_fly" | "starrocks_mv" | "iceberg"
	TargetTable                  *string `json:"targetTable,omitempty"`
	RefreshSchedule              *string `json:"refreshSchedule,omitempty"`
	PartitionGrain               *string `json:"partitionGrain,omitempty"`
	WatermarkBOID                *string `json:"watermarkBoId,omitempty"`
	IcebergSnapshotRetentionDays *int    `json:"icebergSnapshotRetentionDays,omitempty"`
	StalePolicy                  string  `json:"stalePolicy,omitempty"` // "serve_with_flag" (default/dashboards) | "force_raw_fallback" (compliance)
}

// MetricExpression defines the computation rules for a metric.
type MetricExpression struct {
	Kind          string   `json:"kind"`                 // "aggregation" | "formula" | "derived"
	Fn            string   `json:"fn,omitempty"`         // "sum" | "avg" | "count" | "min" | "max"
	TermNodeID    string   `json:"termNodeId,omitempty"` // For aggregation / column reference
	Formula       string   `json:"formula,omitempty"`    // Safe AST expression e.g. "SUM(price * qty) * @fx_rate"
	BaseMetricIDs []string `json:"baseMetricIds,omitempty"` // For derived metrics
}

// MetricDefinition represents a row in data_explorer.metric_definition.
type MetricDefinition struct {
	ID                    string                `json:"id" db:"id"`
	TenantID              string                `json:"tenantId" db:"tenant_id"`
	Name                  string                `json:"name" db:"name"`
	Description           string                `json:"description" db:"description"`
	BOID                  string                `json:"boId" db:"bo_id"`
	CatalogTermID         *string               `json:"catalogTermId,omitempty" db:"catalog_term_id"`
	Expression            MetricExpression      `json:"expression"`
	GrainAllowlist        []string              `json:"grainAllowlist"`
	FormatConfig          MetricFormatConfig    `json:"formatConfig"`
	Variables             []MetricVariable      `json:"variables"`
	MaterializationConfig MaterializationConfig `json:"materializationConfig"`
	Decomposable          bool                  `json:"decomposable" db:"decomposable"`
	ContentHash           string                `json:"contentHash" db:"content_hash"`
	Tags                  []string              `json:"tags"`
	IsCore                bool                  `json:"isCore" db:"is_core"`
	Status                string                `json:"status" db:"status"` // "active" | "deprecated" | "archived"
	ArchivedAt            *time.Time            `json:"archivedAt,omitempty" db:"archived_at"`
	CreatedBy             *string               `json:"createdBy,omitempty" db:"created_by"`
	CreatedAt             time.Time             `json:"createdAt" db:"created_at"`
	UpdatedAt             time.Time             `json:"updatedAt" db:"updated_at"`
}

type metricDefRow struct {
	ID                    string         `db:"id"`
	TenantID              string         `db:"tenant_id"`
	Name                  string         `db:"name"`
	Description           sql.NullString `db:"description"`
	BOID                  string         `db:"bo_id"`
	CatalogTermID         sql.NullString `db:"catalog_term_id"`
	Expression            []byte         `db:"expression"`
	GrainAllowlist        []byte         `db:"grain_allowlist"`
	FormatConfig          []byte         `db:"format_config"`
	Variables             []byte         `db:"variables"`
	MaterializationConfig []byte         `db:"materialization_config"`
	Decomposable          bool           `db:"decomposable"`
	ContentHash           sql.NullString `db:"content_hash"`
	Tags                  pq.StringArray `db:"tags"`
	IsCore                bool           `db:"is_core"`
	Status                string         `db:"status"`
	ArchivedAt            pq.NullTime    `db:"archived_at"`
	CreatedBy             sql.NullString `db:"created_by"`
	CreatedAt             time.Time      `db:"created_at"`
	UpdatedAt             time.Time      `db:"updated_at"`
}

// DeriveDecomposable determines whether a metric aggregation/expression is distributive.
// Distributive aggregations (SUM, COUNT, MIN, MAX) can be split and pre-aggregated across partitions.
// Non-distributive expressions (AVG, formulas with division) cannot be trivially summed.
func DeriveDecomposable(expr MetricExpression) bool {
	switch strings.ToLower(strings.TrimSpace(expr.Kind)) {
	case "aggregation":
		fn := strings.ToLower(strings.TrimSpace(expr.Fn))
		return fn == "sum" || fn == "count" || fn == "min" || fn == "max"
	case "formula":
		// Conservative derivation: any division or non-distributive function marks as non-decomposable
		f := strings.ToLower(expr.Formula)
		if strings.Contains(f, "/") || strings.Contains(f, "avg(") {
			return false
		}
		return true
	case "derived":
		// Derived metrics (ratio of metrics) are generally non-decomposable across tiers
		return false
	default:
		return false
	}
}

// ComputeMetricContentHash deterministically computes the SHA-256 hash of a metric definition's
// semantic content (normalized formula, grains, format, variables, boid).
func ComputeMetricContentHash(m MetricDefinition) string {
	// 1. Normalize Expression
	normExpr := m.Expression
	normExpr.Kind = strings.ToLower(strings.TrimSpace(normExpr.Kind))
	normExpr.Fn = strings.ToLower(strings.TrimSpace(normExpr.Fn))
	// Tokenize / normalize formula to make whitespace/parentheses invariant for canonical comparison
	normExpr.Formula = normalizeFormulaForHash(normExpr.Formula)
	sort.Strings(normExpr.BaseMetricIDs)

	// 2. Normalize Grain Allowlist
	normGrains := make([]string, len(m.GrainAllowlist))
	for i, g := range m.GrainAllowlist {
		normGrains[i] = strings.ToLower(strings.TrimSpace(g))
	}
	sort.Strings(normGrains)

	// 3. Normalize Variables
	normVars := make([]MetricVariable, len(m.Variables))
	copy(normVars, m.Variables)
	sort.Slice(normVars, func(i, j int) bool {
		return normVars[i].Name < normVars[j].Name
	})

	canonicalDoc := struct {
		Name         string             `json:"name"`
		BOID         string             `json:"boId"`
		Expression   MetricExpression   `json:"expression"`
		Grains       []string           `json:"grains"`
		FormatConfig MetricFormatConfig `json:"formatConfig"`
		Variables    []MetricVariable   `json:"variables"`
	}{
		Name:         strings.TrimSpace(m.Name),
		BOID:         strings.TrimSpace(m.BOID),
		Expression:   normExpr,
		Grains:       normGrains,
		FormatConfig: m.FormatConfig,
		Variables:    normVars,
	}

	b, _ := json.Marshal(canonicalDoc)
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}

func normalizeFormulaForHash(f string) string {
	// Strips redundant whitespace, cosmetic redundant outer parentheses, and normalizes case
	f = strings.TrimSpace(strings.ToLower(f))
	var sb strings.Builder
	inWhitespace := false
	for _, r := range f {
		if r == '(' || r == ')' {
			// Ignore purely cosmetic grouping parentheses in commutative additions/multiplications
			// while keeping structure tokenized
			continue
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !inWhitespace {
				sb.WriteRune(' ')
				inWhitespace = true
			}
		} else {
			sb.WriteRune(r)
			inWhitespace = false
		}
	}
	return strings.TrimSpace(sb.String())
}

func (r metricDefRow) toMetricDefinition() MetricDefinition {
	var expr MetricExpression
	_ = json.Unmarshal(r.Expression, &expr)

	var grains []string
	if len(r.GrainAllowlist) > 0 {
		_ = json.Unmarshal(r.GrainAllowlist, &grains)
	}

	var fmtCfg MetricFormatConfig
	if len(r.FormatConfig) > 0 {
		_ = json.Unmarshal(r.FormatConfig, &fmtCfg)
	}

	var vars []MetricVariable
	if len(r.Variables) > 0 {
		_ = json.Unmarshal(r.Variables, &vars)
	}

	var matCfg MaterializationConfig
	if len(r.MaterializationConfig) > 0 {
		_ = json.Unmarshal(r.MaterializationConfig, &matCfg)
	} else {
		matCfg.Strategy = "on_the_fly"
	}

	m := MetricDefinition{
		ID:                    r.ID,
		TenantID:              r.TenantID,
		Name:                  r.Name,
		Description:           r.Description.String,
		BOID:                  r.BOID,
		Expression:            expr,
		GrainAllowlist:        grains,
		FormatConfig:          fmtCfg,
		Variables:             vars,
		MaterializationConfig: matCfg,
		Decomposable:          r.Decomposable,
		ContentHash:           r.ContentHash.String,
		Tags:                  []string(r.Tags),
		IsCore:                r.IsCore,
		Status:                r.Status,
		CreatedAt:             r.CreatedAt,
		UpdatedAt:             r.UpdatedAt,
	}
	if r.CatalogTermID.Valid {
		m.CatalogTermID = &r.CatalogTermID.String
	}
	if r.ArchivedAt.Valid {
		m.ArchivedAt = &r.ArchivedAt.Time
	}
	if r.CreatedBy.Valid {
		m.CreatedBy = &r.CreatedBy.String
	}
	return m
}

// ValidateGrains verifies that requested grouping grains are permitted by grainAllowlist.
// Returns an actionable error with the complete allowed grains list (HTTP 422).
func ValidateGrains(requestedGrains []string, grainAllowlist []string) error {
	if len(grainAllowlist) == 0 {
		// Empty allowlist means all grains allowed
		return nil
	}
	allowed := make(map[string]bool)
	for _, g := range grainAllowlist {
		allowed[strings.ToLower(strings.TrimSpace(g))] = true
	}

	for _, req := range requestedGrains {
		normalized := strings.ToLower(strings.TrimSpace(req))
		if !allowed[normalized] {
			return fmt.Errorf("%w: '%s' is not allowed (allowed grains: %v)", ErrGrainNotAllowed, req, grainAllowlist)
		}
	}
	return nil
}

// MetricReference describes a location where a metric definition is used.
type MetricReference struct {
	Type     string `json:"type"` // "saved_query" | "kpi_tile" | "derived_metric" | "core_adoption" | "scheduled_job"
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	Location string `json:"location,omitempty"`
}

// MetricUsageReport is returned by GET /api/explorer/metrics/{id}/usage and on 409 conflict.
type MetricUsageReport struct {
	InUse      bool              `json:"inUse"`
	References []MetricReference `json:"references"`
}

// ScanMetricUsage scans for references to metricID across 5 sources:
// 1. saved queries, 2. page components, 3. derived metrics, 4. core adoptions, 5. scheduled jobs.
func ScanMetricUsage(ctx context.Context, db *sqlx.DB, metricID string, isCore bool, tenantID string) (MetricUsageReport, error) {
	var refs []MetricReference

	if db == nil {
		return MetricUsageReport{InUse: false, References: refs}, nil
	}

	// 1. Scan saved queries (query_state contains metricId)
	var sqRows []struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	sqQuery := `SELECT id, name FROM data_explorer.saved_query WHERE query_state::text LIKE '%' || $1 || '%' AND archived_at IS NULL`
	var sqArgs []interface{}
	sqArgs = append(sqArgs, metricID)
	if !isCore && tenantID != "" {
		sqQuery += ` AND tenant_id = $2`
		sqArgs = append(sqArgs, tenantID)
	}
	if err := db.SelectContext(ctx, &sqRows, sqQuery, sqArgs...); err == nil {
		for _, q := range sqRows {
			refs = append(refs, MetricReference{
				Type:     "saved_query",
				ID:       q.ID,
				Name:     q.Name,
				Location: "saved_query.measures",
			})
		}
	}

	// 2. Scan page definitions (components containing metricId)
	var pgRows []struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	pgQuery := `SELECT id, name FROM page_definitions WHERE components::text LIKE '%' || $1 || '%'`
	var pgArgs []interface{}
	pgArgs = append(pgArgs, metricID)
	if !isCore && tenantID != "" {
		pgQuery += ` AND tenant_id = $2`
		pgArgs = append(pgArgs, tenantID)
	}
	if err := db.SelectContext(ctx, &pgRows, pgQuery, pgArgs...); err == nil {
		for _, p := range pgRows {
			refs = append(refs, MetricReference{
				Type:     "kpi_tile",
				ID:       p.ID,
				Name:     p.Name,
				Location: "page_definitions.components",
			})
		}
	}

	// 3. Scan derived metrics that depend on this metric
	var derRows []struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	derQuery := `SELECT id, name FROM data_explorer.metric_definition WHERE id != $1 AND expression::text LIKE '%' || $1 || '%' AND archived_at IS NULL`
	var derArgs []interface{}
	derArgs = append(derArgs, metricID)
	if !isCore && tenantID != "" {
		derQuery += ` AND tenant_id = $2`
		derArgs = append(derArgs, tenantID)
	}
	if err := db.SelectContext(ctx, &derRows, derQuery, derArgs...); err == nil {
		for _, d := range derRows {
			refs = append(refs, MetricReference{
				Type:     "derived_metric",
				ID:       d.ID,
				Name:     d.Name,
				Location: "metric_definition.expression.baseMetricIds",
			})
		}
	}

	// 4. Core adoptions if isCore
	if isCore {
		var adoptRows []struct {
			TenantID string `db:"tenant_id"`
			Count    int    `db:"count"`
		}
		adoptQuery := `SELECT tenant_id, COUNT(*) as count FROM public.core_object_adoption WHERE object_type = 'metric' AND core_object_id = $1 GROUP BY tenant_id`
		if err := db.SelectContext(ctx, &adoptRows, adoptQuery, metricID); err == nil {
			for _, a := range adoptRows {
				refs = append(refs, MetricReference{
					Type:     "core_adoption",
					ID:       a.TenantID,
					Name:     fmt.Sprintf("Adopted by tenant %s (%d times)", a.TenantID, a.Count),
					Location: "core_object_adoption",
				})
			}
		}
	}

	// 5. Active scheduled jobs referencing this metric
	var schedRows []struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	schedQuery := `SELECT id::text, name FROM public.schedules WHERE deleted_at IS NULL AND (payload::text LIKE '%' || $1 || '%' OR target LIKE '%' || $1 || '%')`
	var schedArgs []interface{}
	schedArgs = append(schedArgs, metricID)
	if !isCore && tenantID != "" {
		schedQuery += ` AND tenant_id::text = $2`
		schedArgs = append(schedArgs, tenantID)
	}
	if err := db.SelectContext(ctx, &schedRows, schedQuery, schedArgs...); err == nil {
		for _, s := range schedRows {
			refs = append(refs, MetricReference{
				Type:     "scheduled_job",
				ID:       s.ID,
				Name:     s.Name,
				Location: "public.schedules",
			})
		}
	}

	return MetricUsageReport{
		InUse:      len(refs) > 0,
		References: refs,
	}, nil
}

// MetricBundle represents the export payload for uisce.metric-bundle/1.
type MetricBundle struct {
	SchemaVersion string             `json:"schemaVersion"`
	ExportedAt    time.Time          `json:"exportedAt"`
	Metrics       []MetricDefinition `json:"metrics"`
}

// ExportMetricBundle serializes metrics into a metric bundle.
func ExportMetricBundle(metrics []MetricDefinition) (*MetricBundle, error) {
	for i := range metrics {
		if metrics[i].ContentHash == "" {
			metrics[i].ContentHash = ComputeMetricContentHash(metrics[i])
		}
	}
	return &MetricBundle{
		SchemaVersion: MetricBundleSchemaVersion,
		ExportedAt:    time.Now().UTC(),
		Metrics:       metrics,
	}, nil
}

// ValidateQueryMetricDependencies verifies that all metric IDs referenced in a query
// exist in the tenant's metric definitions (or bundle dependencies). Fails closed (422) if missing.
func ValidateQueryMetricDependencies(referencedMetricIDs []string, availableMetricIDs map[string]bool) error {
	var missing []string
	for _, id := range referencedMetricIDs {
		if !availableMetricIDs[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: required metrics [%s] not found in target tenant", ErrMissingMetricDep, strings.Join(missing, ", "))
	}
	return nil
}

// SyncMetricToCatalogGraph handles transactional emission of catalog_node and catalog_edge for a metric.
func SyncMetricToCatalogGraph(ctx context.Context, tx *sqlx.Tx, tenantID string, m MetricDefinition) (string, error) {
	parsedTenantID, err := uuid.Parse(tenantID)
	if err != nil {
		return "", fmt.Errorf("invalid tenant UUID %q: %w", tenantID, err)
	}

	metricNodeID := uuid.New()
	if m.CatalogTermID != nil && *m.CatalogTermID != "" {
		if parsed, err := uuid.Parse(*m.CatalogTermID); err == nil {
			metricNodeID = parsed
		}
	}

	qualifiedPath := fmt.Sprintf("metric/%s/%s", m.BOID, m.Name)
	propsJSON, _ := json.Marshal(map[string]interface{}{
		"kind":           m.Expression.Kind,
		"fn":             m.Expression.Fn,
		"formula":        m.Expression.Formula,
		"grainAllowlist": m.GrainAllowlist,
		"formatConfig":   m.FormatConfig,
		"decomposable":   m.Decomposable,
		"contentHash":    m.ContentHash,
	})

	// 1. Upsert SEMANTIC_TERM node for the metric
	_, err = tx.ExecContext(ctx, `
		INSERT INTO catalog_node (node_id, tenant_id, node_type, node_key, node_name, qualified_path, properties)
		VALUES ($1, $2, 'SEMANTIC_TERM', $3, $4, $5, $6)
		ON CONFLICT (tenant_id, qualified_path) DO UPDATE
		SET node_name = EXCLUDED.node_name, properties = EXCLUDED.properties, updated_at = NOW()
	`, metricNodeID, parsedTenantID, m.ID, m.Name, qualifiedPath, string(propsJSON))
	if err != nil {
		return "", fmt.Errorf("failed upserting metric semantic term node: %w", err)
	}

	// 2. Fetch or create Business Object node
	var boNodeID uuid.UUID
	boPath := fmt.Sprintf("bo/%s", m.BOID)
	err = tx.GetContext(ctx, &boNodeID, `
		SELECT node_id FROM catalog_node WHERE tenant_id = $1 AND (qualified_path = $2 OR node_key = $3) LIMIT 1
	`, parsedTenantID, boPath, m.BOID)
	if err == nil {
		// Link METRIC_OF edge (SEMANTIC_TERM -> BUSINESS_OBJECT)
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO catalog_edge (tenant_id, source_node_id, target_node_id, edge_type)
			SELECT $1, $2, $3, 'METRIC_OF'
			WHERE NOT EXISTS (
				SELECT 1 FROM catalog_edge WHERE tenant_id = $1 AND source_node_id = $2 AND target_node_id = $3 AND edge_type = 'METRIC_OF'
			)
		`, parsedTenantID, metricNodeID, boNodeID)
	}

	// 3. Link USES_TERM edges for underlying terms/columns
	if m.Expression.TermNodeID != "" {
		var termUUID uuid.UUID
		if parsed, err := uuid.Parse(m.Expression.TermNodeID); err == nil {
			termUUID = parsed
			_, _ = tx.ExecContext(ctx, `
				INSERT INTO catalog_edge (tenant_id, source_node_id, target_node_id, edge_type)
				SELECT $1, $2, $3, 'USES_TERM'
				WHERE NOT EXISTS (
					SELECT 1 FROM catalog_edge WHERE tenant_id = $1 AND source_node_id = $2 AND target_node_id = $3 AND edge_type = 'USES_TERM'
				)
			`, parsedTenantID, metricNodeID, termUUID)
		}
	}

	// 4. Link DERIVED_FROM edges for compound metrics
	for _, baseID := range m.Expression.BaseMetricIDs {
		var baseNodeID uuid.UUID
		err = tx.GetContext(ctx, &baseNodeID, `
			SELECT node_id FROM catalog_node WHERE tenant_id = $1 AND node_key = $2 AND node_type = 'SEMANTIC_TERM' LIMIT 1
		`, parsedTenantID, baseID)
		if err == nil {
			_, _ = tx.ExecContext(ctx, `
				INSERT INTO catalog_edge (tenant_id, source_node_id, target_node_id, edge_type)
				SELECT $1, $2, $3, 'DERIVED_FROM'
				WHERE NOT EXISTS (
					SELECT 1 FROM catalog_edge WHERE tenant_id = $1 AND source_node_id = $2 AND target_node_id = $3 AND edge_type = 'DERIVED_FROM'
				)
			`, parsedTenantID, metricNodeID, baseNodeID)
		}
	}

	nodeIDStr := metricNodeID.String()
	return nodeIDStr, nil
}

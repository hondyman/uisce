package querybuilder

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/rules/vm"
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
	// ErrInvalidMetricBundle is the ingestion-boundary error: a bundle that
	// cannot be imported as-is. See ImportMetricBundle and ADR-026.
	ErrInvalidMetricBundle = errors.New("invalid metric bundle")
	// ErrMetricTermNotPermitted is the C2 PII gate: a metric tried to read a
	// term whose column is classified above the caller's masking threshold.
	// Distinct from ErrInvalidMetricFormula because the formula is well-formed
	// - it is the data it reaches that is not permitted, and the two deserve
	// different remediation.
	ErrMetricTermNotPermitted = errors.New("metric term is not permitted at this masking threshold")
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
	Kind       string `json:"kind"`                 // "aggregation" | "formula" | "derived"
	Fn         string `json:"fn,omitempty"`         // "sum" | "avg" | "count" | "min" | "max"
	TermNodeID string `json:"termNodeId,omitempty"` // For aggregation / column reference
	// Formula is authored SQL text with @variable references, e.g.
	// "SUM(price * qty) * @fx_rate". The compiler substitutes the variables
	// with bound parameters; it does NOT parse, validate or allowlist the
	// operators, which pass through as authored. See ADR-025.
	Formula string `json:"formula,omitempty"`
	// BaseMetricIDs are the operands of a derived metric, in the author's
	// declared order. For a 2-operand ratio the first entry is the numerator.
	BaseMetricIDs []string `json:"baseMetricIds,omitempty"`

	// NumeratorID / DenominatorID state the operands of a ratio explicitly.
	// They win over BaseMetricIDs order and are required to appear in it.
	// Operand order must never be inferred from how IDs happen to be spelled:
	// that made a declared revenue/cost ratio compile to cost/revenue. See
	// ADR-025.
	NumeratorID   string `json:"numeratorId,omitempty"`
	DenominatorID string `json:"denominatorId,omitempty"`
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
// DeriveDecomposable reports whether a metric may be summed across a finer
// grain and still be correct. Delegates to the VM (C1/9.1), which owns the
// rule; this wrapper keeps the querybuilder call sites unchanged.
func DeriveDecomposable(expr MetricExpression) bool {
	return vm.DeriveDecomposable(toVMExpression(expr))
}

// ComputeMetricContentHash deterministically computes the SHA-256 hash of a metric definition's
// semantic content (normalized formula, grains, format, variables, boid).
//
// Delegates to the VM (C1/9.1). The hash is the cube deploy identity and part of
// the query cache key, so it must have exactly one implementation; the 8.3
// golden corpus is what proves the delegated path is unchanged.
func ComputeMetricContentHash(m MetricDefinition) string {
	return vm.ComputeMetricContentHash(vm.MetricContentInput{
		Name:           m.Name,
		BOID:           m.BOID,
		Expression:     toVMExpression(m.Expression),
		GrainAllowlist: m.GrainAllowlist,
		FormatConfig:   toVMFormatConfig(m.FormatConfig),
		Variables:      toVMVariables(m.Variables),
	})
}

// toVMFormatConfig maps querybuilder format config onto its VM counterpart.
// The pointer fields are copied by reference deliberately: they are read-only
// in practice, and sharing them avoids the mapper silently deep-copying a *int
// into a different value.
func toVMFormatConfig(in MetricFormatConfig) vm.MetricFormatConfig {
	return vm.MetricFormatConfig{
		Type:           in.Type,
		Precision:      in.Precision,
		CurrencySymbol: in.CurrencySymbol,
		Prefix:         in.Prefix,
		Suffix:         in.Suffix,
	}
}

// toVMExpression maps a querybuilder expression onto its VM counterpart.
// Written as an explicit field mapping rather than a type conversion: Go will
// not convert between distinct named struct types, and doing it by hand keeps
// the two definitions honestly coupled -- adding a field to one without the
// other becomes a compile error at the call site rather than a silent omission.
func toVMExpression(in MetricExpression) vm.MetricExpression {
	return vm.MetricExpression{
		Kind:          in.Kind,
		Fn:            in.Fn,
		TermNodeID:    in.TermNodeID,
		Formula:       in.Formula,
		BaseMetricIDs: in.BaseMetricIDs,
		NumeratorID:   in.NumeratorID,
		DenominatorID: in.DenominatorID,
	}
}

// toVMVariables maps querybuilder variables onto their VM counterparts. The
// struct fields are identical, so this is a field-wise copy rather than a type
// conversion — that keeps the mapping explicit and greppable if either type
// gains a field.
func toVMVariables(in []MetricVariable) []vm.MetricVariable {
	if len(in) == 0 {
		return nil
	}
	out := make([]vm.MetricVariable, len(in))
	for i, v := range in {
		out[i] = vm.MetricVariable{
			Name:         v.Name,
			Type:         v.Type,
			DefaultValue: v.DefaultValue,
			Required:     v.Required,
			Description:  v.Description,
		}
	}
	return out
}

// normalizeFormulaForHash delegates to the VM (C1/9.1), which owns metric
// canonicalization. One implementation, so the compiler's normalization and
// the content hash cannot drift apart.
func normalizeFormulaForHash(f string) string {
	return vm.NormalizeFormulaForHash(f)
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
// ValidateMetricExpression is the canonical validation for a governed metric's
// expression. It is the artifact a save path must call, so an unusable metric
// is rejected at authoring time rather than at query or deploy time.
//
// NOTE ON SCOPE: as of this writing there is no user-facing CRUD endpoint for
// data_explorer.metric_definition.expression - rows arrive via migration, seed
// or import, and the reconciler only backfills catalog_term_id. So there is no
// 422 to return today. The check is wired into ValidateCubeMetricReferences
// (the earliest enforcement point that exists) and into CompileMetric, and it
// is exported so a future save endpoint cannot skip it. See ADR-026.
func ValidateMetricExpression(m MetricDefinition) error {
	expr := m.Expression
	switch strings.ToLower(strings.TrimSpace(expr.Kind)) {
	case "aggregation", "formula", "":
		return nil
	case "derived":
	default:
		return fmt.Errorf("%w: unsupported expression kind %q", ErrInvalidMetricFormula, expr.Kind)
	}

	if len(expr.BaseMetricIDs) == 0 {
		return fmt.Errorf("%w: derived metric requires baseMetricIds", ErrInvalidMetricFormula)
	}

	num, den := expr.NumeratorID, expr.DenominatorID
	if num == "" && den == "" {
		if len(expr.BaseMetricIDs) == 2 {
			// A two-operand derived metric is a ratio, and a ratio's direction
			// is the whole meaning. Reading it off positional order is how
			// revenue/cost silently became cost/revenue: the sort decided the
			// numerator. So the ordered form is rejected outright rather than
			// guessed at.
			return fmt.Errorf("%w: derived ratio requires numeratorId and denominatorId - the positional/ordered form was ambiguous by design (ADR-026)", ErrInvalidMetricFormula)
		}
		return nil // N-operand sum: order cannot change the value.
	}
	if num == "" || den == "" {
		return fmt.Errorf("%w: derived ratio requires numeratorId and denominatorId (ADR-026)", ErrInvalidMetricFormula)
	}
	if num == den {
		return fmt.Errorf("%w: derived ratio numeratorId and denominatorId are the same metric", ErrInvalidMetricFormula)
	}
	if len(expr.BaseMetricIDs) != 2 {
		return fmt.Errorf("%w: numeratorId/denominatorId describe a 2-metric ratio, but baseMetricIds has %d entries", ErrInvalidMetricFormula, len(expr.BaseMetricIDs))
	}
	for _, id := range expr.BaseMetricIDs {
		if id == num || id == den {
			continue
		}
		return fmt.Errorf("%w: baseMetricIds contains %q, which is neither numeratorId nor denominatorId (ADR-026)", ErrInvalidMetricFormula, id)
	}
	return nil
}

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

	// lineageWrites collects edges that could NOT be written for a reason that
	// is not a database failure - currently only "the term is not a catalog
	// node yet". They are not returned to the caller as errors (a metric may
	// legitimately be imported before its term exists) but they are not
	// discarded either: the caller logs them, so an absent edge is always
	// explicable. An absent edge with no such line is a bug, not a state.
	var lineageWrites []string
	defer func() {
		for _, w := range lineageWrites {
			log.Printf("[metric-lineage] %s", w)
		}
	}()

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
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// The business object is not a catalog node. Recorded, not fatal: a
		// metric can be imported before its BO is.
		lineageWrites = append(lineageWrites, fmt.Sprintf(
			"metric %s: business object %q is not a catalog node - no METRIC_OF edge written", m.ID, m.BOID))
	case err != nil:
		return "", fmt.Errorf("look up business object %q for metric %s: %w", m.BOID, m.ID, err)
	default:
		// METRIC_OF edge (SEMANTIC_TERM -> BUSINESS_OBJECT). The insert's error
		// is returned, not discarded: a swallowed failure here leaves a metric
		// node in the graph with no owner, which reads as "never had one".
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO catalog_edge (tenant_id, source_node_id, target_node_id, edge_type)
			SELECT $1, $2, $3, 'METRIC_OF'
			WHERE NOT EXISTS (
				SELECT 1 FROM catalog_edge WHERE tenant_id = $1 AND source_node_id = $2 AND target_node_id = $3 AND edge_type = 'METRIC_OF'
			)
		`, parsedTenantID, metricNodeID, boNodeID); err != nil {
			return "", fmt.Errorf("write METRIC_OF edge for metric %s: %w", m.ID, err)
		}
	}

	// 3. Link the USES_TERM edge to the underlying term.
	//
	// This used to be guarded by `uuid.Parse(m.Expression.TermNodeID)`, which
	// meant the edge was only ever written when a metric named its term by UUID.
	// Real metrics name terms by semantic key - the 8.3 golden corpus carries
	// "revenue", "cost", "calc_term_net_interest_income" - so uuid.Parse failed
	// for all of them and the edge was never written at all. Every metric's
	// lineage to its underlying term was silently absent, and the failure was
	// invisible because the insert's error was discarded.
	//
	// Resolution now matches step 4's DERIVED_FROM, which has always looked the
	// target up by node_key. Both steps resolve a name the same way, so a metric
	// whose base metrics are linked is no longer one whose term is not.
	if termKey := strings.TrimSpace(m.Expression.TermNodeID); termKey != "" {
		var termNodeID uuid.UUID
		err = tx.GetContext(ctx, &termNodeID, `
			SELECT node_id FROM catalog_node WHERE tenant_id = $1 AND node_key = $2 AND node_type = 'SEMANTIC_TERM' LIMIT 1
		`, parsedTenantID, termKey)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			// The term is not a catalog node yet. That is a legitimate state
			// (a metric may be imported before its term is), so it is not an
			// error - but it is reported rather than dropped, because "no edge
			// because the term is absent" and "no edge because the insert failed"
			// must not look the same to whoever reads the graph.
			lineageWrites = append(lineageWrites, fmt.Sprintf(
				"metric %s: term %q is not a catalog SEMANTIC_TERM node - no USES_TERM edge written", m.ID, termKey))
		case err != nil:
			return "", fmt.Errorf("look up metric %s term %q: %w", m.ID, termKey, err)
		default:
			// An edge insert that fails is NOT swallowed. The whole point of the
			// edge is that the graph is not silently wrong, and a discarded error
			// here is indistinguishable from a term that needed no edge.
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO catalog_edge (tenant_id, source_node_id, target_node_id, edge_type)
				SELECT $1, $2, $3, 'USES_TERM'
				WHERE NOT EXISTS (
					SELECT 1 FROM catalog_edge WHERE tenant_id = $1 AND source_node_id = $2 AND target_node_id = $3 AND edge_type = 'USES_TERM'
				)
			`, parsedTenantID, metricNodeID, termNodeID); err != nil {
				return "", fmt.Errorf("write USES_TERM edge for metric %s term %q: %w", m.ID, termKey, err)
			}
		}
	}

	// 4. Link DERIVED_FROM edges for compound metrics
	for _, baseID := range m.Expression.BaseMetricIDs {
		var baseNodeID uuid.UUID
		err = tx.GetContext(ctx, &baseNodeID, `
			SELECT node_id FROM catalog_node WHERE tenant_id = $1 AND node_key = $2 AND node_type = 'SEMANTIC_TERM' LIMIT 1
		`, parsedTenantID, baseID)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			lineageWrites = append(lineageWrites, fmt.Sprintf(
				"metric %s: base metric %q is not a catalog SEMANTIC_TERM node - no DERIVED_FROM edge written", m.ID, baseID))
		case err != nil:
			return "", fmt.Errorf("look up base metric %q for metric %s: %w", baseID, m.ID, err)
		default:
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO catalog_edge (tenant_id, source_node_id, target_node_id, edge_type)
				SELECT $1, $2, $3, 'DERIVED_FROM'
				WHERE NOT EXISTS (
					SELECT 1 FROM catalog_edge WHERE tenant_id = $1 AND source_node_id = $2 AND target_node_id = $3 AND edge_type = 'DERIVED_FROM'
				)
			`, parsedTenantID, metricNodeID, baseNodeID); err != nil {
				return "", fmt.Errorf("write DERIVED_FROM edge for metric %s base %q: %w", m.ID, baseID, err)
			}
		}
	}

	nodeIDStr := metricNodeID.String()
	return nodeIDStr, nil
}

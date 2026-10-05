package querybuilder

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/goldcopy"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// CubeMaterializeRequest is one grain deploy/refresh (hot + cold dual-commit).
type CubeMaterializeRequest struct {
	TenantID    string   `json:"tenant_id"`
	CubeID      string   `json:"cube_id"`
	Grain       []string `json:"grain"`
	SourceTable string   `json:"source_table,omitempty"` // optional override; else BO driver table
	AttemptID   string   `json:"attempt_id,omitempty"`
	Force       bool     `json:"force,omitempty"` // skip content_hash noop
	// FederationKeySamples optional CUBE-2.1 orphan-gate fixtures. Required
	// (fail-closed) when the cube declares federation; live source sampling
	// remains a later datapipeline concern.
	FederationKeySamples []FederationKeySample `json:"federation_key_samples,omitempty"`
}

// CubeMaterializePlan is the validated, compiled dual-tier plan for one grain.
type CubeMaterializePlan struct {
	TenantID            string            `json:"tenant_id"`
	CubeID              string            `json:"cube_id"`
	CubeName            string            `json:"cube_name"`
	ContractVersion     int               `json:"contract_version"`
	ContentHash         string            `json:"content_hash"`
	BOID                string            `json:"bo_id"`
	IsCore              bool              `json:"is_core"`
	Grain               []string          `json:"grain"`
	GrainHash           string            `json:"grain_hash"`
	NodeID              string            `json:"node_id"`
	NodeName            string            `json:"node_name"`
	AttemptID           string            `json:"attempt_id"`
	SourceTable         string            `json:"source_table"`
	TargetDatabase      string            `json:"target_database"`
	DDL                 string            `json:"ddl"`
	DDLContentHash      string            `json:"ddl_content_hash"`
	MaterializationName string            `json:"materialization_name"`
	IcebergCatalog      string            `json:"iceberg_catalog"`
	IcebergDatabase     string            `json:"iceberg_database"`
	IcebergTable        string            `json:"iceberg_table"` // catalog.db.table
	Noop                bool              `json:"noop"`
	NoopReason          string            `json:"noop_reason,omitempty"`
	MeasureColumns      map[string]string `json:"measure_columns,omitempty"`
	GroupByColumns      []string          `json:"group_by_columns,omitempty"`
	LifecycleBefore     string            `json:"lifecycle_before,omitempty"`
}

// CubeMaterializeHotResult is the outcome of applying the hot StarRocks step.
type CubeMaterializeHotResult struct {
	MaterializationName string    `json:"materialization_name"`
	TargetDatabase      string    `json:"target_database"`
	AppliedDDL          bool      `json:"applied_ddl"`
	RowCount            int64     `json:"row_count"`
	CommittedAt         time.Time `json:"committed_at"`
}

// CubeMaterializeColdResult is the outcome of the Iceberg cold commit (CUBE-1.3).
type CubeMaterializeColdResult struct {
	IcebergTable string    `json:"iceberg_table"`
	Applied      bool      `json:"applied"`
	RowCount     int64     `json:"row_count"`
	CommittedAt  time.Time `json:"committed_at"`
}

// CubeColdWriter writes the cold Iceberg tier from the hot StarRocks object.
// The default implementation uses INSERT/CTAS through the StarRocks FE.
type CubeColdWriter interface {
	ApplyCold(ctx context.Context, plan *CubeMaterializePlan, hot *CubeMaterializeHotResult) (*CubeMaterializeColdResult, error)
}

// CubeMaterializer validates, plans, and applies cube grains to StarRocks + Iceberg.
// Federated cubes (CUBE-2.2) compile JOIN SQL via FederationBindingResolver.
type CubeMaterializer struct {
	db          *sqlx.DB
	registry    *CubeMaterializationRegistry
	starrocksDB *sql.DB
	ddl         *CubeDDLGenerator
	cold        CubeColdWriter
	icebergCat  string
	fedResolver FederationBindingResolver
}

// NewCubeMaterializer wires Postgres control plane + optional StarRocks hot/cold plane.
func NewCubeMaterializer(db *sqlx.DB, starrocksDB *sql.DB) *CubeMaterializer {
	var lifecycle *analytics.PreAggLifecycleService
	if db != nil {
		lifecycle = analytics.NewPreAggLifecycleService(db)
	}
	gen := NewCubeDDLGenerator("starrocks")
	// Deploy path: authoring already validated metrics; gate must be present
	// (CubeDDLGenerator fails closed without one) but does not re-litigate PII.
	gen.SetTermGate(func(termNodeID string, field *boresolver.BOField) error { return nil })
	m := &CubeMaterializer{
		db:          db,
		registry:    NewCubeMaterializationRegistry(db, lifecycle),
		starrocksDB: starrocksDB,
		ddl:         gen,
		icebergCat:  defaultIcebergCatalog(),
	}
	m.cold = &starRocksColdWriter{m: m}
	return m
}

// SetColdWriter replaces the Iceberg writer (tests inject a failing writer).
func (m *CubeMaterializer) SetColdWriter(w CubeColdWriter) {
	if m != nil {
		m.cold = w
	}
}

// SetFederationResolver replaces the term→column binding resolver (tests inject a map fixture).
func (m *CubeMaterializer) SetFederationResolver(r FederationBindingResolver) {
	if m != nil {
		m.fedResolver = r
	}
}

func (m *CubeMaterializer) federationResolver() FederationBindingResolver {
	if m != nil && m.fedResolver != nil {
		return m.fedResolver
	}
	if m != nil && m.db != nil {
		return dbFederationBindingResolver{db: m.db}
	}
	return nil
}

// ValidateAndPlan loads the cube, ensures the grain catalog node, compiles DDL,
// and reports content_hash noop when the Active grain already matches.
func (m *CubeMaterializer) ValidateAndPlan(ctx context.Context, req CubeMaterializeRequest) (*CubeMaterializePlan, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("cube materializer: database not configured")
	}
	tenantID := strings.TrimSpace(req.TenantID)
	cubeID := strings.TrimSpace(req.CubeID)
	grain := normalizeGrain(req.Grain)
	if tenantID == "" || cubeID == "" {
		return nil, fmt.Errorf("cube materializer: tenant_id and cube_id are required")
	}
	if len(grain) == 0 {
		return nil, fmt.Errorf("cube materializer: grain is required")
	}

	cube, err := m.loadCube(ctx, tenantID, cubeID)
	if err != nil {
		return nil, err
	}
	if err := ValidateCubeStructural(*cube); err != nil {
		return nil, err
	}

	var fedJoin *CompiledFederationJoinSQL
	if !cube.Federation.Empty() {
		// CUBE-2.1 orphan/transform gate (fail-closed without samples).
		metricIDSet := map[string]bool{}
		if _, _, fedErr := EvaluateFederationPlan(cube.Federation, metricIDSet, req.FederationKeySamples, true); fedErr != nil {
			return nil, fmt.Errorf("cube materializer: federation gate: %w", fedErr)
		}
		// CUBE-2.2: compile StarRocks JOIN FROM via semantic term bindings.
		resolver := m.federationResolver()
		compiled, fedErr := CompileFederationJoinSQL(ctx, tenantID, cube.Federation, resolver)
		if fedErr != nil {
			return nil, fmt.Errorf("cube materializer: federation join compile: %w", fedErr)
		}
		if err := compiled.EnrichGrainTerms(ctx, tenantID, cube.BOID, grain, resolver); err != nil {
			return nil, fmt.Errorf("cube materializer: federation grain bind: %w", err)
		}
		fedJoin = compiled
	}
	if !grainCovered(cube.Grains, grain) {
		return nil, fmt.Errorf("cube materializer: grain %v is not declared on cube %s", grain, cube.ID)
	}

	metrics, err := m.loadMetrics(ctx, tenantID, cube.MetricIDs)
	if err != nil {
		return nil, fmt.Errorf("load metrics: %w", err)
	}
	if err := ValidateCubeMetricReferences(*cube, metrics); err != nil {
		return nil, err
	}

	nodes, err := m.registry.EnsureGrainNodes(ctx, *cube)
	if err != nil {
		return nil, err
	}
	var node *CubeMaterializationNode
	wantHash := GrainHash(grain)
	for i := range nodes {
		if nodes[i].GrainHash == wantHash {
			node = &nodes[i]
			break
		}
	}
	if node == nil {
		return nil, fmt.Errorf("cube materializer: grain node missing after ensure for %v", grain)
	}

	sourceTable := strings.TrimSpace(req.SourceTable)
	var dimExprs map[string]string
	if fedJoin != nil {
		sourceTable = fedJoin.FromSQL
		dimExprs = fedJoin.DimExprsForGrain(grain, cube.BOID)
	} else if sourceTable == "" {
		sourceTable, err = m.resolveSourceTable(ctx, tenantID, cube.BOID)
		if err != nil {
			return nil, err
		}
	}

	var generated *GeneratedCubeDDL
	if fedJoin != nil {
		generated, err = m.ddl.GenerateFederatedCubeMaterializationDDL(
			tenantID, cube.IsCore, *cube, grain, sourceTable, dimExprs, metrics, nil,
		)
	} else {
		generated, err = m.ddl.GenerateCubeMaterializationDDL(
			tenantID, cube.IsCore, *cube, grain, sourceTable, metrics, nil,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("compile cube DDL: %w", err)
	}

	attemptID := strings.TrimSpace(req.AttemptID)
	if attemptID == "" {
		attemptID = uuid.New().String()
	}

	targetDB := fmt.Sprintf("tenant_%s", sanitizeIdentifier(tenantID))
	if cube.IsCore {
		targetDB = "gold"
	}

	icebergDB := "cubes"
	if cube.IsCore {
		icebergDB = "cubes_gold"
	}
	icebergTable := fmt.Sprintf("%s.%s.%s",
		sanitizeIdentifier(m.icebergCat),
		sanitizeIdentifier(icebergDB),
		sanitizeIdentifier(generated.MaterializationName),
	)

	plan := &CubeMaterializePlan{
		TenantID:            tenantID,
		CubeID:              cube.ID,
		CubeName:            cube.Name,
		ContractVersion:     cube.ContractVersion,
		ContentHash:         cube.ContentHash,
		BOID:                cube.BOID,
		IsCore:              cube.IsCore,
		Grain:               grain,
		GrainHash:           node.GrainHash,
		NodeID:              node.ID.String(),
		NodeName:            node.NodeName,
		AttemptID:           attemptID,
		SourceTable:         sourceTable,
		TargetDatabase:      targetDB,
		DDL:                 generated.DDL,
		DDLContentHash:      generated.ContentHash,
		MaterializationName: generated.MaterializationName,
		IcebergCatalog:      sanitizeIdentifier(m.icebergCat),
		IcebergDatabase:     sanitizeIdentifier(icebergDB),
		IcebergTable:        icebergTable,
		MeasureColumns:      generated.MeasureColumns,
		GroupByColumns:      generated.GroupByColumns,
		LifecycleBefore:     node.Properties.LifecycleStatus,
	}

	if !req.Force &&
		node.Properties.LifecycleStatus == models.LifecycleActive &&
		node.Properties.DualCommitWatermark != nil &&
		strings.TrimSpace(node.Properties.CubeContentHash) != "" &&
		node.Properties.CubeContentHash == cube.ContentHash {
		plan.Noop = true
		plan.NoopReason = "content_hash unchanged and grain already dual-committed Active"
	}
	return plan, nil
}

// BeginAttempt marks the grain node Materializing for plan.AttemptID.
func (m *CubeMaterializer) BeginAttempt(ctx context.Context, plan *CubeMaterializePlan) error {
	if plan == nil {
		return fmt.Errorf("cube materializer: plan is required")
	}
	nodeID, err := uuid.Parse(plan.NodeID)
	if err != nil {
		return fmt.Errorf("cube materializer: invalid node_id: %w", err)
	}
	return m.registry.BeginAttempt(ctx, nodeID, plan.AttemptID)
}

// ApplyHot provisions the StarRocks database and applies CREATE MATERIALIZED VIEW
// (which also loads from the source table for single-BO cubes).
func (m *CubeMaterializer) ApplyHot(ctx context.Context, plan *CubeMaterializePlan) (*CubeMaterializeHotResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("cube materializer: plan is required")
	}
	if m.starrocksDB == nil {
		return nil, fmt.Errorf("starrocks connection is not available (check STARROCKS_HOST/PORT/USER/PASSWORD)")
	}

	if _, err := m.starrocksDB.ExecContext(ctx,
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", quoteStarRocksIdent(plan.TargetDatabase)),
	); err != nil {
		return nil, fmt.Errorf("ensure starrocks database %q: %w", plan.TargetDatabase, err)
	}

	qualified := fmt.Sprintf("%s.%s",
		quoteStarRocksIdent(plan.TargetDatabase),
		quoteStarRocksIdent(plan.MaterializationName),
	)
	ddl := qualifyCubeMVName(plan.DDL, plan.MaterializationName, qualified)

	// Replace an existing MV when re-deploying a changed contract.
	_, _ = m.starrocksDB.ExecContext(ctx, fmt.Sprintf("DROP MATERIALIZED VIEW IF EXISTS %s", qualified))

	if _, err := m.starrocksDB.ExecContext(ctx, ddl); err != nil {
		return nil, fmt.Errorf("apply cube materialization DDL: %w", err)
	}

	var rowCount int64
	countSQL := fmt.Sprintf(
		"SELECT IFNULL(table_rows, 0) FROM information_schema.tables WHERE table_schema = %s AND table_name = %s",
		quoteStarRocksString(plan.TargetDatabase),
		quoteStarRocksString(plan.MaterializationName),
	)
	_ = m.starrocksDB.QueryRowContext(ctx, countSQL).Scan(&rowCount)

	return &CubeMaterializeHotResult{
		MaterializationName: plan.MaterializationName,
		TargetDatabase:      plan.TargetDatabase,
		AppliedDDL:          true,
		RowCount:            rowCount,
		CommittedAt:         time.Now().UTC(),
	}, nil
}

// ApplyCold commits the Iceberg cold tier from the hot StarRocks object (CUBE-1.3).
func (m *CubeMaterializer) ApplyCold(ctx context.Context, plan *CubeMaterializePlan, hot *CubeMaterializeHotResult) (*CubeMaterializeColdResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("cube materializer: plan is required")
	}
	if m.cold == nil {
		return nil, fmt.Errorf("cube materializer: cold writer not configured")
	}
	return m.cold.ApplyCold(ctx, plan, hot)
}

// CompensateHot drops the attempt-scoped hot MV after a cold failure so Active
// is never reached with only one tier present (CUBE-1.3 dual-commit).
func (m *CubeMaterializer) CompensateHot(ctx context.Context, plan *CubeMaterializePlan) error {
	if plan == nil {
		return fmt.Errorf("cube materializer: plan is required")
	}
	if m.starrocksDB == nil {
		return fmt.Errorf("starrocks connection is not available (check STARROCKS_HOST/PORT/USER/PASSWORD)")
	}
	qualified := fmt.Sprintf("%s.%s",
		quoteStarRocksIdent(plan.TargetDatabase),
		quoteStarRocksIdent(plan.MaterializationName),
	)
	if _, err := m.starrocksDB.ExecContext(ctx, fmt.Sprintf("DROP MATERIALIZED VIEW IF EXISTS %s", qualified)); err != nil {
		return fmt.Errorf("compensate hot drop %s: %w", qualified, err)
	}
	return nil
}

// CompleteDualCommit marks Active + DualCommitWatermark only after hot and cold both OK.
func (m *CubeMaterializer) CompleteDualCommit(
	ctx context.Context,
	plan *CubeMaterializePlan,
	hot *CubeMaterializeHotResult,
	cold *CubeMaterializeColdResult,
) error {
	if plan == nil {
		return fmt.Errorf("cube materializer: plan is required")
	}
	if hot == nil || cold == nil || !cold.Applied {
		return fmt.Errorf("cube materializer: dual-commit requires successful hot and cold results")
	}
	nodeID, err := uuid.Parse(plan.NodeID)
	if err != nil {
		return fmt.Errorf("cube materializer: invalid node_id: %w", err)
	}
	stats := &models.PreAggStats{RowCount: hot.RowCount}
	if cold.RowCount > 0 {
		stats.RowCount = cold.RowCount
	}
	meta := analytics.DualCommitMeta{
		IcebergTable:    cold.IcebergTable,
		HotCommittedAt:  hot.CommittedAt,
		ColdCommittedAt: cold.CommittedAt,
	}
	return m.registry.CompleteDualCommitAttempt(ctx, nodeID, plan.AttemptID, stats, meta)
}

// CompleteAttempt is retained for callers that only need Active+freshness.
// CubeMaterializeWorkflow uses CompleteDualCommit (CUBE-1.3).
func (m *CubeMaterializer) CompleteAttempt(ctx context.Context, plan *CubeMaterializePlan, hot *CubeMaterializeHotResult) error {
	if plan == nil {
		return fmt.Errorf("cube materializer: plan is required")
	}
	nodeID, err := uuid.Parse(plan.NodeID)
	if err != nil {
		return fmt.Errorf("cube materializer: invalid node_id: %w", err)
	}
	stats := &models.PreAggStats{}
	if hot != nil {
		stats.RowCount = hot.RowCount
	}
	return m.registry.CompleteAttempt(ctx, nodeID, plan.AttemptID, stats)
}

// FailAttempt marks Failed without advancing freshness.
func (m *CubeMaterializer) FailAttempt(ctx context.Context, plan *CubeMaterializePlan, cause error) error {
	if plan == nil {
		return fmt.Errorf("cube materializer: plan is required")
	}
	nodeID, err := uuid.Parse(plan.NodeID)
	if err != nil {
		return fmt.Errorf("cube materializer: invalid node_id: %w", err)
	}
	return m.registry.FailAttempt(ctx, nodeID, plan.AttemptID, cause)
}

func (m *CubeMaterializer) loadCube(ctx context.Context, tenantID, cubeID string) (*CubeDefinition, error) {
	var row cubeDefRow
	err := m.db.GetContext(ctx, &row, `
		SELECT `+cubeDefSelectCols+`
		FROM data_explorer.cube_definition
		WHERE id = $1 AND tenant_id = $2 AND archived_at IS NULL
	`, cubeID, tenantID)
	if err == nil {
		c := row.toCubeDefinition()
		return &c, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	gold := goldcopy.ResolveTenantID(ctx, m.db)
	if gold == uuid.Nil || gold.String() == tenantID {
		return nil, fmt.Errorf("cube %s not found for tenant %s", cubeID, tenantID)
	}
	err = m.db.GetContext(ctx, &row, `
		SELECT `+cubeDefSelectCols+`
		FROM data_explorer.cube_definition
		WHERE id = $1 AND is_core = true AND tenant_id = $2 AND archived_at IS NULL
	`, cubeID, gold.String())
	if err != nil {
		return nil, fmt.Errorf("cube %s not found for tenant %s: %w", cubeID, tenantID, err)
	}
	c := row.toCubeDefinition()
	return &c, nil
}

func (m *CubeMaterializer) loadMetrics(ctx context.Context, tenantID string, ids []string) (map[string]MetricDefinition, error) {
	out := make(map[string]MetricDefinition, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []metricDefRow
	err := m.db.SelectContext(ctx, &rows, `
		SELECT `+metricDefRowColumns+`
		FROM data_explorer.metric_definition
		WHERE (tenant_id = $1 OR is_core = true)
		  AND status = 'active' AND archived_at IS NULL
		  AND id::text = ANY($2)
	`, tenantID, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		mdef := row.toMetricDefinition()
		out[strings.ToLower(strings.TrimSpace(mdef.ID))] = mdef
	}
	return out, nil
}

func (m *CubeMaterializer) resolveSourceTable(ctx context.Context, tenantID, boID string) (string, error) {
	boID = strings.TrimSpace(boID)
	if boID == "" {
		return "", fmt.Errorf("cube materializer: bo_id is required to resolve source table")
	}
	var table string
	err := m.db.GetContext(ctx, &table, `
		SELECT COALESCE(NULLIF(TRIM(driver_table_name), ''), NULLIF(TRIM(bo_key), ''), '')
		FROM public.business_objects
		WHERE tenant_id = $1::uuid
		  AND (bo_key = $2 OR id::text = $2)
		LIMIT 1
	`, tenantID, boID)
	if err == sql.ErrNoRows {
		// Gold BO fallback by key/id without tenant filter.
		err = m.db.GetContext(ctx, &table, `
			SELECT COALESCE(NULLIF(TRIM(driver_table_name), ''), NULLIF(TRIM(bo_key), ''), '')
			FROM public.business_objects
			WHERE bo_key = $1 OR id::text = $1
			LIMIT 1
		`, boID)
	}
	if err != nil {
		return "", fmt.Errorf("resolve source table for BO %q: %w", boID, err)
	}
	if strings.TrimSpace(table) == "" {
		return "", fmt.Errorf("BO %q has no driver_table_name", boID)
	}
	return table, nil
}

func grainCovered(grains [][]string, want []string) bool {
	wantHash := GrainHash(want)
	for _, g := range grains {
		if GrainHash(g) == wantHash {
			return true
		}
	}
	return false
}

func quoteStarRocksIdent(ident string) string {
	return "`" + strings.ReplaceAll(ident, "`", "``") + "`"
}

func quoteStarRocksString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// qualifyCubeMVName rewrites CREATE MATERIALIZED VIEW <name> to a database-qualified
// target without changing the AS SELECT body produced by CubeDDLGenerator.
func qualifyCubeMVName(ddl, bareName, qualified string) string {
	needle := "CREATE MATERIALIZED VIEW " + bareName
	if strings.Contains(ddl, needle) {
		return strings.Replace(ddl, needle, "CREATE MATERIALIZED VIEW "+qualified, 1)
	}
	return ddl
}

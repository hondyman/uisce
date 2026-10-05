package querybuilder

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// CubeRoute is the outcome of a successful cube routing decision.
//
// It maps onto boresolver.CubeHitInfo at the Preview boundary; the conversion
// lives there so this package keeps the richer internal shape (including the
// content hash used for cache invalidation).
type CubeRoute struct {
	CubeID          string   `json:"cubeId"`
	CubeName        string   `json:"cubeName"`
	Materialization string   `json:"materialization"`
	ServedFrom      string   `json:"servedFrom"`
	Grain           []string `json:"grain"`
	Stale           bool     `json:"stale"`
	ContentHash     string   `json:"-"`
}

// CubeMissReason explains why no materialization was served. It exists so
// routing decisions are observable rather than opaque, and so the "no cube
// deployed" case is distinguishable from "cube exists but was not servable".
type CubeMissReason string

const (
	CubeMissNoCube         CubeMissReason = "no_cube_for_bo"
	CubeMissMetricSubset   CubeMissReason = "requested_metric_not_in_cube"
	CubeMissNoGrainMatch   CubeMissReason = "no_grain_superset"
	CubeMissNotDecomposed  CubeMissReason = "metric_not_decomposable_for_rollup"
	CubeMissABACBelowGrain CubeMissReason = "abac_below_grain"
	CubeMissStale          CubeMissReason = "stale_and_force_raw"
	CubeMissNotActive      CubeMissReason = "materialization_not_active"
	CubeMissError          CubeMissReason = "router_error"
)

// CubeRouteDecision is the full result of a routing attempt. Exactly one of
// Route / MissReason is meaningful.
type CubeRouteDecision struct {
	Route      *CubeRoute     `json:"route,omitempty"`
	MissReason CubeMissReason `json:"missReason,omitempty"`
	Detail     string         `json:"detail,omitempty"`
}

// CubeRouter selects a physical materialization for a query, or declines.
//
// uisce owns this decision explicitly (ADR-012). StarRocks native rewrite is
// diagnostics only and must never gate correctness, because the EXPLAIN
// parser is string-matching and version-fragile.
//
// Every failure path returns "no route" rather than an error: a cube is an
// optimization, and the base BO path is always correct. The caller must never
// fail a query because a cube was unavailable.
type CubeRouter struct {
	db *sqlx.DB
	// now is injectable for deterministic staleness tests.
	now func() time.Time
}

// NewCubeRouter creates a router backed by the metadata database. A nil db
// disables routing entirely (every query takes the base path).
func NewCubeRouter(db *sqlx.DB) *CubeRouter {
	return &CubeRouter{db: db, now: func() time.Time { return time.Now().UTC() }}
}

// SetClock overrides the time source (tests).
func (r *CubeRouter) SetClock(now func() time.Time) {
	if now != nil && r.db != nil {
		r.now = now
	}
}

// cubeCandidate is a cube plus the metric definitions it references.
type cubeCandidate struct {
	cube    CubeDefinition
	metrics map[string]MetricDefinition
}

// Route decides whether qd can be served from a cube materialization.
//
// abacRestrictedGrains lists dimension terms the caller is row-restricted on.
// It is a required input: the caller cannot know it implicitly, and defaulting
// it to empty would silently disable the security check (ADR-014).
func (r *CubeRouter) Route(
	ctx context.Context,
	tenantID string,
	qd *boresolver.QueryDef,
	abacRestrictedGrains []string,
) CubeRouteDecision {

	if r == nil || r.db == nil || qd == nil {
		return CubeRouteDecision{MissReason: CubeMissNoCube, Detail: "router unavailable"}
	}

	// Step 1: candidate cubes for this tenant/BO. Core cubes are included:
	// RLS already scopes the read, and an adopting tenant should be served by
	// the shared gold object.
	cube, metrics, err := r.loadCube(ctx, tenantID, qd.Context.BOID)
	if err != nil || cube == nil {
		if err != nil {
			log.Printf("[CubeRouter] load cube for bo=%s: %v", qd.Context.BOID, err)
			return CubeRouteDecision{MissReason: CubeMissError, Detail: err.Error()}
		}
		return CubeRouteDecision{MissReason: CubeMissNoCube}
	}

	// Step 2: every requested measure must be a governed metric of the cube.
	requested := requestedMeasureTerms(qd)
	if len(requested) == 0 {
		return CubeRouteDecision{MissReason: CubeMissMetricSubset,
			Detail: "query has no measures to serve from a cube"}
	}
	cubeMetrics := make(map[string]bool, len(cube.MetricIDs))
	for _, id := range cube.MetricIDs {
		cubeMetrics[strings.ToLower(strings.TrimSpace(id))] = true
	}
	for _, term := range requested {
		if !cubeMetrics[term] {
			return CubeRouteDecision{MissReason: CubeMissMetricSubset,
				Detail: fmt.Sprintf("measure %q is not a cube metric", term)}
		}
	}

	// Step 3/4: grain matching (plain set-subset, ADR-015) and decomposability.
	requestedDims := requestedDimensionTerms(qd)
	grain, matName, matNodeID, decision := r.selectGrain(*cube, tenantID, requestedDims, requested, metrics)
	if decision != nil {
		return *decision
	}

	// Step 5: ABAC. A materialization has already aggregated away dimensions
	// below its grain, so serving it to a caller restricted below that grain
	// would return data they are not cleared to see. This is a hard stop.
	canRoute, reason := EvaluateABACMVCompatibility(grain, abacRestrictedGrains)
	if !canRoute {
		return CubeRouteDecision{MissReason: CubeMissABACBelowGrain, Detail: reason}
	}

	// Step 6: freshness.
	//
	// A materialization is stale when the scheduler has marked it stale, or
	// when it has never been refreshed. It is NOT stale merely because the
	// clock has moved on: EvaluateMVWatermarkStaleness compares against a
	// source watermark, and PreAggProperties carries no watermark column, so
	// passing "now" would mark every materialization stale on sight. The
	// scheduler is the component that decides staleness as it refreshes.
	_, isStale, active, err := r.materializationState(ctx, matNodeID)
	if err != nil {
		log.Printf("[CubeRouter] materialization state %s: %v", matNodeID, err)
		return CubeRouteDecision{MissReason: CubeMissError, Detail: err.Error()}
	}
	if !active {
		return CubeRouteDecision{MissReason: CubeMissNotActive,
			Detail: "materialization is not in the active lifecycle state"}
	}
	if EvaluateStaleMVAction(isStale, cube.Materialization.StalePolicy) == "fallback_raw" {
		return CubeRouteDecision{MissReason: CubeMissStale,
			Detail: "stale materialization under force_raw_fallback policy"}
	}

	return CubeRouteDecision{Route: &CubeRoute{
		CubeID:          cube.ID,
		CubeName:        cube.Name,
		Materialization: matName,
		Grain:           grain,
		Stale:           isStale,
		ContentHash:     cube.ContentHash,
	}}
}

// selectGrain picks the best materialization for a request. It prefers the
// tightest (smallest) matching grain: a coarser materialization that still
// answers the query costs more to scan, so the tightest match is the better
// choice.
//
// tenantID participates because the physical name is tenant-scoped for
// non-gold cubes; omitting it would resolve every route to the gold object.
func (r *CubeRouter) selectGrain(
	cube CubeDefinition,
	tenantID string,
	requestedDims []string,
	requestedMeasures []string,
	metrics map[string]MetricDefinition,
) ([]string, string, string, *CubeRouteDecision) {

	requested := toSet(requestedDims)

	// Candidates: grains that are a superset of the requested dimensions.
	type candidate struct {
		grain []string
	}
	var candidates []candidate
	for _, g := range cube.Grains {
		if len(g) == 0 {
			continue
		}
		superset := true
		for d := range requested {
			if !containsFold(g, d) {
				superset = false
				break
			}
		}
		if superset {
			candidates = append(candidates, candidate{grain: g})
		}
	}
	if len(candidates) == 0 {
		d := CubeRouteDecision{MissReason: CubeMissNoGrainMatch,
			Detail: fmt.Sprintf("no declared grain covers %v", requestedDims)}
		return nil, "", "", &d
	}

	// Sort by grain size ascending (tightest first), then lexicographically so
	// the choice is deterministic when sizes tie.
	sort.SliceStable(candidates, func(i, j int) bool {
		if len(candidates[i].grain) != len(candidates[j].grain) {
			return len(candidates[i].grain) < len(candidates[j].grain)
		}
		return strings.Join(candidates[i].grain, ",") < strings.Join(candidates[j].grain, ",")
	})

	// Time-grain rollup (e.g. request month, materialization day) is only
	// allowed for distributive metrics. DeriveDecomposable already encodes
	// that: AVG, division, and derived metrics are not decomposable.
	for _, c := range candidates {
		if !r.timeRollupAllowed(cube, c.grain, requestedDims, requestedMeasures, metrics) {
			continue
		}
		name := CubeMaterializationName(tenantID, isGoldCube(cube), cube, sortedCopy(c.grain))
		return c.grain, name, cubeMaterializationNodeName(cube, tenantID, c.grain), nil
	}

	d := CubeRouteDecision{MissReason: CubeMissNotDecomposed,
		Detail: "no matching grain is rollable for a non-decomposable metric"}
	return nil, "", "", &d
}

// timeRollupAllowed reports whether a materialization at grain can answer a
// request that differs in time granularity.
//
// Two cases force a rollup, and both need distributive metrics:
//   - the request is coarser than the materialization (day materialization,
//     month request), and
//   - the request omits the time dimension entirely while the materialization
//     carries it (a day-grain materialization collapsed to a single total).
//
// A request at the same granularity needs nothing extra. DeriveDecomposable
// already encodes the safety rule: AVG, division, and derived metrics are not
// distributive, so they must fall through to base tables.
func (r *CubeRouter) timeRollupAllowed(
	cube CubeDefinition,
	grain []string,
	requestedDims []string,
	requestedMeasures []string,
	metrics map[string]MetricDefinition,
) bool {

	// Only relevant when the cube declares a time dimension and the
	// materialization actually carries it.
	if cube.TimeDimension == nil {
		return true
	}
	timeTerm := strings.ToLower(strings.TrimSpace(cube.TimeDimension.TermNodeID))
	if timeTerm == "" || !containsFold(grain, timeTerm) {
		return true
	}

	// Does the request keep the time dimension at the materialization's own
	// granularity? If so there is no rollup and every metric is safe.
	requested := toSet(requestedDims)
	if requested[timeTerm] {
		return true
	}

	// Otherwise this is a rollup across time, so every requested metric must
	// be distributive.
	for _, m := range requestedMeasures {
		def, ok := metrics[m]
		if !ok {
			return false
		}
		if !def.Decomposable {
			return false
		}
	}
	return true
}

// cubeDefRow mirrors the table shape with JSONB columns left as raw bytes.
// sqlx cannot scan JSON into a typed slice or struct, so decoding is explicit
// and a single malformed column degrades that field rather than failing the
// whole row.
type cubeDefRow struct {
	ID              string     `db:"id"`
	TenantID        string     `db:"tenant_id"`
	Name            string     `db:"name"`
	Description     string     `db:"description"`
	BOID            string     `db:"bo_id"`
	Dimensions      []byte     `db:"dimensions"`
	TimeDimension   []byte     `db:"time_dimension"`
	MetricIDs       []byte     `db:"metric_ids"`
	Grains          []byte     `db:"grains"`
	Materialization []byte     `db:"materialization"`
	Federation      []byte     `db:"federation"`
	ContractVersion int        `db:"contract_version"`
	ContentHash     string     `db:"content_hash"`
	IsCore          bool       `db:"is_core"`
	Status          string     `db:"status"`
	ArchivedAt      *time.Time `db:"archived_at"`
	CreatedBy       *string    `db:"created_by"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
}

func (row cubeDefRow) toCubeDefinition() CubeDefinition {
	c := CubeDefinition{
		ID:              row.ID,
		TenantID:        row.TenantID,
		Name:            row.Name,
		Description:     row.Description,
		BOID:            row.BOID,
		ContractVersion: row.ContractVersion,
		ContentHash:     row.ContentHash,
		IsCore:          row.IsCore,
		Status:          row.Status,
		ArchivedAt:      row.ArchivedAt,
		CreatedBy:       row.CreatedBy,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
	if c.ContractVersion < 1 {
		c.ContractVersion = 1
	}
	if len(row.Dimensions) > 0 {
		_ = json.Unmarshal(row.Dimensions, &c.Dimensions)
	}
	if len(row.TimeDimension) > 0 {
		var td CubeTimeDimension
		if json.Unmarshal(row.TimeDimension, &td) == nil {
			c.TimeDimension = &td
		}
	}
	if len(row.MetricIDs) > 0 {
		_ = json.Unmarshal(row.MetricIDs, &c.MetricIDs)
	}
	if len(row.Grains) > 0 {
		_ = json.Unmarshal(row.Grains, &c.Grains)
	}
	if len(row.Materialization) > 0 {
		_ = json.Unmarshal(row.Materialization, &c.Materialization)
	}
	if len(row.Federation) > 0 {
		_ = json.Unmarshal(row.Federation, &c.Federation)
	}
	return c
}

// loadCube loads the single active cube for a tenant/BO plus its metrics.
//
// Ties are broken deterministically by name so routing is stable when more than
// one cube is somehow visible.
func (r *CubeRouter) loadCube(ctx context.Context, tenantID, boID string) (*CubeDefinition, map[string]MetricDefinition, error) {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(boID) == "" {
		return nil, nil, nil
	}

	var rows []cubeDefRow
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, tenant_id, name, COALESCE(description,'') AS description, bo_id,
		       dimensions, time_dimension, metric_ids, grains, materialization,
		       COALESCE(federation, '{}'::jsonb) AS federation,
		       COALESCE(contract_version, 1) AS contract_version,
		       COALESCE(content_hash,'') AS content_hash, is_core, status, archived_at,
		       created_by, created_at, updated_at
		FROM data_explorer.cube_definition
		WHERE tenant_id = $1 AND bo_id = $2
		  AND status = 'active' AND archived_at IS NULL
		ORDER BY name
	`, tenantID, boID)
	if err != nil {
		return nil, nil, err
	}
	if len(rows) == 0 {
		return nil, nil, nil
	}
	cube := rows[0].toCubeDefinition()

	metrics, err := r.loadMetrics(ctx, tenantID, cube.MetricIDs)
	if err != nil {
		return nil, nil, err
	}
	return &cube, metrics, nil
}

// metricDefRowColumns is the projection used to hydrate metricDefRow, kept
// identical to the reconciler's list so both read the same shape.
const metricDefRowColumns = `id, tenant_id, name, description, bo_id, catalog_term_id, expression,
	                 grain_allowlist, format_config, variables, materialization_config,
	                 decomposable, content_hash, tags, is_core, status, archived_at,
	                 created_by, created_at, updated_at`

// loadMetrics resolves metric IDs to definitions, including core metrics.
func (r *CubeRouter) loadMetrics(ctx context.Context, tenantID string, ids []string) (map[string]MetricDefinition, error) {
	out := make(map[string]MetricDefinition, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []metricDefRow
	err := r.db.SelectContext(ctx, &rows, `
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
		m := row.toMetricDefinition()
		out[strings.ToLower(strings.TrimSpace(m.ID))] = m
	}
	return out, nil
}

// materializationState reads lifecycle state for a materialization's catalog
// node. It returns the last refresh time, whether the materialization is stale,
// and whether it is active. A missing node is reported as inactive rather than
// an error, so a cube whose physical object was never deployed simply does not
// route.
func (r *CubeRouter) materializationState(ctx context.Context, nodeName string) (refreshedAt time.Time, stale bool, active bool, err error) {
	var props struct {
		Properties []byte `db:"properties"`
	}
	err = r.db.GetContext(ctx, &props, `
		SELECT n.properties
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'pre_aggregation' AND n.node_name = $1
	`, nodeName)
	if err == sql.ErrNoRows {
		return time.Time{}, false, false, nil
	}
	if err != nil {
		return time.Time{}, false, false, err
	}

	parsed, perr := models.ParsePreAggProperties(props.Properties)
	if perr != nil {
		// Unparseable properties mean we cannot reason about freshness, so
		// treat as inactive rather than guessing.
		return time.Time{}, false, false, nil
	}

	switch parsed.LifecycleStatus {
	case models.LifecycleStale:
		// Stale is servable-but-flagged: the data exists but is behind, and
		// the cube's stalePolicy decides whether to serve or fall back.
		return lastRefresh(parsed), true, true, nil
	case models.LifecycleActive:
		// An active materialization that has never recorded a refresh has no
		// data to trust, so it is stale by definition.
		if parsed.LastRefreshedAt == nil {
			return time.Time{}, true, true, nil
		}
		return *parsed.LastRefreshedAt, false, true, nil
	default:
		// materializing, refreshing, idle, failed: not query-visible yet. A
		// deploy is only visible once it reaches active (ADR: materialize
		// first, then flip the flag).
		return time.Time{}, false, false, nil
	}
}

// lastRefresh returns the recorded refresh time, or zero when absent.
func lastRefresh(p *models.PreAggProperties) time.Time {
	if p == nil || p.LastRefreshedAt == nil {
		return time.Time{}
	}
	return *p.LastRefreshedAt
}

// cubeMaterializationNodeName derives the catalog node name for a grain.
func cubeMaterializationNodeName(cube CubeDefinition, tenantID string, grain []string) string {
	return CubeMaterializationName(tenantID, isGoldCube(cube), cube, sortedCopy(grain))
}

func isGoldCube(cube CubeDefinition) bool { return cube.IsCore }

// requestedDimensionTerms returns the lowercased set of dimension terms the
// query groups by or projects.
func requestedDimensionTerms(qd *boresolver.QueryDef) []string {
	out := make([]string, 0, len(qd.Query.Dimensions)+len(qd.Query.GroupBy))
	for _, d := range qd.Query.Dimensions {
		if id := strings.ToLower(strings.TrimSpace(d.TermNodeID)); id != "" {
			out = append(out, id)
		}
	}
	for _, g := range qd.Query.GroupBy {
		if id := strings.ToLower(strings.TrimSpace(g)); id != "" {
			out = append(out, id)
		}
	}
	return dedupeStrings(out)
}

// requestedMeasureTerms returns the lowercased set of measure terms.
func requestedMeasureTerms(qd *boresolver.QueryDef) []string {
	out := make([]string, 0, len(qd.Query.Measures))
	for _, m := range qd.Query.Measures {
		if id := strings.ToLower(strings.TrimSpace(m.TermNodeID)); id != "" {
			out = append(out, id)
		}
	}
	return dedupeStrings(out)
}

func toSet(items []string) map[string]bool {
	out := make(map[string]bool, len(items))
	for _, i := range items {
		out[strings.ToLower(strings.TrimSpace(i))] = true
	}
	return out
}

func containsFold(haystack []string, needle string) bool {
	for _, h := range haystack {
		if strings.EqualFold(strings.TrimSpace(h), strings.TrimSpace(needle)) {
			return true
		}
	}
	return false
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func sortedCopy(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	for i := range out {
		out[i] = sanitizeIdentifier(out[i])
	}
	sort.Strings(out)
	return out
}

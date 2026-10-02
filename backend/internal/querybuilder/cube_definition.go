package querybuilder

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// CubeBundleSchemaVersion is the official schema version identifier for cube bundles.
const CubeBundleSchemaVersion = "uisce.cube-bundle/1"

var (
	// ErrCubeNoMetrics is returned when a cube declares no governed metrics.
	// A cube with no metrics has no aggregation contract to publish, and
	// allowing one would let an empty surface look deployable.
	ErrCubeNoMetrics = errors.New("cube requires at least one governed metric")

	// ErrCubeNoGrains is returned when a cube declares no materialization
	// grain. A cube exists to be materialized; with no grain it would be a
	// pure query definition and belongs in saved_queries instead.
	ErrCubeNoGrains = errors.New("cube requires at least one materialization grain")

	// ErrCubeUnknownMetric is returned when a referenced metric ID does not
	// resolve to a live metric_definition in the same tenant. This is the
	// guard that keeps cubes on the governed metric layer: a cube can only
	// aggregate metrics that exist.
	ErrCubeUnknownMetric = errors.New("cube references a metric that does not exist or is archived")

	// ErrCubeDimensionNotInGrain is returned when the declared dimension
	// surface is not fully covered by the declared grains. A dimension no
	// materialization carries can never be served, so the cube would advertise
	// an axis it cannot answer.
	ErrCubeDimensionNotInGrain = errors.New("cube dimension is not covered by any declared grain")
)

// CubeDimension is one axis of the cube's dimension surface.
//
// Order is meaningful: it is the axis order the cube exposes. DrillPath
// encodes the optional hierarchy for future hierarchy-aware rollup; the shipped
// router matches by plain set-subset and ignores it (ADR-015).
type CubeDimension struct {
	TermNodeID string   `json:"termNodeId"`
	DrillPath  []string `json:"drillPath,omitempty"`
}

// CubeTimeDimension names the cube's time axis and its default grain.
type CubeTimeDimension struct {
	TermNodeID   string `json:"termNodeId"`
	DefaultGrain string `json:"defaultGrain,omitempty"`
}

// CubeMaterializationConfig is the physical plan for a cube.
//
// StalePolicy reuses the metric-layer vocabulary: "serve_with_flag" (default,
// dashboards) or "force_raw_fallback" (compliance workloads that cannot
// tolerate flagged data).
type CubeMaterializationConfig struct {
	Strategy               string `json:"strategy"`                         // "starrocks_mv" (default) | "aggregate_table"
	RefreshStrategy        string `json:"refreshStrategy,omitempty"`        // "manual" | "interval" | "incremental"
	RefreshIntervalMinutes int    `json:"refreshIntervalMinutes,omitempty"` // compiled into PreAggProperties
	PartitionGrain         string `json:"partitionGrain,omitempty"`
	RetentionDays          int    `json:"retentionDays,omitempty"`
	StalePolicy            string `json:"stalePolicy,omitempty"` // "serve_with_flag" (default) | "force_raw_fallback"
}

// CubeDefinition is one row in data_explorer.cube_definition.
//
// A cube is a published aggregation contract: a BO, an ordered dimension
// surface, a governed metric set, and a declared set of physical grains. It
// stores no physical information at all — BOID is a logical reference — which
// is what keeps cube definitions bundle-portable and gold-copy safe.
type CubeDefinition struct {
	ID              string                    `json:"id" db:"id"`
	TenantID        string                    `json:"tenantId" db:"tenant_id"`
	Name            string                    `json:"name" db:"name"`
	Description     string                    `json:"description" db:"description"`
	BOID            string                    `json:"boId" db:"bo_id"`
	Dimensions      []CubeDimension           `json:"dimensions" db:"dimensions"`
	TimeDimension   *CubeTimeDimension        `json:"timeDimension,omitempty" db:"time_dimension"`
	MetricIDs       []string                  `json:"metricIds" db:"metric_ids"`
	Grains          [][]string                `json:"grains" db:"grains"`
	Materialization CubeMaterializationConfig `json:"materialization" db:"materialization"`
	ContentHash     string                    `json:"contentHash" db:"content_hash"`
	IsCore          bool                      `json:"isCore" db:"is_core"`
	Status          string                    `json:"status" db:"status"` // "active" | "deprecated" | "archived"
	ArchivedAt      *time.Time                `json:"archivedAt,omitempty" db:"archived_at"`
	CreatedBy       *string                   `json:"createdBy,omitempty" db:"created_by"`
	CreatedAt       time.Time                 `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time                 `json:"updatedAt" db:"updated_at"`
}

// ComputeCubeContentHash deterministically computes the SHA-256 hash of a
// cube's semantic content.
//
// Canonicalization mirrors ComputeMetricContentHash: dimension order is
// significant (it is the axis order), but metric IDs and grain members are
// sorted, and each grain is itself sorted, so that a semantically identical
// cube hashes identically regardless of authoring order. Content — not
// identity or timestamps — is what is hashed, which is what makes deploy
// idempotent: an unchanged hash means there is nothing to do.
//
// It also covers what the cube COMPUTES, not just which metrics it NAMES. The
// referenced metrics' content hashes are folded in, sorted, exactly as the
// query cache key does with referenced metrics. Without this, a metric
// definition edit left the cube hash untouched, the deploy stayed a no-op under
// ADR-011, and a materialized view kept serving the pre-edit expression
// indefinitely - the same class as the ADR-016 cache-key defect: a hash that
// cannot see the thing it must distinguish. See ADR-027.
//
// metricContentHashes must hold the content hash of every metric the cube
// names. Callers about to deploy should always pass them; an empty slice is
// tolerated for validation-time use but yields a hash that does not track the
// cube's metrics.
func ComputeCubeContentHash(c CubeDefinition, metricContentHashes []string) string {
	// Metric IDs are a set, not a sequence: authoring order must not change
	// the hash, and a duplicate is a validation error rather than a
	// meaningful distinction.
	metricIDs := make([]string, len(c.MetricIDs))
	copy(metricIDs, c.MetricIDs)
	for i, id := range metricIDs {
		metricIDs[i] = strings.ToLower(strings.TrimSpace(id))
	}
	sort.Strings(metricIDs)

	// Same rule for the referenced content hashes: a set, sorted, so the order
	// metrics happen to be resolved in cannot change the cube's identity.
	contentHashes := make([]string, len(metricContentHashes))
	for i, h := range metricContentHashes {
		contentHashes[i] = strings.ToLower(strings.TrimSpace(h))
	}
	sort.Strings(contentHashes)

	// Each grain is also a set of dimensions, so sort within each. The set of
	// grains is not sorted: grain order can carry intent about which shape is
	// primary, and normalizing it away would merge two genuinely different
	// plans.
	grains := make([][]string, len(c.Grains))
	for i, g := range c.Grains {
		norm := make([]string, len(g))
		for j, dim := range g {
			norm[j] = strings.ToLower(strings.TrimSpace(dim))
		}
		sort.Strings(norm)
		grains[i] = norm
	}

	// Dimensions keep their order: it is the cube's axis order.
	dims := make([]CubeDimension, len(c.Dimensions))
	for i, d := range c.Dimensions {
		drill := make([]string, len(d.DrillPath))
		copy(drill, d.DrillPath)
		for j, p := range drill {
			drill[j] = strings.ToLower(strings.TrimSpace(p))
		}
		sort.Strings(drill)
		dims[i] = CubeDimension{
			TermNodeID: strings.ToLower(strings.TrimSpace(d.TermNodeID)),
			DrillPath:  drill,
		}
	}

	canonicalDoc := struct {
		Name               string                    `json:"name"`
		BOID               string                    `json:"boId"`
		Dimensions         []CubeDimension           `json:"dimensions"`
		TimeDimension      *CubeTimeDimension        `json:"timeDimension"`
		MetricIDs          []string                  `json:"metricIds"`
		MetricContentHashes []string                 `json:"metricContentHashes"`
		Grains             [][]string                `json:"grains"`
		Materialization    CubeMaterializationConfig `json:"materialization"`
	}{
		Name:                strings.TrimSpace(c.Name),
		BOID:                strings.TrimSpace(c.BOID),
		Dimensions:          dims,
		TimeDimension:       c.TimeDimension,
		MetricIDs:           metricIDs,
		MetricContentHashes: contentHashes,
		Grains:              grains,
		Materialization:     c.Materialization,
	}

	b, _ := json.Marshal(canonicalDoc)
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}

// DimensionSet returns the cube's declared dimension term IDs lowercased and
// trimmed, for grain-coverage checks.
func (c CubeDefinition) DimensionSet() []string {
	out := make([]string, 0, len(c.Dimensions))
	for _, d := range c.Dimensions {
		if id := strings.ToLower(strings.TrimSpace(d.TermNodeID)); id != "" {
			out = append(out, id)
		}
	}
	return out
}

// ValidateCubeStructural performs checks that need no database access: a cube
// must name at least one metric and one grain, and its dimension surface must
// be covered by at least one declared grain.
//
// Reference resolution (ErrCubeUnknownMetric) is deliberately NOT checked here
// because it requires loading metric_definition rows; see
// ValidateCubeMetricReferences.
func ValidateCubeStructural(c CubeDefinition) error {
	if len(c.MetricIDs) == 0 {
		return ErrCubeNoMetrics
	}
	if len(c.Grains) == 0 {
		return ErrCubeNoGrains
	}

	// Duplicate metric IDs mean the author listed the same metric twice, which
	// is a mistake worth surfacing rather than silently de-duplicating.
	seen := make(map[string]bool, len(c.MetricIDs))
	for _, id := range c.MetricIDs {
		norm := strings.ToLower(strings.TrimSpace(id))
		if norm == "" {
			return fmt.Errorf("%w: empty metric id", ErrCubeUnknownMetric)
		}
		if seen[norm] {
			return fmt.Errorf("cube lists metric %q more than once", norm)
		}
		seen[norm] = true
	}

	// A dimension that no grain carries can never be served, so the cube would
	// advertise an axis it cannot answer.
	declared := c.DimensionSet()
	covered := make(map[string]bool)
	for _, grain := range c.Grains {
		for _, dim := range grain {
			covered[strings.ToLower(strings.TrimSpace(dim))] = true
		}
	}
	for _, dim := range declared {
		if !covered[dim] {
			return fmt.Errorf("%w: %q", ErrCubeDimensionNotInGrain, dim)
		}
	}
	return nil
}

// ValidateCubeMetricReferences checks that every referenced metric ID resolves
// to a live metric in the same tenant.
//
// This is the constraint that makes a cube a governed artifact rather than an
// ad-hoc aggregate: a cube can only aggregate metrics that exist, and it
// inherits their AST safety and decomposability. known is the set of metric IDs
// that resolved for this tenant (both core and own-tenant metrics).
// ValidateCubeMetricReferences checks that every metric a cube names exists,
// and that each one is a metric the cube could actually materialize. metrics
// maps normalized metric ID to the definition, so an expression is validated
// here rather than being discovered to be unusable at DDL generation.
//
// This is the earliest enforcement point that exists today: there is no CRUD
// endpoint for governed metric expressions (see ValidateMetricExpression), so
// a cube is where an ambiguous metric is first caught. See ADR-026.
func ValidateCubeMetricReferences(c CubeDefinition, metrics map[string]MetricDefinition) error {
	for _, id := range c.MetricIDs {
		norm := strings.ToLower(strings.TrimSpace(id))
		m, ok := metrics[norm]
		if !ok {
			return fmt.Errorf("%w: %q", ErrCubeUnknownMetric, norm)
		}
		if err := ValidateMetricExpression(m); err != nil {
			return fmt.Errorf("cube %q references metric %q, which is not materializable: %w", c.Name, norm, err)
		}
	}
	return nil
}

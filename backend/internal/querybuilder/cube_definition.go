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

	// ErrCubeFederationInvalid is returned when federation.sources/joins are
	// malformed (missing aliases, mismatched composite key lengths, etc.).
	ErrCubeFederationInvalid = errors.New("cube federation plan is invalid")

	// ErrCubeContractVersionInvalid is returned when contract_version is < 1.
	ErrCubeContractVersionInvalid = errors.New("cube contract_version must be >= 1")
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
//
// HotEngine / ColdEngine default to starrocks / iceberg at deploy time when
// empty. RetentionDays remains the legacy single window; RetentionDaysHot and
// RetentionDaysCold override per tier when set (CUBE-0.1 / dual-tier plan).
type CubeMaterializationConfig struct {
	Strategy               string `json:"strategy"`                         // "starrocks_mv" (default) | "aggregate_table"
	RefreshStrategy        string `json:"refreshStrategy,omitempty"`        // "manual" | "interval" | "incremental"
	RefreshIntervalMinutes int    `json:"refreshIntervalMinutes,omitempty"` // compiled into PreAggProperties
	PartitionGrain         string `json:"partitionGrain,omitempty"`
	RetentionDays          int    `json:"retentionDays,omitempty"`
	RetentionDaysHot       int    `json:"retentionDaysHot,omitempty"`
	RetentionDaysCold      int    `json:"retentionDaysCold,omitempty"`
	HotEngine              string `json:"hotEngine,omitempty"`   // "starrocks" (default)
	ColdEngine             string `json:"coldEngine,omitempty"`  // "iceberg" (default)
	StalePolicy            string `json:"stalePolicy,omitempty"` // "serve_with_flag" (default) | "force_raw_fallback"
}

// CubeFederationSource is one BO/binding participating in a federated cube.
type CubeFederationSource struct {
	BOID        string `json:"boId"`
	Alias       string `json:"alias"`
	BindingHint string `json:"bindingHint,omitempty"` // e.g. "orm", "mdm"
}

// CubeFederationJoin declares how two sources share keys.
//
// KeyKind is "common" (same semantic terms) or "transform" (deterministic
// calc/term applied before join). TransformTermID must reference a
// deterministic term/calc — never a metric_definition aggregate (ADR-011).
// Composite keys use parallel LeftTermIDs / RightTermIDs (same length, order).
type CubeFederationJoin struct {
	LeftAlias       string   `json:"leftAlias"`
	RightAlias      string   `json:"rightAlias"`
	KeyKind         string   `json:"keyKind"` // "common" | "transform"
	LeftTermIDs     []string `json:"leftTermIds"`
	RightTermIDs    []string `json:"rightTermIds"`
	TransformTermID string   `json:"transformTermId,omitempty"`
}

// CubeFederation is the declared multi-BO join plan. An empty plan (no
// sources/joins) means the cube is single-BO on CubeDefinition.BOID.
type CubeFederation struct {
	Sources []CubeFederationSource `json:"sources,omitempty"`
	Joins   []CubeFederationJoin   `json:"joins,omitempty"`
	// OrphanRateMaxPercent fails deploy when unmatched keys exceed this on
	// either side. Zero means use the platform default (1.0).
	OrphanRateMaxPercent float64 `json:"orphanRateMaxPercent,omitempty"`
}

// Empty reports whether federation is undeclared (single-BO cube).
func (f CubeFederation) Empty() bool {
	return len(f.Sources) == 0 && len(f.Joins) == 0
}

// CubeDefinition is one row in data_explorer.cube_definition.
//
// A cube is a published aggregation contract: a BO, an ordered dimension
// surface, a governed metric set, and a declared set of physical grains. It
// stores no physical information at all — BOID is a logical reference — which
// is what keeps cube definitions bundle-portable and gold-copy safe.
//
// ContractVersion is the published generation consumers pin. ContentHash
// covers the semantic surface (including Federation); bump ContractVersion
// when DetectCubeContractBreaking reports a break.
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
	Federation      CubeFederation            `json:"federation" db:"federation"`
	ContractVersion int                       `json:"contractVersion" db:"contract_version"`
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
		Name                string                    `json:"name"`
		BOID                string                    `json:"boId"`
		Dimensions          []CubeDimension           `json:"dimensions"`
		TimeDimension       *CubeTimeDimension        `json:"timeDimension"`
		MetricIDs           []string                  `json:"metricIds"`
		MetricContentHashes []string                  `json:"metricContentHashes"`
		Grains              [][]string                `json:"grains"`
		Materialization     CubeMaterializationConfig `json:"materialization"`
		Federation          CubeFederation            `json:"federation"`
	}{
		Name:                strings.TrimSpace(c.Name),
		BOID:                strings.TrimSpace(c.BOID),
		Dimensions:          dims,
		TimeDimension:       c.TimeDimension,
		MetricIDs:           metricIDs,
		MetricContentHashes: contentHashes,
		Grains:              grains,
		Materialization:     c.Materialization,
		Federation:          canonicalizeFederation(c.Federation),
	}

	b, _ := json.Marshal(canonicalDoc)
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}

func canonicalizeFederation(f CubeFederation) CubeFederation {
	out := CubeFederation{
		OrphanRateMaxPercent: f.OrphanRateMaxPercent,
	}
	if len(f.Sources) > 0 {
		out.Sources = make([]CubeFederationSource, len(f.Sources))
		for i, s := range f.Sources {
			out.Sources[i] = CubeFederationSource{
				BOID:        strings.TrimSpace(s.BOID),
				Alias:       strings.ToLower(strings.TrimSpace(s.Alias)),
				BindingHint: strings.ToLower(strings.TrimSpace(s.BindingHint)),
			}
		}
		sort.SliceStable(out.Sources, func(i, j int) bool {
			if out.Sources[i].Alias != out.Sources[j].Alias {
				return out.Sources[i].Alias < out.Sources[j].Alias
			}
			return out.Sources[i].BOID < out.Sources[j].BOID
		})
	}
	if len(f.Joins) > 0 {
		out.Joins = make([]CubeFederationJoin, len(f.Joins))
		for i, jn := range f.Joins {
			left := normalizeTermIDList(jn.LeftTermIDs)
			right := normalizeTermIDList(jn.RightTermIDs)
			out.Joins[i] = CubeFederationJoin{
				LeftAlias:       strings.ToLower(strings.TrimSpace(jn.LeftAlias)),
				RightAlias:      strings.ToLower(strings.TrimSpace(jn.RightAlias)),
				KeyKind:         strings.ToLower(strings.TrimSpace(jn.KeyKind)),
				LeftTermIDs:     left,
				RightTermIDs:    right,
				TransformTermID: strings.ToLower(strings.TrimSpace(jn.TransformTermID)),
			}
		}
		sort.SliceStable(out.Joins, func(i, j int) bool {
			a, b := out.Joins[i], out.Joins[j]
			if a.LeftAlias != b.LeftAlias {
				return a.LeftAlias < b.LeftAlias
			}
			if a.RightAlias != b.RightAlias {
				return a.RightAlias < b.RightAlias
			}
			return strings.Join(a.LeftTermIDs, ",") < strings.Join(b.LeftTermIDs, ",")
		})
	}
	return out
}

func normalizeTermIDList(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = strings.ToLower(strings.TrimSpace(id))
	}
	return out
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
	// Zero is treated as 1 for authoring drafts that have not set the field yet.
	if c.ContractVersion < 0 {
		return ErrCubeContractVersionInvalid
	}
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
	if err := ValidateCubeFederation(c.Federation); err != nil {
		return err
	}
	return nil
}

// DefaultFederationOrphanRatePercent is the platform default when
// federation.orphanRateMaxPercent is unset (CUBE Phase 2 gate).
const DefaultFederationOrphanRatePercent = 1.0

// ValidateCubeFederation checks federation shape without resolving terms in DB.
// Empty federation is valid (single-BO cube).
func ValidateCubeFederation(f CubeFederation) error {
	if f.Empty() {
		return nil
	}
	if f.OrphanRateMaxPercent < 0 {
		return fmt.Errorf("%w: orphanRateMaxPercent must be >= 0", ErrCubeFederationInvalid)
	}
	aliases := make(map[string]string, len(f.Sources)) // alias -> boId
	for i, s := range f.Sources {
		alias := strings.ToLower(strings.TrimSpace(s.Alias))
		boid := strings.TrimSpace(s.BOID)
		if alias == "" {
			return fmt.Errorf("%w: sources[%d] missing alias", ErrCubeFederationInvalid, i)
		}
		if boid == "" {
			return fmt.Errorf("%w: sources[%d] (%s) missing boId", ErrCubeFederationInvalid, i, alias)
		}
		if _, dup := aliases[alias]; dup {
			return fmt.Errorf("%w: duplicate source alias %q", ErrCubeFederationInvalid, alias)
		}
		aliases[alias] = boid
	}
	if len(f.Joins) > 0 && len(f.Sources) < 2 {
		return fmt.Errorf("%w: joins require at least two sources", ErrCubeFederationInvalid)
	}
	for i, jn := range f.Joins {
		left := strings.ToLower(strings.TrimSpace(jn.LeftAlias))
		right := strings.ToLower(strings.TrimSpace(jn.RightAlias))
		kind := strings.ToLower(strings.TrimSpace(jn.KeyKind))
		if left == "" || right == "" {
			return fmt.Errorf("%w: joins[%d] missing leftAlias/rightAlias", ErrCubeFederationInvalid, i)
		}
		if left == right {
			return fmt.Errorf("%w: joins[%d] leftAlias and rightAlias must differ", ErrCubeFederationInvalid, i)
		}
		if _, ok := aliases[left]; !ok {
			return fmt.Errorf("%w: joins[%d] unknown leftAlias %q", ErrCubeFederationInvalid, i, left)
		}
		if _, ok := aliases[right]; !ok {
			return fmt.Errorf("%w: joins[%d] unknown rightAlias %q", ErrCubeFederationInvalid, i, right)
		}
		switch kind {
		case "common", "transform":
		case "":
			return fmt.Errorf("%w: joins[%d] missing keyKind", ErrCubeFederationInvalid, i)
		default:
			return fmt.Errorf("%w: joins[%d] keyKind %q must be common|transform", ErrCubeFederationInvalid, i, kind)
		}
		if len(jn.LeftTermIDs) == 0 || len(jn.RightTermIDs) == 0 {
			return fmt.Errorf("%w: joins[%d] requires leftTermIds and rightTermIds", ErrCubeFederationInvalid, i)
		}
		if len(jn.LeftTermIDs) != len(jn.RightTermIDs) {
			return fmt.Errorf("%w: joins[%d] leftTermIds/rightTermIds length mismatch", ErrCubeFederationInvalid, i)
		}
		for _, id := range jn.LeftTermIDs {
			if strings.TrimSpace(id) == "" {
				return fmt.Errorf("%w: joins[%d] empty leftTermId", ErrCubeFederationInvalid, i)
			}
		}
		for _, id := range jn.RightTermIDs {
			if strings.TrimSpace(id) == "" {
				return fmt.Errorf("%w: joins[%d] empty rightTermId", ErrCubeFederationInvalid, i)
			}
		}
		if kind == "transform" && strings.TrimSpace(jn.TransformTermID) == "" {
			return fmt.Errorf("%w: joins[%d] transform keyKind requires transformTermId", ErrCubeFederationInvalid, i)
		}
		if kind == "common" && strings.TrimSpace(jn.TransformTermID) != "" {
			return fmt.Errorf("%w: joins[%d] common keyKind must not set transformTermId", ErrCubeFederationInvalid, i)
		}
	}
	return nil
}

// CubeContractBreakReason names why a new draft must bump contract_version.
type CubeContractBreakReason string

const (
	CubeBreakGrainChange       CubeContractBreakReason = "grains_changed"
	CubeBreakDimensionRemoved  CubeContractBreakReason = "dimension_removed"
	CubeBreakMetricRemoved     CubeContractBreakReason = "metric_removed"
	CubeBreakFederationChanged CubeContractBreakReason = "federation_changed"
)

// DetectCubeContractBreaking compares a previously published cube to a draft.
// Additive metrics/dimensions alone are not breaking; grain set changes,
// removals, and federation plan changes are.
//
// Callers that receive a non-empty reason list must publish a new
// contract_version (or reject in-place save when consumers pin the old version).
func DetectCubeContractBreaking(published, draft CubeDefinition) []CubeContractBreakReason {
	var reasons []CubeContractBreakReason

	pubGrains := grainSignature(published.Grains)
	draftGrains := grainSignature(draft.Grains)
	if pubGrains != draftGrains {
		reasons = append(reasons, CubeBreakGrainChange)
	}

	draftDims := toSet(draft.DimensionSet())
	for _, d := range published.DimensionSet() {
		if !draftDims[d] {
			reasons = append(reasons, CubeBreakDimensionRemoved)
			break
		}
	}

	pubMetrics := toSet(normalizeTermIDList(published.MetricIDs))
	draftMetrics := toSet(normalizeTermIDList(draft.MetricIDs))
	for id := range pubMetrics {
		if !draftMetrics[id] {
			reasons = append(reasons, CubeBreakMetricRemoved)
			break
		}
	}

	pubFed, _ := json.Marshal(canonicalizeFederation(published.Federation))
	draftFed, _ := json.Marshal(canonicalizeFederation(draft.Federation))
	if string(pubFed) != string(draftFed) {
		reasons = append(reasons, CubeBreakFederationChanged)
	}

	return reasons
}

func grainSignature(grains [][]string) string {
	parts := make([]string, 0, len(grains))
	for _, g := range grains {
		norm := normalizeTermIDList(g)
		cp := make([]string, len(norm))
		copy(cp, norm)
		sort.Strings(cp)
		parts = append(parts, strings.Join(cp, "+"))
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

// EffectiveOrphanRateMax returns the cube override or platform default.
func EffectiveOrphanRateMax(f CubeFederation) float64 {
	if f.OrphanRateMaxPercent > 0 {
		return f.OrphanRateMaxPercent
	}
	return DefaultFederationOrphanRatePercent
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

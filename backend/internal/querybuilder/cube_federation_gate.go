package querybuilder

import (
	"errors"
	"fmt"
	"strings"
)

// ErrCubeFederationOrphanGate is returned when measured unmatched keys exceed
// the cube's orphanRateMaxPercent (or the 1.0% platform default).
var ErrCubeFederationOrphanGate = errors.New("cube federation orphan rate exceeds threshold")

// ErrCubeFederationTransformInvalid is returned when a transformTermId is
// missing, empty, or resolves as a metric aggregate (forbidden — ADR-011).
var ErrCubeFederationTransformInvalid = errors.New("cube federation transform term is invalid")

// FederationKeySample is a measured key-cardinality fixture for one join edge.
// Matched must be ≤ min(LeftKeys, RightKeys). Used by validate/deploy gate and
// unit fixtures until CUBE-2.2 samples live sources.
type FederationKeySample struct {
	LeftAlias  string `json:"leftAlias"`
	RightAlias string `json:"rightAlias"`
	LeftKeys   int64  `json:"leftKeys"`
	RightKeys  int64  `json:"rightKeys"`
	Matched    int64  `json:"matched"`
}

// FederationJoinOrphanStats is the per-join orphan receipt.
type FederationJoinOrphanStats struct {
	LeftAlias      string  `json:"leftAlias"`
	RightAlias     string  `json:"rightAlias"`
	LeftKeys       int64   `json:"leftKeys"`
	RightKeys      int64   `json:"rightKeys"`
	Matched        int64   `json:"matched"`
	LeftOrphanPct  float64 `json:"leftOrphanPct"`
	RightOrphanPct float64 `json:"rightOrphanPct"`
	Ok             bool    `json:"ok"`
	Detail         string  `json:"detail,omitempty"`
}

// FederationOrphanReport is the validate/deploy orphan gate receipt.
type FederationOrphanReport struct {
	Joins              []FederationJoinOrphanStats `json:"joins"`
	MaxPercent         float64                     `json:"maxPercent"`
	ObservedMaxPercent float64                     `json:"observedMaxPercent"`
	Ok                 bool                        `json:"ok"`
	Skipped            bool                        `json:"skipped,omitempty"`
	Detail             string                      `json:"detail,omitempty"`
}

// FederationTransformProjection is the compiled projection for a transform join.
// Physical column binding lands in CUBE-2.2; here we emit a deterministic
// expression handle the materializer can later substitute.
type FederationTransformProjection struct {
	JoinIndex       int      `json:"joinIndex"`
	LeftAlias       string   `json:"leftAlias"`
	RightAlias      string   `json:"rightAlias"`
	TransformTermID string   `json:"transformTermId"`
	LeftTermIDs     []string `json:"leftTermIds"`
	RightTermIDs    []string `json:"rightTermIds"`
	// CompiledExpr is a placeholder SQL projection: transform_term(<id>) AS fed_key_<n>
	CompiledExpr string `json:"compiledExpr"`
}

// OrphanRatePercent returns 100 * (sideKeys - matched) / sideKeys.
// Zero sideKeys yields 0 (no keys ⇒ no orphans to gate).
func OrphanRatePercent(sideKeys, matched int64) float64 {
	if sideKeys <= 0 {
		return 0
	}
	orphans := sideKeys - matched
	if orphans < 0 {
		orphans = 0
	}
	return 100.0 * float64(orphans) / float64(sideKeys)
}

// EvaluateFederationOrphanGate compares key samples to the effective orphan
// threshold. Empty federation ⇒ Ok=true Skipped=true. Non-empty federation with
// no samples ⇒ Ok=false Skipped=true (fail closed for deploy; validate may
// still report shape-only success separately).
func EvaluateFederationOrphanGate(f CubeFederation, samples []FederationKeySample) FederationOrphanReport {
	maxPct := EffectiveOrphanRateMax(f)
	report := FederationOrphanReport{
		MaxPercent: maxPct,
		Ok:         true,
	}
	if f.Empty() {
		report.Skipped = true
		report.Detail = "single-BO cube; orphan gate not applicable"
		return report
	}
	if len(samples) == 0 {
		report.Ok = false
		report.Skipped = true
		report.Detail = "federation declared but no key samples provided; orphan gate fail-closed"
		return report
	}

	byEdge := make(map[string]FederationKeySample, len(samples))
	for _, s := range samples {
		key := federationEdgeKey(s.LeftAlias, s.RightAlias)
		byEdge[key] = s
	}

	observed := 0.0
	for _, jn := range f.Joins {
		left := strings.ToLower(strings.TrimSpace(jn.LeftAlias))
		right := strings.ToLower(strings.TrimSpace(jn.RightAlias))
		sample, ok := byEdge[federationEdgeKey(left, right)]
		stat := FederationJoinOrphanStats{
			LeftAlias:  left,
			RightAlias: right,
			Ok:         true,
		}
		if !ok {
			stat.Ok = false
			stat.Detail = "missing key sample for join edge"
			report.Ok = false
			report.Joins = append(report.Joins, stat)
			continue
		}
		if sample.Matched < 0 || sample.LeftKeys < 0 || sample.RightKeys < 0 {
			stat.Ok = false
			stat.Detail = "key sample counts must be >= 0"
			report.Ok = false
			report.Joins = append(report.Joins, stat)
			continue
		}
		minSide := sample.LeftKeys
		if sample.RightKeys < minSide {
			minSide = sample.RightKeys
		}
		if sample.Matched > minSide {
			stat.Ok = false
			stat.Detail = "matched exceeds min(leftKeys, rightKeys)"
			report.Ok = false
			report.Joins = append(report.Joins, stat)
			continue
		}
		stat.LeftKeys = sample.LeftKeys
		stat.RightKeys = sample.RightKeys
		stat.Matched = sample.Matched
		stat.LeftOrphanPct = OrphanRatePercent(sample.LeftKeys, sample.Matched)
		stat.RightOrphanPct = OrphanRatePercent(sample.RightKeys, sample.Matched)
		worst := stat.LeftOrphanPct
		if stat.RightOrphanPct > worst {
			worst = stat.RightOrphanPct
		}
		if worst > observed {
			observed = worst
		}
		if worst > maxPct {
			stat.Ok = false
			stat.Detail = fmt.Sprintf("orphan %.4f%% exceeds max %.4f%%", worst, maxPct)
			report.Ok = false
		}
		report.Joins = append(report.Joins, stat)
	}
	report.ObservedMaxPercent = observed
	if !report.Ok && report.Detail == "" {
		report.Detail = fmt.Sprintf("observed orphan %.4f%% exceeds max %.4f%%", observed, maxPct)
	}
	return report
}

func federationEdgeKey(left, right string) string {
	return strings.ToLower(strings.TrimSpace(left)) + "→" + strings.ToLower(strings.TrimSpace(right))
}

// CompileFederationTransforms builds deterministic projection handles for every
// transform join. metricIDs is the set of metric definition IDs that must NOT
// appear as transformTermId (aggregates are forbidden on federation keys).
func CompileFederationTransforms(f CubeFederation, metricIDs map[string]bool) ([]FederationTransformProjection, error) {
	if err := ValidateCubeFederation(f); err != nil {
		return nil, err
	}
	if f.Empty() {
		return nil, nil
	}
	out := make([]FederationTransformProjection, 0)
	for i, jn := range f.Joins {
		kind := strings.ToLower(strings.TrimSpace(jn.KeyKind))
		if kind != "transform" {
			continue
		}
		tid := strings.TrimSpace(jn.TransformTermID)
		if tid == "" {
			return nil, fmt.Errorf("%w: joins[%d] missing transformTermId", ErrCubeFederationTransformInvalid, i)
		}
		if metricIDs[strings.ToLower(tid)] {
			return nil, fmt.Errorf("%w: joins[%d] transformTermId %q is a metric aggregate; use a deterministic term/calc",
				ErrCubeFederationTransformInvalid, i, tid)
		}
		leftIDs := append([]string{}, jn.LeftTermIDs...)
		rightIDs := append([]string{}, jn.RightTermIDs...)
		out = append(out, FederationTransformProjection{
			JoinIndex:       i,
			LeftAlias:       strings.ToLower(strings.TrimSpace(jn.LeftAlias)),
			RightAlias:      strings.ToLower(strings.TrimSpace(jn.RightAlias)),
			TransformTermID: tid,
			LeftTermIDs:     leftIDs,
			RightTermIDs:    rightIDs,
			CompiledExpr: fmt.Sprintf(
				"transform_term(%s) AS fed_key_%d",
				sanitizeIdentifier(tid),
				i,
			),
		})
	}
	return out, nil
}

// EvaluateFederationPlan runs shape validation, transform compile, and optional
// orphan gate. samples may be nil for shape+transform-only checks (orphan
// skipped / fail-closed depending on requireOrphanSamples).
func EvaluateFederationPlan(
	f CubeFederation,
	metricIDs map[string]bool,
	samples []FederationKeySample,
	requireOrphanSamples bool,
) (projections []FederationTransformProjection, orphan FederationOrphanReport, err error) {
	if err := ValidateCubeFederation(f); err != nil {
		return nil, FederationOrphanReport{}, err
	}
	projections, err = CompileFederationTransforms(f, metricIDs)
	if err != nil {
		return nil, FederationOrphanReport{}, err
	}
	if f.Empty() {
		return projections, EvaluateFederationOrphanGate(f, nil), nil
	}
	orphan = EvaluateFederationOrphanGate(f, samples)
	if requireOrphanSamples && (!orphan.Ok || orphan.Skipped) {
		return projections, orphan, fmt.Errorf("%w: %s", ErrCubeFederationOrphanGate, orphan.Detail)
	}
	if !orphan.Skipped && !orphan.Ok {
		return projections, orphan, fmt.Errorf("%w: %s", ErrCubeFederationOrphanGate, orphan.Detail)
	}
	return projections, orphan, nil
}

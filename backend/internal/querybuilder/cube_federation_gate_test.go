package querybuilder

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrphanRatePercent(t *testing.T) {
	assert.InDelta(t, 3.0, OrphanRatePercent(1000, 970), 1e-9)
	assert.InDelta(t, 0.0, OrphanRatePercent(0, 0), 1e-9)
	assert.InDelta(t, 0.0, OrphanRatePercent(100, 100), 1e-9)
	assert.InDelta(t, 100.0, OrphanRatePercent(10, 0), 1e-9)
}

func TestEvaluateFederationOrphanGate_FixtureKnownPercent(t *testing.T) {
	f := sampleFederatedCube().Federation // max 1.0%
	// 15 orphans on left of 1000 = 1.5% → fail
	samples := []FederationKeySample{{
		LeftAlias: "pos", RightAlias: "acct",
		LeftKeys: 1000, RightKeys: 1000, Matched: 985,
	}}
	report := EvaluateFederationOrphanGate(f, samples)
	require.False(t, report.Ok)
	assert.InDelta(t, 1.5, report.ObservedMaxPercent, 1e-9)
	assert.InDelta(t, 1.0, report.MaxPercent, 1e-9)
	require.ErrorIs(t, fmt.Errorf("%w: %s", ErrCubeFederationOrphanGate, report.Detail), ErrCubeFederationOrphanGate)
}

func TestEvaluateFederationOrphanGate_WithinThreshold(t *testing.T) {
	f := sampleFederatedCube().Federation
	samples := []FederationKeySample{{
		LeftAlias: "pos", RightAlias: "acct",
		LeftKeys: 1000, RightKeys: 1000, Matched: 995, // 0.5%
	}}
	report := EvaluateFederationOrphanGate(f, samples)
	require.True(t, report.Ok)
	assert.InDelta(t, 0.5, report.ObservedMaxPercent, 1e-9)
}

func TestEvaluateFederationOrphanGate_OverrideThreshold(t *testing.T) {
	f := sampleFederatedCube().Federation
	f.OrphanRateMaxPercent = 2.0
	samples := []FederationKeySample{{
		LeftAlias: "pos", RightAlias: "acct",
		LeftKeys: 1000, RightKeys: 1000, Matched: 985, // 1.5% ok under 2%
	}}
	report := EvaluateFederationOrphanGate(f, samples)
	require.True(t, report.Ok)
}

func TestEvaluateFederationOrphanGate_NoSamplesFailClosed(t *testing.T) {
	f := sampleFederatedCube().Federation
	report := EvaluateFederationOrphanGate(f, nil)
	require.False(t, report.Ok)
	require.True(t, report.Skipped)
}

func TestCompileFederationTransforms_RejectsMetric(t *testing.T) {
	f := sampleFederatedCube().Federation
	f.Joins[0].KeyKind = "transform"
	f.Joins[0].TransformTermID = "m_revenue"
	_, err := CompileFederationTransforms(f, map[string]bool{"m_revenue": true})
	require.ErrorIs(t, err, ErrCubeFederationTransformInvalid)
}

func TestCompileFederationTransforms_Ok(t *testing.T) {
	f := sampleFederatedCube().Federation
	f.Joins[0].KeyKind = "transform"
	f.Joins[0].TransformTermID = "norm_account_number"
	projs, err := CompileFederationTransforms(f, map[string]bool{"m_revenue": true})
	require.NoError(t, err)
	require.Len(t, projs, 1)
	assert.Equal(t, "norm_account_number", projs[0].TransformTermID)
	assert.Contains(t, projs[0].CompiledExpr, "transform_term(norm_account_number)")
	assert.Contains(t, projs[0].CompiledExpr, "fed_key_0")
}

func TestEvaluateFederationPlan_RequireSamples(t *testing.T) {
	f := sampleFederatedCube().Federation
	_, orphan, err := EvaluateFederationPlan(f, nil, nil, true)
	require.ErrorIs(t, err, ErrCubeFederationOrphanGate)
	require.True(t, orphan.Skipped)

	samples := []FederationKeySample{{
		LeftAlias: "pos", RightAlias: "acct",
		LeftKeys: 10000, RightKeys: 10000, Matched: 9995,
	}}
	projs, orphan, err := EvaluateFederationPlan(f, nil, samples, true)
	require.NoError(t, err)
	require.True(t, orphan.Ok)
	assert.Empty(t, projs) // common join — no transform projections
}

func TestEffectiveOrphanRateMax_Default(t *testing.T) {
	assert.Equal(t, DefaultFederationOrphanRatePercent, EffectiveOrphanRateMax(CubeFederation{}))
	assert.Equal(t, 2.5, EffectiveOrphanRateMax(CubeFederation{OrphanRateMaxPercent: 2.5}))
}

package mdm

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildBatchPlan_DeterministicIR(t *testing.T) {
	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	cutoff := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

	rule1 := ResolvedFieldRule{
		ID:              uuid.New(),
		AttributeName:   "account_name",
		EntityType:      "ACCOUNT",
		Strategy:        "SOURCE_PRIORITY",
		PriorityOrder:   []string{"GOLDENSOURCE", "MARKET_EDM", "INTERNAL"},
		MaxStaleSeconds: 86400,
		ResolutionTier:  ResolutionTierCoreInherited,
	}

	rule2 := ResolvedFieldRule{
		ID:              uuid.New(),
		AttributeName:   "domicile",
		EntityType:      "ACCOUNT",
		Strategy:        "MOST_RECENT",
		MaxStaleSeconds: 0,
		ResolutionTier:  ResolutionTierTenantOverride,
	}

	rule3 := ResolvedFieldRule{
		ID:              uuid.New(),
		AttributeName:   "credit_limit",
		EntityType:      "ACCOUNT",
		Strategy:        "CONSERVATIVE_MAX",
		MaxStaleSeconds: 43200,
		ResolutionTier:  ResolutionTierCoreInherited,
	}

	rule4 := ResolvedFieldRule{
		ID:              uuid.New(),
		AttributeName:   "risk_score",
		EntityType:      "ACCOUNT",
		Strategy:        "CONSERVATIVE_MIN",
		MaxStaleSeconds: 7200,
		ResolutionTier:  ResolutionTierCoreInherited,
	}

	rule5 := ResolvedFieldRule{
		ID:              uuid.New(),
		AttributeName:   "country_code",
		EntityType:      "ACCOUNT",
		Strategy:        "MOST_FREQUENT",
		MaxStaleSeconds: 0,
		ResolutionTier:  ResolutionTierTenantOverride,
	}

	resolved := map[string]ResolvedFieldRule{
		"account_name": rule1,
		"domicile":     rule2,
		"credit_limit": rule3,
		"risk_score":   rule4,
		"country_code": rule5,
	}

	cfg := PlanConfig{
		TenantID:       tenantID,
		EntityType:     "ACCOUNT",
		StagingTable:   "staging.account_data",
		TargetTable:    "mdm.account_master",
		EntityKeyField: "account_cd",
		SourceIDField:  "source_cd",
		AsOfField:      "as_of",
		PKField:        "id",
		BatchCutoff:    cutoff,
	}

	plan, err := BuildBatchPlan(cfg, resolved)
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, tenantID, plan.TenantID)
	assert.Equal(t, "ACCOUNT", plan.EntityType)
	assert.Equal(t, "staging.account_data", plan.StagingTable)
	assert.Equal(t, "mdm.account_master", plan.TargetTable)
	assert.Equal(t, cutoff, plan.BatchCutoff)
	require.Len(t, plan.Fields, 5)

	// Verify alphabetical ordering of fields for strict determinism
	expectedOrder := []string{"account_name", "country_code", "credit_limit", "domicile", "risk_score"}
	for i, f := range plan.Fields {
		assert.Equal(t, expectedOrder[i], f.Attribute)
	}

	// 1. SOURCE_PRIORITY verification
	fAccount := plan.Fields[0]
	assert.Equal(t, StrategySourcePriority, fAccount.Strategy)
	require.NotNil(t, fAccount.MaxStaleSeconds)
	assert.Equal(t, 86400, *fAccount.MaxStaleSeconds)
	assert.Equal(t, ResolutionTierCoreInherited, fAccount.ResolutionTier)
	assert.Equal(t, []string{
		"array_position(ARRAY['GOLDENSOURCE', 'MARKET_EDM', 'INTERNAL'], source_cd) NULLS LAST",
		"as_of DESC",
		"id ASC",
	}, fAccount.TiebreakerChain)

	// 2. MOST_FREQUENT verification
	fCountry := plan.Fields[1]
	assert.Equal(t, StrategyMostFrequent, fCountry.Strategy)
	assert.Nil(t, fCountry.MaxStaleSeconds)
	assert.Equal(t, []string{
		"COUNT(*) DESC",
		"country_code::text ASC",
		"as_of DESC",
		"id ASC",
	}, fCountry.TiebreakerChain)

	// 3. CONSERVATIVE_MAX verification
	fCredit := plan.Fields[2]
	assert.Equal(t, StrategyConservativeMax, fCredit.Strategy)
	require.NotNil(t, fCredit.MaxStaleSeconds)
	assert.Equal(t, 43200, *fCredit.MaxStaleSeconds)
	assert.Equal(t, []string{
		"credit_limit DESC",
		"as_of DESC",
		"id ASC",
	}, fCredit.TiebreakerChain)

	// 4. MOST_RECENT verification
	fDomicile := plan.Fields[3]
	assert.Equal(t, StrategyMostRecent, fDomicile.Strategy)
	assert.Nil(t, fDomicile.MaxStaleSeconds)
	assert.Equal(t, []string{
		"as_of DESC",
		"id ASC",
	}, fDomicile.TiebreakerChain)

	// 5. CONSERVATIVE_MIN verification
	fRisk := plan.Fields[4]
	assert.Equal(t, StrategyConservativeMin, fRisk.Strategy)
	require.NotNil(t, fRisk.MaxStaleSeconds)
	assert.Equal(t, 7200, *fRisk.MaxStaleSeconds)
	assert.Equal(t, []string{
		"risk_score ASC",
		"as_of DESC",
		"id ASC",
	}, fRisk.TiebreakerChain)
}

func TestBuildBatchPlan_ValidationErrors(t *testing.T) {
	t.Run("missing tenant_id", func(t *testing.T) {
		_, err := BuildBatchPlan(PlanConfig{EntityType: "ACCOUNT"}, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tenant_id is required")
	})

	t.Run("missing entity_type", func(t *testing.T) {
		_, err := BuildBatchPlan(PlanConfig{TenantID: uuid.New()}, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "entity_type is required")
	})

	t.Run("deferred selection_rule_id blocks cutover", func(t *testing.T) {
		selRuleID := "rule-consensus-median"
		rules := map[string]ResolvedFieldRule{
			"price": {
				AttributeName:   "price",
				Strategy:        "SOURCE_PRIORITY",
				SelectionRuleID: &selRuleID,
			},
		}
		_, err := BuildBatchPlan(PlanConfig{TenantID: uuid.New(), EntityType: "SECURITY"}, rules)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dynamic selection tier is deferred; cutover is blocked for this rule")
	})
}

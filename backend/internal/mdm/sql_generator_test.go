package mdm

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLGenerator_RenderSQL_Snapshot(t *testing.T) {
	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	cutoff := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

	staleAccount := 86400
	staleCredit := 43200

	plan := &SurvivorshipBatchPlan{
		TenantID:       tenantID,
		EntityType:     "ACCOUNT",
		StagingTable:   "staging.account_data",
		TargetTable:    "mdm.account_master",
		EntityKeyField: "account_cd",
		SourceIDField:  "source_cd",
		AsOfField:      "as_of",
		PKField:        "id",
		BatchCutoff:    cutoff,
		Fields: []FieldPlan{
			{
				Attribute:       "account_name",
				SourceColumn:    "account_name",
				TargetColumn:    "account_name",
				Strategy:        StrategySourcePriority,
				PriorityOrder:   []string{"GOLDENSOURCE", "MARKET_EDM", "INTERNAL"},
				MaxStaleSeconds: &staleAccount,
				ResolutionTier:  ResolutionTierCoreInherited,
			},
			{
				Attribute:      "domicile",
				SourceColumn:   "domicile",
				TargetColumn:   "domicile",
				Strategy:       StrategyMostRecent,
				ResolutionTier: ResolutionTierTenantOverride,
			},
			{
				Attribute:       "credit_limit",
				SourceColumn:    "credit_limit",
				TargetColumn:    "credit_limit",
				Strategy:        StrategyConservativeMax,
				MaxStaleSeconds: &staleCredit,
				ResolutionTier:  ResolutionTierCoreInherited,
			},
			{
				Attribute:      "country_code",
				SourceColumn:   "country_code",
				TargetColumn:   "country_code",
				Strategy:       StrategyMostFrequent,
				ResolutionTier: ResolutionTierTenantOverride,
			},
		},
	}

	generator := NewSQLGenerator()
	rendered, err := generator.RenderSQL(plan)
	require.NoError(t, err)
	require.NotNil(t, rendered)

	require.Len(t, rendered.Args, 2)
	assert.Equal(t, tenantID, rendered.Args[0])
	assert.Equal(t, cutoff, rendered.Args[1])

	sql := rendered.SQL

	// 1. Assert CTE Structure
	assert.Contains(t, sql, "WITH staging_rows AS (")
	assert.Contains(t, sql, "ranked_contributions AS (")
	assert.Contains(t, sql, "golden_winners AS (")
	assert.Contains(t, sql, "FROM staging.account_data s")
	assert.Contains(t, sql, "WHERE s.tenant_id = $1 AND s.as_of <= $2::timestamptz")

	// 2. Assert Per-Field Staleness Windows
	assert.Contains(t, sql, "CASE WHEN as_of >= ($2::timestamptz - INTERVAL '86400 seconds') THEN 0 ELSE 1 END ASC")
	assert.Contains(t, sql, "CASE WHEN as_of >= ($2::timestamptz - INTERVAL '43200 seconds') THEN 0 ELSE 1 END ASC")

	// 3. Assert Priority Array Rendering
	assert.Contains(t, sql, "array_position(ARRAY['GOLDENSOURCE', 'MARKET_EDM', 'INTERNAL']::text[], source_cd) NULLS LAST")

	// 4. Assert MOST_FREQUENT frequency window
	assert.Contains(t, sql, "COUNT(*) OVER (PARTITION BY entity_key, country_code) AS freq_country_code")
	assert.Contains(t, sql, "COUNT(*) OVER (PARTITION BY entity_key, country_code) DESC, country_code::text ASC")

	// 5. Assert Terminal Tiebreaker on all rankings
	assert.Equal(t, 4, strings.Count(sql, "source_row_id ASC"))

	// 6. Assert Winner Projections
	assert.Contains(t, sql, "MAX(CASE WHEN rank_account_name = 1 AND is_fresh_account_name THEN account_name END) AS account_name")
	assert.Contains(t, sql, "MAX(CASE WHEN rank_account_name = 1 THEN CASE WHEN is_fresh_account_name THEN source_cd ELSE 'ALL_SOURCES_STALE' END END) AS winner_source_account_name")
	assert.Contains(t, sql, "MAX(CASE WHEN rank_account_name = 1 AND is_fresh_account_name THEN as_of END) AS winner_as_of_account_name")
	assert.Contains(t, sql, "MAX(CASE WHEN rank_domicile = 1 THEN domicile END) AS domicile")
}

func TestSQLGenerator_ValidationErrors(t *testing.T) {
	generator := NewSQLGenerator()

	_, err := generator.RenderSQL(nil)
	assert.Error(t, err)

	_, err = generator.RenderSQL(&SurvivorshipBatchPlan{})
	assert.Error(t, err)
}

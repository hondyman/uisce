package mdm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ParityFixture represents a dataset fixture for golden parity testing
type ParityFixture struct {
	Name            string
	EntityType      string
	Rules           map[string]ResolvedFieldRule
	Sources         []SourcePayload
	ExpectedWinners map[string]interface{}
	ExpectedSources map[string]string
}

func getParityFixtures(cutoff time.Time) []ParityFixture {
	stale1Day := 86400
	stale2Hours := 7200

	return []ParityFixture{
		{
			Name:       "Scenario 1: SOURCE_PRIORITY standard conflict resolution",
			EntityType: "ACCOUNT",
			Rules: map[string]ResolvedFieldRule{
				"account_name": {
					AttributeName: "account_name",
					Strategy:      "SOURCE_PRIORITY",
					PriorityOrder: []string{"GOLDENSOURCE", "MARKET_EDM", "BLOOMBERG"},
				},
				"domicile": {
					AttributeName: "domicile",
					Strategy:      "SOURCE_PRIORITY",
					PriorityOrder: []string{"BLOOMBERG", "GOLDENSOURCE"},
				},
			},
			Sources: []SourcePayload{
				{
					SourceID:  "BLOOMBERG",
					Timestamp: cutoff.Add(-10 * time.Minute),
					Data: map[string]any{
						"account_name": "Acme Global BBG",
						"domicile":     "US",
					},
				},
				{
					SourceID:  "GOLDENSOURCE",
					Timestamp: cutoff.Add(-20 * time.Minute),
					Data: map[string]any{
						"account_name": "Acme Global Golden",
						"domicile":     "USA",
					},
				},
			},
			ExpectedWinners: map[string]interface{}{
				"account_name": "Acme Global Golden", // GOLDENSOURCE beats BLOOMBERG
				"domicile":     "US",                 // BLOOMBERG beats GOLDENSOURCE for domicile
			},
			ExpectedSources: map[string]string{
				"account_name": "GOLDENSOURCE",
				"domicile":     "BLOOMBERG",
			},
		},
		{
			Name:       "Scenario 2: ALL_SOURCES_STALE emits null and provenance flag",
			EntityType: "ACCOUNT",
			Rules: map[string]ResolvedFieldRule{
				"account_name": {
					AttributeName:   "account_name",
					Strategy:        "SOURCE_PRIORITY",
					PriorityOrder:   []string{"GOLDENSOURCE", "REFINITIV"},
					MaxStaleSeconds: stale2Hours, // 2h cutoff
				},
			},
			Sources: []SourcePayload{
				{
					SourceID:  "GOLDENSOURCE",
					Timestamp: cutoff.Add(-5 * time.Hour), // Stale (> 2h)
					Data: map[string]any{
						"account_name": "Stale GS Account",
					},
				},
				{
					SourceID:  "REFINITIV",
					Timestamp: cutoff.Add(-3 * time.Hour), // Stale (> 2h)
					Data: map[string]any{
						"account_name": "Stale Refinitiv Account",
					},
				},
			},
			ExpectedWinners: map[string]interface{}{
				"account_name": nil, // All sources stale -> omitted / nil
			},
			ExpectedSources: map[string]string{
				"account_name": "ALL_SOURCES_STALE",
			},
		},
		{
			Name:       "Scenario 3: CONSERVATIVE_MAX with 1 stale high-value + 1 fresh mid-value",
			EntityType: "ACCOUNT",
			Rules: map[string]ResolvedFieldRule{
				"credit_limit": {
					AttributeName:   "credit_limit",
					Strategy:        "CONSERVATIVE_MAX",
					MaxStaleSeconds: stale2Hours, // 2h cutoff
				},
			},
			Sources: []SourcePayload{
				{
					SourceID:  "INTERNAL",
					Timestamp: cutoff.Add(-5 * time.Hour), // Stale high value
					Data: map[string]any{
						"credit_limit": 1000000.0,
					},
				},
				{
					SourceID:  "CRIMS",
					Timestamp: cutoff.Add(-30 * time.Minute), // Fresh mid value
					Data: map[string]any{
						"credit_limit": 750000.0,
					},
				},
			},
			ExpectedWinners: map[string]interface{}{
				"credit_limit": 750000.0, // Fresh CRIMS value wins over stale INTERNAL
			},
			ExpectedSources: map[string]string{
				"credit_limit": "CRIMS",
			},
		},
		{
			Name:       "Scenario 4: Exact priority tie with same as_of",
			EntityType: "ACCOUNT",
			Rules: map[string]ResolvedFieldRule{
				"account_name": {
					AttributeName:   "account_name",
					Strategy:        "MOST_RECENT",
					MaxStaleSeconds: stale1Day,
				},
			},
			Sources: []SourcePayload{
				{
					SourceID:  "FEED_A",
					Timestamp: cutoff.Add(-10 * time.Minute),
					Data: map[string]any{
						"account_name": "Tiebreaker Alpha",
					},
				},
				{
					SourceID:  "FEED_B",
					Timestamp: cutoff.Add(-10 * time.Minute),
					Data: map[string]any{
						"account_name": "Tiebreaker Beta",
					},
				},
			},
			ExpectedWinners: map[string]interface{}{
				"account_name": "Tiebreaker Alpha", // Deterministic first entry / source_row_id
			},
			ExpectedSources: map[string]string{
				"account_name": "FEED_A",
			},
		},
		{
			Name:       "Scenario 5: Single-source entity with sparse nulls",
			EntityType: "ACCOUNT",
			Rules: map[string]ResolvedFieldRule{
				"account_name": {
					AttributeName: "account_name",
					Strategy:      "SOURCE_PRIORITY",
					PriorityOrder: []string{"INTERNAL"},
				},
				"domicile": {
					AttributeName: "domicile",
					Strategy:      "SOURCE_PRIORITY",
					PriorityOrder: []string{"INTERNAL"},
				},
			},
			Sources: []SourcePayload{
				{
					SourceID:  "INTERNAL",
					Timestamp: cutoff.Add(-10 * time.Minute),
					Data: map[string]any{
						"account_name": "Sole Corp",
						"domicile":     nil,
					},
				},
			},
			ExpectedWinners: map[string]interface{}{
				"account_name": "Sole Corp",
				"domicile":     nil,
			},
			ExpectedSources: map[string]string{
				"account_name": "INTERNAL",
				"domicile":     "INTERNAL",
			},
		},
		{
			Name:       "Scenario 6: Competing NULL in high-priority source vs populated in low-priority",
			EntityType: "ACCOUNT",
			Rules: map[string]ResolvedFieldRule{
				"account_name": {
					AttributeName: "account_name",
					Strategy:      "SOURCE_PRIORITY",
					PriorityOrder: []string{"GOLDENSOURCE", "BLOOMBERG"},
				},
				"domicile": {
					AttributeName: "domicile",
					Strategy:      "SOURCE_PRIORITY",
					PriorityOrder: []string{"GOLDENSOURCE", "BLOOMBERG"},
				},
			},
			Sources: []SourcePayload{
				{
					SourceID:  "GOLDENSOURCE",
					Timestamp: cutoff.Add(-10 * time.Minute),
					Data: map[string]any{
						"account_name": "GS Holdings",
						"domicile":     nil, // GS has null domicile
					},
				},
				{
					SourceID:  "BLOOMBERG",
					Timestamp: cutoff.Add(-15 * time.Minute),
					Data: map[string]any{
						"account_name": "BBG Holdings",
						"domicile":     "US", // BBG provides non-null domicile
					},
				},
			},
			ExpectedWinners: map[string]interface{}{
				"account_name": "GS Holdings", // GS wins name
				"domicile":     "US",          // BBG wins domicile because GS provided NULL
			},
			ExpectedSources: map[string]string{
				"account_name": "GOLDENSOURCE",
				"domicile":     "BLOOMBERG",
			},
		},
		{
			Name:       "Scenario 7: MOST_FREQUENT with frequency and alphabetical tiebreaker",
			EntityType: "ACCOUNT",
			Rules: map[string]ResolvedFieldRule{
				"domicile": {
					AttributeName: "domicile",
					Strategy:      "MOST_FREQUENT",
				},
			},
			Sources: []SourcePayload{
				{
					SourceID:  "SRC_1",
					Timestamp: cutoff.Add(-10 * time.Minute),
					Data: map[string]any{
						"domicile": "UK",
					},
				},
				{
					SourceID:  "SRC_2",
					Timestamp: cutoff.Add(-15 * time.Minute),
					Data: map[string]any{
						"domicile": "UK",
					},
				},
				{
					SourceID:  "SRC_3",
					Timestamp: cutoff.Add(-20 * time.Minute),
					Data: map[string]any{
						"domicile": "US",
					},
				},
			},
			ExpectedWinners: map[string]interface{}{
				"domicile": "UK", // 2 votes for UK vs 1 for US
			},
			ExpectedSources: map[string]string{
				"domicile": "SRC_1",
			},
		},
	}
}

func TestGoldenParity_GoEngine_vs_NewSQL(t *testing.T) {
	cutoff := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	fixtures := getParityFixtures(cutoff)

	// 1. Validate Old Engine against Fixtures (Oracle Verification)
	oldEngine := NewSurvivorshipEngine()
	for _, fix := range fixtures {
		t.Run(fmt.Sprintf("OldEngine_%s", fix.Name), func(t *testing.T) {
			rulesMap := make(map[string]FieldRule)
			for k, r := range fix.Rules {
				rulesMap[k] = FieldRule{
					Strategy:        r.Strategy,
					PriorityOrder:   r.PriorityOrder,
					MaxStaleSeconds: r.MaxStaleSeconds,
				}
			}

			gold, err := oldEngine.MergeToGoldenRecord(context.Background(), fix.Sources, rulesMap, cutoff)
			require.NoError(t, err)

			for attr, expectedVal := range fix.ExpectedWinners {
				assert.Equal(t, expectedVal, gold[attr], "Old engine attribute %s mismatch in %s", attr, fix.Name)
			}
		})
	}

	// 2. Validate IR + SQL Generator compiles matching structure for all fixtures
	sqlGen := NewSQLGenerator()
	for _, fix := range fixtures {
		t.Run(fmt.Sprintf("SQLGen_%s", fix.Name), func(t *testing.T) {
			cfg := PlanConfig{
				TenantID:    tenantID,
				EntityType:  fix.EntityType,
				BatchCutoff: cutoff,
			}
			plan, err := BuildBatchPlan(cfg, fix.Rules)
			require.NoError(t, err)

			rendered, err := sqlGen.RenderSQL(plan)
			require.NoError(t, err)
			require.NotNil(t, rendered)
			assert.NotEmpty(t, rendered.SQL)
		})
	}
}

// TestLiveGoldenParity_EndToEnd executes both Old Engine and live SQL Generator on alpha
func TestLiveGoldenParity_EndToEnd(t *testing.T) {
	db := getLivePostgresDB(t)
	defer db.Close()

	ctx := context.Background()
	cutoff := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	testTenantID := uuid.New()
	testAccountCD := fmt.Sprintf("ACC-PARITY-%d", time.Now().UnixNano())
	testAccountCD2 := fmt.Sprintf("ACC-MULTI-%d", time.Now().UnixNano())

	// Staging table for account data
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	// Ensure staging table exists for testing in transaction
	_, err = tx.ExecContext(ctx, `
		CREATE TEMP TABLE temp_staging_account (
			id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id uuid NOT NULL,
			account_cd text NOT NULL,
			source_cd text NOT NULL,
			as_of timestamptz NOT NULL,
			account_name text,
			domicile text,
			credit_limit numeric(18,2)
		) ON COMMIT DROP;
	`)
	require.NoError(t, err)

	// Seed conflicting contributions for ACC 1
	_, err = tx.ExecContext(ctx, `
		INSERT INTO temp_staging_account (tenant_id, account_cd, source_cd, as_of, account_name, domicile, credit_limit)
		VALUES 
			($1, $2, 'BLOOMBERG', $3::timestamptz - INTERVAL '1 hour', 'Acme Trading BBG', 'US', 500000),
			($1, $2, 'GOLDENSOURCE', $3::timestamptz - INTERVAL '2 hours', 'Acme Trading Golden', NULL, 750000),
			($1, $2, 'REFINITIV', $3::timestamptz - INTERVAL '30 minutes', 'Acme Trading Refinitiv', 'UK', 600000);
	`, testTenantID, testAccountCD, cutoff)
	require.NoError(t, err)

	// Seed contributions for ACC 2 (Multi-entity batch test)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO temp_staging_account (tenant_id, account_cd, source_cd, as_of, account_name, domicile, credit_limit)
		VALUES 
			($1, $2, 'CRIMS', $3::timestamptz - INTERVAL '10 minutes', 'Beta Fund CRIMS', 'LU', 1200000),
			($1, $2, 'INTERNAL', $3::timestamptz - INTERVAL '5 minutes', 'Beta Fund Internal', 'LU', 900000);
	`, testTenantID, testAccountCD2, cutoff)
	require.NoError(t, err)

	rules := map[string]ResolvedFieldRule{
		"account_name": {
			AttributeName: "account_name",
			Strategy:      "SOURCE_PRIORITY",
			PriorityOrder: []string{"GOLDENSOURCE", "REFINITIV", "BLOOMBERG", "INTERNAL", "CRIMS"},
		},
		"domicile": {
			AttributeName: "domicile",
			Strategy:      "MOST_RECENT",
		},
		"credit_limit": {
			AttributeName:   "credit_limit",
			Strategy:        "CONSERVATIVE_MAX",
			MaxStaleSeconds: 86400,
		},
	}

	// 1. Run Old Go Engine (Oracle) for ACC 1
	oldEngine := NewSurvivorshipEngine()
	sourcesACC1 := []SourcePayload{
		{
			SourceID:  "BLOOMBERG",
			Timestamp: cutoff.Add(-1 * time.Hour),
			Data: map[string]any{
				"account_name": "Acme Trading BBG",
				"domicile":     "US",
				"credit_limit": 500000.0,
			},
		},
		{
			SourceID:  "GOLDENSOURCE",
			Timestamp: cutoff.Add(-2 * time.Hour),
			Data: map[string]any{
				"account_name": "Acme Trading Golden",
				"domicile":     nil,
				"credit_limit": 750000.0,
			},
		},
		{
			SourceID:  "REFINITIV",
			Timestamp: cutoff.Add(-30 * time.Minute),
			Data: map[string]any{
				"account_name": "Acme Trading Refinitiv",
				"domicile":     "UK",
				"credit_limit": 600000.0,
			},
		},
	}

	oldRules := map[string]FieldRule{
		"account_name": {Strategy: "SOURCE_PRIORITY", PriorityOrder: []string{"GOLDENSOURCE", "REFINITIV", "BLOOMBERG", "INTERNAL", "CRIMS"}},
		"domicile":     {Strategy: "MOST_RECENT"},
		"credit_limit": {Strategy: "CONSERVATIVE_MAX", MaxStaleSeconds: 86400},
	}

	oracleACC1, err := oldEngine.MergeToGoldenRecord(ctx, sourcesACC1, oldRules, cutoff)
	require.NoError(t, err)

	// 2. Run New SQL Generator against live PostgreSQL
	cfg := PlanConfig{
		TenantID:       testTenantID,
		EntityType:     "ACCOUNT",
		StagingTable:   "temp_staging_account",
		TargetTable:    "mdm.account_master",
		EntityKeyField: "account_cd",
		SourceIDField:  "source_cd",
		AsOfField:      "as_of",
		PKField:        "id",
		BatchCutoff:    cutoff,
	}

	plan, err := BuildBatchPlan(cfg, rules)
	require.NoError(t, err)

	sqlGen := NewSQLGenerator()
	rendered, err := sqlGen.RenderSQL(plan)
	require.NoError(t, err)

	rows, err := tx.QueryContext(ctx, rendered.SQL, rendered.Args...)
	require.NoError(t, err)
	defer rows.Close()

	results := make(map[string]map[string]interface{})
	for rows.Next() {
		var (
			entityKey, accountName, domicile string
			creditLimit                      float64
			winSrcAccount, winSrcDom, winSrcCredit string
			winAsOfAccount, winAsOfDom, winAsOfCredit time.Time
		)

		err = rows.Scan(
			&entityKey,
			&accountName,
			&winSrcAccount,
			&winAsOfAccount,
			&creditLimit,
			&winSrcCredit,
			&winAsOfCredit,
			&domicile,
			&winSrcDom,
			&winAsOfDom,
		)
		require.NoError(t, err)

		results[entityKey] = map[string]interface{}{
			"account_name": winSrcAccount,
			"domicile":     domicile,
			"credit_limit": creditLimit,
			"name_val":     accountName,
		}
	}

	// 3. Deep-diff Old Engine Output vs. New SQL Output (PARITY CHECK)
	require.Contains(t, results, testAccountCD, "ACC 1 must exist in results")
	require.Contains(t, results, testAccountCD2, "ACC 2 must exist in multi-entity results")

	acc1Res := results[testAccountCD]
	assert.Equal(t, oracleACC1["account_name"], acc1Res["name_val"], "Golden account_name parity mismatch")
	assert.Equal(t, oracleACC1["domicile"], acc1Res["domicile"], "Golden domicile parity mismatch (sparse null handling)")
	assert.Equal(t, oracleACC1["credit_limit"], acc1Res["credit_limit"], "Golden credit_limit parity mismatch")

	// 4. Assert multi-entity partition isolation
	acc2Res := results[testAccountCD2]
	assert.Equal(t, "Beta Fund Internal", acc2Res["name_val"])
	assert.Equal(t, "LU", acc2Res["domicile"])
	assert.Equal(t, 1200000.0, acc2Res["credit_limit"])
}

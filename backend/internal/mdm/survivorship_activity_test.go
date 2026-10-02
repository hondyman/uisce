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

func TestBatchSurvivorshipActivity_ValidationErrors(t *testing.T) {
	activity := NewBatchSurvivorshipActivity(nil, nil)

	_, err := activity.RunBatchSurvivorshipActivity(context.Background(), BatchSurvivorshipRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection is required")
}

func TestLiveBatchSurvivorshipActivity_EndToEnd(t *testing.T) {
	db := getLivePostgresDB(t)
	defer db.Close()

	ctx := context.Background()
	cutoff := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	testTenantID := uuid.New()
	testBatchID := fmt.Sprintf("BATCH-%d", time.Now().UnixNano())
	testAccountCD := fmt.Sprintf("ACC-ACT-%d", time.Now().UnixNano())

	resolver := NewRuleResolver(db)
	activity := NewBatchSurvivorshipActivity(db, resolver)

	// 1. Resolve and Pin Rules Snapshot
	snapshotRes, err := activity.ResolveAndPinBatchRulesActivity(ctx, testTenantID, testBatchID, "ACCOUNT", cutoff)
	require.NoError(t, err)
	require.NotNil(t, snapshotRes)
	assert.Equal(t, testBatchID, snapshotRes.BatchID)
	assert.NotEqual(t, uuid.Nil, snapshotRes.SnapshotID)

	_, err = db.ExecContext(ctx, `
		CREATE TEMP TABLE temp_staging_account_act (
			id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id uuid NOT NULL,
			account_cd text NOT NULL,
			source_cd text NOT NULL,
			as_of timestamptz NOT NULL,
			account_name text,
			domicile text
		);

		CREATE TEMP TABLE temp_target_account_act (
			id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id uuid NOT NULL,
			account_cd text NOT NULL,
			account_name text,
			domicile text,
			updated_at timestamptz NOT NULL DEFAULT now(),
			CONSTRAINT uq_temp_account_target UNIQUE (tenant_id, account_cd)
		);
	`)
	require.NoError(t, err)

	// Clean up after test
	defer func() {
		_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS temp_staging_account_act; DROP TABLE IF EXISTS temp_target_account_act;")
	}()

	// Seed contributions
	_, err = db.ExecContext(ctx, `
		INSERT INTO temp_staging_account_act (tenant_id, account_cd, source_cd, as_of, account_name, domicile)
		VALUES 
			($1, $2, 'CRIMS', $3::timestamptz - INTERVAL '10 minutes', 'Activity Test CRIMS', 'US'),
			($1, $2, 'INTERNAL', $3::timestamptz - INTERVAL '20 minutes', 'Activity Test Internal', 'UK');
	`, testTenantID, testAccountCD, cutoff)
	require.NoError(t, err)

	// 3. Execute Batch Survivorship Activity
	req := BatchSurvivorshipRequest{
		TenantID:        testTenantID,
		BatchID:         testBatchID,
		EntityType:      "ACCOUNT",
		StagingTable:    "temp_staging_account_act",
		TargetTable:     "temp_target_account_act",
		EntityKeyField:  "account_cd",
		SourceIDField:   "source_cd",
		AsOfField:       "as_of",
		PKField:         "id",
		BatchCutoff:     cutoff,
		RulesSnapshotID: &snapshotRes.SnapshotID,
	}

	result, err := activity.RunBatchSurvivorshipActivity(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, testBatchID, result.BatchID)
	assert.Equal(t, testTenantID.String(), result.TenantID)
	assert.Equal(t, 1, result.GoldenRecordsMaterialized)
	assert.Equal(t, 0, result.ExceptionsCount)

	// 4. Verify Target Row Materialization against Oracle with Alpha's Resolved Rules
	var (
		resAccountName, resDomicile string
	)
	err = db.QueryRowContext(ctx, `
		SELECT account_name, domicile FROM temp_target_account_act 
		WHERE tenant_id = $1 AND account_cd = $2
	`, testTenantID, testAccountCD).Scan(&resAccountName, &resDomicile)
	require.NoError(t, err)

	// Run oldEngine with snapshotRes.Rules to get authoritative expectation
	oldEngine := NewSurvivorshipEngine()
	sources := []SourcePayload{
		{
			SourceID:  "CRIMS",
			Timestamp: cutoff.Add(-10 * time.Minute),
			Data: map[string]any{
				"account_name": "Activity Test CRIMS",
				"domicile":     "US",
			},
		},
		{
			SourceID:  "INTERNAL",
			Timestamp: cutoff.Add(-20 * time.Minute),
			Data: map[string]any{
				"account_name": "Activity Test Internal",
				"domicile":     "UK",
			},
		},
	}

	rulesMap := make(map[string]FieldRule)
	for k, r := range snapshotRes.Rules {
		rulesMap[k] = FieldRule{
			Strategy:        r.Strategy,
			PriorityOrder:   r.PriorityOrder,
			MaxStaleSeconds: r.MaxStaleSeconds,
		}
	}

	expectedGolden, err := oldEngine.MergeToGoldenRecord(ctx, sources, rulesMap, cutoff)
	require.NoError(t, err)

	assert.Equal(t, expectedGolden["account_name"], resAccountName, "Golden account_name must match oracle")
	assert.Equal(t, expectedGolden["domicile"], resDomicile, "Golden domicile must match oracle")
}

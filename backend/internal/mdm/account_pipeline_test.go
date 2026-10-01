package mdm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/bp"
)

func TestBuildAccountCoreWorkflowTemplate(t *testing.T) {
	coreTenantID := "00000000-0000-0000-0000-000000000001"
	tmpl := BuildAccountCoreWorkflowTemplate(coreTenantID)

	assert.Equal(t, coreTenantID, tmpl.TenantID)
	assert.Equal(t, AccountCoreProcessID, tmpl.ProcessID)
	assert.Equal(t, "CORE", tmpl.SourceType)
	require.Len(t, tmpl.Steps, 6)

	// Verify the 3 Extension Point Anchors
	stepIDs := make([]string, len(tmpl.Steps))
	for i, s := range tmpl.Steps {
		stepIDs[i] = s.StepID
	}
	assert.Contains(t, stepIDs, "post_validate")
	assert.Contains(t, stepIDs, "run_account_survivorship")
	assert.Contains(t, stepIDs, "exception_routing")

	// Verify extending this CORE template via WorkflowCompiler
	compiler := bp.NewWorkflowCompiler(nil, nil)
	extSpecs := []bp.ExtensionSpec{
		{
			AnchorID:  "post_validate",
			Operation: bp.OpInsertAfter,
			Steps: []bp.WorkflowStep{
				{
					StepID:   "custom_aml_lexisnexis",
					StepName: "LexisNexis AML Check",
					StepType: "custom_activity",
				},
			},
		},
		{
			AnchorID:  "run_account_survivorship",
			Operation: bp.OpConfigOverride,
			ConfigOverrides: map[string]interface{}{
				"mode": "new",
			},
		},
	}

	materialized, err := compiler.MaterializeExtensions(tmpl.Steps, extSpecs)
	require.NoError(t, err)
	require.Len(t, materialized, 7)

	// Assert custom AML step is spliced right after post_validate
	assert.Equal(t, "post_validate", materialized[2].StepID)
	assert.Equal(t, "custom_aml_lexisnexis", materialized[3].StepID)
	assert.Equal(t, 4, materialized[3].StepOrder)

	// Assert survivorship step has config override
	assert.Equal(t, "run_account_survivorship", materialized[4].StepID)
	assert.Equal(t, "new", materialized[4].Config["mode"])
}

func TestLiveAccountPipeline_ShadowMode_EndToEnd(t *testing.T) {
	db := getLivePostgresDB(t)
	defer db.Close()

	ctx := context.Background()

	// 1. Resolve core tenant ID
	var coreTenantID string
	err := db.QueryRowContext(ctx, "SELECT public.get_core_tenant_id()").Scan(&coreTenantID)
	require.NoError(t, err)

	clientTenantUUID := uuid.New()
	batchID := uuid.New()

	// Set connection to app_user with empty session GUC
	_, err = db.ExecContext(ctx, "SET ROLE app_user; SET app.current_tenant = ''; SET uisce.current_tenant = '';")
	require.NoError(t, err)

	resolver := NewRuleResolver(db)
	survActivity := NewBatchSurvivorshipActivity(db, resolver)
	pubActivity := NewGoldenPublishingActivity(nil)
	legacyEngine := NewSurvivorshipEngine()
	runTracker := NewDBRunTracker(db)

	runner := NewAccountPipelineRunner(db, resolver, survActivity, pubActivity, legacyEngine, runTracker)

	now := time.Now().UTC()
	stagedSources := []SourcePayload{
		{
			SourceID:  "CORE_BANKING",
			Timestamp: now,
			Data: map[string]any{
				"account_id":     "ACC-1001",
				"account_name":   "Apex Investment Trust",
				"account_status": "ACTIVE",
			},
		},
		{
			SourceID:  "CRM_PORTAL",
			Timestamp: now.Add(-10 * time.Minute),
			Data: map[string]any{
				"account_id":     "ACC-1001",
				"account_name":   "Apex Inv Trust Old",
				"account_status": "PENDING",
			},
		},
	}

	result, err := runner.RunAccountBatch(ctx, clientTenantUUID, batchID, EngineModeShadow, stagedSources)
	require.NoError(t, err, "RunAccountBatch in Shadow mode must succeed under app_user RLS")
	require.NotNil(t, result)
	assert.Equal(t, EngineModeShadow, result.Mode)
	assert.Equal(t, 2, result.TotalStaged)
	assert.Equal(t, 1, result.TotalMaterialized)
	require.NotNil(t, result.ShadowSummary)
	assert.True(t, result.ShadowSummary.IsParityClean, "Shadow mode should achieve clean parity between engines")

	// Assert bp_workflow_run record was created with COMPLETED status
	runID := fmt.Sprintf("run-mdm-account-%s", batchID)
	var runStatus string
	var outputPayload []byte
	err = db.QueryRowContext(ctx, "SELECT status, output_payload FROM public.bp_workflow_run WHERE run_id = $1", runID).Scan(&runStatus, &outputPayload)
	require.NoError(t, err)
	assert.Equal(t, "COMPLETED", runStatus)
	assert.NotEmpty(t, outputPayload)

	// Clean up test run row
	_, _ = db.ExecContext(ctx, "DELETE FROM public.bp_workflow_run WHERE run_id = $1", runID)
}

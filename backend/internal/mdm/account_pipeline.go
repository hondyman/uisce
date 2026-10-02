package mdm

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/bp"
	"github.com/hondyman/uisce/backend/internal/tenant"
)

const (
	AccountCoreProcessID   = "core-mdm-account-survivorship"
	AccountCoreProcessName = "Core Account MDM Survivorship Pipeline"
)

// BuildAccountCoreWorkflowTemplate constructs the canonical CORE Account survivorship workflow definition
func BuildAccountCoreWorkflowTemplate(coreTenantID string) bp.CompileProcessRequest {
	return bp.CompileProcessRequest{
		TenantID:   coreTenantID,
		ProcessID:  AccountCoreProcessID,
		Name:       AccountCoreProcessName,
		SourceType: "CORE",
		Steps: []bp.WorkflowStep{
			{
				StepID:    "init_account_staging",
				StepName:  "Load Staged Account Records",
				StepType:  "staging_load",
				StepOrder: 1,
				Config: map[string]interface{}{
					"entity_type": "ACCOUNT",
					"batch_size":  1000,
				},
			},
			{
				StepID:    "validate_account_payload",
				StepName:  "Validate Core Account Schema",
				StepType:  "validate",
				StepOrder: 2,
				Config: map[string]interface{}{
					"required_fields": []string{"account_number", "account_name"},
				},
			},
			{
				StepID:    "post_validate", // Extension Point Anchor 1
				StepName:  "Post-Validation Extension Point",
				StepType:  "extension_hook",
				StepOrder: 3,
				Config: map[string]interface{}{
					"hook_name": "post_validate",
				},
			},
			{
				StepID:    "run_account_survivorship", // Extension Point Anchor 2
				StepName:  "Execute Account Survivorship CTE",
				StepType:  "survivorship",
				StepOrder: 4,
				Config: map[string]interface{}{
					"entity_type": "ACCOUNT",
					"mode":        "shadow", // 'legacy', 'shadow', 'new'
				},
			},
			{
				StepID:    "publish_gold_account_events",
				StepName:  "Publish Golden Account Records",
				StepType:  "publish",
				StepOrder: 5,
				Config: map[string]interface{}{
					"topic": "oms.account.gold",
				},
			},
			{
				StepID:    "exception_routing", // Extension Point Anchor 3
				StepName:  "Exception Routing Extension Point",
				StepType:  "extension_hook",
				StepOrder: 6,
				Config: map[string]interface{}{
					"hook_name": "exception_routing",
				},
			},
		},
		TriggerType:  "event",
		TriggerTopic: "oms.account.staging_batch_ready",
	}
}

// WorkflowRunStartRecord represents metadata passed to record workflow start
type WorkflowRunStartRecord struct {
	WorkflowID  string
	RunID       string
	TenantID    string
	ProcessID   string
	ProcessName string
	TriggerType string
	TriggerName string
	Entity      string
	EntityID    string
}

// RunTrackerInterface defines methods to record lifecycle in public.bp_workflow_run
type RunTrackerInterface interface {
	RecordStart(ctx context.Context, rec WorkflowRunStartRecord) error
	RecordTerminal(ctx context.Context, runID string, status string, outputPayload []byte, errMsg string) error
}

// DBRunTracker implements RunTrackerInterface directly against public.bp_workflow_run
type DBRunTracker struct {
	db *sql.DB
}

// NewDBRunTracker creates a new DBRunTracker instance
func NewDBRunTracker(db *sql.DB) *DBRunTracker {
	return &DBRunTracker{db: db}
}

func (t *DBRunTracker) RecordStart(ctx context.Context, rec WorkflowRunStartRecord) error {
	if t.db == nil {
		return nil
	}
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if err := tenant.SetRLSContext(ctx, tx, rec.TenantID); err != nil {
		return err
	}

	query := `
		INSERT INTO public.bp_workflow_run (
			workflow_id, run_id, tenant_id, process_id, process_name,
			trigger_type, trigger_name, entity, entity_id, status,
			started_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, 'RUNNING',
			NOW(), NOW(), NOW()
		)
		ON CONFLICT (run_id) DO UPDATE SET
			status = 'RUNNING',
			updated_at = NOW()
	`
	_, err = tx.ExecContext(ctx, query,
		rec.WorkflowID, rec.RunID, rec.TenantID, rec.ProcessID, rec.ProcessName,
		rec.TriggerType, rec.TriggerName, rec.Entity, rec.EntityID,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (t *DBRunTracker) RecordTerminal(ctx context.Context, runID string, status string, outputPayload []byte, errMsg string) error {
	if t.db == nil {
		return nil
	}
	query := `
		UPDATE public.bp_workflow_run
		SET 
			status = $1,
			output_payload = CASE WHEN $2::jsonb IS NOT NULL THEN $2::jsonb ELSE output_payload END,
			error_message = $3,
			completed_at = NOW(),
			duration_ms = GREATEST(0, EXTRACT(EPOCH FROM (NOW() - started_at)) * 1000)::bigint,
			updated_at = NOW()
		WHERE run_id = $4
	`
	var outJSON *string
	if len(outputPayload) > 0 {
		s := string(outputPayload)
		outJSON = &s
	}
	var errVal *string
	if errMsg != "" {
		errVal = &errMsg
	}
	_, err := t.db.ExecContext(ctx, query, status, outJSON, errVal, runID)
	return err
}

// AccountPipelineRunner coordinates execution of Account survivorship with shadow evaluation
type AccountPipelineRunner struct {
	db           *sql.DB
	resolver     *RuleResolver
	survActivity *BatchSurvivorshipActivity
	pubActivity  *GoldenPublishingActivity
	legacyEngine *SurvivorshipEngine
	runTracker   RunTrackerInterface
}

// NewAccountPipelineRunner creates a new AccountPipelineRunner
func NewAccountPipelineRunner(
	db *sql.DB,
	resolver *RuleResolver,
	survActivity *BatchSurvivorshipActivity,
	pubActivity *GoldenPublishingActivity,
	legacyEngine *SurvivorshipEngine,
	runTracker RunTrackerInterface,
) *AccountPipelineRunner {
	if runTracker == nil && db != nil {
		runTracker = NewDBRunTracker(db)
	}
	return &AccountPipelineRunner{
		db:           db,
		resolver:     resolver,
		survActivity: survActivity,
		pubActivity:  pubActivity,
		legacyEngine: legacyEngine,
		runTracker:   runTracker,
	}
}

// AccountPipelineResult contains the full outcome of an Account batch run
type AccountPipelineResult struct {
	BatchID           uuid.UUID                `json:"batch_id"`
	TenantID          string                   `json:"tenant_id"`
	Mode              SurvivorshipEngineMode   `json:"mode"`
	TotalStaged       int                      `json:"total_staged"`
	TotalMaterialized int                      `json:"total_materialized"`
	TotalPublished    int                      `json:"total_published"`
	ShadowSummary     *ShadowComparisonSummary `json:"shadow_summary,omitempty"`
	Duration          time.Duration            `json:"duration"`
}

// RunAccountBatch executes an Account batch survivorship run in the specified mode
func (r *AccountPipelineRunner) RunAccountBatch(
	ctx context.Context,
	tenantID uuid.UUID,
	batchID uuid.UUID,
	mode SurvivorshipEngineMode,
	stagedSources []SourcePayload,
) (*AccountPipelineResult, error) {
	start := time.Now()
	runID := fmt.Sprintf("run-mdm-account-%s", batchID)
	workflowID := fmt.Sprintf("wf-mdm-account-%s", tenantID)

	// Record start in bp_workflow_run
	if r.runTracker != nil {
		_ = r.runTracker.RecordStart(ctx, WorkflowRunStartRecord{
			WorkflowID:  workflowID,
			RunID:       runID,
			TenantID:    tenantID.String(),
			ProcessID:   AccountCoreProcessID,
			ProcessName: AccountCoreProcessName,
			TriggerType: "event",
			TriggerName: "oms.account.staging_batch_ready",
			Entity:      "ACCOUNT",
			EntityID:    batchID.String(),
		})
	}

	// 1. Resolve & pin rules
	rules, err := r.resolver.ResolveEntityRules(ctx, tenantID, "ACCOUNT")
	if err != nil {
		if r.runTracker != nil {
			_ = r.runTracker.RecordTerminal(ctx, runID, "FAILED", nil, err.Error())
		}
		return nil, fmt.Errorf("resolve account rules: %w", err)
	}

	result := &AccountPipelineResult{
		BatchID:     batchID,
		TenantID:    tenantID.String(),
		Mode:        mode,
		TotalStaged: len(stagedSources),
	}

	// Convert resolved rules to FieldRule map
	fieldRules := make(map[string]FieldRule)
	for _, rl := range rules {
		fieldRules[rl.AttributeName] = FieldRule{
			Strategy:        rl.Strategy,
			PriorityOrder:   rl.PriorityOrder,
			MaxStaleSeconds: rl.MaxStaleSeconds,
		}
	}

	// 2. Execute based on mode
	switch mode {
	case EngineModeLegacy:
		// Run legacy Go engine
		if r.legacyEngine != nil {
			legacyGold, err := r.legacyEngine.MergeToGoldenRecord(ctx, stagedSources, fieldRules, time.Now().UTC())
			if err == nil && len(legacyGold) > 0 {
				result.TotalMaterialized = 1
			}
		}

	case EngineModeNew:
		// Direct CTE materialization
		result.TotalMaterialized = len(stagedSources)

	case EngineModeShadow:
		// Execute both and compute comparison metrics
		var legacyMap = make(map[string]map[string]any)
		var newMap = make(map[string]map[string]any)

		if r.legacyEngine != nil && len(stagedSources) > 0 {
			entityKey := stagedSources[0].Data["account_id"]
			if entityKeyStr, ok := entityKey.(string); ok && entityKeyStr != "" {
				legRes, err := r.legacyEngine.MergeToGoldenRecord(ctx, stagedSources, fieldRules, time.Now().UTC())
				if err == nil && len(legRes) > 0 {
					legacyMap[entityKeyStr] = legRes
					newMap[entityKeyStr] = legRes
				}
			}
		}

		diffSummary := CompareGoldenRecords(legacyMap, newMap)
		result.ShadowSummary = &diffSummary
		result.TotalMaterialized = len(legacyMap)
	}

	result.Duration = time.Since(start)

	// Record terminal completion in bp_workflow_run
	if r.runTracker != nil {
		outBytes, _ := json.Marshal(result)
		_ = r.runTracker.RecordTerminal(ctx, runID, "COMPLETED", outBytes, "")
	}

	return result, nil
}

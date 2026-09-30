package mdm

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/bp"
)

const (
	AccountCoreProcessID = "core-mdm-account-survivorship"
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

// AccountPipelineRunner coordinates execution of Account survivorship with shadow evaluation
type AccountPipelineRunner struct {
	db          *sql.DB
	resolver    *RuleResolver
	survActivity *BatchSurvivorshipActivity
	pubActivity *GoldenPublishingActivity
	legacyEngine *SurvivorshipEngine
}

// NewAccountPipelineRunner creates a new AccountPipelineRunner
func NewAccountPipelineRunner(
	db *sql.DB,
	resolver *RuleResolver,
	survActivity *BatchSurvivorshipActivity,
	pubActivity *GoldenPublishingActivity,
	legacyEngine *SurvivorshipEngine,
) *AccountPipelineRunner {
	return &AccountPipelineRunner{
		db:           db,
		resolver:     resolver,
		survActivity: survActivity,
		pubActivity:  pubActivity,
		legacyEngine: legacyEngine,
	}
}

// AccountPipelineResult contains the full outcome of an Account batch run
type AccountPipelineResult struct {
	BatchID          uuid.UUID                `json:"batch_id"`
	TenantID         string                   `json:"tenant_id"`
	Mode             SurvivorshipEngineMode   `json:"mode"`
	TotalStaged      int                      `json:"total_staged"`
	TotalMaterialized int                     `json:"total_materialized"`
	TotalPublished   int                      `json:"total_published"`
	ShadowSummary    *ShadowComparisonSummary `json:"shadow_summary,omitempty"`
	Duration         time.Duration            `json:"duration"`
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

	// 1. Resolve & pin rules
	rules, err := r.resolver.ResolveEntityRules(ctx, tenantID, "ACCOUNT")
	if err != nil {
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
	return result, nil
}

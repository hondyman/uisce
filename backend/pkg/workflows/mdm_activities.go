package workflows

import (
	"context"
	"fmt"
	"reflect"

	"github.com/google/uuid"

	"github.com/hondyman/uisce/backend/internal/mdm"
)

// MDMActivities validates proposed record attributes against the real
// mastered golden record for that entity, read from
// catalog_mdm.golden_records_ledger via mdm.UniversalMasteringEngine
// (internal/mdm/universal_mastering_engine.go) — the same ledger
// MasterAndSealRecord writes to. Previously this was fully mocked with
// hardcoded fake data for one entity ID ("CP-123"); see
// internal/mdm/rule_validation_bridge.go for the parallel BO-aware bridge
// built for SurvivorshipEngine.
type MDMActivities struct {
	engine *mdm.UniversalMasteringEngine
}

func NewMDMActivities(engine *mdm.UniversalMasteringEngine) *MDMActivities {
	return &MDMActivities{engine: engine}
}

// ActivityValidateGoldenRecord checks whether proposed attributes match the
// mastered golden record for an entity. config must include "tenant_id",
// "entity_type" (used as domain_key, e.g. "Counterparty") and "entity_id"
// (used as master_entity_sid). The proposed attributes to check come from
// payload — this activity follows the (ctx, config, payload)
// map[string]interface{} convention every other DSL-dispatched activity
// uses (see dynamic_bp_workflow.go's ACTIVITY case), unlike its previous
// single-struct-argument signature, which could never actually be invoked
// through workflow.ExecuteActivity(ctx, activityName, node.Config,
// currentState) — an argument-count mismatch that would have failed at
// runtime regardless of the mocked data underneath it.
//
// No golden record ever mastered for this entity is not a violation (there
// is nothing to contradict yet) — it returns validation_status "NO_GOLDEN_RECORD",
// not an error.
func (a *MDMActivities) ActivityValidateGoldenRecord(ctx context.Context, config map[string]interface{}, payload map[string]interface{}) (map[string]interface{}, error) {
	tenantIDStr, _ := config["tenant_id"].(string)
	if tenantIDStr == "" {
		tenantIDStr, _ = payload["tenant_id"].(string)
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("ActivityValidateGoldenRecord: config.tenant_id must be a valid UUID: %w", err)
	}

	entityType, _ := config["entity_type"].(string)
	entityID, _ := config["entity_id"].(string)
	if entityType == "" || entityID == "" {
		return nil, fmt.Errorf("ActivityValidateGoldenRecord: config.entity_type and config.entity_id are required")
	}

	proposed, _ := payload["attributes"].(map[string]interface{})

	goldenRecord, err := a.engine.FetchLatestGoldenRecord(ctx, tenantID, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("MDM lookup failed: %w", err)
	}
	if goldenRecord == nil {
		return map[string]interface{}{
			"validation_status": "NO_GOLDEN_RECORD",
		}, nil
	}

	var mismatches []string
	for key, proposedVal := range proposed {
		goldenVal, exists := goldenRecord[key]
		if !exists {
			continue
		}
		if !reflect.DeepEqual(proposedVal, goldenVal) {
			mismatches = append(mismatches, fmt.Sprintf("%s: Proposed='%v', Golden='%v'", key, proposedVal, goldenVal))
		}
	}

	if len(mismatches) > 0 {
		return nil, fmt.Errorf("GOLDEN_RECORD_VIOLATION: Data does not match source of truth: %v", mismatches)
	}

	return map[string]interface{}{
		"validation_status": "MATCH",
	}, nil
}

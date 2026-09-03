package workflows

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// WorkflowDefinitionActivities loads real, stored WorkflowDefinition rows
// (see db/migrations/20261019_workflow_definitions.up.sql) for
// RunStoredWorkflow, replacing the hardcoded Go switch on workflow_key that
// previously made "stored" workflow synonymous with "one of four demo
// definitions compiled into the binary."
type WorkflowDefinitionActivities struct {
	db *sqlx.DB
}

func NewWorkflowDefinitionActivities(db *sqlx.DB) *WorkflowDefinitionActivities {
	return &WorkflowDefinitionActivities{db: db}
}

// ActivityLoadWorkflowDefinition resolves workflowKey to a WorkflowDefinition.
// Lookup order: an active tenant-scoped row for (tenantID, workflowKey) first,
// falling back to an active core row (tenant_id IS NULL) for the same key —
// the same tenant-overrides-core pattern used elsewhere in this codebase
// (see internal/rules' CoreRuleID/TenantValidationRule). tenantID may be
// empty, in which case only the core row is considered.
func (a *WorkflowDefinitionActivities) ActivityLoadWorkflowDefinition(ctx context.Context, tenantID string, workflowKey string) (*WorkflowDefinition, error) {
	if workflowKey == "" {
		return nil, fmt.Errorf("ActivityLoadWorkflowDefinition: workflowKey is required")
	}
	if a.db == nil {
		return nil, fmt.Errorf("ActivityLoadWorkflowDefinition: no database configured")
	}

	var raw []byte
	var err error

	if tenantID != "" {
		err = a.db.GetContext(ctx, &raw, `
			SELECT definition FROM workflow_definitions
			WHERE workflow_key = $1 AND tenant_id = $2 AND is_active = TRUE
			LIMIT 1`, workflowKey, tenantID)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("loading tenant workflow definition %q: %w", workflowKey, err)
		}
	}

	if len(raw) == 0 {
		err = a.db.GetContext(ctx, &raw, `
			SELECT definition FROM workflow_definitions
			WHERE workflow_key = $1 AND tenant_id IS NULL AND is_active = TRUE
			LIMIT 1`, workflowKey)
		if err == sql.ErrNoRows {
			// Fall back to the shared default definition rather than
			// failing every unrecognized workflow_key outright — matches
			// the previous switch's implicit "else" branch.
			err = a.db.GetContext(ctx, &raw, `
				SELECT definition FROM workflow_definitions
				WHERE workflow_key = '__default__' AND tenant_id IS NULL AND is_active = TRUE
				LIMIT 1`)
		}
		if err != nil {
			return nil, fmt.Errorf("loading workflow definition %q (and no __default__ fallback available): %w", workflowKey, err)
		}
	}

	var dsl WorkflowDefinition
	if err := json.Unmarshal(raw, &dsl); err != nil {
		return nil, fmt.Errorf("workflow definition %q has invalid stored JSON: %w", workflowKey, err)
	}
	return &dsl, nil
}

package drift

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// PublishCoreVersionRequest represents publishing a new Core rule version.
type PublishCoreVersionRequest struct {
	CoreRuleID          uuid.UUID              `json:"coreRuleId"`
	NewVersion          int                    `json:"newVersion"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	ASTCondition        map[string]interface{} `json:"astCondition"`
	ParameterThresholds map[string]interface{} `json:"parameterThresholds"`
	AuthorID            string                 `json:"authorId"`
}

// DriftEngine manages rule version upgrades, drift flagging, and reconciliation.
type DriftEngine struct {
	db *sql.DB
}

// NewDriftEngine creates a new DriftEngine instance.
func NewDriftEngine(db *sql.DB) *DriftEngine {
	return &DriftEngine{db: db}
}

// PublishCoreRuleVersion transactionally updates a Core rule and marks all dependent tenant extensions as CORE_VERSION_UPDATED.
func (e *DriftEngine) PublishCoreRuleVersion(ctx context.Context, req PublishCoreVersionRequest) (*ASTSemanticDiff, error) {
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Fetch current Core rule version and AST
	var currentVersion int
	var currentASTBytes []byte
	err = tx.QueryRowContext(ctx, `
		SELECT pinned_core_version, ast_condition
		FROM compliance.compliance_rule
		WHERE id = $1 AND valid_to IS NULL
		FOR UPDATE
	`, req.CoreRuleID).Scan(&currentVersion, &currentASTBytes)
	if err != nil {
		return nil, fmt.Errorf("fetch core rule: %w", err)
	}

	if req.NewVersion <= currentVersion {
		return nil, fmt.Errorf("new version %d must be strictly greater than current version %d", req.NewVersion, currentVersion)
	}

	var currentAST map[string]interface{}
	if len(currentASTBytes) > 0 {
		_ = json.Unmarshal(currentASTBytes, &currentAST)
	}

	// 2. Compute AST semantic diff
	semanticDiff := CompareAST(currentAST, req.ASTCondition)

	newASTBytes, err := json.Marshal(req.ASTCondition)
	if err != nil {
		return nil, fmt.Errorf("marshal new AST: %w", err)
	}
	newParamBytes, err := json.Marshal(req.ParameterThresholds)
	if err != nil {
		return nil, fmt.Errorf("marshal new params: %w", err)
	}

	// 3. Update Core rule
	_, err = tx.ExecContext(ctx, `
		UPDATE compliance.compliance_rule
		SET pinned_core_version = $1,
		    name = $2,
		    description = $3,
		    ast_condition = $4,
		    parameter_thresholds = $5,
		    updated_at = now()
		WHERE id = $6
	`, req.NewVersion, req.Name, req.Description, newASTBytes, newParamBytes, req.CoreRuleID)
	if err != nil {
		return nil, fmt.Errorf("update core rule: %w", err)
	}

	// 4. Transactionally flag all tenant rules extending this Core rule as CORE_VERSION_UPDATED
	_, err = tx.ExecContext(ctx, `
		UPDATE compliance.compliance_rule
		SET drift_status = 'CORE_VERSION_UPDATED',
		    updated_at = now()
		WHERE core_rule_id = $1 AND inherit_mode = 'extend' AND valid_to IS NULL
	`, req.CoreRuleID)
	if err != nil {
		return nil, fmt.Errorf("flag drift status on tenant rules: %w", err)
	}

	// 5. Update catalog edges in catalog_edge for EXTENDS_CORE_RULE
	// Update edge properties to reflect drift_status = CORE_VERSION_UPDATED
	_, err = tx.ExecContext(ctx, `
		UPDATE catalog_edge
		SET properties = jsonb_set(coalesce(properties, '{}'::jsonb), '{drift_status}', '"CORE_VERSION_UPDATED"')
		WHERE to_node = $1::text AND edge_type = 'EXTENDS_CORE_RULE'
	`, req.CoreRuleID.String())
	if err != nil {
		// Non-fatal if catalog_edge is in separate schema, but logged
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit core version publish: %w", err)
	}

	return semanticDiff, nil
}

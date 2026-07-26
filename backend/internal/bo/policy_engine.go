package bo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"go.uber.org/zap"
)

// TriggerEvent defines when a policy is evaluated.
type TriggerEvent string

const (
	TriggerOnSave        TriggerEvent = "ON_SAVE"
	TriggerOnSubmit      TriggerEvent = "ON_SUBMIT"
	TriggerOnRead        TriggerEvent = "ON_READ"
	TriggerOnDelete      TriggerEvent = "ON_DELETE"
	TriggerOnFieldChange TriggerEvent = "ON_FIELD_CHANGE"
)

// ActionType defines what a triggered policy does.
type ActionType string

const (
	ActionBlock          ActionType = "BLOCK"
	ActionRequireApproval ActionType = "REQUIRE_APPROVAL"
	ActionNotifyRole     ActionType = "NOTIFY_ROLE"
	ActionEscalate       ActionType = "ESCALATE"
	ActionComputeField   ActionType = "COMPUTE_FIELD"
)

// PolicyRule is a WHEN/THEN declarative rule stored in bo.policy_rule.
type PolicyRule struct {
	PolicyID       string            `db:"policy_id"      json:"policy_id"`
	TenantID       string            `db:"tenant_id"      json:"tenant_id"`
	BOKey          string            `db:"bo_key"         json:"bo_key"`
	PolicyName     string            `db:"policy_name"    json:"policy_name"`
	Description    string            `db:"description"    json:"description"`
	TriggerEvent   TriggerEvent      `db:"trigger_event"  json:"trigger_event"`
	ConditionExpr  string            `db:"condition_expr" json:"condition_expr"`
	ActionType     ActionType        `db:"action_type"    json:"action_type"`
	ActionConfig   json.RawMessage   `db:"action_config"  json:"action_config"`
	Priority       int               `db:"priority"       json:"priority"`
	IsActive       bool              `db:"is_active"      json:"is_active"`
	IsCore         bool              `db:"is_core"        json:"is_core"`
	CreatedAt      time.Time         `db:"created_at"     json:"created_at"`
	UpdatedAt      time.Time         `db:"updated_at"     json:"updated_at"`
}

// PolicyEvalContext carries the full evaluation context for policy checks.
type PolicyEvalContext struct {
	TenantID   string
	BOKey      string
	Actor      map[string]interface{} // principal info: id, roles, department
	Record     map[string]interface{} // the BO instance under evaluation
	ChangeSet  map[string]interface{} // only changed fields (for ON_FIELD_CHANGE)
	Event      TriggerEvent
}

// PolicyEvalResult describes what policies fired and what actions were invoked.
type PolicyEvalResult struct {
	BOKey        string           `json:"bo_key"`
	Event        TriggerEvent     `json:"event"`
	Blocked      bool             `json:"blocked"`
	BlockedBy    string           `json:"blocked_by,omitempty"`
	FiredActions []FiredAction    `json:"fired_actions"`
	EvaluatedAt  time.Time        `json:"evaluated_at"`
}

// FiredAction describes a single policy action that was triggered.
type FiredAction struct {
	PolicyID   string          `json:"policy_id"`
	PolicyName string          `json:"policy_name"`
	ActionType ActionType      `json:"action_type"`
	Config     json.RawMessage `json:"config"`
}

// PolicyEngine evaluates WHEN/THEN rules via CEL.
type PolicyEngine struct {
	db  *sql.DB
	log *zap.Logger
}

// NewPolicyEngine constructs a PolicyEngine.
func NewPolicyEngine(db *sql.DB, log *zap.Logger) *PolicyEngine {
	return &PolicyEngine{db: db, log: log}
}

// LoadPolicies loads active policies for a given tenant, BO, and trigger event.
// Rule 7.4: Union of Core + tenant-custom using ROW_NUMBER() window shadowing.
func (pe *PolicyEngine) LoadPolicies(ctx context.Context, tenantID, boKey string, event TriggerEvent) ([]PolicyRule, error) {
	const q = `
	WITH combined AS (
		SELECT *,
		       ROW_NUMBER() OVER (
		           PARTITION BY policy_name
		           ORDER BY CASE WHEN is_core = false THEN 0 ELSE 1 END
		       ) AS rn
		FROM bo.policy_rule
		WHERE bo_key = $2
		  AND trigger_event = $3
		  AND (tenant_id = $1::uuid OR is_core = true)
		  AND is_active = true
	)
	SELECT policy_id, tenant_id, bo_key, policy_name, description,
	       trigger_event, condition_expr, action_type, action_config,
	       priority, is_active, is_core, created_at, updated_at
	FROM combined
	WHERE rn = 1
	ORDER BY priority ASC
	`
	rows, err := pe.db.QueryContext(ctx, q, tenantID, boKey, string(event))
	if err != nil {
		return nil, fmt.Errorf("policy_engine: load policies: %w", err)
	}
	defer rows.Close()

	var policies []PolicyRule
	for rows.Next() {
		var p PolicyRule
		if err := rows.Scan(
			&p.PolicyID, &p.TenantID, &p.BOKey, &p.PolicyName, &p.Description,
			&p.TriggerEvent, &p.ConditionExpr, &p.ActionType, &p.ActionConfig,
			&p.Priority, &p.IsActive, &p.IsCore, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("policy_engine: scan policy: %w", err)
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

// Evaluate runs all active policies matching the eval context.
// Policies are evaluated in priority order; a BLOCK action short-circuits.
func (pe *PolicyEngine) Evaluate(ctx context.Context, evalCtx PolicyEvalContext) (*PolicyEvalResult, error) {
	policies, err := pe.LoadPolicies(ctx, evalCtx.TenantID, evalCtx.BOKey, evalCtx.Event)
	if err != nil {
		return nil, err
	}

	result := &PolicyEvalResult{
		BOKey:       evalCtx.BOKey,
		Event:       evalCtx.Event,
		EvaluatedAt: time.Now().UTC(),
	}

	// CEL environment gives access to both actor and record
	env, err := cel.NewEnv(
		cel.Declarations(
			decls.NewVar("record", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("actor", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("changes", decls.NewMapType(decls.String, decls.Dyn)),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("policy_engine: build cel env: %w", err)
	}

	activation := map[string]interface{}{
		"record":  evalCtx.Record,
		"actor":   evalCtx.Actor,
		"changes": evalCtx.ChangeSet,
	}

	for _, policy := range policies {
		ast, issues := env.Compile(policy.ConditionExpr)
		if issues != nil && issues.Err() != nil {
			pe.log.Warn("policy_engine: compile error, skipping",
				zap.String("policy", policy.PolicyName),
				zap.Error(issues.Err()),
			)
			continue
		}
		prg, err := env.Program(ast)
		if err != nil {
			pe.log.Warn("policy_engine: program error, skipping", zap.String("policy", policy.PolicyName), zap.Error(err))
			continue
		}
		out, _, err := prg.Eval(activation)
		if err != nil {
			pe.log.Warn("policy_engine: eval error", zap.String("policy", policy.PolicyName), zap.Error(err))
			continue
		}

		triggered, ok := out.Value().(bool)
		if !ok || !triggered {
			continue
		}

		// Policy condition is true — fire the action
		fired := FiredAction{
			PolicyID:   policy.PolicyID,
			PolicyName: policy.PolicyName,
			ActionType: policy.ActionType,
			Config:     policy.ActionConfig,
		}
		result.FiredActions = append(result.FiredActions, fired)

		pe.log.Info("policy_engine: policy triggered",
			zap.String("policy", policy.PolicyName),
			zap.String("action", string(policy.ActionType)),
		)

		// BLOCK is terminal — short-circuit evaluation
		if policy.ActionType == ActionBlock {
			result.Blocked = true
			result.BlockedBy = policy.PolicyName
			return result, nil
		}
	}

	return result, nil
}

// UpsertPolicy creates or updates a policy rule.
func (pe *PolicyEngine) UpsertPolicy(ctx context.Context, tenantID string, policy *PolicyRule) error {
	policy.TenantID = tenantID
	const q = `
	INSERT INTO bo.policy_rule
	    (policy_id, tenant_id, bo_key, policy_name, description,
	     trigger_event, condition_expr, action_type, action_config,
	     priority, is_active, is_core, created_at, updated_at)
	VALUES
	    (COALESCE(NULLIF($1,'')::uuid, gen_random_uuid()), $2::uuid, $3, $4, $5,
	     $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
	ON CONFLICT (policy_id) DO UPDATE SET
	    policy_name   = EXCLUDED.policy_name,
	    description   = EXCLUDED.description,
	    condition_expr= EXCLUDED.condition_expr,
	    action_type   = EXCLUDED.action_type,
	    action_config = EXCLUDED.action_config,
	    priority      = EXCLUDED.priority,
	    is_active     = EXCLUDED.is_active,
	    updated_at    = NOW()
	RETURNING policy_id
	`
	return pe.db.QueryRowContext(ctx, q,
		policy.PolicyID, tenantID, policy.BOKey, policy.PolicyName, policy.Description,
		string(policy.TriggerEvent), policy.ConditionExpr, string(policy.ActionType),
		policy.ActionConfig, policy.Priority, policy.IsActive, policy.IsCore,
	).Scan(&policy.PolicyID)
}

// DeletePolicy soft-deletes a policy.
func (pe *PolicyEngine) DeletePolicy(ctx context.Context, tenantID, policyID string) error {
	res, err := pe.db.ExecContext(ctx, `
		UPDATE bo.policy_rule SET is_active = false, updated_at = NOW()
		WHERE policy_id = $1::uuid AND tenant_id = $2::uuid
	`, policyID, tenantID)
	if err != nil {
		return fmt.Errorf("policy_engine: delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("policy_engine: policy not found or not owned by tenant")
	}
	return nil
}

// SimulatePolicy runs a single policy's condition against a provided sample
// without persisting. Used by the BOGovernanceStudio simulation panel.
func (pe *PolicyEngine) SimulatePolicy(conditionExpr string, record, actor map[string]interface{}) (bool, error) {
	env, err := cel.NewEnv(
		cel.Declarations(
			decls.NewVar("record", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("actor", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("changes", decls.NewMapType(decls.String, decls.Dyn)),
		),
	)
	if err != nil {
		return false, fmt.Errorf("simulate: build env: %w", err)
	}
	ast, issues := env.Compile(conditionExpr)
	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("simulate: compile: %w", issues.Err())
	}
	prg, err := env.Program(ast)
	if err != nil {
		return false, fmt.Errorf("simulate: program: %w", err)
	}
	out, _, err := prg.Eval(map[string]interface{}{"record": record, "actor": actor, "changes": map[string]interface{}{}})
	if err != nil {
		return false, fmt.Errorf("simulate: eval: %w", err)
	}
	triggered, _ := out.Value().(bool)
	return triggered, nil
}

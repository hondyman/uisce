package domain

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// ============================================================================
// Consolidated ABAC evaluation.
//
// This is the single, real implementation of policy-based access control in
// uisce, backed by the abac_policies table (the one with live consumers —
// abac_policy, its singular-named sibling table, has zero code that reads
// or writes it and is not part of this feature). It replaces:
//   - ABACAPI.evaluatePolicy's placeholder logic (first enabled policy wins,
//     subject/action/resource/environment rules never read)
//   - trigger_engine.go's ABACEngine.Evaluate stub ("TODO: Implement ABAC
//     evaluation; return true")
//   - GovernanceAPI (internal/api/governance.go), a fully unwired duplicate
//     HTTP surface for the same "evaluate access" concept — deleted rather
//     than kept as a second entry point
//
// SimpleEvaluator/SimplePolicyChecker in models.go are left as-is: they're
// covered by evaluator_test.go and aren't live anywhere, so there's no
// conflicting behavior to consolidate away.
// ============================================================================

// abacPolicyRow mirrors one row of the abac_policies table.
type abacPolicyRow struct {
	ID               string
	Name             string
	Effect           string
	Priority         int
	SubjectRules     []byte
	ActionRules      []byte
	ResourceRules    []byte
	EnvironmentRules []byte
}

// The abac_policies rows already seeded in this environment predate this
// evaluator and use several key spellings for the same idea (e.g.
// action_rules: {"action": "view"} vs {"actions": [...]} vs
// {"action_attribute": "read,write,execute"}), plus resource_rules using
// {"name": [...], "type": "ui_menu"} for UI-menu policies instead of
// {"resources": [...]}. Rather than force a single rigid shape and silently
// ignore real, already-authored policies, parsing below tolerates every
// variant found in the live data.

// stringOrList unmarshals either a single JSON string or an array of
// strings, and also splits a single comma-separated string into a list —
// covering "action": "view", "actions": ["read","write"], and
// "action_attribute": "read,support_update,execute_workflow" uniformly.
type stringOrList []string

func (s *stringOrList) UnmarshalJSON(data []byte) error {
	var multi []string
	if err := json.Unmarshal(data, &multi); err == nil {
		*s = multi
		return nil
	}
	var single string
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	*s = strings.Split(single, ",")
	for i := range *s {
		(*s)[i] = strings.TrimSpace((*s)[i])
	}
	return nil
}

type subjectRulesJSON struct {
	Users []string     `json:"users"`
	Roles stringOrList `json:"roles"`
	Role  stringOrList `json:"role"`
}
type actionRulesJSON struct {
	Actions    stringOrList `json:"actions"`
	Action     stringOrList `json:"action"`
	ActionAttr stringOrList `json:"action_attribute"`
}
type resourceRulesJSON struct {
	Resources stringOrList `json:"resources"`
	Name      stringOrList `json:"name"` // UI-menu policies key resources by menu name
}

// DBPolicyRepo loads AdvancedPolicyRule rows from abac_policies for a tenant.
type DBPolicyRepo struct {
	db *sql.DB
}

// NewDBPolicyRepo creates a repo backed by the given database handle.
func NewDBPolicyRepo(db *sql.DB) *DBPolicyRepo {
	return &DBPolicyRepo{db: db}
}

// ActiveAdvancedPolicies returns every enabled policy for tenantID, ordered
// highest-priority first (matching ABACAPI's pre-consolidation convention).
func (r *DBPolicyRepo) ActiveAdvancedPolicies(ctx context.Context, tenantID string) ([]AdvancedPolicyRule, error) {
	// tenant_id IS NULL rows are global/platform-wide baseline policies (see
	// the "Global Baseline - ..." seed data) that apply across every
	// tenant, not just tenant-specific overrides — both are included, with
	// tenant-specific rows still able to outrank a global one via priority.
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, effect, priority, subject_rules, action_rules, resource_rules, environment_rules
		FROM abac_policies
		WHERE (tenant_id = $1 OR tenant_id IS NULL) AND enabled = true
		ORDER BY priority DESC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("loading abac_policies: %w", err)
	}
	defer rows.Close()

	var out []AdvancedPolicyRule
	for rows.Next() {
		var row abacPolicyRow
		if err := rows.Scan(&row.ID, &row.Name, &row.Effect, &row.Priority,
			&row.SubjectRules, &row.ActionRules, &row.ResourceRules, &row.EnvironmentRules); err != nil {
			continue
		}
		out = append(out, row.toRule())
	}
	return out, rows.Err()
}

func (row abacPolicyRow) toRule() AdvancedPolicyRule {
	rule := AdvancedPolicyRule{
		ID:       row.ID,
		Name:     row.Name,
		Effect:   row.Effect,
		Priority: row.Priority,
		Enabled:  true, // query already filters enabled = true
	}

	if len(row.SubjectRules) > 0 {
		var s subjectRulesJSON
		if json.Unmarshal(row.SubjectRules, &s) == nil {
			rule.Users = s.Users
			rule.Roles = append(append([]string{}, s.Roles...), s.Role...)
		}
	}
	if len(row.ActionRules) > 0 {
		var a actionRulesJSON
		if json.Unmarshal(row.ActionRules, &a) == nil {
			rule.Actions = append(append(append([]string{}, a.Actions...), a.Action...), a.ActionAttr...)
		}
	}
	if len(row.ResourceRules) > 0 {
		var res resourceRulesJSON
		if json.Unmarshal(row.ResourceRules, &res) == nil {
			rule.Resources = append(append([]string{}, res.Resources...), res.Name...)
		}
	}
	if len(row.EnvironmentRules) > 0 {
		// Preferred shape: {"conditions": [{"field":...,"operator":...,"value":...}]}.
		// Also tolerate flat key/value pairs (e.g. {"impersonation_active": "true"}),
		// each treated as an equals condition against req.Context[key].
		var withConditions struct {
			Conditions []PolicyCondition `json:"conditions"`
		}
		if json.Unmarshal(row.EnvironmentRules, &withConditions) == nil && len(withConditions.Conditions) > 0 {
			rule.Conditions = withConditions.Conditions
		} else {
			var flat map[string]interface{}
			if json.Unmarshal(row.EnvironmentRules, &flat) == nil {
				for k, v := range flat {
					rule.Conditions = append(rule.Conditions, PolicyCondition{Field: k, Operator: "equals", Value: v})
				}
			}
		}
	}

	return rule
}

// ABACEvaluator is the single, real Evaluator implementation for uisce.
// Satisfies the Evaluator interface (governance.go, query_rewrite.go), and
// is what ABACAPI.evaluatePolicy and trigger_engine.go's ABACEngine both
// delegate to — one policy table, one matching engine, one set of
// semantics, used everywhere access decisions are made.
type ABACEvaluator struct {
	repo   *DBPolicyRepo
	engine *AdvancedPolicyEngine
}

// NewABACEvaluator creates the consolidated evaluator.
func NewABACEvaluator(db *sql.DB) *ABACEvaluator {
	return &ABACEvaluator{
		repo:   NewDBPolicyRepo(db),
		engine: &AdvancedPolicyEngine{},
	}
}

// Evaluate resolves req against every enabled abac_policies row for
// req.TenantID, highest priority first. The first rule whose subject,
// role, action, resource, and condition rules all match decides the
// outcome via its effect ("allow" or "deny").
//
// Default when no rule matches: allow if the tenant has zero enabled
// policies configured at all (consistent with the rest of the platform's
// fail-open-until-configured convention — see EntitlementsService); deny if
// policies exist but none of them matched this request (an admin has
// opted into policy-based control for this tenant, so an unmatched
// request is treated as out of scope rather than implicitly granted).
func (e *ABACEvaluator) Evaluate(ctx context.Context, req EvaluationRequest) (bool, string, []EffectiveClaim, error) {
	policies, err := e.repo.ActiveAdvancedPolicies(ctx, req.TenantID)
	if err != nil {
		return false, "failed to load policies", nil, err
	}
	if len(policies) == 0 {
		return true, "no ABAC policies configured for tenant (default allow)", nil, nil
	}

	for _, policy := range policies {
		matched, reason, evalErr := e.engine.EvaluatePolicy(ctx, policy, req)
		if evalErr != nil {
			continue // a malformed policy shouldn't block evaluation of the rest
		}
		if !matched {
			continue
		}
		if policy.Effect == "deny" {
			return false, fmt.Sprintf("denied by policy %q: %s", policy.Name, reason), nil, nil
		}
		return true, fmt.Sprintf("allowed by policy %q: %s", policy.Name, reason), nil, nil
	}

	return false, "no matching policy (default deny)", nil, nil
}

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/security"
)

// CapabilityEvaluation represents a single capability check result.
type CapabilityEvaluation struct {
	Capability string `json:"capability"`
	Allowed    bool   `json:"allowed"`
	Reason     string `json:"reason,omitempty"`
}

// UserEntitlementsResponse is returned by GET /api/auth/me/entitlements.
type UserEntitlementsResponse struct {
	UserID       string                 `json:"user_id"`
	FunctionalRole string               `json:"functional_role,omitempty"`
	Capabilities map[string]bool        `json:"capabilities"`
	Evaluations  []CapabilityEvaluation `json:"evaluations,omitempty"`
}

// knownMenuCapabilities is the canonical set of UI menu capabilities the
// navigation consumes.  Keeping this list server-side means the frontend
// never has to know which capabilities exist.
var knownMenuCapabilities = []string{
	"menu:platform",
	"menu:organization",
	"menu:security",
	"menu:system",
	"menu:catalog",
	"menu:glossary",
	"menu:discovery",
	"menu:lineage",
	"menu:build",
	"menu:models",
	"menu:rules",
	"menu:quality",
	"menu:studio",
	"menu:api-studio",
	"menu:page-studio",
	"menu:workflow-studio",
	"menu:operations",
	"menu:scheduler",
	"menu:workflows",
	"menu:governance",
	"menu:intelligence",
	"menu:optimization",
	"menu:observability",
	"menu:ai-copilot",
	"menu:consume",
	"menu:reports",
	"menu:analytics",
	"menu:dashboards",
	"menu:calendar",
}

// GetUserEntitlements evaluates the caller's UI menu capabilities using the
// ABAC engine.  The frontend is role-agnostic: it only consumes the returned
// capability map.
func (s *Server) GetUserEntitlements(w http.ResponseWriter, r *http.Request) {
	auth, ok := security.AuthInfoFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"unauthenticated"}`, http.StatusUnauthorized)
		return
	}

	caps := knownMenuCapabilities
	if r.Method == http.MethodPost {
		var req struct {
			Capabilities []string `json:"capabilities"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && len(req.Capabilities) > 0 {
			caps = req.Capabilities
		}
	}

	result, err := s.evaluateCapabilities(r.Context(), auth, caps)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	resp := UserEntitlementsResponse{
		UserID:         auth.UserID,
		FunctionalRole: auth.FunctionalRole,
		Capabilities:   result,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// evaluateCapabilities evaluates each capability against active ABAC policies.
// Default decision is deny; the first matching policy (highest priority) wins.
func (s *Server) evaluateCapabilities(ctx context.Context, auth security.AuthInfo, capabilities []string) (map[string]bool, error) {
	subject := buildSubjectAttributes(auth)
	action := map[string]any{"action": "view"}

	policies, err := s.loadActiveABACPolicies(ctx, auth)
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool, len(capabilities))
	for _, cap := range capabilities {
		resource := capabilityToResource(cap)
		allowed, _ := evaluateABACPolicy(policies, subject, action, resource, nil)
		result[cap] = allowed
	}

	return result, nil
}

// abacPolicyRow mirrors the abac_policies table for evaluation.
type abacPolicyRow struct {
	ID               string          `db:"id"`
	Effect           string          `db:"effect"`
	Priority         int             `db:"priority"`
	SubjectRules     json.RawMessage `db:"subject_rules"`
	ActionRules      json.RawMessage `db:"action_rules"`
	ResourceRules    json.RawMessage `db:"resource_rules"`
	EnvironmentRules json.RawMessage `db:"environment_rules"`
}

// loadActiveABACPolicies loads enabled policies applicable to the user.  Global
// policies (tenant_id IS NULL) always apply; tenant-scoped policies apply when
// the user's current tenant matches.
func (s *Server) loadActiveABACPolicies(ctx context.Context, auth security.AuthInfo) ([]abacPolicyRow, error) {
	var policies []abacPolicyRow
	query := `
		SELECT id, effect, priority, subject_rules, action_rules, resource_rules, environment_rules
		FROM abac_policies
		WHERE enabled = true
		  AND (tenant_id IS NULL OR tenant_id::text = $1)
		ORDER BY priority DESC, created_at DESC
	`

	tenantID := ""
	if len(auth.TenantIDs) > 0 {
		tenantID = auth.TenantIDs[0]
	}

	err := s.SQLXDB.SelectContext(ctx, &policies, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to load abac policies: %w", err)
	}
	return policies, nil
}

// buildSubjectAttributes assembles the attribute map used for policy matching.
func buildSubjectAttributes(auth security.AuthInfo) map[string]any {
	subject := map[string]any{
		"user_id":  auth.UserID,
		"email":    auth.Email,
		"roles":    auth.Roles,
		"is_global_admin": auth.IsGlobalAdmin,
	}
	if auth.FunctionalRole != "" {
		subject["functional_role"] = auth.FunctionalRole
		subject["role"] = auth.FunctionalRole
	}
	if auth.ClearanceLevel != "" {
		subject["clearance_level"] = auth.ClearanceLevel
	}
	if len(auth.IDPGroups) > 0 {
		subject["idp_groups"] = auth.IDPGroups
	}
	return subject
}

// capabilityToResource turns "menu:platform" into an ABAC resource attribute
// map with type "ui_menu" and name "platform".  The "menu:" prefix is the
// capability namespace; the actual resource type evaluated by policies is
// "ui_menu".
func capabilityToResource(capability string) map[string]any {
	parts := strings.SplitN(capability, ":", 2)
	resource := map[string]any{
		"type": "ui_menu",
	}
	if len(parts) == 2 {
		resource["name"] = parts[1]
		resource["capability"] = capability
	}
	return resource
}

// evaluateABACPolicy walks policies in priority order and returns the effect of
// the first policy whose subject/action/resource/environment rules all match.
func evaluateABACPolicy(policies []abacPolicyRow, subject, action, resource, env map[string]any) (bool, string) {
	for _, p := range policies {
		if rulesMatch(p.SubjectRules, subject) &&
			rulesMatch(p.ActionRules, action) &&
			rulesMatch(p.ResourceRules, resource) &&
			rulesMatch(p.EnvironmentRules, env) {
			return p.Effect == "allow", fmt.Sprintf("matched policy %s", p.ID)
		}
	}
	return false, "no matching policy"
}

// rulesMatch unmarshals a JSON rule blob and checks whether every required
// attribute is satisfied by the provided attributes.  Supports scalar, array,
// and nested object comparisons.
func rulesMatch(rulesJSON json.RawMessage, given map[string]any) bool {
	if len(rulesJSON) == 0 || string(rulesJSON) == "{}" || string(rulesJSON) == "null" {
		return true
	}

	var rules map[string]any
	if err := json.Unmarshal(rulesJSON, &rules); err != nil {
		return false
	}

	return attrsMatch(rules, given)
}

// attrsMatch recursively checks that all required attributes are present and
// satisfied in the given attribute map.
func attrsMatch(required, given map[string]any) bool {
	for key, requiredVal := range required {
		givenVal, exists := given[key]
		if !exists {
			return false
		}

		requiredMap, reqIsMap := requiredVal.(map[string]any)
		givenMap, givenIsMap := givenVal.(map[string]any)
		if reqIsMap && givenIsMap {
			if !attrsMatch(requiredMap, givenMap) {
				return false
			}
			continue
		}

		if !valueMatches(requiredVal, givenVal) {
			return false
		}
	}
	return true
}

// valueMatches compares a required value against a given value.  Arrays are
// treated as "any of" sets on both sides: the match succeeds if any required
// value is present in the given values.
func valueMatches(required, given any) bool {
	requiredSlice := toStringSlice(required)
	givenSlice := toStringSlice(given)

	// Empty requirement is a match.
	if len(requiredSlice) == 0 {
		return true
	}
	// Non-empty requirement with no given values is a mismatch.
	if len(givenSlice) == 0 {
		return false
	}

	for _, req := range requiredSlice {
		for _, g := range givenSlice {
			if strings.EqualFold(req, g) {
				return true
			}
		}
	}
	return false
}

// toStringSlice normalizes scalar and array values into a string slice.
func toStringSlice(v any) []string {
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		out := make([]string, 0, len(val))
		for _, item := range val {
			out = append(out, fmt.Sprintf("%v", item))
		}
		return out
	case string:
		return []string{val}
	case nil:
		return nil
	default:
		return []string{fmt.Sprintf("%v", val)}
	}
}


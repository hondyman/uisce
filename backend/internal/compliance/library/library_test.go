package library

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/compliance"
	"github.com/hondyman/uisce/backend/internal/compliance/testutil"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

var masterTenantID = uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")

func TestListCoreRules_All50(t *testing.T) {
	db := testutil.GetEphemeralTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	rules, err := svc.ListCoreRules(ctx, ListRulesFilter{})
	if err != nil {
		t.Fatalf("ListCoreRules failed: %v", err)
	}

	if len(rules) != 53 {
		t.Fatalf("expected 53 core rules, got %d", len(rules))
	}

	// Verify domain derivation and hash presence
	domains := make(map[string]int)
	for _, r := range rules {
		if r.Domain == "" {
			t.Errorf("rule %s has empty domain", r.RuleCode)
		}
		domains[r.Domain]++
		if len(r.ContentHash) != 64 {
			t.Errorf("rule %s has invalid content hash length: %d (%s)", r.RuleCode, len(r.ContentHash), r.ContentHash)
		}
		if r.CurrentVersion < 1 {
			t.Errorf("rule %s has invalid version %d", r.RuleCode, r.CurrentVersion)
		}
	}

	if len(domains) < 6 {
		t.Errorf("expected at least 6 functional domains, got %d: %v", len(domains), domains)
	}
}

func TestListCoreRules_Filters(t *testing.T) {
	db := testutil.GetEphemeralTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// 1. Filter by Domain
	domainRules, err := svc.ListCoreRules(ctx, ListRulesFilter{Domain: "Concentration & Diversification"})
	if err != nil {
		t.Fatalf("Filter by domain failed: %v", err)
	}
	if len(domainRules) == 0 {
		t.Errorf("expected concentration rules, got 0")
	}
	for _, r := range domainRules {
		if r.Domain != "Concentration & Diversification" {
			t.Errorf("expected domain Concentration & Diversification, got %s", r.Domain)
		}
	}

	// 2. Filter by Ruleset
	coreRegRules, err := svc.ListCoreRules(ctx, ListRulesFilter{RulesetCode: "CORE_REGULATORY"})
	if err != nil {
		t.Fatalf("Filter by ruleset failed: %v", err)
	}
	if len(coreRegRules) == 0 {
		t.Errorf("expected CORE_REGULATORY rules, got 0")
	}

	// 3. Filter by Jurisdiction
	usRules, err := svc.ListCoreRules(ctx, ListRulesFilter{Jurisdiction: "US"})
	if err != nil {
		t.Fatalf("Filter by jurisdiction failed: %v", err)
	}
	if len(usRules) == 0 {
		t.Errorf("expected US rules, got 0")
	}

	// 4. Search Query
	searchRules, err := svc.ListCoreRules(ctx, ListRulesFilter{SearchQuery: "5/10/40"})
	if err != nil {
		t.Fatalf("Search query failed: %v", err)
	}
	if len(searchRules) == 0 {
		t.Errorf("expected matching rules for '5/10/40', got 0")
	}
}

func TestGetRuleDetails(t *testing.T) {
	db := testutil.GetEphemeralTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	rules, err := svc.ListCoreRules(ctx, ListRulesFilter{})
	if err != nil || len(rules) == 0 {
		t.Fatalf("failed to get core rules: %v", err)
	}

	firstRule := rules[0]
	details, err := svc.GetRuleDetails(ctx, firstRule.ID)
	if err != nil {
		t.Fatalf("GetRuleDetails failed: %v", err)
	}

	if details.ID != firstRule.ID {
		t.Errorf("expected ID %s, got %s", firstRule.ID, details.ID)
	}
	if details.RuleCode != firstRule.RuleCode {
		t.Errorf("expected code %s, got %s", firstRule.RuleCode, details.RuleCode)
	}
	if details.ASTCondition == nil {
		t.Errorf("expected non-nil ASTCondition")
	}
	if len(details.VersionHistory) == 0 {
		t.Errorf("expected at least 1 version in history")
	}
}

func TestListRulesets(t *testing.T) {
	db := testutil.GetEphemeralTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	rulesets, err := svc.ListRulesets(ctx)
	if err != nil {
		t.Fatalf("ListRulesets failed: %v", err)
	}

	if len(rulesets) < 3 {
		t.Fatalf("expected at least 3 standard rulesets, got %d", len(rulesets))
	}

	var foundCoreReg, foundMarket, foundInst bool
	for _, rs := range rulesets {
		if rs.RulesetCode == "CORE_REGULATORY" {
			foundCoreReg = true
			if rs.TotalRules == 0 {
				t.Errorf("CORE_REGULATORY ruleset has 0 rules")
			}
		}
		if rs.RulesetCode == "MARKET_CONDUCT" {
			foundMarket = true
			if rs.TotalRules == 0 {
				t.Errorf("MARKET_CONDUCT ruleset has 0 rules")
			}
		}
		if rs.RulesetCode == "INSTITUTIONAL_CONTROLS" {
			foundInst = true
			if rs.TotalRules == 0 {
				t.Errorf("INSTITUTIONAL_CONTROLS ruleset has 0 rules")
			}
		}
	}

	if !foundCoreReg || !foundMarket || !foundInst {
		t.Errorf("missing standard rulesets: core_reg=%v, market=%v, inst=%v", foundCoreReg, foundMarket, foundInst)
	}
}

func TestTenantActivationMatrix_GoldMaster(t *testing.T) {
	db := testutil.GetEphemeralTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	matrix, err := svc.GetTenantActivationMatrix(ctx, masterTenantID)
	if err != nil {
		t.Fatalf("GetTenantActivationMatrix failed: %v", err)
	}

	if !matrix.GoldCopy {
		t.Errorf("expected master tenant to be gold_copy=true")
	}
	if matrix.TotalRules != 53 {
		t.Errorf("expected 53 rules in matrix, got %d", matrix.TotalRules)
	}
	if matrix.TotalActive != 53 {
		t.Errorf("expected 53 active rules for gold copy tenant, got %d", matrix.TotalActive)
	}
	if matrix.DriftCount != 0 {
		t.Errorf("expected 0 drift count for gold copy tenant, got %d", matrix.DriftCount)
	}
}

func TestTenantActivationMatrix_CustomTenant_ExtendAndDrift(t *testing.T) {
	db := testutil.GetEphemeralTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// 1. Create a custom tenant
	customTenantID := uuid.New()
	_, err := db.ExecContext(ctx, `
		INSERT INTO public.tenants (id, name, display_name, gold_copy, is_active, plan)
		VALUES ($1, 'apex-capital', 'Apex Capital Management', false, true, 'enterprise')
	`, customTenantID)
	if err != nil {
		t.Fatalf("create custom tenant: %v", err)
	}

	// 2. Fetch initial matrix
	matrix, err := svc.GetTenantActivationMatrix(ctx, customTenantID)
	if err != nil {
		t.Fatalf("get initial matrix: %v", err)
	}
	if matrix.TotalActive != 0 {
		t.Errorf("expected 0 active rules initially for custom tenant, got %d", matrix.TotalActive)
	}

	// 3. Activate a rule with inherit mode
	rules, err := svc.ListCoreRules(ctx, ListRulesFilter{})
	if err != nil || len(rules) == 0 {
		t.Fatalf("failed to list rules: %v", err)
	}
	targetRule := rules[0]

	err = svc.UpdateTenantRuleActivation(ctx, customTenantID, targetRule.ID, UpdateRuleActivationRequest{
		Enabled:     true,
		InheritMode: compliance.Inherit,
		ActorID:     "steward@apex.com",
	})
	if err != nil {
		t.Fatalf("update rule activation (inherit): %v", err)
	}

	matrix, err = svc.GetTenantActivationMatrix(ctx, customTenantID)
	if err != nil {
		t.Fatalf("get matrix after inherit activation: %v", err)
	}
	if matrix.TotalActive != 1 {
		t.Errorf("expected 1 active rule, got %d", matrix.TotalActive)
	}

	// 4. Switch to extend mode with custom parameter overrides
	customParams := map[string]string{
		"max_single_position_pct": "0.08", // override from default
	}
	err = svc.UpdateTenantRuleActivation(ctx, customTenantID, targetRule.ID, UpdateRuleActivationRequest{
		Enabled:            true,
		InheritMode:        compliance.Extend,
		ParameterOverrides: customParams,
		ActorID:            "steward@apex.com",
	})
	if err != nil {
		t.Fatalf("update rule activation (extend): %v", err)
	}

	matrix, err = svc.GetTenantActivationMatrix(ctx, customTenantID)
	if err != nil {
		t.Fatalf("get matrix after extend activation: %v", err)
	}
	var extendedItem *TenantActivationItem
	for i := range matrix.Rules {
		if matrix.Rules[i].RuleID == targetRule.ID {
			extendedItem = &matrix.Rules[i]
			break
		}
	}
	if extendedItem == nil {
		t.Fatalf("extended rule not found in matrix")
	}
	if extendedItem.InheritMode != compliance.Extend {
		t.Errorf("expected InheritMode 'extend', got %s", extendedItem.InheritMode)
	}
	if extendedItem.DriftStatus != compliance.DriftCurrent {
		t.Errorf("expected DriftStatus 'CURRENT', got %s", extendedItem.DriftStatus)
	}
	if extendedItem.TenantThresholds["max_single_position_pct"] != "0.08" {
		t.Errorf("expected parameter override 0.08, got %s", extendedItem.TenantThresholds["max_single_position_pct"])
	}

	// 5. Simulate upstream core rule version advance (Core bumps to v2)
	// Add v2 snapshot for core rule and update current_version
	_, err = db.ExecContext(ctx, `
		INSERT INTO compliance.compliance_rule_version (
			rule_id, version, tenant_id, resolved_ast, parameter_thresholds, citation, effective_from, content_hash, compiled_bytecode_hash, created_by
		) VALUES (
			$1, 2, $2, '{"type": "binary_op", "operator": "<=", "left": {"type": "field", "path": "weight"}, "right": {"type": "param", "name": "max_single_position_pct"}}'::jsonb,
			'{"max_single_position_pct": "0.05"}'::jsonb, 'ESMA Guidelines 2026/02 Rev 2', now(),
			'a000000000000000000000000000000000000000000000000000000000000002', 'b000000000000000000000000000000000000000000000000000000000000002', 'regulator'
		)
	`, targetRule.ID, masterTenantID)
	if err != nil {
		t.Fatalf("insert core v2 snapshot: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		UPDATE compliance.compliance_rule
		SET current_version = 2, updated_at = now()
		WHERE id = $1
	`, targetRule.ID)
	if err != nil {
		t.Fatalf("bump core current_version: %v", err)
	}

	// Update tenant rule drift status to CORE_VERSION_UPDATED
	_, err = db.ExecContext(ctx, `
		UPDATE compliance.compliance_rule
		SET drift_status = 'CORE_VERSION_UPDATED'
		WHERE tenant_id = $1 AND core_rule_id = $2
	`, customTenantID, targetRule.ID)
	if err != nil {
		t.Fatalf("set drift status: %v", err)
	}

	// Fetch matrix again and assert drift count
	matrix, err = svc.GetTenantActivationMatrix(ctx, customTenantID)
	if err != nil {
		t.Fatalf("get matrix after core version bump: %v", err)
	}
	if matrix.DriftCount != 1 {
		t.Errorf("expected 1 drift count, got %d", matrix.DriftCount)
	}

	// 6. Repin tenant rule to core v2
	err = svc.RepinTenantRule(ctx, customTenantID, targetRule.ID, RepinRuleRequest{
		TargetVersion: 2,
		ActorID:       "steward@apex.com",
		StewardNotes:  "Re-aligned with ESMA 2026/02 Rev 2 while retaining custom 8% limit",
	})
	if err != nil {
		t.Fatalf("RepinTenantRule failed: %v", err)
	}

	// Fetch matrix again and assert drift cleared
	matrix, err = svc.GetTenantActivationMatrix(ctx, customTenantID)
	if err != nil {
		t.Fatalf("get matrix after repin: %v", err)
	}
	if matrix.DriftCount != 0 {
		t.Errorf("expected 0 drift count after repin, got %d", matrix.DriftCount)
	}

	// 7. Bulk toggle ruleset
	err = svc.ToggleRulesetActivation(ctx, customTenantID, "MARKET_CONDUCT", true, "steward@apex.com")
	if err != nil {
		t.Fatalf("ToggleRulesetActivation failed: %v", err)
	}

	matrix, err = svc.GetTenantActivationMatrix(ctx, customTenantID)
	if err != nil {
		t.Fatalf("get matrix after ruleset toggle: %v", err)
	}
	if matrix.TotalActive <= 1 {
		t.Errorf("expected multiple active rules after ruleset toggle, got %d", matrix.TotalActive)
	}
}

func TestHTTPHandler_Endpoints(t *testing.T) {
	db := testutil.GetEphemeralTestDB(t)
	svc := NewService(db)
	h := NewHandler(svc)

	r := chi.NewRouter()
	h.RegisterRoutes(r)

	// 1. GET /api/compliance/library/rules
	req := httptest.NewRequest("GET", "/api/compliance/library/rules", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var listResp struct {
		Data       []CoreRuleSummary `json:"data"`
		TotalCount int               `json:"total_count"`
	}
	if err := json.NewDecoder(w.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if listResp.TotalCount != 53 {
		t.Errorf("expected 53 rules, got %d", listResp.TotalCount)
	}

	// 2. GET /api/compliance/library/rules/{id}
	firstID := listResp.Data[0].ID
	req = httptest.NewRequest("GET", "/api/compliance/library/rules/"+firstID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 3. GET /api/compliance/library/rulesets
	req = httptest.NewRequest("GET", "/api/compliance/library/rulesets", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 4. GET /api/compliance/tenants/{tenant_id}/activations
	req = httptest.NewRequest("GET", "/api/compliance/tenants/"+masterTenantID.String()+"/activations", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 5. PUT /api/compliance/tenants/{tenant_id}/activations/rules/{rule_id}
	updateBody, _ := json.Marshal(UpdateRuleActivationRequest{
		Enabled:     true,
		InheritMode: compliance.Inherit,
		ActorID:     "officer@test.com",
	})
	req = httptest.NewRequest("PUT", "/api/compliance/tenants/"+masterTenantID.String()+"/activations/rules/"+firstID.String(), bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_CrossTenantIDOR_Rejected403(t *testing.T) {
	db := testutil.GetEphemeralTestDB(t)
	svc := NewService(db)
	h := NewHandler(svc)

	r := chi.NewRouter()
	h.RegisterRoutes(r)

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Simulate request with JWT claims for tenant A
	claimsA := &jwtmiddleware.JWTClaims{
		TenantID: tenantA.String(),
		Email:    "userA@tenantA.com",
	}

	// 1. Tenant A attempting to access Tenant B matrix -> 403
	req := httptest.NewRequest("GET", "/api/compliance/tenants/"+tenantB.String()+"/activations", nil)
	ctx := context.WithValue(req.Context(), jwtmiddleware.ClaimsContextKey, claimsA)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for cross-tenant GET, got %d", w.Code)
	}

	// 2. Tenant A attempting to update Tenant B rule activation -> 403
	body, _ := json.Marshal(UpdateRuleActivationRequest{Enabled: true})
	req = httptest.NewRequest("PUT", "/api/compliance/tenants/"+tenantB.String()+"/activations/rules/"+uuid.New().String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), jwtmiddleware.ClaimsContextKey, claimsA))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for cross-tenant PUT, got %d", w.Code)
	}
}

func TestTenantFencing_AppUserRLS_Isolation(t *testing.T) {
	db := testutil.GetEphemeralTestDB(t)
	ctx := context.Background()

	tenantA := uuid.New()
	tenantB := uuid.New()
	// Fetch an existing core rule
	var ruleID uuid.UUID
	err := db.QueryRowContext(ctx, "SELECT id FROM compliance.compliance_rule WHERE tenant_id = $1 LIMIT 1", masterTenantID).Scan(&ruleID)
	if err != nil {
		t.Fatalf("failed to fetch core rule: %v", err)
	}

	// Begin transaction as app_user
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx failed: %v", err)
	}
	defer tx.Rollback()

	// Switch to app_user and set tenant A context
	_, err = tx.ExecContext(ctx, "SET ROLE app_user;")
	if err != nil {
		t.Fatalf("SET ROLE app_user failed: %v", err)
	}

	_, err = tx.ExecContext(ctx, "SELECT set_config('app.current_tenant', $1, true);", tenantA.String())
	if err != nil {
		t.Fatalf("set tenantA context failed: %v", err)
	}

	// Insert activation for Tenant A
	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance.tenant_rule_activation (tenant_id, rule_id, enabled, inherit_mode, activated_by)
		VALUES ($1, $2, true, 'inherit', 'adminA');
	`, tenantA, ruleID)
	if err != nil {
		t.Fatalf("tenant A insert activation failed: %v", err)
	}

	// Switch tenant context to Tenant B within the same session
	_, err = tx.ExecContext(ctx, "SELECT set_config('app.current_tenant', $1, true);", tenantB.String())
	if err != nil {
		t.Fatalf("set tenantB context failed: %v", err)
	}

	// Verify Tenant B cannot see Tenant A's activation
	var count int
	err = tx.QueryRowContext(ctx, "SELECT count(*) FROM compliance.tenant_rule_activation").Scan(&count)
	if err != nil {
		t.Fatalf("query as tenant B failed: %v", err)
	}
	if count != 0 {
		t.Errorf("RLS violation: Tenant B can see %d activations belonging to Tenant A", count)
	}
}

package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

type TenantStudioHandler struct {
	db           *sql.DB
	profileSvc   *security.ProfileService
}

func NewTenantStudioHandler(db *sql.DB, profileSvc *security.ProfileService) *TenantStudioHandler {
	return &TenantStudioHandler{
		db:           db,
		profileSvc:   profileSvc,
	}
}

func (h *TenantStudioHandler) RegisterRoutes(r chi.Router) {
	r.Route("/v1/tenant", func(r chi.Router) {
		r.Get("/profiles", h.ListProfiles)
		r.Post("/profiles/clone", h.CloneProfile)

		r.Get("/policies", h.ListPolicies)
		r.Post("/policies/override", h.AppendPolicyOverride)

		r.Get("/entitlements", h.ListEntitlements)
		r.Get("/entitlements/effective", h.EffectiveEntitlements)
		r.Post("/entitlements/map", h.UpsertEntitlement)
		r.Delete("/entitlements/{id}", h.DeleteEntitlement)
	})
}

func (h *TenantStudioHandler) getTenantID(r *http.Request) (uuid.UUID, error) {
	if authInfo, ok := security.AuthInfoFromContext(r.Context()); ok && len(authInfo.TenantIDs) > 0 {
		return uuid.Parse(authInfo.TenantIDs[0])
	}
	tenantIDStr := jwtmiddleware.GetTenantIDFromContext(r)
	if tenantIDStr == "" {
		tenantIDStr = r.Header.Get("X-Tenant-ID")
	}
	if tenantIDStr == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(tenantIDStr)
}

// writeScopeTenantID returns the tenant_id a policy/entitlement write should
// be stored under: nil (global gold-copy baseline) when the caller is scoped
// to the gold-copy tenant, otherwise the caller's own tenant_id. This is what
// lets a gold-copy tenant admin author a rule once and have every other
// tenant inherit it through ResolveEffectiveEntitlements, without ever
// touching a per-tenant row.
func (h *TenantStudioHandler) writeScopeTenantID(ctx context.Context, tenantID uuid.UUID) *uuid.UUID {
	var goldCopyID uuid.NullUUID
	if err := h.db.QueryRowContext(ctx, `SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1`).Scan(&goldCopyID); err != nil {
		return &tenantID
	}
	if goldCopyID.Valid && goldCopyID.UUID == tenantID {
		return nil
	}
	return &tenantID
}

func (h *TenantStudioHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil || tenantID == uuid.Nil {
		http.Error(w, `{"error":"unauthorized or missing tenant scope"}`, http.StatusUnauthorized)
		return
	}

	profiles, err := h.profileSvc.ListProfiles(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type ProfileRow struct {
		ProfileID   string     `json:"profile_id"`
		TenantID   *string    `json:"tenant_id"`
		ProfileKey string     `json:"profile_key"`
		ProfileName string     `json:"profile_name"`
		ParentProfileID *string `json:"parent_profile_id,omitempty"`
		CreatedAt  string     `json:"created_at"`
		UpdatedAt  string     `json:"updated_at"`
	}

	rows := make([]ProfileRow, len(profiles))
	for i, p := range profiles {
		rows[i] = ProfileRow{
			ProfileID:   p.ProfileID.String(),
			ProfileKey:  p.ProfileKey,
			ProfileName: p.ProfileName,
			CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		if p.TenantID != nil {
			tid := p.TenantID.String()
			rows[i].TenantID = &tid
		}
		if p.ParentProfileID != nil {
			pid := p.ParentProfileID.String()
			rows[i].ParentProfileID = &pid
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"profiles": rows,
		"count":    len(rows),
	})
}

type CloneProfileRequest struct {
	SourceProfileKey  string `json:"sourceProfileKey"`
	TargetProfileKey  string `json:"targetProfileKey"`
	TargetProfileName string `json:"targetProfileName"`
}

type CloneProfileResponse struct {
	ProfileID        string `json:"profileId"`
	ProfileKey       string `json:"profile_key"`
	ProfileName      string `json:"profile_name"`
	ClonedRulesCount int    `json:"clonedRulesCount"`
	SourceProfileKey string `json:"sourceProfileKey"`
}

func (h *TenantStudioHandler) CloneProfile(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil || tenantID == uuid.Nil {
		http.Error(w, `{"error":"unauthorized or missing tenant scope"}`, http.StatusUnauthorized)
		return
	}

	var req CloneProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.SourceProfileKey == "" || req.TargetProfileKey == "" || req.TargetProfileName == "" {
		http.Error(w, `{"error":"sourceProfileKey, targetProfileKey, and targetProfileName are required"}`, http.StatusBadRequest)
		return
	}

	profiles, err := h.profileSvc.ListProfiles(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var sourceProfile *security.SecurityProfile
	for i := range profiles {
		if profiles[i].ProfileKey == req.SourceProfileKey {
			sourceProfile = &profiles[i]
			break
		}
	}

	if sourceProfile == nil {
		http.Error(w, `{"error":"source profile not found"}`, http.StatusNotFound)
		return
	}

	newProfile := &security.SecurityProfile{
		TenantID:        &tenantID,
		ProfileKey:      req.TargetProfileKey,
		ProfileName:     req.TargetProfileName,
		ParentProfileID: &sourceProfile.ProfileID,
	}

	created, err := h.profileSvc.CreateProfile(r.Context(), newProfile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// sourceProfile.TenantID is nil for a true gold-copy source, so the
	// predicate must use IS NOT DISTINCT FROM rather than `=` (NULL = NULL
	// is never true in SQL, which silently copied zero rows before this fix).
	// Policy/entitlement rows are NOT physically duplicated on clone: they
	// inherit through parent_profile_id at read time (see
	// security.ResolveEffectiveEntitlements), so a gold-copy edit is
	// instantly visible to every tenant that hasn't locally overridden it.
	// This count reports how many baseline rows the clone will inherit.
	clonedCount := 0
	countErr := h.db.QueryRowContext(r.Context(), `
		SELECT COUNT(*) FROM studio.tenant_abac_policies
		WHERE tenant_id IS NOT DISTINCT FROM $1 AND target_profile_key = $2
	`, sourceProfile.TenantID, req.SourceProfileKey).Scan(&clonedCount)
	if countErr != nil && countErr != sql.ErrNoRows {
		clonedCount = 0
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CloneProfileResponse{
		ProfileID:        created.ProfileID.String(),
		ProfileKey:      created.ProfileKey,
		ProfileName:     created.ProfileName,
		ClonedRulesCount: clonedCount,
		SourceProfileKey: req.SourceProfileKey,
	})
}

type PolicyRow struct {
	PolicyID         string  `json:"policyId"`
	TenantID         *string `json:"tenantId"` // nil means gold-copy global baseline
	IsGoldCopy       bool    `json:"isGoldCopy"`
	TargetProfileKey string  `json:"targetProfileKey"`
	Name             string  `json:"name"`
	Description      string  `json:"description"`
	Effect           string  `json:"effect"`
	Priority         int     `json:"priority"`
	Enabled          bool    `json:"enabled"`
	ActionAttribute  string  `json:"actionAttribute"`
	ConditionDsl     string  `json:"conditionDsl,omitempty"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
}

func (h *TenantStudioHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil || tenantID == uuid.Nil {
		http.Error(w, `{"error":"unauthorized or missing tenant scope"}`, http.StatusUnauthorized)
		return
	}

	targetProfileKey := r.URL.Query().Get("target_profile_key")
	if targetProfileKey == "" {
		http.Error(w, `{"error":"target_profile_key query parameter is required"}`, http.StatusBadRequest)
		return
	}

	// tenant_id IS NULL rows are the gold-copy global baseline, inherited by
	// every tenant alongside their own overrides.
	query := `
		SELECT policy_id, tenant_id, target_profile_key, name, description, effect, priority, enabled, action_attribute, COALESCE(condition_dsl, ''), created_at, updated_at
		FROM studio.tenant_abac_policies
		WHERE (tenant_id = $1 OR tenant_id IS NULL) AND target_profile_key = $2
		ORDER BY priority DESC, created_at DESC
	`
	rows, err := h.db.QueryContext(r.Context(), query, tenantID, targetProfileKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var policies []PolicyRow
	for rows.Next() {
		var p PolicyRow
		if err := rows.Scan(&p.PolicyID, &p.TenantID, &p.TargetProfileKey, &p.Name, &p.Description, &p.Effect, &p.Priority, &p.Enabled, &p.ActionAttribute, &p.ConditionDsl, &p.CreatedAt, &p.UpdatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		p.IsGoldCopy = p.TenantID == nil
		policies = append(policies, p)
	}

	if policies == nil {
		policies = []PolicyRow{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"policies": policies,
		"count":    len(policies),
	})
}

type AppendPolicyOverrideRequest struct {
	TargetProfileKey string `json:"targetProfileKey"`
	ActionAttribute  string `json:"actionAttribute"`
	Effect           string `json:"effect"`
	PriorityRank     int    `json:"priorityRank"`
	ConditionDsl     string `json:"conditionDsl,omitempty"`
	Name             string `json:"name"`
	Description      string `json:"description,omitempty"`
}

type AppendPolicyOverrideResponse struct {
	PolicyID         string `json:"policyId"`
	TenantID         string `json:"tenantId"`
	TargetProfileKey string `json:"targetProfileKey"`
	ActionAttribute  string `json:"actionAttribute"`
	Effect           string `json:"effect"`
	PriorityRank     int    `json:"priorityRank"`
}

func (h *TenantStudioHandler) AppendPolicyOverride(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil || tenantID == uuid.Nil {
		http.Error(w, `{"error":"unauthorized or missing tenant scope"}`, http.StatusUnauthorized)
		return
	}

	var req AppendPolicyOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.TargetProfileKey == "" || req.ActionAttribute == "" || req.Effect == "" || req.Name == "" {
		http.Error(w, `{"error":"targetProfileKey, actionAttribute, effect, and name are required"}`, http.StatusBadRequest)
		return
	}

	if req.Effect != "allow" && req.Effect != "deny" {
		http.Error(w, `{"error":"effect must be 'allow' or 'deny'"}`, http.StatusBadRequest)
		return
	}

	// Written under tenant_id = NULL (global gold-copy baseline) when the
	// caller is scoped to the gold-copy tenant, so every other tenant
	// inherits it via ResolveEffectiveEntitlements without a physical copy.
	writeScope := h.writeScopeTenantID(r.Context(), tenantID)
	conflictClause := "ON CONFLICT (tenant_id, target_profile_key, action_attribute) WHERE tenant_id IS NOT NULL"
	if writeScope == nil {
		conflictClause = "ON CONFLICT (target_profile_key, action_attribute) WHERE tenant_id IS NULL"
	}

	var policyID string
	err = h.db.QueryRowContext(r.Context(), `
		INSERT INTO studio.tenant_abac_policies (tenant_id, target_profile_key, name, description, effect, priority, enabled, action_attribute, condition_dsl)
		VALUES ($1, $2, $3, $4, $5, $6, true, $7, $8)
		`+conflictClause+`
		DO UPDATE SET name = $3, description = $4, effect = $5, priority = $6, condition_dsl = $8, updated_at = NOW()
		RETURNING policy_id
	`, writeScope, req.TargetProfileKey, req.Name, req.Description, req.Effect, req.PriorityRank, req.ActionAttribute, req.ConditionDsl).Scan(&policyID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(AppendPolicyOverrideResponse{
		PolicyID:         policyID,
		TenantID:         tenantID.String(),
		TargetProfileKey: req.TargetProfileKey,
		ActionAttribute:  req.ActionAttribute,
		Effect:           req.Effect,
		PriorityRank:     req.PriorityRank,
	})
}

type EntitlementRow struct {
	EntitlementID  string  `json:"entitlementId"`
	TenantID       *string `json:"tenantId"` // nil means gold-copy global baseline
	IsGoldCopy     bool    `json:"isGoldCopy"`
	TargetProfileKey string `json:"targetProfileKey"`
	EntitlementType string `json:"entitlementType"`
	NodePath       string `json:"nodePath"`
	OverrideState  string `json:"overrideState"`
	ConditionDsl   string `json:"conditionDsl,omitempty"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

func (h *TenantStudioHandler) ListEntitlements(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil || tenantID == uuid.Nil {
		http.Error(w, `{"error":"unauthorized or missing tenant scope"}`, http.StatusUnauthorized)
		return
	}

	targetProfileKey := r.URL.Query().Get("target_profile_key")
	entitlementType := r.URL.Query().Get("entitlement_type")

	if targetProfileKey == "" {
		http.Error(w, `{"error":"target_profile_key query parameter is required"}`, http.StatusBadRequest)
		return
	}

	// tenant_id IS NULL rows are the gold-copy global baseline, inherited by
	// every tenant alongside their own overrides.
	query := `
		SELECT entitlement_id, tenant_id, target_profile_key, entitlement_type::text, node_path, override_state::text, COALESCE(condition_dsl, ''), created_at::text, updated_at::text
		FROM studio.tenant_component_entitlements
		WHERE (tenant_id = $1 OR tenant_id IS NULL) AND target_profile_key = $2
	`
	args := []interface{}{tenantID, targetProfileKey}

	if entitlementType != "" {
		query += " AND entitlement_type = $3"
		args = append(args, entitlementType)
	}

	query += " ORDER BY created_at DESC"

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var entitlements []EntitlementRow
	for rows.Next() {
		var e EntitlementRow
		if err := rows.Scan(&e.EntitlementID, &e.TenantID, &e.TargetProfileKey, &e.EntitlementType, &e.NodePath, &e.OverrideState, &e.ConditionDsl, &e.CreatedAt, &e.UpdatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		e.IsGoldCopy = e.TenantID == nil
		entitlements = append(entitlements, e)
	}

	if entitlements == nil {
		entitlements = []EntitlementRow{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entitlements": entitlements,
		"count":        len(entitlements),
	})
}

// EffectiveEntitlements resolves the given profile's entitlements through
// its security.security_profiles.parent_profile_id chain, so the UI can show
// what a tenant actually ends up with (own overrides + inherited baseline)
// rather than just the rows physically stored for this tenant.
func (h *TenantStudioHandler) EffectiveEntitlements(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil || tenantID == uuid.Nil {
		http.Error(w, `{"error":"unauthorized or missing tenant scope"}`, http.StatusUnauthorized)
		return
	}

	targetProfileKey := r.URL.Query().Get("target_profile_key")
	if targetProfileKey == "" {
		http.Error(w, `{"error":"target_profile_key query parameter is required"}`, http.StatusBadRequest)
		return
	}

	entitlements, err := security.ResolveEffectiveEntitlements(r.Context(), h.db, tenantID, targetProfileKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"target_profile_key": targetProfileKey,
		"entitlements":       entitlements,
		"count":               len(entitlements),
	})
}

type UpsertEntitlementRequest struct {
	TargetProfileKey string `json:"targetProfileKey"`
	EntitlementType  string `json:"entitlementType"`
	NodePath         string `json:"nodePath"`
	OverrideState    string `json:"overrideState"`
	ConditionDsl     string `json:"conditionDsl,omitempty"`
}

type UpsertEntitlementResponse struct {
	EntitlementID   string `json:"entitlementId"`
	TenantID        string `json:"tenantId"`
	TargetProfileKey string `json:"targetProfileKey"`
	EntitlementType string `json:"entitlementType"`
	NodePath        string `json:"nodePath"`
	OverrideState   string `json:"overrideState"`
	UpdatedAt       string `json:"updatedAt"`
}

func (h *TenantStudioHandler) UpsertEntitlement(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil || tenantID == uuid.Nil {
		http.Error(w, `{"error":"unauthorized or missing tenant scope"}`, http.StatusUnauthorized)
		return
	}

	var req UpsertEntitlementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.TargetProfileKey == "" || req.EntitlementType == "" || req.NodePath == "" || req.OverrideState == "" {
		http.Error(w, `{"error":"targetProfileKey, entitlementType, nodePath, and overrideState are required"}`, http.StatusBadRequest)
		return
	}

	validTypes := map[string]bool{"MENU_PAGE": true, "WORKFLOW_STEP": true, "PUBLIC_API": true}
	if !validTypes[req.EntitlementType] {
		http.Error(w, `{"error":"entitlementType must be MENU_PAGE, WORKFLOW_STEP, or PUBLIC_API"}`, http.StatusBadRequest)
		return
	}

	validStates := map[string]bool{"INHERIT_BASELINE": true, "EXPLICIT_ALLOW": true, "FORCE_DENY": true}
	if !validStates[req.OverrideState] {
		http.Error(w, `{"error":"overrideState must be INHERIT_BASELINE, EXPLICIT_ALLOW, or FORCE_DENY"}`, http.StatusBadRequest)
		return
	}

	// Written under tenant_id = NULL (global gold-copy baseline) when the
	// caller is scoped to the gold-copy tenant; see writeScopeTenantID.
	writeScope := h.writeScopeTenantID(r.Context(), tenantID)
	conflictClause := "ON CONFLICT (tenant_id, target_profile_key, entitlement_type, node_path) WHERE tenant_id IS NOT NULL"
	if writeScope == nil {
		conflictClause = "ON CONFLICT (target_profile_key, entitlement_type, node_path) WHERE tenant_id IS NULL"
	}

	var entitlementID string
	var updatedAt string
	err = h.db.QueryRowContext(r.Context(), `
		INSERT INTO studio.tenant_component_entitlements (tenant_id, target_profile_key, entitlement_type, node_path, override_state, condition_dsl)
		VALUES ($1, $2, $3::studio.entitlement_type, $4, $5::studio.override_state, NULLIF($6, ''))
		`+conflictClause+`
		DO UPDATE SET override_state = $5::studio.override_state, condition_dsl = NULLIF($6, ''), updated_at = NOW()
		RETURNING entitlement_id, updated_at::text
	`, writeScope, req.TargetProfileKey, req.EntitlementType, req.NodePath, req.OverrideState, req.ConditionDsl).Scan(&entitlementID, &updatedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(UpsertEntitlementResponse{
		EntitlementID:    entitlementID,
		TenantID:         tenantID.String(),
		TargetProfileKey: req.TargetProfileKey,
		EntitlementType:  req.EntitlementType,
		NodePath:         req.NodePath,
		OverrideState:    req.OverrideState,
		UpdatedAt:        updatedAt,
	})
}

func (h *TenantStudioHandler) DeleteEntitlement(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil || tenantID == uuid.Nil {
		http.Error(w, `{"error":"unauthorized or missing tenant scope"}`, http.StatusUnauthorized)
		return
	}

	entitlementID := chi.URLParam(r, "id")
	if entitlementID == "" {
		http.Error(w, `{"error":"entitlement id is required"}`, http.StatusBadRequest)
		return
	}

	_, err = h.db.ExecContext(r.Context(), `
		DELETE FROM studio.tenant_component_entitlements
		WHERE entitlement_id = $1 AND tenant_id IS NOT DISTINCT FROM $2
	`, entitlementID, h.writeScopeTenantID(r.Context(), tenantID))

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

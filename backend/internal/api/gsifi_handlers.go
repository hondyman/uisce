package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hondyman/uisce/backend/internal/handlers"
)

type GSIFIEvent struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	EventKey    string    `json:"event_key"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	SchemaJSON  string    `json:"schema_json"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

type TAMRule struct {
	ID                   string  `json:"id"`
	TenantID             string  `json:"tenant_id"`
	AssetClass           string  `json:"asset_class"`
	Currency             string  `json:"currency"`
	MinAmount            float64 `json:"min_amount"`
	MaxAmount            float64 `json:"max_amount"`
	RequiredApprovers    int     `json:"required_approvers"`
	RequiresSeniorManager bool    `json:"requires_senior_manager"`
	TimeLimitHours       int     `json:"time_limit_hours"`
}

type SoDRule struct {
	ID           string `json:"id"`
	TenantID     string `json:"tenant_id"`
	RoleKeyA     string `json:"role_key_a"`
	RoleKeyB     string `json:"role_key_b"`
	ConflictType string `json:"conflict_type"`
}

func (h *RBACHandlers) listGSIFIEventRegistry(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID

	query := `SELECT id, COALESCE(tenant_id, ''), event_key, category, COALESCE(description, ''), COALESCE(schema_json, ''), is_active, created_at FROM gsifi_event_registry WHERE is_active = true`
	args := []interface{}{}

	if tenantID != "" {
		query += " AND (tenant_id = $1 OR tenant_id IS NULL)"
		args = append(args, tenantID)
	}
	query += " ORDER BY category, event_key"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to fetch G-SIFI events: " + err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	events := []GSIFIEvent{}
	for rows.Next() {
		var e GSIFIEvent
		if err := rows.Scan(&e.ID, &e.TenantID, &e.EventKey, &e.Category, &e.Description, &e.SchemaJSON, &e.IsActive, &e.CreatedAt); err != nil {
			continue
		}
		events = append(events, e)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

func (h *RBACHandlers) createGSIFIEvent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EventKey    string `json:"event_key"`
		Category    string `json:"category"`
		Description string `json:"description"`
		SchemaJSON  string `json:"schema_json"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request: " + err.Error(), http.StatusBadRequest)
		return
	}

	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID

	var id string
	err = h.db.QueryRow(`
		INSERT INTO gsifi_event_registry (tenant_id, event_key, category, description, schema_json)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		tenantID, req.EventKey, req.Category, req.Description, req.SchemaJSON,
	).Scan(&id)
	if err != nil {
		http.Error(w, "Failed to create event: " + err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": id, "status": "created"})
}

func (h *RBACHandlers) updateGSIFIEvent(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")

	var req struct {
		Category    string `json:"category"`
		Description string `json:"description"`
		SchemaJSON string `json:"schema_json"`
		IsActive   *bool  `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request: " + err.Error(), http.StatusBadRequest)
		return
	}

	_, err := h.db.Exec(`
		UPDATE gsifi_event_registry
		SET category = COALESCE(NULLIF($2, ''), category),
		    description = COALESCE(NULLIF($3, ''), description),
		    schema_json = COALESCE(NULLIF($4, ''), schema_json),
		    is_active = COALESCE($5, is_active)
		WHERE id = $1`,
		eventID, req.Category, req.Description, req.SchemaJSON, req.IsActive,
	)
	if err != nil {
		http.Error(w, "Failed to update event: " + err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *RBACHandlers) listTAMRules(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID

	query := `SELECT id, COALESCE(tenant_id, ''), asset_class, currency, min_amount, COALESCE(max_amount, 0), required_approvers, requires_senior_manager, time_limit_hours FROM transaction_authorization_matrix WHERE 1=1`
	args := []interface{}{}

	if tenantID != "" {
		query += " AND tenant_id = $1"
		args = append(args, tenantID)
	}
	query += " ORDER BY asset_class, currency, min_amount"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to fetch TAM rules: " + err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	rules := []TAMRule{}
	for rows.Next() {
		var t TAMRule
		if err := rows.Scan(&t.ID, &t.TenantID, &t.AssetClass, &t.Currency, &t.MinAmount, &t.MaxAmount, &t.RequiredApprovers, &t.RequiresSeniorManager, &t.TimeLimitHours); err != nil {
			continue
		}
		rules = append(rules, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

func (h *RBACHandlers) createTAMRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AssetClass           string  `json:"asset_class"`
		Currency             string  `json:"currency"`
		MinAmount            float64 `json:"min_amount"`
		MaxAmount            float64 `json:"max_amount"`
		RequiredApprovers    int     `json:"required_approvers"`
		RequiresSeniorManager bool   `json:"requires_senior_manager"`
		TimeLimitHours       int     `json:"time_limit_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request: " + err.Error(), http.StatusBadRequest)
		return
	}

	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID

	var id string
	err = h.db.QueryRow(`
		INSERT INTO transaction_authorization_matrix (tenant_id, asset_class, currency, min_amount, max_amount, required_approvers, requires_senior_manager, time_limit_hours)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		tenantID, req.AssetClass, req.Currency, req.MinAmount, req.MaxAmount, req.RequiredApprovers, req.RequiresSeniorManager, req.TimeLimitHours,
	).Scan(&id)
	if err != nil {
		http.Error(w, "Failed to create TAM rule: " + err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": id, "status": "created"})
}

func (h *RBACHandlers) deleteTAMRule(w http.ResponseWriter, r *http.Request) {
	ruleID := chi.URLParam(r, "ruleID")

	result, err := h.db.Exec("DELETE FROM transaction_authorization_matrix WHERE id = $1", ruleID)
	if err != nil {
		http.Error(w, "Failed to delete TAM rule: " + err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "TAM rule not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (h *RBACHandlers) listSoDRules(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID

	query := `SELECT id, COALESCE(tenant_id, ''), role_key_a, role_key_b, conflict_type FROM role_conflict_rules WHERE 1=1`
	args := []interface{}{}

	if tenantID != "" {
		query += " AND tenant_id = $1"
		args = append(args, tenantID)
	}
	query += " ORDER BY role_key_a, role_key_b"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to fetch SoD rules: " + err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	rules := []SoDRule{}
	for rows.Next() {
		var s SoDRule
		if err := rows.Scan(&s.ID, &s.TenantID, &s.RoleKeyA, &s.RoleKeyB, &s.ConflictType); err != nil {
			continue
		}
		rules = append(rules, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

func (h *RBACHandlers) createSoDRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RoleKeyA     string `json:"role_key_a"`
		RoleKeyB     string `json:"role_key_b"`
		ConflictType string `json:"conflict_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request: " + err.Error(), http.StatusBadRequest)
		return
	}

	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID

	var id string
	err = h.db.QueryRow(`
		INSERT INTO role_conflict_rules (tenant_id, role_key_a, role_key_b, conflict_type)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (tenant_id, role_key_a, role_key_b) DO UPDATE SET conflict_type = EXCLUDED.conflict_type
		RETURNING id`,
		tenantID, req.RoleKeyA, req.RoleKeyB, req.ConflictType,
	).Scan(&id)
	if err != nil {
		http.Error(w, "Failed to create SoD rule: " + err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": id, "status": "created"})
}

func RegisterComplianceRoutes(r chi.Router, h *RBACHandlers) {
	r.Route("/compliance", func(r chi.Router) {
		r.Post("/run", h.runComplianceCheck)

		r.Get("/gsifi/events", h.listGSIFIEventRegistry)
		r.Post("/gsifi/events", h.createGSIFIEvent)
		r.Put("/gsifi/events/{eventID}", h.updateGSIFIEvent)

		r.Get("/gsifi/tam", h.listTAMRules)
		r.Post("/gsifi/tam", h.createTAMRule)
		r.Delete("/gsifi/tam/{ruleID}", h.deleteTAMRule)

		r.Get("/gsifi/sod", h.listSoDRules)
		r.Post("/gsifi/sod", h.createSoDRule)
	})
}

func (h *RBACHandlers) runComplianceCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "compliance check triggered"})
}

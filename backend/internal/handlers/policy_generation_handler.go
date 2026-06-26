package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/auth"
	"github.com/hondyman/semlayer/backend/pkg/llm"
	"github.com/jmoiron/sqlx"
)

type PolicyGenerationHandler struct {
	db            *sqlx.DB
	configService *llm.LLMConfigService
}

func NewPolicyGenerationHandler(db *sqlx.DB) *PolicyGenerationHandler {
	// Initialize LLM Config Service (assuming default path)
	svc := llm.NewLLMConfigService(".runtime/llm_config.json")
	return &PolicyGenerationHandler{
		db:            db,
		configService: svc,
	}
}

func (h *PolicyGenerationHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/v1/policies/generate", h.GeneratePolicy)
	r.Post("/api/v1/policies/save", h.SavePolicy)
}

type PolicyGenRequest struct {
	Prompt  string `json:"prompt"`
	Context string `json:"context,omitempty"` // e.g. "Trade Approval" to give LLM context
}

type PolicyGenResponse struct {
	RegoCode    string `json:"regoCode"`
	Explanation string `json:"explanation"`
}

func (h *PolicyGenerationHandler) GeneratePolicy(w http.ResponseWriter, r *http.Request) {
	var req PolicyGenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 1. Construct Prompt for LLM (Simulated)
	// In a real implementation we would use h.configService.GetProvider(...)
	// but for now we mock it to avoid compilation errors with unused variables.
	_ = h.configService

	// MOCK RESPONSE
	regoCode := fmt.Sprintf(`package tenant.rules

default allow = false

# Generated from: %s
allow {
	input.trade.amount < 1000000
}

deny[msg] {
	input.trade.amount >= 1000000
    input.counterparty.sanctioned == true
	msg := "Trade blocked due to sanctions and high value"
}`, req.Prompt)

	resp := PolicyGenResponse{
		RegoCode:    regoCode,
		Explanation: "This policy blocks trades over $1M if the counterparty is sanctioned.",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

type SavePolicyRequest struct {
	Name        string `json:"name"`
	RegoCode    string `json:"regoCode"`
	Description string `json:"description"`
}

func (h *PolicyGenerationHandler) SavePolicy(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUserFromContext(r.Context())
	if user.TenantID == "" {
		http.Error(w, "Unauthorized: No Tenant Context", http.StatusUnauthorized)
		return
	}

	var req SavePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Insert into core_policy
	query := `
		INSERT INTO core_policy (tenant_id, scope, expression, type, metadata)
		VALUES ($1, 'workflow', $2, 'authorization', $3)
		RETURNING id
	`
	metadata := map[string]string{
		"name":        req.Name,
		"description": req.Description,
		"createdBy":   user.ID,
	}
	metaJSON, _ := json.Marshal(metadata)

	var id string
	err := h.db.QueryRowContext(r.Context(), query, user.TenantID, req.RegoCode, metaJSON).Scan(&id)
	if err != nil {
		http.Error(w, "Failed to save policy: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

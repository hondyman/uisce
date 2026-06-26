package api

import (
	"encoding/json"
	"net/http"

	"github.com/hondyman/semlayer/backend/internal/region"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// handleSemanticQuery is the unified gateway endpoint: POST /api/llm/query
// It accepts natural language, datasource, and mode, then orchestrates the full pipeline:
// NL → Planner LLM → Semantic Query → Executor LLM → SQL → DB
func (srv *Server) handleSemanticQuery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract tenant ID from header
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header is required", http.StatusBadRequest)
		return
	}

	// Parse request
	var req SemanticQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Datasource == "" {
		http.Error(w, "datasource is required", http.StatusBadRequest)
		return
	}

	if req.Prompt == "" {
		http.Error(w, "prompt (natural language) is required", http.StatusBadRequest)
		return
	}

	// Default to exploratory mode if not specified
	if req.Mode == "" {
		req.Mode = "exploratory"
	}

	// Ensure region is present (middleware enforces this but check defensively)
	region, ok := region.GetRegionFromContext(r.Context())
	if !ok {
		http.Error(w, "X-Tenant-Region header is required", http.StatusBadRequest)
		return
	}

	// Create gateway and process
	gateway := NewLLMGateway(srv)
	resp, err := gateway.ProcessQuery(ctx, tenantID, region, &req)

	// Always return the response (including error messages)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to marshal response", http.StatusInternalServerError)
	}
}

// handlePlannerOnly is a debug endpoint: POST /api/llm/planner
// It accepts natural language and returns only the semantic query (no SQL execution)
func (srv *Server) handlePlannerOnly(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract tenant ID from header
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Datasource string `json:"datasource"`
		Version    string `json:"version,omitempty"`
		Prompt     string `json:"prompt"`
		Mode       string `json:"mode,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Datasource == "" || req.Prompt == "" {
		http.Error(w, "datasource and prompt are required", http.StatusBadRequest)
		return
	}

	gateway := NewLLMGateway(srv)

	// Ensure region is present
	region, ok := region.GetRegionFromContext(r.Context())
	if !ok {
		http.Error(w, "X-Tenant-Region header is required", http.StatusBadRequest)
		return
	}

	// Load bundle (region-scoped)
	bundle, err := gateway.loadSemanticBundle(ctx, tenantID, req.Datasource, region, req.Version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Call planner LLM
	semQuery, err := gateway.callPlannerLLM(ctx, bundle, req.Prompt, req.Mode, region)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Validate
	if err := srv.ValidateSemanticQuery(bundle, semQuery); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(semQuery)
}

// handleExecutorOnly is a debug endpoint: POST /api/llm/executor
// It accepts a semantic query (JSON) and returns the generated SQL
func (srv *Server) handleExecutorOnly(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract tenant ID from header
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header is required", http.StatusBadRequest)
		return
	}

	// Ensure region is present
	region, ok := region.GetRegionFromContext(r.Context())
	if !ok {
		http.Error(w, "X-Tenant-Region header is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Datasource string        `json:"datasource"`
		Version    string        `json:"version,omitempty"`
		Query      SemanticQuery `json:"query"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	gateway := NewLLMGateway(srv)

	// Load bundle
	bundle, err := gateway.loadSemanticBundle(ctx, tenantID, req.Datasource, region, req.Version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Validate semantic query
	if err := srv.ValidateSemanticQuery(bundle, &req.Query); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Call executor LLM
	sql, err := gateway.callExecutorLLM(ctx, bundle, &req.Query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"datasource": req.Datasource,
		"sql":        sql,
	})
}

// handleHealthGoldenPrompts is a diagnostic endpoint: GET /api/llm/prompts
// It returns the golden prompts (for documentation and debugging)
func (srv *Server) handleHealthGoldenPrompts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"planner_prompt": map[string]string{
			"exploratory": GoldenPlannerSystemPrompt("exploratory"),
			"strict":      GoldenPlannerSystemPrompt("strict"),
			"crud":        GoldenPlannerSystemPrompt("crud"),
		},
		"executor_prompt": GoldenSQLSystemPrompt(),
		"documentation":   "See https://github.com/yourusername/semlayer/wiki/LLM-Prompts",
	})
}

// handleSemanticQueryModeInfo is a diagnostic endpoint: GET /api/llm/modes
// It returns information about available modes
func (srv *Server) handleSemanticQueryModeInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"modes": map[string]string{
			"exploratory": "Loose defaults; auto-select interesting fields; default limit 100",
			"strict":      "No assumptions; only explicit fields; error if nothing maps",
			"crud":        "Parse as create/update/delete operation; emit CRUDOperation JSON",
		},
		"default_mode": "exploratory",
		"example_request": map[string]interface{}{
			"datasource": "customers",
			"version":    "v1",
			"prompt":     "Show me all US-based retail customers created since January, ordered by most recent",
			"mode":       "exploratory",
		},
	})
}

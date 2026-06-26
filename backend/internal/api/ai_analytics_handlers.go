package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/pkg/llm"
	"github.com/jmoiron/sqlx"
)

// handleNLQAsk is the HTTP handler for the Natural Language Q&A endpoint.
// It processes natural language questions about the catalog and returns structured answers.
func (s *Server) handleNLQAsk(w http.ResponseWriter, r *http.Request) {
	// Extract scope inputs from headers
	datasourceID := r.Header.Get("X-Datasource-Id")
	if datasourceID == "" {
		datasourceID = r.Header.Get("X-Tenant-Datasource-ID")
	}
	region := r.Header.Get("X-Region")
	if datasourceID == "" || region == "" {
		http.Error(w, "X-Datasource-Id and X-Region headers are required", http.StatusBadRequest)
		return
	}
	auth, ok := security.AuthInfoFromContext(r.Context())
	if !ok {
		http.Error(w, "missing auth context", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req services.AskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Question == "" {
		http.Error(w, "question field is required", http.StatusBadRequest)
		return
	}

	resolver := security.NewDBDatasourceResolver(sqlx.NewDb(s.DB, "postgres"))
	secCtx, err := security.BuildContext(r.Context(), auth, security.BuildContextRequest{
		DatasourceID: datasourceID,
		Region:       region,
	}, resolver)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Process the question using the NLQ service
	resp, err := s.NLQService.Ask(r.Context(), secCtx, req)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("NLQ Ask failed: %v", err)
		http.Error(w, fmt.Sprintf("Failed to process question: %v", err), http.StatusInternalServerError)
		return
	}

	// Return structured response
	respond(w, r, resp, nil)
}

// handleGetLLMConfig returns the current LLM configuration (file-backed dev store).
func (s *Server) handleGetLLMConfig(w http.ResponseWriter, r *http.Request) {
	if s.LLMConfigSvc == nil {
		http.Error(w, "LLM config service not configured", http.StatusInternalServerError)
		return
	}
	cfg, err := s.LLMConfigSvc.Get()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load LLM config: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

// handlePutLLMConfig saves the provided LLM configuration to disk.
func (s *Server) handlePutLLMConfig(w http.ResponseWriter, r *http.Request) {
	if s.LLMConfigSvc == nil {
		http.Error(w, "LLM config service not configured", http.StatusInternalServerError)
		return
	}
	var cfg llm.LLMConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	if err := s.LLMConfigSvc.Set(&cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "saved"})
}

// handlePostLLMTest runs a quick test against the configured provider using the provided prompt.
// Request JSON: { "prompt": "...", "api_key": "optional override" }
func (s *Server) handlePostLLMTest(w http.ResponseWriter, r *http.Request) {
	if s.LLMConfigSvc == nil {
		http.Error(w, "LLM config service not configured", http.StatusInternalServerError)
		return
	}
	var req struct {
		Prompt string `json:"prompt"`
		APIKey string `json:"api_key,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		http.Error(w, "prompt is required", http.StatusBadRequest)
		return
	}
	cfg, err := s.LLMConfigSvc.Get()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load LLM config: %v", err), http.StatusInternalServerError)
		return
	}
	// Override API key if provided in request
	if req.APIKey != "" {
		cfg.APIKey = req.APIKey
	}
	resp, err := s.LLMConfigSvc.Test(r.Context(), cfg, req.Prompt)
	if err != nil {
		http.Error(w, fmt.Sprintf("LLM test failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"response": resp})
}

func (s *Server) handleNLQFeedback(w http.ResponseWriter, r *http.Request) {
	var req services.FeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.FeedbackService.SubmitFeedback(r.Context(), req); err != nil {
		http.Error(w, fmt.Sprintf("Failed to submit feedback: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleRunEval(w http.ResponseWriter, r *http.Request) {
	// Parse tenant ID from request or context (simplifying for now)
	var req struct {
		TenantID string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	runID, err := s.EvalService.RunEval(r.Context(), req.TenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to run eval: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"run_id": runID,
		"status": "completed",
	})
}

func (s *Server) handleCubeSync(w http.ResponseWriter, r *http.Request) {
	// Parse tenant ID from request (simplifying for now)
	var req struct {
		TenantID string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.CubeSyncService.SyncSchema(r.Context(), req.TenantID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to sync schema: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/rag"
	"github.com/hondyman/semlayer/backend/internal/tenant"
	"github.com/hondyman/semlayer/backend/internal/workflows"
	"go.temporal.io/sdk/client"
)

// RAGHandler handles RAG-related API requests
type RAGHandler struct {
	TenantManager  *tenant.TenantManager
	SearchService  *rag.SearchService
	TemporalClient client.Client
	ConfigService  *rag.ConfigService
}

// NewRAGHandler creates a new RAGHandler
func NewRAGHandler(tm *tenant.TenantManager, ss *rag.SearchService, tc client.Client, cs *rag.ConfigService) *RAGHandler {
	return &RAGHandler{
		TenantManager:  tm,
		SearchService:  ss,
		TemporalClient: tc,
		ConfigService:  cs,
	}
}

// SearchArgs represents the input arguments for the search action
type SearchArgs struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// SearchResponse represents the output of the search action
type SearchResponse struct {
	Results []rag.SearchResult `json:"results"`
}

// HandleSearch is the HTTP handler for the Hasura 'searchRAG' action
func (h *RAGHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	// 1. Parse Request
	var req struct {
		SessionVariables map[string]interface{} `json:"session_variables"`
		Input            SearchArgs             `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 2. Extract Tenant ID from Session Variables (X-Hasura-Tenant-Id)
	tenantIDStr, ok := req.SessionVariables["x-hasura-tenant-id"].(string)
	if !ok {
		http.Error(w, "Missing tenant ID in session variables", http.StatusUnauthorized)
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "Invalid tenant ID format", http.StatusBadRequest)
		return
	}

	// 3. Get Tenant Connection
	conn, err := h.TenantManager.GetTenantConnection(r.Context(), tenantID)
	if err != nil {
		http.Error(w, "Failed to connect to tenant database", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// 4. Get RAG Config
	config, err := h.ConfigService.GetRAGConfig(r.Context(), tenantID)
	if err != nil {
		http.Error(w, "Failed to get RAG config", http.StatusInternalServerError)
		return
	}

	// 5. Execute Search
	results, err := h.SearchService.HybridSearch(r.Context(), conn, rag.SearchRequest{
		Query:          req.Input.Query,
		Limit:          req.Input.Limit,
		MinScore:       config.RetrievalConfig.SimilarityThreshold,
		SemanticWeight: config.HybridSearch.SemanticWeight,
		KeywordWeight:  config.HybridSearch.KeywordWeight,
	})
	if err != nil {
		http.Error(w, "Search failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 5. Return Response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SearchResponse{Results: results})
}

// UploadArgs represents the input arguments for the upload action
type UploadArgs struct {
	FilePath string `json:"file_path"`
	Title    string `json:"title"`
}

// UploadResponse represents the output of the upload action
type UploadResponse struct {
	DocumentID string `json:"document_id"`
	Status     string `json:"status"`
}

// HandleUpload is the HTTP handler for the Hasura 'uploadDocument' action
func (h *RAGHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	// 1. Parse Request
	var req struct {
		SessionVariables map[string]interface{} `json:"session_variables"`
		Input            UploadArgs             `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 2. Extract Tenant ID
	tenantIDStr, ok := req.SessionVariables["x-hasura-tenant-id"].(string)
	if !ok {
		http.Error(w, "Missing tenant ID", http.StatusUnauthorized)
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	// 3. Trigger Temporal Workflow
	// Note: In a real app, we'd create the document record in DB first to get an ID
	// For simplicity, we'll generate one here and let the workflow handle creation or pass it in
	docID := uuid.New()
	
	// We need to pass the workflow parameters
	workflowParams := workflows.DocumentIngestionWorkflowParam{
		TenantID:   tenantID,
		DocumentID: docID,
		SourcePath: req.Input.FilePath,
	}
	
	workflowOptions := client.StartWorkflowOptions{
		ID:        "doc_ingest_" + docID.String(),
		TaskQueue: "rag-worker",
	}

	_, err = h.TemporalClient.ExecuteWorkflow(r.Context(), workflowOptions, workflows.DocumentIngestionWorkflow, workflowParams)
	if err != nil {
		http.Error(w, "Failed to start workflow", http.StatusInternalServerError)
		return
	}

	// 4. Return Response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(UploadResponse{
		DocumentID: docID.String(),
		Status:     "queued",
	})
}

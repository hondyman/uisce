package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/mcp"
	"github.com/jmoiron/sqlx"
)

type MCPHandler struct {
	db        *sqlx.DB
	mcpServer *mcp.MCPServer
}

func NewMCPHandler(db *sqlx.DB) *MCPHandler {
	return &MCPHandler{
		db:        db,
		mcpServer: mcp.NewMCPServer(db),
	}
}

// ListTools returns registered MCP tools for AI clients
func (h *MCPHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tools := h.mcpServer.ListTools()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"tools": tools})
}

// ExecuteTool handles standardized MCP tool calls
func (h *MCPHandler) ExecuteTool(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		http.Error(w, `{"error":"invalid tenant context"}`, http.StatusBadRequest)
		return
	}

	var req mcp.ToolExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}
	req.TenantID = tenantID

	resp, err := h.mcpServer.ExecuteTool(r.Context(), req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// OmniboxSearch coordinates natural language intent classification and AST compilation
func (h *MCPHandler) OmniboxSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		http.Error(w, `{"error":"invalid tenant context"}`, http.StatusBadRequest)
		return
	}

	var payload struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
		return
	}

	req := mcp.ToolExecutionRequest{
		TenantID:   tenantID,
		ToolName:   "text_to_semantic_ast",
		Actor:      "OMNIBOX_USER",
		Parameters: map[string]interface{}{"prompt": payload.Prompt},
	}

	resp, err := h.mcpServer.ExecuteTool(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error":"failed compiling natural language query"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

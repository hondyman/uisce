package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/mcp"
	"github.com/hondyman/uisce/backend/internal/security"
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

// dispatchTenantFromAuth reads the JWT-derived AuthInfo, dispatches a tenant
// UUID, validates a body-supplied tenant_id against the auth set if present,
// and writes a JSON-RPC 401/-32602/-32001 error to w on failure. Returns the
// dispatched tenant and a boolean indicating success.
//
// Mirrors internal/mcp/tool_handler.go:78-110. Closes the impersonation gap
// where this handler previously read X-Tenant-ID from the request header and
// trusted it verbatim.
func (h *MCPHandler) dispatchTenantFromAuth(w http.ResponseWriter, r *http.Request, bodyTenantID string) (uuid.UUID, bool) {
	auth, ok := security.AuthInfoFromContext(r.Context())
	if !ok || len(auth.TenantIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "auth required: JWT missing tenant_id claim",
		})
		return uuid.Nil, false
	}

	tenantID, err := uuid.Parse(auth.TenantIDs[0])
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "invalid tenant_id format",
		})
		return uuid.Nil, false
	}

	if bodyTenantID == "" {
		return tenantID, true
	}

	bodyUUID, err := uuid.Parse(bodyTenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "invalid body tenant_id format",
		})
		return uuid.Nil, false
	}

	for _, tid := range auth.TenantIDs {
		if bodyUUID.String() == tid {
			return bodyUUID, true
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": "tenant_id in body is not a member of your authenticated tenant set",
	})
	return uuid.Nil, false
}

// ListTools returns registered MCP tools for AI clients
func (h *MCPHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.dispatchTenantFromAuth(w, r, ""); !ok {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	tools := h.mcpServer.ListTools()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"tools": tools})
}

// ExecuteTool handles standardized MCP tool calls
func (h *MCPHandler) ExecuteTool(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.dispatchTenantFromAuth(w, r, "")
	if !ok {
		return
	}

	var req mcp.ToolExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
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
	var payload struct {
		Prompt   string `json:"prompt"`
		TenantID string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
		return
	}

	tenantID, ok := h.dispatchTenantFromAuth(w, r, payload.TenantID)
	if !ok {
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
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error":"failed compiling natural language query"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

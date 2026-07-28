package agentic

import (
	"encoding/json"
	"net/http"

	"github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

type MCPToolCallRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Method  string          `json:"method"` // tools/call
	Params  MCPToolParams   `json:"params"`
}

type MCPToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type MCPToolCallResponse struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      string                 `json:"id"`
	Result  map[string]interface{} `json:"result,omitempty"`
	Error   *MCPError              `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type MCPToolRouter struct {
	mcService *MakerCheckerService
}

func NewMCPToolRouter(db *sqlx.DB) *MCPToolRouter {
	return &MCPToolRouter{
		mcService: NewMakerCheckerService(db),
	}
}

func (router *MCPToolRouter) HandleToolCall(w http.ResponseWriter, r *http.Request) {
	var req MCPToolCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON-RPC request", http.StatusBadRequest)
		return
	}

	tenantID := "core"
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		tenantID = claims.TenantID
	}

	// Intercept AI action and route through Maker-Checker Gateway
	proposal := ProposalRequest{
		TenantID:   tenantID,
		AgentID:    "AutonomousMCP-Agent",
		TargetBOID: "customers",
		ActionType: req.Params.Name,
		Payload:    req.Params.Arguments,
	}

	ticketID, err := router.mcService.SubmitAgentProposal(r.Context(), proposal)
	if err != nil && ticketID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(MCPToolCallResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &MCPError{Code: -32603, Message: err.Error()},
		})
		return
	}

	resp := MCPToolCallResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"status":       "QUEUED_FOR_MAKER_CHECKER_APPROVAL",
			"ticket_id":    ticketID,
			"action_name":  req.Params.Name,
			"four_eyes":    "Human Portfolio Manager review required before execution",
			"is_compliant": err == nil,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

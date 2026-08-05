package agentic

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/tenant/goldcopy"
	"github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

type MCPToolCallRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      string        `json:"id"`
	Method  string        `json:"method"` // tools/call
	Params  MCPToolParams `json:"params"`
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

func (e *MCPError) Error() string {
	return e.Message
}

type MCPToolRouter struct {
	db               *sqlx.DB
	mcService        *MakerCheckerService
	goldcopyResolver *goldcopy.Resolver
}

func NewMCPToolRouter(db *sqlx.DB) *MCPToolRouter {
	return &MCPToolRouter{
		db:        db,
		mcService: NewMakerCheckerService(db),
	}
}

// SetGoldCopyResolver injects the gold copy tenant resolver.
func (r *MCPToolRouter) SetGoldCopyResolver(resolver *goldcopy.Resolver) {
	r.goldcopyResolver = resolver
}

func (router *MCPToolRouter) checkRoleAccess(ctx context.Context, toolName, tenantID, functionalRole string) error {
	if functionalRole == "" {
		return nil
	}

	goldCopyID, err := router.resolveGoldCopyID(ctx)
	if err != nil {
		return nil
	}

	var allowedRoles []string
	err = router.db.GetContext(ctx, &allowedRoles,
		`SELECT allowed_roles FROM mcp_tool_registry
		 WHERE tool_name = $1 AND tenant_id IN ($2, $3)
		 ORDER BY CASE WHEN tenant_id = $3 THEN 1 ELSE 0 END
		 LIMIT 1`,
		toolName, tenantID, goldCopyID.String())
	if err != nil {
		return nil
	}

	for _, role := range allowedRoles {
		if role == functionalRole {
			return nil
		}
	}

	return &MCPError{Code: 403, Message: "role not authorized for this tool"}
}

func (router *MCPToolRouter) HandleToolCall(w http.ResponseWriter, r *http.Request) {
	var req MCPToolCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON-RPC request", http.StatusBadRequest)
		return
	}

	tenantID := "core"
	functionalRole := ""
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil {
		if claims.TenantID != "" {
			tenantID = claims.TenantID
		}
	}
	if authInfo, ok := security.AuthInfoFromContext(r.Context()); ok {
		functionalRole = authInfo.FunctionalRole
	}

	if err := router.checkRoleAccess(r.Context(), req.Params.Name, tenantID, functionalRole); err != nil {
		errResp, ok := err.(*MCPError)
		if !ok {
			errResp = &MCPError{Code: http.StatusForbidden, Message: err.Error()}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(MCPToolCallResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   errResp,
		})
		return
	}

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

func (r *MCPToolRouter) resolveGoldCopyID(ctx context.Context) (uuid.UUID, error) {
	if r.goldcopyResolver != nil {
		return r.goldcopyResolver.Resolve(ctx)
	}
	var id string
	err := r.db.GetContext(ctx, &id, `SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1`)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, goldcopy.ErrGoldCopyNotFound
		}
		return uuid.Nil, err
	}
	return uuid.Parse(id)
}

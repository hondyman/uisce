// Package mcp is the canonical Uisce MCP surface.
//
// # Inventory
//
// As of 2026-09-17 (Phase 2 Step 1 skeleton), the platform's MCP-shaped
// code lives in SIX paths across THREE files, with only one currently
// reachable from the primary mux:
//
//	Path 1 (LIVE): internal/mcp/tool_handler.go — MCPToolHandler.
//	    Mount:  POST /api/mcp (chi, inside /api/* block).
//	    Auth:   security.AuthInfoFromContext (JWT-derived tenant).
//	    Tools:  7 read-only — get_business_object_contract,
//	            resolve_relationship_path, list_business_objects,
//	            get_bo_terms, list_pages, get_page, compile_semantic_query.
//	    Mutation tools (save_record, delete_record, run_sql,
//	    create_business_object) refused by name.
//	    Audit:  none (gap noted; Step 3 in Phase 2 plan).
//
//	Path 2 (INTERNAL): internal/mcp/mcp_server.go — MCPServer.
//	    Mount:  none (programmatic API; called via Path 4).
//	    Auth:   caller-controlled (relies on Path 4's gate).
//	    Tools:  3 — text_to_semantic_ast, triage_mdm_exception,
//	            inspect_schema_drift.
//	    Audit:  writes to catalog_ai.mcp_tool_execution_logs
//	            (input_parameters raw; redaction planned Step 3).
//
//	Path 3 (CLIENT): internal/mcp/tools.go + execution.go — UisceCopilot.
//	    Mount:  none (client); invoked by cmd/mcp-server/main.go stdio loop.
//	    Auth:   outbound bearer (UISCE_API_TOKEN env var).
//	    Role:   thin HTTP client; calls /api/v1/mcp/tools.
//	    Status: collapses to a 30-line shell around Path 1's HTTP
//	            transport at Step 5 of Phase 2.
//
//	Path 4 (GATED-DEAD): internal/api/mcp_handlers.go — MCPHandler.
//	    Mount:  zero callers (constructor never invoked post-merge).
//	    Auth:   was X-Tenant-ID header trust. JWT-gated in commit
//	            114e2c9cd (Phase 2 Step 0). Latent impersonation
//	            vector closed.
//	    Disposition: delete in Step 7.
//
//	Path 5 (GATED-DEAD): internal/handlers/mcp_handler.go — MCPHandler
//	    (HandleMCPRequest at /mcp) + MCPToolsHandler (RegisterRoutes at
//	    /mcp/tools) + MCPToolsHandler (RegisterMuxRoutes at
//	    /api/v1/mcp/tools).
//	    Mount:  MCPHandler.RegisterRoutes returns 404 at runtime;
//	            MCPToolsHandler.RegisterRoutes returns 404 (no caller);
//	            MCPToolsHandler.RegisterMuxRoutes has zero callers.
//	    Auth:   was hardcoded tenantID := "default" / "core" and
//	            X-Functional-Role header trust. JWT-gated in commit
//	            114e2c9cd (Phase 2 Step 0).
//	    Disposition: delete in Step 7; evaluate_compliance_trade and
//	            shadow_evaluate_rule capabilities migrate to Path 2's
//	            tool registry if kept, or to Path 1's read-only list.
//
//	Path 6 (LIVE): api.go:1640 — agentic.MCPToolRouter.
//	    Mount:  POST /api/mcp/tools/call (chi, inside /api/* block).
//	    Auth:   inherits AuthContextMiddleware's JWT auth.
//	    Role:   maker-checker workflow. Different protocol surface than
//	            Path 1 (legacy JSON-RPC). Migrate to Path 1 in a later
//	            step (not Phase 2 Step 0/1/2).
//
// # Refactor status (Step 1, 2026-09-17)
//
// Phase 2 plan: unify Paths 1+2 under one mark3labs/mcp-go-backed
// server, collapse Path 3 to a thin HTTP client, delete Paths 4+5+6 mux
// registrations in Step 7, port 7 read-only tools (Path 1) + 3 audit-logged
// tools (Path 2) to the unified server, add SSE + stdio transports.
//
// This file is the skeleton (Step 1). It compiles, its stub methods
// respond, and the existing 12 tests in tool_handler_test.go and
// integration_auth_test.go still pass. No existing handler is touched;
// no behavior change.
//
// # Dependency
//
// github.com/mark3labs/mcp-go v1.1.0 pinned 2026-09-17. Check
// https://github.com/mark3labs/mcp-go/releases for API stability before
// bumping. v1.x is the API-stable line per the library's README.
package mcp

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server is the unified MCP server. Step 1 skeleton — stub methods
// only. Tools register via RegisterTool; transports (HTTP, SSE, stdio)
// attach via the mark3labs/mcp-go library in subsequent steps.
type Server struct {
	db       *sqlx.DB
	registry *server.MCPServer
}

// NewServer constructs a unified MCP server. Step 1: builds an empty
// registry. Tools added in Step 2+. Transports in Step 4+ (SSE) and
// Step 5+ (stdio wrapper).
func NewServer(db *sqlx.DB) *Server {
	return &Server{
		db:       db,
		registry: server.NewMCPServer("uisce-semantic-mcp-server", "1.0.0"),
	}
}

// RegisterTool adds a tool to the registry. Step 1: signature only,
// not yet called by any tool. Step 2+: each Path 1 tool registers
// through this method.
func (s *Server) RegisterTool(name, description string, inputSchema interface{}, handler func(ctx context.Context, tenantID uuid.UUID, args json.RawMessage) (interface{}, error)) {
	// Step 1 stub. Step 2 wires the mark3libs/mcp-go AddTool API.
}

// ListTools returns the registered tool manifest. Step 1: returns an
// empty list. Step 2+: returns the full tool set from the registry.
func (s *Server) ListTools() []ToolDefinition {
	return []ToolDefinition{}
}

// CallTool dispatches a tool invocation. Step 1: not implemented.
// Step 2+: dispatches via the mark3labs/mcp-go registry.
func (s *Server) CallTool(ctx context.Context, tenantID uuid.UUID, name string, args json.RawMessage) (interface{}, error) {
	return nil, nil
}

// ensure import is referenced (mcplib is the v1 alias; will be used in
// Step 2+). Without this the import would be unused at this step.
var _ = mcplib.Tool{}

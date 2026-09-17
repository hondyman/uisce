// Package mcp is the canonical Uisce MCP surface.
//
// # Inventory (updated 2026-09-17, PR A route-table dump)
//
// chi v5.2.3 does NOT panic on a second Post of the same method+pattern.
// TestChi_DuplicateMethodPattern: last registration wins. chi.Walk lists
// the survivor only.
//
// TestMCP_RouteTableDump against real SetupRouter (ENVIRONMENT=test):
//
//	GET  /api/mcp              Path 1 MCPToolHandler.HandleGetInfo
//	POST /api/mcp              Path 1 MCPToolHandler.HandleRPC
//	POST /api/mcp/tools/call   Path 6 agentic.MCPToolRouter.HandleToolCall
//
// Dead at the mux (404): POST /mcp, GET /mcp/tools, GET /api/mcp/tools,
// GET /api/v1/mcp/tools.
//
// Dump verdict (not boring, not "never-registered"):
// BOTH registrations execute on the SAME /api chi group. Path 5
// RegisterRoutes runs (MCP-REGISTER path5 trace + handlers hook).
// Path 1 RegisterRoutes runs next (path1 trace; GET /api/mcp exists —
// Path 5 never registers GET). chi v5.2.3 last-wins silently; Walk
// lists only the survivor. Path 5 is registered-then-overwritten, not
// dead at the call site. Probe: tools/list → Path 1 catalog;
// mcp.list_tools → Path 1 -32601 (not Path 5's -32001).
//
// Ops surfaces that print the same table: stderr [ROUTES-DUMP] at
// api.go SetupRouter end, and admin GET /_routes. Keep both; the test
// is the gate. Flip checklist: TestMCP_RouteTableDump + those dumps.
//
// PR B must replace or wrap POST /api/mcp, not stack a third
// registration. The mount swap is ATOMIC — Path 1 RegisterRoutes
// removal and the new mount land in the same commit, route-table test
// updated in that diff, live checklist against the deployed binary
// before the commit is done. A split leaves POST /api/mcp 404.
//
// Path 6 URL move (PR D): if a compat shim is used, old+new coexist
// in one commit; shim removal is a later commit with its own
// reachability proof. Shim emits Deprecation / a log line so removal
// is data-driven.
//
// Stdio client invariant (PR B): the binary never reads or stores
// credentials beyond UISCE_API_TOKEN; stdout is the protocol channel
// — logs go to stderr and must never contain the token.
//
//	Path 1 (LIVE HTTP face through PR A): tool_handler.go
//	Path 2 (INTERNAL): mcp_server.go — tools ported onto Server below
//	Path 3 (CLIENT): tools.go + cmd/mcp-server — PR B
//	Path 4 (DEAD): api/mcp_handlers.go — never registered
//	Path 5 (DEAD at mux): handlers/mcp_handler.go — overwritten by Path 1
//	Path 6 (LIVE): POST /api/mcp/tools/call — "core" fallback closed Pre-A
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type toolHandler func(ctx context.Context, tenantID uuid.UUID, args json.RawMessage) (interface{}, error)

type registeredTool struct {
	def     ToolDefinition
	handler toolHandler
}

// Server is the unified MCP server. PR A: tools register and CallTool
// works. Path 1 remains the HTTP face — this type is not mounted yet.
type Server struct {
	db       *sqlx.DB
	registry *server.MCPServer
	path1    *MCPToolHandler
	path2    *MCPServer

	mu    sync.RWMutex
	order []string
	tools map[string]registeredTool
}

func NewServer(db *sqlx.DB) *Server {
	s := &Server{
		db:       db,
		registry: server.NewMCPServer("uisce-semantic-mcp-server", "1.0.0"),
		path1:    NewMCPToolHandler(db),
		path2:    NewMCPServer(db),
		tools:    make(map[string]registeredTool),
	}
	s.registerDefaultTools()
	return s
}

func (s *Server) RegisterTool(name, description string, inputSchema interface{}, handler func(ctx context.Context, tenantID uuid.UUID, args json.RawMessage) (interface{}, error)) {
	if handler == nil {
		handler = func(context.Context, uuid.UUID, json.RawMessage) (interface{}, error) {
			return nil, fmt.Errorf("tool %s has no handler", name)
		}
	}
	schema, _ := inputSchema.(map[string]interface{})
	if schema == nil {
		schema = map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
	}
	def := ToolDefinition{Name: name, Description: description, InputSchema: schema}

	s.mu.Lock()
	if _, exists := s.tools[name]; !exists {
		s.order = append(s.order, name)
	}
	s.tools[name] = registeredTool{def: def, handler: handler}
	s.mu.Unlock()

	raw, err := json.Marshal(schema)
	if err != nil {
		raw = []byte(`{"type":"object","properties":{}}`)
	}
	s.registry.AddTool(mcplib.NewToolWithRawSchema(name, description, raw), s.wrapHandler(name, handler))
}

func (s *Server) wrapHandler(name string, handler toolHandler) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		tenantID, err := tenantFromAuth(ctx)
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		args, err := json.Marshal(req.GetArguments())
		if err != nil {
			args = []byte(`{}`)
		}
		result, err := handler(ctx, tenantID, args)
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		out, err := json.Marshal(result)
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		return mcplib.NewToolResultText(string(out)), nil
	}
}

func (s *Server) ListTools() []ToolDefinition {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ToolDefinition, 0, len(s.order))
	for _, name := range s.order {
		out = append(out, s.tools[name].def)
	}
	return out
}

func (s *Server) CallTool(ctx context.Context, tenantID uuid.UUID, name string, args json.RawMessage) (interface{}, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("%s", authRequiredMsg)
	}
	if IsRefusedTool(name) {
		return nil, refusedError(name)
	}
	s.mu.RLock()
	entry, ok := s.tools[name]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
	if len(args) == 0 {
		args = json.RawMessage(`{}`)
	}
	return entry.handler(ctx, tenantID, args)
}

func (s *Server) path2Tool(name string) toolHandler {
	return func(ctx context.Context, tenantID uuid.UUID, args json.RawMessage) (interface{}, error) {
		params := map[string]interface{}{}
		if len(args) > 0 {
			_ = json.Unmarshal(args, &params)
		}
		resp, err := s.path2.ExecuteTool(ctx, ToolExecutionRequest{
			TenantID:   tenantID,
			ToolName:   name,
			Actor:      "mcp",
			Parameters: params,
		})
		if err != nil {
			if resp != nil {
				return resp, err
			}
			return nil, err
		}
		return resp.Result, nil
	}
}

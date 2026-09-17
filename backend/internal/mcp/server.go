// Package mcp is the canonical Uisce MCP surface.
//
// # Inventory (updated after PR B streamable cutover)
//
// chi v5.2.3 last-wins silently on duplicate method+pattern. Never
// double-register any verb on /api/mcp.
//
// Live mux after verb-complete replace (TestMCP_RouteTableDump):
//
//	ALL /api/mcp                 mark3labs StreamableHTTPServer (owns GET/POST/DELETE/HEAD)
//	POST /api/mcp/tools/call     Path 6 agentic.MCPToolRouter (maker-checker)
//
// Path 1 MCPToolHandler.RegisterRoutes and Path 5 handlers.RegisterMCP
// are NOT registered on the live /api group. Old GET info JSON
// (protocol=json-rpc-2.0) is retired; discovery is MCP tools/list.
//
// CutoverMarker = "mcp-cutover-streamable-v1" — flip checklist greps
// the deployed binary for this string.
//
// Tenant attribution:
//
//	Path 6 envelope result.tenant_id (we own JSON-RPC)
//	Unified Server tool result field tenant_id (not SDK envelope)
//	Maker-checker ticket remains the ledger
//
// Stdio (cmd/mcp-server): streamable-HTTP client of UISCE_API_URL;
// credentials only UISCE_API_TOKEN; stdout=protocol; stderr never logs token.
//
// JWT forge: services.SecurityManager.MintDevToken only (cmd/devjwt wraps it).
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

// Server is the unified MCP server. HTTP transport: HTTPHandler()
// (streamable). Tool implementations still use MCPToolHandler helpers.
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
	s.registerOMSTools()
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
		out, err := json.Marshal(withTenantField(result, tenantID))
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
	result, err := entry.handler(ctx, tenantID, args)
	if err != nil {
		return nil, err
	}
	return withTenantField(result, tenantID), nil
}

// withTenantField adds tenant_id to structured tool results (not the SDK
// JSON-RPC envelope). Path 6 carries tenant on its own envelope.
func withTenantField(result interface{}, tenantID uuid.UUID) interface{} {
	tid := tenantID.String()
	switch v := result.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v)+1)
		for k, val := range v {
			out[k] = val
		}
		out["tenant_id"] = tid
		return out
	case map[string]string:
		out := make(map[string]interface{}, len(v)+1)
		for k, val := range v {
			out[k] = val
		}
		out["tenant_id"] = tid
		return out
	case nil:
		return map[string]interface{}{"tenant_id": tid}
	default:
		return map[string]interface{}{"tenant_id": tid, "data": result}
	}
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

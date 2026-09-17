// Package mcp is the canonical Uisce MCP surface.
//
// # Inventory (PR D — one server)
//
// Live HTTP (chi /api group):
//
//	ALL  /api/mcp                 StreamableHTTPServer (stateless; CutoverMarker)
//	POST /api/agentic/proposals   maker-checker tool proposals (canonical)
//	POST /api/mcp/tools/call      Deprecated shim → same handler + Deprecation header
//
// Deleted: Path 2 MCPServer, Path 4 api.MCPHandler, Path 5 handlers.MCPHandler.
// Catalog: 16 tools. Audit: catalog_mdm_ai via CallTool. Stdio: cmd/mcp-server.
// Flip checklist: dump + mcp-live-probe tools=16 + marker + migrations clean
// + Path 6 new URL + shim Deprecation.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

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
	nlEngine *TextToASTCompiler

	mu    sync.RWMutex
	order []string
	tools map[string]registeredTool
}

func NewServer(db *sqlx.DB) *Server {
	s := &Server{
		db:       db,
		registry: server.NewMCPServer("uisce-semantic-mcp-server", "1.0.0"),
		path1:    NewMCPToolHandler(db),
		nlEngine: NewTextToASTCompiler(db),
		tools:    make(map[string]registeredTool),
	}
	s.registerDefaultTools()
	s.registerOMSTools()
	s.registerGovernanceTools()
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

func (s *Server) wrapHandler(name string, _ toolHandler) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		tenantID, err := tenantFromAuth(ctx)
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		args, err := json.Marshal(req.GetArguments())
		if err != nil {
			args = []byte(`{}`)
		}
		result, err := s.CallTool(ctx, tenantID, name, args)
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
	started := time.Now()
	if len(args) == 0 {
		args = json.RawMessage(`{}`)
	}
	if tenantID == uuid.Nil {
		err := fmt.Errorf("%s", authRequiredMsg)
		s.auditCall(ctx, uuid.Nil, name, args, nil, err, started)
		return nil, err
	}
	if IsRefusedTool(name) {
		err := refusedError(name)
		s.auditCall(ctx, tenantID, name, args, nil, err, started)
		return nil, err
	}
	s.mu.RLock()
	entry, ok := s.tools[name]
	s.mu.RUnlock()
	if !ok {
		err := fmt.Errorf("unknown tool: %s", name)
		s.auditCall(ctx, tenantID, name, args, nil, err, started)
		return nil, err
	}
	result, err := entry.handler(ctx, tenantID, args)
	if err != nil {
		s.auditCall(ctx, tenantID, name, args, nil, err, started)
		return nil, err
	}
	out := withTenantField(result, tenantID)
	s.auditCall(ctx, tenantID, name, args, out, nil, started)
	return out, nil
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

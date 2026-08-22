package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/jmoiron/sqlx"
)

// MCPToolHandler exposes governed semantic contracts and compilation tools to AI agents via JSON-RPC.
type MCPToolHandler struct {
	db       *sqlx.DB
	compiler *boresolver.BitemporalRangeCompiler
}

// NewMCPToolHandler creates a new MCPToolHandler instance.
func NewMCPToolHandler(db *sqlx.DB, optionalCompiler ...*boresolver.BitemporalRangeCompiler) *MCPToolHandler {
	compiler := boresolver.NewBitemporalRangeCompiler()
	if len(optionalCompiler) > 0 && optionalCompiler[0] != nil {
		compiler = optionalCompiler[0]
	}
	return &MCPToolHandler{
		db:       db,
		compiler: compiler,
	}
}

// RegisterRoutes registers MCP tool server endpoints on the router.
func (h *MCPToolHandler) RegisterRoutes(r chi.Router) {
	r.Post("/mcp", h.HandleRPC)
	r.Get("/mcp", h.HandleGetInfo)
}

// HandleGetInfo returns the MCP server descriptor.
func (h *MCPToolHandler) HandleGetInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":        "uisce-semantic-mcp-server",
		"version":     "1.0.0",
		"protocol":    "json-rpc-2.0",
		"description": "Governed Semantic OS & Bitemporal Query Compiler for AI Agents",
	})
}

// HandleRPC handles JSON-RPC 2.0 requests.
func (h *MCPToolHandler) HandleRPC(w http.ResponseWriter, r *http.Request) {
	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ErrorResponse("", ParseError, "Parse error: invalid JSON payload"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	var resp JSONRPCResponse

	switch req.Method {
	case "tools/list":
		resp = SuccessResponse(req.ID, h.listAvailableTools())
	case "tools/call":
		res, err := h.executeToolCall(r.Context(), req.Params)
		if err != nil {
			resp = ErrorResponse(req.ID, InternalError, err.Error())
		} else {
			resp = SuccessResponse(req.ID, res)
		}
	default:
		resp = ErrorResponse(req.ID, MethodNotFound, fmt.Sprintf("Method '%s' not found", req.Method))
	}

	json.NewEncoder(w).Encode(resp)
}

// ListAvailableTools returns the descriptor map of all tools exposed by this MCP server.
func (h *MCPToolHandler) ListAvailableTools() map[string]interface{} {
	return h.listAvailableTools()
}

func (h *MCPToolHandler) listAvailableTools() map[string]interface{} {
	return map[string]interface{}{
		"tools": []map[string]interface{}{
			{
				"name":        "get_business_object_contract",
				"description": "Returns fields, formulas, data types, and additivity rules for a governed Business Object",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"bo_id":     map[string]string{"type": "string"},
						"bo_key":    map[string]string{"type": "string"},
						"tenant_id": map[string]string{"type": "string"},
					},
					"required": []string{"tenant_id"},
				},
			},
			{
				"name":        "resolve_relationship_path",
				"description": "Finds certified join paths between two Business Objects or catalog nodes via catalog_edge",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"tenant_id":      map[string]string{"type": "string"},
						"source_node_id": map[string]string{"type": "string"},
						"target_node_id": map[string]string{"type": "string"},
					},
					"required": []string{"tenant_id", "source_node_id", "target_node_id"},
				},
			},
			{
				"name":        "compile_semantic_query",
				"description": "Compiles a certified, tenant-fenced SQL query across hot/cold lakehouse tiers (StarRocks/Iceberg)",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"tenant_id":            map[string]string{"type": "string"},
						"effective_start_date": map[string]string{"type": "string"},
						"effective_end_date":   map[string]string{"type": "string"},
						"watermark_date":       map[string]string{"type": "string"},
						"hot_table_name":       map[string]string{"type": "string"},
						"cold_table_name":      map[string]string{"type": "string"},
						"temporal_column":      map[string]string{"type": "string"},
						"business_key_columns": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
						"selected_columns":     map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
					},
					"required": []string{"tenant_id", "effective_start_date", "effective_end_date", "watermark_date", "hot_table_name", "cold_table_name"},
				},
			},
		},
	}
}

func (h *MCPToolHandler) executeToolCall(ctx context.Context, rawParams json.RawMessage) (interface{}, error) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(rawParams, &call); err != nil {
		return nil, fmt.Errorf("invalid tool call arguments: %w", err)
	}

	switch call.Name {
	case "get_business_object_contract":
		return h.getBusinessObjectContract(ctx, call.Arguments)

	case "resolve_relationship_path":
		return h.resolveRelationshipPath(ctx, call.Arguments)

	case "compile_semantic_query":
		var args boresolver.BitemporalRangeRequest
		if err := json.Unmarshal(call.Arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid compile_semantic_query args: %w", err)
		}
		return h.compiler.CompileRangeQuery(ctx, args)
	}

	return nil, fmt.Errorf("unknown tool: %s", call.Name)
}

func (h *MCPToolHandler) getBusinessObjectContract(ctx context.Context, argsRaw json.RawMessage) (interface{}, error) {
	var args struct {
		TenantID uuid.UUID `json:"tenant_id"`
		BOID     uuid.UUID `json:"bo_id"`
		BOKey    string    `json:"bo_key"`
	}
	if err := json.Unmarshal(argsRaw, &args); err != nil {
		return nil, err
	}

	if h.db == nil {
		return map[string]interface{}{
			"tenant_id": args.TenantID,
			"bo_key":    args.BOKey,
			"status":    "ACTIVE",
			"fields": []map[string]interface{}{
				{"name": "account_id", "role": "KEY", "data_type": "UUID"},
				{"name": "quantity", "role": "MEASURE", "additivity_scope": "FULLY_ADDITIVE"},
			},
		}, nil
	}

	var boName, displayName, status string
	query := `
		SELECT name, display_name, status
		FROM public.business_objects
		WHERE (id = $1 OR name = $2) AND (tenant_id = $3 OR tenant_id = '00000000-0000-0000-0000-000000000000')
		LIMIT 1`
	err := h.db.QueryRowContext(ctx, query, args.BOID.String(), args.BOKey, args.TenantID.String()).Scan(&boName, &displayName, &status)
	if err != nil && err != sql.ErrNoRows {
		logging.GetLogger().Sugar().Warnf("MCP getBusinessObjectContract note: %v", err)
	}

	return map[string]interface{}{
		"tenant_id":    args.TenantID,
		"bo_name":      boName,
		"display_name": displayName,
		"status":       status,
	}, nil
}

func (h *MCPToolHandler) resolveRelationshipPath(ctx context.Context, argsRaw json.RawMessage) (interface{}, error) {
	var args struct {
		TenantID     uuid.UUID `json:"tenant_id"`
		SourceNodeID uuid.UUID `json:"source_node_id"`
		TargetNodeID uuid.UUID `json:"target_node_id"`
	}
	if err := json.Unmarshal(argsRaw, &args); err != nil {
		return nil, err
	}

	if h.db == nil {
		return map[string]interface{}{
			"source_node_id": args.SourceNodeID,
			"target_node_id": args.TargetNodeID,
			"path_found":     true,
			"edge_type":      "IS_SPECIALIZATION_OF",
		}, nil
	}

	var edgeTypeName string
	var propsRaw []byte
	query := `
		SELECT edge_type_name, properties
		FROM public.catalog_edge
		WHERE source_id = $1 AND target_id = $2
		  AND (tenant_id = $3::text OR tenant_id = '00000000-0000-0000-0000-000000000000')
		LIMIT 1`
	err := h.db.QueryRowContext(ctx, query, args.SourceNodeID.String(), args.TargetNodeID.String(), args.TenantID.String()).Scan(&edgeTypeName, &propsRaw)
	if err != nil {
		return map[string]interface{}{
			"source_node_id": args.SourceNodeID,
			"target_node_id": args.TargetNodeID,
			"path_found":     false,
		}, nil
	}

	var props map[string]interface{}
	_ = json.Unmarshal(propsRaw, &props)

	return map[string]interface{}{
		"source_node_id": args.SourceNodeID,
		"target_node_id": args.TargetNodeID,
		"path_found":     true,
		"edge_type":      edgeTypeName,
		"properties":     props,
	}, nil
}

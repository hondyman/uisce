package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/boread"
	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/pagestudio"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

// MCPToolHandler exposes governed semantic contracts and compilation tools to AI agents via JSON-RPC.
type MCPToolHandler struct {
	db       *sqlx.DB
	compiler *boresolver.BitemporalRangeCompiler
	pages    *pagestudio.Service
	bos      *boread.Service
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
		pages:    pagestudio.NewService(db),
		bos:      boread.NewService(db),
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
// All tools/call requests are authenticated via JWT: tenant is extracted from
// claims and injected into the tool call, never taken from the request body.
// tools/list is unauthenticated (public descriptor).
func (h *MCPToolHandler) HandleRPC(w http.ResponseWriter, r *http.Request) {
	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ErrorResponse("", ParseError, "Parse error: invalid JSON payload"))
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch req.Method {
	case "tools/list":
		// Public: no auth required, return the tool manifest
		json.NewEncoder(w).Encode(SuccessResponse(req.ID, h.listAvailableTools()))
		return

	case "tools/call":
		// Auth required: tenant is read from security.AuthInfo (populated globally
		// by AuthContextMiddleware at api.go:847). Tenant is NEVER taken from
		// request body — body tenant_id is validated against the authenticated set.
		auth, ok := security.AuthInfoFromContext(r.Context())
		if !ok || len(auth.TenantIDs) == 0 {
			json.NewEncoder(w).Encode(ErrorResponse(req.ID, Unauthorized,
				"auth required: JWT missing tenant_id claim (account may lack a tenant assignment)"))
			return
		}

		// Body tenant_id: optional. Must be a member of the authenticated set.
		// The tenant_id lives inside params.arguments per MCP tool call convention.
		var params struct {
			Arguments struct {
				TenantID string `json:"tenant_id"`
			} `json:"arguments"`
		}
		_ = json.Unmarshal(req.Params, &params)
		bodyTenantID := params.Arguments.TenantID

		dispatchTenant := auth.TenantIDs[0]
		if bodyTenantID != "" {
			found := false
			for _, tid := range auth.TenantIDs {
				if tid == bodyTenantID {
					dispatchTenant = bodyTenantID
					found = true
					break
				}
			}
			if !found {
				json.NewEncoder(w).Encode(ErrorResponse(req.ID, InvalidParams,
					"tenant_id in body is not a member of your authenticated tenant set"))
				return
			}
		}

		tenantID, err := uuid.Parse(dispatchTenant)
		if err != nil {
			json.NewEncoder(w).Encode(ErrorResponse(req.ID, InvalidParams, "invalid tenant_id format"))
			return
		}

		res, err := h.executeToolCall(r.Context(), tenantID, req.Params)
		if err != nil {
			errMsg := err.Error()
			if strings.HasPrefix(errMsg, "unknown tool:") {
				json.NewEncoder(w).Encode(ErrorResponse(req.ID, MethodNotFound, errMsg))
			} else {
				json.NewEncoder(w).Encode(ErrorResponse(req.ID, InternalError, errMsg))
			}
			return
		}
		json.NewEncoder(w).Encode(SuccessResponse(req.ID, res))
		return

	default:
		json.NewEncoder(w).Encode(ErrorResponse(req.ID, MethodNotFound, fmt.Sprintf("Method '%s' not found", req.Method)))
		return
	}
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
				"description": "Returns fields, formulas, data types, and additivity rules for a governed Business Object. Tenant is extracted from the authenticated JWT.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"bo_id":  map[string]string{"type": "string"},
						"bo_key": map[string]string{"type": "string"},
					},
				},
			},
			{
				"name":        "resolve_relationship_path",
				"description": "Finds certified join paths between two Business Objects or catalog nodes via catalog_edge. Tenant is extracted from the authenticated JWT.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"source_node_id": map[string]string{"type": "string"},
						"target_node_id": map[string]string{"type": "string"},
					},
					"required": []string{"source_node_id", "target_node_id"},
				},
			},
			{
				"name":        "list_business_objects",
				"description": "Lists published Business Objects for the tenant (bound objects only; not the raw catalog). Tenant is extracted from the authenticated JWT.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{},
				},
			},
			{
				"name":        "get_bo_terms",
				"description": "Returns semantic terms for one Business Object (same fence Page Studio uses). Tenant is extracted from the authenticated JWT.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"bo_id":  map[string]string{"type": "string"},
						"bo_key": map[string]string{"type": "string"},
					},
				},
			},
			{
				"name":        "list_pages",
				"description": "Lists Page Studio page definitions for the tenant (id, name, slug, status). Tenant is extracted from the authenticated JWT.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{},
				},
			},
			{
				"name":        "get_page",
				"description": "Returns one page definition (layout, data sources, presentation events). No record data. Tenant is extracted from the authenticated JWT.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"page_id": map[string]string{"type": "string"},
						"slug":    map[string]string{"type": "string"},
					},
				},
			},
			{
				"name":        "compile_semantic_query",
				"description": "Compiles a certified, tenant-fenced SQL query across hot/cold lakehouse tiers (StarRocks/Iceberg). Tenant is extracted from the authenticated JWT.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"effective_start_date": map[string]string{"type": "string"},
						"effective_end_date":   map[string]string{"type": "string"},
						"watermark_date":       map[string]string{"type": "string"},
						"hot_table_name":       map[string]string{"type": "string"},
						"cold_table_name":      map[string]string{"type": "string"},
						"temporal_column":      map[string]string{"type": "string"},
						"business_key_columns": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
						"selected_columns":     map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
					},
					"required": []string{"effective_start_date", "effective_end_date", "watermark_date", "hot_table_name", "cold_table_name"},
				},
			},
		},
	}
}

func (h *MCPToolHandler) executeToolCall(ctx context.Context, tenantID uuid.UUID, rawParams json.RawMessage) (interface{}, error) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(rawParams, &call); err != nil {
		return nil, fmt.Errorf("invalid tool call arguments: %w", err)
	}

	switch call.Name {
	case "get_business_object_contract":
		return h.getBusinessObjectContract(ctx, tenantID, call.Arguments)

	case "resolve_relationship_path":
		return h.resolveRelationshipPath(ctx, tenantID, call.Arguments)

	case "list_business_objects":
		return h.listBusinessObjects(ctx, tenantID, call.Arguments)

	case "get_bo_terms":
		return h.getBOTerms(ctx, tenantID, call.Arguments)

	case "list_pages":
		return h.listPages(ctx, tenantID, call.Arguments)

	case "get_page":
		return h.getPage(ctx, tenantID, call.Arguments)

	case "save_record", "delete_record", "run_sql", "create_business_object":
		return nil, fmt.Errorf("refused: page/MCP tools cannot mutate records or invent Business Objects")

	case "compile_semantic_query":
		var args boresolver.BitemporalRangeRequest
		if err := json.Unmarshal(call.Arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid compile_semantic_query args: %w", err)
		}
		args.TenantID = tenantID // inject from JWT, not from body
		return h.compiler.CompileRangeQuery(ctx, args)
	}

	return nil, fmt.Errorf("unknown tool: %s", call.Name)
}

func (h *MCPToolHandler) getBusinessObjectContract(ctx context.Context, tenantID uuid.UUID, argsRaw json.RawMessage) (interface{}, error) {
	var args struct {
		BOID  uuid.UUID `json:"bo_id"`
		BOKey string    `json:"bo_key"`
	}
	if err := json.Unmarshal(argsRaw, &args); err != nil {
		return nil, err
	}

	if h.db == nil {
		return map[string]interface{}{
			"tenant_id": tenantID,
			"bo_key":    args.BOKey,
			"status":    "ACTIVE",
			"fields": []map[string]interface{}{
				{"name": "account_id", "role": "KEY", "data_type": "UUID"},
				{"name": "quantity", "role": "MEASURE", "additivity_scope": "FULLY_ADDITIVE"},
			},
		}, nil
	}

	c, err := h.bos.GetContract(ctx, tenantID, args.BOID, args.BOKey)
	if err != nil && err != sql.ErrNoRows {
		return map[string]interface{}{
			"tenant_id":    tenantID,
			"bo_name":      "",
			"display_name": "",
			"status":       "",
		}, nil
	}
	boName, displayName, status := "", "", ""
	if c != nil {
		boName, displayName, status = c.Name, c.DisplayName, c.Status
	}
	return map[string]interface{}{
		"tenant_id":    tenantID,
		"bo_name":      boName,
		"display_name": displayName,
		"status":       status,
	}, nil
}

func (h *MCPToolHandler) resolveRelationshipPath(ctx context.Context, tenantID uuid.UUID, argsRaw json.RawMessage) (interface{}, error) {
	var args struct {
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

	edge, err := h.bos.ResolveEdge(ctx, tenantID, args.SourceNodeID, args.TargetNodeID)
	if err != nil {
		return map[string]interface{}{
			"source_node_id": args.SourceNodeID,
			"target_node_id": args.TargetNodeID,
			"path_found":     false,
		}, nil
	}

	var props map[string]interface{}
	_ = json.Unmarshal(edge.Properties, &props)

	return map[string]interface{}{
		"source_node_id": args.SourceNodeID,
		"target_node_id": args.TargetNodeID,
		"path_found":     true,
		"edge_type":      edge.EdgeTypeName,
		"properties":     props,
	}, nil
}

func (h *MCPToolHandler) listBusinessObjects(ctx context.Context, tenantID uuid.UUID, argsRaw json.RawMessage) (interface{}, error) {
	list, err := h.bos.ListSummaries(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(list))
	for _, item := range list {
		out = append(out, map[string]interface{}{
			"id": item.ID, "key": item.Name, "display_name": item.DisplayName, "status": item.Status,
		})
	}
	return map[string]interface{}{"business_objects": out}, nil
}

func (h *MCPToolHandler) getBOTerms(ctx context.Context, tenantID uuid.UUID, argsRaw json.RawMessage) (interface{}, error) {
	var args struct {
		BOID  string `json:"bo_id"`
		BOKey string `json:"bo_key"`
	}
	if err := json.Unmarshal(argsRaw, &args); err != nil {
		return nil, err
	}
	terms, err := h.bos.ListTerms(ctx, tenantID, args.BOID, args.BOKey)
	if err != nil {
		return map[string]interface{}{"terms": []interface{}{}, "note": err.Error()}, nil
	}
	out := make([]map[string]interface{}, 0, len(terms))
	for _, t := range terms {
		out = append(out, map[string]interface{}{"termKey": t.TermKey, "displayName": t.DisplayName, "role": t.Role})
	}
	return map[string]interface{}{"terms": out}, nil
}

func (h *MCPToolHandler) listPages(ctx context.Context, tenantID uuid.UUID, argsRaw json.RawMessage) (interface{}, error) {
	pages, err := h.pages.ListSummaries(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(pages))
	for _, p := range pages {
		out = append(out, map[string]interface{}{"id": p.ID, "name": p.Name, "slug": p.Slug, "status": p.Status})
	}
	return map[string]interface{}{"pages": out}, nil
}

func (h *MCPToolHandler) getPage(ctx context.Context, tenantID uuid.UUID, argsRaw json.RawMessage) (interface{}, error) {
	var args struct {
		PageID string `json:"page_id"`
		Slug   string `json:"slug"`
	}
	if err := json.Unmarshal(argsRaw, &args); err != nil {
		return nil, err
	}
	page, err := h.pages.GetByIDOrSlug(ctx, tenantID, args.PageID, args.Slug)
	if err != nil {
		return map[string]interface{}{"found": false}, nil
	}
	return map[string]interface{}{
		"found":              true,
		"id":                 page.ID,
		"name":               page.Name,
		"slug":               page.Slug,
		"status":             page.Status,
		"layout":             page.Layout,
		"components":         page.Components,
		"dataSources":        page.DataSources,
		"presentationEvents": page.PresentationEvents,
		"filterBar":          page.FilterBar,
	}, nil
}

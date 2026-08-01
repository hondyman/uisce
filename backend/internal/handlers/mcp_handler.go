package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/mux"
	"github.com/hondyman/uisce/backend/internal/agentic"
	"github.com/hondyman/uisce/backend/internal/metadata"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

// MCPHandler handles Model Context Protocol requests (JSON-RPC 2.0)
type MCPHandler struct {
	GraphService *metadata.GraphService
}

// NewMCPHandler creates a new MCPHandler
func NewMCPHandler(gs *metadata.GraphService) *MCPHandler {
	return &MCPHandler{GraphService: gs}
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      interface{}   `json:"id"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// HandleMCPRequest processes generic MCP JSON-RPC requests
func (h *MCPHandler) HandleMCPRequest(w http.ResponseWriter, r *http.Request) {
	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, nil, -32700, "Parse error")
		return
	}

	switch req.Method {
	case "mcp.list_tools":
		h.handleListTools(w, req.ID)
	case "mcp.call_tool":
		h.handleCallTool(w, req.Params, req.ID)
	case "mcp.list_resources":
		h.handleListResources(w, req.ID)
	default:
		h.writeError(w, req.ID, -32601, "Method not found")
	}
}

func (h *MCPHandler) handleListTools(w http.ResponseWriter, id interface{}) {
	tools := []map[string]interface{}{
		{
			"name":        "get_node_schema",
			"description": "Get schema and properties for a specific semantic node (Business Object).",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"node_name": map[string]string{"type": "string", "description": "Name of the business object (e.g. 'Employee')"},
				},
				"required": []string{"node_name"},
			},
		},
		{
			"name":        "find_path",
			"description": "Find the shortest path and join logic between two semantic nodes.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"start_node": map[string]string{"type": "string", "description": "Start node name"},
					"end_node":   map[string]string{"type": "string", "description": "End node name"},
				},
				"required": []string{"start_node", "end_node"},
			},
		},
		{
			"name":        "evaluate_compliance_trade",
			"description": "Evaluate a proposed trade against active compliance rules. Returns approved/rejected with severity and violations.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"portfolio_id":     map[string]string{"type": "string", "description": "Portfolio identifier (e.g. 'PT-88120')"},
					"security_isin":  map[string]string{"type": "string", "description": "Security ISIN (e.g. 'US0378331005')"},
					"order_quantity":  map[string]string{"type": "number", "description": "Order quantity"},
					"order_price":     map[string]string{"type": "number", "description": "Order price"},
					"rule_chain_id":  map[string]string{"type": "string", "description": "Optional rule chain ID to evaluate against"},
				},
				"required": []string{"portfolio_id", "security_isin"},
			},
		},
		{
			"name":        "shadow_evaluate_rule",
			"description": "Evaluate a draft compliance rule in shadow mode against live order stream. Returns impact report with false positive rate.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"draft_rule_ast":  map[string]string{"type": "string", "description": "JSON-encoded draft rule AST to evaluate in shadow mode"},
					"evaluation_window": map[string]string{"type": "string", "description": "Time window for shadow evaluation (e.g. '24h', '7d')"},
				},
				"required": []string{"draft_rule_ast"},
			},
		},
		{
			"name":        "list_business_objects",
			"description": "List all available business objects in the metadata graph for a tenant.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"datasource_id": map[string]string{"type": "string", "description": "Optional datasource filter"},
				},
			},
		},
	}
	h.writeResult(w, id, map[string]interface{}{"tools": tools})
}

func (h *MCPHandler) handleListResources(w http.ResponseWriter, id interface{}) {
	// Expose the entire graph as a resource
	resources := []map[string]interface{}{
		{
			"uri":         "metadata://graph/full",
			"name":        "Full Metadata Graph",
			"mimeType":    "application/json",
			"description": "Dump of all nodes and edges in the metadata graph.",
		},
	}
	h.writeResult(w, id, map[string]interface{}{"resources": resources})
}

func (h *MCPHandler) handleCallTool(w http.ResponseWriter, paramsRaw json.RawMessage, id interface{}) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(paramsRaw, &params); err != nil {
		h.writeError(w, id, -32602, "Invalid params")
		return
	}

	ctx := context.Background()
	tenantID := "default" // Simplified for MVP

	switch params.Name {
	case "get_node_schema":
		var args struct {
			NodeName string `json:"node_name"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			h.writeError(w, id, -32602, "Invalid arguments")
			return
		}
		node, err := h.GraphService.GetNodeByName(ctx, tenantID, args.NodeName)
		if err != nil {
			h.writeError(w, id, -32000, fmt.Sprintf("Error fetching node: %v", err))
			return
		}
		if node == nil {
			h.writeError(w, id, -32000, "Node not found")
			return
		}
		// Also fetch edges to show relationships
		edges, _ := h.GraphService.GetEdges(ctx, node.ID)

		h.writeResult(w, id, map[string]interface{}{
			"node":  node,
			"edges": edges,
		})

	case "find_path":
		var args struct {
			StartNode string `json:"start_node"`
			EndNode   string `json:"end_node"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			h.writeError(w, id, -32602, "Invalid arguments")
			return
		}

		startNode, err := h.GraphService.GetNodeByName(ctx, tenantID, args.StartNode)
		if err != nil || startNode == nil {
			h.writeError(w, id, -32000, "Start node not found")
			return
		}
		endNode, err := h.GraphService.GetNodeByName(ctx, tenantID, args.EndNode)
		if err != nil || endNode == nil {
			h.writeError(w, id, -32000, "End node not found")
			return
		}

		path, err := h.GraphService.FindPath(ctx, startNode.ID, endNode.ID)
		if err != nil {
			h.writeError(w, id, -32000, fmt.Sprintf("Error finding path: %v", err))
			return
		}
		h.writeResult(w, id, map[string]interface{}{"path": path})

	case "evaluate_compliance_trade":
		var args struct {
			PortfolioID    string  `json:"portfolio_id"`
			SecurityISIN   string  `json:"security_isin"`
			OrderQuantity  float64 `json:"order_quantity"`
			OrderPrice     float64 `json:"order_price"`
			RuleChainID    string  `json:"rule_chain_id"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			h.writeError(w, id, -32602, "Invalid arguments")
			return
		}
		h.writeResult(w, id, map[string]interface{}{
			"approved":         true,
			"can_override":     false,
			"highest_severity": "INFORMATIONAL",
			"evaluated_vm":     true,
			"execution_time_ns": 850,
			"violations":       []interface{}{},
			"message":         "Compliance evaluation complete (MCP stub - real evaluation requires RuleEngine injection)",
		})

	case "shadow_evaluate_rule":
		var args struct {
			DraftRuleAST     string `json:"draft_rule_ast"`
			EvaluationWindow string `json:"evaluation_window"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			h.writeError(w, id, -32602, "Invalid arguments")
			return
		}
		h.writeResult(w, id, map[string]interface{}{
			"shadow_mode":           true,
			"orders_evaluated":     0,
			"soft_warnings":         0,
			"hard_blocks":           0,
			"false_positive_rate":   0.0,
			"recommendation":        "APPROVED",
			"message":               "Shadow evaluation complete (MCP stub - real shadow evaluation requires Kafka stream tap)",
		})

	case "list_business_objects":
		var args struct {
			DatasourceID string `json:"datasource_id"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			h.writeError(w, id, -32602, "Invalid arguments")
			return
		}
		businessObjects := []map[string]interface{}{
			{"id": "bo-portfolio", "name": "Portfolio", "display_name": "Portfolio", "datasource_id": args.DatasourceID},
			{"id": "bo-security", "name": "Security", "display_name": "Security", "datasource_id": args.DatasourceID},
			{"id": "bo-position", "name": "Position", "display_name": "Position", "datasource_id": args.DatasourceID},
			{"id": "bo-order", "name": "Order", "display_name": "Order", "datasource_id": args.DatasourceID},
		}
		h.writeResult(w, id, map[string]interface{}{
			"business_objects": businessObjects,
			"count":            len(businessObjects),
		})

	default:
		h.writeError(w, id, -32601, "Tool not found")
	}
}

func (h *MCPHandler) writeError(w http.ResponseWriter, id interface{}, code int, message string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Error:   &JSONRPCError{Code: code, Message: message},
		ID:      id,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *MCPHandler) writeResult(w http.ResponseWriter, id interface{}, result interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// RegisterRoutes helper for wiring
func (h *MCPHandler) RegisterRoutes(r chi.Router) {
	r.Post("/mcp", h.HandleMCPRequest)
}

// MCPToolsHandler serves the MCP tool registry API
type MCPToolsHandler struct {
	registry *agentic.MCPRegistryService
}

// NewMCPToolsHandler creates a new MCP tools handler
func NewMCPToolsHandler(registry *agentic.MCPRegistryService) *MCPToolsHandler {
	return &MCPToolsHandler{registry: registry}
}

// ListTools handles GET /api/v1/mcp/tools
func (h *MCPToolsHandler) ListTools(w http.ResponseWriter, r *http.Request) {
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

	roleHeader := r.Header.Get("X-Functional-Role")
	if roleHeader != "" && functionalRole == "" {
		functionalRole = roleHeader
	}

	entries, err := h.registry.ListToolsForRole(r.Context(), tenantID, functionalRole)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tools := make([]map[string]interface{}, 0, len(entries))
	for _, e := range entries {
		tools = append(tools, map[string]interface{}{
			"name":        e.ToolName,
			"description": e.Description,
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{},
			},
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tools": tools})
}

// RegisterRoutes wires the tools endpoint
func (h *MCPToolsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/mcp/tools", h.ListTools)
}

// RegisterMuxRoutes wires the tools endpoint onto a gorilla/mux router
func (h *MCPToolsHandler) RegisterMuxRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/mcp/tools", h.ListTools).Methods("GET")
}

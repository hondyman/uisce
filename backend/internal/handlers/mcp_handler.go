package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/metadata"
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

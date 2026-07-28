package mcp

import (
	"context"
)

type UisceCopilot struct {
	Client *SecureAPIClient // Wraps HTTP requests with the user's JWT/API Token
}

func NewUisceCopilot(baseURL, token string) *UisceCopilot {
	return &UisceCopilot{
		Client: NewSecureAPIClient(baseURL, token),
	}
}

func (c *UisceCopilot) HandleRequest(ctx context.Context, req JSONRPCRequest) JSONRPCResponse {
	switch req.Method {
	case "tools/list":
		return c.listTools(req.ID)
	case "tools/call":
		return c.callTool(ctx, req)
	default:
		return ErrorResponse(req.ID, MethodNotFound, "Method not supported")
	}
}

func (c *UisceCopilot) listTools(id string) JSONRPCResponse {
	tools := []ToolDefinition{
		{
			Name:        "search_catalog",
			Description: "Search the Semantic OS catalog for tables, columns, or existing semantic terms. Use this to find the exact node_id (UUID) before creating a Business Object binding.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]string{"type": "string", "description": "Search keyword (e.g., 'trades', 'fx_rates')"},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "draft_business_object",
			Description: "Creates a PENDING_APPROVAL proposal in the Governance Studio for a new Business Object. Generates the exact JSON specification including multi-backend bindings (Hot/Cold/Streaming) and field mappings.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"bo_name":       map[string]string{"type": "string"},
					"bo_key":        map[string]string{"type": "string"},
					"justification": map[string]string{"type": "string", "description": "Why this BO is being created. Will be read by the human Checker."},
					"diff_payload":  map[string]interface{}{"type": "object", "description": "The complex BO specification JSON matching the Uisce schema."},
				},
				"required": []string{"bo_name", "bo_key", "justification", "diff_payload"},
			},
		},
	}
	return SuccessResponse(id, map[string]interface{}{"tools": tools})
}

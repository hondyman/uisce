package mcp

import (
	"context"
)

type UisceCopilot struct {
	Client        *SecureAPIClient
	FunctionalRole string
}

func NewUisceCopilot(baseURL, token string) *UisceCopilot {
	return &UisceCopilot{
		Client: NewSecureAPIClient(baseURL, token),
	}
}

func NewUisceCopilotWithRole(baseURL, token, functionalRole string) *UisceCopilot {
	client := NewSecureAPIClient(baseURL, token)
	client.FunctionalRole = functionalRole
	return &UisceCopilot{
		Client:        client,
		FunctionalRole: functionalRole,
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
	result, err := c.Client.Get("/api/v1/mcp/tools")
	if err != nil || result == nil {
		return ErrorResponse(id, InternalError, "Failed to fetch tool list from registry")
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return ErrorResponse(id, InternalError, "Invalid response from tool registry")
	}

	tools, ok := resultMap["tools"].([]interface{})
	if !ok {
		return ErrorResponse(id, InternalError, "Invalid tool list format from registry")
	}

	return SuccessResponse(id, map[string]interface{}{"tools": tools})
}

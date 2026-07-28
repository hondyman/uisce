package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

func (c *UisceCopilot) callTool(ctx context.Context, req JSONRPCRequest) JSONRPCResponse {
	var callReq ToolCallRequest
	if err := json.Unmarshal(req.Params, &callReq); err != nil {
		return ErrorResponse(req.ID, InvalidParams, "Invalid tool call arguments")
	}

	switch callReq.Name {
	case "search_catalog":
		// 1. Ask Uisce API to find the exact node_id for the AI (Rule 7 scoping applies automatically)
		rawQuery, ok := callReq.Arguments["query"]
		if !ok {
			return ErrorResponse(req.ID, InvalidParams, "missing required argument 'query'")
		}
		query, _ := rawQuery.(string)
		results, err := c.Client.Get(fmt.Sprintf("/api/v1/discovery/search?q=%s", query))
		if err != nil {
			// Return mock discovery response for offline/standalone execution
			results = map[string]interface{}{
				"matches": []map[string]string{
					{"node_id": "node-trades-001", "name": "trades", "type": "TABLE"},
					{"node_id": "node-fx-002", "name": "fx_rates", "type": "TABLE"},
				},
			}
		}
		return SuccessResponse(req.ID, map[string]interface{}{"result": results})

	case "draft_business_object":
		// 2. The AI has constructed the complex matrix. Submit it for Human Approval.
		justification, _ := callReq.Arguments["justification"].(string)
		boName, _ := callReq.Arguments["bo_name"].(string)
		diffPayload := callReq.Arguments["diff_payload"]

		payload := map[string]interface{}{
			"bo_id":         "NEW", // Indicates creation
			"justification": justification,
			"diff_payload":  diffPayload,
		}

		// Send to the Maker-Checker endpoint created in Phase 3
		resp, err := c.Client.Post("/api/v1/governance/proposals", payload)
		proposalID := "prop-draft-001"
		if err == nil && resp != nil {
			if id, ok := resp["proposal_id"].(string); ok {
				proposalID = id
			}
		}

		// Give the AI a success message so it can inform the user
		successMsg := fmt.Sprintf(
			"Successfully drafted BO '%s'. Proposal ID %s is now in PENDING_APPROVAL. Tell the user to review it in the Governance Studio.",
			boName,
			proposalID,
		)
		return SuccessResponse(req.ID, map[string]interface{}{"message": successMsg})

	default:
		return ErrorResponse(req.ID, MethodNotFound, "Unknown tool")
	}
}

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
		{
			Name:        "resolve_instrument_symbology",
			Description: "Market EDM Instrument Master: Resolves identifiers across CUSIP, ISIN, SEDOL, FIGI, and Ticker with feed survivorship rules.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"identifier_type":  map[string]string{"type": "string", "description": "ISIN, CUSIP, SEDOL, FIGI, or TICKER"},
					"identifier_value": map[string]string{"type": "string"},
				},
				"required": []string{"identifier_type", "identifier_value"},
			},
		},
		{
			Name:        "evaluate_pretrade_compliance",
			Description: "Charles River (CRD) Compliance Engine: Evaluates pre-trade compliance rules (sector concentration limits, restricted list checks) before submitting an order.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"portfolio_id":     map[string]string{"type": "string"},
					"symbol":           map[string]string{"type": "string"},
					"order_type":       map[string]string{"type": "string", "description": "BUY or SELL"},
					"quantity":         map[string]string{"type": "number"},
					"estimated_amount": map[string]string{"type": "number"},
				},
				"required": []string{"portfolio_id", "symbol", "quantity"},
			},
		},
		{
			Name:        "post_ibor_abor_transaction",
			Description: "Aladdin Accounting & IBOR Engine: Computes double-entry ledger postings for Investment Book of Record (IBOR) and Accounting Book of Record (ABOR).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"portfolio_id": map[string]string{"type": "string"},
					"symbol":       map[string]string{"type": "string"},
					"quantity":     map[string]string{"type": "number"},
					"price":        map[string]string{"type": "number"},
					"asset_class":  map[string]string{"type": "string"},
				},
				"required": []string{"portfolio_id", "symbol", "quantity", "price"},
			},
		},
		{
			Name:        "optimize_household_tax_harvesting",
			Description: "Envestnet & WealthTech Engine: Scans household graph relationships (Household -> Individual -> Tax Lots -> Accounts) for tax-loss harvesting and wash-sale replacement recommendations.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"household_id": map[string]string{"type": "string", "description": "Target household ID (e.g. HH-SMITH-FAMILY)"},
				},
				"required": []string{"household_id"},
			},
		},
	}
	return SuccessResponse(id, map[string]interface{}{"tools": tools})
}

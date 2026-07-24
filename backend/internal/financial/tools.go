package financial

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ToolFunction represents a financial calculation that can be called by the LLM
type ToolFunction interface {
	Name() string
	Description() string
	Parameters() json.RawMessage
	Execute(ctx context.Context, params json.RawMessage) (interface{}, error)
}

// ToolRegistry manages available financial calculation tools
type ToolRegistry struct {
	tools map[string]ToolFunction
	repo  ToolRepository
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry(repo ToolRepository) *ToolRegistry {
	registry := &ToolRegistry{
		tools: make(map[string]ToolFunction),
		repo:  repo,
	}
	
	// Register built-in tools (legacy/fallback)
	registry.Register(&TimeWeightedReturnTool{})
	registry.Register(&SimpleAttributionTool{})
	
	// Register additional tools
	registry.RegisterAdditionalTools()
	
	return registry
}

// Register adds a tool to the registry
func (r *ToolRegistry) Register(tool ToolFunction) {
	r.tools[tool.Name()] = tool
}

// Get retrieves a tool by name
func (r *ToolRegistry) Get(ctx context.Context, name string) (ToolFunction, bool) {
	// Check in-memory first
	if tool, ok := r.tools[name]; ok {
		return tool, true
	}

	// Check database
	if r.repo != nil {
		ft, err := r.repo.GetByName(ctx, name)
		if err == nil && ft != nil {
			return &DynamicToolAdapter{Tool: ft}, true
		}
	}

	return nil, false
}

// List returns all available tools
func (r *ToolRegistry) List(ctx context.Context) []ToolFunction {
	tools := make([]ToolFunction, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}

	// Append from DB
	if r.repo != nil {
		dbTools, err := r.repo.List(ctx)
		if err == nil {
			for _, t := range dbTools {
				// Avoid duplicates if name collides
				if _, exists := r.tools[t.Name]; !exists {
					tools = append(tools, &DynamicToolAdapter{Tool: &t})
				}
			}
		}
	}

	return tools
}

// DynamicToolAdapter adapts a DB-backed FinancialTool to the ToolFunction interface
type DynamicToolAdapter struct {
	Tool *FinancialTool
}

func (a *DynamicToolAdapter) Name() string {
	return a.Tool.Name
}

func (a *DynamicToolAdapter) Description() string {
	return a.Tool.Description
}

func (a *DynamicToolAdapter) Parameters() json.RawMessage {
	return a.Tool.ParametersSchema
}

func (a *DynamicToolAdapter) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	// TODO: Implement actual execution logic based on HandlerType
	// For now, return a mock response or error
	return map[string]interface{}{
		"status": "executed",
		"tool":   a.Tool.Name,
		"params": params,
		"note":   "Dynamic execution not yet fully implemented",
	}, nil
}

// TimeWeightedReturnTool calculates time-weighted return (GIPS-compliant)
type TimeWeightedReturnTool struct{}

func (t *TimeWeightedReturnTool) Name() string {
	return "calculate_time_weighted_return"
}

func (t *TimeWeightedReturnTool) Description() string {
	return "Calculates GIPS-compliant time-weighted return for a portfolio over a specified period. Accounts for cash flows and provides linked returns."
}

func (t *TimeWeightedReturnTool) Parameters() json.RawMessage {
	schema := `{
		"type": "object",
		"properties": {
			"portfolio_id": {"type": "string", "description": "Portfolio identifier"},
			"start_date": {"type": "string", "format": "date", "description": "Period start date (YYYY-MM-DD)"},
			"end_date": {"type": "string", "format": "date", "description": "Period end date (YYYY-MM-DD)"},
			"include_dividends": {"type": "boolean", "description": "Include dividend reinvestment", "default": true}
		},
		"required": ["portfolio_id", "start_date", "end_date"]
	}`
	return json.RawMessage(schema)
}

func (t *TimeWeightedReturnTool) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var input struct {
		PortfolioID       string `json:"portfolio_id"`
		StartDate         string `json:"start_date"`
		EndDate           string `json:"end_date"`
		IncludeDividends  bool   `json:"include_dividends"`
	}
	
	if err := json.Unmarshal(params, &input); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	startDate, err := time.Parse("2006-01-02", input.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date: %w", err)
	}
	
	endDate, err := time.Parse("2006-01-02", input.EndDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end_date: %w", err)
	}
	
	// TODO: Implement actual TWR calculation
	// This is a placeholder - real implementation would:
	// 1. Fetch portfolio positions and transactions
	// 2. Calculate sub-period returns around cash flows
	// 3. Link returns geometrically
	// 4. Apply GIPS standards
	
	result := map[string]interface{}{
		"portfolio_id": input.PortfolioID,
		"start_date": startDate.Format("2006-01-02"),
		"end_date": endDate.Format("2006-01-02"),
		"time_weighted_return": 0.0523, // Placeholder: 5.23%
		"annualized_return": 0.0523,
		"methodology": "GIPS-compliant TWR with geometric linking",
		"note": "Placeholder implementation - integrate with real calculation engine",
	}
	
	return result, nil
}

// SimpleAttributionTool performs Brinson attribution analysis
type SimpleAttributionTool struct{}

func (t *SimpleAttributionTool) Name() string {
	return "calculate_attribution"
}

func (t *SimpleAttributionTool) Description() string {
	return "Performs Brinson attribution analysis to decompose portfolio returns into allocation, selection, and interaction effects vs a benchmark."
}

func (t *SimpleAttributionTool) Parameters() json.RawMessage {
	schema := `{
		"type": "object",
		"properties": {
			"portfolio_id": {"type": "string", "description": "Portfolio identifier"},
			"benchmark_id": {"type": "string", "description": "Benchmark identifier"},
			"start_date": {"type": "string", "format": "date", "description": "Period start date (YYYY-MM-DD)"},
			"end_date": {"type": "string", "format": "date", "description": "Period end date (YYYY-MM-DD)"},
			"grouping": {"type": "string", "enum": ["sector", "region", "asset_class"], "description": "Attribution grouping", "default": "sector"}
		},
		"required": ["portfolio_id", "benchmark_id", "start_date", "end_date"]
	}`
	return json.RawMessage(schema)
}

func (t *SimpleAttributionTool) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var input struct {
		PortfolioID string `json:"portfolio_id"`
		BenchmarkID string `json:"benchmark_id"`
		StartDate   string `json:"start_date"`
		EndDate     string `json:"end_date"`
		Grouping    string `json:"grouping"`
	}
	
	if err := json.Unmarshal(params, &input); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	// TODO: Implement actual Brinson attribution
	// Real implementation would:
	// 1. Fetch portfolio and benchmark holdings
	// 2. Calculate allocation effect: (wp - wb) * rb
	// 3. Calculate selection effect: wb * (rp - rb)
	// 4. Calculate interaction effect: (wp - wb) * (rp - rb)
	
	result := map[string]interface{}{
		"portfolio_id": input.PortfolioID,
		"benchmark_id": input.BenchmarkID,
		"start_date": input.StartDate,
		"end_date": input.EndDate,
		"allocation_effect": 0.0032,  // 32 bps
		"selection_effect": 0.0018,   // 18 bps
		"interaction_effect": 0.0005, // 5 bps
		"total_active_return": 0.0055, // 55 bps
		"methodology": "Brinson-Hood-Beebower attribution",
		"note": "Placeholder implementation - integrate with real attribution engine",
	}
	
	return result, nil
}

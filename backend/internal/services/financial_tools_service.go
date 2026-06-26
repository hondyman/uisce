package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	
	"github.com/hondyman/semlayer/backend/internal/financial"
)

// FinancialToolService wraps financial calculation tools for LLM usage
type FinancialToolService struct {
	registry *financial.ToolRegistry
}

// NewFinancialToolService creates a new financial tool service
func NewFinancialToolService(db *sql.DB) *FinancialToolService {
	repo := financial.NewSQLToolRepository(db)
	return &FinancialToolService{
		registry: financial.NewToolRegistry(repo),
	}
}

// ListTools returns available financial calculation tools
func (s *FinancialToolService) ListTools(ctx context.Context) []ToolDefinition {
	tools := s.registry.List(ctx)
	definitions := make([]ToolDefinition, len(tools))
	
	for i, tool := range tools {
		definitions[i] = ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		}
	}
	
	return definitions
}

// ExecuteTool executes a financial calculation tool by name
func (s *FinancialToolService) ExecuteTool(ctx context.Context, name string, params json.RawMessage) (interface{}, error) {
	tool, ok := s.registry.Get(ctx, name)
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	
	result, err := tool.Execute(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}
	
	return result, nil
}

// ToolDefinition describes a callable tool for the LLM
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// EnhancePromptWithTools adds tool definitions to the LLM prompt context
func (s *FinancialToolService) EnhancePromptWithTools(ctx context.Context, basePrompt string) string {
	tools := s.ListTools(ctx)
	
	toolsJSON, _ := json.MarshalIndent(tools, "", "  ")
	
	enhanced := basePrompt + "\n\n" + `
AVAILABLE FINANCIAL CALCULATION TOOLS:
You have access to the following deterministic financial calculation tools.
Use these tools instead of fabricating numbers.

Tools:
` + string(toolsJSON) + `

To use a tool, respond with a JSON object in this format:
{
  "action": "call_tool",
  "tool": "tool_name",
  "parameters": { ... }
}

After receiving tool results, incorporate them into your answer with proper citations.
`
	
	return enhanced
}

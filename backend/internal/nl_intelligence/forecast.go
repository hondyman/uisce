package nl_intelligence

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hondyman/semlayer/backend/pkg/llm"
)

// ForecastEngine predicts future risks using graph and historical data
type ForecastEngine struct {
	llmProvider llm.LLMProvider
}

func NewForecastEngine(llmProvider llm.LLMProvider) *ForecastEngine {
	return &ForecastEngine{llmProvider: llmProvider}
}

// PredictFailures identifies assets at risk
func (e *ForecastEngine) PredictFailures(ctx context.Context, history json.RawMessage, graph json.RawMessage) (*ForecastResult, error) {
	prompt := fmt.Sprintf(`
You are a predictive analyst. Based on recent failure history and the dependency graph, predict which assets (DAGs, jobs) are most likely to fail next.
Look for pattern like frequent upstream drift or high centrality of fragile nodes.

Failure History: %s
Dependency Graph: %s

Return JSON:
{
  "predictions": [
    {"asset": "...", "probability": 0.85, "reason": "..."}
  ]
}
`, string(history), string(graph))

	response, err := e.llmProvider.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, err
	}

	cleaned := cleanJSON(response)
	var result ForecastResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("failed to parse forecast result: %w", err)
	}

	return &result, nil
}

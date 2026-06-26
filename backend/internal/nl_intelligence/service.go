package nl_intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/pkg/llm"
	"github.com/jmoiron/sqlx"
)

// NLService is the unified natural language intelligence layer
type NLService struct {
	llmProvider llm.LLMProvider
	db          *sqlx.DB
	planner     *Planner
	explainer   *Explainer
	forecast    *ForecastEngine
}

// NewNLService creates a new NL intelligence service
func NewNLService(llmProvider llm.LLMProvider, db *sqlx.DB) *NLService {
	return &NLService{
		llmProvider: llmProvider,
		db:          db,
		planner:     NewPlanner(llmProvider),
		explainer:   NewExplainer(llmProvider),
		forecast:    NewForecastEngine(llmProvider),
	}
}

// Interpret translates natural language to an intent and query plan
func (s *NLService) Interpret(ctx context.Context, req NLRequest) (*NLResponse, error) {
	// 1. Intent Classification
	intent, entities, filters, reasoning, err := s.planner.ClassifyIntent(ctx, req.Question)
	if err != nil {
		return &NLResponse{
			Answer: "I had trouble understanding that question. Could you try rephrasing it or specifying the entity you're interested in (e.g., 'jobs for Acme')?",
		}, nil
	}

	// 2. Query Planning
	plan, err := s.planner.BuildQueryPlan(ctx, req.Question, intent, entities, filters, req.TenantScope)
	if err != nil {
		return &NLResponse{
			Intent:         intent,
			ReasoningSteps: reasoning,
			Answer:         fmt.Sprintf("I identified the intent as %s, but I couldn't architect a safe query plan. This usually happens if the requested entities aren't mapped in the semantic graph yet.", intent),
		}, nil
	}

	return &NLResponse{
		Intent:         intent,
		QueryPlan:      plan,
		ReasoningSteps: reasoning,
	}, nil
}

// Execute runs a query plan and returns structured data
func (s *NLService) Execute(ctx context.Context, plan *QueryPlan) (json.RawMessage, error) {
	if plan == nil {
		return nil, fmt.Errorf("nil query plan")
	}

	switch plan.Type {
	case "SQL":
		return s.executeSQL(ctx, plan)
	case "CYPHER":
		return s.executeCypher(ctx, plan)
	default:
		return nil, fmt.Errorf("unsupported plan type: %s", plan.Type)
	}
}

// ProposeEndpoint generates an API design proposal
func (s *NLService) ProposeEndpoint(ctx context.Context, prompt string, tenantID string) (any, error) {
	// 1. Build design prompt
	designPrompt := fmt.Sprintf(`
You are an API architect. Design a REST or GraphQL endpoint based on the user's request.
Return a JSON object representing an APIEndpoint.

User Request: "%s"
Tenant ID: %s

Return JSON:
{
  "name": "...",
  "path": "...",
  "method": "...",
  "type": "...",
  "bo_name": "...",
  "fields": ["..."],
  "semantic_version": "1.0.0",
  "status": "active"
}
`, prompt, tenantID)

	response, err := s.llmProvider.GenerateResponse(ctx, designPrompt)
	if err != nil {
		return nil, err
	}

	cleaned := cleanJSON(response)
	var proposal interface{} // Using interface{} to avoid circular dependency on apistudio.APIEndpoint
	if err := json.Unmarshal([]byte(cleaned), &proposal); err != nil {
		return nil, fmt.Errorf("failed to parse design proposal: %w", err)
	}

	return proposal, nil
}

// Summarize explains a query result in natural language
func (s *NLService) Summarize(ctx context.Context, question string, result json.RawMessage) (string, error) {
	prompt := fmt.Sprintf(`
You are a data assistant. Explain the answer to the user's question based on the provided data result.
Be concise, accurate, and mention specific values or entities.

Question: "%s"
Data: %s

Explanation:
`, question, string(result))

	response, err := s.llmProvider.GenerateResponse(ctx, prompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(response), nil
}

func (s *NLService) executeSQL(ctx context.Context, plan *QueryPlan) (json.RawMessage, error) {
	// TODO: Implement dialect-aware SQL execution
	// For now, assume standard Trino/Postgres via sqlx
	rows, err := s.db.QueryxContext(ctx, plan.SQL)
	if err != nil {
		return nil, fmt.Errorf("SQL execution failed: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		row := make(map[string]interface{})
		if err := rows.MapScan(row); err != nil {
			return nil, err
		}
		results = append(results, row)
	}

	return json.Marshal(results)
}

func (s *NLService) executeCypher(ctx context.Context, plan *QueryPlan) (json.RawMessage, error) {
	// Apache AGE specific execution
	// AGE queries are wrapped in SELECT * FROM cypher(...)
	// The planner should have already wrapped it, or we do it here.

	rows, err := s.db.QueryxContext(ctx, plan.Cypher)
	if err != nil {
		return nil, fmt.Errorf("Cypher execution failed: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		row := make(map[string]interface{})
		if err := rows.MapScan(row); err != nil {
			return nil, err
		}
		results = append(results, row)
	}

	return json.Marshal(results)
}

// ExplainIncident provides a narrative for a specific failure
func (s *NLService) ExplainIncident(ctx context.Context, incidentID string, graphContext json.RawMessage) (*IncidentExplanation, error) {
	return s.explainer.ExplainIncident(ctx, incidentID, graphContext)
}

// ProposeChangeSet suggests a fix for a problem
func (s *NLService) ProposeChangeSet(ctx context.Context, problemContext json.RawMessage) (*ChangeSetProposal, error) {
	return s.explainer.ProposeChangeSet(ctx, problemContext)
}

// PredictFailures identifies assets at risk
func (s *NLService) PredictFailures(ctx context.Context, history json.RawMessage, graph json.RawMessage) (*ForecastResult, error) {
	return s.forecast.PredictFailures(ctx, history, graph)
}

func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

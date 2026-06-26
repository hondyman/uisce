package nl_intelligence

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hondyman/semlayer/backend/pkg/llm"
)

type Planner struct {
	llmProvider llm.LLMProvider
	dialects    *DialectEngine
}

func NewPlanner(llmProvider llm.LLMProvider) *Planner {
	return &Planner{
		llmProvider: llmProvider,
		dialects:    &DialectEngine{},
	}
}

// ClassifyIntent parses the question into intent and structured tokens
func (p *Planner) ClassifyIntent(ctx context.Context, question string) (string, []Entity, Filters, []string, error) {
	prompt := BuildIntentPrompt(question)
	response, err := p.llmProvider.GenerateResponse(ctx, prompt)
	if err != nil {
		return "", nil, Filters{}, nil, err
	}

	cleaned := cleanJSON(response)
	var result struct {
		Intent         string   `json:"intent"`
		Entities       []Entity `json:"entities"`
		Filters        Filters  `json:"filters"`
		ReasoningSteps []string `json:"reasoning_steps"`
	}

	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return "", nil, Filters{}, nil, fmt.Errorf("failed to parse intent JSON: %w", err)
	}

	return result.Intent, result.Entities, result.Filters, result.ReasoningSteps, nil
}

// BuildQueryPlan generates the technical execution details
func (p *Planner) BuildQueryPlan(ctx context.Context, question string, intent string, entities []Entity, filters Filters, tenantScope []string) (*QueryPlan, error) {
	switch intent {
	case IntentSQLQuery:
		return p.buildSQLPlan(ctx, entities, filters, tenantScope)
	case IntentGraphQuery, IntentLineageExplanation:
		return p.buildCypherPlan(ctx, entities, filters, tenantScope)
	case IntentAPIDesign:
		return p.buildAPIDesignPlan(ctx, entities, filters, tenantScope)
	case IntentSimulation:
		return p.buildSimulationPlan(ctx, question, entities, filters, tenantScope)
	default:
		// Default to a generic hybrid or specific generator
		return nil, fmt.Errorf("query plan generation not implemented for intent: %s", intent)
	}
}

func (p *Planner) buildSQLPlan(ctx context.Context, entities []Entity, filters Filters, tenantScope []string) (*QueryPlan, error) {
	// For SQL, we also need the original question to provide context to the LLM
	// But BuildQueryPlan doesn't have it. Let's assume we pass it or reconstruct it.
	// Actually, let's keep it simple for now and rely on entities/filters.

	prompt := BuildSQLPrompt("Execute SQL query", entities, filters, tenantScope)
	sql, err := p.llmProvider.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return &QueryPlan{
		Type:       "SQL",
		Engine:     "TRINO", // Defaulting to Trino for operational broad queries
		SQL:        p.dialects.FormatSQL(sql, "trino"),
		Parameters: map[string]any{"tenantScope": tenantScope},
		Dialect:    "trino",
	}, nil
}

func (p *Planner) buildCypherPlan(ctx context.Context, entities []Entity, filters Filters, tenantScope []string) (*QueryPlan, error) {
	prompt := BuildCypherPrompt("Execute Cypher query", entities, filters, tenantScope)
	cypher, err := p.llmProvider.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Wrap in Apache AGE SQL function
	ageQuery := p.dialects.FormatCypher(cypher, "unified_graph")

	return &QueryPlan{
		Type:       "CYPHER",
		Engine:     "AGE",
		Cypher:     ageQuery,
		GraphName:  "unified_graph",
		Parameters: map[string]any{"tenantScope": tenantScope},
	}, nil
}

func (p *Planner) buildAPIDesignPlan(_ context.Context, _ []Entity, _ Filters, _ []string) (*QueryPlan, error) {
	// API design is special, it doesn't return data but a spec
	return &QueryPlan{
		Type: "API_DESIGN",
		// Logic to be implemented in a dedicated generator
	}, nil
}

func (p *Planner) buildSimulationPlan(ctx context.Context, question string, entities []Entity, filters Filters, tenantScope []string) (*QueryPlan, error) {
	prompt := BuildSimulationPrompt(question, entities, filters, tenantScope)
	simulationPlan, err := p.llmProvider.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// The LLM returns a JSON object with the simulation plan (deltas, etc.)
	// We wrap this in a QueryPlan type of "SIMULATION"
	// The caller (Service) will parse the QueryPlan.Parameters or Data to execute it.

	// Clean potentially wrapped JSON
	cleaned := cleanJSON(simulationPlan)
	var simData map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &simData); err != nil {
		return nil, fmt.Errorf("failed to parse simulation plan JSON: %w", err)
	}

	return &QueryPlan{
		Type:       "SIMULATION",
		Engine:     "SIMULATION_ENGINE",
		Parameters: simData,
	}, nil
}

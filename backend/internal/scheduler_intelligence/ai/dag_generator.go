package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
)

// DAGGenerator generates DAG definitions from natural language intents
type DAGGenerator struct {
	llmClient LLMClient
	logger    *slog.Logger
}

// LLMClient interface for calling language models
type LLMClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// NewDAGGenerator creates a new DAG generator
func NewDAGGenerator(llmClient LLMClient, logger *slog.Logger) *DAGGenerator {
	return &DAGGenerator{
		llmClient: llmClient,
		logger:    logger,
	}
}

// DAGIntent represents a natural language intent for DAG creation
type DAGIntent struct {
	Description    string            `json:"description"`
	TenantID       uuid.UUID         `json:"tenant_id"`
	Category       string            `json:"category,omitempty"`
	TargetSchedule string            `json:"target_schedule,omitempty"`
	Context        map[string]string `json:"context,omitempty"`
}

// GeneratedDAG represents an AI-generated DAG specification
type GeneratedDAG struct {
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	Category        string           `json:"category"`
	Nodes           []GeneratedNode  `json:"nodes"`
	Edges           []GeneratedEdge  `json:"edges"`
	ScheduleType    string           `json:"schedule_type"`
	CronExpression  string           `json:"cron_expression,omitempty"`
	MaxParallelJobs int              `json:"max_parallel_jobs"`
	FailFast        bool             `json:"fail_fast"`
	TimeoutSeconds  int              `json:"timeout_seconds"`
	Confidence      float64          `json:"confidence"`
	Reasoning       string           `json:"reasoning"`
	Suggestions     []string         `json:"suggestions,omitempty"`
}

// GeneratedNode represents a node in the generated DAG
type GeneratedNode struct {
	ID          string                 `json:"id"`
	JobType     string                 `json:"job_type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Timeout     int                    `json:"timeout_seconds,omitempty"`
	RetryCount  int                    `json:"retry_count,omitempty"`
}

// GeneratedEdge represents an edge in the generated DAG
type GeneratedEdge struct {
	FromNodeID string `json:"from_node_id"`
	ToNodeID   string `json:"to_node_id"`
	Type       string `json:"type"` // success, completion, any
	Condition  string `json:"condition,omitempty"`
}

// GenerateDAG creates a DAG specification from natural language
func (g *DAGGenerator) GenerateDAG(ctx context.Context, intent DAGIntent) (*GeneratedDAG, error) {
	prompt := g.buildPrompt(intent)
	
	g.logger.Info("Generating DAG from intent",
		"description", intent.Description,
		"tenant_id", intent.TenantID,
	)

	response, err := g.llmClient.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	dag, err := g.parseResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// Validate the generated DAG
	if err := g.validateDAG(dag); err != nil {
		return nil, fmt.Errorf("generated DAG validation failed: %w", err)
	}

	g.logger.Info("Successfully generated DAG",
		"name", dag.Name,
		"nodes", len(dag.Nodes),
		"edges", len(dag.Edges),
		"confidence", dag.Confidence,
	)

	return dag, nil
}

// buildPrompt constructs the LLM prompt for DAG generation
func (g *DAGGenerator) buildPrompt(intent DAGIntent) string {
	var sb strings.Builder

	sb.WriteString(`You are a workflow orchestration expert. Generate a DAG (Directed Acyclic Graph) specification based on the following intent.

## Intent
`)
	sb.WriteString(intent.Description)
	sb.WriteString("\n\n")

	if intent.Category != "" {
		sb.WriteString(fmt.Sprintf("Category: %s\n", intent.Category))
	}
	if intent.TargetSchedule != "" {
		sb.WriteString(fmt.Sprintf("Target Schedule: %s\n", intent.TargetSchedule))
	}

	sb.WriteString(`
## Output Format
Respond with a JSON object containing:
{
  "name": "dag_name",
  "description": "What this DAG does",
  "category": "pre-agg|report|integration|compliance|data_quality|migration",
  "nodes": [
    {
      "id": "node_1",
      "job_type": "sql_query|api_call|data_transform|validation|notification",
      "name": "Step Name",
      "description": "What this step does",
      "parameters": {},
      "timeout_seconds": 300,
      "retry_count": 3
    }
  ],
  "edges": [
    {
      "from_node_id": "node_1",
      "to_node_id": "node_2",
      "type": "success|completion|any",
      "condition": "optional condition"
    }
  ],
  "schedule_type": "cron|event|manual",
  "cron_expression": "0 2 * * *",
  "max_parallel_jobs": 3,
  "fail_fast": true,
  "timeout_seconds": 3600,
  "confidence": 0.85,
  "reasoning": "Explanation of the design choices",
  "suggestions": ["Optional optimization suggestions"]
}

Only respond with the JSON object, no other text.
`)

	return sb.String()
}

// parseResponse extracts the DAG from LLM response
func (g *DAGGenerator) parseResponse(response string) (*GeneratedDAG, error) {
	// Clean up response - extract JSON if wrapped in markdown
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	var dag GeneratedDAG
	if err := json.Unmarshal([]byte(response), &dag); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	return &dag, nil
}

// validateDAG ensures the generated DAG is valid
func (g *DAGGenerator) validateDAG(dag *GeneratedDAG) error {
	if dag.Name == "" {
		return fmt.Errorf("DAG name is required")
	}
	if len(dag.Nodes) == 0 {
		return fmt.Errorf("DAG must have at least one node")
	}

	// Check for duplicate node IDs
	nodeIDs := make(map[string]bool)
	for _, node := range dag.Nodes {
		if nodeIDs[node.ID] {
			return fmt.Errorf("duplicate node ID: %s", node.ID)
		}
		nodeIDs[node.ID] = true
	}

	// Validate edges reference valid nodes
	for _, edge := range dag.Edges {
		if !nodeIDs[edge.FromNodeID] {
			return fmt.Errorf("edge references unknown node: %s", edge.FromNodeID)
		}
		if !nodeIDs[edge.ToNodeID] {
			return fmt.Errorf("edge references unknown node: %s", edge.ToNodeID)
		}
	}

	// Check for cycles (basic topological sort check)
	if err := g.detectCycles(dag); err != nil {
		return err
	}

	return nil
}

// detectCycles checks if the DAG contains cycles
func (g *DAGGenerator) detectCycles(dag *GeneratedDAG) error {
	// Build adjacency list
	adj := make(map[string][]string)
	for _, node := range dag.Nodes {
		adj[node.ID] = []string{}
	}
	for _, edge := range dag.Edges {
		adj[edge.FromNodeID] = append(adj[edge.FromNodeID], edge.ToNodeID)
	}

	// Track visit state: 0=unvisited, 1=visiting, 2=visited
	state := make(map[string]int)

	var hasCycle bool
	var dfs func(node string) bool
	dfs = func(node string) bool {
		state[node] = 1 // visiting
		for _, neighbor := range adj[node] {
			if state[neighbor] == 1 {
				return true // cycle detected
			}
			if state[neighbor] == 0 {
				if dfs(neighbor) {
					return true
				}
			}
		}
		state[node] = 2 // visited
		return false
	}

	for _, node := range dag.Nodes {
		if state[node.ID] == 0 {
			if dfs(node.ID) {
				hasCycle = true
				break
			}
		}
	}

	if hasCycle {
		return fmt.Errorf("generated DAG contains cycles")
	}
	return nil
}

// RefineDAG improves an existing DAG based on feedback
func (g *DAGGenerator) RefineDAG(ctx context.Context, dag *GeneratedDAG, feedback string) (*GeneratedDAG, error) {
	prompt := g.buildRefinePrompt(dag, feedback)

	response, err := g.llmClient.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM refinement failed: %w", err)
	}

	refined, err := g.parseResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse refined DAG: %w", err)
	}

	if err := g.validateDAG(refined); err != nil {
		return nil, fmt.Errorf("refined DAG validation failed: %w", err)
	}

	return refined, nil
}

// buildRefinePrompt creates a prompt for DAG refinement
func (g *DAGGenerator) buildRefinePrompt(dag *GeneratedDAG, feedback string) string {
	dagJSON, _ := json.MarshalIndent(dag, "", "  ")
	
	return fmt.Sprintf(`You are a workflow orchestration expert. Refine the following DAG based on the feedback provided.

## Current DAG
%s

## Feedback
%s

## Instructions
- Apply the feedback to improve the DAG
- Maintain the same JSON output format
- Explain your changes in the "reasoning" field
- Add any new optimization suggestions

Only respond with the updated JSON object.
`, string(dagJSON), feedback)
}

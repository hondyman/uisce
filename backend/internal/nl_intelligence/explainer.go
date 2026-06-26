package nl_intelligence

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hondyman/semlayer/backend/pkg/llm"
)

// Explainer handles detailed reasoning for incidents and changesets
type Explainer struct {
	llmProvider llm.LLMProvider
}

func NewExplainer(llmProvider llm.LLMProvider) *Explainer {
	return &Explainer{llmProvider: llmProvider}
}

// ExplainIncident generates a narrative for a specific failure
func (e *Explainer) ExplainIncident(ctx context.Context, incidentID string, graphContext json.RawMessage) (*IncidentExplanation, error) {
	prompt := fmt.Sprintf(`
You are a site reliability engineer. Explain the root cause and impact of this incident based on the graph data.
Identify why it happened (e.g., schema drift) and what else is affected.

Incident ID: %s
Graph Context: %s

Return JSON:
{
  "narrative": "...",
  "rootCause": "...",
  "blastRadius": "...",
  "recommendedFix": "...",
  "suggestedChangeSetSummary": "..."
}
`, incidentID, string(graphContext))

	response, err := e.llmProvider.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, err
	}

	cleaned := cleanJSON(response)
	var explanation IncidentExplanation
	if err := json.Unmarshal([]byte(cleaned), &explanation); err != nil {
		return nil, fmt.Errorf("failed to parse incident explanation: %w", err)
	}

	return &explanation, nil
}

// ProposeChangeSet generates a ChangeSet proposal to address a problem
func (e *Explainer) ProposeChangeSet(ctx context.Context, problemContext json.RawMessage) (*ChangeSetProposal, error) {
	prompt := fmt.Sprintf(`
You are a data architect. Propose a ChangeSet to fix the described problem (e.g., schema drift, dependency failure).
Identify all impacted entities and assess the risk.

Problem Context: %s

Return JSON:
{
  "title": "...",
  "description": "...",
  "impactedEntities": [{"nodeId": "...", "entityType": "..."}],
  "risk": "MEDIUM",
  "governanceMeta": {}
}
`, string(problemContext))

	response, err := e.llmProvider.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, err
	}

	cleaned := cleanJSON(response)
	var proposal ChangeSetProposal
	if err := json.Unmarshal([]byte(cleaned), &proposal); err != nil {
		return nil, fmt.Errorf("failed to parse changeset proposal: %w", err)
	}

	return &proposal, nil
}

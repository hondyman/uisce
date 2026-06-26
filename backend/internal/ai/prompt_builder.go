package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/audit"
	"github.com/hondyman/semlayer/backend/internal/catalog"
)

// PromptBuilder constructs AI prompts for various audit scenarios
type PromptBuilder struct{}

// NewPromptBuilder creates a new PromptBuilder
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// ExplainJobRunPrompt builds a prompt to explain a failed job run
func (pb *PromptBuilder) ExplainJobRunPrompt(
	jobRun *audit.JobRunEvent,
	linkedJob map[string]interface{},
	linkedDAG map[string]interface{},
	semanticTerms []map[string]interface{},
	recentEvents []map[string]interface{},
	tenantScope []string,
) string {
	prompt := `You are an expert in data pipelines, semantic models, and operational incidents.

You are analyzing a failed job run with the following context:

JOB RUN:
- RunID: %s
- JobID: %s
- DAG ID: %s
- Status: %s
- Started: %s
- Error: %s

LINKED JOB:
%s

LINKED DAG:
%s

SEMANTIC TERMS:
%s

RECENT RELATED EVENTS (same job/DAG):
%s

TENANT SCOPE (events may only reference these tenants):
%s

TASKS:
1. Explain in clear language why this job failed.
2. Identify the most likely root cause using the graph context.
3. Describe the blast radius:
   - impacted jobs
   - impacted DAGs
   - impacted reports or terms
4. Recommend a concrete fix that a Tenant Admin could approve.
5. Propose a concise ChangeSet summary title.

CONSTRAINTS:
- Do NOT mention any tenants outside the allowed scope.
- Be specific but concise.
- Focus on actionable recommendations.

Return ONLY valid JSON with these fields:
{
  "narrative": string,
  "rootCause": string,
  "blastRadius": string,
  "recommendedFix": string,
  "suggestedChangeSetSummary": string
}
`

	return fmt.Sprintf(
		prompt,
		jobRun.RunID,
		jobRun.JobID,
		jobRun.DagID,
		jobRun.Status,
		jobRun.StartTS.String(),
		jobRun.ErrorMessage,
		toJSON(linkedJob),
		toJSON(linkedDAG),
		toJSON(semanticTerms),
		toJSON(recentEvents),
		toJSON(tenantScope),
	)
}

// ExplainIncidentPrompt builds a prompt to explain a multi-tenant incident
func (pb *PromptBuilder) ExplainIncidentPrompt(
	incident *audit.IncidentEvent,
	jobRuns []map[string]interface{},
	dags []map[string]interface{},
	semanticTerms []map[string]interface{},
	tenantScope []string,
) string {
	prompt := `You are analyzing an INCIDENT in a multi-tenant semantic graph.

INCIDENT:
- IncidentID: %s
- Started: %s
- Ended: %s
- Event Count: %d

RELATED JOB RUNS:
%s

RELATED DAGs:
%s

SEMANTIC TERMS INVOLVED:
%s

TENANT SCOPE (only reference these tenants):
%s

TASKS:
1. Summarize the incident in 2–3 sentences.
2. Identify the root cause, referencing semantic terms and jobs.
3. List impacted tenants (within scope only).
4. Describe operational blast radius (jobs, DAGs, reports).
5. Recommend remediation steps.
6. Propose a ChangeSet summary if appropriate.

CONSTRAINTS:
- Do NOT mention any tenants outside the allowed scope.
- Be thorough but concise.
- Highlight multi-tenant implications if any.

Return ONLY valid JSON with these fields:
{
  "narrative": string,
  "rootCause": string,
  "blastRadius": string,
  "impactedTenants": [string],
  "recommendedFix": string,
  "suggestedChangeSetSummary": string
}
`

	return fmt.Sprintf(
		prompt,
		incident.IncidentID,
		incident.StartTS.String(),
		func() string {
			if !incident.EndTS.IsZero() {
				return incident.EndTS.String()
			}
			return "ongoing"
		}(),
		incident.EventCount,
		toJSON(jobRuns),
		toJSON(dags),
		toJSON(semanticTerms),
		toJSON(tenantScope),
	)
}

// AssessChangeSetPrompt builds a prompt for Tenant Admin to assess a ChangeSet
func (pb *PromptBuilder) AssessChangeSetPrompt(
	changeSet *audit.ChangeSetEvent,
	impactedEntities []audit.ImpactedEntity,
	recentIncidents []map[string]interface{},
	complianceContext map[string]interface{},
	tenantScope []string,
) string {
	prompt := `You are assisting a Tenant Admin reviewing a proposed ChangeSet.

CHANGESET:
- Title: %s
- Description: %s
- Source: %s
- Status: %s

IMPACTED ENTITIES:
%s

RECENT INCIDENTS (related to these entities):
%s

COMPLIANCE CONTEXT:
%s

TENANT SCOPE:
%s

TASKS:
1. Explain what this ChangeSet is trying to achieve (1-2 sentences).
2. Assess the risk of applying it (LOW, MEDIUM, HIGH) and justify.
3. Highlight any compliance implications (positive or negative).
4. Suggest any additional checks the Tenant Admin should consider before approval.

Return ONLY valid JSON with these fields:
{
  "narrative": string,
  "riskLevel": "LOW" | "MEDIUM" | "HIGH",
  "riskRationale": string,
  "complianceNotes": string,
  "recommendedChecks": [string]
}
`

	return fmt.Sprintf(
		prompt,
		changeSet.Title,
		changeSet.Description,
		changeSet.Source,
		changeSet.Status,
		toJSON(impactedEntities),
		toJSON(recentIncidents),
		toJSON(complianceContext),
		toJSON(tenantScope),
	)
}

// PostHocRemediationPrompt builds a prompt to explain a completed remediation chain
func (pb *PromptBuilder) PostHocRemediationPrompt(
	initialFailure map[string]interface{},
	aiSuggestion map[string]interface{},
	changeSet *audit.ChangeSetEvent,
	snapshots []map[string]interface{},
	dagVersions []map[string]interface{},
	subsequentRuns []map[string]interface{},
) string {
	prompt := `You are explaining a completed remediation chain for audit and compliance.

INITIAL FAILURE:
%s

AI SUGGESTION:
%s

CHANGESET:
- Title: %s
- Description: %s
- Status: %s

SEMANTIC SNAPSHOTS CREATED:
%s

DAG VERSIONS CREATED:
%s

SUBSEQUENT RUNS (after remediation):
%s

TASKS:
1. Tell the story of what went wrong and how it was fixed.
2. Show how the ChangeSet addressed the root cause.
3. Demonstrate that subsequent runs are now healthy.
4. Highlight any compliance improvements.

Return ONLY valid JSON with these fields:
{
  "narrative": string,
  "rootCause": string,
  "remediationSummary": string,
  "evidenceOfSuccess": [string],
  "complianceImpact": string
}
`

	return fmt.Sprintf(
		prompt,
		toJSON(initialFailure),
		toJSON(aiSuggestion),
		changeSet.Title,
		changeSet.Description,
		changeSet.Status,
		toJSON(snapshots),
		toJSON(dagVersions),
		toJSON(subsequentRuns),
	)
}

// BuildGraphAwarePrompt constructs a graph-aware prompt with context from the catalog
func (pb *PromptBuilder) BuildGraphAwarePrompt(
	ctx context.Context,
	event map[string]interface{},
	catalogWriter catalog.Writer,
	tenantScope []string,
) (string, error) {
	// In a real implementation:
	// 1. Extract node ID from event
	// 2. Query catalogWriter.GetEdges for related nodes
	// 3. Fetch neighbor node details
	// 4. Build enriched context
	// 5. Return appropriate prompt based on event type

	// For now, return a basic template
	return fmt.Sprintf(`Analyze the following event and provide insights:

EVENT:
%s

TENANT SCOPE:
%s

Provide structured analysis in JSON format.
`, toJSON(event), toJSON(tenantScope)), nil
}

// ============================================================================
// Helper functions
// ============================================================================

func toJSON(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return string(b)
}

// ============================================================================
// LLMGenerator defines the interface for calling LLMs (formerly AIService)
type LLMGenerator interface {
	GenerateJSON(ctx context.Context, prompt string, schema string) (string, error)
}

// ExplainService wraps prompt building and AI calling for explanations
type ExplainService struct {
	promptBuilder *PromptBuilder
	generator     LLMGenerator
	logger        interface{}
}

// NewExplainService creates a new explanation service
func NewExplainService(pb *PromptBuilder, gen LLMGenerator) *ExplainService {
	return &ExplainService{
		promptBuilder: pb,
		generator:     gen,
	}
}

// ExplainJobRun builds a prompt and calls AI to explain a job run
func (es *ExplainService) ExplainJobRun(
	ctx context.Context,
	jobRun *audit.JobRunEvent,
	linkedJob map[string]interface{},
	linkedDAG map[string]interface{},
	semanticTerms []map[string]interface{},
	recentEvents []map[string]interface{},
	tenantScope []string,
) (map[string]interface{}, error) {
	prompt := es.promptBuilder.ExplainJobRunPrompt(
		jobRun,
		linkedJob,
		linkedDAG,
		semanticTerms,
		recentEvents,
		tenantScope,
	)

	result, err := es.generator.GenerateJSON(ctx, prompt, "ExplainJobRunResponse")
	if err != nil {
		return nil, fmt.Errorf("ai service error: %w", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		return nil, fmt.Errorf("parse ai response: %w", err)
	}

	return response, nil
}

// ExplainIncident builds a prompt and calls AI to explain an incident
func (es *ExplainService) ExplainIncident(
	ctx context.Context,
	incident *audit.IncidentEvent,
	jobRuns []map[string]interface{},
	dags []map[string]interface{},
	semanticTerms []map[string]interface{},
	tenantScope []string,
) (map[string]interface{}, error) {
	prompt := es.promptBuilder.ExplainIncidentPrompt(
		incident,
		jobRuns,
		dags,
		semanticTerms,
		tenantScope,
	)

	result, err := es.generator.GenerateJSON(ctx, prompt, "ExplainIncidentResponse")
	if err != nil {
		return nil, fmt.Errorf("ai service error: %w", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		return nil, fmt.Errorf("parse ai response: %w", err)
	}

	return response, nil
}

// AssessChangeSet builds a prompt and calls AI to assess a ChangeSet
func (es *ExplainService) AssessChangeSet(
	ctx context.Context,
	changeSet *audit.ChangeSetEvent,
	impactedEntities []audit.ImpactedEntity,
	recentIncidents []map[string]interface{},
	complianceContext map[string]interface{},
	tenantScope []string,
) (map[string]interface{}, error) {
	prompt := es.promptBuilder.AssessChangeSetPrompt(
		changeSet,
		impactedEntities,
		recentIncidents,
		complianceContext,
		tenantScope,
	)

	result, err := es.generator.GenerateJSON(ctx, prompt, "AssessChangeSetResponse")
	if err != nil {
		return nil, fmt.Errorf("ai service error: %w", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		return nil, fmt.Errorf("parse ai response: %w", err)
	}

	return response, nil
}

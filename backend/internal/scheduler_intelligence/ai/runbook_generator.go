package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RunbookGenerator creates automated runbooks for incident response
type RunbookGenerator struct {
	llmClient LLMClient
	logger    *slog.Logger
}

// NewRunbookGenerator creates a new runbook generator
func NewRunbookGenerator(llmClient LLMClient, logger *slog.Logger) *RunbookGenerator {
	return &RunbookGenerator{
		llmClient: llmClient,
		logger:    logger,
	}
}

// RunbookContext provides context for runbook generation
type RunbookContext struct {
	JobID           uuid.UUID        `json:"job_id"`
	JobName         string           `json:"job_name"`
	JobType         string           `json:"job_type"`
	JobDescription  string           `json:"job_description,omitempty"`
	FailurePatterns []FailurePattern `json:"failure_patterns"`
	Dependencies    []DependencyInfo `json:"dependencies,omitempty"`
	Team            string           `json:"team,omitempty"`
	Environment     string           `json:"environment"`
	ExistingRunbook *Runbook         `json:"existing_runbook,omitempty"`
}

// FailurePattern describes a common failure scenario
type FailurePattern struct {
	ErrorType       string   `json:"error_type"`
	Frequency       int      `json:"frequency"`
	AvgImpactMins   float64  `json:"avg_impact_minutes"`
	CommonCauses    []string `json:"common_causes"`
	SuccessfulFixes []string `json:"successful_fixes,omitempty"`
}

// DependencyInfo describes job dependencies
type DependencyInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // database, api, service, file
	HealthURL   string `json:"health_url,omitempty"`
	Criticality string `json:"criticality"` // critical, important, optional
}

// Runbook is the generated runbook document
type Runbook struct {
	ID             string           `json:"id"`
	Version        int              `json:"version"`
	JobID          uuid.UUID        `json:"job_id"`
	JobName        string           `json:"job_name"`
	Title          string           `json:"title"`
	Overview       string           `json:"overview"`
	Sections       []RunbookSection `json:"sections"`
	QuickActions   []QuickAction    `json:"quick_actions"`
	EscalationPath []EscalationStep `json:"escalation_path"`
	LastUpdated    time.Time        `json:"last_updated"`
	GeneratedBy    string           `json:"generated_by"`
	Confidence     float64          `json:"confidence"`
}

// RunbookSection is a section of the runbook
type RunbookSection struct {
	Title       string           `json:"title"`
	Order       int              `json:"order"`
	Content     string           `json:"content"`
	Steps       []RunbookStep    `json:"steps,omitempty"`
	Subsections []RunbookSection `json:"subsections,omitempty"`
}

// RunbookStep is an actionable step
type RunbookStep struct {
	Number    int      `json:"number"`
	Action    string   `json:"action"`
	Command   string   `json:"command,omitempty"` // CLI command if applicable
	Expected  string   `json:"expected,omitempty"`
	IfFails   string   `json:"if_fails,omitempty"`
	Notes     []string `json:"notes,omitempty"`
	Automated bool     `json:"automated"`
}

// QuickAction is a one-click remediation
type QuickAction struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	Command       string `json:"command"`
	IsDestructive bool   `json:"is_destructive"`
	Approval      string `json:"approval_required"` // none, team_lead, manager
}

// EscalationStep defines when and who to escalate to
type EscalationStep struct {
	Level       int    `json:"level"`
	Trigger     string `json:"trigger"`
	Contact     string `json:"contact"`
	Channel     string `json:"channel"` // slack, pager, phone
	TimeoutMins int    `json:"timeout_minutes"`
}

// GenerateRunbook creates a runbook for a job
func (g *RunbookGenerator) GenerateRunbook(ctx context.Context, context RunbookContext) (*Runbook, error) {
	g.logger.Info("Generating runbook",
		"job_id", context.JobID,
		"job_name", context.JobName,
		"failure_patterns", len(context.FailurePatterns),
	)

	// Try LLM-based generation
	if g.llmClient != nil {
		runbook, err := g.generateWithLLM(ctx, context)
		if err == nil {
			return runbook, nil
		}
		g.logger.Warn("LLM generation failed, using template", "error", err)
	}

	// Fall back to template-based generation
	return g.generateFromTemplate(context), nil
}

// generateWithLLM uses AI to create a comprehensive runbook
func (g *RunbookGenerator) generateWithLLM(ctx context.Context, context RunbookContext) (*Runbook, error) {
	prompt := g.buildPrompt(context)

	response, err := g.llmClient.Generate(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return g.parseResponse(response, context)
}

// buildPrompt creates the LLM prompt
func (g *RunbookGenerator) buildPrompt(context RunbookContext) string {
	var sb strings.Builder

	sb.WriteString(`You are a Site Reliability Engineer creating an incident runbook. Generate a comprehensive runbook for the following job.

## Job Information
`)
	sb.WriteString(fmt.Sprintf("Name: %s\n", context.JobName))
	sb.WriteString(fmt.Sprintf("Type: %s\n", context.JobType))
	if context.JobDescription != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", context.JobDescription))
	}

	if len(context.FailurePatterns) > 0 {
		sb.WriteString("\n## Common Failure Patterns\n")
		for _, fp := range context.FailurePatterns {
			sb.WriteString(fmt.Sprintf("- %s (occurs %d times, avg impact: %.1f min)\n",
				fp.ErrorType, fp.Frequency, fp.AvgImpactMins))
			for _, cause := range fp.CommonCauses {
				sb.WriteString(fmt.Sprintf("  - Cause: %s\n", cause))
			}
		}
	}

	if len(context.Dependencies) > 0 {
		sb.WriteString("\n## Dependencies\n")
		for _, dep := range context.Dependencies {
			sb.WriteString(fmt.Sprintf("- %s (%s, %s)\n", dep.Name, dep.Type, dep.Criticality))
		}
	}

	sb.WriteString(`
## Output Format
Generate a JSON runbook with:
{
  "title": "Runbook Title",
  "overview": "Brief overview",
  "sections": [
    {
      "title": "Section Title",
      "order": 1,
      "content": "Section description",
      "steps": [
        {"number": 1, "action": "What to do", "command": "optional CLI command", "expected": "What should happen", "if_fails": "What to do if it fails"}
      ]
    }
  ],
  "quick_actions": [
    {"name": "Action Name", "description": "What it does", "command": "CLI command", "is_destructive": false, "approval_required": "none"}
  ],
  "escalation_path": [
    {"level": 1, "trigger": "When to escalate", "contact": "Who to contact", "channel": "slack", "timeout_minutes": 15}
  ]
}

Include sections for:
1. Initial Triage
2. Common Issues and Fixes (based on failure patterns)
3. Dependency Health Checks
4. Recovery Procedures
5. Post-Incident Steps
`)

	return sb.String()
}

// parseResponse extracts the runbook from LLM response
func (g *RunbookGenerator) parseResponse(response string, context RunbookContext) (*Runbook, error) {
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	var parsed struct {
		Title          string           `json:"title"`
		Overview       string           `json:"overview"`
		Sections       []RunbookSection `json:"sections"`
		QuickActions   []QuickAction    `json:"quick_actions"`
		EscalationPath []EscalationStep `json:"escalation_path"`
	}

	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	runbook := &Runbook{
		ID:             uuid.NewString()[:8],
		Version:        1,
		JobID:          context.JobID,
		JobName:        context.JobName,
		Title:          parsed.Title,
		Overview:       parsed.Overview,
		Sections:       parsed.Sections,
		QuickActions:   parsed.QuickActions,
		EscalationPath: parsed.EscalationPath,
		LastUpdated:    time.Now(),
		GeneratedBy:    "ai",
		Confidence:     0.8,
	}

	return runbook, nil
}

// generateFromTemplate creates a runbook using templates
func (g *RunbookGenerator) generateFromTemplate(context RunbookContext) *Runbook {
	runbook := &Runbook{
		ID:          uuid.NewString()[:8],
		Version:     1,
		JobID:       context.JobID,
		JobName:     context.JobName,
		Title:       fmt.Sprintf("Runbook: %s", context.JobName),
		Overview:    fmt.Sprintf("Incident response runbook for the %s job (%s)", context.JobName, context.JobType),
		LastUpdated: time.Now(),
		GeneratedBy: "template",
		Confidence:  0.6,
	}

	// Initial Triage section
	runbook.Sections = append(runbook.Sections, RunbookSection{
		Title:   "Initial Triage",
		Order:   1,
		Content: "Quickly assess the situation and gather initial information.",
		Steps: []RunbookStep{
			{Number: 1, Action: "Check job status in scheduler console", Expected: "See current state and error message"},
			{Number: 2, Action: "Review recent job runs for patterns", Expected: "Identify if this is new or recurring"},
			{Number: 3, Action: "Check dependency health", Expected: "All dependencies showing healthy"},
			{Number: 4, Action: "Assess blast radius", Expected: "Understand downstream impact"},
		},
	})

	// Common Issues section based on failure patterns
	if len(context.FailurePatterns) > 0 {
		issuesSection := RunbookSection{
			Title:   "Common Issues and Fixes",
			Order:   2,
			Content: "Known failure patterns and their resolutions.",
		}

		for _, fp := range context.FailurePatterns {
			subsection := RunbookSection{
				Title:   fp.ErrorType,
				Content: fmt.Sprintf("Frequency: %d occurrences, Avg impact: %.1f minutes", fp.Frequency, fp.AvgImpactMins),
			}

			stepNum := 1
			for _, cause := range fp.CommonCauses {
				subsection.Steps = append(subsection.Steps, RunbookStep{
					Number:    stepNum,
					Action:    fmt.Sprintf("Check for: %s", cause),
					Automated: false,
				})
				stepNum++
			}

			for _, fix := range fp.SuccessfulFixes {
				subsection.Steps = append(subsection.Steps, RunbookStep{
					Number:    stepNum,
					Action:    fix,
					Automated: false,
				})
				stepNum++
			}

			issuesSection.Subsections = append(issuesSection.Subsections, subsection)
		}

		runbook.Sections = append(runbook.Sections, issuesSection)
	}

	// Dependency Health section
	if len(context.Dependencies) > 0 {
		depSection := RunbookSection{
			Title:   "Dependency Health Checks",
			Order:   3,
			Content: "Verify all upstream dependencies are healthy.",
		}

		stepNum := 1
		for _, dep := range context.Dependencies {
			step := RunbookStep{
				Number: stepNum,
				Action: fmt.Sprintf("Check %s (%s)", dep.Name, dep.Type),
			}
			if dep.HealthURL != "" {
				step.Command = fmt.Sprintf("curl -s %s | jq '.status'", dep.HealthURL)
				step.Expected = "healthy"
			}
			depSection.Steps = append(depSection.Steps, step)
			stepNum++
		}

		runbook.Sections = append(runbook.Sections, depSection)
	}

	// Recovery section
	runbook.Sections = append(runbook.Sections, RunbookSection{
		Title:   "Recovery Procedures",
		Order:   4,
		Content: "Steps to recover from a failure.",
		Steps: []RunbookStep{
			{Number: 1, Action: "Attempt manual retry of the job", Command: "scheduler job trigger <job_id>", Expected: "Job starts successfully"},
			{Number: 2, Action: "If retry fails, check for resource issues", Expected: "Sufficient resources available"},
			{Number: 3, Action: "Review job parameters and configuration", Expected: "Configuration is correct"},
			{Number: 4, Action: "Check for data issues in upstream sources", Expected: "Data is available and valid"},
		},
	})

	// Post-incident section
	runbook.Sections = append(runbook.Sections, RunbookSection{
		Title:   "Post-Incident Steps",
		Order:   5,
		Content: "Complete these steps after resolving the incident.",
		Steps: []RunbookStep{
			{Number: 1, Action: "Verify job completed successfully", Expected: "Job shows success status"},
			{Number: 2, Action: "Verify downstream jobs are unblocked", Expected: "No pending downstream failures"},
			{Number: 3, Action: "Document root cause in incident log", Automated: false},
			{Number: 4, Action: "Create follow-up ticket if needed", Automated: false},
			{Number: 5, Action: "Update this runbook with learnings", Automated: false},
		},
	})

	// Quick actions
	runbook.QuickActions = []QuickAction{
		{
			Name:          "Retry Job",
			Description:   "Trigger a manual retry of the job",
			Command:       fmt.Sprintf("scheduler job trigger %s", context.JobID),
			IsDestructive: false,
			Approval:      "none",
		},
		{
			Name:          "Pause Job",
			Description:   "Pause the job to prevent further failures",
			Command:       fmt.Sprintf("scheduler job pause %s", context.JobID),
			IsDestructive: false,
			Approval:      "team_lead",
		},
		{
			Name:          "Skip Failed Step",
			Description:   "Skip the failed step and continue (if applicable)",
			Command:       fmt.Sprintf("scheduler job skip-step %s --step <step_id>", context.JobID),
			IsDestructive: true,
			Approval:      "manager",
		},
	}

	// Escalation path
	runbook.EscalationPath = []EscalationStep{
		{Level: 1, Trigger: "Issue not resolved in 15 minutes", Contact: "On-call engineer", Channel: "slack", TimeoutMins: 15},
		{Level: 2, Trigger: "Issue not resolved in 30 minutes", Contact: "Team lead", Channel: "pager", TimeoutMins: 30},
		{Level: 3, Trigger: "Business-critical impact or prolonged outage", Contact: "Engineering manager", Channel: "phone", TimeoutMins: 60},
	}

	return runbook
}

// UpdateRunbook updates an existing runbook with new learnings
func (g *RunbookGenerator) UpdateRunbook(ctx context.Context, existing *Runbook, newPatterns []FailurePattern) (*Runbook, error) {
	g.logger.Info("Updating runbook",
		"runbook_id", existing.ID,
		"new_patterns", len(newPatterns),
	)

	// Create updated context
	context := RunbookContext{
		JobID:           existing.JobID,
		JobName:         existing.JobName,
		FailurePatterns: newPatterns,
		ExistingRunbook: existing,
	}

	// Generate new runbook incorporating learnings
	updated, err := g.GenerateRunbook(ctx, context)
	if err != nil {
		return nil, err
	}

	// Increment version
	updated.Version = existing.Version + 1
	updated.ID = existing.ID

	return updated, nil
}

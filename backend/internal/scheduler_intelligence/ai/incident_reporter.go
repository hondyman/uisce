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

// IncidentReporter generates AI-powered incident reports with root cause analysis
type IncidentReporter struct {
	llmClient LLMClient
	logger    *slog.Logger
}

// NewIncidentReporter creates a new incident reporter
func NewIncidentReporter(llmClient LLMClient, logger *slog.Logger) *IncidentReporter {
	return &IncidentReporter{
		llmClient: llmClient,
		logger:    logger,
	}
}

// IncidentContext contains data about the incident
type IncidentContext struct {
	TenantID      uuid.UUID        `json:"tenant_id"`
	JobID         uuid.UUID        `json:"job_id,omitempty"`
	DAGID         uuid.UUID        `json:"dag_id,omitempty"`
	JobName       string           `json:"job_name"`
	FailedAt      time.Time        `json:"failed_at"`
	ErrorMessage  string           `json:"error_message"`
	ErrorStack    string           `json:"error_stack,omitempty"`
	FailedStep    string           `json:"failed_step,omitempty"`
	RecentRuns    []RunSummary     `json:"recent_runs"`
	RelatedJobs   []RelatedJobInfo `json:"related_jobs,omitempty"`
	SystemMetrics *SystemMetrics   `json:"system_metrics,omitempty"`
}

// RunSummary summarizes a job run
type RunSummary struct {
	RunID      string    `json:"run_id"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Status     string    `json:"status"`
	DurationMS int64     `json:"duration_ms"`
	Error      string    `json:"error,omitempty"`
}

// RelatedJobInfo provides context about related jobs
type RelatedJobInfo struct {
	JobID      uuid.UUID `json:"job_id"`
	JobName    string    `json:"job_name"`
	Relation   string    `json:"relation"` // upstream, downstream, same_dag
	LastStatus string    `json:"last_status"`
}

// SystemMetrics contains system state at failure time
type SystemMetrics struct {
	CPUUsage    float64 `json:"cpu_usage_pct"`
	MemoryUsage float64 `json:"memory_usage_pct"`
	DiskIO      float64 `json:"disk_io_pct"`
	NetworkSat  float64 `json:"network_saturation_pct"`
}

// IncidentReport is the generated report
type IncidentReport struct {
	IncidentID        string           `json:"incident_id"`
	Title             string           `json:"title"`
	Severity          string           `json:"severity"` // P1, P2, P3, P4
	Summary           string           `json:"summary"`
	RootCauseAnalysis RootCause        `json:"root_cause_analysis"`
	Impact            ImpactAnalysis   `json:"impact"`
	Timeline          []TimelineEvent  `json:"timeline"`
	Recommendations   []Recommendation `json:"recommendations"`
	SimilarIncidents  []string         `json:"similar_incidents,omitempty"`
	GeneratedAt       time.Time        `json:"generated_at"`
}

// RootCause describes the probable root cause
type RootCause struct {
	Category    string   `json:"category"` // infrastructure, code, data, dependency, config
	Description string   `json:"description"`
	Confidence  float64  `json:"confidence"`
	Evidence    []string `json:"evidence"`
}

// ImpactAnalysis describes the blast radius
type ImpactAnalysis struct {
	AffectedTenants []string `json:"affected_tenants"`
	AffectedJobs    []string `json:"affected_jobs"`
	SLOImpact       string   `json:"slo_impact"`
	BusinessImpact  string   `json:"business_impact"`
	DataImpact      string   `json:"data_impact"`
}

// TimelineEvent is a point in the incident timeline
type TimelineEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Event     string    `json:"event"`
	Source    string    `json:"source"`
}

// Recommendation is a suggested action
type Recommendation struct {
	Priority    int    `json:"priority"` // 1 = immediate, 2 = short term, 3 = long term
	Action      string `json:"action"`
	Effort      string `json:"effort"` // low, medium, high
	Description string `json:"description"`
}

// GenerateReport creates an AI-powered incident report
func (r *IncidentReporter) GenerateReport(ctx context.Context, incident IncidentContext) (*IncidentReport, error) {
	r.logger.Info("Generating incident report",
		"job_name", incident.JobName,
		"failed_at", incident.FailedAt,
	)

	// Build prompt for LLM
	prompt := r.buildPrompt(incident)

	// Generate AI analysis
	response, err := r.llmClient.Generate(ctx, prompt)
	if err != nil {
		// Fall back to rule-based analysis
		r.logger.Warn("LLM generation failed, using rule-based analysis", "error", err)
		return r.generateRuleBasedReport(incident), nil
	}

	// Parse LLM response
	report, err := r.parseResponse(response, incident)
	if err != nil {
		r.logger.Warn("Failed to parse LLM response, using rule-based", "error", err)
		return r.generateRuleBasedReport(incident), nil
	}

	r.logger.Info("Incident report generated",
		"incident_id", report.IncidentID,
		"severity", report.Severity,
	)

	return report, nil
}

// buildPrompt creates the LLM prompt for incident analysis
func (r *IncidentReporter) buildPrompt(incident IncidentContext) string {
	var sb strings.Builder

	sb.WriteString(`You are an expert Site Reliability Engineer analyzing a job failure incident. Generate a detailed incident report with root cause analysis.

## Incident Details
`)
	sb.WriteString(fmt.Sprintf("Job: %s\n", incident.JobName))
	sb.WriteString(fmt.Sprintf("Failed At: %s\n", incident.FailedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Error: %s\n", incident.ErrorMessage))

	if incident.FailedStep != "" {
		sb.WriteString(fmt.Sprintf("Failed Step: %s\n", incident.FailedStep))
	}

	if len(incident.RecentRuns) > 0 {
		sb.WriteString("\n## Recent Run History\n")
		for _, run := range incident.RecentRuns {
			sb.WriteString(fmt.Sprintf("- %s: %s (%dms)\n", run.StartTime.Format("2006-01-02 15:04"), run.Status, run.DurationMS))
		}
	}

	if incident.SystemMetrics != nil {
		sb.WriteString("\n## System Metrics at Failure\n")
		sb.WriteString(fmt.Sprintf("- CPU: %.1f%%\n", incident.SystemMetrics.CPUUsage))
		sb.WriteString(fmt.Sprintf("- Memory: %.1f%%\n", incident.SystemMetrics.MemoryUsage))
	}

	sb.WriteString(`
## Output Format
Respond with JSON:
{
  "severity": "P1|P2|P3|P4",
  "summary": "Brief incident summary",
  "root_cause": {
    "category": "infrastructure|code|data|dependency|config",
    "description": "Detailed root cause explanation",
    "confidence": 0.85,
    "evidence": ["Evidence point 1", "Evidence point 2"]
  },
  "recommendations": [
    {"priority": 1, "action": "Immediate action", "effort": "low", "description": "Details"},
    {"priority": 2, "action": "Short term fix", "effort": "medium", "description": "Details"}
  ]
}
`)

	return sb.String()
}

// parseResponse extracts the report from LLM response
func (r *IncidentReporter) parseResponse(response string, incident IncidentContext) (*IncidentReport, error) {
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	var parsed struct {
		Severity        string           `json:"severity"`
		Summary         string           `json:"summary"`
		RootCause       RootCause        `json:"root_cause"`
		Recommendations []Recommendation `json:"recommendations"`
	}

	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return nil, err
	}

	report := &IncidentReport{
		IncidentID:        uuid.NewString()[:8],
		Title:             fmt.Sprintf("[%s] %s Failure", parsed.Severity, incident.JobName),
		Severity:          parsed.Severity,
		Summary:           parsed.Summary,
		RootCauseAnalysis: parsed.RootCause,
		Recommendations:   parsed.Recommendations,
		GeneratedAt:       time.Now(),
	}

	// Add timeline
	report.Timeline = r.buildTimeline(incident)

	// Analyze impact
	report.Impact = r.analyzeImpact(incident)

	return report, nil
}

// generateRuleBasedReport creates a report without LLM
func (r *IncidentReporter) generateRuleBasedReport(incident IncidentContext) *IncidentReport {
	// Determine severity based on patterns
	severity := r.classifySeverity(incident)

	// Classify root cause
	rootCause := r.classifyRootCause(incident)

	report := &IncidentReport{
		IncidentID:        uuid.NewString()[:8],
		Title:             fmt.Sprintf("[%s] %s Failure", severity, incident.JobName),
		Severity:          severity,
		Summary:           fmt.Sprintf("Job '%s' failed at %s: %s", incident.JobName, incident.FailedAt.Format("15:04"), incident.ErrorMessage),
		RootCauseAnalysis: rootCause,
		Timeline:          r.buildTimeline(incident),
		Impact:            r.analyzeImpact(incident),
		Recommendations:   r.generateRecommendations(rootCause),
		GeneratedAt:       time.Now(),
	}

	return report
}

// classifySeverity determines incident severity
func (r *IncidentReporter) classifySeverity(incident IncidentContext) string {
	// Count recent failures
	failures := 0
	for _, run := range incident.RecentRuns {
		if run.Status == "failed" {
			failures++
		}
	}

	// Check for SLO-impacting patterns
	if failures >= 3 {
		return "P2"
	}
	if strings.Contains(strings.ToLower(incident.ErrorMessage), "timeout") {
		return "P3"
	}
	return "P4"
}

// classifyRootCause attempts to categorize the root cause
func (r *IncidentReporter) classifyRootCause(incident IncidentContext) RootCause {
	error := strings.ToLower(incident.ErrorMessage)

	rc := RootCause{
		Confidence: 0.6,
		Evidence:   []string{fmt.Sprintf("Error: %s", incident.ErrorMessage)},
	}

	switch {
	case strings.Contains(error, "timeout"):
		rc.Category = "infrastructure"
		rc.Description = "Job timed out, possibly due to resource constraints or slow downstream systems"
	case strings.Contains(error, "connection"):
		rc.Category = "infrastructure"
		rc.Description = "Connection failure to external service or database"
	case strings.Contains(error, "permission") || strings.Contains(error, "auth"):
		rc.Category = "config"
		rc.Description = "Authentication or permission issue"
	case strings.Contains(error, "null") || strings.Contains(error, "invalid"):
		rc.Category = "data"
		rc.Description = "Data quality issue or unexpected null values"
	default:
		rc.Category = "code"
		rc.Description = "Unhandled exception or logic error"
	}

	return rc
}

// buildTimeline creates incident timeline
func (r *IncidentReporter) buildTimeline(incident IncidentContext) []TimelineEvent {
	var timeline []TimelineEvent

	// Add failure event
	timeline = append(timeline, TimelineEvent{
		Timestamp: incident.FailedAt,
		Event:     fmt.Sprintf("Job '%s' failed: %s", incident.JobName, incident.ErrorMessage),
		Source:    "scheduler",
	})

	// Add recent run context
	for _, run := range incident.RecentRuns {
		if run.Status == "failed" {
			timeline = append(timeline, TimelineEvent{
				Timestamp: run.StartTime,
				Event:     fmt.Sprintf("Previous failure: %s", run.Error),
				Source:    "history",
			})
		}
	}

	return timeline
}

// analyzeImpact determines the blast radius
func (r *IncidentReporter) analyzeImpact(incident IncidentContext) ImpactAnalysis {
	impact := ImpactAnalysis{
		AffectedTenants: []string{incident.TenantID.String()},
		AffectedJobs:    []string{incident.JobName},
	}

	// Add downstream jobs
	for _, related := range incident.RelatedJobs {
		if related.Relation == "downstream" {
			impact.AffectedJobs = append(impact.AffectedJobs, related.JobName)
		}
	}

	impact.SLOImpact = "potential"
	impact.BusinessImpact = "Data freshness delayed"
	impact.DataImpact = "Downstream aggregations may be stale"

	return impact
}

// generateRecommendations creates fix suggestions
func (r *IncidentReporter) generateRecommendations(rootCause RootCause) []Recommendation {
	var recs []Recommendation

	switch rootCause.Category {
	case "infrastructure":
		recs = append(recs, Recommendation{
			Priority:    1,
			Action:      "Increase timeout",
			Effort:      "low",
			Description: "Increase job timeout to handle slow responses",
		})
		recs = append(recs, Recommendation{
			Priority:    2,
			Action:      "Add retry policy",
			Effort:      "low",
			Description: "Add exponential backoff retry for transient failures",
		})
	case "config":
		recs = append(recs, Recommendation{
			Priority:    1,
			Action:      "Refresh credentials",
			Effort:      "low",
			Description: "Verify and refresh authentication credentials",
		})
	case "data":
		recs = append(recs, Recommendation{
			Priority:    1,
			Action:      "Add data validation",
			Effort:      "medium",
			Description: "Add null checks and data validation before processing",
		})
	default:
		recs = append(recs, Recommendation{
			Priority:    1,
			Action:      "Review error logs",
			Effort:      "medium",
			Description: "Investigate stack trace and recent code changes",
		})
	}

	return recs
}

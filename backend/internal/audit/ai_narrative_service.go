package audit

import (
	"context"
	"encoding/json"
	"fmt"
)

// AIAuditNarrativeService generates AI-powered audit narratives
type AIAuditNarrativeService struct {
	aiClient AuditNarrativeAIClient // Interface to your AI orchestration layer
}

// AuditNarrativeAIClient interface for AI narrative generation
type AuditNarrativeAIClient interface {
	GenerateNarrative(ctx context.Context, prompt string, context map[string]interface{}) (string, error)
}

// NewAIAuditNarrativeService creates a new AI audit narrative service
func NewAIAuditNarrativeService(aiClient AuditNarrativeAIClient) *AIAuditNarrativeService {
	return &AIAuditNarrativeService{
		aiClient: aiClient,
	}
}

// AuditNarrativeRequest contains all context for AI narrative generation
type AuditNarrativeRequest struct {
	AuditRecord       interface{} `json:"audit_record"`
	SemanticContext   interface{} `json:"semantic_context,omitempty"`
	ComplianceContext interface{} `json:"compliance_context,omitempty"`
	SchedulerContext  interface{} `json:"scheduler_context,omitempty"`
	TenantContext     interface{} `json:"tenant_context,omitempty"`
}

// AuditNarrativeResponse contains AI-generated insights
type AuditNarrativeResponse struct {
	Narrative               string   `json:"narrative"`
	RootCause               string   `json:"root_cause"`
	BlastRadius             string   `json:"blast_radius"`
	RecommendedFix          string   `json:"recommended_fix"`
	SuggestedChangeSetTitle string   `json:"suggested_changeset_title"`
	SuggestedChangeSetBody  string   `json:"suggested_changeset_body"`
	AffectedSemanticTerms   []string `json:"affected_semantic_terms"`
	AffectedJobs            []string `json:"affected_jobs"`
	ComplianceImplications  string   `json:"compliance_implications"`
	RiskLevel               string   `json:"risk_level"` // LOW, MEDIUM, HIGH, CRITICAL
	RiskScore               float64  `json:"risk_score"` // 0.0 - 1.0
	Confidence              float64  `json:"confidence"` // 0.0 - 1.0
}

// GenerateJobRunNarrative generates AI narrative for a failed job run
func (s *AIAuditNarrativeService) GenerateJobRunNarrative(ctx context.Context, jobRun SchedulerJobRun) (*AuditNarrativeResponse, error) {
	// Build prompt for AI
	prompt := s.buildJobRunPrompt(jobRun)

	// Prepare context
	contextMap := map[string]interface{}{
		"record_type":        "JOB_RUN",
		"tenant_id":          jobRun.TenantID,
		"job_id":             jobRun.JobID,
		"status":             jobRun.Status,
		"error_message":      jobRun.ErrorMessage,
		"semantic_context":   string(jobRun.SemanticContext),
		"compliance_context": string(jobRun.ComplianceContext),
		"slo_context":        string(jobRun.SLOContext),
	}

	// Call AI service
	responseJSON, err := s.aiClient.GenerateNarrative(ctx, prompt, contextMap)
	if err != nil {
		return nil, fmt.Errorf("failed to generate AI narrative: %w", err)
	}

	// Parse response
	var response AuditNarrativeResponse
	if err := json.Unmarshal([]byte(responseJSON), &response); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return &response, nil
}

// GenerateChangeSetNarrative generates AI narrative for a governance changeset
func (s *AIAuditNarrativeService) GenerateChangeSetNarrative(ctx context.Context, changeSet GovernanceChangeSet) (*AuditNarrativeResponse, error) {
	prompt := s.buildChangeSetPrompt(changeSet)

	contextMap := map[string]interface{}{
		"record_type":       "CHANGESET",
		"tenant_id":         changeSet.TenantID,
		"changeset_id":      changeSet.ChangesetID,
		"type":              changeSet.Type,
		"actor":             changeSet.Actor,
		"payload_old":       string(changeSet.PayloadOld),
		"payload_new":       string(changeSet.PayloadNew),
		"semantic_impact":   string(changeSet.SemanticImpact),
		"compliance_impact": string(changeSet.ComplianceImpact),
		"tenant_impact":     string(changeSet.TenantImpact),
	}

	responseJSON, err := s.aiClient.GenerateNarrative(ctx, prompt, contextMap)
	if err != nil {
		return nil, fmt.Errorf("failed to generate AI narrative: %w", err)
	}

	var response AuditNarrativeResponse
	if err := json.Unmarshal([]byte(responseJSON), &response); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return &response, nil
}

// GenerateComplianceStory generates regulator-ready compliance narrative
func (s *AIAuditNarrativeService) GenerateComplianceStory(ctx context.Context, tenantID string, startDate, endDate string) (string, error) {
	prompt := s.buildComplianceStoryPrompt(tenantID, startDate, endDate)

	contextMap := map[string]interface{}{
		"record_type": "COMPLIANCE_STORY",
		"tenant_id":   tenantID,
		"start_date":  startDate,
		"end_date":    endDate,
	}

	story, err := s.aiClient.GenerateNarrative(ctx, prompt, contextMap)
	if err != nil {
		return "", fmt.Errorf("failed to generate compliance story: %w", err)
	}

	return story, nil
}

// buildJobRunPrompt constructs the AI prompt for job run analysis
func (s *AIAuditNarrativeService) buildJobRunPrompt(jobRun SchedulerJobRun) string {
	return fmt.Sprintf(`You are an expert audit and compliance analyst for a financial data platform.

Analyze the following scheduler job run failure and produce a comprehensive incident narrative.

JOB RUN DETAILS:
- Run ID: %s
- Job ID: %s
- Tenant ID: %s
- Status: %s
- Error: %s
- Start Time: %s
- End Time: %s

SEMANTIC CONTEXT:
%s

COMPLIANCE CONTEXT:
%s

SLO CONTEXT:
%s

REQUIRED OUTPUT (JSON only):
{
  "narrative": "A clear, executive-level summary of what happened",
  "root_cause": "Technical root cause of the failure",
  "blast_radius": "What systems, users, or processes were impacted",
  "recommended_fix": "Concrete steps to remediate this issue",
  "suggested_changeset_title": "Title for a governance changeset to prevent recurrence",
  "suggested_changeset_body": "Detailed proposal for the changeset",
  "affected_semantic_terms": ["list", "of", "semantic", "term", "IDs"],
  "affected_jobs": ["list", "of", "job", "IDs"],
  "compliance_implications": "Any regulatory or compliance concerns",
  "risk_level": "LOW | MEDIUM | HIGH | CRITICAL",
  "risk_score": 0.0 to 1.0,
  "confidence": 0.0 to 1.0
}

Generate the narrative now:`,
		jobRun.RunID,
		jobRun.JobID,
		jobRun.TenantID,
		jobRun.Status,
		jobRun.ErrorMessage,
		jobRun.StartTS.Format("2006-01-02 15:04:05 UTC"),
		jobRun.EndTS.Format("2006-01-02 15:04:05 UTC"),
		string(jobRun.SemanticContext),
		string(jobRun.ComplianceContext),
		string(jobRun.SLOContext),
	)
}

// buildChangeSetPrompt constructs the AI prompt for changeset analysis
func (s *AIAuditNarrativeService) buildChangeSetPrompt(changeSet GovernanceChangeSet) string {
	return fmt.Sprintf(`You are an expert governance and impact analysis specialist.

Analyze the following governance changeset and produce a comprehensive impact narrative.

CHANGESET DETAILS:
- Changeset ID: %s
- Type: %s
- Actor: %s
- Tenant ID: %s
- Created At: %s
- Status: %s

PAYLOAD OLD:
%s

PAYLOAD NEW:
%s

SEMANTIC IMPACT:
%s

COMPLIANCE IMPACT:
%s

TENANT IMPACT:
%s

REQUIRED OUTPUT (JSON only):
{
  "narrative": "Executive summary of this change",
  "root_cause": "Why this change was needed",
  "blast_radius": "What will be affected by this change",
  "recommended_fix": "Steps to safely apply this change",
  "suggested_changeset_title": "Title for this changeset",
  "suggested_changeset_body": "Detailed description",
  "affected_semantic_terms": ["list", "of", "semantic", "term", "IDs"],
  "affected_jobs": ["list", "of", "job", "IDs"],
  "compliance_implications": "Regulatory considerations",
  "risk_level": "LOW | MEDIUM | HIGH | CRITICAL",
  "risk_score": 0.0 to 1.0,
  "confidence": 0.0 to 1.0
}

Generate the impact analysis now:`,
		changeSet.ChangesetID,
		changeSet.Type,
		changeSet.Actor,
		changeSet.TenantID,
		changeSet.CreatedAt.Format("2006-01-02 15:04:05 UTC"),
		changeSet.Status,
		string(changeSet.PayloadOld),
		string(changeSet.PayloadNew),
		string(changeSet.SemanticImpact),
		string(changeSet.ComplianceImpact),
		string(changeSet.TenantImpact),
	)
}

// buildComplianceStoryPrompt constructs the AI prompt for compliance reporting
func (s *AIAuditNarrativeService) buildComplianceStoryPrompt(tenantID, startDate, endDate string) string {
	return fmt.Sprintf(`You are a compliance officer preparing a report for regulators.

Generate a comprehensive compliance narrative for Tenant %s covering the period from %s to %s.

This narrative should include:
- Overview of compliance posture during this period
- Any violations that occurred and how they were remediated
- Time to remediation for each violation
- PII exposure incidents (should be zero)
- Governance changes that improved compliance
- SLO adherence and operational stability
- Recommendations for continued compliance

The narrative should be:
- Clear and professional
- Regulator-ready
- Evidence-based
- Honest about any issues and their resolutions

Generate the compliance story now:`,
		tenantID,
		startDate,
		endDate,
	)
}

// GenerateSLODriftReport generates AI narrative for SLO drift analysis
func (s *AIAuditNarrativeService) GenerateSLODriftReport(ctx context.Context, tenantID string, sloContext map[string]interface{}) (string, error) {
	prompt := fmt.Sprintf(`You are an SLO and operational reliability expert.

Analyze the following SLO context for Tenant %s and generate a drift report.

SLO CONTEXT:
%s

Generate a report that includes:
- Current SLO health
- Trend analysis (improving/degrading)
- Root causes of any drift
- Recommended schedule adjustments
- Jobs at risk of SLO breach
- Preventive actions

Generate the SLO drift report now:`,
		tenantID,
		mustMarshalJSON(sloContext),
	)

	contextMap := map[string]interface{}{
		"record_type": "SLO_DRIFT",
		"tenant_id":   tenantID,
		"slo_context": sloContext,
	}

	report, err := s.aiClient.GenerateNarrative(ctx, prompt, contextMap)
	if err != nil {
		return "", fmt.Errorf("failed to generate SLO drift report: %w", err)
	}

	return report, nil
}

// mustMarshalJSON marshals data to JSON or returns empty object
func mustMarshalJSON(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

// ExplainAuditRecord is a convenience method that routes to the appropriate narrative generator
func (s *AIAuditNarrativeService) ExplainAuditRecord(ctx context.Context, recordType string, record interface{}) (*AuditNarrativeResponse, error) {
	switch recordType {
	case "JOB_RUN":
		if jr, ok := record.(SchedulerJobRun); ok {
			return s.GenerateJobRunNarrative(ctx, jr)
		}
	case "CHANGESET":
		if cs, ok := record.(GovernanceChangeSet); ok {
			return s.GenerateChangeSetNarrative(ctx, cs)
		}
	default:
		return nil, fmt.Errorf("unsupported record type: %s", recordType)
	}
	return nil, fmt.Errorf("invalid record type or casting failed")
}

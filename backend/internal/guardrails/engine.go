package guardrails

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/audit"
)

// GuardrailEngine is a deterministic policy enforcement layer for AI outputs
type GuardrailEngine struct {
	auditService *audit.Service
	policies     *PolicyRegistry
}

// NewGuardrailEngine creates a new guardrail engine
func NewGuardrailEngine(auditService *audit.Service) *GuardrailEngine {
	return &GuardrailEngine{
		auditService: auditService,
		policies:     NewPolicyRegistry(),
	}
}

// GuardrailResult represents the result of guardrail checks
type GuardrailResult struct {
	Approved           bool              `json:"approved"`
	RedactedContent    string            `json:"redacted_content"`
	ViolationsDetected []PolicyViolation `json:"violations_detected"`
	ExecutionTimeMs    int64             `json:"execution_time_ms"`
	PolicyVersionsUsed []string          `json:"policy_versions_used"`
	AuditEventID       string            `json:"audit_event_id"`
}

// PolicyViolation represents a detected policy violation
type PolicyViolation struct {
	PolicyID    string `json:"policy_id"`
	PolicyName  string `json:"policy_name"`
	Severity    string `json:"severity"` // CRITICAL, HIGH, MEDIUM, LOW
	Description string `json:"description"`
	Location    string `json:"location"` // Where in text violation occurred
	Remediation string `json:"remediation"`
}

// FilterAIOutput runs all guardrail checks on AI-generated content
func (g *GuardrailEngine) FilterAIOutput(ctx context.Context, content string, tenantID string, userID string) (*GuardrailResult, error) {
	startTime := time.Now()

	result := &GuardrailResult{
		Approved:           true,
		RedactedContent:    content,
		ViolationsDetected: []PolicyViolation{},
		PolicyVersionsUsed: []string{},
	}

	// 1. PII Detection & Redaction
	piiViolations, redacted := g.detectAndRedactPII(content)
	if len(piiViolations) > 0 {
		result.ViolationsDetected = append(result.ViolationsDetected, piiViolations...)
		result.RedactedContent = redacted
	}

	// 2. Prohibited Content Detection
	prohibitedViolations := g.detectProhibitedContent(result.RedactedContent)
	if len(prohibitedViolations) > 0 {
		result.ViolationsDetected = append(result.ViolationsDetected, prohibitedViolations...)
		result.Approved = false
	}

	// 3. Financial Advice Compliance
	adviceViolations := g.checkFinancialAdviceCompliance(result.RedactedContent)
	if len(adviceViolations) > 0 {
		result.ViolationsDetected = append(result.ViolationsDetected, adviceViolations...)
		result.Approved = false
	}

	// 4. Topic Boundary Enforcement
	topicViolations := g.enforceTopicBoundaries(result.RedactedContent)
	if len(topicViolations) > 0 {
		result.ViolationsDetected = append(result.ViolationsDetected, topicViolations...)
		result.Approved = false
	}

	// 5. Regulatory Keywords Check (SEC, FINRA)
	regulatoryViolations := g.checkRegulatoryKeywords(result.RedactedContent)
	if len(regulatoryViolations) > 0 {
		result.ViolationsDetected = append(result.ViolationsDetected, regulatoryViolations...)

		// Critical regulatory violations block output
		for _, v := range regulatoryViolations {
			if v.Severity == "CRITICAL" {
				result.Approved = false
				break
			}
		}
	}

	result.ExecutionTimeMs = time.Since(startTime).Milliseconds()

	// Audit log the guardrail check
	auditEvent, err := g.auditGuardrailCheck(ctx, tenantID, userID, content, result)
	if err == nil {
		result.AuditEventID = auditEvent.EventID
	}

	return result, nil
}

// detectAndRedactPII identifies and redacts PII using regex patterns
func (g *GuardrailEngine) detectAndRedactPII(content string) ([]PolicyViolation, string) {
	violations := []PolicyViolation{}
	redacted := content

	// SSN pattern: ###-##-####
	ssnPattern := regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	if ssnPattern.MatchString(content) {
		violations = append(violations, PolicyViolation{
			PolicyID:    "PII-001",
			PolicyName:  "Social Security Number Detection",
			Severity:    "CRITICAL",
			Description: "SSN detected in AI output",
			Location:    "Content contains SSN pattern",
			Remediation: "SSN automatically redacted",
		})
		redacted = ssnPattern.ReplaceAllString(redacted, "[SSN REDACTED]")
	}

	// Credit card pattern: ####-####-####-#### or ################
	ccPattern := regexp.MustCompile(`\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`)
	if ccPattern.MatchString(content) {
		violations = append(violations, PolicyViolation{
			PolicyID:    "PII-002",
			PolicyName:  "Credit Card Number Detection",
			Severity:    "CRITICAL",
			Description: "Credit card number detected",
			Remediation: "Credit card number redacted",
		})
		redacted = ccPattern.ReplaceAllString(redacted, "[CARD REDACTED]")
	}

	// Account number pattern (8-17 digits)
	accountPattern := regexp.MustCompile(`\baccount\s*#?\s*:?\s*(\d{8,17})\b`)
	if accountPattern.MatchString(strings.ToLower(content)) {
		violations = append(violations, PolicyViolation{
			PolicyID:    "PII-003",
			PolicyName:  "Account Number Detection",
			Severity:    "HIGH",
			Description: "Account number detected in output",
			Remediation: "Account number redacted",
		})
		redacted = accountPattern.ReplaceAllString(redacted, "account [REDACTED]")
	}

	// Email addresses
	emailPattern := regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)
	if emailPattern.MatchString(content) {
		violations = append(violations, PolicyViolation{
			PolicyID:    "PII-004",
			PolicyName:  "Email Address Detection",
			Severity:    "MEDIUM",
			Description: "Email address detected",
			Remediation: "Email redacted",
		})
		redacted = emailPattern.ReplaceAllString(redacted, "[EMAIL REDACTED]")
	}

	// Phone numbers: (###) ###-#### or ###-###-####
	phonePattern := regexp.MustCompile(`\b(\(\d{3}\)\s*\d{3}-\d{4}|\d{3}-\d{3}-\d{4})\b`)
	if phonePattern.MatchString(content) {
		violations = append(violations, PolicyViolation{
			PolicyID:    "PII-005",
			PolicyName:  "Phone Number Detection",
			Severity:    "MEDIUM",
			Description: "Phone number detected",
			Remediation: "Phone number redacted",
		})
		redacted = phonePattern.ReplaceAllString(redacted, "[PHONE REDACTED]")
	}

	return violations, redacted
}

// detectProhibitedContent checks for prohibited phrases
func (g *GuardrailEngine) detectProhibitedContent(content string) []PolicyViolation {
	violations := []PolicyViolation{}
	lowerContent := strings.ToLower(content)

	prohibitedPhrases := map[string]string{
		"guaranteed return":   "Promises guaranteed returns (SEC violation)",
		"guaranteed profit":   "Promises guaranteed profits",
		"risk-free":           "Claims investment is risk-free",
		"can't lose":          "Claims investment cannot lose money",
		"insider information": "References insider information",
		"hot tip":             "Provides 'hot tips' (unprofessional)",
	}

	for phrase, description := range prohibitedPhrases {
		if strings.Contains(lowerContent, phrase) {
			violations = append(violations, PolicyViolation{
				PolicyID:    "PROHIB-001",
				PolicyName:  "Prohibited Investment Claims",
				Severity:    "CRITICAL",
				Description: description,
				Location:    fmt.Sprintf("Contains phrase: '%s'", phrase),
				Remediation: "Remove prohibited claim from output",
			})
		}
	}

	return violations
}

// checkFinancialAdviceCompliance ensures advice meets regulatory standards
func (g *GuardrailEngine) checkFinancialAdviceCompliance(content string) []PolicyViolation {
	violations := []PolicyViolation{}
	lowerContent := strings.ToLower(content)

	// Check for advice without disclaimers
	adviceIndicators := []string{"you should", "i recommend", "you must", "you need to"}
	hasAdvice := false
	for _, indicator := range adviceIndicators {
		if strings.Contains(lowerContent, indicator) {
			hasAdvice = true
			break
		}
	}

	if hasAdvice {
		// Check if disclaimer present
		disclaimerKeywords := []string{"not financial advice", "consult", "advisor", "professional"}
		hasDisclaimer := false
		for _, keyword := range disclaimerKeywords {
			if strings.Contains(lowerContent, keyword) {
				hasDisclaimer = true
				break
			}
		}

		if !hasDisclaimer {
			violations = append(violations, PolicyViolation{
				PolicyID:    "ADVICE-001",
				PolicyName:  "Financial Advice Disclaimer Required",
				Severity:    "HIGH",
				Description: "Output provides advice without appropriate disclaimer",
				Remediation: "Add disclaimer or rephrase as informational",
			})
		}
	}

	return violations
}

// enforceTopicBoundaries prevents AI from discussing non-financial topics
func (g *GuardrailEngine) enforceTopicBoundaries(content string) []PolicyViolation {
	violations := []PolicyViolation{}
	lowerContent := strings.ToLower(content)

	offTopicKeywords := map[string]string{
		"politics":       "Political discussion detected",
		"religion":       "Religious discussion detected",
		"medical advice": "Medical advice detected",
		"legal advice":   "Legal advice detected (non-financial)",
	}

	for keyword, description := range offTopicKeywords {
		if strings.Contains(lowerContent, keyword) {
			violations = append(violations, PolicyViolation{
				PolicyID:    "TOPIC-001",
				PolicyName:  "Off-Topic Content",
				Severity:    "MEDIUM",
				Description: description,
				Location:    fmt.Sprintf("Contains keyword: '%s'", keyword),
				Remediation: "Redirect to financial topics only",
			})
		}
	}

	return violations
}

// checkRegulatoryKeywords flags potential compliance issues
func (g *GuardrailEngine) checkRegulatoryKeywords(content string) []PolicyViolation {
	violations := []PolicyViolation{}
	lowerContent := strings.ToLower(content)

	// SEC/FINRA red flags
	redFlags := map[string]struct {
		description string
		severity    string
	}{
		"guaranteed return": {
			description: "Claims of guaranteed returns are prohibited by SEC regulations",
			severity:    "HIGH",
		},
		"risk-free": {
			description: "No investment is risk-free; such claims violate FINRA Rule 2210",
			severity:    "HIGH",
		},
		"can't lose": {
			description: "Misleading claim that suggests no possibility of loss",
			severity:    "HIGH",
		},
		"100% safe": {
			description: "No investment is 100% safe; violates fair disclosure requirements",
			severity:    "HIGH",
		},
		"insider information": {
			description: "References to insider information may indicate securities law violations",
			severity:    "CRITICAL",
		},
		"material non-public": {
			description: "MNPI references require immediate compliance review",
			severity:    "CRITICAL",
		},
		"front running": {
			description: "Front running references indicate potential market manipulation",
			severity:    "CRITICAL",
		},
		"pump and dump": {
			description: "Market manipulation scheme reference detected",
			severity:    "CRITICAL",
		},
		"ponzi scheme": {
			description: "Ponzi scheme references indicate potential fraud and require immediate review",
			severity:    "CRITICAL",
		},
		"sure thing": {
			description: "Overly optimistic language violates balanced disclosure requirements",
			severity:    "MEDIUM",
		},
		"double your money": {
			description: "Unrealistic return promises violate advertising regulations",
			severity:    "HIGH",
		},
	}

	for keyword, info := range redFlags {
		if strings.Contains(lowerContent, keyword) {
			violations = append(violations, PolicyViolation{
				PolicyID:    "REG-001",
				PolicyName:  "SEC/FINRA Regulatory Keywords",
				Severity:    info.severity,
				Description: fmt.Sprintf("Regulatory red flag detected: '%s' - %s", keyword, info.description),
				Location:    keyword,
				Remediation: "Remove or rephrase the flagged content to comply with SEC/FINRA regulations",
			})
		}
	}

	return violations
}

// auditGuardrailCheck logs the guardrail execution to audit trail
func (g *GuardrailEngine) auditGuardrailCheck(ctx context.Context, tenantID, userID, content string, result *GuardrailResult) (*AuditEventResult, error) {
	if g.auditService == nil {
		return &AuditEventResult{EventID: ""}, nil
	}

	metadata := map[string]interface{}{
		"violations_count":  len(result.ViolationsDetected),
		"approved":          result.Approved,
		"execution_time_ms": result.ExecutionTimeMs,
		"content_length":    len(content),
		"redacted_pii":      len(result.ViolationsDetected) > 0,
	}

	eventData, _ := json.Marshal(map[string]interface{}{
		"original_content_hash": hashContent(content),
		"result":                result,
	})

	// Build audit record using UnifiedAuditRecord
	auditID := fmt.Sprintf("guardrail_%d", time.Now().UnixNano())
	record := audit.UnifiedAuditRecord{
		AuditID:       auditID,
		EventType:     "ai_guardrail_check",
		Version:       "1.0",
		TenantID:      tenantID,
		ActorID:       userID,
		Timestamp:     time.Now(),
		ObjectType:    "ai_output",
		ObjectID:      hashContent(content),
		PayloadDigest: hashContent(string(eventData)),
		Narrative:     fmt.Sprintf("AI guardrail check: %d violations detected", len(result.ViolationsDetected)),
		Status:        "success",
		Action:        "check",
		Metadata:      metadata,
	}

	if !result.Approved {
		record.Status = "blocked"
	}

	// Log event using the audit service
	err := g.auditService.LogEvent(ctx, record)

	if err != nil {
		return &AuditEventResult{EventID: ""}, err
	}

	return &AuditEventResult{EventID: fmt.Sprintf("guardrail_%d", time.Now().Unix())}, nil
}

// AuditEventResult represents the result of an audit event
type AuditEventResult struct {
	EventID string
}

// PolicyRegistry manages guardrail policies
type PolicyRegistry struct {
	policies map[string]Policy
}

// Policy represents a guardrail policy
type Policy struct {
	ID          string
	Name        string
	Description string
	Version     string
	Active      bool
}

// NewPolicyRegistry creates a new policy registry
func NewPolicyRegistry() *PolicyRegistry {
	return &PolicyRegistry{
		policies: make(map[string]Policy),
	}
}

func hashContent(content string) string {
	// Simple hash for audit purposes
	return fmt.Sprintf("%x", len(content))
}

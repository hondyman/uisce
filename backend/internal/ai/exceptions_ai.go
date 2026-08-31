package ai

import (
	"context"
	"fmt"

	"github.com/hondyman/uisce/backend/internal/audit"
	"github.com/hondyman/uisce/backend/internal/platform_intelligence/exceptions"
)

// ExceptionAIService provides AI assist for platform exceptions
// (root-cause explanation, fix suggestion, closure summary) as thin
// wrappers around the shared audit.AuditNarrativeAIClient seam — no second
// AI client/interface is introduced here.
type ExceptionAIService struct {
	client audit.AuditNarrativeAIClient
}

func NewExceptionAIService(client audit.AuditNarrativeAIClient) *ExceptionAIService {
	return &ExceptionAIService{client: client}
}

// SuggestFix asks the model to explain root cause and propose a fix for an
// open exception. Surfaced via GET /exceptions/{id}/ai-suggestion.
func (s *ExceptionAIService) SuggestFix(ctx context.Context, exc exceptions.Exception) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("exceptions ai: no client configured")
	}
	prompt := fmt.Sprintf(
		"An operational exception was detected on a multi-tenant platform. "+
			"Explain the likely root cause in 2-3 sentences, then suggest a concrete fix.\n\n"+
			"Type: %s\nSeverity: %s\nSource: %s\nDescription: %s\nOccurrences: %d",
		exc.Type, exc.Severity, exc.Source, exc.Description, exc.OccurrenceCount,
	)
	return s.client.GenerateNarrative(ctx, prompt, map[string]interface{}{
		"exception_id": exc.ID,
		"tenant_id":    exc.TenantID,
		"evidence":     exc.Evidence,
		"status":       exc.Status,
	})
}

// SummarizeClosure produces a short closure note once an exception has been
// auto-fixed, to auto-populate the resolution record.
func (s *ExceptionAIService) SummarizeClosure(ctx context.Context, exc exceptions.Exception, autofixAttempts []exceptions.AutofixAttempt) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("exceptions ai: no client configured")
	}
	prompt := fmt.Sprintf(
		"An exception on a multi-tenant platform was auto-remediated and verified fixed. "+
			"Write a 1-2 sentence closure summary suitable for an audit trail.\n\n"+
			"Type: %s\nSource: %s\nDescription: %s\nAttempts: %d",
		exc.Type, exc.Source, exc.Description, len(autofixAttempts),
	)
	return s.client.GenerateNarrative(ctx, prompt, map[string]interface{}{
		"exception_id":     exc.ID,
		"tenant_id":        exc.TenantID,
		"autofix_attempts": autofixAttempts,
	})
}

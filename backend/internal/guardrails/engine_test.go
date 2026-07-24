package guardrails

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPIIDetection_SSN(t *testing.T) {
	engine := NewGuardrailEngine(nil)

	content := "The client's SSN is 123-45-6789 for reference."
	violations, redacted := engine.detectAndRedactPII(content)

	assert.Len(t, violations, 1, "Should detect 1 SSN violation")
	assert.Equal(t, "PII-001", violations[0].PolicyID)
	assert.Equal(t, "CRITICAL", violations[0].Severity)
	assert.Contains(t, redacted, "[SSN REDACTED]")
	assert.NotContains(t, redacted, "123-45-6789")
}

func TestPIIDetection_CreditCard(t *testing.T) {
	engine := NewGuardrailEngine(nil)

	content := "Card number: 4532-1234-5678-9010"
	violations, redacted := engine.detectAndRedactPII(content)

	assert.Len(t, violations, 1)
	assert.Equal(t, "PII-002", violations[0].PolicyID)
	assert.Contains(t, redacted, "[CARD REDACTED]")
}

func TestPIIDetection_Email(t *testing.T) {
	engine := NewGuardrailEngine(nil)

	content := "Contact me at john.doe@example.com for details."
	violations, redacted := engine.detectAndRedactPII(content)

	assert.Len(t, violations, 1, "Should detect email")
	assert.Equal(t, "PII-004", violations[0].PolicyID)
	assert.Contains(t, redacted, "[EMAIL REDACTED]")
}

func TestProhibitedContent_GuaranteedReturns(t *testing.T) {
	engine := NewGuardrailEngine(nil)

	content := "This investment offers guaranteed returns of 15% annually."
	violations := engine.detectProhibitedContent(content)

	assert.Len(t, violations, 1, "Should detect guaranteed returns claim")
	assert.Equal(t, "CRITICAL", violations[0].Severity)
	assert.Contains(t, violations[0].Description, "guaranteed returns")
}

func TestProhibitedContent_RiskFree(t *testing.T) {
	engine := NewGuardrailEngine(nil)

	content := "This is a risk-free opportunity you can't lose on."
	violations := engine.detectProhibitedContent(content)

	assert.GreaterOrEqual(t, len(violations), 2, "Should detect multiple violations")

	// Check for both "risk-free" and "can't lose"
	hasRiskFree := false
	hasCantLose := false
	for _, v := range violations {
		if v.Description == "Claims investment is risk-free" {
			hasRiskFree = true
		}
		if v.Description == "Claims investment cannot lose money" {
			hasCantLose = true
		}
	}
	assert.True(t, hasRiskFree, "Should detect risk-free claim")
	assert.True(t, hasCantLose, "Should detect can't lose claim")
}

func TestFinancialAdviceCompliance_MissingDisclaimer(t *testing.T) {
	engine := NewGuardrailEngine(nil)

	content := "You should invest 60% in stocks and 40% in bonds."
	violations := engine.checkFinancialAdviceCompliance(content)

	assert.Len(t, violations, 1, "Should flag advice without disclaimer")
	assert.Equal(t, "ADVICE-001", violations[0].PolicyID)
	assert.Equal(t, "HIGH", violations[0].Severity)
}

func TestFinancialAdviceCompliance_WithDisclaimer(t *testing.T) {
	engine := NewGuardrailEngine(nil)

	content := "You should consult a financial advisor. This is not financial advice."
	violations := engine.checkFinancialAdviceCompliance(content)

	assert.Len(t, violations, 0, "Should pass with disclaimer")
}

func TestTopicBoundaries_Politics(t *testing.T) {
	engine := NewGuardrailEngine(nil)

	content := "Let's discuss politics and the upcoming election."
	violations := engine.enforceTopicBoundaries(content)

	assert.Len(t, violations, 1, "Should detect off-topic (politics)")
	assert.Equal(t, "TOPIC-001", violations[0].PolicyID)
}

func TestRegulatoryKeywords_PonziScheme(t *testing.T) {
	engine := NewGuardrailEngine(nil)

	content := "This looks like a ponzi scheme to me."
	violations := engine.checkRegulatoryKeywords(content)

	assert.Len(t, violations, 1, "Should detect ponzi reference")
	assert.Equal(t, "CRITICAL", violations[0].Severity)
	assert.Equal(t, "REG-001", violations[0].PolicyID)
}

func TestFilterAIOutput_CleanContent(t *testing.T) {
	engine := NewGuardrailEngine(nil)

	content := "Here is information about portfolio diversification strategies."
	result, err := engine.FilterAIOutput(context.Background(), content, "tenant-123", "user-456")

	assert.NoError(t, err)
	assert.True(t, result.Approved, "Clean content should be approved")
	assert.Len(t, result.ViolationsDetected, 0)
	assert.Equal(t, content, result.RedactedContent, "Content should not be modified")
}

func TestFilterAIOutput_MultipleViolations(t *testing.T) {
	engine := NewGuardrailEngine(nil)

	content := "Contact me at test@example.com. SSN: 123-45-6789. This investment guarantees 20% returns risk-free!"
	result, err := engine.FilterAIOutput(context.Background(), content, "tenant-123", "user-456")

	assert.NoError(t, err)
	assert.False(t, result.Approved, "Should not approve content with critical violations")
	assert.Greater(t, len(result.ViolationsDetected), 2, "Should detect multiple violations")

	// Check PII was redacted
	assert.Contains(t, result.RedactedContent, "[EMAIL REDACTED]")
	assert.Contains(t, result.RedactedContent, "[SSN REDACTED]")
	assert.NotContains(t, result.RedactedContent, "123-45-6789")
	assert.NotContains(t, result.RedactedContent, "test@example.com")
}

func TestFilterAIOutput_PerformanceBenchmark(t *testing.T) {
	engine := NewGuardrailEngine(nil)

	content := "This is a standard financial advisory message about portfolio rebalancing with no violations."
	result, err := engine.FilterAIOutput(context.Background(), content, "tenant-123", "user-456")

	assert.NoError(t, err)
	assert.Less(t, result.ExecutionTimeMs, int64(50), "Guardrail check should complete in <50ms")
}

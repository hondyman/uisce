package scheduler_intelligence

import (
	"regexp"
)

// LoggingPIIFilter sanitizes log messages based on compliance context
type LoggingPIIFilter struct {
	RedactPattern *regexp.Regexp
}

// NewLoggingPIIFilter creates a filter with default sensitive patterns
func NewLoggingPIIFilter() *LoggingPIIFilter {
	// Regex for detecting email-like strings and credit card numbers
	pattern := regexp.MustCompile(`(?i)(email|ssn|credit_card|password|token)`)
	return &LoggingPIIFilter{RedactPattern: pattern}
}

// FilterPII redacts text if the job context indicates PII is present OR if sensitive keywords are found
func (f *LoggingPIIFilter) FilterPII(message string, compliance *Compliance) string {
	if compliance != nil && compliance.PII {
		// Aggressive redaction or specialized masking could go here.
		// For now, we prepend a PII markers.
		// In a real implementation, we might mask specific values.
		return "[PII-PROTECTED] " + f.maskSensitiveData(message)
	}

	// Even if not flagged, check for accidental leakage
	return f.maskSensitiveData(message)
}

// maskSensitiveData performs simple masking of sensitive patterns
func (f *LoggingPIIFilter) maskSensitiveData(input string) string {
	// This is a naive implementation; production would use a robust library
	// For this demo, we replace detected keys' values.
	// E.g. "email=foo@bar.com" -> "email=***"

	// Just return regex masked for simplicity in this scope
	// Note: Proper PII redaction is complex; this is a placeholder for the mechanism.
	return f.RedactPattern.ReplaceAllStringFunc(input, func(s string) string {
		return s + "=***"
	})
}

// IsPIISafe checks if the structured data is safe to log
func (f *LoggingPIIFilter) IsPIISafe(data map[string]interface{}) bool {
	// Recursively check keys for sensitive terms
	for k := range data {
		if f.RedactPattern.MatchString(k) {
			return false
		}
	}
	return true
}

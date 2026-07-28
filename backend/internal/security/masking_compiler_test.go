package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyDynamicMasking_PartialMask(t *testing.T) {
	policies := []MaskingPolicy{
		{TargetField: "ssn", MaskType: "PARTIAL_MASK", RoleExempt: "COMPLIANCE_OFFICER"},
	}

	// Non-exempt user receives masked expression
	masked := ApplyDynamicMasking("account.ssn", "ssn", []string{"ANALYST"}, policies)
	assert.Contains(t, masked, "REGEXP_REPLACE")

	// Exempt user receives unmasked field expression
	unmasked := ApplyDynamicMasking("account.ssn", "ssn", []string{"COMPLIANCE_OFFICER"}, policies)
	assert.Equal(t, "account.ssn", unmasked)
}

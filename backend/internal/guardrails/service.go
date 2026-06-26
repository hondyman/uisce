package guardrails

import (
	"regexp"
	"strings"
)

var (
	reSSN        = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	reAcctNum    = regexp.MustCompile(`\b\d{8,12}\b`)
	reRouting    = regexp.MustCompile(`\b\d{9}\b`)
	banGuarantee = regexp.MustCompile(`(?i)\b(guaranteed returns?|risk[-\s]?free|no risk|cannot lose)\b`)
	banPromNote  = regexp.MustCompile(`(?i)\b(promissory note|private placement without disclosure)\b`)
)

type Outcome struct {
	Allowed       bool     `json:"allowed"`
	RequiresHuman bool     `json:"requires_human"`
	Violations    []string `json:"violations"`
	PolicyVersion string   `json:"policy_version"`
}

// Evaluate checks the text against deterministic guardrails
func Evaluate(text string, allowedTopics []string) Outcome {
	v := []string{}

	// PII
	if reSSN.FindString(text) != "" {
		v = append(v, "pii_ssn")
	}
	if reAcctNum.FindString(text) != "" {
		v = append(v, "pii_account_number")
	}
	if reRouting.FindString(text) != "" {
		v = append(v, "pii_routing_number")
	}

	// Advice restrictions
	if banGuarantee.FindString(text) != "" {
		v = append(v, "advice_guarantee_claim")
	}
	if banPromNote.FindString(text) != "" {
		v = append(v, "advice_promissory_note")
	}

	// Topic adherence (simple deterministic check)
	if !isWithinTopics(text, allowedTopics) {
		v = append(v, "topic_out_of_scope")
	}

	allowed := len(v) == 0
	requiresHuman := !allowed // Simple policy: anything flagged requires human approval
	return Outcome{Allowed: allowed, RequiresHuman: requiresHuman, Violations: v, PolicyVersion: "v1.0.0"}
}

func isWithinTopics(text string, topics []string) bool {
	if len(topics) == 0 {
		return true // No topic restrictions
	}
	t := strings.ToLower(text)
	for _, topic := range topics {
		if strings.Contains(t, strings.ToLower(topic)) {
			return true
		}
	}
	return false
}

package models

// ViolationRecord represents a single evaluated rule violation or rule execution error.
type ViolationRecord struct {
	TenantID     string                 `json:"tenant_id,omitempty"`
	RuleID       string                 `json:"rule_id"`
	RuleKey      string                 `json:"rule_key"`
	RuleVersion  string                 `json:"rule_version"`
	RuleName     string                 `json:"rule_name"`
	BOKey        string                 `json:"bo_key"`
	Severity     string                 `json:"severity"` // "BLOCK" | "WARN"
	RecordID     string                 `json:"record_id,omitempty"`
	Message      string                 `json:"message"`
	Fields       []string               `json:"fields,omitempty"`
	Context      map[string]interface{} `json:"context,omitempty"`
	WriteBlocked bool                   `json:"write_blocked"`
	RuleError    bool                   `json:"rule_error"`
}

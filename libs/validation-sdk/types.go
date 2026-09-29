package validationsdk

// Wire types mirror the /api/validation-rule-nodes/evaluate-* contract.
// Kept deliberately independent of backend/internal (Go internal-package
// visibility) so the SDK is consumable by external services too.

type EvaluateRecordRequest struct {
	BOName string         `json:"bo_name"`
	Domain string         `json:"domain,omitempty"`
	Timing string         `json:"timing,omitempty"`
	Record map[string]any `json:"record"`
}

type Violation struct {
	RuleID       string   `json:"rule_id"`
	RuleKey      string   `json:"rule_key"`
	RuleVersion  string   `json:"rule_version"`
	RuleName     string   `json:"rule_name"`
	BOKey        string   `json:"bo_key"`
	Severity     string   `json:"severity"` // "BLOCK" | "WARN"
	RecordID     string   `json:"record_id,omitempty"`
	Message      string   `json:"message"`
	Fields       []string `json:"fields,omitempty"`
	WriteBlocked bool     `json:"write_blocked"`
	RuleError    bool     `json:"rule_error"`
}

type RuleError struct {
	RuleID   string `json:"rule_id"`
	RuleKey  string `json:"rule_key"`
	RuleName string `json:"rule_name"`
	Message  string `json:"message"`
}

type EvaluateRecordResponse struct {
	Valid               bool        `json:"valid"`
	Blocked             bool        `json:"blocked"`
	EvaluatedRulesCount int         `json:"evaluated_rules_count"`
	SnapshotID          string      `json:"snapshot_id,omitempty"`
	Violations          []Violation `json:"violations"`
	RuleErrors          []RuleError `json:"rule_errors"`
}

// ServerContextRequiredError is returned as a typed error when rules need
// server-side context the payload didn't supply (HTTP 200 + error body).
type ServerContextRequiredError struct {
	MissingFields []string
	Hint          string
}

func (e *ServerContextRequiredError) Error() string {
	return "ERR_SERVER_CONTEXT_REQUIRED: missing context fields: " + join(e.MissingFields)
}

type EvaluateBatchRequest struct {
	BOName  string           `json:"bo_name"`
	Domain  string           `json:"domain,omitempty"`
	Timing  string           `json:"timing,omitempty"`
	Records []map[string]any `json:"records"`
}

type BatchSummary struct {
	SnapshotID         string         `json:"snapshot_id"`
	TotalRecords       int            `json:"total_records"`
	ValidCount         int            `json:"valid_count"`
	InvalidCount       int            `json:"invalid_count"`
	RuleErrorCount     int            `json:"rule_error_count"`
	ErrorCount         int            `json:"error_count"`
	ViolationHistogram map[string]int `json:"violation_histogram"`
	SeverityHistogram  map[string]int `json:"severity_histogram"`
	DurationMs         int64          `json:"duration_ms"`
}

type RecordResult struct {
	Valid               bool        `json:"valid"`
	Blocked             bool        `json:"blocked"`
	EvaluatedRulesCount int         `json:"evaluated_rules_count"`
	SnapshotID          string      `json:"snapshot_id,omitempty"`
	Violations          []Violation `json:"violations"`
	RuleErrors          []RuleError `json:"rule_errors"`
}

type BatchRecordError struct {
	Index   int    `json:"index"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type EvaluateBatchResponse struct {
	Summary BatchSummary       `json:"summary"`
	Records []RecordResult     `json:"records"`
	Errors  []BatchRecordError `json:"errors"`
}

func join(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}

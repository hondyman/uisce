package rules

type Severity string

const (
	SeverityInfo       Severity = "info"
	SeverityWarning    Severity = "warning"
	SeverityError      Severity = "error"
	SeverityHardBlock  Severity = "hard_block"
	SeverityQuarantine Severity = "quarantine"
)

type RuleAction struct {
	Type   string         `json:"type"`
	Params map[string]any `json:"params,omitempty"`
}

type RuleResult struct {
	Passed         bool          `json:"passed"`
	Score          *float64      `json:"score,omitempty"`
	Severity       Severity      `json:"severity"`
	Actions        []RuleAction  `json:"actions,omitempty"`
	RuleID         string        `json:"rule_id"`
	RuleName       string        `json:"rule_name,omitempty"`
	Category       string        `json:"category,omitempty"`
	Details        []string      `json:"details,omitempty"`
	FailureReasons []string      `json:"failure_reasons,omitempty"`
	Violations     []RuleViolation `json:"violations,omitempty"`
	EvalTimeNs     int64         `json:"eval_time_ns"`
}

type RuleWithMetadata struct {
	Node           *RuleNode
	ID             string
	Version        int
	Name           string
	Severity       Severity
	Category       string
	Actions        []RuleAction
	ScoringFormula string
}

type BatchResult struct {
	Results     []*RuleResult `json:"results"`
	TotalTimeNs int64         `json:"total_time_ns"`
	PassedAll   bool          `json:"passed_all"`
}

type RuleChain struct {
	ID          string
	Name        string
	Operator    string
	Rules       []*RuleWithMetadata
	StopOnFirst Severity
}

type RuleViolation struct {
	ConditionID    string  `json:"condition_id,omitempty"`
	FieldPath      string  `json:"field_path"`
	Operator       string  `json:"operator"`
	EvaluatedVal   any     `json:"evaluated_val"`
	ThresholdLimit any     `json:"threshold_limit"`
	Message        string  `json:"message"`
	RuleID         string  `json:"rule_id,omitempty"`
	RuleName       string  `json:"rule_name,omitempty"`
	Severity       Severity `json:"severity,omitempty"`
}

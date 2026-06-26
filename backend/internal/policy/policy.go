package policy

// Policy defines a set of rules for a given scope.
type Policy struct {
	ID          string `yaml:"id,omitempty"`
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Scope       string `yaml:"scope"`
	Rules       []Rule `yaml:"rules"`
}

// Rule defines a single policy rule.
type Rule struct {
	ID          string   `yaml:"id"`
	Description string   `yaml:"description,omitempty"`
	Severity    string   `yaml:"severity"`
	Selectors   []string `yaml:"selectors"`
	Conditions  []string `yaml:"conditions"`
}

// MatchDetail provides a detailed explanation of a policy match.
type MatchDetail struct {
	Selector string      `json:"selector"`
	Path     string      `json:"path"`
	Value    interface{} `json:"value"`
}

// Violation represents a policy violation.
type Violation struct {
	RuleID   string
	Severity string
	Message  string
	Explain  []MatchDetail
}

// Evaluator evaluates policies against changes.
type Evaluator struct {
	policy *Policy
}

// NewEvaluator creates a new policy evaluator.
func NewEvaluator(p *Policy) *Evaluator {
	return &Evaluator{policy: p}
}

// ExplainRule explains how a rule was evaluated against a change.
func (e *Evaluator) ExplainRule(change interface{}, ruleID string) ([]MatchDetail, error) {
	// This is a placeholder implementation.
	// A real implementation would evaluate the rule and return the details.
	return []MatchDetail{
		{
			Selector: "placeholder",
			Path:     "placeholder",
			Value:    "placeholder",
		},
	}, nil
}

package types

// PreaggregationPlan represents the JSON structure for preaggregation plans
type PreaggregationPlan struct {
	BundleID            string              `json:"bundle_id"`
	Domain              string              `json:"domain"`
	Description         string              `json:"description"`
	Version             string              `json:"version"`
	Metrics             []MetricPreagg      `json:"metrics"`
	ImplementationNotes ImplementationNotes `json:"implementation_notes"`
}

type MetricPreagg struct {
	NodeID           string   `json:"node_id"`
	Preaggregate     bool     `json:"preaggregate"`
	SuggestedGrain   []string `json:"suggested_grain"`
	RefreshSchedule  string   `json:"refresh_schedule"`
	Rationale        string   `json:"rationale"`
	EstimatedSavings string   `json:"estimated_savings"`
}

type ImplementationNotes struct {
	HighPriority              []string `json:"high_priority"`
	DataQualityChecks         []string `json:"data_quality_checks"`
	GovernanceConsiderations  []string `json:"governance_considerations"`
	PerformanceConsiderations []string `json:"performance_considerations"`
}

// GenericBundle represents the JSON structure for any bundle
type GenericBundle struct {
	BundleID  string     `json:"bundle_id"`
	Domain    string     `json:"domain"`
	Audience  []string   `json:"audience"`
	Version   string     `json:"version"`
	Owner     string     `json:"owner"`
	Tags      []string   `json:"tags"`
	Functions []Function `json:"functions,omitempty"`
	Metrics   []Metric   `json:"metrics"`
}

type Function struct {
	Name        string `json:"name"`
	Class       string `json:"class"`
	Badge       string `json:"badge"`
	Description string `json:"description"`
}

type Metric struct {
	NodeID        string        `json:"node_id"`
	Category      string        `json:"category"`
	Description   string        `json:"description"`
	FinancialCalc FinancialCalc `json:"financial_calc"`
	Badge         string        `json:"badge,omitempty"`
	FunctionClass string        `json:"function_class,omitempty"`
	FunctionsUsed []string      `json:"functions_used,omitempty"`
	Governance    Governance    `json:"governance"`
}

type FinancialCalc struct {
	Type      string                 `json:"type"`
	Formula   string                 `json:"formula"`
	Arguments map[string]interface{} `json:"arguments"`
}

type Governance struct {
	Status string `json:"status"`
}

package runtime

// RuleNode represents a node in the ASL rule AST
type RuleNode struct {
	Type      string     `json:"Type"`
	Condition *Condition `json:"Condition,omitempty"`
	Group     *Group     `json:"Group,omitempty"`
}

// Condition represents a rule condition
type Condition struct {
	Field    string      `json:"Field"`
	Operator string      `json:"Operator"`
	Value    interface{} `json:"Value"`
}

// Group represents a logical group of rules
type Group struct {
	Operator string     `json:"Operator"` // "AND" | "OR"
	Children []RuleNode `json:"Children"`
}

package business_process

// Condition represents a single condition in advanced logic
type Condition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // ==, !=, >, <, >=, <=, in, contains, startsWith, endsWith
	Value    interface{} `json:"value"`
}

// AdvancedCondition represents advanced conditional branching with boolean operators
type AdvancedCondition struct {
	Operator    string      `json:"operator"` // AND, OR, NOT
	Conditions  []Condition `json:"conditions"`
	TrueBranch  []string    `json:"trueBranch"`  // Step IDs to execute if true
	FalseBranch []string    `json:"falseBranch"` // Step IDs to execute if false
}

// ApprovalChain represents dynamic approval chain configuration
type ApprovalChain struct {
	Type           string   `json:"type"`                     // role, org_hierarchy, custom, multi_role
	Levels         *int     `json:"levels,omitempty"`         // For org_hierarchy
	Roles          []string `json:"roles,omitempty"`          // For multi_role
	ApprovalMode   string   `json:"approvalMode"`             // all, any, majority
	EscalationPath []string `json:"escalationPath,omitempty"` // Fallback roles
}

// NotificationRecipient represents a notification recipient configuration
type NotificationRecipient struct {
	Type  string `json:"type"`  // role, user, dynamic
	Value string `json:"value"` // role name, user ID, or expression
}

// NotificationConfig represents enhanced notification configuration
type NotificationConfig struct {
	TemplateID  string                  `json:"templateId"`
	Channels    []string                `json:"channels"` // email, in_app, sms, slack
	Recipients  []NotificationRecipient `json:"recipients"`
	MergeFields map[string]string       `json:"mergeFields,omitempty"`
}

// EnhancedStep extends Step with advanced features
type EnhancedStep struct {
	Step // Embed base Step

	// Advanced conditional logic
	ConditionLogic *AdvancedCondition `json:"conditionLogic,omitempty"`

	// Parallel execution support
	ExecutionMode string `json:"executionMode"`           // sequential, parallel
	ParallelGroup string `json:"parallelGroup,omitempty"` // Steps with same group execute in parallel
	WaitForAll    bool   `json:"waitForAll"`              // true = all must complete, false = any

	// Approval chain configuration
	ApprovalChain *ApprovalChain `json:"approvalChain,omitempty"`

	// Step dependencies
	DependsOn     []string           `json:"dependsOn,omitempty"`     // Step IDs that must complete first
	SkipCondition *AdvancedCondition `json:"skipCondition,omitempty"` // Skip if condition is true

	// Enhanced notifications
	NotificationConfig *NotificationConfig `json:"notificationConfig,omitempty"`
}

// EnhancedProcessTemplate extends ProcessTemplate with advanced features
type EnhancedProcessTemplate struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Version     int            `json:"version"`
	Steps       []EnhancedStep `json:"steps"`
}

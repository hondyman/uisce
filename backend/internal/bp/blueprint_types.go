package bp

// WorkflowBlueprint represents the compiled version of the designer graph
type WorkflowBlueprint struct {
	Nodes   map[string]*CompiledNode
	StartID string
	EndID   string
}

// CompiledNode is a runtime-optimized representation of a BPStep
type CompiledNode struct {
	StepKey       string
	Type          string // "task", "approval", "branch", "delay", "signal"
	ActivityName  string
	SignalName    string
	ConditionExpr string
	DelayExpr     string
	SLAExpr       string
	ApprovalChain interface{} // Raw JSON map/struct for the activity to parse
	RoutingRules  interface{} // Raw JSON map/struct for the activity to parse
	Escalations   []EscalationStep
	NextNodes     []string
}

type EscalationStep struct {
	ID                     string      `json:"id"`
	StepNumber             int         `json:"stepNumber"`
	DelayAfterPreviousExpr string      `json:"delayAfterPreviousExpr"`
	TargetActorRole        string      `json:"targetActorRole"`
	NotificationTemplate   string      `json:"notificationTemplate"`
	Condition              interface{} `json:"condition"`
}

// BPExecutionContext holds the runtime state for the workflow
type BPExecutionContext struct {
	Blueprint     *WorkflowBlueprint
	CurrentNodeID string
	Values        map[string]interface{}            // step outputs by stepKey
	BOCtx         map[string]map[string]interface{} // Busines Object Context (e.g. "client": {...})
}

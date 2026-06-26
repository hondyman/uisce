package feed

// FeedItem represents a single card in the user's feed
type FeedItem struct {
	CardID           string                 `json:"card_id"`
	Title            string                 `json:"title"`
	Content          string                 `json:"content"`
	Type             string                 `json:"type"` // "action", "insight", "news"
	Score            float64                `json:"score"`
	ActionWorkflowID string                 `json:"action_workflow_id,omitempty"`
	ActionLabel      string                 `json:"action_label,omitempty"`
	Data             map[string]interface{} `json:"data,omitempty"`
}

// CardTemplate defines the rules and content for a type of card
type CardTemplate struct {
	ID              string
	Type            string
	PriorityBase    float64
	TriggerRules    []string
	ContentTemplate string
	ActionWorkflow  string
}

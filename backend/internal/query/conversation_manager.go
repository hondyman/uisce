package query

import (
	"fmt"
	"strings"
	"time"
)

// ConversationContext represents the context of a multi-turn conversation
type ConversationContext struct {
	ConversationID string                 `json:"conversation_id"`
	UserID         string                 `json:"user_id"`
	TenantID       string                 `json:"tenant_id"`
	Datasource     string                 `json:"datasource"`
	QueryHistory   []*ConversationQuery   `json:"query_history"`
	ContextData    map[string]interface{} `json:"context_data"`
	LastActivity   time.Time              `json:"last_activity"`
	CreatedAt      time.Time              `json:"created_at"`
}

// ConversationQuery represents a single query in the conversation
type ConversationQuery struct {
	QueryID      string        `json:"query_id"`
	UserQuery    string        `json:"user_query"`
	ParsedIntent *ParsedIntent `json:"parsed_intent"`
	GeneratedSQL string        `json:"generated_sql"`
	ExecutedAt   time.Time     `json:"executed_at"`
	Success      bool          `json:"success"`
	ContextRefs  []string      `json:"context_refs"` // References to previous queries
}

// ConversationManager manages multi-turn conversations
type ConversationManager struct {
	conversations map[string]*ConversationContext
	maxHistory    int
	contextTTL    time.Duration
}

// NewConversationManager creates a new conversation manager
func NewConversationManager() *ConversationManager {
	return &ConversationManager{
		conversations: make(map[string]*ConversationContext),
		maxHistory:    10,               // Keep last 10 queries
		contextTTL:    30 * time.Minute, // 30 minutes
	}
}

// StartConversation starts a new conversation
func (cm *ConversationManager) StartConversation(userID, tenantID, datasource string) *ConversationContext {
	conversationID := generateConversationID()

	context := &ConversationContext{
		ConversationID: conversationID,
		UserID:         userID,
		TenantID:       tenantID,
		Datasource:     datasource,
		QueryHistory:   []*ConversationQuery{},
		ContextData:    make(map[string]interface{}),
		LastActivity:   time.Now(),
		CreatedAt:      time.Now(),
	}

	cm.conversations[conversationID] = context
	return context
}

// GetConversation retrieves an existing conversation
func (cm *ConversationManager) GetConversation(conversationID string) (*ConversationContext, error) {
	context, exists := cm.conversations[conversationID]
	if !exists {
		return nil, fmt.Errorf("conversation not found: %s", conversationID)
	}

	// Check if conversation has expired
	if time.Since(context.LastActivity) > cm.contextTTL {
		delete(cm.conversations, conversationID)
		return nil, fmt.Errorf("conversation expired: %s", conversationID)
	}

	return context, nil
}

// AddQueryToConversation adds a new query to the conversation
func (cm *ConversationManager) AddQueryToConversation(conversationID string, userQuery string, intent *ParsedIntent, generatedSQL string, success bool) error {
	context, err := cm.GetConversation(conversationID)
	if err != nil {
		return err
	}

	query := &ConversationQuery{
		QueryID:      generateQueryID(),
		UserQuery:    userQuery,
		ParsedIntent: intent,
		GeneratedSQL: generatedSQL,
		ExecutedAt:   time.Now(),
		Success:      success,
		ContextRefs:  cm.extractContextReferences(userQuery, context),
	}

	context.QueryHistory = append(context.QueryHistory, query)
	context.LastActivity = time.Now()

	// Keep only the most recent queries
	if len(context.QueryHistory) > cm.maxHistory {
		context.QueryHistory = context.QueryHistory[len(context.QueryHistory)-cm.maxHistory:]
	}

	// Update context data based on this query
	cm.updateContextData(context, query)

	return nil
}

// extractContextReferences identifies references to previous queries
func (cm *ConversationManager) extractContextReferences(userQuery string, context *ConversationContext) []string {
	var refs []string
	queryLower := strings.ToLower(userQuery)

	// Common reference words
	refWords := []string{"that", "this", "those", "these", "it", "them", "same", "previous", "last"}

	for _, word := range refWords {
		if strings.Contains(queryLower, word) && len(context.QueryHistory) > 0 {
			// Reference the most recent successful query
			for i := len(context.QueryHistory) - 1; i >= 0; i-- {
				if context.QueryHistory[i].Success {
					refs = append(refs, context.QueryHistory[i].QueryID)
					break
				}
			}
			break
		}
	}

	return refs
}

// updateContextData updates the conversation context based on the new query
func (cm *ConversationManager) updateContextData(context *ConversationContext, query *ConversationQuery) {
	if query.ParsedIntent == nil {
		return
	}

	// Store current metrics and dimensions for potential reuse
	if len(query.ParsedIntent.Metrics) > 0 {
		context.ContextData["last_metrics"] = query.ParsedIntent.Metrics
	}

	if len(query.ParsedIntent.Dimensions) > 0 {
		context.ContextData["last_dimensions"] = query.ParsedIntent.Dimensions
	}

	if query.ParsedIntent.TimeRange != nil {
		context.ContextData["last_time_range"] = query.ParsedIntent.TimeRange
	}

	// Store successful query patterns
	if query.Success {
		context.ContextData["successful_queries"] = append(
			cm.getStringSlice(context.ContextData, "successful_queries"),
			query.UserQuery,
		)
	}
}

// EnhanceIntentWithContext enhances intent parsing using conversation context
func (cm *ConversationManager) EnhanceIntentWithContext(conversationID string, intent *ParsedIntent) *ParsedIntent {
	context, err := cm.GetConversation(conversationID)
	if err != nil {
		return intent
	}

	enhanced := &ParsedIntent{
		Metrics:     make([]string, len(intent.Metrics)),
		Dimensions:  make([]string, len(intent.Dimensions)),
		Filters:     make([]IntentFilter, len(intent.Filters)),
		TimeRange:   intent.TimeRange,
		Aggregation: intent.Aggregation,
		Confidence:  intent.Confidence,
		RawEntities: make(map[string]string),
	}

	// Copy original data
	copy(enhanced.Metrics, intent.Metrics)
	copy(enhanced.Dimensions, intent.Dimensions)
	copy(enhanced.Filters, intent.Filters)
	for k, v := range intent.RawEntities {
		enhanced.RawEntities[k] = v
	}

	// Enhance with context
	cm.applyContextEnhancements(context, enhanced)

	return enhanced
}

// applyContextEnhancements applies context-based enhancements to intent
func (cm *ConversationManager) applyContextEnhancements(context *ConversationContext, intent *ParsedIntent) {
	// If no metrics specified but we have context, use last metrics
	if len(intent.Metrics) == 0 {
		if lastMetrics, ok := context.ContextData["last_metrics"].([]string); ok && len(lastMetrics) > 0 {
			intent.Metrics = lastMetrics
			intent.Confidence *= 0.9 // Slightly reduce confidence for inferred data
		}
	}

	// If no dimensions specified but we have context, use last dimensions
	if len(intent.Dimensions) == 0 {
		if lastDims, ok := context.ContextData["last_dimensions"].([]string); ok && len(lastDims) > 0 {
			intent.Dimensions = lastDims
			intent.Confidence *= 0.9
		}
	}

	// If no time range specified but we have context, use last time range
	if intent.TimeRange == nil {
		if lastTR, ok := context.ContextData["last_time_range"].(*TimeRange); ok {
			intent.TimeRange = lastTR
			intent.Confidence *= 0.95
		}
	}

	// Add context-aware filters based on conversation history
	contextFilters := cm.generateContextFilters(context, intent)
	intent.Filters = append(intent.Filters, contextFilters...)
}

// generateContextFilters generates filters based on conversation context
func (cm *ConversationManager) generateContextFilters(context *ConversationContext, _ *ParsedIntent) []IntentFilter {
	var filters []IntentFilter

	// If this is a follow-up query, maintain consistency with previous filters
	if len(context.QueryHistory) > 0 {
		lastQuery := context.QueryHistory[len(context.QueryHistory)-1]

		// Copy important filters from the last successful query
		if lastQuery.Success && lastQuery.ParsedIntent != nil {
			for _, filter := range lastQuery.ParsedIntent.Filters {
				// Only copy filters that are likely still relevant
				if cm.isPersistentFilter(filter) {
					filters = append(filters, filter)
				}
			}
		}
	}

	return filters
}

// isPersistentFilter determines if a filter should persist across queries
func (cm *ConversationManager) isPersistentFilter(filter IntentFilter) bool {
	// Persistent filters are typically related to security, tenant isolation, etc.
	persistentFields := []string{
		"tenant_id", "user_id", "organization_id", "region", "department",
		"status", "type", "category",
	}

	fieldLower := strings.ToLower(filter.Field)
	for _, persistent := range persistentFields {
		if strings.Contains(fieldLower, persistent) {
			return true
		}
	}

	return false
}

// GetConversationSummary returns a summary of the conversation
func (cm *ConversationManager) GetConversationSummary(conversationID string) (map[string]interface{}, error) {
	context, err := cm.GetConversation(conversationID)
	if err != nil {
		return nil, err
	}

	summary := map[string]interface{}{
		"conversation_id": conversationID,
		"user_id":         context.UserID,
		"tenant_id":       context.TenantID,
		"datasource":      context.Datasource,
		"query_count":     len(context.QueryHistory),
		"last_activity":   context.LastActivity,
		"created_at":      context.CreatedAt,
		"duration":        context.LastActivity.Sub(context.CreatedAt),
	}

	// Add query history summary
	var querySummaries []map[string]interface{}
	for _, query := range context.QueryHistory {
		querySummary := map[string]interface{}{
			"query_id":    query.QueryID,
			"user_query":  query.UserQuery,
			"executed_at": query.ExecutedAt,
			"success":     query.Success,
		}
		querySummaries = append(querySummaries, querySummary)
	}
	summary["query_history"] = querySummaries

	// Add context insights
	insights := cm.generateConversationInsights(context)
	summary["insights"] = insights

	return summary, nil
}

// generateConversationInsights generates insights about the conversation
func (cm *ConversationManager) generateConversationInsights(context *ConversationContext) map[string]interface{} {
	insights := map[string]interface{}{
		"total_queries":      len(context.QueryHistory),
		"successful_queries": 0,
		"failed_queries":     0,
		"avg_confidence":     0.0,
		"common_metrics":     []string{},
		"common_dimensions":  []string{},
	}

	metricCount := make(map[string]int)
	dimensionCount := make(map[string]int)
	totalConfidence := 0.0

	for _, query := range context.QueryHistory {
		if query.Success {
			insights["successful_queries"] = insights["successful_queries"].(int) + 1
		} else {
			insights["failed_queries"] = insights["failed_queries"].(int) + 1
		}

		if query.ParsedIntent != nil {
			totalConfidence += query.ParsedIntent.Confidence

			for _, metric := range query.ParsedIntent.Metrics {
				metricCount[metric]++
			}

			for _, dimension := range query.ParsedIntent.Dimensions {
				dimensionCount[dimension]++
			}
		}
	}

	if len(context.QueryHistory) > 0 {
		insights["avg_confidence"] = totalConfidence / float64(len(context.QueryHistory))
	}

	// Find most common metrics and dimensions
	insights["common_metrics"] = cm.getTopItems(metricCount, 3)
	insights["common_dimensions"] = cm.getTopItems(dimensionCount, 3)

	return insights
}

// getTopItems returns the top N items from a count map
func (cm *ConversationManager) getTopItems(countMap map[string]int, n int) []string {
	type itemCount struct {
		item  string
		count int
	}

	var items []itemCount
	for item, count := range countMap {
		items = append(items, itemCount{item, count})
	}

	// Sort by count descending
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].count < items[j].count {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	var result []string
	for i := 0; i < len(items) && i < n; i++ {
		result = append(result, items[i].item)
	}

	return result
}

// CleanupExpiredConversations removes expired conversations
func (cm *ConversationManager) CleanupExpiredConversations() int {
	expired := 0
	for id, context := range cm.conversations {
		if time.Since(context.LastActivity) > cm.contextTTL {
			delete(cm.conversations, id)
			expired++
		}
	}
	return expired
}

// getStringSlice safely gets a string slice from context data
func (cm *ConversationManager) getStringSlice(data map[string]interface{}, key string) []string {
	if value, ok := data[key]; ok {
		if slice, ok := value.([]string); ok {
			return slice
		}
	}
	return []string{}
}

// generateConversationID generates a unique conversation ID
func generateConversationID() string {
	return fmt.Sprintf("conv_%d", time.Now().UnixNano())
}

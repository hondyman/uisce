package query

import (
	"math"
	"testing"
	"time"
)

func TestConversationManager_StartConversation(t *testing.T) {
	cm := NewConversationManager()

	context := cm.StartConversation("user1", "tenant1", "datasource1")

	if context == nil {
		t.Fatal("Expected conversation context, got nil")
	}

	if context.UserID != "user1" {
		t.Errorf("Expected UserID 'user1', got '%s'", context.UserID)
	}

	if context.TenantID != "tenant1" {
		t.Errorf("Expected TenantID 'tenant1', got '%s'", context.TenantID)
	}

	if context.Datasource != "datasource1" {
		t.Errorf("Expected Datasource 'datasource1', got '%s'", context.Datasource)
	}

	if len(context.QueryHistory) != 0 {
		t.Errorf("Expected empty query history, got %d items", len(context.QueryHistory))
	}
}

func TestConversationManager_AddQueryToConversation(t *testing.T) {
	cm := NewConversationManager()
	context := cm.StartConversation("user1", "tenant1", "datasource1")

	intent := &ParsedIntent{
		Metrics:    []string{"revenue"},
		Dimensions: []string{"region"},
		Confidence: 0.9,
	}

	err := cm.AddQueryToConversation(context.ConversationID, "Show me revenue by region", intent, "SELECT * FROM table", true)
	if err != nil {
		t.Fatalf("Failed to add query: %v", err)
	}

	// Retrieve conversation and check
	updatedContext, err := cm.GetConversation(context.ConversationID)
	if err != nil {
		t.Fatalf("Failed to get conversation: %v", err)
	}

	if len(updatedContext.QueryHistory) != 1 {
		t.Errorf("Expected 1 query in history, got %d", len(updatedContext.QueryHistory))
	}

	query := updatedContext.QueryHistory[0]
	if query.UserQuery != "Show me revenue by region" {
		t.Errorf("Expected query text 'Show me revenue by region', got '%s'", query.UserQuery)
	}

	if !query.Success {
		t.Error("Expected query to be successful")
	}
}

func TestConversationManager_EnhanceIntentWithContext(t *testing.T) {
	cm := NewConversationManager()
	context := cm.StartConversation("user1", "tenant1", "datasource1")

	// Add a previous query with metrics and dimensions
	intent := &ParsedIntent{
		Metrics:    []string{"revenue", "profit"},
		Dimensions: []string{"region", "country"},
		Confidence: 0.9,
	}

	cm.AddQueryToConversation(context.ConversationID, "Show me revenue and profit by region and country", intent, "SELECT * FROM table", true)

	// Test enhancing a new intent with no metrics/dimensions
	newIntent := &ParsedIntent{
		Metrics:    []string{}, // Empty
		Dimensions: []string{}, // Empty
		Confidence: 0.8,
	}

	enhancedIntent := cm.EnhanceIntentWithContext(context.ConversationID, newIntent)

	// Should have inherited metrics and dimensions from previous query
	if len(enhancedIntent.Metrics) == 0 {
		t.Error("Expected enhanced intent to inherit metrics from conversation context")
	}

	if len(enhancedIntent.Dimensions) == 0 {
		t.Error("Expected enhanced intent to inherit dimensions from conversation context")
	}

	// Confidence should be slightly reduced for inferred data
	if enhancedIntent.Confidence >= newIntent.Confidence {
		t.Error("Expected confidence to be reduced for inferred data")
	}
}

func TestConversationManager_CleanupExpiredConversations(t *testing.T) {
	cm := NewConversationManager()
	cm.contextTTL = 1 * time.Millisecond // Very short TTL for testing

	context := cm.StartConversation("user1", "tenant1", "datasource1")

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Cleanup should remove expired conversations
	removed := cm.CleanupExpiredConversations()
	if removed != 1 {
		t.Errorf("Expected 1 conversation to be removed, got %d", removed)
	}

	// Now try to get the conversation - should fail
	_, err := cm.GetConversation(context.ConversationID)
	if err == nil {
		t.Error("Expected conversation to be expired")
	}
}

func TestConversationManager_GetConversationSummary(t *testing.T) {
	cm := NewConversationManager()
	context := cm.StartConversation("user1", "tenant1", "datasource1")

	// Add some queries
	intent1 := &ParsedIntent{
		Metrics:    []string{"revenue"},
		Dimensions: []string{"region"},
		Confidence: 0.9,
	}

	intent2 := &ParsedIntent{
		Metrics:    []string{"profit"},
		Dimensions: []string{"country"},
		Confidence: 0.8,
	}

	cm.AddQueryToConversation(context.ConversationID, "Query 1", intent1, "SELECT 1", true)
	cm.AddQueryToConversation(context.ConversationID, "Query 2", intent2, "SELECT 2", false)

	summary, err := cm.GetConversationSummary(context.ConversationID)
	if err != nil {
		t.Fatalf("Failed to get conversation summary: %v", err)
	}

	if queryCount, ok := summary["query_count"].(int); !ok || queryCount != 2 {
		t.Errorf("Expected 2 queries, got %v", summary["query_count"])
	}

	insights, ok := summary["insights"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected insights in summary")
	}

	if successfulQueries, ok := insights["successful_queries"].(int); !ok || successfulQueries != 1 {
		t.Errorf("Expected 1 successful query, got %v", insights["successful_queries"])
	}

	if failedQueries, ok := insights["failed_queries"].(int); !ok || failedQueries != 1 {
		t.Errorf("Expected 1 failed query, got %v", insights["failed_queries"])
	}

	expectedAvgConfidence := (0.9 + 0.8) / 2
	if avgConfidence, ok := insights["avg_confidence"].(float64); !ok || math.Abs(avgConfidence-expectedAvgConfidence) > 0.0001 {
		t.Errorf("Expected avg confidence %.2f, got %v (type: %T)", expectedAvgConfidence, insights["avg_confidence"], insights["avg_confidence"])
	}
}

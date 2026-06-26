package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/query"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

type NLQueryHandler struct {
	nlEngine *query.NLQueryEngine
}

func NewNLQueryHandler(nlEngine *query.NLQueryEngine) *NLQueryHandler {
	return &NLQueryHandler{
		nlEngine: nlEngine,
	}
}

// HandleCompileNLQuery handles NL query compilation
func (h *NLQueryHandler) HandleCompileNLQuery(w http.ResponseWriter, r *http.Request) {
	var req query.NLQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Extract context from headers
	userID := r.Header.Get("X-User-ID")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	if userID == "" || tenantID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Authentication required (missing X-User-ID or X-Tenant-ID)"})
		return
	}

	// Set user and tenant from headers
	req.UserID = userID
	req.TenantID = tenantID

	// Process the NL query
	response, err := h.nlEngine.ProcessNLQuery(r.Context(), &req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to process NL query: " + err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleSimulateNLQuery handles NL query simulation
func (h *NLQueryHandler) HandleSimulateNLQuery(w http.ResponseWriter, r *http.Request) {
	var req query.NLQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Extract context from headers
	userID := r.Header.Get("X-User-ID")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	if userID == "" || tenantID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Authentication required"})
		return
	}

	// Set user and tenant from headers
	req.UserID = userID
	req.TenantID = tenantID

	// Simulate the NL query
	response, err := h.nlEngine.SimulateNLQuery(r.Context(), &req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to simulate NL query: " + err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetNLQueryHistory handles retrieving NL query history
func (h *NLQueryHandler) HandleGetNLQueryHistory(w http.ResponseWriter, r *http.Request) {
	// Extract context from headers
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Authentication required"})
		return
	}

	// TODO: Implement history retrieval from database
	// For now, return empty history
	history := []query.NLQueryResponse{}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id": userID,
		"history": history,
	})
}

// HandleGetNLQuerySuggestions handles getting query suggestions
func (h *NLQueryHandler) HandleGetNLQuerySuggestions(w http.ResponseWriter, r *http.Request) {
	// Extract context from headers
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Authentication required"})
		return
	}

	// TODO: Implement suggestion logic based on user history and schema
	suggestions := []string{
		"Show me average order value by region",
		"What is the total revenue last quarter",
		"Show me customer count by segment",
		"What is the profit margin trend",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":     userID,
		"suggestions": suggestions,
	})
}

// ---- Conversational Query Refinement Endpoints ----

// HandleStartConversation starts a new conversational query refinement session
func (h *NLQueryHandler) HandleStartConversation(w http.ResponseWriter, r *http.Request) {
	var req query.NLQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Extract context from headers
	userID := r.Header.Get("X-User-ID")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	if userID == "" || tenantID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Authentication required"})
		return
	}

	// Set user and tenant from headers
	req.UserID = userID
	req.TenantID = tenantID

	// Start conversation using dialogue manager
	if h.nlEngine.GetDialogueManager() == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Dialogue manager not available"})
		return
	}

	refinementCtx, err := h.nlEngine.GetDialogueManager().StartRefinement(r.Context(), &req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to start conversation: " + err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(refinementCtx)
}

// HandleProcessConversationMessage processes a user message in an ongoing conversation
func (h *NLQueryHandler) HandleProcessConversationMessage(w http.ResponseWriter, r *http.Request) {
	conversationID := chi.URLParam(r, "conversationId")

	var req struct {
		Message string `json:"message" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Process message using dialogue manager
	if h.nlEngine.GetDialogueManager() == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Dialogue manager not available"})
		return
	}

	refinementCtx, err := h.nlEngine.GetDialogueManager().ProcessUserResponse(r.Context(), conversationID, req.Message)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to process message: " + err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(refinementCtx)
}

// HandleGetConversationState retrieves the current state of a conversation
func (h *NLQueryHandler) HandleGetConversationState(w http.ResponseWriter, r *http.Request) {
	conversationID := chi.URLParam(r, "conversationId")

	// Extract context from headers
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Authentication required"})
		return
	}

	// Get conversation state
	if h.nlEngine.GetDialogueManager() == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Dialogue manager not available"})
		return
	}

	refinementCtx, err := h.nlEngine.GetDialogueManager().GetRefinementContext(conversationID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Conversation not found: " + err.Error()})
		return
	}

	// Verify user owns this conversation
	if refinementCtx.UserID != userID {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Access denied to this conversation"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(refinementCtx)
}

// HandleCommitConversation commits a refined query for execution
func (h *NLQueryHandler) HandleCommitConversation(w http.ResponseWriter, r *http.Request) {
	conversationID := chi.URLParam(r, "conversationId")

	// Extract context from headers
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Authentication required"})
		return
	}

	// Get conversation state
	if h.nlEngine.GetDialogueManager() == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Dialogue manager not available"})
		return
	}

	refinementCtx, err := h.nlEngine.GetDialogueManager().GetRefinementContext(conversationID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Conversation not found: " + err.Error()})
		return
	}

	// Verify user owns this conversation
	if refinementCtx.UserID != userID {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Access denied to this conversation"})
		return
	}

	// Check if conversation is ready for execution
	if refinementCtx.State != query.StateReady {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Conversation is not ready for execution. Current state: " + string(refinementCtx.State),
		})
		return
	}

	// Execute the final query
	if refinementCtx.CurrentQuery == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "No query available for execution"})
		return
	}

	// TODO: Execute the query and return results
	// For now, just return the query that would be executed
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"conversation_id": conversationID,
		"query":           refinementCtx.CurrentQuery,
		"message":         "Query committed for execution",
		"status":          "success",
	})
}

// HandleGetConversationSummary gets a summary of a conversation
func (h *NLQueryHandler) HandleGetConversationSummary(w http.ResponseWriter, r *http.Request) {
	conversationID := chi.URLParam(r, "conversationId")

	// Extract context from headers
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Authentication required"})
		return
	}

	// Get conversation summary
	if h.nlEngine.GetDialogueManager() == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Dialogue manager not available"})
		return
	}

	refinementCtx, err := h.nlEngine.GetDialogueManager().GetRefinementContext(conversationID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Conversation not found: " + err.Error()})
		return
	}

	// Verify user owns this conversation
	if refinementCtx.UserID != userID {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Access denied to this conversation"})
		return
	}

	// Generate summary
	summary := h.generateConversationSummary(refinementCtx)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// Helper method to generate conversation summary
func (h *NLQueryHandler) generateConversationSummary(refinementCtx *query.RefinementContext) map[string]interface{} {
	messageCount := len(refinementCtx.Messages)
	clarificationCount := 0
	suggestionCount := 0

	for _, msg := range refinementCtx.Messages {
		switch msg.Type {
		case "clarification":
			clarificationCount++
		case "suggestion":
			suggestionCount++
		}
	}

	return map[string]interface{}{
		"conversation_id":     refinementCtx.ConversationID,
		"user_id":             refinementCtx.UserID,
		"tenant_id":           refinementCtx.TenantID,
		"datasource":          refinementCtx.Datasource,
		"state":               refinementCtx.State,
		"original_query":      refinementCtx.OriginalQuery,
		"message_count":       messageCount,
		"clarification_count": clarificationCount,
		"suggestion_count":    suggestionCount,
		"refinement_steps":    len(refinementCtx.RefinementHistory),
		"last_activity":       refinementCtx.LastActivity,
		"created_at":          refinementCtx.CreatedAt,
		"duration":            refinementCtx.LastActivity.Sub(refinementCtx.CreatedAt).String(),
		"current_query":       refinementCtx.CurrentQuery,
		"current_intent":      refinementCtx.CurrentIntent,
	}
}

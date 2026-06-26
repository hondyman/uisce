package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/query"
)

// DashboardConversationHandler handles dashboard conversation API endpoints
type DashboardConversationHandler struct {
	dashboardManager *query.DashboardConversationManager
	nlEngine         *query.NLQueryEngine
}

// NewDashboardConversationHandler creates a new dashboard conversation handler
func NewDashboardConversationHandler(dashboardManager *query.DashboardConversationManager, nlEngine *query.NLQueryEngine) *DashboardConversationHandler {
	return &DashboardConversationHandler{
		dashboardManager: dashboardManager,
		nlEngine:         nlEngine,
	}
}

// RegisterRoutes registers the routes for DashboardConversationHandler.
func (h *DashboardConversationHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/dashboard/conversations", func(r chi.Router) {
		r.Post("/", h.HandleStartConversation)
		r.Get("/{id}", h.HandleGetConversation)
		r.Post("/{id}/messages", h.HandleProcessMessage)
		r.Post("/{id}/commit", h.HandleCommitConversation)
	})
}

// StartConversationRequest represents the request to start a dashboard conversation
type StartConversationRequest struct {
	UserID     string `json:"user_id" binding:"required"`
	TenantID   string `json:"tenant_id" binding:"required"`
	Datasource string `json:"datasource" binding:"required"`
	Message    string `json:"message" binding:"required"`
}

// StartConversationResponse represents the response from starting a conversation
type StartConversationResponse struct {
	ConversationID string                    `json:"conversation_id"`
	State          string                    `json:"state"`
	Title          string                    `json:"title"`
	Description    string                    `json:"description"`
	Visuals        []query.DashboardVisual   `json:"visuals"`
	Layout         query.DashboardLayout     `json:"layout"`
	Compliance     query.DashboardCompliance `json:"compliance"`
	CreatedAt      time.Time                 `json:"created_at"`
}

// ProcessMessageRequest represents the request to process a message
type ProcessMessageRequest struct {
	Message string `json:"message" binding:"required"`
}

// ProcessMessageResponse represents the response from processing a message
type ProcessMessageResponse struct {
	ConversationID string                    `json:"conversation_id"`
	State          string                    `json:"state"`
	Visuals        []query.DashboardVisual   `json:"visuals"`
	Layout         query.DashboardLayout     `json:"layout"`
	Compliance     query.DashboardCompliance `json:"compliance"`
	LastMessage    query.ConversationMessage `json:"last_message"`
	UpdatedAt      time.Time                 `json:"updated_at"`
}

// CommitConversationRequest represents the request to commit a conversation
type CommitConversationRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
}

// CommitConversationResponse represents the response from committing a conversation
type CommitConversationResponse struct {
	ConversationID string                    `json:"conversation_id"`
	State          string                    `json:"state"`
	Title          string                    `json:"title"`
	Description    string                    `json:"description"`
	Visuals        []query.DashboardVisual   `json:"visuals"`
	Layout         query.DashboardLayout     `json:"layout"`
	Compliance     query.DashboardCompliance `json:"compliance"`
	CommittedAt    time.Time                 `json:"committed_at"`
}

// HandleStartConversation starts a new dashboard conversation
func (h *DashboardConversationHandler) HandleStartConversation(w http.ResponseWriter, r *http.Request) {
	var req StartConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	// Start the conversation
	conversation, err := h.dashboardManager.StartConversation(
		r.Context(),
		req.UserID,
		req.TenantID,
		req.Datasource,
		req.Message,
	)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	// Build response
	response := StartConversationResponse{
		ConversationID: conversation.ID,
		State:          conversation.State,
		Title:          conversation.Title,
		Description:    conversation.Description,
		Visuals:        conversation.Visuals,
		Layout:         conversation.Layout,
		Compliance:     conversation.ComplianceStatus,
		CreatedAt:      conversation.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleProcessMessage processes a message in an existing conversation
func (h *DashboardConversationHandler) HandleProcessMessage(w http.ResponseWriter, r *http.Request) {
	conversationID := chi.URLParam(r, "id")

	var req ProcessMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	// Process the message
	conversation, err := h.dashboardManager.ProcessMessage(
		r.Context(),
		conversationID,
		req.Message,
	)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	// Get the last message (assistant response)
	var lastMessage query.ConversationMessage
	if len(conversation.Messages) > 0 {
		lastMessage = conversation.Messages[len(conversation.Messages)-1]
	}

	// Build response
	response := ProcessMessageResponse{
		ConversationID: conversation.ID,
		State:          conversation.State,
		Visuals:        conversation.Visuals,
		Layout:         conversation.Layout,
		Compliance:     conversation.ComplianceStatus,
		LastMessage:    lastMessage,
		UpdatedAt:      conversation.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetConversation retrieves a dashboard conversation
func (h *DashboardConversationHandler) HandleGetConversation(w http.ResponseWriter, r *http.Request) {
	conversationID := chi.URLParam(r, "id")

	conversation, err := h.dashboardManager.GetConversation(conversationID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conversation)
}

// HandleCommitConversation commits/finalizes a dashboard conversation
func (h *DashboardConversationHandler) HandleCommitConversation(w http.ResponseWriter, r *http.Request) {
	conversationID := chi.URLParam(r, "id")

	var req CommitConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	// Commit the conversation
	conversation, err := h.dashboardManager.CommitConversation(
		conversationID,
		req.Title,
		req.Description,
	)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	// Build response
	response := CommitConversationResponse{
		ConversationID: conversation.ID,
		State:          conversation.State,
		Title:          conversation.Title,
		Description:    conversation.Description,
		Visuals:        conversation.Visuals,
		Layout:         conversation.Layout,
		Compliance:     conversation.ComplianceStatus,
		CommittedAt:    conversation.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

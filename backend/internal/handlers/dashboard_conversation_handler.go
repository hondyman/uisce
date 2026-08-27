package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/query"
)

// DashboardConversationHandler handles dashboard conversation API endpoints
type DashboardConversationHandler struct {
	db               *sql.DB
	dashboardManager *query.DashboardConversationManager
}

// NewDashboardConversationHandler creates a new dashboard conversation handler
func NewDashboardConversationHandler(db *sql.DB, dashboardManager *query.DashboardConversationManager) *DashboardConversationHandler {
	return &DashboardConversationHandler{
		db:               db,
		dashboardManager: dashboardManager,
	}
}

// DashboardConversationRequest structures used by the frontend
type StartConversationRequest struct {
	UserID     string `json:"user_id" binding:"required"`
	TenantID   string `json:"tenant_id" binding:"required"`
	Datasource string `json:"datasource" binding:"required"`
	Message    string `json:"message" binding:"required"`
}

type ProcessMessageRequest struct {
	Message string `json:"message" binding:"required"`
}

type CommitConversationRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type CreateDashboardFromVisualRequest struct {
	DashboardName string `json:"dashboard_name"`
}

type CreateDashboardFromVisualResponse struct {
	DashboardID   string `json:"dashboard_id"`
	VisualID      string `json:"visual_id"`
	DashboardName string `json:"dashboard_name"`
}

type CommitConversationResponse struct {
	ConversationID  string                    `json:"conversation_id"`
	State          string                    `json:"state"`
	Title          string                    `json:"title"`
	Description    string                    `json:"description"`
	DashboardID    string                    `json:"dashboard_id,omitempty"`
	VisualCount    int                       `json:"visual_count"`
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":          conversation.ID,
		"state":       conversation.State,
		"title":       conversation.Title,
		"description": conversation.Description,
		"visuals":     conversation.Visuals,
		"layout":      conversation.Layout,
		"compliance":   conversation.ComplianceStatus,
		"messages":     conversation.Messages,
		"created_at":   conversation.CreatedAt,
	})
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

	conversation, err := h.dashboardManager.ProcessMessage(r.Context(), conversationID, req.Message)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	var lastMessage query.ConversationMessage
	if len(conversation.Messages) > 0 {
		lastMessage = conversation.Messages[len(conversation.Messages)-1]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           conversation.ID,
		"state":        conversation.State,
		"visuals":      conversation.Visuals,
		"layout":       conversation.Layout,
		"compliance":    conversation.ComplianceStatus,
		"last_message":  lastMessage,
		"updated_at":    conversation.UpdatedAt,
	})
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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":          conversation.ID,
		"state":       conversation.State,
		"title":       conversation.Title,
		"description": conversation.Description,
		"visuals":     conversation.Visuals,
		"layout":      conversation.Layout,
		"compliance":  conversation.ComplianceStatus,
		"messages":    conversation.Messages,
	})
}

// HandleCommitConversation saves all visuals from conversation as a new dashboard
func (h *DashboardConversationHandler) HandleCommitConversation(w http.ResponseWriter, r *http.Request) {
	conversationID := chi.URLParam(r, "id")

	var req CommitConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	conversation, err := h.dashboardManager.GetConversation(conversationID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	if len(conversation.Visuals) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "No visuals to save"})
		return
	}

	dashboardID := uuid.New().String()
	now := time.Now()

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(r.Context(), `
		INSERT INTO dashboards (id, name, description, widgets, layout, theme, is_public, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, dashboardID, req.Title, req.Description, "[]", conversation.Layout.Type, "light", false,
		conversation.UserID, now, now)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to create dashboard"})
		return
	}

	for i, visual := range conversation.Visuals {
		querySpecJSON, _ := json.Marshal(visual.QuerySpec)
		visualConfigJSON, _ := json.Marshal(visual.Config)
		positionJSON, _ := json.Marshal(visual.Position)
		complianceJSON, _ := json.Marshal(visual.Compliance)

		_, err = tx.ExecContext(r.Context(), `
			INSERT INTO dashboard_visuals (
				id, dashboard_id, visual_type, title, description,
				query_spec, visual_config, position, compliance,
				created_from_conversation_id, created_from_visual_id,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`, uuid.New(), dashboardID, visual.Type, visual.Title, visual.Description,
			querySpecJSON, visualConfigJSON, positionJSON, complianceJSON,
			conversationID, visual.ID, now, now)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": fmt.Sprintf("Failed to save visual %d: %v", i, err)})
			return
		}
	}

	if err = tx.Commit(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to commit transaction"})
		return
	}

	h.dashboardManager.CommitConversation(conversationID, req.Title, req.Description)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CommitConversationResponse{
		ConversationID: conversationID,
		State:         "completed",
		Title:         req.Title,
		Description:   req.Description,
		DashboardID:   dashboardID,
		VisualCount:   len(conversation.Visuals),
		CommittedAt:   now,
	})
}

// HandleCreateDashboardFromVisual saves a single visual as a new dashboard
func (h *DashboardConversationHandler) HandleCreateDashboardFromVisual(w http.ResponseWriter, r *http.Request) {
	conversationID := chi.URLParam(r, "id")
	visualID := chi.URLParam(r, "visualId")

	var req CreateDashboardFromVisualRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.DashboardName = ""
	}

	conversation, err := h.dashboardManager.GetConversation(conversationID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Conversation not found"})
		return
	}

	var visual *query.DashboardVisual
	for _, v := range conversation.Visuals {
		if v.ID == visualID {
			visual = &v
			break
		}
	}

	if visual == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Visual not found"})
		return
	}

	dashboardID := uuid.New().String()
	visualIDOut := uuid.New().String()
	now := time.Now()

	dashboardName := req.DashboardName
	if dashboardName == "" {
		dashboardName = fmt.Sprintf("%s Dashboard", visual.Title)
	}

	_, err = h.db.ExecContext(r.Context(), `
		INSERT INTO dashboards (id, name, description, widgets, layout, theme, is_public, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, dashboardID, dashboardName, visual.Description, "[]", "grid", "light", false,
		conversation.UserID, now, now)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to create dashboard"})
		return
	}

	querySpecJSON, _ := json.Marshal(visual.QuerySpec)
	visualConfigJSON, _ := json.Marshal(visual.Config)
	positionJSON, _ := json.Marshal(visual.Position)
	complianceJSON, _ := json.Marshal(visual.Compliance)

	_, err = h.db.ExecContext(r.Context(), `
		INSERT INTO dashboard_visuals (
			id, dashboard_id, visual_type, title, description,
			query_spec, visual_config, position, compliance,
			created_from_conversation_id, created_from_visual_id,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, visualIDOut, dashboardID, visual.Type, visual.Title, visual.Description,
		querySpecJSON, visualConfigJSON, positionJSON, complianceJSON,
		conversationID, visual.ID, now, now)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to save visual"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CreateDashboardFromVisualResponse{
		DashboardID:   dashboardID,
		VisualID:      visualIDOut,
		DashboardName: dashboardName,
	})
}

// HandleDeleteConversation abandons a conversation
func (h *DashboardConversationHandler) HandleDeleteConversation(w http.ResponseWriter, r *http.Request) {
	conversationID := chi.URLParam(r, "id")

	err := h.dashboardManager.DeleteConversation(conversationID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "deleted"})
}

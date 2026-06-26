package onboarding

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// Handler handles onboarding HTTP requests
type Handler struct {
	service Service
}

// NewHandler creates a new onboarding handler
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// CreateSession creates a new onboarding session
// POST /api/onboarding/sessions
func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Get tenant ID from header
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID
	tenantUUID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid tenant ID"})
		return
	}

	session, err := h.service.CreateSession(r.Context(), tenantUUID, req.Email)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(session)
}

// ResumeSession retrieves session by resume token
// GET /api/onboarding/sessions/resume/{token}
func (h *Handler) ResumeSession(w http.ResponseWriter, r *http.Request) {
	tokenStr := chi.URLParam(r, "token")
	resumeToken, err := uuid.Parse(tokenStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid resume token"})
		return
	}

	session, err := h.service.GetSessionByToken(r.Context(), resumeToken)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Session not found or expired"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// GetSession retrieves session by ID
// GET /api/onboarding/sessions/{id}
func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := chi.URLParam(r, "id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid session ID"})
		return
	}

	session, err := h.service.GetSession(r.Context(), sessionID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Session not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// UpdateSession updates session progress
// PUT /api/onboarding/sessions/{id}
func (h *Handler) UpdateSession(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := chi.URLParam(r, "id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid session ID"})
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	err = h.service.UpdateSession(r.Context(), sessionID, updates)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Session updated successfully"})
}

// CompleteOnboarding completes the onboarding process
// POST /api/onboarding/sessions/{id}/complete
func (h *Handler) CompleteOnboarding(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := chi.URLParam(r, "id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid session ID"})
		return
	}

	err = h.service.CompleteOnboarding(r.Context(), sessionID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Onboarding completed successfully",
		"status":  "COMPLETED",
	})
}

// UploadDocument handles document uploads
// POST /api/onboarding/sessions/{id}/documents
func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	// Get session ID from URL params
	sessionIDStr := chi.URLParam(r, "id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid session ID"})
		return
	}

	// Check file size (10MB limit)
	if r.ContentLength > 10*1024*1024 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "File too large"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	documentType := r.FormValue("document_type")
	if documentType == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Document type required"})
		return
	}

	// TODO: Upload file to S3 or storage service
	storagePath := "/uploads/" + header.Filename

	// Get client ID from session
	session, err := h.service.GetSession(r.Context(), sessionID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Session not found"})
		return
	}

	clientID := session.ClientID
	if clientID == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Session has no associated client"})
		return
	}

	// Create upload input matching struct definition
	input := UploadDocumentInput{
		ClientID:            *clientID,
		OnboardingSessionID: &sessionID,
		DocumentType:        DocumentType(documentType),
		FileURL:             storagePath,
		FileName:            header.Filename,
		FileSizeBytes:       header.Size,
		MimeType:            header.Header.Get("Content-Type"),
	}

	doc, err := h.service.UploadDocument(r.Context(), input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(doc)
}

func stringPtr(s string) *string {
	return &s
}

// GetDocuments retrieves documents for a session
// GET /api/onboarding/sessions/{id}/documents
func (h *Handler) GetDocuments(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := chi.URLParam(r, "id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid session ID"})
		return
	}

	documents, err := h.service.GetDocuments(r.Context(), sessionID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"documents": documents,
		"total":     len(documents),
	})
}

// GetProgress retrieves onboarding progress
// GET /api/onboarding/sessions/{id}/progress
func (h *Handler) GetProgress(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := chi.URLParam(r, "id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid session ID"})
		return
	}

	progress, err := h.service.CalculateProgress(r.Context(), sessionID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id": sessionID,
		"progress":   progress,
	})
}

// RegisterRoutes registers onboarding routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/onboarding", func(r chi.Router) {
		r.Post("/sessions", h.CreateSession)
		r.Post("/sessions/{token}/resume", h.ResumeSession)
		r.Get("/sessions/{id}", h.GetSession)
		r.Put("/sessions/{id}/step", h.UpdateSession)
		r.Post("/sessions/{id}/complete", h.CompleteOnboarding)
		r.Get("/sessions/{id}/progress", h.GetProgress)

		// Document management
		r.Post("/sessions/{id}/documents", h.UploadDocument)
		r.Get("/sessions/{id}/documents", h.GetDocuments)
	})
}

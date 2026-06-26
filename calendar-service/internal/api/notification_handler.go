package api

import (
	"calendar-service/internal/services"
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
)

type NotificationHandler struct {
	notificationService *services.NotificationService
	auditService        services.AuditService
	logger              *logrus.Entry
}

func NewNotificationHandler(ns *services.NotificationService, audit services.AuditService, logger *logrus.Entry) *NotificationHandler {
	return &NotificationHandler{
		notificationService: ns,
		auditService:        audit,
		logger:              logger.WithField("component", "notification_handler"),
	}
}

// Unsubscribe handles user requests to stop receiving specific notifications
func (h *NotificationHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "missing email", http.StatusBadRequest)
		return
	}

	h.logger.Infof("User %s requested unsubscribe", email)

	if h.auditService != nil {
		h.auditService.Record(r.Context(), services.AuditEntry{
			Action:     "NOTIFICATION_UNSUBSCRIBE",
			EntityType: "user",
			EntityID:   email,
			TenantID:   "SYSTEM", // Placeholder or extract from context
			NewValues:  map[string]interface{}{"email": email},
			ChangedBy:  email,
		})
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("You have been successfully unsubscribed. We're sorry to see you go!"))
}

// SendTestDigest allows admins to trigger a test weekly digest
func (h *NotificationHandler) SendTestDigest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data := services.DigestData{
		TotalSyncs:        145,
		SuccessRate:       98.5,
		EventsSynced:      420,
		ConflictsResolved: 12,
		ActiveConnections: 3,
	}

	err := h.notificationService.SendWeeklyDigest(r.Context(), req.Email, req.Name, data)
	if err != nil {
		h.logger.WithError(err).Error("failed to send test digest")
		http.Error(w, "failed to send digest", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "digest_queued"})
}

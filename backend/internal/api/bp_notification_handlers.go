package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ============================================================================
// TYPES - Business Process Notifications
// ============================================================================

type BPNotificationTemplate struct {
	ID                       string         `json:"id" db:"id"`
	TenantID                 string         `json:"tenant_id" db:"tenant_id"`
	DatasourceID             string         `json:"datasource_id" db:"datasource_id"`
	TemplateKey              string         `json:"template_key" db:"template_key"`
	TemplateName             string         `json:"template_name" db:"template_name"`
	Description              string         `json:"description" db:"description"`
	Category                 string         `json:"category" db:"category"`
	SubjectTemplate          string         `json:"subject_template" db:"subject_template"`
	BodyTemplate             string         `json:"body_template" db:"body_template"`
	TemplateVariables        []string       `json:"template_variables" db:"template_variables"`
	EnabledChannels          []string       `json:"enabled_channels" db:"enabled_channels"`
	DefaultChannel           string         `json:"default_channel" db:"default_channel"`
	SendConditions           sql.NullString `json:"send_conditions" db:"send_conditions"`
	SendDelayMinutes         int            `json:"send_delay_minutes" db:"send_delay_minutes"`
	DigestMode               string         `json:"digest_mode" db:"digest_mode"`
	EscalationEnabled        bool           `json:"escalation_enabled" db:"escalation_enabled"`
	EscalationDelayMinutes   *int           `json:"escalation_delay_minutes" db:"escalation_delay_minutes"`
	EscalationRecipientRoles []string       `json:"escalation_recipient_roles" db:"escalation_recipient_roles"`
	IsSystem                 bool           `json:"is_system" db:"is_system"`
	IsActive                 bool           `json:"is_active" db:"is_active"`
	Priority                 string         `json:"priority" db:"priority"`
	IncludeAttachments       bool           `json:"include_attachments" db:"include_attachments"`
	IncludeQuickActions      bool           `json:"include_quick_actions" db:"include_quick_actions"`
	QuickActions             sql.NullString `json:"quick_actions" db:"quick_actions"`
	CreatedBy                string         `json:"created_by" db:"created_by"`
	CreatedAt                time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time      `json:"updated_at" db:"updated_at"`
}

type BPUserNotificationPreferences struct {
	ID                 string    `json:"id" db:"id"`
	TenantID           string    `json:"tenant_id" db:"tenant_id"`
	DatasourceID       string    `json:"datasource_id" db:"datasource_id"`
	UserID             string    `json:"user_id" db:"user_id"`
	EmailEnabled       bool      `json:"email_enabled" db:"email_enabled"`
	EmailAddress       *string   `json:"email_address" db:"email_address"`
	SmsEnabled         bool      `json:"sms_enabled" db:"sms_enabled"`
	PhoneNumber        *string   `json:"phone_number" db:"phone_number"`
	SlackEnabled       bool      `json:"slack_enabled" db:"slack_enabled"`
	SlackUserID        *string   `json:"slack_user_id" db:"slack_user_id"`
	SlackWebhookURL    *string   `json:"slack_webhook_url" db:"slack_webhook_url"`
	TeamsEnabled       bool      `json:"teams_enabled" db:"teams_enabled"`
	TeamsUserID        *string   `json:"teams_user_id" db:"teams_user_id"`
	TeamsWebhookURL    *string   `json:"teams_webhook_url" db:"teams_webhook_url"`
	PushEnabled        bool      `json:"push_enabled" db:"push_enabled"`
	PushToken          *string   `json:"push_token" db:"push_token"`
	DigestMode         string    `json:"digest_mode" db:"digest_mode"`
	DigestTime         *string   `json:"digest_time" db:"digest_time"`
	DigestDays         []int     `json:"digest_days" db:"digest_days"`
	IncludeSummary     bool      `json:"include_summary" db:"include_summary"`
	IncludeFullDetails bool      `json:"include_full_details" db:"include_full_details"`
	DndEnabled         bool      `json:"dnd_enabled" db:"dnd_enabled"`
	DndStartTime       *string   `json:"dnd_start_time" db:"dnd_start_time"`
	DndEndTime         *string   `json:"dnd_end_time" db:"dnd_end_time"`
	MinPriority        string    `json:"min_priority" db:"min_priority"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

type BPNotificationLog struct {
	ID                string          `json:"id" db:"id"`
	TenantID          string          `json:"tenant_id" db:"tenant_id"`
	DatasourceID      string          `json:"datasource_id" db:"datasource_id"`
	TemplateID        *string         `json:"template_id" db:"template_id"`
	TemplateKey       *string         `json:"template_key" db:"template_key"`
	RecipientUserID   string          `json:"recipient_user_id" db:"recipient_user_id"`
	RecipientEmail    *string         `json:"recipient_email" db:"recipient_email"`
	RecipientPhone    *string         `json:"recipient_phone" db:"recipient_phone"`
	Subject           string          `json:"subject" db:"subject"`
	Body              string          `json:"body" db:"body"`
	RenderedContent   json.RawMessage `json:"rendered_content" db:"rendered_content"`
	Channel           string          `json:"channel" db:"channel"`
	Status            string          `json:"status" db:"status"`
	DeliveryProvider  *string         `json:"delivery_provider" db:"delivery_provider"`
	SentAt            *time.Time      `json:"sent_at" db:"sent_at"`
	DeliveredAt       *time.Time      `json:"delivered_at" db:"delivered_at"`
	OpenedAt          *time.Time      `json:"opened_at" db:"opened_at"`
	ClickedAt         *time.Time      `json:"clicked_at" db:"clicked_at"`
	ActionTaken       *string         `json:"action_taken" db:"action_taken"`
	ActionTakenAt     *time.Time      `json:"action_taken_at" db:"action_taken_at"`
	ErrorMessage      *string         `json:"error_message" db:"error_message"`
	RetryCount        *int            `json:"retry_count" db:"retry_count"`
	NextRetryAt       *time.Time      `json:"next_retry_at" db:"next_retry_at"`
	ProcessID         *string         `json:"process_id" db:"process_id"`
	ProcessInstanceID *string         `json:"process_instance_id" db:"process_instance_id"`
	StepID            *string         `json:"step_id" db:"step_id"`
	RelatedEntityType *string         `json:"related_entity_type" db:"related_entity_type"`
	RelatedEntityID   *string         `json:"related_entity_id" db:"related_entity_id"`
	Priority          *string         `json:"priority" db:"priority"`
	IsDigest          *bool           `json:"is_digest" db:"is_digest"`
	DigestBatchID     *string         `json:"digest_batch_id" db:"digest_batch_id"`
	CreatedAt         *time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt         *time.Time      `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// HANDLERS
// ============================================================================

type BPNotificationHandlers struct {
	db *sqlx.DB
}

func NewBPNotificationHandlers(db *sqlx.DB) *BPNotificationHandlers {
	return &BPNotificationHandlers{db: db}
}

func (h *BPNotificationHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/bp-notifications", func(r chi.Router) {
		// Templates
		r.Get("/templates", h.GetTemplates)
		r.Get("/templates/{id}", h.GetTemplate)
		r.Post("/templates", h.CreateTemplate)
		r.Put("/templates/{id}", h.UpdateTemplate)
		r.Delete("/templates/{id}", h.DeleteTemplate)
		r.Post("/templates/{id}/render", h.RenderTemplate)

		// Send
		r.Post("/send", h.SendNotification)
		r.Post("/send-batch", h.SendBatchNotifications)

		// Preferences
		r.Get("/preferences", h.GetUserPreferences)
		r.Put("/preferences", h.UpdateUserPreferences)

		// Logs
		r.Get("/logs", h.GetLogs)
		r.Get("/logs/{id}", h.GetLog)
		r.Get("/analytics", h.GetAnalytics)

		// Digests
		r.Get("/digests/pending", h.GetPendingDigests)
		r.Post("/digests/process", h.ProcessDigests)

		// Webhooks
		r.Post("/webhook/delivered/{id}", h.MarkDelivered)
		r.Post("/webhook/opened/{id}", h.MarkOpened)
		r.Post("/webhook/clicked/{id}", h.MarkClicked)
		r.Post("/webhook/action/{id}", h.RecordAction)
	})
}

// ============================================================================
// TEMPLATE ENDPOINTS
// ============================================================================

func (h *BPNotificationHandlers) GetTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = jwtmiddleware.GetClaimsFromContext(r).TenantID
	}
	datasourceID := r.URL.Query().Get("datasource_id")
	if datasourceID == "" {
		datasourceID = r.URL.Query().Get("tenant_instance_id")
	}
	if datasourceID == "" {
		datasourceID = r.Header.Get("X-Tenant-Datasource-ID")
	}
	if datasourceID == "" {
		datasourceID = r.Header.Get("X-Tenant-Instance-ID")
	}

	if tenantID == "" {
		respondWithError(w, http.StatusBadRequest, "Missing tenant context")
		return
	}
	category := r.URL.Query().Get("category")

	query := `SELECT * FROM notification_templates 
	          WHERE tenant_id = $1 AND datasource_id = $2 AND is_active = true`
	args := []interface{}{tenantID, datasourceID}

	if category != "" {
		query += " AND category = $3"
		args = append(args, category)
	}

	query += " ORDER BY template_name"

	var templates []BPNotificationTemplate
	if err := h.db.Select(&templates, query, args...); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSONBP(w, templates)
}

func (h *BPNotificationHandlers) GetTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var template BPNotificationTemplate
	err := h.db.Get(&template, "SELECT * FROM notification_templates WHERE id = $1", id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Template not found")
		return
	}

	respondJSONBP(w, template)
}

func (h *BPNotificationHandlers) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	if datasourceID == "" {
		datasourceID = r.URL.Query().Get("datasource_id")
	}

	var req BPNotificationTemplate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	req.ID = uuid.New().String()
	req.TenantID = tenantID
	req.DatasourceID = datasourceID
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	query := `INSERT INTO notification_templates 
	          (id, tenant_id, datasource_id, template_key, template_name, description, category,
	           subject_template, body_template, template_variables, enabled_channels, default_channel,
	           send_conditions, send_delay_minutes, digest_mode, escalation_enabled,
	           escalation_delay_minutes, escalation_recipient_roles, is_system, is_active, priority,
	           include_attachments, include_quick_actions, quick_actions, created_by)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)`

	_, err := h.db.Exec(query, req.ID, req.TenantID, req.DatasourceID, req.TemplateKey, req.TemplateName,
		req.Description, req.Category, req.SubjectTemplate, req.BodyTemplate, req.TemplateVariables,
		req.EnabledChannels, req.DefaultChannel, req.SendConditions, req.SendDelayMinutes, req.DigestMode,
		req.EscalationEnabled, req.EscalationDelayMinutes, req.EscalationRecipientRoles, req.IsSystem,
		req.IsActive, req.Priority, req.IncludeAttachments, req.IncludeQuickActions, req.QuickActions,
		req.CreatedBy)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSONBP(w, req)
}

func (h *BPNotificationHandlers) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req BPNotificationTemplate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	query := `UPDATE notification_templates 
	          SET template_name = $1, description = $2, category = $3, subject_template = $4,
	              body_template = $5, template_variables = $6, enabled_channels = $7,
	              default_channel = $8, send_conditions = $9, send_delay_minutes = $10,
	              digest_mode = $11, escalation_enabled = $12, escalation_delay_minutes = $13,
	              escalation_recipient_roles = $14, is_active = $15, priority = $16,
	              include_attachments = $17, include_quick_actions = $18, quick_actions = $19
	          WHERE id = $20`

	_, err := h.db.Exec(query, req.TemplateName, req.Description, req.Category, req.SubjectTemplate,
		req.BodyTemplate, req.TemplateVariables, req.EnabledChannels, req.DefaultChannel,
		req.SendConditions, req.SendDelayMinutes, req.DigestMode, req.EscalationEnabled,
		req.EscalationDelayMinutes, req.EscalationRecipientRoles, req.IsActive, req.Priority,
		req.IncludeAttachments, req.IncludeQuickActions, req.QuickActions, id)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSONBP(w, map[string]string{"message": "Template updated"})
}

func (h *BPNotificationHandlers) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	_, err := h.db.Exec("UPDATE notification_templates SET is_active = false WHERE id = $1", id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSONBP(w, map[string]string{"message": "Template deactivated"})
}

func (h *BPNotificationHandlers) RenderTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Variables map[string]interface{} `json:"variables"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	var template BPNotificationTemplate
	err := h.db.Get(&template, "SELECT * FROM notification_templates WHERE id = $1", id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Template not found")
		return
	}

	subject := renderTemplateBP(template.SubjectTemplate, req.Variables)
	body := renderTemplateBP(template.BodyTemplate, req.Variables)

	respondJSONBP(w, map[string]string{
		"subject": subject,
		"body":    body,
	})
}

// ============================================================================
// SEND NOTIFICATION
// ============================================================================

func (h *BPNotificationHandlers) SendNotification(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	if datasourceID == "" {
		datasourceID = r.URL.Query().Get("datasource_id")
	}

	var req struct {
		TemplateKey       string                 `json:"template_key"`
		RecipientUserID   string                 `json:"recipient_user_id"`
		RecipientEmail    string                 `json:"recipient_email,omitempty"`
		Variables         map[string]interface{} `json:"variables"`
		Channel           string                 `json:"channel,omitempty"`
		Priority          string                 `json:"priority,omitempty"`
		ProcessID         string                 `json:"process_id,omitempty"`
		ProcessInstanceID string                 `json:"process_instance_id,omitempty"`
		StepID            string                 `json:"step_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get template
	var template BPNotificationTemplate
	err := h.db.Get(&template,
		"SELECT * FROM notification_templates WHERE tenant_id = $1 AND datasource_id = $2 AND template_key = $3 AND is_active = true",
		tenantID, datasourceID, req.TemplateKey)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Template not found")
		return
	}

	// Render template
	subject := renderTemplateBP(template.SubjectTemplate, req.Variables)
	body := renderTemplateBP(template.BodyTemplate, req.Variables)

	// Determine channel
	channel := req.Channel
	if channel == "" {
		channel = template.DefaultChannel
	}

	priority := req.Priority
	if priority == "" {
		priority = template.Priority
	}

	// Create notification log
	logID := uuid.New().String()
	now := time.Now()

	query := `INSERT INTO notification_logs 
	          (id, tenant_id, datasource_id, template_id, template_key, recipient_user_id,
	           recipient_email, subject, body, channel, status, priority,
	           process_id, process_instance_id, step_id, sent_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`

	_, err = h.db.Exec(query, logID, tenantID, datasourceID, template.ID, req.TemplateKey,
		req.RecipientUserID, sqlNullString(req.RecipientEmail), subject, body, channel, "sent",
		priority, sqlNullString(req.ProcessID), sqlNullString(req.ProcessInstanceID),
		sqlNullString(req.StepID), now)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSONBP(w, map[string]interface{}{
		"notification_id": logID,
		"status":          "sent",
		"channel":         channel,
		"subject":         subject,
	})
}

func (h *BPNotificationHandlers) SendBatchNotifications(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	if datasourceID == "" {
		datasourceID = r.URL.Query().Get("datasource_id")
	}

	var req struct {
		Notifications []struct {
			TemplateKey     string                 `json:"template_key"`
			RecipientUserID string                 `json:"recipient_user_id"`
			RecipientEmail  string                 `json:"recipient_email,omitempty"`
			Variables       map[string]interface{} `json:"variables"`
		} `json:"notifications"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	sentCount := 0
	results := []map[string]interface{}{}

	for _, notif := range req.Notifications {
		// Get template
		var template BPNotificationTemplate
		err := h.db.Get(&template,
			"SELECT * FROM notification_templates WHERE tenant_id = $1 AND datasource_id = $2 AND template_key = $3",
			tenantID, datasourceID, notif.TemplateKey)

		if err != nil {
			results = append(results, map[string]interface{}{
				"recipient": notif.RecipientUserID,
				"status":    "error",
				"error":     "Template not found",
			})
			continue
		}

		// Render and send
		subject := renderTemplateBP(template.SubjectTemplate, notif.Variables)
		body := renderTemplateBP(template.BodyTemplate, notif.Variables)

		logID := uuid.New().String()
		_, err = h.db.Exec(
			`INSERT INTO notification_logs 
			 (id, tenant_id, datasource_id, template_id, template_key, recipient_user_id,
			  recipient_email, subject, body, channel, status, priority, sent_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
			logID, tenantID, datasourceID, template.ID, notif.TemplateKey,
			notif.RecipientUserID, sqlNullString(notif.RecipientEmail), subject, body,
			template.DefaultChannel, "sent", template.Priority, time.Now())

		if err == nil {
			sentCount++
			results = append(results, map[string]interface{}{
				"recipient":       notif.RecipientUserID,
				"notification_id": logID,
				"status":          "sent",
			})
		} else {
			results = append(results, map[string]interface{}{
				"recipient": notif.RecipientUserID,
				"status":    "error",
				"error":     err.Error(),
			})
		}
	}

	respondJSONBP(w, map[string]interface{}{
		"sent_count": sentCount,
		"total":      len(req.Notifications),
		"results":    results,
	})
}

// ============================================================================
// PREFERENCES
// ============================================================================

func (h *BPNotificationHandlers) GetUserPreferences(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	if datasourceID == "" {
		datasourceID = r.URL.Query().Get("datasource_id")
	}
	userID := r.URL.Query().Get("user_id")

	var prefs BPUserNotificationPreferences
	err := h.db.Get(&prefs,
		"SELECT * FROM user_notification_preferences WHERE tenant_id = $1 AND datasource_id = $2 AND user_id = $3",
		tenantID, datasourceID, userID)

	if err != nil {
		// Return defaults
		prefs = BPUserNotificationPreferences{
			TenantID:       tenantID,
			DatasourceID:   datasourceID,
			UserID:         userID,
			EmailEnabled:   true,
			DigestMode:     "immediate",
			IncludeSummary: true,
			MinPriority:    "low",
		}
	}

	respondJSONBP(w, prefs)
}

func (h *BPNotificationHandlers) UpdateUserPreferences(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	if datasourceID == "" {
		datasourceID = r.URL.Query().Get("datasource_id")
	}

	var req BPUserNotificationPreferences
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	req.TenantID = tenantID
	req.DatasourceID = datasourceID

	if req.ID == "" {
		req.ID = uuid.New().String()
	}

	query := `INSERT INTO user_notification_preferences 
	          (id, tenant_id, datasource_id, user_id, email_enabled, email_address,
	           sms_enabled, phone_number, slack_enabled, slack_user_id, slack_webhook_url,
	           teams_enabled, teams_user_id, teams_webhook_url, push_enabled, push_token,
	           digest_mode, digest_time, digest_days, include_summary, include_full_details,
	           dnd_enabled, dnd_start_time, dnd_end_time, min_priority)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
	          ON CONFLICT (tenant_id, datasource_id, user_id) 
	          DO UPDATE SET 
	              email_enabled = EXCLUDED.email_enabled,
	              email_address = EXCLUDED.email_address,
	              sms_enabled = EXCLUDED.sms_enabled,
	              phone_number = EXCLUDED.phone_number,
	              slack_enabled = EXCLUDED.slack_enabled,
	              slack_user_id = EXCLUDED.slack_user_id,
	              slack_webhook_url = EXCLUDED.slack_webhook_url,
	              teams_enabled = EXCLUDED.teams_enabled,
	              teams_user_id = EXCLUDED.teams_user_id,
	              teams_webhook_url = EXCLUDED.teams_webhook_url,
	              push_enabled = EXCLUDED.push_enabled,
	              push_token = EXCLUDED.push_token,
	              digest_mode = EXCLUDED.digest_mode,
	              digest_time = EXCLUDED.digest_time,
	              digest_days = EXCLUDED.digest_days,
	              include_summary = EXCLUDED.include_summary,
	              include_full_details = EXCLUDED.include_full_details,
	              dnd_enabled = EXCLUDED.dnd_enabled,
	              dnd_start_time = EXCLUDED.dnd_start_time,
	              dnd_end_time = EXCLUDED.dnd_end_time,
	              min_priority = EXCLUDED.min_priority`

	_, err := h.db.Exec(query, req.ID, req.TenantID, req.DatasourceID, req.UserID,
		req.EmailEnabled, req.EmailAddress, req.SmsEnabled, req.PhoneNumber,
		req.SlackEnabled, req.SlackUserID, req.SlackWebhookURL,
		req.TeamsEnabled, req.TeamsUserID, req.TeamsWebhookURL,
		req.PushEnabled, req.PushToken, req.DigestMode, req.DigestTime, req.DigestDays,
		req.IncludeSummary, req.IncludeFullDetails, req.DndEnabled, req.DndStartTime,
		req.DndEndTime, req.MinPriority)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSONBP(w, map[string]string{"message": "Preferences updated"})
}

// ============================================================================
// LOGS & ANALYTICS
// ============================================================================

func (h *BPNotificationHandlers) GetLogs(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = jwtmiddleware.GetClaimsFromContext(r).TenantID
	}
	datasourceID := r.URL.Query().Get("datasource_id")
	if datasourceID == "" {
		datasourceID = r.URL.Query().Get("tenant_instance_id")
	}
	if datasourceID == "" {
		datasourceID = r.Header.Get("X-Tenant-Datasource-ID")
	}
	if datasourceID == "" {
		datasourceID = r.Header.Get("X-Tenant-Instance-ID")
	}

	if tenantID == "" {
		respondWithError(w, http.StatusBadRequest, "Missing tenant context")
		return
	}
	userID := r.URL.Query().Get("user_id")
	status := r.URL.Query().Get("status")
	processID := r.URL.Query().Get("process_id")

	query := `SELECT id, tenant_id, datasource_id, template_id, template_key, recipient_user_id, recipient_email, recipient_phone, subject, body, COALESCE(rendered_content, '{}'::jsonb) as rendered_content, channel, status, delivery_provider, sent_at, delivered_at, opened_at, clicked_at, action_taken, action_taken_at, error_message, retry_count, next_retry_at, process_id, process_instance_id, step_id, related_entity_type, related_entity_id, priority, is_digest, digest_batch_id, created_at, updated_at FROM notification_logs WHERE tenant_id = $1`
	args := []interface{}{tenantID}

	if datasourceID != "" {
		query += fmt.Sprintf(" AND datasource_id = $%d", len(args)+1)
		args = append(args, datasourceID)
	}

	if userID != "" {
		query += fmt.Sprintf(" AND recipient_user_id = $%d", len(args)+1)
		args = append(args, userID)
	}

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", len(args)+1)
		args = append(args, status)
	}

	if processID != "" {
		query += fmt.Sprintf(" AND process_id = $%d", len(args)+1)
		args = append(args, processID)
	}

	query += " ORDER BY created_at DESC LIMIT 100"

	logs := make([]BPNotificationLog, 0)
	if err := h.db.Select(&logs, query, args...); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSONBP(w, logs)
}

func (h *BPNotificationHandlers) GetLog(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var log BPNotificationLog
	err := h.db.Get(&log, "SELECT * FROM notification_logs WHERE id = $1", id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Log not found")
		return
	}

	respondJSONBP(w, log)
}

func (h *BPNotificationHandlers) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	if datasourceID == "" {
		datasourceID = r.URL.Query().Get("datasource_id")
	}

	var stats struct {
		TotalSent      int     `db:"total_sent"`
		TotalDelivered int     `db:"total_delivered"`
		TotalOpened    int     `db:"total_opened"`
		TotalClicked   int     `db:"total_clicked"`
		TotalFailed    int     `db:"total_failed"`
		DeliveryRate   float64 `db:"delivery_rate"`
		OpenRate       float64 `db:"open_rate"`
		ClickRate      float64 `db:"click_rate"`
	}

	query := `SELECT 
	            COUNT(*) FILTER (WHERE status IN ('sent', 'delivered')) as total_sent,
	            COUNT(*) FILTER (WHERE status = 'delivered') as total_delivered,
	            COUNT(*) FILTER (WHERE opened_at IS NOT NULL) as total_opened,
	            COUNT(*) FILTER (WHERE clicked_at IS NOT NULL) as total_clicked,
	            COUNT(*) FILTER (WHERE status = 'failed') as total_failed,
	            ROUND(COUNT(*) FILTER (WHERE status = 'delivered')::numeric / NULLIF(COUNT(*), 0) * 100, 2) as delivery_rate,
	            ROUND(COUNT(*) FILTER (WHERE opened_at IS NOT NULL)::numeric / NULLIF(COUNT(*) FILTER (WHERE status = 'delivered'), 0) * 100, 2) as open_rate,
	            ROUND(COUNT(*) FILTER (WHERE clicked_at IS NOT NULL)::numeric / NULLIF(COUNT(*) FILTER (WHERE opened_at IS NOT NULL), 0) * 100, 2) as click_rate
	          FROM notification_logs
	          WHERE tenant_id = $1 AND datasource_id = $2
	            AND created_at >= NOW() - INTERVAL '30 days'`

	err := h.db.Get(&stats, query, tenantID, datasourceID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSONBP(w, stats)
}

// ============================================================================
// DIGESTS
// ============================================================================

func (h *BPNotificationHandlers) GetPendingDigests(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	if datasourceID == "" {
		datasourceID = r.URL.Query().Get("datasource_id")
	}

	type Digest struct {
		ID                string    `json:"id" db:"id"`
		RecipientUserID   string    `json:"recipient_user_id" db:"recipient_user_id"`
		DigestPeriod      string    `json:"digest_period" db:"digest_period"`
		NotificationCount int       `json:"notification_count" db:"notification_count"`
		ScheduledSendAt   time.Time `json:"scheduled_send_at" db:"scheduled_send_at"`
		Status            string    `json:"status" db:"status"`
	}

	var digests []Digest
	err := h.db.Select(&digests,
		`SELECT id, recipient_user_id, digest_period, notification_count, scheduled_send_at, status 
		 FROM notification_digests 
		 WHERE tenant_id = $1 AND datasource_id = $2 AND status = 'pending' 
		 AND scheduled_send_at <= NOW()
		 ORDER BY scheduled_send_at`,
		tenantID, datasourceID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSONBP(w, digests)
}

func (h *BPNotificationHandlers) ProcessDigests(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	if datasourceID == "" {
		datasourceID = r.URL.Query().Get("datasource_id")
	}

	result, err := h.db.Exec(
		`UPDATE notification_digests 
		 SET status = 'sent', sent_at = NOW() 
		 WHERE tenant_id = $1 AND datasource_id = $2 
		   AND status = 'pending' AND scheduled_send_at <= NOW()`,
		tenantID, datasourceID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()

	respondJSONBP(w, map[string]interface{}{
		"message":      "Digests processed",
		"digests_sent": rowsAffected,
	})
}

// ============================================================================
// WEBHOOKS
// ============================================================================

func (h *BPNotificationHandlers) MarkDelivered(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	_, err := h.db.Exec(
		"UPDATE notification_logs SET status = 'delivered', delivered_at = NOW() WHERE id = $1",
		id)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSONBP(w, map[string]string{"message": "Marked as delivered"})
}

func (h *BPNotificationHandlers) MarkOpened(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	_, err := h.db.Exec(
		"UPDATE notification_logs SET opened_at = NOW() WHERE id = $1 AND opened_at IS NULL",
		id)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSONBP(w, map[string]string{"message": "Marked as opened"})
}

func (h *BPNotificationHandlers) MarkClicked(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	_, err := h.db.Exec(
		"UPDATE notification_logs SET clicked_at = NOW() WHERE id = $1 AND clicked_at IS NULL",
		id)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSONBP(w, map[string]string{"message": "Marked as clicked"})
}

func (h *BPNotificationHandlers) RecordAction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Action string `json:"action"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	_, err := h.db.Exec(
		"UPDATE notification_logs SET action_taken = $1, action_taken_at = NOW() WHERE id = $2",
		req.Action, id)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSONBP(w, map[string]string{
		"message": "Action recorded",
		"action":  req.Action,
	})
}

// ============================================================================
// HELPERS
// ============================================================================

func renderTemplateBP(template string, variables map[string]interface{}) string {
	result := template
	for key, value := range variables {
		placeholder := fmt.Sprintf("{%s}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

func sqlNullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func respondJSONBP(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

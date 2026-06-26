package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"calendar-service/internal/hasura"
	"calendar-service/internal/middleware"
	"calendar-service/internal/services"

	"github.com/sirupsen/logrus"
)

// ExportImportHandler handles data export/import API endpoints
type ExportImportHandler struct {
	hasuraClient *hasura.Client
	auditService services.AuditService
	logger       *logrus.Entry
}

// NewExportImportHandler creates a new export/import handler
func NewExportImportHandler(hc *hasura.Client, audit services.AuditService, logger *logrus.Entry) *ExportImportHandler {
	return &ExportImportHandler{
		hasuraClient: hc,
		auditService: audit,
		logger:       logger.WithField("handler", "export_import"),
	}
}

// ExportData exports user data
// @Summary Export user data
// @Tags settings
// @Produce json
// @Param request body ExportRequest true "Export request"
// @Success 200 file application/json
// @Router /api/v1/settings/export [post]
func (h *ExportImportHandler) ExportData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := middleware.ExtractTenantIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := middleware.ExtractUserIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Format           string `json:"format"`
		IncludeEvents    bool   `json:"include_events"`
		IncludeCalendars bool   `json:"include_calendars"`
		IncludeSettings  bool   `json:"include_settings"`
		StartDate        string `json:"start_date"`
		EndDate          string `json:"end_date"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	exportData := make(map[string]interface{})

	// Export calendars
	if req.IncludeCalendars {
		calendars, err := h.exportCalendars(ctx, tenantID, userID)
		if err != nil {
			h.logger.WithError(err).Error("Failed to export calendars")
		} else {
			exportData["calendars"] = calendars
		}
	}

	// Export events
	if req.IncludeEvents {
		events, err := h.exportEvents(ctx, tenantID, userID, req.StartDate, req.EndDate)
		if err != nil {
			h.logger.WithError(err).Error("Failed to export events")
		} else {
			exportData["events"] = events
		}
	}

	// Export settings
	if req.IncludeSettings {
		settings, err := h.exportSettings(ctx, userID)
		if err != nil {
			h.logger.WithError(err).Error("Failed to export settings")
		} else {
			exportData["settings"] = settings
		}
	}

	// Set content type based on format
	filename := "calendar-export-" + time.Now().Format("2006-01-02")

	switch req.Format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename+".csv")
		h.exportCSV(w, exportData)
	case "ics":
		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename+".ics")
		h.exportICS(w, exportData)
	default: // json
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename+".json")
		json.NewEncoder(w).Encode(exportData)
	}
}

func (h *ExportImportHandler) exportCalendars(ctx context.Context, tenantID, userID string) ([]map[string]interface{}, error) {
	query := `
	query ExportCalendars($tenant_id: uuid!) {
		calendars(
			where: {tenant_id: {_eq: $tenant_id}, valid_to: {_is_null: true}}
		) {
			id name description region priority holidays
			created_at updated_at
		}
	}
	`

	var result struct {
		Calendars []map[string]interface{} `json:"calendars"`
	}

	if err := h.hasuraClient.QueryRaw(ctx, query, map[string]interface{}{"tenant_id": tenantID}, &result); err != nil {
		return nil, err
	}

	return result.Calendars, nil
}

func (h *ExportImportHandler) exportEvents(ctx context.Context, tenantID, userID, startDate, endDate string) ([]map[string]interface{}, error) {
	query := `
	query ExportEvents($tenant_id: uuid!, $start: timestamptz, $end: timestamptz) {
		synced_google_events(
			where: {
				tenant_id: {_eq: $tenant_id},
				start_time: {_gte: $start, _lte: $end}
			}
		) {
			id google_event_id title description location
			start_time end_time is_all_day is_recurring
			created_at
		}
	}
	`

	var result struct {
		Events []map[string]interface{} `json:"synced_google_events"`
	}

	if err := h.hasuraClient.QueryRaw(ctx, query, map[string]interface{}{
		"tenant_id": tenantID,
		"start":     startDate,
		"end":       endDate,
	}, &result); err != nil {
		return nil, err
	}

	return result.Events, nil
}

func (h *ExportImportHandler) exportSettings(ctx context.Context, userID string) (map[string]interface{}, error) {
	query := `
	query ExportSettings($user_id: uuid!) {
		user_settings(
			where: {user_id: {_eq: $user_id}},
			limit: 1
		) {
			display_name email timezone language
			sync_frequency auto_sync_enabled
			email_notifications push_notifications
		}
	}
	`

	var result struct {
		Settings []map[string]interface{} `json:"user_settings"`
	}

	if err := h.hasuraClient.QueryRaw(ctx, query, map[string]interface{}{"user_id": userID}, &result); err != nil {
		return nil, err
	}

	if len(result.Settings) > 0 {
		return result.Settings[0], nil
	}

	return nil, nil
}

func (h *ExportImportHandler) exportCSV(w http.ResponseWriter, data map[string]interface{}) {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write events as CSV (most common use case)
	if events, ok := data["events"].([]map[string]interface{}); ok {
		writer.Write([]string{"id", "title", "description", "start_time", "end_time", "location", "is_all_day"})

		for _, event := range events {
			writer.Write([]string{
				getString(event, "id"),
				getString(event, "title"),
				getString(event, "description"),
				getString(event, "start_time"),
				getString(event, "end_time"),
				getString(event, "location"),
				getString(event, "is_all_day"),
			})
		}
	}
}

func (h *ExportImportHandler) exportICS(w http.ResponseWriter, data map[string]interface{}) {
	// Simple ICS export
	w.Write([]byte("BEGIN:VCALENDAR\r\n"))
	w.Write([]byte("VERSION:2.0\r\n"))
	w.Write([]byte("PRODID:-//Calendar Sync//EN\r\n"))

	if events, ok := data["events"].([]map[string]interface{}); ok {
		for _, event := range events {
			w.Write([]byte("BEGIN:VEVENT\r\n"))
			w.Write([]byte("UID:" + getString(event, "id") + "\r\n"))
			w.Write([]byte("DTSTART:" + formatICSDate(getString(event, "start_time")) + "\r\n"))
			w.Write([]byte("DTEND:" + formatICSDate(getString(event, "end_time")) + "\r\n"))
			w.Write([]byte("SUMMARY:" + getString(event, "title") + "\r\n"))
			w.Write([]byte("DESCRIPTION:" + getString(event, "description") + "\r\n"))
			w.Write([]byte("LOCATION:" + getString(event, "location") + "\r\n"))
			w.Write([]byte("END:VEVENT\r\n"))
		}
	}

	w.Write([]byte("END:VCALENDAR\r\n"))
}

// ImportData imports user data
// @Summary Import user data
// @Tags settings
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Import file"
// @Param merge_strategy formData string true "Merge strategy"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/settings/import [post]
func (h *ExportImportHandler) ImportData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := middleware.ExtractTenantIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := middleware.ExtractUserIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	mergeStrategy := r.FormValue("merge_strategy")
	if mergeStrategy == "" {
		mergeStrategy = "merge"
	}

	// Parse and import based on file type
	var importData struct {
		Calendars []map[string]interface{} `json:"calendars"`
		Events    []map[string]interface{} `json:"events"`
	}

	if err := json.NewDecoder(file).Decode(&importData); err != nil {
		h.logger.WithError(err).Error("Failed to decode import file")
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	importedCount := 0
	errorCount := 0
	var errors []string

	if len(importData.Calendars) > 0 {
		for _, cal := range importData.Calendars {
			// Mutation to insert or upsert calendar
			mutation := `
			mutation ImportCalendar($object: calendars_insert_input!) {
				insert_calendars_one(object: $object, on_conflict: {constraint: calendars_pkey, update_columns: [name, description]}) {
					id
				}
			}
			`
			// Ensure it has required fields for the user/tenant
			cal["tenant_id"] = tenantID
			if _, ok := cal["id"]; !ok {
				// Generate UUID if missing
			}

			if err := h.hasuraClient.QueryRaw(ctx, mutation, map[string]interface{}{"object": cal}, nil); err != nil {
				h.logger.WithError(err).Error("Failed to import calendar")
				errors = append(errors, fmt.Sprintf("Calendar import failed: %v", err))
				errorCount++
			} else {
				importedCount++
			}
		}
	}

	if len(importData.Events) > 0 {
		for _, event := range importData.Events {
			mutation := `
			mutation ImportEvent($object: synced_google_events_insert_input!) {
				insert_synced_google_events_one(object: $object, on_conflict: {constraint: synced_google_events_pkey, update_columns: [title, description, start_time, end_time]}) {
					id
				}
			}
			`
			event["tenant_id"] = tenantID
			if err := h.hasuraClient.QueryRaw(ctx, mutation, map[string]interface{}{"object": event}, nil); err != nil {
				h.logger.WithError(err).Error("Failed to import event")
				errors = append(errors, fmt.Sprintf("Event import failed: %v", err))
				errorCount++
			} else {
				importedCount++
			}
		}
	}

	// Audit log
	_ = h.auditService.Record(ctx, services.AuditEntry{
		TenantID:   tenantID,
		EntityType: "data_import",
		EntityID:   userID,
		Action:     "IMPORT",
		ChangedBy:  userID,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"imported": importedCount,
		"skipped":  0,
		"errors":   errors,
	})
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func formatICSDate(dateStr string) string {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format("20060102T150405Z")
}

package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"calendar-service/internal/database"
	"calendar-service/internal/middleware"
	"calendar-service/internal/services"

	"github.com/sirupsen/logrus"
)

type ExportImportHandler struct {
	dbClient     *database.Client
	auditService services.AuditService
	logger       *logrus.Entry
}

func NewExportImportHandler(db *database.Client, audit services.AuditService, logger *logrus.Entry) *ExportImportHandler {
	return &ExportImportHandler{
		dbClient:     db,
		auditService: audit,
		logger:       logger.WithField("handler", "export_import"),
	}
}

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

	if req.IncludeCalendars {
		calendars, err := h.exportCalendars(ctx, tenantID, userID)
		if err != nil {
			h.logger.WithError(err).Error("Failed to export calendars")
		} else {
			exportData["calendars"] = calendars
		}
	}

	if req.IncludeEvents {
		events, err := h.exportEvents(ctx, tenantID, userID, req.StartDate, req.EndDate)
		if err != nil {
			h.logger.WithError(err).Error("Failed to export events")
		} else {
			exportData["events"] = events
		}
	}

	if req.IncludeSettings {
		settings, err := h.exportSettings(ctx, userID)
		if err != nil {
			h.logger.WithError(err).Error("Failed to export settings")
		} else {
			exportData["settings"] = settings
		}
	}

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
	default:
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename+".json")
		json.NewEncoder(w).Encode(exportData)
	}
}

func (h *ExportImportHandler) exportCalendars(ctx context.Context, tenantID, userID string) ([]map[string]interface{}, error) {
	query := `
		SELECT id, name, description, region, priority, holidays, created_at, updated_at
		FROM calendars
		WHERE tenant_id = $1 AND valid_to IS NULL
	`

	rows, err := h.dbClient.Pool().Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calendars []map[string]interface{}
	for rows.Next() {
		var id, name, description, region, holidays string
		var priority int
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &name, &description, &region, &priority, &holidays, &createdAt, &updatedAt); err != nil {
			continue
		}
		calendars = append(calendars, map[string]interface{}{
			"id":          id,
			"name":        name,
			"description": description,
			"region":      region,
			"priority":    priority,
			"holidays":    holidays,
			"created_at":  createdAt,
			"updated_at":  updatedAt,
		})
	}
	return calendars, nil
}

func (h *ExportImportHandler) exportEvents(ctx context.Context, tenantID, userID, startDate, endDate string) ([]map[string]interface{}, error) {
	query := `
		SELECT id, google_event_id, title, description, location, start_time, end_time, is_all_day, is_recurring, created_at
		FROM synced_google_events
		WHERE tenant_id = $1 AND start_time >= $2 AND end_time <= $3
	`

	rows, err := h.dbClient.Pool().Query(ctx, query, tenantID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []map[string]interface{}
	for rows.Next() {
		var id, googleEventID, title, description, location string
		var startTime, endTime time.Time
		var isAllDay, isRecurring bool
		var createdAt time.Time
		if err := rows.Scan(&id, &googleEventID, &title, &description, &location, &startTime, &endTime, &isAllDay, &isRecurring, &createdAt); err != nil {
			continue
		}
		events = append(events, map[string]interface{}{
			"id":              id,
			"google_event_id": googleEventID,
			"title":           title,
			"description":     description,
			"location":        location,
			"start_time":      startTime,
			"end_time":        endTime,
			"is_all_day":      isAllDay,
			"is_recurring":    isRecurring,
			"created_at":      createdAt,
		})
	}
	return events, nil
}

func (h *ExportImportHandler) exportSettings(ctx context.Context, userID string) (map[string]interface{}, error) {
	query := `
		SELECT display_name, email, timezone, language, sync_frequency, auto_sync_enabled, email_notifications, push_notifications
		FROM user_settings
		WHERE user_id = $1
	`

	var settings map[string]interface{}
	var displayName, email, timezone, language, syncFreq string
	var autoSync, emailNotif, pushNotif bool

	err := h.dbClient.Pool().QueryRow(ctx, query, userID).Scan(
		&displayName, &email, &timezone, &language, &syncFreq, &autoSync, &emailNotif, &pushNotif,
	)

	if err != nil {
		return nil, err
	}

	settings = map[string]interface{}{
		"display_name":        displayName,
		"email":               email,
		"timezone":            timezone,
		"language":            language,
		"sync_frequency":      syncFreq,
		"auto_sync_enabled":   autoSync,
		"email_notifications": emailNotif,
		"push_notifications":  pushNotif,
	}

	return settings, nil
}

func (h *ExportImportHandler) exportCSV(w http.ResponseWriter, data map[string]interface{}) {
	writer := csv.NewWriter(w)
	defer writer.Flush()

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

	if err := r.ParseMultipartForm(10 << 20); err != nil {
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
			cal["tenant_id"] = tenantID
			query := `
				INSERT INTO calendars (id, tenant_id, name, description, region, priority, holidays)
				VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)
				ON CONFLICT DO NOTHING
			`
			_, err := h.dbClient.Pool().Exec(ctx, query,
				tenantID, cal["name"], cal["description"], cal["region"], cal["priority"], cal["holidays"],
			)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Calendar import failed: %v", err))
				errorCount++
			} else {
				importedCount++
			}
		}
	}

	if len(importData.Events) > 0 {
		for _, event := range importData.Events {
			event["tenant_id"] = tenantID
			query := `
				INSERT INTO synced_google_events (id, tenant_id, google_event_id, title, description, start_time, end_time)
				VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)
				ON CONFLICT DO NOTHING
			`
			_, err := h.dbClient.Pool().Exec(ctx, query,
				tenantID, event["google_event_id"], event["title"], event["description"], event["start_time"], event["end_time"],
			)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Event import failed: %v", err))
				errorCount++
			} else {
				importedCount++
			}
		}
	}

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

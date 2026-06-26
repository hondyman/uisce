package mdm

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"calendar-service/internal/publisher"
	"calendar-service/internal/rules"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// API Handlers (Usice Architecture Section 2.2: API Gateway)
// ============================================================================

type Handler struct {
	orchestrator *IngestionOrchestrator
	rulesEngine  *rules.RulesEngine
	publisher    *publisher.RedpandaPublisher
	logger       *logrus.Entry
}

func NewHandler(
	orchestrator *IngestionOrchestrator,
	rulesEngine *rules.RulesEngine,
	publisher *publisher.RedpandaPublisher,
	logger *logrus.Entry,
) *Handler {
	return &Handler{
		orchestrator: orchestrator,
		rulesEngine:  rulesEngine,
		publisher:    publisher,
		logger:       logger,
	}
}

// ============================================================================
// Ingestion Endpoints
// ============================================================================

// HandleTriggerIngestion triggers a manual ingestion cycle
// POST /api/v1/mdm/calendar/ingest
func (h *Handler) HandleTriggerIngestion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req struct {
		TenantID string   `json:"tenant_id"`
		Regions  []string `json:"regions"`
		Year     int      `json:"year"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Error("Failed to decode request")
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	if len(req.Regions) == 0 || req.Year == 0 {
		http.Error(w, "Missing regions or year", http.StatusBadRequest)
		return
	}

	// Publish ingestion started event
	h.publisher.PublishIngestionStarted(ctx, tenantID, req.Regions, []string{})

	// Trigger ingestion in background
	go func() {
		if err := h.orchestrator.RunIngestionCycle(ctx, tenantID, req.Regions, req.Year); err != nil {
			h.logger.WithError(err).Error("Ingestion cycle failed")
			h.publisher.PublishIngestionCompleted(ctx, tenantID, 0, 0, false)
		} else {
			h.publisher.PublishIngestionCompleted(ctx, tenantID, 100, 5000, true)
		}
	}()

	// Respond immediately
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ACCEPTED",
		"message": "Ingestion cycle triggered",
		"regions": req.Regions,
		"year":    req.Year,
	})
}

// ============================================================================
// Source Management Endpoints
// ============================================================================

// HandleListSources returns all configured sources
// GET /api/v1/mdm/sources
func (h *Handler) HandleListSources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sources, err := h.orchestrator.getActiveSources(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list sources")
		http.Error(w, "Failed to fetch sources", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sources": sources,
		"count":   len(sources),
	})
}

// HandleActivateSource activates a data source
// PATCH /api/v1/mdm/sources/{source_id}/activate
func (h *Handler) HandleActivateSource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sourceID := r.PathValue("source_id")
	if sourceID == "" {
		http.Error(w, "Missing source ID", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(sourceID)
	if err != nil {
		http.Error(w, "Invalid source ID", http.StatusBadRequest)
		return
	}

	// Update source status in database
	query := `UPDATE edm.mdm_source_registry SET is_active = true WHERE id = $1`
	_, err = h.orchestrator.db.ExecContext(ctx, query, id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to activate source")
		http.Error(w, "Failed to activate source", http.StatusInternalServerError)
		return
	}

	// Publish event
	h.publisher.PublishSourceActivation(ctx, sourceID, true)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ACTIVATED",
		"sourceID": sourceID,
	})
}

// HandleDeactivateSource deactivates a data source
// PATCH /api/v1/mdm/sources/{source_id}/deactivate
func (h *Handler) HandleDeactivateSource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sourceID := r.PathValue("source_id")
	if sourceID == "" {
		http.Error(w, "Missing source ID", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(sourceID)
	if err != nil {
		http.Error(w, "Invalid source ID", http.StatusBadRequest)
		return
	}

	// Update source status
	query := `UPDATE edm.mdm_source_registry SET is_active = false WHERE id = $1`
	_, err = h.orchestrator.db.ExecContext(ctx, query, id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to deactivate source")
		http.Error(w, "Failed to deactivate source", http.StatusInternalServerError)
		return
	}

	// Publish event
	h.publisher.PublishSourceActivation(ctx, sourceID, false)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "DEACTIVATED",
		"sourceID": sourceID,
	})
}

// ============================================================================
// Query Endpoints (Calendar Consumption)
// ============================================================================

// HandleGetGoldenCalendar retrieves the golden calendar for a region/date range
// GET /api/v1/calendar/golden?region=US&start_date=2026-01-01&end_date=2026-12-31
func (h *Handler) HandleGetGoldenCalendar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract tenant from JWT (middleware sets this)
	tenantID := getTenantIDFromContext(r.Context())
	region := r.URL.Query().Get("region")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if region == "" || startDate == "" || endDate == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Query golden records from database
	query := `
		SELECT calendar_date, is_business_day, holiday_name, confidence_score, source_system
		FROM edm.mdm_calendar_golden
		WHERE tenant_id = $1
		AND region_code = $2
		AND calendar_date BETWEEN $3::date AND $4::date
		ORDER BY calendar_date ASC
	`

	rows, err := h.orchestrator.db.QueryContext(ctx, query, tenantID, region, startDate, endDate)
	if err != nil {
		h.logger.WithError(err).Error("Failed to query golden calendar")
		http.Error(w, "Failed to query calendar", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type CalendarDay struct {
		Date            string `json:"date"`
		IsBusinessDay   bool   `json:"is_business_day"`
		HolidayName     string `json:"holiday_name"`
		ConfidenceScore int    `json:"confidence_score"`
		SourceSystem    string `json:"source_system"`
	}

	var days []CalendarDay
	for rows.Next() {
		var day CalendarDay
		if err := rows.Scan(&day.Date, &day.IsBusinessDay, &day.HolidayName, &day.ConfidenceScore, &day.SourceSystem); err != nil {
			h.logger.WithError(err).Error("Failed to scan row")
			continue
		}
		days = append(days, day)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id":  tenantID,
		"region":     region,
		"start_date": startDate,
		"end_date":   endDate,
		"days":       days,
		"count":      len(days),
	})
}

// HandleIsBusinessDay checks if a specific date is a business day
// GET /api/v1/calendar/is-business-day?region=US&date=2026-01-15
func (h *Handler) HandleIsBusinessDay(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := getTenantIDFromContext(r.Context())
	region := r.URL.Query().Get("region")
	dateStr := r.URL.Query().Get("date")

	if region == "" || dateStr == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	query := `
		SELECT is_business_day, holiday_name
		FROM edm.mdm_calendar_golden
		WHERE tenant_id = $1 AND region_code = $2 AND calendar_date = $3::date
	`

	var isBusinessDay bool
	var holidayName *string

	err := h.orchestrator.db.QueryRowContext(ctx, query, tenantID, region, dateStr).Scan(&isBusinessDay, &holidayName)
	if err != nil {
		// Default to true if no record (safe default)
		isBusinessDay = true
		w.WriteHeader(http.StatusNotFound)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"date":            dateStr,
		"region":          region,
		"is_business_day": isBusinessDay,
		"holiday_name":    holidayName,
	})
}

// ============================================================================
// Stewardship/Ops Endpoints
// ============================================================================

// HandleGetConflicts returns all conflicts requiring manual review
// GET /api/v1/mdm/conflicts?tenant_id=xxx
func (h *Handler) HandleGetConflicts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "Missing tenant_id", http.StatusBadRequest)
		return
	}

	query := `
		SELECT id, golden_record_id, issue_type, description, conflicting_sources, status, created_at
		FROM edm.mdm_stewardship_queue
		WHERE tenant_id = $1 AND status = 'PENDING'
		ORDER BY priority ASC, created_at DESC
	`

	rows, err := h.orchestrator.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to query conflicts")
		http.Error(w, "Failed to fetch conflicts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Conflict struct {
		ID                 string    `json:"id"`
		GoldenRecordID     string    `json:"golden_record_id"`
		IssueType          string    `json:"issue_type"`
		Description        string    `json:"description"`
		ConflictingSources []string  `json:"conflicting_sources"`
		Status             string    `json:"status"`
		CreatedAt          time.Time `json:"created_at"`
	}

	var conflicts []Conflict
	for rows.Next() {
		var c Conflict
		var sources []byte
		if err := rows.Scan(&c.ID, &c.GoldenRecordID, &c.IssueType, &c.Description, &sources, &c.Status, &c.CreatedAt); err != nil {
			h.logger.WithError(err).Error("Failed to scan conflict")
			continue
		}
		json.Unmarshal(sources, &c.ConflictingSources)
		conflicts = append(conflicts, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"conflicts": conflicts,
		"count":     len(conflicts),
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

func getTenantIDFromContext(ctx context.Context) string {
	// In production, this would extract tenant_id from JWT claims via middleware
	// For now, return a test value
	return "00000000-0000-0000-0000-000000000001"
}

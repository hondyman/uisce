package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"calendar-service/internal/database"
	"calendar-service/internal/middleware"

	"github.com/sirupsen/logrus"
)

type AnalyticsHandler struct {
	dbClient *database.Client
	logger   *logrus.Entry
}

func NewAnalyticsHandler(dc *database.Client, logger *logrus.Entry) *AnalyticsHandler {
	return &AnalyticsHandler{
		dbClient: dc,
		logger:   logger.WithField("handler", "analytics"),
	}
}

// SyncAnalyticsResponse represents sync analytics response
type SyncAnalyticsResponse struct {
	Date          string  `json:"date"`
	TotalSyncs    int     `json:"total_syncs"`
	Successful    int     `json:"successful"`
	Failed        int     `json:"failed"`
	SuccessRate   float64 `json:"success_rate"`
	AvgDuration   float64 `json:"avg_duration_seconds"`
	TotalEvents   int     `json:"total_events_synced"`
	AvgEventsSync int     `json:"avg_events_per_sync"`
}

// GetSyncAnalytics returns sync analytics
func (h *AnalyticsHandler) GetSyncAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := middleware.ExtractTenantIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	sqlQuery := `
		SELECT date, total_syncs, successful_syncs, failed_syncs,
			success_rate, avg_duration_seconds, total_events_synced, avg_events_per_sync
		FROM sync_daily_stats
		WHERE tenant_id = $1 AND date >= $2 AND date <= $3
		ORDER BY date ASC`

	rows, err := h.dbClient.Pool().Query(ctx, sqlQuery, tenantID, startDate, endDate)
	if err != nil {
		http.Error(w, "Failed to get analytics", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var stats []SyncAnalyticsResponse
	for rows.Next() {
		var s SyncAnalyticsResponse
		if err := rows.Scan(&s.Date, &s.TotalSyncs, &s.Successful, &s.Failed, &s.SuccessRate, &s.AvgDuration, &s.TotalEvents, &s.AvgEventsSync); err != nil {
			continue
		}
		stats = append(stats, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  stats,
		"count": len(stats),
	})
}

// ConflictAnalyticsResponse represents conflict analytics
type ConflictAnalyticsResponse struct {
	Date            string  `json:"date"`
	ConflictType    string  `json:"conflict_type"`
	Severity        string  `json:"severity"`
	TotalConflicts  int     `json:"total_conflicts"`
	Resolved        int     `json:"resolved"`
	ResolutionRate  float64 `json:"resolution_rate"`
	AutoResolved    int     `json:"auto_resolved"`
	UserOverrides   int     `json:"user_overrides"`
	AvgMLConfidence float64 `json:"avg_ml_confidence"`
}

// GetConflictAnalytics returns conflict analytics
func (h *AnalyticsHandler) GetConflictAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := middleware.ExtractTenantIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sqlQuery := `
		SELECT date, conflict_type, severity, total_conflicts, resolved_conflicts,
			resolution_rate, auto_resolved, user_overrides, avg_ml_confidence
		FROM conflict_stats
		WHERE tenant_id = $1
		ORDER BY date DESC
		LIMIT 30`

	rows, err := h.dbClient.Pool().Query(ctx, sqlQuery, tenantID)
	if err != nil {
		http.Error(w, "Failed to get analytics", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var stats []ConflictAnalyticsResponse
	for rows.Next() {
		var s ConflictAnalyticsResponse
		if err := rows.Scan(&s.Date, &s.ConflictType, &s.Severity, &s.TotalConflicts, &s.Resolved, &s.ResolutionRate, &s.AutoResolved, &s.UserOverrides, &s.AvgMLConfidence); err != nil {
			continue
		}
		stats = append(stats, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  stats,
		"count": len(stats),
	})
}

// ExecutiveDashboardResponse represents executive dashboard metrics
type ExecutiveDashboardResponse struct {
	Metric string `json:"metric"`
	Value  string `json:"value"`
	Trend  *int   `json:"trend,omitempty"` // Percentage change
}

// GetExecutiveDashboard returns executive dashboard metrics
func (h *AnalyticsHandler) GetExecutiveDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sqlQuery := `SELECT metric, value, new_this_week FROM executive_dashboard`

	rows, err := h.dbClient.Pool().Query(ctx, sqlQuery)
	if err != nil {
		http.Error(w, "Failed to get dashboard", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var metrics []ExecutiveDashboardResponse
	for rows.Next() {
		var m ExecutiveDashboardResponse
		var newThisWeek bool
		if err := rows.Scan(&m.Metric, &m.Value, &newThisWeek); err != nil {
			continue
		}
		metrics = append(metrics, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"metrics": metrics,
	})
}

// UserCohortResponse represents cohort analysis data
type UserCohortResponse struct {
	CohortMonth   string  `json:"cohort_month"`
	WeekNumber    int     `json:"week_number"`
	UsersInCohort int     `json:"users_in_cohort"`
	ActiveUsers   int     `json:"active_users"`
	RetentionRate float64 `json:"retention_rate"`
}

// GetUserCohorts returns user cohort analysis
func (h *AnalyticsHandler) GetUserCohorts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sqlQuery := `SELECT cohort_month, week_number, users_in_cohort, active_users, retention_rate FROM user_cohorts ORDER BY cohort_month DESC`

	rows, err := h.dbClient.Pool().Query(ctx, sqlQuery)
	if err != nil {
		http.Error(w, "Failed to get cohorts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var cohorts []UserCohortResponse
	for rows.Next() {
		var c UserCohortResponse
		if err := rows.Scan(&c.CohortMonth, &c.WeekNumber, &c.UsersInCohort, &c.ActiveUsers, &c.RetentionRate); err != nil {
			continue
		}
		cohorts = append(cohorts, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"cohorts": cohorts,
	})
}

// ExportAnalyticsRequest represents export request
type ExportAnalyticsRequest struct {
	Format        string `json:"format"`
	ReportType    string `json:"report_type"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
	IncludeCharts bool   `json:"include_charts"`
}

// ExportAnalytics exports analytics data
func (h *AnalyticsHandler) ExportAnalytics(w http.ResponseWriter, r *http.Request) {
	var req ExportAnalyticsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=analytics-export."+req.Format)
	w.Write([]byte("Export data here"))
}

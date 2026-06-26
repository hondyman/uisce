package api

import (
	"encoding/json"
	"net/http"
	"time"

	"calendar-service/internal/hasura"
	"calendar-service/internal/middleware"

	"github.com/sirupsen/logrus"
)

// AnalyticsHandler handles analytics API endpoints
type AnalyticsHandler struct {
	hasuraClient *hasura.Client
	logger       *logrus.Entry
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(hc *hasura.Client, logger *logrus.Entry) *AnalyticsHandler {
	return &AnalyticsHandler{
		hasuraClient: hc,
		logger:       logger.WithField("handler", "analytics"),
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

	query := `
    query GetSyncAnalytics($tenant_id: uuid!, $start_date: date!, $end_date: date!) {
        sync_daily_stats(
            where: {
                tenant_id: {_eq: $tenant_id},
                date: {_gte: $start_date, _lte: $end_date}
            },
            order_by: {date: asc}
        ) {
            date total_syncs successful_syncs failed_syncs 
            success_rate avg_duration_seconds total_events_synced avg_events_per_sync
        }
    }
    `

	var result struct {
		Stats []SyncAnalyticsResponse `json:"sync_daily_stats"`
	}

	if err := h.hasuraClient.QueryRaw(ctx, query, map[string]interface{}{
		"tenant_id":  tenantID,
		"start_date": startDate,
		"end_date":   endDate,
	}, &result); err != nil {
		http.Error(w, "Failed to get analytics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  result.Stats,
		"count": len(result.Stats),
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

	query := `
    query GetConflictAnalytics($tenant_id: uuid!) {
        conflict_stats(
            where: {tenant_id: {_eq: $tenant_id}},
            order_by: {date: desc},
            limit: 30
        ) {
            date conflict_type severity total_conflicts resolved_conflicts
            resolution_rate auto_resolved user_overrides avg_ml_confidence
        }
    }
    `

	var result struct {
		Stats []ConflictAnalyticsResponse `json:"conflict_stats"`
	}

	if err := h.hasuraClient.QueryRaw(ctx, query, map[string]interface{}{
		"tenant_id": tenantID,
	}, &result); err != nil {
		http.Error(w, "Failed to get analytics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  result.Stats,
		"count": len(result.Stats),
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

	query := `
    query GetExecutiveDashboard {
        executive_dashboard {
            metric value new_this_week
        }
    }
    `

	var result struct {
		Metrics []ExecutiveDashboardResponse `json:"executive_dashboard"`
	}

	if err := h.hasuraClient.QueryRaw(ctx, query, nil, &result); err != nil {
		http.Error(w, "Failed to get dashboard", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"metrics": result.Metrics,
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

	query := `
    query GetUserCohorts {
        user_cohorts(order_by: {cohort_month: desc}) {
            cohort_month week_number users_in_cohort active_users retention_rate
        }
    }
    `

	var result struct {
		Cohorts []UserCohortResponse `json:"user_cohorts"`
	}

	if err := h.hasuraClient.QueryRaw(ctx, query, nil, &result); err != nil {
		http.Error(w, "Failed to get cohorts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"cohorts": result.Cohorts,
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

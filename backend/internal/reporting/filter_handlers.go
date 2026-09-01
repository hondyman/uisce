package reporting

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

// FilterHandler manages report filter persistence and tenant defaults.
type FilterHandler struct {
	db *sqlx.DB
}

// NewFilterHandler creates a new FilterHandler.
func NewFilterHandler(db *sqlx.DB) *FilterHandler {
	return &FilterHandler{db: db}
}

// GetFilters handles GET /api/reports/{id}/filters.
// Returns the filter model and pre-compiled WHERE clause for a report.
func (h *FilterHandler) GetFilters(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	reportID := chi.URLParam(r, "id")
	if reportID == "" {
		http.Error(w, "report id required", http.StatusBadRequest)
		return
	}

	var model FilterModel
	var compiled sql.NullString
	err := h.db.QueryRowxContext(r.Context(),
		`SELECT filter_model, COALESCE(compiled_where, '') FROM public.report_filters
		 WHERE tenant_id = $1 AND report_id = $2
		 ORDER BY updated_at DESC LIMIT 1`,
		claims.TenantID, reportID).Scan(&model, &compiled)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"model":          FilterModel{Groups: []FilterGroup{}, GroupCombinator: "AND"},
			"compiledWhere": "",
		})
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"model":          model,
		"compiledWhere":   compiled.String,
	})
}

// UpsertFilters handles POST /api/reports/{id}/filters.
// Persists a filter model and returns the compiled SQL WHERE clause.
func (h *FilterHandler) UpsertFilters(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	reportID := chi.URLParam(r, "id")

	var req struct {
		Model      FilterModel       `json:"model"`
		Parameters []ReportParameter `json:"parameters"`
		Defaults   TenantDefaults    `json:"defaults"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	paramsMap := map[string]interface{}{}
	for _, p := range req.Parameters {
		if p.DefaultValue != "" {
			paramsMap[p.Name] = p.DefaultValue
		}
	}
	compiled := CompileFilterModel(&req.Model, paramsMap, &req.Defaults)
	modelJSON, _ := json.Marshal(req.Model)

	// report_id is nullable — we store by the report UUID
	reportUUID, _ := uuid.Parse(reportID)
	_, err := h.db.ExecContext(r.Context(),
		`INSERT INTO public.report_filters (tenant_id, report_id, filter_model, compiled_where, updated_at, updated_by)
		 VALUES ($1, $2, $3, $4, NOW(), $5)
		 ON CONFLICT (tenant_id, report_id)
		 DO UPDATE SET filter_model = $3, compiled_where = $4, updated_at = NOW(), updated_by = $5`,
		claims.TenantID, reportUUID, modelJSON, compiled, claims.UserID)
	if err != nil {
		http.Error(w, "failed to save filters: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":            true,
		"compiledWhere": compiled,
	})
}

// GetTenantDefaults handles GET /api/tenants/{id}/defaults.
// Returns the tenant's default calendar code, fiscal year, and region.
func (h *FilterHandler) GetTenantDefaults(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := chi.URLParam(r, "id")
	if tenantID == "" {
		tenantID = claims.TenantID
	}

	var defaults TenantDefaults
	var calCode sql.NullString
	var fiscalYear sql.NullInt64
	var region sql.NullString

	err := h.db.QueryRowxContext(r.Context(),
		`SELECT default_calendar_code, default_fiscal_year, default_region
		 FROM public.tenants WHERE id = $1`,
		tenantID).Scan(&calCode, &fiscalYear, &region)
	if err == sql.ErrNoRows || !calCode.Valid {
		defaults = TenantDefaults{
			DefaultCalendarCode: "US",
			DefaultFiscalYear:   2026,
			DefaultRegion:       "us-east-1",
		}
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		defaults = TenantDefaults{
			DefaultCalendarCode: calCode.String,
			DefaultFiscalYear:   int(fiscalYear.Int64),
			DefaultRegion:       region.String,
		}
		if defaults.DefaultCalendarCode == "" {
			defaults.DefaultCalendarCode = "US"
		}
		if defaults.DefaultRegion == "" {
			defaults.DefaultRegion = "us-east-1"
		}
	}

	writeJSON(w, http.StatusOK, defaults)
}

// ListTenantCalendars handles GET /api/tenants/{id}/calendars.
// Returns all active calendars configured for a tenant.
func (h *FilterHandler) ListTenantCalendars(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := chi.URLParam(r, "id")
	if tenantID == "" {
		tenantID = claims.TenantID
	}

	rows, err := h.db.QueryxContext(r.Context(),
		`SELECT calendar_code, calendar_name, is_active
		 FROM public.tenant_exchange_calendars
		 WHERE tenant_id = $1 AND is_active = TRUE
		 ORDER BY calendar_code`,
		tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	calendars := []TenantCalendar{}
	for rows.Next() {
		var c TenantCalendar
		if err := rows.Scan(&c.Code, &c.Name, &c.Active); err != nil {
			continue
		}
		calendars = append(calendars, c)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"calendars": calendars})
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

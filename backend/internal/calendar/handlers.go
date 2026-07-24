package calendar

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler handles calendar HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new calendar handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// IsBusinessDay checks if a date is a business day
// GET /api/calendar/{code}/is-business-day?date=2025-01-15
func (h *Handler) IsBusinessDay(w http.ResponseWriter, r *http.Request) {
	calendarCode := chi.URLParam(r, "code")
	dateStr := r.URL.Query().Get("date")

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid date format. Use YYYY-MM-DD"})
		return
	}

	isBusiness, err := h.service.IsBusinessDay(r.Context(), calendarCode, date)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"calendar_code":   calendarCode,
		"date":            dateStr,
		"is_business_day": isBusiness,
	})
}

// NextBusinessDay finds next business day
// GET /api/calendar/{code}/next-business-day?from=2025-01-15
func (h *Handler) NextBusinessDay(w http.ResponseWriter, r *http.Request) {
	calendarCode := chi.URLParam(r, "code")
	fromStr := r.URL.Query().Get("from")

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid date format"})
		return
	}

	nextDay, err := h.service.NextBusinessDay(r.Context(), calendarCode, from)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"from_date":         fromStr,
		"next_business_day": nextDay.Format("2006-01-02"),
	})
}

// AddBusinessDays adds N business days
// GET /api/calendar/{code}/add-business-days?from=2025-01-15&days=5
func (h *Handler) AddBusinessDays(w http.ResponseWriter, r *http.Request) {
	calendarCode := chi.URLParam(r, "code")
	fromStr := r.URL.Query().Get("from")
	daysStr := r.URL.Query().Get("days")

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid date format"})
		return
	}

	var days int
	if _, err := fmt.Sscanf(daysStr, "%d", &days); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid days parameter"})
		return
	}

	result, err := h.service.AddBusinessDays(r.Context(), calendarCode, from, days)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"from_date":   fromStr,
		"days_added":  days,
		"result_date": result.Format("2006-01-02"),
	})
}

// AdjustDate adjusts date per convention
// POST /api/calendar/{code}/adjust-date
func (h *Handler) AdjustDate(w http.ResponseWriter, r *http.Request) {
	calendarCode := chi.URLParam(r, "code")

	var req struct {
		Date       string               `json:"date"`
		Convention AdjustmentConvention `json:"convention"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid date format"})
		return
	}

	adjusted, err := h.service.AdjustDate(r.Context(), calendarCode, date, req.Convention)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"original_date": req.Date,
		"convention":    req.Convention,
		"adjusted_date": adjusted.Format("2006-01-02"),
	})
}

// ListCalendars returns available calendars
// GET /api/calendar
func (h *Handler) ListCalendars(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := r.URL.Query().Get("tenant_id")
	var tenantID *uuid.UUID

	if tenantIDStr != "" {
		tid, err := uuid.Parse(tenantIDStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid tenant_id"})
			return
		}
		tenantID = &tid
	}

	calendars, err := h.service.ListCalendars(r.Context(), tenantID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"calendars": calendars,
		"total":     len(calendars),
	})
}

// GetHolidays returns holidays in a date range
// GET /api/calendar/{code}/holidays?start=2025-01-01&end=2025-12-31
func (h *Handler) GetHolidays(w http.ResponseWriter, r *http.Request) {
	calendarCode := chi.URLParam(r, "code")
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid start date"})
		return
	}

	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid end date"})
		return
	}

	holidays, err := h.service.GetHolidays(r.Context(), calendarCode, start, end)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"calendar_code": calendarCode,
		"start_date":    startStr,
		"end_date":      endStr,
		"holidays":      holidays,
		"total":         len(holidays),
	})
}

// RegisterRoutes registers calendar routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/calendar", func(r chi.Router) {
		r.Get("/", h.ListCalendars)
		r.Get("/{code}/is-business-day", h.IsBusinessDay)
		r.Get("/{code}/next-business-day", h.NextBusinessDay)
		r.Get("/{code}/add-business-days", h.AddBusinessDays)
		r.Post("/{code}/adjust-date", h.AdjustDate)
		r.Get("/{code}/holidays", h.GetHolidays)
	})
}

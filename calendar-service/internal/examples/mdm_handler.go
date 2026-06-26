package examples

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"calendar-service/internal/mdm"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// CalendarHandlerWithMDM demonstrates how to integrate MDM into calendar handlers
type CalendarHandlerWithMDM struct {
	mdmClient *mdm.Client
	logger    *logrus.Entry
}

// NewCalendarHandlerWithMDM creates a new handler with MDM client
func NewCalendarHandlerWithMDM(client *mdm.Client, logger *logrus.Logger) *CalendarHandlerWithMDM {
	return &CalendarHandlerWithMDM{
		mdmClient: client,
		logger:    logger.WithField("component", "calendar_handler"),
	}
}

// RegisterRoutes registers calendar endpoints that use MDM
func (h *CalendarHandlerWithMDM) RegisterRoutes(router *mux.Router) {
	if h.mdmClient == nil {
		h.logger.Info("MDM not enabled, registering handlers without MDM integration")
		return
	}

	router.HandleFunc("/api/v1/calendar/business-days", h.GetBusinessDays).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/calendar/is-business-day", h.IsBusinessDay).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/calendar/holidays", h.GetHolidays).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/calendar/audit-trail/{record-id}", h.GetAuditTrail).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/calendar/health", h.CheckHealth).Methods(http.MethodGet)
}

// GetBusinessDays returns business days for a date range (powered by MDM)
func (h *CalendarHandlerWithMDM) GetBusinessDays(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")
	region := r.URL.Query().Get("region")
	exchangeStr := r.URL.Query().Get("exchange")
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant-id", http.StatusBadRequest)
		return
	}

	if startDateStr == "" || endDateStr == "" {
		http.Error(w, "start_date and end_date are required", http.StatusBadRequest)
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		http.Error(w, "invalid start_date format", http.StatusBadRequest)
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		http.Error(w, "invalid end_date format", http.StatusBadRequest)
		return
	}

	var exchange *string
	if exchangeStr != "" {
		exchange = &exchangeStr
	}

	resp, err := h.mdmClient.GetGoldenCalendar(
		ctx,
		tenantID,
		startDate,
		endDate,
		region,
		exchange,
		r.Header.Get("Authorization"), // JWT token
	)

	if err != nil {
		h.logger.WithError(err).Warn("Failed to get business days from MDM")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Format response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// In this implementation, we extract only business days from the golden records
	businessDayDates := []string{}
	for _, record := range resp.Records {
		if record.IsBusinessDay {
			businessDayDates = append(businessDayDates, record.CalendarDate)
		}
	}

	response := map[string]interface{}{
		"start_date":    startDateStr,
		"end_date":      endDateStr,
		"business_days": businessDayDates,
		"count":         len(businessDayDates),
	}

	b, _ := json.Marshal(response)
	w.Write(b)
}

// IsBusinessDay checks if a specific date is a business day
func (h *CalendarHandlerWithMDM) IsBusinessDay(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	dateStr := r.URL.Query().Get("date")
	region := r.URL.Query().Get("region")
	exchangeStr := r.URL.Query().Get("exchange")
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant-id", http.StatusBadRequest)
		return
	}

	if dateStr == "" {
		http.Error(w, "date is required", http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "invalid date format", http.StatusBadRequest)
		return
	}

	var exchange *string
	if exchangeStr != "" {
		exchange = &exchangeStr
	}

	isBusinessDay, err := h.mdmClient.IsBusinessDay(
		ctx,
		tenantID,
		date,
		region,
		exchange,
		r.Header.Get("Authorization"), // JWT token
	)

	if err != nil {
		h.logger.WithError(err).Warn("Failed to check business day from MDM")
		isBusinessDay = true // Return safe default
	}

	// Format response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"date":            dateStr,
		"is_business_day": isBusinessDay,
		"region":          region,
		"exchange":        exchangeStr,
	}

	b, _ := json.Marshal(response)
	w.Write(b)
}

// GetHolidays returns holidays for a date range
func (h *CalendarHandlerWithMDM) GetHolidays(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")
	region := r.URL.Query().Get("region")
	exchangeStr := r.URL.Query().Get("exchange")
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant-id", http.StatusBadRequest)
		return
	}

	if startDateStr == "" || endDateStr == "" {
		http.Error(w, "start_date and end_date are required", http.StatusBadRequest)
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		http.Error(w, "invalid start_date format", http.StatusBadRequest)
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		http.Error(w, "invalid end_date format", http.StatusBadRequest)
		return
	}

	var exchange *string
	if exchangeStr != "" {
		exchange = &exchangeStr
	}

	resp, err := h.mdmClient.GetGoldenCalendar(
		ctx,
		tenantID,
		startDate,
		endDate,
		region,
		exchange,
		r.Header.Get("Authorization"), // JWT token
	)

	if err != nil {
		h.logger.WithError(err).Warn("Failed to get holidays from MDM")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Format response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	holidays := []map[string]interface{}{}
	for _, record := range resp.Records {
		if !record.IsBusinessDay && record.HolidayName != nil {
			holidays = append(holidays, map[string]interface{}{
				"date": record.CalendarDate,
				"name": *record.HolidayName,
			})
		}
	}

	response := map[string]interface{}{
		"start_date": startDateStr,
		"end_date":   endDateStr,
		"holidays":   holidays,
		"count":      len(holidays),
	}

	b, _ := json.Marshal(response)
	w.Write(b)
}

// GetAuditTrail returns audit trail for a calendar record
func (h *CalendarHandlerWithMDM) GetAuditTrail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	recordID := vars["record-id"]
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant-id", http.StatusBadRequest)
		return
	}

	if recordID == "" {
		http.Error(w, "record-id is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	auditTrail, err := h.mdmClient.GetLineage(
		ctx,
		tenantID,
		recordID,
		r.Header.Get("Authorization"), // JWT token
	)

	if err != nil {
		h.logger.WithError(err).Warn("Failed to get audit trail from MDM")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Format response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"record_id":   recordID,
		"audit_trail": auditTrail,
	}

	b, _ := json.Marshal(response)
	w.Write(b)
}

// CheckHealth returns MDM health status
func (h *CalendarHandlerWithMDM) CheckHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant-id", http.StatusBadRequest)
		return
	}

	healthStatus, err := h.mdmClient.GetHealthMetrics(
		ctx,
		tenantID,
		r.Header.Get("Authorization"), // JWT token
	)

	if err != nil {
		h.logger.WithError(err).Warn("Failed to get health status from MDM")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Format response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"mdm_health": healthStatus,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}

	b, _ := json.Marshal(response)
	w.Write(b)
}

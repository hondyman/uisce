package limits

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

// Handler handles HTTP requests for Limit Utilization and Headroom Dashboards.
type Handler struct {
	service *Service
}

// NewHandler creates a new limit Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers routes on the provided chi Router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/compliance/limits", func(r chi.Router) {
		r.Get("/utilization", h.HandleGetUtilization)
		r.Get("/history", h.HandleGetHistory)
	})
}

// HandleGetUtilization returns limit utilization records for the authenticated tenant.
func (h *Handler) HandleGetUtilization(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		if c, err := jwtmiddleware.ValidateTokenFromRequest(r); err == nil && c != nil {
			claims = c
		}
	}
	if claims == nil {
		http.Error(w, "unauthorized: missing claims", http.StatusUnauthorized)
		return
	}

	callerTenantStr := claims.TenantID
	if callerTenantStr == "" {
		http.Error(w, "unauthorized: tenant claim required", http.StatusUnauthorized)
		return
	}

	callerTenantID, err := uuid.Parse(callerTenantStr)
	if err != nil {
		http.Error(w, "unauthorized: invalid caller tenant id", http.StatusUnauthorized)
		return
	}

	requestedTenantStr := r.URL.Query().Get("tenant_id")
	targetTenantID := callerTenantID
	if requestedTenantStr != "" {
		if err := jwtmiddleware.ValidateTenantAccess(claims, requestedTenantStr); err != nil {
			http.Error(w, "forbidden: cross-tenant access denied", http.StatusForbidden)
			return
		}
		parsedReqID, err := uuid.Parse(requestedTenantStr)
		if err != nil {
			http.Error(w, "invalid tenant_id parameter", http.StatusBadRequest)
			return
		}
		targetTenantID = parsedReqID
	}

	filter := LimitFilter{
		TenantID: targetTenantID,
		RulePack: r.URL.Query().Get("rule_pack"),
		Category: r.URL.Query().Get("category"),
		Status:   r.URL.Query().Get("status"),
	}

	if acctStr := r.URL.Query().Get("account_id"); acctStr != "" {
		if parsedAcct, err := uuid.Parse(acctStr); err == nil {
			filter.AccountID = &parsedAcct
		}
	}

	records, err := h.service.GetLimitUtilization(r.Context(), filter)
	if err != nil {
		http.Error(w, "failed to get limit utilization: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":        records,
		"total_count": len(records),
		"tenant_id":   targetTenantID,
	})
}

// HandleGetHistory returns historical time-series data points for a specific rule.
func (h *Handler) HandleGetHistory(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		if c, err := jwtmiddleware.ValidateTokenFromRequest(r); err == nil && c != nil {
			claims = c
		}
	}
	if claims == nil {
		http.Error(w, "unauthorized: missing claims", http.StatusUnauthorized)
		return
	}

	callerTenantStr := claims.TenantID
	if callerTenantStr == "" {
		http.Error(w, "unauthorized: tenant claim required", http.StatusUnauthorized)
		return
	}

	callerTenantID, err := uuid.Parse(callerTenantStr)
	if err != nil {
		http.Error(w, "unauthorized: invalid caller tenant id", http.StatusUnauthorized)
		return
	}

	requestedTenantStr := r.URL.Query().Get("tenant_id")
	targetTenantID := callerTenantID
	if requestedTenantStr != "" {
		if err := jwtmiddleware.ValidateTenantAccess(claims, requestedTenantStr); err != nil {
			http.Error(w, "forbidden: cross-tenant access denied", http.StatusForbidden)
			return
		}
		parsedReqID, err := uuid.Parse(requestedTenantStr)
		if err != nil {
			http.Error(w, "invalid tenant_id parameter", http.StatusBadRequest)
			return
		}
		targetTenantID = parsedReqID
	}

	ruleID := r.URL.Query().Get("rule_id")
	if ruleID == "" {
		http.Error(w, "missing required rule_id parameter", http.StatusBadRequest)
		return
	}

	acctID := uuid.Nil
	if acctStr := r.URL.Query().Get("account_id"); acctStr != "" {
		if parsedAcct, err := uuid.Parse(acctStr); err == nil {
			acctID = parsedAcct
		}
	}

	days := 30
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	history, err := h.service.GetLimitHistory(r.Context(), targetTenantID, acctID, ruleID, days)
	if err != nil {
		http.Error(w, "failed to get limit history: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":      history,
		"rule_id":   ruleID,
		"tenant_id": targetTenantID,
	})
}

package directindexing

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler handles direct indexing HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new direct indexing handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetAccount retrieves account details
// GET /api/direct-indexing/accounts/{id}
func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	accountIDStr := chi.URLParam(r, "id")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid account ID"})
		return
	}

	account, err := h.service.GetAccount(r.Context(), accountID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}

// ListAccounts lists all accounts for a client
// GET /api/direct-indexing/accounts?client_id=uuid
func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	clientIDStr := r.URL.Query().Get("client_id")
	if clientIDStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "client_id required"})
		return
	}

	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid client_id"})
		return
	}

	accounts, err := h.service.ListAccounts(r.Context(), clientID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accounts": accounts,
		"total":    len(accounts),
	})
}

// GetHoldings retrieves account holdings
// GET /api/direct-indexing/accounts/{id}/holdings
func (h *Handler) GetHoldings(w http.ResponseWriter, r *http.Request) {
	accountIDStr := chi.URLParam(r, "id")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid account ID"})
		return
	}

	holdings, err := h.service.GetHoldings(r.Context(), accountID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"holdings": holdings,
		"total":    len(holdings),
	})
}

// GetOpportunities retrieves harvest opportunities
// GET /api/direct-indexing/accounts/{id}/opportunities?status=PENDING
func (h *Handler) GetOpportunities(w http.ResponseWriter, r *http.Request) {
	accountIDStr := chi.URLParam(r, "id")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid account ID"})
		return
	}

	status := r.URL.Query().Get("status")
	if status == "" {
		status = "PENDING"
	}

	opportunities, err := h.service.GetOpportunities(r.Context(), accountID, status)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Calculate totals
	var totalTaxSavings float64
	for _, opp := range opportunities {
		totalTaxSavings += opp.EstimatedTaxSavings
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"opportunities":     opportunities,
		"total":             len(opportunities),
		"total_tax_savings": totalTaxSavings,
	})
}

// ExecuteHarvest executes a harvest opportunity
// POST /api/direct-indexing/opportunities/{id}/execute
func (h *Handler) ExecuteHarvest(w http.ResponseWriter, r *http.Request) {
	opportunityIDStr := chi.URLParam(r, "id")
	opportunityID, err := uuid.Parse(opportunityIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid opportunity ID"})
		return
	}

	var req struct {
		ApprovedBy uuid.UUID `json:"approved_by"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	err = h.service.ExecuteHarvest(r.Context(), opportunityID, req.ApprovedBy)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"opportunity_id": opportunityID,
		"status":         "EXECUTED",
		"message":        "Harvest executed successfully",
	})
}

// DismissOpportunity dismisses a harvest opportunity
// POST /api/direct-indexing/opportunities/{id}/dismiss
func (h *Handler) DismissOpportunity(w http.ResponseWriter, r *http.Request) {
	opportunityIDStr := chi.URLParam(r, "id")
	opportunityID, err := uuid.Parse(opportunityIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid opportunity ID"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	err = h.service.DismissOpportunity(r.Context(), opportunityID, req.Reason)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"opportunity_id": opportunityID,
		"status":         "DISMISSED",
		"message":        "Opportunity dismissed",
	})
}

// GetPerformance retrieves account performance metrics
// GET /api/direct-indexing/accounts/{id}/performance
func (h *Handler) GetPerformance(w http.ResponseWriter, r *http.Request) {
	accountIDStr := chi.URLParam(r, "id")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid account ID"})
		return
	}

	metrics, err := h.service.GetPerformanceMetrics(r.Context(), accountID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// RegisterRoutes registers direct indexing routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/direct-indexing", func(r chi.Router) {
		// Account routes
		r.Get("/accounts", h.ListAccounts)
		r.Get("/accounts/{id}", h.GetAccount)
		r.Get("/accounts/{id}/holdings", h.GetHoldings)
		r.Get("/accounts/{id}/opportunities", h.GetOpportunities)
		r.Get("/accounts/{id}/performance", h.GetPerformance)

		// Opportunity routes
		r.Post("/opportunities/{id}/execute", h.ExecuteHarvest)
		r.Post("/opportunities/{id}/dismiss", h.DismissOpportunity)
	})
}

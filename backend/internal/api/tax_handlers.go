package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TaxHandlers struct {
	db interface{} // Would use actual tax service
}

func NewTaxHandlers() *TaxHandlers {
	return &TaxHandlers{}
}

func (h *TaxHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/api/tax", func(r chi.Router) {
		// Tax Opportunities
		r.Get("/opportunities", h.ListOpportunities)
		r.Get("/opportunities/{opportunityId}", h.GetOpportunity)
		r.Put("/opportunities/{opportunityId}/status", h.UpdateOpportunityStatus)

		// Quarterly Scans
		r.Post("/scan/quarterly", h.RunQuarterlyScan)
		r.Get("/scans/history", h.GetScanHistory)
	})
}

func (h *TaxHandlers) ListOpportunities(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	// Mock implementation - would query from database
	opportunities := []map[string]interface{}{
		{
			"opportunityId":    uuid.New().String(),
			"opportunityType":  "TAX_LOSS_HARVEST",
			"clientName":       "John Doe",
			"estimatedSavings": 5000,
			"complexity":       3,
			"taxYear":          2024,
			"deadline":         "2024-12-31",
			"description":      "Harvest losses in tech sector to offset capital gains",
			"actionRequired":   "Execute trades before year-end",
			"status":           "PENDING",
			"identifiedAt":     "2024-11-15T10:00:00Z",
		},
		{
			"opportunityId":    uuid.New().String(),
			"opportunityType":  "ROTH_CONVERSION",
			"clientName":       "Jane Smith",
			"estimatedSavings": 15000,
			"complexity":       7,
			"taxYear":          2024,
			"deadline":         "2024-12-31",
			"description":      "Convert traditional IRA to Roth while in lower tax bracket",
			"actionRequired":   "Complete conversion and payment before Dec 31",
			"status":           "PENDING",
			"identifiedAt":     "2024-11-10T14:00:00Z",
		},
	}

	if status != "" {
		// Filter by status
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(opportunities)
}

func (h *TaxHandlers) GetOpportunity(w http.ResponseWriter, r *http.Request) {
	opportunityID := chi.URLParam(r, "opportunityId")

	// Mock implementation
	opportunity := map[string]interface{}{
		"opportunityId":    opportunityID,
		"opportunityType":  "TAX_LOSS_HARVEST",
		"clientName":       "John Doe",
		"estimatedSavings": 5000,
		"complexity":       3,
		"taxYear":          2024,
		"description":      "Detailed opportunity data",
		"status":           "PENDING",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(opportunity)
}

func (h *TaxHandlers) UpdateOpportunityStatus(w http.ResponseWriter, r *http.Request) {
	opportunityID := chi.URLParam(r, "opportunityId")

	var input struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update opportunity status in database
	_ = opportunityID

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":        "updated",
		"opportunityId": opportunityID,
		"newStatus":     input.Status,
	})
}

func (h *TaxHandlers) RunQuarterlyScan(w http.ResponseWriter, r *http.Request) {
	// Trigger quarterly tax optimization scan
	// Would call PostgreSQL function: detect_tax_loss_harvesting_opportunities()
	// and detect_roth_conversion_opportunities()

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":              "scan_initiated",
		"scanId":              uuid.New().String(),
		"estimatedCompletion": "2-5 minutes",
	})
}

func (h *TaxHandlers) GetScanHistory(w http.ResponseWriter, r *http.Request) {
	history := []map[string]interface{}{
		{
			"scanId":             uuid.New().String(),
			"scanDate":           "2024-11-01",
			"opportunitiesFound": 12,
			"totalSavings":       125000,
			"status":             "COMPLETED",
		},
		{
			"scanId":             uuid.New().String(),
			"scanDate":           "2024-10-01",
			"opportunitiesFound": 8,
			"totalSavings":       89000,
			"status":             "COMPLETED",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

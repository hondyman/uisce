package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/altinv"
)

// AltInvestmentHandler handles alternative investment API requests
type AltInvestmentHandler struct {
	service altinv.Service
}

// NewAltInvestmentHandler creates a new alternative investment handler
func NewAltInvestmentHandler(service altinv.Service) *AltInvestmentHandler {
	return &AltInvestmentHandler{service: service}
}

// RegisterRoutes registers the alternative investment routes
func (h *AltInvestmentHandler) RegisterRoutes(r chi.Router) {
	r.Route("/alternative-investments", func(r chi.Router) {
		// Investments
		r.Post("/", h.CreateInvestment)
		r.Get("/{id}", h.GetInvestment)
		r.Put("/{id}", h.UpdateInvestment)
		r.Delete("/{id}", h.DeleteInvestment)
		r.Get("/", h.ListInvestments)

		// Performance
		r.Get("/{id}/performance", h.GetInvestmentPerformance)

		// Capital Calls
		r.Post("/{id}/capital-calls", h.CreateCapitalCall)
		r.Get("/{id}/capital-calls", h.ListCapitalCallsByInvestment)
		r.Get("/capital-calls/upcoming", h.ListUpcomingCapitalCalls)

		// Distributions
		r.Post("/{id}/distributions", h.CreateDistribution)
		r.Get("/{id}/distributions", h.ListDistributionsByInvestment)

		// Documents
		r.Post("/{id}/documents", h.CreateDocument)
		r.Get("/{id}/documents", h.ListDocumentsByInvestment)
	})

	// Capital call updates (not nested under investment)
	r.Patch("/capital-calls/{id}/status", h.UpdateCapitalCallStatus)
}

// CreateInvestment handles POST /alternative-investments
func (h *AltInvestmentHandler) CreateInvestment(w http.ResponseWriter, r *http.Request) {
	var input altinv.CreateInvestmentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	inv, err := h.service.CreateInvestment(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(inv)
}

// GetInvestment handles GET /alternative-investments/{id}
func (h *AltInvestmentHandler) GetInvestment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid investment ID", http.StatusBadRequest)
		return
	}

	inv, err := h.service.GetInvestment(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(inv)
}

// UpdateInvestment handles PUT /alternative-investments/{id}
func (h *AltInvestmentHandler) UpdateInvestment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid investment ID", http.StatusBadRequest)
		return
	}

	var input altinv.UpdateInvestmentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	inv, err := h.service.UpdateInvestment(r.Context(), id, input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(inv)
}

// DeleteInvestment handles DELETE /alternative-investments/{id}
func (h *AltInvestmentHandler) DeleteInvestment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid investment ID", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteInvestment(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListInvestments handles GET /alternative-investments?client_id=uuid
func (h *AltInvestmentHandler) ListInvestments(w http.ResponseWriter, r *http.Request) {
	clientIDStr := r.URL.Query().Get("client_id")
	if clientIDStr == "" {
		http.Error(w, "client_id query parameter required", http.StatusBadRequest)
		return
	}

	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		http.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}

	invs, err := h.service.ListInvestmentsByClient(r.Context(), clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(invs)
}

// GetInvestmentPerformance handles GET /alternative-investments/{id}/performance
func (h *AltInvestmentHandler) GetInvestmentPerformance(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid investment ID", http.StatusBadRequest)
		return
	}

	perf, err := h.service.GetInvestmentPerformance(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(perf)
}

// CreateCapitalCall handles POST /alternative-investments/{id}/capital-calls
func (h *AltInvestmentHandler) CreateCapitalCall(w http.ResponseWriter, r *http.Request) {
	investmentIDStr := chi.URLParam(r, "id")
	investmentID, err := uuid.Parse(investmentIDStr)
	if err != nil {
		http.Error(w, "Invalid investment ID", http.StatusBadRequest)
		return
	}

	var input altinv.CreateCapitalCallInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	input.InvestmentID = investmentID

	call, err := h.service.CreateCapitalCall(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(call)
}

// ListCapitalCallsByInvestment handles GET /alternative-investments/{id}/capital-calls
func (h *AltInvestmentHandler) ListCapitalCallsByInvestment(w http.ResponseWriter, r *http.Request) {
	investmentIDStr := chi.URLParam(r, "id")
	investmentID, err := uuid.Parse(investmentIDStr)
	if err != nil {
		http.Error(w, "Invalid investment ID", http.StatusBadRequest)
		return
	}

	calls, err := h.service.ListCapitalCallsByInvestment(r.Context(), investmentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(calls)
}

// ListUpcomingCapitalCalls handles GET /alternative-investments/capital-calls/upcoming?client_id=uuid
func (h *AltInvestmentHandler) ListUpcomingCapitalCalls(w http.ResponseWriter, r *http.Request) {
	var clientID *uuid.UUID

	clientIDStr := r.URL.Query().Get("client_id")
	if clientIDStr != "" {
		parsed, err := uuid.Parse(clientIDStr)
		if err != nil {
			http.Error(w, "Invalid client ID", http.StatusBadRequest)
			return
		}
		clientID = &parsed
	}

	calls, err := h.service.ListUpcomingCapitalCalls(r.Context(), clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(calls)
}

// UpdateCapitalCallStatus handles PATCH /capital-calls/{id}/status
func (h *AltInvestmentHandler) UpdateCapitalCallStatus(w http.ResponseWriter, r *http.Request) {
	callIDStr := chi.URLParam(r, "id")
	callID, err := uuid.Parse(callIDStr)
	if err != nil {
		http.Error(w, "Invalid capital call ID", http.StatusBadRequest)
		return
	}

	var input struct {
		Status       altinv.CapitalCallStatus `json:"status"`
		AmountFunded float64                  `json:"amount_funded"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateCapitalCallStatus(r.Context(), callID, input.Status, input.AmountFunded); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// CreateDistribution handles POST /alternative-investments/{id}/distributions
func (h *AltInvestmentHandler) CreateDistribution(w http.ResponseWriter, r *http.Request) {
	investmentIDStr := chi.URLParam(r, "id")
	investmentID, err := uuid.Parse(investmentIDStr)
	if err != nil {
		http.Error(w, "Invalid investment ID", http.StatusBadRequest)
		return
	}

	var input altinv.CreateDistributionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	input.InvestmentID = investmentID

	dist, err := h.service.CreateDistribution(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dist)
}

// ListDistributionsByInvestment handles GET /alternative-investments/{id}/distributions
func (h *AltInvestmentHandler) ListDistributionsByInvestment(w http.ResponseWriter, r *http.Request) {
	investmentIDStr := chi.URLParam(r, "id")
	investmentID, err := uuid.Parse(investmentIDStr)
	if err != nil {
		http.Error(w, "Invalid investment ID", http.StatusBadRequest)
		return
	}

	dists, err := h.service.ListDistributionsByInvestment(r.Context(), investmentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(dists)
}

// CreateDocument handles POST /alternative-investments/{id}/documents
func (h *AltInvestmentHandler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	investmentIDStr := chi.URLParam(r, "id")
	investmentID, err := uuid.Parse(investmentIDStr)
	if err != nil {
		http.Error(w, "Invalid investment ID", http.StatusBadRequest)
		return
	}

	var input altinv.CreateDocumentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	input.InvestmentID = investmentID

	doc, err := h.service.CreateDocument(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(doc)
}

// ListDocumentsByInvestment handles GET /alternative-investments/{id}/documents
func (h *AltInvestmentHandler) ListDocumentsByInvestment(w http.ResponseWriter, r *http.Request) {
	investmentIDStr := chi.URLParam(r, "id")
	investmentID, err := uuid.Parse(investmentIDStr)
	if err != nil {
		http.Error(w, "Invalid investment ID", http.StatusBadRequest)
		return
	}

	docs, err := h.service.ListDocumentsByInvestment(r.Context(), investmentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(docs)
}

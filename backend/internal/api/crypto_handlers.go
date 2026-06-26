package api

import (
	"encoding/json"
	"net/http"

	"github.com/hondyman/semlayer/backend/internal/crypto"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CryptoHandlers struct {
	service crypto.Service
}

func NewCryptoHandlers(service crypto.Service) *CryptoHandlers {
	return &CryptoHandlers{service: service}
}

func (h *CryptoHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/api/crypto", func(r chi.Router) {
		// Holdings
		r.Get("/clients/{clientId}/holdings", h.GetClientHoldings)
		r.Get("/holdings/{holdingId}", h.GetHolding)

		// Transactions
		r.Post("/transactions", h.RecordTransaction)
		r.Get("/clients/{clientId}/transactions", h.GetClientTransactions)

		// Market Data
		r.Get("/prices/{symbol}", h.GetLatestPrice)
		r.Get("/prices/{symbol}/history", h.GetPriceHistory)

		// Portfolio Analytics
		r.Get("/clients/{clientId}/portfolio", h.GetPortfolioSummary)
		r.Get("/clients/{clientId}/allocation", h.GetAllocationPercentage)

		// Tax Loss Harvesting
		r.Get("/clients/{clientId}/tax-loss-opportunities", h.GetTaxLossOpportunities)
	})
}

func (h *CryptoHandlers) GetClientHoldings(w http.ResponseWriter, r *http.Request) {
	clientID, err := uuid.Parse(chi.URLParam(r, "clientId"))
	if err != nil {
		http.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}

	holdings, err := h.service.GetClientHoldings(r.Context(), clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(holdings)
}

func (h *CryptoHandlers) GetHolding(w http.ResponseWriter, r *http.Request) {
	holdingID, err := uuid.Parse(chi.URLParam(r, "holdingId"))
	if err != nil {
		http.Error(w, "Invalid holding ID", http.StatusBadRequest)
		return
	}

	holding, err := h.service.GetHolding(r.Context(), holdingID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(holding)
}

func (h *CryptoHandlers) RecordTransaction(w http.ResponseWriter, r *http.Request) {
	var input crypto.RecordTransactionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	transaction, err := h.service.RecordTransaction(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(transaction)
}

func (h *CryptoHandlers) GetClientTransactions(w http.ResponseWriter, r *http.Request) {
	clientID, err := uuid.Parse(chi.URLParam(r, "clientId"))
	if err != nil {
		http.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}

	limit := 100
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		// Parse limit parameter
	}

	transactions, err := h.service.GetClientTransactions(r.Context(), clientID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transactions)
}

func (h *CryptoHandlers) GetLatestPrice(w http.ResponseWriter, r *http.Request) {
	symbol := crypto.AssetSymbol(chi.URLParam(r, "symbol"))

	price, err := h.service.GetLatestPrice(r.Context(), symbol)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"symbol": symbol,
		"price":  price,
	})
}

func (h *CryptoHandlers) GetPriceHistory(w http.ResponseWriter, r *http.Request) {
	symbol := crypto.AssetSymbol(chi.URLParam(r, "symbol"))
	hours := 24 // Default 24 hours

	history, err := h.service.GetPriceHistory(r.Context(), symbol, hours)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func (h *CryptoHandlers) GetPortfolioSummary(w http.ResponseWriter, r *http.Request) {
	clientID, err := uuid.Parse(chi.URLParam(r, "clientId"))
	if err != nil {
		http.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}

	summary, err := h.service.GetClientPortfolioSummary(r.Context(), clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func (h *CryptoHandlers) GetAllocationPercentage(w http.ResponseWriter, r *http.Request) {
	clientID, err := uuid.Parse(chi.URLParam(r, "clientId"))
	if err != nil {
		http.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}

	allocationPct, err := h.service.GetAllocationPercentage(r.Context(), clientID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"clientId":      clientID,
		"allocationPct": allocationPct,
	})
}

func (h *CryptoHandlers) GetTaxLossOpportunities(w http.ResponseWriter, r *http.Request) {
	clientID, err := uuid.Parse(chi.URLParam(r, "clientId"))
	if err != nil {
		http.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}

	minLoss := decimal.NewFromInt(1000) // Minimum $1,000 loss

	opportunities, err := h.service.IdentifyTaxLossOpportunities(r.Context(), clientID, minLoss)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(opportunities)
}

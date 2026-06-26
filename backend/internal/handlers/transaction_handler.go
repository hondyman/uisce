package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/identity"
	"github.com/hondyman/semlayer/backend/internal/transaction"
)

type TransactionHandler struct {
	repo transaction.TransactionRepository
}

func NewTransactionHandler(repo transaction.TransactionRepository) *TransactionHandler {
	return &TransactionHandler{repo: repo}
}

// GET /api/v1/transactions?portfolio_id=xxx&start_date=2026-01-01
func (h *TransactionHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	tenantStr, ok := identity.TenantIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tenantID, err := uuid.Parse(tenantStr)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid tenant format")
		return
	}
	ctx := context.WithValue(r.Context(), "tenant_id", tenantID)

	var portfolioID *uuid.UUID
	if pid := r.URL.Query().Get("portfolio_id"); pid != "" {
		if id, err := uuid.Parse(pid); err == nil {
			portfolioID = &id
		}
	}

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	var startDatePtr, endDatePtr *string
	if startDate != "" {
		startDatePtr = &startDate
	}
	if endDate != "" {
		endDatePtr = &endDate
	}

	txs, err := h.repo.ListTransactions(ctx, portfolioID, startDatePtr, endDatePtr)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"data": txs})
}

// POST /api/v1/transactions/ingest
func (h *TransactionHandler) IngestTransactions(w http.ResponseWriter, r *http.Request) {
	tenantStr, ok := identity.TenantIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tenantID, err := uuid.Parse(tenantStr)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid tenant format")
		return
	}
	ctx := context.WithValue(r.Context(), "tenant_id", tenantID)

	var raw []transaction.TransactionMasterRecord
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	var results []map[string]interface{}
	for _, t := range raw {
		t.TenantID = tenantID // Guarantee tenant ID is applied

		// 1. DQ Validation
		if errs := transaction.ValidateTransaction(&t); len(errs) > 0 {
			var errStrings []string
			for _, e := range errs {
				errStrings = append(errStrings, e.Error())
			}
			results = append(results, map[string]interface{}{"external_reference": t.ExternalReference, "status": "rejected", "errors": errStrings})
			continue
		}

		// 2. Upsert with Survivorship (Accounting > Custodian > OMS)
		gold, err := h.repo.UpsertTransaction(ctx, &t, []string{"AccountingSystem", "Custodian", "OMS"})
		if err != nil {
			results = append(results, map[string]interface{}{"external_reference": t.ExternalReference, "status": "error", "message": err.Error()})
			continue
		}

		results = append(results, map[string]interface{}{"external_reference": t.ExternalReference, "status": "success", "transaction_id": gold.TransactionID})
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"results": results})
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

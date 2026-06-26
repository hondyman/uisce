package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/cash"
	"github.com/hondyman/semlayer/backend/internal/identity"
)

type CashHandler struct {
	repo cash.CashRepository
}

func NewCashHandler(repo cash.CashRepository) *CashHandler {
	return &CashHandler{repo: repo}
}

// GET /api/v1/cash/balances?portfolio_id=xxx&start_date=2026-01-01
func (h *CashHandler) ListCashBalances(w http.ResponseWriter, r *http.Request) {
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

	balances, err := h.repo.ListCashBalances(ctx, portfolioID, startDatePtr, endDatePtr)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"data": balances})
}

// GET /api/v1/cash/ledger?portfolio_id=xxx&start_date=2026-01-01
func (h *CashHandler) ListCashLedger(w http.ResponseWriter, r *http.Request) {
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

	ledger, err := h.repo.ListCashLedger(ctx, portfolioID, startDatePtr, endDatePtr)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"data": ledger})
}

// POST /api/v1/cash/ledger/ingest
func (h *CashHandler) IngestCashLedger(w http.ResponseWriter, r *http.Request) {
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

	var raw []cash.CashLedgerEntryRecord
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	var results []map[string]interface{}
	for _, l := range raw {
		l.TenantID = tenantID

		// 1. DQ Validation
		if errs := cash.ValidateCashLedger(&l); len(errs) > 0 {
			var errStrings []string
			for _, e := range errs {
				errStrings = append(errStrings, e.Error())
			}
			results = append(results, map[string]interface{}{"external_reference": l.ExternalReference, "status": "rejected", "errors": errStrings})
			continue
		}

		// 2. Upsert with Survivorship (Accounting > Custodian > OMS)
		gold, err := h.repo.UpsertCashLedger(ctx, &l, []string{"AccountingSystem", "Custodian", "OMS"})
		if err != nil {
			results = append(results, map[string]interface{}{"external_reference": l.ExternalReference, "status": "error", "message": err.Error()})
			continue
		}

		results = append(results, map[string]interface{}{"external_reference": l.ExternalReference, "status": "success", "cash_ledger_id": gold.CashLedgerID})
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"results": results})
}

// POST /api/v1/cash/balances/rollforward?portfolio_id=xxx&currency=USD&date=2026-02-22
func (h *CashHandler) RunBalanceRollForward(w http.ResponseWriter, r *http.Request) {
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

	portfolioID := r.URL.Query().Get("portfolio_id")
	currency := r.URL.Query().Get("currency")
	valuationDate := r.URL.Query().Get("date")

	if portfolioID == "" || currency == "" || valuationDate == "" {
		respondWithError(w, http.StatusBadRequest, "portfolio_id, currency, and date are required")
		return
	}

	pid, err := uuid.Parse(portfolioID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid portfolio_id")
		return
	}

	balance, err := h.repo.RunBalanceRollForward(ctx, pid, currency, valuationDate)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"data": balance})
}

// POST /api/v1/cash/transactions/map
// Maps settled transactions into Cash Ledger entries (Semantic Execution Fabric §7)
func (h *CashHandler) MapTransactionsToCash(w http.ResponseWriter, r *http.Request) {
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

	var mappings []cash.TransactionCashMapping
	if err := json.NewDecoder(r.Body).Decode(&mappings); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	var results []map[string]interface{}
	for _, m := range mappings {
		m.TenantID = tenantID
		m.MappingID = uuid.New()

		if errs := cash.ValidateTransactionCashMapping(&m); len(errs) > 0 {
			var errStrings []string
			for _, e := range errs {
				errStrings = append(errStrings, e.Error())
			}
			results = append(results, map[string]interface{}{"transaction_id": m.TransactionID, "mapping_type": m.MappingType, "status": "rejected", "errors": errStrings})
			continue
		}

		err := h.repo.CreateTransactionCashMapping(ctx, &m)
		if err != nil {
			results = append(results, map[string]interface{}{"transaction_id": m.TransactionID, "mapping_type": m.MappingType, "status": "error", "message": err.Error()})
			continue
		}

		results = append(results, map[string]interface{}{"transaction_id": m.TransactionID, "mapping_type": m.MappingType, "status": "success", "mapping_id": m.MappingID})
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"results": results})
}

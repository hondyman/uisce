package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/position"
	"github.com/shopspring/decimal"
)

// PositionHandlers provides REST endpoints for the Position Master domain.
type PositionHandlers struct {
	repo *position.Repository
}

// NewPositionHandlers creates new Position handlers.
func NewPositionHandlers(repo *position.Repository) *PositionHandlers {
	return &PositionHandlers{repo: repo}
}

// ─────────────────────────────────────────────────────────────────────────────
// POSITIONS (Holdings)
// ─────────────────────────────────────────────────────────────────────────────

// ListPositions returns gold-copy positions for the tenant.
// GET /api/v1/positions
func (h *PositionHandlers) ListPositions(w http.ResponseWriter, r *http.Request) {
	tenantID, err := parseTenantID(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid or missing X-Tenant-ID header")
		return
	}

	var portfolioID *uuid.UUID
	if pidStr := r.URL.Query().Get("portfolio_id"); pidStr != "" {
		parsed, err := uuid.Parse(pidStr)
		if err != nil {
			sendError(w, http.StatusBadRequest, "Invalid portfolio_id query parameter")
			return
		}
		portfolioID = &parsed
	}

	limit, offset := paginateQuery(r)
	ctx := setupAuthContext(r.Context(), tenantID.String())

	records, err := h.repo.ListPositions(ctx, tenantID, portfolioID, limit, offset)
	if err != nil {
		SendErrorResponse(w, 500, "Failed to list positions", err.Error())
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"positions": records,
		"total":     len(records),
		"limit":     limit,
		"offset":    offset,
	})
}

// IngestPositions accepts raw custodian position records, runs DQ + survivorship.
// POST /api/v1/positions/ingest
func (h *PositionHandlers) IngestPositions(w http.ResponseWriter, r *http.Request) {
	tenantID, err := parseTenantID(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid or missing X-Tenant-ID header")
		return
	}

	var raw []position.RawPositionRecord
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body: expected array of position records")
		return
	}

	ctx := setupAuthContext(r.Context(), tenantID.String())
	var passedCount, failedCount int
	var dqErrors []string
	ingested := 0

	for i := range raw {
		// Parse string fields to typed values
		qty, err := decimal.NewFromString(raw[i].Quantity)
		if err != nil {
			dqErrors = append(dqErrors, "cannot parse quantity: "+raw[i].Quantity)
			failedCount++
			continue
		}

		portfolioID, err := uuid.Parse(raw[i].PortfolioID)
		if err != nil {
			dqErrors = append(dqErrors, "invalid portfolio_id: "+raw[i].PortfolioID)
			failedCount++
			continue
		}
		securityID, err := uuid.Parse(raw[i].SecurityID)
		if err != nil {
			dqErrors = append(dqErrors, "invalid security_id: "+raw[i].SecurityID)
			failedCount++
			continue
		}
		posDate, err := time.Parse("2006-01-02", raw[i].PositionDate)
		if err != nil {
			dqErrors = append(dqErrors, "invalid position_date: "+raw[i].PositionDate)
			failedCount++
			continue
		}

		side := raw[i].Side
		if side == "" {
			if qty.IsNegative() {
				side = "Short"
			} else {
				side = "Long"
			}
		}

		rec := &position.PositionMasterRecord{
			TenantID:         tenantID,
			PortfolioID:      portfolioID,
			SecurityID:       securityID,
			PositionDate:     posDate,
			PositionQuantity: qty,
			PositionSide:     side,
			PositionCurrency: raw[i].Currency,
			PositionSource:   raw[i].Source,
			IsReconciled:     raw[i].Source == "Custodian",
		}

		// Parse optional decimal fields
		if raw[i].MarketValueLocal != "" {
			if v, err := decimal.NewFromString(raw[i].MarketValueLocal); err == nil {
				rec.MarketValueLocal = &v
			}
		}
		if raw[i].MarketValueBase != "" {
			if v, err := decimal.NewFromString(raw[i].MarketValueBase); err == nil {
				rec.MarketValueBase = &v
			}
		}
		if raw[i].ValuationFXRate != "" {
			if v, err := decimal.NewFromString(raw[i].ValuationFXRate); err == nil {
				rec.ValuationFXRate = &v
			}
		}
		if raw[i].CostBasisLocal != "" {
			if v, err := decimal.NewFromString(raw[i].CostBasisLocal); err == nil {
				rec.CostBasisLocal = &v
			}
		}
		if raw[i].PriceID != "" {
			if pid, err := uuid.Parse(raw[i].PriceID); err == nil {
				rec.PriceID = &pid
			}
		}

		// Run DQ validation
		results, hardFail := position.ValidatePosition(rec)
		_, failed, errs := position.DQSummary(results)
		passedCount += len(results) - len(failed)
		failedCount += len(failed)
		dqErrors = append(dqErrors, errs...)

		if hardFail {
			continue
		}

		if err := h.repo.UpsertPosition(ctx, tenantID, rec); err != nil {
			SendErrorResponse(w, 500, "Failed to upsert position", err.Error())
			return
		}
		ingested++
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"dq_rules_passed":  passedCount,
		"dq_rules_failed":  failedCount,
		"dq_errors":        dqErrors,
		"records_ingested": ingested,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// TAX LOTS
// ─────────────────────────────────────────────────────────────────────────────

// ListPositionLots returns all tax lots for a given position.
// GET /api/v1/positions/:positionId/lots
func (h *PositionHandlers) ListPositionLots(w http.ResponseWriter, r *http.Request) {
	posIDStr := chi.URLParam(r, "positionId")
	posID, err := uuid.Parse(posIDStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid position ID")
		return
	}

	tenantID, err := parseTenantID(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid or missing X-Tenant-ID header")
		return
	}

	limit, offset := paginateQuery(r)
	ctx := setupAuthContext(r.Context(), tenantID.String())

	records, err := h.repo.ListPositionLots(ctx, posID, limit, offset)
	if err != nil {
		SendErrorResponse(w, 500, "Failed to list position lots", err.Error())
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"lots":   records,
		"total":  len(records),
		"limit":  limit,
		"offset": offset,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// CASH POSITIONS
// ─────────────────────────────────────────────────────────────────────────────

// ListCashPositions returns cash balances for the tenant.
// GET /api/v1/positions/cash
func (h *PositionHandlers) ListCashPositions(w http.ResponseWriter, r *http.Request) {
	tenantID, err := parseTenantID(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid or missing X-Tenant-ID header")
		return
	}

	var portfolioID *uuid.UUID
	if pidStr := r.URL.Query().Get("portfolio_id"); pidStr != "" {
		parsed, err := uuid.Parse(pidStr)
		if err != nil {
			sendError(w, http.StatusBadRequest, "Invalid portfolio_id query parameter")
			return
		}
		portfolioID = &parsed
	}

	limit, offset := paginateQuery(r)
	ctx := setupAuthContext(r.Context(), tenantID.String())

	records, err := h.repo.ListCashPositions(ctx, tenantID, portfolioID, limit, offset)
	if err != nil {
		SendErrorResponse(w, 500, "Failed to list cash positions", err.Error())
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"cash_positions": records,
		"total":          len(records),
		"limit":          limit,
		"offset":         offset,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// POSITION SNAPSHOTS
// ─────────────────────────────────────────────────────────────────────────────

// ListPositionSnapshots returns historical snapshots for a position.
// GET /api/v1/positions/:positionId/snapshots
func (h *PositionHandlers) ListPositionSnapshots(w http.ResponseWriter, r *http.Request) {
	posIDStr := chi.URLParam(r, "positionId")
	posID, err := uuid.Parse(posIDStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid position ID")
		return
	}

	tenantID, err := parseTenantID(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid or missing X-Tenant-ID header")
		return
	}

	limit, offset := paginateQuery(r)
	ctx := setupAuthContext(r.Context(), tenantID.String())

	records, err := h.repo.ListSnapshots(ctx, posID, limit, offset)
	if err != nil {
		SendErrorResponse(w, 500, "Failed to list snapshots", err.Error())
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"snapshots": records,
		"total":     len(records),
		"limit":     limit,
		"offset":    offset,
	})
}

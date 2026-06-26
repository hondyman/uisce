package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/pricing"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// PricingHandlers provides REST endpoints for the Pricing Master domain.
type PricingHandlers struct {
	repo *pricing.Repository
}

// NewPricingHandlers creates new Pricing handlers.
func NewPricingHandlers(repo *pricing.Repository) *PricingHandlers {
	return &PricingHandlers{repo: repo}
}

// paginate extracts limit/offset query params with sensible defaults.
func paginateQuery(r *http.Request) (int, int) {
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}
	return limit, offset
}

// parseTenantID extracts and parses the tenant UUID from the X-Tenant-ID header.
func parseTenantID(r *http.Request) (uuid.UUID, error) {
	raw := normalizeTenantID(jwtmiddleware.GetClaimsFromContext(r).TenantID)
	return uuid.Parse(raw)
}

// ─────────────────────────────────────────────────────────────────────────────
// PRICES
// ─────────────────────────────────────────────────────────────────────────────

// ListPrices returns all gold-copy prices for the tenant.
// GET /api/v1/pricing/prices
func (h *PricingHandlers) ListPrices(w http.ResponseWriter, r *http.Request) {
	tenantID, err := parseTenantID(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid or missing X-Tenant-ID header")
		return
	}

	limit, offset := paginateQuery(r)
	ctx := setupAuthContext(r.Context(), tenantID.String())

	records, err := h.repo.ListPrices(ctx, tenantID, limit, offset)
	if err != nil {
		SendErrorResponse(w, 500, "Failed to list prices", err.Error())
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"prices": records,
		"total":  len(records),
		"limit":  limit,
		"offset": offset,
	})
}

// IngestPrices accepts raw price records, runs DQ, and upserts survivors.
// POST /api/v1/pricing/prices/ingest
func (h *PricingHandlers) IngestPrices(w http.ResponseWriter, r *http.Request) {
	tenantID, err := parseTenantID(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid or missing X-Tenant-ID header")
		return
	}

	var raw []pricing.RawPriceRecord
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body: expected array of price records")
		return
	}

	ctx := setupAuthContext(r.Context(), tenantID.String())
	var passed, failed, errors []string

	for i := range raw {
		results, hardFail := pricing.ValidatePrice(&raw[i])
		p, f, e := pricing.DQSummary(results)
		passed = append(passed, p...)
		failed = append(failed, f...)
		errors = append(errors, e...)

		if hardFail {
			continue
		}

		rec := &pricing.PriceMasterRecord{
			TenantID:         tenantID,
			SecurityID:       uuid.MustParse(raw[i].SecurityID),
			PriceType:        raw[i].PriceType,
			PriceValue:       raw[i].PriceValue,
			PriceCurrency:    raw[i].PriceCurrency,
			PriceSource:      raw[i].PriceSource,
			PriceConfidence:  80,
			IsCompositePrice: raw[i].IsComposite,
		}
		if err := h.repo.UpsertPriceMaster(ctx, tenantID, rec); err != nil {
			SendErrorResponse(w, 500, "Failed to upsert price record", err.Error())
			return
		}
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"dq_rules_passed":  len(passed),
		"dq_rules_failed":  len(failed),
		"dq_errors":        errors,
		"records_ingested": len(raw) - len(failed),
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// FX RATES
// ─────────────────────────────────────────────────────────────────────────────

// ListFXRates returns all gold-copy FX rates for the tenant.
// GET /api/v1/pricing/fx-rates
func (h *PricingHandlers) ListFXRates(w http.ResponseWriter, r *http.Request) {
	tenantID, err := parseTenantID(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid or missing X-Tenant-ID header")
		return
	}

	limit, offset := paginateQuery(r)
	ctx := setupAuthContext(r.Context(), tenantID.String())

	records, err := h.repo.ListFXRates(ctx, tenantID, limit, offset)
	if err != nil {
		SendErrorResponse(w, 500, "Failed to list FX rates", err.Error())
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"fx_rates": records,
		"total":    len(records),
		"limit":    limit,
		"offset":   offset,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// CURVES
// ─────────────────────────────────────────────────────────────────────────────

// ListCurves returns all gold-copy curves for the tenant.
// GET /api/v1/pricing/curves
func (h *PricingHandlers) ListCurves(w http.ResponseWriter, r *http.Request) {
	tenantID, err := parseTenantID(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid or missing X-Tenant-ID header")
		return
	}

	limit, offset := paginateQuery(r)
	ctx := setupAuthContext(r.Context(), tenantID.String())

	records, err := h.repo.ListCurves(ctx, tenantID, limit, offset)
	if err != nil {
		SendErrorResponse(w, 500, "Failed to list curves", err.Error())
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"curves": records,
		"total":  len(records),
		"limit":  limit,
		"offset": offset,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// VOL SURFACES
// ─────────────────────────────────────────────────────────────────────────────

// ListVolSurfaces returns all gold-copy vol surfaces for the tenant.
// GET /api/v1/pricing/vol-surfaces
func (h *PricingHandlers) ListVolSurfaces(w http.ResponseWriter, r *http.Request) {
	tenantID, err := parseTenantID(r)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid or missing X-Tenant-ID header")
		return
	}

	limit, offset := paginateQuery(r)
	ctx := setupAuthContext(r.Context(), tenantID.String())

	records, err := h.repo.ListVolSurfaces(ctx, tenantID, limit, offset)
	if err != nil {
		SendErrorResponse(w, 500, "Failed to list vol surfaces", err.Error())
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"vol_surfaces": records,
		"total":        len(records),
		"limit":        limit,
		"offset":       offset,
	})
}

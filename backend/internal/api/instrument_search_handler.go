package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/oms/security"
)

// Allowed subtype codes for asset_class filter
var validAssetClasses = map[string]bool{
	string(security.SubtypeEquity):        true,
	string(security.SubtypeSovereignDebt): true,
	string(security.SubtypeCorporateDebt): true,
	string(security.SubtypeStructuredABS):  true,
	string(security.SubtypeETDDerivative): true,
	string(security.SubtypeOTCDerivative): true,
}

type InstrumentSearchResult struct {
	Symbol     string  `json:"symbol"`
	Name       string  `json:"name"`
	ISIN       string  `json:"isin,omitempty"`
	AssetClass string  `json:"asset_class"`
	Exchange   *string `json:"exchange,omitempty"`
	InternalID string  `json:"internal_id"`
}

type InstrumentSearchResponse struct {
	Results []InstrumentSearchResult `json:"results"`
}

type InstrumentSearchHandler struct {
	db *sql.DB
}

func NewInstrumentSearchHandler(db *sql.DB) *InstrumentSearchHandler {
	return &InstrumentSearchHandler{db: db}
}

// Search handles GET /api/instruments/search?q=<query>&limit=<n>&asset_class=<optional>
//
// Tenant Isolation & Gold-Copy Fallback:
//   - Searches oms.security matching caller's tenant_id OR platform gold-copy tenant.
//   - Tenant-specific record overrides core gold-copy record for the same ticker
//     via DISTINCT ON (s.ticker) ... ORDER BY s.ticker, (s.tenant_id = $1) DESC.
//     Tenant record completely wins (no field-level merge).
//
// Ranking:
//   1. Exact ticker match (UPPER(ticker) = UPPER(q))
//   2. Exact ISIN match (UPPER(isin) = UPPER(q))
//   3. Ticker prefix match (ticker ILIKE q || '%')
//   4. Name token match with word boundary (' ' || security_name || ' ' ILIKE '% ' || q || '%')
func (h *InstrumentSearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Auth resolution: Temporarily delegating to WorkspaceLayoutHandler.resolveUserAndTenant
	// for multi-channel auth extraction (JWT context/header, AuthInfo, Identity, and dev fallbacks).
	// TODO: Extract to a unified shared internal/api auth helper to avoid instantiating WorkspaceLayoutHandler.
	layoutHelper := NewWorkspaceLayoutHandler(h.db)
	_, tenantID, err := layoutHelper.resolveUserAndTenant(r)
	if err != nil || tenantID == uuid.Nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	queryParam := strings.TrimSpace(r.URL.Query().Get("q"))
	if queryParam == "" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(InstrumentSearchResponse{Results: []InstrumentSearchResult{}})
		return
	}

	// Limit clamping: default 10, max 20
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 20 {
		limit = 20
	}

	// Asset class filter validation
	assetClass := strings.TrimSpace(r.URL.Query().Get("asset_class"))
	if assetClass != "" && !validAssetClasses[assetClass] {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("invalid asset_class '%s'", assetClass)})
		return
	}

	results, err := h.searchSecurities(r.Context(), tenantID, queryParam, assetClass, limit)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(InstrumentSearchResponse{Results: results})
}

func (h *InstrumentSearchHandler) searchSecurities(
	ctx context.Context,
	tenantID uuid.UUID,
	query string,
	assetClass string,
	limit int,
) ([]InstrumentSearchResult, error) {
	// Step 1: Dedup caller tenant vs gold-copy tenant in a CTE.
	// Caller tenant row wins per ticker (is_tenant_override = 1 vs 0).
	// Bitemporal filter: valid_from <= NOW() AND (valid_to IS NULL OR valid_to > NOW()).
	//
	// Step 2: Calculate rank in outer query and ORDER BY rank ASC, ticker ASC LIMIT $limit.
	sqlQuery := `
	WITH ranked_securities AS (
		SELECT DISTINCT ON (UPPER(COALESCE(s.ticker, s.identifier_value)))
			s.id,
			s.tenant_id,
			COALESCE(s.ticker, s.identifier_value) AS ticker,
			s.security_name,
			s.isin,
			s.subtype_code,
			s.exchange_mic,
			CASE WHEN s.tenant_id = $1 THEN 1 ELSE 0 END AS is_tenant_override
		FROM oms.security s
		WHERE (
			s.tenant_id = $1 
			OR s.tenant_id = public.uisce_gold_copy_tenant_id()
		)
		AND (s.valid_from <= NOW())
		AND (s.valid_to IS NULL OR s.valid_to > NOW())
		AND ($2 = '' OR s.subtype_code = $2)
		AND (
			UPPER(s.ticker) = UPPER($3)
			OR UPPER(COALESCE(s.isin, '')) = UPPER($3)
			OR s.ticker ILIKE $3 || '%'
			OR (' ' || s.security_name || ' ') ILIKE '% ' || $3 || '%'
		)
		ORDER BY UPPER(COALESCE(s.ticker, s.identifier_value)), (s.tenant_id = $1) DESC
	)
	SELECT
		id,
		ticker,
		security_name,
		COALESCE(isin, '') AS isin,
		subtype_code,
		exchange_mic,
		CASE
			WHEN UPPER(ticker) = UPPER($3) THEN 1
			WHEN UPPER(isin) = UPPER($3) THEN 2
			WHEN ticker ILIKE $3 || '%' THEN 3
			ELSE 4
		END AS match_rank
	FROM ranked_securities
	ORDER BY match_rank ASC, ticker ASC
	LIMIT $4;
	`

	rows, err := h.db.QueryContext(ctx, sqlQuery, tenantID, assetClass, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed executing instrument search: %w", err)
	}
	defer rows.Close()

	results := make([]InstrumentSearchResult, 0, limit)
	for rows.Next() {
		var id uuid.UUID
		var ticker, name, isin, subtypeCode string
		var exchange sql.NullString
		var matchRank int

		if err := rows.Scan(&id, &ticker, &name, &isin, &subtypeCode, &exchange, &matchRank); err != nil {
			return nil, fmt.Errorf("failed scanning instrument search row: %w", err)
		}

		res := InstrumentSearchResult{
			Symbol:     ticker,
			Name:       name,
			ISIN:       isin,
			AssetClass: subtypeCode,
			InternalID: id.String(),
		}
		if exchange.Valid && exchange.String != "" {
			res.Exchange = &exchange.String
		}
		results = append(results, res)
	}

	return results, nil
}

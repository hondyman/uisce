package alternative_investment

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/altinv/alternative-investments", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/{id}", h.Get)
		r.Post("/", h.Create)
		r.Delete("/{id}", h.SoftDelete)
	})
}

func (h *Handler) tenantID(r *http.Request) uuid.UUID {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil || claims.TenantID == "" {
		return uuid.Nil
	}
	id, _ := uuid.Parse(claims.TenantID)
	return id
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := h.tenantID(r)
	if tenantID == uuid.Nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	subtype := r.URL.Query().Get("subtype_code")
	query := `
		SELECT id, tenant_id, investment_name, sponsor_name, asset_class, status, subtype_code,
		       vintage_year, committed_capital, called_capital, unfunded_commitment,
		       dpi, rvpi, round_series, pro_rata_rights_flag, lead_investor_name,
		       post_money_valuation, property_type, occupancy_rate_pct, gross_asset_value,
		       loan_to_value_pct, sofr_spread_bps, pik_interest_pct, warrant_coverage_pct,
		       covenant_type, hurdle_rate_pct, high_water_mark_nav,
		       lockup_period_months, redemption_notice_days,
		       project_phase, concession_expiry_year, esg_carbon_offset_tons,
		       created_at, updated_at, valid_from, valid_to
		FROM altinv.alternative_investment
		WHERE tenant_id = $1 AND (valid_to IS NULL OR valid_to > NOW())`
	args := []interface{}{tenantID}

	if subtype != "" {
		query += " AND subtype_code = $2"
		args = append(args, subtype)
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var records []AlternativeInvestmentRecord
	for rows.Next() {
		var rec AlternativeInvestmentRecord
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.InvestmentName, &rec.SponsorName,
			&rec.AssetClass, &rec.Status, &rec.SubtypeCode,
			&rec.VintageYear, &rec.CommittedCapital, &rec.CalledCapital, &rec.UnfundedCommitment,
			&rec.DPI, &rec.RVPI, &rec.RoundSeries, &rec.ProRataRightsFlag, &rec.LeadInvestorName,
			&rec.PostMoneyValuation, &rec.PropertyType, &rec.OccupancyRatePct, &rec.GrossAssetValue,
			&rec.LoanToValuePct, &rec.SOFRSpreadBPS, &rec.PIKInterestPct, &rec.WarrantCoveragePct,
			&rec.CovenantType, &rec.HurdleRatePct, &rec.HighWaterMarkNAV,
			&rec.LockupPeriodMonths, &rec.RedemptionNoticeDays,
			&rec.ProjectPhase, &rec.ConcessionExpiryYear, &rec.ESGCarbonOffsetTons,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
		); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		records = append(records, rec)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := h.tenantID(r)
	if tenantID == uuid.Nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	var rec AlternativeInvestmentRecord
	err = h.db.QueryRowContext(r.Context(), `
		SELECT id, tenant_id, investment_name, sponsor_name, asset_class, status, subtype_code,
		       vintage_year, committed_capital, called_capital, unfunded_commitment,
		       dpi, rvpi, round_series, pro_rata_rights_flag, lead_investor_name,
		       post_money_valuation, property_type, occupancy_rate_pct, gross_asset_value,
		       loan_to_value_pct, sofr_spread_bps, pik_interest_pct, warrant_coverage_pct,
		       covenant_type, hurdle_rate_pct, high_water_mark_nav,
		       lockup_period_months, redemption_notice_days,
		       project_phase, concession_expiry_year, esg_carbon_offset_tons,
		       created_at, updated_at, valid_from, valid_to
		FROM altinv.alternative_investment
		WHERE id = $1 AND tenant_id = $2 AND (valid_to IS NULL OR valid_to > NOW())`,
		id, tenantID,
	).Scan(
		&rec.ID, &rec.TenantID, &rec.InvestmentName, &rec.SponsorName,
		&rec.AssetClass, &rec.Status, &rec.SubtypeCode,
		&rec.VintageYear, &rec.CommittedCapital, &rec.CalledCapital, &rec.UnfundedCommitment,
		&rec.DPI, &rec.RVPI, &rec.RoundSeries, &rec.ProRataRightsFlag, &rec.LeadInvestorName,
		&rec.PostMoneyValuation, &rec.PropertyType, &rec.OccupancyRatePct, &rec.GrossAssetValue,
		&rec.LoanToValuePct, &rec.SOFRSpreadBPS, &rec.PIKInterestPct, &rec.WarrantCoveragePct,
		&rec.CovenantType, &rec.HurdleRatePct, &rec.HighWaterMarkNAV,
		&rec.LockupPeriodMonths, &rec.RedemptionNoticeDays,
		&rec.ProjectPhase, &rec.ConcessionExpiryYear, &rec.ESGCarbonOffsetTons,
		&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
	)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rec)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := h.tenantID(r)
	if tenantID == uuid.Nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	var rec AlternativeInvestmentRecord
	if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}
	if err := rec.Validate(); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	rec.TenantID = tenantID
	rec.ID = uuid.New()

	_, err := h.db.ExecContext(r.Context(), `
		INSERT INTO altinv.alternative_investment (
			id, tenant_id, investment_name, sponsor_name, asset_class, status, subtype_code,
			vintage_year, committed_capital, called_capital, unfunded_commitment,
			dpi, rvpi, round_series, pro_rata_rights_flag, lead_investor_name,
			post_money_valuation, property_type, occupancy_rate_pct, gross_asset_value,
			loan_to_value_pct, sofr_spread_bps, pik_interest_pct, warrant_coverage_pct,
			covenant_type, hurdle_rate_pct, high_water_mark_nav,
			lockup_period_months, redemption_notice_days,
			project_phase, concession_expiry_year, esg_carbon_offset_tons,
			created_at, updated_at, valid_from
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,NOW(),NOW(),NOW())`,
		rec.ID, rec.TenantID, rec.InvestmentName, rec.SponsorName,
		rec.AssetClass, rec.Status, rec.SubtypeCode,
		rec.VintageYear, rec.CommittedCapital, rec.CalledCapital, rec.UnfundedCommitment,
		rec.DPI, rec.RVPI, rec.RoundSeries, rec.ProRataRightsFlag, rec.LeadInvestorName,
		rec.PostMoneyValuation, rec.PropertyType, rec.OccupancyRatePct, rec.GrossAssetValue,
		rec.LoanToValuePct, rec.SOFRSpreadBPS, rec.PIKInterestPct, rec.WarrantCoveragePct,
		rec.CovenantType, rec.HurdleRatePct, rec.HighWaterMarkNAV,
		rec.LockupPeriodMonths, rec.RedemptionNoticeDays,
		rec.ProjectPhase, rec.ConcessionExpiryYear, rec.ESGCarbonOffsetTons,
	)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rec)
}

func (h *Handler) SoftDelete(w http.ResponseWriter, r *http.Request) {
	tenantID := h.tenantID(r)
	if tenantID == uuid.Nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	result, err := h.db.ExecContext(r.Context(),
		`UPDATE altinv.alternative_investment SET valid_to = NOW(), updated_at = NOW() WHERE id = $1 AND tenant_id = $2 AND valid_to IS NULL`,
		id, tenantID,
	)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, `{"error":"not found or already deleted"}`, http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

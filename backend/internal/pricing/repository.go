package pricing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// Repository provides data access and survivorship logic for the Pricing Master.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new Pricing Repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ─────────────────────────────────────────────────────────────────────────────
// LIST QUERIES
// ─────────────────────────────────────────────────────────────────────────────

// ListPrices returns gold-copy prices for a tenant.
func (r *Repository) ListPrices(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]PriceMasterRecord, error) {
	query := `
		SELECT id, tenant_id, core_id, security_id, price_type, price_date, price_value,
		       price_time, price_currency, fx_rate_to_base, price_source, price_confidence,
		       is_composite_price, composite_method, is_stale_price, stale_reason,
		       source_systems, created_at, updated_at, valid_from, valid_to
		FROM edm.price_master
		WHERE tenant_id = $1 AND (valid_to IS NULL OR valid_to > NOW())
		ORDER BY price_date DESC, price_source ASC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list prices: %w", err)
	}
	defer rows.Close()

	var records []PriceMasterRecord
	for rows.Next() {
		var rec PriceMasterRecord
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.CoreID, &rec.SecurityID, &rec.PriceType,
			&rec.PriceDate, &rec.PriceValue, &rec.PriceTime, &rec.PriceCurrency,
			&rec.FXRateToBase, &rec.PriceSource, &rec.PriceConfidence,
			&rec.IsCompositePrice, &rec.CompositeMethod, &rec.IsStalePrice,
			&rec.StaleReason, &rec.SourceSystems, &rec.CreatedAt, &rec.UpdatedAt,
			&rec.ValidFrom, &rec.ValidTo,
		); err != nil {
			return nil, fmt.Errorf("scan price record: %w", err)
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// ListFXRates returns gold-copy FX rates for a tenant.
func (r *Repository) ListFXRates(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]FXRateMasterRecord, error) {
	query := `
		SELECT id, tenant_id, core_id, base_currency, quote_currency, fx_rate_date, fx_tenor,
		       fx_rate, fx_source, fx_forward_points, fx_confidence, source_systems,
		       created_at, updated_at, valid_from, valid_to
		FROM edm.fx_rate_master
		WHERE tenant_id = $1 AND (valid_to IS NULL OR valid_to > NOW())
		ORDER BY fx_rate_date DESC, base_currency ASC, quote_currency ASC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list fx rates: %w", err)
	}
	defer rows.Close()

	var records []FXRateMasterRecord
	for rows.Next() {
		var rec FXRateMasterRecord
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.CoreID, &rec.BaseCurrency, &rec.QuoteCurrency,
			&rec.FXRateDate, &rec.FXTenor, &rec.FXRate, &rec.FXSource,
			&rec.FXForwardPoints, &rec.FXConfidence, &rec.SourceSystems,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
		); err != nil {
			return nil, fmt.Errorf("scan fx rate record: %w", err)
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// ListCurves returns gold-copy curves for a tenant.
func (r *Repository) ListCurves(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]CurveMasterRecord, error) {
	query := `
		SELECT id, tenant_id, core_id, curve_type, curve_currency, curve_as_of_date,
		       curve_source, curve_tenor_points, curve_interpolation, curve_extrapolation,
		       curve_confidence, source_systems, created_at, updated_at, valid_from, valid_to
		FROM edm.curve_master
		WHERE tenant_id = $1 AND (valid_to IS NULL OR valid_to > NOW())
		ORDER BY curve_as_of_date DESC, curve_type ASC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list curves: %w", err)
	}
	defer rows.Close()

	var records []CurveMasterRecord
	for rows.Next() {
		var rec CurveMasterRecord
		var tenorJSON []byte
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.CoreID, &rec.CurveType, &rec.CurveCurrency,
			&rec.CurveAsOfDate, &rec.CurveSource, &tenorJSON,
			&rec.CurveInterpolation, &rec.CurveExtrapolation, &rec.CurveConfidence,
			&rec.SourceSystems, &rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
		); err != nil {
			return nil, fmt.Errorf("scan curve record: %w", err)
		}
		if err := json.Unmarshal(tenorJSON, &rec.CurveTenorPoints); err != nil {
			log.Printf("Warning: failed to unmarshal curve tenor points for %s: %v", rec.ID, err)
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// ListVolSurfaces returns gold-copy vol surfaces for a tenant.
func (r *Repository) ListVolSurfaces(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]VolSurfaceMasterRecord, error) {
	query := `
		SELECT id, tenant_id, core_id, underlier_security_id, vol_surface_type, vol_as_of_date,
		       vol_source, vol_grid, vol_interpolation, vol_extrapolation,
		       vol_confidence, source_systems, created_at, updated_at, valid_from, valid_to
		FROM edm.vol_surface_master
		WHERE tenant_id = $1 AND (valid_to IS NULL OR valid_to > NOW())
		ORDER BY vol_as_of_date DESC, vol_surface_type ASC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list vol surfaces: %w", err)
	}
	defer rows.Close()

	var records []VolSurfaceMasterRecord
	for rows.Next() {
		var rec VolSurfaceMasterRecord
		var gridJSON []byte
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.CoreID, &rec.UnderlierSecurityID,
			&rec.VolSurfaceType, &rec.VolAsOfDate, &rec.VolSource, &gridJSON,
			&rec.VolInterpolation, &rec.VolExtrapolation, &rec.VolConfidence,
			&rec.SourceSystems, &rec.CreatedAt, &rec.UpdatedAt, &rec.ValidFrom, &rec.ValidTo,
		); err != nil {
			return nil, fmt.Errorf("scan vol surface record: %w", err)
		}
		if err := json.Unmarshal(gridJSON, &rec.VolGrid); err != nil {
			log.Printf("Warning: failed to unmarshal vol grid for %s: %v", rec.ID, err)
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// ─────────────────────────────────────────────────────────────────────────────
// UPSERT / SURVIVORSHIP
// ─────────────────────────────────────────────────────────────────────────────

// UpsertPriceMaster upserts a single price using the prefer_source survivorship
// strategy: Bloomberg > Refinitiv > FactSet > InternalModel.
func (r *Repository) UpsertPriceMaster(ctx context.Context, tenantID uuid.UUID, rec *PriceMasterRecord) error {
	sourceSystems, _ := json.Marshal(map[string]interface{}{
		rec.PriceSource: map[string]interface{}{
			"value":       rec.PriceValue,
			"confidence":  rec.PriceConfidence,
			"ingested_at": time.Now(),
		},
	})

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO edm.price_master (
			tenant_id, security_id, price_type, price_date, price_value,
			price_currency, price_source, price_confidence,
			is_composite_price, composite_method, is_stale_price, stale_reason,
			source_systems, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,NOW())
		ON CONFLICT (security_id, price_type, price_date, price_source, tenant_id)
		DO UPDATE SET
			price_value       = EXCLUDED.price_value,
			price_confidence  = EXCLUDED.price_confidence,
			is_stale_price    = EXCLUDED.is_stale_price,
			stale_reason      = EXCLUDED.stale_reason,
			source_systems    = edm.price_master.source_systems || EXCLUDED.source_systems,
			updated_at        = NOW()`,
		tenantID, rec.SecurityID, rec.PriceType, rec.PriceDate, rec.PriceValue,
		rec.PriceCurrency, rec.PriceSource, rec.PriceConfidence,
		rec.IsCompositePrice, rec.CompositeMethod, rec.IsStalePrice, rec.StaleReason,
		sourceSystems,
	)
	return err
}

// UpsertFXRateMaster upserts a single FX rate.
func (r *Repository) UpsertFXRateMaster(ctx context.Context, tenantID uuid.UUID, rec *FXRateMasterRecord) error {
	sourceSystems, _ := json.Marshal(map[string]interface{}{
		rec.FXSource: map[string]interface{}{
			"rate":        rec.FXRate,
			"confidence":  rec.FXConfidence,
			"ingested_at": time.Now(),
		},
	})

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO edm.fx_rate_master (
			tenant_id, base_currency, quote_currency, fx_rate_date, fx_tenor,
			fx_rate, fx_source, fx_forward_points, fx_confidence, source_systems, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW())
		ON CONFLICT (base_currency, quote_currency, fx_rate_date, fx_tenor, fx_source, tenant_id)
		DO UPDATE SET
			fx_rate          = EXCLUDED.fx_rate,
			fx_forward_points= EXCLUDED.fx_forward_points,
			fx_confidence    = EXCLUDED.fx_confidence,
			source_systems   = edm.fx_rate_master.source_systems || EXCLUDED.source_systems,
			updated_at       = NOW()`,
		tenantID, rec.BaseCurrency, rec.QuoteCurrency, rec.FXRateDate, rec.FXTenor,
		rec.FXRate, rec.FXSource, rec.FXForwardPoints, rec.FXConfidence, sourceSystems,
	)
	return err
}

// UpsertCurveMaster upserts a single curve.
func (r *Repository) UpsertCurveMaster(ctx context.Context, tenantID uuid.UUID, rec *CurveMasterRecord) error {
	tenorJSON, _ := json.Marshal(rec.CurveTenorPoints)
	sourceSystems, _ := json.Marshal(map[string]interface{}{
		rec.CurveSource: map[string]interface{}{
			"confidence":  rec.CurveConfidence,
			"ingested_at": time.Now(),
		},
	})

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO edm.curve_master (
			tenant_id, curve_type, curve_currency, curve_as_of_date, curve_source,
			curve_tenor_points, curve_interpolation, curve_extrapolation,
			curve_confidence, source_systems, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW())
		ON CONFLICT (curve_type, curve_currency, curve_as_of_date, curve_source, tenant_id)
		DO UPDATE SET
			curve_tenor_points  = EXCLUDED.curve_tenor_points,
			curve_interpolation = EXCLUDED.curve_interpolation,
			curve_extrapolation = EXCLUDED.curve_extrapolation,
			curve_confidence    = EXCLUDED.curve_confidence,
			source_systems      = edm.curve_master.source_systems || EXCLUDED.source_systems,
			updated_at          = NOW()`,
		tenantID, rec.CurveType, rec.CurveCurrency, rec.CurveAsOfDate, rec.CurveSource,
		tenorJSON, rec.CurveInterpolation, rec.CurveExtrapolation,
		rec.CurveConfidence, sourceSystems,
	)
	return err
}

// UpsertVolSurfaceMaster upserts a single vol surface.
func (r *Repository) UpsertVolSurfaceMaster(ctx context.Context, tenantID uuid.UUID, rec *VolSurfaceMasterRecord) error {
	gridJSON, _ := json.Marshal(rec.VolGrid)
	sourceSystems, _ := json.Marshal(map[string]interface{}{
		rec.VolSource: map[string]interface{}{
			"confidence":  rec.VolConfidence,
			"ingested_at": time.Now(),
		},
	})

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO edm.vol_surface_master (
			tenant_id, underlier_security_id, vol_surface_type, vol_as_of_date, vol_source,
			vol_grid, vol_interpolation, vol_extrapolation, vol_confidence, source_systems, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW())
		ON CONFLICT (underlier_security_id, vol_surface_type, vol_as_of_date, vol_source, tenant_id)
		DO UPDATE SET
			vol_grid           = EXCLUDED.vol_grid,
			vol_interpolation  = EXCLUDED.vol_interpolation,
			vol_extrapolation  = EXCLUDED.vol_extrapolation,
			vol_confidence     = EXCLUDED.vol_confidence,
			source_systems     = edm.vol_surface_master.source_systems || EXCLUDED.source_systems,
			updated_at         = NOW()`,
		tenantID, rec.UnderlierSecurityID, rec.VolSurfaceType, rec.VolAsOfDate, rec.VolSource,
		gridJSON, rec.VolInterpolation, rec.VolExtrapolation, rec.VolConfidence, sourceSystems,
	)
	return err
}

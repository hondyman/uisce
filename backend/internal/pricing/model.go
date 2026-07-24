package pricing

import (
	"time"

	"github.com/google/uuid"
)

// ─────────────────────────────────────────────────────────────────────────────
// Price Master
// ─────────────────────────────────────────────────────────────────────────────

// PriceMasterRecord represents the gold-copy price for a security.
type PriceMasterRecord struct {
	ID               uuid.UUID  `db:"id"               json:"id"`
	TenantID         uuid.UUID  `db:"tenant_id"        json:"tenant_id"`
	CoreID           *uuid.UUID `db:"core_id"          json:"core_id,omitempty"`
	SecurityID       uuid.UUID  `db:"security_id"      json:"security_id"`
	PriceType        string     `db:"price_type"       json:"price_type"`
	PriceDate        time.Time  `db:"price_date"       json:"price_date"`
	PriceValue       float64    `db:"price_value"      json:"price_value"`
	PriceTime        *time.Time `db:"price_time"       json:"price_time,omitempty"`
	PriceCurrency    string     `db:"price_currency"   json:"price_currency"`
	FXRateToBase     *float64   `db:"fx_rate_to_base"  json:"fx_rate_to_base,omitempty"`
	PriceSource      string     `db:"price_source"     json:"price_source"`
	PriceConfidence  int        `db:"price_confidence" json:"price_confidence"`
	IsCompositePrice bool       `db:"is_composite_price" json:"is_composite_price"`
	CompositeMethod  *string    `db:"composite_method" json:"composite_method,omitempty"`
	IsStalePrice     bool       `db:"is_stale_price"   json:"is_stale_price"`
	StaleReason      *string    `db:"stale_reason"     json:"stale_reason,omitempty"`
	SourceSystems    []byte     `db:"source_systems"   json:"source_systems"`
	CreatedAt        time.Time  `db:"created_at"       json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at"       json:"updated_at"`
	ValidFrom        time.Time  `db:"valid_from"       json:"valid_from"`
	ValidTo          *time.Time `db:"valid_to"         json:"valid_to,omitempty"`
}

// RawPriceRecord is a raw inbound price from a source system.
type RawPriceRecord struct {
	SecurityID      string  `json:"security_id"`
	PriceType       string  `json:"price_type"`
	PriceDate       string  `json:"price_date"` // YYYY-MM-DD
	PriceValue      float64 `json:"price_value"`
	PriceCurrency   string  `json:"price_currency"`
	PriceSource     string  `json:"price_source"`
	FXRateToBase    float64 `json:"fx_rate_to_base,omitempty"`
	IsComposite     bool    `json:"is_composite_price,omitempty"`
	CompositeMethod string  `json:"composite_method,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// FX Rate Master
// ─────────────────────────────────────────────────────────────────────────────

// FXRateMasterRecord represents a gold-copy FX rate.
type FXRateMasterRecord struct {
	ID              uuid.UUID  `db:"id"              json:"id"`
	TenantID        uuid.UUID  `db:"tenant_id"       json:"tenant_id"`
	CoreID          *uuid.UUID `db:"core_id"         json:"core_id,omitempty"`
	BaseCurrency    string     `db:"base_currency"   json:"base_currency"`
	QuoteCurrency   string     `db:"quote_currency"  json:"quote_currency"`
	FXRateDate      time.Time  `db:"fx_rate_date"    json:"fx_rate_date"`
	FXTenor         string     `db:"fx_tenor"        json:"fx_tenor"`
	FXRate          float64    `db:"fx_rate"         json:"fx_rate"`
	FXSource        string     `db:"fx_source"       json:"fx_source"`
	FXForwardPoints *float64   `db:"fx_forward_points" json:"fx_forward_points,omitempty"`
	FXConfidence    int        `db:"fx_confidence"   json:"fx_confidence"`
	SourceSystems   []byte     `db:"source_systems"  json:"source_systems"`
	CreatedAt       time.Time  `db:"created_at"      json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"      json:"updated_at"`
	ValidFrom       time.Time  `db:"valid_from"      json:"valid_from"`
	ValidTo         *time.Time `db:"valid_to"        json:"valid_to,omitempty"`
}

// RawFXRecord is a raw inbound FX rate from a source system.
type RawFXRecord struct {
	BaseCurrency    string  `json:"base_currency"`
	QuoteCurrency   string  `json:"quote_currency"`
	FXRateDate      string  `json:"fx_rate_date"` // YYYY-MM-DD
	FXTenor         string  `json:"fx_tenor"`
	FXRate          float64 `json:"fx_rate"`
	FXSource        string  `json:"fx_source"`
	FXForwardPoints float64 `json:"fx_forward_points,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Curve Master
// ─────────────────────────────────────────────────────────────────────────────

// TenorPoint represents a single point on a rate curve.
type TenorPoint struct {
	Tenor          string  `json:"tenor"`
	Rate           float64 `json:"rate"`
	DiscountFactor float64 `json:"discount_factor,omitempty"`
}

// CurveMasterRecord represents a gold-copy rate curve.
type CurveMasterRecord struct {
	ID                 uuid.UUID    `db:"id"                   json:"id"`
	TenantID           uuid.UUID    `db:"tenant_id"            json:"tenant_id"`
	CoreID             *uuid.UUID   `db:"core_id"              json:"core_id,omitempty"`
	CurveType          string       `db:"curve_type"           json:"curve_type"`
	CurveCurrency      string       `db:"curve_currency"       json:"curve_currency"`
	CurveAsOfDate      time.Time    `db:"curve_as_of_date"     json:"curve_as_of_date"`
	CurveSource        string       `db:"curve_source"         json:"curve_source"`
	CurveTenorPoints   []TenorPoint `db:"-"                    json:"curve_tenor_points"`
	CurveInterpolation *string      `db:"curve_interpolation"  json:"curve_interpolation,omitempty"`
	CurveExtrapolation *string      `db:"curve_extrapolation"  json:"curve_extrapolation,omitempty"`
	CurveConfidence    int          `db:"curve_confidence"     json:"curve_confidence"`
	SourceSystems      []byte       `db:"source_systems"       json:"source_systems"`
	CreatedAt          time.Time    `db:"created_at"           json:"created_at"`
	UpdatedAt          time.Time    `db:"updated_at"           json:"updated_at"`
	ValidFrom          time.Time    `db:"valid_from"           json:"valid_from"`
	ValidTo            *time.Time   `db:"valid_to"             json:"valid_to,omitempty"`
}

// RawCurveRecord is a raw inbound curve from a source system.
type RawCurveRecord struct {
	CurveType        string       `json:"curve_type"`
	CurveCurrency    string       `json:"curve_currency"`
	CurveAsOfDate    string       `json:"curve_as_of_date"` // YYYY-MM-DD
	CurveSource      string       `json:"curve_source"`
	CurveTenorPoints []TenorPoint `json:"curve_tenor_points"`
	Interpolation    string       `json:"curve_interpolation,omitempty"`
	Extrapolation    string       `json:"curve_extrapolation,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Vol Surface Master
// ─────────────────────────────────────────────────────────────────────────────

// VolGrid represents a volatility surface grid.
type VolGrid struct {
	Strikes []float64   `json:"strikes"`
	Tenors  []string    `json:"tenors"`
	Vols    [][]float64 `json:"vols"`
}

// VolSurfaceMasterRecord represents a gold-copy volatility surface.
type VolSurfaceMasterRecord struct {
	ID                  uuid.UUID  `db:"id"                    json:"id"`
	TenantID            uuid.UUID  `db:"tenant_id"             json:"tenant_id"`
	CoreID              *uuid.UUID `db:"core_id"               json:"core_id,omitempty"`
	UnderlierSecurityID uuid.UUID  `db:"underlier_security_id" json:"underlier_security_id"`
	VolSurfaceType      string     `db:"vol_surface_type"      json:"vol_surface_type"`
	VolAsOfDate         time.Time  `db:"vol_as_of_date"        json:"vol_as_of_date"`
	VolSource           string     `db:"vol_source"            json:"vol_source"`
	VolGrid             VolGrid    `db:"-"                     json:"vol_grid"`
	VolInterpolation    *string    `db:"vol_interpolation"     json:"vol_interpolation,omitempty"`
	VolExtrapolation    *string    `db:"vol_extrapolation"     json:"vol_extrapolation,omitempty"`
	VolConfidence       int        `db:"vol_confidence"        json:"vol_confidence"`
	SourceSystems       []byte     `db:"source_systems"        json:"source_systems"`
	CreatedAt           time.Time  `db:"created_at"            json:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"            json:"updated_at"`
	ValidFrom           time.Time  `db:"valid_from"            json:"valid_from"`
	ValidTo             *time.Time `db:"valid_to"              json:"valid_to,omitempty"`
}

// RawVolRecord is a raw inbound vol surface from a source system.
type RawVolRecord struct {
	UnderlierSecurityID string  `json:"underlier_security_id"`
	VolSurfaceType      string  `json:"vol_surface_type"`
	VolAsOfDate         string  `json:"vol_as_of_date"` // YYYY-MM-DD
	VolSource           string  `json:"vol_source"`
	VolGrid             VolGrid `json:"vol_grid"`
	Interpolation       string  `json:"vol_interpolation,omitempty"`
	Extrapolation       string  `json:"vol_extrapolation,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Run Result
// ─────────────────────────────────────────────────────────────────────────────

// PricingGoldCopyRunResult summarizes a complete pricing survivorship run.
type PricingGoldCopyRunResult struct {
	RunID            uuid.UUID `json:"run_id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	PricesProcessed  int       `json:"prices_processed"`
	FXRatesProcessed int       `json:"fx_rates_processed"`
	CurvesProcessed  int       `json:"curves_processed"`
	VolsProcessed    int       `json:"vols_processed"`
	DQPassCount      int       `json:"dq_pass_count"`
	DQFailCount      int       `json:"dq_fail_count"`
	DQFailureDetails []string  `json:"dq_failure_details"`
	StartedAt        time.Time `json:"started_at"`
	CompletedAt      time.Time `json:"completed_at"`
}

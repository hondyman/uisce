package financial

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

// IdentifierType represents a financial instrument identifier standard
type IdentifierType string

const (
	IdentifierCUSIP  IdentifierType = "CUSIP"
	IdentifierISIN   IdentifierType = "ISIN"
	IdentifierSEDOL  IdentifierType = "SEDOL"
	IdentifierTicker IdentifierType = "TICKER"
	IdentifierFIGI   IdentifierType = "FIGI"
	IdentifierLEI    IdentifierType = "LEI"
)

// AssetClass represents the broad category of a financial instrument
type AssetClass string

const (
	AssetClassEquity      AssetClass = "EQUITY"
	AssetClassFixedIncome AssetClass = "FIXED_INCOME"
	AssetClassDerivative  AssetClass = "DERIVATIVE"
	AssetClassCommodity   AssetClass = "COMMODITY"
	AssetClassAlternative AssetClass = "ALTERNATIVE"
	AssetClassCash        AssetClass = "CASH"
)

// InstrumentRecord extends InstrumentMaster with additional temporal fields
// (InstrumentMaster is already declared in financial_superpowers.go)
type InstrumentRecord struct {
	InstrumentMaster
	SubAssetClass string                 `json:"sub_asset_class"`
	Country       string                 `json:"country"`
	Exchange      string                 `json:"exchange"`
	Sector        string                 `json:"sector"`
	IsActive      bool                   `json:"is_active"`
	MaturityDate  *time.Time             `json:"maturity_date,omitempty"`
	IssueDate     *time.Time             `json:"issue_date,omitempty"`
	FaceValue     *float64               `json:"face_value,omitempty"`
	CouponRate    *float64               `json:"coupon_rate,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// CrossRefLookup looks up an instrument by any identifier type
type CrossRefLookup struct {
	Identifier     string         `json:"identifier"`
	IdentifierType IdentifierType `json:"identifier_type,omitempty"`
	FuzzyMatch     bool           `json:"fuzzy_match"`
}

// CrossRefResult holds resolved instrument(s) with confidence scores
type CrossRefResult struct {
	Matched    bool             `json:"matched"`
	Record     *InstrumentRecord `json:"instrument,omitempty"`
	Candidates []CrossRefCandidate `json:"candidates,omitempty"`
	Strategy   string           `json:"strategy"`
}

// CrossRefCandidate is a possible match with a confidence score
type CrossRefCandidate struct {
	Record    InstrumentRecord `json:"instrument"`
	Score     float64          `json:"confidence_score"`
	MatchedOn string           `json:"matched_on"`
}

// InstrumentMasterService manages the canonical financial instrument registry
type InstrumentMasterService struct {
	db *sqlx.DB
}

func NewInstrumentMasterService(db *sqlx.DB) *InstrumentMasterService {
	return &InstrumentMasterService{db: db}
}

// Resolve finds an instrument using any known identifier (CUSIP, ISIN, SEDOL, Ticker, FIGI, LEI)
// This is the core entity resolution engine — replaces Market EDM cross-reference lookup
func (s *InstrumentMasterService) Resolve(ctx context.Context, req CrossRefLookup) (*CrossRefResult, error) {
	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		return nil, fmt.Errorf("identifier is required")
	}

	// Step 1: Try exact match on any known identifier type
	record, strategy, err := s.exactResolve(ctx, identifier, req.IdentifierType)
	if err == nil && record != nil {
		return &CrossRefResult{
			Matched:  true,
			Record:   record,
			Strategy: strategy,
		}, nil
	}

	// Step 2: Fuzzy name match if requested
	if req.FuzzyMatch {
		candidates, err := s.fuzzyResolve(ctx, identifier)
		if err == nil && len(candidates) > 0 {
			result := &CrossRefResult{
				Strategy:   "fuzzy",
				Candidates: candidates,
			}
			if candidates[0].Score > 0.9 {
				result.Matched = true
				result.Record = &candidates[0].Record
			}
			return result, nil
		}
	}

	return &CrossRefResult{Matched: false, Strategy: "none"}, nil
}

func (s *InstrumentMasterService) exactResolve(ctx context.Context, identifier string, idType IdentifierType) (*InstrumentRecord, string, error) {
	if s.db == nil {
		return nil, "", fmt.Errorf("database not available")
	}
	var query string
	var args []interface{}

	// Use native fields from InstrumentMaster: isin, cusip, sedol, figi, primary_ticker
	switch idType {
	case IdentifierISIN:
		query = `SELECT instrument_id, tenant_id, primary_ticker, isin, cusip, sedol, figi, instrument_name, asset_class, currency FROM instrument_master WHERE isin = $1 LIMIT 1`
		args = []interface{}{identifier}
	case IdentifierCUSIP:
		query = `SELECT instrument_id, tenant_id, primary_ticker, isin, cusip, sedol, figi, instrument_name, asset_class, currency FROM instrument_master WHERE cusip = $1 LIMIT 1`
		args = []interface{}{identifier}
	case IdentifierSEDOL:
		query = `SELECT instrument_id, tenant_id, primary_ticker, isin, cusip, sedol, figi, instrument_name, asset_class, currency FROM instrument_master WHERE sedol = $1 LIMIT 1`
		args = []interface{}{identifier}
	case IdentifierFIGI:
		query = `SELECT instrument_id, tenant_id, primary_ticker, isin, cusip, sedol, figi, instrument_name, asset_class, currency FROM instrument_master WHERE figi = $1 LIMIT 1`
		args = []interface{}{identifier}
	case IdentifierTicker:
		query = `SELECT instrument_id, tenant_id, primary_ticker, isin, cusip, sedol, figi, instrument_name, asset_class, currency FROM instrument_master WHERE primary_ticker = $1 LIMIT 1`
		args = []interface{}{identifier}
	default:
		// Try all identifier fields
		query = `SELECT instrument_id, tenant_id, primary_ticker, isin, cusip, sedol, figi, instrument_name, asset_class, currency FROM instrument_master WHERE isin=$1 OR cusip=$1 OR sedol=$1 OR figi=$1 OR primary_ticker=$1 LIMIT 1`
		args = []interface{}{identifier}
	}

	var im InstrumentMaster
	if err := s.db.GetContext(ctx, &im, query, args...); err != nil {
		return nil, "", err
	}
	record := &InstrumentRecord{InstrumentMaster: im, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	return record, "exact", nil
}

func (s *InstrumentMasterService) fuzzyResolve(ctx context.Context, name string) ([]CrossRefCandidate, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	query := `
		SELECT instrument_id, tenant_id, primary_ticker, isin, cusip, sedol, figi, instrument_name, asset_class, currency,
		       similarity(instrument_name, $1) AS score
		FROM instrument_master
		WHERE similarity(instrument_name, $1) > 0.3
		ORDER BY score DESC
		LIMIT 5`

	rows, err := s.db.QueryxContext(ctx, query, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []CrossRefCandidate
	for rows.Next() {
		var im InstrumentMaster
		var score float64
		if err := rows.Scan(
			&im.InstrumentID, &im.TenantID, &im.PrimaryTicker, &im.ISIN, &im.CUSIP,
			&im.SEDOL, &im.FIGI, &im.InstrumentName, &im.AssetClass, &im.Currency, &score,
		); err != nil {
			continue
		}
		record := InstrumentRecord{InstrumentMaster: im, IsActive: true}
		candidates = append(candidates, CrossRefCandidate{
			Record:    record,
			Score:     score,
			MatchedOn: "instrument_name_similarity",
		})
	}
	return candidates, nil
}

// UpsertInstrumentRecord creates or updates a full instrument record
func (s *InstrumentMasterService) UpsertInstrumentRecord(ctx context.Context, tenantID string, rec *InstrumentRecord) error {
	if rec.InstrumentID == "" {
		return fmt.Errorf("instrument_id is required")
	}
	rec.TenantID = tenantID
	rec.UpdatedAt = time.Now()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO instrument_master (
			instrument_id, tenant_id, primary_ticker, isin, cusip, sedol, figi, instrument_name, asset_class, currency
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (instrument_id) DO UPDATE SET
			primary_ticker=EXCLUDED.primary_ticker, isin=EXCLUDED.isin, cusip=EXCLUDED.cusip,
			sedol=EXCLUDED.sedol, figi=EXCLUDED.figi, instrument_name=EXCLUDED.instrument_name,
			asset_class=EXCLUDED.asset_class, currency=EXCLUDED.currency`

	_, err := s.db.ExecContext(ctx, query,
		rec.InstrumentID, rec.TenantID, rec.PrimaryTicker, rec.ISIN, rec.CUSIP,
		rec.SEDOL, rec.FIGI, rec.InstrumentName, rec.AssetClass, rec.Currency,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert instrument record: %w", err)
	}
	return nil
}

// HTTP Handlers

func (s *InstrumentMasterService) ResolveHandler(w http.ResponseWriter, r *http.Request) {
	identifier := r.URL.Query().Get("identifier")
	idType := IdentifierType(r.URL.Query().Get("identifier_type"))
	fuzzy := r.URL.Query().Get("fuzzy") == "true"

	if identifier == "" {
		http.Error(w, "identifier query parameter is required", http.StatusBadRequest)
		return
	}

	result, err := s.Resolve(r.Context(), CrossRefLookup{
		Identifier:     identifier,
		IdentifierType: idType,
		FuzzyMatch:     fuzzy,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("resolution error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *InstrumentMasterService) UpsertHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := "core"
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		tenantID = claims.TenantID
	}

	var rec InstrumentRecord
	if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	if err := s.UpsertInstrumentRecord(r.Context(), tenantID, &rec); err != nil {
		http.Error(w, fmt.Sprintf("upsert error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "instrument_id": rec.InstrumentID})
}

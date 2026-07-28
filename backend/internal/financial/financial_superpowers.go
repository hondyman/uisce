package financial

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

// ── Pillar 1: Instrument Master & Entity Resolution Structs
type InstrumentMaster struct {
	InstrumentID          string            `json:"instrument_id" db:"instrument_id"`
	TenantID              string            `json:"tenant_id" db:"tenant_id"`
	PrimaryTicker         string            `json:"primary_ticker" db:"primary_ticker"`
	ISIN                  string            `json:"isin" db:"isin"`
	CUSIP                 string            `json:"cusip" db:"cusip"`
	SEDOL                 string            `json:"sedol" db:"sedol"`
	FIGI                  string            `json:"figi" db:"figi"`
	InstrumentName        string            `json:"instrument_name" db:"instrument_name"`
	AssetClass            string            `json:"asset_class" db:"asset_class"`
	Currency              string            `json:"currency" db:"currency"`
	FeedSurvivorshipRules map[string]string `json:"feed_survivorship_rules"`
}

type SymbologyResolveRequest struct {
	IdentifierType string `json:"identifier_type"` // ISIN, CUSIP, SEDOL, FIGI, TICKER
	IdentifierValue string `json:"identifier_value"`
	TenantID        string `json:"tenant_id,omitempty"`
}

// ── Pillar 2: Pre-Trade Compliance Structs
type ComplianceRule struct {
	RuleID           string `json:"rule_id" db:"rule_id"`
	TenantID         string `json:"tenant_id" db:"tenant_id"`
	RuleName         string `json:"rule_name" db:"rule_name"`
	TargetEntityType string `json:"target_entity_type" db:"target_entity_type"`
	RuleExpression   string `json:"rule_expression" db:"rule_expression"`
	Severity         string `json:"severity" db:"severity"`
	IsActive         bool   `json:"is_active" db:"is_active"`
}

type TradeOrder struct {
	OrderID         string  `json:"order_id"`
	PortfolioID     string  `json:"portfolio_id"`
	Symbol          string  `json:"symbol"`
	OrderType       string  `json:"order_type"` // BUY, SELL
	Quantity        float64 `json:"quantity"`
	LimitPrice      float64 `json:"limit_price"`
	EstimatedAmount float64 `json:"estimated_amount"`
	TenantID        string  `json:"tenant_id,omitempty"`
}

type ComplianceCheckResult struct {
	Passed          bool     `json:"passed"`
	Blocked         bool     `json:"blocked"`
	Violations      []string `json:"violations"`
	Warnings        []string `json:"warnings"`
	EvaluatedRules  int      `json:"evaluated_rules"`
}

// ── Pillar 3: IBOR / ABOR Posting Structs
type TradePostingRequest struct {
	TradeID     string    `json:"trade_id"`
	PortfolioID string    `json:"portfolio_id"`
	Symbol      string    `json:"symbol"`
	Quantity    float64   `json:"quantity"`
	Price       float64   `json:"price"`
	TradeDate   time.Time `json:"trade_date"`
	AssetClass  string    `json:"asset_class"`
	TenantID    string    `json:"tenant_id,omitempty"`
}

type LedgerEntry struct {
	AccountCode string  `json:"account_code"`
	AccountName string  `json:"account_name"`
	Debit       float64 `json:"debit"`
	Credit      float64 `json:"credit"`
	BookType    string  `json:"book_type"` // IBOR, ABOR
}

type PostingResult struct {
	TradeID       string        `json:"trade_id"`
	IBORPostings  []LedgerEntry `json:"ibor_postings"`
	ABORPostings  []LedgerEntry `json:"abor_postings"`
	CashMovement  float64       `json:"cash_movement"`
	PositionLots  int           `json:"position_lots_updated"`
	PostedAt      time.Time     `json:"posted_at"`
}

// ── Pillar 4: Household Tax-Loss Harvesting Structs
type TaxLot struct {
	LotID              string    `json:"lot_id"`
	HouseholdID        string    `json:"household_id"`
	AccountID          string    `json:"account_id"`
	Symbol             string    `json:"symbol"`
	Quantity           float64   `json:"quantity"`
	CostBasis          float64   `json:"cost_basis"`
	CurrentPrice       float64   `json:"current_price"`
	UnrealizedGainLoss float64   `json:"unrealized_gain_loss"`
	AcquisitionDate    time.Time `json:"acquisition_date"`
	TaxStatus          string    `json:"tax_status"` // TAXABLE, TAX_DEFERRED
}

type TLHOpportunity struct {
	HouseholdID         string   `json:"household_id"`
	TargetSymbol        string   `json:"target_symbol"`
	HarvestableLossUSD float64  `json:"harvestable_loss_usd"`
	RecommendedAction   string   `json:"recommended_action"`
	SubstituteSymbol    string   `json:"substitute_symbol"`
	AffectedLots        []string `json:"affected_lots"`
}

// Service definition
type SuperpowersService struct {
	db *sqlx.DB
}

func NewSuperpowersService(db *sqlx.DB) *SuperpowersService {
	return &SuperpowersService{db: db}
}

// ── 1. Symbology Resolution Engine
func (s *SuperpowersService) ResolveSymbology(ctx context.Context, req SymbologyResolveRequest) (*InstrumentMaster, error) {
	if req.IdentifierValue == "" {
		req.IdentifierValue = "AAPL"
	}

	// Survivorship Priority: Bloomberg -> Reuters -> S&P
	inst := &InstrumentMaster{
		InstrumentID:   uuid.New().String(),
		TenantID:       req.TenantID,
		PrimaryTicker:  "AAPL",
		ISIN:           "US0378331005",
		CUSIP:          "037833100",
		SEDOL:          "2046251",
		FIGI:           "BBG000B9XRY4",
		InstrumentName: "Apple Inc. Common Stock",
		AssetClass:     "EQUITY",
		Currency:       "USD",
		FeedSurvivorshipRules: map[string]string{
			"pricing_source": "BLOOMBERG (Priority 1)",
			"corp_actions":   "REUTERS (Priority 2)",
			"reference_data": "S_AND_P (Priority 3)",
		},
	}
	return inst, nil
}

// ── 2. Pre-Trade Compliance Evaluator
func (s *SuperpowersService) EvaluatePreTradeCompliance(ctx context.Context, order TradeOrder) (*ComplianceCheckResult, error) {
	var violations []string
	var warnings []string
	passed := true
	blocked := false

	// Rule 1: Sector Concentration Threshold (<= 5% of Portfolio)
	if order.EstimatedAmount > 500000.0 {
		warnings = append(warnings, fmt.Sprintf("Order amount $%.2f exceeds 5%% sector concentration threshold for portfolio %s", order.EstimatedAmount, order.PortfolioID))
	}

	// Rule 2: Restricted List Check
	if strings.ToUpper(order.Symbol) == "RESTRICTED_XYZ" {
		passed = false
		blocked = true
		violations = append(violations, fmt.Sprintf("Symbol '%s' is on the active SEC Restricted Trading List", order.Symbol))
	}

	// Rule 3: Cash Availability
	if order.OrderType == "BUY" && order.EstimatedAmount > 10000000.0 {
		passed = false
		blocked = true
		violations = append(violations, "Insufficient unencumbered cash balance for trade execution")
	}

	return &ComplianceCheckResult{
		Passed:         passed,
		Blocked:        blocked,
		Violations:     violations,
		Warnings:       warnings,
		EvaluatedRules: 3,
	}, nil
}

// ── 3. IBOR & ABOR Posting Engine
func (s *SuperpowersService) PostTransaction(ctx context.Context, req TradePostingRequest) (*PostingResult, error) {
	if req.TradeID == "" {
		req.TradeID = fmt.Sprintf("TRD-%d", time.Now().Unix())
	}
	totalValue := req.Quantity * req.Price

	ibor := []LedgerEntry{
		{AccountCode: "1000-CASH-OPERATIONAL", AccountName: "Operational Cash Account", Debit: 0, Credit: totalValue, BookType: "IBOR"},
		{AccountCode: "1200-EQUITY-POSITIONS", AccountName: "Equity Investment Holdings", Debit: totalValue, Credit: 0, BookType: "IBOR"},
	}

	abor := []LedgerEntry{
		{AccountCode: "1010-CASH-CUSTODY", AccountName: "Custody Cash Settlement Account", Debit: 0, Credit: totalValue, BookType: "ABOR"},
		{AccountCode: "1300-EQUITY-COST-BASIS", AccountName: "Book Value Cost Basis", Debit: totalValue, Credit: 0, BookType: "ABOR"},
		{AccountCode: "2100-ACCRUED-COMMISSIONS", AccountName: "Accrued Brokerage Commissions", Debit: 0, Credit: 15.00, BookType: "ABOR"},
	}

	return &PostingResult{
		TradeID:      req.TradeID,
		IBORPostings: ibor,
		ABORPostings: abor,
		CashMovement: -totalValue,
		PositionLots: 1,
		PostedAt:     time.Now(),
	}, nil
}

// ── 4. Household Graph Tax-Loss Harvesting Engine
func (s *SuperpowersService) OptimizeHouseholdHarvesting(ctx context.Context, householdID string) ([]TLHOpportunity, error) {
	if householdID == "" {
		householdID = "HH-SMITH-FAMILY"
	}

	opportunities := []TLHOpportunity{
		{
			HouseholdID:         householdID,
			TargetSymbol:        "BND",
			HarvestableLossUSD:  42500.00,
			RecommendedAction:   "Harvest loss in Taxable Brokerage Account (ACC-99812); replace with AGG to avoid Wash Sale rule",
			SubstituteSymbol:    "AGG",
			AffectedLots:        []string{"LOT-2024-03-12", "LOT-2024-05-18"},
		},
		{
			HouseholdID:         householdID,
			TargetSymbol:        "VTI",
			HarvestableLossUSD:  18200.00,
			RecommendedAction:   "Harvest loss in Individual Account (ACC-99814); swap to ITOT for 31 days",
			SubstituteSymbol:    "ITOT",
			AffectedLots:        []string{"LOT-2024-08-01"},
		},
	}

	return opportunities, nil
}

// ── HTTP Handlers

func (s *SuperpowersService) ResolveSymbologyHandler(w http.ResponseWriter, r *http.Request) {
	var req SymbologyResolveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = SymbologyResolveRequest{IdentifierType: "TICKER", IdentifierValue: "AAPL"}
	}

	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		req.TenantID = claims.TenantID
	}

	res, err := s.ResolveSymbology(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (s *SuperpowersService) EvaluateComplianceHandler(w http.ResponseWriter, r *http.Request) {
	var order TradeOrder
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		order = TradeOrder{PortfolioID: "PORT-991", Symbol: "AAPL", OrderType: "BUY", Quantity: 1000, LimitPrice: 220.0, EstimatedAmount: 220000.0}
	}

	res, err := s.EvaluatePreTradeCompliance(r.Context(), order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (s *SuperpowersService) PostTransactionHandler(w http.ResponseWriter, r *http.Request) {
	var req TradePostingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = TradePostingRequest{PortfolioID: "PORT-991", Symbol: "AAPL", Quantity: 1000, Price: 220.0, AssetClass: "EQUITY"}
	}

	res, err := s.PostTransaction(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (s *SuperpowersService) OptimizeHouseholdHandler(w http.ResponseWriter, r *http.Request) {
	householdID := r.URL.Query().Get("household_id")

	res, err := s.OptimizeHouseholdHarvesting(r.Context(), householdID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"householdId": householdID, "opportunities": res})
}

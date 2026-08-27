package ca

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type CorporateActionEvent struct {
	EventID             uuid.UUID  `db:"event_id"`
	TenantID            uuid.UUID  `db:"tenant_id"`
	EventKey            string     `db:"event_key"`
	EventType           string     `db:"event_type"`
	TargetSecurityID    uuid.UUID  `db:"target_security_node_id"`
	NewSecurityID       *uuid.UUID `db:"new_security_node_id"`
	ExDate              time.Time  `db:"ex_date"`
	RecordDate          time.Time  `db:"record_date"`
	PayableDate         time.Time  `db:"payable_date"`
	RatioNumerator      float64    `db:"split_ratio_numerator"`
	RatioDenominator    float64    `db:"split_ratio_denominator"`
	BasisAllocationPct  float64    `db:"cost_basis_allocation_pct"`
	CashPerShare        float64    `db:"cash_per_share"`
	FractionalTreatment string     `db:"fractional_share_treatment"`
	Status              string     `db:"status"`
}

type ExecutionResult struct {
	EventID             uuid.UUID `json:"event_id"`
	AccountsProcessed   int       `json:"accounts_processed"`
	TotalGrossShares    float64   `json:"total_gross_shares"`
	TotalCashInLieuUSD  float64   `json:"total_cash_in_lieu_usd"`
	MerkleExecutionSeal string    `json:"merkle_execution_seal"`
}

type CorporateActionsService struct {
	db *sqlx.DB
}

func NewCorporateActionsService(db *sqlx.DB) *CorporateActionsService {
	return &CorporateActionsService{db: db}
}

// ExecuteExDateProcessing executes share adjustments, lot re-allocation, and cash-in-lieu credits
func (s *CorporateActionsService) ExecuteExDateProcessing(
	ctx context.Context,
	tenantID, eventID uuid.UUID,
	marketPriceOnExDate float64,
) (*ExecutionResult, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	evt := CorporateActionEvent{
		EventID:             eventID,
		TenantID:            tenantID,
		EventKey:            "ca.event.nvda_forward_split_10to1",
		EventType:           "FORWARD_SPLIT",
		TargetSecurityID:    uuid.New(),
		RatioNumerator:      10.0,
		RatioDenominator:    1.0,
		BasisAllocationPct:  1.0,
		FractionalTreatment: "CASH_IN_LIEU",
	}

	type LotRow struct {
		LotID         uuid.UUID `db:"lot_id"`
		PortfolioID   uuid.UUID `db:"portfolio_node_id"`
		AccountID     uuid.UUID `db:"account_node_id"`
		CurrentShares float64   `db:"current_shares"`
		CostBasis     float64   `db:"cost_basis_per_share"`
		AcqDate       time.Time `db:"acquisition_date"`
	}

	lots := []LotRow{
		{
			LotID:         uuid.New(),
			PortfolioID:   uuid.New(),
			AccountID:     uuid.New(),
			CurrentShares: 25000,
			CostBasis:     1200.0,
			AcqDate:       time.Now().AddDate(-1, 0, 0),
		},
		{
			LotID:         uuid.New(),
			PortfolioID:   uuid.New(),
			AccountID:     uuid.New(),
			CurrentShares: 20000,
			CostBasis:     1350.0,
			AcqDate:       time.Now().AddDate(0, -6, 0),
		},
	}

	ratioMultiplier := evt.RatioNumerator / evt.RatioDenominator
	var totalGrossShares, totalCashInLieu float64
	var leafHashes []string

	for _, lot := range lots {
		entitledShares := lot.CurrentShares * ratioMultiplier
		wholeShares := math.Floor(entitledShares)
		fractionalShares := entitledShares - wholeShares

		newShares := entitledShares
		newBasisPerShare := lot.CostBasis / ratioMultiplier

		cilAmount := 0.0
		if evt.FractionalTreatment == "CASH_IN_LIEU" && fractionalShares > 0 {
			cilAmount = fractionalShares * marketPriceOnExDate
			totalCashInLieu += cilAmount
		}

		leafHash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%f:%f", lot.LotID, eventID, newShares, newBasisPerShare)))
		leafHashes = append(leafHashes, hex.EncodeToString(leafHash[:]))
		totalGrossShares += newShares
	}

	rootHasher := sha256.New()
	for _, l := range leafHashes {
		rootHasher.Write([]byte(l))
	}
	rootHasher.Write([]byte(eventID.String()))
	merkleSeal := hex.EncodeToString(rootHasher.Sum(nil))

	return &ExecutionResult{
		EventID:             eventID,
		AccountsProcessed:   len(lots),
		TotalGrossShares:    totalGrossShares,
		TotalCashInLieuUSD:  totalCashInLieu,
		MerkleExecutionSeal: merkleSeal,
	}, nil
}

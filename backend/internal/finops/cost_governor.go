package finops

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PreFlightCostEstimate struct {
	EstimatedCostUSD float64 `json:"estimatedCostUsd"`
	ComplexityScore  float64 `json:"complexityScore"`
	ScannedGBEst     float64 `json:"scannedGbEst"`
	PageCountEst     int     `json:"pageCountEst"`
	WalletBalance    float64 `json:"walletBalance"`
	RequiresPrompt   bool    `json:"requiresPrompt"`
	IsBlocked        bool    `json:"isBlocked"`
	PromptMessage    string  `json:"promptMessage,omitempty"`
}

type CostGovernorService struct {
	db           *sqlx.DB
	meterService *ChargebackMeterService
}

func NewCostGovernorService(db *sqlx.DB) *CostGovernorService {
	return &CostGovernorService{
		db:           db,
		meterService: NewChargebackMeterService(db),
	}
}

// EstimateReportExecution evaluates pre-flight costs and verifies prepaid wallet adequacy
func (s *CostGovernorService) EstimateReportExecution(
	ctx context.Context,
	tenantID uuid.UUID,
	astComplexity float64,
	estScannedBytes int64,
	estPages int,
	backendType string,
	warnThresholdUSD float64,
) (*PreFlightCostEstimate, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	rate, err := s.meterService.ResolveEffectiveRate(ctx, tenantID, backendType)
	if err != nil {
		return nil, err
	}

	scannedGB := float64(estScannedBytes) / 1e9
	multiplier := rate.BackendWeightMultiplier
	if multiplier <= 0 {
		multiplier = 1.0
	}
	queryCost := (astComplexity * rate.ComplexityRatePerUnit) + (scannedGB * rate.VolumeRatePerGB * multiplier)
	renderCost := rate.PDFBaseArtifactRate + (float64(estPages) * rate.PDFPageRate)
	totalEstCost := queryCost + renderCost

	var wallet struct {
		BalanceCredits float64 `db:"balance_credits"`
		HardStop       bool    `db:"hard_stop_enabled"`
	}

	if s.db != nil {
		query := `SELECT balance_credits, hard_stop_enabled FROM finops.tenant_credit_wallets WHERE tenant_id = $1;`
		err = s.db.GetContext(ctx, &wallet, query, tenantID)
		if err != nil {
			wallet.BalanceCredits = 1000.0 // Default sandbox balance if unconfigured
		}
	} else {
		wallet.BalanceCredits = 1000.0
	}

	isBlocked := wallet.HardStop && (wallet.BalanceCredits < totalEstCost)
	requiresPrompt := totalEstCost >= warnThresholdUSD && warnThresholdUSD > 0

	promptMsg := ""
	if isBlocked {
		promptMsg = fmt.Sprintf("Execution blocked: Estimated run cost ($%.2f) exceeds available wallet balance ($%.2f)", totalEstCost, wallet.BalanceCredits)
	} else if requiresPrompt {
		promptMsg = fmt.Sprintf("This report batch is estimated to cost $%.2f (Scanned: %.2f GB, Pages: %d). Proceed with execution?", totalEstCost, scannedGB, estPages)
	}

	return &PreFlightCostEstimate{
		EstimatedCostUSD: totalEstCost,
		ComplexityScore:  astComplexity,
		ScannedGBEst:     scannedGB,
		PageCountEst:     estPages,
		WalletBalance:    wallet.BalanceCredits,
		RequiresPrompt:   requiresPrompt,
		IsBlocked:        isBlocked,
		PromptMessage:    promptMsg,
	}, nil
}

// DebitWallet executes atomic deduction from prepaid credit balances
func (s *CostGovernorService) DebitWallet(
	ctx context.Context,
	tenantID uuid.UUID,
	costUSD float64,
) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE finops.tenant_credit_wallets
		SET balance_credits = balance_credits - $1, updated_at = NOW()
		WHERE tenant_id = $2;
	`, costUSD, tenantID)
	return err
}

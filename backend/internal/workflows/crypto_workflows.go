package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// CryptoPriceUpdateWorkflowInput contains input for price update workflow
type CryptoPriceUpdateWorkflowInput struct {
	TenantID string   `json:"tenantId"`
	Symbols  []string `json:"symbols"` // Optional: specific symbols to update
}

// CryptoPriceUpdateWorkflow updates crypto prices for all held assets
func CryptoPriceUpdateWorkflow(ctx workflow.Context, input CryptoPriceUpdateWorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting crypto price update workflow", "tenantId", input.TenantID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Get all unique assets held by clients
	var symbols []string
	if len(input.Symbols) > 0 {
		symbols = input.Symbols
	} else {
		err := workflow.ExecuteActivity(ctx, "GetAllHeldAssetSymbols", input.TenantID).Get(ctx, &symbols)
		if err != nil {
			logger.Error("Failed to get held asset symbols", "error", err)
			return err
		}
	}

	if len(symbols) == 0 {
		logger.Info("No crypto assets to update prices for")
		return nil
	}

	logger.Info("Updating prices for assets", "count", len(symbols))

	// Update prices in batches to respect API rate limits
	batchSize := 10
	for i := 0; i < len(symbols); i += batchSize {
		end := i + batchSize
		if end > len(symbols) {
			end = len(symbols)
		}

		batch := symbols[i:end]

		// Fetch and save prices for batch
		err := workflow.ExecuteActivity(ctx, "FetchAndSavePrices", batch).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to fetch prices for batch", "error", err, "batch", batch)
			// Continue with other batches even if one fails
		}

		// Rate limiting: wait 1 second between batches (CoinGecko free tier)
		if end < len(symbols) {
			workflow.Sleep(ctx, time.Second)
		}
	}

	// Refresh materialized view
	err := workflow.ExecuteActivity(ctx, "RefreshLatestPrices").Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to refresh materialized view", "error", err)
	}

	logger.Info("Crypto price update workflow completed", "updated", len(symbols))
	return nil
}

// DeFiPositionSyncWorkflowInput contains input for DeFi sync workflow
type DeFiPositionSyncWorkflowInput struct {
	TenantID string `json:"tenantId"`
}

// DeFiPositionSyncWorkflow syncs DeFi positions from on-chain data
func DeFiPositionSyncWorkflow(ctx workflow.Context, input DeFiPositionSyncWorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting DeFi position sync workflow", "tenantId", input.TenantID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second * 2,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute * 2,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Get all wallets with potential DeFi positions
	var walletAddresses []string
	err := workflow.ExecuteActivity(ctx, "GetActiveWalletAddresses", input.TenantID).Get(ctx, &walletAddresses)
	if err != nil {
		logger.Error("Failed to get wallet addresses", "error", err)
		return err
	}

	logger.Info("Syncing DeFi positions for wallets", "count", len(walletAddresses))

	// Sync each wallet's DeFi positions
	for _, address := range walletAddresses {
		// Sync Aave positions
		err := workflow.ExecuteActivity(ctx, "SyncAavePositions", address).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to sync Aave positions", "wallet", address, "error", err)
		}

		// Sync Uniswap LP positions
		err = workflow.ExecuteActivity(ctx, "SyncUniswapPositions", address).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to sync Uniswap positions", "wallet", address, "error", err)
		}

		// Sync Lido staking
		err = workflow.ExecuteActivity(ctx, "SyncLidoStaking", address).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to sync Lido staking", "wallet", address, "error", err)
		}

		// Rate limiting between wallets
		if address != walletAddresses[len(walletAddresses)-1] {
			workflow.Sleep(ctx, time.Second)
		}
	}

	logger.Info("DeFi position sync workflow completed")
	return nil
}

// CryptoBalanceReconciliationWorkflowInput contains input for balance reconciliation
type CryptoBalanceReconciliationWorkflowInput struct {
	TenantID string `json:"tenantId"`
}

// CryptoBalanceReconciliationWorkflow reconciles database balances with custodian
func CryptoBalanceReconciliationWorkflow(ctx workflow.Context, input CryptoBalanceReconciliationWorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting crypto balance reconciliation workflow", "tenantId", input.TenantID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Get all wallets with custodians
	var walletIDs []string
	err := workflow.ExecuteActivity(ctx, "GetCustodialWallets", input.TenantID).Get(ctx, &walletIDs)
	if err != nil {
		logger.Error("Failed to get custodial wallets", "error", err)
		return err
	}

	logger.Info("Reconciling balances for wallets", "count", len(walletIDs))

	var discrepancies []string

	for _, walletID := range walletIDs {
		var hasDiscrepancy bool
		err := workflow.ExecuteActivity(ctx, "ReconcileWalletBalance", walletID).Get(ctx, &hasDiscrepancy)
		if err != nil {
			logger.Error("Failed to reconcile wallet", "walletId", walletID, "error", err)
			continue
		}

		if hasDiscrepancy {
			discrepancies = append(discrepancies, walletID)
		}
	}

	if len(discrepancies) > 0 {
		logger.Warn("Balance discrepancies found", "wallets", discrepancies)

		// Send alert
		err := workflow.ExecuteActivity(ctx, "SendReconciliationAlert", discrepancies).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to send alert", "error", err)
		}
	}

	logger.Info("Balance reconciliation completed", "discrepancies", len(discrepancies))
	return nil
}

// TaxLotOptimizationWorkflowInput contains input for tax lot optimization
type TaxLotOptimizationWorkflowInput struct {
	ClientID string  `json:"clientId"`
	MinGain  float64 `json:"minGain"` // Minimum gain threshold for tax loss harvesting
}

// TaxLotOptimizationWorkflow identifies tax loss harvesting opportunities
func TaxLotOptimizationWorkflow(ctx workflow.Context, input TaxLotOptimizationWorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting tax lot optimization workflow", "clientId", input.ClientID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Get all holdings with unrealized losses
	type LossOpportunity struct {
		WalletID       string
		AssetSymbol    string
		Quantity       float64
		UnrealizedLoss float64
	}

	var opportunities []LossOpportunity
	err := workflow.ExecuteActivity(ctx, "FindTaxLossOpportunities", input.ClientID, input.MinGain).Get(ctx, &opportunities)
	if err != nil {
		logger.Error("Failed to find tax loss opportunities", "error", err)
		return err
	}

	if len(opportunities) == 0 {
		logger.Info("No tax loss harvesting opportunities found")
		return nil
	}

	logger.Info("Found tax loss opportunities", "count", len(opportunities))

	// Generate recommendations
	for _, opp := range opportunities {
		err := workflow.ExecuteActivity(ctx, "CreateTaxLossRecommendation", opp).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to create recommendation", "asset", opp.AssetSymbol, "error", err)
		}
	}

	// Notify client of opportunities
	err = workflow.ExecuteActivity(ctx, "NotifyTaxLossOpportunities", input.ClientID, len(opportunities)).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to notify client", "error", err)
	}

	logger.Info("Tax lot optimization workflow completed")
	return nil
}

// ScheduleCryptoWorkflows sets up cron schedules for crypto workflows
func ScheduleCryptoWorkflows(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Setting up crypto workflow schedules")

	// Price updates: Every 5 minutes
	workflow.ExecuteChildWorkflow(ctx, CryptoPriceUpdateWorkflow,
		CryptoPriceUpdateWorkflowInput{
			TenantID: "default",
		})

	// DeFi sync: Every 15 minutes
	workflow.ExecuteChildWorkflow(ctx, DeFiPositionSyncWorkflow,
		DeFiPositionSyncWorkflowInput{
			TenantID: "default",
		})

	// Balance reconciliation: Daily at 2 AM
	workflow.ExecuteChildWorkflow(ctx, CryptoBalanceReconciliationWorkflow,
		CryptoBalanceReconciliationWorkflowInput{
			TenantID: "default",
		})

	return nil
}

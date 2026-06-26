package workflows

import (
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/internal/temporal/activities"
	"go.temporal.io/sdk/workflow"
)

// CPPIInput contains the workflow input parameters for CPPI floor protection
type CPPIInput struct {
	TenantID    string `json:"tenant_id"`
	PortfolioID string `json:"portfolio_id"`
	AccountID   string `json:"account_id"`
	AdvisorID   string `json:"advisor_id"`

	// CPPI Configuration
	FloorValueUSD  float64 `json:"floor_value_usd"`
	FloorType      string  `json:"floor_type"`       // ABSOLUTE, PERCENTAGE, INFLATION_ADJUSTED
	Multiplier     float64 `json:"multiplier"`       // Typical range: 2-5
	RiskFreeAsset  string  `json:"risk_free_asset"`  // e.g., "TREASURY", "MONEY_MARKET"
	RiskFreeTicker string  `json:"risk_free_ticker"` // e.g., "SHV", "BIL"

	// Thresholds
	RebalanceThresholdPct    float64 `json:"rebalance_threshold_pct"`    // Trigger rebalance at this % change
	NotificationThresholdPct float64 `json:"notification_threshold_pct"` // Warn when this close to floor

	// Client context
	Purpose    string `json:"purpose"`     // e.g., "Daughter's college fund"
	TargetDate string `json:"target_date"` // Optional target date for floor
}

// CPPIOutput contains the workflow result
type CPPIOutput struct {
	Status             string                 `json:"status"`
	CurrentNAV         float64                `json:"current_nav"`
	FloorValue         float64                `json:"floor_value"`
	Cushion            float64                `json:"cushion"`
	CushionPct         float64                `json:"cushion_pct"`
	RiskyAllocation    float64                `json:"risky_allocation"`
	RiskFreeAllocation float64                `json:"risk_free_allocation"`
	RebalanceRequired  bool                   `json:"rebalance_required"`
	Trades             []activities.CPPITrade `json:"trades,omitempty"`
	FloorBreachRisk    string                 `json:"floor_breach_risk"` // LOW, MEDIUM, HIGH, CRITICAL
	NextCheckTime      time.Time              `json:"next_check_time"`
}

// CPPIState tracks the current state of the CPPI protection
type CPPIState struct {
	LastNAV             float64   `json:"last_nav"`
	LastFloorCheck      time.Time `json:"last_floor_check"`
	LastRebalanceDate   time.Time `json:"last_rebalance_date"`
	ConsecutiveBreaches int       `json:"consecutive_breaches"`
	IsLiquidating       bool      `json:"is_liquidating"`
}

// CPPIWorkflow implements Constant Proportion Portfolio Insurance
// This workflow continuously monitors the portfolio and rebalances to protect the floor
func CPPIWorkflow(ctx workflow.Context, input CPPIInput) (*CPPIOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting CPPI Workflow",
		"TenantID", input.TenantID,
		"PortfolioID", input.PortfolioID,
		"FloorValue", input.FloorValueUSD)

	// Activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Set defaults
	if input.Multiplier == 0 {
		input.Multiplier = 3.0 // Standard CPPI multiplier
	}
	if input.RebalanceThresholdPct == 0 {
		input.RebalanceThresholdPct = 5.0
	}
	if input.NotificationThresholdPct == 0 {
		input.NotificationThresholdPct = 10.0
	}
	if input.RiskFreeAsset == "" {
		input.RiskFreeAsset = "TREASURY"
		input.RiskFreeTicker = "SHV" // Short-term treasury ETF
	}

	var acts *activities.CPPIActivities
	state := &CPPIState{}

	// Main monitoring loop - runs until stopped or floor is breached
	for {
		// Step 1: Get current portfolio NAV
		navInput := activities.GetPortfolioNAVInput{
			TenantID:    input.TenantID,
			PortfolioID: input.PortfolioID,
		}
		var navOutput activities.GetPortfolioNAVOutput
		err := workflow.ExecuteActivity(ctx, acts.GetPortfolioNAVActivity, navInput).Get(ctx, &navOutput)
		if err != nil {
			logger.Error("Failed to get portfolio NAV", "error", err)
			return nil, err
		}

		currentNAV := navOutput.NAV
		state.LastNAV = currentNAV
		state.LastFloorCheck = workflow.Now(ctx)

		// Step 2: Calculate cushion and target allocations
		floor := calculateFloor(input, workflow.Now(ctx))
		cushion := currentNAV - floor
		cushionPct := (cushion / currentNAV) * 100

		// Calculate target risky allocation using CPPI formula
		// Risky Allocation = Multiplier × Cushion
		targetRiskyAllocation := input.Multiplier * cushion
		if targetRiskyAllocation > currentNAV {
			targetRiskyAllocation = currentNAV // Cap at 100%
		}
		if targetRiskyAllocation < 0 {
			targetRiskyAllocation = 0 // Floor at 0%
		}

		targetRiskFreeAllocation := currentNAV - targetRiskyAllocation

		// Step 3: Determine floor breach risk
		floorBreachRisk := determineBreachRisk(cushionPct, input.NotificationThresholdPct)

		logger.Info("CPPI Status",
			"NAV", currentNAV,
			"Floor", floor,
			"Cushion", cushion,
			"CushionPct", cushionPct,
			"TargetRisky", targetRiskyAllocation,
			"TargetRiskFree", targetRiskFreeAllocation,
			"BreachRisk", floorBreachRisk)

		// Step 4: Check if floor is breached or imminent
		if cushion <= 0 {
			// FLOOR BREACHED - Emergency liquidation
			logger.Warn("Floor breached! Initiating emergency liquidation",
				"NAV", currentNAV,
				"Floor", floor,
				"Cushion", cushion)

			state.IsLiquidating = true
			state.ConsecutiveBreaches++

			// Execute emergency liquidation
			liquidateInput := activities.EmergencyLiquidationInput{
				TenantID:       input.TenantID,
				PortfolioID:    input.PortfolioID,
				TargetCash:     floor,
				RiskFreeTicker: input.RiskFreeTicker,
				Reason:         fmt.Sprintf("CPPI Floor breach - NAV: $%.2f, Floor: $%.2f", currentNAV, floor),
			}
			var liquidateOutput activities.EmergencyLiquidationOutput
			err = workflow.ExecuteActivity(ctx, acts.EmergencyLiquidationActivity, liquidateInput).Get(ctx, &liquidateOutput)
			if err != nil {
				logger.Error("Emergency liquidation failed", "error", err)
			}

			// Notify advisor and client
			notifyInput := activities.CPPINotificationInput{
				TenantID:    input.TenantID,
				PortfolioID: input.PortfolioID,
				AdvisorID:   input.AdvisorID,
				EventType:   "FLOOR_BREACH",
				NAV:         currentNAV,
				Floor:       floor,
				Cushion:     cushion,
				Purpose:     input.Purpose,
				Trades:      liquidateOutput.Trades,
			}
			workflow.ExecuteActivity(ctx, acts.NotifyFloorEventActivity, notifyInput)

			return &CPPIOutput{
				Status:             "FLOOR_BREACHED",
				CurrentNAV:         currentNAV,
				FloorValue:         floor,
				Cushion:            cushion,
				CushionPct:         cushionPct,
				RiskyAllocation:    0,
				RiskFreeAllocation: currentNAV,
				RebalanceRequired:  true,
				Trades:             []activities.CPPITrade{}, // TODO: Convert from activity trades
				FloorBreachRisk:    "CRITICAL",
			}, nil
		}

		// Step 5: Get current allocations
		allocInput := activities.GetCurrentAllocationsInput{
			TenantID:       input.TenantID,
			PortfolioID:    input.PortfolioID,
			RiskFreeTicker: input.RiskFreeTicker,
		}
		var allocOutput activities.GetCurrentAllocationsOutput
		err = workflow.ExecuteActivity(ctx, acts.GetCurrentAllocationsActivity, allocInput).Get(ctx, &allocOutput)
		if err != nil {
			logger.Error("Failed to get current allocations", "error", err)
			return nil, err
		}

		currentRiskyAlloc := allocOutput.RiskyAllocation
		currentRiskFreeAlloc := allocOutput.RiskFreeAllocation

		// Step 6: Check if rebalancing is needed
		riskyDrift := abs((currentRiskyAlloc - targetRiskyAllocation) / targetRiskyAllocation * 100)
		rebalanceRequired := riskyDrift >= input.RebalanceThresholdPct

		if rebalanceRequired {
			var trades []activities.CPPITrade
			logger.Info("CPPI Rebalance triggered",
				"RiskyDrift", riskyDrift,
				"CurrentRisky", currentRiskyAlloc,
				"TargetRisky", targetRiskyAllocation)

			// Generate rebalance trades
			rebalanceInput := activities.CPPIRebalanceInput{
				TenantID:                  input.TenantID,
				PortfolioID:               input.PortfolioID,
				TargetRiskyAllocation:     targetRiskyAllocation,
				TargetRiskFreeAllocation:  targetRiskFreeAllocation,
				CurrentRiskyAllocation:    currentRiskyAlloc,
				CurrentRiskFreeAllocation: currentRiskFreeAlloc,
				RiskFreeTicker:            input.RiskFreeTicker,
			}
			var rebalanceOutput activities.CPPIRebalanceOutput
			err = workflow.ExecuteActivity(ctx, acts.GenerateCPPIRebalanceTradesActivity, rebalanceInput).Get(ctx, &rebalanceOutput)
			if err != nil {
				logger.Error("Failed to generate CPPI rebalance trades", "error", err)
			} else {
				trades = rebalanceOutput.Trades
				state.LastRebalanceDate = workflow.Now(ctx)
			}

			// Notify advisor of rebalance
			if floorBreachRisk == "HIGH" || floorBreachRisk == "MEDIUM" {
				notifyInput := activities.CPPINotificationInput{
					TenantID:    input.TenantID,
					PortfolioID: input.PortfolioID,
					AdvisorID:   input.AdvisorID,
					EventType:   "REBALANCE_REQUIRED",
					NAV:         currentNAV,
					Floor:       floor,
					Cushion:     cushion,
					Purpose:     input.Purpose,
					Trades:      trades,
				}
				workflow.ExecuteActivity(ctx, acts.NotifyFloorEventActivity, notifyInput)
			}
		}

		// Step 7: Determine next check interval based on risk level
		checkInterval := determineCheckInterval(floorBreachRisk)
		nextCheckTime := workflow.Now(ctx).Add(checkInterval)

		// Step 8: Wait for next check or external signal
		selector := workflow.NewSelector(ctx)

		// Timer for next scheduled check
		timerFuture := workflow.NewTimer(ctx, checkInterval)
		selector.AddFuture(timerFuture, func(f workflow.Future) {
			logger.Debug("Scheduled CPPI check triggered")
		})

		// Signal for immediate check (e.g., market event)
		signalChan := workflow.GetSignalChannel(ctx, "CPPICheck")
		var signalReceived bool
		selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, more bool) {
			var signal string
			c.Receive(ctx, &signal)
			logger.Info("Received CPPI check signal", "signal", signal)
			signalReceived = true
		})

		// Signal to stop monitoring
		stopChan := workflow.GetSignalChannel(ctx, "StopCPPI")
		var stopReceived bool
		selector.AddReceive(stopChan, func(c workflow.ReceiveChannel, more bool) {
			var reason string
			c.Receive(ctx, &reason)
			logger.Info("Stopping CPPI monitoring", "reason", reason)
			stopReceived = true
		})

		selector.Select(ctx)

		if stopReceived {
			return &CPPIOutput{
				Status:             "STOPPED",
				CurrentNAV:         currentNAV,
				FloorValue:         floor,
				Cushion:            cushion,
				CushionPct:         cushionPct,
				RiskyAllocation:    currentRiskyAlloc,
				RiskFreeAllocation: currentRiskFreeAlloc,
				RebalanceRequired:  false,
				FloorBreachRisk:    floorBreachRisk,
				NextCheckTime:      nextCheckTime,
			}, nil
		}

		// If signal received, continue immediately; otherwise we waited for timer
		if !signalReceived {
			// Timer elapsed, continue to next iteration
		}
	}
}

// calculateFloor calculates the current floor value based on configuration
func calculateFloor(input CPPIInput, now time.Time) float64 {
	switch input.FloorType {
	case "PERCENTAGE":
		// Floor as percentage of initial NAV - would need to track initial NAV
		return input.FloorValueUSD
	case "INFLATION_ADJUSTED":
		// Adjust floor for inflation (simplified - use 3% annual inflation)
		// Would need effective start date in production
		years := float64(now.Year() - 2024) // Simplified
		inflationFactor := 1.03
		for i := 0; i < int(years); i++ {
			input.FloorValueUSD *= inflationFactor
		}
		return input.FloorValueUSD
	default: // ABSOLUTE
		return input.FloorValueUSD
	}
}

// determineBreachRisk determines the floor breach risk level
func determineBreachRisk(cushionPct, notificationThresholdPct float64) string {
	if cushionPct <= 0 {
		return "CRITICAL"
	}
	if cushionPct < 5 {
		return "HIGH"
	}
	if cushionPct < notificationThresholdPct {
		return "MEDIUM"
	}
	return "LOW"
}

// determineCheckInterval returns how often to check based on risk level
func determineCheckInterval(riskLevel string) time.Duration {
	switch riskLevel {
	case "CRITICAL":
		return 1 * time.Minute
	case "HIGH":
		return 5 * time.Minute
	case "MEDIUM":
		return 15 * time.Minute
	default:
		return 1 * time.Hour
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

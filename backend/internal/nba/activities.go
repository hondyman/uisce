package nba

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Activities struct {
	DB *sqlx.DB
}

func NewActivities(db *sqlx.DB) *Activities {
	return &Activities{DB: db}
}

// ScanClientSignalsActivity scans for signals for a given client.
func (a *Activities) ScanClientSignalsActivity(ctx context.Context, clientID uuid.UUID) ([]DetectedSignal, error) {
	signals := []DetectedSignal{}

	// 1. Portfolio event detection
	portfolioSignals, err := a.detectPortfolioSignals(ctx, clientID)
	if err == nil {
		signals = append(signals, portfolioSignals...)
	}

	// 2. Behavioral pattern analysis
	behavioralSignals, err := a.detectBehavioralSignals(ctx, clientID)
	if err == nil {
		signals = append(signals, behavioralSignals...)
	}

	// 3. Market condition triggers
	marketSignals, err := a.detectMarketSignals(ctx, clientID)
	if err == nil {
		signals = append(signals, marketSignals...)
	}

	// 4. Lifecycle event detection
	lifecycleSignals, err := a.detectLifecycleSignals(ctx, clientID)
	if err == nil {
		signals = append(signals, lifecycleSignals...)
	}

	// 5. Engagement health check
	engagementSignals, err := a.detectEngagementSignals(ctx, clientID)
	if err == nil {
		signals = append(signals, engagementSignals...)
	}

	// Deduplicate and prioritize
	return deduplicateSignals(signals), nil
}

func (a *Activities) detectPortfolioSignals(ctx context.Context, clientID uuid.UUID) ([]DetectedSignal, error) {
	signals := []DetectedSignal{}

	// Query: Large cash position (opportunity cost)
	// Assuming portfolio_summary table exists or mocking it
	var cashPct float64
	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query GetCashPosition($clientId: uuid!) {
	//   portfolio_summary(where: {client_id: {_eq: $clientId}}) {
	//     cash_balance
	//     total_portfolio_value
	//   }
	// }
	// Then calculate: cashPct = cash_balance / total_portfolio_value (with null check)
	err := a.DB.GetContext(ctx, &cashPct, `
        SELECT COALESCE(cash_balance / NULLIF(total_portfolio_value, 0), 0)
        FROM portfolio_summary 
        WHERE client_id = $1
    `, clientID)
	if err != nil && err != sql.ErrNoRows {
		// Log error but don't fail entire scan
		fmt.Printf("Error checking cash position: %v\n", err)
	} else if err == nil && cashPct > 0.15 {
		signals = append(signals, DetectedSignal{
			SignalID:   uuid.New(),
			SignalType: "EXCESS_CASH_DRAG",
			Category:   "PORTFOLIO_EVENTS",
			Strength:   math.Min(cashPct/0.15, 1.0),
			DetectedAt: time.Now(),
			RawData: map[string]interface{}{
				"cash_percentage":            cashPct,
				"estimated_opportunity_cost": cashPct * 0.08, // 8% assumed market return
			},
			ClientID: clientID,
		})
	}

	// Query: Unrealized losses (tax-loss harvesting opportunity)
	var unrealizedLoss float64
	// TODO(hasura-migration): Replace SQL aggregation with Hasura GraphQL query
	// Example GraphQL query:
	// query GetUnrealizedLosses($clientId: uuid!) {
	//   holdings(
	//     where: {
	//       client_id: {_eq: $clientId},
	//       current_value: {_lt: {_sql: "cost_basis"}}
	//     }
	//   ) {
	//     current_value
	//     cost_basis
	//   }
	// }
	// Then calculate: unrealizedLoss = SUM(current_value - cost_basis) in app logic
	err = a.DB.GetContext(ctx, &unrealizedLoss, `
        SELECT COALESCE(SUM(current_value - cost_basis), 0)
        FROM holdings
        WHERE client_id = $1 AND current_value < cost_basis
    `, clientID)
	if err != nil && err != sql.ErrNoRows {
		fmt.Printf("Error checking unrealized losses: %v\n", err)
	} else if err == nil && unrealizedLoss < -10000 {
		signals = append(signals, DetectedSignal{
			SignalID:   uuid.New(),
			SignalType: "TAX_LOSS_HARVEST_OPPORTUNITY",
			Category:   "PORTFOLIO_EVENTS",
			Strength:   math.Min(math.Abs(unrealizedLoss)/10000*0.8, 1.0),
			DetectedAt: time.Now(),
			RawData: map[string]interface{}{
				"total_unrealized_loss": unrealizedLoss,
				"estimated_tax_savings": math.Abs(unrealizedLoss) * 0.37,
			},
			ClientID: clientID,
		})
	}

	return signals, nil
}

func (a *Activities) detectBehavioralSignals(ctx context.Context, clientID uuid.UUID) ([]DetectedSignal, error) {
	signals := []DetectedSignal{}

	// Query: Portal login frequency drop
	var recentLogins, priorLogins int
	// TODO(hasura-migration): Replace SQL COUNT with Hasura GraphQL aggregate query
	// Example GraphQL query:
	// query GetRecentLogins($clientId: uuid!, $since: timestamptz!) {
	//   client_portal_logins_aggregate(
	//     where: {
	//       client_id: {_eq: $clientId},
	//       login_at: {_gt: $since}
	//     }
	//   ) {
	//     aggregate { count }
	//   }
	// }
	err := a.DB.GetContext(ctx, &recentLogins, `
		SELECT COUNT(*) 
		FROM client_portal_logins
		WHERE client_id = $1 AND login_at > NOW() - INTERVAL '30 days'
	`, clientID)
	if err == nil {
		// TODO(hasura-migration): Replace SQL COUNT with Hasura GraphQL aggregate query
		// Example GraphQL query:
		// query GetPriorLogins($clientId: uuid!, $start: timestamptz!, $end: timestamptz!) {
		//   client_portal_logins_aggregate(
		//     where: {
		//       client_id: {_eq: $clientId},
		//       login_at: {_gte: $start, _lt: $end}
		//     }
		//   ) {
		//     aggregate { count }
		//   }
		// }
		_ = a.DB.GetContext(ctx, &priorLogins, `
			SELECT COUNT(*)
			FROM client_portal_logins
			WHERE client_id = $1 
			  AND login_at BETWEEN NOW() - INTERVAL '60 days' AND NOW() - INTERVAL '30 days'
		`, clientID)

		if priorLogins > 0 && float64(recentLogins)/float64(priorLogins) < 0.5 {
			declinePct := (1 - float64(recentLogins)/float64(priorLogins)) * 100
			signals = append(signals, DetectedSignal{
				SignalID:   uuid.New(),
				SignalType: "ENGAGEMENT_DECLINE",
				Category:   "BEHAVIORAL_PATTERNS",
				Strength:   0.75,
				DetectedAt: time.Now(),
				RawData: map[string]interface{}{
					"recent_logins": recentLogins,
					"prior_logins":  priorLogins,
					"decline_pct":   declinePct,
				},
				ClientID: clientID,
			})
		}
	}

	// Query: Email open rate drop
	var openRate float64
	// TODO(hasura-migration): Replace SQL AVG with Hasura GraphQL query
	// Note: CASE expression requires fetching data and calculating in app logic
	// Example GraphQL query:
	// query GetEmailTracking($clientId: uuid!, $since: timestamptz!) {
	//   email_tracking(
	//     where: {
	//       client_id: {_eq: $clientId},
	//       sent_at: {_gt: $since}
	//     }
	//   ) {
	//     opened_at
	//   }
	// }
	// Then calculate: openRate = COUNT(opened_at IS NOT NULL) / COUNT(*) in app logic
	err = a.DB.GetContext(ctx, &openRate, `
		SELECT COALESCE(AVG(CASE WHEN opened_at IS NOT NULL THEN 1.0 ELSE 0.0 END), 0.5)
		FROM email_tracking
		WHERE client_id = $1 AND sent_at > NOW() - INTERVAL '90 days'
	`, clientID)
	if err == nil && openRate < 0.20 {
		signals = append(signals, DetectedSignal{
			SignalID:   uuid.New(),
			SignalType: "LOW_EMAIL_ENGAGEMENT",
			Category:   "BEHAVIORAL_PATTERNS",
			Strength:   1.0 - openRate,
			DetectedAt: time.Now(),
			RawData: map[string]interface{}{
				"open_rate": openRate * 100,
			},
			ClientID: clientID,
		})
	}

	// Query: Last meeting was too long ago
	var lastMeetingDays int
	// TODO(hasura-migration): Replace SQL aggregation with Hasura GraphQL query
	// Example GraphQL query:
	// query GetLastMeeting($clientId: uuid!) {
	//   client_meetings(
	//     where: {client_id: {_eq: $clientId}},
	//     order_by: {meeting_date: desc},
	//     limit: 1
	//   ) {
	//     meeting_date
	//   }
	// }
	// Then calculate: lastMeetingDays = DAYS(NOW() - meeting_date) in app logic
	err = a.DB.GetContext(ctx, &lastMeetingDays, `
		SELECT COALESCE(EXTRACT(DAY FROM NOW() - MAX(meeting_date))::int, 365)
		FROM client_meetings
		WHERE client_id = $1
	`, clientID)
	if err == nil && lastMeetingDays > 180 {
		signals = append(signals, DetectedSignal{
			SignalID:   uuid.New(),
			SignalType: "ENGAGEMENT_DECLINE",
			Category:   "BEHAVIORAL_PATTERNS",
			Strength:   math.Min(float64(lastMeetingDays)/365.0, 1.0),
			DetectedAt: time.Now(),
			RawData: map[string]interface{}{
				"days_since_last_meeting": lastMeetingDays,
			},
			ClientID: clientID,
		})
	}

	return signals, nil
}

func (a *Activities) detectMarketSignals(ctx context.Context, clientID uuid.UUID) ([]DetectedSignal, error) {
	signals := []DetectedSignal{}

	// Get client's equity allocation
	var equityAllocation float64
	// TODO(hasura-migration): Replace SQL aggregation with Hasura GraphQL query
	// Example GraphQL query:
	// query GetEquityAllocation($clientId: uuid!) {
	//   holdings(
	//     where: {client_id: {_eq: $clientId}}
	//   ) {
	//     asset_class
	//     position_value
	//   }
	// }
	// Then calculate: equityAllocation = SUM(equity position_value) / SUM(all position_value) in app logic
	err := a.DB.GetContext(ctx, &equityAllocation, `
		SELECT COALESCE(
			SUM(CASE WHEN asset_class = 'EQUITY' THEN position_value ELSE 0 END) / 
			NULLIF(SUM(position_value), 0),
			0
		)
		FROM holdings
		WHERE client_id = $1
	`, clientID)

	// Get market volatility (VIX proxy from market_data table or external API)
	var vix float64 = 25.0 // Default moderate volatility
	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query GetLatestVIX {
	//   market_indicators(
	//     where: {indicator_name: {_eq: "VIX"}},
	//     order_by: {recorded_at: desc},
	//     limit: 1
	//   ) {
	//     value
	//   }
	// }
	_ = a.DB.GetContext(ctx, &vix, `
		SELECT COALESCE(value, 25.0)
		FROM market_indicators
		WHERE indicator_name = 'VIX'
		ORDER BY recorded_at DESC
		LIMIT 1
	`)

	// High volatility with significant equity exposure
	if err == nil && vix > 30 && equityAllocation > 0.60 {
		signals = append(signals, DetectedSignal{
			SignalID:   uuid.New(),
			SignalType: "VOLATILITY_EXPOSURE",
			Category:   "MARKET_CONDITIONS",
			Strength:   math.Min((vix-20)/30, 1.0),
			DetectedAt: time.Now(),
			RawData: map[string]interface{}{
				"current_vix":     vix,
				"equity_exposure": equityAllocation * 100,
				"risk_level":      "HIGH",
			},
			ClientID: clientID,
		})
	}

	// Check for sector concentration during sector rotation
	var maxSectorPct float64
	var topSector string
	// TODO(hasura-migration): Replace SQL aggregation with Hasura GraphQL query
	// Example GraphQL query:
	// query GetSectorConcentration($clientId: uuid!) {
	//   holdings(
	//     where: {client_id: {_eq: $clientId}}
	//   ) {
	//     sector
	//     position_value
	//   }
	// }
	// Then calculate: GROUP BY sector, SUM position_value, find max % in app logic
	_ = a.DB.QueryRowContext(ctx, `
		SELECT COALESCE(sector, 'Unknown'), 
		       SUM(position_value) / NULLIF((SELECT SUM(position_value) FROM holdings WHERE client_id = $1), 0) as pct
		FROM holdings
		WHERE client_id = $1
		GROUP BY sector
		ORDER BY pct DESC
		LIMIT 1
	`, clientID).Scan(&topSector, &maxSectorPct)

	if maxSectorPct > 0.40 {
		signals = append(signals, DetectedSignal{
			SignalID:   uuid.New(),
			SignalType: "CONCENTRATED_POSITION_ALERT",
			Category:   "MARKET_CONDITIONS",
			Strength:   math.Min(maxSectorPct/0.40, 1.0),
			DetectedAt: time.Now(),
			RawData: map[string]interface{}{
				"sector":     topSector,
				"sector_pct": maxSectorPct * 100,
			},
			ClientID: clientID,
		})
	}

	return signals, nil
}

func (a *Activities) detectLifecycleSignals(ctx context.Context, clientID uuid.UUID) ([]DetectedSignal, error) {
	signals := []DetectedSignal{}

	// Query: Retirement approaching (within 12 months)
	var retirementDays int
	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query GetRetirementDate($clientId: uuid!) {
	//   clients(
	//     where: {
	//       id: {_eq: $clientId},
	//       target_retirement_date: {_is_null: false}
	//     }
	//   ) {
	//     target_retirement_date
	//   }
	// }
	// Then calculate: retirementDays = DAYS(target_retirement_date - NOW()) in app logic
	err := a.DB.GetContext(ctx, &retirementDays, `
		SELECT COALESCE(EXTRACT(DAY FROM target_retirement_date - NOW())::int, 9999)
		FROM clients
		WHERE id = $1 AND target_retirement_date IS NOT NULL
	`, clientID)
	if err == nil && retirementDays > 0 && retirementDays <= 365 {
		signals = append(signals, DetectedSignal{
			SignalID:   uuid.New(),
			SignalType: "RETIREMENT_APPROACHING",
			Category:   "LIFECYCLE",
			Strength:   1.0 - (float64(retirementDays) / 365.0),
			DetectedAt: time.Now(),
			RawData: map[string]interface{}{
				"days_until_retirement": retirementDays,
			},
			ClientID: clientID,
		})
	}

	// Query: Large inflow detected (potential inheritance)
	var largeInflow float64
	var inflowDate time.Time
	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query GetLargeInflows($clientId: uuid!, $minAmount: numeric!, $since: timestamptz!) {
	//   transactions(
	//     where: {
	//       client_id: {_eq: $clientId},
	//       amount: {_gt: $minAmount},
	//       transaction_type: {_eq: "DEPOSIT"},
	//       transaction_date: {_gt: $since}
	//     },
	//     order_by: {amount: desc},
	//     limit: 1
	//   ) {
	//     amount
	//     transaction_date
	//   }
	// }
	err = a.DB.QueryRowContext(ctx, `
		SELECT amount, transaction_date
		FROM transactions
		WHERE client_id = $1 
		  AND amount > 100000
		  AND transaction_type = 'DEPOSIT'
		  AND transaction_date > NOW() - INTERVAL '30 days'
		ORDER BY amount DESC
		LIMIT 1
	`, clientID).Scan(&largeInflow, &inflowDate)
	if err == nil && largeInflow > 0 {
		signals = append(signals, DetectedSignal{
			SignalID:   uuid.New(),
			SignalType: "INHERITANCE_DETECTED",
			Category:   "LIFECYCLE",
			Strength:   math.Min(largeInflow/500000.0, 1.0),
			DetectedAt: time.Now(),
			RawData: map[string]interface{}{
				"inflow_amount": largeInflow,
				"inflow_date":   inflowDate,
			},
			ClientID: clientID,
		})
	}

	// Query: Client anniversary upcoming (within 30 days)
	var anniversaryDays int
	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query GetOnboardingDate($clientId: uuid!) {
	//   clients(where: {id: {_eq: $clientId}}) {
	//     onboarding_date
	//   }
	// }
	// Then calculate: anniversaryDays = complex date math in app logic
	err = a.DB.GetContext(ctx, &anniversaryDays, `
		SELECT EXTRACT(DAY FROM (
			DATE_TRUNC('year', NOW()) + 
			(onboarding_date - DATE_TRUNC('year', onboarding_date)) - 
			NOW()
		))::int
		FROM clients
		WHERE id = $1
	`, clientID)
	if err == nil && anniversaryDays >= 0 && anniversaryDays <= 30 {
		signals = append(signals, DetectedSignal{
			SignalID:   uuid.New(),
			SignalType: "ANNIVERSARY_UPCOMING",
			Category:   "LIFECYCLE",
			Strength:   0.6,
			DetectedAt: time.Now(),
			RawData: map[string]interface{}{
				"days_until_anniversary": anniversaryDays,
			},
			ClientID: clientID,
		})
	}

	// Query: RMD deadline approaching (for clients 73+)
	var clientAge int
	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query GetClientDOB($clientId: uuid!) {
	//   clients(where: {id: {_eq: $clientId}}) {
	//     date_of_birth
	//   }
	// }
	// Then calculate: clientAge = YEAR(NOW()) - YEAR(date_of_birth) in app logic
	err = a.DB.GetContext(ctx, &clientAge, `
		SELECT EXTRACT(YEAR FROM AGE(date_of_birth))::int
		FROM clients
		WHERE id = $1
	`, clientID)
	if err == nil && clientAge >= 73 {
		// Check if RMD was taken this year
		var rmdTaken bool
		// TODO(hasura-migration): Replace SQL EXISTS with Hasura GraphQL aggregate query
		// Example GraphQL query:
		// query CheckRMDTaken($clientId: uuid!, $year: Int!) {
		//   transactions_aggregate(
		//     where: {
		//       client_id: {_eq: $clientId},
		//       transaction_type: {_eq: "RMD"},
		//       _and: [{_sql: {_expression: "EXTRACT(YEAR FROM transaction_date) = $year"}}]
		//     }
		//   ) {
		//     aggregate { count }
		//   }
		// }
		// Then check: rmdTaken = count > 0
		_ = a.DB.GetContext(ctx, &rmdTaken, `
			SELECT EXISTS(
				SELECT 1 FROM transactions 
				WHERE client_id = $1 
				  AND transaction_type = 'RMD'
				  AND EXTRACT(YEAR FROM transaction_date) = EXTRACT(YEAR FROM NOW())
			)
		`, clientID)

		if !rmdTaken {
			signals = append(signals, DetectedSignal{
				SignalID:   uuid.New(),
				SignalType: "COMPLIANCE_DEADLINE",
				Category:   "LIFECYCLE",
				Strength:   0.95,
				DetectedAt: time.Now(),
				RawData: map[string]interface{}{
					"client_age":   clientAge,
					"deadline":     "RMD",
					"rmd_required": true,
				},
				ClientID: clientID,
			})
		}
	}

	return signals, nil
}

func (a *Activities) detectEngagementSignals(ctx context.Context, clientID uuid.UUID) ([]DetectedSignal, error) {
	signals := []DetectedSignal{}

	// Query: Overall engagement score drop
	var engagementScore float64
	// TODO(hasura-migration): Replace complex SQL query with multiple Hasura GraphQL queries
	// Note: This requires 3 separate aggregate queries and client-side calculation
	// Example GraphQL queries:
	// query GetEngagementMetrics($clientId: uuid!) {
	//   portal_logins: client_portal_logins_aggregate(
	//     where: {
	//       client_id: {_eq: $clientId},
	//       login_at: {_gt: "90 days ago"}
	//     }
	//   ) {
	//     aggregate { count }
	//   }
	//   email_opens: email_tracking(
	//     where: {
	//       client_id: {_eq: $clientId},
	//       sent_at: {_gt: "90 days ago"}
	//     }
	//   ) {
	//     opened_at
	//   }
	//   meetings: client_meetings_aggregate(
	//     where: {
	//       client_id: {_eq: $clientId},
	//       meeting_date: {_gt: "180 days ago"}
	//     }
	//   ) {
	//     aggregate { count }
	//   }
	// }
	// Then calculate: engagementScore = (portal*0.4 + email*0.3 + meeting*0.3) in app logic
	err := a.DB.GetContext(ctx, &engagementScore, `
		SELECT COALESCE(
			(
				-- Portal activity (40%)
				COALESCE(
					(SELECT COUNT(*) FROM client_portal_logins WHERE client_id = $1 AND login_at > NOW() - INTERVAL '90 days')::float / 30.0,
					0
				) * 0.4 +
				-- Email engagement (30%)
				COALESCE(
					(SELECT AVG(CASE WHEN opened_at IS NOT NULL THEN 1.0 ELSE 0.0 END) FROM email_tracking WHERE client_id = $1 AND sent_at > NOW() - INTERVAL '90 days'),
					0.5
				) * 0.3 +
				-- Meeting frequency (30%)
				COALESCE(
					(SELECT COUNT(*) FROM client_meetings WHERE client_id = $1 AND meeting_date > NOW() - INTERVAL '180 days')::float / 2.0,
					0.5
				) * 0.3
			),
			0.5
		)
	`, clientID)

	if err == nil && engagementScore < 0.3 {
		signals = append(signals, DetectedSignal{
			SignalID:   uuid.New(),
			SignalType: "ENGAGEMENT_DECLINE",
			Category:   "ENGAGEMENT",
			Strength:   1.0 - engagementScore,
			DetectedAt: time.Now(),
			RawData: map[string]interface{}{
				"engagement_score": engagementScore,
				"threshold":        0.3,
			},
			ClientID: clientID,
		})
	}

	// Query: Large pending withdrawal (potential attrition risk)
	var pendingWithdrawal float64
	// TODO(hasura-migration): Replace SQL aggregation with Hasura GraphQL query
	// Example GraphQL query:
	// query GetPendingWithdrawals($clientId: uuid!) {
	//   pending_transactions(
	//     where: {
	//       client_id: {_eq: $clientId},
	//       amount: {_lt: 0},
	//       status: {_eq: "pending"}
	//     }
	//   ) {
	//     amount
	//   }
	// }
	// Then calculate: pendingWithdrawal = SUM(ABS(amount)) in app logic
	err = a.DB.GetContext(ctx, &pendingWithdrawal, `
		SELECT COALESCE(SUM(ABS(amount)), 0)
		FROM pending_transactions
		WHERE client_id = $1 
		  AND amount < 0
		  AND status = 'pending'
	`, clientID)
	if err == nil && pendingWithdrawal > 50000 {
		signals = append(signals, DetectedSignal{
			SignalID:   uuid.New(),
			SignalType: "LARGE_WITHDRAWAL_PENDING",
			Category:   "ENGAGEMENT",
			Strength:   math.Min(pendingWithdrawal/100000.0, 1.0),
			DetectedAt: time.Now(),
			RawData: map[string]interface{}{
				"withdrawal_amount": pendingWithdrawal,
			},
			ClientID: clientID,
		})
	}

	return signals, nil
}

func deduplicateSignals(signals []DetectedSignal) []DetectedSignal {
	// Simple deduplication logic
	unique := make(map[string]DetectedSignal)
	for _, s := range signals {
		key := s.SignalType + s.Category
		if existing, ok := unique[key]; !ok || s.Strength > existing.Strength {
			unique[key] = s
		}
	}
	result := []DetectedSignal{}
	for _, s := range unique {
		result = append(result, s)
	}
	return result
}

// GenerateNextBestActionActivity calls the Python inference engine
func (a *Activities) GenerateNextBestActionActivity(ctx context.Context, signal DetectedSignal) ([]NextBestAction, error) {
	mlServiceURL := os.Getenv("NBA_ML_SERVICE_URL")
	if mlServiceURL == "" {
		mlServiceURL = "http://localhost:5001"
	}

	payload := map[string]interface{}{
		"client_id": signal.ClientID,
		"signal":    signal,
		// In a real scenario, we'd fetch more context here
		"text": fmt.Sprintf("Signal detected: %s. Category: %s.", signal.SignalType, signal.Category),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", mlServiceURL+"/predict", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Fallback to mock if service is unavailable (for demo resilience)
		fmt.Printf("ML Service unavailable (%v), using fallback logic.\n", err)
		return a.fallbackMockActions(signal), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ml service returned status: %d", resp.StatusCode)
	}

	var actions []NextBestAction
	if err := json.NewDecoder(resp.Body).Decode(&actions); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Enrich with UUIDs and ClientID since the Python model might not return them fully populated
	for i := range actions {
		if actions[i].ActionID == uuid.Nil {
			actions[i].ActionID = uuid.New()
		}
		actions[i].ClientID = signal.ClientID
		// actions[i].ClientName = "Fetched Name" // In real app, fetch from DB
	}

	return actions, nil
}

func (a *Activities) fallbackMockActions(signal DetectedSignal) []NextBestAction {
	actions := []NextBestAction{}

	if signal.SignalType == "EXCESS_CASH_DRAG" {
		actions = append(actions, NextBestAction{
			ActionID:           uuid.New(),
			ClientID:           signal.ClientID,
			ClientName:         "Client Name", // Should fetch from DB
			ActionType:         "INVEST_CASH",
			ActionName:         "Discuss Cash Investment Options",
			Confidence:         0.9,
			UrgencyScore:       0.8,
			ExpectedValue:      1000.0,
			SuccessProbability: 0.7,
			TriggerSignal:      signal.SignalType,
			Reasoning:          "Client has high cash balance in a rising market.",
			RecommendedChannel: "PHONE",
			DurationMinutes:    30,
			TemplateContent: map[string]interface{}{
				"script": "I noticed you have a significant cash balance...",
			},
		})
	} else if signal.SignalType == "TAX_LOSS_HARVEST_OPPORTUNITY" {
		actions = append(actions, NextBestAction{
			ActionID:           uuid.New(),
			ClientID:           signal.ClientID,
			ClientName:         "Client Name",
			ActionType:         "PROACTIVE_TAX_LOSS_HARVEST",
			ActionName:         "Initiate Tax-Loss Harvesting Review",
			Confidence:         0.95,
			UrgencyScore:       0.9,
			ExpectedValue:      2500.0,
			SuccessProbability: 0.8,
			TriggerSignal:      signal.SignalType,
			Reasoning:          "Significant unrealized losses detected.",
			RecommendedChannel: "PHONE",
			DurationMinutes:    30,
			TemplateContent: map[string]interface{}{
				"email_subject": "Opportunity to Reduce Your 2025 Tax Bill",
			},
		})
	}
	return actions
}

func (a *Activities) SaveRecommendedActionsActivity(ctx context.Context, actions []NextBestAction) error {
	// This would save the actions to a table so they appear in the dashboard.
	// The user didn't explicitly provide a `recommendations` table, but `nba_action_outcomes` has `recommended_at`.
	// Maybe `nba_action_outcomes` is used for both pending and completed actions?
	// "executed_at" is nullable. So yes, we insert here.

	for _, action := range actions {
		// TODO(hasura-migration): Replace SQL INSERT with Hasura GraphQL mutation
		// Example GraphQL mutation:
		// mutation InsertActionOutcome($object: nba_action_outcomes_insert_input!) {
		//   insert_nba_action_outcomes_one(object: $object) {
		//     action_id
		//     client_id
		//   }
		// }
		// Variables: {
		//   "object": {
		//     "action_id": "<uuid>",
		//     "client_id": "<uuid>",
		//     "trigger_signal_type": "EXCESS_CASH_DRAG",
		//     "recommended_at": "2025-12-08T10:00:00Z",
		//     "revenue_generated": 1000.0,
		//     "client_satisfaction_change": 0.0
		//   }
		// }
		_, err := a.DB.ExecContext(ctx, `
            INSERT INTO nba_action_outcomes (
                action_id, client_id, trigger_signal_type, recommended_at, 
                revenue_generated, client_satisfaction_change
            ) VALUES ($1, $2, $3, $4, $5, 0.0)
        `, action.ActionID, action.ClientID, action.TriggerSignal, time.Now(), action.ExpectedValue)
		if err != nil {
			return err
		}
	}
	return nil
}

package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/nba"
	"github.com/jmoiron/sqlx"
)

type NBAHandler struct {
	db  *sqlx.DB
	hub *nba.WebSocketHub
}

func NewNBAHandler(db *sqlx.DB) *NBAHandler {
	return &NBAHandler{
		db:  db,
		hub: nba.GetWebSocketHub(),
	}
}

// GetRecommendations returns pending NBA recommendations for the current advisor
func (h *NBAHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clientID := r.URL.Query().Get("client_id") // Optional: filter by specific client

	// Try to fetch real data from database
	actions, err := h.fetchRecommendationsFromDB(ctx, clientID)
	if err != nil || len(actions) == 0 {
		// Fallback to mock data for demo
		log.Printf("NBA: Using mock data (error: %v, count: %d)", err, len(actions))
		actions = h.getMockRecommendations()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(actions)
}

func (h *NBAHandler) fetchRecommendationsFromDB(ctx context.Context, clientID string) ([]nba.NextBestAction, error) {
	query := `
		SELECT 
			o.outcome_id as action_id,
			o.client_id,
			COALESCE(cl.first_name || ' ' || cl.last_name, 'Client') as client_name,
			COALESCE(c.action_code, 'OUTREACH') as action_type,
			COALESCE(c.action_name, 'Follow Up') as action_name,
			0.85 as confidence,
			0.7 as urgency_score,
			COALESCE(c.estimated_revenue_impact, 1000)::float as expected_value,
			0.65 as success_probability,
			o.trigger_signal_type,
			COALESCE(c.description, 'Recommended action based on detected signal.') as reasoning,
			COALESCE(c.default_channel, 'PHONE') as recommended_channel,
			COALESCE(c.estimated_duration_minutes, 30) as duration_minutes,
			COALESCE(c.template_content::text, '{}') as template_content,
			o.recommended_at
		FROM nba_action_outcomes o
		LEFT JOIN nba_action_catalog c ON o.action_id = c.action_id
		LEFT JOIN clients cl ON o.client_id = cl.id
		WHERE o.executed_at IS NULL
		  AND (o.dismiss_reason IS NULL OR o.dismiss_reason = '')
	`

	args := []interface{}{}
	argCount := 0

	if clientID != "" {
		argCount++
		query += ` AND o.client_id = $` + string(rune('0'+argCount))
		args = append(args, clientID)
	}

	query += ` ORDER BY o.recommended_at DESC LIMIT 50`

	rows, err := h.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []nba.NextBestAction
	for rows.Next() {
		var action nba.NextBestAction
		var templateContentStr string
		var recommendedAt time.Time

		err := rows.Scan(
			&action.ActionID,
			&action.ClientID,
			&action.ClientName,
			&action.ActionType,
			&action.ActionName,
			&action.Confidence,
			&action.UrgencyScore,
			&action.ExpectedValue,
			&action.SuccessProbability,
			&action.TriggerSignal,
			&action.Reasoning,
			&action.RecommendedChannel,
			&action.DurationMinutes,
			&templateContentStr,
			&recommendedAt,
		)
		if err != nil {
			log.Printf("NBA: Failed to scan row: %v", err)
			continue
		}

		// Parse template content JSON
		if templateContentStr != "" && templateContentStr != "{}" {
			_ = json.Unmarshal([]byte(templateContentStr), &action.TemplateContent)
		}

		actions = append(actions, action)
	}

	return actions, nil
}

func (h *NBAHandler) getMockRecommendations() []nba.NextBestAction {
	return []nba.NextBestAction{
		{
			ActionID:           uuid.New(),
			ClientID:           uuid.New(),
			ClientName:         "John Doe",
			ActionType:         "PROACTIVE_TAX_LOSS_HARVEST",
			ActionName:         "Initiate Tax-Loss Harvesting Review",
			Confidence:         0.95,
			UrgencyScore:       0.9,
			ExpectedValue:      2500.0,
			SuccessProbability: 0.8,
			TriggerSignal:      "TAX_LOSS_HARVEST_OPPORTUNITY",
			Reasoning:          "Portfolio has $15,000 in unrealized losses that could offset gains. Estimated tax savings of $2,500.",
			RecommendedChannel: "PHONE",
			DurationMinutes:    30,
			TemplateContent: map[string]interface{}{
				"email_subject": "Opportunity to Reduce Your 2025 Tax Bill",
				"email_body":    "Hi John,\n\nI noticed some unrealized losses in your portfolio that could save you approximately $2,500 in taxes this year through strategic tax-loss harvesting.\n\nWould you have 20 minutes this week to discuss this opportunity?\n\nBest regards,\nYour Advisor",
				"call_script":   "Opening: I wanted to reach out because our system flagged a potential tax savings opportunity in your account...",
			},
		},
		{
			ActionID:           uuid.New(),
			ClientID:           uuid.New(),
			ClientName:         "Jane Smith",
			ActionType:         "REENGAGEMENT_OUTREACH",
			ActionName:         "Client Re-engagement Call",
			Confidence:         0.85,
			UrgencyScore:       0.7,
			ExpectedValue:      5000.0,
			SuccessProbability: 0.6,
			TriggerSignal:      "ENGAGEMENT_DECLINE",
			Reasoning:          "Client portal logins dropped 80% in the last 30 days. Last meeting was 120 days ago. Proactive outreach can prevent attrition.",
			RecommendedChannel: "PHONE",
			DurationMinutes:    20,
			TemplateContent: map[string]interface{}{
				"call_script": "Hi Jane, I realized we haven't connected in a while and wanted to check in. How have things been going for you?\n\n[Listen actively]\n\nI want to make sure we're providing the level of service and communication that works best for you.",
			},
		},
		{
			ActionID:           uuid.New(),
			ClientID:           uuid.New(),
			ClientName:         "Robert Johnson",
			ActionType:         "CONCENTRATED_POSITION_REVIEW",
			ActionName:         "Diversification Strategy Discussion",
			Confidence:         0.88,
			UrgencyScore:       0.65,
			ExpectedValue:      8000.0,
			SuccessProbability: 0.7,
			TriggerSignal:      "CONCENTRATED_POSITION_RISK",
			Reasoning:          "AAPL position represents 35% of portfolio. Market volatility has increased. Consider diversification to reduce single-stock risk.",
			RecommendedChannel: "VIDEO_CALL",
			DurationMinutes:    45,
			TemplateContent: map[string]interface{}{
				"meeting_agenda": "1. Review current portfolio concentration\n2. Discuss risks of single-position overweight\n3. Present diversification strategies\n4. Address tax implications\n5. Create implementation timeline",
			},
		},
		{
			ActionID:           uuid.New(),
			ClientID:           uuid.New(),
			ClientName:         "Sarah Williams",
			ActionType:         "RETIREMENT_PLANNING_REVIEW",
			ActionName:         "Schedule Retirement Readiness Review",
			Confidence:         0.92,
			UrgencyScore:       0.8,
			ExpectedValue:      12000.0,
			SuccessProbability: 0.85,
			TriggerSignal:      "RETIREMENT_APPROACHING",
			Reasoning:          "Client retirement is in 8 months. Portfolio needs final adjustments for income generation and risk reduction.",
			RecommendedChannel: "IN_PERSON",
			DurationMinutes:    60,
			TemplateContent: map[string]interface{}{
				"email_subject": "Your Retirement is Approaching - Let's Finalize Your Plan",
				"email_body":    "Hi Sarah,\n\nWith your retirement approaching in just 8 months, I wanted to schedule a comprehensive review to ensure everything is in place for a smooth transition.\n\nI'd like to cover:\n- Income planning strategy\n- Social Security timing\n- Healthcare bridge (Medicare gap)\n- Tax-efficient withdrawal strategy\n\nWould next Tuesday at 2pm work for an in-person meeting?\n\nBest,\nYour Advisor",
			},
		},
	}
}

// ExecuteAction marks an action as started/executed
func (h *NBAHandler) ExecuteAction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ActionID uuid.UUID `json:"action_id"`
		Channel  string    `json:"channel,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update DB to mark as executing
	_, err := h.db.ExecContext(r.Context(), `
		UPDATE nba_action_outcomes 
		SET executed_at = NOW(),
			execution_channel = COALESCE($2, execution_channel)
		WHERE outcome_id = $1
	`, req.ActionID, req.Channel)

	if err != nil {
		log.Printf("NBA: Failed to mark action as executed: %v", err)
		// Don't fail the request, action tracking is non-critical
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "executing",
		"action_id":   req.ActionID,
		"executed_at": time.Now(),
	})
}

// CompleteAction marks an action as completed with outcome tracking
func (h *NBAHandler) CompleteAction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ActionID                 uuid.UUID `json:"action_id"`
		Outcome                  string    `json:"outcome"` // SUCCESS, PARTIAL, FAILED
		Notes                    string    `json:"notes,omitempty"`
		ClientResponded          bool      `json:"client_responded"`
		RevenueGenerated         float64   `json:"revenue_generated,omitempty"`
		ClientSatisfactionChange float64   `json:"client_satisfaction_change,omitempty"`
		AdvisorRating            int       `json:"advisor_rating,omitempty"` // 1-5
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Calculate action_successful based on outcome
	actionSuccessful := req.Outcome == "SUCCESS"

	// Update DB with completion data
	_, err := h.db.ExecContext(r.Context(), `
		UPDATE nba_action_outcomes 
		SET completed_at = NOW(),
			client_responded = $2,
			action_successful = $3,
			revenue_generated = $4,
			client_satisfaction_change = $5,
			advisor_rating = $6,
			advisor_feedback = $7
		WHERE outcome_id = $1
	`, req.ActionID, req.ClientResponded, actionSuccessful,
		req.RevenueGenerated, req.ClientSatisfactionChange, req.AdvisorRating, req.Notes)

	if err != nil {
		log.Printf("NBA: Failed to complete action: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "completed",
		"action_id":    req.ActionID,
		"outcome":      req.Outcome,
		"completed_at": time.Now(),
	})
}

// DismissAction marks an action as dismissed
func (h *NBAHandler) DismissAction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ActionID      uuid.UUID `json:"action_id"`
		DismissReason string    `json:"dismiss_reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update DB with dismissal
	_, err := h.db.ExecContext(r.Context(), `
		UPDATE nba_action_outcomes 
		SET completed_at = NOW(),
			dismiss_reason = $2,
			action_successful = false
		WHERE outcome_id = $1
	`, req.ActionID, req.DismissReason)

	if err != nil {
		log.Printf("NBA: Failed to dismiss action: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "dismissed",
		"action_id":      req.ActionID,
		"dismiss_reason": req.DismissReason,
	})
}

// GetSignals returns recent detected signals
func (h *NBAHandler) GetSignals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := r.URL.Query().Get("tenant_id")
	_ = tenantID // Reserved for multi-tenant filtering

	query := `
		SELECT 
			signal_id,
			client_id,
			signal_type,
			signal_category,
			signal_source,
			strength,
			detected_at,
			expiry_at,
			raw_data,
			processed_insights
		FROM client_signals
		WHERE detected_at > NOW() - INTERVAL '7 days'
		  AND (processed_at IS NULL OR processed_at > NOW() - INTERVAL '24 hours')
		ORDER BY detected_at DESC
		LIMIT 100
	`

	rows, err := h.db.QueryxContext(ctx, query)
	if err != nil {
		log.Printf("NBA: Failed to fetch signals: %v", err)
		// Return empty array on error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}
	defer rows.Close()

	var signals []map[string]interface{}
	for rows.Next() {
		var (
			signalID, clientID                       uuid.UUID
			signalType, signalCategory, signalSource string
			strength                                 float64
			detectedAt                               time.Time
			expiryAt                                 sql.NullTime
			rawDataStr, insightsStr                  sql.NullString
		)

		err := rows.Scan(
			&signalID, &clientID, &signalType, &signalCategory, &signalSource,
			&strength, &detectedAt, &expiryAt, &rawDataStr, &insightsStr,
		)
		if err != nil {
			continue
		}

		signal := map[string]interface{}{
			"signal_id":       signalID,
			"client_id":       clientID,
			"signal_type":     signalType,
			"signal_category": signalCategory,
			"signal_source":   signalSource,
			"strength":        strength,
			"detected_at":     detectedAt,
		}

		if expiryAt.Valid {
			signal["expiry_at"] = expiryAt.Time
		}
		if rawDataStr.Valid {
			var rawData map[string]interface{}
			if json.Unmarshal([]byte(rawDataStr.String), &rawData) == nil {
				signal["raw_data"] = rawData
			}
		}

		signals = append(signals, signal)
	}

	if signals == nil {
		signals = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(signals)
}

// GetActionCatalog returns the available action types
func (h *NBAHandler) GetActionCatalog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := `
		SELECT 
			action_id,
			action_code,
			action_name,
			action_category,
			description,
			default_channel,
			estimated_duration_minutes,
			estimated_revenue_impact,
			client_value_impact,
			automation_eligible,
			template_content,
			required_advisor_skills,
			compliance_review_required,
			success_metrics
		FROM nba_action_catalog
		ORDER BY action_category, action_name
	`

	rows, err := h.db.QueryxContext(ctx, query)
	if err != nil {
		log.Printf("NBA: Failed to fetch action catalog: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}
	defer rows.Close()

	var actions []map[string]interface{}
	for rows.Next() {
		var (
			actionID                                                     uuid.UUID
			actionCode, actionName, actionCategory, description, channel string
			durationMinutes, revenueImpact, valueImpact                  int
			automationEligible, complianceRequired                       bool
			templateStr, skillsStr, metricsStr                           sql.NullString
		)

		err := rows.Scan(
			&actionID, &actionCode, &actionName, &actionCategory, &description,
			&channel, &durationMinutes, &revenueImpact, &valueImpact,
			&automationEligible, &templateStr, &skillsStr, &complianceRequired, &metricsStr,
		)
		if err != nil {
			continue
		}

		action := map[string]interface{}{
			"action_id":                  actionID,
			"action_code":                actionCode,
			"action_name":                actionName,
			"action_category":            actionCategory,
			"description":                description,
			"default_channel":            channel,
			"estimated_duration_minutes": durationMinutes,
			"estimated_revenue_impact":   revenueImpact,
			"client_value_impact":        valueImpact,
			"automation_eligible":        automationEligible,
			"compliance_review_required": complianceRequired,
		}

		if templateStr.Valid {
			var template map[string]interface{}
			if json.Unmarshal([]byte(templateStr.String), &template) == nil {
				action["template_content"] = template
			}
		}

		actions = append(actions, action)
	}

	if actions == nil {
		actions = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(actions)
}

// GetOutcomeStats returns statistics on action outcomes for ML feedback
func (h *NBAHandler) GetOutcomeStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	lookbackDays := 90

	query := `
		SELECT 
			trigger_signal_type,
			COUNT(*) as total_actions,
			COUNT(*) FILTER (WHERE action_successful) as successful_actions,
			AVG(CASE WHEN action_successful THEN 1.0 ELSE 0.0 END) as success_rate,
			SUM(COALESCE(revenue_generated, 0)) as total_revenue,
			AVG(COALESCE(advisor_rating, 0)) as avg_rating
		FROM nba_action_outcomes
		WHERE completed_at > NOW() - INTERVAL '%d days'
		GROUP BY trigger_signal_type
		ORDER BY total_actions DESC
	`

	rows, err := h.db.QueryxContext(ctx, query, lookbackDays)
	if err != nil {
		log.Printf("NBA: Failed to fetch outcome stats: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{})
		return
	}
	defer rows.Close()

	stats := map[string]interface{}{}
	for rows.Next() {
		var (
			signalType                           string
			totalActions, successfulActions      int
			successRate, totalRevenue, avgRating float64
		)

		err := rows.Scan(
			&signalType, &totalActions, &successfulActions,
			&successRate, &totalRevenue, &avgRating,
		)
		if err != nil {
			continue
		}

		stats[signalType] = map[string]interface{}{
			"total_actions":      totalActions,
			"successful_actions": successfulActions,
			"success_rate":       successRate,
			"total_revenue":      totalRevenue,
			"avg_rating":         avgRating,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// WebSocket handler for real-time updates
func (h *NBAHandler) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	h.hub.ServeWs(w, r)
}

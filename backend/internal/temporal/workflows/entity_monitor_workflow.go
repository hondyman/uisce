package workflows

import (
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// EntityMonitorConfig defines the configuration for an entity monitor.
type EntityMonitorConfig struct {
	CheckInterval   time.Duration          `json:"check_interval"`
	AlertThresholds map[string]float64     `json:"alert_thresholds"`
	DataSources     []string               `json:"data_sources"`
	NotifyChannels  []string               `json:"notify_channels"`
	CustomConfig    map[string]interface{} `json:"custom_config,omitempty"`
}

// EntityMonitorState maintains the state of an entity monitor workflow.
type EntityMonitorState struct {
	MonitorID         string              `json:"monitor_id"`
	TenantID          string              `json:"tenant_id"`
	EntityID          string              `json:"entity_id"`
	MonitorType       string              `json:"monitor_type"`
	Config            EntityMonitorConfig `json:"config"`
	LastCheckTime     time.Time           `json:"last_check_time"`
	LastAlertTime     time.Time           `json:"last_alert_time,omitempty"`
	LastRiskScore     float64             `json:"last_risk_score"`
	CheckCount        int                 `json:"check_count"`
	AlertCount        int                 `json:"alert_count"`
	ConsecutiveErrors int                 `json:"consecutive_errors"`
}

// RiskSignal represents a risk event signal sent to the workflow.
type RiskSignal struct {
	EventType string                 `json:"event_type"`
	Severity  string                 `json:"severity"`
	Title     string                 `json:"title"`
	Details   map[string]interface{} `json:"details"`
	SourceURL string                 `json:"source_url,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// EntityDataResult contains the result of fetching entity data.
type EntityDataResult struct {
	EntityID     string                 `json:"entity_id"`
	RiskScore    float64                `json:"risk_score"`
	PriceChange  float64                `json:"price_change,omitempty"`
	NewsCount    int                    `json:"news_count"`
	FilingsCount int                    `json:"filings_count"`
	Alerts       []RiskSignal           `json:"alerts,omitempty"`
	Properties   map[string]interface{} `json:"properties"`
	FetchedAt    time.Time              `json:"fetched_at"`
}

// EntityMonitorWorkflow is a long-running workflow that monitors an entity for risk signals.
// It uses ContinueAsNew to prevent history bloat.
func EntityMonitorWorkflow(ctx workflow.Context, state EntityMonitorState) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("EntityMonitorWorkflow started",
		"monitor_id", state.MonitorID,
		"entity_id", state.EntityID,
		"monitor_type", state.MonitorType)

	// Activity options with retry policy
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Loop for N iterations before ContinueAsNew to prevent history bloat
	// At 100 checks with 1-hour intervals, this is ~4 days before restart
	const maxIterations = 100

	for i := 0; i < maxIterations; i++ {
		// Create selector to handle both timer and signals
		selector := workflow.NewSelector(ctx)

		// Signal channel for external risk events (webhooks, news alerts)
		signalChan := workflow.GetSignalChannel(ctx, "RiskEventSignal")
		var receivedSignal *RiskSignal

		selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, more bool) {
			var signal RiskSignal
			c.Receive(ctx, &signal)
			receivedSignal = &signal
			logger.Info("Received external risk signal",
				"event_type", signal.EventType,
				"severity", signal.Severity)
		})

		// Timer for periodic polling
		checkInterval := state.Config.CheckInterval
		if checkInterval == 0 {
			checkInterval = time.Hour // Default to 1 hour
		}
		timerFuture := workflow.NewTimer(ctx, checkInterval)

		selector.AddFuture(timerFuture, func(f workflow.Future) {
			// Timer fired, will proceed to polling logic
		})

		// Wait for either signal or timer
		selector.Select(ctx)

		// Process external signal if received
		if receivedSignal != nil {
			if err := processRiskSignal(ctx, state, *receivedSignal); err != nil {
				logger.Error("Failed to process risk signal", "error", err)
				state.ConsecutiveErrors++
			} else {
				state.ConsecutiveErrors = 0
				state.AlertCount++
				state.LastAlertTime = receivedSignal.Timestamp
			}
			continue // Skip polling this iteration since we processed a signal
		}

		// Execute polling activity
		var currentData EntityDataResult
		err := workflow.ExecuteActivity(ctx, "FetchEntityData", state.TenantID, state.EntityID, state.Config.DataSources).Get(ctx, &currentData)
		if err != nil {
			logger.Error("Failed to fetch entity data", "error", err)
			state.ConsecutiveErrors++

			// Record error if too many consecutive failures
			if state.ConsecutiveErrors >= 5 {
				workflow.ExecuteActivity(ctx, "RecordMonitorError", state.MonitorID, err.Error())
			}
			continue
		}

		state.ConsecutiveErrors = 0
		state.CheckCount++
		state.LastCheckTime = workflow.Now(ctx)

		// Detect drift and changes
		if hasSignificantChange(state, currentData) {
			// Generate alert
			alert := RiskSignal{
				EventType: determineEventType(state, currentData),
				Severity:  determineSeverity(state, currentData),
				Title:     generateAlertTitle(state.MonitorType, currentData),
				Details: map[string]interface{}{
					"previous_risk_score": state.LastRiskScore,
					"current_risk_score":  currentData.RiskScore,
					"change_delta":        currentData.RiskScore - state.LastRiskScore,
					"properties":          currentData.Properties,
				},
				Timestamp: workflow.Now(ctx),
			}

			if err := processRiskSignal(ctx, state, alert); err != nil {
				logger.Error("Failed to process generated alert", "error", err)
			} else {
				state.AlertCount++
				state.LastAlertTime = workflow.Now(ctx)
			}
		}

		// Update state with latest data
		state.LastRiskScore = currentData.RiskScore

		// Record successful check
		workflow.ExecuteActivity(ctx, "RecordMonitorCheck", state.MonitorID)
	}

	// ContinueAsNew to reset history while preserving state
	logger.Info("ContinueAsNew after max iterations", "check_count", state.CheckCount)
	return workflow.NewContinueAsNewError(ctx, EntityMonitorWorkflow, state)
}

// processRiskSignal handles a risk signal by creating an event and notifying.
func processRiskSignal(ctx workflow.Context, state EntityMonitorState, signal RiskSignal) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Create risk event in database
	eventID := uuid.New().String()
	err := workflow.ExecuteActivity(ctx, "CreateRiskEvent", map[string]interface{}{
		"event_id":    eventID,
		"tenant_id":   state.TenantID,
		"entity_id":   state.EntityID,
		"monitor_id":  state.MonitorID,
		"event_type":  signal.EventType,
		"severity":    signal.Severity,
		"title":       signal.Title,
		"description": signal.Details["description"],
		"event_data":  signal.Details,
		"source_url":  signal.SourceURL,
	}).Get(ctx, nil)
	if err != nil {
		return err
	}

	// Send notifications based on configured channels
	for _, channel := range state.Config.NotifyChannels {
		workflow.ExecuteActivity(ctx, "SendNotification", channel, map[string]interface{}{
			"event_id":   eventID,
			"entity_id":  state.EntityID,
			"event_type": signal.EventType,
			"severity":   signal.Severity,
			"title":      signal.Title,
			"details":    signal.Details,
		})
	}

	return nil
}

// hasSignificantChange determines if the entity data has changed significantly.
func hasSignificantChange(state EntityMonitorState, data EntityDataResult) bool {
	// Check risk score threshold
	if threshold, ok := state.Config.AlertThresholds["risk_score_change"]; ok {
		delta := data.RiskScore - state.LastRiskScore
		if delta > 0 && delta >= threshold {
			return true
		}
	}

	// Check absolute risk score threshold
	if threshold, ok := state.Config.AlertThresholds["risk_score_max"]; ok {
		if data.RiskScore >= threshold && state.LastRiskScore < threshold {
			return true
		}
	}

	// Check for any alerts in the data
	if len(data.Alerts) > 0 {
		return true
	}

	return false
}

// determineEventType determines the event type based on state and data.
func determineEventType(state EntityMonitorState, data EntityDataResult) string {
	switch state.MonitorType {
	case "price_alert":
		return "PRICE_CHANGE"
	case "news_sentiment":
		return "NEWS_SENTIMENT"
	case "filing_watch":
		return "NEW_FILING"
	case "risk_threshold":
		return "RISK_THRESHOLD_BREACH"
	case "ownership_change":
		return "OWNERSHIP_CHANGE"
	default:
		return "GENERAL_ALERT"
	}
}

// determineSeverity determines alert severity based on thresholds.
func determineSeverity(state EntityMonitorState, data EntityDataResult) string {
	delta := data.RiskScore - state.LastRiskScore

	if data.RiskScore >= 80 || delta >= 20 {
		return "critical"
	}
	if data.RiskScore >= 60 || delta >= 10 {
		return "high"
	}
	if data.RiskScore >= 40 || delta >= 5 {
		return "medium"
	}
	return "low"
}

// generateAlertTitle generates a human-readable alert title.
func generateAlertTitle(monitorType string, data EntityDataResult) string {
	switch monitorType {
	case "price_alert":
		if data.PriceChange > 0 {
			return "Price increased significantly"
		}
		return "Price decreased significantly"
	case "risk_threshold":
		return "Risk score threshold breached"
	case "news_sentiment":
		return "Negative news sentiment detected"
	case "filing_watch":
		return "New regulatory filing detected"
	default:
		return "Entity alert triggered"
	}
}

// PauseMonitorSignal is sent to pause a monitor.
type PauseMonitorSignal struct {
	Reason string `json:"reason,omitempty"`
}

// ResumeMonitorSignal is sent to resume a monitor.
type ResumeMonitorSignal struct{}

// UpdateConfigSignal is sent to update monitor configuration.
type UpdateConfigSignal struct {
	Config EntityMonitorConfig `json:"config"`
}

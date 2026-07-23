package intelligence

import (
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

type MonitorConfig struct {
	CheckInterval time.Duration
}

type MonitorState struct {
	EntityID      string
	LastCheckTime time.Time
	LastRiskScore float64
	Config        MonitorConfig
	RunID         uuid.UUID // Unique ID for this monitor chain
	Seq           int64     // Sequence number for event sourcing
	LastHash      string    // Hash of the previous event
}

type FinancialData struct {
	RiskScore float64
	// Other fields...
}

// EntityMonitorWorkflow continuously monitors a financial entity for risk signals
func EntityMonitorWorkflow(ctx workflow.Context, state MonitorState) error {
	logger := workflow.GetLogger(ctx)

	// Initialize state if new
	if state.RunID == uuid.Nil {
		state.RunID = uuid.New()
		state.Seq = 0
		state.LastHash = "" // Genesis hash
	}

	// Loop for a fixed number of iterations to manage history size
	for i := 0; i < 100; i++ {
		// 1. Wait for the next check interval
		workflow.Sleep(ctx, state.Config.CheckInterval)

		// 2. Execute Activity to fetch new data
		var currentData FinancialData
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
		})

		// Mock data for compilation
		currentData = FinancialData{RiskScore: 85.0}

		// 3. Log "Check Performed" Event
		state.Seq++
		var eventHash string
		err := workflow.ExecuteActivity(ctx, LogEventActivity, LogEventInput{
			RunID:      state.RunID,
			Seq:        state.Seq,
			EventType:  "CHECK_PERFORMED",
			Payload:    map[string]interface{}{"entity_id": state.EntityID, "risk_score": currentData.RiskScore},
			ParentHash: state.LastHash,
		}).Get(ctx, &eventHash)
		
		if err != nil {
			logger.Error("Failed to log event", "error", err)
			// In a strict regulatory environment, we might fail here. 
			// For availability, we might continue but log a critical error.
		} else {
			state.LastHash = eventHash
		}

		// 4. Detect Drift
		if currentData.RiskScore != state.LastRiskScore {
			logger.Info("Risk Score Changed", "Old", state.LastRiskScore, "New", currentData.RiskScore)
			
			// 5. Log "Alert Generated" Event
			state.Seq++
			err = workflow.ExecuteActivity(ctx, LogEventActivity, LogEventInput{
				RunID:      state.RunID,
				Seq:        state.Seq,
				EventType:  "RISK_ALERT",
				Payload:    map[string]interface{}{"entity_id": state.EntityID, "old_score": state.LastRiskScore, "new_score": currentData.RiskScore},
				ParentHash: state.LastHash,
			}).Get(ctx, &eventHash)

			if err == nil {
				state.LastHash = eventHash
			}
			
			state.LastRiskScore = currentData.RiskScore
		}

		state.LastCheckTime = workflow.Now(ctx)
	}

	// 6. ContinueAsNew: Restart workflow with clean history but preserved state (including hash chain)
	return workflow.NewContinueAsNewError(ctx, EntityMonitorWorkflow, state)
}

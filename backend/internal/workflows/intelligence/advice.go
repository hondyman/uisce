package intelligence

import (
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/guardrails"
	"go.temporal.io/sdk/workflow"
)

type AdviceRequest struct {
	ClientID      string
	Objective     string
	DraftContent  string
	PolicyVersion string
}

type AdvisorSignal struct {
	Action  string // "approve", "reject", "revise"
	Comment string
	ActorID string
}

// AdviceWorkflow orchestrates the generation and approval of advice
func AdviceWorkflow(ctx workflow.Context, req AdviceRequest) (string, error) {
	logger := workflow.GetLogger(ctx)
	runID := uuid.New()
	seq := int64(0)
	lastHash := ""

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// 1. Log Start & Create Run
	seq++
	var eventHash string
	err := workflow.ExecuteActivity(ctx, "StartAdviceSession", LogEventInput{
		RunID:      runID,
		Seq:        seq,
		EventType:  "WORKFLOW_STARTED",
		Payload:    map[string]interface{}{"client_id": req.ClientID, "objective": req.Objective},
		ParentHash: lastHash,
	}, req.ClientID, req.Objective, req.PolicyVersion).Get(ctx, &eventHash)
	if err != nil {
		return "", err
	}
	lastHash = eventHash

	// 2. Evaluate Guardrails
	var outcome guardrails.Outcome
	err = workflow.ExecuteActivity(ctx, EvaluateGuardrailsActivity, req.DraftContent).Get(ctx, &outcome)
	if err != nil {
		return "", err
	}

	// 3. Log Guardrails Outcome & Update Status
	seq++
	err = workflow.ExecuteActivity(ctx, "RecordGuardrailOutcome", LogEventInput{
		RunID:      runID,
		Seq:        seq,
		EventType:  "GUARDRAILS_EVALUATED",
		Payload:    map[string]interface{}{"allowed": outcome.Allowed, "violations": outcome.Violations},
		ParentHash: lastHash,
	}, outcome).Get(ctx, &eventHash)
	if err != nil {
		return "", err
	}
	lastHash = eventHash

	// 4. Human-in-the-Loop Gate
	if outcome.RequiresHuman {
		logger.Info("Guardrails flagged content, awaiting advisor approval", "violations", outcome.Violations)

		// Wait for signal
		sigChan := workflow.GetSignalChannel(ctx, "advisor-signal")
		var sig AdvisorSignal

		// Block until signal received
		sigChan.Receive(ctx, &sig)

		// Log Signal
		seq++
		_ = workflow.ExecuteActivity(ctx, LogEventActivity, LogEventInput{
			RunID:      runID,
			Seq:        seq,
			EventType:  "ADVISOR_SIGNAL_RECEIVED",
			Payload:    map[string]interface{}{"action": sig.Action, "actor_id": sig.ActorID, "comment": sig.Comment},
			ParentHash: lastHash,
		}).Get(ctx, &eventHash)
		lastHash = eventHash

		if sig.Action == "reject" {
			return "REJECTED", nil
		}
		if sig.Action == "revise" {
			// In a real flow, this might loop back to generation
			return "REVISION_REQUESTED", nil
		}
	}

	// 5. Finalize
	seq++
	_ = workflow.ExecuteActivity(ctx, LogEventActivity, LogEventInput{
		RunID:      runID,
		Seq:        seq,
		EventType:  "ADVICE_PUBLISHED",
		Payload:    map[string]interface{}{"final_content_len": len(req.DraftContent)},
		ParentHash: lastHash,
	}).Get(ctx, &eventHash)

	return req.DraftContent, nil
}

package regulatory

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"github.com/hondyman/uisce/backend/internal/compliance/drift"
)

const (
	RegulatoryChangeTaskQueue = "compliance-regulatory-change"

	SignalTriageCase   = "TriageCaseSignal"
	SignalReviewCase   = "ReviewCaseSignal"
	SignalApproveCase  = "ApproveCaseSignal"
	SignalPublishCase  = "PublishCaseSignal"
	SignalRejectCase   = "RejectCaseSignal"
	SignalCloseNoImpact = "CloseNoImpactSignal"
)

type RegulatoryWorkflowInput struct {
	IntakeReq IntakeRequest `json:"intake_req"`
}

type TriageSignalPayload struct {
	Classification  CaseClassification `json:"classification"`
	TriageNotes     string             `json:"triage_notes"`
	TriagedBy       string             `json:"triaged_by"`
	AffectedRuleIDs []uuid.UUID        `json:"affected_rule_ids"`
}

type ReviewSignalPayload struct {
	StewardID string         `json:"steward_id"`
	DiffViews []RuleDiffView `json:"diff_views"`
}

type ApproveSignalPayload struct {
	StewardID string      `json:"steward_id"`
	Notes     string      `json:"notes"`
	Drafts    []RuleDraft `json:"drafts"`
}

type PublishSignalPayload struct {
	StewardID string      `json:"steward_id"`
	Drafts    []RuleDraft `json:"drafts"`
}

type RejectSignalPayload struct {
	Actor  string `json:"actor"`
	Reason string `json:"reason"`
}

type CloseNoImpactSignalPayload struct {
	Actor  string `json:"actor"`
	Reason string `json:"reason"`
}

type RegulatoryWorkflowResult struct {
	CaseID                uuid.UUID                   `json:"case_id"`
	CaseCode              string                      `json:"case_code"`
	Status                CaseStatus                  `json:"status"`
	PublishedRuleVersions []PublishedRuleVersionEntry `json:"published_rule_versions,omitempty"`
	CompletedAt           time.Time                   `json:"completed_at"`
	Notes                 string                      `json:"notes"`
}

// RegulatoryChangeWorkflow orchestrates the 5-stage regulatory change lifecycle
func RegulatoryChangeWorkflow(ctx workflow.Context, input RegulatoryWorkflowInput) (*RegulatoryWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("RegulatoryChangeWorkflow started", "case_code", input.IntakeReq.CaseCode)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &sdktemporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Open Case (INTAKED)
	var createdCase RegulatoryChangeCase
	err := workflow.ExecuteActivity(ctx, "CreateCaseActivity", input.IntakeReq).Get(ctx, &createdCase)
	if err != nil {
		return nil, fmt.Errorf("failed to open case: %w", err)
	}

	caseID := createdCase.ID
	caseCode := createdCase.CaseCode
	dueDuration := createdCase.DueAt.Sub(workflow.Now(ctx))
	if dueDuration <= 0 {
		dueDuration = 30 * 24 * time.Hour
	}

	// Channels for human steward signals
	triageChan := workflow.GetSignalChannel(ctx, SignalTriageCase)
	reviewChan := workflow.GetSignalChannel(ctx, SignalReviewCase)
	approveChan := workflow.GetSignalChannel(ctx, SignalApproveCase)
	publishChan := workflow.GetSignalChannel(ctx, SignalPublishCase)
	rejectChan := workflow.GetSignalChannel(ctx, SignalRejectCase)
	closeChan := workflow.GetSignalChannel(ctx, SignalCloseNoImpact)

	currentStatus := StatusIntaked
	escalated := false
	var publishedEntries []PublishedRuleVersionEntry

	for {
		selector := workflow.NewSelector(ctx)

		// TTL Escalation Timer
		if !escalated {
			ttlTimer := workflow.NewTimer(ctx, dueDuration)
			selector.AddFuture(ttlTimer, func(f workflow.Future) {
				logger.Warn("Case exceeded TTL without resolution; escalating", "case_code", caseCode)
				var actErr error
				_ = workflow.ExecuteActivity(ctx, "EscalateCaseActivity", caseID).Get(ctx, &actErr)
				escalated = true
			})
		}

		// Reject Signal (Available in all non-terminal states)
		selector.AddReceive(rejectChan, func(c workflow.ReceiveChannel, more bool) {
			var sig RejectSignalPayload
			c.Receive(ctx, &sig)
			logger.Info("Received reject signal", "actor", sig.Actor, "reason", sig.Reason)
			var actErr error
			_ = workflow.ExecuteActivity(ctx, "RejectCaseActivity", caseID, sig.Actor, sig.Reason).Get(ctx, &actErr)
			currentStatus = StatusRejected
		})

		// Close No Impact Signal (Available in TRIAGED state)
		selector.AddReceive(closeChan, func(c workflow.ReceiveChannel, more bool) {
			var sig CloseNoImpactSignalPayload
			c.Receive(ctx, &sig)
			logger.Info("Received close-no-impact signal", "actor", sig.Actor, "reason", sig.Reason)
			var actErr error
			_ = workflow.ExecuteActivity(ctx, "CloseCaseActivity", caseID, sig.Actor, sig.Reason).Get(ctx, &actErr)
			currentStatus = StatusClosedNoImpact
		})

		// Stage 2: Triage Signal
		selector.AddReceive(triageChan, func(c workflow.ReceiveChannel, more bool) {
			var sig TriageSignalPayload
			c.Receive(ctx, &sig)
			triageReq := TriageRequest{
				CaseID:          caseID,
				Classification:  sig.Classification,
				TriageNotes:     sig.TriageNotes,
				TriagedBy:       sig.TriagedBy,
				AffectedRuleIDs: sig.AffectedRuleIDs,
			}
			var actErr error
			err := workflow.ExecuteActivity(ctx, "TriageCaseActivity", triageReq).Get(ctx, &actErr)
			if err == nil {
				currentStatus = StatusTriaged
			}
		})

		// Stage 3: Start Review Signal
		selector.AddReceive(reviewChan, func(c workflow.ReceiveChannel, more bool) {
			var sig ReviewSignalPayload
			c.Receive(ctx, &sig)
			var actErr error
			err := workflow.ExecuteActivity(ctx, "StartReviewActivity", caseID, sig.StewardID, sig.DiffViews).Get(ctx, &actErr)
			if err == nil {
				currentStatus = StatusUnderReview
			}
		})

		// Stage 4: Approve Signal (Runs Corpus Gate + Approves)
		selector.AddReceive(approveChan, func(c workflow.ReceiveChannel, more bool) {
			var sig ApproveSignalPayload
			c.Receive(ctx, &sig)
			var corpusResult drift.CorpusRunResult
			err := workflow.ExecuteActivity(ctx, "ExecuteCorpusGateActivity", caseID, sig.StewardID, sig.Drafts).Get(ctx, &corpusResult)
			if err != nil {
				logger.Error("Corpus gate rejected approval", "error", err)
				return
			}

			var actErr error
			err = workflow.ExecuteActivity(ctx, "ApproveCaseActivity", caseID, sig.StewardID, sig.Notes).Get(ctx, &actErr)
			if err == nil {
				currentStatus = StatusApprovedForPublish
			}
		})

		// Stage 5: Publish Release Signal
		selector.AddReceive(publishChan, func(c workflow.ReceiveChannel, more bool) {
			var sig PublishSignalPayload
			c.Receive(ctx, &sig)
			pubReq := PublishRequest{
				CaseID:    caseID,
				StewardID: sig.StewardID,
				Drafts:    sig.Drafts,
			}
			err := workflow.ExecuteActivity(ctx, "PublishReleaseActivity", pubReq).Get(ctx, &publishedEntries)
			if err == nil {
				currentStatus = StatusPublished
			}
		})

		selector.Select(ctx)

		// Terminal state check
		if currentStatus == StatusPublished || currentStatus == StatusClosedNoImpact || currentStatus == StatusRejected || currentStatus == StatusExpired {
			break
		}
	}

	return &RegulatoryWorkflowResult{
		CaseID:                caseID,
		CaseCode:              caseCode,
		Status:                currentStatus,
		PublishedRuleVersions: publishedEntries,
		CompletedAt:           workflow.Now(ctx),
	}, nil
}

// Activities implementation wrapping Service methods
type Activities struct {
	service *Service
}

func NewActivities(service *Service) *Activities {
	return &Activities{service: service}
}

func (a *Activities) CreateCaseActivity(ctx context.Context, req IntakeRequest) (*RegulatoryChangeCase, error) {
	return a.service.CreateCase(ctx, req)
}

func (a *Activities) TriageCaseActivity(ctx context.Context, req TriageRequest) error {
	return a.service.TriageCase(ctx, req)
}

func (a *Activities) StartReviewActivity(ctx context.Context, caseID uuid.UUID, stewardID string, diffs []RuleDiffView) error {
	return a.service.StartReview(ctx, caseID, stewardID, diffs)
}

func (a *Activities) ExecuteCorpusGateActivity(ctx context.Context, caseID uuid.UUID, stewardID string, drafts []RuleDraft) (*drift.CorpusRunResult, error) {
	return a.service.ExecuteCorpusGate(ctx, caseID, stewardID, drafts)
}

func (a *Activities) ApproveCaseActivity(ctx context.Context, caseID uuid.UUID, stewardID string, notes string) error {
	return a.service.ApproveCase(ctx, caseID, stewardID, notes)
}

func (a *Activities) PublishReleaseActivity(ctx context.Context, req PublishRequest) ([]PublishedRuleVersionEntry, error) {
	return a.service.PublishRelease(ctx, req)
}

func (a *Activities) EscalateCaseActivity(ctx context.Context, caseID uuid.UUID) error {
	return a.service.EscalateCase(ctx, caseID)
}

func (a *Activities) RejectCaseActivity(ctx context.Context, caseID uuid.UUID, actor, reason string) error {
	return a.service.RejectCase(ctx, caseID, actor, reason)
}

func (a *Activities) CloseCaseActivity(ctx context.Context, caseID uuid.UUID, actor, reason string) error {
	return a.service.CloseCaseNoImpact(ctx, caseID, actor, reason)
}

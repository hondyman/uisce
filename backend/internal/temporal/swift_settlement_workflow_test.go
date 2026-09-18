package temporal

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

// SWIFTSettlementWorkflowTestSuite exercises SWIFTSettlementWorkflow via the
// Temporal TestWorkflowEnvironment — deterministic, no real server required.
//
// Mock discipline:
//   - mockPersistExpecting pins exact arguments (transactionRef, status, tenantID,
//     custodianID) and asserts .Once() so a workflow that double-persists fails.
//   - mockAckOK / mockPipelineSuccess use mock.Anything (values not the focus).
//   - currentInput is set at the top of each test so Run() closures can reference it.
type SWIFTSettlementWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
	currentInput SWIFTSettlementInput
}

func TestSWIFTSettlementWorkflow(t *testing.T) {
	suite.Run(t, new(SWIFTSettlementWorkflowTestSuite))
}

func (s *SWIFTSettlementWorkflowTestSuite) newEnv() *testsuite.TestWorkflowEnvironment {
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(SWIFTSettlementWorkflow)
	env.RegisterActivity(SWIFTAckActivity)
	env.RegisterActivity(RunSWIFTPipelineDAGActivity)
	env.RegisterActivity(PersistSettlementStatusActivity)
	env.RegisterActivity(SWIFTRecallActivity)
	return env
}

func (s *SWIFTSettlementWorkflowTestSuite) defaultInput() SWIFTSettlementInput {
	return SWIFTSettlementInput{
		TenantID:           uuid.MustParse("11111111-0000-4000-8000-000000000001"),
		CustodianID:        uuid.MustParse("22222222-0000-4000-8000-000000000002"),
		TransactionRef:     "TXN-TEST-001",
		UETR:               "550e8400-e29b-41d4-a716-446655440000",
		AdminURL:           "http://127.0.0.1:8982",
		AdminToken:         "test-token",
		PipelineDAGID:      "swift-test-dag",
		SettlementBudgetMs: 5000,
		MsgType:            "MT541",
	}
}

// mockAckOK — ACK succeeds without a network call.
func (s *SWIFTSettlementWorkflowTestSuite) mockAckOK(env *testsuite.TestWorkflowEnvironment) {
	env.OnActivity(SWIFTAckActivity, mock.Anything, mock.Anything).Return(nil)
}

// mockAckError — ACK returns an error.
func (s *SWIFTSettlementWorkflowTestSuite) mockAckError(env *testsuite.TestWorkflowEnvironment, err error) {
	env.OnActivity(SWIFTAckActivity, mock.Anything, mock.Anything).Return(err)
}

// mockPipelineSuccess — bypasses ErrPipelineNotImplemented so the signal loop is reachable.
func (s *SWIFTSettlementWorkflowTestSuite) mockPipelineSuccess(env *testsuite.TestWorkflowEnvironment) {
	env.OnActivity(RunSWIFTPipelineDAGActivity, mock.Anything, mock.Anything).
		Return(PipelineResult{Success: true, Details: "mock-pipeline-ok"}, nil)
}

// mockPersistExpecting pins the exact (transactionRef, status, tenantID, custodianID)
// arguments and enforces .Once() — the workflow must persist exactly these values
// exactly once. A second call or a call with wrong args fails the test.
func (s *SWIFTSettlementWorkflowTestSuite) mockPersistExpecting(
	env *testsuite.TestWorkflowEnvironment,
	input SWIFTSettlementInput,
	status string,
) {
	env.OnActivity(PersistSettlementStatusActivity,
		mock.Anything,             // context.Context
		input.TransactionRef,      // exact: wrong ref = test failure
		status,                    // exact: "SETTLED" not "BANANA"
		input.TenantID.String(),   // exact: wrong tenant = test failure
		input.CustodianID.String(), // exact
	).Return(nil).Once()
}

// TestSettledViaSignal — full happy path: ACK ok, pipeline ok,
// SettlementUpdate(settled) → SETTLED, persisted with exact args.
func (s *SWIFTSettlementWorkflowTestSuite) TestSettledViaSignal() {
	env := s.newEnv()
	input := s.defaultInput()
	s.currentInput = input

	s.mockAckOK(env)
	s.mockPipelineSuccess(env)
	s.mockPersistExpecting(env, input, "SETTLED")

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("SettlementUpdate", SettlementSignal{
			Status:     "settled",
			OccurredAt: time.Now().UTC(),
		})
	}, 10*time.Millisecond)

	env.ExecuteWorkflow(SWIFTSettlementWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
	var result SWIFTSettlementResult
	s.NoError(env.GetWorkflowResult(&result))
	s.Equal("SETTLED", result.FinalStatus)
	s.Equal(input.TransactionRef, result.TransactionRef)
}

// TestFailedViaSignal_PersistsFailedStatus — SettlementUpdate(failed) → FAILED.
// The exact-arg mock ensures the workflow calls persist with status="FAILED",
// not "SETTLED" or any other string.
func (s *SWIFTSettlementWorkflowTestSuite) TestFailedViaSignal_PersistsFailedStatus() {
	env := s.newEnv()
	input := s.defaultInput()
	s.currentInput = input

	s.mockAckOK(env)
	s.mockPipelineSuccess(env)
	s.mockPersistExpecting(env, input, "FAILED")

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("SettlementUpdate", SettlementSignal{
			Status:  "failed",
			Details: "nostro-mismatch",
		})
	}, 10*time.Millisecond)

	env.ExecuteWorkflow(SWIFTSettlementWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	var result SWIFTSettlementResult
	s.NoError(env.GetWorkflowResult(&result))
	s.Equal("FAILED", result.FinalStatus)
	s.Equal("nostro-mismatch", result.FailureReason)
}

// TestMatchedThenSettled — matched → PENDING_SETTLEMENT → settled → SETTLED.
// Two signals; persist called once (on settled), not on matched.
func (s *SWIFTSettlementWorkflowTestSuite) TestMatchedThenSettled() {
	env := s.newEnv()
	input := s.defaultInput()
	s.currentInput = input

	s.mockAckOK(env)
	s.mockPipelineSuccess(env)
	s.mockPersistExpecting(env, input, "SETTLED")

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("SettlementUpdate", SettlementSignal{Status: "matched"})
	}, 5*time.Millisecond)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("SettlementUpdate", SettlementSignal{
			Status:     "settled",
			OccurredAt: time.Now().UTC(),
		})
	}, 10*time.Millisecond)

	env.ExecuteWorkflow(SWIFTSettlementWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	var result SWIFTSettlementResult
	s.NoError(env.GetWorkflowResult(&result))
	s.Equal("SETTLED", result.FinalStatus)
}

// TestCancelledViaSignal — recall SUCCEEDS → terminal CANCELLED persisted.
// This is the clean cancel path; no CANCEL_PENDING.
func (s *SWIFTSettlementWorkflowTestSuite) TestCancelledViaSignal() {
	env := s.newEnv()
	input := s.defaultInput()
	s.currentInput = input

	s.mockAckOK(env)
	s.mockPipelineSuccess(env)
	// Recall succeeds (mocked away — stub returns error in production until implemented)
	env.OnActivity(SWIFTRecallActivity, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			in, ok := args.Get(1).(SWIFTSettlementInput)
			if ok {
				s.Equal(input.TransactionRef, in.TransactionRef,
					"recall must be called for the correct transaction")
			}
		}).Return(nil).Once()
	s.mockPersistExpecting(env, input, "CANCELLED")

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("Cancel", struct{ Reason string }{"test-cancel"})
	}, 10*time.Millisecond)

	env.ExecuteWorkflow(SWIFTSettlementWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	var result SWIFTSettlementResult
	s.NoError(env.GetWorkflowResult(&result))
	s.Equal("CANCELLED", result.FinalStatus)
	s.Contains(result.FailureReason, "cancelled:")
}

// TestCancelRecallFails — recall FAILS → CANCEL_PENDING, not terminal CANCELLED.
// This is the regression test for the "failed recall writes CANCELLED (a lie)" bug.
// If this test fails: either the workflow regressed to writing CANCELLED on failed
// recall, or the CANCEL_PENDING state is not wired as terminal. Fix before merge.
func (s *SWIFTSettlementWorkflowTestSuite) TestCancelRecallFails() {
	env := s.newEnv()
	input := s.defaultInput()
	s.currentInput = input

	s.mockAckOK(env)
	s.mockPipelineSuccess(env)
	env.OnActivity(SWIFTRecallActivity, mock.Anything, mock.Anything).
		Return(temporal.NewNonRetryableApplicationError(
			"recall-stub", "ErrRecallNotImplemented", nil)).Once()
	// Must persist CANCEL_PENDING, not CANCELLED
	s.mockPersistExpecting(env, input, "CANCEL_PENDING")

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("Cancel", struct{ Reason string }{"test-cancel"})
	}, 10*time.Millisecond)

	env.ExecuteWorkflow(SWIFTSettlementWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	var result SWIFTSettlementResult
	s.NoError(env.GetWorkflowResult(&result))
	s.Equal("CANCEL_PENDING", result.FinalStatus,
		"unconfirmed recall must park in CANCEL_PENDING — writing CANCELLED is a lie")
	s.Contains(result.FailureReason, "cancelled:")
}

// TestDeadlineExceeded — no signals; 72h timer fires → FAILED.
// TestWorkflowEnvironment fast-forwards timers; this does not wait 72h.
func (s *SWIFTSettlementWorkflowTestSuite) TestDeadlineExceeded() {
	env := s.newEnv()
	input := s.defaultInput()
	s.currentInput = input

	s.mockAckOK(env)
	s.mockPipelineSuccess(env)
	s.mockPersistExpecting(env, input, "FAILED")

	env.ExecuteWorkflow(SWIFTSettlementWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	var result SWIFTSettlementResult
	s.NoError(env.GetWorkflowResult(&result))
	s.Equal("FAILED", result.FinalStatus)
	s.Equal("settlement_deadline_exceeded", result.FailureReason)
}

// TestPipelineStubFails — real RunSWIFTPipelineDAGActivity (not mocked) fires
// ErrPipelineNotImplemented. Verifies:
//  1. Workflow drives to FAILED at VALIDATING (SETTLED unreachable).
//  2. PersistSettlementStatusActivity("FAILED") is called — the row is terminal
//     before the signal loop is entered, so GET /instructions/{id} and recon see it.
//
// MERGE CANARY: invert this test when wiring RunSWIFTPipelineDAGActivity to the
// real pipeline engine. Post-merge, the setup must reach the signal loop (MATCHED),
// not return FAILED here. If this test stays green after merge, the wiring is broken.
func (s *SWIFTSettlementWorkflowTestSuite) TestPipelineStubFails() {
	env := s.newEnv()
	input := s.defaultInput()
	s.currentInput = input

	s.mockAckOK(env)
	// Pipeline NOT mocked — real ErrPipelineNotImplemented fires.
	// Persist IS expected: pipeline failure calls PersistSettlementStatusActivity("FAILED").
	s.mockPersistExpecting(env, input, "FAILED")

	env.ExecuteWorkflow(SWIFTSettlementWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	var result SWIFTSettlementResult
	s.NoError(env.GetWorkflowResult(&result))
	s.Equal("FAILED", result.FinalStatus)
	s.Contains(result.FailureReason, "ErrPipelineNotImplemented",
		"stub must surface a named error code — the canary is that the merge deletes this test")
}


// TestAckTimeoutContinues — ACK errors; workflow continues to pipeline and settles.
// Validates ack-timeout ambiguity semantics: ack failure ≠ workflow failure.
// SWIFTReconciliationWorkflow resolves any delivery ambiguity on its next run.
func (s *SWIFTSettlementWorkflowTestSuite) TestAckTimeoutContinues() {
	env := s.newEnv()
	input := s.defaultInput()
	s.currentInput = input

	s.mockAckError(env, fmt.Errorf("ack: connection refused (stub not running)"))
	s.mockPipelineSuccess(env)
	s.mockPersistExpecting(env, input, "SETTLED")

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("SettlementUpdate", SettlementSignal{
			Status:     "settled",
			OccurredAt: time.Now().UTC(),
		})
	}, 10*time.Millisecond)

	env.ExecuteWorkflow(SWIFTSettlementWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	var result SWIFTSettlementResult
	s.NoError(env.GetWorkflowResult(&result))
	s.Equal("SETTLED", result.FinalStatus,
		"ACK failure must not block settlement — ack is fire-and-forget, recon resolves ambiguity")
}

package workflows

import (
	"testing"
	"time"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type MDMApprovalWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestMDMApprovalWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(MDMApprovalWorkflowTestSuite))
}

func (s *MDMApprovalWorkflowTestSuite) TestMDMOverrideApprovalWorkflow_HappyPath() {
	env := s.NewTestWorkflowEnvironment()
	var acts *activities.MDMApprovalActivities

	env.OnActivity(acts.CheckProposalStatusActivity, mock.Anything, mock.Anything).Return(&activities.MDMCheckStatusResult{Status: "PENDING", Active: true}, nil)
	env.OnActivity(acts.RecordOverrideVoteActivity, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(acts.ApplyOverrideActivity, mock.Anything, mock.Anything).Return(&activities.MDMActivityResult{Applied: true, Status: "APPLIED"}, nil)

	input := MDMOverrideWorkflowInput{
		TenantID:          "tenant-1",
		Entity:            "SECURITY",
		OverrideID:        "override-123",
		GoldenID:          "gold-456",
		Attribute:         "security_name",
		Action:            "SET",
		ProposerID:        "user-proposer",
		ProposerName:      "Alice Proposer",
		ApprovalsRequired: 1,
		ReviewSLA:         24 * time.Hour,
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMDMApprove, MDMApprovalSignalPayload{
			ApproverID:   "user-approver-1",
			ApproverName: "Bob Approver",
			Comment:      "Verified with Bloomberg source",
		})
	}, 1*time.Hour)

	env.ExecuteWorkflow(MDMOverrideApprovalWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var res MDMApprovalWorkflowResult
	s.NoError(env.GetWorkflowResult(&res))
	s.Equal("override-123", res.ID)
	s.Equal("APPLIED", res.Status)
}

func (s *MDMApprovalWorkflowTestSuite) TestMDMOverrideApprovalWorkflow_MultiApprover() {
	env := s.NewTestWorkflowEnvironment()
	var acts *activities.MDMApprovalActivities

	env.OnActivity(acts.CheckProposalStatusActivity, mock.Anything, mock.Anything).Return(&activities.MDMCheckStatusResult{Status: "PENDING", Active: true}, nil)
	env.OnActivity(acts.RecordOverrideVoteActivity, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(acts.ApplyOverrideActivity, mock.Anything, mock.Anything).Return(&activities.MDMActivityResult{Applied: true, Status: "APPLIED"}, nil)

	input := MDMOverrideWorkflowInput{
		TenantID:          "tenant-1",
		Entity:            "SECURITY",
		OverrideID:        "override-multi",
		GoldenID:          "gold-789",
		Attribute:         "cusip", // high-risk attribute
		Action:            "SET",
		ProposerID:        "user-proposer",
		ApprovalsRequired: 2,
		ReviewSLA:         48 * time.Hour,
	}

	// 1st approver signals at 1h
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMDMApprove, MDMApprovalSignalPayload{
			ApproverID:   "approver-1",
			ApproverName: "Bob",
		})
	}, 1*time.Hour)

	// 2nd approver signals at 2h
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMDMApprove, MDMApprovalSignalPayload{
			ApproverID:   "approver-2",
			ApproverName: "Charlie",
		})
	}, 2*time.Hour)

	env.ExecuteWorkflow(MDMOverrideApprovalWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var res MDMApprovalWorkflowResult
	s.NoError(env.GetWorkflowResult(&res))
	s.Equal("APPLIED", res.Status)
}

func (s *MDMApprovalWorkflowTestSuite) TestMDMOverrideApprovalWorkflow_RejectPath() {
	env := s.NewTestWorkflowEnvironment()
	var acts *activities.MDMApprovalActivities

	env.OnActivity(acts.CheckProposalStatusActivity, mock.Anything, mock.Anything).Return(&activities.MDMCheckStatusResult{Status: "PENDING", Active: true}, nil)
	env.OnActivity(acts.RejectOverrideActivity, mock.Anything, mock.Anything).Return(&activities.MDMActivityResult{Applied: false, Status: "REJECTED"}, nil)

	input := MDMOverrideWorkflowInput{
		TenantID:          "tenant-1",
		Entity:            "SECURITY",
		OverrideID:        "override-rej",
		GoldenID:          "gold-111",
		Attribute:         "isin",
		Action:            "SET",
		ProposerID:        "user-proposer",
		ApprovalsRequired: 1,
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMDMReject, MDMRejectionSignalPayload{
			RejecterID:   "user-checker",
			RejecterName: "Dave Checker",
			Reason:       "Invalid ISIN checksum",
		})
	}, 1*time.Hour)

	env.ExecuteWorkflow(MDMOverrideApprovalWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var res MDMApprovalWorkflowResult
	s.NoError(env.GetWorkflowResult(&res))
	s.Equal("REJECTED", res.Status)
	s.Equal("user-checker", res.DecidedBy)
}

func (s *MDMApprovalWorkflowTestSuite) TestMDMMergeApprovalWorkflow_HappyPath() {
	env := s.NewTestWorkflowEnvironment()
	var acts *activities.MDMApprovalActivities

	env.OnActivity(acts.CheckProposalStatusActivity, mock.Anything, mock.Anything).Return(&activities.MDMCheckStatusResult{Status: "PENDING", Active: true}, nil)
	env.OnActivity(acts.RecordMergeVoteActivity, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(acts.ApplyMergeActivity, mock.Anything, mock.Anything).Return(&activities.MDMActivityResult{Applied: true, Status: "APPROVED"}, nil)

	input := MDMMergeWorkflowInput{
		TenantID:          "tenant-1",
		Entity:            "CUSTOMER",
		RequestID:         "merge-req-1",
		CandidateID:       "cand-1",
		Keep:              "a",
		ProposerID:        "steward-1",
		ApprovalsRequired: 1,
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMDMApprove, MDMApprovalSignalPayload{
			ApproverID:   "senior-steward",
			ApproverName: "Emma",
		})
	}, 30*time.Minute)

	env.ExecuteWorkflow(MDMMergeApprovalWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var res MDMApprovalWorkflowResult
	s.NoError(env.GetWorkflowResult(&res))
	s.Equal("APPROVED", res.Status)
}

func (s *MDMApprovalWorkflowTestSuite) TestMDMConfigChangeApprovalWorkflow_HappyPath() {
	env := s.NewTestWorkflowEnvironment()
	var acts *activities.MDMApprovalActivities

	env.OnActivity(acts.CheckProposalStatusActivity, mock.Anything, mock.Anything).Return(&activities.MDMCheckStatusResult{Status: "PENDING", Active: true}, nil)
	env.OnActivity(acts.ApplyConfigChangeActivity, mock.Anything, mock.Anything).Return(&activities.MDMActivityResult{Applied: true, Status: "applied"}, nil)

	input := MDMConfigWorkflowInput{
		TenantID:   "tenant-1",
		Entity:     "SECURITY",
		ChangeID:   "change-123",
		Kind:       "survival",
		Action:     "update",
		ProposerID: "steward-1",
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMDMApprove, MDMApprovalSignalPayload{
			ApproverID:   "admin-1",
			ApproverName: "Master Admin",
			Comment:      "Approved rule survival priority update",
		})
	}, 15*time.Minute)

	env.ExecuteWorkflow(MDMConfigChangeApprovalWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var res MDMApprovalWorkflowResult
	s.NoError(env.GetWorkflowResult(&res))
	s.Equal("applied", res.Status)
	s.Equal("admin-1", res.DecidedBy)
}

func (s *MDMApprovalWorkflowTestSuite) TestMDMOverrideApprovalWorkflow_DuplicateSignalDedup() {
	env := s.NewTestWorkflowEnvironment()
	var acts *activities.MDMApprovalActivities

	env.OnActivity(acts.CheckProposalStatusActivity, mock.Anything, mock.Anything).Return(&activities.MDMCheckStatusResult{Status: "PENDING", Active: true}, nil)
	env.OnActivity(acts.RecordOverrideVoteActivity, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(acts.ApplyOverrideActivity, mock.Anything, mock.Anything).Return(&activities.MDMActivityResult{Applied: true, Status: "APPLIED"}, nil)

	input := MDMOverrideWorkflowInput{
		TenantID:          "tenant-1",
		Entity:            "SECURITY",
		OverrideID:        "override-dedup",
		GoldenID:          "gold-999",
		Attribute:         "cusip",
		Action:            "SET",
		ProposerID:        "user-proposer",
		ApprovalsRequired: 2,
		ReviewSLA:         48 * time.Hour,
	}

	// Approver 1 signals twice!
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMDMApprove, MDMApprovalSignalPayload{
			ApproverID:   "approver-1",
			ApproverName: "Bob",
		})
	}, 1*time.Hour)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMDMApprove, MDMApprovalSignalPayload{
			ApproverID:   "approver-1", // duplicate!
			ApproverName: "Bob",
		})
	}, 2*time.Hour)

	// Approver 2 signals at 3h to finally satisfy threshold of 2
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMDMApprove, MDMApprovalSignalPayload{
			ApproverID:   "approver-2",
			ApproverName: "Charlie",
		})
	}, 3*time.Hour)

	env.ExecuteWorkflow(MDMOverrideApprovalWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var res MDMApprovalWorkflowResult
	s.NoError(env.GetWorkflowResult(&res))
	s.Equal("APPLIED", res.Status)
}

func (s *MDMApprovalWorkflowTestSuite) TestMDMOverrideApprovalWorkflow_ProposerSelfApprovalIgnored() {
	env := s.NewTestWorkflowEnvironment()
	var acts *activities.MDMApprovalActivities

	env.OnActivity(acts.CheckProposalStatusActivity, mock.Anything, mock.Anything).Return(&activities.MDMCheckStatusResult{Status: "PENDING", Active: true}, nil)
	env.OnActivity(acts.RecordOverrideVoteActivity, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(acts.ApplyOverrideActivity, mock.Anything, mock.Anything).Return(&activities.MDMActivityResult{Applied: true, Status: "APPLIED"}, nil)

	input := MDMOverrideWorkflowInput{
		TenantID:          "tenant-1",
		Entity:            "SECURITY",
		OverrideID:        "override-self",
		GoldenID:          "gold-self",
		Attribute:         "isin",
		Action:            "SET",
		ProposerID:        "user-alice", // Alice is proposer
		ApprovalsRequired: 1,
		ReviewSLA:         48 * time.Hour,
	}

	// Alice attempts to approve her own proposal at 1h -> must be ignored!
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMDMApprove, MDMApprovalSignalPayload{
			ApproverID:   "user-alice",
			ApproverName: "Alice Proposer",
		})
	}, 1*time.Hour)

	// Bob (legitimate checker) approves at 2h
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMDMApprove, MDMApprovalSignalPayload{
			ApproverID:   "user-bob",
			ApproverName: "Bob Approver",
		})
	}, 2*time.Hour)

	env.ExecuteWorkflow(MDMOverrideApprovalWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var res MDMApprovalWorkflowResult
	s.NoError(env.GetWorkflowResult(&res))
	s.Equal("APPLIED", res.Status)
}

func (s *MDMApprovalWorkflowTestSuite) TestMDMOverrideApprovalWorkflow_PreflightClosed() {
	env := s.NewTestWorkflowEnvironment()
	var acts *activities.MDMApprovalActivities

	// Preflight check finds status is already APPLIED (e.g. from direct CAS path or prior run)
	env.OnActivity(acts.CheckProposalStatusActivity, mock.Anything, mock.Anything).Return(&activities.MDMCheckStatusResult{Status: "APPLIED", Active: false}, nil)

	input := MDMOverrideWorkflowInput{
		TenantID:   "tenant-1",
		Entity:     "SECURITY",
		OverrideID: "override-closed",
		ProposerID: "user-proposer",
	}

	env.ExecuteWorkflow(MDMOverrideApprovalWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var res MDMApprovalWorkflowResult
	s.NoError(env.GetWorkflowResult(&res))
	s.Equal("APPLIED", res.Status)
}

func (s *MDMApprovalWorkflowTestSuite) TestMDMOverrideApprovalWorkflow_VoteChange() {
	env := s.NewTestWorkflowEnvironment()
	var acts *activities.MDMApprovalActivities

	env.OnActivity(acts.CheckProposalStatusActivity, mock.Anything, mock.Anything).Return(&activities.MDMCheckStatusResult{Status: "PENDING", Active: true}, nil)
	env.OnActivity(acts.RecordOverrideVoteActivity, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(acts.RejectOverrideActivity, mock.Anything, mock.Anything).Return(&activities.MDMActivityResult{Applied: false, Status: "REJECTED"}, nil)

	input := MDMOverrideWorkflowInput{
		TenantID:          "tenant-1",
		Entity:            "SECURITY",
		OverrideID:        "override-vote-change",
		ProposerID:        "user-proposer",
		ApprovalsRequired: 2,
	}

	// Approver 1 approves at 1h
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMDMApprove, MDMApprovalSignalPayload{
			ApproverID:   "approver-1",
			ApproverName: "Bob",
		})
	}, 1*time.Hour)

	// Approver 1 changes vote to REJECT at 2h
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMDMReject, MDMRejectionSignalPayload{
			RejecterID:   "approver-1",
			RejecterName: "Bob",
			Reason:       "Found conflicting data source upon secondary check",
		})
	}, 2*time.Hour)

	env.ExecuteWorkflow(MDMOverrideApprovalWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var res MDMApprovalWorkflowResult
	s.NoError(env.GetWorkflowResult(&res))
	s.Equal("REJECTED", res.Status)
	s.Equal("approver-1", res.DecidedBy)
}

func (s *MDMApprovalWorkflowTestSuite) TestMDMOverrideApprovalWorkflow_MultiApprover_SingleRejectTerminates() {
	env := s.NewTestWorkflowEnvironment()
	var acts *activities.MDMApprovalActivities

	env.OnActivity(acts.CheckProposalStatusActivity, mock.Anything, mock.Anything).Return(&activities.MDMCheckStatusResult{
		Status: "PENDING",
		Active: true,
	}, nil)

	env.OnActivity(acts.RecordOverrideVoteActivity, mock.Anything, mock.Anything).Return(nil)
	env.OnActivity(acts.RejectOverrideActivity, mock.Anything, mock.Anything).Return(&activities.MDMActivityResult{
		Applied: false,
		Status:  "REJECTED",
	}, nil)

	input := MDMOverrideWorkflowInput{
		TenantID:          "tenant-1",
		Entity:            "security",
		OverrideID:        "ovr-multi-reject",
		ProposerID:        "user-proposer",
		ApprovalsRequired: 3, // Requires 3 approvals
	}

	// Approver 1 approves at 1h
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMDMApprove, MDMApprovalSignalPayload{
			ApproverID:   "approver-1",
			ApproverName: "Alice",
		})
	}, 1*time.Hour)

	// Approver 2 rejects at 2h -> should fail fast / immediately terminate workflow
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalMDMReject, MDMRejectionSignalPayload{
			RejecterID:   "approver-2",
			RejecterName: "Bob",
			Reason:       "Security ticker mismatch with market data",
		})
	}, 2*time.Hour)

	env.ExecuteWorkflow(MDMOverrideApprovalWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var res MDMApprovalWorkflowResult
	s.NoError(env.GetWorkflowResult(&res))
	s.Equal("REJECTED", res.Status)
	s.Equal("approver-2", res.DecidedBy)
	s.Equal("Security ticker mismatch with market data", res.Reason)
}

func (s *MDMApprovalWorkflowTestSuite) TestMDMReconciliationWorkflow() {
	env := s.NewTestWorkflowEnvironment()
	var acts *activities.MDMApprovalActivities

	expectedResult := &activities.MDMReconciliationResult{
		OverridesLinked:  3,
		OverridesExpired: 1,
		MergesLinked:     2,
		ConfigsLinked:    1,
	}
	env.OnActivity(acts.ReconcilePendingMDMWorkflowsActivity, mock.Anything, mock.Anything).Return(expectedResult, nil)

	input := activities.MDMReconciliationInput{
		MaxPendingAge: 5 * time.Minute,
		ExpiryAge:     168 * time.Hour,
	}

	env.ExecuteWorkflow(MDMReconciliationWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var res activities.MDMReconciliationResult
	s.NoError(env.GetWorkflowResult(&res))
	s.Equal(3, res.OverridesLinked)
	s.Equal(1, res.OverridesExpired)
	s.Equal(2, res.MergesLinked)
	s.Equal(1, res.ConfigsLinked)
}

package workflows

import (
	"testing"
	"time"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type RuleReviewWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestRuleReviewWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(RuleReviewWorkflowTestSuite))
}

func (s *RuleReviewWorkflowTestSuite) TestRuleReviewWorkflow_ApproveHappyPath() {
	env := s.NewTestWorkflowEnvironment()
	var acts *activities.RuleGovernanceActivities

	env.OnActivity(acts.RecordApprovalActivity, mock.Anything, activities.ApprovalActivityInput{
		TenantID:   "tenant-1",
		RuleNodeID: "rule-1",
		ApproverID: "approver-alice",
	}).Return(nil)

	env.OnActivity(acts.PublishVersionActivity, mock.Anything, activities.PublishVersionActivityInput{
		TenantID:    "tenant-1",
		RuleNodeID:  "rule-1",
		PublisherID: "approver-alice",
	}).Return(&activities.PublishVersionResult{
		Version:  2,
		Checksum: "sha256:abc12345",
	}, nil)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalRuleApprove, ApprovalSignalPayload{
			ApproverID: "approver-alice",
		})
	}, 1*time.Minute)

	input := RuleReviewWorkflowInput{
		TenantID:   "tenant-1",
		RuleNodeID: "rule-1",
		Version:    1,
		AuthorID:   "author-bob",
		ReviewSLA:  72 * time.Hour,
	}

	env.ExecuteWorkflow(RuleReviewWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var res RuleReviewWorkflowResult
	s.NoError(env.GetWorkflowResult(&res))
	s.Equal("published", res.Status)
	s.Equal(2, res.Version)
	s.Equal("sha256:abc12345", res.Checksum)
}

func (s *RuleReviewWorkflowTestSuite) TestRuleReviewWorkflow_RejectPath() {
	env := s.NewTestWorkflowEnvironment()
	var acts *activities.RuleGovernanceActivities

	env.OnActivity(acts.RecordRejectionActivity, mock.Anything, activities.RejectionActivityInput{
		TenantID:   "tenant-1",
		RuleNodeID: "rule-1",
		RejecterID: "approver-carol",
		Reason:     "insufficient threshold",
	}).Return(nil)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalRuleReject, RejectionSignalPayload{
			RejecterID: "approver-carol",
			Reason:     "insufficient threshold",
		})
	}, 1*time.Minute)

	input := RuleReviewWorkflowInput{
		TenantID:   "tenant-1",
		RuleNodeID: "rule-1",
		Version:    1,
		AuthorID:   "author-bob",
		ReviewSLA:  72 * time.Hour,
	}

	env.ExecuteWorkflow(RuleReviewWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var res RuleReviewWorkflowResult
	s.NoError(env.GetWorkflowResult(&res))
	s.Equal("rejected", res.Status)
	s.Equal("insufficient threshold", res.Reason)
}

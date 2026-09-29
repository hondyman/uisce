package workflows

import (
	"testing"
	"time"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type AuditAnchorWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestAuditAnchorWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(AuditAnchorWorkflowTestSuite))
}

func (s *AuditAnchorWorkflowTestSuite) TestViolationAuditAnchorWorkflow_HappyPath() {
	env := s.NewTestWorkflowEnvironment()
	var acts *activities.AuditAnchorActivities

	anchorTime := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	env.OnActivity(acts.RunAuditAnchorActivity, mock.Anything, activities.AuditAnchorActivityInput{
		TenantID: "tenant-anchor-1",
	}).Return(&activities.AuditAnchorActivityResult{
		Anchored: true,
		AnchorInfo: &analytics.AuditAnchorResult{
			AnchorID:   "anchor-uuid-1",
			TenantID:   "tenant-anchor-1",
			SeqFrom:    1,
			SeqTo:      50,
			AnchorHash: "sha256:anchor_fold_root_hash",
			RowCount:   50,
			AnchoredAt: anchorTime,
		},
	}, nil)

	input := AuditAnchorWorkflowInput{
		TenantID: "tenant-anchor-1",
	}

	env.ExecuteWorkflow(ViolationAuditAnchorWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var res AuditAnchorWorkflowResult
	s.NoError(env.GetWorkflowResult(&res))
	s.True(res.Anchored)
	s.Equal(50, res.RowCount)
	s.Equal(int64(1), res.SeqFrom)
	s.Equal(int64(50), res.SeqTo)
	s.Equal("sha256:anchor_fold_root_hash", res.AnchorHash)
}

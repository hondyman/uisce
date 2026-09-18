package temporal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type SWIFTReconciliationWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestSWIFTReconciliationWorkflow(t *testing.T) {
	suite.Run(t, new(SWIFTReconciliationWorkflowTestSuite))
}

func (s *SWIFTReconciliationWorkflowTestSuite) newEnv() *testsuite.TestWorkflowEnvironment {
	env := s.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(SWIFTReconciliationWorkflow)
	env.RegisterActivity(LoadSWIFTExpectedSettlementsActivity)
	env.RegisterActivity(MatchSWIFTSettlementsActivity)
	env.RegisterActivity(PersistSWIFTReconciliationReportActivity)
	env.RegisterActivity(EscalateUnmatchedActivity)
	env.RegisterActivity(ResolveCancelPendingActivity)
	return env
}

func (s *SWIFTReconciliationWorkflowTestSuite) TestReconciliationWithCancelPendingResolver() {
	env := s.newEnv()
	tenantID := "11111111-0000-4000-8000-000000000001"

	env.OnActivity(LoadSWIFTExpectedSettlementsActivity, mock.Anything, mock.Anything).Return(10, nil)
	env.OnActivity(MatchSWIFTSettlementsActivity, mock.Anything, mock.Anything).Return([]SWIFTMismatch{}, nil)
	env.OnActivity(PersistSWIFTReconciliationReportActivity, mock.Anything, mock.Anything).Return(nil)

	// Pin exact tenantID argument and .Once()
	expectedResult := ResolveCancelPendingResult{
		Scanned:      2,
		Resolved:     1,
		AutoFailed:   1,
		StillPending: 0,
	}
	env.OnActivity(ResolveCancelPendingActivity, mock.Anything, tenantID).Return(expectedResult, nil).Once()

	input := SWIFTReconciliationInput{
		TenantID:      tenantID,
		CustodianID:   "22222222-0000-4000-8000-000000000002",
		LookbackStart: time.Now().Add(-24 * time.Hour),
		LookbackEnd:   time.Now(),
	}

	env.ExecuteWorkflow(SWIFTReconciliationWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var report SWIFTReconciliationReport
	s.NoError(env.GetWorkflowResult(&report))
	s.Equal(tenantID, report.TenantID)
	s.Equal(10, report.InstructionsScanned)
	s.Empty(report.Mismatches)
}

package workflows_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/mdm"
	"github.com/hondyman/uisce/backend/internal/temporal/workflows"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

type DownstreamSyncWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
	env *testsuite.TestWorkflowEnvironment
}

func (s *DownstreamSyncWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *DownstreamSyncWorkflowTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func (s *DownstreamSyncWorkflowTestSuite) TestDownstreamGoldSyncWorkflow_Success() {
	tenantID := uuid.New()
	boID := uuid.New()
	req := mdm.DownstreamSyncRequest{
		TenantID:  tenantID,
		BOID:      boID,
		EntitySID: "CA_US0378331005_20260901_DIV",
		GoldAttributes: map[string]interface{}{
			"security_isin": "US0378331005",
			"gross_amount":  0.26,
		},
		MutationType:  "UPSERT",
		KnowledgeTime: time.Now().UTC(),
	}

	targets := []mdm.BindingTargetDescriptor{
		{
			BindingID:       uuid.New(),
			TargetName:      "OPERATIONAL_POSTGRES",
			DeliveryChannel: "SQL_MERGE",
		},
		{
			BindingID:       uuid.New(),
			TargetName:      "ANALYTICS_STARROCKS",
			DeliveryChannel: "SQL_MERGE",
		},
	}

	resolveActivity := func(ctx context.Context, tenantID, boID uuid.UUID) ([]mdm.BindingTargetDescriptor, error) {
		return nil, nil
	}
	dispatchActivity := func(ctx context.Context, tenantID, boID uuid.UUID, target mdm.BindingTargetDescriptor, entitySID string, goldAttributes map[string]interface{}) (map[string]interface{}, error) {
		return nil, nil
	}

	s.env.RegisterActivityWithOptions(resolveActivity, activity.RegisterOptions{Name: "ResolveTargetBindingsActivity"})
	s.env.RegisterActivityWithOptions(dispatchActivity, activity.RegisterOptions{Name: "TransformAndDispatchActivity"})

	s.env.OnActivity("ResolveTargetBindingsActivity", mock.Anything, tenantID, boID).Return(targets, nil)
	s.env.OnActivity("TransformAndDispatchActivity", mock.Anything, tenantID, boID, targets[0], req.EntitySID, req.GoldAttributes).
		Return(map[string]interface{}{"targetName": "OPERATIONAL_POSTGRES", "status": "DELIVERED", "checksum": "abc"}, nil)
	s.env.OnActivity("TransformAndDispatchActivity", mock.Anything, tenantID, boID, targets[1], req.EntitySID, req.GoldAttributes).
		Return(map[string]interface{}{"targetName": "ANALYTICS_STARROCKS", "status": "DELIVERED", "checksum": "def"}, nil)

	s.env.ExecuteWorkflow(workflows.DownstreamGoldSyncWorkflow, req)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func TestDownstreamSyncWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(DownstreamSyncWorkflowTestSuite))
}

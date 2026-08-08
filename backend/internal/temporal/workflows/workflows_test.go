package workflows

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
)

// HourlyRollupWorkflowTestSuite tests the HourlyRollupWorkflow
type HourlyRollupWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

// TestHourlyRollupWorkflow_Success validates successful hourly rollup computation
func (s *HourlyRollupWorkflowTestSuite) TestHourlyRollupWorkflow_Success() {
	env := s.NewTestWorkflowEnvironment()

	// Mock the child workflow executions
	env.OnWorkflow(RegionHourlyRollupWorkflow, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Mock the activity executions
	env.OnActivity(activities.PublishEventActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Execute workflow
	input := HourlyRollupInput{
		RunID:   "test-run-001",
		Regions: []string{"us-east-1", "eu-west-1"},
	}

	env.ExecuteWorkflow(HourlyRollupWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	s.NoError(err)
}

// TestHourlyRollupWorkflow_PartialFailure validates handling of failed regions
func (s *HourlyRollupWorkflowTestSuite) TestHourlyRollupWorkflow_PartialFailure() {
	env := s.NewTestWorkflowEnvironment()

	input := HourlyRollupInput{
		RunID:   "test-run-001",
		Regions: []string{"us-east-1", "eu-west-1"},
	}

	// Mock successful region
	env.OnWorkflow(RegionHourlyRollupWorkflow, mock.Anything, "us-east-1", "test-run-001").Return(nil)

	// Mock failed region
	env.OnWorkflow(RegionHourlyRollupWorkflow, mock.Anything, "eu-west-1", "test-run-001").Return(fmt.Errorf("region rollup failed"))

	env.OnActivity(activities.PublishEventActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(HourlyRollupWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	// Workflow returns error on partial failures in current implementation
	s.Error(err)
}

// TestRegionHourlyRollupWorkflow_Success validates region-specific rollup workflow
func (s *HourlyRollupWorkflowTestSuite) TestRegionHourlyRollupWorkflow_Success() {
	env := s.NewTestWorkflowEnvironment()

	env.OnActivity(activities.PublishEventActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	// RegionHourlyRollupWorkflow unwraps arguments directly
	env.ExecuteWorkflow(RegionHourlyRollupWorkflow, "us-east-1", "test-run-001")

	s.True(env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	s.NoError(err)
}

// DailySLAWorkflowTestSuite tests the DailySLAWorkflow
type DailySLAWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

// TestDailySLAWorkflow_Success validates successful daily SLA computation
func (s *DailySLAWorkflowTestSuite) TestDailySLAWorkflow_Success() {
	env := s.NewTestWorkflowEnvironment()

	env.OnActivity(activities.PublishEventActivity, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	input := DailySLAInput{
		Date:  "2026-02-09",
		RunID: "daily-sla-001",
	}

	env.ExecuteWorkflow(DailySLAWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	s.NoError(err)
}

// TestDailySLAWorkflow_DateValidation validates date parameter validation
func (s *DailySLAWorkflowTestSuite) TestDailySLAWorkflow_DateValidation() {
	env := s.NewTestWorkflowEnvironment()

	input := DailySLAInput{
		Date:  "2026-02-09",
		RunID: "daily-sla-001",
	}

	env.OnActivity(activities.PublishEventActivity, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	env.ExecuteWorkflow(DailySLAWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	s.NoError(err)
}

// MLTrainingWorkflowTestSuite tests the MLTrainingWorkflow
type MLTrainingWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

// TestMLTrainingWorkflow_FullPipeline validates complete ML training pipeline
func (s *MLTrainingWorkflowTestSuite) TestMLTrainingWorkflow_FullPipeline() {
	env := s.NewTestWorkflowEnvironment()

	// Mock Python script executions
	env.OnActivity(activities.RunPythonScriptActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(`{"status":"completed"}`, nil)

	env.OnActivity(activities.PublishEventActivity, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	input := MLTrainingInput{
		TrainingDate: "2026-02-09",
		ModelName:    "chain_failure_predictor",
		RunID:        "ml-train-001",
	}

	env.ExecuteWorkflow(MLTrainingWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	s.NoError(err)
}

// TestMLTrainingWorkflow_FeatureExtractionFailure validates error propagation
func (s *MLTrainingWorkflowTestSuite) TestMLTrainingWorkflow_FeatureExtractionFailure() {
	env := s.NewTestWorkflowEnvironment()

	// Mock feature extraction to fail
	env.OnActivity(activities.RunPythonScriptActivity, mock.Anything, "extract_features.py", mock.Anything).
		Return("", fmt.Errorf("feature extraction failed"))

	input := MLTrainingInput{
		TrainingDate: "2026-02-09",
		ModelName:    "chain_failure_predictor",
		RunID:        "ml-train-001",
	}

	env.ExecuteWorkflow(MLTrainingWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	s.Error(err)
}

// RunTests executes all workflow tests
func TestHourlyRollupWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(HourlyRollupWorkflowTestSuite))
}

func TestDailySLAWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(DailySLAWorkflowTestSuite))
}

func TestMLTrainingWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(MLTrainingWorkflowTestSuite))
}

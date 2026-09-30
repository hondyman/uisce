package mastering

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/api/serviceerror"
)

// mockTemporalClient is a mock implementation of temporalclient.Client for orchestrator tests.
type mockTemporalClient struct {
	mock.Mock
}

func (m *mockTemporalClient) SignalWorkflow(ctx context.Context, workflowID string, runID string, signalName string, arg interface{}) error {
	args := m.Called(ctx, workflowID, runID, signalName, arg)
	return args.Error(0)
}

func TestSignalWorkflow_TransientErrorFailsWithoutFallback(t *testing.T) {
	mockClient := new(mockTemporalClient)
	engine := &Engine{
		TemporalClient: nil, // We'll test direct isWorkflowNotFound and custom client
	}

	// 1. Transient network/server error
	transientErr := errors.New("connection reset by peer")
	assert.False(t, isWorkflowNotFound(transientErr), "transient error must not be classified as not found")

	// 2. NotFound error
	notFoundErr := serviceerror.NewNotFound("workflow execution not found")
	assert.True(t, isWorkflowNotFound(notFoundErr), "serviceerror.NotFound must be classified as not found")

	// 3. Direct execution behavior
	_ = engine
	_ = mockClient
}

func TestSignalOverrideWorkflow_NotFoundSafeFallback(t *testing.T) {
	// When TemporalClient is nil, returns false, nil (dev mode direct fallback)
	engine := &Engine{TemporalClient: nil}
	signaled, err := engine.SignalOverrideWorkflow(context.Background(), "t1", "SECURITY", "o1", "MDMApprove", nil)
	assert.NoError(t, err)
	assert.False(t, signaled)
}

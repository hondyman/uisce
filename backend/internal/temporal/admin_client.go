package temporal

import (
	"context"
	"fmt"

	enums "go.temporal.io/api/enums/v1"
	taskqueue "go.temporal.io/api/taskqueue/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
)

// AdminClient wraps the generated WorkflowServiceClient for admin RPCs
type AdminClient struct {
	svc workflowservice.WorkflowServiceClient
	ns  string
}

// NewAdminClientFromTarget creates an AdminClient by dialing the given target with provided grpc options
func NewAdminClientFromTarget(ctx context.Context, target, namespace string, opts ...grpc.DialOption) (*AdminClient, error) {
	conn, err := grpc.DialContext(ctx, target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial temporal server: %w", err)
	}
	svc := workflowservice.NewWorkflowServiceClient(conn)
	return &AdminClient{svc: svc, ns: namespace}, nil
}

// DescribeTaskQueue calls the DescribeTaskQueue RPC
func (ac *AdminClient) DescribeTaskQueue(ctx context.Context, tq string, activity bool) (*workflowservice.DescribeTaskQueueResponse, error) {
	tqt := enums.TASK_QUEUE_TYPE_WORKFLOW
	if activity {
		tqt = enums.TASK_QUEUE_TYPE_ACTIVITY
	}
	req := &workflowservice.DescribeTaskQueueRequest{
		Namespace:     ac.ns,
		TaskQueue:     &taskqueue.TaskQueue{Name: tq},
		TaskQueueType: tqt,
		ReportPollers: true,
		ReportStats:   true,
	}
	return ac.svc.DescribeTaskQueue(ctx, req)
}

package temporal

import (
	"context"
	"net"
	"testing"
	"time"

	workflowservice "go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockWorkflowService implements the DescribeTaskQueue RPC handler only
type mockWorkflowService struct {
	workflowservice.UnimplementedWorkflowServiceServer
}

func (m *mockWorkflowService) DescribeTaskQueue(ctx context.Context, req *workflowservice.DescribeTaskQueueRequest) (*workflowservice.DescribeTaskQueueResponse, error) {
	if req == nil || req.TaskQueue == nil {
		return nil, status.Error(codes.InvalidArgument, "missing task queue")
	}
	// Return an empty but non-nil response (fields not required for this test)
	return &workflowservice.DescribeTaskQueueResponse{}, nil
}

func TestDescribeTaskQueue_MockServer(t *testing.T) {
	// Start a gRPC server with our mock service
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	srv := grpc.NewServer()
	workflowservice.RegisterWorkflowServiceServer(srv, &mockWorkflowService{})

	go srv.Serve(lis)
	defer srv.Stop()

	// Create AdminClient pointing to the mock server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	adminClient, err := NewAdminClientFromTarget(ctx, lis.Addr().String(), "default", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to create admin client: %v", err)
	}

	// Call DescribeTaskQueue via admin client and verify response
	resp, err := adminClient.DescribeTaskQueue(ctx, "my-queue", false)
	if err != nil {
		t.Fatalf("DescribeTaskQueue failed: %v", err)
	}
	if resp == nil {
		t.Fatalf("expected non-nil response from DescribeTaskQueue")
	}

	// Now verify WorkflowAdminService.DescribeTaskQueue proxies to AdminClient when configured
	was := &WorkflowAdminService{admin: adminClient}
	v, err := was.DescribeTaskQueue(ctx, "my-queue")
	if err != nil {
		t.Fatalf("WorkflowAdminService.DescribeTaskQueue failed: %v", err)
	}
	if _, ok := v.(*workflowservice.DescribeTaskQueueResponse); !ok {
		t.Fatalf("unexpected type from DescribeTaskQueue: %T", v)
	}
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	namespacepb "go.temporal.io/api/namespace/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
)

// mockWorkflowService implements DescribeTaskQueue only
type mockWorkflowService struct {
	workflowservice.UnimplementedWorkflowServiceServer
}

func (m *mockWorkflowService) DescribeTaskQueue(ctx context.Context, req *workflowservice.DescribeTaskQueueRequest) (*workflowservice.DescribeTaskQueueResponse, error) {
	if req == nil || req.TaskQueue == nil {
		return nil, fmt.Errorf("missing task queue")
	}
	return &workflowservice.DescribeTaskQueueResponse{}, nil
}

// RegisterNamespace returns success for register requests
func (m *mockWorkflowService) RegisterNamespace(ctx context.Context, req *workflowservice.RegisterNamespaceRequest) (*workflowservice.RegisterNamespaceResponse, error) {
	if req == nil || req.Namespace == "" {
		return nil, fmt.Errorf("missing namespace")
	}
	log.Printf("RegisterNamespace called: %s", req.Namespace)
	return &workflowservice.RegisterNamespaceResponse{}, nil
}

// DescribeNamespace returns minimal namespace info
func (m *mockWorkflowService) DescribeNamespace(ctx context.Context, req *workflowservice.DescribeNamespaceRequest) (*workflowservice.DescribeNamespaceResponse, error) {
	if req == nil || req.Namespace == "" {
		return nil, fmt.Errorf("missing namespace")
	}
	ni := &namespacepb.NamespaceInfo{
		Id:   "mock-id",
		Name: req.Namespace,
	}
	cfg := &namespacepb.NamespaceConfig{
		WorkflowExecutionRetentionTtl: durationpb.New(24 * time.Hour),
	}
	return &workflowservice.DescribeNamespaceResponse{NamespaceInfo: ni, Config: cfg}, nil
}

// ListNamespaces returns a single namespace matching if provided
func (m *mockWorkflowService) ListNamespaces(ctx context.Context, req *workflowservice.ListNamespacesRequest) (*workflowservice.ListNamespacesResponse, error) {
	// return an empty list (tests should call DescribeNamespace/RegisterNamespace directly)
	return &workflowservice.ListNamespacesResponse{}, nil
}

// UpdateNamespace returns success for update requests
func (m *mockWorkflowService) UpdateNamespace(ctx context.Context, req *workflowservice.UpdateNamespaceRequest) (*workflowservice.UpdateNamespaceResponse, error) {
	if req == nil || req.Namespace == "" {
		return nil, fmt.Errorf("missing namespace")
	}
	log.Printf("UpdateNamespace called: %s", req.Namespace)
	return &workflowservice.UpdateNamespaceResponse{}, nil
}

func main() {
	port := flag.Int("port", 7234, "port to listen on")
	flag.Parse()
	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	workflowservice.RegisterWorkflowServiceServer(s, &mockWorkflowService{})
	log.Printf("mock admin server listening on %s", addr)
	if err := s.Serve(l); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

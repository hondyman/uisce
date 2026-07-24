package admin

import (
	"context"
	"fmt"
	"time"

	enums "go.temporal.io/api/enums/v1"
	namespacepb "go.temporal.io/api/namespace/v1"
	taskqueue "go.temporal.io/api/taskqueue/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Client is a lightweight admin client for Temporal admin gRPC calls.
type Client struct {
	svc  workflowservice.WorkflowServiceClient
	conn *grpc.ClientConn
}

// AdminClient defines the subset of admin operations used by the CLI and tests.
type AdminClient interface {
	Close() error
	DescribeTaskQueue(ctx context.Context, namespace, queue string, activity bool) (*workflowservice.DescribeTaskQueueResponse, error)
	RegisterNamespace(ctx context.Context, namespace string, retentionSeconds int64) (*workflowservice.RegisterNamespaceResponse, error)
	DescribeNamespace(ctx context.Context, namespace string) (*workflowservice.DescribeNamespaceResponse, error)
	UpdateNamespace(ctx context.Context, namespace string, retentionSeconds int64) (*workflowservice.UpdateNamespaceResponse, error)
	ListNamespaces(ctx context.Context) (*workflowservice.ListNamespacesResponse, error)
}

// Ensure Client implements AdminClient
var _ AdminClient = (*Client)(nil)

// NewClient dials target and returns a Client. Caller should call Close().
func NewClient(ctx context.Context, target string, opts ...grpc.DialOption) (*Client, error) {
	if len(opts) == 0 {
		// default to plaintext transport; callers can override with TLS options.
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx2, target, opts...)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", target, err)
	}
	svc := workflowservice.NewWorkflowServiceClient(conn)
	return &Client{svc: svc, conn: conn}, nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// DescribeTaskQueue proxies to the admin DescribeTaskQueue RPC.
func (c *Client) DescribeTaskQueue(ctx context.Context, namespace, queue string, activity bool) (*workflowservice.DescribeTaskQueueResponse, error) {
	tqt := enums.TASK_QUEUE_TYPE_WORKFLOW
	if activity {
		tqt = enums.TASK_QUEUE_TYPE_ACTIVITY
	}
	req := &workflowservice.DescribeTaskQueueRequest{
		Namespace:     namespace,
		TaskQueue:     &taskqueue.TaskQueue{Name: queue},
		TaskQueueType: tqt,
		ReportPollers: true,
		ReportStats:   true,
	}
	return c.svc.DescribeTaskQueue(ctx, req)
}

// RegisterNamespace registers a new namespace with a given retention duration (in seconds).
func (c *Client) RegisterNamespace(ctx context.Context, namespace string, retentionSeconds int64) (*workflowservice.RegisterNamespaceResponse, error) {
	if retentionSeconds <= 0 {
		retentionSeconds = int64(24 * time.Hour / time.Second)
	}
	req := &workflowservice.RegisterNamespaceRequest{
		Namespace:                        namespace,
		WorkflowExecutionRetentionPeriod: durationpb.New(time.Duration(retentionSeconds) * time.Second),
	}
	return c.svc.RegisterNamespace(ctx, req)
}

// DescribeNamespace returns namespace info and config.
func (c *Client) DescribeNamespace(ctx context.Context, namespace string) (*workflowservice.DescribeNamespaceResponse, error) {
	req := &workflowservice.DescribeNamespaceRequest{Namespace: namespace}
	return c.svc.DescribeNamespace(ctx, req)
}

// UpdateNamespace updates the namespace retention configuration.
func (c *Client) UpdateNamespace(ctx context.Context, namespace string, retentionSeconds int64) (*workflowservice.UpdateNamespaceResponse, error) {
	if retentionSeconds <= 0 {
		retentionSeconds = int64(24 * time.Hour / time.Second)
	}
	req := &workflowservice.UpdateNamespaceRequest{
		Namespace: namespace,
		Config: &namespacepb.NamespaceConfig{
			WorkflowExecutionRetentionTtl: durationpb.New(time.Duration(retentionSeconds) * time.Second),
		},
	}
	return c.svc.UpdateNamespace(ctx, req)
}

// ListNamespaces lists namespaces (server may paginate; this returns a single response page)
func (c *Client) ListNamespaces(ctx context.Context) (*workflowservice.ListNamespacesResponse, error) {
	req := &workflowservice.ListNamespacesRequest{}
	return c.svc.ListNamespaces(ctx, req)
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	workflowservice "go.temporal.io/api/workflowservice/v1"
)

type fakeClient struct{}

func (f *fakeClient) Close() error { return nil }
func (f *fakeClient) DescribeTaskQueue(ctx context.Context, namespace, queue string, activity bool) (*workflowservice.DescribeTaskQueueResponse, error) {
	return &workflowservice.DescribeTaskQueueResponse{}, nil
}
func (f *fakeClient) RegisterNamespace(ctx context.Context, namespace string, retentionSeconds int64) (*workflowservice.RegisterNamespaceResponse, error) {
	return &workflowservice.RegisterNamespaceResponse{}, nil
}
func (f *fakeClient) DescribeNamespace(ctx context.Context, namespace string) (*workflowservice.DescribeNamespaceResponse, error) {
	return &workflowservice.DescribeNamespaceResponse{}, nil
}
func (f *fakeClient) UpdateNamespace(ctx context.Context, namespace string, retentionSeconds int64) (*workflowservice.UpdateNamespaceResponse, error) {
	return &workflowservice.UpdateNamespaceResponse{}, nil
}
func (f *fakeClient) ListNamespaces(ctx context.Context) (*workflowservice.ListNamespacesResponse, error) {
	return &workflowservice.ListNamespacesResponse{}, nil
}

func TestListJSONOutput(t *testing.T) {
	// capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	// enable json flag
	formatJSON = new(bool)
	*formatJSON = true
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := listNamespaces(ctx, &fakeClient{}); err != nil {
		t.Fatalf("listNamespaces error: %v", err)
	}
	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old
	// verify it's valid json
	var js interface{}
	if err := json.Unmarshal(buf.Bytes(), &js); err != nil {
		t.Fatalf("output not json: %v, out=%s", err, buf.String())
	}
}

func TestCreateUpdateDescribeText(t *testing.T) {
	// ensure text output
	formatJSON = new(bool)
	*formatJSON = false
	ctx := context.Background()
	if err := createNamespace(ctx, &fakeClient{}, "ns1", 3600); err != nil {
		t.Fatalf("createNamespace failed: %v", err)
	}
	if err := updateNamespace(ctx, &fakeClient{}, "ns1", 7200); err != nil {
		t.Fatalf("updateNamespace failed: %v", err)
	}
	if err := describeQueue(ctx, &fakeClient{}, "default", "q1", false); err != nil {
		t.Fatalf("describeQueue failed: %v", err)
	}
}

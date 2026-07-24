package admin

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// This integration test starts the repo's mock admin server binary and calls DescribeTaskQueue.
func TestDescribeTaskQueueAgainstMockServer(t *testing.T) {
	// compute backend module dir relative to current working directory
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	// walk up until we find the backend module root (presence of cmd/mock_admin_server)
	backendDir := ""
	cur := wd
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(cur, "cmd", "mock_admin_server")
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			backendDir = cur
			break
		}
		cur = filepath.Dir(cur)
	}
	if backendDir == "" {
		t.Fatalf("could not locate backend module root from wd=%s", wd)
	}

	// Build the mock server into a temp dir (avoid writing into the repo).
	mockBin := filepath.Join(t.TempDir(), "mock_admin_server")
	cmd := exec.Command("go", "build", "-o", mockBin, "./cmd/mock_admin_server")
	cmd.Dir = backendDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build mock admin server: %v\noutput:\n%s", err, string(out))
	}
	// ask OS for free port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	// start the mock server using absolute path
	proc := exec.Command(mockBin, "-port", fmt.Sprintf("%d", port))
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	if err := proc.Start(); err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer proc.Process.Kill()

	// wait for server to accept connections
	deadline := time.Now().Add(2 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("mock server did not start listening: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	c, err := NewClient(ctx, addr)
	if err != nil {
		t.Fatalf("failed to dial mock admin: %v", err)
	}
	defer c.Close()

	if _, err := c.DescribeTaskQueue(ctx, "default", "test-queue", false); err != nil {
		t.Fatalf("DescribeTaskQueue failed: %v", err)
	}
	// exercise namespace register and update
	if _, err := c.RegisterNamespace(ctx, "itest-ns", 3600); err != nil {
		t.Fatalf("RegisterNamespace failed: %v", err)
	}
	if _, err := c.UpdateNamespace(ctx, "itest-ns", 7200); err != nil {
		t.Fatalf("UpdateNamespace failed: %v", err)
	}
}

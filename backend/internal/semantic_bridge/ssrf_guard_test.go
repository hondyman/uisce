package semantic_bridge

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIsBlockedTarget(t *testing.T) {
	cases := []struct {
		ip      string
		blocked bool
	}{
		{"127.0.0.1", true},
		{"169.254.169.254", true}, // cloud metadata endpoint
		{"10.0.0.5", true},
		{"172.16.0.5", true},
		{"192.168.1.5", true},
		{"::1", true},
		{"0.0.0.0", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
	}
	for _, c := range cases {
		ip := net.ParseIP(c.ip)
		if ip == nil {
			t.Fatalf("failed to parse test IP %q", c.ip)
		}
		if got := isBlockedTarget(ip); got != c.blocked {
			t.Errorf("isBlockedTarget(%s) = %v, want %v", c.ip, got, c.blocked)
		}
	}
}

func TestDialGuard_BlocksLoopbackEvenAfterDNSResolution(t *testing.T) {
	// A real local server bound to loopback — proves the guard blocks by
	// resolved IP, not just by matching a hostname string like "localhost".
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	dialer := &net.Dialer{Timeout: 2 * time.Second}
	guarded := dialGuard(dialer.DialContext)

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: guarded,
		},
	}

	_, err := client.Get(srv.URL)
	if err == nil {
		t.Fatalf("expected the guard to block a request to a loopback address, but it succeeded")
	}
	if !strings.Contains(err.Error(), "ssrf guard") {
		t.Fatalf("expected an ssrf guard error, got: %v", err)
	}
}

func TestDialGuard_AllowsPublicAddress(t *testing.T) {
	// Uses an IP literal (not a hostname) so this doesn't depend on DNS
	// being reachable in the test environment — net.DefaultResolver
	// short-circuits IP literals without a network round-trip. Proves the
	// guard passes public addresses through to the real dialer rather than
	// blocking everything.
	fakeDial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return nil, errFakeDialReached
	}
	guarded := dialGuard(fakeDial)

	_, err := guarded(context.Background(), "tcp", "1.1.1.1:443")
	if err != errFakeDialReached {
		t.Fatalf("expected the guard to allow a public address through to the real dialer, got: %v", err)
	}
}

var errFakeDialReached = &fakeDialError{"reached real dial"}

type fakeDialError struct{ msg string }

func (e *fakeDialError) Error() string { return e.msg }

func TestValidateHostable(t *testing.T) {
	valid := []string{"xy12345.us-east-1", "dbc-abc123.cloud.databricks.com"}
	for _, h := range valid {
		if err := validateHostable(h); err != nil {
			t.Errorf("validateHostable(%q) unexpectedly failed: %v", h, err)
		}
	}

	invalid := []string{"", "https://evil.example.com", "host/with/path", "host with space"}
	for _, h := range invalid {
		if err := validateHostable(h); err == nil {
			t.Errorf("validateHostable(%q) should have failed", h)
		}
	}
}

package manager

import (
	"strings"
	"testing"
)

func TestScrubSecrets(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{
			input:    "Received request with Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.doNotLeakThisSignature",
			expected: "Received request with Authorization: Bearer [REDACTED_JWT]",
		},
		{
			input:    "Window spawned with url /view/rebalancer?init_token=d7a5e8f901bc4321&other=val",
			expected: "Window spawned with url /view/rebalancer?init_token=[REDACTED]&other=val",
		},
		{
			input:    "Connecting to backend with password=SuperSecretPassword123 and token=abc12345",
			expected: "Connecting to backend with password=[REDACTED] and token=[REDACTED]",
		},
		{
			input:    "Normal operational log: [DeskWindowManager] Window closed and deregistered: win_orders",
			expected: "Normal operational log: [DeskWindowManager] Window closed and deregistered: win_orders",
		},
	}

	for i, c := range cases {
		out := ScrubSecrets(c.input)
		if !strings.Contains(out, "[REDACTED") && strings.Contains(c.input, "token=") {
			t.Errorf("Case %d failed to redact token: got %s", i, out)
		}
		if strings.Contains(out, "SuperSecretPassword123") {
			t.Errorf("Case %d leaked password: got %s", i, out)
		}
		if strings.Contains(out, "doNotLeakThisSignature") {
			t.Errorf("Case %d leaked JWT: got %s", i, out)
		}
		if out != c.expected {
			t.Errorf("Case %d mismatch:\nGot:  %s\nWant: %s", i, out, c.expected)
		}
	}
}

func TestScrubbingWriter(t *testing.T) {
	var buf strings.Builder
	sw := &ScrubbingWriter{target: &buf}

	testMsg := "Connecting with Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.superSecretSig and init_token=secToken999\n"
	n, err := sw.Write([]byte(testMsg))
	if err != nil {
		t.Fatalf("ScrubbingWriter error: %v", err)
	}
	if n != len(testMsg) {
		t.Errorf("Expected n=%d, got %d", len(testMsg), n)
	}

	out := buf.String()
	if strings.Contains(out, "superSecretSig") || strings.Contains(out, "secToken999") {
		t.Fatalf("ScrubbingWriter leaked credentials to target: %s", out)
	}
	if !strings.Contains(out, "Bearer [REDACTED_JWT]") || !strings.Contains(out, "init_token=[REDACTED]") {
		t.Errorf("ScrubbingWriter output unexpected: %s", out)
	}
}

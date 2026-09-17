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

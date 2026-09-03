package semantic_bridge

import (
	"testing"
	"time"
)

func TestCredentialRotationDue(t *testing.T) {
	fresh := time.Now().Add(-1 * time.Hour)
	stale := time.Now().Add(-100 * 24 * time.Hour)

	cases := []struct {
		name     string
		rotated  *time.Time
		expected bool
	}{
		{"never configured", nil, false},
		{"rotated recently", &fresh, false},
		{"rotated 100 days ago", &stale, true},
	}

	for _, c := range cases {
		target := &BridgeTarget{CredentialsRotatedAt: c.rotated}
		if got := target.CredentialRotationDue(); got != c.expected {
			t.Errorf("%s: CredentialRotationDue() = %v, want %v", c.name, got, c.expected)
		}
	}
}

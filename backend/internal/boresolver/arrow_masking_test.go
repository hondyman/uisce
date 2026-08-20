package boresolver

import (
	"testing"
)

func TestDetermineMaskingTier(t *testing.T) {
	tests := []struct {
		tag       string
		role      string
		clearance string
		want      MaskingTier
	}{
		{"contains:pii", "analyst", "PUBLIC", MaskingTierRedactFull},
		{"financial:confidential", "auditor", "STANDARD", MaskingTierHashSHA256},
		{"quant:market_data", "researcher", "STANDARD", MaskingTierNoiseAddition},
		{"contains:pii", "platform_trader", "CONFIDENTIAL", MaskingTierPassthrough},
		{"standard:cleared", "analyst", "PUBLIC", MaskingTierPassthrough},
	}

	for _, tt := range tests {
		got := DetermineMaskingTier(tt.tag, tt.role, tt.clearance)
		if got != tt.want {
			t.Errorf("DetermineMaskingTier(%q, %q, %q) = %v; want %v", tt.tag, tt.role, tt.clearance, got, tt.want)
		}
	}
}

func TestApplyVectorMasking(t *testing.T) {
	redacted := ApplyVectorMasking("john.doe@example.com", MaskingTierRedactFull)
	if redacted != "***REDACTED***" {
		t.Errorf("expected ***REDACTED***, got %v", redacted)
	}

	hashed := ApplyVectorMasking("1234-5678-9012", MaskingTierHashSHA256)
	if len(hashed.(string)) != 64 {
		t.Errorf("expected 64-char sha256 hex string, got %v", hashed)
	}

	passed := ApplyVectorMasking(42.5, MaskingTierPassthrough)
	if passed != 42.5 {
		t.Errorf("expected 42.5, got %v", passed)
	}
}

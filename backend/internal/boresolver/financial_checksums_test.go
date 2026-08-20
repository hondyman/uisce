package boresolver

import (
	"testing"
)

func TestValidateISIN(t *testing.T) {
	tests := []struct {
		isin  string
		valid bool
	}{
		{"US0378331005", true},  // Apple Inc.
		{"US5949181045", true},  // Microsoft Corp.
		{"GB0002634946", true},  // BAE Systems
		{"US0378331006", false}, // Bad check digit
		{"US037833100", false},  // Too short
		{"120378331005", false}, // Invalid country code
	}

	for _, tt := range tests {
		got := ValidateISIN(tt.isin)
		if got != tt.valid {
			t.Errorf("ValidateISIN(%q) = %v; want %v", tt.isin, got, tt.valid)
		}
	}
}

func TestValidateCUSIP(t *testing.T) {
	tests := []struct {
		cusip string
		valid bool
	}{
		{"037833100", true},  // Apple Inc.
		{"594918104", true},  // Microsoft Corp.
		{"037833101", false}, // Bad check digit
		{"03783310", false},  // Too short
	}

	for _, tt := range tests {
		got := ValidateCUSIP(tt.cusip)
		if got != tt.valid {
			t.Errorf("ValidateCUSIP(%q) = %v; want %v", tt.cusip, got, tt.valid)
		}
	}
}

func TestValidateSEDOL(t *testing.T) {
	tests := []struct {
		sedol string
		valid bool
	}{
		{"0263494", true},  // BAE Systems
		{"2936921", true},  // GlaxoSmithKline
		{"0263495", false}, // Bad check digit
		{"026349", false},  // Too short
	}

	for _, tt := range tests {
		got := ValidateSEDOL(tt.sedol)
		if got != tt.valid {
			t.Errorf("ValidateSEDOL(%q) = %v; want %v", tt.sedol, got, tt.valid)
		}
	}
}

func TestValidateLEI(t *testing.T) {
	tests := []struct {
		lei   string
		valid bool
	}{
		{"5493006MHB84DD0ZWV18", true},  // Barclays Bank PLC
		{"5493006MHB84DD0ZWV19", false}, // Bad check digits
		{"5493006MHB84DD0ZWV", false},   // Too short
	}

	for _, tt := range tests {
		got := ValidateLEI(tt.lei)
		if got != tt.valid {
			t.Errorf("ValidateLEI(%q) = %v; want %v", tt.lei, got, tt.valid)
		}
	}
}

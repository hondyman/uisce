package ip

import (
	"testing"
)

func TestIsIpAllowed(t *testing.T) {
	whitelist := []string{
		"192.168.1.1",
		"10.0.0.0/8",
		"172.16.*.*",
	}

	tests := []struct {
		ip      string
		allowed bool
	}{
		{"192.168.1.1", true},
		{"192.168.1.2", false},
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"11.0.0.1", false},
		{"172.16.0.1", true},
		{"172.16.255.255", true},
		{"172.17.0.1", false},
		{"invalid", false},
	}

	for _, tc := range tests {
		if got := IsIpAllowed(whitelist, tc.ip); got != tc.allowed {
			t.Errorf("IsIpAllowed(%q) = %v; want %v", tc.ip, got, tc.allowed)
		}
	}
}

func TestIsValidIPv4WildcardOrCIDR(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"192.168.1.1", true},
		{"192.168.*.*", true},
		{"10.0.0.0/8", true},
		{"256.0.0.0", false},
		{"1.2.3", false},
		{"1.2.3.4.5", false},
		{"10.0.0.0/33", false},
		{"10.0.0.0/-1", false},
		{"not-an-ip", false},
	}

	for _, tc := range tests {
		if got := IsValidIPv4WildcardOrCIDR(tc.input); got != tc.valid {
			t.Errorf("IsValidIPv4WildcardOrCIDR(%q) = %v; want %v", tc.input, got, tc.valid)
		}
	}
}

func TestPatternsOverlapOrCIDR(t *testing.T) {
	tests := []struct {
		a, b    string
		overlap bool
	}{
		{"192.168.1.1", "192.168.1.1", true},
		{"192.168.1.1", "192.168.1.2", false},
		{"192.168.*.*", "192.168.1.1", true},
		{"10.0.0.0/8", "10.1.1.1", true},
		{"10.0.0.0/8", "11.1.1.1", false},
		{"192.168.0.0/16", "192.168.1.0/24", true},
		{"192.168.0.0/16", "10.0.0.0/8", false},
		{"192.168.*.*", "192.168.0.0/16", true}, // Wildcard and CIDR overlap
	}

	for _, tc := range tests {
		if got := PatternsOverlapOrCIDR(tc.a, tc.b); got != tc.overlap {
			t.Errorf("PatternsOverlapOrCIDR(%q, %q) = %v; want %v", tc.a, tc.b, got, tc.overlap)
		}
	}
}

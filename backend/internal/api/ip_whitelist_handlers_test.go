package api

import (
	"testing"

	"github.com/hondyman/semlayer/backend/internal/utils/ip"
)

func TestIsValidIpOrWildcard(t *testing.T) {
	good := []string{
		"192.168.1.1",
		"0.0.0.0",
		"255.255.255.255",
		"192.168.*.*",
		"*.*.*.*",
		"10.*.1.*",
	}
	for _, s := range good {
		if !ip.IsValidIPv4WildcardOrCIDR(s) {
			t.Fatalf("expected valid: %s", s)
		}
	}

	bad := []string{
		"",
		"192.168.1",
		"192.168.1.256",
		"300.0.0.1",
		"192.168.*.x",
		"192.168.1.*.1",
	}
	for _, s := range bad {
		if ip.IsValidIPv4WildcardOrCIDR(s) {
			t.Fatalf("expected invalid: %s", s)
		}
	}
}

func TestPatternsOverlap(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"192.168.*.*", "192.168.1.1", true},
		{"192.168.1.1", "192.168.1.2", false},
		{"*.168.1.*", "192.168.1.5", true},
		{"10.*.*.*", "11.*.*.*", false},
		{"*.*.*.*", "1.2.3.4", true},
		{"192.168.1.*", "192.168.1.*", true},
	}

	for _, c := range cases {
		got := ip.PatternsOverlapOrCIDR(c.a, c.b)
		if got != c.want {
			t.Fatalf("PatternsOverlapOrCIDR(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

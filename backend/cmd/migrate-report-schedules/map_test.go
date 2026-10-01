package main

import (
	"testing"

	"github.com/hondyman/uisce/backend/internal/schedule"
)

func TestNormalizeCalendar(t *testing.T) {
	if got := normalizeCalendar(""); got != "" {
		t.Fatalf("empty → %q", got)
	}
	if got := normalizeCalendar("NYSE"); got != "XNYS" {
		t.Fatalf("NYSE → %q", got)
	}
	if got := normalizeCalendar("LSE"); got != "XLON" {
		t.Fatalf("LSE → %q", got)
	}
	if got := normalizeCalendar("TARGET2"); got != "TARGET2" {
		t.Fatalf("TARGET2 → %q", got)
	}
}

func TestMapBehavior(t *testing.T) {
	rule, bd := mapBehavior("SKIP", 0, true)
	if rule != schedule.RuleSkip || bd != 0 {
		t.Fatalf("SKIP+cal → %s %d", rule, bd)
	}
	rule, _ = mapBehavior("SKIP", 0, false)
	if rule != schedule.RuleNone {
		t.Fatalf("SKIP without cal → %s", rule)
	}
	rule, _ = mapBehavior("RUN_NEXT_BUS_DAY", 0, true)
	if rule != schedule.RuleNextBusinessDay {
		t.Fatalf("NEXT → %s", rule)
	}
	rule, bd = mapBehavior("BUSINESS_DAY_OF_MONTH", -1, true)
	if rule != schedule.RuleBusinessDayOfMonth || bd != -1 {
		t.Fatalf("BD month → %s %d", rule, bd)
	}
}

func TestZoneForRegion(t *testing.T) {
	if zoneForRegion("us-west") != "America/Los_Angeles" {
		t.Fatal(zoneForRegion("us-west"))
	}
	if zoneForRegion("us-east-1") != "America/New_York" {
		t.Fatal(zoneForRegion("us-east-1"))
	}
}

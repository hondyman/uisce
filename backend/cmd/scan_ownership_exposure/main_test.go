package main

import "testing"

// These cover the pure helpers only - reportLabel, columnLabel, and
// unresolvedPath's contract with JoinPath.RootOwnership. assembleScannedColumns
// itself needs a live database connection (DATABASE_URL) and isn't
// exercised here; this is what's actually testable without one.

func TestReportLabel(t *testing.T) {
	if got := reportLabel("abc-123", "Q4 Revenue"); got != "Q4 Revenue (abc-123)" {
		t.Errorf("got %q", got)
	}
	if got := reportLabel("abc-123", ""); got != "abc-123" {
		t.Errorf("expected bare id when name is empty, got %q", got)
	}
}

func TestColumnLabel(t *testing.T) {
	if got := columnLabel("Revenue Total", "term-1"); got != "Revenue Total" {
		t.Errorf("expected alias to win, got %q", got)
	}
	if got := columnLabel("", "term-1"); got != "term-1" {
		t.Errorf("expected term node id fallback, got %q", got)
	}
}

// TestUnresolvedPath_ClassifiesAsUnresolved pins the one property this
// helper actually needs: feeding it into RootOwnership() must yield
// "unresolved", never "unique" - that's what makes a column whose join
// path couldn't be resolved show up as a finding instead of silently
// passing the scan.
func TestUnresolvedPath_ClassifiesAsUnresolved(t *testing.T) {
	if got := unresolvedPath().RootOwnership(); got != "unresolved" {
		t.Fatalf("unresolvedPath() must classify as \"unresolved\", got %q - an unresolvable saved-query column would silently pass the scan instead of being flagged", got)
	}
}

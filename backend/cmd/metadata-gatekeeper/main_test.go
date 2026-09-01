package main

import (
	"strings"
	"testing"
)

func TestRunAllGatekeeperChecks(t *testing.T) {
	report := RunAllGatekeeperChecks()

	if report.TotalChecks == 0 {
		t.Fatalf("expected TotalChecks > 0")
	}
	if report.FailedChecks > 0 {
		t.Errorf("expected 0 failed checks, got %d", report.FailedChecks)
	}
	if report.PassedChecks != report.TotalChecks {
		t.Errorf("expected %d passed checks, got %d", report.TotalChecks, report.PassedChecks)
	}

	md := GenerateMarkdownReport(report)
	if !strings.Contains(md, "Uisce GitOps CI/CD Metadata Gatekeeper Report") {
		t.Errorf("expected report header in markdown output")
	}
	if !strings.Contains(md, "PASSED (G-SIFI Compliant)") {
		t.Errorf("expected PASSED status in markdown output")
	}
}

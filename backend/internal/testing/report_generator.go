package testing

import (
	"encoding/json"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// TestReport represents the complete regression test report
type TestReport struct {
	ReportID       string        `json:"report_id" yaml:"report_id"`
	TenantID       string        `json:"tenant_id" yaml:"tenant_id"`
	CoreVersion    string        `json:"core_version" yaml:"core_version"`
	OverlayVersion string        `json:"overlay_version" yaml:"overlay_version"`
	Timestamp      time.Time     `json:"timestamp" yaml:"timestamp"`
	Tests          []TestEntry   `json:"tests" yaml:"tests"`
	Summary        TestSummary   `json:"summary" yaml:"summary"`
	UARLogID       string        `json:"uar_log_id" yaml:"uar_log_id"`
	SnapshotID     string        `json:"snapshot_id" yaml:"snapshot_id"`
}

// TestEntry represents a single test result
type TestEntry struct {
	Name       string `json:"name" yaml:"name"`
	EntityType string `json:"entity_type" yaml:"entity_type"`
	EntityID   string `json:"entity_id" yaml:"entity_id"`
	Status     string `json:"status" yaml:"status"` // PASS, FAIL, SKIP
	Details    string `json:"details" yaml:"details"`
}

// TestSummary provides aggregate statistics
type TestSummary struct {
	Total     int `json:"total" yaml:"total"`
	Passed    int `json:"passed" yaml:"passed"`
	Failed    int `json:"failed" yaml:"failed"`
	Conflicts int `json:"conflicts" yaml:"conflicts"`
}

// TestReportGenerator creates test reports in various formats
type TestReportGenerator struct{}

// NewTestReportGenerator creates a new report generator
func NewTestReportGenerator() *TestReportGenerator {
	return &TestReportGenerator{}
}

// GenerateReport creates a test report from test results
func (g *TestReportGenerator) GenerateReport(tenantID, coreVersion string, results []TestResult) *TestReport {
	report := &TestReport{
		ReportID:       fmt.Sprintf("report-%s-%d", tenantID, time.Now().Unix()),
		TenantID:       tenantID,
		CoreVersion:    coreVersion,
		OverlayVersion: fmt.Sprintf("%s-overlay", coreVersion),
		Timestamp:      time.Now(),
		Tests:          make([]TestEntry, 0, len(results)),
		UARLogID:       fmt.Sprintf("uar-%d", time.Now().Unix()),
		SnapshotID:     fmt.Sprintf("snap-%d", time.Now().Unix()),
	}

	passed := 0
	failed := 0

	for _, result := range results {
		entry := TestEntry{
			Name:       result.Name,
			EntityType: g.extractEntityType(result.Name),
			EntityID:   g.extractEntityID(result.Name),
			Status:     result.Status,
			Details:    result.Details,
		}
		report.Tests = append(report.Tests, entry)

		if result.Status == "PASS" {
			passed++
		} else if result.Status == "FAIL" {
			failed++
		}
	}

	report.Summary = TestSummary{
		Total:     len(results),
		Passed:    passed,
		Failed:    failed,
		Conflicts: 0, // Set by caller if conflicts detected
	}

	return report
}

// ToJSON exports report as JSON
func (g *TestReportGenerator) ToJSON(report *TestReport) (string, error) {
	bytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ToYAML exports report as YAML (regulator-friendly)
func (g *TestReportGenerator) ToYAML(report *TestReport) (string, error) {
	bytes, err := yaml.Marshal(report)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Helper methods to extract entity info from test name
func (g *TestReportGenerator) extractEntityType(testName string) string {
	// Simple parsing - in production use regex
	if len(testName) > 3 {
		if testName[:3] == "BO_" {
			return "BusinessObject"
		} else if testName[:3] == "BP_" {
			return "BusinessProcess"
		} else if testName[:5] == "View_" {
			return "UIView"
		} else if testName[:7] == "Metric_" {
			return "Metric"
		}
	}
	return "Unknown"
}

func (g *TestReportGenerator) extractEntityID(testName string) string {
	// Simple extraction - in production use proper parsing
	return "Unknown"
}

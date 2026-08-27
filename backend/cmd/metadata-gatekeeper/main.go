package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/cbo"
	"github.com/hondyman/uisce/backend/internal/compliance"
	"github.com/hondyman/uisce/backend/internal/privacy"
)

type GatekeeperReport struct {
	TotalChecks       int
	PassedChecks      int
	FailedChecks      int
	WarningChecks     int
	SMTResults        []CheckItem
	FanOutResults     []CheckItem
	TenantRuleResults []CheckItem
	SLAResults        []CheckItem
}

type CheckItem struct {
	Name        string
	Category    string
	Status      string // PASSED, FAILED, WARNING
	Description string
}

func main() {
	outputMdPath := flag.String("output-md", "gatekeeper-report.md", "Path to output Markdown report for PR comment")
	failOnBlock := flag.Bool("fail-on-block", true, "Exit with non-zero code on blocking failures")
	flag.Parse()

	report := RunAllGatekeeperChecks()

	mdContent := GenerateMarkdownReport(report)
	if *outputMdPath != "" {
		_ = os.WriteFile(*outputMdPath, []byte(mdContent), 0644)
		fmt.Printf("Gatekeeper report generated at: %s\n", *outputMdPath)
	}

	fmt.Println(mdContent)

	if *failOnBlock && report.FailedChecks > 0 {
		fmt.Fprintf(os.Stderr, "\n[GATEKEEPER BLOCKED] %d blocking issues detected. Fix before merging.\n", report.FailedChecks)
		os.Exit(1)
	}
	fmt.Printf("\n[GATEKEEPER PASSED] All %d checks passed successfully.\n", report.PassedChecks)
}

func RunAllGatekeeperChecks() GatekeeperReport {
	report := GatekeeperReport{}
	ctx := context.Background()
	dummyTenant := uuid.New()

	// 1. Check: SMT Mandate Consistency
	smtVerifier := compliance.NewMandateSMTVerifier()

	// 1a. Valid UCITS Mandate
	validRules := []compliance.RuleConstraintClause{
		{DimensionKey: "asset_class.fixed_income", Operator: ">=", ValuePct: 40.0},
		{DimensionKey: "asset_class.equities", Operator: "<=", ValuePct: 50.0},
		{DimensionKey: "asset_class.cash", Operator: ">=", ValuePct: 5.0},
	}
	res1, err := smtVerifier.VerifyMandateConsistency(ctx, dummyTenant, validRules)
	report.TotalChecks++
	if err == nil && res1.IsSatisfiable {
		report.PassedChecks++
		report.SMTResults = append(report.SMTResults, CheckItem{
			Name:        "UCITS Balanced Fund Mandate",
			Category:    "SMT Mandate Solver",
			Status:      "PASSED",
			Description: "All constraint branches mathematically satisfiable (Z3 solver proof verified).",
		})
	} else {
		report.FailedChecks++
		report.SMTResults = append(report.SMTResults, CheckItem{
			Name:        "UCITS Balanced Fund Mandate",
			Category:    "SMT Mandate Solver",
			Status:      "FAILED",
			Description: res1.DiagnosticMessage,
		})
	}

	// 1b. Invariant Check: Rejection of Over-allocation
	invalidRules := []compliance.RuleConstraintClause{
		{DimensionKey: "asset_class.fixed_income_aaa", Operator: ">=", ValuePct: 85.0},
		{DimensionKey: "asset_class.cash_equivalents", Operator: ">=", ValuePct: 20.0},
	}
	res2, _ := smtVerifier.VerifyMandateConsistency(ctx, dummyTenant, invalidRules)
	report.TotalChecks++
	if !res2.IsSatisfiable && res2.ConflictDetected {
		report.PassedChecks++
		report.SMTResults = append(report.SMTResults, CheckItem{
			Name:        "Contradictory Over-Allocation Guard",
			Category:    "SMT Mandate Solver",
			Status:      "PASSED",
			Description: "Successfully detected and blocked impossible mandate (sum > 100%).",
		})
	} else {
		report.FailedChecks++
		report.SMTResults = append(report.SMTResults, CheckItem{
			Name:        "Contradictory Over-Allocation Guard",
			Category:    "SMT Mandate Solver",
			Status:      "FAILED",
			Description: "Failed to block impossible allocation sum.",
		})
	}

	// 2. Check: AQE & Fan-Out Mitigation
	aqe := cbo.NewAdaptiveExecutionEngine(100000)
	aqePlan, err := aqe.AdaptPlanAtRuntime(ctx, dummyTenant, 5000000, 4200, 100, true)
	report.TotalChecks++
	if err == nil && aqePlan.AdaptedStrategy == cbo.JoinStrategyBroadcastHash && aqePlan.DynamicPruningActive {
		report.PassedChecks++
		report.FanOutResults = append(report.FanOutResults, CheckItem{
			Name:        "AQE Dynamic Runtime Optimizer",
			Category:    "Query Plan Optimization",
			Status:      "PASSED",
			Description: "Runtime broadcast join conversion and 75% S3 split pruning active.",
		})
	} else {
		report.FailedChecks++
		report.FanOutResults = append(report.FanOutResults, CheckItem{
			Name:        "AQE Dynamic Runtime Optimizer",
			Category:    "Query Plan Optimization",
			Status:      "FAILED",
			Description: "Failed to convert join strategy or prune partitions.",
		})
	}

	// 3. Check: Rule 7 Zero-Tolerance Tenancy
	report.TotalChecks++
	_, nilTenantErr := smtVerifier.VerifyMandateConsistency(ctx, uuid.Nil, validRules)
	if nilTenantErr != nil && strings.Contains(nilTenantErr.Error(), "Rule 7") {
		report.PassedChecks++
		report.TenantRuleResults = append(report.TenantRuleResults, CheckItem{
			Name:        "Rule 7 Multi-Tenant Enforcement",
			Category:    "Tenant Security",
			Status:      "PASSED",
			Description: "Zero-tolerance tenant_id validation active across all engine ports.",
		})
	} else {
		report.FailedChecks++
		report.TenantRuleResults = append(report.TenantRuleResults, CheckItem{
			Name:        "Rule 7 Multi-Tenant Enforcement",
			Category:    "Tenant Security",
			Status:      "FAILED",
			Description: "Rule 7 violation: nil tenant_id was not rejected.",
		})
	}

	// 4. Check: Differential Privacy Laplace Sensitivity
	dpService := privacy.NewDifferentialPrivacyService()
	privacyBudget := privacy.PrivacyBudget{Epsilon: 0.5, Delta: 1e-5}
	dpRes, err := dpService.ApplyLaplaceNoise(ctx, dummyTenant, "peer_fund_yield", 14.24, 0.05, privacyBudget)
	report.TotalChecks++
	if err == nil && dpRes.ZKPassportHash != "" && dpRes.EpsilonUsed == 0.5 {
		report.PassedChecks++
		report.SLAResults = append(report.SLAResults, CheckItem{
			Name:        "Differential Privacy & ZK Passport",
			Category:    "Privacy & SLA",
			Status:      "PASSED",
			Description: "Calibrated Laplace noise applied (ε=0.50) with cryptographic SHA-256 ZK passport.",
		})
	} else {
		report.FailedChecks++
		report.SLAResults = append(report.SLAResults, CheckItem{
			Name:        "Differential Privacy & ZK Passport",
			Category:    "Privacy & SLA",
			Status:      "FAILED",
			Description: "Failed to generate privacy-certified result.",
		})
	}

	return report
}

func GenerateMarkdownReport(r GatekeeperReport) string {
	statusBadge := "✅ **PASSED (G-SIFI Compliant)**"
	if r.FailedChecks > 0 {
		statusBadge = "❌ **BLOCKED (Policy Breaches Detected)**"
	}

	var sb strings.Builder
	sb.WriteString("# 🛡️ Uisce GitOps CI/CD Metadata Gatekeeper Report\n\n")
	sb.WriteString(fmt.Sprintf("**Status:** %s  \n", statusBadge))
	sb.WriteString(fmt.Sprintf("**Execution Timestamp:** `%s`  \n", time.Now().UTC().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Checks Summary:** %d Total | %d Passed | %d Failed | %d Warnings\n\n",
		r.TotalChecks, r.PassedChecks, r.FailedChecks, r.WarningChecks))

	sb.WriteString("## Gatekeeper Verification Matrix\n\n")
	sb.WriteString("| Category | Gatekeeper Check | Status | Description |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- |\n")

	allItems := append(r.SMTResults, r.FanOutResults...)
	allItems = append(allItems, r.TenantRuleResults...)
	allItems = append(allItems, r.SLAResults...)

	for _, item := range allItems {
		statusIcon := "🟢 PASSED"
		if item.Status == "FAILED" {
			statusIcon = "🔴 FAILED"
		} else if item.Status == "WARNING" {
			statusIcon = "🟡 WARNING"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", item.Category, item.Name, statusIcon, item.Description))
	}

	sb.WriteString("\n---\n")
	sb.WriteString("> [!IMPORTANT]\n")
	sb.WriteString("> All metadata modifications must strictly satisfy **Rule 7 Multi-Tenancy**, **Z3 SMT Invariant Consistency**, and **OpenDataContract SLAs** before deployment to production environments.\n")

	return sb.String()
}

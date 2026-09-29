package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"

	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/rules/vm"
)

const (
	// Exit codes for CI/CD pipelines:
	ExitSuccess         = 0 // Clean execution, no diff/import/checksum errors
	ExitValidationError = 1 // Preflight/Import report errors, diff conflicts, or checksum mismatch
	ExitTransportError   = 2 // Transport/network failure, server error (5xx/401/403), or bad config
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(ExitTransportError)
	}

	subcommand := os.Args[1]
	args := os.Args[2:]

	switch subcommand {
	case "export":
		runExport(args)
	case "diff", "preview":
		runDiff(args)
	case "import":
		runImport(args)
	case "validate-bundle", "validate":
		runValidateBundle(args)
	case "help", "--help", "-h":
		printUsage()
		os.Exit(ExitSuccess)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", subcommand)
		printUsage()
		os.Exit(ExitTransportError)
	}
}

func printUsage() {
	fmt.Println("uisce-rules - Centralized Validation Rules Portability CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  uisce-rules <command> [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  export           Export rules from an environment as JSON/YAML bundle")
	fmt.Println("  diff / preview   Dry-run preview bundle diffs against a target environment")
	fmt.Println("  import           Import rule bundle into a target environment")
	fmt.Println("  validate-bundle  Validate and/or stamp checksum on a local bundle file")
	fmt.Println()
	fmt.Println("Precedence Note:")
	fmt.Println("  Command line flags and query parameters override properties in bundle files.")
	fmt.Println()
	fmt.Println("Exit Codes (for CI/CD):")
	fmt.Println("  0 - Success (clean import / valid bundle / no errors)")
	fmt.Println("  1 - Validation / Preflight / Import Error (diff conflicts, rule AST errors, checksum mismatch)")
	fmt.Println("  2 - Transport / Configuration Error (network down, 401/403/500, missing required flags)")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  UISCE_API_URL    API server base URL (default: http://localhost:8080)")
	fmt.Println("  UISCE_TOKEN      Bearer authentication token / JWT")
	fmt.Println("  UISCE_TENANT_ID  Target tenant UUID")
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func runExport(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	apiURL := fs.String("api-url", getEnvOrDefault("UISCE_API_URL", "http://localhost:8080"), "Base API URL")
	token := fs.String("token", getEnvOrDefault("UISCE_TOKEN", os.Getenv("UISCE_JWT")), "Auth token")
	tenantID := fs.String("tenant", getEnvOrDefault("UISCE_TENANT_ID", ""), "Tenant ID")
	boName := fs.String("bo", "", "Filter by Business Object (optional)")
	domain := fs.String("domain", "", "Filter by domain (optional)")
	origin := fs.String("origin", "", "Filter by origin ('core', 'custom', or '' for all)")
	format := fs.String("format", "yaml", "Output format: yaml or json")
	outputFile := fs.String("o", "", "Output file path (default stdout)")

	fs.Parse(args)

	if *tenantID == "" {
		fmt.Fprintln(os.Stderr, "Error: --tenant or UISCE_TENANT_ID is required")
		os.Exit(ExitTransportError)
	}

	endpoint := strings.TrimRight(*apiURL, "/")
	if !strings.HasSuffix(endpoint, "/api") {
		endpoint += "/api"
	}
	endpoint += fmt.Sprintf("/validation-rule-nodes/export?format=%s", *format)
	if *boName != "" {
		endpoint += "&bo_name=" + *boName
	}
	if *domain != "" {
		endpoint += "&domain=" + *domain
	}
	if *origin != "" {
		endpoint += "&origin=" + *origin
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(ExitTransportError)
	}
	req.Header.Set("X-Tenant-ID", *tenantID)
	if *token != "" {
		req.Header.Set("Authorization", "Bearer "+*token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server: %v\n", err)
		os.Exit(ExitTransportError)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(ExitTransportError)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Export failed (HTTP %d): %s\n", resp.StatusCode, string(body))
		os.Exit(ExitTransportError)
	}

	if *outputFile != "" {
		if err := os.WriteFile(*outputFile, body, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file %s: %v\n", *outputFile, err)
			os.Exit(ExitTransportError)
		}
		fmt.Printf("Successfully exported rules to %s\n", *outputFile)
	} else {
		fmt.Print(string(body))
	}
	os.Exit(ExitSuccess)
}

func readPayload(filePath string) ([]byte, error) {
	if filePath == "" || filePath == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(filePath)
}

func runDiff(args []string) {
	fs := flag.NewFlagSet("diff", flag.ExitOnError)
	apiURL := fs.String("api-url", getEnvOrDefault("UISCE_API_URL", "http://localhost:8080"), "Base API URL")
	token := fs.String("token", getEnvOrDefault("UISCE_TOKEN", os.Getenv("UISCE_JWT")), "Auth token")
	tenantID := fs.String("tenant", getEnvOrDefault("UISCE_TENANT_ID", ""), "Tenant ID")
	filePath := fs.String("f", "", "Bundle file path (JSON or YAML, - for stdin)")
	overwrite := fs.String("overwrite", models.ImportOverwriteFail, "Overwrite policy (fail_on_conflict|skip_existing|overwrite)")
	preserveStatus := fs.Bool("preserve-status", false, "Preserve target environment governance status")
	prune := fs.Bool("prune", false, "Prune unreferenced custom rules")

	fs.Parse(args)

	if *tenantID == "" {
		fmt.Fprintln(os.Stderr, "Error: --tenant or UISCE_TENANT_ID is required")
		os.Exit(ExitTransportError)
	}

	data, err := readPayload(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading bundle: %v\n", err)
		os.Exit(ExitTransportError)
	}

	endpoint := strings.TrimRight(*apiURL, "/")
	if !strings.HasSuffix(endpoint, "/api") {
		endpoint += "/api"
	}
	endpoint += fmt.Sprintf("/validation-rule-nodes/import/preview?overwrite_policy=%s&preserve_status=%t&prune_missing=%t", *overwrite, *preserveStatus, *prune)

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(ExitTransportError)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", *tenantID)
	if *token != "" {
		req.Header.Set("Authorization", "Bearer "+*token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server: %v\n", err)
		os.Exit(ExitTransportError)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(ExitTransportError)
	}

	var report models.RuleImportReport
	if err := json.Unmarshal(body, &report); err != nil {
		fmt.Fprintf(os.Stderr, "Server response (HTTP %d): %s\n", resp.StatusCode, string(body))
		os.Exit(ExitTransportError)
	}

	printReport(&report, true)
	if resp.StatusCode != http.StatusOK || !report.Success || len(report.Errors) > 0 {
		os.Exit(ExitValidationError)
	}
	os.Exit(ExitSuccess)
}

func runImport(args []string) {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	apiURL := fs.String("api-url", getEnvOrDefault("UISCE_API_URL", "http://localhost:8080"), "Base API URL")
	token := fs.String("token", getEnvOrDefault("UISCE_TOKEN", os.Getenv("UISCE_JWT")), "Auth token")
	tenantID := fs.String("tenant", getEnvOrDefault("UISCE_TENANT_ID", ""), "Tenant ID")
	filePath := fs.String("f", "", "Bundle file path (JSON or YAML, - for stdin)")
	dryRun := fs.Bool("dry-run", false, "Execute preview only without committing changes")
	overwrite := fs.String("overwrite", models.ImportOverwriteFail, "Overwrite policy (fail_on_conflict|skip_existing|overwrite)")
	preserveStatus := fs.Bool("preserve-status", false, "Preserve target environment governance status")
	prune := fs.Bool("prune", false, "Prune unreferenced custom rules")
	idemKey := fs.String("idempotency-key", "", "Custom idempotency key")

	fs.Parse(args)

	if *tenantID == "" {
		fmt.Fprintln(os.Stderr, "Error: --tenant or UISCE_TENANT_ID is required")
		os.Exit(ExitTransportError)
	}

	data, err := readPayload(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading bundle: %v\n", err)
		os.Exit(ExitTransportError)
	}

	endpoint := strings.TrimRight(*apiURL, "/")
	if !strings.HasSuffix(endpoint, "/api") {
		endpoint += "/api"
	}
	endpoint += fmt.Sprintf("/validation-rule-nodes/import?dry_run=%t&overwrite_policy=%s&preserve_status=%t&prune_missing=%t", *dryRun, *overwrite, *preserveStatus, *prune)

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(ExitTransportError)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", *tenantID)
	if *token != "" {
		req.Header.Set("Authorization", "Bearer "+*token)
	}
	if *idemKey != "" {
		req.Header.Set("X-Idempotency-Key", *idemKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server: %v\n", err)
		os.Exit(ExitTransportError)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(ExitTransportError)
	}

	var report models.RuleImportReport
	if err := json.Unmarshal(body, &report); err != nil {
		fmt.Fprintf(os.Stderr, "Server response (HTTP %d): %s\n", resp.StatusCode, string(body))
		os.Exit(ExitTransportError)
	}

	printReport(&report, *dryRun)
	if resp.StatusCode != http.StatusOK || !report.Success || len(report.Errors) > 0 {
		os.Exit(ExitValidationError)
	}
	os.Exit(ExitSuccess)
}

func printReport(report *models.RuleImportReport, isDryRun bool) {
	fmt.Println("==================================================")
	if isDryRun || report.DryRun {
		fmt.Println(" RULE IMPORT PREVIEW / DRY RUN REPORT")
	} else {
		fmt.Println(" RULE IMPORT EXECUTION REPORT")
	}
	fmt.Println("==================================================")
	status := "SUCCESS"
	if !report.Success || len(report.Errors) > 0 {
		status = "FAILED"
	}
	fmt.Printf("Status:       %s\n", status)
	fmt.Printf("Total Rules:  %d\n", report.TotalRules)
	if !isDryRun && !report.DryRun {
		fmt.Printf("Created:      %d\n", len(report.Created))
		fmt.Printf("Updated:      %d\n", len(report.Updated))
		fmt.Printf("Skipped:      %d\n", len(report.Skipped))
		if len(report.Pruned) > 0 {
			fmt.Printf("Pruned:       %d\n", len(report.Pruned))
		}
	}
	fmt.Println()

	if len(report.DiffSummary) > 0 {
		fmt.Println("PLANNED ACTIONS:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ACTION\tRULE KEY\tDETAILS")
		fmt.Fprintln(w, "------\t--------\t-------")
		for _, d := range report.DiffSummary {
			details := "-"
			if len(d.Changes) > 0 {
				details = strings.Join(d.Changes, ", ")
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", d.Action, d.RuleKey, details)
		}
		w.Flush()
		fmt.Println()
	}

	if len(report.Errors) > 0 {
		fmt.Printf("ERRORS (%d):\n", len(report.Errors))
		w := tabwriter.NewWriter(os.Stderr, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "CODE\tRULE KEY\tREASON")
		fmt.Fprintln(w, "----\t--------\t------")
		for _, e := range report.Errors {
			fmt.Fprintf(w, "%s\t%s\t%s\n", e.Code, e.RuleKey, e.Reason)
		}
		w.Flush()
		fmt.Println()
	}
}

func runValidateBundle(args []string) {
	fs := flag.NewFlagSet("validate-bundle", flag.ExitOnError)
	filePath := fs.String("f", "", "Bundle file path (JSON or YAML)")
	stamp := fs.Bool("stamp", false, "Recalculate and stamp checksum back into file")
	format := fs.String("format", "yaml", "Output format when stamping: yaml or json")

	fs.Parse(args)

	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "Error: -f is required")
		os.Exit(ExitTransportError)
	}

	data, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", *filePath, err)
		os.Exit(ExitTransportError)
	}

	var bundle models.RuleBundle
	if err := json.Unmarshal(data, &bundle); err != nil || len(bundle.Rules) == 0 {
		var raw interface{}
		if yErr := yaml.Unmarshal(data, &raw); yErr != nil {
			fmt.Fprintf(os.Stderr, "Error parsing file as JSON or YAML: %v (yaml: %v)\n", err, yErr)
			os.Exit(ExitTransportError)
		}
		jsonBytes, mErr := json.Marshal(raw)
		if mErr != nil {
			fmt.Fprintf(os.Stderr, "Error normalizing YAML structure: %v\n", mErr)
			os.Exit(ExitTransportError)
		}
		if err := json.Unmarshal(jsonBytes, &bundle); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing bundle: %v\n", err)
			os.Exit(ExitTransportError)
		}
	}

	fmt.Printf("Bundle Version: %s\n", bundle.BundleVersion)
	fmt.Printf("Rules Count:    %d\n", len(bundle.Rules))

	// Verify ASTs
	var astErrors []string
	for _, r := range bundle.Rules {
		if !r.Deleted && len(r.RuleAST) > 0 {
			var node vm.RuleNode
			if err := json.Unmarshal(r.RuleAST, &node); err != nil {
				astErrors = append(astErrors, fmt.Sprintf("Rule %s: invalid AST JSON (%v)", r.RuleKey, err))
			} else if _, err := vm.Compact(node); err != nil {
				astErrors = append(astErrors, fmt.Sprintf("Rule %s: invalid AST (%v)", r.RuleKey, err))
			}
		}
	}

	if len(astErrors) > 0 {
		fmt.Fprintf(os.Stderr, "Validation failed with %d AST errors:\n", len(astErrors))
		for _, e := range astErrors {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		os.Exit(ExitValidationError)
	}

	storedChecksum := bundle.Checksum
	if err := bundle.ComputeChecksum(); err != nil {
		fmt.Fprintf(os.Stderr, "Error computing checksum: %v\n", err)
		os.Exit(ExitValidationError)
	}
	computedSum := bundle.Checksum

	if *stamp {
		bundle.Checksum = computedSum
		var outData []byte
		if *format == "json" {
			outData, err = json.MarshalIndent(bundle, "", "  ")
		} else {
			jsonBytes, mErr := json.Marshal(bundle)
			if mErr != nil {
				fmt.Fprintf(os.Stderr, "Error formatting bundle: %v\n", mErr)
				os.Exit(ExitTransportError)
			}
			var raw interface{}
			_ = json.Unmarshal(jsonBytes, &raw)
			outData, err = yaml.Marshal(raw)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling stamped bundle: %v\n", err)
			os.Exit(ExitTransportError)
		}
		if err := os.WriteFile(*filePath, outData, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing stamped bundle: %v\n", *filePath, err)
			os.Exit(ExitTransportError)
		}
		fmt.Printf("Successfully stamped checksum: %s\n", computedSum)
		os.Exit(ExitSuccess)
	}

	if storedChecksum == "" {
		fmt.Printf("Checksum:       (not stamped)\n")
		fmt.Printf("Computed Sum:   %s\n", computedSum)
		fmt.Println("Status:         VALID ASTs (run with --stamp to save checksum)")
		os.Exit(ExitSuccess)
	}

	bundle.Checksum = storedChecksum
	if err := bundle.VerifyChecksum(); err != nil {
		fmt.Fprintf(os.Stderr, "CHECKSUM MISMATCH!\n  %v\n", err)
		os.Exit(ExitValidationError)
	}

	fmt.Printf("Checksum:       %s (VERIFIED)\n", bundle.Checksum)
	fmt.Println("Status:         VALID")
	os.Exit(ExitSuccess)
}

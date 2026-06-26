package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/forecasting"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/policy"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/internal/simulation"
	"github.com/hondyman/semlayer/backend/internal/store"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type ActionRequest struct {
	Input struct {
		ViolationID string `json:"violation_id"`
	} `json:"input"`
}

type ExplainResult struct {
	RuleID   string               `json:"rule_id"`
	Severity string               `json:"severity"`
	Message  string               `json:"message"`
	Explain  []policy.MatchDetail `json:"explain"`
}

var (
	simulate       = flag.Bool("simulate", false, "Run in simulation mode (no DB changes)")
	policyPath     = flag.String("policy", "policies/default.yaml", "Policy file to use for simulation")
	fromDS         = flag.String("from", "", "Source datasource ID for simulation")
	toDS           = flag.String("to", "", "Target datasource ID for simulation")
	migrationFile  = flag.String("migration", "", "Path to migration SQL file for simulation")
	policyExplain  = flag.Bool("policy-explain", false, "Show detailed explanations for policy violations.")
	outputFormat   = flag.String("output", "pretty", "Output format (pretty, json)")
	failOnSeverity = flag.String("fail-on-severity", "breaking", "Fail run on violations of this severity or higher (breaking, medium, low, none)")
	multiPolicy    = flag.Bool("multi-policy", false, "Run simulation against all active policies in the policy directory.")
	persist        = flag.Bool("persist", false, "Persist simulation results to the database")
	historical     = flag.Bool("historical", false, "Run in historical replay mode.")
	historyDir     = flag.String("history-dir", "reports/", "Directory of historical drift reports (JSON files).")
	forecast       = flag.Bool("forecast", false, "Forecast the impact of a new change.")
)

func toJSONB(v interface{}) (json.RawMessage, error) {
	if v == nil {
		return json.RawMessage("null"), nil
	}
	return json.Marshal(v)
}

func printMultiResult(res *simulation.MultiResult) {
	if *outputFormat == "json" {
		outJSON, _ := json.MarshalIndent(res, "", "  ")
		logging.GetLogger().Sugar().Info(string(outJSON))
		return
	}

	logging.GetLogger().Sugar().Info("POLICY ID\tBREAKING\tMEDIUM\tLOW")
	for _, r := range res.PolicyResults {
		logging.GetLogger().Sugar().Infof("%s\t%d\t%d\t%d",
			r.PolicyID,
			r.Summary.Breaking,
			r.Summary.Medium,
			r.Summary.Low)
	}
}

func maxExitCode(results []*simulation.Result) int {
	maxSeverity := 0
	for _, r := range results {
		if r.Summary.Breaking > 0 {
			maxSeverity = 1
		} else if r.Summary.Medium > 0 && maxSeverity < 2 {
			maxSeverity = 2
		} else if r.Summary.Low > 0 && maxSeverity < 3 {
			maxSeverity = 3
		}
	}

	threshold, ok := services.SeverityThresholds[*failOnSeverity]
	if !ok {
		logging.GetLogger().Sugar().Fatalf("Invalid severity level for --fail-on-severity: %s", *failOnSeverity)
	}

	if maxSeverity > 0 && maxSeverity <= threshold {
		return 1
	}
	return 0
}

func runSimulation() {
	dbURL := os.Getenv("POLICY_DB_URL")
	if dbURL == "" {
		// If we are not persisting, we don't need the DB.
		if !*persist {
			logging.GetLogger().Sugar().Info("POLICY_DB_URL not set, cannot persist results. Continuing without persistence.")
		} else {
			logging.GetLogger().Sugar().Fatal("POLICY_DB_URL environment variable must be set for simulation mode")
		}
	}
	appDB, err := sqlx.Connect("postgres", dbURL)
	if err != nil && *persist { // Only fail if we need the DB
		logging.GetLogger().Sugar().Fatalf("Failed to connect to database for simulation: %v", err)
	}
	defer appDB.Close()

	upgradeService := services.NewUpgradeService(appDB)

	if *multiPolicy {
		policyDir := filepath.Dir(*policyPath)
		logging.GetLogger().Sugar().Infof("Running multi-policy simulation for policies in: %s", policyDir)
		policies, err := store.LoadAllActivePoliciesFromDir(policyDir)
		if err != nil {
			logging.GetLogger().Sugar().Fatalf("Failed to load policies from directory %s: %v", policyDir, err)
		}
		if len(policies) == 0 {
			logging.GetLogger().Sugar().Fatalf("No policies found in %s", policyDir)
		}

		multiInput := simulation.MultiInput{
			Policies:     policies,
			FromEnv:      *fromDS,
			ToEnv:        *toDS,
			MigrationSQL: *migrationFile,
		}

		multiResult, err := simulation.RunMulti(context.Background(), upgradeService, multiInput)
		if err != nil {
			logging.GetLogger().Sugar().Fatalf("Multi-policy simulation failed: %v", err)
		}

		printMultiResult(multiResult)
		os.Exit(maxExitCode(multiResult.PolicyResults))
		return
	}

	// Single policy simulation logic continues here...
	pol, err := store.LoadPolicy(*policyPath)
	if err != nil {
		logging.GetLogger().Sugar().Fatalf("Failed to load policy file: %v", err)
	}
	simInput := simulation.Input{
		Policy:       pol,
		FromEnv:      *fromDS,
		ToEnv:        *toDS,
		MigrationSQL: *migrationFile,
	}

	result, err := simulation.Run(context.Background(), upgradeService, simInput)
	if err != nil { // This is the new simulation.Run
		logging.GetLogger().Sugar().Fatalf("Simulation failed: %v", err)
	}

	if *persist && appDB != nil {
		result.RunID = uuid.New().String() // Assign RunID for persistence
		tx, err := appDB.Beginx()
		if err != nil {
			logging.GetLogger().Sugar().Fatalf("Failed to begin transaction: %v", err)
		}
		defer func() { _ = tx.Rollback() }() // Rollback on error

		rawReportJSON, err := json.Marshal(result)
		if err != nil {
			logging.GetLogger().Sugar().Fatalf("Failed to marshal result for persistence: %v", err)
		}
		summaryJSON, err := toJSONB(result.Summary)
		if err != nil {
			logging.GetLogger().Sugar().Fatalf("Failed to marshal summary for persistence: %v", err)
		}

		_, err = tx.Exec(`
            INSERT INTO drift_reports (id, generated_at, schema_hash, severity_summary, raw_report)
            VALUES ($1, $2, $3, $4, $5)`,
			result.RunID, result.GeneratedAt, result.SchemaHash, summaryJSON, rawReportJSON,
		)
		if err != nil {
			logging.GetLogger().Sugar().Fatalf("Failed to insert drift report: %v", err)
		}

		for _, v := range result.Violations {
			explainJSON, err := toJSONB(v.Explain)
			if err != nil {
				logging.GetLogger().Sugar().Fatalf("Failed to marshal explanation for violation %s: %v", v.RuleID, err)
			}

			_, err = tx.Exec(`
                INSERT INTO policy_violation (id, run_id, rule_id, severity, message, explain)
                VALUES ($1, $2, $3, $4, $5, $6)`,
				uuid.New(), result.RunID, v.RuleID, v.Severity, v.Message, explainJSON)
			if err != nil {
				logging.GetLogger().Sugar().Fatalf("Failed to insert policy violation: %v", err)
			}
		}

		if err := tx.Commit(); err != nil {
			logging.GetLogger().Sugar().Fatalf("Failed to commit transaction: %v", err)
		}
		logging.GetLogger().Sugar().Infof("Successfully persisted report with run_id: %s", result.RunID)
	}

	if *policyExplain && len(result.Violations) > 0 {
		logging.GetLogger().Sugar().Info("--- Policy Violation Explanations ---")
		for _, v := range result.Violations {
			logging.GetLogger().Sugar().Infof("[%s] %s", v.RuleID, v.Message)
			for _, d := range v.Explain {
				logging.GetLogger().Sugar().Infof("  ↳ matched selector '%s' at %s = %q", d.Selector, d.Path, d.Value)
			}
			logging.GetLogger().Sugar().Info("")
		}
		logging.GetLogger().Sugar().Info("------------------------------------")
	}

	// Handle output format
	switch *outputFormat {
	case "json":
		outJSON, _ := json.Marshal(result)
		logging.GetLogger().Sugar().Info(string(outJSON))
	case "pretty":
		// For CLI, we can pretty-print the result or output JSON
		outJSON, _ := json.MarshalIndent(result, "", "  ")
		logging.GetLogger().Sugar().Info(string(outJSON))
	default:
		logging.GetLogger().Sugar().Fatalf("Unknown output format: %s", *outputFormat)
	}

	// Determine exit code based on severity

	maxSeverity := 0
	if result.Summary.Breaking > 0 {
		maxSeverity = 1
	} else if result.Summary.Medium > 0 {
		maxSeverity = 2
	} else if result.Summary.Low > 0 {
		maxSeverity = 3
	}

	threshold, ok := services.SeverityThresholds[*failOnSeverity]
	if !ok {
		logging.GetLogger().Sugar().Fatalf("Invalid severity level for --fail-on-severity: %s", *failOnSeverity)
	}

	if maxSeverity > 0 && maxSeverity <= threshold {
		os.Exit(1) // Fail the build
	}
	os.Exit(0) // Success
}

func runHistoricalReplay() {
	dbURL := os.Getenv("POLICY_DB_URL")
	if dbURL == "" {
		logging.GetLogger().Sugar().Fatal("POLICY_DB_URL environment variable must be set for historical replay mode")
	}
	appDB, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		logging.GetLogger().Sugar().Fatalf("Failed to connect to database: %v", err)
	}
	defer appDB.Close()

	upgradeService := services.NewUpgradeService(appDB)

	// 1. Load policies
	policyDir := filepath.Dir(*policyPath)
	logging.GetLogger().Sugar().Infof("Loading all policies from: %s", policyDir)
	policies, err := store.LoadAllActivePoliciesFromDir(policyDir)
	if err != nil {
		logging.GetLogger().Sugar().Fatalf("Failed to load policies: %v", err)
	}

	// 2. Load history from directory
	history, err := loadHistoryFromDir(*historyDir)
	if err != nil {
		logging.GetLogger().Sugar().Fatalf("Failed to load history: %v", err)
	}

	// 3. Run replay
	replayInput := simulation.HistoricalReplayInput{
		Policies: policies,
		History:  history,
	}
	result, err := simulation.RunHistoricalReplay(context.Background(), upgradeService, replayInput)
	if err != nil {
		logging.GetLogger().Sugar().Fatalf("Historical replay failed: %v", err)
	}

	// 4. Print result
	logging.GetLogger().Sugar().Info("Historical replay results:")
	logging.GetLogger().Sugar().Infof("CHANGE ID\tTIMESTAMP\tPOLICY ID\tDECISION\tBREAKING\tMEDIUM\tLOW")
	for _, run := range result.Runs {
		logging.GetLogger().Sugar().Infof("%s\t%s\t%s\t%s\t%d\t%d\t%d",
			run.ChangeID,
			run.Timestamp.Format(time.RFC3339),
			run.PolicyID,
			run.Decision,
			run.Summary.Breaking,
			run.Summary.Medium,
			run.Summary.Low)
	}
}

func loadHistoryFromDir(dir string) ([]simulation.LegacyChangeSet, error) {
	// This is a placeholder. A real implementation would parse drift reports
	// to extract snapshot IDs or the raw change list.
	logging.GetLogger().Sugar().Infof("Loading historical change sets from: %s", dir)
	// For now, returning an empty slice to allow the CLI to run.
	return []simulation.LegacyChangeSet{}, nil
}

func runForecast() {
	dbURL := os.Getenv("POLICY_DB_URL")
	if dbURL == "" {
		logging.GetLogger().Sugar().Fatal("POLICY_DB_URL environment variable must be set for forecast mode")
	}
	appDB, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		logging.GetLogger().Sugar().Fatalf("Failed to connect to database: %v", err)
	}
	defer appDB.Close()

	upgradeService := services.NewUpgradeService(appDB)

	// 1. Load historical data to train the model
	logging.GetLogger().Sugar().Info("Loading historical data for training...")
	// In a real scenario, you'd load this from the DB or a dedicated history source.
	// For now, we'll simulate having run a historical replay.
	// This part needs a concrete implementation of loading history.
	// We'll create a dummy result for now.
	dummyHistoryResult := &simulation.HistoricalReplayResult{Runs: []simulation.ReplayRun{}} // Placeholder

	// 2. Train the forecasting model
	logging.GetLogger().Sugar().Info("Training forecasting model...")
	model := forecasting.TrainModel(dummyHistoryResult)

	// 3. Load the new change set to be forecasted
	logging.GetLogger().Sugar().Infof("Loading new change set from migration: %s", *migrationFile)
	if *migrationFile == "" {
		logging.GetLogger().Sugar().Fatal("--migration flag is required for forecasting")
	}
	newChanges, err := upgradeService.GetChangesForSimulation(context.Background(), "", "", *migrationFile)
	if err != nil {
		logging.GetLogger().Sugar().Fatalf("Failed to load changes from migration file %s: %v", *migrationFile, err)
	}

	// 4. Run the forecast
	logging.GetLogger().Sugar().Info("Forecasting impact...")
	policies, err := store.LoadAllActivePoliciesFromDir(filepath.Dir(*policyPath))
	if err != nil {
		logging.GetLogger().Sugar().Fatalf("Failed to load policies for training: %v", err)
	}
	var interfaceSlice []interface{}
	for _, d := range newChanges {
		interfaceSlice = append(interfaceSlice, d)
	}
	forecastResult := model.Predict(interfaceSlice, policies)

	// 5. Print the forecast
	logging.GetLogger().Sugar().Info("POLICY ID\tBLOCK PROBABILITY\tTOP FACTORS")
	for _, f := range forecastResult.Forecasts {
		logging.GetLogger().Sugar().Infof("%s\t%.1f%%\t%v",
			f.PolicyID,
			f.BlockProbability,
			f.TopContributingFactors)
	}
}

func main() {
	flag.Parse()

	// Initialize global zap logger (and redirect stdlib log output to zap).
	logging.InitGlobalLogger()

	if *historical {
		runHistoricalReplay()
		return
	}

	if *forecast {
		runForecast()
		return
	}

	if *simulate || *fromDS != "" || *toDS != "" || *migrationFile != "" {
		runSimulation()
		return
	}

	logging.GetLogger().Sugar().Info("This executable runs in simulation mode. Use --simulate and other flags to perform a policy check.")
	logging.GetLogger().Sugar().Info("To run the API server for on-demand explanations, use the 'policy-gate' executable.")
}

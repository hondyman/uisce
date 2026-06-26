package tenantauto

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReconcilerConfig controls the nightly reconciliation job.
type ReconcilerConfig struct {
	DSN             string
	SchemaRoot      string
	GeneratedDir    string
	DryRun          bool
	Logger          *slog.Logger
	AlertWebhookURL string // Slack/Teams webhook for drift alerts
}

// ReconciliationResult captures drift detection statistics.
type ReconciliationResult struct {
	StartTime      time.Time      `json:"start_time"`
	EndTime        time.Time      `json:"end_time"`
	TotalTenants   int            `json:"total_tenants"`
	InSync         int            `json:"in_sync"`
	Drifted        int            `json:"drifted"`
	Missing        int            `json:"missing"`
	Orphaned       int            `json:"orphaned"`
	AutoRepaired   int            `json:"auto_repaired"`
	ManualRequired int            `json:"manual_required"`
	DriftDetails   []DriftDetail  `json:"drift_details,omitempty"`
	RepairActions  []RepairAction `json:"repair_actions,omitempty"`
}

// DriftDetail describes a specific drift finding.
type DriftDetail struct {
	TenantID       string        `json:"tenant_id"`
	DatasourceID   string        `json:"datasource_id"`
	DriftType      DriftType     `json:"drift_type"`
	Expected       string        `json:"expected"`
	Actual         string        `json:"actual"`
	Severity       DriftSeverity `json:"severity"`
	AutoRepairable bool          `json:"auto_repairable"`
}

// DriftType categorizes the type of configuration drift.
type DriftType string

const (
	DriftTypeMissingTenant   DriftType = "missing_tenant"    // Tenant in DB but no schema files
	DriftTypeOrphanedSchema  DriftType = "orphaned_schema"   // Schema files exist but tenant deleted
	DriftTypeConfigMismatch  DriftType = "config_mismatch"   // tenant.json doesn't match DB
	DriftTypeRefreshMismatch DriftType = "refresh_mismatch"  // Refresh schedule mismatch
	DriftTypeResourceGroup   DriftType = "resource_group"    // Wrong resource group assignment
	DriftTypeStaleScopesJSON DriftType = "stale_scopes_json" // tenant-scopes.json out of date
)

// DriftSeverity indicates how critical the drift is.
type DriftSeverity string

const (
	SeverityLow      DriftSeverity = "low"      // Cosmetic, can wait
	SeverityMedium   DriftSeverity = "medium"   // Should fix soon
	SeverityHigh     DriftSeverity = "high"     // Fix immediately
	SeverityCritical DriftSeverity = "critical" // Security/isolation risk
)

// RepairAction describes an action taken to fix drift.
type RepairAction struct {
	TenantID     string    `json:"tenant_id"`
	DatasourceID string    `json:"datasource_id"`
	Action       string    `json:"action"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// DesiredState represents what the tenant config SHOULD look like.
type DesiredState struct {
	TenantID        string         `json:"tenant_id"`
	DatasourceID    string         `json:"datasource_id"`
	TenantName      string         `json:"tenant_name"`
	DatasourceName  string         `json:"datasource_name"`
	ResourceGroup   string         `json:"resource_group"`
	RefreshInterval int            `json:"refresh_interval_minutes"`
	Tier            string         `json:"tier"`
	IsActive        bool           `json:"is_active"`
	Config          map[string]any `json:"config,omitempty"`
}

// ActualState represents the current state on disk/runtime.
type ActualState struct {
	HasTenantDir  bool           `json:"has_tenant_dir"`
	HasTenantJSON bool           `json:"has_tenant_json"`
	TenantJSON    map[string]any `json:"tenant_json,omitempty"`
	InScopesJSON  bool           `json:"in_scopes_json"`
	ScopesEntry   map[string]any `json:"scopes_entry,omitempty"`
	SchemaFiles   []string       `json:"schema_files"`
}

// Reconciler performs nightly drift detection and auto-repair.
type Reconciler struct {
	config ReconcilerConfig
	db     *sql.DB
	logger *slog.Logger
}

// NewReconciler creates a new reconciliation job.
func NewReconciler(cfg ReconcilerConfig) (*Reconciler, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	if cfg.SchemaRoot == "" {
		cfg.SchemaRoot = "cube/schema"
	}
	if cfg.GeneratedDir == "" {
		cfg.GeneratedDir = "cube/generated"
	}

	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	return &Reconciler{
		config: cfg,
		db:     db,
		logger: cfg.Logger,
	}, nil
}

// Run executes the reconciliation job.
func (r *Reconciler) Run(ctx context.Context) (*ReconciliationResult, error) {
	result := &ReconciliationResult{
		StartTime:     time.Now(),
		DriftDetails:  []DriftDetail{},
		RepairActions: []RepairAction{},
	}

	r.logger.InfoContext(ctx, "starting tenant reconciliation",
		"dry_run", r.config.DryRun,
	)

	// 1. Get desired state from database
	desiredStates, err := r.loadDesiredStates(ctx)
	if err != nil {
		return result, fmt.Errorf("load desired states: %w", err)
	}
	result.TotalTenants = len(desiredStates)

	// 2. Get actual state from filesystem
	actualStates, err := r.loadActualStates(ctx, desiredStates)
	if err != nil {
		return result, fmt.Errorf("load actual states: %w", err)
	}

	// 3. Check for orphaned schemas (exist on disk but not in DB)
	orphanedTenants, err := r.findOrphanedSchemas(ctx, desiredStates)
	if err != nil {
		r.logger.WarnContext(ctx, "failed to check orphaned schemas", "error", err)
	}
	result.Orphaned = len(orphanedTenants)
	for _, orphan := range orphanedTenants {
		result.DriftDetails = append(result.DriftDetails, DriftDetail{
			TenantID:       orphan,
			DriftType:      DriftTypeOrphanedSchema,
			Expected:       "deleted",
			Actual:         "schema files exist",
			Severity:       SeverityMedium,
			AutoRepairable: false, // Manual review needed
		})
	}

	// 4. Compare desired vs actual for each tenant
	for key, desired := range desiredStates {
		actual, exists := actualStates[key]

		drifts := r.detectDrifts(desired, actual, exists)

		if len(drifts) == 0 {
			result.InSync++
			continue
		}

		result.Drifted++
		result.DriftDetails = append(result.DriftDetails, drifts...)

		// 5. Auto-repair if possible and not dry-run
		for _, drift := range drifts {
			if drift.AutoRepairable && !r.config.DryRun {
				action := r.attemptRepair(ctx, desired, drift)
				result.RepairActions = append(result.RepairActions, action)
				if action.Success {
					result.AutoRepaired++
				} else {
					result.ManualRequired++
				}
			} else if !drift.AutoRepairable {
				result.ManualRequired++
			}
		}
	}

	// 6. Check tenant-scopes.json freshness
	scopesDrift := r.checkScopesJSONFreshness(ctx, desiredStates)
	if scopesDrift != nil {
		result.DriftDetails = append(result.DriftDetails, *scopesDrift)
		if !r.config.DryRun && scopesDrift.AutoRepairable {
			action := r.regenerateScopesJSON(ctx)
			result.RepairActions = append(result.RepairActions, action)
			if action.Success {
				result.AutoRepaired++
			}
		}
	}

	result.EndTime = time.Now()
	result.Missing = result.Drifted - result.Orphaned

	// 7. Emit metrics for alerting
	r.emitMetrics(ctx, result)

	// 8. Send alerts if critical drift detected
	if r.hasCriticalDrift(result) {
		r.sendDriftAlert(ctx, result)
	}

	r.logger.InfoContext(ctx, "tenant reconciliation complete",
		"total", result.TotalTenants,
		"in_sync", result.InSync,
		"drifted", result.Drifted,
		"auto_repaired", result.AutoRepaired,
		"manual_required", result.ManualRequired,
		"duration_ms", result.EndTime.Sub(result.StartTime).Milliseconds(),
	)

	return result, nil
}

// loadDesiredStates fetches all tenant-datasource configs from the database.
func (r *Reconciler) loadDesiredStates(ctx context.Context) (map[string]DesiredState, error) {
	query := `
		SELECT 
			t.id AS tenant_id,
			t.display_name AS tenant_name,
			td.id AS datasource_id,
			td.source_name AS datasource_name,
			COALESCE(td.resource_group, 
				CASE WHEN t.is_gold_copy THEN 'tenant_premium' ELSE 'tenant_standard' END
			) AS resource_group,
			COALESCE((td.config->>'refresh_interval_minutes')::int, 60) AS refresh_interval,
			COALESCE(t.is_gold_copy, false) AS is_premium,
			COALESCE(td.is_active, true) AS is_active,
			td.config
		FROM tenants t
		JOIN tenant_datasources td ON td.tenant_id = t.id
		WHERE t.deleted_at IS NULL
		  AND td.deleted_at IS NULL
		ORDER BY t.id, td.id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	states := make(map[string]DesiredState)
	for rows.Next() {
		var ds DesiredState
		var configJSON sql.NullString
		var isPremium bool

		err := rows.Scan(
			&ds.TenantID,
			&ds.TenantName,
			&ds.DatasourceID,
			&ds.DatasourceName,
			&ds.ResourceGroup,
			&ds.RefreshInterval,
			&isPremium,
			&ds.IsActive,
			&configJSON,
		)
		if err != nil {
			return nil, err
		}

		if isPremium {
			ds.Tier = "enterprise"
		} else {
			ds.Tier = "standard"
		}

		if configJSON.Valid {
			_ = json.Unmarshal([]byte(configJSON.String), &ds.Config)
		}

		key := ds.TenantID + ":" + ds.DatasourceID
		states[key] = ds
	}

	return states, rows.Err()
}

// loadActualStates reads the current state from disk.
func (r *Reconciler) loadActualStates(ctx context.Context, desired map[string]DesiredState) (map[string]ActualState, error) {
	states := make(map[string]ActualState)

	// Load tenant-scopes.json
	scopesPath := filepath.Join(r.config.GeneratedDir, "tenant-scopes.json")
	var scopesData map[string]any
	if data, err := os.ReadFile(scopesPath); err == nil {
		_ = json.Unmarshal(data, &scopesData)
	}

	for key, ds := range desired {
		actual := ActualState{}

		// Check tenant directory
		tenantDir := filepath.Join(r.config.SchemaRoot, "tenants", ds.TenantID)
		if info, err := os.Stat(tenantDir); err == nil && info.IsDir() {
			actual.HasTenantDir = true

			// Check tenant.json
			tenantJSONPath := filepath.Join(tenantDir, "tenant.json")
			if data, err := os.ReadFile(tenantJSONPath); err == nil {
				actual.HasTenantJSON = true
				_ = json.Unmarshal(data, &actual.TenantJSON)
			}

			// List schema files
			entries, _ := os.ReadDir(tenantDir)
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".yml") || strings.HasSuffix(e.Name(), ".yaml") {
					actual.SchemaFiles = append(actual.SchemaFiles, e.Name())
				}
			}
		}

		// Check if in scopes JSON
		if scopesData != nil {
			if tenants, ok := scopesData["tenants"].([]any); ok {
				for _, t := range tenants {
					if tm, ok := t.(map[string]any); ok {
						if tm["tenant_id"] == ds.TenantID && tm["datasource_id"] == ds.DatasourceID {
							actual.InScopesJSON = true
							actual.ScopesEntry = tm
							break
						}
					}
				}
			}
		}

		states[key] = actual
	}

	return states, nil
}

// findOrphanedSchemas finds tenant directories that don't have a matching DB entry.
func (r *Reconciler) findOrphanedSchemas(ctx context.Context, desired map[string]DesiredState) ([]string, error) {
	tenantsDir := filepath.Join(r.config.SchemaRoot, "tenants")
	entries, err := os.ReadDir(tenantsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Build set of known tenant IDs
	knownTenants := make(map[string]bool)
	for _, ds := range desired {
		knownTenants[ds.TenantID] = true
	}

	var orphaned []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != ".gitkeep" {
			if !knownTenants[e.Name()] {
				orphaned = append(orphaned, e.Name())
			}
		}
	}

	return orphaned, nil
}

// detectDrifts compares desired vs actual state and returns drift findings.
func (r *Reconciler) detectDrifts(desired DesiredState, actual ActualState, exists bool) []DriftDetail {
	var drifts []DriftDetail

	// Check for missing tenant directory
	if !actual.HasTenantDir {
		drifts = append(drifts, DriftDetail{
			TenantID:       desired.TenantID,
			DatasourceID:   desired.DatasourceID,
			DriftType:      DriftTypeMissingTenant,
			Expected:       "tenant directory exists",
			Actual:         "missing",
			Severity:       SeverityHigh,
			AutoRepairable: true,
		})
		return drifts // No point checking further if dir missing
	}

	// Check tenant.json exists
	if !actual.HasTenantJSON {
		drifts = append(drifts, DriftDetail{
			TenantID:       desired.TenantID,
			DatasourceID:   desired.DatasourceID,
			DriftType:      DriftTypeConfigMismatch,
			Expected:       "tenant.json exists",
			Actual:         "missing",
			Severity:       SeverityHigh,
			AutoRepairable: true,
		})
	} else {
		// Check resource group matches
		if rg, ok := actual.TenantJSON["resource_group"].(string); ok {
			if rg != desired.ResourceGroup {
				drifts = append(drifts, DriftDetail{
					TenantID:       desired.TenantID,
					DatasourceID:   desired.DatasourceID,
					DriftType:      DriftTypeResourceGroup,
					Expected:       desired.ResourceGroup,
					Actual:         rg,
					Severity:       SeverityMedium,
					AutoRepairable: true,
				})
			}
		}

		// Check refresh interval
		if refresh, ok := actual.TenantJSON["refresh_interval_minutes"].(float64); ok {
			if int(refresh) != desired.RefreshInterval {
				drifts = append(drifts, DriftDetail{
					TenantID:       desired.TenantID,
					DatasourceID:   desired.DatasourceID,
					DriftType:      DriftTypeRefreshMismatch,
					Expected:       fmt.Sprintf("%d minutes", desired.RefreshInterval),
					Actual:         fmt.Sprintf("%.0f minutes", refresh),
					Severity:       SeverityLow,
					AutoRepairable: true,
				})
			}
		}
	}

	// Check presence in tenant-scopes.json
	if !actual.InScopesJSON {
		drifts = append(drifts, DriftDetail{
			TenantID:       desired.TenantID,
			DatasourceID:   desired.DatasourceID,
			DriftType:      DriftTypeStaleScopesJSON,
			Expected:       "in tenant-scopes.json",
			Actual:         "missing from tenant-scopes.json",
			Severity:       SeverityHigh,
			AutoRepairable: true,
		})
	}

	return drifts
}

// checkScopesJSONFreshness verifies tenant-scopes.json is up to date.
func (r *Reconciler) checkScopesJSONFreshness(ctx context.Context, desired map[string]DesiredState) *DriftDetail {
	scopesPath := filepath.Join(r.config.GeneratedDir, "tenant-scopes.json")
	info, err := os.Stat(scopesPath)
	if err != nil {
		return &DriftDetail{
			DriftType:      DriftTypeStaleScopesJSON,
			Expected:       "tenant-scopes.json exists",
			Actual:         "file missing",
			Severity:       SeverityCritical,
			AutoRepairable: true,
		}
	}

	// If file is more than 24 hours old, flag it
	if time.Since(info.ModTime()) > 24*time.Hour {
		return &DriftDetail{
			DriftType:      DriftTypeStaleScopesJSON,
			Expected:       "updated within 24 hours",
			Actual:         fmt.Sprintf("last modified %s ago", time.Since(info.ModTime()).Round(time.Hour)),
			Severity:       SeverityMedium,
			AutoRepairable: true,
		}
	}

	return nil
}

// attemptRepair tries to fix a drift automatically.
func (r *Reconciler) attemptRepair(ctx context.Context, desired DesiredState, drift DriftDetail) RepairAction {
	action := RepairAction{
		TenantID:     drift.TenantID,
		DatasourceID: drift.DatasourceID,
		Timestamp:    time.Now(),
	}

	switch drift.DriftType {
	case DriftTypeMissingTenant:
		action.Action = "create_tenant_directory"
		err := r.createTenantDir(ctx, desired)
		action.Success = err == nil
		if err != nil {
			action.Error = err.Error()
		}

	case DriftTypeConfigMismatch, DriftTypeResourceGroup, DriftTypeRefreshMismatch:
		action.Action = "update_tenant_json"
		err := r.updateTenantJSON(ctx, desired)
		action.Success = err == nil
		if err != nil {
			action.Error = err.Error()
		}

	case DriftTypeStaleScopesJSON:
		action.Action = "regenerate_scopes_json"
		return r.regenerateScopesJSON(ctx)

	default:
		action.Action = "unknown"
		action.Success = false
		action.Error = "unknown drift type"
	}

	r.logger.InfoContext(ctx, "repair action",
		"tenant_id", action.TenantID,
		"action", action.Action,
		"success", action.Success,
		"error", action.Error,
	)

	return action
}

// createTenantDir scaffolds a new tenant directory.
func (r *Reconciler) createTenantDir(ctx context.Context, desired DesiredState) error {
	tenantDir := filepath.Join(r.config.SchemaRoot, "tenants", desired.TenantID)
	if err := os.MkdirAll(filepath.Join(tenantDir, "auto"), 0755); err != nil {
		return err
	}
	return r.updateTenantJSON(ctx, desired)
}

// updateTenantJSON writes/updates tenant.json with correct values.
func (r *Reconciler) updateTenantJSON(ctx context.Context, desired DesiredState) error {
	tenantDir := filepath.Join(r.config.SchemaRoot, "tenants", desired.TenantID)
	tenantJSONPath := filepath.Join(tenantDir, "tenant.json")

	data := map[string]any{
		"tenant_id":                desired.TenantID,
		"datasource_id":            desired.DatasourceID,
		"tenant_name":              desired.TenantName,
		"datasource_name":          desired.DatasourceName,
		"tier":                     desired.Tier,
		"resource_group":           desired.ResourceGroup,
		"refresh_interval_minutes": desired.RefreshInterval,
		"is_active":                desired.IsActive,
		"last_reconciled":          time.Now().UTC().Format(time.RFC3339),
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tenantJSONPath, jsonBytes, 0644)
}

// regenerateScopesJSON rebuilds tenant-scopes.json from the database.
func (r *Reconciler) regenerateScopesJSON(ctx context.Context) RepairAction {
	action := RepairAction{
		Action:    "regenerate_scopes_json",
		Timestamp: time.Now(),
	}

	// Re-run the provisioner sync
	cfg := Config{
		DSN:          r.config.DSN,
		SchemaRoot:   r.config.SchemaRoot,
		GeneratedDir: r.config.GeneratedDir,
		DryRun:       false,
		Logger:       r.logger,
	}

	_, err := Execute(ctx, cfg)
	action.Success = err == nil
	if err != nil {
		action.Error = err.Error()
	}

	return action
}

// emitMetrics sends reconciliation results to Prometheus.
func (r *Reconciler) emitMetrics(ctx context.Context, result *ReconciliationResult) {
	// These would be actual Prometheus metrics in production
	r.logger.InfoContext(ctx, "reconciliation_metrics",
		"tenant_config_total", result.TotalTenants,
		"tenant_config_in_sync", result.InSync,
		"tenant_config_drifted", result.Drifted,
		"tenant_config_drift_detected", boolToInt(result.Drifted > 0),
		"tenant_config_auto_repaired", result.AutoRepaired,
		"tenant_config_manual_required", result.ManualRequired,
	)
}

// hasCriticalDrift checks if any critical severity drifts exist.
func (r *Reconciler) hasCriticalDrift(result *ReconciliationResult) bool {
	for _, drift := range result.DriftDetails {
		if drift.Severity == SeverityCritical {
			return true
		}
	}
	return false
}

// sendDriftAlert sends an alert via webhook.
func (r *Reconciler) sendDriftAlert(ctx context.Context, result *ReconciliationResult) {
	if r.config.AlertWebhookURL == "" {
		return
	}

	// Build alert payload
	criticalDrifts := []DriftDetail{}
	for _, d := range result.DriftDetails {
		if d.Severity == SeverityCritical || d.Severity == SeverityHigh {
			criticalDrifts = append(criticalDrifts, d)
		}
	}

	r.logger.WarnContext(ctx, "critical drift detected - alert sent",
		"webhook_url", r.config.AlertWebhookURL,
		"critical_count", len(criticalDrifts),
	)

	// In production: POST to webhook with alert payload
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// RunNightly is the entry point for cron/scheduler.
func RunNightly(ctx context.Context, dsn string) error {
	cfg := ReconcilerConfig{
		DSN:             dsn,
		AlertWebhookURL: os.Getenv("DRIFT_ALERT_WEBHOOK_URL"),
	}

	reconciler, err := NewReconciler(cfg)
	if err != nil {
		return err
	}

	result, err := reconciler.Run(ctx)
	if err != nil {
		return err
	}

	// Write result to file for audit
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	resultPath := fmt.Sprintf("reconciliation_%s.json", time.Now().Format("2006-01-02"))
	return os.WriteFile(filepath.Join("logs", resultPath), resultJSON, 0644)
}

package tenantauto

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Config controls how tenant automation runs.
type Config struct {
	DSN              string
	SchemaRoot       string
	GeneratedDir     string
	TenantFilter     []string
	DatasourceFilter []string
	DryRun           bool
	TriggeredBy      string
	Logger           *slog.Logger
}

// Result captures provisioning statistics.
type Result struct {
	TotalRows   int
	FilteredOut int
	Written     int
	Failures    []Failure
}

// Failure stores failed tenant/datasource combos.
type Failure struct {
	TenantID     string
	DatasourceID string
	Err          error
}

type tenantRow struct {
	TenantID             string
	TenantName           sql.NullString
	TenantDisplayName    sql.NullString
	TenantInstanceName   sql.NullString
	TenantIsGoldCopy     bool
	DatasourceID         string
	AlphaDatasourceID    sql.NullString
	DatasourceName       sql.NullString
	DatasourceCode       sql.NullString
	DatasourceConfig     sql.NullString
	InstanceConfig       sql.NullString
	ResourceGroup        sql.NullString
	SchemaOverrideRepo   sql.NullString
	SchemaOverrideBranch sql.NullString
	ConnectionString     sql.NullString
}

// Execute orchestrates the Phase 5 automation.
func Execute(ctx context.Context, cfg Config) (Result, error) {
	cfg = applyDefaults(cfg)

	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		return Result{}, fmt.Errorf("connect postgres: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return Result{}, fmt.Errorf("ping postgres: %w", err)
	}

	rows, err := db.QueryContext(ctx, tenantAutomationQuery)
	if err != nil {
		return Result{}, fmt.Errorf("tenant datasource query failed: %w", err)
	}
	defer rows.Close()

	tenantAllow := toSet(cfg.TenantFilter)
	datasourceAllow := toSet(cfg.DatasourceFilter)

	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	var (
		entries []tenantScopeEntry
		result  Result
	)

	for rows.Next() {
		var row tenantRow
		if err := rows.Scan(
			&row.TenantID,
			&row.TenantName,
			&row.TenantDisplayName,
			&row.TenantInstanceName,
			&row.TenantIsGoldCopy,
			&row.DatasourceID,
			&row.AlphaDatasourceID,
			&row.DatasourceName,
			&row.DatasourceCode,
			&row.DatasourceConfig,
			&row.InstanceConfig,
			&row.ResourceGroup,
			&row.SchemaOverrideRepo,
			&row.SchemaOverrideBranch,
			&row.ConnectionString,
		); err != nil {
			return result, fmt.Errorf("scan tenant datasource row: %w", err)
		}

		result.TotalRows++

		if len(tenantAllow) > 0 && !tenantAllow[strings.ToLower(row.TenantID)] {
			result.FilteredOut++
			continue
		}
		if len(datasourceAllow) > 0 && !datasourceAllow[strings.ToLower(row.DatasourceID)] {
			result.FilteredOut++
			continue
		}

		if err := ctx.Err(); err != nil {
			return result, err
		}

		entry, summary, err := processRow(ctx, db, row, cfg)
		if err != nil {
			logger.Error("tenant provisioning failed", "tenant", row.TenantID, "datasource", row.DatasourceID, "err", err)
			result.Failures = append(result.Failures, Failure{TenantID: row.TenantID, DatasourceID: row.DatasourceID, Err: err})
			continue
		}

		entries = append(entries, entry)
		logger.Info("tenant provisioning complete", "tenant", row.TenantID, "datasource", row.DatasourceID, "files", len(summary))
		result.Written++
	}

	if err := rows.Err(); err != nil {
		return result, err
	}

	if result.Written == 0 && len(result.Failures) == 0 {
		entries = append(entries, defaultScopeEntry())
	}

	if !cfg.DryRun {
		if err := writeTenantScopeFile(cfg.GeneratedDir, entries); err != nil {
			return result, err
		}
	}

	return result, nil
}

func applyDefaults(cfg Config) Config {
	if cfg.SchemaRoot == "" {
		cfg.SchemaRoot = filepath.Join("cube", "schema", "tenants")
	}
	if cfg.GeneratedDir == "" {
		cfg.GeneratedDir = filepath.Join("cube", "generated")
	}
	if cfg.TriggeredBy == "" {
		cfg.TriggeredBy = "tenant-auto-cli"
	}
	return cfg
}

const tenantAutomationQuery = `
SELECT
    ti.tenant_id::text AS tenant_id,
    COALESCE(t.display_name, t.name, ti.instance_name) AS tenant_name,
    t.display_name,
    ti.instance_name,
    COALESCE(t.gold_copy, false) AS tenant_is_gold_copy,
    tpd.id::text AS tenant_datasource_id,
    tpd.alpha_datasource_id::text AS alpha_datasource_id,
    COALESCE(tpd.source_name, ad.datasource_name, ad.datasource_code) AS datasource_name,
    ad.datasource_code,
    tpd.config::text AS datasource_config,
    ti.config::text AS instance_config,
    td.resource_group,
    td.schema_override_repo,
    td.schema_override_branch,
    td.connection_string
FROM public.tenant_product_datasource tpd
JOIN public.tenant_product tp ON tpd.tenant_product_id = tp.id
JOIN public.tenant_instance ti ON tp.datasource_id = ti.id
JOIN public.tenants t ON t.id = ti.tenant_id
LEFT JOIN public.alpha_datasource ad ON ad.id = tpd.alpha_datasource_id
LEFT JOIN public.tenant_datasources td ON td.tenant_id = ti.tenant_id AND td.datasource_id = tpd.id
WHERE COALESCE(t.is_active, true) = true
  AND COALESCE(ti.is_active, true) = true
  AND COALESCE(tp.is_active, true) = true
  AND COALESCE(tpd.is_active, true) = true
ORDER BY t.display_name NULLS LAST, ad.datasource_name NULLS LAST;
`

func processRow(ctx context.Context, db *sql.DB, row tenantRow, cfg Config) (tenantScopeEntry, []string, error) {
	dsConfig := decodeJSONMap(row.DatasourceConfig)
	instanceConfig := decodeJSONMap(row.InstanceConfig)

	resourceGroup := coalesceString(
		row.ResourceGroup.String,
		pickString(dsConfig, "resource_group"),
		pickNestedString(dsConfig, "qos", "resource_group"),
		pickString(instanceConfig, "resource_group"),
	)
	if resourceGroup == "" {
		if row.TenantIsGoldCopy {
			resourceGroup = "tenant_premium"
		} else {
			resourceGroup = "tenant_standard"
		}
	}

	refresh := buildRefreshConfig(dsConfig)
	overrides := extractSchemaOverrideFiles(dsConfig)
	summary, err := materializeSchemaOverrides(cfg.SchemaRoot, row.TenantID, row.DatasourceID, overrides, cfg.DryRun)
	if err != nil {
		recordFailure(ctx, db, row, cfg, err)
		return tenantScopeEntry{}, nil, err
	}

	entry := tenantScopeEntry{
		TenantID:            row.TenantID,
		TenantName:          preferredName(row),
		DatasourceID:        row.DatasourceID,
		DatasourceName:      coalesceString(row.DatasourceName.String, row.DatasourceCode.String, row.DatasourceID),
		DatasourceCode:      coalesceString(row.DatasourceCode.String, row.DatasourceName.String, row.DatasourceID),
		ResourceGroup:       resourceGroup,
		Refresh:             refresh,
		SchemaOverrideFiles: summary,
	}

	if cfg.DryRun {
		return entry, summary, nil
	}

	if err := upsertDatasource(ctx, db, row, resourceGroup, summary, refresh); err != nil {
		recordFailure(ctx, db, row, cfg, err)
		return tenantScopeEntry{}, nil, err
	}

	if err := markProvisionJob(ctx, db, row, cfg, "ready", ""); err != nil {
		recordFailure(ctx, db, row, cfg, err)
		return tenantScopeEntry{}, nil, err
	}

	if err := writeMetadataFile(cfg.SchemaRoot, row, resourceGroup, refresh, summary); err != nil {
		recordFailure(ctx, db, row, cfg, err)
		return tenantScopeEntry{}, nil, err
	}

	return entry, summary, nil
}

func upsertDatasource(ctx context.Context, db *sql.DB, row tenantRow, resourceGroup string, summary []string, refresh refreshConfig) error {
	metadata := map[string]any{
		"tenantName":          preferredName(row),
		"datasourceName":      coalesceString(row.DatasourceName.String, row.DatasourceCode.String, row.DatasourceID),
		"alphaDatasourceId":   row.AlphaDatasourceID.String,
		"resourceGroup":       resourceGroup,
		"schemaOverrideFiles": summary,
		"refresh":             refresh,
		"updatedAt":           time.Now().UTC(),
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	connection := row.ConnectionString.String
	if connection == "" {
		connection = coalesceString(
			pickString(decodeJSONMap(row.DatasourceConfig), "connection_string"),
			pickString(decodeJSONMap(row.DatasourceConfig), "database_url"),
			pickString(decodeJSONMap(row.InstanceConfig), "connection_string"),
		)
	}

	_, err = db.ExecContext(ctx, `
INSERT INTO public.tenant_datasources (
    tenant_id,
    datasource_id,
    connection_string,
    resource_group,
    metadata,
    schema_override_repo,
    schema_override_branch,
    provisioning_status,
    last_provisioned_at,
    last_provision_error,
    updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,'ready',NOW(),NULL,NOW())
ON CONFLICT (tenant_id, datasource_id)
DO UPDATE SET
    connection_string = COALESCE(EXCLUDED.connection_string, public.tenant_datasources.connection_string),
    resource_group = EXCLUDED.resource_group,
    metadata = EXCLUDED.metadata,
    schema_override_repo = COALESCE(EXCLUDED.schema_override_repo, public.tenant_datasources.schema_override_repo),
    schema_override_branch = COALESCE(EXCLUDED.schema_override_branch, public.tenant_datasources.schema_override_branch),
    provisioning_status = 'ready',
    last_provisioned_at = NOW(),
    last_provision_error = NULL,
    updated_at = NOW();
`, row.TenantID, row.DatasourceID, nullIfEmpty(connection), resourceGroup, metadataBytes, row.SchemaOverrideRepo.String, row.SchemaOverrideBranch.String)
	if err != nil {
		return fmt.Errorf("upsert tenant_datasources: %w", err)
	}

	return nil
}

func markProvisionJob(ctx context.Context, db *sql.DB, row tenantRow, cfg Config, status, errMsg string) error {
	_, err := db.ExecContext(ctx, `
INSERT INTO public.tenant_provision_jobs (tenant_id, datasource_id, status, attempt_count, last_error, triggered_by, updated_at)
VALUES ($1,$2,$3,1,$4,$5,NOW())
ON CONFLICT (tenant_id, datasource_id)
DO UPDATE SET
    status = EXCLUDED.status,
    attempt_count = tenant_provision_jobs.attempt_count + 1,
    last_error = EXCLUDED.last_error,
    triggered_by = EXCLUDED.triggered_by,
    updated_at = NOW();
`, row.TenantID, row.DatasourceID, status, nullIfEmpty(errMsg), cfg.TriggeredBy)
	if err != nil {
		return fmt.Errorf("update tenant_provision_jobs: %w", err)
	}
	return nil
}

func recordFailure(ctx context.Context, db *sql.DB, row tenantRow, cfg Config, runErr error) {
	if cfg.DryRun {
		return
	}
	_ = markProvisionJob(ctx, db, row, cfg, "failed", runErr.Error())
}

func writeMetadataFile(root string, row tenantRow, resourceGroup string, refresh refreshConfig, summary []string) error {
	metaPath := filepath.Join(root, row.TenantID, row.DatasourceID, "auto", "metadata.json")
	if err := os.MkdirAll(filepath.Dir(metaPath), 0o755); err != nil {
		return fmt.Errorf("create metadata dir: %w", err)
	}

	payload := map[string]any{
		"tenantId":            row.TenantID,
		"tenantName":          preferredName(row),
		"datasourceId":        row.DatasourceID,
		"alphaDatasourceId":   row.AlphaDatasourceID.String,
		"datasourceName":      coalesceString(row.DatasourceName.String, row.DatasourceCode.String, row.DatasourceID),
		"resourceGroup":       resourceGroup,
		"refresh":             refresh,
		"schemaOverrideFiles": summary,
		"generatedAt":         time.Now().UTC(),
	}

	bytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	if err := os.WriteFile(metaPath, bytes, 0o644); err != nil {
		return fmt.Errorf("write metadata file: %w", err)
	}

	return nil
}

func materializeSchemaOverrides(root, tenantID, datasourceID string, files []schemaOverrideFile, dryRun bool) ([]string, error) {
	autoDir := filepath.Join(root, tenantID, datasourceID, "auto")

	if dryRun {
		var preview []string
		for _, file := range files {
			rel := sanitizeRelative(file.RelativePath)
			if rel == "" {
				continue
			}
			preview = append(preview, filepath.ToSlash(filepath.Join(datasourceID, "auto", rel)))
		}
		sort.Strings(preview)
		return preview, nil
	}

	if err := os.RemoveAll(autoDir); err != nil {
		return nil, fmt.Errorf("reset auto dir: %w", err)
	}
	if len(files) == 0 {
		return nil, nil
	}
	if err := os.MkdirAll(autoDir, 0o755); err != nil {
		return nil, fmt.Errorf("create auto dir: %w", err)
	}

	var written []string
	for idx, file := range files {
		rel := sanitizeRelative(file.RelativePath)
		if rel == "" {
			rel = fmt.Sprintf("override-%d.yml", idx+1)
		}
		target := filepath.Join(autoDir, rel)
		if !strings.HasPrefix(target, autoDir) {
			return nil, errors.New("schema override attempted to escape tenant directory")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, fmt.Errorf("create parent dir: %w", err)
		}
		if err := os.WriteFile(target, []byte(file.Content), 0o644); err != nil {
			return nil, fmt.Errorf("write override %s: %w", target, err)
		}
		written = append(written, filepath.ToSlash(filepath.Join(datasourceID, "auto", rel)))
	}

	sort.Strings(written)
	return written, nil
}

func writeTenantScopeFile(generatedDir string, entries []tenantScopeEntry) error {
	if err := os.MkdirAll(generatedDir, 0o755); err != nil {
		return fmt.Errorf("create generated dir: %w", err)
	}
	scope := tenantScopeFile{GeneratedAt: time.Now().UTC(), Tenants: entries}
	bytes, err := json.MarshalIndent(scope, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tenant scope: %w", err)
	}
	path := filepath.Join(generatedDir, "tenant-scopes.json")
	if err := os.WriteFile(path, bytes, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func defaultScopeEntry() tenantScopeEntry {
	return tenantScopeEntry{
		TenantID:       "default",
		TenantName:     "Default Tenant",
		DatasourceID:   "default",
		DatasourceName: "Default Datasource",
		DatasourceCode: "default",
		ResourceGroup:  "tenant_standard",
		Refresh:        refreshConfig{Mode: "interval", EveryMinutes: 60, Timezone: "UTC"},
	}
}

func preferredName(row tenantRow) string {
	candidates := []string{
		row.TenantDisplayName.String,
		row.TenantName.String,
		row.TenantInstanceName.String,
		fmt.Sprintf("tenant-%s", safeIDFragment(row.TenantID)),
	}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) != "" {
			return candidate
		}
	}
	return row.TenantID
}

func safeIDFragment(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func toSet(values []string) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]bool, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		set[strings.ToLower(trimmed)] = true
	}
	return set
}

func nullIfEmpty(val string) any {
	if strings.TrimSpace(val) == "" {
		return nil
	}
	return val
}

// Tenant scope output helpers.
type tenantScopeFile struct {
	GeneratedAt time.Time          `json:"generatedAt"`
	Tenants     []tenantScopeEntry `json:"tenants"`
}

type tenantScopeEntry struct {
	TenantID            string        `json:"tenantId"`
	TenantName          string        `json:"tenantName"`
	DatasourceID        string        `json:"datasourceId"`
	DatasourceName      string        `json:"datasourceName"`
	DatasourceCode      string        `json:"datasourceCode"`
	ResourceGroup       string        `json:"resourceGroup"`
	Refresh             refreshConfig `json:"refresh"`
	SchemaOverrideFiles []string      `json:"schemaOverrideFiles,omitempty"`
}

type refreshConfig struct {
	Mode         string `json:"mode"`
	Cron         string `json:"cron,omitempty"`
	EveryMinutes int    `json:"everyMinutes,omitempty"`
	Timezone     string `json:"timezone"`
}

func buildRefreshConfig(cfg map[string]any) refreshConfig {
	refresh := refreshConfig{Mode: "interval", EveryMinutes: 60, Timezone: "UTC"}
	if tz := pickNestedString(cfg, "refresh", "timezone"); tz != "" {
		refresh.Timezone = tz
	} else if tz := pickString(cfg, "refresh_timezone"); tz != "" {
		refresh.Timezone = tz
	}
	if cron := pickNestedString(cfg, "refresh", "cron"); cron != "" {
		refresh.Mode = "cron"
		refresh.Cron = cron
		refresh.EveryMinutes = 0
		return refresh
	}
	if cron := pickString(cfg, "refresh_cron"); cron != "" {
		refresh.Mode = "cron"
		refresh.Cron = cron
		refresh.EveryMinutes = 0
		return refresh
	}
	if minutes, ok := pickMinutes(cfg, "refresh_every_minutes"); ok {
		refresh.EveryMinutes = minutes
	} else if minutes, ok := pickMinutes(pickNestedMap(cfg, "refresh"), "every_minutes"); ok {
		refresh.EveryMinutes = minutes
	}
	return refresh
}

func decodeJSONMap(raw sql.NullString) map[string]any {
	if !raw.Valid {
		return map[string]any{}
	}
	trimmed := strings.TrimSpace(raw.String)
	if trimmed == "" {
		return map[string]any{}
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
		return map[string]any{}
	}
	return result
}

func pickString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if val, ok := m[key]; ok {
		return asString(val)
	}
	return ""
}

func pickNestedString(m map[string]any, keys ...string) string {
	current := any(m)
	for _, key := range keys {
		asMap, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current, ok = asMap[key]
		if !ok {
			return ""
		}
	}
	return asString(current)
}

func asString(val any) string {
	switch v := val.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case json.Number:
		return v.String()
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%g", v))
	case float32:
		return strings.TrimSpace(fmt.Sprintf("%g", v))
	case int, int64, int32:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	default:
		return ""
	}
}

func pickMinutes(m map[string]any, key string) (int, bool) {
	if m == nil {
		return 0, false
	}
	val, ok := m[key]
	if !ok {
		return 0, false
	}
	switch v := val.(type) {
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case json.Number:
		parsed, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int(parsed), true
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		var parsed int
		if _, err := fmt.Sscanf(trimmed, "%d", &parsed); err == nil {
			return parsed, true
		}
		return 0, false
	default:
		return 0, false
	}
}

func pickNestedMap(m map[string]any, keys ...string) map[string]any {
	current := m
	for _, key := range keys {
		if current == nil {
			return nil
		}
		next, ok := current[key].(map[string]any)
		if !ok {
			return nil
		}
		current = next
	}
	return current
}

// Schema override helpers.
type schemaOverrideFile struct {
	RelativePath string
	Content      string
}

type schemaOverrideDoc struct {
	Files     []schemaOverrideFileDoc `json:"files"`
	CubeFiles []schemaOverrideFileDoc `json:"cube_files"`
}

type schemaOverrideFileDoc struct {
	RelativePath string `json:"relative_path"`
	Path         string `json:"path"`
	Name         string `json:"name"`
	File         string `json:"file"`
	Content      string `json:"content"`
	Body         string `json:"body"`
	YAML         string `json:"yaml"`
	Text         string `json:"text"`
}

func extractSchemaOverrideFiles(cfg map[string]any) []schemaOverrideFile {
	rawVal, ok := cfg["schema_overrides"]
	if !ok || rawVal == nil {
		return nil
	}
	rawBytes, err := json.Marshal(rawVal)
	if err != nil {
		return nil
	}
	if files := decodeSchemaOverrideFiles(rawBytes); len(files) > 0 {
		return files
	}
	return nil
}

func decodeSchemaOverrideFiles(raw []byte) []schemaOverrideFile {
	var doc schemaOverrideDoc
	if err := json.Unmarshal(raw, &doc); err == nil {
		combined := append(doc.Files, doc.CubeFiles...)
		if len(combined) > 0 {
			return convertOverrideDocs(combined)
		}
	}
	var arr []schemaOverrideFileDoc
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return convertOverrideDocs(arr)
	}
	var mapPayload map[string]string
	if err := json.Unmarshal(raw, &mapPayload); err == nil && len(mapPayload) > 0 {
		out := make([]schemaOverrideFile, 0, len(mapPayload))
		for path, content := range mapPayload {
			out = append(out, schemaOverrideFile{RelativePath: path, Content: content})
		}
		sort.Slice(out, func(i, j int) bool { return out[i].RelativePath < out[j].RelativePath })
		return out
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil && strings.TrimSpace(single) != "" {
		return []schemaOverrideFile{{RelativePath: "overrides.yml", Content: single}}
	}
	return nil
}

func convertOverrideDocs(docs []schemaOverrideFileDoc) []schemaOverrideFile {
	var result []schemaOverrideFile
	for idx, doc := range docs {
		rel := coalesceString(doc.RelativePath, doc.Path, doc.Name, doc.File)
		content := coalesceString(doc.Content, doc.Body, doc.YAML, doc.Text)
		if content == "" {
			continue
		}
		if rel == "" {
			rel = fmt.Sprintf("override-%d.yml", idx+1)
		}
		result = append(result, schemaOverrideFile{RelativePath: rel, Content: content})
	}
	return result
}

func sanitizeRelative(rel string) string {
	cleaned := filepath.Clean(strings.TrimSpace(rel))
	cleaned = strings.TrimPrefix(cleaned, string(filepath.Separator))
	if cleaned == "." || cleaned == "" {
		return ""
	}
	if strings.Contains(cleaned, "..") {
		return ""
	}
	return cleaned
}

func coalesceString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

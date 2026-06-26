//go:build ignore

package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

const (
	generatedDir   = "cube/generated"
	schemaRootDir  = "cube/schema/tenants"
	tenantScopeOut = "tenant-scopes.json"
)

var tenantDatasourceQuery = `
SELECT
    ti.tenant_id::text AS tenant_id,
    COALESCE(t.display_name, t.name, ti.instance_name) AS tenant_name,
    COALESCE(t.display_name, t.name) AS tenant_display_name,
    COALESCE(ti.instance_name, t.name) AS tenant_instance_name,
    COALESCE(t.gold_copy, false) AS tenant_is_gold_copy,
    tpd.id::text AS tenant_datasource_id,
    COALESCE(tpd.source_name, ad.datasource_name, ad.datasource_code) AS datasource_name,
    ad.datasource_code,
    tpd.config::text AS datasource_config,
    ti.config::text AS instance_config
FROM public.tenant_product_datasource tpd
JOIN public.tenant_product tp ON tpd.tenant_product_id = tp.id
JOIN public.tenant_instance ti ON tp.tenant_instance_id = ti.id
JOIN public.tenants t ON t.id = ti.tenant_id
JOIN public.alpha_datasource ad ON tpd.alpha_datasource_id = ad.id
WHERE COALESCE(t.is_active, true) = true
  AND COALESCE(ti.is_active, true) = true
  AND COALESCE(tp.is_active, true) = true
  AND COALESCE(tpd.is_active, true) = true
ORDER BY t.display_name NULLS LAST, ad.datasource_name NULLS LAST;
`

type tenantRow struct {
	TenantID            string
	TenantName          sql.NullString
	TenantDisplayName   sql.NullString
	TenantInstanceName  sql.NullString
	TenantIsGoldCopy    bool
	DatasourceID        string
	DatasourceName      sql.NullString
	DatasourceCode      sql.NullString
	DatasourceConfigRaw sql.NullString
	InstanceConfigRaw   sql.NullString
}

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
	Timezone     string `json:"timezone,omitempty"`
}

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

func main() {
	log.SetFlags(0)

	dsn := resolveDSN()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping postgres: %v", err)
	}

	rows, err := db.Query(tenantDatasourceQuery)
	if err != nil {
		log.Fatalf("tenant datasource query failed: %v", err)
	}
	defer rows.Close()

	var entries []tenantScopeEntry
	for rows.Next() {
		var row tenantRow
		if err := rows.Scan(
			&row.TenantID,
			&row.TenantName,
			&row.TenantDisplayName,
			&row.TenantInstanceName,
			&row.TenantIsGoldCopy,
			&row.DatasourceID,
			&row.DatasourceName,
			&row.DatasourceCode,
			&row.DatasourceConfigRaw,
			&row.InstanceConfigRaw,
		); err != nil {
			log.Fatalf("failed to scan tenant datasource row: %v", err)
		}

		dsConfig := decodeJSONMap(row.DatasourceConfigRaw)
		instanceConfig := decodeJSONMap(row.InstanceConfigRaw)

		resourceGroup := coalesceString(
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
		overrideFiles := extractSchemaOverrideFiles(dsConfig)
		summary, err := materializeSchemaOverrides(row.TenantID, row.DatasourceID, overrideFiles)
		if err != nil {
			log.Fatalf("failed to write schema overrides for tenant %s datasource %s: %v", row.TenantID, row.DatasourceID, err)
		}

		entries = append(entries, tenantScopeEntry{
			TenantID:            row.TenantID,
			TenantName:          preferredName(row),
			DatasourceID:        row.DatasourceID,
			DatasourceName:      coalesceString(row.DatasourceName.String, row.DatasourceCode.String, row.DatasourceID),
			DatasourceCode:      coalesceString(row.DatasourceCode.String, row.DatasourceName.String, row.DatasourceID),
			ResourceGroup:       resourceGroup,
			Refresh:             refresh,
			SchemaOverrideFiles: summary,
		})
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("tenant datasource iteration failed: %v", err)
	}

	if len(entries) == 0 {
		entries = append(entries, tenantScopeEntry{
			TenantID:       "default",
			TenantName:     "Default Tenant",
			DatasourceID:   "default",
			DatasourceName: "Default Datasource",
			DatasourceCode: "default",
			ResourceGroup:  "tenant_standard",
			Refresh: refreshConfig{
				Mode:         "interval",
				EveryMinutes: 60,
				Timezone:     "UTC",
			},
		})
		log.Printf("warning: no active tenant datasources found; wrote fallback scope entry")
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].TenantID == entries[j].TenantID {
			return entries[i].DatasourceID < entries[j].DatasourceID
		}
		return entries[i].TenantID < entries[j].TenantID
	})

	if err := os.MkdirAll(generatedDir, 0o755); err != nil {
		log.Fatalf("failed to create generated dir: %v", err)
	}

	scopePath := filepath.Join(generatedDir, tenantScopeOut)
	scopeBytes, err := json.MarshalIndent(tenantScopeFile{
		GeneratedAt: time.Now().UTC(),
		Tenants:     entries,
	}, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal tenant scope: %v", err)
	}

	if err := os.WriteFile(scopePath, scopeBytes, 0o644); err != nil {
		log.Fatalf("failed to write %s: %v", scopePath, err)
	}

	log.Printf("✅ synced %d tenant datasource records → %s", len(entries), scopePath)
}

func resolveDSN() string {
	if dsn := strings.TrimSpace(os.Getenv("DATABASE_URL")); dsn != "" {
		return dsn
	}
	if dsn := strings.TrimSpace(os.Getenv("ALPHA_DB_URL")); dsn != "" {
		return dsn
	}
	return "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
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
		log.Printf("warning: failed to parse JSON config: %v", err)
		return map[string]any{}
	}
	return result
}

func preferredName(row tenantRow) string {
	choices := []string{
		row.TenantDisplayName.String,
		row.TenantName.String,
		row.TenantInstanceName.String,
		fmt.Sprintf("tenant-%s", safeIDFragment(row.TenantID)),
	}
	for _, c := range choices {
		if strings.TrimSpace(c) != "" {
			return c
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

func coalesceString(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func pickString(m map[string]any, key string) string {
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

func buildRefreshConfig(cfg map[string]any) refreshConfig {
	const defaultMinutes = 60
	refresh := refreshConfig{
		Mode:         "interval",
		EveryMinutes: defaultMinutes,
		Timezone:     "UTC",
	}

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

func extractSchemaOverrideFiles(cfg map[string]any) []schemaOverrideFile {
	rawVal, ok := cfg["schema_overrides"]
	if !ok || rawVal == nil {
		return nil
	}

	rawBytes, err := json.Marshal(rawVal)
	if err != nil {
		log.Printf("warning: failed to marshal schema_overrides for writing: %v", err)
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

func materializeSchemaOverrides(tenantID, datasourceID string, files []schemaOverrideFile) ([]string, error) {
	autoDir := filepath.Join(schemaRootDir, tenantID, datasourceID, "auto")

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
		targetPath := filepath.Join(autoDir, rel)
		if !strings.HasPrefix(targetPath, autoDir) {
			return nil, errors.New("refusing to write schema override outside tenant directory")
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return nil, fmt.Errorf("create parent dir: %w", err)
		}
		if err := os.WriteFile(targetPath, []byte(file.Content), 0o644); err != nil {
			return nil, fmt.Errorf("write override %s: %w", targetPath, err)
		}
		relSummary := filepath.ToSlash(filepath.Join(datasourceID, "auto", rel))
		written = append(written, relSummary)
	}

	sort.Strings(written)
	return written, nil
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

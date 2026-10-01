package querybuilder

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// MetricBundleSchemaVersion is the official schema version identifier for metric bundles.
const MetricBundleSchemaVersion = "uisce.metric-bundle/1"

var (
	ErrGrainNotAllowed    = errors.New("requested grouping grain is not in the metric grain allowlist")
	ErrMetricInUse        = errors.New("metric is currently referenced and cannot be deleted")
	ErrCoreMetricImmutable = errors.New("core metrics are managed by the master tenant and cannot be deleted by client tenants")
)

// MetricFormatConfig specifies how metric values are rendered.
type MetricFormatConfig struct {
	Type           string  `json:"type"` // "currency" | "percentage" | "compact" | "number"
	Precision      *int    `json:"precision,omitempty"`
	CurrencySymbol *string `json:"currencySymbol,omitempty"`
	Prefix         *string `json:"prefix,omitempty"`
	Suffix         *string `json:"suffix,omitempty"`
}

// MetricExpression defines the computation rules for a metric.
type MetricExpression struct {
	Kind          string   `json:"kind"` // "aggregation" | "formula" | "derived"
	Fn            string   `json:"fn,omitempty"` // "sum" | "avg" | "count" | "min" | "max"
	TermNodeID    string   `json:"termNodeId,omitempty"`
	Formula       string   `json:"formula,omitempty"`
	BaseMetricIDs []string `json:"baseMetricIds,omitempty"` // For derived metrics
}

// MetricDefinition represents a row in data_explorer.metric_definition.
type MetricDefinition struct {
	ID             string             `json:"id" db:"id"`
	TenantID       string             `json:"tenantId" db:"tenant_id"`
	Name           string             `json:"name" db:"name"`
	Description    string             `json:"description" db:"description"`
	BOID           string             `json:"boId" db:"bo_id"`
	Expression     MetricExpression   `json:"expression"`
	GrainAllowlist []string           `json:"grainAllowlist"`
	FormatConfig   MetricFormatConfig `json:"formatConfig"`
	Tags           []string           `json:"tags"`
	IsCore         bool               `json:"isCore" db:"is_core"`
	Status         string             `json:"status" db:"status"` // "active" | "deprecated" | "archived"
	ArchivedAt     *time.Time         `json:"archivedAt,omitempty" db:"archived_at"`
	CreatedBy      *string            `json:"createdBy,omitempty" db:"created_by"`
	CreatedAt      time.Time          `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time          `json:"updatedAt" db:"updated_at"`
}

type metricDefRow struct {
	ID             string         `db:"id"`
	TenantID       string         `db:"tenant_id"`
	Name           string         `db:"name"`
	Description    sql.NullString `db:"description"`
	BOID           string         `db:"bo_id"`
	Expression     []byte         `db:"expression"`
	GrainAllowlist []byte         `db:"grain_allowlist"`
	FormatConfig   []byte         `db:"format_config"`
	Tags           pq.StringArray `db:"tags"`
	IsCore         bool           `db:"is_core"`
	Status         string         `db:"status"`
	ArchivedAt     pq.NullTime    `db:"archived_at"`
	CreatedBy      sql.NullString `db:"created_by"`
	CreatedAt      time.Time      `db:"created_at"`
	UpdatedAt      time.Time      `db:"updated_at"`
}

func (r metricDefRow) toMetricDefinition() MetricDefinition {
	var expr MetricExpression
	_ = json.Unmarshal(r.Expression, &expr)

	var grains []string
	if len(r.GrainAllowlist) > 0 {
		_ = json.Unmarshal(r.GrainAllowlist, &grains)
	}

	var fmtCfg MetricFormatConfig
	if len(r.FormatConfig) > 0 {
		_ = json.Unmarshal(r.FormatConfig, &fmtCfg)
	}

	m := MetricDefinition{
		ID:             r.ID,
		TenantID:       r.TenantID,
		Name:           r.Name,
		Description:    r.Description.String,
		BOID:           r.BOID,
		Expression:     expr,
		GrainAllowlist: grains,
		FormatConfig:   fmtCfg,
		Tags:           []string(r.Tags),
		IsCore:         r.IsCore,
		Status:         r.Status,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
	if r.ArchivedAt.Valid {
		m.ArchivedAt = &r.ArchivedAt.Time
	}
	if r.CreatedBy.Valid {
		m.CreatedBy = &r.CreatedBy.String
	}
	return m
}

// ValidateGrains verifies that requested grouping grains are permitted by grainAllowlist.
func ValidateGrains(requestedGrains []string, grainAllowlist []string) error {
	if len(grainAllowlist) == 0 {
		// Empty allowlist means all grains allowed
		return nil
	}
	allowed := make(map[string]bool)
	for _, g := range grainAllowlist {
		allowed[strings.ToLower(strings.TrimSpace(g))] = true
	}

	for _, req := range requestedGrains {
		normalized := strings.ToLower(strings.TrimSpace(req))
		if !allowed[normalized] {
			return fmt.Errorf("%w: '%s' (allowed: %v)", ErrGrainNotAllowed, req, grainAllowlist)
		}
	}
	return nil
}

// MetricReference describes a location where a metric definition is used.
type MetricReference struct {
	Type     string `json:"type"` // "saved_query" | "kpi_tile" | "derived_metric" | "core_adoption"
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	Location string `json:"location,omitempty"`
}

// MetricUsageReport is returned by GET /api/explorer/metrics/{id}/usage and on 409 conflict.
type MetricUsageReport struct {
	InUse      bool              `json:"inUse"`
	References []MetricReference `json:"references"`
}

// ScanMetricUsage scans for references to metricID in saved queries, page components, and derived metrics.
func ScanMetricUsage(ctx context.Context, db *sqlx.DB, metricID string, isCore bool, tenantID string) (MetricUsageReport, error) {
	var refs []MetricReference

	if db == nil {
		return MetricUsageReport{InUse: false, References: refs}, nil
	}

	// 1. Scan saved queries (query_state contains metricId)
	var sqRows []struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	sqQuery := `SELECT id, name FROM data_explorer.saved_query WHERE query_state::text LIKE '%' || $1 || '%' AND archived_at IS NULL`
	var sqArgs []interface{}
	sqArgs = append(sqArgs, metricID)
	if !isCore && tenantID != "" {
		sqQuery += ` AND tenant_id = $2`
		sqArgs = append(sqArgs, tenantID)
	}
	if err := db.SelectContext(ctx, &sqRows, sqQuery, sqArgs...); err == nil {
		for _, q := range sqRows {
			refs = append(refs, MetricReference{
				Type:     "saved_query",
				ID:       q.ID,
				Name:     q.Name,
				Location: "saved_query.measures",
			})
		}
	}

	// 2. Scan page definitions (components containing metricId)
	var pgRows []struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	pgQuery := `SELECT id, name FROM page_definitions WHERE components::text LIKE '%' || $1 || '%'`
	var pgArgs []interface{}
	pgArgs = append(pgArgs, metricID)
	if !isCore && tenantID != "" {
		pgQuery += ` AND tenant_id = $2`
		pgArgs = append(pgArgs, tenantID)
	}
	if err := db.SelectContext(ctx, &pgRows, pgQuery, pgArgs...); err == nil {
		for _, p := range pgRows {
			refs = append(refs, MetricReference{
				Type:     "kpi_tile",
				ID:       p.ID,
				Name:     p.Name,
				Location: "page_definitions.components",
			})
		}
	}

	// 3. Scan derived metrics that depend on this metric
	var derRows []struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	derQuery := `SELECT id, name FROM data_explorer.metric_definition WHERE id != $1 AND expression::text LIKE '%' || $1 || '%' AND archived_at IS NULL`
	var derArgs []interface{}
	derArgs = append(derArgs, metricID)
	if !isCore && tenantID != "" {
		derQuery += ` AND tenant_id = $2`
		derArgs = append(derArgs, tenantID)
	}
	if err := db.SelectContext(ctx, &derRows, derQuery, derArgs...); err == nil {
		for _, d := range derRows {
			refs = append(refs, MetricReference{
				Type:     "derived_metric",
				ID:       d.ID,
				Name:     d.Name,
				Location: "metric_definition.expression.baseMetricIds",
			})
		}
	}

	// 4. Core adoptions if isCore
	if isCore {
		var adoptRows []struct {
			TenantID string `db:"tenant_id"`
			Count    int    `db:"count"`
		}
		adoptQuery := `SELECT tenant_id, COUNT(*) as count FROM public.core_object_adoption WHERE object_type = 'metric' AND core_object_id = $1 GROUP BY tenant_id`
		if err := db.SelectContext(ctx, &adoptRows, adoptQuery, metricID); err == nil {
			for _, a := range adoptRows {
				refs = append(refs, MetricReference{
					Type:     "core_adoption",
					ID:       a.TenantID,
					Name:     fmt.Sprintf("Adopted by tenant %s (%d times)", a.TenantID, a.Count),
					Location: "core_object_adoption",
				})
			}
		}
	}

	return MetricUsageReport{
		InUse:      len(refs) > 0,
		References: refs,
	}, nil
}

// MetricBundle represents the export payload for uisce.metric-bundle/1.
type MetricBundle struct {
	SchemaVersion string             `json:"schemaVersion"`
	ExportedAt    time.Time          `json:"exportedAt"`
	Metrics       []MetricDefinition `json:"metrics"`
}

// ExportMetricBundle serializes metrics into a metric bundle.
func ExportMetricBundle(metrics []MetricDefinition) (*MetricBundle, error) {
	return &MetricBundle{
		SchemaVersion: MetricBundleSchemaVersion,
		ExportedAt:    time.Now().UTC(),
		Metrics:       metrics,
	}, nil
}

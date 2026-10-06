package querybuilder

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/lib/pq"
)

// Cube impact report (A1) — composition + consumers. Dry-run only.

// CubeImpactMetric is a governed metric referenced by the cube.
type CubeImpactMetric struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
}

// CubeImpactPhysicalGrain is one pre_aggregation catalog grain for the cube.
type CubeImpactPhysicalGrain struct {
	NodeName            string   `json:"nodeName,omitempty"`
	GrainHash           string   `json:"grainHash,omitempty"`
	Grain               []string `json:"grain,omitempty"`
	LifecycleStatus     string   `json:"lifecycleStatus,omitempty"`
	IcebergTable        string   `json:"icebergTable,omitempty"`
	DualCommitWatermark *string  `json:"dualCommitWatermark,omitempty"`
	AttemptID           string   `json:"attemptId,omitempty"`
	ContractVersion     int      `json:"contractVersion,omitempty"`
}

// CubeImpactComposition is the bill of materials for a cube.
type CubeImpactComposition struct {
	CubeID          string                      `json:"cubeId"`
	Name            string                      `json:"name"`
	Status          string                      `json:"status"`
	ContractVersion int                         `json:"contractVersion"`
	ContentHash     string                      `json:"contentHash"`
	IsCore          bool                        `json:"isCore"`
	BOID            string                      `json:"boId"`
	Dimensions      []CubeDimension             `json:"dimensions"`
	TimeDimension   *CubeTimeDimension          `json:"timeDimension,omitempty"`
	Metrics         []CubeImpactMetric          `json:"metrics"`
	Grains          [][]string                  `json:"grains"`
	Federation      CubeFederation              `json:"federation"`
	Materialization CubeMaterializationConfig   `json:"materialization"`
	Physical        *CubeImpactPhysical         `json:"physical,omitempty"`
}

// CubeImpactPhysical groups physical grain rows.
type CubeImpactPhysical struct {
	Grains []CubeImpactPhysicalGrain `json:"grains"`
}

// CubeImpactConsumer is one outbound usage of the cube.
type CubeImpactConsumer struct {
	Kind     string `json:"kind"` // saved_query | page_tile | schedule | pipeline | report
	ID       string `json:"id"`
	Label    string `json:"label"`
	Href     string `json:"href,omitempty"`
	Severity string `json:"severity"` // info | warning | blocking
	Blocking bool   `json:"blocking"`
	Detail   string `json:"detail,omitempty"`
	Enabled  *bool  `json:"enabled,omitempty"`
	Pin      *struct {
		ContractVersion interface{} `json:"contractVersion,omitempty"`
	} `json:"pin,omitempty"`
}

// CubeImpactSummary rolls up consumer counts.
type CubeImpactSummary struct {
	ConsumerCount      int `json:"consumerCount"`
	BlockingCount      int `json:"blockingCount"`
	WarningCount       int `json:"warningCount"`
	PhysicalGrainCount int `json:"physicalGrainCount"`
}

// CubeImpactReport is the GET /api/cubes/{id}/impact payload.
type CubeImpactReport struct {
	CubeID       string                `json:"cubeId"`
	Composition  CubeImpactComposition `json:"composition"`
	Consumers    []CubeImpactConsumer  `json:"consumers"`
	Summary      CubeImpactSummary     `json:"summary"`
}

// BuildCubeImpact loads composition + consumers for a cube visible to tenantID.
func (h *CubeHandler) BuildCubeImpact(ctx context.Context, tenantID, cubeID string, includePhysical bool) (*CubeImpactReport, error) {
	cube, err := h.getByID(ctx, tenantID, cubeID)
	if err != nil {
		return nil, err
	}

	metrics, err := h.loadMetrics(ctx, tenantID, cube.MetricIDs)
	if err != nil {
		return nil, fmt.Errorf("load metrics: %w", err)
	}
	metricRows := make([]CubeImpactMetric, 0, len(cube.MetricIDs))
	for _, id := range cube.MetricIDs {
		norm := strings.ToLower(strings.TrimSpace(id))
		row := CubeImpactMetric{ID: id}
		if m, ok := metrics[norm]; ok {
			row.Name = m.Name
			row.Status = m.Status
		}
		metricRows = append(metricRows, row)
	}

	comp := CubeImpactComposition{
		CubeID:          cube.ID,
		Name:            cube.Name,
		Status:          cube.Status,
		ContractVersion: cube.ContractVersion,
		ContentHash:     cube.ContentHash,
		IsCore:          cube.IsCore,
		BOID:            cube.BOID,
		Dimensions:      cube.Dimensions,
		TimeDimension:   cube.TimeDimension,
		Metrics:         metricRows,
		Grains:          cube.Grains,
		Federation:      cube.Federation,
		Materialization: cube.Materialization,
	}

	var physicalCount int
	if includePhysical {
		grains, perr := h.listCubePhysicalGrains(ctx, cube.ID)
		if perr != nil {
			return nil, fmt.Errorf("list physical grains: %w", perr)
		}
		comp.Physical = &CubeImpactPhysical{Grains: grains}
		physicalCount = len(grains)
	}

	consumers, err := h.listCubeConsumers(ctx, tenantID, cube)
	if err != nil {
		return nil, fmt.Errorf("list consumers: %w", err)
	}

	summary := CubeImpactSummary{
		ConsumerCount:      len(consumers),
		PhysicalGrainCount: physicalCount,
	}
	for _, c := range consumers {
		if c.Blocking {
			summary.BlockingCount++
		}
		if c.Severity == "warning" {
			summary.WarningCount++
		}
	}

	return &CubeImpactReport{
		CubeID:      cube.ID,
		Composition: comp,
		Consumers:   consumers,
		Summary:     summary,
	}, nil
}

func (h *CubeHandler) listCubePhysicalGrains(ctx context.Context, cubeID string) ([]CubeImpactPhysicalGrain, error) {
	type row struct {
		NodeName   string          `db:"node_name"`
		Properties json.RawMessage `db:"properties"`
	}
	var rows []row
	err := h.db.SelectContext(ctx, &rows, `
		SELECT node_name, COALESCE(properties, '{}'::jsonb) AS properties
		FROM catalog_node
		WHERE properties->>'cube_id' = $1
		ORDER BY node_name
	`, cubeID)
	if err != nil {
		return nil, err
	}
	out := make([]CubeImpactPhysicalGrain, 0, len(rows))
	for _, rw := range rows {
		var props map[string]interface{}
		_ = json.Unmarshal(rw.Properties, &props)
		g := CubeImpactPhysicalGrain{NodeName: rw.NodeName}
		if v, ok := props["grain_hash"].(string); ok {
			g.GrainHash = v
		}
		if v, ok := props["lifecycle_status"].(string); ok {
			g.LifecycleStatus = v
		}
		if v, ok := props["iceberg_table"].(string); ok {
			g.IcebergTable = v
		}
		if v, ok := props["attempt_id"].(string); ok {
			g.AttemptID = v
		}
		if v, ok := props["contract_version"].(float64); ok {
			g.ContractVersion = int(v)
		}
		if arr, ok := props["grain"].([]interface{}); ok {
			for _, x := range arr {
				if s, ok := x.(string); ok {
					g.Grain = append(g.Grain, s)
				}
			}
		}
		if v, ok := props["dual_commit_watermark"].(string); ok && v != "" {
			g.DualCommitWatermark = &v
		}
		out = append(out, g)
	}
	return out, nil
}

func (h *CubeHandler) listCubeConsumers(ctx context.Context, tenantID string, cube *CubeDefinition) ([]CubeImpactConsumer, error) {
	var out []CubeImpactConsumer

	sqs, err := h.listSavedQueryConsumers(ctx, tenantID, cube.ID)
	if err != nil {
		return nil, err
	}
	out = append(out, sqs...)

	pages, err := h.listPageTileConsumers(ctx, tenantID, cube.ID, sqs)
	if err != nil {
		return nil, err
	}
	out = append(out, pages...)

	scheds, err := h.listScheduleConsumers(ctx, tenantID, cube.ID)
	if err != nil {
		return nil, err
	}
	out = append(out, scheds...)

	pipes, err := h.listPipelineConsumers(ctx, tenantID, cube.ID)
	if err != nil {
		return nil, err
	}
	out = append(out, pipes...)

	reps, err := h.listReportConsumers(ctx, tenantID, cube)
	if err != nil {
		return nil, err
	}
	out = append(out, reps...)

	return out, nil
}

func (h *CubeHandler) listSavedQueryConsumers(ctx context.Context, tenantID, cubeID string) ([]CubeImpactConsumer, error) {
	type row struct {
		ID         string         `db:"id"`
		Name       string         `db:"name"`
		Status     string         `db:"status"`
		QueryState []byte         `db:"query_state"`
		ArchivedAt sql.NullTime   `db:"archived_at"`
	}
	var rows []row
	err := h.db.SelectContext(ctx, &rows, `
		SELECT id::text, name, COALESCE(status, 'active') AS status, query_state, archived_at
		FROM data_explorer.saved_query
		WHERE tenant_id = $1
		  AND source_kind = 'cube'
		  AND source_id = $2
		  AND archived_at IS NULL
		ORDER BY name
	`, tenantID, cubeID)
	if err != nil {
		return nil, err
	}
	out := make([]CubeImpactConsumer, 0, len(rows))
	for _, rw := range rows {
		c := CubeImpactConsumer{
			Kind:     "saved_query",
			ID:       rw.ID,
			Label:    rw.Name,
			Href:     "/query-builder?savedQueryId=" + rw.ID,
			Severity: "blocking",
			Blocking: true,
			Detail:   "source_kind=cube",
		}
		if pin := subjectPinFromQueryState(rw.QueryState); pin != nil {
			c.Pin = pin
			c.Detail = fmt.Sprintf("pinned contractVersion=%v", pin.ContractVersion)
		}
		if rw.Status != "" && rw.Status != "active" {
			c.Severity = "warning"
			c.Blocking = false
			c.Detail = "status=" + rw.Status
		}
		out = append(out, c)
	}
	return out, nil
}

func subjectPinFromQueryState(raw []byte) *struct {
	ContractVersion interface{} `json:"contractVersion,omitempty"`
} {
	if len(raw) == 0 {
		return nil
	}
	var state struct {
		Subject *struct {
			Kind            string      `json:"kind"`
			CubeID          string      `json:"cubeId"`
			ContractVersion interface{} `json:"contractVersion"`
		} `json:"subject"`
	}
	if err := json.Unmarshal(raw, &state); err != nil || state.Subject == nil {
		return nil
	}
	if state.Subject.Kind != "cube" {
		return nil
	}
	return &struct {
		ContractVersion interface{} `json:"contractVersion,omitempty"`
	}{ContractVersion: state.Subject.ContractVersion}
}

func (h *CubeHandler) listPageTileConsumers(ctx context.Context, tenantID, cubeID string, cubeSQs []CubeImpactConsumer) ([]CubeImpactConsumer, error) {
	sqIDs := make([]string, 0, len(cubeSQs))
	for _, sq := range cubeSQs {
		sqIDs = append(sqIDs, sq.ID)
	}

	type row struct {
		ID     string `db:"id"`
		Slug   string `db:"slug"`
		Name   string `db:"name"`
		Status string `db:"status"`
	}
	var rows []row
	// Text search is intentional for A1: page JSON shapes vary (components vs app_model).
	q := `
		SELECT id::text, slug, COALESCE(name, slug) AS name, status
		FROM page_definitions
		WHERE tenant_id = $1::uuid
		  AND (
		    COALESCE(components::text, '') LIKE '%' || $2 || '%'
		    OR COALESCE(app_model::text, '') LIKE '%' || $2 || '%'
		  )
	`
	args := []interface{}{tenantID, cubeID}
	if len(sqIDs) > 0 {
		q += ` OR (
		  tenant_id = $1::uuid AND (
		    COALESCE(components::text, '') LIKE ANY($3)
		    OR COALESCE(app_model::text, '') LIKE ANY($3)
		  )
		)`
		likes := make([]string, len(sqIDs))
		for i, id := range sqIDs {
			likes[i] = "%" + id + "%"
		}
		args = append(args, pq.Array(likes))
	}
	q += ` ORDER BY slug`
	if err := h.db.SelectContext(ctx, &rows, q, args...); err != nil {
		return nil, err
	}

	out := make([]CubeImpactConsumer, 0, len(rows))
	seen := map[string]bool{}
	for _, rw := range rows {
		if seen[rw.ID] {
			continue
		}
		seen[rw.ID] = true
		c := CubeImpactConsumer{
			Kind:     "page_tile",
			ID:       rw.ID,
			Label:    rw.Name + " (" + rw.Slug + ")",
			Href:     "/pages/" + rw.Slug,
			Severity: "blocking",
			Blocking: true,
			Detail:   "page references cube id or cube saved query",
		}
		if rw.Status != "published" {
			c.Severity = "warning"
			c.Blocking = false
			c.Detail = "status=" + rw.Status
		}
		out = append(out, c)
	}
	return out, nil
}

func (h *CubeHandler) listScheduleConsumers(ctx context.Context, tenantID, cubeID string) ([]CubeImpactConsumer, error) {
	type row struct {
		ID      string `db:"id"`
		Name    string `db:"name"`
		Enabled bool   `db:"enabled"`
	}
	var rows []row
	err := h.db.SelectContext(ctx, &rows, `
		SELECT id::text, name, enabled
		FROM schedules
		WHERE tenant_id = $1::uuid
		  AND deleted_at IS NULL
		  AND target_kind = 'cube_refresh'
		  AND target_ref = $2
		ORDER BY name
	`, tenantID, cubeID)
	if err != nil {
		return nil, err
	}
	out := make([]CubeImpactConsumer, 0, len(rows))
	for _, rw := range rows {
		en := rw.Enabled
		c := CubeImpactConsumer{
			Kind:     "schedule",
			ID:       rw.ID,
			Label:    rw.Name,
			Href:     "/schedules/" + rw.ID,
			Enabled:  &en,
			Severity: "blocking",
			Blocking: true,
			Detail:   "target_kind=cube_refresh",
		}
		if !rw.Enabled {
			c.Severity = "warning"
			c.Blocking = false
			c.Detail = "schedule disabled"
		}
		out = append(out, c)
	}
	return out, nil
}

func (h *CubeHandler) listPipelineConsumers(ctx context.Context, tenantID, cubeID string) ([]CubeImpactConsumer, error) {
	type row struct {
		ID       string `db:"id"`
		Name     string `db:"name"`
		IsActive bool   `db:"is_active"`
	}
	var rows []row
	err := h.db.SelectContext(ctx, &rows, `
		SELECT id::text, name, is_active
		FROM data_pipeline_definitions
		WHERE tenant_id = $1::uuid
		  AND dag_json::text LIKE '%' || $2 || '%'
		  AND dag_json::text LIKE '%cube_materialize%'
		ORDER BY name
	`, tenantID, cubeID)
	if err != nil {
		// Table may be absent in some test DBs — treat as empty.
		if strings.Contains(err.Error(), "does not exist") {
			return nil, nil
		}
		return nil, err
	}
	out := make([]CubeImpactConsumer, 0, len(rows))
	for _, rw := range rows {
		en := rw.IsActive
		c := CubeImpactConsumer{
			Kind:     "pipeline",
			ID:       rw.ID,
			Label:    rw.Name,
			Href:     "/data-pipelines/" + rw.ID,
			Enabled:  &en,
			Severity: "blocking",
			Blocking: true,
			Detail:   "cube_materialize node",
		}
		if !rw.IsActive {
			c.Severity = "warning"
			c.Blocking = false
			c.Detail = "pipeline inactive"
		}
		out = append(out, c)
	}
	return out, nil
}

func (h *CubeHandler) listReportConsumers(ctx context.Context, tenantID string, cube *CubeDefinition) ([]CubeImpactConsumer, error) {
	type row struct {
		ID          string `db:"id"`
		ReportKey   string `db:"report_key"`
		DisplayName string `db:"display_name"`
		LegacyCube  string `db:"legacy_cube"`
		SubjectID   string `db:"subject_cube_id"`
	}
	var rows []row
	err := h.db.SelectContext(ctx, &rows, `
		SELECT id::text,
		       report_key,
		       display_name,
		       COALESCE(definition #>> '{dataBindings,primary,cube}', '') AS legacy_cube,
		       COALESCE(definition #>> '{dataBindings,primary,subject,cubeId}', '') AS subject_cube_id
		FROM report_definitions
		WHERE tenant_id = $1::uuid
		  AND (
		    definition #>> '{dataBindings,primary,subject,cubeId}' = $2
		    OR definition #>> '{dataBindings,primary,cube}' = $3
		  )
		ORDER BY report_key
	`, tenantID, cube.ID, cube.Name)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return nil, nil
		}
		return nil, err
	}
	out := make([]CubeImpactConsumer, 0, len(rows))
	for _, rw := range rows {
		c := CubeImpactConsumer{
			Kind:  "report",
			ID:    rw.ID,
			Label: rw.DisplayName + " (" + rw.ReportKey + ")",
			Href:  "/reports/" + rw.ID,
		}
		if rw.SubjectID == cube.ID {
			c.Severity = "blocking"
			c.Blocking = true
			c.Detail = "subject.cubeId pin"
		} else {
			c.Severity = "warning"
			c.Blocking = false
			c.Detail = "legacy dataBindings.primary.cube name=" + rw.LegacyCube
		}
		out = append(out, c)
	}
	return out, nil
}

// HandleGetCubeImpact handles GET /api/cubes/{id}/impact?includePhysical=0|1
func (h *CubeHandler) HandleGetCubeImpact(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	id := chi.URLParam(r, "id")
	if strings.TrimSpace(id) == "" {
		h.writeError(w, fmt.Errorf("cube id is required"), http.StatusBadRequest)
		return
	}
	includePhysical := true
	if v := strings.TrimSpace(r.URL.Query().Get("includePhysical")); v != "" {
		includePhysical, _ = strconv.ParseBool(v)
	}
	report, err := h.BuildCubeImpact(r.Context(), secCtx.TenantID, id, includePhysical)
	if err == sql.ErrNoRows {
		h.writeError(w, fmt.Errorf("cube not found"), http.StatusNotFound)
		return
	}
	if err != nil {
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, http.StatusOK, report)
}

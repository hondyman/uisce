package analytics

import (
	"bytes"
	"text/template"
)

// PreAggTemplateData holds the data for rendering pre-aggregation SQL templates.
type PreAggTemplateData struct {
	Tenant     string
	Datasource string
	PreAggID   string
	GroupBy    []string
	Measures   []MeasureDef
	Filters    []FilterDef
}

// MeasureDef represents a measure with its SQL expression and alias.
type MeasureDef struct {
	Expression string
	Alias      string
}

// FilterDef represents a filter condition.
type FilterDef struct {
	Field string
	Op    string
	Value string
}

// TrinoIcebergRollupTemplate is the Go template for creating Iceberg rollup tables via Trino.
const TrinoIcebergRollupTemplate = `CREATE TABLE iceberg.{{.Tenant}}_analytics.agg_{{.Datasource}}__{{.PreAggID}} AS
SELECT
{{- range $i, $col := .GroupBy }}
  {{ $col }}{{ if lt $i (sub (len $.GroupBy) 1) }},{{ end }}
{{- end }}{{ if .Measures }},{{ end }}
{{- range $i, $m := .Measures }}
  {{ $m.Expression }} AS {{ $m.Alias }}{{ if lt $i (sub (len $.Measures) 1) }},{{ end }}
{{- end }}
FROM iceberg.{{.Tenant}}_analytics.fact_{{.Datasource}}
{{- if .Filters }}
WHERE
{{- range $i, $f := .Filters }}
  {{ $f.Field }} {{ $f.Op }} {{ $f.Value }}{{ if lt $i (sub (len $.Filters) 1) }} AND{{ end }}
{{- end }}
{{- end }}
GROUP BY
{{- range $i, $col := .GroupBy }}
  {{ $col }}{{ if lt $i (sub (len $.GroupBy) 1) }},{{ end }}
{{- end }}`

// StarRocksMVTemplate is the Go template for creating StarRocks Materialized Views.
const StarRocksMVTemplate = `CREATE MATERIALIZED VIEW mv_{{.Datasource}}__{{.PreAggID}}
BUILD IMMEDIATE
REFRESH ASYNC
AS
SELECT
{{- range $i, $col := .GroupBy }}
  {{ $col }}{{ if lt $i (sub (len $.GroupBy) 1) }},{{ end }}
{{- end }}{{ if .Measures }},{{ end }}
{{- range $i, $m := .Measures }}
  {{ $m.Expression }} AS {{ $m.Alias }}{{ if lt $i (sub (len $.Measures) 1) }},{{ end }}
{{- end }}
FROM fact_{{.Datasource}}
{{- if .Filters }}
WHERE
{{- range $i, $f := .Filters }}
  {{ $f.Field }} {{ $f.Op }} {{ $f.Value }}{{ if lt $i (sub (len $.Filters) 1) }} AND{{ end }}
{{- end }}
{{- end }}
GROUP BY
{{- range $i, $col := .GroupBy }}
  {{ $col }}{{ if lt $i (sub (len $.GroupBy) 1) }},{{ end }}
{{- end }}`

// StarRocksRefreshTemplate is the Go template for refreshing a StarRocks MV.
const StarRocksRefreshTemplate = `REFRESH MATERIALIZED VIEW mv_{{.Datasource}}__{{.PreAggID}}`

// StarRocksDropMVTemplate is the Go template for dropping a StarRocks MV.
const StarRocksDropMVTemplate = `DROP MATERIALIZED VIEW IF EXISTS mv_{{.Datasource}}__{{.PreAggID}}`

// TrinoDropRollupTemplate is the Go template for dropping an Iceberg rollup table.
const TrinoDropRollupTemplate = `DROP TABLE IF EXISTS iceberg.{{.Tenant}}_analytics.agg_{{.Datasource}}__{{.PreAggID}}`

// PreAggTemplateRenderer provides methods for rendering pre-aggregation SQL templates.
type PreAggTemplateRenderer struct {
	trinoRollup   *template.Template
	starRocksMV   *template.Template
	starRocksRefr *template.Template
	starRocksDrop *template.Template
	trinoDropRoll *template.Template
}

// templateFuncs provides helper functions for templates.
var templateFuncs = template.FuncMap{
	"sub": func(a, b int) int { return a - b },
}

// NewPreAggTemplateRenderer creates a new template renderer with all templates parsed.
func NewPreAggTemplateRenderer() (*PreAggTemplateRenderer, error) {
	trinoRollup, err := template.New("trino_rollup").Funcs(templateFuncs).Parse(TrinoIcebergRollupTemplate)
	if err != nil {
		return nil, err
	}

	starRocksMV, err := template.New("starrocks_mv").Funcs(templateFuncs).Parse(StarRocksMVTemplate)
	if err != nil {
		return nil, err
	}

	starRocksRefr, err := template.New("starrocks_refresh").Funcs(templateFuncs).Parse(StarRocksRefreshTemplate)
	if err != nil {
		return nil, err
	}

	starRocksDrop, err := template.New("starrocks_drop").Funcs(templateFuncs).Parse(StarRocksDropMVTemplate)
	if err != nil {
		return nil, err
	}

	trinoDropRoll, err := template.New("trino_drop").Funcs(templateFuncs).Parse(TrinoDropRollupTemplate)
	if err != nil {
		return nil, err
	}

	return &PreAggTemplateRenderer{
		trinoRollup:   trinoRollup,
		starRocksMV:   starRocksMV,
		starRocksRefr: starRocksRefr,
		starRocksDrop: starRocksDrop,
		trinoDropRoll: trinoDropRoll,
	}, nil
}

// RenderTrinoIcebergRollup renders the Trino CREATE TABLE statement for an Iceberg rollup.
func (r *PreAggTemplateRenderer) RenderTrinoIcebergRollup(data PreAggTemplateData) (string, error) {
	var buf bytes.Buffer
	if err := r.trinoRollup.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderStarRocksMV renders the CREATE MATERIALIZED VIEW statement for StarRocks.
func (r *PreAggTemplateRenderer) RenderStarRocksMV(data PreAggTemplateData) (string, error) {
	var buf bytes.Buffer
	if err := r.starRocksMV.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderStarRocksRefresh renders the REFRESH statement for a StarRocks MV.
func (r *PreAggTemplateRenderer) RenderStarRocksRefresh(data PreAggTemplateData) (string, error) {
	var buf bytes.Buffer
	if err := r.starRocksRefr.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderStarRocksDrop renders the DROP MATERIALIZED VIEW statement.
func (r *PreAggTemplateRenderer) RenderStarRocksDrop(data PreAggTemplateData) (string, error) {
	var buf bytes.Buffer
	if err := r.starRocksDrop.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderTrinoDropRollup renders the DROP TABLE statement for an Iceberg rollup.
func (r *PreAggTemplateRenderer) RenderTrinoDropRollup(data PreAggTemplateData) (string, error) {
	var buf bytes.Buffer
	if err := r.trinoDropRoll.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

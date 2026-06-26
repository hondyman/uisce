package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"
)

// HasuraMetadataGenerator generates Hasura metadata from business object definitions
type HasuraMetadataGenerator struct {
	hasuraEndpoint string
	adminSecret    string
}

func NewHasuraMetadataGenerator(endpoint, secret string) *HasuraMetadataGenerator {
	return &HasuraMetadataGenerator{
		hasuraEndpoint: endpoint,
		adminSecret:    secret,
	}
}

// GenerateAndApply generates Hasura metadata for a business object and applies it
func (g *HasuraMetadataGenerator) GenerateAndApply(ctx context.Context, bo *BusinessObjectDefinition) error {
	// Generate table tracking
	trackTable := g.generateTrackTable(bo)

	// Generate relationships
	relationships := g.generateRelationships(bo)

	// Generate permissions (RLS)
	permissions := g.generatePermissions(bo)

	// Apply to Hasura
	if err := g.applyMetadata(ctx, trackTable); err != nil {
		return fmt.Errorf("failed to track table: %w", err)
	}

	for _, rel := range relationships {
		if err := g.applyMetadata(ctx, rel); err != nil {
			return fmt.Errorf("failed to create relationship: %w", err)
		}
	}

	for _, perm := range permissions {
		if err := g.applyMetadata(ctx, perm); err != nil {
			return fmt.Errorf("failed to create permission: %w", err)
		}
	}

	return nil
}

func (g *HasuraMetadataGenerator) generateTrackTable(bo *BusinessObjectDefinition) map[string]any {
	tableName := fmt.Sprintf("t_%s_%s", bo.TenantID, strings.ToLower(bo.Name))

	return map[string]any{
		"type": "pg_track_table",
		"args": map[string]any{
			"table": map[string]string{
				"schema": "public",
				"name":   tableName,
			},
		},
	}
}

func (g *HasuraMetadataGenerator) generateRelationships(bo *BusinessObjectDefinition) []map[string]any {
	var relationships []map[string]any

	// Generate relationships based on ref fields
	for _, field := range bo.Fields {
		if field.Type == FieldRef && field.RefObjectID != nil {
			relationships = append(relationships, map[string]any{
				"type": "pg_create_object_relationship",
				"args": map[string]any{
					"table": fmt.Sprintf("t_%s_%s", bo.TenantID, strings.ToLower(bo.Name)),
					"name":  field.Name,
					"using": map[string]any{
						"foreign_key_constraint_on": field.Name + "_id",
					},
				},
			})
		}
	}

	return relationships
}

func (g *HasuraMetadataGenerator) generatePermissions(bo *BusinessObjectDefinition) []map[string]any {
	tableName := fmt.Sprintf("t_%s_%s", bo.TenantID, strings.ToLower(bo.Name))

	return []map[string]any{
		{
			"type": "pg_create_select_permission",
			"args": map[string]any{
				"table": tableName,
				"role":  "user",
				"permission": map[string]any{
					"columns": "*",
					"filter": map[string]any{
						"tenant_id": map[string]string{
							"_eq": "X-Hasura-Tenant-Id",
						},
					},
				},
			},
		},
		{
			"type": "pg_create_insert_permission",
			"args": map[string]any{
				"table": tableName,
				"role":  "user",
				"permission": map[string]any{
					"columns": "*",
					"check": map[string]any{
						"tenant_id": map[string]string{
							"_eq": "X-Hasura-Tenant-Id",
						},
					},
				},
			},
		},
	}
}

func (g *HasuraMetadataGenerator) applyMetadata(ctx context.Context, metadata map[string]any) error {
	payload, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		g.hasuraEndpoint+"/v1/metadata",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hasura-Admin-Secret", g.adminSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hasura metadata API error: %d", resp.StatusCode)
	}

	return nil
}

// SQL migration template
const migrationTemplate = `
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
`

// GenerateMigration generates SQL migration for a business object
func (g *HasuraMetadataGenerator) GenerateMigration(bo *BusinessObjectDefinition) (string, error) {
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

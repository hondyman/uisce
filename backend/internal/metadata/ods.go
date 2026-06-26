package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

// ObjectDefinition represents the blueprint of a business object
type ObjectDefinition struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	TenantID    uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	Name        string          `db:"name" json:"name"`
	Slug        string          `db:"slug" json:"slug"`
	Version     int             `db:"version" json:"version"`
	Description string          `db:"description" json:"description"`
	JSONSchema  json.RawMessage `db:"json_schema" json:"json_schema"`
	UISchema    json.RawMessage `db:"ui_schema" json:"ui_schema"`
	IsActive    bool            `db:"is_active" json:"is_active"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
}

// FieldDefinition is a simplified input for creating schemas
type FieldDefinition struct {
	Key         string   `json:"key"`
	Type        string   `json:"type"` // string, number, boolean, date, enum
	Label       string   `json:"label"`
	Required    bool     `json:"required"`
	Options     []string `json:"options,omitempty"` // For enums
	Description string   `json:"description,omitempty"`
}

// CreateObjectInput is the payload to define a new object
type CreateObjectInput struct {
	TenantID    uuid.UUID         `json:"tenant_id"`
	Name        string            `json:"name"`
	Slug        string            `json:"slug"`
	Description string            `json:"description"`
	Fields      []FieldDefinition `json:"fields"`
}

type ODSService struct {
	db *sqlx.DB
}

func NewODSService(db *sqlx.DB) *ODSService {
	return &ODSService{db: db}
}

// CreateDefinition compiles a simplified field list into a JSON Schema and persists it
func (s *ODSService) CreateDefinition(ctx context.Context, input CreateObjectInput) (*ObjectDefinition, error) {
	// 1. Generate JSON Schema
	schemaMap, err := generateJSONSchema(input.Name, input.Fields)
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema: %w", err)
	}
	schemaBytes, _ := json.Marshal(schemaMap)

	// 2. Generate Default UI Schema (Simple vertical layout)
	uiSchemaMap := generateDefaultUISchema(input.Fields)
	uiSchemaBytes, _ := json.Marshal(uiSchemaMap)

	// 3. Persist
	def := &ObjectDefinition{
		TenantID:    input.TenantID,
		Name:        input.Name,
		Slug:        input.Slug,
		Version:     1,
		Description: input.Description,
		JSONSchema:  schemaBytes,
		UISchema:    uiSchemaBytes,
		IsActive:    true,
	}

	query := `
		INSERT INTO object_definitions (tenant_id, name, slug, version, description, json_schema, ui_schema, is_active)
		VALUES (:tenant_id, :name, :slug, :version, :description, :json_schema, :ui_schema, :is_active)
		RETURNING id, created_at, updated_at
	`
	rows, err := s.db.NamedQueryContext(ctx, query, def)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		rows.Scan(&def.ID, &def.CreatedAt, &def.UpdatedAt)
	}

	return def, nil
}

// ValidateEntity validates a JSON payload against the stored schema for a definition
func (s *ODSService) ValidateEntity(ctx context.Context, definitionID uuid.UUID, payload []byte) error {
	// 1. Fetch Schema
	var schemaStr string
	err := s.db.GetContext(ctx, &schemaStr, "SELECT json_schema FROM object_definitions WHERE id = $1", definitionID)
	if err != nil {
		return fmt.Errorf("definition not found: %w", err)
	}

	// 2. Compile Schema
	sch, err := jsonschema.CompileString("schema.json", schemaStr)
	if err != nil {
		return fmt.Errorf("invalid stored schema: %w", err)
	}

	// 3. Parse Payload
	var v interface{}
	if err := json.Unmarshal(payload, &v); err != nil {
		return fmt.Errorf("invalid json payload: %w", err)
	}

	// 4. Validate
	if err := sch.Validate(v); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

// Helper: Generate JSON Schema 2020-12
func generateJSONSchema(title string, fields []FieldDefinition) (map[string]interface{}, error) {
	properties := make(map[string]interface{})
	required := []string{}

	for _, f := range fields {
		prop := make(map[string]interface{})
		prop["title"] = f.Label
		if f.Description != "" {
			prop["description"] = f.Description
		}

		switch f.Type {
		case "string":
			prop["type"] = "string"
		case "number":
			prop["type"] = "number"
		case "boolean":
			prop["type"] = "boolean"
		case "date":
			prop["type"] = "string"
			prop["format"] = "date"
		case "enum":
			prop["type"] = "string"
			prop["enum"] = f.Options
		default:
			prop["type"] = "string"
		}

		properties[f.Key] = prop
		if f.Required {
			required = append(required, f.Key)
		}
	}

	return map[string]interface{}{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"title":                title,
		"type":                 "object",
		"properties":           properties,
		"required":             required,
		"additionalProperties": false, // Strict schema
	}, nil
}

// Helper: Generate Default UI Schema
func generateDefaultUISchema(fields []FieldDefinition) map[string]interface{} {
	// Simple ordering
	order := []string{}
	for _, f := range fields {
		order = append(order, f.Key)
	}
	return map[string]interface{}{
		"ui:order": order,
	}
}

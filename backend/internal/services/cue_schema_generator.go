package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// CueSchemaGenerator generates and caches CUE schemas from DB metadata
type CueSchemaGenerator struct {
	db     *sqlx.DB
	logger *zap.Logger
	cache  map[string]*cue.Value
	mu     sync.RWMutex
}

// NewCueSchemaGenerator creates a new generator
func NewCueSchemaGenerator(db *sqlx.DB) *CueSchemaGenerator {
	logger, _ := zap.NewProduction()
	return &CueSchemaGenerator{
		db:     db,
		logger: logger,
		cache:  make(map[string]*cue.Value),
	}
}

// InvalidateCache clears the cache for a specific BO ID
func (g *CueSchemaGenerator) InvalidateCache(boID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.cache, boID)
	g.logger.Info("Invalidated CUE schema cache", zap.String("bo_id", boID))
}

// GetSchema returns a compiled CUE value for the given Business Object ID.
// It checks the cache first, and if missing, generates it from the database.
func (g *CueSchemaGenerator) GetSchema(ctx context.Context, tenantID, boID string) (*cue.Value, error) {
	return g.GetSchemaWithLocale(ctx, tenantID, boID, "")
}

// GetSchemaWithLocale returns a compiled CUE value for the given BO ID and locale.
func (g *CueSchemaGenerator) GetSchemaWithLocale(ctx context.Context, tenantID, boID, locale string) (*cue.Value, error) {
	// Cache key includes locale
	cacheKey := fmt.Sprintf("%s:%s", boID, locale)
	g.mu.RLock()
	if v, ok := g.cache[cacheKey]; ok {
		g.mu.RUnlock()
		return v, nil
	}
	g.mu.RUnlock()

	var schemaStr string
	var err error
	if locale == "" {
		schemaStr, err = g.generateSchemaString(ctx, tenantID, boID)
	} else {
		schemaStr, err = g.generateMultilingualSchemaString(ctx, tenantID, boID, locale)
	}

	if err != nil {
		return nil, err
	}

	c := cuecontext.New()
	val := c.CompileString(schemaStr)
	if val.Err() != nil {
		return nil, fmt.Errorf("failed to compile CUE schema: %w", val.Err())
	}

	g.mu.Lock()
	g.cache[cacheKey] = &val
	g.mu.Unlock()

	return &val, nil
}

// GenerateSchemaStringPublic exposes schema string generation for API usage (IntelliSense)
func (g *CueSchemaGenerator) GenerateSchemaStringPublic(ctx context.Context, tenantID, boID, locale string) (string, error) {
	if locale == "" {
		return g.generateSchemaString(ctx, tenantID, boID)
	}
	return g.generateMultilingualSchemaString(ctx, tenantID, boID, locale)
}

// generateSchemaString fetches metadata and builds the CUE definition
func (g *CueSchemaGenerator) generateSchemaString(ctx context.Context, tenantID, boID string) (string, error) {
	// 1. Fetch Business Object Core
	var bo models.BusinessObjectDefinition
	err := g.db.GetContext(ctx, &bo, `
		SELECT id, name, technical_name, display_name, description 
		FROM business_objects 
		WHERE id = $1 AND tenant_id = $2
	`, boID, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("business object not found: %s", boID)
		}
		return "", fmt.Errorf("failed to fetch business object: %w", err)
	}

	// 2. Fetch Fields for BO (subtype_id IS NULL)
	var allFields []struct {
		models.FieldDefinition
	}
	err = g.db.SelectContext(ctx, &allFields, `
		SELECT id, name, technical_name, key, type, is_required, is_core
		FROM bo_fields
		WHERE business_object_id = $1 AND subtype_id IS NULL
	`, boID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch BO fields: %w", err)
	}

	var coreFields, customFields []models.FieldDefinition
	for _, f := range allFields {
		if f.IsCore {
			coreFields = append(coreFields, f.FieldDefinition)
		} else {
			customFields = append(customFields, f.FieldDefinition)
		}
	}

	// 3. Fetch Subtypes
	var subtypes []models.SubtypeDefinition
	err = g.db.SelectContext(ctx, &subtypes, `
		SELECT id, name, technical_name
		FROM bo_subtypes
		WHERE business_object_id = $1
	`, boID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch subtypes: %w", err)
	}

	// 4. Build CUE String
	var sb strings.Builder

	// Define the base schema #ObjectName
	schemaName := sanitizeIdentifier(bo.TechnicalName)
	if schemaName == "" {
		schemaName = sanitizeIdentifier(bo.Name)
	}

	sb.WriteString(fmt.Sprintf("// Generated Schema for %s (%s)\n", bo.DisplayName, bo.ID))
	sb.WriteString(fmt.Sprintf("#%s: {\n", schemaName))

	// Write Core Fields (at root)
	for _, f := range coreFields {
		writeField(&sb, f)
	}

	// Write Custom Fields (in 'custom' struct)
	if len(customFields) > 0 {
		sb.WriteString("\tcustom?: {\n")
		for _, f := range customFields {
			sb.WriteString("\t") // Extra indent
			writeField(&sb, f)
		}
		sb.WriteString("\t}\n")
	}

	sb.WriteString("\t...\n")
	sb.WriteString("}\n")

	// Subtypes definitions
	for _, st := range subtypes {
		stName := sanitizeIdentifier(st.TechnicalName)
		sb.WriteString(fmt.Sprintf("\n#%s_%s: #%s & {\n", schemaName, stName, schemaName))

		var stFields []models.FieldDefinition
		_ = g.db.SelectContext(ctx, &stFields, `
			SELECT id, name, technical_name, key, type, is_required
			FROM bo_fields
			WHERE subtype_id = $1
		`, st.ID)

		for _, f := range stFields {
			writeField(&sb, f)
		}

		sb.WriteString("}\n")
	}

	return sb.String(), nil
}

func (g *CueSchemaGenerator) generateMultilingualSchemaString(ctx context.Context, tenantID, boID, locale string) (string, error) {
	// 1. Fetch BO Core
	var bo models.BusinessObjectDefinition
	err := g.db.GetContext(ctx, &bo, `SELECT id, name, technical_name, display_name FROM business_objects WHERE id=$1 AND tenant_id=$2`, boID, tenantID)
	if err != nil {
		return "", err
	}

	// 2. Fetch Fields
	var allFields []struct {
		models.FieldDefinition
	}
	err = g.db.SelectContext(ctx, &allFields, `SELECT id, name, technical_name, key, type, is_required, is_core FROM bo_fields WHERE business_object_id=$1 AND subtype_id IS NULL`, boID)
	if err != nil {
		return "", err
	}

	// Mock translations for now. In reality, query an i18n_table.
	translate := func(text string) string {
		if locale == "es" {
			return "Traducido_" + text // Simulation
		}
		return text
	}

	var sb strings.Builder
	schemaName := sanitizeIdentifier(bo.TechnicalName)
	if schemaName == "" {
		schemaName = sanitizeIdentifier(bo.Name)
	}

	sb.WriteString(fmt.Sprintf("// Generated Schema for %s (%s) - Locale: %s\n", bo.DisplayName, bo.ID, locale))
	sb.WriteString(fmt.Sprintf("#%s_%s: {\n", schemaName, locale))

	// Inline separation logic
	var core, custom []models.FieldDefinition
	for _, f := range allFields {
		if f.IsCore {
			core = append(core, f.FieldDefinition)
		} else {
			custom = append(custom, f.FieldDefinition)
		}
	}

	for _, f := range core {
		writeFieldWithLabel(&sb, f, translate(f.Name))
	}
	if len(custom) > 0 {
		sb.WriteString("\tcustom?: {\n")
		for _, f := range custom {
			sb.WriteString("\t")
			writeFieldWithLabel(&sb, f, translate(f.Name))
		}
		sb.WriteString("\t}\n")
	}

	sb.WriteString("\t...\n")
	sb.WriteString("}\n")

	return sb.String(), nil
}

func writeField(sb *strings.Builder, f models.FieldDefinition) {
	fieldName := f.TechnicalName
	if fieldName == "" {
		fieldName = f.Key
	}
	if fieldName == "" {
		return
	}

	cueType := mapFieldTypeToCue(f.Type)
	constraint := ""
	if !f.IsRequired {
		constraint = "?"
	}

	sb.WriteString(fmt.Sprintf("\t%s%s: %s\n", fieldName, constraint, cueType))
}

func writeFieldWithLabel(sb *strings.Builder, f models.FieldDefinition, label string) {
	fieldName := f.TechnicalName
	if fieldName == "" {
		fieldName = f.Key
	}
	if fieldName == "" {
		return
	}

	cueType := mapFieldTypeToCue(f.Type)
	constraint := ""
	if !f.IsRequired {
		constraint = "?"
	}

	// Add @label attribute
	sb.WriteString(fmt.Sprintf("\t%s%s: %s @label(\"%s\")\n", fieldName, constraint, cueType, label))
}

// sanitizeIdentifier makes a string safe for CUE identifiers
func sanitizeIdentifier(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, s)
}

// mapFieldTypeToCue maps app types to CUE types
func mapFieldTypeToCue(t string) string {
	switch strings.ToLower(t) {
	case "text", "string", "email", "url", "image":
		return "string"
	case "number", "currency":
		return "number"
	case "integer":
		return "int"
	case "boolean", "bool":
		return "bool"
	case "date":
		// Simple regex for YYYY-MM-DD
		return `string & =~"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"`
	case "datetime":
		return `string` // simplified
	case "json":
		return "{...}"
	default:
		return "_" // Any
	}
}

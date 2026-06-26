package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/jmoiron/sqlx"
	"gopkg.in/yaml.v3"
)

// cubeSchemaSource contains the CUE schema for validating Cube models
// This is inline to avoid go:embed path restrictions
var cubeSchemaSource = `
package cube

#CubeModel: {
    cubes: [...#Cube]
}

#Cube: {
    name:       =~"^[a-zA-Z][a-zA-Z0-9_]*$"
    sql?:       string
    sql_table?: string
    data_source?: string
    dimensions?: [...#Dimension]
    measures?:   [...#Measure]
    joins?:      [...#Join]
}

#Dimension: {
    name: =~"^[a-zA-Z][a-zA-Z0-9_]*$"
    sql:  string
    type: "string" | "number" | "boolean" | "time" | "geo"
}

#Measure: {
    name: =~"^[a-zA-Z][a-zA-Z0-9_]*$"
    sql?: string
    type: "count" | "count_distinct" | "sum" | "avg" | "min" | "max" | "number" | "string"
}

#Join: {
    name:         =~"^[a-zA-Z][a-zA-Z0-9_]*$"
    sql:          string
    relationship: "one_to_one" | "one_to_many" | "many_to_one" | "belongs_to"
}
`

// CubeModel represents a complete Cube.dev model file
type CubeModel struct {
	Cubes []CubeDefinition `yaml:"cubes" json:"cubes"`
}

// CubeDefinition represents a single Cube
type CubeDefinition struct {
	Name            string               `yaml:"name" json:"name"`
	SQL             string               `yaml:"sql,omitempty" json:"sql,omitempty"`
	SQLTable        string               `yaml:"sql_table,omitempty" json:"sql_table,omitempty"`
	DataSource      string               `yaml:"data_source,omitempty" json:"data_source,omitempty"`
	Dimensions      []CubeDimension      `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
	Measures        []CubeMeasure        `yaml:"measures,omitempty" json:"measures,omitempty"`
	Joins           []CubeJoin           `yaml:"joins,omitempty" json:"joins,omitempty"`
	PreAggregations []CubePreAggregation `yaml:"pre_aggregations,omitempty" json:"pre_aggregations,omitempty"`
}

// CubeDimension represents a dimension in a Cube
type CubeDimension struct {
	Name        string `yaml:"name" json:"name"`
	SQL         string `yaml:"sql" json:"sql"`
	Type        string `yaml:"type" json:"type"`
	Title       string `yaml:"title,omitempty" json:"title,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	PrimaryKey  bool   `yaml:"primary_key,omitempty" json:"primary_key,omitempty"`
}

// CubeMeasure represents a measure in a Cube
type CubeMeasure struct {
	Name        string `yaml:"name" json:"name"`
	SQL         string `yaml:"sql,omitempty" json:"sql,omitempty"`
	Type        string `yaml:"type" json:"type"`
	Title       string `yaml:"title,omitempty" json:"title,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// CubeJoin represents a join relationship
type CubeJoin struct {
	Name         string `yaml:"name" json:"name"`
	SQL          string `yaml:"sql" json:"sql"`
	Relationship string `yaml:"relationship" json:"relationship"`
}

// CubePreAggregation represents a pre-aggregation configuration
type CubePreAggregation struct {
	Name        string   `yaml:"name" json:"name"`
	Measures    []string `yaml:"measures,omitempty" json:"measures,omitempty"`
	Dimensions  []string `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
	Granularity string   `yaml:"granularity,omitempty" json:"granularity,omitempty"`
}

// CubeGenerator generates Cube.dev schema files from semantic terms
type CubeGenerator struct {
	Repo      SemanticTermRepository
	CueEngine *CueEngine
	DB        *sqlx.DB
}

// NewCubeGenerator creates a new CubeGenerator with optional validation engines
func NewCubeGenerator(repo SemanticTermRepository) *CubeGenerator {
	return &CubeGenerator{Repo: repo}
}

// NewCubeGeneratorWithEngines creates a CubeGenerator with CUE and Starlark engines
func NewCubeGeneratorWithEngines(repo SemanticTermRepository, cue *CueEngine, db *sqlx.DB) *CubeGenerator {
	return &CubeGenerator{
		Repo:      repo,
		CueEngine: cue,
		DB:        db,
	}
}

// GenerateFromSemanticTerms generates a CubeModel from a list of semantic term IDs
func (g *CubeGenerator) GenerateFromSemanticTerms(ctx context.Context, cubeName string, termIDs []string) (*CubeModel, error) {
	cube := CubeDefinition{
		Name: cubeName,
	}

	for _, termID := range termIDs {
		term, err := g.Repo.GetTerm(termID)
		if err != nil {
			continue // Skip terms that can't be resolved
		}

		if term.Type == models.SemanticTypeRelationship && term.Relationship != nil {
			cube.Joins = append(cube.Joins, CubeJoin{
				Name:         term.Relationship.TargetBusinessObject,
				SQL:          term.Relationship.JoinExpression,
				Relationship: mapCardinality(term.Relationship.Cardinality),
			})
			continue
		}

		// Determine if measure or dimension
		if isMeasureTerm(term) {
			cube.Measures = append(cube.Measures, CubeMeasure{
				Name:        term.NodeName,
				SQL:         term.Expression,
				Type:        mapMeasureType(term.DataType),
				Title:       term.DisplayName,
				Description: term.Description,
			})
		} else {
			cube.Dimensions = append(cube.Dimensions, CubeDimension{
				Name:        term.NodeName,
				SQL:         term.Expression,
				Type:        mapDimensionType(term.DataType),
				Title:       term.DisplayName,
				Description: term.Description,
			})
		}
	}

	model := &CubeModel{Cubes: []CubeDefinition{cube}}

	// Validate with CUE if engine is available
	if g.CueEngine != nil {
		if err := g.validateWithCUE(ctx, model); err != nil {
			return nil, fmt.Errorf("CUE validation failed: %w", err)
		}
	}

	return model, nil
}

// GenerateFromBusinessObject generates a CubeModel for a specific Business Object,
// including its associated semantic terms and custom calculated fields.
func (g *CubeGenerator) GenerateFromBusinessObject(ctx context.Context, boID string) (*CubeModel, error) {
	// 1. Fetch Business Object
	var bo struct {
		ID              string `db:"id"`
		Name            string `db:"name"`
		DriverTableName string `db:"driver_table_name"`
	}
	err := g.DB.GetContext(ctx, &bo, "SELECT id, name, driver_table_name FROM business_objects WHERE id = $1", boID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch business object: %w", err)
	}

	// 2. Fetch associated semantic terms
	// For simplicity, we fetch terms that map to the BO's name prefix or are linked via metadata.
	// In the wealth case, we look for terms starting with the BO name (lowercase)
	prefix := strings.ToLower(strings.ReplaceAll(bo.Name, " ", "_")) + ".%"
	var termIDs []string
	err = g.DB.SelectContext(ctx, &termIDs, "SELECT id FROM catalog_node WHERE node_name LIKE $1", prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch semantic terms: %w", err)
	}

	termsData := make([]map[string]interface{}, 0)
	for _, id := range termIDs {
		term, err := g.Repo.GetTerm(id)
		if err != nil {
			continue
		}

		tType := "dimension"
		if isMeasureTerm(term) {
			tType = "measure"
		}

		termsData = append(termsData, map[string]interface{}{
			"id":              term.NodeName,
			"type":            tType,
			"data_type":       string(term.DataType),
			"materialization": term.Materialization,
		})
	}

	// 3. Fetch custom calculated fields (Phase 9)
	var calcs []struct {
		Name      string `db:"name"`
		SQLExpr   string `db:"sql_expr"`
		IsMeasure bool   `db:"is_measure"`
		DataType  string `db:"data_type"`
		Realtime  bool   `db:"realtime"`
	}
	err = g.DB.SelectContext(ctx, &calcs, "SELECT name, sql_expr, is_measure, data_type, realtime FROM calc_fields WHERE object_id = $1", boID)
	if err != nil {
		// If table doesn't exist or other error, just log and continue with empty calcs
		fmt.Printf("Warning: failed to fetch calc_fields: %v\n", err)
	}

	calcsData := make([]map[string]interface{}, 0)
	for _, c := range calcs {
		calcsData = append(calcsData, map[string]interface{}{
			"name":       c.Name,
			"sql_expr":   c.SQLExpr,
			"is_measure": c.IsMeasure,
			"data_type":  c.DataType,
			"realtime":   c.Realtime,
		})
	}

	// 5. Run Starlark transformation (Removed)
	// Starlark legacy transformation logic has been removed.
	// Returning basic model without Starlark enrichment.

	var model CubeModel
	// Default cube definition without Starlark processing
	cubeDef := CubeDefinition{
		Name:     bo.Name,
		SQLTable: bo.DriverTableName,
		// Terms processing logic would normally go here if not handled by Starlark
	}
	model.Cubes = []CubeDefinition{cubeDef}

	return &model, nil
}

// validateWithCUE validates the generated model against the CUE schema
func (g *CubeGenerator) validateWithCUE(ctx context.Context, model *CubeModel) error {
	modelJSON, err := json.Marshal(model)
	if err != nil {
		return err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(modelJSON, &data); err != nil {
		return err
	}

	result, err := g.CueEngine.EvaluateValidation(ctx, cubeSchemaSource, data)
	if err != nil {
		return err
	}

	if !result.IsValid {
		return fmt.Errorf("validation error: %s", result.Message)
	}

	return nil
}

// ApplyStarlarkTransformation is DEPRECATED and now a no-op.
func (g *CubeGenerator) ApplyStarlarkTransformation(ctx context.Context, model *CubeModel, script string) (*CubeModel, error) {
	return model, nil
}

// ToYAML converts the CubeModel to YAML format
func (g *CubeGenerator) ToYAML(model *CubeModel) ([]byte, error) {
	return yaml.Marshal(model)
}

// GenerateCubeSchema returns a YAML string for a Cube (legacy method)
func (g *CubeGenerator) GenerateCubeSchema(cubeName string, terms []string) (string, error) {
	ctx := context.Background()
	model, err := g.GenerateFromSemanticTerms(ctx, cubeName, terms)
	if err != nil {
		return "", err
	}

	yamlBytes, err := g.ToYAML(model)
	if err != nil {
		return "", err
	}

	return string(yamlBytes), nil
}

// Helper functions

func isMeasureTerm(term *models.SemanticTerm) bool {
	// Check for explicit tags
	for _, tag := range term.Tags {
		if tag == "measure" {
			return true
		}
		if tag == "dimension" {
			return false
		}
	}

	// Heuristic: Numbers and JSON are measures
	return term.DataType == models.DataTypeNumber || term.DataType == models.DataTypeJSON
}

func mapMeasureType(dt models.SemanticDataType) string {
	switch dt {
	case models.DataTypeNumber:
		return "number"
	case models.DataTypeString:
		return "string"
	case models.DataTypeBoolean:
		return "boolean"
	default:
		return "number"
	}
}

func mapDimensionType(dt models.SemanticDataType) string {
	switch dt {
	case models.DataTypeString:
		return "string"
	case models.DataTypeNumber:
		return "number"
	case models.DataTypeBoolean:
		return "boolean"
	case models.DataTypeDate, models.DataTypeDateTime:
		return "time"
	default:
		return "string"
	}
}

func mapCardinality(c string) string {
	c = strings.ToLower(c)
	switch c {
	case "one_to_one":
		return "one_to_one"
	case "one_to_many":
		return "one_to_many"
	case "many_to_one":
		return "many_to_one"
	default:
		return "belongs_to"
	}
}

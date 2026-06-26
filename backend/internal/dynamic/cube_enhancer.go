package dynamic

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/cube"
)

// CubeDynamicEnhancer enhances Cube.js with dynamic parameter support
type CubeDynamicEnhancer struct {
	baseCube *cube.Cube
}

// NewCubeDynamicEnhancer creates a new Cube dynamic enhancer
func NewCubeDynamicEnhancer(baseCube *cube.Cube) *CubeDynamicEnhancer {
	return &CubeDynamicEnhancer{
		baseCube: baseCube,
	}
}

// EnhanceWithDynamicParameters adds dynamic parameter support to Cube definition
func (cde *CubeDynamicEnhancer) EnhanceWithDynamicParameters(params []DynamicParameter) (*cube.Cube, error) {
	enhanced := *cde.baseCube // Copy the base cube

	// Add dynamic dimensions based on parameters
	for _, param := range params {
		if param.Type == "dimension" {
			if enhanced.Dimensions == nil {
				enhanced.Dimensions = make(map[string]map[string]interface{})
			}

			dimensionDef := map[string]interface{}{
				"sql":   fmt.Sprintf("{%s}", param.Name),
				"type":  "string",
				"title": param.Description,
				"meta": map[string]interface{}{
					"dynamic":   true,
					"parameter": param.Name,
					"required":  param.Required,
					"default":   param.DefaultValue,
					"options":   param.Options,
				},
			}

			enhanced.Dimensions[param.Name] = dimensionDef
		}
	}

	return &enhanced, nil
}

// EnhanceWithDynamicMeasures adds dynamic measure support to Cube definition
func (cde *CubeDynamicEnhancer) EnhanceWithDynamicMeasures(measures []DynamicMeasure) (*cube.Cube, error) {
	enhanced := *cde.baseCube // Copy the base cube

	// Add dynamic measures
	for _, measure := range measures {
		if enhanced.Measures == nil {
			enhanced.Measures = make(map[string]map[string]interface{})
		}

		measureDef := map[string]interface{}{
			"sql":   measure.SQL,
			"type":  measure.Type,
			"title": measure.Name,
			"meta": map[string]interface{}{
				"dynamic":    true,
				"parameters": measure.Parameters,
			},
		}

		// Add filters if specified
		if measure.Meta != nil {
			if filters, exists := measure.Meta["filters"]; exists {
				measureDef["filters"] = filters
			}
		}

		enhanced.Measures[measure.Name] = measureDef
	}

	return &enhanced, nil
}

// EnhanceWithDynamicMeasuresFromCube applies dynamic measures to the provided cube
// (non-destructively) and returns the enhanced copy. This lets callers chain
// parameter-based enhancements first and then apply measures.
func (cde *CubeDynamicEnhancer) EnhanceWithDynamicMeasuresFromCube(base *cube.Cube, measures []DynamicMeasure) (*cube.Cube, error) {
	enhanced := *base // Copy the provided cube

	// Add dynamic measures
	for _, measure := range measures {
		if enhanced.Measures == nil {
			enhanced.Measures = make(map[string]map[string]interface{})
		}

		measureDef := map[string]interface{}{
			"sql":   measure.SQL,
			"type":  measure.Type,
			"title": measure.Name,
			"meta": map[string]interface{}{
				"dynamic":    true,
				"parameters": measure.Parameters,
			},
		}

		if measure.Meta != nil {
			if filters, exists := measure.Meta["filters"]; exists {
				measureDef["filters"] = filters
			}
		}

		enhanced.Measures[measure.Name] = measureDef
	}

	return &enhanced, nil
}

// GenerateCubeJSConfig generates Cube.js configuration with dynamic enhancements
func (cde *CubeDynamicEnhancer) GenerateCubeJSConfig(params []DynamicParameter, measures []DynamicMeasure) (string, error) {
	base, err := cde.EnhanceWithDynamicParameters(params)
	if err != nil {
		return "", fmt.Errorf("failed to enhance with parameters: %w", err)
	}

	// Apply measures to the base enhanced cube and get the final enhanced cube
	enhancedCube, err := cde.EnhanceWithDynamicMeasuresFromCube(base, measures)
	if err != nil {
		return "", fmt.Errorf("failed to enhance with measures: %w", err)
	}

	// Convert to Cube.js YAML format
	config, err := cde.convertToCubeJSYAML(enhancedCube)
	if err != nil {
		return "", fmt.Errorf("failed to convert to Cube.js config: %w", err)
	}

	return config, nil
}

// convertToCubeJSYAML converts our internal Cube representation to Cube.js YAML
func (cde *CubeDynamicEnhancer) convertToCubeJSYAML(cube *cube.Cube) (string, error) {
	var yaml strings.Builder

	yaml.WriteString("cubes:\n")
	yaml.WriteString(fmt.Sprintf("  - name: %s\n", cube.Name))

	if cube.SQL != "" {
		yaml.WriteString("    sql: " + cube.SQL + "\n")
	}

	if cube.SQLTable != "" {
		yaml.WriteString("    sql_table: " + cube.SQLTable + "\n")
	}

	// Add dimensions
	if len(cube.Dimensions) > 0 {
		yaml.WriteString("    dimensions:\n")
		for name, def := range cube.Dimensions {
			yaml.WriteString(fmt.Sprintf("      %s:\n", name))
			if sql, ok := def["sql"].(string); ok {
				yaml.WriteString(fmt.Sprintf("        sql: %s\n", sql))
			}
			if typ, ok := def["type"].(string); ok {
				yaml.WriteString(fmt.Sprintf("        type: %s\n", typ))
			}
			if title, ok := def["title"].(string); ok {
				yaml.WriteString(fmt.Sprintf("        title: %s\n", title))
			}
			if meta, ok := def["meta"].(map[string]interface{}); ok {
				if dynamic, ok := meta["dynamic"].(bool); ok && dynamic {
					yaml.WriteString("        meta:\n")
					yaml.WriteString("          dynamic: true\n")
					if param, ok := meta["parameter"].(string); ok {
						yaml.WriteString(fmt.Sprintf("          parameter: %s\n", param))
					}
				}
			}
		}
	}

	// Add measures
	if len(cube.Measures) > 0 {
		yaml.WriteString("    measures:\n")
		for name, def := range cube.Measures {
			yaml.WriteString(fmt.Sprintf("      %s:\n", name))
			if sql, ok := def["sql"].(string); ok {
				yaml.WriteString(fmt.Sprintf("        sql: %s\n", sql))
			}
			if typ, ok := def["type"].(string); ok {
				yaml.WriteString(fmt.Sprintf("        type: %s\n", typ))
			}
			if title, ok := def["title"].(string); ok {
				yaml.WriteString(fmt.Sprintf("        title: %s\n", title))
			}
			if meta, ok := def["meta"].(map[string]interface{}); ok {
				if dynamic, ok := meta["dynamic"].(bool); ok && dynamic {
					yaml.WriteString("        meta:\n")
					yaml.WriteString("          dynamic: true\n")
				}
			}
		}
	}

	return yaml.String(), nil
}

// GenerateParameterSchema generates JSON schema for dynamic parameters
func (cde *CubeDynamicEnhancer) GenerateParameterSchema(params []DynamicParameter) (string, error) {
	schema := map[string]interface{}{
		"type":       "object",
		"properties": make(map[string]interface{}),
		"required":   make([]string, 0),
	}

	properties := schema["properties"].(map[string]interface{})

	for _, param := range params {
		paramSchema := map[string]interface{}{
			"type":        cde.mapParameterType(param.Type),
			"description": param.Description,
		}

		if param.DefaultValue != nil {
			paramSchema["default"] = param.DefaultValue
		}

		if len(param.Options) > 0 {
			paramSchema["enum"] = param.Options
		}

		properties[param.Name] = paramSchema

		if param.Required {
			schema["required"] = append(schema["required"].([]string), param.Name)
		}
	}

	schemaBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal parameter schema: %w", err)
	}

	return string(schemaBytes), nil
}

// mapParameterType maps our parameter types to JSON schema types
func (cde *CubeDynamicEnhancer) mapParameterType(paramType string) string {
	switch paramType {
	case "string", "dimension":
		return "string"
	case "number", "measure":
		return "number"
	case "boolean":
		return "boolean"
	case "date", "time_range":
		return "string"
	default:
		return "string"
	}
}

package models

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

func extractScopeTables(scope map[string]interface{}) []string {
	var tableNames []string
	if scope["type"] == "tables" {
		if names, ok := scope["names"].([]interface{}); ok {
			for _, n := range names {
				if s, ok := n.(string); ok {
					tableNames = append(tableNames, s)
				}
			}
		}
	}
	return tableNames
}

func partitionByExistence(metaMap map[string]ModelMetadata, tables []string, overwrite bool) (toGenerate, skipped, overwritten []string) {
	for _, t := range tables {
		if meta, ok := metaMap[t]; ok && meta.Exists {
			if overwrite {
				overwritten = append(overwritten, t)
			} else {
				skipped = append(skipped, t)
			}
		} else {
			toGenerate = append(toGenerate, t)
		}
	}
	return
}

// fabricDefnsToSemanticModels converts the raw DB-oriented model definitions
// into a richer intermediate representation that includes detailed column info.
func fabricDefnsToSemanticModels(fabricDefns []*FabricDefn) []SemanticModel {
	// Initialize as a non-nil empty slice to ensure JSON marshals to [] instead of null.
	if fabricDefns == nil {
		return make([]SemanticModel, 0)
	}
	semanticModels := make([]SemanticModel, 0, len(fabricDefns))
	for _, defn := range fabricDefns {
		var config ResolvedModelConfig
		if err := json.Unmarshal(defn.ResolvedConfig, &config); err != nil {
			log.Printf("Warning: could not unmarshal resolved config for model %s: %v", defn.ModelKey, err)
			continue
		}

		if len(config.Cubes) > 0 {
			modelCube := config.Cubes[0]
			// Derive a qualified table name from either sql_table, sql path, or SELECT SQL.
			qualifiedTableName := ""
			if modelCube.SQLTable != "" {
				qualifiedTableName = modelCube.SQLTable
			} else if modelCube.SQL != "" {
				if strings.HasPrefix(modelCube.SQL, "/") {
					qualifiedTableName = strings.Replace(strings.TrimPrefix(modelCube.SQL, "/"), "/", ".", 1)
				} else {
					// crude parse of SELECT * FROM schema.table
					up := strings.ToUpper(modelCube.SQL)
					if idx := strings.Index(up, "FROM "); idx >= 0 {
						after := strings.TrimSpace(modelCube.SQL[idx+5:])
						parts := strings.Fields(after)
						if len(parts) > 0 {
							qualifiedTableName = strings.Trim(parts[0], "`\"")
						}
					}
				}
			}

			// Initialize with empty slices to avoid `null` in JSON
			sm := SemanticModel{
				TableName:  qualifiedTableName,
				SqlTable:   qualifiedTableName,
				ModelName:  modelCube.Name,
				Dimensions: make([]SemanticMember, 0),
				Measures:   make([]SemanticMember, 0),
				Joins:      make([]CubeJoin, 0),
			}

			// Process dimensions (guard nil)
			for name, props := range modelCube.Dimensions {
				desc, _ := props["title"].(string)
				sm.Dimensions = append(sm.Dimensions, SemanticMember{
					Name:        name,
					Type:        props["type"].(string),
					SQL:         props["sql"].(string),
					Description: desc,
				})
			}

			// Process measures (guard nil)
			for name, props := range modelCube.Measures {
				desc, _ := props["title"].(string)
				sm.Measures = append(sm.Measures, SemanticMember{
					Name:        name,
					Type:        props["type"].(string),
					SQL:         props["sql"].(string),
					Description: desc,
				})
			}

			// Process joins into the Cube.dev format for the API response.
			for joinKey, joinProps := range modelCube.Joins {
				sql, _ := joinProps["sql"].(string)
				relationship, _ := joinProps["relationship"].(string)

				sm.Joins = append(sm.Joins, CubeJoin{
					Name:         joinKey,
					SQL:          sql,
					Relationship: relationship,
				})
			}
			semanticModels = append(semanticModels, sm)
		}
	}
	return semanticModels
}

// ExplainMeta creates a metadata block for generated model elements.
func ExplainMeta(ruleID, table, column string) map[string]any {
	return map[string]any{
		"provenance":   fmt.Sprintf("%s.%s", table, column),
		"rule_id":      ruleID,
		"source":       "schema_introspection",
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}
}

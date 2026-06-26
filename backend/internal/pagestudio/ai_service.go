package pagestudio

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/semantic"
)

// GenerateAILayout uses heuristics to suggest a page layout for a Business Object
func GenerateAILayout(bo *semantic.SemanticObject, intent string) (*AIGenerateResponse, error) {
	var boPayload map[string]interface{}
	if err := json.Unmarshal(bo.Payload, &boPayload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal BO payload: %w", err)
	}

	// Simple Heuristics:
	// 1. Identify "Measures" (numbers like value, price, amount)
	// 2. Identify "Dimensions" (strings like status, category, account)
	// 3. Identify "Time" (fields like as_of_date, timestamp)

	var measures []string
	var dimensions []string
	var timeFields []string

	// In a real system, we'd use the semantic graph metadata.
	// Here we peek at the 'fields' or top-level keys if it's a BO definition.
	if fields, ok := boPayload["fields"].([]interface{}); ok {
		for _, f := range fields {
			fieldMap, ok := f.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := fieldMap["name"].(string)
			typ, _ := fieldMap["type"].(string)

			switch strings.ToLower(typ) {
			case "number", "decimal", "currency", "integer":
				measures = append(measures, name)
			case "string", "text", "enum":
				dimensions = append(dimensions, name)
			case "date", "timestamp", "datetime":
				timeFields = append(timeFields, name)
			}
		}
	}

	// Default Fallback: Use keys
	if len(measures) == 0 && len(dimensions) == 0 {
		for k := range boPayload {
			if k == "id" || k == "name" {
				continue
			}
			dimensions = append(dimensions, k)
		}
	}

	// Generate Layout based on Intent
	switch intent {
	case "dashboard":
		return generateDashboardLayout(bo.ID, measures, dimensions, timeFields)
	case "list":
		return generateListLayout(bo.ID, dimensions)
	default:
		return generateDashboardLayout(bo.ID, measures, dimensions, timeFields)
	}
}

func generateDashboardLayout(boID string, measures, dimensions, timeFields []string) (*AIGenerateResponse, error) {
	// Layout Tree
	layout := map[string]interface{}{
		"root": "root",
		"nodes": map[string]interface{}{
			"root":  map[string]interface{}{"id": "root", "type": "Row", "children": []string{"left", "right"}},
			"left":  map[string]interface{}{"id": "left", "type": "Column", "children": []string{"mainTable"}},
			"right": map[string]interface{}{"id": "right", "type": "Column", "children": []string{"summaryKPI", "trendChart"}},
		},
	}

	// Components
	tableColumns := []map[string]string{}
	for _, d := range dimensions {
		tableColumns = append(tableColumns, map[string]string{"field": d, "label": strings.Title(d)})
	}
	for _, m := range measures {
		tableColumns = append(tableColumns, map[string]string{"field": m, "label": strings.Title(m)})
	}

	yField := "value"
	if len(measures) > 0 {
		yField = measures[0]
	}
	xField := "date"
	if len(timeFields) > 0 {
		xField = timeFields[0]
	}

	components := map[string]interface{}{
		"mainTable": map[string]interface{}{
			"id":    "mainTable",
			"type":  "Table",
			"props": map[string]interface{}{"columns": tableColumns},
		},
		"summaryKPI": map[string]interface{}{
			"id":    "summaryKPI",
			"type":  "KPIGroup",
			"props": map[string]interface{}{"label": "Total " + strings.Title(yField)},
		},
		"trendChart": map[string]interface{}{
			"id":    "trendChart",
			"type":  "LineChart",
			"props": map[string]interface{}{"xField": xField, "yField": yField},
		},
	}

	// Data Bindings
	dataBindings := map[string]interface{}{
		"sources": map[string]interface{}{
			"boSource": map[string]interface{}{"type": "BO", "id": boID},
		},
		"bindings": []map[string]interface{}{
			{"componentId": "mainTable", "prop": "rows", "sourceId": "boSource"},
			{"componentId": "summaryKPI", "prop": "data", "sourceId": "boSource"},
			{"componentId": "trendChart", "prop": "data", "sourceId": "boSource"},
		},
	}

	lBytes, _ := json.Marshal(layout)
	cBytes, _ := json.Marshal(components)
	dBytes, _ := json.Marshal(dataBindings)

	return &AIGenerateResponse{
		Layout:       lBytes,
		Components:   cBytes,
		DataBindings: dBytes,
	}, nil
}

func generateListLayout(boID string, dimensions []string) (*AIGenerateResponse, error) {
	// Simple list layout
	// ... (implementation for list)
	return generateDashboardLayout(boID, nil, dimensions, nil) // Fallback for brevity
}

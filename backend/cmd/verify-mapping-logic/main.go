package main

import (
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/analytics"
)

func main() {
	// Mock service or just calling the relevant functions

	// Test case 1: "id" -> "IDENTIFIER" with table context
	col1 := &analytics.DatabaseColumn{
		Column: "id",
		Table:  "employees",
	}

	// Test case 2: "name" -> table context
	col2 := &analytics.DatabaseColumn{
		Column: "name",
		Table:  "departments",
	}

	// Test case 3: "city" -> table context (ambiguous)
	col3 := &analytics.DatabaseColumn{
		Column: "city",
		Table:  "office_locations",
	}

	testColumns := []*analytics.DatabaseColumn{col1, col2, col3}

	for _, col := range testColumns {
		// Simulations based on the logic in semantic_mapping_wizard.go and semantic_mapping_service.go

		// 1. Expansion (from service)
		expandedName := col.Column
		// Manual simulation of normalizeColumnName and replacements
		replacements := map[string]string{
			"ID":  "Identifier",
			"DT":  "Date",
			"AMT": "Amount",
		}
		name := strings.ToUpper(col.Column)
		if val, ok := replacements[name]; ok {
			expandedName = val
		}

		// 2. Ambiguity check and table context (from wizard)
		semanticTerm := expandedName
		if isAmbiguous(col.Column) {
			semanticTerm = col.Table + "_" + expandedName
		}

		// 3. Formatting (from wizard)
		finalTerm := strings.ToUpper(strings.ReplaceAll(semanticTerm, " ", "_"))

		fmt.Printf("Column: %s, Table: %s -> Final Semantic Term: %s\n", col.Column, col.Table, finalTerm)
	}
}

func isAmbiguous(columnName string) bool {
	ambiguousNames := []string{
		"id", "name", "description", "type", "status", "code", "value",
		"city", "state", "country", "address", "street", "zip", "postal",
		"email", "phone", "fax", "website", "url",
		"date", "time", "timestamp", "created", "updated", "modified",
		"amount", "price", "cost", "total", "quantity", "qty",
		"flag", "active", "enabled", "deleted", "archived",
		"notes", "comments", "remarks", "memo",
		"key", "label", "text", "category", "group", "class",
		"start_date", "end_date", "start", "end",
	}

	lowerName := strings.ToLower(columnName)
	for _, ambiguous := range ambiguousNames {
		if lowerName == ambiguous || strings.HasSuffix(lowerName, "_"+ambiguous) {
			return true
		}
	}
	if len(columnName) <= 2 {
		return true
	}
	return false
}

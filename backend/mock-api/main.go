package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// Mock view skeleton with all Cube.dev parameters
var mockView = map[string]interface{}{
	"name":        "enhanced-skeleton",
	"title":       "Enhanced Cube.dev View",
	"description": "Complete Cube.dev view with all supported parameters",
	"extends":     "",
	"public":      true,
	"meta": map[string]interface{}{
		"author":      "semlayer",
		"version":     "1.0.0",
		"description": "Enhanced view with complete Cube.dev support",
		"tags":        []string{"financial", "analytics", "cube"},
	},
	"access_policy": map[string]interface{}{
		"can_access": []string{"*"},
		"row_level_security": map[string]interface{}{
			"enabled": false,
			"filter":  "",
		},
	},
	"cubes": []interface{}{
		"orders",
		map[string]interface{}{
			"join_path": "users.orders",
			"includes":  "*",
			"excludes":  []string{"internal_id", "temp_field"},
			"prefix":    true,
			"alias":     "order_data",
		},
	},
	"dimensions": []interface{}{
		map[string]interface{}{
			"name":        "status",
			"sql":         "${cube.status}",
			"type":        "string",
			"title":       "Order Status",
			"description": "Current status of the order",
		},
		map[string]interface{}{
			"name":        "created_date",
			"sql":         "${cube.created_at}",
			"type":        "time",
			"title":       "Created Date",
			"description": "When the order was created",
		},
	},
	"measures": []interface{}{
		map[string]interface{}{
			"name":        "count",
			"sql":         "COUNT(*)",
			"type":        "count",
			"title":       "Total Orders",
			"description": "Total number of orders",
		},
		map[string]interface{}{
			"name":        "total_amount",
			"sql":         "SUM(${cube.amount})",
			"type":        "sum",
			"title":       "Total Amount",
			"description": "Sum of all order amounts",
		},
	},
	"folders": []interface{}{
		map[string]interface{}{
			"name":        "basic_metrics",
			"title":       "Basic Metrics",
			"description": "Essential order metrics",
			"includes":    []string{"count", "total_amount"},
		},
		map[string]interface{}{
			"name":        "order_details",
			"title":       "Order Details",
			"description": "Detailed order information",
			"includes":    []string{"status", "created_date"},
		},
	},
	"schema_documentation": map[string]interface{}{
		"extends": map[string]interface{}{
			"type":        "string",
			"description": "Name of another view to extend from",
			"example":     "base_view",
			"required":    false,
		},
		"public": map[string]interface{}{
			"type":        "boolean",
			"description": "Whether this view is publicly accessible",
			"default":     true,
			"required":    false,
		},
		"meta": map[string]interface{}{
			"type":        "object",
			"description": "Metadata about this view",
			"properties": map[string]interface{}{
				"author":      "Author of the view",
				"version":     "Version number",
				"description": "Detailed description",
				"tags":        "Array of tags for categorization",
			},
			"required": false,
		},
		"access_policy": map[string]interface{}{
			"type":        "object",
			"description": "Access control settings",
			"properties": map[string]interface{}{
				"can_access":         "Array of roles that can access this view",
				"row_level_security": "Row-level security configuration",
			},
			"required": false,
		},
	},
}

func main() {
	// Simple in-memory sample mappings for demo
	var sampleMappings = []map[string]interface{}{
		{
			"database_column":  map[string]interface{}{"node_id": "col-1", "schema": "public", "table": "users", "column": "email", "data_type": "VARCHAR"},
			"semantic_term":    "EMAIL",
			"semantic_term_id": "term-1",
			"confidence":       1.0,
			"is_new_term":      false,
			"selected":         false,
		},
		{
			"database_column":  map[string]interface{}{"node_id": "col-2", "schema": "public", "table": "users", "column": "id", "data_type": "UUID"},
			"semantic_term":    "USER*ID",
			"semantic_term_id": "",
			"confidence":       0.72,
			"is_new_term":      true,
			"selected":         false,
		},
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Enable CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Handle view skeleton requests
		if strings.Contains(r.URL.Path, "/api/views/") && r.URL.Query().Get("create") == "true" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockView)
			return
		}

		// Handle view validation requests
		if strings.Contains(r.URL.Path, "/validate") {
			w.Header().Set("Content-Type", "application/json")
			result := map[string]interface{}{
				"valid":   true,
				"issues":  []interface{}{},
				"message": "View validation successful",
			}
			json.NewEncoder(w).Encode(result)
			return
		}

		// Default response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"message": "Mock API for testing enhanced ViewEditor",
		})
	})

	// Semantic mappings endpoints for the UI
	http.HandleFunc("/api/semantic-mappings", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" {
			json.NewEncoder(w).Encode(sampleMappings)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	// Add singular route for compatibility
	http.HandleFunc("/api/semantic-mapping/generate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// Return sample results
		json.NewEncoder(w).Encode(sampleMappings)
	})

	http.HandleFunc("/api/semantic-mappings/edges", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var body map[string][]map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload"})
			return
		}

		mappings := body["mappings"]
		createdEdges := 0
		createdTerms := 0
		for _, m := range mappings {
			if ov, ok := m["override"].(bool); ok && ov {
				// Pretend we overwrote an existing mapping
			}
			if isNew, ok := m["is_new_term"].(bool); ok && isNew {
				createdTerms++
			}
			createdEdges++
		}

		json.NewEncoder(w).Encode(map[string]interface{}{"created_edges": createdEdges, "created_terms": createdTerms})
	})

	http.HandleFunc("/api/semantic-mappings/ignore", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// Accept the ignore payload and return 200
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload"})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	http.HandleFunc("/api/semantic-terms/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload"})
			return
		}

		query, _ := body["query"].(string)
		limitVal, _ := body["limit"].(float64)
		limit := int(limitVal)
		if limit <= 0 {
			limit = 10
		}

		// Mock semantic terms - common business terms
		mockTerms := []map[string]interface{}{
			{"node_id": "term-1", "term_name": "EMAIL", "node_type": "semantic_column", "qualified_path": "/semantic/EMAIL"},
			{"node_id": "term-2", "term_name": "USER_ID", "node_type": "semantic_column", "qualified_path": "/semantic/USER_ID"},
			{"node_id": "term-3", "term_name": "CUSTOMER_NAME", "node_type": "semantic_column", "qualified_path": "/semantic/CUSTOMER_NAME"},
			{"node_id": "term-4", "term_name": "ORDER_DATE", "node_type": "semantic_column", "qualified_path": "/semantic/ORDER_DATE"},
			{"node_id": "term-5", "term_name": "TOTAL_AMOUNT", "node_type": "semantic_column", "qualified_path": "/semantic/TOTAL_AMOUNT"},
			{"node_id": "term-6", "term_name": "PRODUCT_NAME", "node_type": "semantic_column", "qualified_path": "/semantic/PRODUCT_NAME"},
			{"node_id": "term-7", "term_name": "QUANTITY", "node_type": "semantic_column", "qualified_path": "/semantic/QUANTITY"},
			{"node_id": "term-8", "term_name": "PRICE", "node_type": "semantic_column", "qualified_path": "/semantic/PRICE"},
			{"node_id": "term-9", "term_name": "CATEGORY", "node_type": "semantic_column", "qualified_path": "/semantic/CATEGORY"},
			{"node_id": "term-10", "term_name": "STATUS", "node_type": "semantic_column", "qualified_path": "/semantic/STATUS"},
			{"node_id": "term-11", "term_name": "CREATED_AT", "node_type": "semantic_column", "qualified_path": "/semantic/CREATED_AT"},
			{"node_id": "term-12", "term_name": "UPDATED_AT", "node_type": "semantic_column", "qualified_path": "/semantic/UPDATED_AT"},
			{"node_id": "term-13", "term_name": "DESCRIPTION", "node_type": "semantic_column", "qualified_path": "/semantic/DESCRIPTION"},
			{"node_id": "term-14", "term_name": "PHONE_NUMBER", "node_type": "semantic_column", "qualified_path": "/semantic/PHONE_NUMBER"},
			{"node_id": "term-15", "term_name": "ADDRESS", "node_type": "semantic_column", "qualified_path": "/semantic/ADDRESS"},
		}

		// Filter by query if provided
		results := make([]map[string]interface{}, 0)
		if query == "" {
			results = mockTerms
		} else {
			queryLower := strings.ToLower(query)
			for _, term := range mockTerms {
				termName, _ := term["term_name"].(string)
				if strings.Contains(strings.ToLower(termName), queryLower) {
					results = append(results, term)
				}
			}
		}

		// Apply limit
		if len(results) > limit {
			results = results[:limit]
		}

		json.NewEncoder(w).Encode(results)
	})

	fmt.Println("🚀 Mock API server running on http://localhost:3001")
	fmt.Println("📝 Serving enhanced Cube.dev view skeleton for frontend testing")
	log.Fatal(http.ListenAndServe(":3001", nil))
}

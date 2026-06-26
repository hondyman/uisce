package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// SeedAPIEndpointsCatalog seeds the API endpoints catalog with validation-related endpoints
func SeedAPIEndpointsCatalog(db *sql.DB, tenantID string) error {
	endpoints := []APIEndpointRequest{
		// Validation Rules CRUD
		{
			EndpointName: "List Validation Rules",
			Description:  "Retrieve all validation rules for the tenant with pagination and filtering support",
			HTTPMethod:   "GET",
			URLPath:      "/validation-rules",
			Category:     "validation",
			Subcategory:  "rules",
			Purpose:      "read",
			RequiresAuth: true,
			IsActive:     true,
			Version:      "1.0.0",
			Parameters: []EndpointParameter{
				{
					Name:        "page",
					In:          "query",
					Required:    false,
					Description: "Page number for pagination (default: 1)",
					DataType:    "integer",
					Example:     1,
				},
				{
					Name:        "limit",
					In:          "query",
					Required:    false,
					Description: "Items per page (default: 50, max: 200)",
					DataType:    "integer",
					Example:     50,
				},
				{
					Name:        "entity_id",
					In:          "query",
					Required:    false,
					Description: "Filter by entity ID",
					DataType:    "string",
					Example:     "uuid",
				},
				{
					Name:        "status",
					In:          "query",
					Required:    false,
					Description: "Filter by status: active, inactive, all",
					DataType:    "string",
					Example:     "active",
				},
			},
			ResponseSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
						},
					},
					"pagination": map[string]interface{}{
						"type": "object",
					},
				},
			},
			Tags: []string{"validation", "rules", "listing"},
		},
		{
			EndpointName: "Create Validation Rule",
			Description:  "Create a new validation rule for one or more entities",
			HTTPMethod:   "POST",
			URLPath:      "/validation-rules",
			Category:     "validation",
			Subcategory:  "rules",
			Purpose:      "create",
			RequiresAuth: true,
			IsActive:     true,
			Version:      "1.0.0",
			RequestSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"name", "description", "condition"},
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
					"description": map[string]interface{}{
						"type": "string",
					},
					"condition": map[string]interface{}{
						"type": "object",
					},
				},
			},
			Tags: []string{"validation", "rules", "creation"},
		},
		{
			EndpointName: "Get Validation Rule",
			Description:  "Retrieve details of a specific validation rule",
			HTTPMethod:   "GET",
			URLPath:      "/validation-rules/{id}",
			Category:     "validation",
			Subcategory:  "rules",
			Purpose:      "read",
			RequiresAuth: true,
			IsActive:     true,
			Version:      "1.0.0",
			Parameters: []EndpointParameter{
				{
					Name:        "id",
					In:          "path",
					Required:    true,
					Description: "Validation rule ID",
					DataType:    "string",
				},
			},
			Tags: []string{"validation", "rules", "detail"},
		},
		{
			EndpointName: "Update Validation Rule",
			Description:  "Update an existing validation rule",
			HTTPMethod:   "PATCH",
			URLPath:      "/validation-rules/{id}",
			Category:     "validation",
			Subcategory:  "rules",
			Purpose:      "update",
			RequiresAuth: true,
			IsActive:     true,
			Version:      "1.0.0",
			Parameters: []EndpointParameter{
				{
					Name:        "id",
					In:          "path",
					Required:    true,
					Description: "Validation rule ID",
					DataType:    "string",
				},
			},
			Tags: []string{"validation", "rules", "update"},
		},
		{
			EndpointName: "Delete Validation Rule",
			Description:  "Delete a validation rule",
			HTTPMethod:   "DELETE",
			URLPath:      "/validation-rules/{id}",
			Category:     "validation",
			Subcategory:  "rules",
			Purpose:      "delete",
			RequiresAuth: true,
			IsActive:     true,
			Version:      "1.0.0",
			Parameters: []EndpointParameter{
				{
					Name:        "id",
					In:          "path",
					Required:    true,
					Description: "Validation rule ID",
					DataType:    "string",
				},
			},
			Tags: []string{"validation", "rules", "deletion"},
		},

		// Validation Execution
		{
			EndpointName: "Execute Single Validation Rule",
			Description:  "Execute a validation rule against a specific entity record",
			HTTPMethod:   "POST",
			URLPath:      "/validation-rules/{id}/execute",
			Category:     "validation",
			Subcategory:  "execution",
			Purpose:      "execute",
			RequiresAuth: true,
			IsActive:     true,
			Version:      "1.0.0",
			Parameters: []EndpointParameter{
				{
					Name:        "id",
					In:          "path",
					Required:    true,
					Description: "Validation rule ID",
					DataType:    "string",
				},
			},
			RequestSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"data"},
				"properties": map[string]interface{}{
					"data": map[string]interface{}{
						"type":        "object",
						"description": "The entity data to validate",
					},
				},
			},
			ResponseSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"is_valid": map[string]interface{}{
						"type": "boolean",
					},
					"errors": map[string]interface{}{
						"type": "array",
					},
				},
			},
			Tags: []string{"validation", "execution", "single"},
		},
		{
			EndpointName: "Execute Batch Validation Rules",
			Description:  "Execute validation rules against multiple entity records in a batch",
			HTTPMethod:   "POST",
			URLPath:      "/validation-rules/execute-batch",
			Category:     "validation",
			Subcategory:  "execution",
			Purpose:      "execute",
			RequiresAuth: true,
			IsActive:     true,
			Version:      "1.0.0",
			RequestSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"records"},
				"properties": map[string]interface{}{
					"records": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
						},
					},
				},
			},
			Tags: []string{"validation", "execution", "batch"},
		},

		// Audit and History
		{
			EndpointName: "Get Validation Rule Audit Trail",
			Description:  "Retrieve the audit trail for a specific validation rule",
			HTTPMethod:   "GET",
			URLPath:      "/validation-rules/{id}/audit",
			Category:     "validation",
			Subcategory:  "audit",
			Purpose:      "read",
			RequiresAuth: true,
			IsActive:     true,
			Version:      "1.0.0",
			Parameters: []EndpointParameter{
				{
					Name:        "id",
					In:          "path",
					Required:    true,
					Description: "Validation rule ID",
					DataType:    "string",
				},
				{
					Name:        "page",
					In:          "query",
					Required:    false,
					Description: "Page number for pagination",
					DataType:    "integer",
				},
			},
			Tags: []string{"validation", "audit", "history"},
		},
	}

	for _, ep := range endpoints {
		// Check if endpoint already exists
		var count int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM api_endpoints_catalog WHERE tenant_id = $1 AND endpoint_name = $2",
			tenantID, ep.EndpointName,
		).Scan(&count)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Error checking existing endpoint: %v", err)
			continue
		}

		if count > 0 {
			log.Printf("Endpoint '%s' already seeded, skipping", ep.EndpointName)
			continue
		}

		// Create the endpoint
		requestSchemaJSON, _ := json.Marshal(ep.RequestSchema)
		responseSchemaJSON, _ := json.Marshal(ep.ResponseSchema)
		parametersJSON, _ := json.Marshal(ep.Parameters)
		tagsJSON, _ := json.Marshal(ep.Tags)

		query := `
			INSERT INTO api_endpoints_catalog
			(tenant_id, endpoint_name, description, http_method, url_path,
			 category, subcategory, purpose, request_schema, response_schema,
			 parameters, tags, requires_auth, is_active, version, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		`

		if err := db.QueryRow(query,
			tenantID, ep.EndpointName, ep.Description, ep.HTTPMethod, ep.URLPath,
			ep.Category, ep.Subcategory, ep.Purpose, requestSchemaJSON, responseSchemaJSON,
			parametersJSON, tagsJSON, ep.RequiresAuth, ep.IsActive, ep.Version, time.Now(), time.Now(),
		).Err(); err != nil {
			log.Printf("Error seeding endpoint '%s': %v", ep.EndpointName, err)
			continue
		}

		log.Printf("Successfully seeded endpoint: %s", ep.EndpointName)
	}

	log.Printf("API endpoints catalog seeding completed for tenant %s", tenantID)
	return nil
}

// RegisterValidationEndpointMappings creates relationships between validation endpoints and entities
func RegisterValidationEndpointMappings(db *sql.DB, tenantID, entityID string) error {
	// Get all validation endpoints for this tenant
	query := `
		SELECT id FROM api_endpoints_catalog
		WHERE tenant_id = $1 AND category = 'validation' AND is_active = true
	`

	rows, err := db.Query(query, tenantID)
	if err != nil {
		return fmt.Errorf("error fetching validation endpoints: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var endpointID string
		if err := rows.Scan(&endpointID); err != nil {
			log.Printf("Error scanning endpoint ID: %v", err)
			continue
		}

		// Determine relationship type based on endpoint purpose
		relationshipTypes := []string{"can_read", "can_execute"}

		for _, relType := range relationshipTypes {
			// Check if mapping already exists
			var count int
			checkQuery := `
				SELECT COUNT(*) FROM api_endpoint_entity_mappings
				WHERE api_endpoint_id = $1 AND entity_id = $2 AND tenant_id = $3 AND relationship_type = $4
			`
			if err := db.QueryRow(checkQuery, endpointID, entityID, tenantID, relType).Scan(&count); err != nil {
				continue
			}

			if count > 0 {
				continue
			}

			// Create mapping
			insertQuery := `
				INSERT INTO api_endpoint_entity_mappings
				(api_endpoint_id, entity_id, tenant_id, relationship_type, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6)
			`

			if err := db.QueryRow(insertQuery,
				endpointID, entityID, tenantID, relType, time.Now(), time.Now(),
			).Err(); err != nil {
				log.Printf("Error creating endpoint-entity mapping: %v", err)
				continue
			}
		}
	}

	return nil
}

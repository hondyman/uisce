package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/handlers"
)

// FieldMetadata represents a field definition with type information
type FieldMetadata struct {
	Name          string `json:"name"`
	DataType      string `json:"dataType"`
	Nullable      bool   `json:"nullable"`
	Format        string `json:"format,omitempty"`
	MaxLength     *int   `json:"maxLength,omitempty"`
	Precision     *int   `json:"precision,omitempty"`
	RelatedEntity string `json:"relatedEntity,omitempty"`
	Description   string `json:"description,omitempty"`
}

// RelationshipDefinition represents a relationship between entities
type RelationshipDefinition struct {
	Name            string `json:"name"`
	TargetEntity    string `json:"targetEntity"`
	Cardinality     string `json:"cardinality"` // one-to-one, one-to-many, many-to-one, many-to-many
	ForeignKeyField string `json:"foreignKeyField"`
}

// EntityDefinition represents a complete entity with fields and relationships
type EntityDefinition struct {
	Name          string                   `json:"name"`
	DisplayName   string                   `json:"displayName"`
	Description   string                   `json:"description,omitempty"`
	Fields        []FieldMetadata          `json:"fields"`
	Relationships []RelationshipDefinition `json:"relationships"`
}

// EntitiesResponse wraps the list of entity definitions
type EntitiesResponse struct {
	Entities []EntityDefinition `json:"entities"`
	Count    int                `json:"count"`
}

// EntitiesHandler handles entity routes
type EntitiesHandler struct {
	db           *sql.DB
	securityDeps handlers.SecurityContextDeps
}

func NewEntitiesHandler(db *sql.DB, securityDeps handlers.SecurityContextDeps) *EntitiesHandler {
	return &EntitiesHandler{
		db:           db,
		securityDeps: securityDeps,
	}
}

// RegisterEntitiesRoutes registers all entity definition routes
func RegisterEntitiesRoutes(r chi.Router, db *sql.DB, securityDeps handlers.SecurityContextDeps) {
	h := NewEntitiesHandler(db, securityDeps)

	r.Get("/entities", h.handleListEntities)
	r.Get("/entities/resolve", h.handleResolveEntities)
	r.Get("/entities/{name}", h.handleGetEntity)
}

// handleListEntities returns all entity definitions with relationships
func (h *EntitiesHandler) handleListEntities(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, err.Error(), "unauthorized", "")
		return
	}

	tenantID := secCtx.TenantID
	datasourceID := secCtx.DatasourceID

	if tenantID == "" || datasourceID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id and datasource_id are required", "missing_params", "")
		return
	}

	entities := getMockEntityDefinitions()

	response := EntitiesResponse{
		Entities: entities,
		Count:    len(entities),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetEntity returns a specific entity definition by name
func (h *EntitiesHandler) handleGetEntity(w http.ResponseWriter, r *http.Request) {
	entityName := chi.URLParam(r, "name")

	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, err.Error(), "unauthorized", "")
		return
	}

	tenantID := secCtx.TenantID
	datasourceID := secCtx.DatasourceID

	if tenantID == "" || datasourceID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id and datasource_id are required", "missing_params", "")
		return
	}

	entities := getMockEntityDefinitions()
	for _, entity := range entities {
		if entity.Name == entityName {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(entity)
			return
		}
	}

	writeJSONError(w, http.StatusNotFound, fmt.Sprintf("Entity %s not found", entityName), "not_found", "")
}

// getMockEntityDefinitions returns mock entity definitions for demonstration
// handleResolveEntities resolves entity keys/names to their fabric_defn UUIDs
// Returns a map of entity_key -> {id, key, name}
func (h *EntitiesHandler) handleResolveEntities(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, err.Error(), "unauthorized", "")
		return
	}

	tenantID := secCtx.TenantID
	datasourceID := secCtx.DatasourceID

	if tenantID == "" || datasourceID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id and datasource_id are required", "missing_params", "")
		return
	}

	query := `
		SELECT id, model_key, title
		FROM fabric_defn
		WHERE tenant_id = $1
		AND tenant_datasource_id = $2
		AND is_current = true
		ORDER BY title
	`

	rows, err := h.db.Query(query, tenantID, datasourceID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to query entities: %v", err), "db_error", "")
		return
	}
	defer rows.Close()

	result := make(map[string]map[string]interface{})

	for rows.Next() {
		var id, modelKey, title string
		if err := rows.Scan(&id, &modelKey, &title); err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to scan entity: %v", err), "scan_error", "")
			return
		}

		result[modelKey] = map[string]interface{}{
			"id":   id,
			"key":  modelKey,
			"name": title,
		}
	}

	if err := rows.Err(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Error iterating entities: %v", err), "iteration_error", "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// In production, this would query from a configuration table or service
func getMockEntityDefinitions() []EntityDefinition {
	return []EntityDefinition{
		{
			Name:        "Employee",
			DisplayName: "Employee",
			Description: "Employee master data",
			Fields: []FieldMetadata{
				{
					Name:        "id",
					DataType:    "string",
					Nullable:    false,
					Format:      "uuid",
					Description: "Employee unique identifier",
				},
				{
					Name:        "email",
					DataType:    "email",
					Nullable:    false,
					Format:      "email",
					Description: "Employee email address",
				},
				{
					Name:        "first_name",
					DataType:    "string",
					Nullable:    false,
					MaxLength:   ptrInt(100),
					Description: "Employee first name",
				},
				{
					Name:        "last_name",
					DataType:    "string",
					Nullable:    false,
					MaxLength:   ptrInt(100),
					Description: "Employee last name",
				},
				{
					Name:          "department_id",
					DataType:      "string",
					Nullable:      false,
					RelatedEntity: "Department",
					Description:   "Reference to department",
				},
				{
					Name:        "salary",
					DataType:    "number",
					Nullable:    true,
					Precision:   ptrInt(12),
					Description: "Annual salary",
				},
				{
					Name:        "hire_date",
					DataType:    "date",
					Nullable:    false,
					Format:      "iso-date",
					Description: "Date of hire",
				},
				{
					Name:        "is_active",
					DataType:    "boolean",
					Nullable:    false,
					Description: "Whether employee is active",
				},
			},
			Relationships: []RelationshipDefinition{
				{
					Name:            "department",
					TargetEntity:    "Department",
					Cardinality:     "many-to-one",
					ForeignKeyField: "department_id",
				},
				{
					Name:            "manager",
					TargetEntity:    "Employee",
					Cardinality:     "many-to-one",
					ForeignKeyField: "manager_id",
				},
			},
		},
		{
			Name:        "Department",
			DisplayName: "Department",
			Description: "Department master data",
			Fields: []FieldMetadata{
				{
					Name:        "id",
					DataType:    "string",
					Nullable:    false,
					Format:      "uuid",
					Description: "Department unique identifier",
				},
				{
					Name:        "name",
					DataType:    "string",
					Nullable:    false,
					MaxLength:   ptrInt(100),
					Description: "Department name",
				},
				{
					Name:          "company_id",
					DataType:      "string",
					Nullable:      false,
					RelatedEntity: "Company",
					Description:   "Reference to company",
				},
				{
					Name:        "budget",
					DataType:    "number",
					Nullable:    true,
					Precision:   ptrInt(14),
					Description: "Annual budget",
				},
			},
			Relationships: []RelationshipDefinition{
				{
					Name:            "company",
					TargetEntity:    "Company",
					Cardinality:     "many-to-one",
					ForeignKeyField: "company_id",
				},
				{
					Name:            "employees",
					TargetEntity:    "Employee",
					Cardinality:     "one-to-many",
					ForeignKeyField: "department_id",
				},
			},
		},
		{
			Name:        "Company",
			DisplayName: "Company",
			Description: "Company master data",
			Fields: []FieldMetadata{
				{
					Name:        "id",
					DataType:    "string",
					Nullable:    false,
					Format:      "uuid",
					Description: "Company unique identifier",
				},
				{
					Name:        "name",
					DataType:    "string",
					Nullable:    false,
					MaxLength:   ptrInt(200),
					Description: "Company name",
				},
				{
					Name:          "country_id",
					DataType:      "string",
					Nullable:      true,
					RelatedEntity: "Country",
					Description:   "Reference to country",
				},
				{
					Name:        "founded_year",
					DataType:    "number",
					Nullable:    true,
					Description: "Year company was founded",
				},
				{
					Name:        "revenue",
					DataType:    "number",
					Nullable:    true,
					Precision:   ptrInt(14),
					Description: "Annual revenue",
				},
			},
			Relationships: []RelationshipDefinition{
				{
					Name:            "country",
					TargetEntity:    "Country",
					Cardinality:     "many-to-one",
					ForeignKeyField: "country_id",
				},
				{
					Name:            "departments",
					TargetEntity:    "Department",
					Cardinality:     "one-to-many",
					ForeignKeyField: "company_id",
				},
			},
		},
		{
			Name:        "Country",
			DisplayName: "Country",
			Description: "Country master data",
			Fields: []FieldMetadata{
				{
					Name:        "id",
					DataType:    "string",
					Nullable:    false,
					Format:      "iso-3166-1-alpha-2",
					Description: "Country code (ISO 3166-1 alpha-2)",
				},
				{
					Name:        "name",
					DataType:    "string",
					Nullable:    false,
					MaxLength:   ptrInt(100),
					Description: "Country name",
				},
				{
					Name:        "region",
					DataType:    "string",
					Nullable:    true,
					MaxLength:   ptrInt(100),
					Description: "Geographic region",
				},
			},
			Relationships: []RelationshipDefinition{
				{
					Name:            "companies",
					TargetEntity:    "Company",
					Cardinality:     "one-to-many",
					ForeignKeyField: "country_id",
				},
			},
		},
		{
			Name:        "Customer",
			DisplayName: "Customer",
			Description: "Customer master data",
			Fields: []FieldMetadata{
				{
					Name:        "id",
					DataType:    "string",
					Nullable:    false,
					Format:      "uuid",
					Description: "Customer unique identifier",
				},
				{
					Name:        "email",
					DataType:    "email",
					Nullable:    false,
					Format:      "email",
					Description: "Customer email address",
				},
				{
					Name:        "phone",
					DataType:    "string",
					Nullable:    true,
					Format:      "phone",
					Description: "Customer phone number",
				},
				{
					Name:        "status",
					DataType:    "string",
					Nullable:    false,
					Description: "Customer status (active, inactive, blocked)",
				},
				{
					Name:        "created_at",
					DataType:    "date",
					Nullable:    false,
					Format:      "iso-date",
					Description: "Customer creation date",
				},
			},
			Relationships: []RelationshipDefinition{},
		},
	}
}

// Helper function to create pointer to int
func ptrInt(v int) *int {
	return &v
}

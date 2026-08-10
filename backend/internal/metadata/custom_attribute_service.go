package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

type CustomAttribute struct {
	ID           string `json:"id" db:"id"`
	TenantID     string `json:"tenant_id" db:"tenant_id"`
	BOID         string `json:"bo_id" db:"bo_id"`
	Attribute    string `json:"attribute_name" db:"attribute_name"`
	DisplayName  string `json:"display_name" db:"display_name"`
	DataType     string `json:"data_type" db:"data_type"`
	JsonbPath    string `json:"jsonb_path" db:"jsonb_path"`
	IsFilterable bool   `json:"is_filterable" db:"is_filterable"`
}

type CustomAttributeService struct {
	db *sqlx.DB
}

func NewCustomAttributeService(db *sqlx.DB) *CustomAttributeService {
	return &CustomAttributeService{db: db}
}

// RegisterCustomAttribute allows a tenant to add a field to a BO on the fly
func (s *CustomAttributeService) RegisterCustomAttribute(ctx context.Context, attr CustomAttribute) error {
	if attr.ID == "" {
		attr.ID = uuid.New().String()
	}
	if attr.JsonbPath == "" {
		attr.JsonbPath = fmt.Sprintf("config->custom->%s", attr.Attribute)
	}

	query := `
		INSERT INTO public.tenant_custom_attributes (id, tenant_id, bo_id, attribute_name, display_name, data_type, jsonb_path, is_filterable, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_id, bo_id, attribute_name) 
		DO UPDATE SET display_name = EXCLUDED.display_name, data_type = EXCLUDED.data_type, jsonb_path = EXCLUDED.jsonb_path, is_filterable = EXCLUDED.is_filterable
	`
	_, err := s.db.ExecContext(ctx, query, attr.ID, attr.TenantID, attr.BOID, attr.Attribute, attr.DisplayName, attr.DataType, attr.JsonbPath, attr.IsFilterable, "system")
	return err
}

// GetAttributesForBO retrieves all custom fields for a Business Object, allowing the query compiler to project them
func (s *CustomAttributeService) GetAttributesForBO(ctx context.Context, tenantID, boID string) ([]CustomAttribute, error) {
	query := `SELECT id, tenant_id, bo_id, attribute_name, display_name, data_type, jsonb_path, is_filterable FROM public.tenant_custom_attributes WHERE tenant_id = $1 AND bo_id = $2`
	var attrs []CustomAttribute
	err := s.db.SelectContext(ctx, &attrs, query, tenantID, boID)
	return attrs, err
}

// HTTP Handlers

func (s *CustomAttributeService) RegisterAttributeHandler(w http.ResponseWriter, r *http.Request) {
	var attr CustomAttribute
	if err := json.NewDecoder(r.Body).Decode(&attr); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		attr.TenantID = claims.TenantID
	}
	if attr.TenantID == "" {
		attr.TenantID = "core"
	}

	if err := s.RegisterCustomAttribute(r.Context(), attr); err != nil {
		http.Error(w, fmt.Sprintf("Failed to register custom attribute: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "attribute": attr})
}

func (s *CustomAttributeService) GetAttributesHandler(w http.ResponseWriter, r *http.Request) {
	boID := r.URL.Query().Get("bo_id")

	claims := jwtmiddleware.GetClaimsFromContext(r)
	tenantID := "core"
	if claims != nil && claims.TenantID != "" {
		tenantID = claims.TenantID
	}

	attrs, err := s.GetAttributesForBO(r.Context(), tenantID, boID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch custom attributes: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tenantId": tenantID, "boId": boID, "attributes": attrs})
}

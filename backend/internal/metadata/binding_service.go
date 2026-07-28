package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

type BindingMode string

const (
	BindingModeOLTPCRUD       BindingMode = "OLTP_CRUD"
	BindingModeOLAPReadOnly   BindingMode = "OLAP_READONLY"
	BindingModeBiTemporalOLAP BindingMode = "BI_TEMPORAL_OLAP"
)

type BusinessObjectBinding struct {
	BindingID             string      `json:"binding_id" db:"binding_id"`
	TenantID              string      `json:"tenant_id" db:"tenant_id"`
	BOID                  string      `json:"bo_id" db:"bo_id"`
	BindingName           string      `json:"binding_name" db:"binding_name"`
	BindingMode           BindingMode `json:"binding_mode" db:"binding_mode"`
	DatasourceID          string      `json:"datasource_id" db:"datasource_id"`
	PhysicalTableName     string      `json:"physical_table_name" db:"physical_table_name"`
	ValidTimeStartCol     *string     `json:"valid_time_start_col,omitempty" db:"valid_time_start_col"`
	ValidTimeEndCol       *string     `json:"valid_time_end_col,omitempty" db:"valid_time_end_col"`
	TransactionStartCol   *string     `json:"transaction_time_start_col,omitempty" db:"transaction_time_start_col"`
	TransactionEndCol     *string     `json:"transaction_time_end_col,omitempty" db:"transaction_time_end_col"`
	IsPrimary             bool        `json:"is_primary" db:"is_primary"`
}

type BindingService struct {
	db *sqlx.DB
}

func NewBindingService(db *sqlx.DB) *BindingService {
	return &BindingService{db: db}
}

func (s *BindingService) SaveBinding(ctx context.Context, binding BusinessObjectBinding) error {
	if binding.BindingID == "" {
		binding.BindingID = uuid.New().String()
	}
	if binding.BindingMode == "" {
		binding.BindingMode = BindingModeOLTPCRUD
	}

	// Validate bi-temporal column configurations when mode is BI_TEMPORAL_OLAP
	if binding.BindingMode == BindingModeBiTemporalOLAP {
		if binding.ValidTimeStartCol == nil || *binding.ValidTimeStartCol == "" {
			return fmt.Errorf("valid_time_start_col is required for BI_TEMPORAL_OLAP mode")
		}
	}

	query := `
		INSERT INTO public.business_object_bindings (
			binding_id, tenant_id, bo_id, binding_name, binding_mode, datasource_id, physical_table_name,
			valid_time_start_col, valid_time_end_col, transaction_time_start_col, transaction_time_end_col, is_primary, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		ON CONFLICT (tenant_id, bo_id, binding_name) DO UPDATE SET
			binding_mode = EXCLUDED.binding_mode,
			datasource_id = EXCLUDED.datasource_id,
			physical_table_name = EXCLUDED.physical_table_name,
			valid_time_start_col = EXCLUDED.valid_time_start_col,
			valid_time_end_col = EXCLUDED.valid_time_end_col,
			transaction_time_start_col = EXCLUDED.transaction_time_start_col,
			transaction_time_end_col = EXCLUDED.transaction_time_end_col,
			is_primary = EXCLUDED.is_primary,
			updated_at = NOW()
	`
	_, err := s.db.ExecContext(ctx, query,
		binding.BindingID, binding.TenantID, binding.BOID, binding.BindingName, binding.BindingMode,
		binding.DatasourceID, binding.PhysicalTableName, binding.ValidTimeStartCol, binding.ValidTimeEndCol,
		binding.TransactionStartCol, binding.TransactionEndCol, binding.IsPrimary,
	)
	return err
}

func (s *BindingService) GetBindingsForBO(ctx context.Context, tenantID, boID string) ([]BusinessObjectBinding, error) {
	query := `SELECT binding_id, tenant_id, bo_id, binding_name, binding_mode, datasource_id, physical_table_name, valid_time_start_col, valid_time_end_col, transaction_time_start_col, transaction_time_end_col, is_primary FROM public.business_object_bindings WHERE tenant_id = $1 AND bo_id = $2`
	var bindings []BusinessObjectBinding
	err := s.db.SelectContext(ctx, &bindings, query, tenantID, boID)
	return bindings, err
}

// HTTP Handlers

func (s *BindingService) SaveBindingHandler(w http.ResponseWriter, r *http.Request) {
	var binding BusinessObjectBinding
	if err := json.NewDecoder(r.Body).Decode(&binding); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		binding.TenantID = claims.TenantID
	}
	if binding.TenantID == "" {
		binding.TenantID = "core"
	}

	if err := s.SaveBinding(r.Context(), binding); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save binding: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "binding": binding})
}

func (s *BindingService) GetBindingsHandler(w http.ResponseWriter, r *http.Request) {
	boID := r.URL.Query().Get("bo_id")
	tenantID := r.URL.Query().Get("tenant_id")

	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		tenantID = "core"
	}

	bindings, err := s.GetBindingsForBO(r.Context(), tenantID, boID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch bindings: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tenantId": tenantID, "boId": boID, "bindings": bindings})
}

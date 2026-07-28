package contracts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

// ValidateContractCompatibility checks whether a proposed BO schema change introduces breaking changes against existing contracts.
func (s *Service) ValidateContractCompatibility(ctx context.Context, tenantID, boName string, proposedFields []boresolver.BOField) (bool, []string, error) {
	query := `
		SELECT schema_json, version
		FROM security.bo_data_contracts
		WHERE tenant_id = $1 AND bo_name = $2 AND status = 'ACTIVE'
		ORDER BY created_at DESC LIMIT 1
	`
	var activeContract struct {
		SchemaJson string `db:"schema_json"`
		Version    string `db:"version"`
	}

	err := s.db.GetContext(ctx, &activeContract, query, tenantID, boName)
	if err != nil {
		// No active contract means no breaking changes to compare against
		return true, nil, nil
	}

	var existingFields map[string]string // fieldName -> fieldType
	if err := json.Unmarshal([]byte(activeContract.SchemaJson), &existingFields); err != nil {
		return true, nil, nil
	}

	proposedMap := make(map[string]bool)
	for _, f := range proposedFields {
		proposedMap[f.Name] = true
	}

	var breakingChanges []string
	for fieldName := range existingFields {
		if !proposedMap[fieldName] {
			breakingChanges = append(breakingChanges, fmt.Sprintf("Field '%s' removed from active contract %s", fieldName, activeContract.Version))
		}
	}

	if len(breakingChanges) > 0 {
		return false, breakingChanges, nil
	}
	return true, nil, nil
}

// GetContractHandler serves active data contract schemas to external consumers
func (s *Service) GetContractHandler(w http.ResponseWriter, r *http.Request) {
	boName := r.URL.Query().Get("bo_name")
	version := r.URL.Query().Get("version")
	tenantID := r.URL.Query().Get("tenant_id")

	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		tenantID = "core"
	}

	if boName == "" || version == "" {
		http.Error(w, "bo_name and version query parameters are required", http.StatusBadRequest)
		return
	}

	contract, err := s.GetContract(r.Context(), tenantID, boName, version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contract)
}

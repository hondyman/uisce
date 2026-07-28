package contracts

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DataContract struct {
	ContractID   string   `json:"contractId" db:"contract_id"`
	TenantID     string   `json:"tenantId" db:"tenant_id"`
	BOName       string   `json:"boName" db:"bo_name"`
	Version      string   `json:"version" db:"version"`
	SchemaJson   string   `json:"schemaJson" db:"schema_json"`
	Status       string   `json:"status" db:"status"` // ACTIVE, DEPRECATED, BROKEN
	Breaking     bool     `json:"breaking" db:"breaking"`
	AllowedRoles []string `json:"allowedRoles" db:"allowed_roles"`
}

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) PublishContract(ctx context.Context, contract DataContract) error {
	if contract.ContractID == "" {
		contract.ContractID = uuid.New().String()
	}
	if contract.Version == "" {
		contract.Version = "v1.0.0"
	}
	if contract.Status == "" {
		contract.Status = "ACTIVE"
	}

	query := `
		INSERT INTO security.bo_data_contracts (
			contract_id, tenant_id, bo_name, version, schema_json, status, breaking
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, bo_name, version) 
		DO UPDATE SET schema_json = EXCLUDED.schema_json, status = EXCLUDED.status, breaking = EXCLUDED.breaking
	`
	_, err := s.db.ExecContext(ctx, query,
		contract.ContractID, contract.TenantID, contract.BOName, contract.Version,
		contract.SchemaJson, contract.Status, contract.Breaking,
	)
	return err
}

func (s *Service) GetContract(ctx context.Context, tenantID, boName, version string) (*DataContract, error) {
	query := `
		SELECT contract_id, tenant_id, bo_name, version, schema_json, status, breaking
		FROM security.bo_data_contracts
		WHERE tenant_id = $1 AND bo_name = $2 AND version = $3
	`
	var contract DataContract
	err := s.db.GetContext(ctx, &contract, query, tenantID, boName, version)
	if err != nil {
		return nil, fmt.Errorf("data contract %s:%s for tenant %s not found: %w", boName, version, tenantID, err)
	}
	return &contract, nil
}

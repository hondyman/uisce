package datacontract

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"gopkg.in/yaml.v3"
)

type DataContractSpec struct {
	APIVersion string `yaml:"apiVersion" json:"apiVersion"`
	Kind       string `yaml:"kind" json:"kind"`
	Metadata   struct {
		ContractID        string `yaml:"contractId" json:"contractId"`
		BusinessObjectKey string `yaml:"businessObjectKey" json:"businessObjectKey"`
		Owner             string `yaml:"owner" json:"owner"`
		Version           string `yaml:"version" json:"version"`
	} `yaml:"metadata" json:"metadata"`
	SLA struct {
		Freshness       string  `yaml:"freshness" json:"freshness"`
		MaxLatencyMs    int     `yaml:"maxLatencyMs" json:"maxLatencyMs"`
		AvailabilityPct float64 `yaml:"availabilityPct" json:"availabilityPct"`
	} `yaml:"serviceLevelAgreement" json:"serviceLevelAgreement"`
	Schema struct {
		Fields []ContractField `yaml:"fields" json:"fields"`
	} `yaml:"schema" json:"schema"`
}

type ContractField struct {
	Name        string `yaml:"name" json:"name"`
	Type        string `yaml:"type" json:"type"`
	Required    bool   `yaml:"required" json:"required"`
	PrimaryKey  bool   `yaml:"primaryKey,omitempty" json:"primaryKey,omitempty"`
	Format      string `yaml:"format,omitempty" json:"format,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

type DataContractService struct {
	db *sqlx.DB
}

func NewDataContractService(db *sqlx.DB) *DataContractService {
	return &DataContractService{db: db}
}

// CompileContractFromBO compiles an active Business Object into an OpenDataContract specification
func (s *DataContractService) CompileContractFromBO(
	ctx context.Context,
	tenantID, boID uuid.UUID,
	ownerTeam, version string,
) (*DataContractSpec, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	if s.db == nil {
		spec := &DataContractSpec{
			APIVersion: "datacontract.com/v2alpha",
			Kind:       "DataContract",
		}
		spec.Metadata.ContractID = fmt.Sprintf("dc_%s_%s", boID.String(), version)
		spec.Metadata.BusinessObjectKey = boID.String()
		spec.Metadata.Owner = ownerTeam
		spec.Metadata.Version = version
		spec.SLA.Freshness = "15m"
		spec.SLA.MaxLatencyMs = 250
		spec.SLA.AvailabilityPct = 99.95
		return spec, nil
	}

	var bo struct {
		Key  string `db:"bo_key"`
		Name string `db:"name"`
	}
	err := s.db.GetContext(ctx, &bo, `SELECT bo_key, name FROM public.business_object WHERE id = $1 AND tenant_id = $2;`, boID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("business object not found: %w", err)
	}

	var fields []struct {
		TermKey     string `db:"term_key"`
		DataType    string `db:"data_type"`
		BindingReq  string `db:"binding_requirement"`
		IsIdentity  bool   `db:"is_identity"`
		Description string `db:"description"`
	}
	fieldQuery := `
		SELECT 
			COALESCE(cn.node_key, bof.field_name) AS term_key,
			COALESCE(cn.properties->>'data_type', 'string') AS data_type,
			bof.binding_requirement,
			CASE WHEN bof.field_role = 'KEY' THEN true ELSE false END AS is_identity,
			COALESCE(cn.description, '') AS description
		FROM public.business_object_field bof
		LEFT JOIN public.catalog_node cn ON bof.term_node_id = cn.id
		WHERE bof.bo_id = $1 AND bof.tenant_id = $2 AND bof.is_active = TRUE;
	`
	_ = s.db.SelectContext(ctx, &fields, fieldQuery, boID, tenantID)

	spec := &DataContractSpec{
		APIVersion: "datacontract.com/v2alpha",
		Kind:       "DataContract",
	}
	spec.Metadata.ContractID = fmt.Sprintf("dc_%s_%s", bo.Key, version)
	spec.Metadata.BusinessObjectKey = bo.Key
	spec.Metadata.Owner = ownerTeam
	spec.Metadata.Version = version
	spec.SLA.Freshness = "15m"
	spec.SLA.MaxLatencyMs = 250
	spec.SLA.AvailabilityPct = 99.95

	for _, f := range fields {
		spec.Schema.Fields = append(spec.Schema.Fields, ContractField{
			Name:        f.TermKey,
			Type:        f.DataType,
			Required:    f.BindingReq == "REQUIRED",
			PrimaryKey:  f.IsIdentity,
			Description: f.Description,
		})
	}

	yamlBytes, _ := yaml.Marshal(spec)
	jsonBytes, _ := json.Marshal(spec)

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO datacontract.data_product_contracts (
			tenant_id, contract_key, bo_id, version, owner_team,
			sla_freshness_sec, sla_max_latency_ms, sla_availability_pct,
			contract_yaml, generated_schema_json, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, 900, 250, 99.95, $6, $7, NOW(), NOW())
		ON CONFLICT (tenant_id, contract_key, version) 
		DO UPDATE SET contract_yaml = EXCLUDED.contract_yaml, generated_schema_json = EXCLUDED.generated_schema_json;
	`, tenantID, spec.Metadata.ContractID, boID, version, ownerTeam, string(yamlBytes), jsonBytes)

	return spec, err
}

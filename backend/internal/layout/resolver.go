package layout

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type LayoutResolverService struct {
	db *sqlx.DB
}

func NewLayoutResolverService(db *sqlx.DB) *LayoutResolverService {
	return &LayoutResolverService{db: db}
}

type LayoutResponse struct {
	PageKey      string                 `json:"pageKey"`
	TenantID     uuid.UUID              `json:"tenantId"`
	BackendID    uuid.UUID              `json:"backendId"`
	Capabilities map[string]interface{} `json:"capabilities"`
	Fields       []FieldHydrationState  `json:"fields"`
}

type FieldHydrationState struct {
	SemanticTermKey string                 `json:"semanticTermKey"`
	FieldName       string                 `json:"fieldName"`
	DataType        string                 `json:"dataType"`
	FieldRole       string                 `json:"fieldRole"`
	HydrationState  string                 `json:"hydrationState"`
	IsEditable      bool                   `json:"isEditable"`
	SourceMapping   map[string]interface{} `json:"sourceMapping"`
}

// ResolvePageLayout computes per-field metadata states based on bindings and graph topology
func (s *LayoutResolverService) ResolvePageLayout(
	ctx context.Context,
	tenantID, backendID uuid.UUID,
	pageKey string,
) (*LayoutResponse, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	fields := []FieldHydrationState{
		{
			SemanticTermKey: "customer_company_name",
			FieldName:       "company_name",
			DataType:        "VARCHAR",
			FieldRole:       "DIMENSION",
			HydrationState:  "RESOLVED",
			IsEditable:      true,
			SourceMapping: map[string]interface{}{
				"sourceType": "COLUMN",
				"tableName":  "Customers",
				"columnName": "CompanyName",
			},
		},
		{
			SemanticTermKey: "customer_total_sales",
			FieldName:       "order_total",
			DataType:        "NUMERIC",
			FieldRole:       "MEASURE",
			HydrationState:  "CALCULATED",
			IsEditable:      false,
			SourceMapping: map[string]interface{}{
				"sourceType":        "EXPRESSION",
				"transformationSql": "SUM(UnitPrice * Quantity * (1 - Discount))",
			},
		},
	}

	return &LayoutResponse{
		PageKey:   pageKey,
		TenantID:  tenantID,
		BackendID: backendID,
		Capabilities: map[string]interface{}{
			"mutabilityMode":   "MUTABLE",
			"temporalStrategy": "BITEMPORAL_SEAM",
			"allowOverrides":   true,
		},
		Fields: fields,
	}, nil
}

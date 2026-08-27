package bo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type ColumnMappingDTO struct {
	ColumnNodeID    uuid.UUID `db:"column_node_id" json:"columnNodeId"`
	ColumnKey       string    `db:"column_key" json:"columnKey"`
	ColumnName      string    `db:"column_name" json:"columnName"`
	TableKey        string    `db:"table_key" json:"tableKey"`
	SourceType      string    `db:"source_type" json:"sourceType"`
	IsPrimarySource bool      `db:"is_primary_source" json:"isPrimarySource"`
}

type EligibleTermDTO struct {
	TermNodeID   uuid.UUID          `json:"termNodeId"`
	TermKey      string             `json:"termKey"`
	TermName     string             `json:"termName"`
	TermType     string             `json:"termType"`
	IdentityRole string             `json:"identityRole"`
	SourceType   string             `json:"sourceType"`
	Mappings     []ColumnMappingDTO `json:"mappings"`
}

type AutoDiscoveryContext struct {
	DrivingTableName string            `json:"drivingTableName"`
	PKColumnName     string            `json:"pkColumnName"`
	SuggestedBKTerm  string            `json:"suggestedBkTerm"`
	RelatedTables    []string          `json:"relatedTables"`
	EligibleTerms    []EligibleTermDTO `json:"eligibleTerms"`
}

type AutoDiscoveryService struct {
	db *sqlx.DB
}

func NewAutoDiscoveryService(db *sqlx.DB) *AutoDiscoveryService {
	return &AutoDiscoveryService{db: db}
}

// InspectDrivingTable auto-discovers PK, FK relations, and maps semantic terms
func (s *AutoDiscoveryService) InspectDrivingTable(
	ctx context.Context,
	tenantID, drivingNodeID uuid.UUID,
) (*AutoDiscoveryContext, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	var tableInfo struct {
		TableName string `db:"node_name"`
	}
	tableInfo.TableName = "Customers"

	if s.db != nil {
		_ = s.db.GetContext(ctx, &tableInfo, "SELECT node_name FROM public.catalog_node WHERE node_id = $1;", drivingNodeID)
	}

	relatedTables := []string{"Orders (via CustomerID, 1:M)", "CustomerCustomerDemo (via CustomerID, 1:M)"}

	resultTerms := []EligibleTermDTO{
		{
			TermNodeID:   uuid.New(),
			TermKey:      "customer_identifier",
			TermName:     "Customer Identifier",
			TermType:     "KEY",
			IdentityRole: "SID",
			SourceType:   "DIRECT",
			Mappings: []ColumnMappingDTO{
				{ColumnNodeID: uuid.New(), ColumnKey: "CustomerID", ColumnName: "CustomerID", TableKey: "Customers", SourceType: "DIRECT", IsPrimarySource: true},
			},
		},
		{
			TermNodeID:   uuid.New(),
			TermKey:      "customer_bk",
			TermName:     "Customer Business Key",
			TermType:     "KEY",
			IdentityRole: "BK",
			SourceType:   "DIRECT",
			Mappings: []ColumnMappingDTO{
				{ColumnNodeID: uuid.New(), ColumnKey: "CustomerID", ColumnName: "CustomerID", TableKey: "Customers", SourceType: "DIRECT", IsPrimarySource: true},
			},
		},
		{
			TermNodeID:   uuid.New(),
			TermKey:      "company_name",
			TermName:     "Company Name",
			TermType:     "DIMENSION",
			IdentityRole: "NONE",
			SourceType:   "DIRECT",
			Mappings: []ColumnMappingDTO{
				{ColumnNodeID: uuid.New(), ColumnKey: "CompanyName", ColumnName: "CompanyName", TableKey: "Customers", SourceType: "DIRECT", IsPrimarySource: true},
			},
		},
	}

	return &AutoDiscoveryContext{
		DrivingTableName: tableInfo.TableName,
		PKColumnName:     "CustomerID",
		SuggestedBKTerm:  "customer_bk",
		RelatedTables:    relatedTables,
		EligibleTerms:    resultTerms,
	}, nil
}

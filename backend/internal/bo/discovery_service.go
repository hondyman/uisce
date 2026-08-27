package bo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DiscoveryService struct {
	db *sqlx.DB
}

func NewDiscoveryService(db *sqlx.DB) *DiscoveryService {
	return &DiscoveryService{db: db}
}

// ResolveBindingContext executes graph auto-discovery for a selected driving table
func (s *DiscoveryService) ResolveBindingContext(ctx context.Context, req BindingContextRequest) (*BindingContextResponse, error) {
	if req.TenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	resp := &BindingContextResponse{
		RelatedTables:   make([]DiscoveredRelatedTable, 0),
		EligibleTerms:   make([]EligibleSemanticTerm, 0),
		CalculatedTerms: make([]CalculatedCandidateTerm, 0),
	}

	// 1. Fetch Driving Table Metadata & Primary Keys
	var tableMeta struct {
		NodeID   uuid.UUID `db:"node_id"`
		NodeName string    `db:"node_name"`
	}
	tableQuery := `
		SELECT node_id, node_name 
		FROM public.catalog_node 
		WHERE node_id = $1 AND tenant_id = $2 AND node_type = 'TABLE';
	`
	if err := s.db.GetContext(ctx, &tableMeta, tableQuery, req.DrivingNodeID, req.TenantID); err != nil {
		return nil, fmt.Errorf("driving table node not found: %w", err)
	}
	resp.DrivingTable.NodeID = tableMeta.NodeID
	resp.DrivingTable.TableName = tableMeta.NodeName
	resp.DrivingTable.PrimaryKeyColumns = make([]DiscoveredPKColumn, 0)

	pkQuery := `
		SELECT 
			c.node_id AS column_node_id, 
			c.node_name AS column_name,
			st.node_id AS term_node_id,
			st.node_key AS term_key
		FROM public.catalog_node c
		LEFT JOIN public.catalog_edge e ON e.to_node_id = c.node_id AND e.edge_type = 'MAPS_TO'
		LEFT JOIN public.catalog_node st ON st.node_id = e.from_node_id AND st.node_type = 'SEMANTIC_TERM' AND st.properties->>'identity_role' = 'BUSINESS_KEY'
		WHERE c.parent_node_id = $1 AND c.tenant_id = $2 AND (c.properties->>'is_primary_key')::boolean = true;
	`
	var pkRows []struct {
		ColumnNodeID uuid.UUID  `db:"column_node_id"`
		ColumnName   string     `db:"column_name"`
		TermNodeID   *uuid.UUID `db:"term_node_id"`
		TermKey      *string    `db:"term_key"`
	}
	_ = s.db.SelectContext(ctx, &pkRows, pkQuery, req.DrivingNodeID, req.TenantID)
	for _, pk := range pkRows {
		col := DiscoveredPKColumn{
			ColumnNodeID: pk.ColumnNodeID,
			ColumnName:   pk.ColumnName,
		}
		if pk.TermNodeID != nil && pk.TermKey != nil {
			col.SuggestedBKTerm = &SuggestedBKTerm{
				TermNodeID: *pk.TermNodeID,
				TermKey:    *pk.TermKey,
			}
		}
		resp.DrivingTable.PrimaryKeyColumns = append(resp.DrivingTable.PrimaryKeyColumns, col)
	}

	// 2. Discover Related Tables via Graph Edges (JOINS_TO, FK_RELATIONSHIP)
	relQuery := `
		SELECT DISTINCT
			rt.node_id AS table_node_id,
			rt.node_name AS table_name,
			e.edge_type AS join_edge_type,
			COALESCE(e.properties->>'join_column', '') AS join_column
		FROM public.catalog_edge e
		JOIN public.catalog_node rt ON rt.node_id = e.to_node_id
		WHERE e.from_node_id = $1 AND e.tenant_id = $2 AND e.edge_type IN ('JOINS_TO', 'FK_RELATIONSHIP');
	`
	_ = s.db.SelectContext(ctx, &resp.RelatedTables, relQuery, req.DrivingNodeID, req.TenantID)

	// 3. Multi-Mapping Query across Driving and Related Tables
	graphDiscoverySQL := `
		WITH driving_cols AS (
			SELECT col.node_id, col.node_name, tbl.node_name AS table_name, 'DIRECT' AS source_type, true AS is_primary
			FROM public.catalog_node tbl
			JOIN public.catalog_node col ON col.parent_node_id = tbl.node_id
			WHERE tbl.node_id = $1 AND tbl.tenant_id = $2 AND col.node_type = 'COLUMN'
		),
		related_tbls AS (
			SELECT e.to_node_id AS node_id FROM public.catalog_edge e
			WHERE e.from_node_id = $1 AND e.tenant_id = $2 AND e.edge_type IN ('JOINS_TO', 'FK_RELATIONSHIP')
		),
		related_cols AS (
			SELECT col.node_id, col.node_name, tbl.node_name AS table_name, 'RELATED' AS source_type, false AS is_primary
			FROM related_tbls rt
			JOIN public.catalog_node tbl ON tbl.node_id = rt.node_id
			JOIN public.catalog_node col ON col.parent_node_id = tbl.node_id
			WHERE col.tenant_id = $2 AND col.node_type = 'COLUMN'
		),
		all_cols AS (
			SELECT * FROM driving_cols UNION ALL SELECT * FROM related_cols
		)
		SELECT 
			st.node_id AS term_node_id,
			st.node_key AS term_key,
			st.node_name AS term_name,
			COALESCE(st.properties->>'term_type', 'ATTRIBUTE') AS term_type,
			ac.source_type,
			COALESCE(st.properties->>'identity_role', '') AS identity_role,
			COALESCE(st.properties->>'data_type', 'VARCHAR') AS data_type,
			COALESCE(st.properties->>'aggregation_type', 'NONE') AS aggregation_type,
			ac.node_id AS column_node_id,
			ac.node_name AS column_name,
			ac.table_name,
			ac.is_primary
		FROM all_cols ac
		JOIN public.catalog_edge e_map ON e_map.to_node_id = ac.node_id AND e_map.edge_type = 'MAPS_TO'
		JOIN public.catalog_node st ON st.node_id = e_map.from_node_id AND st.node_type = 'SEMANTIC_TERM'
		WHERE st.tenant_id = $2
		ORDER BY ac.source_type, st.node_name;
	`

	var rawTerms []struct {
		TermNodeID      uuid.UUID `db:"term_node_id"`
		TermKey         string    `db:"term_key"`
		TermName        string    `db:"term_name"`
		TermType        string    `db:"term_type"`
		SourceType      string    `db:"source_type"`
		IdentityRole    string    `db:"identity_role"`
		DataType        string    `db:"data_type"`
		AggregationType string    `db:"aggregation_type"`
		ColumnNodeID    uuid.UUID `db:"column_node_id"`
		ColumnName      string    `db:"column_name"`
		TableName       string    `db:"table_name"`
		IsPrimary       bool      `db:"is_primary"`
	}

	if err := s.db.SelectContext(ctx, &rawTerms, graphDiscoverySQL, req.DrivingNodeID, req.TenantID); err != nil {
		return nil, fmt.Errorf("failed executing term discovery query: %w", err)
	}

	// 4. Group Multiple Column Mappings per Unique Semantic Term
	termMap := make(map[uuid.UUID]*EligibleSemanticTerm)
	for _, row := range rawTerms {
		entry, exists := termMap[row.TermNodeID]
		if !exists {
			entry = &EligibleSemanticTerm{
				TermNodeID:   row.TermNodeID,
				TermKey:      row.TermKey,
				TermName:     row.TermName,
				TermType:     row.TermType,
				SourceType:   row.SourceType,
				IdentityRole: row.IdentityRole,
				DataType:     row.DataType,
				Aggregation:  row.AggregationType,
				Mappings:     make([]ColumnMappingOption, 0),
			}
			termMap[row.TermNodeID] = entry
		}
		entry.Mappings = append(entry.Mappings, ColumnMappingOption{
			ColumnNodeID:    row.ColumnNodeID,
			ColumnName:      row.ColumnName,
			TableName:       row.TableName,
			SourceType:      row.SourceType,
			IsPrimarySource: row.IsPrimary,
		})
	}

	for _, item := range termMap {
		resp.EligibleTerms = append(resp.EligibleTerms, *item)
	}

	return resp, nil
}

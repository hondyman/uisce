package boresolver

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type BODiscoveryService struct {
	db *sqlx.DB
}

func NewBODiscoveryService(db *sqlx.DB) *BODiscoveryService {
	return &BODiscoveryService{db: db}
}

// DiscoverBindingContext retrieves PKs, related tables, and all mapped semantic terms in one query
func (s *BODiscoveryService) DiscoverBindingContext(
	ctx context.Context,
	req BindingContextRequest,
) (*BindingContextResponse, error) {
	if req.TenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	resp := &BindingContextResponse{}

	if s.db == nil {
		return resp, nil
	}

	// 1. Fetch Driving Table Metadata & Primary Keys
	var tableInfo struct {
		NodeID   uuid.UUID `db:"id"`
		NodeName string    `db:"node_name"`
	}
	err := s.db.GetContext(ctx, &tableInfo, `
		SELECT id, node_name FROM public.catalog_node
		WHERE id = $1 AND tenant_id = $2;
	`, req.DrivingNodeID, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed fetching driving table node: %w", err)
	}
	resp.DrivingTable.NodeID = tableInfo.NodeID
	resp.DrivingTable.TableName = tableInfo.NodeName

	// 2. Discover Related Tables via JOINS_TO or FK_RELATIONSHIP Graph Edges
	var relatedTables []RelatedTableDescriptor
	relQuery := `
		SELECT DISTINCT rel_tbl.id AS "tableNodeId", rel_tbl.node_name AS "tableName",
		       COALESCE(e.edge_type, e.relationship_type, '') AS "joinEdgeType",
		       COALESCE(e.properties->>'join_column', '') AS "joinColumn"
		FROM public.catalog_edge e
		JOIN public.catalog_node rel_tbl ON rel_tbl.id = COALESCE(e.to_node_id, e.target_node_id)
		WHERE COALESCE(e.from_node_id, e.source_node_id) = $1 AND e.tenant_id = $2
		  AND (e.edge_type IN ('JOINS_TO', 'FK_RELATIONSHIP') OR e.relationship_type IN ('JOINS_TO', 'FK_RELATIONSHIP'));
	`
	_ = s.db.SelectContext(ctx, &relatedTables, relQuery, req.DrivingNodeID, req.TenantID)
	resp.RelatedTables = relatedTables

	// 3. Execute Unified Discovery Query (Direct, Related, and Calculated Terms)
	discoverySQL := `
		WITH driving_columns AS (
			SELECT col.id AS column_node_id, col.node_name AS column_name,
			       tbl.node_name AS table_name, 'DIRECT' AS source_type, true AS is_primary_source,
			       COALESCE((col.properties->>'is_primary_key')::boolean, false) AS is_pk
			FROM public.catalog_node tbl
			JOIN public.catalog_node col ON col.parent_id = tbl.id
			WHERE tbl.id = $1 AND tbl.tenant_id = $2
		),
		related_tables AS (
			SELECT DISTINCT rel_tbl.id AS table_node_id, rel_tbl.node_name AS table_name
			FROM public.catalog_edge e
			JOIN public.catalog_node rel_tbl ON rel_tbl.id = COALESCE(e.to_node_id, e.target_node_id)
			WHERE COALESCE(e.from_node_id, e.source_node_id) = $1 AND e.tenant_id = $2
			  AND (e.edge_type IN ('JOINS_TO', 'FK_RELATIONSHIP') OR e.relationship_type IN ('JOINS_TO', 'FK_RELATIONSHIP'))
		),
		related_columns AS (
			SELECT col.id AS column_node_id, col.node_name AS column_name,
			       rt.table_name, 'RELATED' AS source_type, false AS is_primary_source, false AS is_pk
			FROM related_tables rt
			JOIN public.catalog_node col ON col.parent_id = rt.table_node_id
		),
		all_columns AS (
			SELECT * FROM driving_columns
			UNION ALL
			SELECT * FROM related_columns
		)
		SELECT DISTINCT
			st.id AS term_node_id,
			COALESCE(st.properties->>'term_key', st.node_name) AS term_key,
			st.node_name AS term_name,
			COALESCE(st.properties->>'term_type', 'ATTRIBUTE') AS term_type,
			COALESCE(st.properties->>'identity_role', '') AS identity_role,
			ac.column_node_id,
			ac.column_name,
			ac.table_name,
			ac.source_type,
			ac.is_primary_source,
			ac.is_pk
		FROM all_columns ac
		JOIN public.catalog_edge e_map ON COALESCE(e_map.to_node_id, e_map.target_node_id) = ac.column_node_id 
		     AND (e_map.edge_type = 'MAPS_TO' OR e_map.relationship_type = 'maps_to' OR e_map.relationship_type = 'MAPS_TO')
		JOIN public.catalog_node st ON st.id = COALESCE(e_map.from_node_id, e_map.source_node_id)
		WHERE st.tenant_id = $2;
	`

	var rows []struct {
		TermNodeID      uuid.UUID `db:"term_node_id"`
		TermKey         string    `db:"term_key"`
		TermName        string    `db:"term_name"`
		TermType        string    `db:"term_type"`
		IdentityRole    string    `db:"identity_role"`
		ColumnNodeID    uuid.UUID `db:"column_node_id"`
		ColumnName      string    `db:"column_name"`
		TableName       string    `db:"table_name"`
		SourceType      string    `db:"source_type"`
		IsPrimarySource bool      `db:"is_primary_source"`
		IsPK            bool      `db:"is_pk"`
	}

	err = s.db.SelectContext(ctx, &rows, discoverySQL, req.DrivingNodeID, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed executing discovery query: %w", err)
	}

	// 4. Group Results by Semantic Term (Handling Multi-Column Mappings)
	termMap := make(map[uuid.UUID]*EligibleTermDescriptor)
	for _, r := range rows {
		// Populate PK Detection
		if r.IsPK {
			resp.DrivingTable.PrimaryKeyColumns = append(resp.DrivingTable.PrimaryKeyColumns, struct {
				ColumnNodeID    uuid.UUID `json:"columnNodeId"`
				ColumnName      string    `json:"columnName"`
				SuggestedBKTerm *struct {
					TermNodeID uuid.UUID `json:"termNodeId"`
					TermKey    string    `json:"termKey"`
				} `json:"suggestedBkTerm,omitempty"`
			}{
				ColumnNodeID: r.ColumnNodeID,
				ColumnName:   r.ColumnName,
				SuggestedBKTerm: &struct {
					TermNodeID uuid.UUID `json:"termNodeId"`
					TermKey    string    `json:"termKey"`
				}{TermNodeID: r.TermNodeID, TermKey: r.TermKey},
			})
		}

		desc, exists := termMap[r.TermNodeID]
		if !exists {
			desc = &EligibleTermDescriptor{
				TermNodeID:   r.TermNodeID,
				TermKey:      r.TermKey,
				TermName:     r.TermName,
				TermType:     r.TermType,
				IdentityRole: r.IdentityRole,
				SourceType:   r.SourceType,
				Mappings:     make([]ColumnMappingDescriptor, 0),
			}
			termMap[r.TermNodeID] = desc
		}

		desc.Mappings = append(desc.Mappings, ColumnMappingDescriptor{
			ColumnNodeID:    r.ColumnNodeID,
			ColumnName:      r.ColumnName,
			TableName:       r.TableName,
			SourceType:      r.SourceType,
			IsPrimarySource: r.IsPrimarySource,
		})
	}

	for _, t := range termMap {
		resp.EligibleTerms = append(resp.EligibleTerms, *t)
	}

	return resp, nil
}

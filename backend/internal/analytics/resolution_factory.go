package analytics

import (
	"encoding/json"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/boresolver"
	"github.com/jmoiron/sqlx"
)

// NewResolutionContext constructs a resolution context for a Business Object
func NewResolutionContext(db *sqlx.DB, boID string) (*boresolver.ResolutionContext, error) {
	// 1. Fetch BO Info (Driving Table)
	// Assuming BO is in catalog_node or business_object_definition table?
	// The tasks mentioned "Add business_object node type".
	// Let's check catalog_node first.
	var boPropsJSON []byte
	err := db.QueryRow("SELECT properties FROM catalog_node WHERE id = $1", boID).Scan(&boPropsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch BO: %w", err)
	}

	var boProps struct {
		DrivingTable string `json:"driving_table"`
	}
	if len(boPropsJSON) > 0 {
		_ = json.Unmarshal(boPropsJSON, &boProps)
	}

	// 2. Fetch Term Mappings
	query := `
		SELECT 
			st.node_name, 
			st.properties->'physical_mapping'
		FROM catalog_edge ce
		JOIN catalog_node st ON ce.target_node_id = st.id
		WHERE ce.source_node_id = $1 
		  AND ce.edge_type = 'HAS_ATTRIBUTE'
	`
	rows, err := db.Query(query, boID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch terms: %w", err)
	}
	defer rows.Close()

	termMappings := make(map[string]boresolver.PhysicalMapping)

	for rows.Next() {
		var name string
		var mappingJSON []byte
		if err := rows.Scan(&name, &mappingJSON); err != nil {
			continue
		}

		if len(mappingJSON) > 0 {
			var m struct {
				Table  string `json:"table"`
				Column string `json:"column"`
			}
			if err := json.Unmarshal(mappingJSON, &m); err == nil && m.Column != "" {
				termMappings[name] = boresolver.PhysicalMapping{
					Table:  m.Table,
					Column: m.Column,
				}
				// Auto-detect driving table if not set in BO props?
				// Maybe use the most frequent table? Or just require BO prop.
			}
		}
	}

	return &boresolver.ResolutionContext{
		BOID:         boID,
		DrivingTable: boProps.DrivingTable, // Might be empty if not in props
		TermMappings: termMappings,
		JoinPaths:    make(map[string][]boresolver.JoinStep), // TODO: Populate Join Paths from Graph
	}, nil
}

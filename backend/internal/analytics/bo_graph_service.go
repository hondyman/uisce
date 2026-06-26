package analytics

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// GraphNode represents a node in the BO lineage graph
type GraphNode struct {
	ID    string                 `json:"id"`
	Type  string                 `json:"type"`
	Label string                 `json:"label"`
	Data  map[string]interface{} `json:"data"`
}

// GraphEdge represents an edge in the BO lineage graph
type GraphEdge struct {
	ID     string                 `json:"id"`
	Source string                 `json:"source"`
	Target string                 `json:"target"`
	Type   string                 `json:"type"`
	Label  string                 `json:"label,omitempty"`
	Data   map[string]interface{} `json:"data,omitempty"`
}

// BOGraph represents the complete graph structure
type BOGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// BOGraphService generates lineage graphs for Business Objects
type BOGraphService struct {
	db *sqlx.DB
}

// NewBOGraphService creates a new graph service
func NewBOGraphService(db *sqlx.DB) *BOGraphService {
	return &BOGraphService{db: db}
}

// GenerateGraph creates a complete lineage graph for a Business Object
func (s *BOGraphService) GenerateGraph(boID string) (*BOGraph, error) {
	graph := &BOGraph{
		Nodes: []GraphNode{},
		Edges: []GraphEdge{},
	}

	// 1. Add BO Node (centerpiece)
	boNode, err := s.buildBONode(boID)
	if err != nil {
		return nil, fmt.Errorf("failed to build BO node: %w", err)
	}
	graph.Nodes = append(graph.Nodes, boNode)

	// 2. Add Term Nodes + BO→Term edges
	terms, err := s.fetchTerms(boID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch terms: %w", err)
	}

	tableNodes := make(map[string]GraphNode)
	columnNodes := make(map[string]GraphNode)

	for _, term := range terms {
		termNode := s.buildTermNode(term)
		graph.Nodes = append(graph.Nodes, termNode)

		// BO → Term edge
		graph.Edges = append(graph.Edges, GraphEdge{
			ID:     fmt.Sprintf("bo_term_%s", term.ID),
			Source: boNode.ID,
			Target: termNode.ID,
			Type:   "contains",
		})

		// 3. Add Physical Mapping (Term → Column → Table)
		if term.PhysicalMapping != nil {
			tableID := fmt.Sprintf("TABLE:%s.%s", term.PhysicalMapping.Schema, term.PhysicalMapping.Table)
			columnID := fmt.Sprintf("COLUMN:%s.%s.%s", term.PhysicalMapping.Schema, term.PhysicalMapping.Table, term.PhysicalMapping.Column)

			// Create table node if not exists
			if _, exists := tableNodes[tableID]; !exists {
				tableNode := s.buildTableNode(term.PhysicalMapping.Schema, term.PhysicalMapping.Table)
				tableNodes[tableID] = tableNode
			}

			// Create column node if not exists
			if _, exists := columnNodes[columnID]; !exists {
				columnNode := s.buildColumnNode(term.PhysicalMapping)
				columnNodes[columnID] = columnNode

				// Column → Table edge
				graph.Edges = append(graph.Edges, GraphEdge{
					ID:     fmt.Sprintf("col_table_%s", columnID),
					Source: columnID,
					Target: tableID,
					Type:   "belongs_to",
				})
			}

			// Term → Column edge
			graph.Edges = append(graph.Edges, GraphEdge{
				ID:     fmt.Sprintf("term_col_%s", term.ID),
				Source: termNode.ID,
				Target: columnID,
				Type:   "maps_to",
			})
		}
	}

	// Add all table and column nodes
	for _, node := range tableNodes {
		graph.Nodes = append(graph.Nodes, node)
	}
	for _, node := range columnNodes {
		graph.Nodes = append(graph.Nodes, node)
	}

	// 4. Add Calculation Nodes + Dependencies
	calcs, err := s.fetchCalculations(boID)
	if err != nil {
		// Non-fatal, just log
		fmt.Printf("Warning: failed to fetch calculations: %v\n", err)
	} else {
		for _, calc := range calcs {
			calcNode := s.buildCalculationNode(calc)
			graph.Nodes = append(graph.Nodes, calcNode)

			// Parse dependencies from formula
			deps := s.extractDependencies(calc.Formula, terms)
			for _, depTermID := range deps {
				graph.Edges = append(graph.Edges, GraphEdge{
					ID:     fmt.Sprintf("calc_dep_%s_%s", calc.ID, depTermID),
					Source: calcNode.ID,
					Target: fmt.Sprintf("TERM:%s", depTermID),
					Type:   "uses",
				})
			}
		}
	}

	// 5. Add Related BOs (via FK paths)
	relatedBOs, err := s.fetchRelatedBOs(boID)
	if err != nil {
		// Non-fatal
		fmt.Printf("Warning: failed to fetch related BOs: %v\n", err)
	} else {
		for _, relBO := range relatedBOs {
			relNode := s.buildRelatedBONode(relBO)
			graph.Nodes = append(graph.Nodes, relNode)

			graph.Edges = append(graph.Edges, GraphEdge{
				ID:     fmt.Sprintf("bo_rel_%s", relBO.ID),
				Source: boNode.ID,
				Target: relNode.ID,
				Type:   "relates_to",
				Data: map[string]interface{}{
					"relationshipType": relBO.RelationshipType,
				},
			})
		}
	}

	return graph, nil
}

// Term represents a semantic term with metadata
type Term struct {
	ID              string
	NodeName        string
	TermType        string
	DataType        string
	IsKey           bool
	IsForeignKey    bool
	Aggregation     string
	PhysicalMapping *PhysicalMapping
}

// PhysicalMapping represents the physical table/column mapping
type PhysicalMapping struct {
	Schema string
	Table  string
	Column string
}

// Calculation represents a calculation definition
type Calculation struct {
	ID         string
	Name       string
	Formula    string
	ReturnType string
}

// RelatedBO represents a related Business Object
type RelatedBO struct {
	ID               string
	Name             string
	RelationshipType string
}

func (s *BOGraphService) buildBONode(boID string) (GraphNode, error) {
	var nodeName, description string
	var termCount int

	// Try business_objects table first (wizard-created BOs)
	err := s.db.QueryRow(`
		SELECT 
			COALESCE(display_name, name) as node_name,
			COALESCE(description, '') as description
		FROM business_objects
		WHERE id = $1::uuid
	`, boID).Scan(&nodeName, &description)

	if err != nil {
		// Fallback to catalog_node (legacy BOs)
		err = s.db.QueryRow(`
			SELECT 
				node_name,
				COALESCE(description, '') as description
			FROM catalog_node
			WHERE id = $1
		`, boID).Scan(&nodeName, &description)
		if err != nil {
			return GraphNode{}, err
		}
	}

	// Count terms from business_objects first
	err = s.db.QueryRow(`
		SELECT COUNT(*)
		FROM bo_fields
		WHERE bo_id = $1::uuid AND subtype_id IS NULL
	`, boID).Scan(&termCount)
	if err != nil || termCount == 0 {
		// Fallback to catalog_edge
		s.db.QueryRow(`
			SELECT COUNT(*)
			FROM catalog_edge
			WHERE source_node_id = $1 AND edge_type_name = 'HAS_ATTRIBUTE'
		`, boID).Scan(&termCount)
	}

	return GraphNode{
		ID:    fmt.Sprintf("BO:%s", boID),
		Type:  "bo",
		Label: nodeName,
		Data: map[string]interface{}{
			"boId":        boID,
			"name":        nodeName,
			"description": description,
			"termCount":   termCount,
		},
	}, nil
}

func (s *BOGraphService) fetchTerms(boID string) ([]Term, error) {
	// Try business_objects bo_fields first
	rows, err := s.db.Query(`
		SELECT 
			f.id,
			COALESCE(f.display_label, f.name) as node_name,
			CASE WHEN f.field_type = 'semantic_term' THEN 'dimension' ELSE 'dimension' END as term_type,
			'string' as data_type,
			false as is_key,
			false as is_foreign_key,
			'' as aggregation,
			NULL::jsonb as physical_mapping
		FROM bo_fields f
		WHERE f.bo_id = $1::uuid 
		  AND f.subtype_id IS NULL
	`, boID)

	if err != nil {
		// Fallback to catalog_node/catalog_edge
		rows, err = s.db.Query(`
			SELECT 
				st.id,
				st.node_name,
				COALESCE(st.properties->>'term_type', 'dimension') as term_type,
				COALESCE(st.properties->>'data_type', 'string') as data_type,
				COALESCE((st.properties->>'is_key')::boolean, false) as is_key,
				COALESCE((st.properties->>'is_foreign_key')::boolean, false) as is_foreign_key,
				COALESCE(st.properties->>'aggregation', '') as aggregation,
				st.properties->'physical_mapping' as physical_mapping
			FROM catalog_edge ce
			JOIN catalog_node st ON ce.target_node_id = st.id
			WHERE ce.source_node_id = $1 
			  AND ce.edge_type_name = 'HAS_ATTRIBUTE'
		`, boID)
		if err != nil {
			return nil, err
		}
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var terms []Term
	for rows.Next() {
		var term Term
		var mappingJSON []byte

		err := rows.Scan(
			&term.ID,
			&term.NodeName,
			&term.TermType,
			&term.DataType,
			&term.IsKey,
			&term.IsForeignKey,
			&term.Aggregation,
			&mappingJSON,
		)
		if err != nil {
			continue
		}

		// Parse physical mapping if exists
		if len(mappingJSON) > 0 && string(mappingJSON) != "null" {
			var mapping struct {
				Schema string `json:"schema"`
				Table  string `json:"table"`
				Column string `json:"column"`
			}
			if err := json.Unmarshal(mappingJSON, &mapping); err == nil && mapping.Column != "" {
				term.PhysicalMapping = &PhysicalMapping{
					Schema: mapping.Schema,
					Table:  mapping.Table,
					Column: mapping.Column,
				}
			}
		}

		terms = append(terms, term)
	}

	return terms, nil
}

func (s *BOGraphService) buildTermNode(term Term) GraphNode {
	data := map[string]interface{}{
		"termId":       term.ID,
		"termName":     term.NodeName,
		"termType":     term.TermType,
		"dataType":     term.DataType,
		"isKey":        term.IsKey,
		"isForeignKey": term.IsForeignKey,
	}

	if term.Aggregation != "" {
		data["aggregation"] = term.Aggregation
	}

	if term.PhysicalMapping != nil {
		data["physicalMapping"] = map[string]string{
			"schema": term.PhysicalMapping.Schema,
			"table":  term.PhysicalMapping.Table,
			"column": term.PhysicalMapping.Column,
		}
	}

	return GraphNode{
		ID:    fmt.Sprintf("TERM:%s", term.ID),
		Type:  "term",
		Label: term.NodeName,
		Data:  data,
	}
}

func (s *BOGraphService) buildTableNode(schema, table string) GraphNode {
	return GraphNode{
		ID:    fmt.Sprintf("TABLE:%s.%s", schema, table),
		Type:  "table",
		Label: table,
		Data: map[string]interface{}{
			"schema":    schema,
			"tableName": table,
		},
	}
}

func (s *BOGraphService) buildColumnNode(mapping *PhysicalMapping) GraphNode {
	return GraphNode{
		ID:    fmt.Sprintf("COLUMN:%s.%s.%s", mapping.Schema, mapping.Table, mapping.Column),
		Type:  "column",
		Label: mapping.Column,
		Data: map[string]interface{}{
			"schema":     mapping.Schema,
			"tableName":  mapping.Table,
			"columnName": mapping.Column,
		},
	}
}

func (s *BOGraphService) fetchCalculations(boID string) ([]Calculation, error) {
	rows, err := s.db.Query(`
		SELECT 
			id, name, formula, COALESCE(return_type, 'unknown') as return_type
		FROM calculations
		WHERE domain_id = $1
	`, boID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calcs []Calculation
	for rows.Next() {
		var calc Calculation
		if err := rows.Scan(&calc.ID, &calc.Name, &calc.Formula, &calc.ReturnType); err != nil {
			continue
		}
		calcs = append(calcs, calc)
	}

	return calcs, nil
}

func (s *BOGraphService) buildCalculationNode(calc Calculation) GraphNode {
	return GraphNode{
		ID:    fmt.Sprintf("CALC:%s", calc.ID),
		Type:  "calculation",
		Label: calc.Name,
		Data: map[string]interface{}{
			"calcId":     calc.ID,
			"name":       calc.Name,
			"formula":    calc.Formula,
			"returnType": calc.ReturnType,
		},
	}
}

func (s *BOGraphService) extractDependencies(formula string, terms []Term) []string {
	// Simple dependency extraction: find term names in formula
	var deps []string
	for _, term := range terms {
		if strings.Contains(formula, term.NodeName) {
			deps = append(deps, term.ID)
		}
	}
	return deps
}

func (s *BOGraphService) fetchRelatedBOs(boID string) ([]RelatedBO, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT
			target_bo.id,
			target_bo.node_name,
			COALESCE(fk.properties->>'relationship_type', 'related') as relationship_type
		FROM catalog_edge fk
		JOIN catalog_node target_bo ON fk.target_node_id = target_bo.id
		WHERE fk.source_node_id = $1
		  AND fk.edge_type_name IN ('FOREIGN_KEY', 'RELATES_TO')
	`, boID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relatedBOs []RelatedBO
	for rows.Next() {
		var relBO RelatedBO
		if err := rows.Scan(&relBO.ID, &relBO.Name, &relBO.RelationshipType); err != nil {
			continue
		}
		relatedBOs = append(relatedBOs, relBO)
	}

	return relatedBOs, nil
}

func (s *BOGraphService) buildRelatedBONode(relBO RelatedBO) GraphNode {
	return GraphNode{
		ID:    fmt.Sprintf("BO:%s", relBO.ID),
		Type:  "related_bo",
		Label: relBO.Name,
		Data: map[string]interface{}{
			"boId":             relBO.ID,
			"name":             relBO.Name,
			"relationshipType": relBO.RelationshipType,
		},
	}
}

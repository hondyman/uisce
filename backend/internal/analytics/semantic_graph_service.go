package analytics

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// SemanticGraphService manages the semantic graph model
type SemanticGraphService struct {
	db *sqlx.DB

	// Cached node type IDs
	calcTermTypeID     uuid.UUID
	semanticTermTypeID uuid.UUID
	boTypeID           uuid.UUID
	tableTypeID        uuid.UUID
	columnTypeID       uuid.UUID

	listeners []func(nodeID uuid.UUID)
}

// NewSemanticGraphService creates a new graph service
func NewSemanticGraphService(db *sqlx.DB) *SemanticGraphService {
	return &SemanticGraphService{db: db}
}

// RegisterChangeListener adds a listener for graph changes
func (s *SemanticGraphService) RegisterChangeListener(callback func(nodeID uuid.UUID)) {
	s.listeners = append(s.listeners, callback)
}

func (s *SemanticGraphService) notifyListeners(nodeID uuid.UUID) {
	for _, l := range s.listeners {
		go l(nodeID) // Async notification
	}
}

// Edge types specific to semantic graph (non-impact analysis)
const (
	// Calculation edges
	EdgeTypeCalcUsesTerm  EdgeType = "CALC_USES_TERM"
	EdgeTypeCalcUsesCalc  EdgeType = "CALC_USES_CALC"
	EdgeTypeCalcUsesTable EdgeType = "CALC_USES_TABLE"

	// Business Object edges
	EdgeTypeBOHasTerm     EdgeType = "BO_HAS_TERM"
	EdgeTypeBOHasCalc     EdgeType = "BO_HAS_CALC"
	EdgeTypeBORelatesToBO EdgeType = "BO_RELATES_TO_BO"
	EdgeTypeBODrivesTable EdgeType = "BO_DRIVES_TABLE"

	// Term edges
	EdgeTypeTermMapsToColumn EdgeType = "TERM_MAPS_TO_COLUMN"
	EdgeTypeHasAttribute     EdgeType = "HAS_ATTRIBUTE"
	EdgeTypeBOHasAttribute   EdgeType = "BO_HAS_ATTRIBUTE"
	EdgeTypeBOHasColumn      EdgeType = "BO_HAS_COLUMN"
)

// Node types specific to semantic graph (physical layer)
const (
	NodeTypePhysicalTable  NodeType = "table"
	NodeTypePhysicalColumn NodeType = "column"
)

// SemanticNode represents a node in the semantic graph
type SemanticNode struct {
	ID            uuid.UUID              `json:"id"`
	NodeType      string                 `json:"node_type"`
	NodeName      string                 `json:"node_name"`
	Description   string                 `json:"description,omitempty"`
	Properties    map[string]interface{} `json:"properties,omitempty"`
	Config        map[string]interface{} `json:"config,omitempty"`
	QualifiedPath string                 `json:"qualified_path"`
}

// SemanticEdge represents an edge in the semantic graph
type SemanticEdge struct {
	ID           uuid.UUID              `json:"id"`
	SourceNodeID uuid.UUID              `json:"source_node_id"`
	TargetNodeID uuid.UUID              `json:"target_node_id"`
	EdgeType     EdgeType               `json:"edge_type"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

// Initialize loads node type IDs
func (s *SemanticGraphService) Initialize() error {
	nodeTypes := map[string]*uuid.UUID{
		"calculation_term": &s.calcTermTypeID,
		"semantic_term":    &s.semanticTermTypeID,
		"business_object":  &s.boTypeID,
		"table":            &s.tableTypeID,
		"column":           &s.columnTypeID,
	}

	for typeName, typeIDPtr := range nodeTypes {
		var typeID uuid.UUID
		err := s.db.Get(&typeID, `
			SELECT id FROM catalog_node_type WHERE node_type = $1 LIMIT 1
		`, typeName)
		if err == sql.ErrNoRows {
			// Create node type if it doesn't exist
			typeID = uuid.New()
			_, err = s.db.Exec(`
				INSERT INTO catalog_node_type (id, node_type, display_name, description)
				VALUES ($1, $2, $3, $4)
			`, typeID, typeName, formatNodeTypeName(typeName), "Semantic graph node type")
			if err != nil {
				return fmt.Errorf("failed to create node type %s: %w", typeName, err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to lookup node type %s: %w", typeName, err)
		}
		*typeIDPtr = typeID
	}

	return nil
}

// CreateNode creates a new node in the semantic graph
func (s *SemanticGraphService) CreateNode(
	nodeType NodeType,
	nodeName string,
	description string,
	properties map[string]interface{},
	config map[string]interface{},
	tenantID uuid.UUID,
	datasourceID uuid.UUID,
) (uuid.UUID, error) {
	nodeTypeID := s.getNodeTypeID(nodeType)
	if nodeTypeID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("unknown node type: %s", nodeType)
	}

	qualifiedPath := fmt.Sprintf("%s/%s", nodeType, nodeName)

	propertiesJSON, _ := json.Marshal(properties)
	configJSON, _ := json.Marshal(config)

	nodeID := uuid.New()
	now := time.Now()

	_, err := s.db.Exec(`
		INSERT INTO catalog_node (
			id, node_type_id, node_name, description, qualified_path,
			properties, config, tenant_id, tenant_datasource_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
		ON CONFLICT (tenant_datasource_id, node_type_id, qualified_path)
		DO UPDATE SET
			description = EXCLUDED.description,
			properties = EXCLUDED.properties,
			config = EXCLUDED.config,
			updated_at = EXCLUDED.updated_at
		RETURNING id
	`,
		nodeID, nodeTypeID, nodeName, description, qualifiedPath,
		propertiesJSON, configJSON, tenantID, datasourceID,
		now, now,
	)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.notifyListeners(nodeID)

	return nodeID, nil
}

// CreateEdge creates an edge between two nodes
func (s *SemanticGraphService) CreateEdge(
	sourceNodeID uuid.UUID,
	targetNodeID uuid.UUID,
	edgeType EdgeType,
	tenantID uuid.UUID,
	datasourceID uuid.UUID,
	properties map[string]interface{},
) (uuid.UUID, error) {
	propertiesJSON, _ := json.Marshal(properties)

	edgeID := uuid.New()
	now := time.Now()

	_, err := s.db.Exec(`
		INSERT INTO catalog_edge (
			id, source_node_id, target_node_id, edge_type_name,
			tenant_id, tenant_datasource_id, properties,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		ON CONFLICT DO NOTHING
	`,
		edgeID, sourceNodeID, targetNodeID, string(edgeType),
		tenantID, datasourceID, propertiesJSON,
		now, now,
	)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.notifyListeners(sourceNodeID)
	s.notifyListeners(targetNodeID)

	return edgeID, nil
}

// GetNodeByID retrieves a node by ID
func (s *SemanticGraphService) GetNodeByID(nodeID uuid.UUID) (*SemanticNode, error) {
	var node struct {
		ID            uuid.UUID      `db:"id"`
		NodeType      string         `db:"node_type"` // Raw from join
		NodeName      string         `db:"node_name"`
		Description   sql.NullString `db:"description"`
		Properties    string         `db:"properties"`
		Config        string         `db:"config"`
		QualifiedPath string         `db:"qualified_path"`
	}

	err := s.db.Get(&node, `
		SELECT n.id, nt.node_type, n.node_name, n.description, n.properties, n.config, n.qualified_path
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE n.id = $1
	`, nodeID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	result := &SemanticNode{
		ID:            node.ID,
		NodeType:      node.NodeType,
		NodeName:      node.NodeName,
		Description:   node.Description.String,
		QualifiedPath: node.QualifiedPath,
	}

	if len(node.Properties) > 0 {
		json.Unmarshal([]byte(node.Properties), &result.Properties)
	}
	if len(node.Config) > 0 {
		json.Unmarshal([]byte(node.Config), &result.Config)
	}

	return result, nil
}

// GetNodeByName retrieves a node by type and name
func (s *SemanticGraphService) GetNodeByName(
	nodeType NodeType,
	nodeName string,
	datasourceID uuid.UUID,
) (*SemanticNode, error) {
	nodeTypeID := s.getNodeTypeID(nodeType)
	if nodeTypeID == uuid.Nil {
		return nil, fmt.Errorf("unknown node type: %s", nodeType)
	}

	var node struct {
		ID            uuid.UUID       `db:"id"`
		NodeName      string          `db:"node_name"`
		Description   sql.NullString  `db:"description"`
		Properties    json.RawMessage `db:"properties"`
		Config        json.RawMessage `db:"config"`
		QualifiedPath string          `db:"qualified_path"`
	}

	err := s.db.Get(&node, `
		SELECT id, node_name, description, properties, config, qualified_path
		FROM catalog_node
		WHERE node_type_id = $1 AND node_name = $2 AND tenant_datasource_id = $3
	`, nodeTypeID, nodeName, datasourceID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	result := &SemanticNode{
		ID:            node.ID,
		NodeType:      string(nodeType),
		NodeName:      node.NodeName,
		Description:   node.Description.String,
		QualifiedPath: node.QualifiedPath,
	}

	json.Unmarshal(node.Properties, &result.Properties)
	json.Unmarshal(node.Config, &result.Config)

	return result, nil
}

// GetOutgoingEdges retrieves all outgoing edges from a node
func (s *SemanticGraphService) GetOutgoingEdges(nodeID uuid.UUID) ([]SemanticEdge, error) {
	var edges []struct {
		ID           uuid.UUID       `db:"id"`
		SourceNodeID uuid.UUID       `db:"source_node_id"`
		TargetNodeID uuid.UUID       `db:"target_node_id"`
		EdgeType     string          `db:"edge_type_name"`
		Properties   json.RawMessage `db:"properties"`
	}

	err := s.db.Select(&edges, `
		SELECT id, source_node_id, target_node_id, edge_type_name, properties
		FROM catalog_edge
		WHERE source_node_id = $1
	`, nodeID)

	if err != nil {
		return nil, err
	}

	result := make([]SemanticEdge, len(edges))
	for i, e := range edges {
		result[i] = SemanticEdge{
			ID:           e.ID,
			SourceNodeID: e.SourceNodeID,
			TargetNodeID: e.TargetNodeID,
			EdgeType:     EdgeType(e.EdgeType),
		}
		json.Unmarshal(e.Properties, &result[i].Properties)
	}

	return result, nil
}

// GetIncomingEdges retrieves all incoming edges to a node
func (s *SemanticGraphService) GetIncomingEdges(nodeID uuid.UUID) ([]SemanticEdge, error) {
	var edges []struct {
		ID           uuid.UUID       `db:"id"`
		SourceNodeID uuid.UUID       `db:"source_node_id"`
		TargetNodeID uuid.UUID       `db:"target_node_id"`
		EdgeType     string          `db:"edge_type_name"`
		Properties   json.RawMessage `db:"properties"`
	}

	err := s.db.Select(&edges, `
		SELECT id, source_node_id, target_node_id, edge_type_name, properties
		FROM catalog_edge
		WHERE target_node_id = $1
	`, nodeID)

	if err != nil {
		return nil, err
	}

	result := make([]SemanticEdge, len(edges))
	for i, e := range edges {
		result[i] = SemanticEdge{
			ID:           e.ID,
			SourceNodeID: e.SourceNodeID,
			TargetNodeID: e.TargetNodeID,
			EdgeType:     EdgeType(e.EdgeType),
		}
		json.Unmarshal(e.Properties, &result[i].Properties)
	}

	return result, nil
}

// GetEdgesByType retrieves edges of a specific type from a node
func (s *SemanticGraphService) GetEdgesByType(nodeID uuid.UUID, edgeType EdgeType) ([]SemanticEdge, error) {
	var edges []struct {
		ID           uuid.UUID       `db:"id"`
		SourceNodeID uuid.UUID       `db:"source_node_id"`
		TargetNodeID uuid.UUID       `db:"target_node_id"`
		EdgeType     string          `db:"edge_type"`
		Properties   json.RawMessage `db:"properties"`
	}

	err := s.db.Select(&edges, `
		SELECT id, source_node_id, target_node_id, edge_type, properties
		FROM catalog_edge
		WHERE source_node_id = $1 AND edge_type = $2
	`, nodeID, string(edgeType))

	if err != nil {
		return nil, err
	}

	result := make([]SemanticEdge, len(edges))
	for i, e := range edges {
		result[i] = SemanticEdge{
			ID:           e.ID,
			SourceNodeID: e.SourceNodeID,
			TargetNodeID: e.TargetNodeID,
			EdgeType:     EdgeType(e.EdgeType),
		}
		json.Unmarshal(e.Properties, &result[i].Properties)
	}

	return result, nil
}

// DeleteEdge deletes an edge
func (s *SemanticGraphService) DeleteEdge(edgeID uuid.UUID) error {
	// First get the edge to know source/target for notification
	var edge struct {
		SourceID uuid.UUID `db:"source_node_id"`
		TargetID uuid.UUID `db:"target_node_id"`
	}
	err := s.db.Get(&edge, "SELECT source_node_id, target_node_id FROM catalog_edge WHERE id = $1", edgeID)
	if err == nil {
		s.notifyListeners(edge.SourceID)
		s.notifyListeners(edge.TargetID)
	}

	_, err = s.db.Exec(`DELETE FROM catalog_edge WHERE id = $1`, edgeID)
	return err
}

// DeleteEdgesBySource deletes all edges from a source node
func (s *SemanticGraphService) DeleteEdgesBySource(sourceNodeID uuid.UUID, edgeType EdgeType) error {
	_, err := s.db.Exec(`
		DELETE FROM catalog_edge 
		WHERE source_node_id = $1 AND edge_type = $2
	`, sourceNodeID, string(edgeType))
	return err
}

// GetNodesByType retrieves all nodes of a specific type
func (s *SemanticGraphService) GetNodesByType(nodeType NodeType, tenantID uuid.UUID) ([]SemanticNode, error) {
	nodeTypeID := s.getNodeTypeID(nodeType)
	if nodeTypeID == uuid.Nil {
		return nil, fmt.Errorf("unknown node type: %s", nodeType)
	}

	var nodes []struct {
		ID            uuid.UUID       `db:"id"`
		NodeName      string          `db:"node_name"`
		Description   sql.NullString  `db:"description"`
		Properties    json.RawMessage `db:"properties"`
		Config        json.RawMessage `db:"config"`
		QualifiedPath string          `db:"qualified_path"`
	}

	err := s.db.Select(&nodes, `
		SELECT id, node_name, description, properties, config, qualified_path
		FROM catalog_node
		WHERE node_type_id = $1 AND tenant_id = $2
	`, nodeTypeID, tenantID)

	if err != nil {
		return nil, err
	}

	result := make([]SemanticNode, len(nodes))
	for i, n := range nodes {
		result[i] = SemanticNode{
			ID:            n.ID,
			NodeType:      string(nodeType),
			NodeName:      n.NodeName,
			Description:   n.Description.String,
			QualifiedPath: n.QualifiedPath,
		}
		json.Unmarshal(n.Properties, &result[i].Properties)
		json.Unmarshal(n.Config, &result[i].Config)
	}

	return result, nil
}

// GetEdgeProperties retrieves properties for a specific edge
func (s *SemanticGraphService) GetEdgeProperties(sourceID, targetID uuid.UUID, edgeType EdgeType) (map[string]interface{}, error) {
	var propsJSON json.RawMessage
	err := s.db.Get(&propsJSON, `
		SELECT properties FROM catalog_edge
		WHERE source_node_id = $1 AND target_node_id = $2 AND edge_type_name = $3
		LIMIT 1
	`, sourceID, targetID, string(edgeType))

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var props map[string]interface{}
	json.Unmarshal(propsJSON, &props)
	return props, nil
}

// Helper functions
func (s *SemanticGraphService) getNodeTypeID(nodeType NodeType) uuid.UUID {
	switch nodeType {
	case NodeTypeCalculationTerm:
		return s.calcTermTypeID
	case NodeTypeSemanticTerm:
		return s.semanticTermTypeID
	case NodeTypeBusinessObject:
		return s.boTypeID
	case NodeTypePhysicalTable:
		return s.tableTypeID
	case NodeTypePhysicalColumn:
		return s.columnTypeID
	default:
		return uuid.Nil
	}
}

func formatNodeTypeName(nodeType string) string {
	switch nodeType {
	case "calculation_term":
		return "Calculation Term"
	case "semantic_term":
		return "Semantic Term"
	case "business_object":
		return "Business Object"
	case "table":
		return "Physical Table"
	case "column":
		return "Physical Column"
	default:
		return nodeType
	}
}

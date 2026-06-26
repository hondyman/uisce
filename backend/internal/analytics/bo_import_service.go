package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

type BOImportService struct {
	db          *sqlx.DB
	diffService *BODiffService
	exportSvc   *BOExportService
}

func NewBOImportService(db *sqlx.DB, diffSvc *BODiffService, exportSvc *BOExportService) *BOImportService {
	return &BOImportService{
		db:          db,
		diffService: diffSvc,
		exportSvc:   exportSvc,
	}
}

func (s *BOImportService) ImportBO(ctx context.Context, secCtx *security.Context, req models.ImportRequest, userID string) (*models.ImportResult, error) {
	if secCtx == nil {
		return nil, fmt.Errorf("security context is required")
	}
	if strings.TrimSpace(secCtx.TenantID) == "" || strings.TrimSpace(secCtx.DatasourceID) == "" {
		return nil, fmt.Errorf("security context missing tenant or datasource")
	}
	if strings.TrimSpace(secCtx.Region) == "" {
		return nil, fmt.Errorf("security context missing region")
	}

	result := &models.ImportResult{
		Mode: req.Mode,
		Summary: models.ImportSummary{
			NodesToCreate:    0,
			NodesToUpdate:    0,
			NodesConflicting: 0,
			EdgesToCreate:    0,
			EdgesToUpdate:    0,
		},
	}

	tenantID := secCtx.TenantID
	datasourceID := secCtx.DatasourceID

	// Transaction for Apply mode
	var (
		tx  *sqlx.Tx
		err error
	)
	if req.Mode == models.ImportModeApply {
		tx, err = s.db.BeginTxx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
	}

	// 1. Process Business Objects
	for _, boExport := range req.Bundle.BusinessObjects {
		err := s.processNode(ctx, tx, req, boExport, result, "business_object", tenantID, datasourceID, secCtx.OperatingScope)
		if err != nil {
			return nil, err
		}
	}

	// 2. Process Semantic Terms
	for _, termExport := range req.Bundle.SemanticTerms {
		err := s.processNode(ctx, tx, req, termExport, result, "semantic_term", tenantID, datasourceID, secCtx.OperatingScope)
		if err != nil {
			return nil, err
		}
	}

	// 3. Process Calculation Terms
	for _, calcExport := range req.Bundle.CalculationTerms {
		err := s.processNode(ctx, tx, req, calcExport, result, "calculation_term", tenantID, datasourceID, secCtx.OperatingScope)
		if err != nil {
			return nil, err
		}
	}

	// 4. Process Edges (second pass after nodes are in place)
	if err := s.processEdges(ctx, tx, req, result, tenantID, datasourceID); err != nil {
		return nil, err
	}

	// 5. Commit if Apply
	if req.Mode == models.ImportModeApply {
		if len(result.Errors) > 0 {
			return result, errors.New("import failed validation")
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (s *BOImportService) processNode(ctx context.Context, tx *sqlx.Tx, req models.ImportRequest, item models.NodeExport, result *models.ImportResult, nodeType string, tenantID string, datasourceID string, operatingScope string) error {
	validatedType := normalizeNodeTypeName(nodeType)
	if err := s.validateNodeExport(ctx, item.Node, validatedType, tenantID); err != nil {
		return s.recordValidationError(result, err.Error())
	}

	normalizedNode, err := s.normalizeNodeForImport(item.Node, validatedType, operatingScope)
	if err != nil {
		return s.recordValidationError(result, err.Error())
	}

	// 1. Check if Node Exists (by Name + Type)
	// Why Name? Import portability.
	existingNode, err := s.fetchNodeByName(ctx, normalizedNode.NodeName, validatedType, tenantID, datasourceID)
	if err != nil {
		return err
	}

	if existingNode == nil {
		// CREATE case
		result.Summary.NodesToCreate++
		result.NodeDiffs = append(result.NodeDiffs, models.NodeDiff{
			NodeType: validatedType,
			NodeName: normalizedNode.NodeName,
			Status:   models.DiffMissing, // "Missing" in target -> needs create
			Incoming: &models.NodeSnapshot{Properties: normalizedNode.Properties, Config: normalizedNode.Config},
		})

		if req.Mode == models.ImportModeApply {
			// Insert Node
			newID := uuid.New().String()
			err := s.createNode(ctx, tx, newID, validatedType, normalizedNode, tenantID, datasourceID)
			if err != nil {
				return err
			}
			// Insert Edges
			for range item.Edges {
				// We need to resolve Target ID by Name.
				// This implies order dependency or deferred edge creation.
				// For simple implementation, try to resolve target.
				// If target is also being created in this bundle, we might not find it yet if dependent on later items.
				// But we are processing types in order: BO -> Terms -> Calcs.
				// Actually Terms should be first if BO depends on them.
				// Reordering: SemanticTerms -> CalculationTerms -> BOs usually makes sense for dependencies.
				// But user UI order is BO first.
				// We'll handle edges in a second pass?
				// Or assume the DB constraint isn't enforced strictly if we defer FK checks?
				// Actually catalog_edge uses UUIDs.
				// Let's implement Edge creation in a second pass if possible.
				// For now, assume simple case or just skip edge if target missing.

				// Wait, the item.Edges describes OUTGOING edges.
				// If BO has "HAS_TERM", the Term must exist.
				// So Terms should be processed BEFORE BOs.
			}
		}
	} else {
		// UPDATE/CONFLICT case

		// Fetch Iceberg Snapshot
		// Mock implementation: For now assume nil or fetch from "Iceberg" service if we had one.
		// In production this would call e.g. s.icebergClient.GetSnapshot(bundle.IcebergVersion, nodeID)
		var icebergNode *models.CatalogNodeExport = nil
		// TODO: Implement actual Iceberg fetch
		// icebergNode = s.fetchIcebergSnapshot(req.Bundle.IcebergVersion, existingNode.QualifiedPath)

		diff, err := s.diffService.DiffNodes(existingNode, icebergNode, &normalizedNode)
		if err != nil {
			return err
		}
		result.NodeDiffs = append(result.NodeDiffs, *diff)

		if diff.Status == models.DiffExistsDifferent || diff.Status == models.DiffConflict {
			if diff.Status == models.DiffExistsDifferent {
				result.Summary.NodesToUpdate++
			} else {
				result.Summary.NodesConflicting++
			}

			// Governance Check: Golden Rule
			if isGolden(existingNode.Properties) && !isGolden(normalizedNode.Properties) {
				// If strictly golden rule applies:
				result.Summary.NodesConflicting++
				diff.Status = models.DiffConflict
				diff.Errors = append(diff.Errors, "Governance: Cannot overwrite golden node with non-golden version")
				result.Errors = append(result.Errors, fmt.Sprintf("Cannot overwrite golden node '%s'", normalizedNode.NodeName))
			}

			// Governance Check: Conflict on Golden Field
			// If existing is golden, and we have ANY conflict, block it?
			if isGolden(existingNode.Properties) && diff.Status == models.DiffConflict {
				result.Errors = append(result.Errors, fmt.Sprintf("Governance: Conflict detected on golden node '%s'. Manual resolution required in source.", normalizedNode.NodeName))
				return nil
			}

			if req.Mode == models.ImportModeApply {
				if diff.Status != models.DiffConflict && (req.ConflictStrategy == models.ConflictReplace || req.ConflictStrategy == models.ConflictMerge) {
					// Update Node
					if err := s.updateNode(ctx, tx, existingNode.QualifiedPath, validatedType, normalizedNode, tenantID, datasourceID); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// Helpers

func (s *BOImportService) fetchNodeByName(ctx context.Context, name, typeName string, tenantID string, datasourceID string) (*models.CatalogNodeExport, error) {
	var node models.CatalogNodeExport
	query := `
		SELECT
			nt.catalog_type_name as node_type_id,
			n.node_name,
			COALESCE(n.description, '') as description,
			COALESCE(n.qualified_path, '') as qualified_path,
			n.properties,
			n.config
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE n.node_name = $1
		  AND nt.catalog_type_name = $2
		  AND n.tenant_id = $3
		  AND n.tenant_datasource_id = $4
		LIMIT 1
	`

	err := s.db.GetContext(ctx, &node, query, name, typeName, tenantID, datasourceID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &node, nil
}

func (s *BOImportService) createNode(ctx context.Context, tx *sqlx.Tx, id, typeName string, node models.CatalogNodeExport, tenantID string, datasourceID string) error {
	nodeTypeID, err := s.getNodeTypeID(ctx, typeName, tenantID)
	if err != nil {
		return err
	}

	qualifiedPath := node.QualifiedPath
	if strings.TrimSpace(qualifiedPath) == "" {
		qualifiedPath = fmt.Sprintf("%s/%s", typeName, node.NodeName)
	}

	now := time.Now()
	query := `
		INSERT INTO catalog_node (
			id, node_type_id, node_name, description, qualified_path,
			properties, config, tenant_id, tenant_datasource_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11
		)
	`

	_, err = tx.ExecContext(ctx, query,
		id,
		nodeTypeID,
		node.NodeName,
		node.Description,
		qualifiedPath,
		node.Properties,
		node.Config,
		tenantID,
		datasourceID,
		now,
		now,
	)
	if err != nil {
		return err
	}
	return nil
}

func (s *BOImportService) updateNode(ctx context.Context, tx *sqlx.Tx, qualifiedPath string, typeName string, node models.CatalogNodeExport, tenantID string, datasourceID string) error {
	nodeTypeID, err := s.getNodeTypeID(ctx, typeName, tenantID)
	if err != nil {
		return err
	}

	targetQualifiedPath := qualifiedPath
	if strings.TrimSpace(targetQualifiedPath) == "" {
		targetQualifiedPath = node.QualifiedPath
	}
	if strings.TrimSpace(targetQualifiedPath) == "" {
		targetQualifiedPath = fmt.Sprintf("%s/%s", typeName, node.NodeName)
	}

	now := time.Now()
	query := `
		UPDATE catalog_node
		SET node_name = $1,
			description = $2,
			properties = $3,
			config = $4,
			updated_at = $5
		WHERE tenant_id = $6
		  AND tenant_datasource_id = $7
		  AND node_type_id = $8
		  AND qualified_path = $9
	`

	res, err := tx.ExecContext(ctx, query,
		node.NodeName,
		node.Description,
		node.Properties,
		node.Config,
		now,
		tenantID,
		datasourceID,
		nodeTypeID,
		targetQualifiedPath,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *BOImportService) getNodeTypeID(ctx context.Context, typeName string, tenantID string) (string, error) {
	var nodeTypeID string
	err := s.db.GetContext(ctx, &nodeTypeID, `
		SELECT id FROM catalog_node_type
		WHERE catalog_type_name = $1
		  AND tenant_id = $2
		LIMIT 1
	`, typeName, tenantID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("catalog_node_type not found: %s", typeName)
	}
	if err != nil {
		return "", err
	}
	return nodeTypeID, nil
}

func (s *BOImportService) processEdges(ctx context.Context, tx *sqlx.Tx, req models.ImportRequest, result *models.ImportResult, tenantID string, datasourceID string) error {
	itemGroups := []struct {
		nodeType string
		items    []models.NodeExport
	}{
		{nodeType: "semantic_term", items: req.Bundle.SemanticTerms},
		{nodeType: "calculation_term", items: req.Bundle.CalculationTerms},
		{nodeType: "business_object", items: req.Bundle.BusinessObjects},
	}

	for _, group := range itemGroups {
		for _, item := range group.items {
			if len(item.Edges) == 0 {
				continue
			}

			sourceName := strings.TrimSpace(item.Node.NodeName)
			sourceType := normalizeNodeTypeName(group.nodeType)
			if sourceName == "" {
				return s.recordValidationError(result, "edge source name missing")
			}

			sourceID, err := s.getNodeIDByName(ctx, sourceName, sourceType, tenantID, datasourceID)
			if err == sql.ErrNoRows {
				return s.recordValidationError(result, fmt.Sprintf("edge source not found: %s (%s)", sourceName, sourceType))
			}
			if err != nil {
				return err
			}

			for _, edge := range item.Edges {
				targetName := strings.TrimSpace(edge.TargetName)
				targetType := normalizeNodeTypeName(edge.TargetType)
				if targetName == "" || targetType == "" {
					return s.recordValidationError(result, fmt.Sprintf("edge target missing for source %s (%s)", sourceName, sourceType))
				}

				if err := validateRawJSON(edge.Properties, "edge.properties"); err != nil {
					return s.recordValidationError(result, err.Error())
				}

				targetID, err := s.getNodeIDByName(ctx, targetName, targetType, tenantID, datasourceID)
				if err == sql.ErrNoRows {
					return s.recordValidationError(result, fmt.Sprintf("edge target not found: %s (%s)", targetName, targetType))
				}
				if err != nil {
					return err
				}

				edgeTypeName := strings.TrimSpace(edge.EdgeType)
				if edgeTypeName == "" {
					return s.recordValidationError(result, fmt.Sprintf("edge type missing for %s -> %s", sourceName, targetName))
				}

				edgeTypeID, err := s.getEdgeTypeID(ctx, edgeTypeName, tenantID)
				if err == sql.ErrNoRows {
					return s.recordValidationError(result, fmt.Sprintf("edge type not found: %s", edgeTypeName))
				}
				if err != nil {
					return err
				}

				exists, err := s.edgeExists(ctx, sourceID, targetID, edgeTypeID, tenantID, datasourceID)
				if err != nil {
					return err
				}

				edgeDiff := models.EdgeDiff{
					EdgeType: edgeTypeName,
					Source: models.EdgeEndRef{
						Type: sourceType,
						Name: sourceName,
					},
					Target: models.EdgeEndRef{
						Type: targetType,
						Name: targetName,
					},
				}

				if exists {
					existingProps, err := s.getEdgeProperties(ctx, sourceID, targetID, edgeTypeID, tenantID, datasourceID)
					if err != nil {
						return err
					}
					equal, err := jsonEqual(existingProps, edge.Properties)
					if err != nil {
						return s.recordValidationError(result, err.Error())
					}
					if equal {
						edgeDiff.Status = models.DiffExistsSame
						result.EdgeDiffs = append(result.EdgeDiffs, edgeDiff)
						continue
					}

					edgeDiff.Status = models.DiffExistsDifferent
					result.EdgeDiffs = append(result.EdgeDiffs, edgeDiff)
					result.Summary.EdgesToUpdate++

					if req.ConflictStrategy == models.ConflictCreate {
						return s.recordValidationError(result, fmt.Sprintf("edge already exists: %s -> %s", sourceName, targetName))
					}

					if req.Mode == models.ImportModeApply {
						merge := req.ConflictStrategy == models.ConflictMerge
						if err := s.updateEdge(ctx, tx, sourceID, targetID, edgeTypeID, edgeTypeName, edge.Properties, tenantID, datasourceID, merge); err != nil {
							return err
						}
					}
					continue
				}

				result.Summary.EdgesToCreate++
				edgeDiff.Status = models.DiffMissing
				result.EdgeDiffs = append(result.EdgeDiffs, edgeDiff)

				if req.Mode == models.ImportModeApply {
					if err := s.createEdge(ctx, tx, sourceID, targetID, edgeTypeID, edgeTypeName, edge.Properties, tenantID, datasourceID); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (s *BOImportService) getNodeIDByName(ctx context.Context, name string, typeName string, tenantID string, datasourceID string) (string, error) {
	var nodeID string
	query := `
		SELECT n.id
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE n.node_name = $1
		  AND nt.catalog_type_name = $2
		  AND n.tenant_id = $3
		  AND n.tenant_datasource_id = $4
		LIMIT 1
	`
	if err := s.db.GetContext(ctx, &nodeID, query, name, typeName, tenantID, datasourceID); err != nil {
		return "", err
	}
	return nodeID, nil
}

func (s *BOImportService) getEdgeTypeID(ctx context.Context, edgeTypeName string, tenantID string) (string, error) {
	var edgeTypeID string
	query := `
		SELECT id
		FROM catalog_edge_type
		WHERE edge_type_name = $1
		  AND tenant_id = $2
		LIMIT 1
	`
	if err := s.db.GetContext(ctx, &edgeTypeID, query, edgeTypeName, tenantID); err != nil {
		return "", err
	}
	return edgeTypeID, nil
}

func (s *BOImportService) edgeExists(ctx context.Context, sourceID string, targetID string, edgeTypeID string, tenantID string, datasourceID string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM catalog_edge
			WHERE tenant_id = $1
			  AND tenant_datasource_id = $2
			  AND source_node_id = $3
			  AND target_node_id = $4
			  AND edge_type_id = $5
		)
	`
	if err := s.db.GetContext(ctx, &exists, query, tenantID, datasourceID, sourceID, targetID, edgeTypeID); err != nil {
		return false, err
	}
	return exists, nil
}

func (s *BOImportService) getEdgeProperties(ctx context.Context, sourceID string, targetID string, edgeTypeID string, tenantID string, datasourceID string) (json.RawMessage, error) {
	var props json.RawMessage
	query := `
		SELECT properties
		FROM catalog_edge
		WHERE tenant_id = $1
		  AND tenant_datasource_id = $2
		  AND source_node_id = $3
		  AND target_node_id = $4
		  AND edge_type_id = $5
		LIMIT 1
	`
	if err := s.db.GetContext(ctx, &props, query, tenantID, datasourceID, sourceID, targetID, edgeTypeID); err != nil {
		return nil, err
	}
	return props, nil
}

func (s *BOImportService) createEdge(ctx context.Context, tx *sqlx.Tx, sourceID string, targetID string, edgeTypeID string, edgeTypeName string, properties json.RawMessage, tenantID string, datasourceID string) error {
	query := `
		INSERT INTO catalog_edge (
			id, tenant_datasource_id, source_node_id, target_node_id,
			relationship_type, edge_type_id, tenant_id, created_at, updated_at, properties
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id) DO NOTHING
	`
	_, err := tx.ExecContext(ctx, query,
		uuid.New().String(),
		datasourceID,
		sourceID,
		targetID,
		edgeTypeName,
		edgeTypeID,
		tenantID,
		time.Now(),
		time.Now(),
		properties,
	)
	return err
}

func (s *BOImportService) updateEdge(ctx context.Context, tx *sqlx.Tx, sourceID string, targetID string, edgeTypeID string, edgeTypeName string, properties json.RawMessage, tenantID string, datasourceID string, merge bool) error {
	query := `
		UPDATE catalog_edge
		SET relationship_type = $1,
			properties = CASE WHEN $2 THEN COALESCE(properties, '{}'::jsonb) || $3 ELSE $3 END,
			updated_at = $4
		WHERE tenant_id = $5
		  AND tenant_datasource_id = $6
		  AND source_node_id = $7
		  AND target_node_id = $8
		  AND edge_type_id = $9
	`
	_, err := tx.ExecContext(ctx, query,
		edgeTypeName,
		merge,
		properties,
		time.Now(),
		tenantID,
		datasourceID,
		sourceID,
		targetID,
		edgeTypeID,
	)
	return err
}

func (s *BOImportService) validateNodeExport(ctx context.Context, node models.CatalogNodeExport, nodeType string, tenantID string) error {
	if strings.TrimSpace(node.NodeName) == "" {
		return fmt.Errorf("node name is required for type %s", nodeType)
	}
	if nodeType == "" {
		return fmt.Errorf("node type is required")
	}
	if err := validateRawJSON(node.Properties, "node.properties"); err != nil {
		return err
	}
	if err := validateRawJSON(node.Config, "node.config"); err != nil {
		return err
	}
	if _, err := s.getNodeTypeID(ctx, nodeType, tenantID); err != nil {
		return err
	}
	return nil
}

func (s *BOImportService) normalizeNodeForImport(node models.CatalogNodeExport, nodeType string, operatingScope string) (models.CatalogNodeExport, error) {
	if nodeType != "business_object" {
		return node, nil
	}

	props := make(map[string]interface{})
	if len(node.Properties) > 0 && string(node.Properties) != "null" {
		if err := json.Unmarshal(node.Properties, &props); err != nil {
			return node, fmt.Errorf("invalid business_object properties: %w", err)
		}
	}

	if operatingScope == "" {
		return node, fmt.Errorf("security context missing operating scope")
	}
	if rawScope, ok := props["operating_scope"]; ok {
		if scopeValue, ok := rawScope.(string); ok {
			if strings.TrimSpace(scopeValue) != operatingScope {
				return node, fmt.Errorf("business_object operating_scope does not match security context")
			}
		} else {
			return node, fmt.Errorf("business_object operating_scope must be a string")
		}
	}
	props["operating_scope"] = operatingScope

	updatedProps, err := json.Marshal(props)
	if err != nil {
		return node, fmt.Errorf("failed to marshal business_object properties: %w", err)
	}
	node.Properties = updatedProps

	return node, nil
}

func validateRawJSON(raw json.RawMessage, fieldName string) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	if !json.Valid(raw) {
		return fmt.Errorf("%s is invalid JSON", fieldName)
	}
	return nil
}

func jsonEqual(left json.RawMessage, right json.RawMessage) (bool, error) {
	if (len(left) == 0 || string(left) == "null") && (len(right) == 0 || string(right) == "null") {
		return true, nil
	}
	var leftVal interface{}
	var rightVal interface{}
	if len(left) > 0 && string(left) != "null" {
		if err := json.Unmarshal(left, &leftVal); err != nil {
			return false, fmt.Errorf("edge.properties invalid JSON")
		}
	}
	if len(right) > 0 && string(right) != "null" {
		if err := json.Unmarshal(right, &rightVal); err != nil {
			return false, fmt.Errorf("edge.properties invalid JSON")
		}
	}
	return reflect.DeepEqual(leftVal, rightVal), nil
}

func normalizeNodeTypeName(name string) string {
	value := strings.TrimSpace(strings.ToLower(name))
	switch value {
	case "bo", "businessobject", "business_object":
		return "business_object"
	case "term", "semanticterm", "semantic_term":
		return "semantic_term"
	case "calc", "calculationterm", "calculation_term":
		return "calculation_term"
	default:
		return value
	}
}

func (s *BOImportService) recordValidationError(result *models.ImportResult, message string) error {
	result.Errors = append(result.Errors, message)
	return errors.New(message)
}

func isGolden(props json.RawMessage) bool {
	// Parse properties["governance_status"] == "golden"
	return false
}

package graphql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ============================================================================
// Semantic Layer Types (for GraphQL)
// ============================================================================

type SemanticAssetGQL struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenantId"`
	DatasourceID     string    `json:"datasourceId"`
	BusinessEntityID string    `json:"businessEntityId"`
	CoreModelID      *string   `json:"coreModelId"`
	CoreViewID       *string   `json:"coreViewId"`
	CustomModelID    *string   `json:"customModelId"`
	CustomViewID     *string   `json:"customViewId"`
	SourceTables     []string  `json:"sourceTables"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type RelationshipSuggestionGQL struct {
	ID               string              `json:"id"`
	TenantID         string              `json:"tenantId"`
	DatasourceID     string              `json:"datasourceId"`
	SourceEntityID   string              `json:"sourceEntityId"`
	TargetEntityID   string              `json:"targetEntityId"`
	Confidence       float64             `json:"confidence"`
	Rationale        *string             `json:"rationale"`
	ScoringBreakdown ScoringBreakdownGQL `json:"scoringBreakdown"`
	Accepted         bool                `json:"accepted"`
	AcceptedAt       *time.Time          `json:"acceptedAt"`
	CreatedAt        time.Time           `json:"createdAt"`
	UpdatedAt        time.Time           `json:"updatedAt"`
}

type ScoringBreakdownGQL struct {
	ForeignKeyPresence float64 `json:"foreignKeyPresence"`
	JoinFrequency      float64 `json:"joinFrequency"`
	NameSimilarity     float64 `json:"nameSimilarity"`
	TextSimilarity     float64 `json:"textSimilarity"`
	EdgeTypePrior      float64 `json:"edgeTypePrior"`
}

type SemanticModelGQL struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // "core" or "custom"
	Description *string   `json:"description"`
	SourceKeys  []string  `json:"sourceKeys"`
	CreatedAt   time.Time `json:"createdAt"`
}

type SemanticViewGQL struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"` // "core" or "custom"
	Description     *string   `json:"description"`
	SelectedColumns []string  `json:"selectedColumns"`
	CreatedAt       time.Time `json:"createdAt"`
}

type ObjectGraphNodeGQL struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	LinksTo   []string `json:"linksTo"`
	LinksFrom []string `json:"linksFrom"`
}

type ObjectGraphPathGQL struct {
	Nodes []string `json:"nodes"`
	Path  string   `json:"path"`
}

type RelationshipSuggestionListGQL struct {
	Suggestions []*RelationshipSuggestionGQL `json:"suggestions"`
	Count       int                          `json:"count"`
}

// ============================================================================
// Query Resolvers
// ============================================================================

func (r *Resolver) SemanticAssets(ctx context.Context, entityID string) (*SemanticAssetGQL, error) {
	tenantID := ctx.Value("tenant_id").(string)
	datasourceID := ctx.Value("datasource_id").(string)

	query := `
		SELECT id, tenant_id, datasource_id, business_entity_id,
		       core_model_id, core_view_id, custom_model_id, custom_view_id,
		       source_tables, created_at, updated_at
		FROM semantic_assets
		WHERE tenant_id = $1 AND datasource_id = $2 AND business_entity_id = $3
	`

	var asset SemanticAssetGQL
	var sourceTables pq.StringArray

	err := r.DB.QueryRowx(query, tenantID, datasourceID, entityID).Scan(
		&asset.ID, &asset.TenantID, &asset.DatasourceID, &asset.BusinessEntityID,
		&asset.CoreModelID, &asset.CoreViewID, &asset.CustomModelID, &asset.CustomViewID,
		&sourceTables, &asset.CreatedAt, &asset.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Create new record if doesn't exist
		asset.ID = uuid.New().String()
		asset.TenantID = tenantID
		asset.DatasourceID = datasourceID
		asset.BusinessEntityID = entityID
		asset.CreatedAt = time.Now()
		asset.UpdatedAt = time.Now()

		insertQuery := `
			INSERT INTO semantic_assets (id, tenant_id, datasource_id, business_entity_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (tenant_id, datasource_id, business_entity_id) DO NOTHING
		`
		r.DB.ExecContext(ctx, insertQuery, asset.ID, tenantID, datasourceID, entityID, asset.CreatedAt, asset.UpdatedAt)
		asset.SourceTables = []string{}
		return &asset, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to query semantic assets: %w", err)
	}

	asset.SourceTables = []string(sourceTables)
	return &asset, nil
}

func (r *Resolver) RelationshipSuggestions(ctx context.Context, entityID string, limit *int, minConfidence *float64) (*RelationshipSuggestionListGQL, error) {
	tenantID := ctx.Value("tenant_id").(string)
	datasourceID := ctx.Value("datasource_id").(string)

	limitVal := 20
	if limit != nil && *limit > 0 {
		limitVal = *limit
	}

	minConfidenceVal := 0.5
	if minConfidence != nil {
		minConfidenceVal = *minConfidence
	}

	query := `
		SELECT id, tenant_id, datasource_id, source_entity_id, target_entity_id,
		       confidence, rationale, scoring_breakdown, accepted, accepted_at,
		       created_at, updated_at
		FROM relationship_suggestions
		WHERE tenant_id = $1 AND datasource_id = $2 AND source_entity_id = $3
		      AND confidence >= $4
		ORDER BY confidence DESC
		LIMIT $5
	`

	rows, err := r.DB.QueryContext(ctx, query, tenantID, datasourceID, entityID, minConfidenceVal, limitVal)
	if err != nil {
		return nil, fmt.Errorf("failed to query suggestions: %w", err)
	}
	defer rows.Close()

	var suggestions []*RelationshipSuggestionGQL
	for rows.Next() {
		var s RelationshipSuggestionGQL
		var scoringJSON []byte

		err := rows.Scan(
			&s.ID, &s.TenantID, &s.DatasourceID, &s.SourceEntityID, &s.TargetEntityID,
			&s.Confidence, &s.Rationale, &scoringJSON, &s.Accepted, &s.AcceptedAt,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			continue
		}

		// Parse scoring breakdown
		err = json.Unmarshal(scoringJSON, &s.ScoringBreakdown)
		if err != nil {
			// Default breakdown if parsing fails
			s.ScoringBreakdown = ScoringBreakdownGQL{
				ForeignKeyPresence: 0,
				JoinFrequency:      0,
				NameSimilarity:     0,
				TextSimilarity:     0,
				EdgeTypePrior:      0,
			}
		}

		suggestions = append(suggestions, &s)
	}

	return &RelationshipSuggestionListGQL{
		Suggestions: suggestions,
		Count:       len(suggestions),
	}, nil
}

func (r *Resolver) LinkedModels(ctx context.Context, entityID string) ([]*SemanticModelGQL, error) {
	tenantID := ctx.Value("tenant_id").(string)
	datasourceID := ctx.Value("datasource_id").(string)

	// Get semantic assets to find linked model IDs
	query := `
		SELECT core_model_id, custom_model_id
		FROM semantic_assets
		WHERE tenant_id = $1 AND datasource_id = $2 AND business_entity_id = $3
	`

	var coreModelID, customModelID *string
	err := r.DB.QueryRowContext(ctx, query, tenantID, datasourceID, entityID).Scan(&coreModelID, &customModelID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query semantic assets: %w", err)
	}

	var models []*SemanticModelGQL
	var modelIDs []string
	if coreModelID != nil {
		modelIDs = append(modelIDs, *coreModelID)
	}
	if customModelID != nil {
		modelIDs = append(modelIDs, *customModelID)
	}

	if len(modelIDs) == 0 {
		return []*SemanticModelGQL{}, nil
	}

	// Fetch model details from catalog_node
	query = `
		SELECT id, node_name, node_type, node_description, created_at
		FROM catalog_node
		WHERE id = ANY($1::uuid[]) AND tenant_id = $2
	`

	rows, err := r.DB.QueryContext(ctx, query, pq.Array(modelIDs), tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query models: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var m SemanticModelGQL
		err := rows.Scan(&m.ID, &m.Name, &m.Type, &m.Description, &m.CreatedAt)
		if err != nil {
			continue
		}
		m.SourceKeys = []string{}
		models = append(models, &m)
	}

	return models, nil
}

func (r *Resolver) RelatedObjects(ctx context.Context, entityID string) (*ObjectGraphNodeGQL, error) {
	tenantID := ctx.Value("tenant_id").(string)
	datasourceID := ctx.Value("datasource_id").(string)

	// Get node details
	nodeQuery := `SELECT node_name, node_type FROM catalog_node WHERE id = $1 AND tenant_id = $2`
	var nodeName, nodeType string
	err := r.DB.QueryRowContext(ctx, nodeQuery, entityID, tenantID).Scan(&nodeName, &nodeType)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("node not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query node: %w", err)
	}

	// Get links to
	linksToQuery := `
		SELECT target_node_id FROM catalog_edge
		WHERE source_node_id = $1 AND tenant_id = $2 AND datasource_id = $3
	`
	linksToRows, err := r.DB.QueryContext(ctx, linksToQuery, entityID, tenantID, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query links_to: %w", err)
	}
	defer linksToRows.Close()

	var linksTo []string
	for linksToRows.Next() {
		var targetID string
		if err := linksToRows.Scan(&targetID); err != nil {
			continue
		}
		linksTo = append(linksTo, targetID)
	}

	// Get links from
	linksFromQuery := `
		SELECT source_node_id FROM catalog_edge
		WHERE target_node_id = $1 AND tenant_id = $2 AND datasource_id = $3
	`
	linksFromRows, err := r.DB.QueryContext(ctx, linksFromQuery, entityID, tenantID, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query links_from: %w", err)
	}
	defer linksFromRows.Close()

	var linksFrom []string
	for linksFromRows.Next() {
		var sourceID string
		if err := linksFromRows.Scan(&sourceID); err != nil {
			continue
		}
		linksFrom = append(linksFrom, sourceID)
	}

	return &ObjectGraphNodeGQL{
		ID:        entityID,
		Name:      nodeName,
		Type:      nodeType,
		LinksTo:   linksTo,
		LinksFrom: linksFrom,
	}, nil
}

func (r *Resolver) TraverseObjectGraph(ctx context.Context, startNodeID string, dotPath string) (*ObjectGraphPathGQL, error) {
	tenantID := ctx.Value("tenant_id").(string)
	datasourceID := ctx.Value("datasource_id").(string)

	var nodes []string
	currentID := startNodeID

	segments := strings.Split(dotPath, ".")
	for _, segment := range segments {
		query := `
			SELECT target_node_id FROM catalog_edge
			WHERE source_node_id = $1 AND tenant_id = $2 AND datasource_id = $3
		`

		rows, err := r.DB.QueryContext(ctx, query, currentID, tenantID, datasourceID)
		if err != nil {
			return nil, fmt.Errorf("failed to query edges: %w", err)
		}
		defer rows.Close()

		found := false
		for rows.Next() {
			var targetID string
			if err := rows.Scan(&targetID); err != nil {
				continue
			}

			// Check if target matches segment name
			nodeQuery := "SELECT id FROM catalog_node WHERE id = $1 AND node_name ILIKE $2 AND tenant_id = $3"
			var matchID string
			err := r.DB.QueryRowContext(ctx, nodeQuery, targetID, "%"+segment+"%", tenantID).Scan(&matchID)
			if err == nil {
				nodes = append(nodes, matchID)
				currentID = matchID
				found = true
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("path segment '%s' not found", segment)
		}
	}

	return &ObjectGraphPathGQL{
		Nodes: nodes,
		Path:  dotPath,
	}, nil
}

// ============================================================================
// Mutation Resolvers
// ============================================================================

func (r *Resolver) GenerateCoreModel(ctx context.Context, entityID string, modelName string, sourceKeys []string) (*SemanticModelGQL, error) {
	tenantID := ctx.Value("tenant_id").(string)
	datasourceID := ctx.Value("datasource_id").(string)

	// Create catalog node
	modelID := uuid.New().String()
	now := time.Now()

	insertNodeQuery := `
		INSERT INTO catalog_node (
			id, tenant_id, datasource_id, node_name, node_type,
			node_description, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var createdID string
	err := r.DB.QueryRowContext(ctx, insertNodeQuery,
		modelID, tenantID, datasourceID, modelName, "model",
		"Auto-generated core model", now, now,
	).Scan(&createdID)

	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	// Update semantic assets
	assetID := uuid.New().String()
	updateAssetQuery := `
		INSERT INTO semantic_assets (
			id, tenant_id, datasource_id, business_entity_id, core_model_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, datasource_id, business_entity_id)
		DO UPDATE SET core_model_id = $5, updated_at = $6
	`

	_, err = r.DB.ExecContext(ctx, updateAssetQuery,
		assetID, tenantID, datasourceID, entityID, createdID, now, now,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update semantic assets: %w", err)
	}

	return &SemanticModelGQL{
		ID:         createdID,
		Name:       modelName,
		Type:       "core",
		SourceKeys: sourceKeys,
		CreatedAt:  now,
	}, nil
}

func (r *Resolver) GenerateCoreView(ctx context.Context, entityID string, viewName string, selectedColumns []string) (*SemanticViewGQL, error) {
	tenantID := ctx.Value("tenant_id").(string)
	datasourceID := ctx.Value("datasource_id").(string)

	// Create catalog node
	viewID := uuid.New().String()
	now := time.Now()

	insertNodeQuery := `
		INSERT INTO catalog_node (
			id, tenant_id, datasource_id, node_name, node_type,
			node_description, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var createdID string
	err := r.DB.QueryRowContext(ctx, insertNodeQuery,
		viewID, tenantID, datasourceID, viewName, "view",
		"Auto-generated core view", now, now,
	).Scan(&createdID)

	if err != nil {
		return nil, fmt.Errorf("failed to create view: %w", err)
	}

	// Update semantic assets
	assetID := uuid.New().String()
	updateAssetQuery := `
		INSERT INTO semantic_assets (
			id, tenant_id, datasource_id, business_entity_id, core_view_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, datasource_id, business_entity_id)
		DO UPDATE SET core_view_id = $5, updated_at = $6
	`

	_, err = r.DB.ExecContext(ctx, updateAssetQuery,
		assetID, tenantID, datasourceID, entityID, createdID, now, now,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update semantic assets: %w", err)
	}

	return &SemanticViewGQL{
		ID:              createdID,
		Name:            viewName,
		Type:            "core",
		SelectedColumns: selectedColumns,
		CreatedAt:       now,
	}, nil
}

func (r *Resolver) CreateCustomModel(ctx context.Context, entityID string, modelName string, expression string, sourceKeys []string) (*SemanticModelGQL, error) {
	tenantID := ctx.Value("tenant_id").(string)
	datasourceID := ctx.Value("datasource_id").(string)

	// Create catalog node
	modelID := uuid.New().String()
	now := time.Now()
	description := fmt.Sprintf("Custom model: %s", expression)

	insertNodeQuery := `
		INSERT INTO catalog_node (
			id, tenant_id, datasource_id, node_name, node_type,
			node_description, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var createdID string
	err := r.DB.QueryRowContext(ctx, insertNodeQuery,
		modelID, tenantID, datasourceID, modelName, "model", description, now, now,
	).Scan(&createdID)

	if err != nil {
		return nil, fmt.Errorf("failed to create custom model: %w", err)
	}

	// Update semantic assets
	assetID := uuid.New().String()
	updateAssetQuery := `
		INSERT INTO semantic_assets (
			id, tenant_id, datasource_id, business_entity_id, custom_model_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, datasource_id, business_entity_id)
		DO UPDATE SET custom_model_id = $5, updated_at = $6
	`

	_, err = r.DB.ExecContext(ctx, updateAssetQuery,
		assetID, tenantID, datasourceID, entityID, createdID, now, now,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update semantic assets: %w", err)
	}

	return &SemanticModelGQL{
		ID:         createdID,
		Name:       modelName,
		Type:       "custom",
		SourceKeys: sourceKeys,
		CreatedAt:  now,
	}, nil
}

func (r *Resolver) CreateCustomView(ctx context.Context, entityID string, viewName string, expression string, sourceKeys []string) (*SemanticViewGQL, error) {
	tenantID := ctx.Value("tenant_id").(string)
	datasourceID := ctx.Value("datasource_id").(string)

	// Create catalog node
	viewID := uuid.New().String()
	now := time.Now()
	description := fmt.Sprintf("Custom view: %s", expression)

	insertNodeQuery := `
		INSERT INTO catalog_node (
			id, tenant_id, datasource_id, node_name, node_type,
			node_description, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var createdID string
	err := r.DB.QueryRowContext(ctx, insertNodeQuery,
		viewID, tenantID, datasourceID, viewName, "view", description, now, now,
	).Scan(&createdID)

	if err != nil {
		return nil, fmt.Errorf("failed to create custom view: %w", err)
	}

	// Update semantic assets
	assetID := uuid.New().String()
	updateAssetQuery := `
		INSERT INTO semantic_assets (
			id, tenant_id, datasource_id, business_entity_id, custom_view_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, datasource_id, business_entity_id)
		DO UPDATE SET custom_view_id = $5, updated_at = $6
	`

	_, err = r.DB.ExecContext(ctx, updateAssetQuery,
		assetID, tenantID, datasourceID, entityID, createdID, now, now,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update semantic assets: %w", err)
	}

	return &SemanticViewGQL{
		ID:              createdID,
		Name:            viewName,
		Type:            "custom",
		SelectedColumns: sourceKeys,
		CreatedAt:       now,
	}, nil
}

func (r *Resolver) ApplyRelationshipSuggestion(ctx context.Context, suggestionID string) (*RelationshipSuggestionGQL, error) {
	tenantID := ctx.Value("tenant_id").(string)
	datasourceID := ctx.Value("datasource_id").(string)

	// Fetch suggestion
	query := `
		SELECT id, source_entity_id, target_entity_id
		FROM relationship_suggestions
		WHERE id = $1 AND tenant_id = $2
	`

	var sourceID, targetID string
	err := r.DB.QueryRowContext(ctx, query, suggestionID, tenantID).Scan(&suggestionID, &sourceID, &targetID)
	if err != nil {
		return nil, fmt.Errorf("suggestion not found: %w", err)
	}

	// Create edge in catalog
	edgeID := uuid.New().String()
	now := time.Now()

	createEdgeQuery := `
		INSERT INTO catalog_edge
		(id, tenant_id, datasource_id, source_node_id, target_node_id, edge_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'suggests', $6, $7)
	`

	_, err = r.DB.ExecContext(ctx, createEdgeQuery, edgeID, tenantID, datasourceID, sourceID, targetID, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create edge: %w", err)
	}

	// Mark suggestion as accepted
	updateQuery := `
		UPDATE relationship_suggestions
		SET accepted = true, accepted_at = $1, updated_at = $2
		WHERE id = $3
	`
	_, err = r.DB.ExecContext(ctx, updateQuery, now, now, suggestionID)
	if err != nil {
		return nil, fmt.Errorf("failed to mark suggestion as accepted: %w", err)
	}

	// Return updated suggestion
	suggestion := &RelationshipSuggestionGQL{
		ID:             suggestionID,
		TenantID:       tenantID,
		DatasourceID:   datasourceID,
		SourceEntityID: sourceID,
		TargetEntityID: targetID,
		Accepted:       true,
		AcceptedAt:     &now,
		UpdatedAt:      now,
	}

	return suggestion, nil
}

func (r *Resolver) TraverseObjectGraphMutation(ctx context.Context, startNodeID string, dotPath string) (*ObjectGraphPathGQL, error) {
	return r.TraverseObjectGraph(ctx, startNodeID, dotPath)
}

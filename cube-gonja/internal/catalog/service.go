package catalog

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"
)

// CubeModel represents a parsed Cube.js model
type CubeModel struct {
	Cubes []Cube `yaml:"cubes"`
}

type Cube struct {
	Name          string         `yaml:"name"`
	SQLTable      string         `yaml:"sql_table"`
	DataSource    string         `yaml:"data_source"`
	Title         string         `yaml:"title"`
	Description   string         `yaml:"description"`
	BusinessTerms []BusinessTerm `yaml:"business_terms"`
	Measures      []Measure      `yaml:"measures"`
	Dimensions    []Dimension    `yaml:"dimensions"`
	Joins         []Join         `yaml:"joins"`
	Views         []View         `yaml:"views"`
}

type BusinessTerm struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Category    string   `yaml:"category"`
	SubCategory string   `yaml:"sub_category,omitempty"`
	Owner       string   `yaml:"owner,omitempty"`
	Steward     string   `yaml:"steward,omitempty"`
	Status      string   `yaml:"status,omitempty"` // draft, approved, deprecated
	Version     string   `yaml:"version,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	ParentID    string   `yaml:"parent_id,omitempty"`
}

type BusinessTermSearchRequest struct {
	Query    string   `json:"query,omitempty"`    // Search query
	Category string   `json:"category,omitempty"` // Filter by category
	Status   string   `json:"status,omitempty"`   // Filter by status
	Tags     []string `json:"tags,omitempty"`     // Filter by tags
	Limit    int      `json:"limit,omitempty"`    // Max results (default 50)
	Offset   int      `json:"offset,omitempty"`   // Pagination offset
}

type BusinessTermResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	SubCategory string   `json:"sub_category,omitempty"`
	Owner       string   `json:"owner,omitempty"`
	Steward     string   `json:"steward,omitempty"`
	Status      string   `json:"status,omitempty"`
	Version     string   `json:"version,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	ParentID    string   `json:"parent_id,omitempty"`
}

type BusinessTermValidationResponse struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type Join struct {
	Name         string `yaml:"name"`
	SQL          string `yaml:"sql"`
	Relationship string `yaml:"relationship"`
}

type View struct {
	Name string `yaml:"name"`
}

type Measure struct {
	Name          string         `yaml:"name"`
	Type          string         `yaml:"type"`
	SQL           string         `yaml:"sql,omitempty"`
	Description   string         `yaml:"description"`
	BusinessTerms []BusinessTerm `yaml:"business_terms"`
}

type Dimension struct {
	Name          string         `yaml:"name"`
	SQL           string         `yaml:"sql"`
	Type          string         `yaml:"type"`
	PrimaryKey    bool           `yaml:"primary_key,omitempty"`
	Description   string         `yaml:"description"`
	BusinessTerms []BusinessTerm `yaml:"business_terms"`
}

// CatalogService handles catalog table updates
type CatalogService struct {
	db           *sql.DB
	tenantID     string
	datasourceID string
}

// DB returns the underlying sql.DB connection
func (s *CatalogService) DB() *sql.DB {
	return s.db
}

// NewCatalogService creates a new catalog service
func NewCatalogService(db *sql.DB, tenantID, datasourceID string) *CatalogService {
	return &CatalogService{
		db:           db,
		tenantID:     tenantID,
		datasourceID: datasourceID,
	}
}

// UpdateCatalogFromModels parses generated model files and updates catalog tables
func (s *CatalogService) UpdateCatalogFromModels(modelDir string) error {
	files, err := filepath.Glob(filepath.Join(modelDir, "*.yml"))
	if err != nil {
		return fmt.Errorf("failed to list model files: %w", err)
	}

	for _, file := range files {
		if err := s.processModelFile(file); err != nil {
			log.Printf("Error processing model file %s: %v", file, err)
			continue
		}
	}

	return nil
}

// processModelFile parses a single Cube.js model file and updates catalog
func (s *CatalogService) processModelFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read model file: %w", err)
	}

	var model CubeModel
	if err := yaml.Unmarshal(data, &model); err != nil {
		return fmt.Errorf("failed to parse model file: %w", err)
	}

	for _, cube := range model.Cubes {
		if err := s.updateCatalogForCube(cube); err != nil {
			return fmt.Errorf("failed to update catalog for cube %s: %w", cube.Name, err)
		}
	}

	return nil
}

// updateCatalogForCube updates catalog tables for a single cube
func (s *CatalogService) updateCatalogForCube(cube Cube) error {
	// Determine if this is a view or a regular model
	isView := s.isCubeAView(cube)
	nodeTypeName := "semantic_model"
	if isView {
		nodeTypeName = "semantic_view"
	}

	// Get node type ID
	nodeTypeID, err := s.getNodeTypeID(nodeTypeName)
	if err != nil {
		return fmt.Errorf("failed to get %s node type: %w", nodeTypeName, err)
	}

	// Get semantic column node type ID
	semanticColumnTypeID, err := s.getNodeTypeID("semantic_column")
	if err != nil {
		return fmt.Errorf("failed to get semantic_column node type: %w", err)
	}

	// Insert or update semantic model/view node
	tc := cases.Title(language.Und)
	modelNodeID, err := s.upsertCatalogNode(
		nodeTypeID,
		cube.Name,
		fmt.Sprintf("Semantic %s for %s", tc.String(nodeTypeName), cube.Title),
		map[string]interface{}{
			"sql_table":         cube.SQLTable,
			"data_source":       cube.DataSource,
			"is_view":           isView,
			"description":       cube.Description,
			"business_term_ids": s.extractBusinessTermIDs(cube.BusinessTerms),
			"business_terms":    s.extractBusinessTermNames(cube.BusinessTerms),
		},
		fmt.Sprintf("/semantic_%s/%s", nodeTypeName, cube.Name),
		nil, // parent_id
	)
	if err != nil {
		return fmt.Errorf("failed to upsert semantic %s node: %w", nodeTypeName, err)
	}

	// Process measures
	for _, measure := range cube.Measures {
		measureNodeID, err := s.upsertCatalogNode(
			semanticColumnTypeID,
			fmt.Sprintf("%s.%s", cube.Name, measure.Name),
			fmt.Sprintf("Measure: %s (%s)", measure.Name, measure.Type),
			map[string]interface{}{
				"type":              "measure",
				"measure_type":      measure.Type,
				"sql":               measure.SQL,
				"description":       measure.Description,
				"business_term_ids": s.extractBusinessTermIDs(measure.BusinessTerms),
				"business_terms":    s.extractBusinessTermNames(measure.BusinessTerms),
			},
			fmt.Sprintf("/semantic_models/%s/measures/%s", cube.Name, measure.Name),
			&modelNodeID,
		)
		if err != nil {
			log.Printf("Failed to upsert measure node %s: %v", measure.Name, err)
			continue
		}

		// Process business terms for this measure
		if len(measure.BusinessTerms) > 0 {
			if err := s.processElementBusinessTerms(measure.BusinessTerms, measureNodeID, "measure", measure.Name); err != nil {
				log.Printf("Failed to process business terms for measure %s: %v", measure.Name, err)
			}
		}
	}

	// Process dimensions
	for _, dimension := range cube.Dimensions {
		dimensionNodeID, err := s.upsertCatalogNode(
			semanticColumnTypeID,
			fmt.Sprintf("%s.%s", cube.Name, dimension.Name),
			fmt.Sprintf("Dimension: %s (%s)", dimension.Name, dimension.Type),
			map[string]interface{}{
				"type":              "dimension",
				"dimension_type":    dimension.Type,
				"sql":               dimension.SQL,
				"primary_key":       dimension.PrimaryKey,
				"description":       dimension.Description,
				"business_term_ids": s.extractBusinessTermIDs(dimension.BusinessTerms),
				"business_terms":    s.extractBusinessTermNames(dimension.BusinessTerms),
			},
			fmt.Sprintf("/semantic_models/%s/dimensions/%s", cube.Name, dimension.Name),
			&modelNodeID,
		)
		if err != nil {
			log.Printf("Failed to upsert dimension node %s: %v", dimension.Name, err)
			continue
		}

		// Process business terms for this dimension
		if len(dimension.BusinessTerms) > 0 {
			if err := s.processElementBusinessTerms(dimension.BusinessTerms, dimensionNodeID, "dimension", dimension.Name); err != nil {
				log.Printf("Failed to process business terms for dimension %s: %v", dimension.Name, err)
			}
		}
	}

	// Process joins and create edges between models
	if err := s.processCubeJoins(cube, modelNodeID); err != nil {
		log.Printf("Failed to process joins for cube %s: %v", cube.Name, err)
	}

	// Process references in measures and dimensions
	if err := s.processCubeReferences(cube, modelNodeID); err != nil {
		log.Printf("Failed to process references for cube %s: %v", cube.Name, err)
	}

	// Process business terms and create links
	if err := s.processCubeBusinessTerms(cube, modelNodeID); err != nil {
		log.Printf("Failed to process business terms for cube %s: %v", cube.Name, err)
	}

	return nil
}

// processCubeJoins processes joins in a cube and creates edges between models
func (s *CatalogService) processCubeJoins(cube Cube, sourceModelID string) error {
	for _, join := range cube.Joins {
		// Extract the target cube name from the join SQL or name
		targetCubeName := s.extractTargetCubeName(join)

		if targetCubeName == "" {
			log.Printf("Could not extract target cube name from join: %+v", join)
			continue
		}

		// Get the target model node ID
		targetModelID, err := s.getModelNodeID(targetCubeName)
		if err != nil {
			log.Printf("Could not find target model node for %s: %v", targetCubeName, err)
			continue
		}

		// Create edge between models
		edgeTypeID, err := s.getEdgeTypeID("joins")
		if err != nil {
			log.Printf("Could not get joins edge type: %v", err)
			continue
		}

		if err := s.createModelRelationshipEdge(sourceModelID, targetModelID, edgeTypeID, join); err != nil {
			log.Printf("Failed to create join edge from %s to %s: %v", cube.Name, targetCubeName, err)
		}
	}

	return nil
}

// extractTargetCubeName extracts the target cube name from a join definition
func (s *CatalogService) extractTargetCubeName(join Join) string {
	// Try to extract from join name first
	if join.Name != "" {
		return join.Name
	}

	// Try to parse from SQL (basic parsing for common patterns)
	// This is a simplified implementation - in practice, you'd want more sophisticated SQL parsing
	sql := join.SQL
	if sql == "" {
		return ""
	}

	// Look for patterns like "JOIN cube_name ON ..." or "FROM cube_name"
	// This is a basic implementation - you might want to use a proper SQL parser
	return s.parseCubeNameFromSQL(sql)
}

// parseCubeNameFromSQL attempts to extract cube name from SQL
func (s *CatalogService) parseCubeNameFromSQL(sql string) string {
	// This is a simplified implementation
	// In a real implementation, you'd use a proper SQL parser
	// For now, we'll look for common patterns

	// Remove extra whitespace and convert to lowercase for easier parsing
	sql = strings.ToLower(strings.TrimSpace(sql))

	// Look for "join cube_name" pattern
	if strings.Contains(sql, "join ") {
		parts := strings.Split(sql, "join ")
		if len(parts) > 1 {
			afterJoin := strings.TrimSpace(parts[1])
			// Take the first word as the cube name
			words := strings.Fields(afterJoin)
			if len(words) > 0 {
				return words[0]
			}
		}
	}

	return ""
}

// getModelNodeID gets the node ID for a semantic model by name
func (s *CatalogService) getModelNodeID(cubeName string) (string, error) {
	var nodeID string
	err := s.db.QueryRow(`
		SELECT cn.id
		FROM public.catalog_node cn
		JOIN public.catalog_node_type cnt ON cn.node_type_id = cnt.id
		WHERE cn.tenant_datasource_id = $1
		AND cnt.catalog_type_name = 'semantic_model'
		AND cn.node_name = $2
	`, s.datasourceID, cubeName).Scan(&nodeID)

	return nodeID, err
}

// createModelRelationshipEdge creates an edge between two semantic models
func (s *CatalogService) createModelRelationshipEdge(sourceID, targetID, edgeTypeID string, join Join) error {
	properties := map[string]interface{}{
		"relationship": join.Relationship,
		"sql":          join.SQL,
		"join_type":    "join",
	}

	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		return fmt.Errorf("failed to marshal edge properties: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO public.catalog_edge (
			tenant_datasource_id, source_node_id, target_node_id, relationship_type,
			properties, edge_type_id, created_at, updated_at, tenant_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id) DO UPDATE SET
			properties = EXCLUDED.properties,
			updated_at = EXCLUDED.updated_at
	`, s.datasourceID, sourceID, targetID, "joins", propertiesJSON, edgeTypeID,
		time.Now(), time.Now(), s.tenantID)

	return err
}

// Scheduled job persistence and history
type ScheduledJob struct {
	ID                string                 `json:"id"`
	TenantID          string                 `json:"tenant_id"`
	DatasourceID      string                 `json:"datasource_id"`
	CubeName          string                 `json:"cube_name"`
	PreName           string                 `json:"pre_name"`
	CronExpr          string                 `json:"cron_expr,omitempty"`
	Storage           string                 `json:"storage,omitempty"`
	RefreshKey        map[string]interface{} `json:"refresh_key,omitempty"`
	LastRun           *time.Time             `json:"last_run,omitempty"`
	LastRefreshKeyVal string                 `json:"last_refresh_key_val,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// UpsertScheduledJob inserts or updates a scheduled pre-aggregation entry
func (s *CatalogService) UpsertScheduledJob(job ScheduledJob) error {
	rkJSON := []byte("null")
	if job.RefreshKey != nil {
		b, _ := json.Marshal(job.RefreshKey)
		rkJSON = b
	}
	_, err := s.db.Exec(`
		INSERT INTO public.scheduled_jobs (id, tenant_id, datasource_id, cube_name, pre_name, cron_expr, storage, refresh_key, last_run, last_refresh_key_val, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (id) DO UPDATE SET
			cron_expr = EXCLUDED.cron_expr,
			storage = EXCLUDED.storage,
			refresh_key = EXCLUDED.refresh_key,
			last_run = EXCLUDED.last_run,
			last_refresh_key_val = EXCLUDED.last_refresh_key_val,
			updated_at = EXCLUDED.updated_at
	`, job.ID, job.TenantID, job.DatasourceID, job.CubeName, job.PreName, job.CronExpr, job.Storage, rkJSON, job.LastRun, job.LastRefreshKeyVal, job.CreatedAt, job.UpdatedAt)
	return err
}

// DeleteScheduledJob removes a scheduled job by id
func (s *CatalogService) DeleteScheduledJob(id string) error {
	_, err := s.db.Exec(`DELETE FROM public.scheduled_jobs WHERE id = $1`, id)
	return err
}

// ListScheduledJobs returns all scheduled jobs for the datasource
func (s *CatalogService) ListScheduledJobs() ([]ScheduledJob, error) {
	rows, err := s.db.Query(`SELECT id, tenant_id, datasource_id, cube_name, pre_name, cron_expr, storage, refresh_key, last_run, last_refresh_key_val, created_at, updated_at FROM public.scheduled_jobs WHERE datasource_id = $1`, s.datasourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ScheduledJob
	for rows.Next() {
		var r ScheduledJob
		var rk sql.NullString
		var lastRun sql.NullTime
		var lastKey sql.NullString
		if err := rows.Scan(&r.ID, &r.TenantID, &r.DatasourceID, &r.CubeName, &r.PreName, &r.CronExpr, &r.Storage, &rk, &lastRun, &lastKey, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		if rk.Valid && rk.String != "" {
			var m map[string]interface{}
			_ = json.Unmarshal([]byte(rk.String), &m)
			r.RefreshKey = m
		}
		if lastRun.Valid {
			r.LastRun = &lastRun.Time
		}
		if lastKey.Valid {
			r.LastRefreshKeyVal = lastKey.String
		}
		out = append(out, r)
	}
	return out, nil
}

// RecordJobRun inserts a run record for a scheduled job
func (s *CatalogService) RecordJobRun(jobID string, started time.Time, finished *time.Time, success bool, message string) error {
	_, err := s.db.Exec(`INSERT INTO public.scheduled_job_runs (job_id, started_at, finished_at, success, message) VALUES ($1,$2,$3,$4,$5)`, jobID, started, finished, success, message)
	return err
}

// isCubeAView determines if a cube represents a view based on its structure
func (s *CatalogService) isCubeAView(cube Cube) bool {
	// A cube is considered a view if:
	// 1. It has joins (combines multiple tables)
	// 2. Its SQL table contains keywords like 'view', 'union', 'join'
	// 3. It has multiple data sources
	// 4. It references other cubes in its measures/dimensions

	if len(cube.Joins) > 0 {
		return true
	}

	sqlTable := strings.ToLower(cube.SQLTable)
	if strings.Contains(sqlTable, "view") ||
		strings.Contains(sqlTable, "union") ||
		strings.Contains(sqlTable, "join") {
		return true
	}

	// Check if measures or dimensions reference other cubes
	for _, measure := range cube.Measures {
		if measure.SQL != "" && len(s.extractReferencedCubes(measure.SQL)) > 0 {
			return true
		}
	}

	for _, dimension := range cube.Dimensions {
		if dimension.SQL != "" && len(s.extractReferencedCubes(dimension.SQL)) > 0 {
			return true
		}
	}

	return false
}

// validateBusinessTerm validates a business term structure
func (s *CatalogService) validateBusinessTerm(term BusinessTerm) error {
	if term.ID == "" {
		return fmt.Errorf("business term ID is required")
	}
	if term.Name == "" {
		return fmt.Errorf("business term name is required")
	}
	if term.Category == "" {
		return fmt.Errorf("business term category is required")
	}

	// Validate ID format (should be something like BT-CATEGORY-XXX)
	if !strings.HasPrefix(term.ID, "BT-") {
		return fmt.Errorf("business term ID should start with 'BT-'")
	}

	// Validate status if provided
	if term.Status != "" {
		validStatuses := []string{"draft", "approved", "deprecated", "archived"}
		valid := false
		for _, status := range validStatuses {
			if term.Status == status {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid business term status: %s", term.Status)
		}
	}

	return nil
}

// validateBusinessTermReferences validates that referenced business terms exist
func (s *CatalogService) validateBusinessTermReferences(businessTerms []BusinessTerm) error {
	for _, term := range businessTerms {
		if term.ParentID != "" {
			// Check if parent business term exists
			exists, err := s.businessTermExists(term.ParentID)
			if err != nil {
				return fmt.Errorf("error checking parent business term %s: %w", term.ParentID, err)
			}
			if !exists {
				return fmt.Errorf("parent business term %s does not exist", term.ParentID)
			}
		}
	}
	return nil
}

// extractBusinessTermIDs extracts business term IDs from business terms list
func (s *CatalogService) extractBusinessTermIDs(businessTerms []BusinessTerm) []string {
	var ids []string
	for _, term := range businessTerms {
		if term.ID != "" {
			ids = append(ids, term.ID)
		}
	}
	return ids
}

// businessTermExists checks if a business term exists in the catalog
func (s *CatalogService) businessTermExists(termID string) (bool, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM public.catalog_node cn
		JOIN public.catalog_node_type cnt ON cn.node_type_id = cnt.id
		WHERE cnt.name = 'business_term'
		AND cn.name = $1
		AND cn.tenant_id = $2
	`, termID, s.tenantID).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// extractBusinessTermNames extracts business term names from business terms list
func (s *CatalogService) extractBusinessTermNames(businessTerms []BusinessTerm) []string {
	var names []string
	for _, term := range businessTerms {
		names = append(names, term.Name)
	}
	return names
}

// processElementBusinessTerms processes business terms for a semantic element (measure/dimension)
func (s *CatalogService) processElementBusinessTerms(businessTerms []BusinessTerm, elementNodeID, elementType, elementName string) error {
	// Apply inheritance logic if no explicit business terms
	if len(businessTerms) == 0 {
		log.Printf("No explicit business terms for %s %s, inheritance will be handled at cube level", elementType, elementName)
		return nil
	}

	for _, businessTerm := range businessTerms {
		// Validate business term
		if err := s.validateBusinessTerm(businessTerm); err != nil {
			log.Printf("Business term validation failed for %s: %v", businessTerm.ID, err)
			continue
		}

		// Validate references
		if err := s.validateBusinessTermReferences([]BusinessTerm{businessTerm}); err != nil {
			log.Printf("Business term reference validation failed for %s: %v", businessTerm.ID, err)
			continue
		}

		// Create or update business term node
		businessTermNodeID, err := s.upsertBusinessTermNode(businessTerm)
		if err != nil {
			log.Printf("Failed to upsert business term node %s: %v", businessTerm.Name, err)
			continue
		}

		// Create edge between element and business term
		if err := s.createElementBusinessTermLink(elementNodeID, businessTermNodeID, businessTerm, elementType, elementName); err != nil {
			log.Printf("Failed to create business term link for %s: %v", businessTerm.Name, err)
		}

		// Create parent-child relationships if applicable
		if businessTerm.ParentID != "" {
			if err := s.createBusinessTermHierarchy(businessTerm, businessTermNodeID); err != nil {
				log.Printf("Failed to create business term hierarchy for %s: %v", businessTerm.ID, err)
			}
		}
	}

	return nil
}

// createElementBusinessTermLink creates an edge between a semantic element and a business term
func (s *CatalogService) createElementBusinessTermLink(elementNodeID, businessTermNodeID string, businessTerm BusinessTerm, elementType, elementName string) error {
	edgeTypeID, err := s.getEdgeTypeID("member_of")
	if err != nil {
		return fmt.Errorf("could not get member_of edge type: %w", err)
	}

	properties := map[string]interface{}{
		"business_term_id":  businessTerm.ID,
		"element_type":      elementType,
		"element_name":      elementName,
		"relationship_type": "semantic_mapping",
		"category":          businessTerm.Category,
	}

	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		return fmt.Errorf("failed to marshal edge properties: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO public.catalog_edge (
			tenant_datasource_id, source_node_id, target_node_id, relationship_type,
			properties, edge_type_id, created_at, updated_at, tenant_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id) DO UPDATE SET
			properties = EXCLUDED.properties,
			updated_at = EXCLUDED.updated_at
	`, s.datasourceID, businessTermNodeID, elementNodeID, "member_of", propertiesJSON, edgeTypeID,
		time.Now(), time.Now(), s.tenantID)

	return err
}

// processCubeBusinessTerms processes business terms for a cube and creates links
func (s *CatalogService) processCubeBusinessTerms(cube Cube, modelNodeID string) error {
	for _, businessTerm := range cube.BusinessTerms {
		// Validate business term
		if err := s.validateBusinessTerm(businessTerm); err != nil {
			log.Printf("Business term validation failed for %s: %v", businessTerm.ID, err)
			continue
		}

		// Validate references
		if err := s.validateBusinessTermReferences([]BusinessTerm{businessTerm}); err != nil {
			log.Printf("Business term reference validation failed for %s: %v", businessTerm.ID, err)
			continue
		}

		// Create or update business term node
		businessTermNodeID, err := s.upsertBusinessTermNode(businessTerm)
		if err != nil {
			log.Printf("Failed to upsert business term node %s: %v", businessTerm.Name, err)
			continue
		}

		// Create edge between model and business term
		if err := s.createBusinessTermLink(modelNodeID, businessTermNodeID, businessTerm); err != nil {
			log.Printf("Failed to create business term link for %s: %v", businessTerm.Name, err)
		}

		// Create parent-child relationships if applicable
		if businessTerm.ParentID != "" {
			if err := s.createBusinessTermHierarchy(businessTerm, businessTermNodeID); err != nil {
				log.Printf("Failed to create business term hierarchy for %s: %v", businessTerm.ID, err)
			}
		}
	}

	return nil
}

// upsertBusinessTermNode creates or updates a business term node
func (s *CatalogService) upsertBusinessTermNode(businessTerm BusinessTerm) (string, error) {
	businessTermTypeID, err := s.getNodeTypeID("business_term")
	if err != nil {
		return "", fmt.Errorf("failed to get business_term node type: %w", err)
	}

	// Use business term ID as the node name if available, otherwise use name
	nodeName := businessTerm.Name
	if businessTerm.ID != "" {
		nodeName = businessTerm.ID
	}

	properties := map[string]interface{}{
		"business_term_id": businessTerm.ID,
		"name":             businessTerm.Name,
		"description":      businessTerm.Description,
		"category":         businessTerm.Category,
		"sub_category":     businessTerm.SubCategory,
		"owner":            businessTerm.Owner,
		"steward":          businessTerm.Steward,
		"status":           businessTerm.Status,
		"version":          businessTerm.Version,
		"tags":             businessTerm.Tags,
		"parent_id":        businessTerm.ParentID,
	}

	return s.upsertCatalogNode(
		businessTermTypeID,
		nodeName,
		fmt.Sprintf("Business Term: %s", businessTerm.Name),
		properties,
		fmt.Sprintf("/business_terms/%s", nodeName),
		nil, // parent_id
	)
}

// createBusinessTermHierarchy creates parent-child relationships between business terms
func (s *CatalogService) createBusinessTermHierarchy(businessTerm BusinessTerm, childNodeID string) error {
	// Get parent business term node ID
	parentNodeID, err := s.getBusinessTermNodeID(businessTerm.ParentID)
	if err != nil {
		return fmt.Errorf("failed to get parent business term node: %w", err)
	}

	// Create hierarchy edge
	edgeTypeID, err := s.getEdgeTypeID("parent_of")
	if err != nil {
		return fmt.Errorf("could not get parent_of edge type: %w", err)
	}

	properties := map[string]interface{}{
		"relationship_type": "hierarchy",
		"child_id":          businessTerm.ID,
		"parent_id":         businessTerm.ParentID,
	}

	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		return fmt.Errorf("failed to marshal hierarchy edge properties: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO public.catalog_edge (
			tenant_datasource_id, source_node_id, target_node_id, relationship_type,
			properties, edge_type_id, created_at, updated_at, tenant_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id) DO UPDATE SET
			properties = EXCLUDED.properties,
			updated_at = EXCLUDED.updated_at
	`, s.datasourceID, parentNodeID, childNodeID, "parent_of", propertiesJSON, edgeTypeID,
		time.Now(), time.Now(), s.tenantID)

	return err
}

// getBusinessTermNodeID gets the node ID for a business term
func (s *CatalogService) getBusinessTermNodeID(termID string) (string, error) {
	var nodeID string
	err := s.db.QueryRow(`
		SELECT cn.id FROM public.catalog_node cn
		JOIN public.catalog_node_type cnt ON cn.node_type_id = cnt.id
		WHERE cnt.name = 'business_term'
		AND cn.name = $1
		AND cn.tenant_id = $2
		LIMIT 1
	`, termID, s.tenantID).Scan(&nodeID)

	if err != nil {
		return "", fmt.Errorf("failed to get business term node ID for %s: %w", termID, err)
	}

	return nodeID, nil
}

// createBusinessTermLink creates an edge between a semantic model and a business term
func (s *CatalogService) createBusinessTermLink(modelNodeID, businessTermNodeID string, businessTerm BusinessTerm) error {
	edgeTypeID, err := s.getEdgeTypeID("has_semantic")
	if err != nil {
		return fmt.Errorf("could not get has_semantic edge type: %w", err)
	}

	properties := map[string]interface{}{
		"business_term_id":  businessTerm.ID,
		"relationship_type": "semantic_mapping",
		"category":          businessTerm.Category,
		"sub_category":      businessTerm.SubCategory,
		"status":            businessTerm.Status,
		"version":           businessTerm.Version,
		"tags":              businessTerm.Tags,
	}

	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		return fmt.Errorf("failed to marshal edge properties: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO public.catalog_edge (
			tenant_datasource_id, source_node_id, target_node_id, relationship_type,
			properties, edge_type_id, created_at, updated_at, tenant_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id) DO UPDATE SET
			properties = EXCLUDED.properties,
			updated_at = EXCLUDED.updated_at
	`, s.datasourceID, businessTermNodeID, modelNodeID, "has_semantic", propertiesJSON, edgeTypeID,
		time.Now(), time.Now(), s.tenantID)

	return err
}

// processCubeReferences processes references in measures and dimensions
func (s *CatalogService) processCubeReferences(cube Cube, sourceModelID string) error {
	// Process measure references
	for _, measure := range cube.Measures {
		if measure.SQL != "" {
			referencedCubes := s.extractReferencedCubes(measure.SQL)
			for _, refCube := range referencedCubes {
				if err := s.createReferenceEdge(sourceModelID, refCube, measure.Name, "measure"); err != nil {
					log.Printf("Failed to create reference edge for measure %s: %v", measure.Name, err)
				}
			}
		}
	}

	// Process dimension references
	for _, dimension := range cube.Dimensions {
		if dimension.SQL != "" {
			referencedCubes := s.extractReferencedCubes(dimension.SQL)
			for _, refCube := range referencedCubes {
				if err := s.createReferenceEdge(sourceModelID, refCube, dimension.Name, "dimension"); err != nil {
					log.Printf("Failed to create reference edge for dimension %s: %v", dimension.Name, err)
				}
			}
		}
	}

	return nil
}

// extractReferencedCubes extracts cube names referenced in SQL expressions
func (s *CatalogService) extractReferencedCubes(sql string) []string {
	var cubes []string
	if sql == "" {
		return cubes
	}

	// Look for patterns like cube_name.column_name
	// This is a simplified implementation - in practice, you'd want more sophisticated parsing
	sql = strings.ToLower(sql)

	// Split by common separators and look for cube.column patterns
	parts := strings.FieldsFunc(sql, func(r rune) bool {
		return r == ' ' || r == ',' || r == '(' || r == ')' || r == '+' || r == '-' || r == '*' || r == '/'
	})

	for _, part := range parts {
		if strings.Contains(part, ".") {
			dotParts := strings.Split(part, ".")
			if len(dotParts) >= 2 {
				cubeName := strings.TrimSpace(dotParts[0])
				if cubeName != "" && !contains(cubes, cubeName) {
					cubes = append(cubes, cubeName)
				}
			}
		}
	}

	return cubes
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// createReferenceEdge creates a reference edge between models
func (s *CatalogService) createReferenceEdge(sourceModelID, targetCubeName, elementName, elementType string) error {
	// Get target model node ID
	targetModelID, err := s.getModelNodeID(targetCubeName)
	if err != nil {
		return fmt.Errorf("could not find target model %s: %w", targetCubeName, err)
	}

	// Get references edge type
	edgeTypeID, err := s.getEdgeTypeID("references")
	if err != nil {
		return fmt.Errorf("could not get references edge type: %w", err)
	}

	properties := map[string]interface{}{
		"element_name":   elementName,
		"element_type":   elementType,
		"reference_type": "sql_reference",
	}

	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		return fmt.Errorf("failed to marshal edge properties: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO public.catalog_edge (
			tenant_datasource_id, source_node_id, target_node_id, relationship_type,
			properties, edge_type_id, created_at, updated_at, tenant_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id) DO UPDATE SET
			properties = EXCLUDED.properties,
			updated_at = EXCLUDED.updated_at
	`, s.datasourceID, sourceModelID, targetModelID, "references", propertiesJSON, edgeTypeID,
		time.Now(), time.Now(), s.tenantID)

	return err
}

// upsertCatalogNode inserts or updates a catalog node
func (s *CatalogService) upsertCatalogNode(
	nodeTypeID, nodeName, description string,
	properties map[string]interface{},
	qualifiedPath string,
	parentID *string,
) (string, error) {
	// Convert properties to JSON
	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		return "", fmt.Errorf("failed to marshal properties: %w", err)
	}

	var nodeID string
	var existingID *string

	// Check if node exists
	err = s.db.QueryRow(`
		SELECT id FROM public.catalog_node
		WHERE tenant_datasource_id = $1 AND node_type_id = $2 AND qualified_path = $3
	`, s.datasourceID, nodeTypeID, qualifiedPath).Scan(&existingID)

	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to check existing node: %w", err)
	}

	now := time.Now()

	if existingID != nil {
		// Update existing node
		_, err = s.db.Exec(`
			UPDATE public.catalog_node
			SET node_name = $1, description = $2, properties = $3, updated_at = $4
			WHERE id = $5
		`, nodeName, description, propertiesJSON, now, *existingID)
		if err != nil {
			return "", fmt.Errorf("failed to update catalog node: %w", err)
		}
		nodeID = *existingID
	} else {
		// Insert new node
		err = s.db.QueryRow(`
			INSERT INTO public.catalog_node (
				tenant_datasource_id, node_type_id, node_name, description,
				properties, qualified_path, parent_id, created_at, updated_at, tenant_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id
		`, s.datasourceID, nodeTypeID, nodeName, description, propertiesJSON,
			qualifiedPath, parentID, now, now, s.tenantID).Scan(&nodeID)
		if err != nil {
			return "", fmt.Errorf("failed to insert catalog node: %w", err)
		}
	}

	return nodeID, nil
}

// getNodeTypeID gets the ID of a node type by name
func (s *CatalogService) getNodeTypeID(typeName string) (string, error) {
	var typeID string
	err := s.db.QueryRow(`
		SELECT id FROM public.catalog_node_type
		WHERE catalog_type_name = $1 AND tenant_id = $2
		LIMIT 1
	`, typeName, s.tenantID).Scan(&typeID)

	if err != nil {
		return "", fmt.Errorf("failed to get node type ID for %s: %w", typeName, err)
	}

	return typeID, nil
}

// InitializeCatalogTypes creates the basic node and edge types needed for catalog management
func (s *CatalogService) InitializeCatalogTypes() error {
	// Create basic node types
	nodeTypes := []struct {
		name        string
		description string
		config      map[string]interface{}
	}{
		{"schema", "Database Schema", map[string]interface{}{"description": "Represents a database schema"}},
		{"table", "Database Table", map[string]interface{}{"description": "Represents a database table"}},
		{"column", "Database Column", map[string]interface{}{"description": "Represents a database column"}},
		{"semantic_model", "Semantic Model", map[string]interface{}{"description": "Represents a semantic model/cube in the data catalog"}},
		{"semantic_column", "Semantic Column", map[string]interface{}{"description": "Represents a measure or dimension in a semantic model"}},
		{"semantic_view", "Semantic View", map[string]interface{}{"description": "Represents a semantic view that combines multiple models"}},
		{"business_term", "Business Term", map[string]interface{}{"description": "Represents a business term or concept"}},
		{"semantic_term", "Semantic Term", map[string]interface{}{"description": "Represents a semantic term mapped to business terms"}},
		{"metrics", "Metrics", map[string]interface{}{"description": "Represents wealth management metrics and calculations"}},
	}

	for _, nt := range nodeTypes {
		if err := s.createNodeTypeIfNotExists(nt.name, nt.description, nt.config); err != nil {
			return fmt.Errorf("failed to create %s node type: %w", nt.name, err)
		}
	}

	// Create edge types
	edgeTypes := []struct {
		predicate   string
		description string
		subjectType string
		objectType  string
		properties  map[string]interface{}
	}{
		{"has_semantic", "Has Semantic", "business_term", "semantic_term", map[string]interface{}{"description": "Links business terms to semantic terms"}},
		{"member of", "Member Of", "semantic_term", "semantic_column", map[string]interface{}{"description": "Links semantic terms to semantic columns"}},
		{"mapped to", "Mapped To", "semantic_column", "column", map[string]interface{}{"description": "Links semantic columns to physical columns"}},
		{"foreign_key", "Foreign Key", "table", "table", map[string]interface{}{"description": "Links tables via foreign key relationships"}},
		{"joins", "Joins", "semantic_model", "semantic_model", map[string]interface{}{"description": "Links semantic models via joins"}},
		{"references", "References", "semantic_model", "semantic_model", map[string]interface{}{"description": "Links semantic models via references"}},
		{"extends", "Extends", "semantic_model", "semantic_model", map[string]interface{}{"description": "Links semantic models via inheritance/extension"}},
	}

	for _, et := range edgeTypes {
		if err := s.createEdgeTypeIfNotExists(et.predicate, et.description, et.subjectType, et.objectType, et.properties); err != nil {
			return fmt.Errorf("failed to create '%s' edge type: %w", et.predicate, err)
		}
	}

	return nil
}

// createNodeTypeIfNotExists creates a node type if it doesn't already exist
func (s *CatalogService) createNodeTypeIfNotExists(typeName, description string, config map[string]interface{}) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO public.catalog_node_type (
			tenant_dataource_id, catalog_type_name, description, config, tenant_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (catalog_type_name, tenant_id) DO NOTHING
	`, s.datasourceID, typeName, description, configJSON, s.tenantID, time.Now(), time.Now())

	if err != nil {
		return fmt.Errorf("failed to create node type %s: %w", typeName, err)
	}

	return nil
}

// createEdgeTypeIfNotExists creates an edge type if it doesn't already exist
func (s *CatalogService) createEdgeTypeIfNotExists(predicate, description, subjectType, objectType string, properties map[string]interface{}) error {
	// Get subject node type ID
	subjectID, err := s.getNodeTypeID(subjectType)
	if err != nil {
		// If subject type doesn't exist, skip creating edge type
		log.Printf("Warning: Subject node type %s not found, skipping edge type %s", subjectType, predicate)
		return nil
	}

	// Get object node type ID
	objectID, err := s.getNodeTypeID(objectType)
	if err != nil {
		// If object type doesn't exist, skip creating edge type
		log.Printf("Warning: Object node type %s not found, skipping edge type %s", objectType, predicate)
		return nil
	}

	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		return fmt.Errorf("failed to marshal properties: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO public.catalog_edge_types (
			edge_type_name, description, source_node_type_id, target_node_type_id,
			config, tenant_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (edge_type_name, tenant_id) DO NOTHING
	`, predicate, description, subjectID, objectID, propertiesJSON, s.tenantID, time.Now(), time.Now())

	if err != nil {
		return fmt.Errorf("failed to create edge type %s: %w", predicate, err)
	}

	return nil
}

// InsertSampleData inserts sample catalog data for testing
func (s *CatalogService) InsertSampleData() error {
	// Insert sample node types
	sampleNodeTypes := []struct {
		id          string
		typeName    string
		description string
		isActive    bool
		parentType  *string
	}{
		{"c53f9e99-8d02-4dfb-bc1b-914747d35edb", "semantic_model", "Semantic Model", true, nil},
		{"1439f761-606a-44cb-b4f8-7aa6b27a9bf5", "semantic_column", "Semantic Column", true, nil},
		{"68d6d495-0992-4d92-ad2f-7f66dc1e7d78", "schema", "Schema", true, nil},
		{"49a50271-ae58-4d3e-ae1c-2f5b89d89192", "table", "Table", true, nil},
		{"a64c1011-16e8-4ddf-b447-363bf8e15c9a", "column", "Column", true, nil},
		{"21645d21-de5f-4feb-af99-99273ea75626", "business_term", "Business Term", true, nil},
		{"820b942a-9c9e-4abc-acdc-84616db33098", "semantic_term", "Semantic Term", true, nil},
		{"d4e5f6g7-8901-2345-6789-012345678901", "semantic_view", "Semantic View", true, nil},
		{"e5f6g7h8-9012-3456-7890-123456789012", "metrics", "Wealth Management Metrics", true, nil},
	}

	for _, nt := range sampleNodeTypes {
		_, err := s.db.Exec(`
			INSERT INTO public.catalog_node_type (
				id, tenant_dataource_id, catalog_type_name, description, is_active,
				parent_type_id, tenant_id, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO NOTHING
		`, nt.id, s.datasourceID, nt.typeName, nt.description, nt.isActive,
			nt.parentType, s.tenantID, time.Now(), time.Now())
		if err != nil {
			log.Printf("Warning: Failed to insert node type %s: %v", nt.typeName, err)
		}
	}

	// Insert sample edge types
	sampleEdgeTypes := []struct {
		id          string
		predicate   string
		description string
		subjectID   string
		objectID    string
	}{
		{"f21b4a8f-05af-43b9-92cd-061265ed54e0", "foreign_key", "Foreign Key", "49a50271-ae58-4d3e-ae1c-2f5b89d89192", "49a50271-ae58-4d3e-ae1c-2f5b89d89192"},
		{"3be9d6ae-1598-4628-a3dd-b606921a9193", "has_semantic", "Has Semantic", "21645d21-de5f-4feb-af99-99273ea75626", "820b942a-9c9e-4abc-acdc-84616db33098"},
		{"97d82101-2b84-47a6-9ec0-f930fe389c3c", "mapped to", "Mapped To", "1439f761-606a-44cb-b4f8-7aa6b27a9bf5", "a64c1011-16e8-4ddf-b447-363bf8e15c9a"},
		{"99c86836-98ef-45a3-82df-4c62b5730ac6", "member of", "Member of", "820b942a-9c9e-4abc-acdc-84616db33098", "1439f761-606a-44cb-b4f8-7aa6b27a9bf5"},
		{"a1b2c3d4-5678-9012-3456-789012345678", "joins", "Joins", "c53f9e99-8d02-4dfb-bc1b-914747d35edb", "c53f9e99-8d02-4dfb-bc1b-914747d35edb"},
		{"b2c3d4e5-6789-0123-4567-890123456789", "references", "References", "c53f9e99-8d02-4dfb-bc1b-914747d35edb", "c53f9e99-8d02-4dfb-bc1b-914747d35edb"},
		{"c3d4e5f6-7890-1234-5678-901234567890", "extends", "Extends", "c53f9e99-8d02-4dfb-bc1b-914747d35edb", "c53f9e99-8d02-4dfb-bc1b-914747d35edb"},
	}

	for _, et := range sampleEdgeTypes {
		_, err := s.db.Exec(`
			INSERT INTO public.catalog_edge_types (
				id, edge_type_name, description, source_node_type_id, target_node_type_id,
				tenant_id, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO NOTHING
		`, et.id, et.predicate, et.description, et.subjectID, et.objectID,
			s.tenantID, time.Now(), time.Now())
		if err != nil {
			log.Printf("Warning: Failed to insert edge type %s: %v", et.predicate, err)
		}
	}

	// Insert sample wealth management metrics node
	metricsNodeID := "f6g7h8i9-0123-4567-8901-234567890123"
	metricsProperties := map[string]interface{}{
		"description": "Risk, tax, liquidity, goal-based, and behavioral KPIs for portfolio analysis",
		"version":     "v1.0",
		"domain":      "wealth",
		"category":    "metrics",
		"tags":        []string{"wealth", "metrics", "portfolio", "risk"},
	}

	propertiesJSON, err := json.Marshal(metricsProperties)
	if err != nil {
		log.Printf("Warning: Failed to marshal metrics properties: %v", err)
	} else {
		_, err = s.db.Exec(`
			INSERT INTO public.catalog_node (
				id, node_type_id, tenant_dataource_id, node_name, display_name,
				properties, tenant_id, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO NOTHING
		`, metricsNodeID, "e5f6g7h8-9012-3456-7890-123456789012", s.datasourceID,
			"wealth_metrics_v1", "Wealth Management Metrics v1", propertiesJSON,
			s.tenantID, time.Now(), time.Now())
		if err != nil {
			log.Printf("Warning: Failed to insert wealth metrics node: %v", err)
		} else {
			log.Printf("Successfully inserted wealth management metrics node")
		}
	}

	return nil
}

// SearchBusinessTerms searches for business terms based on criteria
func (s *CatalogService) SearchBusinessTerms(req BusinessTermSearchRequest) ([]BusinessTermResponse, int, error) {
	query := `
		SELECT cn.name, cn.display_name, cn.properties
		FROM public.catalog_node cn
		JOIN public.catalog_node_type cnt ON cn.node_type_id = cnt.id
		WHERE cnt.name = 'business_term'
		AND cn.tenant_id = $1
	`

	args := []interface{}{s.tenantID}
	argCount := 1

	// Add search filters
	if req.Query != "" {
		argCount++
		query += fmt.Sprintf(` AND (cn.name ILIKE $%d OR cn.display_name ILIKE $%d OR cn.properties->>'description' ILIKE $%d)`, argCount, argCount, argCount)
		args = append(args, "%"+req.Query+"%")
	}

	if req.Category != "" {
		argCount++
		query += fmt.Sprintf(` AND cn.properties->>'category' = $%d`, argCount)
		args = append(args, req.Category)
	}

	if req.Status != "" {
		argCount++
		query += fmt.Sprintf(` AND cn.properties->>'status' = $%d`, argCount)
		args = append(args, req.Status)
	}

	if len(req.Tags) > 0 {
		// Check if any of the requested tags are in the business term's tags array
		for _, tag := range req.Tags {
			argCount++
			query += fmt.Sprintf(` AND cn.properties->'tags' ? $%d`, argCount)
			args = append(args, tag)
		}
	}

	// Get total count
	countQuery := strings.Replace(query, "SELECT cn.name, cn.display_name, cn.properties", "SELECT COUNT(*)", 1)
	var total int
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Add ordering and pagination
	query += ` ORDER BY cn.name LIMIT $` + strconv.Itoa(argCount+1) + ` OFFSET $` + strconv.Itoa(argCount+2)
	args = append(args, req.Limit, req.Offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search business terms: %w", err)
	}
	defer rows.Close()

	var terms []BusinessTermResponse
	for rows.Next() {
		var name, displayName string
		var propertiesJSON []byte

		if err := rows.Scan(&name, &displayName, &propertiesJSON); err != nil {
			log.Printf("Error scanning business term row: %v", err)
			continue
		}

		var properties map[string]interface{}
		if err := json.Unmarshal(propertiesJSON, &properties); err != nil {
			log.Printf("Error unmarshaling properties: %v", err)
			continue
		}

		term := BusinessTermResponse{
			ID:          getStringProperty(properties, "business_term_id", name),
			Name:        getStringProperty(properties, "name", displayName),
			Description: getStringProperty(properties, "description", ""),
			Category:    getStringProperty(properties, "category", ""),
			SubCategory: getStringProperty(properties, "sub_category", ""),
			Owner:       getStringProperty(properties, "owner", ""),
			Steward:     getStringProperty(properties, "steward", ""),
			Status:      getStringProperty(properties, "status", ""),
			Version:     getStringProperty(properties, "version", ""),
			ParentID:    getStringProperty(properties, "parent_id", ""),
		}

		// Handle tags array
		if tags, ok := properties["tags"].([]interface{}); ok {
			for _, tag := range tags {
				if tagStr, ok := tag.(string); ok {
					term.Tags = append(term.Tags, tagStr)
				}
			}
		}

		terms = append(terms, term)
	}

	return terms, total, nil
}

// ValidateBusinessTerms validates a list of business terms
func (s *CatalogService) ValidateBusinessTerms(businessTerms []BusinessTerm) (*BusinessTermValidationResponse, error) {
	response := &BusinessTermValidationResponse{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	for _, term := range businessTerms {
		// Validate required fields
		if err := s.validateBusinessTerm(term); err != nil {
			response.Valid = false
			response.Errors = append(response.Errors, fmt.Sprintf("Business term %s: %v", term.ID, err))
		}

		// Validate references
		if err := s.validateBusinessTermReferences([]BusinessTerm{term}); err != nil {
			response.Valid = false
			response.Errors = append(response.Errors, fmt.Sprintf("Business term %s: %v", term.ID, err))
		}

		// Check for warnings
		if term.Status == "" {
			response.Warnings = append(response.Warnings, fmt.Sprintf("Business term %s: status not specified, defaulting to 'draft'", term.ID))
		}

		if term.Version == "" {
			response.Warnings = append(response.Warnings, fmt.Sprintf("Business term %s: version not specified", term.ID))
		}

		if len(term.Tags) == 0 {
			response.Warnings = append(response.Warnings, fmt.Sprintf("Business term %s: no tags specified", term.ID))
		}
	}

	return response, nil
}

// Helper function to safely get string properties
func getStringProperty(properties map[string]interface{}, key, defaultValue string) string {
	if val, ok := properties[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// getEdgeTypeID gets the ID of an edge type by predicate
func (s *CatalogService) getEdgeTypeID(predicate string) (string, error) {
	var typeID string
	err := s.db.QueryRow(`
		SELECT id FROM public.catalog_edge_types
		WHERE edge_type_name = $1 AND tenant_id = $2
		LIMIT 1
	`, predicate, s.tenantID).Scan(&typeID)

	if err != nil {
		return "", fmt.Errorf("failed to get edge type ID for %s: %w", predicate, err)
	}

	return typeID, nil
}

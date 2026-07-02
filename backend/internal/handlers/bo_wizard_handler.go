package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/events"
	"github.com/jmoiron/sqlx"
	jwtmiddleware "github.com/hondyman/semlayer/libs/jwt-middleware"
)

// BOWizardHandler handles Business Object creation wizard endpoints
type BOWizardHandler struct {
	db *sqlx.DB
}

// NewBOWizardHandler creates a new wizard handler
func NewBOWizardHandler(db *sqlx.DB) *BOWizardHandler {
	return &BOWizardHandler{db: db}
}

// RegisterRoutes registers wizard API routes
func (h *BOWizardHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/bo-wizard/context/{tableId}", h.GetDrivingTableContext)
	r.Get("/api/bo-wizard/related-bos/{tableId}", h.GetRelatedBusinessObjects)
	r.Post("/api/bo-wizard/save", h.SaveBusinessObject)
	r.Get("/api/bo-wizard/pending/{boId}", h.GetPendingEdges)
}

// ============================================================================
// Request/Response Types
// ============================================================================

// WizardSemanticTerm represents a semantic term available for inclusion in a BO
type WizardSemanticTerm struct {
	TermID      string  `json:"termId" db:"term_id"`
	TermName    string  `json:"termName" db:"term_name"`
	DisplayName string  `json:"displayName" db:"display_name"`
	ColumnID    string  `json:"columnId" db:"column_id"`
	ColumnName  string  `json:"columnName" db:"column_name"`
	DataType    *string `json:"dataType,omitempty" db:"data_type"`
	Description *string `json:"description,omitempty" db:"description"`
}

// WizardRelatedTable represents a table related to the driving table via FK
type WizardRelatedTable struct {
	TableID        string               `json:"tableId" db:"related_table_id"`
	TableName      string               `json:"tableName" db:"related_table_name"`
	FKName         string               `json:"fkName" db:"fk_name"`
	ExistingBOID   *string              `json:"existingBOId,omitempty" db:"existing_bo_id"`
	ExistingBOName *string              `json:"existingBOName,omitempty" db:"existing_bo_name"`
	SemanticTerms  []WizardSemanticTerm `json:"semanticTerms,omitempty"`
	LinkType       string               `json:"linkType"` // include_terms, link_bo, create_new, ignore
}

// WizardDrivingTable represents the selected driving table
type WizardDrivingTable struct {
	ID            string `json:"id" db:"id"`
	Name          string `json:"name" db:"node_name"`
	QualifiedPath string `json:"qualifiedPath" db:"qualified_path"`
	ColumnCount   int    `json:"columnCount" db:"column_count"`
	TermCount     int    `json:"termCount" db:"term_count"`
	RelatedCount  int    `json:"relatedCount" db:"related_count"`
}

// WizardContextResponse is the response for the driving table context endpoint
type WizardContextResponse struct {
	DrivingTable  WizardDrivingTable   `json:"drivingTable"`
	SemanticTerms []WizardSemanticTerm `json:"semanticTerms"`
	RelatedTables []WizardRelatedTable `json:"relatedTables"`
}

// SaveWizardRequest is the request body for saving a business object via wizard
type SaveWizardRequest struct {
	BOKey                   string               `json:"bo_key"`
	Name                    string               `json:"name"`
	DisplayName             string               `json:"display_name"`
	Description             *string              `json:"description,omitempty"`
	DriverTableID           string               `json:"driver_table_id"`
	SelectedTerms           []string             `json:"selected_terms"`
	LinkedBOs               []LinkedBORequest    `json:"linked_bos"`
	IncludedTermsFromTables []IncludedTableTerms `json:"included_terms_from_tables"`
}

// LinkedBORequest represents a request to link to another BO
type LinkedBORequest struct {
	BOID             string `json:"bo_id"`
	RelationshipType string `json:"relationship_type"`
}

// IncludedTableTerms represents terms to include from a related table
type IncludedTableTerms struct {
	TableID string   `json:"table_id"`
	TermIDs []string `json:"term_ids"`
}

// ============================================================================
// Handlers
// ============================================================================

// GetDrivingTableContext returns semantic terms and related tables for a driving table
func (h *BOWizardHandler) GetDrivingTableContext(w http.ResponseWriter, r *http.Request) {
	tableID := chi.URLParam(r, "tableId")
	if tableID == "" {
		http.Error(w, "tableId is required", http.StatusBadRequest)
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	if tenantID == "" || datasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers required", http.StatusBadRequest)
		return
	}

	fmt.Printf("[BO_WIZARD] GetDrivingTableContext called: tableID=%s, tenantID=%s, datasourceID=%s\n", tableID, tenantID, datasourceID)

	ctx := r.Context()

	// 1. Get driving table info
	drivingTable, err := h.getDrivingTableInfo(ctx, tableID, tenantID, datasourceID)
	if err != nil {
		http.Error(w, "Failed to get driving table: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Optional: boId to filter out existing fields
	boID := r.URL.Query().Get("boId")

	// 2. Get semantic terms mapped to columns of this table
	terms, err := h.getSemanticTermsForTable(ctx, tableID, tenantID, datasourceID, boID)
	if err != nil {
		http.Error(w, "Failed to get semantic terms: "+err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("[BO_WIZARD] Got %d semantic terms for table %s\n", len(terms), tableID)

	// 3. Get related tables via FK relationships
	relatedTables, err := h.getRelatedTables(ctx, tableID, tenantID, datasourceID)
	if err != nil {
		http.Error(w, "Failed to get related tables: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. For each related table, get its semantic terms
	for i := range relatedTables {
		relTerms, err := h.getSemanticTermsForTable(ctx, relatedTables[i].TableID, tenantID, datasourceID, "")
		if err == nil {
			relatedTables[i].SemanticTerms = relTerms
		}
		// Default link type
		if relatedTables[i].ExistingBOID != nil {
			relatedTables[i].LinkType = "link_bo"
		} else {
			relatedTables[i].LinkType = "include_terms"
		}
	}

	// Update counts
	drivingTable.TermCount = len(terms)
	drivingTable.RelatedCount = len(relatedTables)

	response := WizardContextResponse{
		DrivingTable:  *drivingTable,
		SemanticTerms: terms,
		RelatedTables: relatedTables,
	}

	fmt.Printf("[BO_WIZARD] Sending response: %d terms, %d related tables\n", len(terms), len(relatedTables))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetRelatedBusinessObjects returns existing BOs that use related tables as driving tables
func (h *BOWizardHandler) GetRelatedBusinessObjects(w http.ResponseWriter, r *http.Request) {
	tableID := chi.URLParam(r, "tableId")
	if tableID == "" {
		http.Error(w, "tableId is required", http.StatusBadRequest)
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	ctx := r.Context()

	query := `
		SELECT DISTINCT
			bo.id as bo_id,
			bo.node_name as bo_name,
			bo.properties->>'display_name' as display_name,
			rt.id as related_table_id,
			rt.node_name as related_table_name
		FROM catalog_edge fk
		INNER JOIN catalog_node rt ON rt.id = fk.target_node_id
		INNER JOIN catalog_node bo ON bo.properties->>'driver_table_id' = rt.id::text
		WHERE fk.source_node_id = $1
		  AND fk.edge_type_name = 'FOREIGN_KEY'
		  AND (bo.tenant_id = $2 OR EXISTS (SELECT 1 FROM tenants WHERE id = bo.tenant_id AND gold_copy = true))
		  AND bo.tenant_datasource_id = $3
		ORDER BY bo.node_name
	`

	type RelatedBO struct {
		BOID             string  `json:"boId" db:"bo_id"`
		BOName           string  `json:"boName" db:"bo_name"`
		DisplayName      *string `json:"displayName" db:"display_name"`
		RelatedTableID   string  `json:"relatedTableId" db:"related_table_id"`
		RelatedTableName string  `json:"relatedTableName" db:"related_table_name"`
	}

	var results []RelatedBO
	err := h.db.SelectContext(ctx, &results, query, tableID, tenantID, datasourceID)
	if err != nil {
		http.Error(w, "Failed to query related BOs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// SaveBusinessObject creates a new BO and queues edge creation
func (h *BOWizardHandler) SaveBusinessObject(w http.ResponseWriter, r *http.Request) {
	var req SaveWizardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	if tenantID == "" || datasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers required", http.StatusBadRequest)
		return
	}

	// Validate datasourceID is a proper UUID (or empty)
	if datasourceID != "" {
		if _, err := uuid.Parse(datasourceID); err != nil {
			http.Error(w, "Invalid X-Tenant-Datasource-ID header: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	ctx := r.Context()

	// Validate the request
	validationResult := ValidateBusinessObject(ctx, h.db, req, tenantID, datasourceID)
	if !validationResult.Valid {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(validationResult)
		return
	}

	// 1. Reserve BO identifier (used across write model and catalog worker)
	boID := uuid.New().String()

	// Persist the canonical Business Object definition into business_objects via direct SQL
	configPayload, _ := json.Marshal(map[string]interface{}{
		"driver_table_id":            req.DriverTableID,
		"selected_terms":             req.SelectedTerms,
		"linked_bos":                 req.LinkedBOs,
		"included_terms_from_tables": req.IncludedTermsFromTables,
	})

	var datasourceIDArg interface{} = datasourceID
	if datasourceID == "" {
		datasourceIDArg = nil
	}

	fmt.Printf("[BO_WIZARD] Inserting BO via SQL: %s\n", boID)

	_, err := h.db.ExecContext(ctx, `
		INSERT INTO business_objects (
			id, tenant_id, tenant_datasource_id, key, name, display_name,
			technical_name, description, config, driver_table_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			NOW(), NOW()
		)
	`, boID, tenantID, datasourceIDArg, req.BOKey, req.Name, req.DisplayName,
		req.BOKey, req.Description, string(configPayload), req.DriverTableID)

	if err != nil {
		fmt.Printf("[BO_WIZARD] Failed to create business object: %v\n", err)
		http.Error(w, "Failed to create business object: "+err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("[BO_WIZARD] Successfully created BO %s\n", boID)

	// 2. Populate bo_fields with selected semantic terms via Hasura Mutation
	if len(req.SelectedTerms) > 0 {
		var fieldsToInsert []map[string]interface{}

		for i, termID := range req.SelectedTerms {
			var termName string
			// We still use SQL to fetch term name from catalog_node (read-only op)
			err := h.db.GetContext(ctx, &termName, `
				SELECT node_name FROM catalog_node WHERE id = $1::uuid LIMIT 1
			`, termID)
			if err != nil {
				fmt.Printf("[BO_WIZARD] Warning: Failed to fetch term name for %s: %v\n", termID, err)
				termName = termID
			}

			termDisplayName := termName

			fieldsToInsert = append(fieldsToInsert, map[string]interface{}{
				"id":                 uuid.New().String(),
				"tenant_id":          tenantID,
				"business_object_id": boID,
				"key":                termID, // Using termID as key for now
				"name":               termName,
				"field_name":         termName,
				"display_label":      termDisplayName,
				"technical_name":     termID,
				"field_type":         "semantic_term",
				"is_core":            false,
				"display_order":      i,
				"semantic_term_id":   termID,
			})
		}

		if len(fieldsToInsert) > 0 {
			fmt.Printf("[BO_WIZARD] Inserting %d bo_fields via SQL\n", len(fieldsToInsert))
			for _, field := range fieldsToInsert {
				_, err = h.db.ExecContext(ctx, `
					INSERT INTO bo_fields (
						id, tenant_id, business_object_id, key, name, field_name,
						display_label, technical_name, field_type, is_core,
						display_order, semantic_term_id, created_at, updated_at
					) VALUES (
						$1, $2, $3, $4, $5, $6,
						$7, $8, $9, $10,
						$11, $12, NOW(), NOW()
					) ON CONFLICT DO NOTHING
				`,
					field["id"], field["tenant_id"], field["business_object_id"],
					field["key"], field["name"], field["field_name"],
					field["display_label"], field["technical_name"], field["field_type"],
					field["is_core"], field["display_order"], field["semantic_term_id"],
				)
				if err != nil {
					fmt.Printf("[BO_WIZARD] Warning: Failed to insert bo_field %v: %v\n", field["id"], err)
				}
			}
		}
	}

	// Publish event via SQL (using a new transaction since PublishEvent requires *sqlx.Tx)
	// We do this despite having already committed the data to Hasura.
	catalogEvent := map[string]interface{}{
		"bo_id":                      boID,
		"bo_key":                     req.BOKey,
		"name":                       req.Name,
		"display_name":               req.DisplayName,
		"description":                req.Description,
		"driver_table_id":            req.DriverTableID,
		"selected_terms":             req.SelectedTerms,
		"linked_bos":                 req.LinkedBOs,
		"included_terms_from_tables": req.IncludedTermsFromTables,
		"tenant_id":                  tenantID,
		"datasource_id":              datasourceID,
	}

	tx, err := h.db.BeginTxx(ctx, nil)
	if err != nil {
		fmt.Printf("[BO_WIZARD] Warning: Failed to start transaction for event publishing: %v\n", err)
	} else {
		err = events.PublishEvent(ctx, tx, "BusinessObject.CatalogSync", catalogEvent)
		if err != nil {
			fmt.Printf("[BO_WIZARD] Warning: Failed to queue catalog sync event: %v\n", err)
			_ = tx.Rollback()
		} else {
			if err := tx.Commit(); err != nil {
				fmt.Printf("[BO_WIZARD] Warning: Failed to commit event transaction: %v\n", err)
			}
		}
	}

	// Return created BO
	response := map[string]interface{}{
		"id":              boID,
		"bo_key":          req.BOKey,
		"name":            req.Name,
		"display_name":    req.DisplayName,
		"driver_table_id": req.DriverTableID,
		"status":          "draft",
		"queued_edges":    0,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// syncBusinessObjectToCatalog creates/updates catalog node and edges for a BO
func (h *BOWizardHandler) syncBusinessObjectToCatalog(
	ctx context.Context,
	tx *sqlx.Tx,
	boID, boKey, boName, boDisplayName string,
	driverTableID string,
	selectedTermIDs []string,
	tenantID, datasourceID string,
) error {
	// 1. Get business_object node type ID
	var boNodeTypeID string
	err := tx.GetContext(ctx, &boNodeTypeID, `
		SELECT id FROM catalog_node_type 
		WHERE catalog_type_name = 'business_object'
	`)
	if err != nil {
		return fmt.Errorf("failed to get BO node type: %w", err)
	}

	// 2. Upsert catalog_node for the business object
	_, err = tx.ExecContext(ctx, `
		INSERT INTO catalog_node (
			id, node_name, node_type_id, tenant_id, tenant_datasource_id,
			properties, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			jsonb_build_object(
				'bo_key', $6,
				'display_name', $7,
				'driver_table_id', $8
			),
			NOW(), NOW()
		)
		ON CONFLICT (id) DO UPDATE SET
			node_name = EXCLUDED.node_name,
			properties = EXCLUDED.properties,
			updated_at = NOW()
	`, boID, boKey, boNodeTypeID, tenantID, datasourceID, boKey, boDisplayName, driverTableID)
	if err != nil {
		return fmt.Errorf("failed to upsert BO catalog node: %w", err)
	}

	// 3. Delete existing semantic term edges for this BO
	_, err = tx.ExecContext(ctx, `
		DELETE FROM catalog_edge
		WHERE target_node_id = $1
		  AND edge_type_name IN ('member_of', 'USES_TERM')
	`, boID)
	if err != nil {
		return fmt.Errorf("failed to delete old edges: %w", err)
	}

	// 4. Create new edges to semantic terms
	for _, termID := range selectedTermIDs {
		edgeID := uuid.New().String()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO catalog_edge (
				id, source_node_id, target_node_id, edge_type_name,
				tenant_id, tenant_datasource_id,
				created_at, updated_at
			) VALUES (
				$1, $3, $2, 'member_of',
				$4, $5, NOW(), NOW()
			)
		`, edgeID, boID, termID, tenantID, datasourceID)
		if err != nil {
			return fmt.Errorf("failed to create edge to term %s: %w", termID, err)
		}
	}

	return nil
}

// ============================================================================
// Helper Methods
// ============================================================================

func (h *BOWizardHandler) getDrivingTableInfo(ctx context.Context, tableID, tenantID, datasourceID string) (*WizardDrivingTable, error) {
	query := `
		SELECT 
			cn.id,
			cn.node_name,
			COALESCE(cn.qualified_path, cn.node_name) as qualified_path,
			(SELECT COUNT(*) FROM catalog_node col 
			 WHERE col.parent_id = cn.id) as column_count
		FROM catalog_node cn
		WHERE cn.id = $1::uuid
		  AND (
		    (cn.tenant_id = $2::uuid AND cn.tenant_datasource_id = $3::uuid)
		    OR 
		    (EXISTS (SELECT 1 FROM tenants WHERE id = cn.tenant_id AND gold_copy = true))
		  )
	`

	var result WizardDrivingTable
	err := h.db.GetContext(ctx, &result, query, tableID, tenantID, datasourceID)
	return &result, err
}

func (h *BOWizardHandler) getSemanticTermsForTable(ctx context.Context, tableID, tenantID, datasourceID, boID string) ([]WizardSemanticTerm, error) {
	query := `
		SELECT DISTINCT 
			st.id as term_id,
			st.node_name as term_name,
			COALESCE(st.properties->>'title', st.node_name) as display_name,
			col.id as column_id,
			col.node_name as column_name,
			col.properties->>'data_type' as data_type,
			st.description
		FROM catalog_node col
		INNER JOIN catalog_edge ce ON ce.source_node_id = col.id 
			AND (
				ce.edge_type_id IN ('1991f82b-1268-4614-9146-11cb63fe0dc9', '0434ca1a-6543-42d3-9fce-f0b58b5fba34')
				OR
				ce.edge_type_name IN ('maps_to', 'has_context', 'MAPS_TO', 'HAS_CONTEXT')
			)
		INNER JOIN catalog_node st ON st.id = ce.target_node_id
		INNER JOIN catalog_node_type cnt ON st.node_type_id = cnt.id 
			AND cnt.catalog_type_name IN ('semantic_term', 'business_term')
		WHERE col.parent_id = $1::uuid
		  AND (
		    (col.tenant_id = $2::uuid AND col.tenant_datasource_id = $3::uuid)
		    OR 
		    (EXISTS (SELECT 1 FROM tenants WHERE id = col.tenant_id AND gold_copy = true))
		  )
	`

	args := []interface{}{tableID, tenantID, datasourceID}

	// Filter out already assigned terms if boID is provided
	if boID != "" {
		query += `
		  AND NOT EXISTS (
		    SELECT 1 FROM bo_fields bf
		    WHERE bf.business_object_id = $4::uuid
		      AND bf.semantic_term_id = st.id
		  )`
		args = append(args, boID)
	}

	query += ` ORDER BY st.node_name`

	var results []WizardSemanticTerm
	err := h.db.SelectContext(ctx, &results, query, args...)
	if err != nil {
		fmt.Printf("[BO_WIZARD] ERROR in getSemanticTermsForTable: tableID=%s, tenantID=%s, datasourceID=%s, error=%v\n", tableID, tenantID, datasourceID, err)
		return []WizardSemanticTerm{}, err
	}
	fmt.Printf("[BO_WIZARD] getSemanticTermsForTable: tableID=%s, found %d terms\n", tableID, len(results))
	return results, nil
}

func (h *BOWizardHandler) getRelatedTables(ctx context.Context, tableID, tenantID, datasourceID string) ([]WizardRelatedTable, error) {
	query := `
		SELECT DISTINCT
			rt.id as related_table_id,
			rt.node_name as related_table_name,
			COALESCE(fk.properties->>'constraint_name', 'FK') as fk_name,
			bo.id as existing_bo_id,
			bo.node_name as existing_bo_name
		FROM catalog_edge fk
		INNER JOIN catalog_node rt ON rt.id = fk.target_node_id
		LEFT JOIN catalog_node bo ON bo.properties->>'driver_table_id' = rt.id::text
			AND (
				(bo.tenant_id = $2::uuid AND bo.tenant_datasource_id = $3::uuid)
				OR 
				(EXISTS (SELECT 1 FROM tenants WHERE id = bo.tenant_id AND gold_copy = true))
			)
		WHERE fk.source_node_id = $1::uuid
		  AND fk.edge_type_name = 'FOREIGN_KEY'
		ORDER BY rt.node_name
	`

	var results []WizardRelatedTable
	err := h.db.SelectContext(ctx, &results, query, tableID, tenantID, datasourceID)
	if err != nil {
		return []WizardRelatedTable{}, err
	}
	return results, nil
}

func (h *BOWizardHandler) queueEdgeCreation(ctx context.Context, tx *sqlx.Tx, tenantID, datasourceID, sourceID, targetID, edgeType string, properties []byte) error {
	if properties == nil {
		properties = []byte("{}")
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO edge_creation_queue (id, tenant_id, datasource_id, source_node_id, target_node_id, edge_type, properties, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8)
	`, uuid.New().String(), tenantID, datasourceID, sourceID, targetID, edgeType, properties, time.Now())

	return err
}

// ProcessEdgeQueue processes pending edges from the queue (called by worker)
func (h *BOWizardHandler) ProcessEdgeQueue(ctx context.Context) (int, error) {
	// Get pending items
	query := `
		SELECT id, tenant_id, datasource_id, source_node_id, target_node_id, edge_type, properties
		FROM edge_creation_queue
		WHERE status = 'pending'
		ORDER BY created_at
		LIMIT 100
	`

	type QueueItem struct {
		ID           string          `db:"id"`
		TenantID     string          `db:"tenant_id"`
		DatasourceID string          `db:"datasource_id"`
		SourceID     string          `db:"source_node_id"`
		TargetID     string          `db:"target_node_id"`
		EdgeType     string          `db:"edge_type"`
		Properties   json.RawMessage `db:"properties"`
	}

	var items []QueueItem
	if err := h.db.SelectContext(ctx, &items, query); err != nil {
		return 0, err
	}

	processed := 0
	for _, item := range items {
		// Mark as processing
		_, _ = h.db.ExecContext(ctx, `UPDATE edge_creation_queue SET status = 'processing' WHERE id = $1`, item.ID)

		// Create the edge
		_, err := h.db.ExecContext(ctx, `
			INSERT INTO catalog_edge (id, tenant_id, tenant_datasource_id, source_node_id, target_node_id, edge_type_name, properties, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
			ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_name, target_node_id) DO UPDATE SET
				properties = EXCLUDED.properties,
				updated_at = EXCLUDED.updated_at
		`, uuid.New().String(), item.TenantID, item.DatasourceID, item.SourceID, item.TargetID, item.EdgeType, item.Properties, time.Now())

		if err != nil {
			// Mark as failed
			_, _ = h.db.ExecContext(ctx, `UPDATE edge_creation_queue SET status = 'failed', error_message = $2, attempts = attempts + 1 WHERE id = $1`, item.ID, err.Error())
		} else {
			// Mark as completed
			_, _ = h.db.ExecContext(ctx, `UPDATE edge_creation_queue SET status = 'completed', processed_at = $2 WHERE id = $1`, item.ID, time.Now())
			processed++
		}
	}

	return processed, nil
}

// GetPendingEdges returns the pending edge count for a BO (to show processing state in UI)
func (h *BOWizardHandler) GetPendingEdges(w http.ResponseWriter, r *http.Request) {
	boID := chi.URLParam(r, "boId")
	if boID == "" {
		http.Error(w, "boId is required", http.StatusBadRequest)
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	ctx := r.Context()

	// Count pending, processing, completed, and failed edges for this BO
	type EdgeStats struct {
		Pending    int `db:"pending" json:"pending"`
		Processing int `db:"processing" json:"processing"`
		Completed  int `db:"completed" json:"completed"`
		Failed     int `db:"failed" json:"failed"`
		DLQ        int `db:"dlq" json:"dlq"`
	}

	var stats EdgeStats
	err := h.db.GetContext(ctx, &stats, `
		SELECT 
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COUNT(*) FILTER (WHERE status = 'processing') as processing,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COUNT(*) FILTER (WHERE status = 'failed') as failed,
			COUNT(*) FILTER (WHERE status = 'dlq') as dlq
		FROM edge_creation_queue
		WHERE source_node_id = $1
		  AND tenant_id = $2
		  AND datasource_id = $3
	`, boID, tenantID, datasourceID)

	if err != nil {
		// Return empty stats if query fails
		stats = EdgeStats{}
	}

	response := map[string]interface{}{
		"bo_id":      boID,
		"stats":      stats,
		"processing": stats.Pending > 0 || stats.Processing > 0,
		"has_errors": stats.Failed > 0 || stats.DLQ > 0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

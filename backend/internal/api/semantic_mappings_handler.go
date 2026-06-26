package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

type SemanticMappingsHandler struct {
	service         *analytics.SemanticMappingService
	semanticService *analytics.SemanticService
	db              *sqlx.DB
}

func NewSemanticMappingsHandler(service *analytics.SemanticMappingService, semanticService *analytics.SemanticService, db *sqlx.DB) *SemanticMappingsHandler {
	return &SemanticMappingsHandler{
		service:         service,
		semanticService: semanticService,
		db:              db,
	}
}

func (h *SemanticMappingsHandler) RegisterRoutes(r chi.Router) {
	r.Post("/semantic-mappings/edges", h.CreateMappingEdges)
	r.Post("/semantic-mappings/replace", h.ReplaceMapping)
	r.Post("/semantic-mappings/ignore", h.IgnoreMapping)
	r.Post("/semantic-mappings/apply-custom", h.ApplyCustomMapping)
	r.Post("/semantic-mappings/business-term-edges", h.CreateBusinessTermEdge)
}

func (h *SemanticMappingsHandler) CreateMappingEdges(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var body map[string][]map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload"})
		return
	}

	mappings := body["mappings"]
	createdEdges := 0
	createdTerms := 0
	skippedExisting := 0
	var createdEdgeColIDs []string
	var createdTermIDs []string
	var perMappingResults []map[string]interface{}
	ctx := r.Context()

	// Build a map of datasources to check for datamart→alpha_dwh mappings
	datasourceResolutions := make(map[string]string)

	for _, m := range mappings {
		// Each mapping should contain database_column and semantic info
		dbCol, _ := m["database_column"].(map[string]interface{})
		colNodeID, _ := dbCol["node_id"].(string)
		tenantID, _ := dbCol["tenant_id"].(string)
		tenantDatasourceID, _ := dbCol["tenant_datasource_id"].(string)

		// Resolve datamart to alpha_dwh if needed
		resolvedDatasourceID := tenantDatasourceID
		if resolved, exists := datasourceResolutions[tenantDatasourceID]; exists {
			resolvedDatasourceID = resolved
		} else {
			// Check if this datasource is "datamart" and map it to "alpha_dwh"
			var datasourceName string
			// Assuming h.service.DB() returns *sqlx.DB or similar that has GetContext
			// If not, we might need to use h.db
			err := h.db.GetContext(ctx, &datasourceName, "SELECT datasource_name FROM alpha_datasource WHERE id = $1", tenantDatasourceID)
			if err == nil && strings.EqualFold(datasourceName, "datamart") {
				var alphaDwhID string
				err = h.db.GetContext(ctx, &alphaDwhID, "SELECT id FROM alpha_datasource WHERE datasource_name = 'alpha_dwh'")
				if err == nil && alphaDwhID != "" {
					logging.GetLogger().Sugar().Infof("[edges] Resolved 'datamart' datasource to 'alpha_dwh' ID: %s", alphaDwhID)
					resolvedDatasourceID = alphaDwhID
					datasourceResolutions[tenantDatasourceID] = alphaDwhID
				}
			}
		}

		// Determine semantic term id or create new term if needed
		semanticTermID, _ := m["semantic_term_id"].(string)
		isNew, _ := m["is_new_term"].(bool)
		// Track whether we created a new term for this mapping
		var createdTermIDForThisMapping string
		if isNew || semanticTermID == "" {
			// Create semantic term via mapping service
			termName, _ := m["semantic_term"].(string)
			dataType, _ := dbCol["data_type"].(string)
			columnName, _ := dbCol["column"].(string)
			tableName, _ := dbCol["table"].(string)
			newID, err := h.service.CreateSemanticTerm(ctx, tenantID, tenantDatasourceID, termName, dataType, columnName, tableName)
			if err == nil {
				semanticTermID = newID
				createdTerms++
				createdTermIDForThisMapping = newID
			} else {
				logging.GetLogger().Sugar().Warnf("Failed to create semantic term: %v", err)
			}
		}

		if semanticTermID != "" {
			// Check if the column is a primary key to store as edge property
			var colProperties []byte
			var isPrimaryKey bool
			colPropsQuery := `SELECT COALESCE(properties, '{}'::jsonb) FROM catalog_node WHERE id = $1`
			if err := h.db.QueryRowContext(ctx, colPropsQuery, colNodeID).Scan(&colProperties); err == nil {
				var props map[string]interface{}
				if json.Unmarshal(colProperties, &props) == nil {
					if pk, ok := props["is_primary_key"].(bool); ok && pk {
						isPrimaryKey = true
					}
				}
			}

			// Build edge properties
			var edgeProps map[string]interface{}
			if isPrimaryKey {
				edgeProps = map[string]interface{}{
					"is_primary_key": true,
				}
			}

			// Create mapping edge using the resolved datasource ID with properties
			created, err := h.service.CreateMappingEdgeWithProperties(ctx, tenantID, resolvedDatasourceID, semanticTermID, colNodeID, edgeProps)
			if err != nil {
				logging.GetLogger().Sugar().Warnf("Failed to create mapping edge: %v", err)
				// still report per-mapping result as failed
				perMappingResults = append(perMappingResults, map[string]interface{}{
					"col_node_id":      colNodeID,
					"semantic_term_id": semanticTermID,
					"created_edge":     false,
					"created_term_id":  createdTermIDForThisMapping,
					"error":            err.Error(),
				})
				continue
			}
			if created {
				createdEdges++
				createdEdgeColIDs = append(createdEdgeColIDs, colNodeID)
				if createdTermIDForThisMapping != "" {
					createdTermIDs = append(createdTermIDs, createdTermIDForThisMapping)
				}
				perMappingResults = append(perMappingResults, map[string]interface{}{
					"col_node_id":      colNodeID,
					"semantic_term_id": semanticTermID,
					"created_edge":     true,
					"created_term_id":  createdTermIDForThisMapping,
					"skipped":          false,
				})
			} else {
				skippedExisting++
				perMappingResults = append(perMappingResults, map[string]interface{}{
					"col_node_id":      colNodeID,
					"semantic_term_id": semanticTermID,
					"created_edge":     false,
					"created_term_id":  createdTermIDForThisMapping,
					"skipped":          true,
				})
			}
		} else {
			// If service not available or semanticTermID missing, log and skip
			logging.GetLogger().Sugar().Warn("Semantic service unavailable or missing semantic term id; skipping mapping")
			perMappingResults = append(perMappingResults, map[string]interface{}{
				"col_node_id":      colNodeID,
				"semantic_term_id": semanticTermID,
				"created_edge":     false,
				"created_term_id":  nil,
				"skipped":          true,
			})
		}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"created_edges":        createdEdges,
		"created_terms":        createdTerms,
		"skipped_existing":     skippedExisting,
		"created_edge_col_ids": createdEdgeColIDs,
		"created_term_ids":     createdTermIDs,
		"per_mapping_results":  perMappingResults,
	})
}

func (h *SemanticMappingsHandler) ReplaceMapping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	var body map[string]map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload"})
		return
	}

	m := body["mapping"]
	if m == nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "mapping required"})
		return
	}

	dbCol, _ := m["database_column"].(map[string]interface{})
	colNodeID, _ := dbCol["node_id"].(string)
	termID, _ := m["semantic_term_id"].(string)

	ctx := r.Context()
	var totalDeleted int64
	var totalCreated int64
	var totalSkipped int64
	var deletedEdgeColIDs []string
	var createdEdgeColIDs []string
	var createdTermIDs []string

	// Delete existing edge if present
	if termID != "" {
		deleted, err := h.service.DeleteMappingEdge(ctx, tenantID, tenantDatasourceID, termID, colNodeID)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to delete existing edge: %v", err)})
			return
		}
		totalDeleted += deleted
		if deleted > 0 {
			deletedEdgeColIDs = append(deletedEdgeColIDs, colNodeID)
		}
	}

	// Create new edge using existing create flow
	var perMappingResult = map[string]interface{}{
		"col_node_id":      colNodeID,
		"semantic_term_id": termID,
		"created_edge":     false,
		"deleted_edge":     false,
		"created_term_id":  nil,
	}
	if termID != "" {
		created, err := h.service.CreateMappingEdge(ctx, tenantID, tenantDatasourceID, termID, colNodeID)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to create new edge: %v", err)})
			return
		}
		if created {
			totalCreated++
			createdEdgeColIDs = append(createdEdgeColIDs, colNodeID)
			perMappingResult["created_edge"] = true
		} else {
			totalSkipped++
		}
	}

	// Mark deleted edges in result
	if totalDeleted > 0 {
		deletedEdgeColIDs = append(deletedEdgeColIDs, colNodeID)
		perMappingResult["deleted_edge"] = true
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"created":              totalCreated,
		"deleted":              totalDeleted,
		"skipped":              totalSkipped,
		"created_edge_col_ids": createdEdgeColIDs,
		"deleted_edge_col_ids": deletedEdgeColIDs,
		"created_term_ids":     createdTermIDs,
		"per_mapping_result":   perMappingResult,
	})
}

func (h *SemanticMappingsHandler) IgnoreMapping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var body map[string][]map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload"})
		return
	}
	mappings := body["mappings"]
	ctx := r.Context()
	for _, m := range mappings {
		dbCol, _ := m["database_column"].(map[string]interface{})
		colNodeID, _ := dbCol["node_id"].(string)
		tenantID, _ := dbCol["tenant_id"].(string)
		tenantDatasourceID, _ := dbCol["tenant_datasource_id"].(string)
		termName, _ := m["semantic_term"].(string)

		if h.semanticService != nil {
			if err := h.semanticService.PersistIgnore(ctx, tenantID, tenantDatasourceID, colNodeID, termName); err != nil {
				logging.GetLogger().Sugar().Warnf("Failed to persist ignore: %v", err)
			}
		}
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *SemanticMappingsHandler) ApplyCustomMapping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	var req struct {
		ColumnNodeID     string `json:"column_node_id"`
		SemanticTermName string `json:"semantic_term_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ColumnNodeID == "" || req.SemanticTermName == "" {
		http.Error(w, "column_node_id and semantic_term_name are required", http.StatusBadRequest)
		return
	}

	err := h.service.ApplyCustomMapping(r.Context(), tenantID, tenantDatasourceID, req.ColumnNodeID, req.SemanticTermName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to apply custom mapping: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Custom mapping applied successfully",
	})
}

func (h *SemanticMappingsHandler) CreateBusinessTermEdge(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Support two payload shapes for backward compatibility:
	// 1) { semantic_term_id, business_term_id }
	// 2) { subject_node_id, object_node_id }
	var bodyMap map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&bodyMap); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload"})
		return
	}

	// Extract IDs from either shape
	getString := func(key string) string {
		if v, ok := bodyMap[key]; ok {
			if s, ok2 := v.(string); ok2 {
				return s
			}
		}
		return ""
	}

	body := struct {
		SemanticTermID string
		BusinessTermID string
	}{
		SemanticTermID: getString("semantic_term_id"),
		BusinessTermID: getString("business_term_id"),
	}

	// If legacy keys not present, try subject/object shape
	if strings.TrimSpace(body.SemanticTermID) == "" && strings.TrimSpace(body.BusinessTermID) == "" {
		// subject_node_id is business term, object_node_id is semantic term
		body.BusinessTermID = getString("subject_node_id")
		body.SemanticTermID = getString("object_node_id")
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	// Validate input IDs
	if strings.TrimSpace(body.SemanticTermID) == "" || strings.TrimSpace(body.BusinessTermID) == "" {
		http.Error(w, "semantic_term_id and business_term_id are required", http.StatusBadRequest)
		return
	}
	if _, err := uuid.Parse(body.SemanticTermID); err != nil {
		http.Error(w, "semantic_term_id must be a valid UUID", http.StatusBadRequest)
		return
	}
	if _, err := uuid.Parse(body.BusinessTermID); err != nil {
		http.Error(w, "business_term_id must be a valid UUID", http.StatusBadRequest)
		return
	}

	created, err := h.service.CreateBusinessTermEdge(r.Context(), tenantID, tenantDatasourceID, body.SemanticTermID, body.BusinessTermID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create business term edge: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"created": created,
	})
}

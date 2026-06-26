package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/cube"
	charts "github.com/hondyman/semlayer/backend/internal/db/charts"
	"github.com/hondyman/semlayer/backend/internal/handlers"
	"github.com/hondyman/semlayer/backend/internal/scanner"

	// services package not required here; we use analytics.SemanticModelService
	coremodels "github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
)

// registerFabricRoutes moves the fabric-related route handlers out of the large api.go file.
//
//lint:ignore U1000 retained as an example and for future integration
func (s *Server) registerFabricRoutes(r chi.Router) {
	// Prepare sqlx DB and semantic service
	sqlxDB := sqlx.NewDb(s.DB, "postgres")
	// Use the analytics package semantic model service which returns models.FabricDefn types
	semanticSvc := analytics.NewSemanticModelService(sqlxDB)

	// Create a semantic model handler (not currently used directly here)
	// _ = handlers.NewSemanticModelHandler(semanticSvc) // Type mismatch: expects *analytics.SemanticModelService

	// Fabric (semantic model) endpoints
	r.Get("/fabric/models", func(w http.ResponseWriter, r *http.Request) {
		datasourceIDStr := r.URL.Query().Get("datasource_id")
		if datasourceIDStr == "" {
			http.Error(w, "datasource_id query parameter is required", http.StatusBadRequest)
			return
		}
		datasourceUUID, err := uuid.Parse(datasourceIDStr)
		if err != nil {
			http.Error(w, "invalid datasource_id format", http.StatusBadRequest)
			return
		}

		var tenantID uuid.UUID
		err = s.DB.QueryRowContext(r.Context(), `
            SELECT t.id FROM public.tenants t
            JOIN public.tenant_instance ti ON t.id = ti.tenant_id
            JOIN public.tenant_product tp ON ti.id = tp.datasource_id
            JOIN public.tenant_product_datasource tpd ON tp.id = tpd.tenant_product_id
            WHERE tpd.id = $1 LIMIT 1
        `, datasourceUUID).Scan(&tenantID)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not determine tenant_id for datasource %s: %v", datasourceUUID, err), http.StatusInternalServerError)
			return
		}

		catalogScanner, err := scanner.NewFabricCatalogScanner(sqlxDB.DB, tenantID, datasourceUUID)
		if err != nil {
			http.Error(w, "failed to initialize catalog scanner", http.StatusInternalServerError)
			return
		}
		modelsList, err := catalogScanner.ExtractModels()
		if err != nil {
			http.Error(w, "failed to retrieve models", http.StatusInternalServerError)
			return
		}
		resp := map[string]interface{}{"models": modelsList, "count": len(modelsList)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	r.Get("/fabric/extensions", func(w http.ResponseWriter, r *http.Request) {
		dsStr := r.URL.Query().Get("datasource_id")
		if dsStr == "" {
			http.Error(w, "datasource_id query parameter is required", http.StatusBadRequest)
			return
		}
		dsID, err := uuid.Parse(dsStr)
		if err != nil {
			http.Error(w, "invalid datasource_id format", http.StatusBadRequest)
			return
		}
		items, err := semanticSvc.ListExtensionModels(dsID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	})

	r.Post("/fabric/extensions", func(w http.ResponseWriter, r *http.Request) {
		var req handlers.SaveExtensionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		dsStr := r.URL.Query().Get("datasource_id")
		if dsStr == "" {
			http.Error(w, "datasource_id query parameter is required", http.StatusBadRequest)
			return
		}
		dsID, err := uuid.Parse(dsStr)
		if err != nil {
			http.Error(w, "invalid datasource_id format", http.StatusBadRequest)
			return
		}
		b, err := json.Marshal(req.ModelObject)
		if err != nil {
			http.Error(w, "invalid model_object payload", http.StatusBadRequest)
			return
		}
		var ext cube.Cube
		if err := json.Unmarshal(b, &ext); err != nil {
			http.Error(w, "invalid model_object structure", http.StatusBadRequest)
			return
		}
		actor := uuid.Nil
		if req.ActorID != "" {
			if a, err := uuid.Parse(req.ActorID); err == nil {
				actor = a
			}
		}
		saved, issues, err := semanticSvc.SaveExtensionModel(dsID, analytics.SaveExtensionModelRequest{
			BaseModelKey: req.BaseModelKey,
			ModelKey:     req.ModelKey,
			ModelObject:  ext,
			ActorID:      actor,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"model": saved, "issues": issues})
	})

	r.Get("/catalog/tables", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		datasourceIDStr := r.URL.Query().Get("datasource_id")
		if datasourceIDStr == "" {
			http.Error(w, "datasource_id query parameter is required", http.StatusBadRequest)
			return
		}
		if _, err := uuid.Parse(datasourceIDStr); err != nil {
			http.Error(w, "invalid datasource_id format", http.StatusBadRequest)
			return
		}

		txx, err := sqlxDB.BeginTxx(r.Context(), nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to begin transaction: %v", err), http.StatusInternalServerError)
			return
		}
		defer func() { _ = txx.Rollback() }()

		tableQuery := `
            SELECT cnv.node_id, cnv.node_name, cnv.catalog_type_name, cnv.qualified_path, cnv.properties, cnv.parent_id, cnv.source_name, cnv.node_type_id, cn.core_id
            FROM public.catalog_node_vw cnv
            LEFT JOIN public.catalog_node cn ON cn.id = cnv.node_id
            WHERE cnv.tenant_datasource_id = $1 AND cnv.node_type_id = $2
            ORDER BY cnv.qualified_path`

		rows, err := txx.QueryContext(r.Context(), tableQuery, datasourceIDStr, charts.TABLE_NODE_TYPE_ID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to query catalog_node_vw tables: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		outNodes := []map[string]interface{}{}
		for rows.Next() {
			var nodeID, nodeName, catalogType, qualifiedPath string
			var properties []byte
			var parentID sql.NullString
			var sourceName sql.NullString
			var nodeTypeID sql.NullString
			var coreID sql.NullString

			if err := rows.Scan(&nodeID, &nodeName, &catalogType, &qualifiedPath, &properties, &parentID, &sourceName, &nodeTypeID, &coreID); err != nil {
				fmt.Printf("warning: failed to scan table row: %v\n", err)
				continue
			}

			colQuery := `
                SELECT cnv.node_id, cnv.node_name, COALESCE(cnv.properties->>'data_type','unknown') as data_type,
                       COALESCE((cnv.properties->>'is_nullable')::boolean,false) as is_nullable,
                       COALESCE(cnv.properties->>'default_value','') as default_value,
                       COALESCE(cn.core_id IS NOT NULL, false) as is_core,
                       COALESCE((cnv.properties->>'ordinal_position')::integer,999) as ordinal_position,
                       cnv.qualified_path, cnv.parent_id,
                       COALESCE((cnv.properties->>'is_primary_key')::boolean,false) as is_primary_key,
                       COALESCE((cnv.properties->>'is_foreign_key')::boolean,false) as is_foreign_key
                FROM public.catalog_node_vw cnv
                LEFT JOIN public.catalog_node cn ON cn.id = cnv.node_id
                WHERE cnv.parent_id = $1 AND cnv.tenant_datasource_id = $2 AND cnv.node_type_id = $3
                ORDER BY ordinal_position`

			enhancedCols := []map[string]interface{}{}
			colRows, err := txx.QueryContext(r.Context(), colQuery, nodeID, datasourceIDStr, charts.COLUMN_NODE_TYPE_ID)
			if err != nil {
				fmt.Printf("warning: failed to query columns for table %s: %v (table will be included without columns)\n", nodeName, err)
			} else {
				for colRows.Next() {
					var colID, colName, dataType, colQualifiedPath, colParentID, defaultValue string
					var isNullable, isCore, isPrimaryKey, isForeignKey bool
					var ordinalPosition int
					if err := colRows.Scan(&colID, &colName, &dataType, &isNullable, &defaultValue, &isCore, &ordinalPosition, &colQualifiedPath, &colParentID, &isPrimaryKey, &isForeignKey); err != nil {
						fmt.Printf("warning: failed to scan column for table %s: %v\n", nodeName, err)
						continue
					}

					qPath := colQualifiedPath
					if qPath == "" {
						if qualifiedPath != "" && strings.Contains(qualifiedPath, "/") {
							p := strings.TrimPrefix(qualifiedPath, "/")
							parts := strings.Split(p, "/")
							if len(parts) >= 2 {
								qPath = fmt.Sprintf("%s.%s.%s", parts[0], parts[1], colName)
							}
						} else if strings.Contains(qualifiedPath, ".") {
							qPath = fmt.Sprintf("%s.%s", qualifiedPath, colName)
						} else {
							qPath = fmt.Sprintf("%s.%s", nodeName, colName)
						}
					}

					enhancedCols = append(enhancedCols, map[string]interface{}{
						"id":            colID,
						"name":          colName,
						"type":          dataType,
						"isCore":        isCore,
						"nullable":      isNullable,
						"default":       defaultValue,
						"qualifiedPath": qPath,
						"isPrimaryKey":  isPrimaryKey,
						"isForeignKey":  isForeignKey,
					})
				}
				if colRows != nil {
					colRows.Close()
				}
			}

			tableQualifiedPath := qualifiedPath
			if tableQualifiedPath == "" {
				tableQualifiedPath = nodeName
			} else {
				if strings.HasPrefix(tableQualifiedPath, "/") {
					p := strings.TrimPrefix(tableQualifiedPath, "/")
					parts := strings.Split(p, "/")
					if len(parts) >= 2 {
						tableQualifiedPath = fmt.Sprintf("%s.%s", parts[0], parts[1])
					}
				}
			}

			isTableCore := false
			var tableCoreID interface{} = nil
			if coreID.Valid && coreID.String != "" {
				isTableCore = true
				tableCoreID = coreID.String
			}

			node := map[string]interface{}{
				"id":   nodeID,
				"type": "table",
				"data": map[string]interface{}{
					"label":     nodeName,
					"tableName": tableQualifiedPath,
					"schemaName": func() string {
						if strings.Contains(tableQualifiedPath, ".") {
							return strings.Split(tableQualifiedPath, ".")[0]
						}
						return ""
					}(),
					"schema": func() string {
						if strings.Contains(tableQualifiedPath, ".") {
							return strings.Split(tableQualifiedPath, ".")[0]
						}
						return ""
					}(),
					"nodeType":      "table",
					"nodeId":        nodeID,
					"isCore":        isTableCore,
					"core_id":       tableCoreID,
					"columns":       enhancedCols,
					"qualifiedPath": tableQualifiedPath,
					"description":   fmt.Sprintf("Table: %s", tableQualifiedPath),
					"columnCount":   len(enhancedCols),
				},
			}
			outNodes = append(outNodes, node)
		}

		if err := txx.Commit(); err != nil {
			http.Error(w, fmt.Sprintf("failed to commit transaction: %v", err), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{"tables": outNodes, "count": len(outNodes)})
	})

	r.Post("/fabric/models/validate", func(w http.ResponseWriter, r *http.Request) {
		var req handlers.SaveExtensionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request format", http.StatusBadRequest)
			return
		}
		b, err := json.Marshal(req.ModelObject)
		if err != nil {
			http.Error(w, "invalid model_object payload", http.StatusBadRequest)
			return
		}
		var ext cube.Cube
		if err := json.Unmarshal(b, &ext); err != nil {
			http.Error(w, "invalid model_object structure", http.StatusBadRequest)
			return
		}
		baseKey := req.BaseModelKey
		if baseKey == "" {
			if s, ok := ext.Extends.(string); ok {
				baseKey = s
			} else if s, ok := ext.Metadata["inherits_from"].(string); ok {
				baseKey = s
			}
		}
		if baseKey == "" {
			http.Error(w, "base_model_key is required or must be present in model_object", http.StatusBadRequest)
			return
		}
		if !strings.HasPrefix(baseKey, "/") {
			baseKey = "/" + baseKey
		}
		dsIDStr := r.URL.Query().Get("datasource_id")
		if dsIDStr == "" {
			http.Error(w, "datasource_id query parameter is required", http.StatusBadRequest)
			return
		}
		dsID, err := uuid.Parse(dsIDStr)
		if err != nil {
			http.Error(w, "invalid datasource_id format", http.StatusBadRequest)
			return
		}
		baseDefn, err := semanticSvc.GetModelDefinition(dsID, baseKey)
		if err != nil {
			http.Error(w, "base model not found", http.StatusNotFound)
			return
		}
		var baseConfig coremodels.ResolvedModelConfig

		// ResolvedConfig in FabricDefn is JSONB (alias []byte) - convert directly
		raw := []byte(baseDefn.ResolvedConfig)

		if err := json.Unmarshal(raw, &baseConfig); err != nil || len(baseConfig.Cubes) == 0 {
			http.Error(w, "failed to parse base model resolved_config", http.StatusInternalServerError)
			return
		}
		baseCube := baseConfig.Cubes[0]
		issues := cube.ValidateExtension(baseCube, ext)
		colsMap, errCols := semanticSvc.GatherColumnsMapForDatasource(dsID)
		pruning := []cube.ValidationIssue{}
		if errCols == nil {
			pruning = semanticSvc.PruneMissingColumnsFromExtension(&ext, colsMap, baseCube.Name)
		}
		all := append(issues, pruning...)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"issues": all})
	})

	r.Get("/fabric/models/definition", func(w http.ResponseWriter, r *http.Request) {
		datasourceIDStr := r.URL.Query().Get("datasource_id")
		modelKey := r.URL.Query().Get("model_key")
		if datasourceIDStr == "" || modelKey == "" {
			http.Error(w, "datasource_id and model_key query parameters are required", http.StatusBadRequest)
			return
		}
		datasourceUUID, err := uuid.Parse(datasourceIDStr)
		if err != nil {
			http.Error(w, "invalid datasource_id format", http.StatusBadRequest)
			return
		}
		if !strings.HasPrefix(modelKey, "/") {
			modelKey = "/" + modelKey
		}
		defn, err := semanticSvc.GetModelDefinition(datasourceUUID, modelKey)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, "failed to retrieve model definition", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(defn)
	})

	r.Patch("/tenants/{tenant_id}/datasources/{datasource_id}/models/{model_id}", s.ModelCatalogHandler.UpdateModel)

	r.Post("/fabric/models/metadata", func(w http.ResponseWriter, r *http.Request) {
		var req handlers.ModelMetadataBatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request format", http.StatusBadRequest)
			return
		}
		datasourceUUID, err := uuid.Parse(req.DatasourceID)
		if err != nil {
			http.Error(w, "invalid datasource_id", http.StatusBadRequest)
			return
		}
		metadataMap, err := semanticSvc.GetModelMetadata(datasourceUUID, req.TableNames)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp := map[string]interface{}{"results": metadataMap}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	r.Post("/fabric/models/generate", func(w http.ResponseWriter, r *http.Request) {
		var req coremodels.GenerateModelsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request format", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.DatasourceID) == "" {
			writeJSONError(w, http.StatusBadRequest, "datasource_id is required", "missing_datasource_id", nil)
			return
		}
		if _, err := uuid.Parse(req.DatasourceID); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid datasource_id", "invalid_datasource_id", err.Error())
			return
		}
		svcs := coremodels.Services{SemanticModelService: semanticSvc}
		resp, err := coremodels.Generate(svcs, req)
		if err != nil {
			if strings.Contains(err.Error(), "validation failed") {
				writeJSONError(w, http.StatusBadRequest, err.Error(), "validation_failed", nil)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, err.Error(), "internal_error", nil)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	r.Post("/fabric/models/generate-defaults", func(w http.ResponseWriter, r *http.Request) {
		handleGenerateDefaults(w, r, semanticSvc)
	})

	r.Get("/metadata/{datasourceId}", func(w http.ResponseWriter, r *http.Request) {
		datasourceIDStr := chi.URLParam(r, "datasourceId")
		if datasourceIDStr == "" {
			http.Error(w, "datasource_id is required", http.StatusBadRequest)
			return
		}
		datasourceUUID, err := uuid.Parse(datasourceIDStr)
		if err != nil {
			http.Error(w, "invalid datasource_id format", http.StatusBadRequest)
			return
		}
		mockColumns := []map[string]interface{}{ /* minimal mock columns */ }
		resp := map[string]interface{}{"columns": mockColumns, "datasource_id": datasourceUUID.String()}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	r.Get("/fabric/joins/{datasourceId}", func(w http.ResponseWriter, r *http.Request) {
		datasourceIDStr := chi.URLParam(r, "datasourceId")
		if datasourceIDStr == "" {
			http.Error(w, "datasource_id is required", http.StatusBadRequest)
			return
		}
		datasourceUUID, err := uuid.Parse(datasourceIDStr)
		if err != nil {
			http.Error(w, "invalid datasource_id format", http.StatusBadRequest)
			return
		}
		joinExtractor := cube.NewDatabaseJoinExtractor(s.DB)
		joins, err := joinExtractor.ExtractJoins(datasourceUUID.String())
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to extract joins: %v", err), http.StatusInternalServerError)
			return
		}
		resp := map[string]interface{}{"joins": joins, "count": len(joins), "datasource_id": datasourceUUID.String()}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	r.Get("/fabric/joins/{datasourceId}/table/{tableName}", func(w http.ResponseWriter, r *http.Request) {
		datasourceIDStr := chi.URLParam(r, "datasourceId")
		tableName := chi.URLParam(r, "tableName")
		if datasourceIDStr == "" || tableName == "" {
			http.Error(w, "datasource_id and table_name are required", http.StatusBadRequest)
			return
		}
		datasourceUUID, err := uuid.Parse(datasourceIDStr)
		if err != nil {
			http.Error(w, "invalid datasource_id format", http.StatusBadRequest)
			return
		}
		joinExtractor := cube.NewDatabaseJoinExtractor(s.DB)
		joinDefs, err := joinExtractor.GenerateJoinDefinitions(tableName, datasourceUUID.String())
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to generate joins for table %s: %v", tableName, err), http.StatusInternalServerError)
			return
		}
		resp := map[string]interface{}{"table_name": tableName, "joins": joinDefs, "count": len(joinDefs), "datasource_id": datasourceUUID.String()}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	r.Post("/fabric/cubes/generate-from-table", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			DatasourceID string `json:"datasource_id"`
			TableName    string `json:"table_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request format", http.StatusBadRequest)
			return
		}
		if req.DatasourceID == "" || req.TableName == "" {
			http.Error(w, "datasource_id and table_name are required", http.StatusBadRequest)
			return
		}
		datasourceUUID, err := uuid.Parse(req.DatasourceID)
		if err != nil {
			http.Error(w, "invalid datasource_id format", http.StatusBadRequest)
			return
		}
		joinExtractor := cube.NewDatabaseJoinExtractor(s.DB)
		generatedCube, err := joinExtractor.GenerateCubeFromTable(req.TableName, datasourceUUID.String())
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to generate cube from table %s: %v", req.TableName, err), http.StatusInternalServerError)
			return
		}
		resp := map[string]interface{}{"cube": generatedCube, "table_name": req.TableName, "datasource_id": datasourceUUID.String()}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	r.Get("/fabric/extensions/compatibility-report", func(w http.ResponseWriter, r *http.Request) {
		datasourceIDStr := r.URL.Query().Get("datasource_id")
		if datasourceIDStr == "" {
			http.Error(w, "datasource_id query parameter is required", http.StatusBadRequest)
			return
		}
		if _, err := uuid.Parse(datasourceIDStr); err != nil {
			http.Error(w, "invalid datasource_id format", http.StatusBadRequest)
			return
		}
		resp := map[string]interface{}{"compatible": true, "version": "1.0.0", "features": []string{"core_models", "custom_models", "semantic_layer", "data_catalog"}, "warnings": []string{}, "datasource_id": datasourceIDStr}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
}

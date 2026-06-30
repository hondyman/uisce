package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

// registerRestQueryRoutes registers the standard REST query routes replacing GraphQL
func (s *Server) registerRestQueryRoutes(r chi.Router) {
	r.Get("/rest/products", s.handleRestProducts)
	r.Get("/rest/semantic-models", s.handleRestSemanticModels)
	r.Get("/rest/charts", s.handleRestCharts)
	r.Get("/rest/catalog-nodes", s.handleRestCatalogNodes)
	r.Get("/rest/catalog-edges", s.handleRestCatalogEdges)
	r.Get("/rest/semantic-assets", s.handleRestSemanticAssets)
	r.Get("/rest/relationship-suggestions", s.handleRestRelationshipSuggestions)
	r.Get("/rest/datasources", s.handleRestDatasources)
	r.Post("/rest/fabric-defn", s.handlePostFabricDefn)

	// Microservices REST integration
	r.Get("/rest/screen-configs", s.handleRestGetScreenConfigs)
	r.Post("/rest/screen-configs", s.handleRestPostScreenConfigs)
	r.Put("/rest/screen-configs", s.handleRestPutScreenConfigs)
	r.Delete("/rest/screen-configs", s.handleRestDeleteScreenConfigs)
	
	r.Get("/rest/workflow-history", s.handleRestGetWorkflowHistory)
	r.Post("/rest/workflow-history", s.handleRestPostWorkflowHistory)
	r.Get("/rest/workflow-rules", s.handleRestGetWorkflowRules)
}

func queryToJSON(db *sqlx.DB, query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := db.Queryx(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []map[string]interface{}{}
	for rows.Next() {
		row := make(map[string]interface{})
		err := rows.MapScan(row)
		if err != nil {
			return nil, err
		}
		for k, v := range row {
			if b, ok := v.([]byte); ok {
				var js interface{}
				if err := json.Unmarshal(b, &js); err == nil {
					row[k] = js
				} else {
					row[k] = string(b)
				}
			}
		}
		results = append(results, row)
	}
	return results, nil
}

func writeJSONResponse(w http.ResponseWriter, data interface{}, err error) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(data)
}

func (s *Server) handleRestProducts(w http.ResponseWriter, r *http.Request) {
	res, err := queryToJSON(s.SQLXDB, "SELECT id, product_name, is_active, product_code, status, created_at, updated_at FROM alpha_product ORDER BY product_name ASC")
	writeJSONResponse(w, res, err)
}

func (s *Server) handleRestSemanticModels(w http.ResponseWriter, r *http.Request) {
	res, err := queryToJSON(s.SQLXDB, "SELECT id, name, model_type, dataset_id, configuration, created_at, updated_at FROM semantic_models ORDER BY name ASC")
	writeJSONResponse(w, res, err)
}

func (s *Server) handleRestCharts(w http.ResponseWriter, r *http.Request) {
	datasourceID := r.URL.Query().Get("tenant_datasource_id")
	chartName := r.URL.Query().Get("chart_name")

	q := "SELECT id, chart_name, chart, created_at, updated_at, tenant_datasource_id FROM tenant_chart WHERE 1=1"
	args := []interface{}{}

	if datasourceID != "" {
		q += " AND tenant_datasource_id = $1"
		args = append(args, datasourceID)
	}
	if chartName != "" {
		if len(args) > 0 {
			q += " AND chart_name = $2"
		} else {
			q += " AND chart_name = $1"
		}
		args = append(args, chartName)
	}
	q += " ORDER BY chart_name ASC"

	res, err := queryToJSON(s.SQLXDB, q, args...)
	writeJSONResponse(w, res, err)
}

func (s *Server) handleRestCatalogNodes(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	datasourceID := r.URL.Query().Get("tenant_datasource_id")
	nodeTypeID := r.URL.Query().Get("node_type_id")
	parentID := r.URL.Query().Get("parent_id")
	searchQuery := r.URL.Query().Get("q")
	useView := r.URL.Query().Get("use_view") == "true"
	limitStr := r.URL.Query().Get("limit")

	limit := 100
	if limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
		}
	}

	table := "catalog_node"
	if useView {
		table = "catalog_node_vw"
	}

	q := "SELECT * FROM " + table + " WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if id != "" {
		field := "id"
		if useView {
			field = "node_id"
		}
		q += " AND " + field + " = $" + strconv.Itoa(argIdx)
		args = append(args, id)
		argIdx++
	}

	if datasourceID != "" {
		field := "tenant_datasource_id"
		if useView {
			field = "tenant_tenant_instance_id"
		}
		q += " AND " + field + " = $" + strconv.Itoa(argIdx)
		args = append(args, datasourceID)
		argIdx++
	}

	if nodeTypeID != "" {
		q += " AND node_type_id = $" + strconv.Itoa(argIdx)
		args = append(args, nodeTypeID)
		argIdx++
	}

	if parentID != "" {
		q += " AND parent_id = $" + strconv.Itoa(argIdx)
		args = append(args, parentID)
		argIdx++
	}

	if searchQuery != "" {
		q += " AND (node_name ILIKE $" + strconv.Itoa(argIdx) + " OR qualified_path ILIKE $" + strconv.Itoa(argIdx)
		if useView {
			q += " OR source_name ILIKE $" + strconv.Itoa(argIdx)
		}
		q += ")"
		args = append(args, "%"+searchQuery+"%")
		argIdx++
	}

	q += " ORDER BY qualified_path ASC LIMIT $" + strconv.Itoa(argIdx)
	args = append(args, limit)

	res, err := queryToJSON(s.SQLXDB, q, args...)
	writeJSONResponse(w, res, err)
}

func (s *Server) handleRestCatalogEdges(w http.ResponseWriter, r *http.Request) {
	datasourceID := r.URL.Query().Get("tenant_datasource_id")
	sourceNodeID := r.URL.Query().Get("source_node_id")
	relTypesStr := r.URL.Query().Get("relationship_types")

	q := "SELECT id, source_node_id, target_node_id, relationship_type, properties, created_at FROM catalog_edge WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if datasourceID != "" {
		q += " AND tenant_datasource_id = $" + strconv.Itoa(argIdx)
		args = append(args, datasourceID)
		argIdx++
	}

	if sourceNodeID != "" {
		q += " AND source_node_id = $" + strconv.Itoa(argIdx)
		args = append(args, sourceNodeID)
		argIdx++
	}

	if relTypesStr != "" {
		types := strings.Split(relTypesStr, ",")
		placeholders := make([]string, len(types))
		for i := range types {
			placeholders[i] = "$" + strconv.Itoa(argIdx)
			args = append(args, types[i])
			argIdx++
		}
		q += " AND relationship_type IN (" + strings.Join(placeholders, ",") + ")"
	}

	res, err := queryToJSON(s.SQLXDB, q, args...)
	writeJSONResponse(w, res, err)
}

func (s *Server) handleRestSemanticAssets(w http.ResponseWriter, r *http.Request) {
	businessEntityID := r.URL.Query().Get("business_entity_id")
	datasourceID := r.URL.Query().Get("tenant_instance_id")

	q := "SELECT * FROM semantic_assets WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if businessEntityID != "" {
		q += " AND business_entity_id = $" + strconv.Itoa(argIdx)
		args = append(args, businessEntityID)
		argIdx++
	}

	if datasourceID != "" {
		q += " AND tenant_instance_id = $" + strconv.Itoa(argIdx)
		args = append(args, datasourceID)
		argIdx++
	}

	res, err := queryToJSON(s.SQLXDB, q, args...)
	writeJSONResponse(w, res, err)
}

func (s *Server) handleRestRelationshipSuggestions(w http.ResponseWriter, r *http.Request) {
	sourceEntityID := r.URL.Query().Get("source_entity_id")
	datasourceID := r.URL.Query().Get("tenant_instance_id")
	limitStr := r.URL.Query().Get("limit")

	limit := 100
	if limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
		}
	}

	q := "SELECT id, source_entity_id, target_entity_id, confidence, rationale, scoring_breakdown, accepted, created_at FROM relationship_suggestions WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if sourceEntityID != "" {
		q += " AND source_entity_id = $" + strconv.Itoa(argIdx)
		args = append(args, sourceEntityID)
		argIdx++
	}

	if datasourceID != "" {
		q += " AND tenant_instance_id = $" + strconv.Itoa(argIdx)
		args = append(args, datasourceID)
		argIdx++
	}

	q += " ORDER BY confidence DESC LIMIT $" + strconv.Itoa(argIdx)
	args = append(args, limit)

	res, err := queryToJSON(s.SQLXDB, q, args...)
	writeJSONResponse(w, res, err)
}

func (s *Server) handleRestDatasources(w http.ResponseWriter, r *http.Request) {
	res, err := queryToJSON(s.SQLXDB, "SELECT id, datasource_code, config, datasource_name, is_active FROM alpha_datasource ORDER BY datasource_name ASC")
	writeJSONResponse(w, res, err)
}

func (s *Server) handlePostFabricDefn(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var body struct {
		Input struct {
			ModelKey       string      `json:"model_key"`
			Version        int         `json:"version"`
			Title          *string     `json:"title"`
			Description    *string     `json:"description"`
			SourceConfig   interface{} `json:"source_config"`
			ResolvedConfig interface{} `json:"resolved_config"`
			ChecksumSHA256 *string     `json:"checksum_sha256"`
		} `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload: " + err.Error()})
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := claims.UserID
	if userID == "" {
		userID = "00000000-0000-0000-0000-000000000000"
	}
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	if tenantDatasourceID == "" {
		tenantDatasourceID = "00000000-0000-0000-0000-000000000000"
	}

	sourceConfigJSON, _ := json.Marshal(body.Input.SourceConfig)
	resolvedConfigJSON, _ := json.Marshal(body.Input.ResolvedConfig)

	var id string
	var status string
	var createdAt string

	query := `
		INSERT INTO fabric_defn (
			tenant_id, tenant_datasource_id, model_key, version, title, description,
			source_config, resolved_config, created_by, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'draft')
		RETURNING id, status, created_at
	`

	err := s.SQLXDB.QueryRow(
		query,
		tenantID,
		tenantDatasourceID,
		body.Input.ModelKey,
		body.Input.Version,
		body.Input.Title,
		body.Input.Description,
		sourceConfigJSON,
		resolvedConfigJSON,
		userID,
	).Scan(&id, &status, &createdAt)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "insert failed: " + err.Error()})
		return
	}

	response := map[string]interface{}{
		"insert_fabric_defn_one": map[string]interface{}{
			"id":         id,
			"tenant_id":  tenantID,
			"model_key":  body.Input.ModelKey,
			"version":    body.Input.Version,
			"status":     status,
			"created_by": userID,
			"created_at": createdAt,
		},
	}
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleRestGetScreenConfigs(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	boType := r.URL.Query().Get("bo_type")
	id := r.URL.Query().Get("id")

	q := "SELECT * FROM screen_configs WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if id != "" {
		q += " AND id = $" + strconv.Itoa(argIdx)
		args = append(args, id)
		argIdx++
	}
	if tenantID != "" {
		q += " AND tenant_id = $" + strconv.Itoa(argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if boType != "" {
		q += " AND bo_type = $" + strconv.Itoa(argIdx)
		args = append(args, boType)
		argIdx++
	}

	q += " ORDER BY created_at DESC"
	res, err := queryToJSON(s.SQLXDB, q, args...)
	writeJSONResponse(w, res, err)
}

func (s *Server) handleRestPostScreenConfigs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	layout, _ := json.Marshal(body["layout_json"])
	filters, _ := json.Marshal(body["filters_json"])
	actions, _ := json.Marshal(body["actions_json"])
	perms, _ := json.Marshal(body["permissions_json"])

	query := `
		INSERT INTO screen_configs (
			id, tenant_id, bo_type, screen_name, screen_type, layout_json,
			filters_json, actions_json, permissions_json, is_published, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := s.SQLXDB.Exec(
		query,
		body["id"],
		body["tenant_id"],
		body["bo_type"],
		body["screen_name"],
		body["screen_type"],
		layout,
		filters,
		actions,
		perms,
		body["is_published"],
		body["created_by"],
	)

	writeJSONResponse(w, map[string]string{"status": "created"}, err)
}

func (s *Server) handleRestPutScreenConfigs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var body struct {
		TenantID string                 `json:"tenant_id"`
		ScreenID string                 `json:"screen_id"`
		Updates  map[string]interface{} `json:"updates"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if len(body.Updates) == 0 {
		writeJSONResponse(w, map[string]int{"affected_rows": 0}, nil)
		return
	}

	q := "UPDATE screen_configs SET "
	args := []interface{}{}
	argIdx := 1

	sets := []string{}
	for k, v := range body.Updates {
		// Convert maps/slices to json
		if m, ok := v.(map[string]interface{}); ok {
			b, _ := json.Marshal(m)
			args = append(args, b)
		} else if sl, ok := v.([]interface{}); ok {
			b, _ := json.Marshal(sl)
			args = append(args, b)
		} else {
			args = append(args, v)
		}
		sets = append(sets, fmt.Sprintf("%s = $%d", k, argIdx))
		argIdx++
	}

	q += strings.Join(sets, ", ")
	q += fmt.Sprintf(" WHERE tenant_id = $%d AND id = $%d", argIdx, argIdx+1)
	args = append(args, body.TenantID, body.ScreenID)

	res, err := s.SQLXDB.Exec(q, args...)
	if err != nil {
		writeJSONResponse(w, nil, err)
		return
	}

	rows, _ := res.RowsAffected()
	writeJSONResponse(w, map[string]int64{"affected_rows": rows}, nil)
}

func (s *Server) handleRestDeleteScreenConfigs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID := r.URL.Query().Get("tenant_id")
	screenID := r.URL.Query().Get("screen_id")

	res, err := s.SQLXDB.Exec("DELETE FROM screen_configs WHERE tenant_id = $1 AND id = $2", tenantID, screenID)
	if err != nil {
		writeJSONResponse(w, nil, err)
		return
	}

	rows, _ := res.RowsAffected()
	writeJSONResponse(w, map[string]int64{"affected_rows": rows}, nil)
}

func (s *Server) handleRestGetWorkflowHistory(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	boType := r.URL.Query().Get("bo_type")
	boID := r.URL.Query().Get("bo_id")

	q := "SELECT id, tenant_id, bo_type, bo_id, workflow_name, step_name, status, details, created_at, user_id FROM workflow_history WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if tenantID != "" {
		q += " AND tenant_id = $" + strconv.Itoa(argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if boType != "" {
		q += " AND bo_type = $" + strconv.Itoa(argIdx)
		args = append(args, boType)
		argIdx++
	}
	if boID != "" {
		q += " AND bo_id = $" + strconv.Itoa(argIdx)
		args = append(args, boID)
		argIdx++
	}

	q += " ORDER BY created_at DESC"
	res, err := queryToJSON(s.SQLXDB, q, args...)
	writeJSONResponse(w, res, err)
}

func (s *Server) handleRestPostWorkflowHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	details, _ := json.Marshal(body["details"])

	query := `
		INSERT INTO workflow_history (
			id, tenant_id, bo_type, bo_id, workflow_name, step_name, status, details, user_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := s.SQLXDB.Exec(
		query,
		body["id"],
		body["tenant_id"],
		body["bo_type"],
		body["bo_id"],
		body["workflow_name"],
		body["step_name"],
		body["status"],
		details,
		body["user_id"],
	)

	writeJSONResponse(w, map[string]string{"status": "created"}, err)
}

func (s *Server) handleRestGetWorkflowRules(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	workflowName := r.URL.Query().Get("workflow_name")
	stepName := r.URL.Query().Get("step_name")
	isActive := r.URL.Query().Get("is_active")

	q := "SELECT id, workflow_name, step_name, step_order, condition_json, action_on_success, action_on_failure, error_message, timeout_seconds, retry_count FROM workflow_rules WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if tenantID != "" {
		q += " AND tenant_id = $" + strconv.Itoa(argIdx)
		args = append(args, tenantID)
		argIdx++
	}
	if workflowName != "" {
		q += " AND workflow_name = $" + strconv.Itoa(argIdx)
		args = append(args, workflowName)
		argIdx++
	}
	if stepName != "" {
		q += " AND step_name = $" + strconv.Itoa(argIdx)
		args = append(args, stepName)
		argIdx++
	}
	if isActive != "" {
		val, _ := strconv.ParseBool(isActive)
		q += " AND is_active = $" + strconv.Itoa(argIdx)
		args = append(args, val)
		argIdx++
	}

	q += " ORDER BY step_order ASC"
	res, err := queryToJSON(s.SQLXDB, q, args...)
	writeJSONResponse(w, res, err)
}

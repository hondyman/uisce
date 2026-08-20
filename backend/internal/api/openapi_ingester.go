package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// OpenAPIIngestRequest is the body of POST /api/api-dispatcher/ingest-openapi.
//
// Exactly one of `Spec` (a parsed OpenAPI 3.0 JSON object) or `URL` is required.
// If `TenantID` is empty, the gold-copy tenant is used so the resulting
// inventory is visible to every tenant.
type OpenAPIIngestRequest struct {
	TenantID  string          `json:"tenant_id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Spec      json.RawMessage `json:"spec,omitempty"`
	URL       string          `json:"url,omitempty"`
	HTTPClient *http.Client   `json:"-"`
}

// OpenAPIIngestResult is returned on a successful ingestion.
type OpenAPIIngestResult struct {
	DatasourceID      string `json:"datasource_id"`
	DatasourceName    string `json:"datasource_name"`
	ResourcesCreated  int    `json:"resources_created"`
	EndpointsCreated  int    `json:"endpoints_created"`
	FieldsCreated     int    `json:"fields_created"`
	SemanticEdgesLinked int  `json:"semantic_edges_linked"`
}

// HTTP methods that the ingester recognises. Order is also the rendering order
// in the catalogue UI.
var openAPIMethods = []string{"get", "post", "put", "patch", "delete"}

// parsedSpec is the minimal projection of an OpenAPI 3.0 document that this
// ingester needs. Anything we don't model is dropped on the floor; we
// deliberately do not pull in a full OpenAPI 3.0 parser to keep the binary
// small.
type parsedSpec struct {
	OpenAPI string         `json:"openapi"`
	Info    parsedInfo     `json:"info"`
	Servers []parsedServer `json:"servers"`
	Paths   map[string]json.RawMessage `json:"paths"`
	Components struct {
		Schemes map[string]json.RawMessage `json:"securitySchemes"`
		Schemas map[string]json.RawMessage `json:"schemas"`
	} `json:"components"`
}

type parsedInfo struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type parsedServer struct {
	URL string `json:"url"`
}

// parsedOperation is the projection of an HTTP method's OpenAPI Operation.
type parsedOperation struct {
	Summary     string `json:"summary"`
	Description string `json:"description"`
	OperationID string `json:"operationId"`
	Tags        []string `json:"tags"`
	Responses   map[string]struct {
		Content map[string]struct {
			Schema json.RawMessage `json:"schema"`
		} `json:"content"`
	} `json:"responses"`
	RequestBody *struct {
		Content map[string]struct {
			Schema json.RawMessage `json:"schema"`
		} `json:"content"`
	} `json:"requestBody,omitempty"`
}

// fieldDef is the projection of a single property inside a response/request
// schema. We extract just enough to populate an api_field catalog_node.
type fieldDef struct {
	Name           string
	JSONPath       string
	DataType       string
	Description    string
	IsPrimaryKey   bool
	SemanticTerm   string // x-semantic-term extension value
}

// IngestOpenAPISpec parses an OpenAPI 3.0 document (passed inline or fetched
// from a URL) and creates catalog_node + catalog_edge rows under the target
// tenant. The created structure is:
//
//	api_datasource
//	  └─ api_resource (one per path prefix or OpenAPI tag)
//	       └─ api_endpoint (one per (path, method))
//	            └─ api_field (one per schema property)
//
// The function is idempotent at the (tenant, datasource_name) level: re-running
// the same ingest produces no duplicate datasources (the second run updates the
// existing rows in place).
func (h *ApiDispatcherHandler) IngestOpenAPISpec(ctx context.Context, req OpenAPIIngestRequest) (*OpenAPIIngestResult, error) {
	if len(req.Spec) == 0 && strings.TrimSpace(req.URL) == "" {
		return nil, fmt.Errorf("either 'spec' or 'url' is required")
	}

	specBytes, err := h.resolveSpecBytes(ctx, req)
	if err != nil {
		return nil, err
	}

	var spec parsedSpec
	if err := json.Unmarshal(specBytes, &spec); err != nil {
		return nil, fmt.Errorf("parse OpenAPI spec: %w", err)
	}
	if !strings.HasPrefix(spec.OpenAPI, "3.") {
		return nil, fmt.Errorf("only OpenAPI 3.x is supported (got %q); use a converter for 2.0 / Swagger", spec.OpenAPI)
	}

	// System-level prerequisites (now that input has been validated).
	if h.db == nil {
		return nil, fmt.Errorf("api dispatcher has no database handle")
	}

	// Resolve tenant. Default to gold copy when no tenant_id is supplied so
	// inventories are visible to every tenant.
	tenantID := req.TenantID
	if tenantID == "" {
		tenantID = h.resolveGoldCopyTenant(ctx)
		if tenantID == "" {
			return nil, fmt.Errorf("no tenant_id supplied and no gold-copy tenant exists")
		}
	}

	// Ensure the api_* node + edge types exist for this tenant.
	types, err := h.ensureApiNodeTypes(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("ensure api node types: %w", err)
	}

	// Compose datasource name from info.title or the caller-provided override.
	dsName := strings.TrimSpace(req.Name)
	if dsName == "" {
		dsName = strings.TrimSpace(spec.Info.Title)
	}
	if dsName == "" {
		dsName = "Imported API"
	}

	baseURL := ""
	if len(spec.Servers) > 0 {
		baseURL = strings.TrimRight(spec.Servers[0].URL, "/")
	}
	if baseURL == "" {
		baseURL = "https://"
	}
	if u, err := url.Parse(baseURL); err == nil && u.Scheme == "" {
		baseURL = "https://" + strings.TrimLeft(baseURL, "/")
	}

	defaultAuth := inferDefaultAuth(spec.Components.Schemes)

	dsConfig, _ := json.Marshal(map[string]interface{}{
		"service_type":    "openapi",
		"default_auth":    defaultAuth,
		"default_base_url": baseURL,
		"openapi_version": spec.OpenAPI,
	})
	dsProps, _ := json.Marshal(map[string]interface{}{
		"protocol":      "REST",
		"api_version":   spec.Info.Version,
		"spec_source":   "openapi_ingester",
	})

	dsID, err := h.upsertCatalogNode(ctx, tenantID, types.DatasourceType, "", dsName, "/api/"+slug(dsName), spec.Info.Description, dsConfig, dsProps)
	if err != nil {
		return nil, fmt.Errorf("upsert datasource: %w", err)
	}

	result := &OpenAPIIngestResult{
		DatasourceID:   dsID,
		DatasourceName: dsName,
	}

	// Walk paths × methods. Group endpoints into resources by the first path
	// segment OR (if present) the first OpenAPI tag.
	for rawPath, rawPathItem := range spec.Paths {
		var methods map[string]json.RawMessage
		if err := json.Unmarshal(rawPathItem, &methods); err != nil {
			continue
		}
		path := rawPath
		for _, method := range openAPIMethods {
			rawOp, ok := methods[method]
			if !ok {
				continue
			}
			var op parsedOperation
			if err := json.Unmarshal(rawOp, &op); err != nil {
				continue
			}
			resourceName := resourceNameFor(path, op.Tags)

			resourceID, err := h.upsertCatalogNode(ctx, tenantID, types.ResourceType, dsID, resourceName, "/api/"+slug(dsName)+"/"+slug(resourceName), "", json.RawMessage(`{}`), json.RawMessage(`{}`))
			if err != nil {
				return nil, fmt.Errorf("upsert resource %q: %w", resourceName, err)
			}
			result.ResourcesCreated++

			// Compose endpoint node.
			epName := strings.ToUpper(method) + " " + path
			if op.Summary != "" && !strings.Contains(op.Summary, path) {
				epName = strings.ToUpper(method) + " " + path
			}
			epConfig, _ := json.Marshal(map[string]interface{}{
				"method":        strings.ToUpper(method),
				"path_template": path,
				"operation_id":  op.OperationID,
				"response_root": "$",
			})
			epProps, _ := json.Marshal(map[string]interface{}{
				"data_type": "json",
			})
			epDesc := op.Description
			if epDesc == "" {
				epDesc = op.Summary
			}
			epID, err := h.upsertCatalogNode(ctx, tenantID, types.EndpointType, resourceID, epName, "/api/"+slug(dsName)+"/"+slug(resourceName)+"/"+slug(path)+"_"+strings.ToUpper(method), epDesc, epConfig, epProps)
			if err != nil {
				return nil, fmt.Errorf("upsert endpoint %q: %w", epName, err)
			}
			result.EndpointsCreated++

			// Walk response schemas → fields.
			fields := collectFields(op, spec.Components.Schemas)
			for _, f := range fields {
				fcfg, _ := json.Marshal(map[string]interface{}{
					"json_path": f.JSONPath,
				})
				fprops, _ := json.Marshal(map[string]interface{}{
					"data_type":      f.DataType,
					"is_primary_key": f.IsPrimaryKey,
				})
				fid, err := h.upsertCatalogNode(ctx, tenantID, types.FieldType, epID, f.Name, "/api/"+slug(dsName)+"/"+slug(resourceName)+"/"+slug(path)+"_"+strings.ToUpper(method)+"/"+slug(f.Name), f.Description, fcfg, fprops)
				if err != nil {
					return nil, fmt.Errorf("upsert field %q: %w", f.Name, err)
				}
				result.FieldsCreated++

				// Optional semantic-term mapping via x-semantic-term.
				if f.SemanticTerm != "" {
					if _, ok := h.linkSemanticTermByName(ctx, tenantID, fid, f.SemanticTerm); ok {
						result.SemanticEdgesLinked++
					}
				}
			}
		}
	}

	return result, nil
}

// resolveSpecBytes returns the raw JSON bytes of the spec, either from the
// inline `Spec` field or by fetching `URL`.
func (h *ApiDispatcherHandler) resolveSpecBytes(ctx context.Context, req OpenAPIIngestRequest) ([]byte, error) {
	if len(req.Spec) > 0 {
		// Validate it's JSON; reject YAML for now.
		var probe interface{}
		if err := json.Unmarshal(req.Spec, &probe); err != nil {
			return nil, fmt.Errorf("spec field is not valid JSON: %w (only OpenAPI 3.0 JSON is supported in v1; convert YAML via `yq -o=json`)", err)
		}
		return req.Spec, nil
	}
	if strings.TrimSpace(req.URL) == "" {
		return nil, fmt.Errorf("either spec or url is required")
	}
	client := req.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("build fetch request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json, application/yaml")
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("fetch spec: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("fetch spec: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read spec body: %w", err)
	}
	// Trim leading whitespace for YAML-style specs.
	body = []byte(strings.TrimSpace(string(body)))
	return body, nil
}

// resolveGoldCopyTenant returns the id of the tenant marked gold_copy=true.
func (h *ApiDispatcherHandler) resolveGoldCopyTenant(ctx context.Context) string {
	var id string
	row := h.db.QueryRowContext(ctx, "SELECT id FROM tenants WHERE gold_copy = true ORDER BY created_at LIMIT 1")
	if err := row.Scan(&id); err != nil {
		return ""
	}
	return id
}

// apiNodeTypeIDs groups the UUIDs of the api_* catalog_node_types rows for a
// tenant. Each is created on demand if absent.
type apiNodeTypeIDs struct {
	DatasourceType string
	ResourceType   string
	EndpointType   string
	FieldType      string
	ResourceEdge   string
	EndpointEdge   string
	FieldEdge      string
}

// ensureApiNodeTypes creates the api_* node types (and their edge types) for
// the target tenant if they do not exist. Idempotent.
func (h *ApiDispatcherHandler) ensureApiNodeTypes(ctx context.Context, tenantID string) (*apiNodeTypeIDs, error) {
	type def struct {
		TypeName string
		Desc     string
		Config   string
	}
	nodeTypes := []def{
		{"api_datasource", "API Datasource Connection / Service Base", `{"category":"datasource","protocol":"http"}`},
		{"api_resource", "API Resource / Object Group", `{"category":"resource"}`},
		{"api_endpoint", "API Operation / HTTP Endpoint", `{"category":"endpoint"}`},
		{"api_field", "API Payload / Parameter Field", `{"category":"field"}`},
	}
	typeIDs := map[string]string{}
	for _, nt := range nodeTypes {
		id, err := h.upsertCatalogNodeType(ctx, tenantID, nt.TypeName, nt.Desc, nt.Config)
		if err != nil {
			return nil, fmt.Errorf("upsert node type %q: %w", nt.TypeName, err)
		}
		typeIDs[nt.TypeName] = id
	}

	type edgeDef struct {
		Name string
		Desc string
	}
	edgeTypes := []edgeDef{
		{"contains_resource", "Datasource contains API Resource"},
		{"contains_endpoint", "Resource contains API Endpoint"},
		{"contains_field", "Endpoint contains Payload Field"},
	}
	edgeIDs := map[string]string{}
	for _, et := range edgeTypes {
		id, err := h.upsertCatalogEdgeType(ctx, tenantID, et.Name, et.Desc)
		if err != nil {
			return nil, fmt.Errorf("upsert edge type %q: %w", et.Name, err)
		}
		edgeIDs[et.Name] = id
	}

	return &apiNodeTypeIDs{
		DatasourceType: typeIDs["api_datasource"],
		ResourceType:   typeIDs["api_resource"],
		EndpointType:   typeIDs["api_endpoint"],
		FieldType:      typeIDs["api_field"],
		ResourceEdge:   edgeIDs["contains_resource"],
		EndpointEdge:   edgeIDs["contains_endpoint"],
		FieldEdge:      edgeIDs["contains_field"],
	}, nil
}

// upsertCatalogNodeType inserts the (tenant_id, catalog_type_name) row if
// missing and returns its UUID.
func (h *ApiDispatcherHandler) upsertCatalogNodeType(ctx context.Context, tenantID, typeName, description, configJSON string) (string, error) {
	var id string
	err := h.db.QueryRowContext(ctx, `
		WITH ins AS (
			INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config, is_active)
			VALUES ($1::uuid, $2, $3, $4::jsonb, true)
			ON CONFLICT (tenant_id, catalog_type_name) DO NOTHING
			RETURNING id
		)
		SELECT id FROM ins
		UNION ALL
		SELECT id FROM catalog_node_types WHERE tenant_id = $1::uuid AND catalog_type_name = $2
		LIMIT 1
	`, tenantID, typeName, description, configJSON).Scan(&id)
	if err != nil {
		return "", err
	}
	if id == "" {
		// Fallback for schemas that use the singular table name catalog_node_type.
		err := h.db.QueryRowContext(ctx, `
			WITH ins AS (
				INSERT INTO catalog_node_type (tenant_id, catalog_type_name, description, config, is_active)
				VALUES ($1::uuid, $2, $3, $4::jsonb, true)
				ON CONFLICT (tenant_id, catalog_type_name) DO NOTHING
				RETURNING id
			)
			SELECT id FROM ins
			UNION ALL
			SELECT id FROM catalog_node_type WHERE tenant_id = $1::uuid AND catalog_type_name = $2
			LIMIT 1
		`, tenantID, typeName, description, configJSON).Scan(&id)
		if err != nil {
			return "", err
		}
	}
	return id, nil
}

// upsertCatalogEdgeType inserts the (tenant_id, edge_type_name) row if
// missing and returns its UUID. The catalog_edge_types.tenant_id column is
// `text` (matching the seed migration's `v_gold_tenant_id::text` literal), so
// we pass the tenant id without the `::uuid` cast.
func (h *ApiDispatcherHandler) upsertCatalogEdgeType(ctx context.Context, tenantID, edgeName, description string) (string, error) {
	var id string
	err := h.db.QueryRowContext(ctx, `
		WITH ins AS (
			INSERT INTO catalog_edge_types (tenant_id, edge_type_name, description, is_active)
			VALUES ($1, $2, $3, true)
			ON CONFLICT (tenant_id, edge_type_name) DO NOTHING
			RETURNING id
		)
		SELECT id FROM ins
		UNION ALL
		SELECT id FROM catalog_edge_types WHERE tenant_id = $1 AND edge_type_name = $2
		LIMIT 1
	`, tenantID, edgeName, description).Scan(&id)
	if err != nil {
		return "", err
	}
	if id == "" {
		err = h.db.QueryRowContext(ctx, `
			WITH ins AS (
				INSERT INTO catalog_edge_type (tenant_id, edge_type_name, description, is_active)
				VALUES ($1, $2, $3, true)
				ON CONFLICT (tenant_id, edge_type_name) DO NOTHING
				RETURNING id
			)
			SELECT id FROM ins
			UNION ALL
			SELECT id FROM catalog_edge_type WHERE tenant_id = $1 AND edge_type_name = $2
			LIMIT 1
		`, tenantID, edgeName, description).Scan(&id)
		if err != nil {
			return "", err
		}
	}
	return id, nil
}

// upsertCatalogNode inserts a new catalog_node with a generated UUID, or
// returns the existing node's id when a row with the same qualified_path +
// type already exists. Re-ingesting the same spec updates in place rather
// than creating duplicates.
func (h *ApiDispatcherHandler) upsertCatalogNode(
	ctx context.Context,
	tenantID, typeID, parentID, name, qualifiedPath, description string,
	configJSON, propsJSON json.RawMessage,
) (string, error) {
	if configJSON == nil {
		configJSON = json.RawMessage(`{}`)
	}
	if propsJSON == nil {
		propsJSON = json.RawMessage(`{}`)
	}
	if tenantID == "" {
		return "", fmt.Errorf("upsertCatalogNode: tenantID is required")
	}
	var existingID string
	row := h.db.QueryRowContext(ctx, `
		SELECT id FROM catalog_node
		WHERE qualified_path = $1 AND node_type_id = $2::uuid
		ORDER BY created_at ASC LIMIT 1
	`, qualifiedPath, typeID)
	if err := row.Scan(&existingID); err == nil {
		// Update in place.
		if _, err := h.db.ExecContext(ctx, `
			UPDATE catalog_node
			SET node_name = $2, description = $3, config = $4::jsonb, properties = $5::jsonb, parent_id = NULLIF($6, '')::uuid, updated_at = NOW()
			WHERE id = $1::uuid
		`, existingID, name, description, string(configJSON), string(propsJSON), parentID); err != nil {
			return "", err
		}
		return existingID, nil
	} else if err != sql.ErrNoRows {
		return "", err
	}

	id := uuid.New().String()
	if _, err := h.db.ExecContext(ctx, `
		INSERT INTO catalog_node (
			id, tenant_id, node_type_id, parent_id, node_name, qualified_path,
			description, config, properties, is_active
		)
		VALUES ($1::uuid, $2::uuid, $3::uuid, NULLIF($4, '')::uuid, $5, $6,
		        $7, $8::jsonb, $9::jsonb, true)
	`, id, tenantID, typeID, parentID, name, qualifiedPath, description, string(configJSON), string(propsJSON)); err != nil {
		return "", err
	}
	return id, nil
}

// linkSemanticTermByName inserts a `has_context` edge between a field node
// and the first semantic_term catalog_node whose node_name matches `termName`.
// Returns true when an edge was inserted.
func (h *ApiDispatcherHandler) linkSemanticTermByName(ctx context.Context, tenantID, fieldNodeID, termName string) (string, bool) {
	var termNodeID string
	row := h.db.QueryRowContext(ctx, `
		SELECT id FROM catalog_node
		WHERE node_type_id = (SELECT id FROM catalog_node_types WHERE catalog_type_name = 'semantic_term' AND (tenant_id = $1::uuid OR tenant_id IN (SELECT id FROM tenants WHERE gold_copy = true)) LIMIT 1)
		  AND LOWER(node_name) = LOWER($2)
		ORDER BY (tenant_id = $1::uuid) DESC, created_at ASC LIMIT 1
	`, tenantID, termName)
	if err := row.Scan(&termNodeID); err != nil {
		return "", false
	}

	var hasContextEdgeID string
	row = h.db.QueryRowContext(ctx, `
		SELECT id FROM catalog_edge_types
		WHERE edge_type_name = 'has_context'
		  AND (tenant_id = $1::uuid OR tenant_id IN (SELECT id FROM tenants WHERE gold_copy = true))
		ORDER BY (tenant_id = $1::uuid) DESC LIMIT 1
	`, tenantID)
	if err := row.Scan(&hasContextEdgeID); err != nil {
		return "", false
	}

	// Skip duplicates.
	var existing string
	row = h.db.QueryRowContext(ctx, `
		SELECT id FROM catalog_edge
		WHERE (source_node_id = $1::uuid OR target_node_id = $1::uuid)
		  AND edge_type_id = $2::uuid
		  AND (source_node_id = $3::uuid OR target_node_id = $3::uuid)
		LIMIT 1
	`, fieldNodeID, hasContextEdgeID, termNodeID)
	if err := row.Scan(&existing); err == nil {
		return existing, false
	}

	edgeID := uuid.New().String()
	if _, err := h.db.ExecContext(ctx, `
		INSERT INTO catalog_edge (id, source_node_id, target_node_id, edge_type_id, relationship_type, tenant_id, properties, is_active, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'has_context', $5::uuid, '{}'::jsonb, true, NOW(), NOW())
	`, edgeID, fieldNodeID, termNodeID, hasContextEdgeID, tenantID); err != nil {
		return "", false
	}
	return edgeID, true
}

// resourceNameFor picks the resource bucket name for a given path.
// Preference order:
//   1. The first OpenAPI tag on the operation
//   2. The first non-parameter path segment, title-cased
//   3. The literal "_root" if the path is "/" or empty
func resourceNameFor(path string, tags []string) string {
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t != "" {
			return t
		}
	}
	trimmed := strings.TrimLeft(path, "/")
	if trimmed == "" {
		return "Root"
	}
	parts := strings.Split(trimmed, "/")
	for _, p := range parts {
		if !strings.HasPrefix(p, "{") {
			return strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return "Root"
}

// slug converts an arbitrary name into a URL-safe path segment.
func slug(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "x"
	}
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_' || r == '/' || r == '.':
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		return "x"
	}
	return out
}

// collectFields walks a parsed operation and returns one fieldDef per
// property in any 2xx response schema (and the request body, if present).
// Recognises inline object schemas, $ref pointers to components.schemas, and
// array item schemas. Skips composition operators (allOf/oneOf/anyOf) which
// v1 doesn't fully resolve.
func collectFields(op parsedOperation, schemas map[string]json.RawMessage) []fieldDef {
	var out []fieldDef
	seen := map[string]bool{}

	visitSchema := func(schema json.RawMessage, parentPath string) {
		if len(schema) == 0 {
			return
		}
		var generic map[string]interface{}
		if err := json.Unmarshal(schema, &generic); err != nil {
			return
		}

		// Dereference top-level $ref to components.schemas/<Name>.
		if ref, _ := generic["$ref"].(string); ref != "" {
			name := strings.TrimPrefix(ref, "#/components/schemas/")
			if body, ok := schemas[name]; ok {
				generic = nil
				_ = json.Unmarshal(body, &generic)
			}
		}

		// Handle top-level array of objects.
		if typeStr, _ := generic["type"].(string); typeStr == "array" {
			if items, ok := generic["items"].(map[string]interface{}); ok {
				// Follow $ref nested inside items.
				if ref, _ := items["$ref"].(string); ref != "" {
					name := strings.TrimPrefix(ref, "#/components/schemas/")
					if body, ok := schemas[name]; ok {
						var resolved map[string]interface{}
						_ = json.Unmarshal(body, &resolved)
						emitFieldsFromProperties(resolved["properties"], "$[*]", &out, seen)
						return
					}
				}
				emitFieldsFromProperties(items["properties"], "$[*]", &out, seen)
			}
			return
		}

		// Handle top-level object.
		emitFieldsFromProperties(generic["properties"], parentPath, &out, seen)
	}

	// Walk 2xx response schemas.
	for status, resp := range op.Responses {
		if !strings.HasPrefix(status, "2") {
			continue
		}
		for _, media := range resp.Content {
			visitSchema(media.Schema, "$")
		}
	}

	// Walk request body schema (POST/PUT/PATCH) so users see what the
	// endpoint accepts.
	if op.RequestBody != nil {
		for _, media := range op.RequestBody.Content {
			visitSchema(media.Schema, "$")
		}
	}

	return out
}

// emitFieldsFromProperties turns a generic OpenAPI `properties` map into
// fieldDef entries. parentPath is prepended to each generated json_path.
func emitFieldsFromProperties(raw interface{}, parentPath string, out *[]fieldDef, seen map[string]bool) {
	props, ok := raw.(map[string]interface{})
	if !ok {
		return
	}
	for name, propRaw := range props {
		if seen[name] {
			continue
		}
		propMap, _ := propRaw.(map[string]interface{})
		dataType := "varchar"
		if v, ok := propMap["type"].(string); ok {
			dataType = mapOpenAPIType(v)
		} else if _, hasRef := propMap["$ref"]; hasRef {
			dataType = "object"
		}
		desc, _ := propMap["description"].(string)
		isPK := false
		if v, ok := propMap["x-primary-key"].(bool); ok && v {
			isPK = true
		}
		semTerm := ""
		if v, ok := propMap["x-semantic-term"].(string); ok {
			semTerm = v
		}
		*out = append(*out, fieldDef{
			Name:         name,
			JSONPath:     parentPath + "." + name,
			DataType:     dataType,
			Description:  desc,
			IsPrimaryKey: isPK,
			SemanticTerm: semTerm,
		})
		seen[name] = true
	}
}

// mapOpenAPIType converts an OpenAPI primitive type to the data_type values
// already used in catalog_node.properties.data_type (varchar, numeric, integer,
// boolean, timestamp).
func mapOpenAPIType(t string) string {
	switch t {
	case "string":
		return "varchar"
	case "integer":
		return "integer"
	case "number":
		return "numeric"
	case "boolean":
		return "boolean"
	case "null":
		return "varchar"
	default:
		return "varchar"
	}
}

// inferDefaultAuth picks a default auth_type string based on the
// components.securitySchemes map. Returns "none" when no scheme is declared.
func inferDefaultAuth(schemes map[string]json.RawMessage) string {
	for _, raw := range schemes {
		var s struct {
			Type string `json:"type"`
			Flows map[string]struct {
				TokenURL string `json:"tokenUrl"`
			} `json:"flows"`
		}
		if err := json.Unmarshal(raw, &s); err != nil {
			continue
		}
		switch s.Type {
		case "oauth2":
			// Look for a refresh-token-capable flow.
			if _, ok := s.Flows["password"]; ok {
				return "oauth2_bearer"
			}
			if _, ok := s.Flows["clientCredentials"]; ok {
				return "oauth2_bearer"
			}
			if _, ok := s.Flows["authorizationCode"]; ok {
				return "oauth2_bearer"
			}
		case "http":
			return "basic_auth"
		case "apiKey":
			return "api_key"
		}
	}
	return "none"
}
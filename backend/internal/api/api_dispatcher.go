package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/oauth"
	"github.com/hondyman/uisce/backend/internal/security"
)

// ApiDispatcherHandler manages API datasource invocation and execution
type ApiDispatcherHandler struct {
	db            *sql.DB
	securityDeps  handlers.SecurityContextDeps
	httpClient    *http.Client
	encryptor     *security.TokenEncryptor
	oauthProvider *oauth.ApiOAuthProvider
}

func NewApiDispatcherHandler(db *sql.DB, securityDeps handlers.SecurityContextDeps, encryptor *security.TokenEncryptor, oauthProvider *oauth.ApiOAuthProvider) *ApiDispatcherHandler {
	return &ApiDispatcherHandler{
		db:            db,
		securityDeps:  securityDeps,
		encryptor:     encryptor,
		oauthProvider: oauthProvider,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// decryptAuthConfig reads a stored ciphertext blob (base64-URL) and returns
// the plaintext JSON map. If the blob is empty/nil it returns an empty map.
// A nil encryptor is treated as a server misconfiguration and returns an error.
func (h *ApiDispatcherHandler) decryptAuthConfig(ciphertext []byte) (map[string]interface{}, error) {
	if len(ciphertext) == 0 {
		return map[string]interface{}{}, nil
	}
	if h.encryptor == nil {
		return nil, fmt.Errorf("api dispatcher has no encryption key configured (API_TOKEN_ENCRYPTION_KEY)")
	}
	plaintext, err := h.encryptor.Decrypt(string(ciphertext))
	if err != nil {
		return nil, fmt.Errorf("decrypt auth_config: %w", err)
	}
	out := map[string]interface{}{}
	if err := json.Unmarshal([]byte(plaintext), &out); err != nil {
		return nil, fmt.Errorf("parse decrypted auth_config: %w", err)
	}
	return out, nil
}

// encryptAuthConfig marshals a config map to JSON and encrypts it with the
// configured TokenEncryptor. Returns nil (no error) when the input map is empty
// so that a tenant can save a base_url without any credentials.
func (h *ApiDispatcherHandler) encryptAuthConfig(cfg map[string]interface{}) ([]byte, error) {
	if len(cfg) == 0 {
		return nil, nil
	}
	if h.encryptor == nil {
		return nil, fmt.Errorf("api dispatcher has no encryption key configured (API_TOKEN_ENCRYPTION_KEY)")
	}
	plaintext, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal auth_config: %w", err)
	}
	ciphertext, err := h.encryptor.Encrypt(string(plaintext))
	if err != nil {
		return nil, fmt.Errorf("encrypt auth_config: %w", err)
	}
	return []byte(ciphertext), nil
}

func (h *ApiDispatcherHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api-dispatcher", func(r chi.Router) {
		r.Get("/datasources", h.ListApiDatasources)
		r.Get("/endpoints", h.ListApiEndpoints)
		r.Get("/endpoints/{id}", h.GetApiEndpointDetail)
		r.Get("/fields", h.ListApiFields)
		r.Post("/fields", h.CreateApiField)
		r.Delete("/fields/{id}", h.DeleteApiField)
		r.Post("/map-term", h.MapFieldToSemanticTerm)
		r.Get("/semantic-terms", h.ListSemanticTerms)
		r.Get("/lineage", h.GetApiLineage)
		r.Get("/connections", h.GetTenantConnection)
		r.Post("/connections", h.SaveTenantConnection)
		r.Post("/execute", h.ExecuteEndpoint)
		r.Get("/audit", h.ListDispatchAudit)
		r.Post("/ingest-openapi", h.IngestOpenAPI)
	})
}

// IngestOpenAPI handles POST /api-dispatcher/ingest-openapi.
func (h *ApiDispatcherHandler) IngestOpenAPI(w http.ResponseWriter, r *http.Request) {
	var req OpenAPIIngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Spec) == 0 && strings.TrimSpace(req.URL) == "" {
		http.Error(w, "either 'spec' or 'url' is required", http.StatusBadRequest)
		return
	}
	if req.TenantID == "" {
		req.TenantID = r.URL.Query().Get("tenant_id")
	}
	if req.TenantID == "" {
		req.TenantID = r.Header.Get("X-Tenant-ID")
	}

	result, err := h.IngestOpenAPISpec(r.Context(), req)
	if err != nil {
		http.Error(w, "OpenAPI ingestion failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    result,
	})
}

type ApiEndpointExecutionRequest struct {
	EndpointNodeID  string                 `json:"endpoint_node_id"`
	TenantID        string                 `json:"tenant_id,omitempty"`
	BaseURLOverride string                 `json:"base_url_override,omitempty"`
	PathParams      map[string]string      `json:"path_params,omitempty"`
	QueryParams     map[string]string      `json:"query_params,omitempty"`
	Headers         map[string]string      `json:"headers,omitempty"`
	Body            map[string]interface{} `json:"body,omitempty"`
}

type ApiEndpointExecutionResponse struct {
	Success      bool                     `json:"success"`
	StatusCode   int                      `json:"status_code"`
	EndpointPath string                   `json:"endpoint_path"`
	Method       string                   `json:"method"`
	DurationMs   int64                    `json:"duration_ms"`
	Records      []map[string]interface{} `json:"records"`
	RawResponse  interface{}              `json:"raw_response,omitempty"`
	Error        string                   `json:"error,omitempty"`
}

type SaveTenantConnectionRequest struct {
	TenantID        string                 `json:"tenant_id"`
	ApiDatasourceID string                 `json:"api_datasource_id"`
	BaseURL         string                 `json:"base_url"`
	AuthType        string                 `json:"auth_type"` // 'oauth2_bearer', 'basic_auth', 'api_key', 'none'
	AuthConfig      map[string]interface{} `json:"auth_config"`
	OAuthClientID   string                 `json:"oauth_client_id,omitempty"`
	OAuthSecret     string                 `json:"oauth_client_secret,omitempty"`
	OAuthRefresh    string                 `json:"oauth_refresh_token,omitempty"`
	OAuthTokenURL   string                 `json:"oauth_token_url,omitempty"`
	OAuthScopes     string                 `json:"oauth_scopes,omitempty"`
	IsActive        bool                   `json:"is_active"`
}

type MapTermRequest struct {
	FieldID        string `json:"field_id"`
	SemanticTermID string `json:"semantic_term_id"` // Empty string to unmap
	TenantID       string `json:"tenant_id,omitempty"`
}

type CreateFieldRequest struct {
	EndpointID     string `json:"endpoint_id"`
	NodeName       string `json:"node_name"`
	DataType       string `json:"data_type"`
	JSONPath       string `json:"json_path"`
	IsPrimaryKey   bool   `json:"is_primary_key"`
	Description    string `json:"description"`
	SemanticTermID string `json:"semantic_term_id,omitempty"`
	TenantID       string `json:"tenant_id,omitempty"`
}

// ListApiDatasources lists all inventoried API datasources
func (h *ApiDispatcherHandler) ListApiDatasources(w http.ResponseWriter, r *http.Request) {
	// getSecureTenantID (helpers.go) validates the X-Tenant-ID header /
	// tenant_id query param against the caller's JWT-issued tenant list /
	// global-admin status before trusting it; it never trusts either raw
	// client-supplied value directly.
	tenantID := getSecureTenantID(r)

	query := `
		SELECT cn.id, cn.node_name, cn.qualified_path, cn.description, 
		       COALESCE(cn.config, '{}'::jsonb) as config, 
		       COALESCE(cn.properties, '{}'::jsonb) as properties,
		       cn.tenant_id, cn.is_active,
		       COALESCE(tac.base_url, '') as tenant_base_url,
		       COALESCE(tac.auth_type, '') as tenant_auth_type
		FROM catalog_node cn
		JOIN catalog_node_types cnt ON cn.node_type_id = cnt.id
		LEFT JOIN tenant_api_connections tac ON (tac.api_datasource_id = cn.id AND tac.tenant_id = NULLIF($1, '')::uuid)
		WHERE cnt.catalog_type_name = 'api_datasource'
		  AND (cn.tenant_id = NULLIF($1, '')::uuid OR cn.tenant_id IN (SELECT id FROM tenants WHERE gold_copy = true))
		ORDER BY cn.node_name ASC
	`
	rows, err := h.db.QueryContext(r.Context(), query, tenantID)
	if err != nil {
		http.Error(w, "Failed to query API datasources: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id, nodeName, qualPath, desc string
		var configBytes, propsBytes []byte
		var tID string
		var isActive bool
		var tenantBaseURL, tenantAuthType string

		if err := rows.Scan(&id, &nodeName, &qualPath, &desc, &configBytes, &propsBytes, &tID, &isActive, &tenantBaseURL, &tenantAuthType); err != nil {
			continue
		}

		var config, props map[string]interface{}
		_ = json.Unmarshal(configBytes, &config)
		_ = json.Unmarshal(propsBytes, &props)

		results = append(results, map[string]interface{}{
			"id":               id,
			"node_name":        nodeName,
			"qualified_path":   qualPath,
			"description":      desc,
			"config":           config,
			"properties":       props,
			"tenant_id":        tID,
			"is_active":        isActive,
			"tenant_base_url":  tenantBaseURL,
			"tenant_auth_type": tenantAuthType,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    results,
	})
}

// ListApiEndpoints lists all endpoints with their parent API, resource, and counts
func (h *ApiDispatcherHandler) ListApiEndpoints(w http.ResponseWriter, r *http.Request) {
	// getSecureTenantID (helpers.go) validates the X-Tenant-ID header /
	// tenant_id query param against the caller's JWT-issued tenant list /
	// global-admin status before trusting it; it never trusts either raw
	// client-supplied value directly.
	tenantID := getSecureTenantID(r)

	query := `
		SELECT ep.id, ep.node_name, ep.qualified_path, ep.description,
		       COALESCE(ep.config, '{}'::jsonb) as config,
		       COALESCE(ep.properties, '{}'::jsonb) as properties,
		       ep.parent_id,
		       COALESCE(res.node_name, '') as resource_name,
		       COALESCE(ds.id::text, '') as datasource_id,
		       COALESCE(ds.node_name, 'API Service') as datasource_name,
		       (SELECT count(*) FROM catalog_node f WHERE f.parent_id = ep.id) as fields_count,
		       (SELECT count(DISTINCT ce.id) 
		        FROM catalog_node f 
		        JOIN catalog_edge ce ON (ce.source_node_id = f.id OR ce.target_node_id = f.id)
		        WHERE f.parent_id = ep.id AND (ce.edge_type_id = '0434ca1a-6543-42d3-9fce-f0b58b5fba34' OR ce.relationship_type = 'has_context')) as semantic_terms_count
		FROM catalog_node ep
		JOIN catalog_node_types cnt ON ep.node_type_id = cnt.id
		LEFT JOIN catalog_node res ON ep.parent_id = res.id
		LEFT JOIN catalog_node ds ON (res.parent_id = ds.id OR ep.parent_id = ds.id)
		WHERE cnt.catalog_type_name = 'api_endpoint'
		  AND (ep.tenant_id = NULLIF($1, '')::uuid OR ep.tenant_id IN (SELECT id FROM tenants WHERE gold_copy = true))
		ORDER BY ep.node_name ASC
	`
	rows, err := h.db.QueryContext(r.Context(), query, tenantID)
	if err != nil {
		http.Error(w, "Failed to query API endpoints: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id, nodeName, qualPath, desc string
		var configBytes, propsBytes []byte
		var parentID sql.NullString
		var resName, dsID, dsName string
		var fieldsCount, termsCount int

		if err := rows.Scan(&id, &nodeName, &qualPath, &desc, &configBytes, &propsBytes, &parentID, &resName, &dsID, &dsName, &fieldsCount, &termsCount); err != nil {
			continue
		}

		var config, props map[string]interface{}
		_ = json.Unmarshal(configBytes, &config)
		_ = json.Unmarshal(propsBytes, &props)

		results = append(results, map[string]interface{}{
			"id":                   id,
			"node_name":            nodeName,
			"qualified_path":       qualPath,
			"description":          desc,
			"config":               config,
			"properties":           props,
			"parent_id":            parentID.String,
			"resource_name":        resName,
			"datasource_id":        dsID,
			"datasource_name":      dsName,
			"fields_count":         fieldsCount,
			"semantic_terms_count": termsCount,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    results,
	})
}

// GetApiEndpointDetail returns detailed metadata for a single endpoint
func (h *ApiDispatcherHandler) GetApiEndpointDetail(w http.ResponseWriter, r *http.Request) {
	endpointID := chi.URLParam(r, "id")
	// getSecureTenantID (helpers.go) validates the X-Tenant-ID header /
	// tenant_id query param against the caller's JWT-issued tenant list /
	// global-admin status before trusting it; it never trusts either raw
	// client-supplied value directly.
	tenantID := getSecureTenantID(r)

	query := `
		SELECT ep.id, ep.node_name, ep.qualified_path, ep.description,
		       COALESCE(ep.config, '{}'::jsonb) as config,
		       COALESCE(ep.properties, '{}'::jsonb) as properties,
		       ep.parent_id,
		       COALESCE(res.node_name, '') as resource_name,
		       COALESCE(ds.id::text, '') as datasource_id,
		       COALESCE(ds.node_name, 'API Service') as datasource_name,
		       COALESCE(ds.description, '') as datasource_description,
		       COALESCE(ds.config, '{}'::jsonb) as datasource_config,
		       COALESCE(tac.base_url, '') as tenant_base_url,
		       COALESCE(tac.auth_type, '') as tenant_auth_type
		FROM catalog_node ep
		LEFT JOIN catalog_node res ON ep.parent_id = res.id
		LEFT JOIN catalog_node ds ON (res.parent_id = ds.id OR ep.parent_id = ds.id)
		LEFT JOIN tenant_api_connections tac ON (tac.api_datasource_id = ds.id AND tac.tenant_id = NULLIF($2, '')::uuid)
		WHERE ep.id = $1::uuid
	`
	var id, nodeName, qualPath, desc string
	var configBytes, propsBytes, dsConfigBytes []byte
	var parentID sql.NullString
	var resName, dsID, dsName, dsDesc string
	var tenantBaseURL, tenantAuthType string

	err := h.db.QueryRowContext(r.Context(), query, endpointID, tenantID).Scan(
		&id, &nodeName, &qualPath, &desc, &configBytes, &propsBytes, &parentID,
		&resName, &dsID, &dsName, &dsDesc, &dsConfigBytes, &tenantBaseURL, &tenantAuthType,
	)
	if err != nil {
		http.Error(w, "Endpoint not found: "+err.Error(), http.StatusNotFound)
		return
	}

	var config, props, dsConfig map[string]interface{}
	_ = json.Unmarshal(configBytes, &config)
	_ = json.Unmarshal(propsBytes, &props)
	_ = json.Unmarshal(dsConfigBytes, &dsConfig)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"id":                     id,
			"node_name":               nodeName,
			"qualified_path":          qualPath,
			"description":             desc,
			"config":                  config,
			"properties":              props,
			"parent_id":               parentID.String,
			"resource_name":           resName,
			"datasource_id":           dsID,
			"datasource_name":         dsName,
			"datasource_description":  dsDesc,
			"datasource_config":       dsConfig,
			"tenant_base_url":         tenantBaseURL,
			"tenant_auth_type":        tenantAuthType,
		},
	})
}

// ListApiFields returns all api_field payload nodes belonging to an endpoint
func (h *ApiDispatcherHandler) ListApiFields(w http.ResponseWriter, r *http.Request) {
	endpointID := r.URL.Query().Get("endpoint_id")
	if endpointID == "" {
		http.Error(w, "endpoint_id is required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT f.id, f.node_name, f.qualified_path, f.description,
		       COALESCE(f.config, '{}'::jsonb) as config,
		       COALESCE(f.properties, '{}'::jsonb) as properties,
		       COALESCE(st.node_name, '') as mapped_semantic_term_name,
		       COALESCE(st.id::text, '') as mapped_semantic_term_id,
		       COALESCE(st.description, '') as mapped_semantic_term_desc,
		       COALESCE(ce.id::text, '') as edge_id
		FROM catalog_node f
		JOIN catalog_node_types cnt ON f.node_type_id = cnt.id
		LEFT JOIN catalog_edge ce ON (
			(ce.source_node_id = f.id AND (ce.edge_type_id = '0434ca1a-6543-42d3-9fce-f0b58b5fba34' OR ce.relationship_type = 'has_context'))
			OR
			(ce.target_node_id = f.id AND (ce.edge_type_id = '0434ca1a-6543-42d3-9fce-f0b58b5fba34' OR ce.relationship_type = 'has_context'))
		)
		LEFT JOIN catalog_node st ON (
			(ce.source_node_id = st.id AND st.id != f.id)
			OR
			(ce.target_node_id = st.id AND st.id != f.id)
		)
		WHERE cnt.catalog_type_name = 'api_field'
		  AND f.parent_id = $1::uuid
		ORDER BY f.node_name ASC
	`
	rows, err := h.db.QueryContext(r.Context(), query, endpointID)
	if err != nil {
		http.Error(w, "Failed to query API fields: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var fields []map[string]interface{}
	for rows.Next() {
		var id, nodeName, qualPath, desc string
		var configBytes, propsBytes []byte
		var semName, semID, semDesc, edgeID string

		if err := rows.Scan(&id, &nodeName, &qualPath, &desc, &configBytes, &propsBytes, &semName, &semID, &semDesc, &edgeID); err != nil {
			continue
		}

		var config, props map[string]interface{}
		_ = json.Unmarshal(configBytes, &config)
		_ = json.Unmarshal(propsBytes, &props)

		fields = append(fields, map[string]interface{}{
			"id":                        id,
			"node_name":                 nodeName,
			"qualified_path":            qualPath,
			"description":               desc,
			"config":                    config,
			"properties":                props,
			"mapped_semantic_term_name": semName,
			"mapped_semantic_term_id":   semID,
			"mapped_semantic_term_desc": semDesc,
			"edge_id":                   edgeID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    fields,
	})
}

// ListSemanticTerms lists searchable semantic terms for mapping
func (h *ApiDispatcherHandler) ListSemanticTerms(w http.ResponseWriter, r *http.Request) {
	// getSecureTenantID (helpers.go) validates the X-Tenant-ID header /
	// tenant_id query param against the caller's JWT-issued tenant list /
	// global-admin status before trusting it; it never trusts either raw
	// client-supplied value directly.
	tenantID := getSecureTenantID(r)
	search := r.URL.Query().Get("q")

	query := `
		SELECT cn.id, cn.node_name, cn.description, COALESCE(cn.properties->>'data_type', 'varchar') as data_type
		FROM catalog_node cn
		JOIN catalog_node_types cnt ON cn.node_type_id = cnt.id
		WHERE cnt.catalog_type_name = 'semantic_term'
		  AND (cn.tenant_id = NULLIF($1, '')::uuid OR cn.tenant_id IN (SELECT id FROM tenants WHERE gold_copy = true))
		  AND ($2 = '' OR cn.node_name ILIKE '%' || $2 || '%' OR cn.description ILIKE '%' || $2 || '%')
		ORDER BY cn.node_name ASC
		LIMIT 100
	`
	rows, err := h.db.QueryContext(r.Context(), query, tenantID, search)
	if err != nil {
		http.Error(w, "Failed to query semantic terms: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var terms []map[string]interface{}
	for rows.Next() {
		var id, nodeName, desc, dType string
		if err := rows.Scan(&id, &nodeName, &desc, &dType); err != nil {
			continue
		}
		terms = append(terms, map[string]interface{}{
			"id":          id,
			"node_name":   nodeName,
			"description": desc,
			"data_type":   dType,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    terms,
	})
}

// MapFieldToSemanticTerm links or unlinks an api_field to a semantic_term
func (h *ApiDispatcherHandler) MapFieldToSemanticTerm(w http.ResponseWriter, r *http.Request) {
	var req MapTermRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.FieldID == "" {
		http.Error(w, "field_id is required", http.StatusBadRequest)
		return
	}

	tenantID := req.TenantID
	if tenantID == "" {
		tenantID = "99e99e99-99e9-49e9-89e9-99e99e99e999" // Default to Gold Copy if unspecified
	}

	// 1. Remove existing has_context edge for this field
	deleteQuery := `
		DELETE FROM catalog_edge
		WHERE (source_node_id = $1::uuid OR target_node_id = $1::uuid)
		  AND (relationship_type = 'has_context' OR edge_type_id = '0434ca1a-6543-42d3-9fce-f0b58b5fba34')
	`
	_, _ = h.db.ExecContext(r.Context(), deleteQuery, req.FieldID)

	// 2. If semantic_term_id is provided, create new has_context edge
	if req.SemanticTermID != "" {
		edgeID := uuid.New().String()
		insertQuery := `
			INSERT INTO catalog_edge (id, source_node_id, target_node_id, edge_type_id, relationship_type, tenant_id, properties, is_active, created_at, updated_at)
			VALUES ($1::uuid, $2::uuid, $3::uuid, '0434ca1a-6543-42d3-9fce-f0b58b5fba34', 'has_context', $4::uuid, '{}'::jsonb, true, NOW(), NOW())
		`
		_, err := h.db.ExecContext(r.Context(), insertQuery, edgeID, req.FieldID, req.SemanticTermID, tenantID)
		if err != nil {
			http.Error(w, "Failed to map semantic term edge: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Semantic term mapping updated successfully",
	})
}

// CreateApiField creates a new payload/response field node for an endpoint
func (h *ApiDispatcherHandler) CreateApiField(w http.ResponseWriter, r *http.Request) {
	var req CreateFieldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.EndpointID == "" || req.NodeName == "" {
		http.Error(w, "endpoint_id and node_name are required", http.StatusBadRequest)
		return
	}

	tenantID := req.TenantID
	if tenantID == "" {
		tenantID = "99e99e99-99e9-49e9-89e9-99e99e99e999"
	}
	if req.DataType == "" {
		req.DataType = "varchar"
	}
	if req.JSONPath == "" {
		req.JSONPath = "$." + req.NodeName
	}

	fieldID := uuid.New().String()
	qualPath := "/api/endpoint/" + req.EndpointID + "/" + req.NodeName

	configJSON, _ := json.Marshal(map[string]interface{}{
		"json_path": req.JSONPath,
	})
	propsJSON, _ := json.Marshal(map[string]interface{}{
		"data_type":      req.DataType,
		"is_primary_key": req.IsPrimaryKey,
	})

	// 1. Insert catalog_node
	insertNodeQuery := `
		INSERT INTO catalog_node (id, tenant_id, node_type_id, parent_id, node_name, qualified_path, description, config, properties, is_active)
		VALUES ($1::uuid, $2::uuid, '9657a779-2e69-408f-a374-f3672bb27abf', $3::uuid, $4, $5, $6, $7::jsonb, $8::jsonb, true)
	`
	_, err := h.db.ExecContext(r.Context(), insertNodeQuery, fieldID, tenantID, req.EndpointID, req.NodeName, qualPath, req.Description, string(configJSON), string(propsJSON))
	if err != nil {
		http.Error(w, "Failed to create field node: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Insert contains_field edge
	edgeID := uuid.New().String()
	insertEdgeQuery := `
		INSERT INTO catalog_edge (id, source_node_id, target_node_id, edge_type_id, relationship_type, tenant_id, properties, is_active, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, '656bab37-02c1-4fc9-b276-5e2b88078619', 'contains_field', $4::uuid, '{}'::jsonb, true, NOW(), NOW())
	`
	_, _ = h.db.ExecContext(r.Context(), insertEdgeQuery, edgeID, req.EndpointID, fieldID, tenantID)

	// 3. If semantic_term_id provided, map it immediately
	if req.SemanticTermID != "" {
		stEdgeID := uuid.New().String()
		insertStEdgeQuery := `
			INSERT INTO catalog_edge (id, source_node_id, target_node_id, edge_type_id, relationship_type, tenant_id, properties, is_active, created_at, updated_at)
			VALUES ($1::uuid, $2::uuid, $3::uuid, '0434ca1a-6543-42d3-9fce-f0b58b5fba34', 'has_context', $4::uuid, '{}'::jsonb, true, NOW(), NOW())
		`
		_, _ = h.db.ExecContext(r.Context(), insertStEdgeQuery, stEdgeID, fieldID, req.SemanticTermID, tenantID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"field_id": fieldID,
		"message":  "API field created successfully",
	})
}

// DeleteApiField deletes a payload field node
func (h *ApiDispatcherHandler) DeleteApiField(w http.ResponseWriter, r *http.Request) {
	fieldID := chi.URLParam(r, "id")
	if fieldID == "" {
		http.Error(w, "field ID is required", http.StatusBadRequest)
		return
	}

	_, _ = h.db.ExecContext(r.Context(), `DELETE FROM catalog_edge WHERE source_node_id = $1::uuid OR target_node_id = $1::uuid`, fieldID)
	_, err := h.db.ExecContext(r.Context(), `DELETE FROM catalog_node WHERE id = $1::uuid`, fieldID)
	if err != nil {
		http.Error(w, "Failed to delete field: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Field deleted successfully",
	})
}

// GetApiLineage returns ReactFlow-compatible nodes and edges for an API endpoint
func (h *ApiDispatcherHandler) GetApiLineage(w http.ResponseWriter, r *http.Request) {
	endpointID := r.URL.Query().Get("endpoint_id")
	if endpointID == "" {
		http.Error(w, "endpoint_id is required", http.StatusBadRequest)
		return
	}

	// 1. Fetch endpoint and its parent service
	epQuery := `
		SELECT ep.id, ep.node_name, ep.qualified_path,
		       COALESCE(ep.config, '{}'::jsonb),
		       COALESCE(ds.node_name, 'API Service')
		FROM catalog_node ep
		LEFT JOIN catalog_node res ON ep.parent_id = res.id
		LEFT JOIN catalog_node ds ON (res.parent_id = ds.id OR ep.parent_id = ds.id)
		WHERE ep.id = $1::uuid
	`
	var epID, epName, qualPath, dsName string
	var epConfigBytes []byte
	err := h.db.QueryRowContext(r.Context(), epQuery, endpointID).Scan(&epID, &epName, &qualPath, &epConfigBytes, &dsName)
	if err != nil {
		http.Error(w, "Endpoint not found", http.StatusNotFound)
		return
	}
	var epConfig map[string]interface{}
	_ = json.Unmarshal(epConfigBytes, &epConfig)
	method := "GET"
	if m, ok := epConfig["method"].(string); ok && m != "" {
		method = strings.ToUpper(m)
	}

	// 2. Fetch all payload fields, mapped semantic terms, and upstream business terms
	lineageQuery := `
		SELECT f.id as field_id, f.node_name as field_name, COALESCE(f.properties->>'data_type', 'varchar') as data_type,
		       COALESCE(st.id::text, '') as sem_id, COALESCE(st.node_name, '') as sem_name,
		       COALESCE(bt.id::text, '') as bus_id, COALESCE(bt.node_name, '') as bus_name
		FROM catalog_node f
		JOIN catalog_node_types cnt ON f.node_type_id = cnt.id
		LEFT JOIN catalog_edge ce ON (
			(ce.source_node_id = f.id AND (ce.edge_type_id = '0434ca1a-6543-42d3-9fce-f0b58b5fba34' OR ce.relationship_type = 'has_context'))
			OR
			(ce.target_node_id = f.id AND (ce.edge_type_id = '0434ca1a-6543-42d3-9fce-f0b58b5fba34' OR ce.relationship_type = 'has_context'))
		)
		LEFT JOIN catalog_node st ON (
			(ce.source_node_id = st.id AND st.id != f.id)
			OR
			(ce.target_node_id = st.id AND st.id != f.id)
		)
		LEFT JOIN catalog_edge ce_bt ON (
			(ce_bt.target_node_id = st.id OR ce_bt.source_node_id = st.id)
			AND ce_bt.relationship_type = 'maps_to'
		)
		LEFT JOIN catalog_node bt ON (
			(ce_bt.source_node_id = bt.id AND bt.id != st.id)
			OR
			(ce_bt.target_node_id = bt.id AND bt.id != st.id)
		)
		WHERE cnt.catalog_type_name = 'api_field'
		  AND f.parent_id = $1::uuid
		ORDER BY f.node_name ASC
	`
	rows, err := h.db.QueryContext(r.Context(), lineageQuery, endpointID)
	if err != nil {
		http.Error(w, "Failed to query lineage: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	nodes := []map[string]interface{}{}
	edges := []map[string]interface{}{}
	nodeSet := make(map[string]bool)

	// API Endpoint Node
	epNodeID := "ep-" + epID
	nodes = append(nodes, map[string]interface{}{
		"id": epNodeID,
		"data": map[string]interface{}{
			"label":    epName,
			"type":     "api_endpoint",
			"method":   method,
			"service":  dsName,
			"path":     qualPath,
			"icon":     "⚡",
			"category": "API Operation",
		},
		"type": "default",
	})
	nodeSet[epNodeID] = true

	for rows.Next() {
		var fieldID, fieldName, dataType, semID, semName, busID, busName string
		if err := rows.Scan(&fieldID, &fieldName, &dataType, &semID, &semName, &busID, &busName); err != nil {
			continue
		}

		// Field Node
		fNodeID := "f-" + fieldID
		if !nodeSet[fNodeID] {
			nodes = append(nodes, map[string]interface{}{
				"id": fNodeID,
				"data": map[string]interface{}{
					"label":    fieldName,
					"type":     "api_field",
					"dataType": dataType,
					"icon":     "🔹",
					"category": "Payload Attribute",
				},
				"type": "default",
			})
			nodeSet[fNodeID] = true

			// Edge: Endpoint -> Field
			edges = append(edges, map[string]interface{}{
				"id":     fmt.Sprintf("e-%s-%s", epNodeID, fNodeID),
				"source": epNodeID,
				"target": fNodeID,
				"label":  "contains_field",
			})
		}

		// Semantic Term Node
		if semID != "" {
			sNodeID := "sem-" + semID
			if !nodeSet[sNodeID] {
				nodes = append(nodes, map[string]interface{}{
					"id": sNodeID,
					"data": map[string]interface{}{
						"label":    semName,
						"type":     "semantic_term",
						"icon":     "🧠",
						"category": "Semantic Concept",
					},
					"type": "default",
				})
				nodeSet[sNodeID] = true
			}

			// Edge: Semantic Term -> Field
			edgeID := fmt.Sprintf("e-%s-%s", sNodeID, fNodeID)
			edges = append(edges, map[string]interface{}{
				"id":     edgeID,
				"source": sNodeID,
				"target": fNodeID,
				"label":  "has_context",
			})

			// Business Term Node
			if busID != "" {
				bNodeID := "bus-" + busID
				if !nodeSet[bNodeID] {
					nodes = append(nodes, map[string]interface{}{
						"id": bNodeID,
						"data": map[string]interface{}{
							"label":    busName,
							"type":     "business_term",
							"icon":     "💼",
							"category": "Business Glossary",
						},
						"type": "default",
					})
					nodeSet[bNodeID] = true
				}

				// Edge: Business Term -> Semantic Term
				btEdgeID := fmt.Sprintf("e-%s-%s", bNodeID, sNodeID)
				edges = append(edges, map[string]interface{}{
					"id":     btEdgeID,
					"source": bNodeID,
					"target": sNodeID,
					"label":  "maps_to",
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"nodes":   nodes,
		"edges":   edges,
	})
}

// GetTenantConnection returns the tenant's instance URL and credentials
func (h *ApiDispatcherHandler) GetTenantConnection(w http.ResponseWriter, r *http.Request) {
	// getSecureTenantID (helpers.go) validates the X-Tenant-ID header /
	// tenant_id query param against the caller's JWT-issued tenant list /
	// global-admin status before trusting it; it never trusts either raw
	// client-supplied value directly.
	tenantID := getSecureTenantID(r)
	dsID := r.URL.Query().Get("api_datasource_id")
	if tenantID == "" || dsID == "" {
		http.Error(w, "tenant_id and api_datasource_id are required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT id, base_url, auth_type, auth_config_encrypted, is_active,
		       COALESCE(oauth_client_id, ''),
		       oauth_client_secret_encrypted,
		       oauth_refresh_token_encrypted,
		       COALESCE(oauth_token_url, ''),
		       COALESCE(oauth_scopes, '')
		FROM tenant_api_connections
		WHERE tenant_id = $1::uuid AND api_datasource_id = $2::uuid
		LIMIT 1
	`
	var id, baseURL, authType string
	var authConfigEncrypted []byte
	var isActive bool
	var oauthClientID, oauthTokenURL, oauthScopes string
	var oauthClientSecretBytes, oauthRefreshBytes []byte

	err := h.db.QueryRowContext(r.Context(), query, tenantID, dsID).Scan(
		&id, &baseURL, &authType, &authConfigEncrypted, &isActive,
		&oauthClientID, &oauthClientSecretBytes, &oauthRefreshBytes,
		&oauthTokenURL, &oauthScopes,
	)
	if err == sql.ErrNoRows {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    nil,
		})
		return
	} else if err != nil {
		http.Error(w, "Failed to query tenant API connection: "+err.Error(), http.StatusInternalServerError)
		return
	}

	authConfig, decryptErr := h.decryptAuthConfig(authConfigEncrypted)
	if decryptErr != nil {
		http.Error(w, "Failed to decrypt tenant API credentials: "+decryptErr.Error(), http.StatusInternalServerError)
		return
	}

	// OAuth metadata (non-secret values + booleans indicating which secrets are present).
	oauth := map[string]interface{}{
		"client_id":            oauthClientID,
		"token_url":            oauthTokenURL,
		"scopes":               oauthScopes,
		"has_client_secret":    len(oauthClientSecretBytes) > 0,
		"has_refresh_token":    len(oauthRefreshBytes) > 0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"id":                id,
			"tenant_id":         tenantID,
			"api_datasource_id": dsID,
			"base_url":          baseURL,
			"auth_type":         authType,
			"auth_config":       authConfig,
			"is_active":         isActive,
			"oauth":             oauth,
		},
	})
}

// SaveTenantConnection creates or updates the tenant's instance URL and credentials
func (h *ApiDispatcherHandler) SaveTenantConnection(w http.ResponseWriter, r *http.Request) {
	var req SaveTenantConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.TenantID == "" || req.ApiDatasourceID == "" || req.BaseURL == "" {
		http.Error(w, "tenant_id, api_datasource_id, and base_url are required", http.StatusBadRequest)
		return
	}
	if req.AuthType == "" {
		req.AuthType = "oauth2_bearer"
	}

	authConfigEncrypted, err := h.encryptAuthConfig(req.AuthConfig)
	if err != nil {
		http.Error(w, "Failed to encrypt tenant API credentials: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Encrypt OAuth long-lived secrets (client_secret, refresh_token). The
	// remaining OAuth fields (client_id, token_url, scopes) are stored in
	// cleartext alongside the connection row.
	var oauthSecretEncrypted, oauthRefreshEncrypted []byte
	if req.OAuthSecret != "" {
		if h.encryptor == nil {
			http.Error(w, "Cannot encrypt OAuth client_secret without API_TOKEN_ENCRYPTION_KEY", http.StatusInternalServerError)
			return
		}
		s, err := h.encryptor.Encrypt(req.OAuthSecret)
		if err != nil {
			http.Error(w, "Failed to encrypt OAuth client_secret: "+err.Error(), http.StatusInternalServerError)
			return
		}
		oauthSecretEncrypted = []byte(s)
	}
	if req.OAuthRefresh != "" {
		if h.encryptor == nil {
			http.Error(w, "Cannot encrypt OAuth refresh_token without API_TOKEN_ENCRYPTION_KEY", http.StatusInternalServerError)
			return
		}
		s, err := h.encryptor.Encrypt(req.OAuthRefresh)
		if err != nil {
			http.Error(w, "Failed to encrypt OAuth refresh_token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		oauthRefreshEncrypted = []byte(s)
	}

	query := `
		INSERT INTO tenant_api_connections (
			tenant_id, api_datasource_id, base_url, auth_type,
			auth_config_encrypted, is_active, updated_at,
			oauth_client_id, oauth_client_secret_encrypted, oauth_refresh_token_encrypted,
			oauth_token_url, oauth_scopes
		)
		VALUES (
			$1::uuid, $2::uuid, $3, $4, $5, $6, NOW(),
			NULLIF($7, ''), $8, $9, NULLIF($10, ''), NULLIF($11, '')
		)
		ON CONFLICT (tenant_id, api_datasource_id) DO UPDATE SET
			base_url = EXCLUDED.base_url,
			auth_type = EXCLUDED.auth_type,
			auth_config_encrypted = EXCLUDED.auth_config_encrypted,
			is_active = EXCLUDED.is_active,
			updated_at = NOW(),
			oauth_client_id = EXCLUDED.oauth_client_id,
			oauth_client_secret_encrypted = COALESCE(EXCLUDED.oauth_client_secret_encrypted, tenant_api_connections.oauth_client_secret_encrypted),
			oauth_refresh_token_encrypted = COALESCE(EXCLUDED.oauth_refresh_token_encrypted, tenant_api_connections.oauth_refresh_token_encrypted),
			oauth_token_url = EXCLUDED.oauth_token_url,
			oauth_scopes = EXCLUDED.oauth_scopes
		RETURNING id
	`
	var connectionID string
	err = h.db.QueryRowContext(r.Context(), query,
		req.TenantID, req.ApiDatasourceID, req.BaseURL, req.AuthType,
		authConfigEncrypted, req.IsActive,
		req.OAuthClientID, oauthSecretEncrypted, oauthRefreshEncrypted,
		req.OAuthTokenURL, req.OAuthScopes,
	).Scan(&connectionID)
	if err != nil {
		http.Error(w, "Failed to save tenant API connection: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Note: We deliberately do NOT proactively invalidate the cached access
	// token when OAuth secrets rotate. The next dispatch will either find the
	// cached token still valid (request succeeds), or it will fail server-side
	// validation (dispatcher retries with a fresh refresh using the new
	// credentials) — both paths converge on the correct outcome without
	// forcing cache cleanup here.

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      connectionID,
		"message": "Tenant API instance connection saved successfully",
	})
}

// ExecuteEndpoint executes a parameterized API request using tenant-scoped credentials
func (h *ApiDispatcherHandler) ExecuteEndpoint(w http.ResponseWriter, r *http.Request) {
	var req ApiEndpointExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.EndpointNodeID == "" {
		http.Error(w, "endpoint_node_id is required", http.StatusBadRequest)
		return
	}

	// getSecureTenantID (helpers.go) validates the X-Tenant-ID header /
	// tenant_id query param against the caller's JWT-issued tenant list /
	// global-admin status before trusting it. req.TenantID (request body) is
	// client-supplied too, so it is only honored as a fallback for verified
	// global admins/ops — never trusted for a regular tenant-scoped caller.
	tenantID := getSecureTenantID(r)
	if tenantID == "" && req.TenantID != "" {
		if auth, ok := security.AuthInfoFromContext(r.Context()); ok && auth.IsGlobalAdmin {
			tenantID = req.TenantID
		}
	}

	// 1. Fetch endpoint node, parent resource, and parent datasource
	query := `
		SELECT ep.node_name, COALESCE(ep.config, '{}'::jsonb),
		       COALESCE(ds.id::text, ''),
		       COALESCE(ds.config, '{}'::jsonb),
		       COALESCE(ds.node_name, 'API Service')
		FROM catalog_node ep
		LEFT JOIN catalog_node res ON ep.parent_id = res.id
		LEFT JOIN catalog_node ds ON (res.parent_id = ds.id OR ep.parent_id = ds.id)
		WHERE ep.id = $1::uuid
	`
	var epName, dsID, dsName string
	var epConfigBytes, dsConfigBytes []byte

	err := h.db.QueryRowContext(r.Context(), query, req.EndpointNodeID).Scan(&epName, &epConfigBytes, &dsID, &dsConfigBytes, &dsName)
	if err != nil {
		http.Error(w, "Endpoint node not found in catalog: "+err.Error(), http.StatusNotFound)
		return
	}

	var epConfig, dsConfig map[string]interface{}
	_ = json.Unmarshal(epConfigBytes, &epConfig)
	_ = json.Unmarshal(dsConfigBytes, &dsConfig)

	// 2. Resolve Tenant-specific base URL, Auth credentials, and OAuth creds.
	var (
		tenantBaseURL               string
		tenantAuthType              string
		tenantAuthConfigEncrypted   []byte
		oauthClientID, oauthTokenURL, oauthScopes string
		oauthClientSecretEncrypted  []byte
		oauthRefreshEncrypted       []byte
	)

	if tenantID != "" && dsID != "" {
		connQuery := `
			SELECT base_url, auth_type, auth_config_encrypted,
			       COALESCE(oauth_client_id, ''),
			       oauth_client_secret_encrypted,
			       oauth_refresh_token_encrypted,
			       COALESCE(oauth_token_url, ''),
			       COALESCE(oauth_scopes, '')
			FROM tenant_api_connections
			WHERE tenant_id = $1::uuid AND api_datasource_id = $2::uuid AND is_active = true
			LIMIT 1
		`
		err := h.db.QueryRowContext(r.Context(), connQuery, tenantID, dsID).Scan(
			&tenantBaseURL, &tenantAuthType, &tenantAuthConfigEncrypted,
			&oauthClientID, &oauthClientSecretEncrypted, &oauthRefreshEncrypted,
			&oauthTokenURL, &oauthScopes,
		)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, "Failed to load tenant API connection: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Decrypt auth_config (the JSON map of token / username / api_key etc.).
	var tenantAuthConfig map[string]interface{}
	decrypted, decErr := h.decryptAuthConfig(tenantAuthConfigEncrypted)
	if decErr != nil {
		http.Error(w, "Failed to decrypt tenant API credentials: "+decErr.Error(), http.StatusInternalServerError)
		return
	}
	tenantAuthConfig = decrypted

	// Build the OAuth credentials struct (decrypted client_secret + refresh_token)
	// that the OAuth provider will use if it needs to refresh.
	var oauthCreds oauth.TokenCredentials
	if oauthRefreshEncrypted != nil && oauthClientID != "" && oauthTokenURL != "" {
		if h.encryptor == nil {
			http.Error(w, "OAuth credentials stored but API_TOKEN_ENCRYPTION_KEY is not configured", http.StatusInternalServerError)
			return
		}
		secretPlain, err := h.encryptor.Decrypt(string(oauthClientSecretEncrypted))
		if err != nil {
			http.Error(w, "Failed to decrypt OAuth client_secret: "+err.Error(), http.StatusInternalServerError)
			return
		}
		refreshPlain, err := h.encryptor.Decrypt(string(oauthRefreshEncrypted))
		if err != nil {
			http.Error(w, "Failed to decrypt OAuth refresh_token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		oauthCreds = oauth.TokenCredentials{
			ClientID:     oauthClientID,
			ClientSecret: secretPlain,
			RefreshToken: refreshPlain,
			TokenURL:     oauthTokenURL,
			Scopes:       oauthScopes,
		}
	}

	baseURL := ""
	if req.BaseURLOverride != "" {
		baseURL = req.BaseURLOverride
	} else if tenantBaseURL != "" {
		baseURL = tenantBaseURL
	} else if b, ok := dsConfig["default_base_url"].(string); ok {
		baseURL = b
	}

	method := "GET"
	if m, ok := epConfig["method"].(string); ok && m != "" {
		method = strings.ToUpper(m)
	}

	pathTemplate := ""
	if pt, ok := epConfig["path_template"].(string); ok {
		pathTemplate = pt
	}

	// 3. Interpolate path parameters
	targetPath := pathTemplate
	for k, v := range req.PathParams {
		targetPath = strings.ReplaceAll(targetPath, "{"+k+"}", v)
		targetPath = strings.ReplaceAll(targetPath, ":"+k, v)
	}

	// 4. Construct full target URL
	fullURL := baseURL + targetPath
	if len(req.QueryParams) > 0 {
		sep := "?"
		if strings.Contains(fullURL, "?") {
			sep = "&"
		}
		var qp []string
		for k, v := range req.QueryParams {
			qp = append(qp, fmt.Sprintf("%s=%s", k, v))
		}
		fullURL += sep + strings.Join(qp, "&")
	}

	// 5. Build Request Body if POST/PUT/PATCH
	var bodyReader io.Reader
	if req.Body != nil && (method == "POST" || method == "PUT" || method == "PATCH") {
		jsonBody, _ := json.Marshal(req.Body)
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	startTime := time.Now()
	httpReq, err := http.NewRequestWithContext(r.Context(), method, fullURL, bodyReader)
	if err != nil {
		http.Error(w, "Failed to create HTTP request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Default Headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Resolve the access token for OAuth, or apply other auth schemes.
	serviceType := ""
	if st, ok := dsConfig["service_type"].(string); ok {
		serviceType = st
	}
	if err := h.injectAuthHeaders(r.Context(), httpReq, tenantAuthType, tenantAuthConfig, serviceType, tenantID, dsID, oauthCreds); err != nil {
		http.Error(w, "Failed to resolve auth headers: "+err.Error(), http.StatusInternalServerError)
		return
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// 6. Execute HTTP Request
	resp, err := h.httpClient.Do(httpReq)
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ApiEndpointExecutionResponse{
			Success:      false,
			EndpointPath: fullURL,
			Method:       method,
			DurationMs:   duration,
			Error:        "HTTP call failed: " + err.Error(),
		})
		h.recordAudit(auditEntry{
			TenantID:     tenantID,
			DatasourceID: dsID,
			EndpointID:   req.EndpointNodeID,
			Method:       method,
			Path:         fullURL,
			StatusCode:   0,
			DurationMs:   duration,
			Success:      false,
			Error:        "HTTP call failed: " + err.Error(),
		})
		return
	}

	// 6a. OAuth refresh-on-401 retry. If the upstream returned 401 and we
	// have an OAuth refresh_token, force a refresh and retry the request once.
	if resp.StatusCode == 401 && tenantAuthType == "oauth2_bearer" && oauthCreds.RefreshToken != "" && h.oauthProvider != nil {
		resp.Body.Close()
		refreshed, refreshErr := h.oauthProvider.RefreshWithConfig(r.Context(), oauthCreds)
		if refreshErr == nil {
			_ = h.oauthProvider.SaveToken(r.Context(), serviceType, tenantID, dsID, refreshed)
			httpReq2, reqErr := http.NewRequestWithContext(r.Context(), method, fullURL, bodyReader)
			if reqErr == nil {
				httpReq2.Header.Set("Content-Type", "application/json")
				httpReq2.Header.Set("Accept", "application/json")
				httpReq2.Header.Set("Authorization", "Bearer "+refreshed.AccessToken)
				for k, v := range req.Headers {
					httpReq2.Header.Set(k, v)
				}
				startTime2 := time.Now()
				resp2, retryErr := h.httpClient.Do(httpReq2)
				if retryErr == nil {
					defer resp2.Body.Close()
					respBody, _ := io.ReadAll(resp2.Body)
					var parsedJSON interface{}
					_ = json.Unmarshal(respBody, &parsedJSON)
					var records []map[string]interface{}
					if obj, ok := parsedJSON.(map[string]interface{}); ok {
						if recs, ok := obj["records"].([]interface{}); ok {
							for _, r := range recs {
								if m, ok := r.(map[string]interface{}); ok {
									records = append(records, m)
								}
							}
						} else if res, ok := obj["result"].([]interface{}); ok {
							for _, r := range res {
								if m, ok := r.(map[string]interface{}); ok {
									records = append(records, m)
								}
							}
						} else {
							records = append(records, obj)
						}
					} else if arr, ok := parsedJSON.([]interface{}); ok {
						for _, r := range arr {
							if m, ok := r.(map[string]interface{}); ok {
								records = append(records, m)
							}
						}
					}
					retryDuration := time.Since(startTime2).Milliseconds()
					h.recordAudit(auditEntry{
						TenantID:     tenantID,
						DatasourceID: dsID,
						EndpointID:   req.EndpointNodeID,
						Method:       method,
						Path:         fullURL,
						StatusCode:   resp2.StatusCode,
						DurationMs:   duration + retryDuration,
						Success:      resp2.StatusCode >= 200 && resp2.StatusCode < 300,
						RecordCount:  len(records),
						Error:        "",
					})
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(ApiEndpointExecutionResponse{
						Success:      resp2.StatusCode >= 200 && resp2.StatusCode < 300,
						StatusCode:   resp2.StatusCode,
						EndpointPath: fullURL,
						Method:       method,
						DurationMs:   retryDuration,
						Records:      records,
						RawResponse:  parsedJSON,
					})
					return
				}
			}
		}
		// If refresh/retry failed, fall through and return the original 401.
		// Re-issue the original response so the caller sees the auth failure.
		h.recordAudit(auditEntry{
			TenantID:     tenantID,
			DatasourceID: dsID,
			EndpointID:   req.EndpointNodeID,
			Method:       method,
			Path:         fullURL,
			StatusCode:   401,
			DurationMs:   duration,
			Success:      false,
			Error:        "OAuth refresh-on-401 failed; credentials may be revoked",
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ApiEndpointExecutionResponse{
			Success:      false,
			StatusCode:   401,
			EndpointPath: fullURL,
			Method:       method,
			DurationMs:   duration,
			Error:        "OAuth refresh-on-401 failed; credentials may be revoked",
		})
		return
	}

	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var parsedJSON interface{}
	_ = json.Unmarshal(respBody, &parsedJSON)

	// 7. Extract records based on response_root (e.g. $.records or $.result)
	var records []map[string]interface{}
	if obj, ok := parsedJSON.(map[string]interface{}); ok {
		if recs, ok := obj["records"].([]interface{}); ok {
			for _, r := range recs {
				if m, ok := r.(map[string]interface{}); ok {
					records = append(records, m)
				}
			}
		} else if res, ok := obj["result"].([]interface{}); ok {
			for _, r := range res {
				if m, ok := r.(map[string]interface{}); ok {
					records = append(records, m)
				}
			}
		} else {
			records = append(records, obj)
		}
	} else if arr, ok := parsedJSON.([]interface{}); ok {
		for _, r := range arr {
			if m, ok := r.(map[string]interface{}); ok {
				records = append(records, m)
			}
		}
	}

	h.recordAudit(auditEntry{
		TenantID:     tenantID,
		DatasourceID: dsID,
		EndpointID:   req.EndpointNodeID,
		Method:       method,
		Path:         fullURL,
		StatusCode:   resp.StatusCode,
		DurationMs:   duration,
		Success:      resp.StatusCode >= 200 && resp.StatusCode < 300,
		RecordCount:  len(records),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ApiEndpointExecutionResponse{
		Success:      resp.StatusCode >= 200 && resp.StatusCode < 300,
		StatusCode:   resp.StatusCode,
		EndpointPath: fullURL,
		Method:       method,
		DurationMs:   duration,
		Records:      records,
		RawResponse:  parsedJSON,
	})
}

// injectAuthHeaders applies the correct authorization scheme to httpReq:
//   - oauth2_bearer: try Redis cache, fall back to refresh, fall back to a
//     static bearer token pasted in the connection form.
//   - api_key:       set the configured header (default X-API-Key).
//   - basic_auth:    Authorization: Basic base64(user:pass).
//
// tenantAuthConfig and oauthCreds are decrypted inputs the caller already
// loaded from the database.
func (h *ApiDispatcherHandler) injectAuthHeaders(
	ctx context.Context,
	httpReq *http.Request,
	authType string,
	tenantAuthConfig map[string]interface{},
	serviceType, tenantID, datasourceID string,
	oauthCreds oauth.TokenCredentials,
) error {
	switch authType {
	case "oauth2_bearer":
		// 1. Try cache.
		if h.oauthProvider != nil {
			cached, _ := h.oauthProvider.GetCachedToken(ctx, serviceType, tenantID, datasourceID)
			if cached != nil && cached.Valid() && cached.AccessToken != "" {
				httpReq.Header.Set("Authorization", "Bearer "+cached.AccessToken)
				return nil
			}
		}
		// 2. Refresh using stored OAuth creds, if available.
		if h.oauthProvider != nil && oauthCreds.RefreshToken != "" && oauthCreds.ClientID != "" && oauthCreds.TokenURL != "" {
			refreshed, err := h.oauthProvider.RefreshWithConfig(ctx, oauthCreds)
			if err == nil && refreshed != nil && refreshed.AccessToken != "" {
				_ = h.oauthProvider.SaveToken(ctx, serviceType, tenantID, datasourceID, refreshed)
				httpReq.Header.Set("Authorization", "Bearer "+refreshed.AccessToken)
				return nil
			}
		}
		// 3. Fall back to static token the tenant pasted in auth_config.token.
		if tenantAuthConfig != nil {
			if token, ok := tenantAuthConfig["token"].(string); ok && token != "" {
				httpReq.Header.Set("Authorization", "Bearer "+token)
				return nil
			}
		}
		// No auth applied.
		return nil

	case "api_key":
		if tenantAuthConfig == nil {
			return nil
		}
		if apiKey, ok := tenantAuthConfig["api_key"].(string); ok && apiKey != "" {
			headerName := "X-API-Key"
			if hn, ok := tenantAuthConfig["header_name"].(string); ok && hn != "" {
				headerName = hn
			}
			httpReq.Header.Set(headerName, apiKey)
		}
		return nil

	case "basic_auth":
		if tenantAuthConfig == nil {
			return nil
		}
		if user, ok := tenantAuthConfig["username"].(string); ok {
			pass, _ := tenantAuthConfig["password"].(string)
			authStr := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
			httpReq.Header.Set("Authorization", "Basic "+authStr)
		}
		return nil

	case "none", "":
		return nil

	default:
		// Unknown auth type — leave the request unauthenticated; surfacing an
		// error here would prevent testing of newly added types.
		return nil
	}
}

// auditEntry is the in-memory representation of a row to be inserted into
// api_dispatch_audit_log. The dispatcher writes these via recordAudit, which
// fires a goroutine so the user-facing HTTP response is never blocked on
// the audit insert.
type auditEntry struct {
	TenantID       string                 `json:"tenant_id"`
	UserID         string                 `json:"user_id,omitempty"`
	DatasourceID   string                 `json:"api_datasource_id"`
	EndpointID     string                 `json:"api_endpoint_id"`
	Method         string                 `json:"method"`
	Path           string                 `json:"path"`
	StatusCode     int                    `json:"status_code"`
	DurationMs     int64                  `json:"duration_ms"`
	Success        bool                   `json:"success"`
	RecordCount    int                    `json:"record_count"`
	Error          string                 `json:"error"`
	RequestParams  map[string]interface{} `json:"request_params"`
}

// auditBuffer is a bounded buffered channel that absorbs audit writes
// without coupling dispatch latency to DB latency. When full, calls are
// dropped (counted in the drop counter) rather than blocking the goroutine.
var auditBuffer = make(chan auditEntry, 1024)
var auditDrops atomic.Uint64

// recordAudit queues an audit entry for asynchronous insertion. Returns
// immediately; the goroutine worker pool drains the buffer. If the buffer
// is full (e.g. during a DB outage), the entry is silently dropped rather
// than blocking the user's dispatch response. Drops are observable via the
// Prometheus metric api_oauth_audit_drops_total.
func (h *ApiDispatcherHandler) recordAudit(entry auditEntry) {
	select {
	case auditBuffer <- entry:
	default:
		auditDrops.Add(1)
	}
}

// auditWorker drains auditBuffer and writes to the database. Stop when the
// provided context is cancelled.
func (h *ApiDispatcherHandler) auditWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case entry := <-auditBuffer:
			h.writeAuditEntry(ctx, entry)
		}
	}
}

// StartAuditWorker launches a single background goroutine that drains
// auditBuffer and writes entries into api_dispatch_audit_log. Idempotent.
var auditOnce sync.Once

func (h *ApiDispatcherHandler) StartAuditWorker(ctx context.Context) {
	auditOnce.Do(func() {
		go h.auditWorker(ctx)
	})
}

// writeAuditEntry performs the actual DB insert. Errors are logged but never
// propagated (fire-and-forget semantics).
func (h *ApiDispatcherHandler) writeAuditEntry(ctx context.Context, entry auditEntry) {
	if h.db == nil {
		return
	}
	paramsJSON, _ := json.Marshal(entry.RequestParams)
	if paramsJSON == nil {
		paramsJSON = []byte(`{}`)
	}
	var userID interface{}
	if entry.UserID != "" {
		userID = entry.UserID
	}
	_, err := h.db.ExecContext(ctx, `
		INSERT INTO api_dispatch_audit_log (
			tenant_id, user_id, api_datasource_id, api_endpoint_id,
			method, path, status_code, duration_ms,
			success, record_count, error, request_params
		) VALUES (
			$1::uuid, $2, $3::uuid, $4::uuid,
			$5, $6, $7, $8,
			$9, $10, $11, $12::jsonb
		)
	`, entry.TenantID, userID, entry.DatasourceID, entry.EndpointID,
		entry.Method, entry.Path, entry.StatusCode, entry.DurationMs,
		entry.Success, entry.RecordCount, entry.Error, string(paramsJSON))
	if err != nil {
		// Don't break the dispatch flow; just log.
		fmt.Printf("[api-dispatcher] audit insert failed: %v\n", err)
	}
}

// ListDispatchAudit returns the most recent audit entries for the current
// tenant. When endpoint_id is supplied, only entries for that endpoint are
// returned. Results are limited to the most recent N rows.
func (h *ApiDispatcherHandler) ListDispatchAudit(w http.ResponseWriter, r *http.Request) {
	// getSecureTenantID (helpers.go) validates the X-Tenant-ID header /
	// tenant_id query param against the caller's JWT-issued tenant list /
	// global-admin status before trusting it; it never trusts either raw
	// client-supplied value directly.
	tenantID := getSecureTenantID(r)
	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}
	endpointID := r.URL.Query().Get("endpoint_id")

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	var (
		rows *sql.Rows
		err  error
	)
	if endpointID != "" {
		rows, err = h.db.QueryContext(r.Context(), `
			SELECT id, created_at, method, path, status_code, duration_ms,
			       success, record_count, error, api_endpoint_id
			FROM api_dispatch_audit_log
			WHERE tenant_id = $1::uuid AND api_endpoint_id = $2::uuid
			ORDER BY created_at DESC
			LIMIT $3
		`, tenantID, endpointID, limit)
	} else {
		rows, err = h.db.QueryContext(r.Context(), `
			SELECT id, created_at, method, path, status_code, duration_ms,
			       success, record_count, error, api_endpoint_id
			FROM api_dispatch_audit_log
			WHERE tenant_id = $1::uuid
			ORDER BY created_at DESC
			LIMIT $2
		`, tenantID, limit)
	}
	if err != nil {
		http.Error(w, "Failed to query audit log: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	out := []map[string]interface{}{}
	for rows.Next() {
		var (
			id, method, path, errMsg, epID string
			createdAt                      time.Time
			statusCode, recordCount        int
			durationMs                     int64
			success                        bool
		)
		if err := rows.Scan(&id, &createdAt, &method, &path, &statusCode, &durationMs, &success, &recordCount, &errMsg, &epID); err != nil {
			continue
		}
		out = append(out, map[string]interface{}{
			"id":              id,
			"created_at":      createdAt,
			"method":          method,
			"path":            path,
			"status_code":     statusCode,
			"duration_ms":     durationMs,
			"success":         success,
			"record_count":    recordCount,
			"error":           errMsg,
			"api_endpoint_id": epID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":             true,
		"data":                out,
		"audit_drops_pending": auditDrops.Load(),
	})
}

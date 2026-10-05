package iceberg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type LakekeeperProvisioner struct {
	baseURL       string
	s3Bucket      string
	s3Endpoint    string
	warehouseID   string // Iceberg REST catalog prefix; Lakekeeper warehouse UUID
	warehouseName string // used to resolve warehouseID when unset
	httpClient    *http.Client
	tokenMgr      *TokenManager
}

func NewLakekeeperProvisioner(baseURL, s3Bucket, s3Endpoint string) *LakekeeperProvisioner {
	if baseURL == "" {
		baseURL = os.Getenv("LAKEKEEPER_URL")
		if baseURL == "" {
			baseURL = "http://lakekeeper:8181"
		}
	}
	if s3Bucket == "" {
		s3Bucket = os.Getenv("S3_BUCKET")
		if s3Bucket == "" {
			s3Bucket = "iceberg-warehouse"
		}
	}
	if s3Endpoint == "" {
		s3Endpoint = os.Getenv("S3_ENDPOINT")
		if s3Endpoint == "" {
			s3Endpoint = "http://minio:9000"
		}
	}
	tm := newDefaultTokenManager(baseURL)
	return &LakekeeperProvisioner{
		baseURL:       baseURL,
		s3Bucket:      s3Bucket,
		s3Endpoint:    s3Endpoint,
		warehouseID:   os.Getenv("LAKEKEEPER_WAREHOUSE_ID"),
		warehouseName: envOr("LAKEKEEPER_WAREHOUSE_NAME", "uisce-raw"),
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		tokenMgr:      tm,
	}
}

func NewAuthedLakekeeperProvisioner(baseURL, s3Bucket, s3Endpoint string, tokenMgr *TokenManager) *LakekeeperProvisioner {
	return &LakekeeperProvisioner{
		baseURL:       baseURL,
		s3Bucket:      s3Bucket,
		s3Endpoint:    s3Endpoint,
		warehouseID:   os.Getenv("LAKEKEEPER_WAREHOUSE_ID"),
		warehouseName: envOr("LAKEKEEPER_WAREHOUSE_NAME", "uisce-raw"),
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		tokenMgr:      tokenMgr,
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// catalogPrefix is the Iceberg REST {prefix} for namespace/table routes:
// /catalog/v1/{prefix}/namespaces. Lakekeeper uses the warehouse UUID.
func (p *LakekeeperProvisioner) catalogPrefix(ctx context.Context) (string, error) {
	if p.warehouseID != "" {
		return p.warehouseID, nil
	}
	name := p.warehouseName
	if name == "" {
		name = "uisce-raw"
	}
	id, status, err := p.GetWarehouseByName(ctx, name)
	if err != nil {
		return "", fmt.Errorf("resolve warehouse %q: %w", name, err)
	}
	if id == "" {
		return "", fmt.Errorf("resolve warehouse %q: not found (status %d); set LAKEKEEPER_WAREHOUSE_ID or create the warehouse", name, status)
	}
	p.warehouseID = id
	return id, nil
}

func newDefaultTokenManager(baseURL string) *TokenManager {
	tokenURL := os.Getenv("LAKEKEEPER_TOKEN_URL")
	if tokenURL == "" {
		tokenURL = "https://keycloak:8443/realms/uisce/protocol/openid-connect/token"
	}
	clientID := os.Getenv("LAKEKEEPER_CLIENT_ID")
	if clientID == "" {
		clientID = "uisce-provisioner"
	}
	clientSecret := os.Getenv("LAKEKEEPER_CLIENT_SECRET")
	return NewTokenManager(tokenURL, clientID, clientSecret)
}

type NamespaceConfig struct {
	Namespace []string          `json:"namespace"`
	Properties map[string]string `json:"properties,omitempty"`
}

func (p *LakekeeperProvisioner) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if p.tokenMgr != nil {
		token, err := p.tokenMgr.Token(ctx)
		if err != nil {
			return nil, fmt.Errorf("acquire token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	return resp, nil
}

func (p *LakekeeperProvisioner) NamespaceExists(ctx context.Context, namespace string) (bool, error) {
	prefix, err := p.catalogPrefix(ctx)
	if err != nil {
		return false, err
	}
	resp, err := p.doRequest(ctx, http.MethodGet, fmt.Sprintf("/catalog/v1/%s/namespaces/%s", prefix, namespace), nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("namespace check returned %d: %s", resp.StatusCode, string(body))
}

func (p *LakekeeperProvisioner) CreateNamespace(ctx context.Context, tenantCode string) error {
	prefix, err := p.catalogPrefix(ctx)
	if err != nil {
		return err
	}
	namespace := NamespaceConfig{
		Namespace: []string{tenantCode},
		Properties: map[string]string{
			"default-base-location": fmt.Sprintf("s3://%s/%s/%s", p.s3Bucket, p.warehouseName, tenantCode),
		},
	}

	resp, err := p.doRequest(ctx, http.MethodPost, fmt.Sprintf("/catalog/v1/%s/namespaces", prefix), namespace)
	if err != nil {
		return fmt.Errorf("create namespace request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		return nil
	}
	if resp.StatusCode == http.StatusConflict {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("create namespace returned %d: %s", resp.StatusCode, string(body))
}

func (p *LakekeeperProvisioner) DeleteNamespace(ctx context.Context, tenantCode string) error {
	prefix, err := p.catalogPrefix(ctx)
	if err != nil {
		return err
	}
	resp, err := p.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/catalog/v1/%s/namespaces/%s", prefix, tenantCode), nil)
	if err != nil {
		return fmt.Errorf("delete namespace request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("delete namespace returned %d: %s", resp.StatusCode, string(body))
}

func (p *LakekeeperProvisioner) GetNamespace(ctx context.Context, tenantCode string) (*NamespaceConfig, error) {
	prefix, err := p.catalogPrefix(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := p.doRequest(ctx, http.MethodGet, fmt.Sprintf("/catalog/v1/%s/namespaces/%s", prefix, tenantCode), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get namespace returned %d: %s", resp.StatusCode, string(body))
	}

	var ns NamespaceConfig
	if err := json.NewDecoder(resp.Body).Decode(&ns); err != nil {
		return nil, fmt.Errorf("decode namespace response: %w", err)
	}
	return &ns, nil
}

func (p *LakekeeperProvisioner) AddColumnToIcebergTable(ctx context.Context, warehouse, namespace, tableName, colName, colType string) error {
	path := fmt.Sprintf("/catalog/v1/namespaces/%s/tables/%s", namespace, tableName)
	
	reqBody := map[string]interface{}{
		"requirements": []map[string]interface{}{},
		"updates": []map[string]interface{}{
			{
				"action": "add-schema",
				"schema": map[string]interface{}{
					"type": "struct",
					"fields": []map[string]interface{}{
						{
							"name":     colName,
							"type":     colType,
							"required": false,
						},
					},
				},
			},
		},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create update schema request: %w", err)
	}
	req.Header.Set("X-Iceberg-Warehouse", warehouse)
	req.Header.Set("Content-Type", "application/json")

	b, _ := json.Marshal(reqBody)
	req.Body = io.NopCloser(bytes.NewBuffer(b))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute schema update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("schema evolution failed with status %d: %s", resp.StatusCode, string(body))
}

func (p *LakekeeperProvisioner) HealthCheck(ctx context.Context) error {
	resp, err := p.doRequest(ctx, http.MethodGet, "/health", nil)
	if err != nil {
		return fmt.Errorf("health check request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

type warehouseResponse struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
}

func (p *LakekeeperProvisioner) CreateWarehouse(ctx context.Context, payload map[string]interface{}) error {
	resp, err := p.doRequest(ctx, http.MethodPost, "/management/v1/warehouse", payload)
	if err != nil {
		return fmt.Errorf("create warehouse request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create warehouse returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (p *LakekeeperProvisioner) GetWarehouseByName(ctx context.Context, name string) (string, int, error) {
	resp, err := p.doRequest(ctx, http.MethodGet, fmt.Sprintf("/management/v1/warehouse?name=%s", name), nil)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", http.StatusNotFound, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", resp.StatusCode, fmt.Errorf("get warehouse returned %d: %s", resp.StatusCode, string(body))
	}

	var listResp struct {
		Warehouses []warehouseResponse `json:"warehouses"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return "", resp.StatusCode, fmt.Errorf("decode warehouse list: %w", err)
	}
	for _, w := range listResp.Warehouses {
		if w.Name == name {
			return w.ID, resp.StatusCode, nil
		}
	}
	return "", resp.StatusCode, nil
}

func (p *LakekeeperProvisioner) DeleteWarehouse(ctx context.Context, id string) error {
	resp, err := p.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/management/v1/warehouse/%s", id), nil)
	if err != nil {
		return fmt.Errorf("delete warehouse request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete warehouse returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
package iceberg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

type LakekeeperProvisioner struct {
	baseURL    string
	s3Bucket   string
	s3Endpoint string
	httpClient *http.Client
	token      string
	tokenMu    sync.RWMutex
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
	return &LakekeeperProvisioner{
		baseURL:    baseURL,
		s3Bucket:   s3Bucket,
		s3Endpoint: s3Endpoint,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
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

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	return resp, nil
}

func (p *LakekeeperProvisioner) NamespaceExists(ctx context.Context, namespace string) (bool, error) {
	resp, err := p.doRequest(ctx, http.MethodGet, fmt.Sprintf("/v1/namespaces/%s", namespace), nil)
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
	namespace := NamespaceConfig{
		Namespace: []string{tenantCode},
		Properties: map[string]string{
			"default-base-location": fmt.Sprintf("s3://%s/%s", p.s3Bucket, tenantCode),
		},
	}

	resp, err := p.doRequest(ctx, http.MethodPost, "/v1/namespaces", namespace)
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
	resp, err := p.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/v1/namespaces/%s", tenantCode), nil)
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
	resp, err := p.doRequest(ctx, http.MethodGet, fmt.Sprintf("/v1/namespaces/%s", tenantCode), nil)
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
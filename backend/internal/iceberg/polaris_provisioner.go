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

type PolarisProvisioner struct {
	baseURL    string
	clientID   string
	clientSecret string
	s3Bucket   string
	s3Endpoint string
	httpClient *http.Client
	token      string
	tokenMu    sync.RWMutex
}

func NewPolarisProvisioner(baseURL, clientID, clientSecret, s3Bucket, s3Endpoint string) *PolarisProvisioner {
	return &PolarisProvisioner{
		baseURL:     baseURL,
		clientID:    clientID,
		clientSecret: clientSecret,
		s3Bucket:   s3Bucket,
		s3Endpoint: s3Endpoint,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type CatalogConfig struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	ReadOnly       bool   `json:"readOnly"`
	Properties     map[string]string `json:"properties"`
	StorageConfigInfo StorageConfigInfo `json:"storageConfigInfo"`
}

type StorageConfigInfo struct {
	StorageType     string   `json:"storageType"`
	AllowedLocations []string `json:"allowedLocations"`
}

type provisionPayload struct {
	Catalog CatalogConfig `json:"catalog"`
}

func (p *PolarisProvisioner) getToken(ctx context.Context) (string, error) {
	p.tokenMu.RLock()
	if p.token != "" {
		defer p.tokenMu.RUnlock()
		return p.token, nil
	}
	p.tokenMu.RUnlock()

	p.tokenMu.Lock()
	defer p.tokenMu.Unlock()
	if p.token != "" {
		return p.token, nil
	}

	reqBody := fmt.Sprintf(
		"grant_type=client_credentials&client_id=%s&client_secret=%s&scope=PRINCIPAL_ROLE:ALL",
		p.clientID, p.clientSecret,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/catalog/v1/oauth/tokens",
		bytes.NewBufferString(reqBody))
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed (%d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	p.token = tokenResp.AccessToken
	return p.token, nil
}

func (p *PolarisProvisioner) doReq(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	token, err := p.getToken(ctx)
	if err != nil {
		return nil, err
	}

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
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	return resp, nil
}

func (p *PolarisProvisioner) Provision(ctx context.Context, tenantKey string) error {
	catalogLocation := fmt.Sprintf("s3://%s/%s", p.s3Bucket, tenantKey)

	payload := provisionPayload{
		Catalog: CatalogConfig{
			Name:      tenantKey,
			Type:      "INTERNAL",
			ReadOnly:  false,
			Properties: map[string]string{
				"default-base-location": catalogLocation,
				"s3.credentials-type":   "MANUAL",
				"s3.endpoint":          p.s3Endpoint,
				"s3.path-style-access": "true",
			},
			StorageConfigInfo: StorageConfigInfo{
				StorageType:      "S3",
				AllowedLocations: []string{catalogLocation},
			},
		},
	}

	resp, err := p.doReq(ctx, http.MethodPost, "/api/management/v1/catalogs", payload)
	if err != nil {
		return fmt.Errorf("create catalog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create catalog returned %d: %s", resp.StatusCode, string(body))
	}

	grantRolePath := fmt.Sprintf(
		"/api/management/v1/principal-roles/service_admin/catalog-roles/%s",
		tenantKey,
	)
	grantRoleBody := map[string]any{"catalogRole": map[string]string{"name": "catalog_admin"}}
	grResp, err := p.doReq(ctx, http.MethodPut, grantRolePath, grantRoleBody)
	if err == nil {
		grResp.Body.Close()
	}

	grantPrivPath := fmt.Sprintf(
		"/api/management/v1/catalogs/%s/catalog-roles/catalog_admin/grants",
		tenantKey,
	)
	grantPrivBody := map[string]any{"grant": map[string]string{"type": "catalog", "privilege": "CATALOG_MANAGE_CONTENT"}}
	gpResp, err := p.doReq(ctx, http.MethodPut, grantPrivPath, grantPrivBody)
	if err == nil {
		gpResp.Body.Close()
	}

	return nil
}

func (p *PolarisProvisioner) CreateAnalyticsNamespace(ctx context.Context, catalogName string) error {
	nsPayload := map[string]any{
		"namespace": map[string][]string{
			"names": {"analytics"},
		},
	}
	resp, err := p.doReq(ctx, http.MethodPost,
		fmt.Sprintf("/api/catalog/v1/%s/namespaces", catalogName), nsPayload)
	if err != nil {
		return fmt.Errorf("create analytics namespace: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create namespace returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func PolarisFromEnv() *PolarisProvisioner {
	baseURL := os.Getenv("POLARIS_URL")
	if baseURL == "" {
		baseURL = "http://uisce-polaris:8185"
	}
	clientID := os.Getenv("POLARIS_CLIENT_ID")
	if clientID == "" {
		clientID = "root"
	}
	clientSecret := os.Getenv("POLARIS_CLIENT_SECRET")
	if clientSecret == "" {
		clientSecret = "secret"
	}
	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket == "" {
		s3Bucket = "iceberg-warehouse"
	}
	s3Endpoint := os.Getenv("S3_ENDPOINT")
	if s3Endpoint == "" {
		s3Endpoint = "http://uisce-minio:9000"
	}
	return NewPolarisProvisioner(baseURL, clientID, clientSecret, s3Bucket, s3Endpoint)
}

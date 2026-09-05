package datapipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/apistudio"
)

type mockAPIRepo struct {
	endpoints map[uuid.UUID]*apistudio.APIEndpoint
}

func (m *mockAPIRepo) GetEndpoint(ctx context.Context, id uuid.UUID) (*apistudio.APIEndpoint, error) {
	if ep, ok := m.endpoints[id]; ok {
		return ep, nil
	}
	return nil, fmt.Errorf("endpoint not found")
}

func (m *mockAPIRepo) LogTelemetry(ctx context.Context, t *apistudio.APITelemetry) error {
	return nil
}

func newMockRepo(tenantID, env string) *mockAPIRepo {
	return &mockAPIRepo{
		endpoints: make(map[uuid.UUID]*apistudio.APIEndpoint),
	}
}

func TestAPICaller_T1_RealCall_Success(t *testing.T) {
	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"result": "ok", "value": 42})
	}))
	defer srv.Close()

	epID := uuid.New()
	tenantID := uuid.New().String()
	repo := &mockAPIRepo{
		endpoints: map[uuid.UUID]*apistudio.APIEndpoint{
			epID: {
				ID:       epID,
				Env:      "default",
				TenantID: tenantID,
				Path:     "/test",
				Method:   "GET",
				Type:     "rest",
				Status:   "active",
			},
		},
	}

	caller := &APICallerTransformer{
		APIEndpointID: epID,
		registry:      repo,
		httpClient:    srv.Client(),
		BaseURL:       srv.URL,
	}

	records := []PipelineRecord{{"id": "1", "name": "test"}}
	out, errs, err := caller.Transform(context.Background(), records)

	require.NoError(t, err)
	assert.Empty(t, errs)
	require.Len(t, out, 1)
	assert.True(t, hit)
	assert.Equal(t, "success", out[0]["api_status"])
	assert.Equal(t, 200, out[0]["api_status_code"])
}

func TestAPICaller_T2_5xx_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error": "downstream error"}`))
	}))
	defer srv.Close()

	epID := uuid.New()
	tenantID := uuid.New().String()
	repo := &mockAPIRepo{
		endpoints: map[uuid.UUID]*apistudio.APIEndpoint{
			epID: {
				ID:       epID,
				Env:      "default",
				TenantID: tenantID,
				Path:     "/test",
				Method:   "GET",
				Type:     "rest",
				Status:   "active",
			},
		},
	}

	caller := &APICallerTransformer{
		APIEndpointID: epID,
		registry:      repo,
		httpClient:    srv.Client(),
		BaseURL:       srv.URL,
	}

	records := []PipelineRecord{{"id": "1"}}
	out, _, err := caller.Transform(context.Background(), records)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "api_caller")
	assert.Empty(t, out)
}

func TestAPICaller_T3a_SSRF_PrivateIPBlocked(t *testing.T) {
	savedAllowLoopback := allowLoopback
	origEnv := os.Getenv("APICALLER_ALLOW_LOOPBACK")
	allowLoopback = false
	os.Setenv("APICALLER_ALLOW_LOOPBACK", "")
	defer func() {
		allowLoopback = savedAllowLoopback
		os.Setenv("APICALLER_ALLOW_LOOPBACK", origEnv)
	}()

	epID := uuid.New()
	tenantID := uuid.New().String()
	repo := &mockAPIRepo{
		endpoints: map[uuid.UUID]*apistudio.APIEndpoint{
			epID: {
				ID:       epID,
				Env:      "default",
				TenantID: tenantID,
				Path:     "/test",
				Method:   "GET",
				Type:     "rest",
				Status:   "active",
			},
		},
	}

	caller := &APICallerTransformer{
		APIEndpointID: epID,
		registry:      repo,
		httpClient:    &http.Client{Timeout: 5 * time.Second},
		BaseURL:       "http://this-host-does-not-exist.invalid",
	}

	records := []PipelineRecord{{"id": "1"}}
	out, _, err := caller.Transform(context.Background(), records)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "ssrf guard") || strings.Contains(err.Error(), "could not resolve") || strings.Contains(err.Error(), "no such host") || strings.Contains(err.Error(), "lookup"))
	assert.Empty(t, out)
}

func TestAPICaller_T3b_SSRF_LoopbackAllowedWithExemption(t *testing.T) {
	savedAllowLoopback := allowLoopback
	origAllowLoopback := os.Getenv("APICALLER_ALLOW_LOOPBACK")
	allowLoopback = true
	os.Setenv("APICALLER_ALLOW_LOOPBACK", "1")
	defer func() {
		allowLoopback = savedAllowLoopback
		os.Setenv("APICALLER_ALLOW_LOOPBACK", origAllowLoopback)
	}()

	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer srv.Close()

	epID := uuid.New()
	tenantID := uuid.New().String()
	repo := &mockAPIRepo{
		endpoints: map[uuid.UUID]*apistudio.APIEndpoint{
			epID: {
				ID:       epID,
				Env:      "default",
				TenantID: tenantID,
				Path:     "/test",
				Method:   "GET",
				Type:     "rest",
				Status:   "active",
			},
		},
	}

	caller := &APICallerTransformer{
		APIEndpointID: epID,
		registry:      repo,
		httpClient:    srv.Client(),
		BaseURL:       srv.URL,
	}

	records := []PipelineRecord{{"id": "1"}}
	out, errs, err := caller.Transform(context.Background(), records)

	require.NoError(t, err)
	assert.Empty(t, errs)
	require.Len(t, out, 1)
	assert.True(t, hit)
}

func TestAPICaller_T4_1MB_Cap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Transfer-Encoding", "identity")
		// Write exactly 1 MiB + 1 byte — Transfer-Encoding: identity disables chunked
		buf := make([]byte, 1<<20+1)
		buf[0] = '{'
		buf[len(buf)-1] = '}'
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf)
	}))
	defer srv.Close()

	epID := uuid.New()
	tenantID := uuid.New().String()
	repo := &mockAPIRepo{
		endpoints: map[uuid.UUID]*apistudio.APIEndpoint{
			epID: {
				ID:       epID,
				Env:      "default",
				TenantID: tenantID,
				Path:     "/test",
				Method:   "GET",
				Type:     "rest",
				Status:   "active",
			},
		},
	}

	caller := &APICallerTransformer{
		APIEndpointID: epID,
		registry:      repo,
		httpClient:    srv.Client(),
		BaseURL:       srv.URL,
	}

	records := []PipelineRecord{{"id": "1"}}
	out, _, err := caller.Transform(context.Background(), records)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeded 1 MiB limit")
	assert.Empty(t, out)
}

func TestAPICaller_T5_MissingEndpointID_ReturnsError(t *testing.T) {
	caller := &APICallerTransformer{
		APIEndpointID: uuid.Nil,
		BaseURL:       "http://localhost:9999",
	}

	records := []PipelineRecord{{"id": "1"}}
	out, _, err := caller.Transform(context.Background(), records)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint_id")
	assert.Empty(t, out)
}

func TestAPICaller_T6_EndpointNotFoundError(t *testing.T) {
	savedAllowLoopback := allowLoopback
	origEnv := os.Getenv("APICALLER_ALLOW_LOOPBACK")
	allowLoopback = true
	os.Setenv("APICALLER_ALLOW_LOOPBACK", "1")
	defer func() {
		allowLoopback = savedAllowLoopback
		os.Setenv("APICALLER_ALLOW_LOOPBACK", origEnv)
	}()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	epID := uuid.New()
	repo := &mockAPIRepo{
		endpoints: map[uuid.UUID]*apistudio.APIEndpoint{},
	}

	caller := &APICallerTransformer{
		APIEndpointID: epID,
		registry:      repo,
		httpClient:    srv.Client(),
		BaseURL:       srv.URL,
	}

	records := []PipelineRecord{{"id": "1"}}
	out, _, err := caller.Transform(context.Background(), records)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Empty(t, out)
}

func TestAPICaller_NoBaseURL_ReturnsError(t *testing.T) {
	origBaseURL := os.Getenv("APICALLER_TARGET_BASE_URL")
	os.Unsetenv("APICALLER_TARGET_BASE_URL")
	defer os.Setenv("APICALLER_TARGET_BASE_URL", origBaseURL)

	epID := uuid.New()
	repo := &mockAPIRepo{
		endpoints: map[uuid.UUID]*apistudio.APIEndpoint{
			epID: {
				ID:       epID,
				Env:      "default",
				TenantID: "tenant-a",
				Path:     "/test",
				Method:   "GET",
				Type:     "rest",
				Status:   "active",
			},
		},
	}

	caller := &APICallerTransformer{
		APIEndpointID: epID,
		registry:   repo,
	}

	records := []PipelineRecord{{"id": "1"}}
	out, _, err := caller.Transform(context.Background(), records)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "APICALLER_TARGET_BASE_URL")
	assert.Empty(t, out)
}

func TestAPICaller_BearerAuth(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer srv.Close()

	epID := uuid.New()
	tenantID := uuid.New().String()
	repo := &mockAPIRepo{
		endpoints: map[uuid.UUID]*apistudio.APIEndpoint{
			epID: {
				ID:           epID,
				Env:          "default",
				TenantID:     tenantID,
				Path:         "/test",
				Method:       "GET",
				Type:         "rest",
				Status:       "active",
				AuthType:     "bearer",
				AuthConfig:   json.RawMessage(`{"token": "secret-token-123"}`),
			},
		},
	}

	caller := &APICallerTransformer{
		APIEndpointID: epID,
		registry:      repo,
		httpClient:    srv.Client(),
		BaseURL:       srv.URL,
	}

	records := []PipelineRecord{{"id": "1"}}
	out, _, err := caller.Transform(context.Background(), records)

	require.NoError(t, err)
	assert.Equal(t, "Bearer secret-token-123", authHeader)
	assert.Len(t, out, 1)
}

func TestAPICaller_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	epID := uuid.New()
	tenantID := uuid.New().String()
	repo := &mockAPIRepo{
		endpoints: map[uuid.UUID]*apistudio.APIEndpoint{
			epID: {
				ID:       epID,
				Env:      "default",
				TenantID: tenantID,
				Path:     "/test",
				Method:   "GET",
				Type:     "rest",
				Status:   "active",
			},
		},
	}

	transport := &mockSSRFTransport{base: srv.Client().Transport}
	caller := &APICallerTransformer{
		APIEndpointID: epID,
		registry:      repo,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   100 * time.Millisecond,
		},
		BaseURL: srv.URL,
	}

	records := []PipelineRecord{{"id": "1"}}
	out, _, err := caller.Transform(context.Background(), records)

	assert.Error(t, err)
	assert.Empty(t, out)
}

type mockSSRFTransport struct {
	base http.RoundTripper
}

func (t *mockSSRFTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return t.base.RoundTrip(r)
}

func TestAPICaller_PostMethod_WithRequestTemplate(t *testing.T) {
	var method, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		body = string(buf[:n])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"confirmed": true}`))
	}))
	defer srv.Close()

	epID := uuid.New()
	tenantID := uuid.New().String()
	repo := &mockAPIRepo{
		endpoints: map[uuid.UUID]*apistudio.APIEndpoint{
			epID: {
				ID:       epID,
				Env:      "default",
				TenantID: tenantID,
				Path:     "/test",
				Method:   "POST",
				Type:     "rest",
				Status:   "active",
			},
		},
	}

	caller := &APICallerTransformer{
		APIEndpointID: epID,
		RequestTemplate: map[string]interface{}{
			"source_id": "{{.id}}",
			"action":    "verify",
		},
		registry:   repo,
		httpClient: srv.Client(),
		BaseURL:   srv.URL,
	}

	records := []PipelineRecord{{"id": "rec-123"}}
	out, _, err := caller.Transform(context.Background(), records)

	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "POST", method)
	assert.Contains(t, body, "source_id")
	assert.Contains(t, body, "verify")
}

func TestAPICaller_MergeOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"verified": true, "score": 95}`))
	}))
	defer srv.Close()

	epID := uuid.New()
	tenantID := uuid.New().String()
	repo := &mockAPIRepo{
		endpoints: map[uuid.UUID]*apistudio.APIEndpoint{
			epID: {
				ID:       epID,
				Env:      "default",
				TenantID: tenantID,
				Path:     "/test",
				Method:   "GET",
				Type:     "rest",
				Status:   "active",
			},
		},
	}

	caller := &APICallerTransformer{
		APIEndpointID: epID,
		MergeOutput:   true,
		registry:      repo,
		httpClient:    srv.Client(),
		BaseURL:       srv.URL,
	}

	records := []PipelineRecord{{"id": "1"}}
	out, _, err := caller.Transform(context.Background(), records)

	require.NoError(t, err)
	require.Len(t, out, 1)
	resp, ok := out[0]["_api_response"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, resp["verified"])
	assert.Equal(t, float64(95), resp["score"])
}

func TestAPICaller_TargetField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"score": 100}`))
	}))
	defer srv.Close()

	epID := uuid.New()
	tenantID := uuid.New().String()
	repo := &mockAPIRepo{
		endpoints: map[uuid.UUID]*apistudio.APIEndpoint{
			epID: {
				ID:       epID,
				Env:      "default",
				TenantID: tenantID,
				Path:     "/test",
				Method:   "GET",
				Type:     "rest",
				Status:   "active",
			},
		},
	}

	caller := &APICallerTransformer{
		APIEndpointID: epID,
		TargetField:   "verification_result",
		registry:      repo,
		httpClient:    srv.Client(),
		BaseURL:       srv.URL,
	}

	records := []PipelineRecord{{"id": "1"}}
	out, _, err := caller.Transform(context.Background(), records)

	require.NoError(t, err)
	require.Len(t, out, 1)
	result, ok := out[0]["verification_result"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(100), result["score"])
}

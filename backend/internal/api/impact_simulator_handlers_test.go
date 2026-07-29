package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hondyman/uisce/backend/internal/lineage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSimJWTSecret = "test-sim-jwt-secret-2024"

func createSimulatorTestToken(tenantID, userID, jwtSecret string) string {
	claims := jwt.MapClaims{
		"user_id":    userID,
		"tenant_id":  tenantID,
		"email":      userID + "@test.com",
		"tenant_ids": []string{tenantID},
		"roles":      []string{"admin"},
		"is_active":  true,
		"exp":        time.Now().Add(time.Hour).Unix(),
		"iat":        time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(jwtSecret))
	return tokenString
}

type mockLineageRepoForHandler struct {
	findUpstreamFn   func(ctx context.Context, rootID string, depth int) (*lineage.Graph, error)
	findDownstreamFn func(ctx context.Context, rootID string, depth int) (*lineage.Graph, error)
}

func (m *mockLineageRepoForHandler) UpsertNode(ctx context.Context, node lineage.LineageNode) error {
	return nil
}
func (m *mockLineageRepoForHandler) UpsertEdge(ctx context.Context, edge lineage.LineageEdge) error {
	return nil
}
func (m *mockLineageRepoForHandler) DeleteNode(ctx context.Context, id string) error {
	return nil
}
func (m *mockLineageRepoForHandler) DeleteEdge(ctx context.Context, fromID, toID, edgeType string) error {
	return nil
}
func (m *mockLineageRepoForHandler) FindDownstreamGraph(ctx context.Context, rootID string, depth int) (*lineage.Graph, error) {
	if m.findDownstreamFn != nil {
		return m.findDownstreamFn(ctx, rootID, depth)
	}
	return &lineage.Graph{}, nil
}
func (m *mockLineageRepoForHandler) FindUpstreamGraph(ctx context.Context, rootID string, depth int) (*lineage.Graph, error) {
	if m.findUpstreamFn != nil {
		return m.findUpstreamFn(ctx, rootID, depth)
	}
	return &lineage.Graph{}, nil
}
func (m *mockLineageRepoForHandler) FindBiDirectionalGraph(ctx context.Context, rootID string, depth int) (*lineage.Graph, error) {
	return &lineage.Graph{}, nil
}
func (m *mockLineageRepoForHandler) FindGraphByDatasource(ctx context.Context, datasourceID string) (*lineage.Graph, error) {
	return &lineage.Graph{}, nil
}
func (m *mockLineageRepoForHandler) SyncDatasource(ctx context.Context, datasourceID string) error {
	return nil
}

func TestSimulateImpact_Unauthorized(t *testing.T) {
	handler := NewImpactSimulatorHandler(lineage.NewLineageService(&mockLineageRepoForHandler{}))
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	body := `{"target_node": "node-1", "action": "drop_column"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lineage/simulate-impact", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSimulateImpact_EmptyBody(t *testing.T) {
	t.Setenv("JWT_SECRET", testSimJWTSecret)
	tenant := "tenant-1"
	repo := &mockLineageRepoForHandler{}
	svc := lineage.NewLineageService(repo)
	handler := NewImpactSimulatorHandler(svc)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createSimulatorTestToken(tenant, "user-1", testSimJWTSecret)
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lineage/simulate-impact", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSimulateImpact_MissingTargetNode(t *testing.T) {
	t.Setenv("JWT_SECRET", testSimJWTSecret)
	tenant := "tenant-1"
	repo := &mockLineageRepoForHandler{}
	svc := lineage.NewLineageService(repo)
	handler := NewImpactSimulatorHandler(svc)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createSimulatorTestToken(tenant, "user-1", testSimJWTSecret)
	body := `{"action": "drop_column"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lineage/simulate-impact", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSimulateImpact_InvalidJSON(t *testing.T) {
	t.Setenv("JWT_SECRET", testSimJWTSecret)
	tenant := "tenant-1"
	repo := &mockLineageRepoForHandler{}
	svc := lineage.NewLineageService(repo)
	handler := NewImpactSimulatorHandler(svc)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createSimulatorTestToken(tenant, "user-1", testSimJWTSecret)
	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lineage/simulate-impact", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSimulateImpact_Success(t *testing.T) {
	t.Setenv("JWT_SECRET", testSimJWTSecret)
	tenant := "tenant-1"
	repo := &mockLineageRepoForHandler{
		findUpstreamFn: func(ctx context.Context, rootID string, depth int) (*lineage.Graph, error) {
			return &lineage.Graph{Nodes: []lineage.LineageNode{
				{ID: "up1", Type: lineage.NodeTable, Name: "orders"},
			}}, nil
		},
		findDownstreamFn: func(ctx context.Context, rootID string, depth int) (*lineage.Graph, error) {
			return &lineage.Graph{Nodes: []lineage.LineageNode{
				{ID: "dn1", Type: lineage.NodePage, Name: "Dashboard"},
				{ID: "dn2", Type: lineage.NodeAPIEndpoint, Name: "/api/users"},
			}}, nil
		},
	}
	svc := lineage.NewLineageService(repo)
	handler := NewImpactSimulatorHandler(svc)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createSimulatorTestToken(tenant, "user-1", testSimJWTSecret)
	body := `{"target_node": "table-orders", "action": "drop_column", "depth": 3}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lineage/simulate-impact", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var report lineage.BlastRadiusReport
	err := json.Unmarshal(w.Body.Bytes(), &report)
	require.NoError(t, err)
	assert.Equal(t, "table-orders", report.TargetNode)
	assert.Equal(t, "drop_column", report.Action)
	assert.Equal(t, 2, report.BlastRadiusSummary.TotalImpactedArtifacts)
	assert.Len(t, report.UpstreamNodes, 1)
	assert.Len(t, report.DownstreamNodes, 2)
}

func TestSimulateImpact_DefaultDepth(t *testing.T) {
	t.Setenv("JWT_SECRET", testSimJWTSecret)
	tenant := "tenant-1"
	var capturedDepth int
	repo := &mockLineageRepoForHandler{
		findUpstreamFn: func(ctx context.Context, rootID string, depth int) (*lineage.Graph, error) {
			capturedDepth = depth
			return &lineage.Graph{}, nil
		},
		findDownstreamFn: func(ctx context.Context, rootID string, depth int) (*lineage.Graph, error) {
			capturedDepth = depth
			return &lineage.Graph{}, nil
		},
	}
	svc := lineage.NewLineageService(repo)
	handler := NewImpactSimulatorHandler(svc)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createSimulatorTestToken(tenant, "user-1", testSimJWTSecret)
	body := `{"target_node": "node-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lineage/simulate-impact", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 5, capturedDepth)
}

func TestSimulateImpact_EmptyGraphs(t *testing.T) {
	t.Setenv("JWT_SECRET", testSimJWTSecret)
	tenant := "tenant-1"
	repo := &mockLineageRepoForHandler{
		findUpstreamFn:   func(ctx context.Context, rootID string, depth int) (*lineage.Graph, error) { return &lineage.Graph{}, nil },
		findDownstreamFn: func(ctx context.Context, rootID string, depth int) (*lineage.Graph, error) { return &lineage.Graph{}, nil },
	}
	svc := lineage.NewLineageService(repo)
	handler := NewImpactSimulatorHandler(svc)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createSimulatorTestToken(tenant, "user-1", testSimJWTSecret)
	body := `{"target_node": "orphan-node", "action": "drop_table"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lineage/simulate-impact", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var report lineage.BlastRadiusReport
	err := json.Unmarshal(w.Body.Bytes(), &report)
	require.NoError(t, err)
	assert.Equal(t, lineage.BlastSeverityLow, report.BlastRadiusSummary.Severity)
	assert.Equal(t, 0, report.BlastRadiusSummary.TotalImpactedArtifacts)
}

func TestGetBlastRadius_Unauthorized(t *testing.T) {
	handler := NewImpactSimulatorHandler(lineage.NewLineageService(&mockLineageRepoForHandler{}))
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/lineage/node/node-1/blast-radius", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetBlastRadius_MissingNodeID(t *testing.T) {
	t.Setenv("JWT_SECRET", testSimJWTSecret)
	tenant := "tenant-1"
	repo := &mockLineageRepoForHandler{}
	svc := lineage.NewLineageService(repo)
	handler := NewImpactSimulatorHandler(svc)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createSimulatorTestToken(tenant, "user-1", testSimJWTSecret)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/lineage/node//blast-radius", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetBlastRadius_Success(t *testing.T) {
	t.Setenv("JWT_SECRET", testSimJWTSecret)
	tenant := "tenant-1"
	repo := &mockLineageRepoForHandler{
		findUpstreamFn: func(ctx context.Context, rootID string, depth int) (*lineage.Graph, error) {
			return &lineage.Graph{Nodes: []lineage.LineageNode{
				{ID: "t1", Type: lineage.NodeTable, Name: "orders"},
			}}, nil
		},
		findDownstreamFn: func(ctx context.Context, rootID string, depth int) (*lineage.Graph, error) {
			return &lineage.Graph{Nodes: []lineage.LineageNode{
				{ID: "p1", Type: lineage.NodePage, Name: "Dashboard"},
				{ID: "a1", Type: lineage.NodeASOOpt, Name: "RevenueReport"},
				{ID: "b1", Type: lineage.NodeBO, Name: "OrderBO"},
			}}, nil
		},
	}
	svc := lineage.NewLineageService(repo)
	handler := NewImpactSimulatorHandler(svc)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createSimulatorTestToken(tenant, "user-1", testSimJWTSecret)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/lineage/node/table-orders/blast-radius?depth=3", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var report lineage.BlastRadiusReport
	err := json.Unmarshal(w.Body.Bytes(), &report)
	require.NoError(t, err)
	assert.Equal(t, "table-orders", report.TargetNode)
	assert.Equal(t, "DEPRECATE_OR_MODIFY", report.Action)
	assert.Equal(t, 3, report.BlastRadiusSummary.TotalImpactedArtifacts)
}

func TestGetBlastRadius_DefaultDepth(t *testing.T) {
	t.Setenv("JWT_SECRET", testSimJWTSecret)
	tenant := "tenant-1"
	var capturedDepth int
	repo := &mockLineageRepoForHandler{
		findUpstreamFn: func(ctx context.Context, rootID string, depth int) (*lineage.Graph, error) {
			capturedDepth = depth
			return &lineage.Graph{}, nil
		},
		findDownstreamFn: func(ctx context.Context, rootID string, depth int) (*lineage.Graph, error) {
			capturedDepth = depth
			return &lineage.Graph{}, nil
		},
	}
	svc := lineage.NewLineageService(repo)
	handler := NewImpactSimulatorHandler(svc)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createSimulatorTestToken(tenant, "user-1", testSimJWTSecret)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/lineage/node/node-1/blast-radius", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 5, capturedDepth)
}

func TestGetBlastRadius_InvalidDepthQuery(t *testing.T) {
	t.Setenv("JWT_SECRET", testSimJWTSecret)
	tenant := "tenant-1"
	repo := &mockLineageRepoForHandler{
		findUpstreamFn:   func(ctx context.Context, rootID string, depth int) (*lineage.Graph, error) { return &lineage.Graph{}, nil },
		findDownstreamFn: func(ctx context.Context, rootID string, depth int) (*lineage.Graph, error) { return &lineage.Graph{}, nil },
	}
	svc := lineage.NewLineageService(repo)
	handler := NewImpactSimulatorHandler(svc)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createSimulatorTestToken(tenant, "user-1", testSimJWTSecret)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/lineage/node/node-1/blast-radius?depth=invalid", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSimulateImpact_TenantMismatch(t *testing.T) {
	t.Setenv("JWT_SECRET", testSimJWTSecret)
	tenant := "tenant-1"
	repo := &mockLineageRepoForHandler{}
	svc := lineage.NewLineageService(repo)
	handler := NewImpactSimulatorHandler(svc)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createSimulatorTestToken(tenant, "user-1", testSimJWTSecret)
	body := `{"target_node": "node-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lineage/simulate-impact", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", "different-tenant")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetBlastRadius_TenantMismatch(t *testing.T) {
	t.Setenv("JWT_SECRET", testSimJWTSecret)
	tenant := "tenant-1"
	repo := &mockLineageRepoForHandler{}
	svc := lineage.NewLineageService(repo)
	handler := NewImpactSimulatorHandler(svc)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createSimulatorTestToken(tenant, "user-1", testSimJWTSecret)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/lineage/node/node-1/blast-radius", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", "different-tenant")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

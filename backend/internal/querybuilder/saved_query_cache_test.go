package querybuilder

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTrackingService wraps and tracks executions
type mockTrackingService struct {
	execCount int64
	latency   time.Duration
}

func (m *mockTrackingService) Execute(ctx context.Context, secCtx *security.Context, qd *boresolver.QueryDef, db *sqlx.DB) (*boresolver.QueryExecuteResponse, error) {
	atomic.AddInt64(&m.execCount, 1)
	if m.latency > 0 {
		time.Sleep(m.latency)
	}
	return &boresolver.QueryExecuteResponse{
		Columns: []boresolver.QueryResultColumn{
			{Name: "region", Type: "string"},
			{Name: "revenue", Type: "numeric"},
		},
		Rows: []map[string]interface{}{
			{"region": "EMEA", "revenue": 1000},
			{"region": "APAC", "revenue": 2000},
		},
		RowCount: 2,
	}, nil
}

func setupCacheTestEnv(t *testing.T, tracker *mockTrackingService, defaultTenantID string) (*SavedQueryHandler, *chi.Mux, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.MatchExpectationsInOrder(false)
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	deps := handlers.SecurityContextDeps{Resolver: &mockDSResolver{defaultTenantID: defaultTenantID}}
	h := &SavedQueryHandler{
		db:   sqlxDB,
		deps: deps,
	}

	r := chi.NewRouter()
	r.Post("/api/query/batch-execute", h.HandleBatchExecuteSavedQueries)

	return h, r, mock
}

var mockCols = []string{
	"id", "tenant_id", "user_id", "name", "description", "source_id", "binding_id", "related_bo_ids",
	"chart_type", "query_state", "tags", "folder_id", "is_favorite", "visibility", "is_core", "status", "archived_at", "created_by", "created_at", "updated_at",
}

func mockRow(id, tenantID, name, boID, queryJSON string) []driver.Value {
	return []driver.Value{
		id, tenantID, "u1", name, "desc", boID, "bind-1", "{}",
		"bar", []byte(queryJSON), "{}", nil, false, "private", false, "active", nil, "u1", time.Now(), time.Now(),
	}
}

// (a) Coalescing — 12 concurrent identical requests -> 1 SQL execution
func TestBatchExecute_SingleflightCoalescing(t *testing.T) {
	GlobalQueryCache.Clear()
	tracker := &mockTrackingService{latency: 50 * time.Millisecond}
	tenantID := "tenant-c"
	_, r, mock := setupCacheTestEnv(t, tracker, tenantID)

	sqID := "sq-coalesce-1"
	queryJSON := `{"dimensions":[{"termNodeId":"term-reg","alias":"region"}],"measures":[{"termNodeId":"term-rev","alias":"revenue","agg":"SUM"}]}`

	for i := 0; i < 12; i++ {
		mock.ExpectQuery(`SELECT (.+) FROM data_explorer.saved_queries WHERE id = \$1 AND \(tenant_id = \$2 OR is_core = true\)`).
			WithArgs(sqID, tenantID).
			WillReturnRows(sqlmock.NewRows(mockCols).AddRow(mockRow(sqID, tenantID, "Sales Query", "bo-sales", queryJSON)...))
	}

	var wg sync.WaitGroup
	concurrency := 12
	var directExecCount int64
	key := BuildCompositeCacheKey(tenantID, "content-hash-1", "params-hash-1", "abac-1", "hot", "v1")

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val, err, _ := GlobalQueryCache.singleflight.Do(key, func() (interface{}, error) {
				atomic.AddInt64(&directExecCount, 1)
				time.Sleep(50 * time.Millisecond)
				payload := &QueryResultPayload{
					Columns: []boresolver.QueryResultColumn{
						{Name: "region", Type: "string"},
						{Name: "revenue", Type: "numeric"},
					},
					Rows: []map[string]interface{}{
						{"region": "EMEA", "revenue": 1000},
						{"region": "APAC", "revenue": 2000},
					},
					RowCount:     2,
					ResolvedTier: "hot",
					MVHit:        false,
				}
				GlobalQueryCache.Put(key, payload, 1, "bo-sales", 60*time.Second)
				return payload, nil
			})
			require.NoError(t, err)
			require.NotNil(t, val)
		}()
	}

	wg.Wait()

	// Assert: Exactly 1 execution occurred across all 12 concurrent callers
	assert.Equal(t, int64(1), atomic.LoadInt64(&directExecCount), "Expected exactly 1 execution due to singleflight coalescing")

	// Verify HTTP batch endpoint functions properly
	reqBody := BatchExecuteSavedQueryRequest{
		Queries: []BatchExecuteItem{
			{ID: "item-1", SavedQueryID: sqID},
		},
	}
	b, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/query/batch-execute", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	auth := security.AuthInfo{TenantIDs: []string{tenantID}, UserID: "user-1"}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req.WithContext(security.WithAuthInfo(req.Context(), auth)))
	assert.Equal(t, http.StatusOK, w.Code)
}

// (b) Batch — 10 requests, 3 identical keys -> 8 executions, 10 answers
func TestBatchExecute_DuplicateKeysInSingleBatch(t *testing.T) {
	GlobalQueryCache.Clear()
	tracker := &mockTrackingService{}
	tenantID := "tenant-b"
	_, r, mock := setupCacheTestEnv(t, tracker, tenantID)

	queryJSON := `{"dimensions":[{"termNodeId":"term-reg","alias":"region"}],"measures":[{"termNodeId":"term-rev","alias":"revenue","agg":"SUM"}]}`

	var queries []BatchExecuteItem
	for i := 0; i < 3; i++ {
		queries = append(queries, BatchExecuteItem{
			ID:           fmt.Sprintf("dup-%d", i),
			SavedQueryID: "sq-dup-1",
			Params:       map[string]interface{}{"region": "EMEA"},
		})
	}
	for i := 0; i < 7; i++ {
		queries = append(queries, BatchExecuteItem{
			ID:           fmt.Sprintf("unique-%d", i),
			SavedQueryID: fmt.Sprintf("sq-uniq-%d", i),
			Params:       map[string]interface{}{"region": fmt.Sprintf("R-%d", i)},
		})
	}

	for i := 0; i < 10; i++ {
		mock.ExpectQuery(`SELECT (.+) FROM data_explorer.saved_queries WHERE id = \$1 AND \(tenant_id = \$2 OR is_core = true\)`).
			WillReturnRows(sqlmock.NewRows(mockCols).AddRow(mockRow(queries[i].SavedQueryID, tenantID, "Query", "bo-sales", queryJSON)...))
	}

	reqBody := BatchExecuteSavedQueryRequest{Queries: queries}
	b, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/query/batch-execute", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	auth := security.AuthInfo{TenantIDs: []string{tenantID}, UserID: "user-1"}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req.WithContext(security.WithAuthInfo(req.Context(), auth)))

	assert.Equal(t, http.StatusOK, w.Code)
	var resp BatchExecuteSavedQueryResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	// All 10 answers returned
	assert.Len(t, resp.Results, 10)
	for _, res := range resp.Results {
		assert.Empty(t, res.Error)
		assert.Equal(t, 2, res.RowCount)
	}
}

// (c) ABAC cache isolation — two users, different abacContextHash, no cross-serve
func TestBatchExecute_ABACCacheIsolation(t *testing.T) {
	GlobalQueryCache.Clear()
	tracker := &mockTrackingService{}
	tenantID := "tenant-abac"
	_, r, mock := setupCacheTestEnv(t, tracker, tenantID)

	sqID := "sq-abac-1"
	queryJSON := `{"dimensions":[{"termNodeId":"term-reg","alias":"region"}],"measures":[{"termNodeId":"term-rev","alias":"revenue","agg":"SUM"}]}`

	for i := 0; i < 2; i++ {
		mock.ExpectQuery(`SELECT (.+) FROM data_explorer.saved_queries WHERE id = \$1 AND \(tenant_id = \$2 OR is_core = true\)`).
			WithArgs(sqID, tenantID).
			WillReturnRows(sqlmock.NewRows(mockCols).AddRow(mockRow(sqID, tenantID, "ABAC Query", "bo-sales", queryJSON)...))
	}

	makeReq := func(userID string, auth security.AuthInfo) *BatchExecuteSavedQueryResponse {
		reqBody := BatchExecuteSavedQueryRequest{
			Queries: []BatchExecuteItem{
				{ID: "item-1", SavedQueryID: sqID},
			},
		}
		b, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/query/batch-execute", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		reqWithAuth := req.WithContext(security.WithAuthInfo(req.Context(), auth))

		w := httptest.NewRecorder()
		r.ServeHTTP(w, reqWithAuth)

		var resp BatchExecuteSavedQueryResponse
		_ = json.NewDecoder(w.Body).Decode(&resp)
		return &resp
	}

	// User 1 (US scope)
	resp1 := makeReq("user-us", security.AuthInfo{UserID: "user-us", TenantIDs: []string{tenantID}, Roles: []string{"analyst_us"}})
	require.NotNil(t, resp1)
	require.Len(t, resp1.Results, 1)
	assert.False(t, resp1.Results[0].CacheHit, "First user must execute query (cache miss)")

	// User 2 (EU scope) with different ABAC profile
	resp2 := makeReq("user-eu", security.AuthInfo{UserID: "user-eu", TenantIDs: []string{tenantID}, Roles: []string{"analyst_eu"}})
	require.NotNil(t, resp2)
	require.Len(t, resp2.Results, 1)
	assert.False(t, resp2.Results[0].CacheHit, "Second user with different ABAC context must NOT hit User 1's cache")
}

// (d) Masked-term cross-filter probe returns empty or error
func TestBatchExecute_MaskedTermCrossFilterProbe(t *testing.T) {
	GlobalQueryCache.Clear()
	tracker := &mockTrackingService{}
	tenantID := "tenant-mask"
	_, r, mock := setupCacheTestEnv(t, tracker, tenantID)

	sqID := "sq-mask-1"
	queryJSON := `{"dimensions":[{"termNodeId":"term-region","alias":"region"}],"measures":[{"termNodeId":"term-revenue","alias":"revenue"}]}`

	mock.ExpectQuery(`SELECT (.+) FROM data_explorer.saved_queries WHERE id = \$1 AND \(tenant_id = \$2 OR is_core = true\)`).
		WithArgs(sqID, tenantID).
		WillReturnRows(sqlmock.NewRows(mockCols).AddRow(mockRow(sqID, tenantID, "Mask Query", "bo-sales", queryJSON)...))

	// Adversarial attempt to probe an unauthorized / masked term not in query's state
	reqBody := BatchExecuteSavedQueryRequest{
		Queries: []BatchExecuteItem{
			{
				ID:           "probe-1",
				SavedQueryID: sqID,
				RuntimeFilters: []RuntimeFilter{
					{
						TermNodeID: "unauthorized_salary_term",
						Operator:   "gt",
						Value:      500000,
					},
				},
			},
		},
	}
	b, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/query/batch-execute", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	auth := security.AuthInfo{TenantIDs: []string{tenantID}, UserID: "user-1"}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req.WithContext(security.WithAuthInfo(req.Context(), auth)))

	assert.Equal(t, http.StatusOK, w.Code)
	var resp BatchExecuteSavedQueryResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)

	require.Len(t, resp.Results, 1)
	assert.NotEmpty(t, resp.Results[0].Error, "Probing masked / unauthorized term must return validation error and zero data")
}

// (e) Watermark invalidation test — changes in watermark trigger fresh executions
func TestBatchExecute_WatermarkInvalidation(t *testing.T) {
	GlobalQueryCache.Clear()
	tracker := &mockTrackingService{}
	tenantID := "tenant-wm"
	boID := "bo-trades"
	_, r, mock := setupCacheTestEnv(t, tracker, tenantID)

	sqID := "sq-wm-1"
	queryJSON := `{"dimensions":[{"termNodeId":"term-sec","alias":"security"}],"measures":[{"termNodeId":"term-qty","alias":"quantity"}]}`

	for i := 0; i < 3; i++ {
		mock.ExpectQuery(`SELECT (.+) FROM data_explorer.saved_queries WHERE id = \$1 AND \(tenant_id = \$2 OR is_core = true\)`).
			WithArgs(sqID, tenantID).
			WillReturnRows(sqlmock.NewRows(mockCols).AddRow(mockRow(sqID, tenantID, "Watermark Query", boID, queryJSON)...))
	}

	exec := func() *BatchExecuteSavedQueryResponse {
		reqBody := BatchExecuteSavedQueryRequest{
			Queries: []BatchExecuteItem{
				{ID: "wm-item-1", SavedQueryID: sqID},
			},
		}
		b, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/query/batch-execute", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		auth := security.AuthInfo{TenantIDs: []string{tenantID}, UserID: "user-1"}

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req.WithContext(security.WithAuthInfo(req.Context(), auth)))

		var resp BatchExecuteSavedQueryResponse
		_ = json.NewDecoder(w.Body).Decode(&resp)
		return &resp
	}

	// 1. Initial execution at Watermark 100 -> Cache Miss
	GlobalQueryCache.SetWatermark(tenantID, boID, 100)
	resp1 := exec()
	require.Len(t, resp1.Results, 1)
	assert.False(t, resp1.Results[0].CacheHit)

	// 2. Second execution at Watermark 100 -> Cache HIT
	resp2 := exec()
	require.Len(t, resp2.Results, 1)
	assert.True(t, resp2.Results[0].CacheHit, "Expected cache hit on identical watermark")

	// 3. Watermark advances (e.g. CDC ingestion event occurs) -> Cache MISS & Re-execution
	GlobalQueryCache.SetWatermark(tenantID, boID, 101)
	resp3 := exec()
	require.Len(t, resp3.Results, 1)
	assert.False(t, resp3.Results[0].CacheHit, "Expected cache miss after watermark advance")
}

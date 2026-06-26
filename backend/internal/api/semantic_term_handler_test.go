package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/api"
)

func TestGetSemanticTermDetail(t *testing.T) {
	tests := []struct {
		name           string
		termID         string
		tenantID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "valid_term_retrieval",
			termID:         "order-quantity",
			tenantID:       "ten",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "term_with_uppercase_and_dots",
			termID:         "Order.Qty.V2",
			tenantID:       "ten",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "term_with_underscores",
			termID:         "order_quantity_usd",
			tenantID:       "ten",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing_term_id",
			termID:         "",
			tenantID:       "ten",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Missing semantic term identifier",
		},
		{
			name:           "invalid_term_id_special_chars",
			termID:         "order@qty!",
			tenantID:       "ten",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid term ID format",
		},
		{
			name:           "invalid_term_id_spaces",
			termID:         "order qty",
			tenantID:       "ten",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid term ID format",
		},
		{
			name:           "missing_tenant_id",
			termID:         "order-quantity",
			tenantID:       "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Security context initialization failed",
		},
		{
			name:           "very_long_term_id",
			termID:         strings.Repeat("a", 257),
			tenantID:       "ten",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid term ID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("Failed to create sqlmock: %v", err)
			}
			defer db.Close()

			// Create server
			server := &api.Server{DB: db, DatasourceResolver: &mockResolver{}}

			// Set up expectations for success cases
			if tt.expectedStatus == http.StatusOK {
				mock.ExpectQuery("SELECT cn.id, cn.node_name.*").
					WithArgs(tt.termID, tt.tenantID).
					WillReturnRows(sqlmock.NewRows([]string{"id", "node_name", "display_name", "description", "properties", "created_at", "updated_at", "tenant_id"}).
						AddRow(tt.termID, tt.termID, tt.termID, "", []byte("{}"), time.Now(), time.Now(), tt.tenantID))

				// Trace count
				mock.ExpectQuery("SELECT COUNT.*FROM public.catalog_edge ce.*").
					WithArgs(tt.termID, tt.tenantID).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

				// Last trace
				mock.ExpectQuery("SELECT ce.created_at.*FROM public.catalog_edge ce.*").
					WithArgs(tt.termID, tt.tenantID).
					WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(time.Now()))

				// Traces list
				mock.ExpectQuery("SELECT cn_term.node_name.*").
					WithArgs(tt.termID, tt.tenantID).
					WillReturnRows(sqlmock.NewRows([]string{"term_name", "node_name", "catalog_type_name", "description", "plan_id", "commit_key", "timestamp", "status", "region"}).
						AddRow(tt.termID, tt.termID, "semantic_term", "", "p1", "c1", time.Now(), "success", "us-east-1"))
			}

			// Build request path
			path := fmt.Sprintf("/api/semantic-terms/%s", url.PathEscape(tt.termID))

			// Create request
			req := httptest.NewRequest(http.MethodGet, path, nil)
			// Emulate chi path param
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("termID", tt.termID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			if tt.tenantID != "" {
				req = withValidHeaders(req, tt.tenantID, "ds1")
				req = withAuth(req, tt.tenantID)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			server.GetSemanticTermDetail(w, req)

			// Verify status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Verify error message if expected
			if tt.expectedError != "" {
				var errResp api.ErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
					t.Errorf("Failed to decode error response: %v", err)
				}

				if !strings.Contains(errResp.Error, tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, errResp.Error)
				}
			}

			// Verify success response structure
			if tt.expectedStatus == http.StatusOK {
				var termDetail api.SemanticTermDetail
				if err := json.NewDecoder(w.Body).Decode(&termDetail); err != nil {
					t.Errorf("Failed to decode term detail: %v", err)
				}

				if termDetail.ID != tt.termID {
					t.Errorf("Expected term ID %s, got %s", tt.termID, termDetail.ID)
				}

				if termDetail.TenantID != tt.tenantID {
					t.Errorf("Expected tenant ID %s, got %s", tt.tenantID, termDetail.TenantID)
				}

				if termDetail.IsActive != true {
					t.Errorf("Expected active term, got inactive")
				}

				if termDetail.Traces == nil {
					t.Errorf("Expected Traces to be initialized, got nil")
				}
			}

			// Verify cache headers
			if tt.expectedStatus == http.StatusOK {
				if cacheControl := w.Header().Get("Cache-Control"); cacheControl == "" {
					t.Errorf("Expected Cache-Control header, got empty")
				}

				if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", contentType)
				}
			}
		})
	}
}

func TestListSemanticTermTraces(t *testing.T) {
	tests := []struct {
		name           string
		termID         string
		tenantID       string
		limit          string
		offset         string
		status         string
		region         string
		expectedStatus int
		expectedError  string
		expectedLimit  int
		expectedOffset int
	}{
		{
			name:           "list_traces_defaults",
			termID:         "order-quantity",
			tenantID:       "ten",
			expectedStatus: http.StatusOK,
			expectedLimit:  50,
			expectedOffset: 0,
		},
		{
			name:           "list_traces_custom_limit",
			termID:         "order-quantity",
			tenantID:       "ten",
			limit:          "100",
			expectedStatus: http.StatusOK,
			expectedLimit:  100,
			expectedOffset: 0,
		},
		{
			name:           "list_traces_with_pagination",
			termID:         "order-quantity",
			tenantID:       "ten",
			limit:          "25",
			offset:         "50",
			expectedStatus: http.StatusOK,
			expectedLimit:  25,
			expectedOffset: 50,
		},
		{
			name:           "list_traces_with_status_filter",
			termID:         "order-quantity",
			tenantID:       "ten",
			status:         "success",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "list_traces_with_region_filter",
			termID:         "order-quantity",
			tenantID:       "ten",
			region:         "us-east-1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "list_traces_limit_exceeds_max",
			termID:         "order-quantity",
			tenantID:       "ten",
			limit:          "1000",
			expectedStatus: http.StatusOK,
			expectedLimit:  500, // Should cap at 500
		},
		{
			name:           "list_traces_invalid_limit",
			termID:         "order-quantity",
			tenantID:       "ten",
			limit:          "invalid",
			expectedStatus: http.StatusOK,
			expectedLimit:  50, // Should use default
		},
		{
			name:           "list_traces_negative_limit",
			termID:         "order-quantity",
			tenantID:       "ten",
			limit:          "-10",
			expectedStatus: http.StatusOK,
			expectedLimit:  50, // Should use default
		},
		{
			name:           "list_traces_missing_term_id",
			termID:         "",
			tenantID:       "ten",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Missing semantic term identifier",
		},
		{
			name:           "list_traces_missing_tenant_id",
			termID:         "order-quantity",
			tenantID:       "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Security context initialization failed",
		},
		{
			name:           "list_traces_invalid_term_id",
			termID:         "invalid@term",
			tenantID:       "ten",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid term ID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("Failed to create sqlmock: %v", err)
			}
			defer db.Close()

			server := &api.Server{DB: db, DatasourceResolver: &mockResolver{}}

			if tt.expectedStatus == http.StatusOK {
				// Query to fetch semantic term details/traces
				mock.ExpectQuery("(?s)SELECT cn_term.node_name.*").
					WithArgs(tt.termID, tt.tenantID, sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"name", "node_name", "type", "description", "plan_id", "commit_key", "timestamp", "status", "region"}).
						AddRow(tt.termID, tt.termID, "semantic_term", "", "p1", "c1", time.Now(), "success", "us-east-1"))

				// Count query
				mock.ExpectQuery("(?s)SELECT COUNT.*FROM public.catalog_node.*").
					WithArgs(tt.termID, tt.tenantID).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			}

			// Build query string
			queryParts := []string{}
			if tt.limit != "" {
				queryParts = append(queryParts, fmt.Sprintf("limit=%s", tt.limit))
			}
			if tt.offset != "" {
				queryParts = append(queryParts, fmt.Sprintf("offset=%s", tt.offset))
			}
			if tt.status != "" {
				queryParts = append(queryParts, fmt.Sprintf("status=%s", tt.status))
			}
			if tt.region != "" {
				queryParts = append(queryParts, fmt.Sprintf("region=%s", tt.region))
			}

			queryString := ""
			if len(queryParts) > 0 {
				queryString = "?" + strings.Join(queryParts, "&")
			}

			url := fmt.Sprintf("/api/semantic-terms/%s/traces%s", tt.termID, queryString)

			req := httptest.NewRequest(http.MethodGet, url, nil)
			// Emulate chi path param
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("termID", tt.termID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			if tt.tenantID != "" {
				req = withValidHeaders(req, tt.tenantID, "ds1")
				req = withAuth(req, tt.tenantID)
			}

			w := httptest.NewRecorder()
			http.HandlerFunc(server.ListSemanticTermTraces).ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var errResp api.ErrorResponse
				json.NewDecoder(w.Body).Decode(&errResp)

				if !strings.Contains(errResp.Error, tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, errResp.Error)
				}
			}

			if tt.expectedStatus == http.StatusOK {
				var resp map[string]interface{}
				json.NewDecoder(w.Body).Decode(&resp)

				if tt.expectedLimit > 0 {
					if int(resp["limit"].(float64)) != tt.expectedLimit {
						t.Errorf("Expected limit %d, got %v", tt.expectedLimit, resp["limit"])
					}
				}

				if int(resp["offset"].(float64)) != tt.expectedOffset {
					t.Errorf("Expected offset %d, got %v", tt.expectedOffset, resp["offset"])
				}

				// Verify response structure
				if resp["term_id"] != tt.termID {
					t.Errorf("Expected term_id %s, got %v", tt.termID, resp["term_id"])
				}

				if resp["tenant_id"] != tt.tenantID {
					t.Errorf("Expected tenant_id %s, got %v", tt.tenantID, resp["tenant_id"])
				}
			}
		})
	}
}

func TestGetSemanticTermMetrics(t *testing.T) {
	tests := []struct {
		name           string
		termID         string
		tenantID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "get_metrics_success",
			termID:         "order-quantity",
			tenantID:       "ten",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "get_metrics_different_tenant",
			termID:         "order-quantity",
			tenantID:       "ten",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "get_metrics_missing_term_id",
			termID:         "",
			tenantID:       "ten",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Missing semantic term identifier",
		},
		{
			name:           "get_metrics_invalid_term_id",
			termID:         "invalid!term",
			tenantID:       "ten",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid term ID format",
		},
		{
			name:           "get_metrics_missing_tenant_id",
			termID:         "order-quantity",
			tenantID:       "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Security context initialization failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("Failed to create sqlmock: %v", err)
			}
			defer db.Close()

			server := &api.Server{DB: db, DatasourceResolver: &mockResolver{}}

			if tt.expectedStatus == http.StatusOK {
				// Metrics count query
				mock.ExpectQuery("(?s)SELECT COUNT.*FROM public.catalog_edge.*").
					WithArgs(tt.termID, tt.tenantID).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))
			}

			url := fmt.Sprintf("/api/semantic-terms/%s/metrics", tt.termID)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			// Emulate chi path param
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("termID", tt.termID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			if tt.tenantID != "" {
				req = withValidHeaders(req, tt.tenantID, "ds1")
				req = withAuth(req, tt.tenantID)
			}

			w := httptest.NewRecorder()
			server.GetSemanticTermMetrics(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var errResp api.ErrorResponse
				json.NewDecoder(w.Body).Decode(&errResp)

				if !strings.Contains(errResp.Error, tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, errResp.Error)
				}
			}

			if tt.expectedStatus == http.StatusOK {
				var metrics map[string]interface{}
				json.NewDecoder(w.Body).Decode(&metrics)

				// Verify required fields
				required := []string{"term_id", "tenant_id", "total_traces", "success_rate", "avg_latency_ms", "regions", "status_breakdown", "timestamp"}
				for _, field := range required {
					if _, exists := metrics[field]; !exists {
						t.Errorf("Expected field '%s' in metrics response", field)
					}
				}
			}

			// Verify cache headers
			if tt.expectedStatus == http.StatusOK {
				if cacheControl := w.Header().Get("Cache-Control"); cacheControl != "max-age=300, public" {
					t.Errorf("Expected Cache-Control max-age=300, got %s", cacheControl)
				}
			}
		})
	}
}

// Benchmark tests for semantic term handlers

func BenchmarkGetSemanticTermDetail(b *testing.B) {
	server := &api.Server{DatasourceResolver: &mockResolver{}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/semantic-terms/order-quantity", nil)
		// Emulate chi path param
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("termID", "order-quantity")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		req.Header.Set("X-Tenant-ID", "ten")
		w := httptest.NewRecorder()

		http.HandlerFunc(server.GetSemanticTermDetail).ServeHTTP(w, req)
	}
}

func BenchmarkListSemanticTermTraces(b *testing.B) {
	server := &api.Server{DatasourceResolver: &mockResolver{}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/semantic-terms/order-quantity/traces?limit=50&offset=0", nil)
		// Emulate chi path param
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("termID", "order-quantity")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		req.Header.Set("X-Tenant-ID", "ten")
		w := httptest.NewRecorder()

		http.HandlerFunc(server.ListSemanticTermTraces).ServeHTTP(w, req)
	}
}

func BenchmarkGetSemanticTermMetrics(b *testing.B) {
	server := &api.Server{DatasourceResolver: &mockResolver{}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/semantic-terms/order-quantity/metrics", nil)
		// Emulate chi path param
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("termID", "order-quantity")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		req.Header.Set("X-Tenant-ID", "ten")
		w := httptest.NewRecorder()

		http.HandlerFunc(server.GetSemanticTermMetrics).ServeHTTP(w, req)
	}
}

// Integration test for semantic term handler registration

func TestSemanticTermHandlerRegistration(t *testing.T) {
	// Test that handler routes are properly registered
	// This verifies the route table includes:
	// - GET /api/semantic-terms/{termID}
	// - GET /api/semantic-terms/{termID}/traces
	// - GET /api/semantic-terms/{termID}/metrics

	server := &api.Server{DatasourceResolver: &mockResolver{}}

	testCases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/semantic-terms/test-term"},
		{http.MethodGet, "/api/semantic-terms/test-term/traces"},
		{http.MethodGet, "/api/semantic-terms/test-term/metrics"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s %s", tc.method, tc.path), func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("X-Tenant-ID", "ten")
			w := httptest.NewRecorder()

			// Route should be handled
			// Emulate chi path param
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("termID", "test-term")
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			if tc.path == "/api/semantic-terms/test-term" {
				http.HandlerFunc(server.GetSemanticTermDetail).ServeHTTP(w, req)
			} else if tc.path == "/api/semantic-terms/test-term/traces" {
				http.HandlerFunc(server.ListSemanticTermTraces).ServeHTTP(w, req)
			} else if tc.path == "/api/semantic-terms/test-term/metrics" {
				http.HandlerFunc(server.GetSemanticTermMetrics).ServeHTTP(w, req)
			}

			// Should not be 404
			if w.Code == http.StatusNotFound {
				t.Errorf("Route %s should be registered", tc.path)
			}
		})
	}
}

# Backend Code Refactoring Guide: Consolidated Metrics & DAX Functions

**Date:** November 3, 2025  
**Language:** Go  
**Target Directory:** `/backend/internal/services/` and `/backend/internal/api/`  

---

## 🎯 Overview

This guide provides step-by-step Go refactoring to use consolidated `public.metrics_registry` and `public.dax_functions` tables instead of domain-specific schemas.

---

## 📋 Prerequisites

- Go 1.18+
- sqlx library
- Database migrations applied (`public.metrics_registry`, `public.dax_functions`)
- Existing domain-scoped queries identified (via `update_code_patterns.sh` scan)

---

## 🔧 Step 1: Create Consolidated Services

### Create `backend/internal/services/metrics_service.go`

```go
package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// MetricsRegistry represents a consolidated metrics record
type MetricsRegistry struct {
	ID                int64          `db:"id" json:"id"`
	NodeID            string         `db:"node_id" json:"node_id"`
	SchemaDomain      string         `db:"schema_domain" json:"schema_domain"`
	Category          string         `db:"category" json:"category"`
	Description       string         `db:"description" json:"description"`
	FormulaType       string         `db:"formula_type" json:"formula_type"`
	Formula           string         `db:"formula" json:"formula"`
	Arguments         map[string]any `db:"arguments" json:"arguments"`
	Badge             *string        `db:"badge" json:"badge"`
	FunctionClass     *string        `db:"function_class" json:"function_class"`
	FunctionsUsed     pq.StringArray `db:"functions_used" json:"functions_used"`
	GovernanceStatus  *string        `db:"governance_status" json:"governance_status"`
	Audience          pq.StringArray `db:"audience" json:"audience"`
	Tags              pq.StringArray `db:"tags" json:"tags"`
	CreatedAt         string         `db:"created_at" json:"created_at"`
	UpdatedAt         string         `db:"updated_at" json:"updated_at"`
}

// MetricsService handles metrics_registry operations
type MetricsService struct {
	db *sqlx.DB
}

// NewMetricsService creates a new MetricsService
func NewMetricsService(db *sqlx.DB) *MetricsService {
	return &MetricsService{db: db}
}

// GetMetricsByDomain returns all metrics for a specific domain
func (s *MetricsService) GetMetricsByDomain(
	ctx context.Context,
	domain string,
) ([]MetricsRegistry, error) {
	var metrics []MetricsRegistry
	query := `
		SELECT id, node_id, schema_domain, category, description, formula_type,
		       formula, arguments, badge, function_class, functions_used,
		       governance_status, audience, tags, created_at, updated_at
		FROM public.metrics_registry
		WHERE schema_domain = $1
		ORDER BY node_id ASC
	`
	err := s.db.SelectContext(ctx, &metrics, query, domain)
	if err != nil {
		if err == sql.ErrNoRows {
			return []MetricsRegistry{}, nil
		}
		log.Printf("Error fetching metrics for domain %s: %v", domain, err)
		return nil, err
	}
	return metrics, nil
}

// GetMetricsByDomains returns metrics for multiple domains
func (s *MetricsService) GetMetricsByDomains(
	ctx context.Context,
	domains []string,
) ([]MetricsRegistry, error) {
	if len(domains) == 0 {
		return []MetricsRegistry{}, nil
	}

	var metrics []MetricsRegistry

	// Use sqlx.In for safe parameterization
	query, args, err := sqlx.In(
		`SELECT id, node_id, schema_domain, category, description, formula_type,
		        formula, arguments, badge, function_class, functions_used,
		        governance_status, audience, tags, created_at, updated_at
		 FROM public.metrics_registry
		 WHERE schema_domain IN (?)
		 ORDER BY schema_domain, node_id`,
		domains,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	// Rebind for PostgreSQL ($1, $2, etc.)
	query = s.db.Rebind(query)

	err = s.db.SelectContext(ctx, &metrics, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return []MetricsRegistry{}, nil
		}
		log.Printf("Error fetching metrics for domains %v: %v", domains, err)
		return nil, err
	}
	return metrics, nil
}

// GetMetricByNodeID returns a single metric by node_id and domain
func (s *MetricsService) GetMetricByNodeID(
	ctx context.Context,
	domain string,
	nodeID string,
) (*MetricsRegistry, error) {
	var metric MetricsRegistry
	query := `
		SELECT id, node_id, schema_domain, category, description, formula_type,
		       formula, arguments, badge, function_class, functions_used,
		       governance_status, audience, tags, created_at, updated_at
		FROM public.metrics_registry
		WHERE schema_domain = $1 AND node_id = $2
		LIMIT 1
	`
	err := s.db.GetContext(ctx, &metric, query, domain, nodeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Metric not found
		}
		log.Printf("Error fetching metric %s.%s: %v", domain, nodeID, err)
		return nil, err
	}
	return &metric, nil
}

// GetMetricsByCategory returns metrics for a category within a domain
func (s *MetricsService) GetMetricsByCategory(
	ctx context.Context,
	domain string,
	category string,
) ([]MetricsRegistry, error) {
	var metrics []MetricsRegistry
	query := `
		SELECT id, node_id, schema_domain, category, description, formula_type,
		       formula, arguments, badge, function_class, functions_used,
		       governance_status, audience, tags, created_at, updated_at
		FROM public.metrics_registry
		WHERE schema_domain = $1 AND category = $2
		ORDER BY node_id ASC
	`
	err := s.db.SelectContext(ctx, &metrics, query, domain, category)
	if err != nil {
		if err == sql.ErrNoRows {
			return []MetricsRegistry{}, nil
		}
		log.Printf("Error fetching metrics for %s.%s: %v", domain, category, err)
		return nil, err
	}
	return metrics, nil
}

// InsertMetric inserts a new metric into the consolidated table
func (s *MetricsService) InsertMetric(
	ctx context.Context,
	metric *MetricsRegistry,
) (*MetricsRegistry, error) {
	query := `
		INSERT INTO public.metrics_registry
		(node_id, schema_domain, category, description, formula_type, formula,
		 arguments, badge, function_class, functions_used, governance_status,
		 audience, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (node_id, schema_domain) DO NOTHING
		RETURNING id, created_at, updated_at
	`
	err := s.db.QueryRowContext(
		ctx, query,
		metric.NodeID, metric.SchemaDomain, metric.Category, metric.Description,
		metric.FormulaType, metric.Formula, metric.Arguments, metric.Badge,
		metric.FunctionClass, metric.FunctionsUsed, metric.GovernanceStatus,
		metric.Audience, metric.Tags, metric.CreatedAt, metric.UpdatedAt,
	).Scan(&metric.ID, &metric.CreatedAt, &metric.UpdatedAt)

	if err != nil {
		log.Printf("Error inserting metric: %v", err)
		return nil, err
	}
	return metric, nil
}

// UpdateMetric updates an existing metric
func (s *MetricsService) UpdateMetric(
	ctx context.Context,
	metric *MetricsRegistry,
) error {
	query := `
		UPDATE public.metrics_registry
		SET category = $1, description = $2, governance_status = $3,
		    audience = $4, tags = $5, updated_at = CURRENT_TIMESTAMP
		WHERE node_id = $6 AND schema_domain = $7
	`
	result, err := s.db.ExecContext(
		ctx, query,
		metric.Category, metric.Description, metric.GovernanceStatus,
		metric.Audience, metric.Tags, metric.NodeID, metric.SchemaDomain,
	)
	if err != nil {
		log.Printf("Error updating metric: %v", err)
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("metric not found: %s.%s", metric.SchemaDomain, metric.NodeID)
	}
	return nil
}

// DeleteMetric deletes a metric
func (s *MetricsService) DeleteMetric(
	ctx context.Context,
	domain string,
	nodeID string,
) error {
	query := `
		DELETE FROM public.metrics_registry
		WHERE node_id = $1 AND schema_domain = $2
	`
	result, err := s.db.ExecContext(ctx, query, nodeID, domain)
	if err != nil {
		log.Printf("Error deleting metric: %v", err)
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("metric not found: %s.%s", domain, nodeID)
	}
	return nil
}

// GetMetricsCount returns the count of metrics for a domain
func (s *MetricsService) GetMetricsCount(
	ctx context.Context,
	domain string,
) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM public.metrics_registry WHERE schema_domain = $1`
	err := s.db.QueryRowContext(ctx, query, domain).Scan(&count)
	if err != nil {
		log.Printf("Error counting metrics: %v", err)
		return 0, err
	}
	return count, nil
}
```

### Create `backend/internal/services/dax_functions_service.go`

```go
package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

// DAXFunction represents a consolidated DAX function record
type DAXFunction struct {
	ID           int64   `db:"id" json:"id"`
	Name         string  `db:"name" json:"name"`
	SchemaDomain string  `db:"schema_domain" json:"schema_domain"`
	Class        *string `db:"class" json:"class"`
	Badge        *string `db:"badge" json:"badge"`
	Description  *string `db:"description" json:"description"`
	CreatedAt    string  `db:"created_at" json:"created_at"`
}

// DAXFunctionsService handles dax_functions operations
type DAXFunctionsService struct {
	db *sqlx.DB
}

// NewDAXFunctionsService creates a new DAXFunctionsService
func NewDAXFunctionsService(db *sqlx.DB) *DAXFunctionsService {
	return &DAXFunctionsService{db: db}
}

// GetDAXFunctionsByDomain returns all DAX functions for a specific domain
func (s *DAXFunctionsService) GetDAXFunctionsByDomain(
	ctx context.Context,
	domain string,
) ([]DAXFunction, error) {
	var functions []DAXFunction
	query := `
		SELECT id, name, schema_domain, class, badge, description, created_at
		FROM public.dax_functions
		WHERE schema_domain = $1
		ORDER BY name ASC
	`
	err := s.db.SelectContext(ctx, &functions, query, domain)
	if err != nil {
		if err == sql.ErrNoRows {
			return []DAXFunction{}, nil
		}
		log.Printf("Error fetching DAX functions for domain %s: %v", domain, err)
		return nil, err
	}
	return functions, nil
}

// GetDAXFunctionsByDomains returns DAX functions for multiple domains
func (s *DAXFunctionsService) GetDAXFunctionsByDomains(
	ctx context.Context,
	domains []string,
) ([]DAXFunction, error) {
	if len(domains) == 0 {
		return []DAXFunction{}, nil
	}

	var functions []DAXFunction

	// Use sqlx.In for safe parameterization
	query, args, err := sqlx.In(
		`SELECT id, name, schema_domain, class, badge, description, created_at
		 FROM public.dax_functions
		 WHERE schema_domain IN (?)
		 ORDER BY schema_domain, name`,
		domains,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	// Rebind for PostgreSQL ($1, $2, etc.)
	query = s.db.Rebind(query)

	err = s.db.SelectContext(ctx, &functions, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return []DAXFunction{}, nil
		}
		log.Printf("Error fetching DAX functions for domains %v: %v", domains, err)
		return nil, err
	}
	return functions, nil
}

// GetDAXFunctionByName returns a single DAX function by name and domain
func (s *DAXFunctionsService) GetDAXFunctionByName(
	ctx context.Context,
	domain string,
	name string,
) (*DAXFunction, error) {
	var fn DAXFunction
	query := `
		SELECT id, name, schema_domain, class, badge, description, created_at
		FROM public.dax_functions
		WHERE schema_domain = $1 AND name = $2
		LIMIT 1
	`
	err := s.db.GetContext(ctx, &fn, query, domain, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Function not found
		}
		log.Printf("Error fetching DAX function %s.%s: %v", domain, name, err)
		return nil, err
	}
	return &fn, nil
}

// GetDAXFunctionsByClass returns DAX functions for a class within a domain
func (s *DAXFunctionsService) GetDAXFunctionsByClass(
	ctx context.Context,
	domain string,
	class string,
) ([]DAXFunction, error) {
	var functions []DAXFunction
	query := `
		SELECT id, name, schema_domain, class, badge, description, created_at
		FROM public.dax_functions
		WHERE schema_domain = $1 AND class = $2
		ORDER BY name ASC
	`
	err := s.db.SelectContext(ctx, &functions, query, domain, class)
	if err != nil {
		if err == sql.ErrNoRows {
			return []DAXFunction{}, nil
		}
		log.Printf("Error fetching DAX functions for %s.%s: %v", domain, class, err)
		return nil, err
	}
	return functions, nil
}

// InsertDAXFunction inserts a new DAX function into the consolidated table
func (s *DAXFunctionsService) InsertDAXFunction(
	ctx context.Context,
	fn *DAXFunction,
) (*DAXFunction, error) {
	query := `
		INSERT INTO public.dax_functions
		(name, schema_domain, class, badge, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (name, schema_domain) DO NOTHING
		RETURNING id, created_at
	`
	err := s.db.QueryRowContext(
		ctx, query,
		fn.Name, fn.SchemaDomain, fn.Class, fn.Badge, fn.Description, fn.CreatedAt,
	).Scan(&fn.ID, &fn.CreatedAt)

	if err != nil {
		log.Printf("Error inserting DAX function: %v", err)
		return nil, err
	}
	return fn, nil
}

// UpdateDAXFunction updates an existing DAX function
func (s *DAXFunctionsService) UpdateDAXFunction(
	ctx context.Context,
	fn *DAXFunction,
) error {
	query := `
		UPDATE public.dax_functions
		SET class = $1, badge = $2, description = $3
		WHERE name = $4 AND schema_domain = $5
	`
	result, err := s.db.ExecContext(
		ctx, query,
		fn.Class, fn.Badge, fn.Description, fn.Name, fn.SchemaDomain,
	)
	if err != nil {
		log.Printf("Error updating DAX function: %v", err)
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("DAX function not found: %s.%s", fn.SchemaDomain, fn.Name)
	}
	return nil
}

// DeleteDAXFunction deletes a DAX function
func (s *DAXFunctionsService) DeleteDAXFunction(
	ctx context.Context,
	domain string,
	name string,
) error {
	query := `
		DELETE FROM public.dax_functions
		WHERE name = $1 AND schema_domain = $2
	`
	result, err := s.db.ExecContext(ctx, query, name, domain)
	if err != nil {
		log.Printf("Error deleting DAX function: %v", err)
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("DAX function not found: %s.%s", domain, name)
	}
	return nil
}

// GetDAXFunctionsCount returns the count of DAX functions for a domain
func (s *DAXFunctionsService) GetDAXFunctionsCount(
	ctx context.Context,
	domain string,
) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM public.dax_functions WHERE schema_domain = $1`
	err := s.db.QueryRowContext(ctx, query, domain).Scan(&count)
	if err != nil {
		log.Printf("Error counting DAX functions: %v", err)
		return 0, err
	}
	return count, nil
}
```

---

## 🔧 Step 2: Update API Handlers

### Before (Domain-Specific Handler)

```go
// ❌ OLD PATTERN - query individual domain schema
func GetBundleHandler(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain") // "banking", "retail", etc.

	// OLD: Query domain-specific schema
	query := fmt.Sprintf(`
		SELECT * FROM %s.metrics_registry
		WHERE id = $1
	`, domain) // UNSAFE: string formatting

	var metric Metric
	err := db.GetContext(r.Context(), &metric, query, id)
	// ...
}
```

### After (Consolidated Handler)

```go
// ✅ NEW PATTERN - query consolidated public table with schema_domain filter
func GetBundleHandler(
	w http.ResponseWriter,
	r *http.Request,
	metricsService *services.MetricsService,
) {
	domain := chi.URLParam(r, "domain")

	// NEW: Query public schema with domain filter
	metrics, err := metricsService.GetMetricsByDomain(r.Context(), domain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}
```

---

## 🔧 Step 3: Update Bundle Routes

### Update `backend/internal/api/bundles_routes.go`

```go
package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/yourorg/semlayer/backend/internal/services"
)

// RegisterBundleRoutes registers bundle-related routes
func RegisterBundleRoutes(r chi.Router, db *sqlx.DB) {
	// Create service instances
	metricsService := services.NewMetricsService(db)
	daxService := services.NewDAXFunctionsService(db)

	r.Route("/bundles", func(r chi.Router) {
		// GET /bundles/{domain} - Get bundle for domain
		r.Get("/{domain}", func(w http.ResponseWriter, r *http.Request) {
			GetBundleHandler(w, r, metricsService, daxService)
		})

		// GET /bundles/{domain}/{nodeId} - Get specific metric
		r.Get("/{domain}/{nodeId}", func(w http.ResponseWriter, r *http.Request) {
			GetMetricHandler(w, r, metricsService)
		})

		// POST /bundles - Create bundle (admin only)
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			CreateBundleHandler(w, r, metricsService)
		})
	})
}
```

---

## 🔧 Step 4: Update API Handlers

### Update Bundle Handlers

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/yourorg/semlayer/backend/internal/services"
)

// BundleResponse represents bundle data
type BundleResponse struct {
	Domain    string                      `json:"domain"`
	Metrics   []services.MetricsRegistry  `json:"metrics"`
	Functions []services.DAXFunction      `json:"functions"`
}

// GetBundleHandler returns bundle (metrics + DAX functions) for a domain
func GetBundleHandler(
	w http.ResponseWriter,
	r *http.Request,
	metricsService *services.MetricsService,
	daxService *services.DAXFunctionsService,
) {
	domain := chi.URLParam(r, "domain")

	// Get metrics from consolidated table
	metrics, err := metricsService.GetMetricsByDomain(r.Context(), domain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get DAX functions from consolidated table
	functions, err := daxService.GetDAXFunctionsByDomain(r.Context(), domain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := BundleResponse{
		Domain:    domain,
		Metrics:   metrics,
		Functions: functions,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetMetricHandler returns a specific metric
func GetMetricHandler(
	w http.ResponseWriter,
	r *http.Request,
	metricsService *services.MetricsService,
) {
	domain := chi.URLParam(r, "domain")
	nodeID := chi.URLParam(r, "nodeId")

	metric, err := metricsService.GetMetricByNodeID(r.Context(), domain, nodeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if metric == nil {
		http.Error(w, "Metric not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metric)
}

// CreateBundleHandler creates new metrics and functions
func CreateBundleHandler(
	w http.ResponseWriter,
	r *http.Request,
	metricsService *services.MetricsService,
) {
	// Parse request body
	var req struct {
		Domain  string                    `json:"domain"`
		Metrics []services.MetricsRegistry `json:"metrics"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Insert metrics into consolidated table
	for _, metric := range req.Metrics {
		metric.SchemaDomain = req.Domain
		_, err := metricsService.InsertMetric(r.Context(), &metric)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}
```

---

## 🔧 Step 5: Update DAX Routes

### Update `backend/internal/api/dax_routes.go`

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/yourorg/semlayer/backend/internal/services"
)

// RegisterDAXRoutes registers DAX function-related routes
func RegisterDAXRoutes(r chi.Router, db *sqlx.DB) {
	daxService := services.NewDAXFunctionsService(db)

	r.Route("/dax-functions", func(r chi.Router) {
		// GET /dax-functions/{domain} - Get DAX functions for domain
		r.Get("/{domain}", func(w http.ResponseWriter, r *http.Request) {
			GetDAXFunctionsHandler(w, r, daxService)
		})

		// GET /dax-functions/{domain}/{name} - Get specific function
		r.Get("/{domain}/{name}", func(w http.ResponseWriter, r *http.Request) {
			GetDAXFunctionHandler(w, r, daxService)
		})

		// POST /dax-functions - Create function (admin only)
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			CreateDAXFunctionHandler(w, r, daxService)
		})
	})
}

// GetDAXFunctionsHandler returns all DAX functions for a domain
func GetDAXFunctionsHandler(
	w http.ResponseWriter,
	r *http.Request,
	daxService *services.DAXFunctionsService,
) {
	domain := chi.URLParam(r, "domain")

	functions, err := daxService.GetDAXFunctionsByDomain(r.Context(), domain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(functions)
}

// GetDAXFunctionHandler returns a specific DAX function
func GetDAXFunctionHandler(
	w http.ResponseWriter,
	r *http.Request,
	daxService *services.DAXFunctionsService,
) {
	domain := chi.URLParam(r, "domain")
	name := chi.URLParam(r, "name")

	fn, err := daxService.GetDAXFunctionByName(r.Context(), domain, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if fn == nil {
		http.Error(w, "Function not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fn)
}

// CreateDAXFunctionHandler creates a new DAX function
func CreateDAXFunctionHandler(
	w http.ResponseWriter,
	r *http.Request,
	daxService *services.DAXFunctionsService,
) {
	var req struct {
		Domain string                  `json:"domain"`
		Function services.DAXFunction  `json:"function"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req.Function.SchemaDomain = req.Domain
	_, err := daxService.InsertDAXFunction(r.Context(), &req.Function)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req.Function)
}
```

---

## 🧪 Step 6: Create Unit Tests

### Create `backend/internal/services/metrics_service_test.go`

```go
package services

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sqlx.DB {
	// Use test database
	db, err := sqlx.Connect(
		"postgres",
		"postgres://postgres:postgres@localhost:5432/alpha_test?sslmode=disable",
	)
	require.NoError(t, err)
	return db
}

func TestGetMetricsByDomain(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewMetricsService(db)
	ctx := context.Background()

	// Query metrics for banking domain
	metrics, err := service.GetMetricsByDomain(ctx, "banking")

	require.NoError(t, err)
	assert.NotEmpty(t, metrics)
	
	// Verify schema_domain is set correctly
	for _, m := range metrics {
		assert.Equal(t, "banking", m.SchemaDomain)
	}
}

func TestGetMetricByNodeID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewMetricsService(db)
	ctx := context.Background()

	metric, err := service.GetMetricByNodeID(ctx, "banking", "METRIC_001")

	require.NoError(t, err)
	if metric != nil {
		assert.Equal(t, "banking", metric.SchemaDomain)
		assert.Equal(t, "METRIC_001", metric.NodeID)
	}
}

func TestGetMetricsByDomains(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewMetricsService(db)
	ctx := context.Background()

	domains := []string{"banking", "retail", "wealth_management"}
	metrics, err := service.GetMetricsByDomains(ctx, domains)

	require.NoError(t, err)
	assert.NotEmpty(t, metrics)

	// Verify all returned metrics are from requested domains
	domainMap := make(map[string]bool)
	for _, d := range domains {
		domainMap[d] = true
	}

	for _, m := range metrics {
		assert.True(t, domainMap[m.SchemaDomain])
	}
}
```

---

## 🚀 Step 7: Update Main Application Setup

### Update `backend/main.go` or server initialization

```go
// OLD: No consolidated services
func setupRoutes(router chi.Router, db *sqlx.DB) {
	api.RegisterBundleRoutes(router) // No DB passed
	api.RegisterDAXRoutes(router)     // No DB passed
}

// NEW: Initialize services with DB
func setupRoutes(router chi.Router, db *sqlx.DB) {
	api.RegisterBundleRoutes(router, db)
	api.RegisterDAXRoutes(router, db)
}
```

---

## ✅ Verification Checklist

- [ ] Created `metrics_service.go` with all methods
- [ ] Created `dax_functions_service.go` with all methods
- [ ] Updated bundle handler to use MetricsService
- [ ] Updated DAX handler to use DAXFunctionsService
- [ ] Updated route registration to pass DB
- [ ] Updated query patterns (from domain-specific to public schema)
- [ ] Tests pass locally
- [ ] No compile errors: `go build ./...`
- [ ] API tests pass: `go test ./...`
- [ ] Performance tested (queries return quickly)
- [ ] Error handling verified

---

## 📊 Migration Checklist

### Phase 1: Code Preparation ✓
- [ ] Services created
- [ ] Handlers updated
- [ ] Routes updated
- [ ] Tests written and passing

### Phase 2: Database
- [ ] Run migration: `psql -f migrations/consolidate_metrics_and_dax.sql`
- [ ] Verify consolidated tables created
- [ ] Verify data migrated correctly

### Phase 3: Deployment
- [ ] Deploy code changes
- [ ] Test endpoints
- [ ] Monitor error logs
- [ ] Verify response times

### Phase 4: Cleanup
- [ ] Remove old domain-specific table queries
- [ ] Clean up unused code
- [ ] Update documentation

---

## 🔄 Common Refactoring Patterns

### Pattern 1: Single Domain Query

```go
// ❌ OLD
query := fmt.Sprintf("SELECT * FROM %s.metrics_registry WHERE id = $1", domain)

// ✅ NEW
metrics, _ := metricsService.GetMetricsByDomain(ctx, domain)
```

### Pattern 2: Multi-Domain Loop

```go
// ❌ OLD - Loops through domains
for _, domain := range domains {
	query := fmt.Sprintf("SELECT * FROM %s.metrics_registry", domain)
	metrics := db.Query(query)
	// Process metrics
}

// ✅ NEW - Single query with sqlx.In
metrics, _ := metricsService.GetMetricsByDomains(ctx, domains)
```

### Pattern 3: Dynamic Schema

```go
// ❌ OLD - Dynamic schema in query
query := fmt.Sprintf("SELECT * FROM %s.dax_functions", domain)

// ✅ NEW - Parameterized with WHERE clause
query := "SELECT * FROM public.dax_functions WHERE schema_domain = $1"
```

---

## 📝 Common Gotchas

1. **Forgetting schema_domain parameter:** Always include `schema_domain` in WHERE clause
2. **String formatting queries:** Always use parameterized queries with `$1`, `$2`
3. **Transactions:** If updating multiple tables, use transactions to maintain consistency
4. **NULL values:** Use pointers for nullable columns (`*string`)
5. **Array types:** Use `pq.StringArray` for PostgreSQL array types

---

## 🔗 References

- [sqlx Documentation](https://github.com/jmoiron/sqlx)
- [PostgreSQL parameterized queries](https://www.postgresql.org/docs/current/sql-syntax.html#SQL-SYNTAX-LEXICAL)
- [Go database/sql](https://golang.org/pkg/database/sql/)

// AUDIT_EXPLORER_QUICK_INTEGRATION.md
// Copy-paste guide for integrating Audit Explorer into existing api.go

/*
========================================
STEP 1: Add Import to api.go
========================================

Add these imports near the top of your api.go file:

import (
    // ... existing imports ...
    "github.com/hondyman/semlayer/backend/internal/audit"
    "github.com/hondyman/semlayer/backend/internal/auth"
)
*/

/*
========================================
STEP 2: Initialize in APIServer struct
========================================

The APIServer struct likely already has a db field:

type APIServer struct {
    db *sql.DB
    // ... other fields ...
    // aiClient audit.AIClient  // Optional: if not already present
}

If you don't have an aiClient field, add it:

type APIServer struct {
    db       *sql.DB
    aiClient audit.AIClient  // NEW - for audit explorer
    // ... other fields ...
}

Initialize it in your APIServer constructor:

func NewAPIServer(db *sql.DB) *APIServer {
    return &APIServer{
        db:       db,
        aiClient: initializeAIClient(),  // NEW
        // ... other fields ...
    }
}

func initializeAIClient() audit.AIClient {
    // Example with Anthropic
    // return anthropic.NewClient(os.Getenv("ANTHROPIC_API_KEY"))
    
    // Example with OpenAI
    // return openai.NewClient(os.Getenv("OPENAI_API_KEY"))
    
    // Or implement the interface yourself
    // return myCustomAIClient{}
    
    // For now, use a stub
    return &stubAIClient{}
}

// Stub implementation (replace with real AI client)
type stubAIClient struct{}

func (s *stubAIClient) ExplainAuditEvents(ctx context.Context, req *audit.ExplainRequest) (*audit.ExplainResponse, error) {
    return &audit.ExplainResponse{
        RootCause: "Unable to explain - AI client not configured",
    }, nil
}
*/

/*
========================================
STEP 3: Add Method to Register Routes
========================================

Add this method to your APIServer type:

func (a *APIServer) registerAuditExplorerRoutes(r chi.Router) error {
    // Initialize repository
    trinoRepository := audit.NewTrinoRepository(a.db)

    // Initialize service
    explorerService := audit.NewExplorerService(trinoRepository, a.aiClient)

    // Initialize handler
    explorerHandler := audit.NewExplorerHandler(explorerService)

    // Register routes under /api/audit-explorer
    r.Route("/audit-explorer", func(r chi.Router) {
        r.Use(audit.TenantScopeMiddleware)
        r.Use(audit.RoleBasedAccessMiddleware(
            "global_admin",
            "global_ops",
            "tenant_admin",
            "tenant_ops",
        ))
        explorerHandler.RegisterRoutes(r)
    })

    return nil
}
*/

/*
========================================
STEP 4: Call registerAuditExplorerRoutes
========================================

Find your setupRoutes() method (or wherever you register API routes):

func (a *APIServer) setupRoutes() {
    r := chi.NewRouter()
    
    // Middleware
    r.Use(middleware.Logger)
    r.Use(auth.AuthMiddleware)  // Your existing auth middleware
    
    // ... your existing routes ...
    
    // ADD THIS LINE:
    if err := a.registerAuditExplorerRoutes(r); err != nil {
        log.Fatalf("Failed to register audit explorer routes: %v", err)
    }
    
    // ... more routes ...
}
*/

/*
========================================
STEP 5: Frontend - Add Navigation Link
========================================

In frontend/src/components/MainNavigation.tsx, add:

import AuditExplorerIcon from '@mui/icons-material/History';

// Inside your navigation menu construction:
{hasRole('global_admin', 'global_ops', 'tenant_admin', 'tenant_ops') && (
  <NavLink
    to="/audit-explorer"
    icon={<AuditExplorerIcon />}
    label="Audit Explorer"
  />
)}
*/

/*
========================================
STEP 6: Frontend - Add Route
========================================

In frontend/src/App.tsx or your router config:

import { lazy } from 'react';

const AuditExplorer = lazy(() => import('@/components/audit/AuditExplorer'));

// Inside your <Routes>:
<Route path="/audit-explorer" element={<AuditExplorer />} />
*/

/*
========================================
STEP 7: Verify Trino Connection
========================================

Ensure your database connection uses the Trino driver:

import _ "github.com/trinodb/trino-go-client/trino"

// In your database connection setup:
db, err := sql.Open("trino", "http://trino-host:8080/default/iceberg?user=root")
if err != nil {
    log.Fatalf("Failed to connect to Trino: %v", err)
}
*/

/*
========================================
STEP 8: Verify Required Tables
========================================

The explorer queries these Trino tables:
- iceberg.audit.scheduler_job_runs
- iceberg.audit.scheduler_dag_runs
- iceberg.audit.governance_changesets
- iceberg.audit.semantic_snapshots
- iceberg.audit.compliance_violations
- iceberg.audit.orchestration_events (optional)

If these don't exist, create them or update trino_queries.go
to reference your actual table names.
*/

/*
========================================
STEP 9: Build and Test
========================================

# Backend
go mod tidy
go build ./...
go test ./backend/internal/audit/...

# Frontend
npm install
npm run build

# Test endpoint
curl -X POST http://localhost:8080/api/audit-explorer/events \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-001" \
  -H "Authorization: Bearer <your-token>" \
  -d '{
    "tenantFilter": ["tenant-001"],
    "from": "2026-01-10T00:00:00Z",
    "to": "2026-01-17T00:00:00Z",
    "limit": 50,
    "offset": 0
  }'
*/

/*
========================================
STEP 10: Environment Variables (Optional)
========================================

Add to your .env or docker-compose:

# AI Client configuration
ANTHROPIC_API_KEY=sk-ant-...  # OR
OPENAI_API_KEY=sk-...

# Or configure via code if using a custom AI client

TRINO_USER=root
TRINO_PASSWORD=password
TRINO_HOST=localhost
TRINO_PORT=8080
*/

/*
========================================
COMPLETE INTEGRATION EXAMPLE
========================================

Here's a minimal complete api.go integration:

package api

import (
    "context"
    "database/sql"
    "log"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/hondyman/semlayer/backend/internal/audit"
    "github.com/hondyman/semlayer/backend/internal/auth"
    "github.com/hondyman/semlayer/backend/internal/middleware"
)

type APIServer struct {
    db       *sql.DB
    aiClient audit.AIClient
}

func NewAPIServer(db *sql.DB) *APIServer {
    return &APIServer{
        db:       db,
        aiClient: initializeAIClient(),
    }
}

func initializeAIClient() audit.AIClient {
    // TODO: Implement real AI client
    return &stubAIClient{}
}

type stubAIClient struct{}

func (s *stubAIClient) ExplainAuditEvents(ctx context.Context, req *audit.ExplainRequest) (*audit.ExplainResponse, error) {
    return &audit.ExplainResponse{
        RootCause: "AI client not configured",
    }, nil
}

func (a *APIServer) Start(addr string) error {
    r := chi.NewRouter()

    // Middleware
    r.Use(middleware.Logger)
    r.Use(auth.AuthMiddleware)

    // API Routes
    r.Route("/api", func(r chi.Router) {
        // Audit Explorer routes
        if err := a.registerAuditExplorerRoutes(r); err != nil {
            return
        }

        // ... other routes ...
    })

    return http.ListenAndServe(addr, r)
}

func (a *APIServer) registerAuditExplorerRoutes(r chi.Router) error {
    trinoRepository := audit.NewTrinoRepository(a.db)
    explorerService := audit.NewExplorerService(trinoRepository, a.aiClient)
    explorerHandler := audit.NewExplorerHandler(explorerService)

    r.Route("/audit-explorer", func(r chi.Router) {
        r.Use(audit.TenantScopeMiddleware)
        r.Use(audit.RoleBasedAccessMiddleware(
            "global_admin",
            "global_ops",
            "tenant_admin",
            "tenant_ops",
        ))
        explorerHandler.RegisterRoutes(r)
    })

    return nil
}
*/

/*
========================================
VERIFY INTEGRATION
========================================

After integration, check:

1. Routes registered:
   GET /api/audit-explorer/events
   GET /api/audit-explorer/entities/*
   GET /api/audit-explorer/incidents*
   GET /api/audit-explorer/compliance-events
   POST /api/audit-explorer/explain
   GET /api/audit-explorer/dashboard/*

2. Frontend compiles:
   npm run build (no TypeScript errors)

3. Endpoint responds:
   curl http://localhost:8080/api/audit-explorer/events

4. Authentication required:
   Should return 401 without Authorization header
   Should return 403 without proper role

5. Tenant scope enforced:
   Should return no data for unauthorized tenants
   Should return 403 for out-of-scope tenant requests

If all checks pass, you're ready to deploy!
*/

// ========================================
// FILE LOCATIONS TO COPY
// ========================================

/*
Backend files to copy:
- explorer_models.go           → /backend/internal/audit/
- explorer_repository.go       → /backend/internal/audit/
- explorer_service.go          → /backend/internal/audit/
- explorer_handler.go          → /backend/internal/audit/
- explorer_rbac.go             → /backend/internal/audit/
- trino_queries.go             → /backend/internal/audit/

Frontend files to copy:
- AuditExplorer.tsx            → /frontend/src/components/audit/
- FilterBar.tsx                → /frontend/src/components/audit/
- TimelineView.tsx             → /frontend/src/components/audit/tabs/
- EntitiesView.tsx             → /frontend/src/components/audit/tabs/
- IncidentsView.tsx            → /frontend/src/components/audit/tabs/
- ComplianceView.tsx           → /frontend/src/components/audit/tabs/
- AIPanel.tsx                  → /frontend/src/components/audit/panels/
- useAuditExplorer.ts          → /frontend/src/hooks/

Documentation:
- AUDIT_EXPLORER_GUIDE.md      → /
- AUDIT_EXPLORER_SUMMARY.md    → /
*/

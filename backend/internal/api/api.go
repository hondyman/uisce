package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/ai"
	"github.com/hondyman/semlayer/backend/internal/altinvest"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/audit"
	"github.com/hondyman/semlayer/backend/internal/billing"
	"github.com/hondyman/semlayer/backend/internal/bp"
	"github.com/hondyman/semlayer/backend/internal/cube"
	"github.com/hondyman/semlayer/backend/internal/data_intelligence/tiering"
	charts "github.com/hondyman/semlayer/backend/internal/db/charts"
	"github.com/hondyman/semlayer/backend/internal/events"
	"github.com/hondyman/semlayer/backend/internal/goldcopy"
	"github.com/hondyman/semlayer/backend/internal/handlers"
	"github.com/hondyman/semlayer/backend/internal/household"
	"github.com/hondyman/semlayer/backend/internal/infrastructure"
	"github.com/hondyman/semlayer/backend/internal/lineage"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/mdm"
	appmid "github.com/hondyman/semlayer/backend/internal/middleware"
	models "github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/oauth"
	"github.com/hondyman/semlayer/backend/internal/platform"
	"github.com/hondyman/semlayer/backend/internal/portfoliomaster"
	"github.com/hondyman/semlayer/backend/internal/preference"
	"github.com/hondyman/semlayer/backend/internal/profiler"
	"github.com/hondyman/semlayer/backend/internal/rag"
	"github.com/hondyman/semlayer/backend/internal/region"
	"github.com/hondyman/semlayer/backend/internal/reports"
	"github.com/hondyman/semlayer/backend/internal/repository"
	"github.com/hondyman/semlayer/backend/internal/rules"
	si "github.com/hondyman/semlayer/backend/internal/scheduler_intelligence"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/internal/succession"
	"github.com/hondyman/semlayer/backend/internal/taxplan"
	"github.com/hondyman/semlayer/backend/internal/telemetry/optimize"
	temporal "github.com/hondyman/semlayer/backend/internal/temporal"
	"github.com/hondyman/semlayer/backend/internal/tenant"
	"github.com/hondyman/semlayer/backend/internal/trino"
	coremodels "github.com/hondyman/semlayer/backend/models"
	"github.com/hondyman/semlayer/backend/pkg/ingestion"
	"github.com/hondyman/semlayer/backend/pkg/llm"
	"github.com/hondyman/semlayer/backend/pkg/semantic"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	calendarsync "github.com/hondyman/semlayer/backend/internal/sync"

	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx"

	catalogmeta "github.com/hondyman/semlayer/backend/internal/metadata"
	hasuraclient "github.com/hondyman/semlayer/libs/hasura-client"
	temporalclientlib "github.com/hondyman/semlayer/libs/temporal-client"
	temporalclient "go.temporal.io/sdk/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// Create server instance

// Server holds services and handlers used by the HTTP API.
type Server struct {
	DB                      *sql.DB
	SQLXDB                  *sqlx.DB
	AggregatesDB            *sql.DB
	Reg                     *Registry
	WsHub                   *WebSocketHub
	SemanticNameResolver    *SemanticNameResolver
	AuditSvc                *audit.Service
	NotificationSvc         *services.EngagementNotificationService
	CampaignSvc             *services.NotificationCampaignService
	NotificationHandlers    *NotificationAPIHandlers
	DashboardHandlers       *DashboardAPIHandlers
	ModelCatalogHandler     *handlers.ModelCatalogHandler
	CatalogScanHandler      *handlers.CatalogScanHandler
	TestConnectionHandler   *handlers.TestConnectionHandler
	MetricRegistryHandler   *handlers.MetricRegistryHandler
	ValuesHandler           *handlers.ValuesHandler
	AIHandler               *handlers.AIHandler
	ComplianceHandler       *handlers.ComplianceHandler
	SemanticSvc             *analytics.SemanticService
	SemanticMappingSvc      *analytics.SemanticMappingService
	SemanticMappingHandler  *handlers.SemanticMappingHandler
	AbbreviationSvc         *services.AbbreviationService
	Validate                *validator.Validate
	SecMgr                  *services.SecurityManager
	RelationshipSvc         *RelationshipService
	SemanticCalculationSvc  *analytics.SemanticCalculationService
	CalculationHandler      *handlers.CalculationHandler
	ChartHandler            *handlers.ChartHandler
	ExecutionMonitorHandler *handlers.ExecutionMonitorHandler
	ProfileJobs             sync.Map
	NLQService              *services.NLQService
	FeedbackService         *services.FeedbackService
	EvalService             *services.EvalService
	CubeSyncService         *analytics.CubeSyncService
	LLMConfigSvc            *llm.LLMConfigService
	TemporalClient          temporalclient.Client
	EvidenceBundleService   *services.EvidenceBundleService
	ApprovalService         *services.ApprovalService
	SemanticLayerHandler    *SemanticLayerHandler
	GeminiClient            LLMProvider
	HouseholdService        *household.Service
	AltInvestService        *altinvest.Service
	BillingService          *billing.Service
	TaxPlanService          *taxplan.Service
	SuccessionService       *succession.Service
	GraphService            *catalogmeta.GraphService
	WriteHandler            *handlers.WriteHandler
	MCPHandler              *handlers.MCPHandler
	IgniteClient            *infrastructure.IgniteClient
	FolderHandler           *handlers.FolderHandler
	LineageSvc              *services.LineageService
	CueEngine               *services.CueEngine

	PageLayoutHandler       *handlers.PageLayoutHandler
	PipelineHandler         *handlers.PipelineHandler
	EventsHandler           *ingestion.EventsHandler
	GenAICopilotHandler     *handlers.GenAICopilotHandler
	PolicyGenerationHandler *handlers.PolicyGenerationHandler
	CalcHandler             *handlers.CalcHandler
	CubeHandler             *CubeHandler
	DatasourceResolver      security.DatasourceResolver
	BusinessObjectService   *catalogmeta.BusinessObjectService
	QueryHandler            *handlers.QueryHandler
	SavedQueryHandler       *handlers.SavedQueryHandler
	SearchHandler           *handlers.SearchHandler
	NLQHandler              *handlers.NLQHandler
	AuditHistoryHandler     *handlers.AuditHistoryHandler
	RelationshipHandler     *RelationshipHandler
	LineageHandler          *LineageHandler
	AdminAPIKeyHandler      *handlers.AdminAPIKeyHandler
	AdminHandler            *AdminHandler
	RAGHandler              *RAGHandler
	EventBus                EventBus
	ExportHandlers          *handlers.ExportHandlers
	SchedulerHandlers       *handlers.SchedulerHandlers

	// Phase 8: Advanced Cross-Domain Intelligence
	PortfolioSecuritySvc *mdm.PortfolioSecurityService
	SecurityLineageSvc   *mdm.SecurityLineageService
	ExecutionEngine      *mdm.ExecutionEngine
}

// SchemaTable represents a schema and table pair
type SchemaTable struct {
	Schema string
	Table  string
}

// BusinessEntity represents a single entity in the entity_attribute table.
// Each entity is stored as its own row with a parent_id for hierarchy and
// catalog_node_id linking to semantic terms (preventing stale name references).
type BusinessEntity struct {
	ID                 string         `db:"id"`
	TenantID           string         `db:"tenant_id"`
	TenantDatasourceID string         `db:"tenant_datasource_id"`
	ParentID           sql.NullString `db:"parent_id"`
	CatalogNodeID      sql.NullString `db:"catalog_node_id"`
	Key                string         `db:"entity_key"`
	Name               string         `db:"name"`
	IsCore             bool           `db:"is_core"`
	BusinessName       sql.NullString `db:"business_name"`
	TechnicalName      sql.NullString `db:"technical_name"`
}

// BusinessEntityResponse is the structure for the API response.
// CatalogNodeID is included to enable linking back to semantic terms.
type BusinessEntityResponse struct {
	Key           string                            `json:"key"`
	Name          string                            `json:"name"`
	IsCore        bool                              `json:"isCore"`
	CatalogNodeID string                            `json:"catalogNodeId,omitempty"`
	BusinessName  string                            `json:"businessName,omitempty"`
	TechnicalName string                            `json:"technicalName,omitempty"`
	Subtypes      map[string]BusinessEntityResponse `json:"subtypes,omitempty"`
}

// ===== SEMANTIC BUNDLE STRUCTURES (LLM-FRIENDLY CONTRACT) =====

// PhysicalMapping defines where a field physically lives in the database.
// This is the canonical truth for SQL generation.
type PhysicalMapping struct {
	DatasourceID string `json:"datasource_id"` // e.g., "postgres", "snowflake"
	Table        string `json:"table"`         // e.g., "customers"
	Column       string `json:"column"`        // e.g., "company_identifier"
}

// SemanticField is the complete field definition that the LLM uses.
// UUID is identity, names are labels. Physical mapping is canonical.
type SemanticField struct {
	FieldID      string          `json:"field_id"`      // UUID: immutable identity
	Name         string          `json:"name"`          // logical name: may change
	DisplayName  string          `json:"display_name"`  // UI label: may change
	SemanticTerm string          `json:"semantic_term"` // meaning: may change
	Subtype      *string         `json:"subtype,omitempty"`
	Aliases      []string        `json:"aliases,omitempty"` // old names for migration
	Physical     PhysicalMapping `json:"physical"`          // canonical location
	Description  string          `json:"description,omitempty"`
}

// SubtypeDefinition describes a subtype value and its constraints.
type SubtypeDefinition struct {
	ID                 string   `json:"id"`                  // e.g., "RETAIL"
	Label              string   `json:"label"`               // e.g., "Retail Customer"
	DiscriminatorValue string   `json:"discriminator_value"` // value in the column
	Fields             []string `json:"fields,omitempty"`    // field IDs of this subtype
	RequiredFields     []string `json:"required_fields,omitempty"`
}

// DiscriminatorMetadata defines how subtypes are distinguished in the database.
type DiscriminatorMetadata struct {
	ColumnName string              `json:"column_name"` // e.g., "customer_type"
	Subtypes   []SubtypeDefinition `json:"subtypes"`
}

// SemanticRelationship defines a join in the schema.
type SemanticRelationship struct {
	TargetBOID   string `json:"target_bo_id"`  // UUID of target business object
	JoinType     string `json:"join_type"`     // "INNER", "LEFT", "RIGHT", "FULL"
	SourceColumn string `json:"source_column"` // column in driving table
	TargetColumn string `json:"target_column"` // column in target table
	TargetTable  string `json:"target_table"`  // e.g., "orders" (for convenience)
}

// SemanticBundle is the complete, deterministic contract for the LLM.
// It contains everything needed to generate SQL, with no guessing required.
type SemanticBundle struct {
	BusinessObjectID   string                  `json:"business_object_id"`
	BusinessObjectName string                  `json:"business_object_name"`
	DatasourceID       string                  `json:"datasource_id"` // where the data lives
	DrivingTable       string                  `json:"driving_table"` // main table
	Version            string                  `json:"version"`       // semantic model version
	Discriminator      *DiscriminatorMetadata  `json:"discriminator,omitempty"`
	Fields             []SemanticField         `json:"fields"`                  // all fields
	Relationships      []SemanticRelationship  `json:"relationships,omitempty"` // all joins
	Snapshot           *audit.SemanticSnapshot `json:"snapshot,omitempty"`      // optional, region-scoped snapshot
	CreatedAt          string                  `json:"created_at,omitempty"`
	UpdatedAt          string                  `json:"updated_at,omitempty"`
}

// ===== LEGACY STRUCTURES (For backward compatibility) =====

// ===== METADATA VERSIONING STRUCTURES =====

// MetadataVersion tracks changes to semantic metadata over time
type MetadataVersion struct {
	VersionID        string                 `json:"version_id"`               // UUID
	BusinessObjectID string                 `json:"business_object_id"`       // UUID
	Version          int                    `json:"version"`                  // Incrementing counter
	CreatedAt        string                 `json:"created_at"`               // RFC3339
	CreatedBy        string                 `json:"created_by"`               // User ID
	ChangeType       string                 `json:"change_type"`              // e.g., "field_added", "field_renamed", "field_removed"
	ChangeDetail     map[string]interface{} `json:"change_detail"`            // JSON payload of what changed
	PreviousValue    map[string]interface{} `json:"previous_value,omitempty"` // Before value
	NewValue         map[string]interface{} `json:"new_value,omitempty"`      // After value
}

// FieldAlias tracks old field names for backward compatibility
type FieldAlias struct {
	AliasID     string `json:"alias_id"`              // UUID
	FieldID     string `json:"field_id"`              // UUID of current field
	OldName     string `json:"old_name"`              // Former field name
	RenamedAt   string `json:"renamed_at"`            // RFC3339
	RenamedBy   string `json:"renamed_by"`            // User ID
	IsActive    bool   `json:"is_active"`             // Whether alias still resolves to field
	Description string `json:"description,omitempty"` // Why it was renamed
}

// ===== LEGACY STRUCTURES (For backward compatibility) =====

// listBusinessObjects returns a flat list of business objects for the BP Designer.
// Frontend expects: [{ id, name, display_name, description?, fields: [{name,type,label}], icon?, config? }]
func (s *Server) listBusinessObjects(w http.ResponseWriter, r *http.Request) {
	// Build security context with datasource + region validation
	// Legacy handlers in api.go don't always have the full middleware stack,
	// so we use the helper to ensure we have a valid context.
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: s.DatasourceResolver,
	})
	if err != nil {
		// Log but try to proceed with minimal context if tenant is present (legacy fallback)
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		if tenantID != "" {
			secCtx = &security.Context{TenantID: tenantID}
			ctx = r.Context()
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	items, err := s.BusinessObjectService.ListBusinessObjectsLegacy(ctx, secCtx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch business objects: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(items); err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to encode response: %v", err)
	}
}

// getBusinessObjectByID fetches a single business object by its ID
// Supports accessing objects owned by tenant OR inherited from gold copy tenant
func (s *Server) getBusinessObjectByID(w http.ResponseWriter, r *http.Request) {
	// Build security context
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: s.DatasourceResolver,
	})
	if err != nil {
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		if tenantID != "" {
			secCtx = &security.Context{TenantID: tenantID}
			ctx = r.Context()
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	boID := chi.URLParam(r, "id")
	if boID == "" {
		http.Error(w, "Business object ID is required", http.StatusBadRequest)
		return
	}

	item, err := s.BusinessObjectService.GetBusinessObjectLegacy(ctx, secCtx, boID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Business object not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to fetch business object: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(item); err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to encode response: %v", err)
	}
}

// getSemanticBundle returns the complete, deterministic semantic metadata for an LLM.
// It requires bo_id and tenant_id as query parameters or bo_id in path + tenant_id header.
// Response contains all fields with exact database locations, relationships, subtypes, and version history.
func (s *Server) getSemanticBundle(w http.ResponseWriter, r *http.Request) {
	// Read bo_id from query parameter or path parameter
	boID := r.URL.Query().Get("bo_id")
	if boID == "" {
		// Try to get from path if registered with :bo_id pattern
		boID = chi.URLParam(r, "bo_id")
	}

	// Read tenant_id from header or query parameter
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}

	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header or tenant_id query parameter is required", http.StatusBadRequest)
		return
	}

	if boID == "" {
		http.Error(w, "bo_id query parameter is required", http.StatusBadRequest)
		return
	}

	// Use transaction to set RLS context
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to start transaction: %v", err), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Set RLS context for this request
	if _, err := tx.ExecContext(r.Context(), "SELECT set_config('hasura.tenant_id', $1, true)", tenantID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to set tenant context: %v", err), http.StatusInternalServerError)
		return
	}

	// Query business object details
	var boName, drivingTable, datasourceID string
	err = tx.QueryRowContext(r.Context(),
		`SELECT display_name, driver_table_name, datasource_id 
		 FROM public.business_objects 
		 WHERE id = $1 AND tenant_id = $2`,
		boID, tenantID).Scan(&boName, &drivingTable, &datasourceID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Business object not found: %v", err), http.StatusNotFound)
		return
	}

	// Query all fields for this business object
	fRows, err := tx.QueryContext(r.Context(),
		`SELECT id, field_name, display_label, column_name, field_type, field_description
		 FROM public.bo_fields
		 WHERE tenant_id = $1 AND bo_id = $2
		 ORDER BY display_order`,
		tenantID, boID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch fields: %v", err), http.StatusInternalServerError)
		return
	}
	defer fRows.Close()

	fields := make([]SemanticField, 0)
	for fRows.Next() {
		var fieldID, fieldName, displayLabel, columnName, fieldType, description string
		if err := fRows.Scan(&fieldID, &fieldName, &displayLabel, &columnName, &fieldType, &description); err != nil {
			http.Error(w, fmt.Sprintf("Failed to scan field: %v", err), http.StatusInternalServerError)
			return
		}

		// Build physical mapping
		physical := PhysicalMapping{
			DatasourceID: datasourceID,
			Table:        drivingTable,
			Column:       columnName,
		}

		field := SemanticField{
			FieldID:      fieldID,
			Name:         fieldName,
			DisplayName:  displayLabel,
			SemanticTerm: fieldName, // Default to field name for now
			Aliases:      []string{},
			Physical:     physical,
			Description:  description,
		}

		// Get aliases for this field if resolver is available
		if s.SemanticNameResolver != nil {
			field.Aliases = s.SemanticNameResolver.ResolveFieldIDToTermNames(fieldID)
		}

		fields = append(fields, field)
	}

	if err := fRows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error iterating fields: %v", err), http.StatusInternalServerError)
		return
	}

	// Get latest metadata version if available
	latestVersion := 1
	var latestVersionTime string
	err = tx.QueryRowContext(r.Context(),
		`SELECT COALESCE(MAX(version), 0), MAX(created_at)
		 FROM public.metadata_versions 
		 WHERE tenant_id = $1 AND business_object_id = $2`,
		tenantID, boID).Scan(&latestVersion, &latestVersionTime)
	// Error here is okay - metadata_versions table might not exist yet
	if latestVersion == 0 {
		latestVersion = 1
	}

	// Build semantic bundle
	bundle := SemanticBundle{
		BusinessObjectID:   boID,
		BusinessObjectName: boName,
		DatasourceID:       datasourceID,
		DrivingTable:       drivingTable,
		Version:            fmt.Sprintf("v%d", latestVersion),
		Fields:             fields,
		Relationships:      []SemanticRelationship{},
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:          latestVersionTime,
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to finalize request: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(bundle); err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to encode response: %v", err)
	}
}

// Debug endpoint to echo request headers and basic info — useful to verify Vite proxy forwards headers
func (s *Server) debugProxyHeaders(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"path":   r.URL.Path,
		"method": r.Method,
		"headers": map[string]string{
			"X-Tenant-ID":            jwtmiddleware.GetClaimsFromContext(r).TenantID,
			"X-Tenant-Datasource-ID": r.Header.Get("X-Tenant-Datasource-ID"),
			"Host":                   r.Host,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func registerAuthRoutes(r chi.Router, srv *Server) {
	r.Route("/auth", func(r chi.Router) {

		// Authentication
		srv.RegisterAuthRoutes(r)
	})
}

func SetupRouter(db *sql.DB, dynatraceManager interface{}, perf ProfilerService, temporalClient temporalclient.Client, qosManager *services.QoSManager, trinoAuditService *audit.TrinoAuditService, geminiClient *GeminiClient, resolver security.DatasourceResolver, redisClient *redis.Client) *chi.Mux {

	// Create chi router and helper services required for setup
	fmt.Println("DEBUG: SetupRouter INVOKED! [Version 3]")
	r := chi.NewRouter()

	// Initialize sqlxDB early for services that need it
	sqlxDB := sqlx.NewDb(db, "postgres")

	// Initialize relational lineage repository (replacing AGE)
	sqlRepo := lineage.NewDBLineageRepository(sqlxDB)

	// Initialize catalog scan service and handler early for raw route registration
	catalogScanService := catalogmeta.NewCatalogScanService(sqlxDB, sqlRepo)
	catalogScanHandler := handlers.NewCatalogScanHandler(catalogScanService)

	// Register SSE endpoint RAW on the router to bypass blocking middleware (caching/buffering)
	// Moved to end of function using rootMux mounting strategy to avoid panic
	// Development middleware: log every incoming request (method, path, headers, body)
	// This is intentionally verbose and should only be enabled during local debugging.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Read body (if any) for logging and restore it for downstream handlers
			var bodyBytes []byte
			if req.Body != nil {
				bodyBytes, _ = io.ReadAll(req.Body)
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
			// Collect a subset of headers for brevity
			headersToLog := []string{"Content-Type", "Authorization", "Origin", "X-Tenant-ID", "X-Tenant-Datasource-ID"}
			headerParts := []string{}
			for _, h := range headersToLog {
				headerParts = append(headerParts, fmt.Sprintf("%s=%s", h, req.Header.Get(h)))
			}
			fmt.Fprintf(os.Stderr, "[REQ] %s %s Headers:%s Body:%s\n", req.Method, req.URL.Path, strings.Join(headerParts, ","), string(bodyBytes))
			fmt.Fprintf(os.Stderr, "[DEBUG-MARKER] path=%s method=%s contains_bo=%v\n", req.URL.Path, req.Method, strings.Contains(req.URL.Path, "/business-objects"))

			// Additional detailed logging for business-objects endpoint
			if strings.Contains(req.URL.Path, "/business-objects") && req.Method == "POST" {
				fmt.Fprintf(os.Stderr, "[REQ-MIDDLEWARE] Matched business-objects POST, path=%s\n", req.URL.Path)
				var reqBody map[string]interface{}
				if err := json.Unmarshal(bodyBytes, &reqBody); err == nil {
					fmt.Fprintf(os.Stderr, "[REQ-DETAIL] Business Object Create: datasource_id=%v parent_id=%v name=%v header_ds=%v\n",
						reqBody["datasource_id"], reqBody["parent_id"], reqBody["name"], req.Header.Get("X-Tenant-Datasource-ID"))
				} else {
					fmt.Fprintf(os.Stderr, "[REQ-MIDDLEWARE] Failed to unmarshal body: %v\n", err)
				}
			}

			next.ServeHTTP(w, req)
		})
	})

	// Request tracing middleware: captures response handler header and status
	r.Use(RequestTracingMiddleware)
	logging.GetLogger().Sugar().Info("RequestTracingMiddleware registered on router")

	// Simple CORS middleware for chi to allow Vite dev origins during
	// local development. This mirrors the Gin-based CORS configuration
	// used in the API gateway but is compatible with the chi router.
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			origin := req.Header.Get("Origin")
			// Allow common dev origins - these are allowed WITH credentials
			allowedOrigins := getEnv("ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:5174")
			isAllowed := false
			for _, o := range strings.Split(allowedOrigins, ",") {
				trimmedOrigin := strings.TrimSpace(o)
				if origin == trimmedOrigin || (trimmedOrigin == "*" && origin != "") {
					isAllowed = true
					break
				}
			}

			// Always allow localhost dev origins
			if !isAllowed && (strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1")) {
				isAllowed = true
			}

			// Centralize allowed headers so we can extend them safely
			allowedHeaders := "Origin, X-Requested-With, Content-Type, Accept, Authorization, X-Request-ID, X-API-Key, X-Datasource-Id, X-Region, X-Tenant-Datasource-ID, X-Tenant-ID, X-User-ID, X-Tenant-Region, X-Hasura-Admin-Secret"

			if isAllowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Max-Age", "3600")
			} else if origin != "" {
				// For other origins, use wildcard (no credentials allowed with this)
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			}
			if req.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, req)
		})
	}

	// Register the CORS middleware early so preflight requests are handled
	// before route matching.
	r.Use(corsMiddleware)

	// Apply metadata caching middleware for layout/schema endpoints
	// This implements stale-while-revalidate to solve the N+1 fetch problem
	// for metadata-driven UI components
	r.Use(appmid.MetadataCacheMiddlewareDefault())

	// Initialize security manager with JWT secret
	jwtSecret := []byte(getEnv("JWT_SECRET", "dev-jwt-secret-key-change-in-production"))
	secMgr := services.NewSecurityManager(nil, nil, jwtSecret)
	apiKeyStore := services.NewDBAPIKeyStore(sqlxDB)
	secMgr.SetAPIKeyStore(apiKeyStore)
	if err := secMgr.LoadAPIKeysFromFile(getEnv("API_KEYS_FILE", "config/api_keys.json")); err != nil {
		log.Printf("[WARN] Failed to load API keys from file: %v", err)
	}

	// Development helper: optionally seed an API key for a test user so local
	// runs can exercise authenticated endpoints. Set SEED_API_KEY_USER to a
	// user id (e.g., tester@example.com) before starting the server. The
	// generated key will be logged to stdout.
	if seedUser := os.Getenv("SEED_API_KEY_USER"); seedUser != "" {
		key := secMgr.GenerateAPIKey(seedUser, "", []string{"temporal.admin"})
		log.Printf("[DEV] Seeded API key for user %s: %s", seedUser, key)
	}

	// Apply Auth Context Middleware globally (does not block, but populates context)
	r.Use(appmid.AuthContextMiddleware(secMgr))
	// Enforce region header and validate tenant scoping on all semantic requests
	// Use new TenantRegionResolver for cleaner region + Gold Copy handling
	regionResolver := region.NewTenantRegionResolver(db)
	r.Use(region.RegionValidationMiddleware(regionResolver))

	// Health check endpoint for Docker health checks
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// Temporary root-level debug endpoint to capture raw POST bodies quickly.
	// Use when /api/... routes appear to be unreachable. This logs the raw
	// request body to stderr and returns 204.

	// Diagnostic route: list registered routes at request time to help debug missing handlers.
	r.Get("/_routes", func(w http.ResponseWriter, req *http.Request) {
		routes := []string{}
		_ = chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
			routes = append(routes, fmt.Sprintf("%s %s", method, route))
			return nil
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"routes": routes})
	})

	// Log any unmatched requests so we can diagnose 404s during local debugging.
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		fmt.Fprintf(os.Stderr, "[NOTFOUND] %s %s Body: %s\n", r.Method, r.URL.Path, string(bodyBytes))
		http.NotFound(w, r)
	})

	// Note: Gin-based CORSMiddleware exists in internal/middleware, but the
	// chi router requires middleware with signature func(http.Handler) http.Handler.
	// We use the chi-compatible corsMiddleware above for dev CORS handling.
	runtimeBase := getEnv("SEMLAYER_RUNTIME_DIR", ".")
	auditSvc := audit.NewService(db)
	notificationSvc := services.NewEngagementNotificationService(db)
	campaignSvc := services.NewNotificationCampaignService(db, notificationSvc)

	// Analytics & Governance
	registerAnalyticsRoutes(r)

	// Initialize Semantic Name Resolver (pre-loads all name-to-UUID mappings)
	// Uses the database connection to load semantic field mappings
	semanticNameResolver := NewSemanticNameResolver(db)
	// Note: Refresh is called automatically in NewSemanticNameResolver,
	// but errors are logged and non-fatal to allow server to continue

	// Create the HTTP server
	if resolver == nil {
		resolver = security.NewDBDatasourceResolver(sqlxDB)
	}
	srv := &Server{
		DB:                     db,
		Reg:                    &Registry{DB: db}, // This needs to be adjusted based on the actual store structure
		WsHub:                  newWebSocketHub(),
		SemanticNameResolver:   semanticNameResolver,
		AuditSvc:               auditSvc,
		NotificationSvc:        notificationSvc,
		CampaignSvc:            campaignSvc,
		NotificationHandlers:   NewNotificationAPIHandlers(notificationSvc, campaignSvc),
		DashboardHandlers:      NewDashboardAPIHandlers(db),
		ModelCatalogHandler:    handlers.NewModelCatalogHandler(db),
		CatalogScanHandler:     catalogScanHandler, // Set early initialized handler
		DatasourceResolver:     resolver,
		TestConnectionHandler:  nil, // Will be set after initialization
		MetricRegistryHandler:  nil, // Will be set after initialization
		SemanticSvc:            nil, // Will be set after initialization
		Validate:               validator.New(),
		SecMgr:                 secMgr,
		SemanticCalculationSvc: nil, // Will be set after initialization
		CalculationHandler:     nil, // Will be set after initialization
		LineageSvc:             nil, // Will be set after initialization

		CueEngine: services.NewCueEngine(),

		CalcHandler: nil, // Will be set after initialization

		ExportHandlers:    nil, // Will be set after initialization
		SchedulerHandlers: nil, // Will be set after initialization
	}

	// Register trace proxy and metrics endpoints
	r.Get("/api/tempo/traces", srv.proxyTempoTraces)
	r.Get("/api/tempo/traces/{traceId}", srv.proxyTempoGetTrace)
	r.Get("/api/v1/metrics/commit", srv.commitMetricsV1Handler)

	// Observability Console endpoints
	r.Get("/api/metrics/global", srv.globalMetricsHandler)
	r.Get("/api/metrics/region-heatmap", srv.regionHeatmapHandler)
	r.Get("/api/metrics/tenant/{tenantId}", srv.tenantMetricsHandler)
	r.Get("/api/plans", srv.tenantPlansHandler)
	r.Get("/api/plans/timeline", srv.planTimelineHandler)
	r.Get("/api/iceberg/lineage", srv.icebergLineageHandler)

	// Commit metrics (versioned) — already registered above, skip duplicate
	// r.Get("/api/v1/metrics/commit", srv.commitMetricsV1Handler)

	// Initialize IP Whitelist handler
	ipWhitelistHandler := NewIpWhitelistAPIHandlers(db)

	// Initialize Tenant Access handler for multi-tenant access control
	tenantAccessHandler := NewTenantAccessHandlers(db)

	// Initialize BP Notification handlers
	// Note: sqlxDB is initialized below, so we need to move this or use db if compatible,
	// but NewBPNotificationHandlers takes *sqlx.DB.
	// Let's initialize sqlxDB first.

	// Shared sqlx DB handle for services needing sqlx features
	// sqlxDB initialized at top of function

	// Initialize semantic service for catalog-backed semantic object lookups
	srv.SemanticSvc = analytics.NewSemanticService(sqlxDB)
	// Initialize the business term matcher
	_ = services.NewSimpleFIBOMatcher()

	// Initialize Semantic Layer Handler
	semanticLayerService := semantic.NewService(sqlxDB)
	srv.SemanticLayerHandler = NewSemanticLayerHandler(semanticLayerService, srv.SemanticSvc, srv)

	// Initialize Lineage Service
	srv.LineageSvc = services.NewLineageService(sqlxDB)

	// Initialize LLM Provider (moved up for dependency injection)
	// Uses GEMINI_API_KEY from env by default
	llmProvider := llm.NewGeminiProvider("", "")

	// Assign Gemini client to server (passed from main.go)
	srv.GeminiClient = geminiClient
	if srv.GeminiClient != nil {
		logging.GetLogger().Sugar().Info("✅ Gemini client assigned to LLM gateway (Planner & Executor)")
	} else {
		logging.GetLogger().Sugar().Warn("⚠️  Gemini client not configured - LLM gateway will not work")
	}

	// Initialize Kafka audit publishers
	var auditPublisher *events.AuditEventPublisher
	var semanticPublisher *events.SemanticPublisher // New Semantic Publisher
	{
		kafkaBrokers := getEnv("KAFKA_BROKERS", "redpanda:9092")
		if kafkaBrokers != "" {
			auditPublisher = events.NewAuditEventPublisher(kafkaBrokers)
			semanticPublisher, _ = events.NewSemanticPublisher(kafkaBrokers) // Initialize Semantic Publisher
			logging.GetLogger().Sugar().Info("Audit and Semantic publishers initialized for Kafka")
		}
	}

	// Initialize abbreviation service for database-backed abbreviation expansion
	srv.AbbreviationSvc = services.NewAbbreviationService(sqlxDB, llmProvider)

	// Initialize analytics abbreviation service for semantic mapping
	analyticsAbbrevSvc := analytics.NewAbbreviationService(sqlxDB.DB, logging.GetLogger())

	// Initialize abbreviation handler
	abbreviationHandler := handlers.NewAbbreviationHandler(srv.AbbreviationSvc)

	// Initialize semantic mapping service with fuzzy logic and abbreviation support
	// Initialize semantic mapping service with fuzzy logic, abbreviation support, and auditing
	srv.SemanticMappingSvc = analytics.NewSemanticMappingService(sqlxDB, analytics.NewSimpleFIBOMatcher(), analyticsAbbrevSvc, llmProvider, semanticPublisher, sqlRepo)
	srv.SemanticMappingHandler = handlers.NewSemanticMappingHandler(srv.SemanticMappingSvc)

	// Initialize semantic calculation service and handler
	semanticCalculationSvc := analytics.NewSemanticCalculationService(sqlxDB)
	srv.SemanticCalculationSvc = semanticCalculationSvc

	// Initialize LLM config service (file-backed for dev/admin UI)
	llmCfgPath := filepath.Join(runtimeBase, ".runtime", "llm_config.json")
	llmCfgSvc := llm.NewLLMConfigService(llmCfgPath)
	_ = llmCfgSvc.EnsurePathDir()
	srv.LLMConfigSvc = llmCfgSvc

	// Initialize Search Service
	searchSvc := services.NewSearchService(sqlxDB, llmProvider)
	securityDeps := handlers.SecurityContextDeps{
		Resolver: srv.DatasourceResolver,
	}
	optService := optimize.NewService(sqlxDB)
	analyticsModelProvider := analytics.NewModelProvider(sqlxDB)
	queryService := analytics.NewQueryService(sqlxDB, optService, analyticsModelProvider)
	queryHandler := handlers.NewQueryHandler(queryService, securityDeps)
	srv.QueryHandler = queryHandler
	savedQueryHandler := handlers.NewSavedQueryHandler(queryService, securityDeps)
	srv.SavedQueryHandler = savedQueryHandler
	searchHandler := handlers.NewSearchHandler(searchSvc, securityDeps)
	srv.SearchHandler = searchHandler

	// Initialize Reasoning Engine
	reasoningEngine := services.NewReasoningEngine(llmProvider)

	// Initialize Async Feature Services
	exportStoragePath := getEnv("EXPORT_STORAGE_PATH", "./storage/exports")
	exportURLBasePath := getEnv("EXPORT_URL_BASE_PATH", "http://localhost:8080")
	exportService := services.NewPostgresExportService(db, exportStoragePath, exportURLBasePath)
	srv.ExportHandlers = handlers.NewExportHandlers(exportService)

	// Initialize Job Queue for async operations
	jobQueue := services.NewPostgresJobQueue(db)

	schedulerService := services.NewPostgresSchedulerService(db)
	srv.SchedulerHandlers = handlers.NewSchedulerHandlers(schedulerService)

	// Start Scheduler background loop
	if err := schedulerService.Start(context.Background(), jobQueue); err != nil {
		log.Printf("[Scheduler] Warning: Failed to start background loop: %v", err)
	}

	// Initialize NLQ Service
	nlqService := services.NewNLQService(sqlxDB, llmProvider, searchSvc, reasoningEngine, nil)
	srv.NLQService = nlqService
	nlqHandler := handlers.NewNLQHandler(nlqService, securityDeps)
	srv.NLQHandler = nlqHandler
	adminAPIKeyHandler := handlers.NewAdminAPIKeyHandler(apiKeyStore)
	srv.AdminAPIKeyHandler = adminAPIKeyHandler

	// Initialize Quality Services
	srv.FeedbackService = services.NewFeedbackService(sqlxDB)
	srv.EvalService = services.NewEvalService(sqlxDB, nlqService)

	// Initialize Cube Sync Service
	// Defaulting to a local 'cube_schema' directory for now
	_ = filepath.Join(runtimeBase, "cube_schema")
	srv.CubeSyncService = nil // Stub: NewCubeSyncService returns interface{}

	// --- Audit & History Wiring ---
	// initialize Trino Client for Bitemporal Tracker (Iceberg)
	trinoDSN := getEnv("TRINO_DSN", "http://admin@trino:8080?catalog=iceberg&schema=audit")
	trinoClient, err := trino.NewClient(trinoDSN)
	var auditHistoryHandler *handlers.AuditHistoryHandler

	if err != nil {
		logging.GetLogger().Sugar().Warnf("Failed to initialize Trino client for audit history: %v", err)
		// We still create the handler but maybe it should handle nil tracker gracefully?
		// For now we skip registration if trino fails, or register a dummy?
		// User expects 404 if not found? No, they want it to work.
		// If trino fails, we can't search history.
	} else {
		// Initialize Bitemporal Tracker
		bitemporalTracker := audit.NewBitemporalTracker(trinoClient)

		// Initialize Async Audit Service (Queue -> Worker -> Trino)
		// Buffer size 1000, 5 workers
		asyncAuditService := audit.NewAsyncAuditService(bitemporalTracker, 1000)
		asyncAuditService.Start(5)
		// Note: We should stop it on shutdown, but SetupRouter doesn't support shutdown hooks easily.
		// We rely on process termination or context cancellation propagated from main if we passed it.
		// For now, let it run until server exit.

		// Initialize Audit History Handler
		auditHistoryHandler = handlers.NewAuditHistoryHandler(bitemporalTracker, asyncAuditService)
		srv.AuditHistoryHandler = auditHistoryHandler
	}

	srv.CalculationHandler = handlers.NewCalculationHandler(semanticCalculationSvc)
	srv.ChartHandler = handlers.NewChartHandler(db)
	srv.ExecutionMonitorHandler = handlers.NewExecutionMonitorHandler(sqlxDB)

	// Initialize CatalogScanService and Handler
	// catalogScanService initialized at top
	srv.CatalogScanHandler = catalogScanHandler
	srv.TestConnectionHandler = handlers.NewTestConnectionHandler(catalogScanService)

	// Initialize relationship service
	srv.RelationshipSvc = NewRelationshipService(db)

	// Initialize relationship inference service and handler
	relationshipInferenceService := analytics.NewRelationshipInferenceService(sqlxDB)
	relationshipHandler := NewRelationshipHandler(relationshipInferenceService)
	srv.RelationshipHandler = relationshipHandler
	// catalogScanService initialized at top
	srv.CatalogScanHandler = catalogScanHandler
	srv.TestConnectionHandler = handlers.NewTestConnectionHandler(catalogScanService)

	// Upsert business term and create edge (atomic) - NOTE: moved under /api router registration below

	// Initialize services and handlers for Data Bundles
	policyService := services.NewPolicyService()
	// Use DB-backed bundle service wrapper which reads bundles directly from DB
	bundleService, _ := services.NewBundleServiceWithDB(policyService, sqlxDB)
	bundleHandler := handlers.NewBundleHandler(bundleService)

	// Initialize services and handlers for Role Management
	// roleService := services.NewRoleService(sqlxDB, policyService, bundleRoleManager)
	// roleHandler := handlers.NewRoleHandler(roleService)

	// Initialize services and handlers for Data Domain Management
	domainService := services.NewDomainService(sqlxDB)
	domainHandler := handlers.NewDomainHandler(domainService)

	// Initialize wealth management handler (registered via its RegisterRoutes where needed)
	_ = handlers.NewWealthManagementHandler(db)

	// Initialize DAX handler
	daxHandler := handlers.NewDAXHandler()

	srv.TemporalClient = temporalClient

	// Initialize timeout triggers handler
	timeoutTriggersHandler := handlers.NewTimeoutTriggersHandler(sqlxDB)

	// Initialize catalog scan service and handler - already done at top
	srv.CatalogScanHandler = catalogScanHandler

	// Initialize test connection handler
	testConnectionHandler := handlers.NewTestConnectionHandler(catalogScanService)
	srv.TestConnectionHandler = testConnectionHandler

	// Initialize Instance Clone Handler (Hasura Event Trigger)
	instanceCloneHandler := handlers.NewInstanceCloneHandler(sqlxDB)
	instanceCloneHandler.RegisterRoutes(r)

	// Initialize Admin Handler
	adminHandler := NewAdminHandler(qosManager)
	srv.AdminHandler = adminHandler

	// Initialize metric registry service and handler
	metricRegistryService := services.NewMetricRegistryService(sqlxDB)
	metricRegistryHandler := handlers.NewMetricRegistryHandler(metricRegistryService)
	srv.MetricRegistryHandler = metricRegistryHandler

	// Initialize Values Service (Hyper-Personalized Direct Indexing)
	valuesService := services.NewValuesService(sqlxDB, auditSvc)
	valuesHandler := handlers.NewValuesHandler(valuesService)
	srv.ValuesHandler = valuesHandler
	valuesHandler.RegisterRoutes(r)

	// Initialize Compliance Service - MOVED to main.go for Epic 12
	// complianceService := services.NewComplianceService(auditSvc)
	// complianceHandler := handlers.NewComplianceHandler(complianceService)
	// srv.ComplianceHandler = complianceHandler
	// complianceHandler.RegisterRoutes(r)

	// Initialize AI Service (Gemini Integration)
	aiRuleRepo := rules.NewSQLRuleRepository(db)
	aiScenarioSvc := rules.NewScenarioService(aiRuleRepo)
	aiService := ai.NewAIService(db, llmProvider, aiScenarioSvc)
	aiHandler := handlers.NewAIHandler(aiService)
	srv.AIHandler = aiHandler
	aiHandler.RegisterRoutes(r)

	// Initialize Source Preference Management (Phase 6)
	prefRepo := preference.NewRepository(db)
	prefSvc := preference.NewService(prefRepo)
	prefHandler := handlers.NewSourcePreferenceHandler(prefSvc)
	prefHandler.RegisterRoutes(r)

	// Initialize Portfolio Master Gold Copy (Phase 6 extension)
	pmRepo := portfoliomaster.NewRepository(db)
	pmSvc := portfoliomaster.NewService(pmRepo)
	pmHandler := handlers.NewPortfolioMasterHandler(pmSvc)
	pmHandler.RegisterRoutes(r)

	// Initialize Gold Copy Engine (full entity suite)
	// NOTE: GoldCopy routes will be registered inside the main /api Route block below
	gcPublisher, _ := services.NewGoldCopyPublisher(os.Getenv("KAFKA_BROKERS"))
	gcRepo := goldcopy.NewRepository(db, gcPublisher)
	gcEngine := goldcopy.NewEngine(gcRepo)
	gcHandler := handlers.NewGoldCopyHandler(gcEngine, gcRepo)

	// Initialize Phase 8: Advanced Cross-Domain Intelligence
	// 1. Semantic Graph Service for MDM
	mdmGraph := analytics.NewSemanticGraphService(sqlxDB)
	if db != nil {
		_ = mdmGraph.Initialize()
	}

	// 2. Execution Engine for recursive NAV/analytics
	execEngine, _ := mdm.NewExecutionEngine(context.Background(), mdmGraph, nil)
	srv.ExecutionEngine = execEngine

	// 3. Domain Services
	psSvc := mdm.NewPortfolioSecurityService(pmRepo, gcRepo, execEngine, mdmGraph)
	srv.PortfolioSecuritySvc = psSvc

	slSvc := mdm.NewSecurityLineageService(mdmGraph, gcRepo)
	srv.SecurityLineageSvc = slSvc

	// 4. Handlers
	paHandler := handlers.NewPortfolioAnalyticsHandler(db, psSvc)
	paHandler.RegisterAnalyticsRoutes(r)

	slHandler := handlers.NewSecurityLineageHandler(slSvc)
	slHandler.RegisterRoutes(r)

	// Initialize report service and handler
	reportService := reports.NewReportService(db)
	reportHandler := NewReportHandler(reportService)
	reportHandler.RegisterRoutes(r)

	// Initialize Storage Tiering (Phase 2)
	tieringRepo := tiering.NewTieringRepository(sqlxDB)
	tieringService := tiering.NewStorageTiering(tieringRepo, logging.GetLogger())
	tieringHandler := NewTieringHandler(tieringService)
	tieringHandler.RegisterRoutes(r)

	// Initialize Scheduler Intelligence (Epic 31)
	schedulerSemanticAdapter := si.NewSemanticAdapter(sqlxDB, srv.SemanticSvc)
	schedulerHandlers := NewSchedulerHandlers(sqlxDB, schedulerSemanticAdapter, logging.GetLogger())
	schedulerHandlers.RegisterRoutes(r)

	// Phase 13: Subscribe scheduler to storage events
	tieringService.Subscribe(schedulerHandlers.Service())

	// Initialize Governance (Epic 31 - Phase 11)
	schedulerRepo := si.NewRepository(sqlxDB)
	schedulerBlastRadius := si.NewBlastRadiusCalculator(schedulerRepo, schedulerSemanticAdapter)
	schedulerGovSvc := si.NewGovernanceService(schedulerRepo, schedulerSemanticAdapter, schedulerBlastRadius)
	schedulerAuditTrailSvc := si.NewAuditTrailService(schedulerRepo)
	governanceHandler := NewGovernanceHandler(schedulerGovSvc, schedulerAuditTrailSvc)
	governanceHandler.RegisterRoutes(r)

	// Initialize Business Components (Business Objects)
	// Uses platform tenant manager for multi-tenant data isolation
	tenantManager := platform.NewTenantDBManager(db)
	// Initialize Kafka audit publisher

	boService := catalogmeta.NewBusinessObjectService(sqlxDB, tenantManager, auditPublisher, sqlRepo)
	srv.BusinessObjectService = boService
	boHandler := NewBusinessObjectHandler(boService, srv.DatasourceResolver)
	// boHandler.RegisterRoutes(r) - Moved below into /api group

	// Initialize Catalog Handler (Phase 18)
	catalogHandler := NewCatalogHandler(boService)
	// Registration moved to /api group below

	// Initialize Semantic Terms handler for catalog_node queries
	semanticTermsHandler := NewSemanticTermsHandler(db)
	// Registration moved to /api group

	// Initialize Folder Service and Handler
	folderService := services.NewFolderService(sqlxDB)
	folderHandler := handlers.NewFolderHandler(folderService)
	srv.FolderHandler = folderHandler
	folderHandler.RegisterRoutes(r)

	// Initialize Graph-Native Lineage Service (Phase 12)
	// sqlRepo already created above
	lineageSvc := lineage.NewLineageService(sqlRepo)
	_ = lineageSvc // Suppress unused for now as it's passed around elsewhere or used for background tasks

	// Note: Registration moved to /api group below

	// Initialize Semantic Reporting handler (SSRS-style reporting on Cube.dev)
	cubeURL := getEnv("CUBE_API_URL", "http://cube:4000/cubejs-api/v1")
	semanticReportingHandler := NewSemanticReportingHandler(sqlxDB, cubeURL)
	semanticReportingHandler.RegisterRoutes(r)

	// Initialize Page Layouts and Pipelines
	pageLayoutHandler := handlers.NewPageLayoutHandler(sqlxDB)
	srv.PageLayoutHandler = pageLayoutHandler
	pageLayoutHandler.RegisterRoutes(r)

	pipelineHandler := handlers.NewPipelineHandler(sqlxDB, temporalClient)
	srv.PipelineHandler = pipelineHandler
	pipelineHandler.RegisterRoutes(r)

	// Initialize Events Ingestion Handler
	srv.EventsHandler = ingestion.NewEventsHandler(temporalClient)

	// Initialize GenAI Copilot Handler (Phase 6)
	srv.GenAICopilotHandler = handlers.NewGenAICopilotHandler(sqlxDB, temporalClient, llmCfgSvc)

	// Initialize Policy Generation Handler (Phase 9)
	srv.PolicyGenerationHandler = handlers.NewPolicyGenerationHandler(sqlxDB)

	// Initialize Hasura Client and Calc Handler (Phase 9)
	hasuraCfg := &hasuraclient.HasuraConfig{
		Endpoint:    getEnv("HASURA_GRAPHQL_ENDPOINT", "http://hasura:8080/v1/graphql"),
		AdminSecret: getEnv("HASURA_GRAPHQL_ADMIN_SECRET", "myadminsecret"),
	}
	hasura := hasuraclient.NewHasuraClient(hasuraCfg)
	srv.CalcHandler = handlers.NewCalcHandler(hasura)

	// Initialize Cube Client and Generator (Phase 9)
	cubeURL = getEnv("CUBE_API_URL", "http://cube:4000")
	cubeClient := cube.NewClient(cubeURL, "") // No secret for dev
	cubeTermRepo := &services.SQLTermRepository{DB: db}
	cubeGenerator := services.NewCubeGeneratorWithEngines(cubeTermRepo, srv.CueEngine, sqlxDB)
	srv.CubeHandler = NewCubeHandler(cubeClient, cubeGenerator)

	// Initialize RAG Services
	ragConfigService := rag.NewConfigService(db)
	ragTenantManager := tenant.NewTenantManager(db, trinoAuditService)
	// Use dummy key or load from env
	ragEmbedder := rag.NewOpenAIEmbedder("dummy-key", "text-embedding-ada-002")
	ragSearchService := rag.NewSearchService(ragEmbedder)

	srv.SQLXDB = sqlxDB
	ragHandler := NewRAGHandler(ragTenantManager, ragSearchService, temporalClient, ragConfigService)
	srv.RAGHandler = ragHandler

	// Register RAG Routes
	r.Route("/api/rag", func(r chi.Router) {
		r.Post("/search", ragHandler.HandleSearch)
		r.Post("/upload", ragHandler.HandleUpload)
	})

	// Shared model provider and view service for view management endpoints
	// modelProvider := services.NewModelProvider(sqlxDB)
	// viewService := services.NewViewService(sqlxDB)

	// Set up broadcasting functions for real-time notifications
	notificationSvc.SetBroadcastFunctions(
		func(userID string, message []byte) {
			srv.broadcastToUser(userID, message)
		},
		func(message []byte) {
			srv.WsHub.broadcast <- message
		},
		func(audience string, message []byte) {
			// For audience broadcasting, we'll send the raw message
			srv.WsHub.broadcastToAudience(audience, message)
		},
	)

	// Start the WebSocket hub
	go srv.WsHub.run()

	// DISABLED: Internal GraphQL engine - now using Hasura exclusively
	// RegisterGraphQLPlayground(r)

	// Start real-time updates simulation
	go srv.simulateRealTimeUpdates()

	// Register layout routes (saved UI layouts)
	srv.registerLayoutRoutes(r)

	// GraphQL proxy endpoints (proxied to Hasura)
	registerGraphQLProxyRoutesV2(r)

	// API routes
	routes := NewRoutes()

	r.Route("/api", func(r chi.Router) {
		// Register audit history routes if handler is active (INSIDE /api group)
		if auditHistoryHandler != nil {
			routes.RegisterAudit(r, auditHistoryHandler)
		}

		// Register Gold Copy Engine routes
		gcHandler.RegisterRoutes(r)

		// Register Catalog Node/Edge Type Routes
		RegisterNodeTypesRoutes(r, db)
		RegisterEdgeTypesRoutes(r, db)

		// Log unmatched /api requests for debugging 404s inside the /api subrouter
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			bodyBytes, _ := io.ReadAll(r.Body)
			fmt.Fprintf(os.Stderr, "[API-NOTFOUND] %s %s Body: %s\n", r.Method, r.URL.Path, string(bodyBytes))
			http.NotFound(w, r)
		})

		// Auth routes must come BEFORE middleware to avoid chicken-and-egg problem
		registerAuthRoutes(r, srv)

		// Policy Generation Requests (No auth required for prototype, but should be protected)
		// Adding it here before middleware for simplicity if needed, but optimally after.
		// Let's rely on middleware.

		// Middleware must be applied to specific groups or routes if added after router init
		// But chi.Router.Use() applies to all FUTURE routes registered on the router or group.
		// Since we are inside the r.Route("/api", ...) block, and most routes are below,
		// we can enable the middleware here and it will apply to the routes registered below.

		// Enable Auth Context Middleware to parse tokens and set X-User-ID header
		// Middleware was causing panic here because it was added after routes were registered above.
		// Moving it to a specific group below.

		// Note: We are NOT enabling SessionAuthMiddleware yet as it might conflict or isn't needed for this flow.
		// r.Use(appmid.SessionAuthMiddleware(appmid.SessionAuthConfig{DB: db, SessionCookie: "session_token", AllowBearerFallback: true}))
		// r.Use(srv.auditMiddleware())

		// DISABLED: Internal GraphQL API endpoint - now using Hasura exclusively
		// RegisterGraphQLAPI(r, db)

		// --- Modular Service Registrations ---
		srv.registerTemplateRoutes(r)
		srv.registerCalculationRoutes(r)
		srv.registerAuditTrinoRoutes(r)
		srv.registerAIRoutes(r)
		srv.registerExplorerRoutes(r)
		srv.registerEventRoutes(r)

		// Register Async Feature Routes
		if srv.ExportHandlers != nil {
			routes.RegisterExports(r, srv.ExportHandlers)
		}
		if srv.SchedulerHandlers != nil {
			routes.RegisterScheduler(r, srv.SchedulerHandlers)
		}
		srv.registerAdminRoutes(r)
		srv.registerLineageRoutes(r)
		srv.registerDebugRoutes(r)
		srv.registerSemanticRoutes(r)
		srv.registerLLMGatewayRoutes(r)
		srv.registerMetadataRoutes(r, boHandler, catalogHandler, boService)
		srv.registerCatalogRoutes(r, db, routes, temporalClient)
		srv.registerWorkflowRoutes(r, db, cron.New())
		srv.registerProcessRoutes(r, db, sqlxDB)
		srv.registerTriggerEngineRoutes(r, sqlxDB)
		srv.registerNBAEngineRoutes(r, sqlxDB)
		srv.registerBillingRoutes(r)
		srv.registerFeedbackRoutes(r)
		srv.registerTemporalAdminRoutes(r, db)
		srv.registerAlphaTemporalRoutes(r, temporalClient)
		srv.registerSemanticMappingRoutes(r)
		srv.registerCatalogNodeRoutes(r)
		srv.registerTemporalWebhookRoute(r)

		// Initializations & External Handlers
		igniteAddr := os.Getenv("IGNITE_ADDR")
		if igniteAddr == "" {
			igniteAddr = "ignite:10800"
		}
		var errIgnite error
		srv.IgniteClient, errIgnite = infrastructure.NewIgniteClient(igniteAddr)
		if errIgnite != nil {
			fmt.Printf("Warning: Failed to connect to Ignite: %v. Cache operations will be skipped.\n", errIgnite)
		}

		srv.GraphService = catalogmeta.NewGraphService(sqlxDB)
		abacService := services.NewAbacService(sqlxDB)
		srv.WriteHandler = handlers.NewWriteHandler(srv.GraphService, sqlxDB, srv.IgniteClient, abacService)
		srv.MCPHandler = handlers.NewMCPHandler(srv.GraphService)

		semanticTermsHandler.RegisterRoutes(r)
		srv.GenAICopilotHandler.RegisterRoutes(r)
		srv.ChartHandler.RegisterRoutes(r)
		routes.RegisterMetadataWrite(r, srv.WriteHandler)

		routes.RegisterMCP(r, srv.MCPHandler)

		// Register handlers that were previously orphaned
		ipWhitelistHandler.RegisterRoutes(r)
		tenantAccessHandler.RegisterRoutes(r)
		abbreviationHandler.RegisterRoutes(r)
		bundleHandler.RegisterRoutes(r)
		domainHandler.RegisterRoutes(r)
		daxHandler.RegisterRoutes(r)
		timeoutTriggersHandler.RegisterRoutes(r)

		// Initialize refactored handlers
		semanticMappingsHandler := NewSemanticMappingsHandler(srv.SemanticMappingSvc, srv.SemanticSvc, sqlxDB)
		businessTermsHandler := NewBusinessTermsHandler(srv.SemanticMappingSvc, sqlxDB)
		if semanticTermsHandler != nil {
			semanticTermsHandler.SetService(srv.SemanticMappingSvc)
		}

		semanticMappingsHandler.RegisterRoutes(r)
		businessTermsHandler.RegisterRoutes(r)
		RegisterValidationRulesRoutes(r, db, srv.CueEngine, srv.BusinessObjectService, srv.DatasourceResolver)

		adminHandler.RegisterRoutes(r)
// Admin Impersonation Routes
		if db != nil {
			impersonateHandler := handlers.NewAdminImpersonateHandler(db)
			r.Post("/admin/impersonate", impersonateHandler.AssumeContext)
			r.Delete("/admin/impersonate/{sessionId}", impersonateHandler.ExitContext)

			// Tenant search + scope (impersonation picker)
			tenantSearchHandler := handlers.NewAdminTenantSearchHandler(db)
			r.Get("/admin/tenants/search", tenantSearchHandler.SearchTenants)
			r.Get("/admin/tenants/{tenantID}/scope", tenantSearchHandler.GetTenantScope)
		}

		// WebSocket token issuance

		// Admin Impersonation Routes
		if db != nil {
			impersonateHandler := handlers.NewAdminImpersonateHandler(db)
			r.Post("/admin/impersonate", impersonateHandler.AssumeContext)
			r.Delete("/admin/impersonate/{sessionId}", impersonateHandler.ExitContext)
		}

		// WebSocket token issuance
		r.Post("/ws/token", srv.getWsToken)

		// Legacy Compatibility & Misc
	})

	// Debug: dump registered routes to stderr to help diagnose missing handlers during local runs
	// This prints method + pattern for each registered route.
	fmt.Fprintf(os.Stderr, "[ROUTES-DUMP-START]\n")
	_ = chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		fmt.Fprintf(os.Stderr, "[ROUTE] %s %s\n", method, route)
		return nil
	})
	fmt.Fprintf(os.Stderr, "[ROUTES-DUMP-END]\n")

	// Wrap everything in a root Mux to allow raw SSE handling without middleware
	// This avoids the "middleware defined after routes" panic while still bypassing cache/buffer
	rootMux := chi.NewRouter()
	rootMux.Get("/api/catalog/scan/stream", catalogScanHandler.HandleScanStream)

	// Add OPTIONS handler for CORS preflight for SSE
	rootMux.Options("/api/catalog/scan/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.WriteHeader(http.StatusOK)
	})

	// --- Google Calendar Sync ---
	if redisClient != nil {
		logging.GetLogger().Sugar().Info("Initializing Google Calendar Sync...")

		// Token Encryption
		encKey := os.Getenv("OAUTH_TOKEN_ENCRYPTION_KEY")
		if encKey == "" {
			logging.GetLogger().Sugar().Warn("OAUTH_TOKEN_ENCRYPTION_KEY not set. Generating ephemeral key tokens will be lost on restart.")
		}

		var tokenEncryptor *security.TokenEncryptor
		if len(encKey) == 32 {
			var err error
			tokenEncryptor, err = security.NewTokenEncryptor([]byte(encKey))
			if err != nil {
				logging.GetLogger().Sugar().Errorf("Failed to create token encryptor: %v", err)
			}
		}

		// OAuth Provider
		googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
		googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
		// Host URL for redirect
		serverURL := os.Getenv("SERVER_URL")
		if serverURL == "" {
			serverURL = "http://localhost:8080"
		}
		redirectURL := serverURL + "/auth/google/callback"

		if googleClientID != "" && googleClientSecret != "" && tokenEncryptor != nil {
			oauthProvider := oauth.NewGoogleOAuth2Provider(
				googleClientID,
				googleClientSecret,
				redirectURL,
				redisClient,
				tokenEncryptor,
			)

			// Sync Repo
			hasuraURL := os.Getenv("HASURA_URL")
			if hasuraURL == "" {
				hasuraURL = "http://localhost:8085/v1/graphql"
			}
			hasuraAdminSecret := os.Getenv("HASURA_GRAPHQL_ADMIN_SECRET")
			if hasuraAdminSecret == "" {
				hasuraAdminSecret = os.Getenv("HASURA_ADMIN_SECRET")
				if hasuraAdminSecret == "" {
					hasuraAdminSecret = "myadminsecret"
				}
			}

			syncHasuraClient := hasuraclient.NewHasuraClient(&hasuraclient.HasuraConfig{
				Endpoint:    hasuraURL,
				AdminSecret: hasuraAdminSecret,
			})

			googleSyncRepo := repository.NewGoogleSyncRepo(syncHasuraClient)

			// Sync Processor
			// Note: SyncProcessor uses logrus, while the rest of the app uses zap.
			// We create a logrus logger here for compatibility.
			logrusLogger := logrus.New()
			logrusEntry := logrus.NewEntry(logrusLogger)

			syncProcessor := calendarsync.NewSyncProcessor(
				oauthProvider,
				googleSyncRepo,
				logrusEntry,
				10, // Max concurrent
			)

			// Sync Handler
			syncHandler := NewSyncHandler(oauthProvider, syncProcessor, googleSyncRepo, logrusEntry)
			syncHandler.RegisterRoutes(r)

			// Conflict Handler
			conflictHandler := NewConflictHandler(googleSyncRepo, logrusEntry)
			conflictHandler.RegisterRoutes(r)

			// Internal Event Handler (for testing bi-directional sync)
			// Need a publisher
			brokers := getEnv("KAFKA_BROKERS", "redpanda:9092")
			eventPublisher, _ := services.NewEventPublisher(brokers) // Ignoring error for brevity, logs inside
			internalEventService := services.NewInternalEventService(googleSyncRepo, eventPublisher)
			internalEventHandler := NewInternalEventHandler(internalEventService, logrusEntry)
			internalEventHandler.RegisterRoutes(r)

			// Google Sync Listener
			kafkaBrokers := getEnv("KAFKA_BROKERS", "redpanda:9092")
			eventConsumer, err := services.NewEventConsumer(kafkaBrokers, "google-sync-service")
			if err != nil {
				logging.GetLogger().Sugar().Warnf("Failed to init event consumer for google sync: %v", err)
			} else {
				listener := calendarsync.NewGoogleSyncListener(syncProcessor, logrusEntry, eventConsumer)
				listener.Start()
			}

			logging.GetLogger().Sugar().Info("✅ Google Calendar Sync initialized")
		} else {
			logging.GetLogger().Sugar().Warn("Skipping Google Calendar Sync (missing credentials or encryption key)")
		}
	}

	rootMux.Mount("/", r)
	return rootMux
}

func (s *Server) listTemplates(w http.ResponseWriter, r *http.Request) {
	filter := map[string]string{
		"domain":      r.URL.Query().Get("domain"),
		"category":    r.URL.Query().Get("category"),
		"subcategory": r.URL.Query().Get("subcategory"),
		"status":      r.URL.Query().Get("status"),
	}
	tag := r.URL.Query().Get("tag")
	rows, err := s.Reg.ListTemplates(r.Context(), filter, tag)
	respond(w, r, rows, err)
}

// handleGenerateDefaults processes POST /api/fabric/models/generate-defaults
// Extracted into a function so it can be unit-tested with a mocked semantic service.
func handleGenerateDefaults(w http.ResponseWriter, r *http.Request, semanticSvc interface{}) {
	// We'll decode into a small struct
	var payload struct {
		DatasourceID string `json:"datasource_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request format", "invalid_request", nil)
		return
	}
	if strings.TrimSpace(payload.DatasourceID) == "" {
		writeJSONError(w, http.StatusBadRequest, "datasource_id is required", "missing_datasource_id", nil)
		return
	}
	dsUUID, err := uuid.Parse(payload.DatasourceID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid datasource_id", "invalid_datasource_id", err.Error())
		return
	}

	// We need to call the semantic service method; reflection-free approach: type assert to expected interface if possible
	type genDefaulter interface {
		GenerateDefaultSemanticModel(uuid.UUID) ([]*coremodels.FabricDefn, error)
	}
	svc, ok := semanticSvc.(genDefaulter)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "semantic service not available", "service_unavailable", nil)
		return
	}

	modelsCreated, err := svc.GenerateDefaultSemanticModel(dsUUID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeJSONError(w, http.StatusNotFound, err.Error(), "not_found", nil)
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "internal_error", nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"models": modelsCreated, "success": true})
}

func (s *Server) getTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "node_id")
	t, err := s.Reg.GetTemplate(r.Context(), id)
	respond(w, r, t, err)
}

func (s *Server) listVersions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "node_id")
	rows, err := s.DB.QueryContext(r.Context(), `SELECT version, schema_hash, created_at FROM public.template_versions WHERE node_id=$1 ORDER BY created_at DESC`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var out []map[string]any
	for rows.Next() {
		var v, h string
		var ts time.Time
		if err := rows.Scan(&v, &h, &ts); err != nil {
			respond(w, r, nil, err)
			return
		}
		out = append(out, map[string]any{"version": v, "schema_hash": h, "created_at": ts})
	}
	respond(w, r, out, nil)
}

func (s *Server) saveTemplate(w http.ResponseWriter, r *http.Request) {
	var t Template
	// Use a decoder to prevent unknown fields, enforcing the contract.
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&t); err != nil {
		http.Error(w, "Invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: Add schema validation
	// if err := validate.Validate(t); err != nil {
	//     http.Error(w, "Schema validation failed: "+err.Error(), http.StatusUnprocessableEntity)
	//     return
	// }

	err := s.Reg.UpsertTemplate(r.Context(), &t)
	respond(w, r, map[string]string{"ok": "true", "node_id": t.NodeID}, err)
}

func (s *Server) promoteTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "node_id")
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "reviewed"
	}
	// Basic validation for status
	validStatuses := map[string]bool{"draft": true, "reviewed": true, "golden": true, "deprecated": true}
	if !validStatuses[status] {
		http.Error(w, "Invalid status value", http.StatusBadRequest)
		return
	}

	_, err := s.Reg.PromoteTemplate(r.Context(), id, status)
	respond(w, r, map[string]string{"status": status}, err)
}

func (s *Server) runVectorizedCalculations(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Metrics  []string `json:"metrics"`
		Entities []string `json:"entities"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Metrics) == 0 || len(req.Entities) == 0 {
		http.Error(w, "metrics and entities arrays cannot be empty", http.StatusBadRequest)
		return
	}

	// Execute vectorized calculations
	results, err := DispatchVectorized(req.Metrics, req.Entities, sqlx.NewDb(s.DB, "postgres"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"results": results,
		"batch_info": map[string]interface{}{
			"metric_count":       len(req.Metrics),
			"entity_count":       len(req.Entities),
			"total_calculations": len(req.Metrics) * len(req.Entities),
		},
	}

	respond(w, r, response, nil)
}

// resolveNodeIDsToSchemaTables converts catalog node IDs to schema/table pairs
func (s *Server) resolveNodeIDsToSchemaTables(ctx context.Context, tenantID, datasourceID string, nodeIDs []string) ([]SchemaTable, error) {
	if len(nodeIDs) == 0 {
		return nil, nil
	}

	// Build query to get qualified_path for the node IDs
	placeholders := make([]string, len(nodeIDs))
	args := make([]interface{}, len(nodeIDs)+2)
	args[0] = tenantID
	args[1] = datasourceID
	for i, nodeID := range nodeIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+3)
		args[i+2] = nodeID
	}

	query := fmt.Sprintf(`
		SELECT id, qualified_path
		FROM public.catalog_node
		WHERE tenant_id = $1 AND tenant_datasource_id = $2 AND id IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemaTables []SchemaTable
	for rows.Next() {
		var nodeID, qualifiedPath string
		if err := rows.Scan(&nodeID, &qualifiedPath); err != nil {
			continue
		}

		// Parse qualified_path to extract schema and table
		// Format can be "/schema/table" for tables or "/schema/table/column" for columns
		if strings.HasPrefix(qualifiedPath, "/") {
			parts := strings.Split(strings.TrimPrefix(qualifiedPath, "/"), "/")
			if len(parts) >= 2 {
				schemaTables = append(schemaTables, SchemaTable{
					Schema: parts[0],
					Table:  parts[1],
				})
			}
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	var unique []SchemaTable
	for _, st := range schemaTables {
		key := st.Schema + "." + st.Table
		if !seen[key] {
			seen[key] = true
			unique = append(unique, st)
		}
	}

	return unique, nil
}

// Private Markets Explorer API endpoints

func (s *Server) getUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	// Get role from URL params
	role := r.URL.Query().Get("role")
	if role == "" {
		role = "lp" // default
	}

	// Query user from database
	var user models.User
	query := `
		SELECT id, email, name, role, organization, permissions, is_core_admin, is_active
		FROM public.users
		WHERE (
			char_length($1) = 36
			AND $1 ~ '^[0-9a-fA-F0-9-]{36}$'
			AND id = $1::uuid
		) OR (email = $1)
		AND is_active = true
		LIMIT 1`

	var permissions []byte
	var email string
	err := s.DB.QueryRowContext(r.Context(), query, userID).Scan(
		&user.ID, &email, &user.Name, &user.Role, &user.Organization, &permissions, &user.IsCoreAdmin, &user.IsActive,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return mock user for development if not found in database
			user = models.User{
				ID:           userID,
				Name:         "John Doe",
				Role:         role,
				Organization: "Sample Organization",
				Permissions:  []string{"read", "write", "admin"},
			}
		} else {
			respond(w, r, nil, err)
			return
		}
	}

	// Parse permissions JSON if from database
	if permissions != nil {
		if err := json.Unmarshal(permissions, &user.Permissions); err != nil {
			// If JSON parsing fails, use default permissions
			user.Permissions = []string{"read"}
		}
	}
	user.Email = email
	if user.Email == "admin@example.com" {
		user.IsCoreAdmin = true
	}

	respond(w, r, user, nil)
}

func (s *Server) listBundles(w http.ResponseWriter, r *http.Request) {
	audience := r.URL.Query().Get("audience")
	if audience == "" {
		audience = "lp" // default
	}

	// Query bundles from database
	query := `
		SELECT bundle_id, name, audience, version, modules, metrics, governance
		FROM private_markets_bundles
		WHERE audience = $1 AND is_active = true
		ORDER BY name`

	rows, err := s.DB.QueryContext(r.Context(), query, audience)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	defer rows.Close()

	var bundles []models.Bundle
	for rows.Next() {
		var bundle models.Bundle
		var modules, metrics, governance []byte

		err := rows.Scan(
			&bundle.ID, &bundle.Name, &bundle.Audience, &bundle.Version,
			&modules, &metrics, &governance,
		)
		if err != nil {
			respond(w, r, nil, err)
			return
		}

		// Parse JSON fields
		if err := json.Unmarshal(modules, &bundle.Modules); err != nil {
			bundle.Modules = []models.BundleModule{}
		}
		if err := json.Unmarshal(metrics, &bundle.Metrics); err != nil {
			bundle.Metrics = []models.BundleMetric{}
		}
		if err := json.Unmarshal(governance, &bundle.Governance); err != nil {
			bundle.Governance = models.BundleGovernance{}
		}

		bundles = append(bundles, bundle)
	}

	// If no bundles found in database, return mock data for development
	env := os.Getenv("ENVIRONMENT")
	if len(bundles) == 0 && (env == "development" || env == "dev") {
		bundles = s.getMockBundles(audience)
	}

	respond(w, r, bundles, nil)
}

// getMockBundles returns mock bundle data for development when database is empty
func (s *Server) getMockBundles(audience string) []models.Bundle {
	switch audience {
	case "lp":
		return []models.Bundle{
			{
				ID:       "lp_private_markets_bundle",
				Name:     "LP Private Markets Bundle",
				Audience: "lp",
				Version:  "1.0.0",
				Modules: []models.BundleModule{
					{ID: "fund-selector", Name: "Fund Selector", Type: "selector", Config: map[string]interface{}{"multiSelect": true}},
					{ID: "irr-curve", Name: "IRR Curve Chart", Type: "chart", Config: map[string]interface{}{"timeRange": "5y"}},
					{ID: "j-curve", Name: "J-Curve Plot", Type: "chart", Config: map[string]interface{}{"showBenchmark": true}},
					{ID: "benchmark-comparison", Name: "Benchmark Comparison", Type: "comparison", Config: map[string]interface{}{"indices": []string{"S&P 500", "NASDAQ"}}},
					{ID: "liquidity-panel", Name: "Liquidity Panel", Type: "panel", Config: map[string]interface{}{"showProjections": true}},
				},
				Metrics: []models.BundleMetric{
					{ID: "tvpi", Name: "TVPI", Type: "ratio", Formula: "(distributions + residual_value) / paid_in_capital"},
					{ID: "irr", Name: "IRR", Type: "percentage", Formula: "XIRR(cash_flows, dates)"},
					{ID: "pme", Name: "PME", Type: "ratio", Formula: "PME(cash_flows, benchmark)"},
				},
				Governance: models.BundleGovernance{
					Status:       "active",
					StewardGroup: "data-stewards",
					SchemaHash:   "abc123",
					SLA:          models.BundleSLA{RefreshFrequency: "daily", MaxLatency: "4h"},
				},
			},
		}
	case "gp":
		return []models.Bundle{
			{
				ID:       "gp_private_markets_bundle",
				Name:     "GP Private Markets Bundle",
				Audience: "gp",
				Version:  "1.0.0",
				Modules: []models.BundleModule{
					{ID: "deployment-pacing", Name: "Deployment Pacing Chart", Type: "chart", Config: map[string]interface{}{"targetPacing": "24months"}},
					{ID: "irr-nav-tracking", Name: "IRR/NAV Tracking", Type: "tracking", Config: map[string]interface{}{"frequency": "quarterly"}},
					{ID: "fee-analysis", Name: "Fee Analysis", Type: "analysis", Config: map[string]interface{}{"feeTypes": []string{"management", "performance"}}},
					{ID: "value-attribution", Name: "Value Attribution", Type: "attribution", Config: map[string]interface{}{"methodology": "brinson"}},
					{ID: "exit-analysis", Name: "Exit Analysis", Type: "analysis", Config: map[string]interface{}{"exitTypes": []string{"ipo", "merger", "sale"}}},
				},
				Metrics: []models.BundleMetric{
					{ID: "dpi", Name: "DPI", Type: "ratio", Formula: "distributions / paid_in_capital"},
					{ID: "rvpi", Name: "RVPI", Type: "ratio", Formula: "residual_value / paid_in_capital"},
					{ID: "tvpi", Name: "TVPI", Type: "ratio", Formula: "dpi + rvpi"},
				},
				Governance: models.BundleGovernance{
					Status:       "active",
					StewardGroup: "gp-stewards",
					SchemaHash:   "def456",
					SLA:          models.BundleSLA{RefreshFrequency: "weekly", MaxLatency: "24h"},
				},
			},
		}
	case "fof":
		return []models.Bundle{
			{
				ID:       "fof_private_markets_bundle",
				Name:     "FoF Private Markets Bundle",
				Audience: "fof",
				Version:  "1.0.0",
				Modules: []models.BundleModule{
					{ID: "portfolio-overview", Name: "Portfolio Overview", Type: "overview", Config: map[string]interface{}{"groupBy": "strategy"}},
					{ID: "manager-performance", Name: "Manager Performance", Type: "performance", Config: map[string]interface{}{"benchmark": true}},
					{ID: "allocation-analysis", Name: "Allocation Analysis", Type: "analysis", Config: map[string]interface{}{"dimensions": []string{"geography", "vintage"}}},
					{ID: "risk-attribution", Name: "Risk Attribution", Type: "attribution", Config: map[string]interface{}{"method": "factor"}},
				},
				Metrics: []models.BundleMetric{
					{ID: "portfolio-irr", Name: "Portfolio IRR", Type: "percentage", Formula: "weighted_average(irr)"},
					{ID: "diversification", Name: "Diversification Score", Type: "score", Formula: "1 - concentration_ratio"},
					{ID: "alpha", Name: "Alpha vs Benchmark", Type: "percentage", Formula: "irr - benchmark_irr"},
				},
				Governance: models.BundleGovernance{
					Status:       "active",
					StewardGroup: "fof-stewards",
					SchemaHash:   "ghi789",
					SLA:          models.BundleSLA{RefreshFrequency: "monthly", MaxLatency: "48h"},
				},
			},
		}
	default:
		return []models.Bundle{}
	}
}

func (s *Server) listFunds(w http.ResponseWriter, r *http.Request) {
	// Try to get funds from database first
	rows, err := s.DB.QueryContext(r.Context(), `
		SELECT id, name, vintage, manager, strategy, geography, status, created_at, updated_at
		FROM private_markets_funds
		ORDER BY created_at DESC
	`)
	if err != nil {
		// If database query fails, return mock data
		funds := s.getMockFunds()
		respond(w, r, funds, nil)
		return
	}
	defer rows.Close()

	var funds []models.Fund
	for rows.Next() {
		var fund models.Fund
		err := rows.Scan(
			&fund.ID,
			&fund.Name,
			&fund.Vintage,
			&fund.Manager,
			&fund.Strategy,
			&fund.Geography,
			&fund.Status,
			&fund.CreatedAt,
			&fund.UpdatedAt,
		)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		funds = append(funds, fund)
	}

	// If no funds found in database, return mock data for development
	env := os.Getenv("ENVIRONMENT")
	if len(funds) == 0 && (env == "development" || env == "dev") {
		funds = s.getMockFunds()
	}

	respond(w, r, funds, nil)
}

// getMockFunds returns mock fund data for development when database is empty
func (s *Server) getMockFunds() []models.Fund {
	return []models.Fund{
		{
			ID:        "fund-1",
			Name:      "Tech Growth Fund III",
			Vintage:   2020,
			Manager:   "TechVentures Capital",
			Strategy:  "Venture Capital",
			Geography: "North America",
			Status:    "active",
			CreatedAt: time.Now().AddDate(0, 0, -30),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "fund-2",
			Name:      "Infrastructure Partners II",
			Vintage:   2019,
			Manager:   "Global Infra Investments",
			Strategy:  "Infrastructure",
			Geography: "Global",
			Status:    "active",
			CreatedAt: time.Now().AddDate(0, 0, -45),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "fund-3",
			Name:      "Real Estate Opportunity Fund",
			Vintage:   2021,
			Manager:   "Urban Property Group",
			Strategy:  "Real Estate",
			Geography: "Europe",
			Status:    "active",
			CreatedAt: time.Now().AddDate(0, 0, -15),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "fund-4",
			Name:      "Healthcare Innovation Fund",
			Vintage:   2022,
			Manager:   "MedTech Ventures",
			Strategy:  "Healthcare",
			Geography: "North America",
			Status:    "active",
			CreatedAt: time.Now().AddDate(0, 0, -5),
			UpdatedAt: time.Now(),
		},
	}
}

func (s *Server) getFundMetrics(w http.ResponseWriter, r *http.Request) {
	fundID := chi.URLParam(r, "fundId")

	// Try to get metrics from database first
	var metrics models.FundMetrics
	err := s.DB.QueryRowContext(r.Context(), `
		SELECT fund_id, tvpi, rvpi, irr, xirr, pme, paid_in_capital, distributions, residual_value, as_of_date
		FROM private_markets_metrics
		WHERE fund_id = $1
		ORDER BY as_of_date DESC
		LIMIT 1
	`, fundID).Scan(
		&metrics.FundID,
		&metrics.TVPI,
		&metrics.RVPI,
		&metrics.IRR,
		&metrics.XIRR,
		&metrics.PME,
		&metrics.PaidInCapital,
		&metrics.Distributions,
		&metrics.ResidualValue,
		&metrics.AsOfDate,
	)

	if err != nil {
		env := os.Getenv("ENVIRONMENT")
		if env == "development" || env == "dev" {
			// If database query fails or no data found, return mock data for development
			metrics = s.getMockFundMetrics(fundID)
		} else {
			respond(w, r, nil, err)
			return
		}
	}

	respond(w, r, metrics, nil)
}

// getMockFundMetrics returns mock metrics data for development when database is empty
func (s *Server) getMockFundMetrics(fundID string) models.FundMetrics {
	metrics := models.FundMetrics{
		FundID:        fundID,
		PaidInCapital: 100000000,
		Distributions: 85000000,
		ResidualValue: 123000000,
		AsOfDate:      time.Now(),
	}

	// Customize metrics based on fund ID
	switch fundID {
	case "fund-1": // Tech Growth Fund
		metrics.TVPI = 1.85
		metrics.RVPI = 1.23
		metrics.IRR = 0.156
		metrics.XIRR = 0.142
		metrics.PME = 1.12
	case "fund-2": // Infrastructure
		metrics.TVPI = 1.65
		metrics.RVPI = 1.45
		metrics.IRR = 0.123
		metrics.XIRR = 0.118
		metrics.PME = 1.08
	case "fund-3": // Real Estate
		metrics.TVPI = 1.92
		metrics.RVPI = 1.67
		metrics.IRR = 0.145
		metrics.XIRR = 0.138
		metrics.PME = 1.15
	case "fund-4": // Healthcare
		metrics.TVPI = 2.05
		metrics.RVPI = 1.89
		metrics.IRR = 0.178
		metrics.XIRR = 0.165
		metrics.PME = 1.22
	default:
		metrics.TVPI = 1.75
		metrics.RVPI = 1.35
		metrics.IRR = 0.135
		metrics.XIRR = 0.128
		metrics.PME = 1.10
	}

	return metrics
}

// Authentication handlers were moved to auth_handlers.go

// Helper methods

// NOTE: session/token helpers were refactored into the auth package. These
// local helpers removed to avoid unused-code warnings.

// WebSocket Handler
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Support two WS auth flows:
	// 1) Job-scoped connections: path /ws/profiler/<jobId> with short-lived token query param
	// 2) Generic session-based connections: Authorization: Bearer <session_token>
	path := r.URL.Path
	var userID, audience string
	var jobAudience string

	if strings.HasPrefix(path, "/ws/profiler/") {
		// Extract jobId from path
		jobId := strings.TrimPrefix(path, "/ws/profiler/")
		if jobId == "" {
			http.Error(w, "missing job id", http.StatusBadRequest)
			return
		}

		// Expect short-lived token in query param
		tokenString := r.URL.Query().Get("token")
		if tokenString == "" {
			http.Error(w, "token required for profiler websocket", http.StatusUnauthorized)
			return
		}

		claims, err := s.validateWsToken(tokenString, jobId)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid token: %v", err), http.StatusUnauthorized)
			return
		}
		// Optionally enforce tenant/datasource scoping if present in claims
		if t, ok := claims["tenant_id"].(string); ok && t != "" {
			r.Header.Set("X-Tenant-ID", t)
		}
		if ds, ok := claims["datasource_id"].(string); ok && ds != "" {
			r.Header.Set("X-Tenant-Datasource-ID", ds)
		}

		jobAudience = jobId
	} else {
		// Session auth flow
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "No token provided", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, "Invalid token format", http.StatusUnauthorized)
			return
		}

		// Verify session token via DB
		err := s.DB.QueryRowContext(r.Context(), `
			SELECT u.id, u.role
			FROM private_markets_sessions s
			JOIN public.users u ON s.user_id = u.id
			WHERE s.session_token = $1 AND s.expires_at > now() AND s.is_active = true
		`, tokenString).Scan(&userID, &audience)

		if err != nil {
			http.Error(w, "Invalid session", http.StatusUnauthorized)
			return
		}
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}

	// Create WebSocket client; for job-scoped connections set audience to job id so hub can target
	client := &WebSocketClient{
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   userID,
		audience: audience,
		hub:      s.WsHub,
	}
	if jobAudience != "" {
		client.audience = jobAudience
	}

	// Register client
	client.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// Real-time broadcasting methods

func (s *Server) broadcastFundUpdate(fundID string, metrics map[string]interface{}) {
	message := RealTimeMessage{
		Type: "fund_update",
		Data: FundUpdateMessage{
			FundID:    fundID,
			Metrics:   metrics,
			UpdatedAt: time.Now(),
		},
		Timestamp: time.Now(),
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return
	}

	// Broadcast to all connected clients (in production, you might want to filter by audience)
	s.WsHub.broadcast <- messageBytes
}

func (s *Server) broadcastToUser(userID string, message []byte) {
	s.WsHub.mutex.RLock()
	defer s.WsHub.mutex.RUnlock()

	for client := range s.WsHub.clients {
		if client.userID == userID {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(s.WsHub.clients, client)
			}
		}
	}
}

// Example method to simulate real-time updates (call this when fund metrics are updated)
func (s *Server) simulateRealTimeUpdates() {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			// Simulate fund metric updates
			metrics := map[string]interface{}{
				"tvpi": 1.85 + float64(time.Now().Unix()%10)*0.01,
				"irr":  0.156 + float64(time.Now().Unix()%5)*0.001,
			}
			s.broadcastFundUpdate("fund-1", metrics)
		}
	}()
}

// Helper functions for creating pointers
// ...existing code...

// getBundleByDomain retrieves a bundle from the semantic layer registry by domain
func (s *Server) getBundleByDomain(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	if domain == "" {
		http.Error(w, "Domain parameter is required", http.StatusBadRequest)
		return
	}

	// Special case: "by-id" delegates to getSemanticBundle for LLM-friendly response
	if domain == "by-id" {
		s.getSemanticBundle(w, r)
		return
	}

	// Return a stable bundle payload even when registry tables are empty/missing.
	// This endpoint is used to populate optional UI bundle libraries and should not
	// hard-fail (404/500) in local/dev environments.
	type bundleResponse struct {
		BundleID  string          `json:"bundle_id"`
		Domain    string          `json:"domain"`
		Audience  []string        `json:"audience"`
		Version   string          `json:"version"`
		Owner     string          `json:"owner"`
		Tags      []string        `json:"tags"`
		Functions json.RawMessage `json:"functions"`
		Metrics   json.RawMessage `json:"metrics"`
	}

	bundle := bundleResponse{
		BundleID:  domain,
		Domain:    domain,
		Audience:  []string{},
		Version:   "v1.0.0",
		Owner:     "patrick",
		Tags:      []string{},
		Functions: json.RawMessage("[]"),
		Metrics:   json.RawMessage("[]"),
	}

	if s.DB == nil {
		respond(w, r, bundle, nil)
		return
	}

	// Fetch functions for the domain.
	{
		query := `
			SELECT COALESCE(
				json_agg(
					json_build_object(
						'name', f.name,
						'class', f.class,
						'badge', f.badge,
						'description', f.description
					)
				) FILTER (WHERE f.name IS NOT NULL),
				'[]'::json
			)
			FROM public.dax_functions f
			WHERE f.schema_domain = $1
		`

		var raw []byte
		if err := s.DB.QueryRowContext(r.Context(), query, domain).Scan(&raw); err != nil {
			// If the registry table doesn't exist or is empty, keep an empty array.
			logging.GetLogger().Sugar().Warnf("semantic bundle functions lookup failed for domain=%s: %v", domain, err)
		} else if len(raw) > 0 {
			bundle.Functions = json.RawMessage(raw)
		}
	}

	// Fetch metrics for the domain.
	{
		query := `
			SELECT COALESCE(
				json_agg(
					json_build_object(
						'node_id', m.node_id,
						'category', m.category,
						'description', m.description,
						'financial_calc', json_build_object(
							'type', m.formula_type,
							'formula', m.formula,
							'arguments', m.arguments
						),
						'badge', m.badge,
						'function_class', m.function_class,
						'functions_used', m.functions_used,
						'governance', json_build_object('status', m.governance_status)
					)
				) FILTER (WHERE m.node_id IS NOT NULL),
				'[]'::json
			)
			FROM public.metrics_registry m
			WHERE m.schema_domain = $1
		`

		var raw []byte
		if err := s.DB.QueryRowContext(r.Context(), query, domain).Scan(&raw); err != nil {
			logging.GetLogger().Sugar().Warnf("semantic bundle metrics lookup failed for domain=%s: %v", domain, err)
		} else if len(raw) > 0 {
			bundle.Metrics = json.RawMessage(raw)
		}
	}

	respond(w, r, bundle, nil)
}

// getSemanticObjects retrieves available semantic objects (measures and dimensions) for a tenant and datasource
// NOTE: previously there were helper handlers for semantic object and
// catalog node listing here. They were unused in the current API router
// layout and have been removed to reduce dead code (staticcheck U1000).

type dynamicCRUDRequest struct {
	EntityType string                 `json:"entity_type"`
	Data       map[string]interface{} `json:"data"`
	IDs        []string               `json:"ids"`
}

func buildDynamicCRUDMutation(entityType string, data map[string]interface{}, ids []string, method string) (string, map[string]interface{}, error) {
	vars := make(map[string]interface{})
	switch method {
	case http.MethodPost:
		if entityType == "" {
			return "", nil, fmt.Errorf("entity_type is required for insert")
		}
		vars["object"] = data
		query := fmt.Sprintf(`mutation InsertEntity($object: %s_insert_input!) { insert_%s_one(object: $object) { id } }`, entityType, entityType)
		return query, vars, nil
	case http.MethodPut, http.MethodPatch:
		if entityType == "" {
			return "", nil, fmt.Errorf("entity_type is required for update")
		}
		if data == nil || data["id"] == nil {
			return "", nil, fmt.Errorf("missing id for update")
		}
		vars["id"] = data["id"]
		vars["changes"] = data
		query := fmt.Sprintf(`mutation UpdateEntity($id: uuid!, $changes: %s_set_input!) { update_%s_by_pk(pk_columns: {id: $id}, _set: $changes) { id } }`, entityType, entityType)
		return query, vars, nil
	case http.MethodDelete:
		if entityType == "" {
			return "", nil, fmt.Errorf("entity_type is required for delete")
		}
		if len(ids) == 0 {
			return "", nil, fmt.Errorf("ids are required for delete")
		}
		vars["ids"] = ids
		query := fmt.Sprintf(`mutation DeleteEntities($ids: [uuid!]!) { delete_%s(where: {id: {_in: $ids}}) { affected_rows } }`, entityType)
		return query, vars, nil
	default:
		return "", nil, fmt.Errorf("unsupported method: %s", method)
	}
}

func (s *Server) handleDynamicCrud(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req dynamicCRUDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Data == nil {
		req.Data = make(map[string]interface{})
	}

	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		req.Data["tenant_id"] = claims.TenantID
	}
	if datasourceID := r.Header.Get("X-Tenant-Datasource-ID"); datasourceID != "" {
		req.Data["tenant_datasource_id"] = datasourceID
	}

	query, vars, err := buildDynamicCRUDMutation(req.EntityType, req.Data, req.IDs, r.Method)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := executeGraphQLQuery(vars, query)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("dynamic CRUD failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, r, result, nil)
}

// executeGraphQLQuery sends a query to the Hasura GraphQL endpoint.
func executeGraphQLQuery(variables map[string]interface{}, query string) (map[string]interface{}, error) {
	hasuraURL := os.Getenv("HASURA_URL")
	if hasuraURL == "" {
		return nil, fmt.Errorf("HASURA_URL environment variable not set")
	}
	// Ensure URL has the correct path
	if !strings.HasSuffix(hasuraURL, "/v1/graphql") {
		hasuraURL = strings.TrimSuffix(hasuraURL, "/") + "/v1/graphql"
	}

	hasuraSecret := os.Getenv("HASURA_ADMIN_SECRET")

	reqBody, err := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal graphQL request body: %w", err)
	}

	req, err := http.NewRequest("POST", hasuraURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create graphQL request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if hasuraSecret != "" {
		req.Header.Set("X-Hasura-Admin-Secret", hasuraSecret)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute graphQL request: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logging.GetLogger().Sugar().Errorf("Failed to decode GraphQL response. Status: %s, Body: %s", resp.Status, string(bodyBytes))
		return nil, fmt.Errorf("failed to decode graphQL response: %w", err)
	}

	if errs, ok := result["errors"]; ok {
		return nil, fmt.Errorf("graphql query failed: %v", errs)
	}

	return result, nil
}

// Profiler methods
func (s *Server) startProfile(w http.ResponseWriter, r *http.Request) {
	var req ProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set tenant and datasource from headers
	req.TenantID = jwtmiddleware.GetClaimsFromContext(r).TenantID
	req.DatasourceID = r.Header.Get("X-Tenant-Datasource-ID")

	// If node_ids are provided, resolve them to schema/tables
	if len(req.NodeIDs) > 0 {
		schemaTables, err := s.resolveNodeIDsToSchemaTables(r.Context(), req.TenantID, req.DatasourceID, req.NodeIDs)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to resolve node IDs: %v", err), http.StatusBadRequest)
			return
		}
		// Use the resolved schema/tables. If the selection spans multiple schemas
		// we store qualified table names (schema.table) so the runner can group
		// them correctly and avoid profiling every schema in the system.
		if len(schemaTables) > 0 {
			// If all resolved entries belong to the same schema, keep the
			// existing behavior (set req.Schema and simple table list) for
			// backward compatibility. Otherwise, store qualified names.
			sameSchema := true
			firstSchema := schemaTables[0].Schema
			for _, st := range schemaTables {
				if st.Schema != firstSchema {
					sameSchema = false
					break
				}
			}
			if sameSchema {
				req.Schema = firstSchema
				req.Tables = make([]string, len(schemaTables))
				for i, st := range schemaTables {
					req.Tables[i] = st.Table
				}
			} else {
				// store qualified table identifiers so runProfile can group by
				// schema and call the profiler per-schema with the correct
				// table list.
				req.Schema = ""
				req.Tables = make([]string, len(schemaTables))
				for i, st := range schemaTables {
					req.Tables[i] = fmt.Sprintf("%s.%s", st.Schema, st.Table)
				}
			}
		}
	}

	// Look up datasource DSN
	var connectionString string
	err := s.DB.QueryRow("SELECT connection_string FROM public.tenant_datasources WHERE tenant_id = $1 AND datasource_id = $2", req.TenantID, req.DatasourceID).Scan(&connectionString)
	if err != nil {
		// For development, use alpha database as the source database
		req.DataSource = getEnv("ALPHA_DB_URL", "postgres://postgres:postgres@postgres:5432/alpha?sslmode=disable")
	} else {
		req.DataSource = connectionString
	}

	if err := s.Validate.Struct(req); err != nil {
		http.Error(w, fmt.Sprintf("Validation failed: %v", err), http.StatusBadRequest)
		return
	}

	jobID := generateJobID()
	job := &ProfileJob{
		ID:        jobID,
		Status:    "pending",
		CreatedAt: time.Now(),
		Req:       req,
	}

	s.ProfileJobs.Store(jobID, job)

	// Start profiling in background
	// Log job creation and important metadata (batch size)
	logging.GetLogger().Sugar().Infow("profiler queued job", "job_id", jobID, "batch_size", req.BatchSize, "tenant", req.TenantID, "datasource", req.DatasourceID, "node_ids", len(req.NodeIDs))
	go s.runProfile(jobID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jobId": jobID,
	})
}

func (s *Server) getProfileStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")

	jobInterface, exists := s.ProfileJobs.Load(jobID)
	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	job := jobInterface.(*ProfileJob)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": job.Status,
		"error":  job.Error,
	})
}

func (s *Server) getProfileResults(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	// allow fallback to query params for non-shimmed requests (helpful in dev)
	if tenantID == "" || datasourceID == "" {
		if qtid := r.URL.Query().Get("tenant_id"); qtid != "" {
			tenantID = qtid
		}
		if qdid := r.URL.Query().Get("datasource_id"); qdid != "" {
			datasourceID = qdid
		}
	}
	if tenantID == "" || datasourceID == "" {
		http.Error(w, "tenant and datasource headers required", http.StatusBadRequest)
		return
	}

	// Get jobId from query params and filter results by the job's tables
	jobID := r.URL.Query().Get("jobId")
	var jobTables []string
	if jobID != "" {
		jobInterface, exists := s.ProfileJobs.Load(jobID)
		if exists {
			job := jobInterface.(*ProfileJob)
			jobTables = job.Req.Tables
		}
	}

	// Build query that joins column_profiles with catalog_node to get schema/table info
	baseQuery := `
		SELECT 
			cn.qualified_path,
			cn.node_name as column_name,
			cp.data_type, 
			cp.cardinality, 
			cp.min_length, 
			cp.max_length, 
			cp.avg_length,
			COALESCE(cp.properties->>'frequent_values', '[]') as frequent_values,
			COALESCE(cp.properties->>'inferred_patterns', '[]') as inferred_patterns
		FROM sml.column_profiles cp
		JOIN public.catalog_node cn ON cp.id = cn.id
		WHERE cn.tenant_id = $1 AND cn.tenant_datasource_id = $2`
	args := []interface{}{tenantID, datasourceID}
	paramIdx := 3

	// Filter by job's tables if jobId was provided and job exists
	if len(jobTables) > 0 {
		tableFilters := make([]string, len(jobTables))
		for i, tableName := range jobTables {
			tableFilters[i] = fmt.Sprintf("cn.qualified_path LIKE $%d", paramIdx)
			// Handle both qualified (schema.table) and unqualified table names
			if strings.Contains(tableName, ".") {
				parts := strings.SplitN(tableName, ".", 2)
				args = append(args, "/"+parts[0]+"/"+parts[1]+"/%")
			} else {
				args = append(args, "%/"+tableName+"/%")
			}
			paramIdx++
		}
		baseQuery += fmt.Sprintf(" AND (%s)", strings.Join(tableFilters, " OR "))
	}

	// Validate optional schema/table query params and filter by qualified_path
	identRe := regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	if qSchema := r.URL.Query().Get("schema"); qSchema != "" {
		if len(qSchema) > 63 || !identRe.MatchString(qSchema) {
			http.Error(w, "invalid schema parameter", http.StatusBadRequest)
			return
		}
		baseQuery += fmt.Sprintf(" AND cn.qualified_path LIKE $%d", paramIdx)
		args = append(args, "/"+qSchema+"/%")
		paramIdx++
	}
	if qTable := r.URL.Query().Get("table"); qTable != "" {
		if len(qTable) > 63 || !identRe.MatchString(qTable) {
			http.Error(w, "invalid table parameter", http.StatusBadRequest)
			return
		}
		baseQuery += fmt.Sprintf(" AND cn.qualified_path LIKE $%d", paramIdx)
		args = append(args, "%/"+qTable+"/%")
		paramIdx++
	}
	// Paging: allow optional ?limit=<n>&offset=<n>
	// Default limit 100, cap to 500 to avoid very large responses.
	limit := 100
	offset := 0
	if lstr := r.URL.Query().Get("limit"); lstr != "" {
		if li, err := strconv.Atoi(lstr); err == nil {
			limit = li
		} else {
			http.Error(w, "invalid limit parameter", http.StatusBadRequest)
			return
		}
	}
	if ostr := r.URL.Query().Get("offset"); ostr != "" {
		if oi, err := strconv.Atoi(ostr); err == nil {
			offset = oi
		} else {
			http.Error(w, "invalid offset parameter", http.StatusBadRequest)
			return
		}
	}
	if limit < 1 || limit > 500 {
		http.Error(w, "limit out of range", http.StatusBadRequest)
		return
	}
	if offset < 0 {
		http.Error(w, "offset out of range", http.StatusBadRequest)
		return
	}

	// Attach paging as positional parameters
	baseQuery += fmt.Sprintf(" ORDER BY cn.created_at DESC LIMIT $%d OFFSET $%d", paramIdx, paramIdx+1)
	args = append(args, limit, offset)

	// Query DB for profiles for this tenant/datasource (and optional filters)
	rows, err := s.DB.Query(baseQuery, args...)
	if err != nil {
		// Log detailed information to aid debugging (dev-only helpful output)
		logging.GetLogger().Sugar().Errorw("failed to query results", "error", err, "query", baseQuery, "args", args)
		// Return a more informative error during local development to help trace the root cause.
		http.Error(w, fmt.Sprintf("failed to query results: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var profiles []map[string]interface{}
	for rows.Next() {
		var qualifiedPath, col, dtype string
		var cardinality int64
		var minLen, maxLen sql.NullInt64
		var avgLen sql.NullFloat64
		var frequent sql.NullString
		var inferred sql.NullString
		if err := rows.Scan(&qualifiedPath, &col, &dtype, &cardinality, &minLen, &maxLen, &avgLen, &frequent, &inferred); err != nil {
			continue
		}

		// Parse qualified_path to extract schema and table
		// Format is typically "/schema/table/column" for columns
		var schema, table string
		if strings.HasPrefix(qualifiedPath, "/") {
			parts := strings.Split(strings.TrimPrefix(qualifiedPath, "/"), "/")
			if len(parts) >= 2 {
				schema = parts[0]
				table = parts[1]
			}
		}
		var freqVals []string
		if frequent.Valid && frequent.String != "" {
			s := strings.TrimSpace(frequent.String)
			// If it looks like JSON array, try decoding
			if strings.HasPrefix(s, "[") {
				var js []string
				if err := json.Unmarshal([]byte(s), &js); err == nil {
					freqVals = js
				} else {
					// fallback to simple parsing
					s = strings.TrimPrefix(s, "[")
					s = strings.TrimSuffix(s, "]")
					if s != "" {
						for _, part := range strings.Split(s, ",") {
							p := strings.Trim(part, "\" ")
							if p != "" {
								freqVals = append(freqVals, p)
							}
						}
					}
				}
			} else {
				// Postgres array format like {a,b}
				s = strings.TrimPrefix(s, "{")
				s = strings.TrimSuffix(s, "}")
				if s != "" {
					for _, part := range strings.Split(s, ",") {
						p := strings.Trim(part, "\" ")
						if p != "" {
							freqVals = append(freqVals, p)
						}
					}
				}
			}
		}
		var inferredVals []string
		if inferred.Valid && inferred.String != "" {
			t := strings.TrimSpace(inferred.String)
			if strings.HasPrefix(t, "[") {
				var js []string
				if err := json.Unmarshal([]byte(t), &js); err == nil {
					inferredVals = js
				} else {
					t = strings.TrimPrefix(t, "[")
					t = strings.TrimSuffix(t, "]")
					if t != "" {
						for _, part := range strings.Split(t, ",") {
							p := strings.Trim(part, "\" ")
							if p != "" {
								inferredVals = append(inferredVals, p)
							}
						}
					}
				}
			} else {
				t = strings.TrimPrefix(t, "{")
				t = strings.TrimSuffix(t, "}")
				if t != "" {
					for _, part := range strings.Split(t, ",") {
						p := strings.Trim(part, "\" ")
						if p != "" {
							inferredVals = append(inferredVals, p)
						}
					}
				}
			}
		}

		profiles = append(profiles, map[string]interface{}{
			"Schema":           schema,
			"TableName":        table,
			"ColumnName":       col,
			"DataType":         dtype,
			"Cardinality":      cardinality,
			"MinLength":        nilIfNullInt64(minLen),
			"MaxLength":        nilIfNullInt64(maxLen),
			"AvgLength":        nilIfNullFloat64(avgLen),
			"FrequentValues":   freqVals,
			"InferredPatterns": inferredVals,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"profiles": profiles, "status": "completed"})
}

func (s *Server) runProfile(jobID string) {
	logging.GetLogger().Sugar().Infow("DEBUG: runProfile started", "jobID", jobID)
	defer func() {
		if r := recover(); r != nil {
			logging.GetLogger().Sugar().Errorw("panic in runProfile", "jobID", jobID, "panic", r)
		}
	}()
	jobInterface, exists := s.ProfileJobs.Load(jobID)
	if !exists {
		return
	}

	job := jobInterface.(*ProfileJob)
	job.mu.Lock()
	job.Status = "running"
	job.mu.Unlock()

	logging.GetLogger().Sugar().Infow("profiler running job", "job_id", jobID, "batch_size", job.Req.BatchSize, "tables", len(job.Req.Tables))

	logging.GetLogger().Sugar().Infow("DEBUG: profiler running job", "schema", job.Req.Schema)

	progress := func(current, total int, message string) {
		prog := map[string]interface{}{"type": "progress", "current": current, "total": total, "message": message, "results": nil}
		if s.WsHub != nil {
			if b, err := json.Marshal(prog); err == nil {
				s.WsHub.broadcastToAudience(jobID, b)
			}
		}
	}

	alphaPool := (*pgxpool.Pool)(nil)
	if getEnv("SEMLAYER_TEST_SKIP_ALPHA_POOL", "") != "1" {
		alphaURL := getEnv("ALPHA_DB_URL", "postgres://postgres:postgres@postgres:5432/alpha?sslmode=disable")
		var err error
		alphaPool, err = pgxpool.New(context.Background(), alphaURL)
		if err != nil {
			job.mu.Lock()
			job.Status = "failed"
			job.Error = fmt.Sprintf("failed to open alpha pool: %v", err)
			job.mu.Unlock()
			return
		}
		logging.GetLogger().Sugar().Infow("alpha pool created", "url", alphaURL)
		defer alphaPool.Close()
	}

	// The profiler request may specify either a single schema (job.Req.Schema)
	// or a list of tables that may be qualified (schema.table). Build a map
	// of schema -> []tables so we can call the profiler once per schema with
	// the correct table list. If no schema/table is provided, default to
	// profiling 'public' (legacy behavior).
	schemaTableMap := make(map[string][]string)
	if job.Req.Schema != "" {
		// simple case: explicit schema provided; tables (if any) are unqualified
		if len(job.Req.Tables) == 0 {
			// profile all tables in the schema? Legacy code used ['public'] when
			// no tables were listed. We'll pass an empty table slice and the
			// profiler will iterate over provided tables as before.
			schemaTableMap[job.Req.Schema] = []string{}
		} else {
			schemaTableMap[job.Req.Schema] = job.Req.Tables
		}
	} else if len(job.Req.Tables) > 0 {
		// Tables may be qualified as 'schema.table' or simply 'table'. Parse
		// both and group by schema; unqualified tables default to 'public'.
		for _, t := range job.Req.Tables {
			if strings.Contains(t, ".") {
				parts := strings.SplitN(t, ".", 2)
				sch := parts[0]
				tbl := parts[1]
				schemaTableMap[sch] = append(schemaTableMap[sch], tbl)
			} else {
				schemaTableMap["public"] = append(schemaTableMap["public"], t)
			}
		}
	} else {
		// legacy default
		schemaTableMap["public"] = []string{}
	}

	logging.GetLogger().Sugar().Infow("schemas to profile (grouped)", "schemas", schemaTableMap)
	var allErrors []string
	for schema, tables := range schemaTableMap {
		logging.GetLogger().Sugar().Infow("starting profiler", "schema", schema, "tables", tables, "datasource", job.Req.DataSource)
		if err := profiler.ProfileTablesFunc(context.Background(), logging.GetLogger(), alphaPool, job.Req.TenantID, job.Req.DatasourceID, job.Req.DataSource, schema, tables, job.Req.SampleSize, job.Req.FPRate, job.Req.BatchSize, progress); err != nil {
			allErrors = append(allErrors, err.Error())
		}
	}

	if len(allErrors) > 0 {
		job.mu.Lock()
		job.Status = "failed"
		job.Error = strings.Join(allErrors, "; ")
		job.Results = nil
		job.mu.Unlock()
		if s.WsHub != nil {
			if b, err2 := json.Marshal(map[string]interface{}{"type": "failed", "current": 0, "total": 0, "message": job.Error, "results": nil}); err2 == nil {
				s.WsHub.broadcastToAudience(jobID, b)
			}
		}
		return
	}

	job.mu.Lock()
	job.Status = "completed"
	if job.Results == nil {
		job.Results = []interface{}{}
	}
	results := job.Results
	job.mu.Unlock()

	completedMsg := map[string]interface{}{"type": "completed", "current": 100, "total": 100, "message": "Profiling completed", "results": results}
	if s.WsHub != nil {
		if b, err := json.Marshal(completedMsg); err == nil {
			s.WsHub.broadcastToAudience(jobID, b)
		}
	}
}

// Event represents a field change on a business object.

type Event struct {
	BOType   string      `json:"bo_type"`
	BOID     string      `json:"bo_id"`
	Field    string      `json:"field_name"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
}

// registerAIRoutes mounts AI-related endpoints
func (s *Server) registerAIRoutes(r chi.Router) {
	// AI Logic-to-Config (Natural Language to DAG)
	r.Post("/ai/generate-dag", s.GenerateDAGFromNL)

	// AI Migration Engine (Legacy Code to Titan Config)
	r.Route("/migrations", func(r chi.Router) {
		r.Get("/", s.ListMigrations)
		r.Post("/", s.CreateMigration)
		r.Get("/{id}", s.GetMigration)
		r.Post("/{id}/approve", s.ApproveMigration)
		r.Post("/{id}/reject", s.RejectMigration)
	})

	r.Post("/calc/vectorized", s.runVectorizedCalculations)
}

// registerExplorerRoutes mounts query, search, and saved query endpoints
func (s *Server) registerExplorerRoutes(r chi.Router) {
	r.Route("/explorer", func(r chi.Router) {
		r.Post("/query/execute", s.QueryHandler.HandleExecuteQuery)
		r.Post("/query/compile", s.QueryHandler.HandleCompileQuery)
		r.Post("/query/export", s.QueryHandler.HandleExportQuery)
		r.Get("/query/history", s.QueryHandler.HandleListHistory)

		r.Post("/search", s.SearchHandler.HandleSemanticSearch)
		r.Get("/search/suggestions", s.SearchHandler.HandleGetSuggestions)

		r.Route("/saved-queries", func(r chi.Router) {
			r.Get("/", s.SavedQueryHandler.HandleListSavedQueries)
			r.Post("/", s.SavedQueryHandler.HandleCreateSavedQuery)
			r.Get("/duplicates", s.SavedQueryHandler.HandleGetDuplicates)
			r.Get("/{id}", s.SavedQueryHandler.HandleGetSavedQuery)
			r.Put("/{id}", s.SavedQueryHandler.HandleUpdateSavedQuery)
			r.Delete("/{id}", s.SavedQueryHandler.HandleDeleteSavedQuery)
			r.Post("/{id}/clone", s.SavedQueryHandler.HandleCloneSavedQuery)
			r.Post("/{id}/share", s.SavedQueryHandler.HandleShareQuery)
			r.Get("/{id}/preview", s.SavedQueryHandler.HandleGetPreview)
			r.Get("/{id}/diff", s.SavedQueryHandler.HandleGetDiff)
		})
	})

	// Legacy compatibility aliases for explorer
	r.Route("/query", func(r chi.Router) {
		r.Post("/execute", s.QueryHandler.HandleExecuteQuery)
		r.Post("/compile", s.QueryHandler.HandleCompileQuery)
		r.Post("/export", s.QueryHandler.HandleExportQuery)
		r.Get("/history", s.QueryHandler.HandleListHistory)
	})
	r.Post("/search", s.SearchHandler.HandleSemanticSearch)
	r.Post("/nlq", s.NLQHandler.HandleAsk)
	r.Route("/saved", func(r chi.Router) {
		r.Get("/", s.SavedQueryHandler.HandleListSavedQueries)
		r.Post("/", s.SavedQueryHandler.HandleCreateSavedQuery)
		r.Get("/{id}", s.SavedQueryHandler.HandleGetSavedQuery)
		r.Put("/{id}", s.SavedQueryHandler.HandleUpdateSavedQuery)
		r.Delete("/{id}", s.SavedQueryHandler.HandleDeleteSavedQuery)
	})

	// Natural Language Q&A endpoint
	r.Route("/nlq", func(r chi.Router) {
		r.Post("/ask", s.NLQHandler.HandleAsk)
		r.Post("/search", s.NLQHandler.HandleSearch)
		r.Post("/feedback", s.handleNLQFeedback)
	})

	// Private Markets Explorer endpoints
	r.Get("/user/{id}", s.getUser)
	r.Get("/bundles", s.listBundles)
	r.Get("/funds", s.listFunds)
	r.Get("/metrics/{fundId}", s.getFundMetrics)
}

// registerAdminRoutes mounts administrative and configuration endpoints
func (s *Server) registerAdminRoutes(r chi.Router) {
	// Admin LLM Configuration
	r.Route("/admin/llm", func(r chi.Router) {
		r.Get("/config", s.handleGetLLMConfig)
		r.Put("/config", s.handlePutLLMConfig)
		r.Post("/test", s.handlePostLLMTest)
	})

	// Admin API key issuance
	s.AdminAPIKeyHandler.RegisterRoutes(r)

	// Admin Eval
	r.Route("/admin/eval", func(r chi.Router) {
		r.Post("/run", s.handleRunEval)
	})

	// Admin Cube Sync
	r.Route("/admin/cube", func(r chi.Router) {
		r.Post("/sync", s.handleCubeSync)
	})

	// Role Management
	s.RegisterRoleRoutes(r)

	// User & Audit (Admin only)
	r.Get("/users", s.listUsers)
	r.Get("/audit/events", s.listIAMEvents)
	r.Get("/audit/stats", s.getSecurityStats)
}

// registerLineageRoutes mounts lineage and relationship graph endpoints
func (s *Server) registerLineageRoutes(r chi.Router) {
	r.Route("/relationships", func(r chi.Router) {
		RegisterRelationshipRoutes(r, s.RelationshipHandler)
		s.RegisterRelationshipRoutes(r)
	})

	r.Route("/lineage", func(r chi.Router) {
		r.Get("/node/{id}/graph", s.LineageHandler.GetDependencyGraph)
		r.Get("/node/{id}/impact", s.LineageHandler.GetImpactAnalysis)
		r.Get("/dual", s.LineageHandler.GetDualLineage)
	})
}

// registerTemplateRoutes mounts template and dynamic CRUD endpoints
func (s *Server) registerTemplateRoutes(r chi.Router) {
	r.Get("/templates", s.listTemplates)
	r.Post("/templates", s.saveTemplate)
	r.Get("/templates/{node_id}", s.getTemplate)
	r.Post("/templates/{node_id}/promote", s.promoteTemplate)
	r.Get("/templates/{node_id}/versions", s.listVersions)

	r.Post("/crud", s.handleDynamicCrud)
	r.Put("/crud", s.handleDynamicCrud)
	r.Delete("/crud", s.handleDynamicCrud)
}

// registerCalculationRoutes mounts calculation and cube-related routes
func (s *Server) registerCalculationRoutes(r chi.Router) {
	r.Route("/calc", func(r chi.Router) {
		r.Post("/", s.CalcHandler.Create)
		r.Post("/preview", s.CalcHandler.Preview)
		r.Post("/vectorized", s.runVectorizedCalculations)
	})
	s.CubeHandler.RegisterRoutes(r)
	r.Mount("/calculations", s.CalculationHandler.Routes())
	r.Mount("/execution-logs", s.ExecutionMonitorHandler.Routes())
}

// registerAuditTrinoRoutes mounts the Trino-backed audit and snapshot routes
func (s *Server) registerAuditTrinoRoutes(r chi.Router) {
	auditTrinoHost := getEnv("AUDIT_TRINO_HOST", "localhost")
	auditTrinoPortStr := getEnv("AUDIT_TRINO_PORT", "8090")
	if auditTrinoPort, err := strconv.Atoi(auditTrinoPortStr); err == nil {
		auditTrinoQuerier, err := audit.NewTrinoAuditQuerier(auditTrinoHost, auditTrinoPort, "iceberg", "audit")
		if err != nil {
			logging.GetLogger().Sugar().Warnf("Failed to initialize audit Trino querier: %v", err)
		} else {
			ginRouter := gin.New()
			ginRouter.Use(gin.Recovery())

			auditHandler := audit.NewAuditAPIHandler(auditTrinoQuerier)
			auditGroup := ginRouter.Group("")
			auditHandler.RegisterRoutes(auditGroup)

			r.Mount("/audit-trino", ginRouter)
			logging.GetLogger().Sugar().Info("Audit Trino & Snapshot Plane routes registered at /api/audit-trino/*")
		}
	}
}

// registerSemanticRoutes mounts semantic model management endpoints
func (s *Server) registerSemanticRoutes(r chi.Router) {
	// Dedicated route for LLM-friendly semantic bundle by business object ID
	r.Get("/semantic/bundles/by-id", s.getSemanticBundle)

	// Metadata versioning endpoints
	r.Post("/metadata/versions", s.createMetadataVersion)
	r.Get("/metadata/versions/{bo_id}", s.getMetadataVersionHistory)

	// Field alias endpoints
	r.Post("/field-aliases", s.createFieldAlias)
	r.Get("/field-aliases/{field_id}", s.getFieldAliases)

	// Semantic name resolver stats
	r.Get("/semantic/name-resolver/stats", s.getSemanticNameResolverStats)

	// Business Term Suggestions
	r.Post("/business-term/suggestions", s.generateBusinessTermSuggestions)
}

// registerLLMGatewayRoutes mounts natural language query gateway endpoints
func (s *Server) registerLLMGatewayRoutes(r chi.Router) {
	// Main gateway: NL -> Planner -> Semantic Query -> Executor -> SQL -> DB
	r.Post("/llm/query", s.handleSemanticQuery)

	// Debug endpoints for pipeline stages
	r.Post("/llm/planner", s.handlePlannerOnly)
	r.Post("/llm/executor", s.handleExecutorOnly)

	// Diagnostic endpoints
	r.Get("/llm/prompts", s.handleHealthGoldenPrompts)
	r.Get("/llm/modes", s.handleSemanticQueryModeInfo)
}

// registerProcessRoutes mounts process-related dashboard and optimization endpoints
func (s *Server) registerProcessRoutes(r chi.Router, db *sql.DB, sqlxDB *sqlx.DB) {
	// BP Designer (Workday-Plus) Persistence
	bpService := bp.NewDesignerService(db)
	approvalEvents := NewApprovalEventService(db)
	bpHandler := NewBPHandler(bpService, approvalEvents)
	bpHandler.RegisterRoutes(r)

	// Process Analytics Dashboard
	processAnalyticsHandler := NewProcessAnalyticsHandlers(sqlxDB)
	processAnalyticsHandler.RegisterRoutes(r)

	// Process Benchmarking System
	benchmarkingHandler := NewBenchmarkingHandler(db)
	benchmarkingHandler.RegisterRoutes(r)

	// Process Live Monitoring Dashboard
	processMonitorHandler := NewProcessMonitorHandlers(sqlxDB)
	processMonitorHandler.RegisterRoutes(r)

	// AI-Powered Process Optimization
	processOptimizationHandler := NewProcessOptimizationHandlers(sqlxDB)
	processOptimizationHandler.RegisterRoutes(r)

	// Integration Marketplace
	marketplaceIntegrationHandler := NewMarketplaceIntegrationHandlers(sqlxDB)
	marketplaceIntegrationHandler.RegisterRoutes(r)

	// Process Templates Library
	processTemplateHandler := NewProcessTemplateHandlers(sqlxDB)
	processTemplateHandler.RegisterRoutes(r)

	r.Post("/bp/start-execution", StartBPExecution)
}

// registerDebugRoutes mounts debug and internal state inspection endpoints
func (s *Server) registerDebugRoutes(r chi.Router) {
	// Debug endpoint to show raw edges for a node
	r.Get("/debug/edges/{id}", func(w http.ResponseWriter, req *http.Request) {
		nodeID := chi.URLParam(req, "id")
		log.Printf("[DEBUG] Looking for edges involving node: %s", nodeID)

		query := `
				SELECT ce.id, ce.source_node_id, ce.target_node_id, ce.edge_type_id,
				       ns.node_name as source_name, nt.node_name as target_name
				FROM catalog_edge ce
				LEFT JOIN catalog_node ns ON ce.source_node_id = ns.id
				LEFT JOIN catalog_node nt ON ce.target_node_id = nt.id
				WHERE ce.source_node_id = $1 OR ce.target_node_id = $1
			`

		rows, err := s.DB.QueryContext(req.Context(), query, nodeID)
		if err != nil {
			log.Printf("[DEBUG] Error querying edges: %v", err)
			http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type DebugEdge struct {
			ID           string  `json:"id"`
			SourceNodeID string  `json:"source_node_id"`
			TargetNodeID string  `json:"target_node_id"`
			EdgeTypeID   *string `json:"edge_type_id"`
			SourceName   *string `json:"source_name"`
			TargetName   *string `json:"target_name"`
		}

		var edges []DebugEdge
		for rows.Next() {
			var e DebugEdge
			if err := rows.Scan(&e.ID, &e.SourceNodeID, &e.TargetNodeID, &e.EdgeTypeID, &e.SourceName, &e.TargetName); err != nil {
				log.Printf("[DEBUG] Error scanning row: %v", err)
				continue
			}
			edges = append(edges, e)
		}

		log.Printf("[DEBUG] Found %d edges for node %s", len(edges), nodeID)

	})
}

// registerMetadataRoutes mounts semantic layer and model management endpoints
func (s *Server) registerMetadataRoutes(r chi.Router, boHandler *BusinessObjectHandler, catalogHandler *CatalogHandler, boService *catalogmeta.BusinessObjectService) {
	// Semantic layer endpoints
	s.RegisterSemanticLayerRoutes(r)

	// Register Semantic Layer Handler routes
	if s.SemanticLayerHandler != nil {
		s.SemanticLayerHandler.RegisterRoutes(r)
	}

	// Relationship discovery and model regeneration endpoints (Phase 3b)
	r.Post("/relationships/discover", s.postDiscoverRelationships)
	r.Post("/relationships/existing", s.postGetExistingRelationships)
	r.Post("/models/regenerate", s.postTriggerModelRegeneration)
	r.Get("/models/version", s.getModelVersion)

	// Custom Components endpoints
	s.registerCustomComponentRoutes(r)

	// Business Components (Business Objects)
	boHandler.RegisterRoutes(r)
	catalogHandler.RegisterRoutes(r)

	// Entity Schema endpoints (wraps business objects for frontend compatibility)
	entitySchemaHandler := NewEntitySchemaHandler(boService)
	entitySchemaHandler.RegisterRoutes(r)
}

// registerCatalogRoutes mounts catalog scan, connection, and marketplace endpoints
func (s *Server) registerCatalogRoutes(r chi.Router, db *sql.DB, routes *Routes, temporalClient temporalclient.Client) {
	// Node Types endpoints
	RegisterNodeTypesRoutes(r, db)
	RegisterEdgeTypesRoutes(r, db)

	// Marketplace endpoints
	RegisterMarketplaceRoutes(r, db)

	// Catalog Scan endpoints
	routes.RegisterCatalogScan(r, s.CatalogScanHandler)
	routes.RegisterConnection(r, s.TestConnectionHandler)

	// Calc Engine - Metric computation endpoints
	RegisterCalcEngineRoutes(r, db, temporalClient)

	// RDL (Rule Definition Language)
	RegisterRDLRoutes(r, s.SQLXDB)

	// Catalog Chart endpoints
	r.Post("/catalog/{datasourceId}/refresh-charts", s.handleRefreshCharts)
}

// registerWorkflowRoutes mounts workflow, notification, and background job endpoints
func (s *Server) registerWorkflowRoutes(r chi.Router, db *sql.DB, cronJob *cron.Cron) {
	// Workflow Routes (Access, Queues, Jobs)
	registerWorkflowRoutes(r, db, cronJob)

	// Notification Routes (Slack, Email, Jobs)
	registerNotificationRoutes(r, db, cronJob)

	// Advanced Notification Engine
	bpNotificationHandler := NewBPNotificationHandlers(s.SQLXDB)
	bpNotificationHandler.RegisterRoutes(r)

	// Advanced RBAC & Permissions
	rbacHandlers := NewRBACHandlers(s.SQLXDB)
	rbacHandlers.RegisterRoutes(r)
}

// registerTriggerEngineRoutes mounts the Trigger and Automation engine endpoints
func (s *Server) registerTriggerEngineRoutes(r chi.Router, sqlxDB *sqlx.DB) {
	// Wire full TriggerEngine dependencies and register chi-based trigger routes.
	abacEngine := &ABACEngine{db: sqlxDB}

	// Try to wire a real AMQP-backed EventBus for production/dev if configured.
	if s.EventBus == nil {
		s.EventBus = &noopEventBus{}
	}
	notifAdapter := &notificationAdapter{svc: s.NotificationSvc}

	triggerEngine := NewTriggerEngine(sqlxDB, abacEngine, s.EventBus, notifAdapter)
	// Register chi-based trigger routes directly on the chi router
	RegisterTriggerRoutesChi(r, sqlxDB, triggerEngine)
}

// registerNBAEngineRoutes mounts next-best-action and recommendation engine endpoints
func (s *Server) registerNBAEngineRoutes(r chi.Router, sqlxDB *sqlx.DB) {
	// Initialize NBA Handler
	nbaHandler := NewNBAHandler(sqlxDB)

	// NBA Engine Routes
	r.Route("/nba", func(r chi.Router) {
		r.Get("/recommendations", nbaHandler.GetRecommendations)
		r.Post("/execute", nbaHandler.ExecuteAction)
		r.Post("/complete", nbaHandler.CompleteAction)
		r.Post("/dismiss", nbaHandler.DismissAction)
		r.Get("/signals", nbaHandler.GetSignals)
		r.Get("/catalog", nbaHandler.GetActionCatalog)
		r.Get("/stats", nbaHandler.GetOutcomeStats)
		r.Get("/stream", nbaHandler.WebSocketHandler)
	})
}

// registerBillingRoutes mounts platform billing, invoice management, and financial endpoints
func (s *Server) registerBillingRoutes(r chi.Router) {
	// Financial Services (Households, AltInvest, Billing, TaxPlan, Succession)
	registerFinancialRoutes(r, s)

	// Platform Billing & Cost Analytics
	promQuerier := billing.NewHTTPPrometheusQuerier()
	platformBillingSvc := billing.NewPlatformBillingService(promQuerier, billing.DefaultCostModel())
	platformBillingHandlers := NewBillingHandlers(platformBillingSvc)

	r.Route("/platform-billing", func(r chi.Router) {
		r.Get("/tenant/{tenantId}", platformBillingHandlers.GetTenantBilling)
		r.Get("/platform", platformBillingHandlers.GetPlatformBilling)
		r.Get("/anomalies", platformBillingHandlers.GetAnomalies)
		r.Get("/forecast", platformBillingHandlers.GetForecast)
		r.Post("/simulate", platformBillingHandlers.SimulateCost)
	})

	// Invoice Management
	invoiceStore := billing.NewInvoiceStore()
	invoiceSvc := billing.NewInvoiceService(platformBillingSvc, invoiceStore)
	invoiceHandlers := NewInvoiceHandlers(invoiceSvc)
	invoiceHandlers.RegisterRoutes(r)
}

// registerFeedbackRoutes mounts the business term suggestion feedback endpoint
func (s *Server) registerFeedbackRoutes(r chi.Router) {
	r.Post("/business-term/suggestion-feedback", s.handleBusinessTermFeedback)
}

func (s *Server) handleBusinessTermFeedback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var body struct {
		SemanticTermID   string  `json:"semantic_term_id"`
		BusinessTermID   string  `json:"business_term_id,omitempty"`
		BusinessTermName string  `json:"business_term_name"`
		Action           string  `json:"action"` // "accept" or "reject"
		Confidence       float64 `json:"confidence,omitempty"`
		Reason           string  `json:"reason,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload"})
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	if body.SemanticTermID == "" || body.BusinessTermName == "" || body.Action == "" {
		http.Error(w, "semantic_term_id, business_term_name, and action are required", http.StatusBadRequest)
		return
	}

	if body.Action != "accept" && body.Action != "reject" {
		http.Error(w, "action must be either 'accept' or 'reject'", http.StatusBadRequest)
		return
	}

	// Log the feedback for learning
	logging.GetLogger().Sugar().Infow("Business term suggestion feedback",
		"tenant_id", tenantID,
		"datasource_id", tenantDatasourceID,
		"semantic_term_id", body.SemanticTermID,
		"business_term_id", body.BusinessTermID,
		"business_term_name", body.BusinessTermName,
		"action", body.Action,
		"confidence", body.Confidence,
		"reason", body.Reason,
	)

	// Store feedback in database for future ML training
	feedbackID := uuid.New()
	_, err := s.DB.ExecContext(r.Context(), `
				INSERT INTO public.suggestion_feedback (
					id, tenant_id, tenant_datasource_id, semantic_term_id, business_term_id,
					business_term_name, action, confidence, reason, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			`, feedbackID, tenantID, tenantDatasourceID, body.SemanticTermID,
		body.BusinessTermID, body.BusinessTermName, body.Action, body.Confidence, body.Reason, time.Now())

	if err != nil {
		// If table doesn't exist yet, log but don't fail the request
		logging.GetLogger().Sugar().Warnf("Failed to store suggestion feedback (table may not exist): %v", err)
	}

	result := map[string]interface{}{
		"success": true,
		"message": "Feedback recorded successfully",
	}
	json.NewEncoder(w).Encode(result)
}

// registerTemporalWebhookRoute mounts the Hasura event trigger webhook for Temporal
func (s *Server) registerTemporalWebhookRoute(r chi.Router) {
	r.Post("/temporal", s.handleTemporalWebhook)
}

func (s *Server) handleTemporalWebhook(w http.ResponseWriter, r *http.Request) {
	// Generic Hasura webhook adapter: decode the incoming payload and forward to the EventBus if available.
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		logging.GetLogger().Sugar().Warnf("temporal webhook: failed to decode payload: %v", err)
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	// Determine a routing key / event name from common fields if present.
	eventName := "hasura.event"
	if e, ok := payload["event"].(string); ok && e != "" {
		eventName = e
	} else if e, ok := payload["type"].(string); ok && e != "" {
		eventName = e
	}

	// Prefer the "data" field for the event payload; otherwise use the entire payload.
	var dataBytes []byte
	if d, ok := payload["data"]; ok {
		if b, err := json.Marshal(d); err == nil {
			dataBytes = b
		} else {
			dataBytes = []byte("{}")
		}
	} else {
		if b, err := json.Marshal(payload); err == nil {
			dataBytes = b
		} else {
			dataBytes = []byte("{}")
		}
	}

	// Forward to the configured eventBus if present; otherwise accept and log for debugging.
	if s.EventBus != nil {
		_ = s.EventBus.Emit(r.Context(), eventName, json.RawMessage(dataBytes))
		w.WriteHeader(http.StatusAccepted)
		return
	}

	logging.GetLogger().Sugar().Infow("temporal webhook received (no event bus configured)", "event", eventName)
	w.WriteHeader(http.StatusAccepted)
}

// registerAlphaTemporalRoutes mounts various alpha test routes that interact with Temporal
func (s *Server) registerAlphaTemporalRoutes(r chi.Router, temporalClient temporalclient.Client) {
	// UMA Alpha route
	r.Post("/api/uma/{id}/alpha", func(w http.ResponseWriter, r *http.Request) {
		umaID := chi.URLParam(r, "id")
		if umaID == "" {
			http.Error(w, "UMA ID required", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "alpha initiated",
			"uma_id": umaID,
		})
	})

	// Attribution Alpha route
	r.Post("/api/portfolio/{id}/attribute", func(w http.ResponseWriter, r *http.Request) {
		portfolioID := chi.URLParam(r, "id")
		if portfolioID == "" {
			http.Error(w, "Portfolio ID required", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "alpha initiated",
			"portfolio_id": portfolioID,
		})
	})

	// Tax Harvest route
	r.Post("/api/uma/{id}/tax", func(w http.ResponseWriter, r *http.Request) {
		umaID := chi.URLParam(r, "id")
		if umaID == "" {
			http.Error(w, "UMA ID required", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "tax optimization initiated",
			"uma_id": umaID,
		})
	})

	// Direct Indexing Alpha route
	r.Post("/api/index/{id}/alpha", func(w http.ResponseWriter, r *http.Request) {
		indexID := chi.URLParam(r, "id")
		if indexID == "" {
			http.Error(w, "Index ID required", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "alpha optimization initiated",
			"index_id": indexID,
		})
	})
}

// registerEventRoutes mounts market event and data stream ingestion endpoints
func (s *Server) registerEventRoutes(r chi.Router) {
	r.Post("/events/market", s.EventsHandler.IngestMarketEvent)
}

// registerSemanticMappingRoutes mounts semantic mapping and fuzzy matching endpoints
func (s *Server) registerSemanticMappingRoutes(r chi.Router) {
	// Upsert business term and create edge (atomic)
	r.Post("/semantic-mappings/upsert-business-term-edge", s.handleUpsertBusinessTermEdge)

	// Semantic mappings endpoints with fuzzy logic
	r.Get("/semantic-mappings", s.handleGetSemanticMappings)
	r.Post("/semantic-terms/search", s.handleSearchSemanticTerms)
}

func (s *Server) handleUpsertBusinessTermEdge(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var body struct {
		BusinessTermName string `json:"business_term_name"`
		SemanticTermID   string `json:"semantic_term_id"`
		EdgeTypeID       string `json:"edge_type_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid payload"})
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(body.BusinessTermName) == "" || strings.TrimSpace(body.SemanticTermID) == "" {
		http.Error(w, "business_term_name and semantic_term_id are required", http.StatusBadRequest)
		return
	}

	businessID, createdEdge, err := s.SemanticMappingSvc.UpsertBusinessTermAndEdge(r.Context(), tenantID, tenantDatasourceID, body.BusinessTermName, body.SemanticTermID, body.EdgeTypeID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to upsert business term and edge: %v", err)
		http.Error(w, fmt.Sprintf("Failed to upsert business term and edge: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"business_term_id": businessID,
		"edge_created":     createdEdge,
	})
}

func (s *Server) handleGetSemanticMappings(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	mappings, err := s.SemanticMappingSvc.GenerateMappings(r.Context(), tenantID, tenantDatasourceID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	respond(w, r, mappings, nil)
}

func (s *Server) handleSearchSemanticTerms(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || tenantDatasourceID == "" {
		http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
		return
	}

	var body analytics.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	terms, err := s.SemanticMappingSvc.SearchSemanticTerms(r.Context(), body, tenantID, tenantDatasourceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, r, terms, nil)
}

// registerCatalogNodeRoutes mounts low-level catalog node and object discovery endpoints
func (s *Server) registerCatalogNodeRoutes(r chi.Router) {
	r.Get("/semantic/objects", s.handleListSemanticObjects)
	r.Get("/catalog/nodes", s.handleListCatalogNodes)
}

func (s *Server) handleListSemanticObjects(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if s.SemanticSvc == nil {
		respond(w, r, []models.SemanticObjectReference{}, nil)
		return
	}

	objects, err := s.SemanticSvc.ListSemanticObjects(r.Context(), tenantID, tenantDatasourceID)
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	respond(w, r, objects, nil)
}

func (s *Server) handleListCatalogNodes(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tenantDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	// For debugging, allow querying without headers
	if tenantID == "" || tenantDatasourceID == "" {
		// Return all nodes for debugging
		query := `
					SELECT id, node_name, COALESCE(description, ''), tenant_id, tenant_datasource_id, created_at, updated_at, COALESCE(properties, '{}'::jsonb) as properties
					FROM catalog_node
					ORDER BY tenant_id, tenant_datasource_id, node_name
					LIMIT 100
				`

		rows, err := s.DB.QueryContext(r.Context(), query)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		defer rows.Close()

		var nodes []map[string]interface{}
		for rows.Next() {
			var id, nodeName, description, tenantID, datasourceID string
			var createdAt, updatedAt time.Time
			var propsJSON []byte

			err := rows.Scan(&id, &nodeName, &description, &tenantID, &datasourceID, &createdAt, &updatedAt, &propsJSON)
			if err != nil {
				continue
			}

			var props map[string]interface{}
			if err := json.Unmarshal(propsJSON, &props); err != nil {
				props = make(map[string]interface{})
			}

			node := map[string]interface{}{
				"id":                   id,
				"node_id":              id,
				"node_name":            nodeName,
				"qualified_path":       nodeName,
				"catalog_type":         "table",
				"node_type":            "table",
				"tenant_id":            tenantID,
				"tenant_datasource_id": datasourceID,
				"created_at":           createdAt,
				"updated_at":           updatedAt,
				"description":          description,
				"properties":           props,
			}

			nodes = append(nodes, node)
		}

		respond(w, r, nodes, nil)
		return
	}

	// Parse query parameters
	nodeType := r.URL.Query().Get("type")
	q := r.URL.Query().Get("q")
	limitStr := r.URL.Query().Get("limit")

	limit := 50 // default limit
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	// Build query using actual schema (catalog_node with node_name column)
	query := `
				SELECT cn.id, cn.node_name, COALESCE(cn.description, ''), cn.tenant_id, cn.tenant_datasource_id, cn.created_at, cn.updated_at, COALESCE(cn.properties, '{}'::jsonb) as properties
				FROM catalog_node cn
				LEFT JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
				WHERE (cn.tenant_id = $1::uuid OR cn.tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999') AND cn.tenant_datasource_id = $2::uuid
			`
	args := []interface{}{tenantID, tenantDatasourceID}
	argIndex := 3

	if nodeType != "" {
		query += fmt.Sprintf(" AND cnt.catalog_type_name = $%d", argIndex)
		args = append(args, nodeType)
		argIndex++
	}

	if q != "" {
		query += fmt.Sprintf(" AND cn.node_name ILIKE $%d", argIndex)
		args = append(args, "%"+q+"%")
		argIndex++
	}

	query += " ORDER BY cn.node_name LIMIT " + strconv.Itoa(limit)

	rows, err := s.DB.QueryContext(r.Context(), query, args...)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	defer rows.Close()

	nodes := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, nodeName, description, tID, dsID string
		var createdAt, updatedAt time.Time
		var propsJSON []byte

		err := rows.Scan(&id, &nodeName, &description, &tID, &dsID, &createdAt, &updatedAt, &propsJSON)
		if err != nil {
			continue
		}

		var props map[string]interface{}
		if err := json.Unmarshal(propsJSON, &props); err != nil {
			props = make(map[string]interface{})
		}

		node := map[string]interface{}{
			"id":                   id,
			"node_id":              id,
			"node_name":            nodeName,
			"qualified_path":       nodeName,
			"catalog_type":         "table",
			"node_type":            "table",
			"tenant_id":            tID,
			"tenant_datasource_id": dsID,
			"created_at":           createdAt,
			"updated_at":           updatedAt,
			"description":          description,
			"properties":           props,
		}

		nodes = append(nodes, node)
	}

	respond(w, r, nodes, nil)
}

func (s *Server) handleRefreshCharts(w http.ResponseWriter, r *http.Request) {
	datasourceID := chi.URLParam(r, "datasourceId")
	if datasourceID == "" {
		http.Error(w, "datasourceId is required", http.StatusBadRequest)
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(datasourceID); err != nil {
		http.Error(w, "Invalid datasourceId format", http.StatusBadRequest)
		return
	}

	// Use a longer timeout for chart generation (5 minutes)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	// Generate all charts for this datasource
	err := charts.RefreshAllCharts(ctx, s.DB, datasourceID, false)
	if err != nil {
		log.Printf("Failed to refresh charts for datasource %s: %v", datasourceID, err)
		http.Error(w, fmt.Sprintf("Failed to refresh charts: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"message":       "Charts refreshed successfully",
		"datasource_id": datasourceID,
	})
}

// registerTemporalAdminRoutes handles the async initialization of Temporal client and registration of its admin routes
func (s *Server) registerTemporalAdminRoutes(r chi.Router, db *sql.DB) {
	go func() {
		var temporalClient temporalclient.Client
		// Attempt to create a Temporal SDK client using the centralized
		// helper (env-driven + retries). If it fails we'll continue and
		// register routes without it so admin endpoints are still
		// available (they'll return errors but will still log audits).
		tc, err := temporalclientlib.NewClientWithRetry()
		if err != nil {
			logging.GetLogger().Sugar().Warnf("Temporal client not available at startup: %v", err)
		} else {
			temporalClient = tc
		}

		// Build AdminClient gRPC connection options
		// Admin gRPC target: prefer explicit endpoint env var, otherwise
		// fall back to the same service name used in Docker Compose so
		// services running in the same network can reach Temporal.
		target := getEnv("TEMPORAL_GRPC_ENDPOINT", "temporal:7233")
		tlsEnabled := strings.ToLower(getEnv("TEMPORAL_GRPC_TLS", "false")) == "true"
		authToken := getEnv("TEMPORAL_AUTH_TOKEN", "")

		var dialOpts []grpc.DialOption
		if tlsEnabled {
			// Support optional CA bundle and client cert/key for mTLS
			caPath := getEnv("TEMPORAL_GRPC_CA_CERT", "")
			clientCertPath := getEnv("TEMPORAL_GRPC_CLIENT_CERT", "")
			clientKeyPath := getEnv("TEMPORAL_GRPC_CLIENT_KEY", "")

			// Build tls.Config
			tlsConfig := &tls.Config{}

			// Load CA cert if provided
			if caPath != "" {
				caBytes, err := os.ReadFile(caPath)
				if err != nil {
					logging.GetLogger().Sugar().Warnf("failed to read TEMPORAL_GRPC_CA_CERT: %v", err)
					RegisterTemporalAdminRoutes(r, temporalClient, db, s.SecMgr, nil)
					return
				}
				roots := x509.NewCertPool()
				if !roots.AppendCertsFromPEM(caBytes) {
					logging.GetLogger().Sugar().Warnf("failed to parse CA cert PEM")
					RegisterTemporalAdminRoutes(r, temporalClient, db, s.SecMgr, nil)
					return
				}
				tlsConfig.RootCAs = roots
			}

			// Load client cert/key if provided (mTLS)
			if clientCertPath != "" && clientKeyPath != "" {
				cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
				if err != nil {
					logging.GetLogger().Sugar().Warnf("failed to load client cert/key: %v", err)
					RegisterTemporalAdminRoutes(r, temporalClient, db, s.SecMgr, nil)
					return
				}
				tlsConfig.Certificates = []tls.Certificate{cert}
			}

			creds := credentials.NewTLS(tlsConfig)
			dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
		} else {
			dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		}

		// If an auth token is provided, add an interceptor to inject metadata
		if authToken != "" {
			unary := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
				ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+authToken)
				return invoker(ctx, method, req, reply, cc, opts...)
			}
			dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(unary))
		}

		// Create AdminClient
		adminClient, err := temporal.NewAdminClientFromTarget(context.Background(), target, "default", dialOpts...)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("Temporal admin gRPC client not available at startup: %v", err)
			// Register routes without admin client (fallback to placeholder)
			RegisterTemporalAdminRoutes(r, temporalClient, db, s.SecMgr, nil)
			return
		}

		// Register admin routes under /api/temporal with AdminClient
		RegisterTemporalAdminRoutes(r, temporalClient, db, s.SecMgr, adminClient)
		logging.GetLogger().Info("Temporal admin routes registered")

		// Register an active Temporal health endpoint that pings the admin API.
		if adminClient != nil {
			r.Get("/_health/temporal", temporal.HealthHandler(adminClient))
		}
	}()
}

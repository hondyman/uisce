package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/apistudio"
	"github.com/hondyman/uisce/backend/internal/cbo"
	"github.com/hondyman/uisce/backend/internal/semantic"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

// actorFromRequest resolves the acting user's ID from the request's JWT
// claims, falling back to "unknown" (audit trails should never see an
// empty actor string).
func actorFromRequest(r *http.Request) string {
	claims, _ := jwtmiddleware.ValidateTokenFromRequest(r)
	if claims != nil && claims.UserID != "" {
		return claims.UserID
	}
	return "unknown"
}

// APIStudioHandler exposes HTTP routes for backend/internal/apistudio, a
// Repository/Service pair that already existed against the real
// semantic.api_endpoints table but had never been wired to any route —
// see PLAN_STUDIO_EVENTS_AUDIT.md. It reuses that package instead of a
// parallel table, and fires the api_endpoint_save trigger_types event on
// every save through the same TriggerEngine path BOCRUDHandler uses for
// BO row events.
type APIStudioHandler struct {
	svc     *apistudio.Service
	repo    *apistudio.Repository
	trigger *TriggerEngine
}

// noopDesignAI is a placeholder DesignAI until an AI endpoint-proposal
// provider is wired; AIGenerateEndpoint is not exposed as a route here.
type noopDesignAI struct{}

func (noopDesignAI) ProposeEndpoint(ctx context.Context, prompt string, tenantID string) (any, error) {
	return nil, fmt.Errorf("AI endpoint generation is not configured")
}

func NewAPIStudioHandler(db *sqlx.DB, trigger *TriggerEngine) *APIStudioHandler {
	repo := apistudio.NewRepository(db)
	versions := semantic.NewSemanticVersionStore(db)
	svc := apistudio.NewService(repo, versions, noopDesignAI{})
	return &APIStudioHandler{svc: svc, repo: repo, trigger: trigger}
}

func (h *APIStudioHandler) emitEndpointSaveEvent(tenantID uuid.UUID, userID string, ep *apistudio.APIEndpoint) {
	if h.trigger == nil {
		return
	}
	go func() {
		_, err := h.trigger.EvaluateTriggers(context.Background(), &TriggerContext{
			TenantID:     tenantID.String(),
			UserID:       userID,
			TriggerKey:   "api_endpoint_save",
			TargetEntity: "api_definition",
			EntityID:     ep.ID.String(),
			EventData: map[string]interface{}{
				"id": ep.ID.String(), "name": ep.Name, "path": ep.Path, "method": ep.Method, "version": ep.Version,
			},
			RequestedAt: time.Now(),
		})
		if err != nil {
			log.Printf("[WARN] Studio event %q for api_definition/%s failed: %v", "api_endpoint_save", ep.ID, err)
		}
	}()
}

func (h *APIStudioHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api-studio", func(r chi.Router) {
		r.Get("/endpoints", h.HandleListEndpoints)
		r.Post("/endpoints", h.HandleSaveEndpoint)
		r.Get("/endpoints/{id}", h.HandleGetEndpoint)
		r.Post("/endpoints/{id}/deprecate", h.HandleDeprecateEndpoint)
		r.Post("/endpoints/{id}/retire", h.HandleRetireEndpoint)
		r.Get("/openapi", h.HandleGetOpenAPI)
		r.Get("/sdk/{lang}", h.HandleDownloadSDK)
	})
}

// HandleGetOpenAPI returns a schema-complete OpenAPI 3.0 spec for every
// published REST endpoint, suitable for a client generator
// (openapi-generator, oapi-codegen) rather than being informational-only.
func (h *APIStudioHandler) HandleGetOpenAPI(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)
	env := r.URL.Query().Get("env")
	if env == "" {
		env = "default"
	}

	eps, err := h.repo.ListEndpoints(r.Context(), env, tenantID.String())
	if err != nil {
		http.Error(w, fmt.Sprintf("failed listing endpoints: %v", err), http.StatusInternalServerError)
		return
	}

	spec, err := apistudio.GenerateOpenAPI(env, tenantID.String(), eps)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed generating OpenAPI spec: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(spec))
}

// HandleDownloadSDK generates and returns a client SDK for the requested
// language, derived from the same endpoint definitions as HandleGetOpenAPI.
func (h *APIStudioHandler) HandleDownloadSDK(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)
	env := r.URL.Query().Get("env")
	if env == "" {
		env = "default"
	}
	lang := chi.URLParam(r, "lang")

	eps, err := h.repo.ListEndpoints(r.Context(), env, tenantID.String())
	if err != nil {
		http.Error(w, fmt.Sprintf("failed listing endpoints: %v", err), http.StatusInternalServerError)
		return
	}

	var sdk string
	switch lang {
	case "typescript":
		sdk = apistudio.GenerateTypeScriptSDK(env, tenantID.String(), eps)
		w.Header().Set("Content-Disposition", "attachment; filename=api-client.ts")
	case "python":
		sdk = apistudio.GeneratePythonSDK(env, tenantID.String(), eps)
		w.Header().Set("Content-Disposition", "attachment; filename=api_client.py")
	default:
		http.Error(w, fmt.Sprintf("unsupported SDK language %q (supported: typescript, python)", lang), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(sdk))
}

// APIRuntimeMountPrefix is where the dynamic REST execution engine is
// mounted. Endpoint paths configured in API Studio are stored relative to
// this prefix (see apistudio/sdk.go's generated client baseUrl and
// apistudio.APIRuntime.SetMountPrefix).
const APIRuntimeMountPrefix = "/api/runtime"

// ODataMountPrefix is where the read-only OData v4 surface (Excel /
// Power Query) is mounted — see apistudio.ODataHandler.
const ODataMountPrefix = "/odata"

// RegisterAPIRuntimeRoutes mounts the dynamic BO-query execution engines —
// the REST runtime (backend/internal/apistudio.APIRuntime) and the OData
// v4 read surface (apistudio.ODataHandler) — that resolve and run the
// endpoints defined through API Studio. It assembles the CBO planner stack
// (SemanticGraphService + cbo.Planner with its DB-backed repositories)
// since no other part of the live server constructs one yet.
//
// Both engines enforce entitlement policies (semantic.bo_entitlement_policies
// — role gate, row filter, field masking) via cbo.Planner.Plan, plus
// Postgres RLS tenant isolation via apistudio.withTenantScopedQuery.
// Platform-wide RBAC (ABACEngine.Evaluate) remains a stubbed no-op outside
// this BO entitlement path — see trigger_engine.go.
func RegisterAPIRuntimeRoutes(r chi.Router, db *sqlx.DB, redisClient *redis.Client) {
	graphService := analytics.NewSemanticGraphService(db)
	// Initialize resolves the semantic node-type ids (business_object,
	// table, column, ...) that GetNodeByName filters on — without it,
	// every lookup used ids of uuid.Nil (they default to Go's zero value)
	// and would always report "unknown node type", regardless of whether
	// the catalog itself was populated. This was never called anywhere.
	if err := graphService.Initialize(); err != nil {
		log.Printf("[api-studio] semantic graph service Initialize failed: %v", err)
	}
	resolver := analytics.NewBOContextResolver(db, graphService)

	planner := cbo.NewPlanner(
		analytics.NewSemanticRepoAdapter(resolver),
		cbo.NewDBPreAggRepository(db),
		cbo.NewDBEntitlementRepository(db),
		cbo.NewDBTelemetryRepository(db),
		cbo.NewDBSLOProvider(db),
	)
	resolver.SetPlanner(planner)

	repo := apistudio.NewRepository(db)

	rt := apistudio.NewAPIRuntime(repo, resolver, db, redisClient)
	rt.SetMountPrefix(APIRuntimeMountPrefix)
	r.Route(APIRuntimeMountPrefix, func(sr chi.Router) {
		sr.HandleFunc("/*", rt.ServeHTTP)
	})

	odata := apistudio.NewODataHandler(repo, resolver, db, redisClient)
	odata.SetMountPrefix(ODataMountPrefix)
	r.Route(ODataMountPrefix, func(sr chi.Router) {
		sr.HandleFunc("/*", odata.ServeHTTP)
		sr.HandleFunc("/", odata.ServeHTTP)
	})
}

func (h *APIStudioHandler) HandleListEndpoints(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)
	env := r.URL.Query().Get("env")
	if env == "" {
		env = "default"
	}
	endpoints, err := h.repo.ListEndpoints(r.Context(), env, tenantID.String())
	if err != nil {
		http.Error(w, fmt.Sprintf("failed listing endpoints: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, endpoints)
}

func (h *APIStudioHandler) HandleGetEndpoint(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid endpoint id", http.StatusBadRequest)
		return
	}
	ep, err := h.repo.GetEndpoint(r.Context(), id)
	if err != nil {
		http.Error(w, "endpoint not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, ep)
}

func (h *APIStudioHandler) HandleSaveEndpoint(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)
	actor := actorFromRequest(r)

	var ep apistudio.APIEndpoint
	if err := json.NewDecoder(r.Body).Decode(&ep); err != nil {
		http.Error(w, "invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	ep.TenantID = tenantID.String()
	if ep.ID == uuid.Nil {
		ep.ID = uuid.New()
		ep.CreatedAt = time.Now()
		ep.CreatedBy = actor
	}
	if ep.Env == "" {
		ep.Env = "default"
	}
	if ep.Method == "" {
		ep.Method = "GET"
	}
	if ep.Type == "" {
		ep.Type = "rest"
	}
	if ep.Status == "" {
		ep.Status = "active"
	}
	if ep.SemanticVersion == "" {
		ep.SemanticVersion = "v1"
	}
	ep.Version++

	if err := h.svc.SaveEndpoint(r.Context(), &ep, actor); err != nil {
		http.Error(w, fmt.Sprintf("failed saving endpoint: %v", err), http.StatusInternalServerError)
		return
	}

	h.emitEndpointSaveEvent(tenantID, actor, &ep)
	writeJSON(w, http.StatusOK, ep)
}

func (h *APIStudioHandler) HandleDeprecateEndpoint(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	actor := actorFromRequest(r)
	if err := h.svc.DeprecateEndpoint(r.Context(), id, actor); err != nil {
		http.Error(w, fmt.Sprintf("failed deprecating endpoint: %v", err), http.StatusInternalServerError)
		return
	}
	h.HandleGetEndpoint(w, r)
}

func (h *APIStudioHandler) HandleRetireEndpoint(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	actor := actorFromRequest(r)
	if err := h.svc.RetireEndpoint(r.Context(), id, actor); err != nil {
		http.Error(w, fmt.Sprintf("failed retiring endpoint: %v", err), http.StatusInternalServerError)
		return
	}
	h.HandleGetEndpoint(w, r)
}

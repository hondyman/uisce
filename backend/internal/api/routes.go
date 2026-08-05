package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/services"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

// Routes groups route registration methods for centralized wiring and easier testing.
type Routes struct{}

func NewRoutes() *Routes { return &Routes{} }

func (rs *Routes) RegisterBundles(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}

func (rs *Routes) RegisterRoles(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}

func (rs *Routes) RegisterDomains(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}

func (rs *Routes) RegisterAbbreviations(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}

func (rs *Routes) RegisterDAX(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}

func (rs *Routes) RegisterIPWhitelist(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}

func (rs *Routes) RegisterTenantAccess(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}

func (rs *Routes) RegisterTimeoutTriggers(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}

func (rs *Routes) RegisterCatalogScan(r chi.Router, handler interface {
	HandleCatalogScan(http.ResponseWriter, *http.Request)
}) {
	r.Post("/catalog/scan", handler.HandleCatalogScan)
}

func (rs *Routes) RegisterConnection(r chi.Router, handler interface {
	HandleTestConnection(http.ResponseWriter, *http.Request)
}) {
	r.Post("/connections/test", handler.HandleTestConnection)
}

func (rs *Routes) RegisterAudit(r chi.Router, handler interface {
	HandleGetEntityHistory(http.ResponseWriter, *http.Request)
	HandleGetEntityAtTime(http.ResponseWriter, *http.Request)
	HandleRestoreEntity(http.ResponseWriter, *http.Request)
	HandleGetAuditChanges(http.ResponseWriter, *http.Request)
	HandleTenantUpdateEvent(http.ResponseWriter, *http.Request)
}) {
	r.Route("/audit", func(r chi.Router) {
		r.Get("/history/{entityType}/{entityId}", handler.HandleGetEntityHistory)
		r.Get("/history/{entityType}/{entityId}/at/{timestamp}", handler.HandleGetEntityAtTime)
		r.Post("/restore/{entityType}/{entityId}", handler.HandleRestoreEntity)
		r.Get("/changes", handler.HandleGetAuditChanges)
		r.Post("/events/tenant-update", handler.HandleTenantUpdateEvent)
	})
}

// CapacityMiddleware enforces QoS limits
func CapacityMiddleware(qos *services.QoSManager, resource string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
			if tenantID == "" {
				tenantID = "default"
			}
			allowed, err := qos.CheckAccess(tenantID, resource)
			if err != nil {
				http.Error(w, fmt.Sprintf("Capacity Limit Exceeded: %v", err), http.StatusTooManyRequests)
				return
			}
			if !allowed {
				http.Error(w, "Capacity Limit Exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RegisterAnalytics mounts the factor analytics endpoints.
func (rs *Routes) RegisterAnalytics(r chi.Router, handler *AnalyticsHandler, qos *services.QoSManager) {
	r.Route("/analytics", func(r chi.Router) {
		r.Use(CapacityMiddleware(qos, "analytics_requests"))
		r.Get("/factors/exposure/{portfolioID}", handler.GetFactorExposure)
		r.Get("/factors/attribution/{portfolioID}", handler.GetAttribution)
	})
}

// RegisterImpact mounts the impact analysis endpoints.
func (rs *Routes) RegisterImpact(r chi.Router, handler *ImpactHandler) {
	r.Route("/api/impact", func(r chi.Router) {
		r.Get("/graph/{nodeType}/{nodeId}", handler.GetImpactGraph)
		r.Get("/explain/{nodeType}/{nodeId}", handler.GetImpactExplanation)
		r.Post("/query", handler.QueryImpact)
	})
}

// RegisterPolicyRoutes is delegated to the policies_routes implementation which
// needs a CollaborationServiceAPI; this helper keeps the grouping consistent.
func (rs *Routes) RegisterPolicyRoutes(r chi.Router, srv *Server, collabService CollaborationServiceAPI) {
	RegisterPolicyRoutes(r, srv, collabService)
}

// RegisterNotifications mounts notification endpoints using the concrete
// NotificationAPIHandlers type.
func (rs *Routes) RegisterNotifications(r chi.Router, h *NotificationAPIHandlers) {
	r.Route("/notifications", func(r chi.Router) {
		r.Get("/user/{userId}", h.GetUserNotifications)
		r.Post("/", h.CreateNotification)
		r.Post("/{id}/send", h.SendNotification)
		r.Post("/{id}/read", h.MarkNotificationAsRead)
		r.Post("/engagement", h.TrackEngagementEvent)
		r.Get("/preferences/{userId}", h.GetUserPreferences)
		r.Put("/preferences/{userId}", h.UpdateUserPreferences)
		r.Post("/templates", h.CreateNotificationTemplate)
		r.Get("/analytics", h.GetEngagementAnalytics)
		r.Post("/campaigns", h.CreateCampaign)
		r.Post("/campaigns/{id}/launch", h.LaunchCampaign)
		r.Get("/campaigns/{id}", h.GetCampaign)
		r.Get("/campaigns/{id}/analytics", h.GetCampaignAnalytics)
		r.Get("/campaigns/active", h.GetActiveCampaigns)
		r.Post("/campaigns/{id}/pause", h.PauseCampaign)
		r.Post("/campaigns/{id}/resume", h.ResumeCampaign)
		r.Post("/campaigns/{id}/stop", h.StopCampaign)
	})
}

// RegisterDashboards mounts dashboard endpoints using the handler that
// exposes the necessary methods.
type dashboardHandlerIface interface {
	GetUserDashboards(http.ResponseWriter, *http.Request)
	GetDashboard(http.ResponseWriter, *http.Request)
	CreateDashboard(http.ResponseWriter, *http.Request)
	UpdateDashboard(http.ResponseWriter, *http.Request)
	DeleteDashboard(http.ResponseWriter, *http.Request)
	GetPublicDashboards(http.ResponseWriter, *http.Request)
	GetDashboardTemplates(http.ResponseWriter, *http.Request)
	DuplicateDashboard(http.ResponseWriter, *http.Request)
}

func (rs *Routes) RegisterDashboards(r chi.Router, h dashboardHandlerIface) {
	r.Route("/dashboards", func(r chi.Router) {
		r.Get("/user/{userId}", h.GetUserDashboards)
		r.Get("/{id}", h.GetDashboard)
		r.Post("/", h.CreateDashboard)
		r.Put("/{id}", h.UpdateDashboard)
		r.Delete("/{id}", h.DeleteDashboard)
		r.Get("/public", h.GetPublicDashboards)
		r.Get("/templates", h.GetDashboardTemplates)
		r.Post("/{id}/duplicate", h.DuplicateDashboard)
	})
}

// RegisterModelCatalog mounts the model catalog handler.
func (rs *Routes) RegisterModelCatalog(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}

// RegisterMetadata mounts the metadata engine endpoints.
func (rs *Routes) RegisterMetadata(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}

// RegisterAlts mounts the alternative assets endpoints.
func (rs *Routes) RegisterAlts(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}

// RegisterSemanticPlatform mounts the new semantic layer platform endpoints
func (rs *Routes) RegisterSemanticPlatform(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}

// RegisterMetadataWrite mounts the generic write handler
func (rs *Routes) RegisterMetadataWrite(r chi.Router, handler interface {
	HandleGenericWrite(http.ResponseWriter, *http.Request)
}) {
	r.Post("/object/{ObjectType}", handler.HandleGenericWrite)
}

// RegisterMCP mounts the Agentic AI Interface (MCP)
func (rs *Routes) RegisterMCP(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}

// RegisterCalendarSync mounts the calendar sync endpoints
func (rs *Routes) RegisterCalendarSync(r chi.Router, handler interface{ RegisterRoutes(chi.Router) }) {
	handler.RegisterRoutes(r)
}

// RegisterExports mounts the result export endpoints
func (rs *Routes) RegisterExports(r chi.Router, h *handlers.ExportHandlers) {
	r.Route("/v1/exports", func(r chi.Router) {
		r.Get("/{exportId}", h.GetExportStatus)
		r.Get("/{exportId}/download", h.DownloadExport)
		r.Post("/{exportId}/download-url", h.GetDownloadURL)
	})
	r.Route("/v1/jobs/{jobId}/exports", func(r chi.Router) {
		r.Post("/", h.CreateExport)
		r.Get("/", h.ListExports)
	})
}

// RegisterScheduler mounts the job scheduler endpoints
func (rs *Routes) RegisterScheduler(r chi.Router, h *handlers.SchedulerHandlers) {
	r.Route("/v1/schedules", func(r chi.Router) {
		r.Post("/", h.CreateScheduledJob)
		r.Get("/", h.ListSchedules)
		r.Get("/{scheduleId}", h.GetSchedule)
		r.Post("/{scheduleId}/pause", h.PauseSchedule)
		r.Post("/{scheduleId}/resume", h.ResumeSchedule)
		r.Delete("/{scheduleId}", h.DeleteSchedule)
	})
}

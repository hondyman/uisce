package api

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"calendar-service/internal/auth"
	"calendar-service/internal/availability"
	"calendar-service/internal/cache"
	"calendar-service/internal/hasura"
	"calendar-service/internal/metrics"
	"calendar-service/internal/middleware"
	"calendar-service/internal/oauth"
	"calendar-service/internal/repository"
	"calendar-service/internal/services"
	"calendar-service/internal/sync"
)

// Router sets up all API routes
type Router struct {
	router                   *mux.Router
	logger                   *logrus.Entry
	availabilityHandler      *AvailabilityHandler
	blackoutHandler          *BlackoutHandler
	calendarHandler          *CalendarHandler
	tenantHandler            *TenantHandler
	externalSyncHandler      *ExternalSyncHandler
	notificationHandler      *NotificationHandler
	jwtSecret                string
	rateLimiter              *middleware.TenantRateLimiter
	auditService             services.AuditService
	googleOAuth              *oauth.GoogleOAuth2Provider
	syncProcessor            *sync.GoogleSyncProcessor
	msOAuth                  *oauth.MicrosoftOAuth2Provider
	msSyncProcessor          *sync.MicrosoftSyncProcessor
	hasuraClient             *hasura.Client
	settingsHandler          *SettingsHandler
	notificationPrefsHandler *NotificationPreferencesHandler
	exportImportHandler      *ExportImportHandler
	adminHandler             *AdminHandler
	healthHandlers           *HealthHandlers
	microsoftHandler         *MicrosoftHandler
	ssoHandler               *SSOHandler
	teamHandler              *TeamHandler
	mdmAdapter               *services.MDMAdapter
}

// NewRouter creates a new router with all handlers
func NewRouter(
	logger *logrus.Entry,
	googleOAuth *oauth.GoogleOAuth2Provider,
	syncProcessor *sync.GoogleSyncProcessor,
	msOAuth *oauth.MicrosoftOAuth2Provider,
	msSyncProcessor *sync.MicrosoftSyncProcessor,
	healthHandlers *HealthHandlers,
	mdmAdapter *services.MDMAdapter,
) *Router {
	router := mux.NewRouter()
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		// Fallback for development, but warn
		logger.Warn("JWT_SECRET not configured, using default (INSECURE for production)")
		jwtSecret = "dev-jwt-secret-key-change-in-production"
	}

	// Initialize audit service
	auditService := services.NewAuditService(logger)

	// Initialize rate limiter (default: 10 req/s per tenant, burst 20)
	rateLimitRPS := 10.0
	rateLimitBurst := 20
	if rpsStr := os.Getenv("RATE_LIMIT_RPS"); rpsStr != "" {
		if rps, err := strconv.ParseFloat(rpsStr, 64); err == nil && rps > 0 {
			rateLimitRPS = rps
		}
	}
	if burstStr := os.Getenv("RATE_LIMIT_BURST"); burstStr != "" {
		if burst, err := strconv.Atoi(burstStr); err == nil && burst > 0 {
			rateLimitBurst = burst
		}
	}
	rateLimiter := middleware.NewTenantRateLimiter(rateLimitRPS, rateLimitBurst, logger)

	// Initialize repository and service layers for calendar (in-memory default)
	calendarRepo := repository.NewInMemoryCalendarRepository(logger)
	calendarRepoAdapter := services.NewRepositoryAdapter(calendarRepo, logger)
	calendarService := services.NewCalendarServiceImpl(calendarRepoAdapter, logger)

	// === Redis-backed availability checker (optional, enabled via env) ===
	var availabilityService services.AvailabilityServiceTenantAwareInterface
	blackoutService := services.NewBlackoutServiceImpl(logger)
	tenantService := services.NewTenantServiceImpl(logger)

	// Detect Redis + Hasura configuration to wire the cached availability checker
	cacheEnabled := true
	if v := os.Getenv("CACHE_ENABLED"); v != "" {
		cacheEnabled = (v == "true" || v == "1" || v == "yes")
	}
	redisURL := os.Getenv("REDIS_URL")
	redisPrefix := os.Getenv("REDIS_PREFIX")
	if redisPrefix == "" {
		redisPrefix = "calendar"
	}
	// TTL in seconds (fallback to 3600s)
	ttlSecs := 3600
	if s := os.Getenv("REDIS_CACHE_TTL"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			ttlSecs = v
		}
	}

	// Create Hasura client unconditionally for multiple handlers
	hasuraEndpoint := os.Getenv("HASURA_ENDPOINT")
	hasuraAdmin := os.Getenv("HASURA_ADMIN_SECRET")
	hClient := hasura.NewClient(hasuraEndpoint, hasuraAdmin)

	// If Redis is enabled, initialize cache client + Hasura-backed availability checker
	if cacheEnabled && redisURL != "" {
		// lazy import usage: create cache client
		cacheClient := cache.NewClient(redisURL, redisPrefix, time.Duration(ttlSecs)*time.Second, logger.WithField("component", "cache"))

		// Subscribe to invalidation pub/sub for multi-instance sync
		go cacheClient.SubscribeToInvalidations(context.Background(), func(tenantID, region string) {
			logger.Infof("Received cache invalidation for %s/%s", tenantID, region)
		})

		// Initialize metrics collector
		metricsCollector := metrics.NewMetricsCollector("calendar", "service")

		// Wire availability checker that uses Hasura + Redis cache
		checker := availability.NewChecker(hClient, cacheClient, time.Duration(ttlSecs)*time.Second, logger, metricsCollector)
		availabilityService = services.NewAvailabilityAdapter(checker, os.Getenv("DEFAULT_REGION"), "default", logger)
	} else {
		// Fallback to in-process stub implementation
		availabilityService = services.NewAvailabilityServiceImpl(logger)
	}

	// Initialize external sync service (Phase 4.5)
	externalSyncRepo := repository.NewInMemoryCalendarRepository(logger)
	externalSyncRepoAdapter := services.NewRepositoryAdapter(externalSyncRepo, logger)
	externalSyncService := services.NewExternalSyncService(externalSyncRepoAdapter, auditService, logger)

	// Initialize notification service
	notificationService, _ := services.NewNotificationService(services.NotificationConfig{
		SendGridAPIKey: os.Getenv("SENDGRID_API_KEY"),
		FromEmail:      os.Getenv("SENDGRID_FROM_EMAIL"),
		FromName:       os.Getenv("SENDGRID_FROM_NAME"),
		Logger:         logger,
	})

	// SSO Providrs initialization (Enterprise Auth)
	samlCfg := auth.SAMLConfig{
		MetadataURL: os.Getenv("SAML_IDP_METADATA_URL"),
		EntityID:    os.Getenv("SAML_ENTITY_ID"),
	}
	var samlProvider *auth.SAMLProvider
	if samlCfg.MetadataURL != "" {
		if p, err := auth.NewSAMLProvider(samlCfg); err == nil {
			samlProvider = p
		} else {
			logger.WithError(err).Warn("Failed to initialize SAML provider")
		}
	}

	oidcCfg := auth.OIDCConfig{
		Issuer:       os.Getenv("OIDC_ISSUER_URL"),
		ClientID:     os.Getenv("OIDC_CLIENT_ID"),
		ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("OIDC_REDIRECT_URL"),
	}
	var oidcProvider *auth.OIDCProvider
	if oidcCfg.Issuer != "" {
		if p, err := auth.NewOIDCProvider(context.Background(), oidcCfg); err == nil {
			oidcProvider = p
		} else {
			logger.WithError(err).Warn("Failed to initialize OIDC provider")
		}
	}

	calendarHandler := NewCalendarHandler(calendarService, auditService, logger)

	// Set MDM adapter on calendar handler if available
	if mdmAdapter != nil {
		calendarHandler.SetMDMAdapter(mdmAdapter)
	}

	return &Router{
		router:                   router,
		logger:                   logger.WithField("component", "router"),
		availabilityHandler:      NewAvailabilityHandler(availabilityService, auditService, logger),
		blackoutHandler:          NewBlackoutHandler(blackoutService, auditService, logger),
		calendarHandler:          calendarHandler,
		tenantHandler:            NewTenantHandler(tenantService, auditService, logger),
		externalSyncHandler:      NewExternalSyncHandler(externalSyncService, logger),
		notificationHandler:      NewNotificationHandler(notificationService, auditService, logger),
		notificationPrefsHandler: NewNotificationPreferencesHandler(hClient, auditService, logger),
		settingsHandler:          NewSettingsHandler(hClient, auditService, logger),
		jwtSecret:                jwtSecret,
		rateLimiter:              rateLimiter,
		auditService:             auditService,
		googleOAuth:              googleOAuth,
		syncProcessor:            syncProcessor,
		msOAuth:                  msOAuth,
		msSyncProcessor:          msSyncProcessor,
		hasuraClient:             hClient,
		exportImportHandler:      NewExportImportHandler(hClient, auditService, logger),
		adminHandler:             NewAdminHandler(hClient, healthHandlers, auditService, logger),
		healthHandlers:           healthHandlers,
		microsoftHandler:         NewMicrosoftHandler(msOAuth, msSyncProcessor, logger),
		ssoHandler:               NewSSOHandler(samlProvider, oidcProvider, logger),
		teamHandler:              NewTeamHandler(hClient, auditService, logger),
		mdmAdapter:               mdmAdapter,
	}
}

// syncHandler is a package-level variable to store the sync handler
var syncHandlerInstance *SyncHandler

// RegisterRoutes registers all API routes with authentication middleware
func (r *Router) RegisterRoutes() {
	api := r.router.PathPrefix("/api/v1").Subrouter()

	// Apply middleware to all API routes in correct order:
	// 1. JWT validation
	// 2. Tenant isolation guard
	// 3. Rate limiting (NEW)

	// JWTMiddleware validates the JWT token
	jwtMW := middleware.JWTMiddleware(r.jwtSecret, r.logger)
	api.Use(jwtMW)

	// TenantGuardMiddleware enforces tenant isolation
	tenantMW := middleware.TenantGuardMiddleware(r.logger)
	api.Use(tenantMW)

	// TenantRateLimiter enforces per-tenant rate limits (NEW in Phase 6)
	api.Use(r.rateLimiter.RateLimit)

	// Availability routes
	api.HandleFunc("/availability", r.availabilityHandler.Check).Methods("POST")
	api.HandleFunc("/availability/bulk", r.availabilityHandler.CheckBulk).Methods("POST")
	api.HandleFunc("/availability/metrics", r.availabilityHandler.GetMetrics).Methods("GET")

	// Blackout routes
	api.HandleFunc("/blackouts", r.blackoutHandler.Create).Methods("POST")
	api.HandleFunc("/blackouts/{id}/occurrences", r.blackoutHandler.GetOccurrences).Methods("GET")
	api.HandleFunc("/blackouts/{id}", r.blackoutHandler.Delete).Methods("DELETE")

	// Calendar routes
	api.HandleFunc("/calendars", r.calendarHandler.List).Methods("GET")
	api.HandleFunc("/calendars", r.calendarHandler.Create).Methods("POST")
	api.HandleFunc("/calendars/{id}", r.calendarHandler.Get).Methods("GET")
	api.HandleFunc("/calendars/{id}", r.calendarHandler.Update).Methods("PUT")
	api.HandleFunc("/calendars/{id}", r.calendarHandler.Delete).Methods("DELETE")

	// Profile routes (Phase 4.3)
	profileRepo := repository.NewInMemoryCalendarRepository(r.logger)
	profileRepoAdapter := services.NewRepositoryAdapter(profileRepo, r.logger)
	profileService := services.NewProfileService(profileRepoAdapter, r.auditService, r.logger)
	profileHandler := NewProfileHandler(profileService, r.auditService, r.logger)

	api.HandleFunc("/profiles", profileHandler.Create).Methods("POST")
	api.HandleFunc("/profiles", profileHandler.List).Methods("GET")
	api.HandleFunc("/profiles/{id}", profileHandler.Get).Methods("GET")
	api.HandleFunc("/profiles/{id}", profileHandler.Update).Methods("PUT")
	api.HandleFunc("/profiles/{id}", profileHandler.Delete).Methods("DELETE")
	api.HandleFunc("/profiles/{id}/versions", profileHandler.ListVersions).Methods("GET")

	// External sync routes (Phase 4.5)
	api.HandleFunc("/external-sync", r.externalSyncHandler.CreateSyncConfig).Methods("POST")
	api.HandleFunc("/external-sync", r.externalSyncHandler.ListSyncConfigs).Methods("GET")
	api.HandleFunc("/external-sync/{id}", r.externalSyncHandler.GetSyncConfig).Methods("GET")
	api.HandleFunc("/external-sync/{id}", r.externalSyncHandler.UpdateSyncConfig).Methods("PUT")
	api.HandleFunc("/external-sync/{id}", r.externalSyncHandler.DeleteSyncConfig).Methods("DELETE")
	api.HandleFunc("/external-sync/{id}/trigger", r.externalSyncHandler.TriggerSync).Methods("POST")
	api.HandleFunc("/external-sync/{id}/logs", r.externalSyncHandler.GetSyncLogs).Methods("GET")
	api.HandleFunc("/external-sync/{id}/last-log", r.externalSyncHandler.GetLastSyncLog).Methods("GET")
	api.HandleFunc("/external-sync/validate-provider", r.externalSyncHandler.ValidateProvider).Methods("POST")
	api.HandleFunc("/profiles/{profileId}/external-sync", r.externalSyncHandler.ListSyncConfigsByProfile).Methods("GET")

	// Tenant routes
	api.HandleFunc("/tenants", r.tenantHandler.Create).Methods("POST")
	api.HandleFunc("/tenants/{id}", r.tenantHandler.Get).Methods("GET")
	api.HandleFunc("/tenants/{id}", r.tenantHandler.Update).Methods("PUT")
	api.HandleFunc("/tenants/{id}/config", r.tenantHandler.GetConfig).Methods("GET")
	api.HandleFunc("/tenants/{id}/config", r.tenantHandler.UpdateConfig).Methods("PUT")

	// Settings routes
	api.HandleFunc("/settings/user/{user_id}", r.settingsHandler.GetUserSettings).Methods("GET")
	api.HandleFunc("/settings/user/{user_id}", r.settingsHandler.UpdateUserSettings).Methods("PUT")
	api.HandleFunc("/settings/connected-accounts", r.settingsHandler.GetConnectedAccounts).Methods("GET")
	api.HandleFunc("/settings/connected-accounts/{account_id}", r.settingsHandler.DisconnectAccount).Methods("DELETE")

	// Export/Import routes
	api.HandleFunc("/settings/export", r.exportImportHandler.ExportData).Methods("POST")
	api.HandleFunc("/settings/import", r.exportImportHandler.ImportData).Methods("POST")

	// Admin routes
	api.HandleFunc("/admin/stats", r.adminHandler.GetAdminStats).Methods("GET")
	api.HandleFunc("/admin/users", r.adminHandler.ListUsers).Methods("GET")
	api.HandleFunc("/admin/users/{user_id}", r.adminHandler.DeleteUser).Methods("DELETE")
	api.HandleFunc("/admin/users/{user_id}/role", r.adminHandler.UpdateUserRole).Methods("PUT")
	api.HandleFunc("/admin/sync-stats", r.adminHandler.GetSyncStats).Methods("GET")
	api.HandleFunc("/admin/error-logs", r.adminHandler.GetErrorLogs).Methods("GET")

	// Microsoft routes
	api.HandleFunc("/microsoft/calendars", r.microsoftHandler.ListCalendars).Methods("GET")
	api.HandleFunc("/microsoft/sync", r.microsoftHandler.StartSync).Methods("POST")
	api.HandleFunc("/admin/health", r.adminHandler.GetSystemHealth).Methods("GET")
	api.HandleFunc("/admin/audit-logs", r.adminHandler.GetAuditLogs).Methods("GET")

	// SSO routes
	api.HandleFunc("/auth/saml/login", r.ssoHandler.SAMLLogin).Methods("GET")
	api.HandleFunc("/auth/oidc/login", r.ssoHandler.OIDCLogin).Methods("GET")
	api.HandleFunc("/auth/oidc/callback", r.ssoHandler.OIDCCallback).Methods("GET")

	// Team routes
	api.HandleFunc("/teams", r.teamHandler.ListTeams).Methods("GET")
	api.HandleFunc("/teams", r.teamHandler.CreateTeam).Methods("POST")
	api.HandleFunc("/teams/{id}", r.teamHandler.GetTeam).Methods("GET")

	// Notification routes
	api.HandleFunc("/notifications/unsubscribe", r.notificationHandler.Unsubscribe).Methods("GET")
	api.HandleFunc("/notifications/test-digest", r.notificationHandler.SendTestDigest).Methods("POST")
	api.HandleFunc("/notifications/preferences/{user_id}", r.notificationPrefsHandler.GetPreferences).Methods("GET")
	api.HandleFunc("/notifications/preferences/{user_id}", r.notificationPrefsHandler.UpdatePreferences).Methods("PUT")

	// Settings routes
	api.HandleFunc("/settings", r.settingsHandler.GetUserSettingsWithContext).Methods("GET")
	api.HandleFunc("/settings", r.settingsHandler.UpdateUserSettingsWithContext).Methods("POST")

	// Health check (no auth required)
	r.router.HandleFunc("/api/v1/health", r.healthHandlers.Health).Methods("GET")

	// Ready check (no auth required)
	r.router.HandleFunc("/api/v1/ready", r.healthHandlers.Ready).Methods("GET")

	// Service info (no auth required)
	r.router.HandleFunc("/api/v1/info", r.Info).Methods("GET")

	// Register sync routes if at least one provider is configured
	hasGoogleProvider := r.googleOAuth != nil && r.syncProcessor != nil
	hasMicrosoftProvider := r.msOAuth != nil && r.msSyncProcessor != nil

	if !hasGoogleProvider && !hasMicrosoftProvider {
		r.logger.Warn("No sync handler dependencies configured, skipping sync routes")
	} else {
		var syncHandler *SyncHandler

		// Create sync handler with available providers (nil providers are handled by the handler)
		if hasGoogleProvider || hasMicrosoftProvider {
			syncHandler = NewSyncHandler(r.syncProcessor, r.googleOAuth, r.msSyncProcessor, r.msOAuth, r.auditService, r.logger)
			syncHandlerInstance = syncHandler
		}

		if syncHandler != nil {
			syncRoutes := api.PathPrefix("/sync").Subrouter()

			// Register Google routes if provider is available
			if hasGoogleProvider {
				syncRoutes.HandleFunc("/google", syncHandler.SyncGoogle).Methods("POST")
				syncRoutes.HandleFunc("/google/push/{id}", syncHandler.PushEventToGoogle).Methods("POST")
				syncRoutes.HandleFunc("/google/sync-all", syncHandler.SyncAllToGoogle).Methods("POST")
				syncRoutes.HandleFunc("/google/direction", syncHandler.GetSyncDirection).Methods("GET")
				syncRoutes.HandleFunc("/google/auth-url-pkce", syncHandler.GetPKCEAuthURL).Methods("GET")
				syncRoutes.HandleFunc("/google/callback-pkce", syncHandler.PKCECallback).Methods("GET")
				r.logger.Info("Google sync routes registered")
			} else {
				r.logger.Warn("Google sync handler dependencies not configured, skipping google sync routes")
			}

			// Register Microsoft routes if provider is available
			if hasMicrosoftProvider {
				syncRoutes.HandleFunc("/microsoft", syncHandler.SyncMicrosoft).Methods("POST")
				syncRoutes.HandleFunc("/microsoft/auth-url-pkce", syncHandler.GetMicrosoftPKCEAuthURL).Methods("GET")
				syncRoutes.HandleFunc("/microsoft/callback-pkce", syncHandler.MicrosoftPKCECallback).Methods("GET")
				r.logger.Info("Microsoft sync routes registered")
			} else {
				r.logger.Warn("Microsoft sync handler dependencies not configured, skipping microsoft sync routes")
			}

			// Register common routes if at least one provider is available
			if hasGoogleProvider || hasMicrosoftProvider {
				syncRoutes.HandleFunc("/status", syncHandler.GetStatus).Methods("GET")
				syncRoutes.HandleFunc("/cancel", syncHandler.CancelSync).Methods("POST")
				syncRoutes.HandleFunc("/active", syncHandler.ListActiveSyncs).Methods("GET")
			}
		}
	}

	r.logger.Info("API routes registered with JWT authentication")
}

// Handler returns the underlying mux.Router
func (r *Router) Handler() *mux.Router {
	return r.router
}

// Health is a simple health check endpoint (no auth required)
func (r *Router) Health(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// Info returns service information (no auth required)
func (r *Router) Info(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{
		"service": "calendar-service",
		"version": "1.0.0",
		"status": "running"
	}`))
}

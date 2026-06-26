package reporting

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EnhancedReportingService integrates all reporting features
// This is the main entry point for production use with all enhancements
type EnhancedReportingService struct {
	// Core services
	baseService *Service
	repository  *Repository
	cubeClient  *CubeClient
	renderer    *Renderer

	// Enhancement services
	defCache     *DefinitionCache
	queryCache   *QueryResultCache
	renderCache  *RenderedReportCache
	i18n         *TranslationService
	realtime     *RealtimeMetrics
	intelligence *ReportIntelligence
	rateLimiter  *RateLimiter
	quotaManager *QuotaManager
	auditLogger  *AuditLogger
	dataMasker   *DataMasker
	rlsEngine    *RLSEngine
	workerPool   *WorkerPool

	// Collaboration
	collabHub      *CollaborationHub
	commentService *CommentService
	sharingService *SharingService
	versionService *VersionService

	// Circuit breakers for external services
	cubeBreaker *CircuitBreaker

	mu sync.RWMutex
}

// EnhancedServiceConfig configures the enhanced service
type EnhancedServiceConfig struct {
	// Cache configuration
	CacheEnabled bool
	CacheConfig  *CacheConfig

	// Rate limiting
	RateLimitEnabled bool
	RateLimitConfig  *RateLimitConfig

	// Pool configuration
	PoolConfig *PoolConfig

	// Features
	AIEnabled        bool
	CollabEnabled    bool
	AnalyticsEnabled bool

	// External services
	CubeBaseURL string
	CubeAPIKey  string
	RedisURL    string
}

// DefaultEnhancedServiceConfig returns production defaults
func DefaultEnhancedServiceConfig() *EnhancedServiceConfig {
	return &EnhancedServiceConfig{
		CacheEnabled:     true,
		CacheConfig:      DefaultCacheConfig(),
		RateLimitEnabled: true,
		RateLimitConfig:  DefaultRateLimitConfig(),
		PoolConfig:       DefaultPoolConfig(),
		AIEnabled:        true,
		CollabEnabled:    true,
		AnalyticsEnabled: true,
	}
}

// NewEnhancedReportingService creates a fully-featured reporting service
func NewEnhancedReportingService(
	config *EnhancedServiceConfig,
	repository *Repository,
	cubeClient *CubeClient,
	renderer *Renderer,
) *EnhancedReportingService {

	svc := &EnhancedReportingService{
		repository: repository,
		cubeClient: cubeClient,
		renderer:   renderer,
	}

	// Initialize base service
	svc.baseService = NewService(repository, cubeClient, renderer)

	// Initialize caches
	if config.CacheEnabled && config.CacheConfig != nil {
		svc.defCache = NewDefinitionCache(config.CacheConfig)
		svc.queryCache = NewQueryResultCache(config.CacheConfig)
		svc.renderCache = NewRenderedReportCache(config.CacheConfig)
	}

	// Initialize i18n
	svc.i18n = NewTranslationService(repository)

	// Initialize analytics
	if config.AnalyticsEnabled {
		svc.realtime = NewRealtimeMetrics()
	}

	// Initialize AI intelligence
	if config.AIEnabled {
		svc.intelligence = NewReportIntelligence()
	}

	// Initialize rate limiting
	if config.RateLimitEnabled && config.RateLimitConfig != nil {
		svc.rateLimiter = NewRateLimiter(config.RateLimitConfig)
	}

	// Initialize quotas
	svc.quotaManager = NewQuotaManager()

	// Initialize audit logging
	svc.auditLogger = NewAuditLogger()

	// Initialize security
	svc.dataMasker = NewDataMasker()
	svc.rlsEngine = NewRLSEngine()

	// Initialize worker pool
	if config.PoolConfig != nil {
		svc.workerPool = NewWorkerPool(
			config.PoolConfig.RenderWorkers,
			1000, // Queue size
		)
	}

	// Initialize collaboration
	if config.CollabEnabled {
		svc.collabHub = NewCollaborationHub()
		svc.commentService = NewCommentService()
		svc.sharingService = NewSharingService()
		svc.versionService = NewVersionService()
	}

	// Initialize circuit breaker for Cube.dev
	svc.cubeBreaker = NewCircuitBreaker("cube.dev", 5, 3, 30*time.Second)

	return svc
}

// ============================================================================
// REPORT OPERATIONS WITH ENHANCEMENTS
// ============================================================================

// GetReport retrieves a report with caching and rate limiting
func (s *EnhancedReportingService) GetReport(
	ctx context.Context,
	tenantID uuid.UUID,
	datasourceID uuid.UUID,
	reportID uuid.UUID,
	secCtx *SecurityContext,
) (*ReportDefinition, error) {

	// Rate limiting
	if s.rateLimiter != nil {
		if err := s.rateLimiter.AllowRequest(ctx, tenantID, &secCtx.UserID); err != nil {
			return nil, err
		}
	}

	// Check cache first
	if s.defCache != nil {
		if cached, ok := s.defCache.Get(tenantID, datasourceID, reportID); ok {
			s.trackView(tenantID, reportID, true)
			return cached, nil
		}
	}

	// Fetch from repository
	report, err := s.repository.GetDefinition(ctx, reportID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if s.defCache != nil {
		s.defCache.Set(report)
	}

	// Track analytics
	s.trackView(tenantID, reportID, false)

	// Audit log
	if s.auditLogger != nil {
		s.auditLogger.LogReportAction(ctx, tenantID, secCtx.UserID, AuditEventReportViewed, reportID, nil, "success")
	}

	return report, nil
}

// GetLocalizedReport returns a report with translations applied
func (s *EnhancedReportingService) GetLocalizedReport(
	ctx context.Context,
	tenantID uuid.UUID,
	datasourceID uuid.UUID,
	reportID uuid.UUID,
	locale SupportedLocale,
	secCtx *SecurityContext,
) (*LocalizedReportDefinition, error) {

	// Get the base report
	report, err := s.GetReport(ctx, tenantID, datasourceID, reportID, secCtx)
	if err != nil {
		return nil, err
	}

	// Apply translations
	if s.i18n != nil {
		return s.i18n.GetLocalizedDefinition(report, locale), nil
	}

	// Return without translations
	return &LocalizedReportDefinition{
		ReportDefinition: report,
		Locale:           locale,
	}, nil
}

// ============================================================================
// AI / INTELLIGENCE FEATURES
// ============================================================================

// DetectDataAnomalies checks for anomalies in report data
func (s *EnhancedReportingService) DetectDataAnomalies(
	data []map[string]interface{},
	numericFields []string,
) *AnomalyReport {
	if s.intelligence == nil || s.intelligence.anomalyDetector == nil {
		return nil
	}
	return s.intelligence.anomalyDetector.DetectAnomalies(data, numericFields)
}

// AnalyzeDataTrends analyzes trends in time-series data
func (s *EnhancedReportingService) AnalyzeDataTrends(
	data []map[string]interface{},
	valueField string,
	timeField string,
) *TrendReport {
	if s.intelligence == nil || s.intelligence.trendAnalyzer == nil {
		return nil
	}
	return s.intelligence.trendAnalyzer.AnalyzeTrends(data, valueField, timeField)
}

// GenerateDataInsights generates AI-powered insights from report data
func (s *EnhancedReportingService) GenerateDataInsights(
	ctx context.Context,
	data []map[string]interface{},
	definition *ReportDefinition,
) *InsightReport {
	if s.intelligence == nil || s.intelligence.insightGenerator == nil {
		return nil
	}

	// Detect anomalies and trends to feed into insight generation
	var anomalies *AnomalyReport
	var trends *TrendReport

	// Try to detect anomalies from numeric fields in the data
	if len(data) > 0 {
		numericFields := s.detectNumericFields(data[0])
		if len(numericFields) > 0 {
			anomalies = s.intelligence.anomalyDetector.DetectAnomalies(data, numericFields)
			if len(numericFields) > 0 {
				trends = s.intelligence.trendAnalyzer.AnalyzeTrends(data, numericFields[0], "date")
			}
		}
	}

	return s.intelligence.insightGenerator.GenerateInsights(data, definition, anomalies, trends)
}

// detectNumericFields identifies numeric fields from a data sample
func (s *EnhancedReportingService) detectNumericFields(sample map[string]interface{}) []string {
	var fields []string
	for k, v := range sample {
		switch v.(type) {
		case int, int32, int64, float32, float64:
			fields = append(fields, k)
		}
	}
	return fields
}

// GetQuerySuggestions returns AI-powered query suggestions
func (s *EnhancedReportingService) GetQuerySuggestions(
	ctx context.Context,
	definition *ReportDefinition,
	currentParams map[string]interface{},
) []QuerySuggestion {
	if s.intelligence == nil || s.intelligence.querySuggester == nil {
		return nil
	}
	return s.intelligence.querySuggester.SuggestQueries(ctx, definition, currentParams)
}

// ============================================================================
// COLLABORATION
// ============================================================================

// JoinCollaborationSession joins a collaborative editing session
func (s *EnhancedReportingService) JoinCollaborationSession(
	ctx context.Context,
	tenantID uuid.UUID,
	reportID uuid.UUID,
	userID uuid.UUID,
	displayName string,
) (*CollaborationSession, *CollabClient, error) {
	if s.collabHub == nil {
		return nil, nil, nil // Collaboration not enabled
	}
	return s.collabHub.JoinSession(ctx, tenantID, reportID, userID, displayName)
}

// AddReportComment adds a comment to a report
func (s *EnhancedReportingService) AddReportComment(
	ctx context.Context,
	tenantID uuid.UUID,
	reportID uuid.UUID,
	userID uuid.UUID,
	userName string,
	content string,
	anchor *CommentAnchor,
) (*Comment, error) {
	if s.commentService == nil {
		return nil, nil // Comments not enabled
	}

	comment := &Comment{
		TenantID: tenantID,
		ReportID: reportID,
		UserID:   userID,
		UserName: userName,
		Content:  content,
		Anchor:   anchor,
	}

	if err := s.commentService.AddComment(comment); err != nil {
		return nil, err
	}

	if s.auditLogger != nil {
		s.auditLogger.Log(&AuditEvent{
			TenantID:     tenantID,
			UserID:       userID,
			EventType:    "comment.created",
			ResourceType: "report",
			ResourceID:   reportID,
			Outcome:      "success",
		})
	}

	return comment, nil
}

// ShareReport creates a share configuration for a report
func (s *EnhancedReportingService) ShareReport(
	ctx context.Context,
	config *ShareConfig,
) (*ShareConfig, error) {
	if s.sharingService == nil {
		return nil, nil // Sharing not enabled
	}

	share, err := s.sharingService.Share(config)
	if err != nil {
		return nil, err
	}

	if s.auditLogger != nil {
		s.auditLogger.LogReportAction(ctx, config.TenantID, config.SharedBy, AuditEventReportShared, config.ReportID,
			map[string]interface{}{"share_type": config.ShareType}, "success")
	}

	return share, nil
}

// ============================================================================
// ANALYTICS & METRICS
// ============================================================================

// GetRealtimeMetrics returns current realtime metrics
func (s *EnhancedReportingService) GetRealtimeMetrics() *MetricsSnapshot {
	if s.realtime == nil {
		return nil
	}
	return s.realtime.GetSnapshot()
}

// GetTenantMetrics returns metrics for a specific tenant
func (s *EnhancedReportingService) GetTenantMetrics(tenantID uuid.UUID) *TenantMetrics {
	if s.realtime == nil {
		return nil
	}
	return s.realtime.GetTenantMetrics(tenantID)
}

// ============================================================================
// HELPERS
// ============================================================================

func (s *EnhancedReportingService) trackView(tenantID uuid.UUID, reportID uuid.UUID, cacheHit bool) {
	if s.realtime == nil {
		return
	}
	s.realtime.RecordView(tenantID, reportID)
}

// Shutdown gracefully shuts down all services
func (s *EnhancedReportingService) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.workerPool != nil {
		s.workerPool.Stop()
	}

	if s.collabHub != nil {
		s.collabHub.Stop()
	}

	if s.auditLogger != nil {
		s.auditLogger.Stop()
	}

	return nil
}

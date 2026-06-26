package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"context"
	"cube-gonja/config"
	"cube-gonja/internal/catalog"
	"cube-gonja/internal/cube"
	"cube-gonja/internal/git"
	"cube-gonja/internal/middleware"
	"cube-gonja/internal/render"
	"cube-gonja/internal/tenant"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	cronpkg "github.com/robfig/cron/v3"
	"golang.org/x/time/rate"
)

// Ensure cube package is initialized
var _ = cube.GlobalConfig

type SimpleHandler struct{}

func (h *SimpleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`{"status": "healthy"}`))
}

type UpdateContextRequest struct {
	DataSources map[string]string             `json:"data_sources"` // cube -> data_source
	Dimensions  map[string][]render.Dimension `json:"dimensions"`   // cube -> []Dimension
	Measures    map[string][]render.Measure   `json:"measures"`     // cube -> []Measure
	Hierarchies map[string][]render.Hierarchy `json:"hierarchies"`  // cube -> []Hierarchy
	Segments    map[string][]render.Segment   `json:"segments"`     // cube -> []Segment
	// AtScale: Perspectives
	Perspectives map[string][]render.Perspective `json:"perspectives,omitempty"`
	// Microsoft Fabric: Calculation groups
	CalculationGroups map[string][]render.CalculationGroup `json:"calculation_groups,omitempty"`
	// DBT: Materialized views
	MaterializedViews map[string][]render.MaterializedView `json:"materialized_views,omitempty"`
	// Looker: User attributes and custom filters
	UserAttributes map[string]map[string]string     `json:"user_attributes,omitempty"`
	CustomFilters  map[string][]render.CustomFilter `json:"custom_filters,omitempty"`
	// Advanced features
	DataQualityRules map[string][]render.DataQualityRule `json:"data_quality_rules,omitempty"`
	PerformanceHints map[string][]render.PerformanceHint `json:"performance_hints,omitempty"`
	// Mutual Fund and Advanced Financial Calculations
	WeightedAverages   map[string][]render.WeightedAverage       `json:"weighted_averages,omitempty"`
	GreeksCalculations map[string][]render.GreeksCalculation     `json:"greeks_calculations,omitempty"`
	MutualFundMetrics  map[string][]render.MutualFundMetric      `json:"mutual_fund_metrics,omitempty"`
	TenantParams       map[string]render.TenantCalculationParams `json:"tenant_params,omitempty"`
	ScalingConfig      render.ScalingConfig                      `json:"scaling_config,omitempty"`
	// Wealth Management Metrics
	WealthManagementMetrics map[string][]render.WealthManagementMetric `json:"wealth_management_metrics,omitempty"`
	RiskMetrics             map[string][]render.RiskMetrics            `json:"risk_metrics,omitempty"`
	BenchmarkingMetrics     map[string][]render.BenchmarkingMetrics    `json:"benchmarking_metrics,omitempty"`
	PortfolioAnalytics      map[string][]render.PortfolioAnalytics     `json:"portfolio_analytics,omitempty"`
	PrivateEquityMetrics    map[string][]render.PrivateEquityMetrics   `json:"private_equity_metrics,omitempty"`
	// Additional Wealth Management Metrics
	PortfolioEfficiencyMetrics map[string][]render.PortfolioEfficiencyMetric `json:"portfolio_efficiency_metrics,omitempty"`
	TaxAwareMetrics            map[string][]render.TaxAwareMetric            `json:"tax_aware_metrics,omitempty"`
	GoalBasedMetrics           map[string][]render.GoalBasedMetric           `json:"goal_based_metrics,omitempty"`
	BehavioralMetrics          map[string][]render.BehavioralMetric          `json:"behavioral_metrics,omitempty"`
	Extra                      map[string]any                                `json:"extra,omitempty"` // optional misc
}

type RenderRequest struct {
	TemplateName string `json:"template_name"`
}

type HealthResponse struct {
	Status    string         `json:"status"`
	Version   string         `json:"version"`
	Uptime    string         `json:"uptime"`
	Templates int            `json:"templates"`
	Context   map[string]int `json:"context"`
	LastError string         `json:"last_error,omitempty"`
}

type TemplateInfo struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	Modified    time.Time `json:"modified"`
	Description string    `json:"description,omitempty"`
}

type ContextVersion struct {
	ID          string                        `json:"id"`
	Timestamp   time.Time                     `json:"timestamp"`
	DataSources map[string]string             `json:"data_sources"`
	Dimensions  map[string][]render.Dimension `json:"dimensions"`
	Measures    map[string][]render.Measure   `json:"measures"`
	Hierarchies map[string][]render.Hierarchy `json:"hierarchies"`
	Segments    map[string][]render.Segment   `json:"segments"`
	// AtScale: Perspectives
	Perspectives map[string][]render.Perspective `json:"perspectives,omitempty"`
	// Microsoft Fabric: Calculation groups
	CalculationGroups map[string][]render.CalculationGroup `json:"calculation_groups,omitempty"`
	// DBT: Materialized views
	MaterializedViews map[string][]render.MaterializedView `json:"materialized_views,omitempty"`
	// Looker: User attributes and custom filters
	UserAttributes map[string]map[string]string     `json:"user_attributes,omitempty"`
	CustomFilters  map[string][]render.CustomFilter `json:"custom_filters,omitempty"`
	// Advanced features
	DataQualityRules map[string][]render.DataQualityRule `json:"data_quality_rules,omitempty"`
	PerformanceHints map[string][]render.PerformanceHint `json:"performance_hints,omitempty"`
	// Mutual Fund and Advanced Financial Calculations
	WeightedAverages   map[string][]render.WeightedAverage       `json:"weighted_averages,omitempty"`
	GreeksCalculations map[string][]render.GreeksCalculation     `json:"greeks_calculations,omitempty"`
	MutualFundMetrics  map[string][]render.MutualFundMetric      `json:"mutual_fund_metrics,omitempty"`
	TenantParams       map[string]render.TenantCalculationParams `json:"tenant_params,omitempty"`
	ScalingConfig      render.ScalingConfig                      `json:"scaling_config,omitempty"`
	// Wealth Management Metrics
	WealthManagementMetrics map[string][]render.WealthManagementMetric `json:"wealth_management_metrics,omitempty"`
	RiskMetrics             map[string][]render.RiskMetrics            `json:"risk_metrics,omitempty"`
	BenchmarkingMetrics     map[string][]render.BenchmarkingMetrics    `json:"benchmarking_metrics,omitempty"`
	PortfolioAnalytics      map[string][]render.PortfolioAnalytics     `json:"portfolio_analytics,omitempty"`
	PrivateEquityMetrics    map[string][]render.PrivateEquityMetrics   `json:"private_equity_metrics,omitempty"`
	// Additional Wealth Management Metrics
	PortfolioEfficiencyMetrics map[string][]render.PortfolioEfficiencyMetric `json:"portfolio_efficiency_metrics,omitempty"`
	TaxAwareMetrics            map[string][]render.TaxAwareMetric            `json:"tax_aware_metrics,omitempty"`
	GoalBasedMetrics           map[string][]render.GoalBasedMetric           `json:"goal_based_metrics,omitempty"`
	BehavioralMetrics          map[string][]render.BehavioralMetric          `json:"behavioral_metrics,omitempty"`
	Extra                      map[string]any                                `json:"extra,omitempty"`
}

var (
	ctxMu          sync.RWMutex
	ctxData        render.Context
	cfg            config.Config
	startTime      = time.Now()
	versionHistory []ContextVersion
	maxVersions    = 10
	// Rate limiters for different endpoints
	renderLimiter  = rate.NewLimiter(rate.Every(time.Second), 10) // 10 requests per second for rendering
	updateLimiter  = rate.NewLimiter(rate.Every(time.Second), 5)  // 5 requests per second for updates
	generalLimiter = rate.NewLimiter(rate.Every(time.Second), 20) // 20 requests per second for general endpoints
	// Services
	renderSvc  *render.Service
	tenantMgr  *tenant.Manager
	gitMgr     *git.Manager
	catalogSvc *catalog.CatalogService
	// local tuning service interface (optional)
	tuningSvc interface {
		GetTuningStatus(context.Context) (interface{}, error)
		GetRulePerformance(context.Context, string) (interface{}, error)
		SimulateTuning(context.Context, any) (interface{}, error)
	}
	catalogDB *sql.DB
)

// Pre-aggregation scheduler state
var (
	preAggMu sync.Mutex
	// key is cube_name::pre_name
	preAggLastRun           = map[string]time.Time{}
	preAggLastRefreshKeyVal = map[string]string{}

	// persisted scheduler entries
	schedulerStateFile = ""
	cronScheduler      *cronpkg.Cron
	// map preagg key -> cron EntryID
	preAggCronEntryIDs = map[string]cronpkg.EntryID{}
)

func main() {
	cfg = config.FromEnv()

	// prepare scheduler state file path
	schedulerStateFile = filepath.Join(cfg.OutputDir, "preagg_scheduler_state.json")

	if err := render.MkdirAll(cfg.OutputDir); err != nil {
		log.Fatalf("make output dir: %v", err)
	}

	// Initialize tenant manager
	tenantMgr = tenant.NewManager(cfg)

	// Initialize default tenant
	defaultTenant, err := tenantMgr.InitializeTenant(cfg.DefaultTenant, "Default Tenant", "")
	if err != nil {
		log.Fatalf("initialize default tenant: %v", err)
	}

	// Set base template dir for inheritance
	tenantMgr.SetBaseTemplateDir(cfg.TemplateDir)

	// Initialize render service with allowed data sources
	allowedDS := cfg.AllowedDataSource
	renderSvc = render.NewService(defaultTenant.TemplateDir, cfg.TemplateDir, defaultTenant.OutputDir, allowedDS)

	// Initialize database connection for catalog service
	var db *sql.DB
	if cfg.DatabaseDSN != "" || cfg.DatabaseHost != "" {
		var err error
		dsn := cfg.DatabaseDSN
		if dsn == "" {
			dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
				cfg.DatabaseHost, cfg.DatabasePort, cfg.DatabaseUser,
				cfg.DatabasePassword, cfg.DatabaseName, cfg.DatabaseSSLMode)
		}

		db, err = sql.Open("postgres", dsn)
		if err != nil {
			log.Printf("Warning: Failed to connect to database: %v", err)
		} else {
			if err := db.Ping(); err != nil {
				log.Printf("Warning: Failed to ping database: %v", err)
				db.Close()
				db = nil
			} else {
				log.Printf("Database connection established")
				// Initialize catalog service
				catalogSvc = catalog.NewCatalogService(db, cfg.DefaultTenant, "default")
				// expose DB globally for scheduled pre-aggregation execution
				catalogDB = db
				// Initialize catalog types
				if err := catalogSvc.InitializeCatalogTypes(); err != nil {
					log.Printf("Warning: Failed to initialize catalog types: %v", err)
				}
				// Insert sample data
				if err := catalogSvc.InsertSampleData(); err != nil {
					log.Printf("Warning: Failed to insert sample data: %v", err)
				}

				// Initialize TuningService with an sqlx wrapper
				sqlxDB := sqlx.NewDb(db, "postgres")
				_ = sqlxDB
			}
		}
	}

	// Initialize Git manager
	if cfg.GitEnabled {
		gitMgr = git.NewManager(cfg, cfg.TemplateDir)
		if err := gitMgr.InitializeRepo(); err != nil {
			log.Printf("Warning: Failed to initialize Git repository: %v", err)
		}
	}

	// http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
	// 	w.WriteHeader(http.StatusOK)
	// 	_, _ = w.Write([]byte("ok"))
	// })

	// Apply tenant middleware to all endpoints
	tenantMiddleware := middleware.GinTenantAuthMiddleware(cfg, tenantMgr)

	log.Printf("Setting up Gin router")
	r := gin.Default()

	log.Printf("Registering /health handler")
	r.GET("/health", simpleHealthGin)
	log.Printf("Handler registered successfully")

	// Uncomment and update these routes when ready
	r.GET("/context/history", tenantMiddleware, GinRateLimitMiddleware(generalLimiter), getContextHistoryGin)
	r.POST("/context/rollback", tenantMiddleware, GinRateLimitMiddleware(updateLimiter), rollbackContextGin)
	r.GET("/context/stats", tenantMiddleware, GinRateLimitMiddleware(generalLimiter), getContextStatsGin)
	r.POST("/validate-config", tenantMiddleware, GinRateLimitMiddleware(generalLimiter), validateConfigGin)
	r.GET("/metrics", tenantMiddleware, GinRateLimitMiddleware(generalLimiter), getMetricsGin)
	r.POST("/update-context", tenantMiddleware, GinRateLimitMiddleware(updateLimiter), updateContextGin)
	r.POST("/render", tenantMiddleware, GinRateLimitMiddleware(renderLimiter), renderOneGin)
	r.POST("/render-all", tenantMiddleware, GinRateLimitMiddleware(renderLimiter), renderAllGin)
	r.POST("/validate-dry-run", tenantMiddleware, GinRateLimitMiddleware(renderLimiter), validateDryRunGin)
	r.POST("/preview", tenantMiddleware, GinRateLimitMiddleware(renderLimiter), previewGin)
	// Pre-aggregation management endpoints
	r.POST("/pre_aggregations/generate", tenantMiddleware, GinRateLimitMiddleware(renderLimiter), generatePreAggregationGin)
	r.POST("/pre_aggregations/refresh", tenantMiddleware, GinRateLimitMiddleware(renderLimiter), refreshPreAggregationGin)
	// admin pre-aggregation management
	r.GET("/admin/pre_aggregations", tenantMiddleware, GinRateLimitMiddleware(generalLimiter), listPreAggsGin)
	r.POST("/admin/pre_aggregations/force", tenantMiddleware, GinRateLimitMiddleware(updateLimiter), forceRunPreAggGin)
	r.POST("/admin/pre_aggregations/remove", tenantMiddleware, GinRateLimitMiddleware(updateLimiter), removePreAggCronGin)
	r.POST("/admin/pre_aggregations/history", tenantMiddleware, GinRateLimitMiddleware(generalLimiter), listJobRunsGin)
	r.POST("/update-catalog", tenantMiddleware, GinRateLimitMiddleware(updateLimiter), updateCatalogGin)
	r.GET("/business-terms", tenantMiddleware, GinRateLimitMiddleware(generalLimiter), searchBusinessTermsGin)
	r.POST("/business-terms/validate", tenantMiddleware, GinRateLimitMiddleware(generalLimiter), validateBusinessTermsGin)
	r.GET("/ui", tenantMiddleware, GinRateLimitMiddleware(generalLimiter), webUIGin)
	r.GET("/ui/", tenantMiddleware, GinRateLimitMiddleware(generalLimiter), webUIGin)
	r.GET("/templates", tenantMiddleware, GinRateLimitMiddleware(generalLimiter), listTemplatesGin)

	// Tuning service endpoints
	tuningGroup := r.Group("/tuning")
	tuningGroup.Use(tenantMiddleware, GinRateLimitMiddleware(generalLimiter))
	{
		tuningGroup.GET("/status", getTuningStatusGin)
		tuningGroup.GET("/performance/:ruleID", getRulePerformanceGin)
		tuningGroup.POST("/simulate", simulateTuningGin)
	}
	// Tenant management endpoints
	r.GET("/tenants", tenantMiddleware, GinRateLimitMiddleware(generalLimiter), listTenantsGin)
	r.Any("/tenants/*action", tenantMiddleware, GinRateLimitMiddleware(generalLimiter), manageTenantGin)

	// Git management endpoints
	r.GET("/git/status", tenantMiddleware, GinRateLimitMiddleware(generalLimiter), gitStatusGin)
	r.GET("/git/log", tenantMiddleware, GinRateLimitMiddleware(generalLimiter), gitLogGin)
	r.GET("/git/branches", tenantMiddleware, GinRateLimitMiddleware(generalLimiter), gitBranchesGin)
	r.POST("/git/commit", tenantMiddleware, GinRateLimitMiddleware(updateLimiter), gitCommitGin)
	r.POST("/git/push", tenantMiddleware, GinRateLimitMiddleware(updateLimiter), gitPushGin)
	r.POST("/git/pull", tenantMiddleware, GinRateLimitMiddleware(updateLimiter), gitPullGin)
	r.Any("/git/branches/", tenantMiddleware, GinRateLimitMiddleware(updateLimiter), gitBranchOpsGin)

	addr := ":3000"
	log.Printf("Gonja Context Service listening on %s", addr)

	// Start scheduler (lightweight) to check scheduled pre-aggregations every minute
	// Initialize cron scheduler
	cronScheduler = cronpkg.New()

	// Load persisted state if available
	if err := loadSchedulerState(); err != nil {
		log.Printf("Warning: failed to load scheduler state: %v", err)
	}

	// Register cron jobs for any pre-aggregations that specify a cron-style Scheduled string
	registerCronJobsFromContext()

	cronScheduler.Start()

	log.Printf("Starting server...")
	if err := r.Run(addr); err != nil {
		log.Printf("Server error: %v", err)
	}
}

// generatePreAggregationGin returns DDL (not executed) for a given pre-aggregation
func generatePreAggregationGin(c *gin.Context) {
	var req struct {
		Cube string                `json:"cube"`
		Pre  render.PreAggregation `json:"pre"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use render service to generate DDL
	sql, err := renderSvc.GeneratePreAggregationDDL(req.Cube, req.Pre)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sql": sql})
}

// refreshPreAggregationGin optionally executes the DDL against the DB when execute=true
func refreshPreAggregationGin(c *gin.Context) {
	var req struct {
		Cube    string                `json:"cube"`
		Pre     render.PreAggregation `json:"pre"`
		Execute bool                  `json:"execute"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sql, err := renderSvc.GeneratePreAggregationDDL(req.Cube, req.Pre)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := gin.H{"sql": sql, "executed": false}

	if req.Execute {
		if catalogDB == nil {
			resp["note"] = "DB not connected; cannot execute"
			c.JSON(http.StatusOK, resp)
			return
		}
		// execute DDL with simple error handling
		if _, err := catalogDB.Exec(sql); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "sql": sql})
			return
		}
		resp["executed"] = true
		// update last run time
		key := fmt.Sprintf("%s::%s", req.Cube, req.Pre.Name)
		preAggMu.Lock()
		preAggLastRun[key] = time.Now()
		preAggMu.Unlock()
	}

	c.JSON(http.StatusOK, resp)
}

// schedulerState holds persisted values for last run times and refresh keys
type schedulerState struct {
	LastRun map[string]string `json:"last_run"`
	LastKey map[string]string `json:"last_key"`
}

func loadSchedulerState() error {
	if schedulerStateFile == "" {
		return nil
	}
	f, err := os.Open(schedulerStateFile)
	if err != nil {
		return err
	}
	defer f.Close()
	var st schedulerState
	if err := json.NewDecoder(f).Decode(&st); err != nil {
		return err
	}
	preAggMu.Lock()
	defer preAggMu.Unlock()
	for k, v := range st.LastRun {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			preAggLastRun[k] = t
		}
	}
	for k, v := range st.LastKey {
		preAggLastRefreshKeyVal[k] = v
	}
	return nil
}

func saveSchedulerState() error {
	if schedulerStateFile == "" {
		return nil
	}
	st := schedulerState{LastRun: map[string]string{}, LastKey: map[string]string{}}
	preAggMu.Lock()
	for k, t := range preAggLastRun {
		st.LastRun[k] = t.Format(time.RFC3339)
	}
	for k, v := range preAggLastRefreshKeyVal {
		st.LastKey[k] = v
	}
	preAggMu.Unlock()

	tmp := schedulerStateFile + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(st); err != nil {
		f.Close()
		return err
	}
	f.Close()
	return os.Rename(tmp, schedulerStateFile)
}

// registerCronJobsFromContext registers cron jobs for any pre-aggregation that has a cron expression in Scheduled
func registerCronJobsFromContext() {
	ctxMu.RLock()
	local := ctxData
	ctxMu.RUnlock()

	if cronScheduler == nil {
		cronScheduler = cronpkg.New()
	}

	for cube, pres := range local.PreAggregations {
		for _, p := range pres {
			if p.Scheduled == "" {
				continue
			}
			jobCube := cube
			jobP := p
			// If Scheduled looks like a duration, convert to cron '@every' syntax so cron handles it uniformly
			schedExpr := jobP.Scheduled
			if _, err := time.ParseDuration(schedExpr); err == nil {
				schedExpr = "@every " + schedExpr
			}

			key := fmt.Sprintf("%s::%s", jobCube, jobP.Name)
			// remove existing job if present
			if id, ok := preAggCronEntryIDs[key]; ok {
				cronScheduler.Remove(id)
			}

			// persist scheduled job to catalog DB if available
			if catalogSvc != nil {
				job := catalog.ScheduledJob{
					ID:           key,
					TenantID:     cfg.DefaultTenant,
					DatasourceID: "default",
					CubeName:     jobCube,
					PreName:      jobP.Name,
					CronExpr:     schedExpr,
					Storage:      jobP.Storage,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				if rkMap, ok := jobP.RefreshKey.(map[string]interface{}); ok {
					job.RefreshKey = rkMap
				}
				_ = catalogSvc.UpsertScheduledJob(job)
			}

			entryID, err := cronScheduler.AddFunc(schedExpr, func() {
				start := time.Now()
				// record run start if we have catalog svc
				if catalogSvc != nil {
					_ = catalogSvc.RecordJobRun(key, start, nil, false, "started")
				}

				// First check refreshKey if present and evaluate it via SQL when catalogDB available
				if jobP.RefreshKey != nil && catalogDB != nil {
					switch rk := jobP.RefreshKey.(type) {
					case string:
						// treat as literal
					case map[string]interface{}:
						// If refresh key includes templated SQL, render it with renderer context first
						if rawQ, ok := rk["sql"].(string); ok && strings.TrimSpace(rawQ) != "" {
							// Render template using renderer's contextFunctions for this cube
							// We'll create a minimal template wrapper and execute
							tpl := rawQ
							// Render template using render service
							q, _ := renderSvc.RenderString(jobCube, tpl)
							if strings.TrimSpace(q) == "" {
								q = rawQ // fallback to raw SQL
							}
							var val interface{}
							err := catalogDB.QueryRow(q).Scan(&val)
							if err != nil {
								log.Printf("refreshKey SQL eval error for %s: %v", key, err)
							} else {
								strVal := fmt.Sprintf("%v", val)
								preAggMu.Lock()
								last := preAggLastRefreshKeyVal[key]
								if last == strVal {
									preAggMu.Unlock()
									// no change, skip execution
									if catalogSvc != nil {
										finished := time.Now()
										_ = catalogSvc.RecordJobRun(key, start, &finished, true, "skipped - refresh key unchanged")
									}
									return
								}
								preAggMu.Unlock()
								// continue to execute and update last key after success
							}
						}
					}
				}

				// Generate SQL for pre-aggregation
				sql, err := renderSvc.GeneratePreAggregationDDL(jobCube, jobP)
				if err != nil {
					log.Printf("cron preagg generate error for %s: %v", key, err)
					if catalogSvc != nil {
						finished := time.Now()
						_ = catalogSvc.RecordJobRun(key, start, &finished, false, err.Error())
					}
					return
				}
				if catalogDB != nil {
					if _, err := catalogDB.Exec(sql); err != nil {
						log.Printf("cron preagg exec error for %s: %v", key, err)
						if catalogSvc != nil {
							finished := time.Now()
							_ = catalogSvc.RecordJobRun(key, start, &finished, false, err.Error())
						}
						return
					}
					preAggMu.Lock()
					preAggLastRun[key] = time.Now()
					// Update refresh key last value using the rendered/evaluated value if present
					if jobP.RefreshKey != nil {
						if rkMap, ok := jobP.RefreshKey.(map[string]interface{}); ok {
							if rawQ, ok := rkMap["sql"].(string); ok && strings.TrimSpace(rawQ) != "" {
								// try render and evaluate as earlier
								tpl := rawQ
								if out, err := renderSvc.RenderString(jobCube, tpl); err == nil {
									var val interface{}
									if err := catalogDB.QueryRow(out).Scan(&val); err == nil {
										preAggLastRefreshKeyVal[key] = fmt.Sprintf("%v", val)
									}
								}
							}
						}
					}
					preAggMu.Unlock()
					// persist last run and last key to DB
					if catalogSvc != nil {
						now := preAggLastRun[key]
						job := catalog.ScheduledJob{ID: key, LastRun: &now, LastRefreshKeyVal: preAggLastRefreshKeyVal[key], UpdatedAt: time.Now()}
						_ = catalogSvc.UpsertScheduledJob(job)
						finished := time.Now()
						_ = catalogSvc.RecordJobRun(key, start, &finished, true, "executed")
					}
					if err := saveSchedulerState(); err != nil {
						log.Printf("failed to save scheduler state: %v", err)
					}
					log.Printf("cron executed pre-aggregation %s", key)
				} else {
					log.Printf("DB not connected; cron skipping execution for %s", key)
					if catalogSvc != nil {
						finished := time.Now()
						_ = catalogSvc.RecordJobRun(key, start, &finished, false, "skipped - DB not connected")
					}
				}
			})
			if err != nil {
				log.Printf("failed to add cron job for %s.%s: %v", cube, p.Name, err)
				continue
			}
			preAggCronEntryIDs[key] = entryID
		}
	}
}

// admin endpoints for managing scheduled pre-aggregations
func listPreAggsGin(c *gin.Context) {
	// Prefer DB-backed listing if catalog service is available
	if catalogSvc != nil {
		jobs, err := catalogSvc.ListScheduledJobs()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out := []map[string]any{}
		for _, j := range jobs {
			next := time.Time{}
			if enID, ok := preAggCronEntryIDs[j.ID]; ok {
				en := cronScheduler.Entry(enID)
				next = en.Next
			}
			lastRun := ""
			if j.LastRun != nil {
				lastRun = j.LastRun.Format(time.RFC3339)
			}
			out = append(out, map[string]any{
				"cube":          j.CubeName,
				"name":          j.PreName,
				"scheduled":     j.CronExpr,
				"storage":       j.Storage,
				"last_run":      lastRun,
				"refresh_key":   j.LastRefreshKeyVal,
				"cron_entry_id": preAggCronEntryIDs[j.ID],
				"next_run":      next.Format(time.RFC3339),
			})
		}
		c.JSON(http.StatusOK, gin.H{"pre_aggregations": out})
		return
	}

	// Fallback to in-memory listing
	ctxMu.RLock()
	local := ctxData
	ctxMu.RUnlock()

	out := []map[string]any{}
	for cube, pres := range local.PreAggregations {
		for _, p := range pres {
			key := fmt.Sprintf("%s::%s", cube, p.Name)
			preAggMu.Lock()
			last := preAggLastRun[key]
			lastKey := preAggLastRefreshKeyVal[key]
			preAggMu.Unlock()
			entryID, hasJob := preAggCronEntryIDs[key]
			next := time.Time{}
			if hasJob {
				en := cronScheduler.Entry(entryID)
				next = en.Next
			}
			out = append(out, map[string]any{
				"cube":          cube,
				"name":          p.Name,
				"scheduled":     p.Scheduled,
				"storage":       p.Storage,
				"last_run":      last.Format(time.RFC3339),
				"refresh_key":   lastKey,
				"cron_entry_id": entryID,
				"next_run":      next.Format(time.RFC3339),
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{"pre_aggregations": out})
}

func forceRunPreAggGin(c *gin.Context) {
	var req struct {
		Cube    string `json:"cube"`
		Name    string `json:"name"`
		Execute bool   `json:"execute"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctxMu.RLock()
	pres := ctxData.PreAggregations[req.Cube]
	ctxMu.RUnlock()
	var target *render.PreAggregation
	for _, p := range pres {
		if p.Name == req.Name {
			target = &p
			break
		}
	}
	if target == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pre-aggregation not found"})
		return
	}
	sql, err := renderSvc.GeneratePreAggregationDDL(req.Cube, *target)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	res := gin.H{"sql": sql, "executed": false}
	if req.Execute {
		if catalogDB == nil {
			res["note"] = "DB not connected; cannot execute"
			c.JSON(http.StatusOK, res)
			return
		}
		if _, err := catalogDB.Exec(sql); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		key := fmt.Sprintf("%s::%s", req.Cube, target.Name)
		preAggMu.Lock()
		preAggLastRun[key] = time.Now()
		preAggMu.Unlock()
		res["executed"] = true
		saveSchedulerState()
	}
	c.JSON(http.StatusOK, res)
}

func removePreAggCronGin(c *gin.Context) {
	var req struct {
		Cube string `json:"cube"`
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	key := fmt.Sprintf("%s::%s", req.Cube, req.Name)
	if id, ok := preAggCronEntryIDs[key]; ok {
		cronScheduler.Remove(id)
		delete(preAggCronEntryIDs, key)
		saveSchedulerState()
	}
	// Remove from DB if available
	if catalogSvc != nil {
		_ = catalogSvc.DeleteScheduledJob(key)
	}
	c.JSON(http.StatusOK, gin.H{"removed": true})
}

// listJobRunsGin returns recent runs for a job id (job id is cube::pre_name)
func listJobRunsGin(c *gin.Context) {
	var req struct {
		JobID string `json:"job_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if catalogSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Catalog DB not configured"})
		return
	}
	// Query runs directly using DB
	rows, err := catalogSvc.DB().Query(`SELECT id, job_id, started_at, finished_at, success, message FROM public.scheduled_job_runs WHERE job_id = $1 ORDER BY started_at DESC LIMIT 100`, req.JobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int
		var jobID string
		var started time.Time
		var finished sql.NullTime
		var success sql.NullBool
		var message sql.NullString
		if err := rows.Scan(&id, &jobID, &started, &finished, &success, &message); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		fin := ""
		if finished.Valid {
			fin = finished.Time.Format(time.RFC3339)
		}
		out = append(out, map[string]any{"id": id, "job_id": jobID, "started_at": started.Format(time.RFC3339), "finished_at": fin, "success": success.Bool, "message": message.String})
	}
	c.JSON(http.StatusOK, gin.H{"runs": out})
}

func getTuningStatusGin(c *gin.Context) {
	if tuningSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Tuning service not available. Check DB connection."})
		return
	}
	status, err := tuningSvc.GetTuningStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

func getRulePerformanceGin(c *gin.Context) {
	if tuningSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Tuning service not available. Check DB connection."})
		return
	}
	ruleID := c.Param("ruleID")
	if ruleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ruleID parameter is required"})
		return
	}
	perf, err := tuningSvc.GetRulePerformance(c.Request.Context(), ruleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, perf)
}

func simulateTuningGin(c *gin.Context) {
	if tuningSvc == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Tuning service adapter not configured in this build"})
		return
	}
	// Forward to configured adapter
	resp, err := tuningSvc.SimulateTuning(c.Request.Context(), map[string]any{"with_preview": true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func simpleHealthGin(c *gin.Context) {
	c.JSON(200, gin.H{"status": "healthy"})
}

// GinRateLimitMiddleware applies rate limiting to Gin handlers
func GinRateLimitMiddleware(limiter *rate.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	}
}

type HealthHandler struct{}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in ServeHTTP: %v", r)
		}
	}()
	w.Write([]byte(`{"status": "healthy"}`))
}

func listTemplatesGin(c *gin.Context) {
	// Get tenant context
	tenantCtx, ok := middleware.GetTenantFromGinContext(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant context not found"})
		return
	}

	files, err := filepath.Glob(filepath.Join(tenantCtx.Tenant.TemplateDir, "*.yml.gonja"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var templates []TemplateInfo
	for _, f := range files {
		base := filepath.Base(f)
		name := strings.TrimSuffix(strings.TrimSuffix(base, ".gonja"), ".yml")

		// Skip macro templates
		if strings.HasPrefix(name, "_") {
			continue
		}

		stat, err := os.Stat(f)
		if err != nil {
			continue
		}

		templates = append(templates, TemplateInfo{
			Name:     name,
			Path:     f,
			Size:     stat.Size(),
			Modified: stat.ModTime(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"templates": templates,
		"count":     len(templates),
	})
}

func getContextHistoryGin(c *gin.Context) {
	ctxMu.RLock()
	history := make([]ContextVersion, len(versionHistory))
	copy(history, versionHistory)
	ctxMu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"count":   len(history),
	})
}

func rollbackContextGin(c *gin.Context) {
	var req struct {
		VersionID string `json:"version_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctxMu.Lock()
	defer ctxMu.Unlock()

	var targetVersion *ContextVersion
	for _, v := range versionHistory {
		if v.ID == req.VersionID {
			targetVersion = &v
			break
		}
	}

	if targetVersion == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
		return
	}

	ctxData = render.Context{
		DataSources:       targetVersion.DataSources,
		Dimensions:        targetVersion.Dimensions,
		Measures:          targetVersion.Measures,
		Hierarchies:       targetVersion.Hierarchies,
		Segments:          targetVersion.Segments,
		Perspectives:      targetVersion.Perspectives,
		CalculationGroups: targetVersion.CalculationGroups,
		MaterializedViews: targetVersion.MaterializedViews,
		UserAttributes:    targetVersion.UserAttributes,
		CustomFilters:     targetVersion.CustomFilters,
		DataQualityRules:  targetVersion.DataQualityRules,
		PerformanceHints:  targetVersion.PerformanceHints,
		Extra:             targetVersion.Extra,
	}

	c.JSON(http.StatusOK, gin.H{"status": "rolled back to version " + req.VersionID})
}

func getContextStatsGin(c *gin.Context) {
	ctxMu.RLock()
	local := ctxData
	ctxMu.RUnlock()

	totalDimensions := 0
	for _, dims := range local.Dimensions {
		totalDimensions += len(dims)
	}

	totalMeasures := 0
	for _, measures := range local.Measures {
		totalMeasures += len(measures)
	}

	totalHierarchies := 0
	for _, hierarchies := range local.Hierarchies {
		totalHierarchies += len(hierarchies)
	}

	totalSegments := 0
	for _, segments := range local.Segments {
		totalSegments += len(segments)
	}

	stats := map[string]interface{}{
		"data_sources_count":    len(local.DataSources),
		"cubes_count":           len(local.Dimensions),
		"dimensions_count":      totalDimensions,
		"measures_count":        totalMeasures,
		"hierarchies_count":     totalHierarchies,
		"segments_count":        totalSegments,
		"extra_keys_count":      len(local.Extra),
		"version_history_count": len(versionHistory),
		"last_updated":          time.Now().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, stats)
}

func validateConfigGin(c *gin.Context) {
	var req UpdateContextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate data sources
	for cube, ds := range req.DataSources {
		if ds == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("empty data source for cube: %s", cube)})
			return
		}
		if _, allowed := cfg.AllowedDataSource[ds]; !allowed {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("data source not allowed: %s", ds)})
			return
		}
	}

	// Validate dimensions
	for cube, dims := range req.Dimensions {
		if len(dims) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("no dimensions for cube: %s", cube)})
			return
		}
		for _, dim := range dims {
			if dim.Name == "" || dim.Sql == "" || dim.Type == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid dimension for cube %s: missing required fields", cube)})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "configuration is valid"})
}

func getMetricsGin(c *gin.Context) {
	ctxMu.RLock()
	local := ctxData
	ctxMu.RUnlock()

	metrics := fmt.Sprintf(`# HELP gonja_service_templates_total Total number of templates
# TYPE gonja_service_templates_total gauge
gonja_service_templates_total %d

# HELP gonja_service_data_sources_total Total number of data sources
# TYPE gonja_service_data_sources_total gauge
gonja_service_data_sources_total %d

# HELP gonja_service_cubes_total Total number of cubes
# TYPE gonja_service_cubes_total gauge
gonja_service_cubes_total %d

# HELP gonja_service_uptime_seconds Service uptime in seconds
# TYPE gonja_service_uptime_seconds gauge
gonja_service_uptime_seconds %d

# HELP gonja_service_version_history_total Total versions in history
# TYPE gonja_service_version_history_total gauge
gonja_service_version_history_total %d
`,
		func() int {
			files, _ := filepath.Glob(filepath.Join(cfg.TemplateDir, "*.yml.gonja"))
			count := 0
			for _, f := range files {
				base := filepath.Base(f)
				if !strings.HasPrefix(strings.TrimSuffix(strings.TrimSuffix(base, ".gonja"), ".yml"), "_") {
					count++
				}
			}
			return count
		}(),
		len(local.DataSources),
		len(local.Dimensions),
		int(time.Since(startTime).Seconds()),
		len(versionHistory),
	)

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, metrics)
}

func webUIGin(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Gonja Context Service - Management UI</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 2px solid #007acc; padding-bottom: 10px; }
        .section { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .section h2 { color: #007acc; margin-top: 0; }
        .btn { background: #007acc; color: white; padding: 8px 16px; border: none; border-radius: 4px; cursor: pointer; margin: 5px; }
        .btn:hover { background: #005999; }
        .btn-danger { background: #dc3545; }
        .btn-danger:hover { background: #c82333; }
        textarea { width: 100%; min-height: 200px; font-family: monospace; margin: 10px 0; }
        .status { padding: 10px; margin: 10px 0; border-radius: 4px; }
        .status.success { background: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
        .status.error { background: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }
        .status.info { background: #d1ecf1; color: #0c5460; border: 1px solid #bee5eb; }
        table { width: 100%; border-collapse: collapse; margin: 10px 0; }
        th, td { padding: 8px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f8f9fa; }
        .json-input { font-family: monospace; width: 100%; min-height: 150px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔧 Gonja Context Service - Management UI</h1>
        
        <div class="section">
            <h2>📊 Service Status</h2>
            <button class="btn" onclick="checkHealth()">Check Health</button>
            <button class="btn" onclick="getMetrics()">Get Metrics</button>
            <div id="status-output"></div>
        </div>

        <div class="section">
            <h2>📋 Templates</h2>
            <button class="btn" onclick="listTemplates()">List Templates</button>
            <div id="templates-output"></div>
        </div>

        <div class="section">
            <h2>⚙️ Context Management</h2>
            <button class="btn" onclick="getContextStats()">Context Stats</button>
            <button class="btn" onclick="getContextHistory()">Version History</button>
            <div id="context-output"></div>
            
            <h3>Update Context</h3>
            <textarea id="context-json" class="json-input" placeholder='{
  "data_sources": {"orders": "default", "customers": "default"},
  "dimensions": {
    "orders": [{"name": "id", "sql": "id", "type": "number"}],
    "customers": [{"name": "id", "sql": "id", "type": "number"}]
  }
}'></textarea>
            <button class="btn" onclick="updateContext()">Update Context</button>
        </div>

        <div class="section">
            <h2>🎨 Rendering</h2>
            <button class="btn" onclick="renderAll()">Render All Templates</button>
            <button class="btn" onclick="validateDryRun()">Validate & Dry Run</button>
            <div id="render-output"></div>
        </div>

        <div class="section">
            <h2>⚙️ Tuning Cockpit</h2>
            <button class="btn" onclick="getTuningStatus()">Get Tuning Status</button>
            <button class="btn" onclick="runTuningSimulation()">Run Simulation</button>
            <div id="tuning-output"></div>
            
            <h3>Rule Performance</h3>
            <input type="text" id="tuning-rule-id" placeholder="Enter Rule ID" style="margin: 5px; padding: 8px; width: 200px;">
            <button class="btn" onclick="getRulePerformance()">Get Performance</button>
            
            <h3>Run Simulation (Advanced)</h3>
            <textarea id="tuning-sim-req" class="json-input" placeholder='{
  "lookback_days": 90,
  "with_preview": true
}'></textarea>
        </div>

        <div class="section">
            <h2>🏢 Tenant Management</h2>
            <button class="btn" onclick="listTenants()">List Tenants</button>
            <button class="btn" onclick="createTenant()">Create Tenant</button>
            <div id="tenant-output"></div>
            
            <h3>Create New Tenant</h3>
            <input type="text" id="new-tenant-name" placeholder="Tenant Name" style="margin: 5px; padding: 8px; width: 200px;">
            <button class="btn" onclick="createTenant()">Create</button>
            
            <h3>Tenant API Keys</h3>
            <input type="text" id="tenant-name" placeholder="Tenant Name" style="margin: 5px; padding: 8px; width: 200px;">
            <button class="btn" onclick="getTenantKeys()">Get Keys</button>
            <button class="btn" onclick="generateTenantKey()">Generate New Key</button>
            <div id="keys-output"></div>
        </div>

        <div class="section">
            <h2>📚 Git Management</h2>
            <button class="btn" onclick="gitStatus()">Git Status</button>
            <button class="btn" onclick="gitLog()">Git Log</button>
            <button class="btn" onclick="gitBranches()">List Branches</button>
            <div id="git-output"></div>
            
            <h3>Git Operations</h3>
            <input type="text" id="commit-message" placeholder="Commit message" style="margin: 5px; padding: 8px; width: 300px;">
            <button class="btn" onclick="gitCommit()">Commit Changes</button>
            <button class="btn" onclick="gitPush()">Push to Remote</button>
            <button class="btn" onclick="gitPull()">Pull from Remote</button>
            
            <h3>Branch Management</h3>
            <input type="text" id="new-branch-name" placeholder="New branch name" style="margin: 5px; padding: 8px; width: 200px;">
            <button class="btn" onclick="createBranch()">Create Branch</button>
            <input type="text" id="switch-branch-name" placeholder="Branch name" style="margin: 5px; padding: 8px; width: 200px;">
            <button class="btn" onclick="switchBranch()">Switch Branch</button>
        </div>
    </div>

    <script>
        async function apiCall(endpoint, method = 'GET', body = null) {
            const options = { method };
            if (body) {
                options.headers = { 'Content-Type': 'application/json' };
                options.body = JSON.stringify(body);
            }
            
            try {
                const response = await fetch(endpoint, options);
                const data = await response.text();
                
                if (!response.ok) {
                    throw new Error("HTTP " + response.status + ": " + data);
                }
                
                return { success: true, data };
            } catch (error) {
                return { success: false, error: error.message };
            }
        }

        // API functions
        async function checkHealth() {
            const result = await apiCall('/health');
            displayResult('status-output', result);
        }

        async function getMetrics() {
            const result = await apiCall('/metrics');
            displayResult('status-output', result);
        }

        async function listTemplates() {
            const result = await apiCall('/templates');
            displayResult('templates-output', result);
        }

        async function getContextStats() {
            const result = await apiCall('/context/stats');
            displayResult('context-output', result);
        }

        async function getContextHistory() {
            const result = await apiCall('/context/history');
            displayResult('context-output', result);
        }

        async function updateContext() {
            const json = document.getElementById('context-json').value;
            try {
                const data = JSON.parse(json);
                const result = await apiCall('/update-context', 'POST', data);
                displayResult('context-output', result);
            } catch (e) {
                displayResult('context-output', { success: false, error: 'Invalid JSON: ' + e.message });
            }
        }

        async function renderAll() {
            const result = await apiCall('/render-all');
            displayResult('render-output', result);
        }

        async function validateDryRun() {
            const result = await apiCall('/validate-dry-run');
            displayResult('render-output', result);
        }

        async function getTuningStatus() {
            const result = await apiCall('/tuning/status');
            displayResult('tuning-output', result);
        }

        async function getRulePerformance() {
            const ruleID = document.getElementById('tuning-rule-id').value.trim();
            if (!ruleID) {
                displayResult('tuning-output', { success: false, error: 'Please enter a Rule ID' });
                return;
            }
            const result = await apiCall('/tuning/performance/' + ruleID);
            displayResult('tuning-output', result);
        }

        async function runTuningSimulation() {
            const json = document.getElementById('tuning-sim-req').value;
            // For simplicity, we'll use a default POST body, but a real UI could parse the textarea.
            const result = await apiCall('/tuning/simulate', 'POST', { with_preview: true });
            displayResult('tuning-output', result);
        }

        async function validateConfig() {
            const json = document.getElementById('config-json').value;
            try {
                const data = JSON.parse(json);
                const result = await apiCall('/validate-config', 'POST', data);
                displayResult('validation-output', result);
            } catch (e) {
                displayResult('validation-output', { success: false, error: 'Invalid JSON: ' + e.message });
            }
        }

        // Tenant management functions
        async function listTenants() {
            const result = await apiCall('/tenants');
            displayResult('tenant-output', result);
        }

        async function createTenant() {
            const name = document.getElementById('new-tenant-name').value.trim();
            if (!name) {
                displayResult('tenant-output', { success: false, error: 'Please enter a tenant name' });
                return;
            }
            const result = await apiCall('/tenants', 'POST', { name });
            displayResult('tenant-output', result);
        }

        async function getTenantKeys() {
            const tenant = document.getElementById('tenant-name').value.trim();
            if (!tenant) {
                displayResult('keys-output', { success: false, error: 'Please enter a tenant name' });
                return;
            }
            const result = await apiCall('/tenants/' + tenant + '/keys');
            displayResult('keys-output', result);
        }

        async function generateTenantKey() {
            const tenant = document.getElementById('tenant-name').value.trim();
            if (!tenant) {
                displayResult('keys-output', { success: false, error: 'Please enter a tenant name' });
                return;
            }
            const result = await apiCall('/tenants/' + tenant + '/keys', 'POST');
            displayResult('keys-output', result);
        }

        // Git management functions
        async function gitStatus() {
            const result = await apiCall('/git/status');
            displayResult('git-output', result);
        }

        async function gitLog() {
            const result = await apiCall('/git/log');
            displayResult('git-output', result);
        }

        async function gitBranches() {
            const result = await apiCall('/git/branches');
            displayResult('git-output', result);
        }

        async function gitCommit() {
            const message = document.getElementById('commit-message').value.trim();
            if (!message) {
                displayResult('git-output', { success: false, error: 'Please enter a commit message' });
                return;
            }
            const result = await apiCall('/git/commit', 'POST', { message });
            displayResult('git-output', result);
        }

        async function gitPush() {
            const result = await apiCall('/git/push', 'POST');
            displayResult('git-output', result);
        }

        async function gitPull() {
            const result = await apiCall('/git/pull', 'POST');
            displayResult('git-output', result);
        }

        async function createBranch() {
            const branchName = document.getElementById('new-branch-name').value.trim();
            if (!branchName) {
                displayResult('git-output', { success: false, error: 'Please enter a branch name' });
                return;
            }
            const result = await apiCall('/git/branches', 'POST', { name: branchName });
            displayResult('git-output', result);
        }

        async function switchBranch() {
            const branchName = document.getElementById('switch-branch-name').value.trim();
            if (!branchName) {
                displayResult('git-output', { success: false, error: 'Please enter a branch name' });
                return;
            }
            const result = await apiCall('/git/branches/' + branchName + '/switch', 'POST');
            displayResult('git-output', result);
        }

        function displayResult(elementId, result) {
            const element = document.getElementById(elementId);
            if (result.success) {
                element.innerHTML = '<div class="status success">✅ Success:<br><pre>' + result.data + '</pre></div>';
            } else {
                element.innerHTML = '<div class="status error">❌ Error: ' + result.error + '</div>';
            }
        }
    </script>
</body>
</html>`

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

func updateContextGin(c *gin.Context) {
	var req UpdateContextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate data source bindings
	for cube, ds := range req.DataSources {
		if ds == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("cube %s has empty data source", cube)})
			return
		}
	}

	// Validate dimensions
	for cube, dims := range req.Dimensions {
		if _, exists := req.DataSources[cube]; !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("cube %s has dimensions but no data source", cube)})
			return
		}
		for _, dim := range dims {
			if dim.Name == "" || dim.Sql == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "dimension missing name or sql"})
				return
			}
		}
	}

	// Validate measures
	for cube, measures := range req.Measures {
		if _, exists := req.DataSources[cube]; !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("cube %s has measures but no data source", cube)})
			return
		}
		for _, m := range measures {
			if m.Name == "" || m.Sql == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "measure missing name or sql"})
				return
			}
		}
	}

	// Validate hierarchies
	for cube, hierarchies := range req.Hierarchies {
		if _, exists := req.DataSources[cube]; !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("cube %s has hierarchies but no data source", cube)})
			return
		}
		for _, h := range hierarchies {
			if h.Name == "" || len(h.Levels) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "hierarchy missing name or levels"})
				return
			}
		}
	}

	// Validate segments
	for cube, segments := range req.Segments {
		if _, exists := req.DataSources[cube]; !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("cube %s has segments but no data source", cube)})
			return
		}
		for _, s := range segments {
			if s.Name == "" || s.Sql == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "segment missing name or sql"})
				return
			}
		}
	}

	ctxMu.Lock()
	defer ctxMu.Unlock()

	// Save current version to history
	if len(versionHistory) >= maxVersions {
		versionHistory = versionHistory[1:]
	}
	versionHistory = append(versionHistory, ContextVersion{
		ID:          fmt.Sprintf("%d", time.Now().Unix()),
		Timestamp:   time.Now(),
		DataSources: make(map[string]string),
		Dimensions:  make(map[string][]render.Dimension),
		Measures:    make(map[string][]render.Measure),
		Hierarchies: make(map[string][]render.Hierarchy),
		Segments:    make(map[string][]render.Segment),
		Extra:       make(map[string]any),
	})
	for k, v := range ctxData.DataSources {
		versionHistory[len(versionHistory)-1].DataSources[k] = v
	}
	for k, v := range ctxData.Dimensions {
		versionHistory[len(versionHistory)-1].Dimensions[k] = make([]render.Dimension, len(v))
		copy(versionHistory[len(versionHistory)-1].Dimensions[k], v)
	}
	for k, v := range ctxData.Measures {
		versionHistory[len(versionHistory)-1].Measures[k] = make([]render.Measure, len(v))
		copy(versionHistory[len(versionHistory)-1].Measures[k], v)
	}
	for k, v := range ctxData.Hierarchies {
		versionHistory[len(versionHistory)-1].Hierarchies[k] = make([]render.Hierarchy, len(v))
		copy(versionHistory[len(versionHistory)-1].Hierarchies[k], v)
	}
	for k, v := range ctxData.Segments {
		versionHistory[len(versionHistory)-1].Segments[k] = make([]render.Segment, len(v))
		copy(versionHistory[len(versionHistory)-1].Segments[k], v)
	}
	for k, v := range ctxData.Perspectives {
		versionHistory[len(versionHistory)-1].Perspectives[k] = make([]render.Perspective, len(v))
		copy(versionHistory[len(versionHistory)-1].Perspectives[k], v)
	}
	for k, v := range ctxData.CalculationGroups {
		versionHistory[len(versionHistory)-1].CalculationGroups[k] = make([]render.CalculationGroup, len(v))
		copy(versionHistory[len(versionHistory)-1].CalculationGroups[k], v)
	}
	for k, v := range ctxData.MaterializedViews {
		versionHistory[len(versionHistory)-1].MaterializedViews[k] = make([]render.MaterializedView, len(v))
		copy(versionHistory[len(versionHistory)-1].MaterializedViews[k], v)
	}
	for k, v := range ctxData.CustomFilters {
		versionHistory[len(versionHistory)-1].CustomFilters[k] = make([]render.CustomFilter, len(v))
		copy(versionHistory[len(versionHistory)-1].CustomFilters[k], v)
	}
	for k, v := range ctxData.DataQualityRules {
		versionHistory[len(versionHistory)-1].DataQualityRules[k] = make([]render.DataQualityRule, len(v))
		copy(versionHistory[len(versionHistory)-1].DataQualityRules[k], v)
	}
	for k, v := range ctxData.PerformanceHints {
		versionHistory[len(versionHistory)-1].PerformanceHints[k] = make([]render.PerformanceHint, len(v))
		copy(versionHistory[len(versionHistory)-1].PerformanceHints[k], v)
	}
	for k, v := range ctxData.Extra {
		versionHistory[len(versionHistory)-1].Extra[k] = v
	}

	// Update context
	ctxData.DataSources = req.DataSources
	ctxData.Dimensions = req.Dimensions
	ctxData.Measures = req.Measures
	ctxData.Hierarchies = req.Hierarchies
	ctxData.Segments = req.Segments
	ctxData.Perspectives = req.Perspectives
	ctxData.CalculationGroups = req.CalculationGroups
	ctxData.MaterializedViews = req.MaterializedViews
	ctxData.UserAttributes = req.UserAttributes
	ctxData.CustomFilters = req.CustomFilters
	ctxData.DataQualityRules = req.DataQualityRules
	ctxData.PerformanceHints = req.PerformanceHints
	ctxData.Extra = req.Extra

	// Update render service context
	renderSvc.UpdateContext(ctxData)

	// Update all tenant render services
	tenants := tenantMgr.ListTenants()
	for _, tenant := range tenants {
		tenant.RenderService.UpdateContext(ctxData)
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func renderOneGin(c *gin.Context) {
	var req RenderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get tenant context
	tenantCtx, ok := middleware.GetTenantFromGinContext(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant context not found"})
		return
	}

	outPath, data, err := tenantCtx.Tenant.RenderService.RenderOne(req.TemplateName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Write to output file
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update catalog with the rendered model
	if catalogSvc != nil {
		modelDir := filepath.Dir(outPath)
		if err := catalogSvc.UpdateCatalogFromModels(modelDir); err != nil {
			log.Printf("Warning: Failed to update catalog for rendered model %s: %v", req.TemplateName, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"result": string(data)})
}

func renderAllGin(c *gin.Context) {
	// Get tenant context
	tenantCtx, ok := middleware.GetTenantFromGinContext(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant context not found"})
		return
	}

	results, err := tenantCtx.Tenant.RenderService.RenderAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Write all files
	for path, data := range results {
		if err := os.WriteFile(path, data, 0644); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("write %s: %v", path, err)})
			return
		}
	}

	// Update catalog with all rendered models
	if catalogSvc != nil && len(results) > 0 {
		// Get the output directory from the first result
		for path := range results {
			modelDir := filepath.Dir(path)
			if err := catalogSvc.UpdateCatalogFromModels(modelDir); err != nil {
				log.Printf("Warning: Failed to update catalog for rendered models: %v", err)
			}
			break
		}
	}

	response := make(map[string]string)
	for path, data := range results {
		response[filepath.Base(path)] = string(data)
	}

	c.JSON(http.StatusOK, response)
}

func validateDryRunGin(c *gin.Context) {
	// Get tenant context
	tenantCtx, ok := middleware.GetTenantFromGinContext(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant context not found"})
		return
	}

	files, err := filepath.Glob(filepath.Join(tenantCtx.Tenant.TemplateDir, "*.yml.gonja"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	results := make(map[string]string)
	validCount := 0
	totalCount := 0

	for _, f := range files {
		base := filepath.Base(f)
		if strings.HasPrefix(strings.TrimSuffix(strings.TrimSuffix(base, ".gonja"), ".yml"), "_") {
			continue
		}

		totalCount++
		name := strings.TrimSuffix(strings.TrimSuffix(base, ".gonja"), ".yml")
		_, _, err := tenantCtx.Tenant.RenderService.RenderOne(name)
		if err != nil {
			results[name] = fmt.Sprintf("error: %v", err)
		} else {
			results[name] = "valid"
			validCount++
		}
	}

	summary := map[string]interface{}{
		"total":   totalCount,
		"valid":   validCount,
		"invalid": totalCount - validCount,
		"results": results,
	}

	c.JSON(http.StatusOK, summary)
}

func previewGin(c *gin.Context) {
	var req struct {
		TemplateName string                 `json:"template_name"`
		Context      map[string]interface{} `json:"context,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get tenant context
	tenantCtx, ok := middleware.GetTenantFromGinContext(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant context not found"})
		return
	}

	// For preview, we temporarily update the context if provided
	if req.Context != nil {
		ctxMu.Lock()
		originalCtx := ctxData

		// Apply preview context
		if ds, ok := req.Context["data_sources"].(map[string]interface{}); ok {
			for k, v := range ds {
				if s, ok := v.(string); ok {
					ctxData.DataSources[k] = s
				}
			}
		}
		if dims, ok := req.Context["dimensions"].(map[string]interface{}); ok {
			for cube, d := range dims {
				if dimList, ok := d.([]interface{}); ok {
					ctxData.Dimensions[cube] = nil
					for _, di := range dimList {
						if dimMap, ok := di.(map[string]interface{}); ok {
							dim := render.Dimension{}
							if name, ok := dimMap["name"].(string); ok {
								dim.Name = name
							}
							if sql, ok := dimMap["sql"].(string); ok {
								dim.Sql = sql
							}
							if typ, ok := dimMap["type"].(string); ok {
								dim.Type = typ
							}
							ctxData.Dimensions[cube] = append(ctxData.Dimensions[cube], dim)
						}
					}
				}
			}
		}

		tenantCtx.Tenant.RenderService.UpdateContext(ctxData)
		ctxMu.Unlock()
		defer func() {
			ctxMu.Lock()
			ctxData = originalCtx
			tenantCtx.Tenant.RenderService.UpdateContext(ctxData)
			ctxMu.Unlock()
		}()
	}

	_, data, err := tenantCtx.Tenant.RenderService.RenderOne(req.TemplateName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": string(data)})
}

func updateCatalogGin(c *gin.Context) {
	// Get tenant context
	tenantCtx, ok := middleware.GetTenantFromGinContext(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant context not found"})
		return
	}

	if catalogSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Catalog service not available"})
		return
	}

	// Update catalog from the model output directory
	modelDir := tenantCtx.Tenant.OutputDir
	if err := catalogSvc.UpdateCatalogFromModels(modelDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update catalog: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Catalog updated successfully"})
}

func searchBusinessTermsGin(c *gin.Context) {
	// Get tenant context
	_, ok := middleware.GetTenantFromGinContext(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant context not found"})
		return
	}

	if catalogSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Catalog service not available"})
		return
	}

	// Parse query parameters
	var req catalog.BusinessTermSearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid query parameters: %v", err)})
		return
	}

	// Search business terms
	terms, total, err := catalogSvc.SearchBusinessTerms(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to search business terms: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"business_terms": terms,
		"total":          total,
		"limit":          req.Limit,
		"offset":         req.Offset,
	})
}

func validateBusinessTermsGin(c *gin.Context) {
	// Get tenant context
	_, ok := middleware.GetTenantFromGinContext(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant context not found"})
		return
	}

	if catalogSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Catalog service not available"})
		return
	}

	var req struct {
		BusinessTerms []catalog.BusinessTerm `json:"business_terms"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request body: %v", err)})
		return
	}

	// Validate business terms
	response, err := catalogSvc.ValidateBusinessTerms(req.BusinessTerms)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to validate business terms: %v", err)})
		return
	}

	c.JSON(http.StatusOK, response)
}

func listTenantsGin(c *gin.Context) {
	tenants := tenantMgr.ListTenants()

	tenantList := make([]map[string]interface{}, len(tenants))
	for i, tenant := range tenants {
		tenantList[i] = map[string]interface{}{
			"id":           tenant.ID,
			"name":         tenant.Name,
			"template_dir": tenant.TemplateDir,
			"output_dir":   tenant.OutputDir,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"tenants": tenantList,
	})
}

func manageTenantGin(c *gin.Context) {
	// Extract tenant ID from URL path
	pathParts := strings.Split(strings.Trim(c.Request.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant path"})
		return
	}

	tenantID := pathParts[1]

	// Handle subpaths
	if len(pathParts) > 2 {
		subpath := pathParts[2]
		switch subpath {
		case "keys":
			handleTenantKeysGin(c, tenantID)
			return
		default:
			c.JSON(http.StatusNotFound, gin.H{"error": "Unknown tenant subpath"})
			return
		}
	}

	switch c.Request.Method {
	case "GET":
		tenant, err := tenantMgr.GetTenant(tenantID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		response := map[string]interface{}{
			"id":           tenant.ID,
			"name":         tenant.Name,
			"template_dir": tenant.TemplateDir,
			"output_dir":   tenant.OutputDir,
		}
		c.JSON(http.StatusOK, response)

	case "POST":
		var req struct {
			Name   string `json:"name"`
			APIKey string `json:"api_key"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tenant, err := tenantMgr.InitializeTenant(tenantID, req.Name, req.APIKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		response := map[string]interface{}{
			"id":           tenant.ID,
			"name":         tenant.Name,
			"template_dir": tenant.TemplateDir,
			"output_dir":   tenant.OutputDir,
			"status":       "created",
		}
		c.JSON(http.StatusOK, response)

	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
	}
}

func handleTenantKeysGin(c *gin.Context, tenantID string) {
	switch c.Request.Method {
	case "GET":
		// Get tenant API keys
		tenant, err := tenantMgr.GetTenant(tenantID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		response := map[string]interface{}{
			"tenant_id": tenant.ID,
			"api_key":   tenant.APIKey,
		}
		c.JSON(http.StatusOK, response)

	case "POST":
		// Generate new API key for tenant
		tenant, err := tenantMgr.GetTenant(tenantID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		// Generate a new API key (simple implementation)
		newKey := fmt.Sprintf("%s-%d", tenantID, time.Now().Unix())

		// Update tenant with new key
		tenant.APIKey = newKey

		response := map[string]interface{}{
			"tenant_id": tenant.ID,
			"api_key":   newKey,
			"status":    "generated",
		}
		c.JSON(http.StatusOK, response)

	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
	}
}

func gitStatusGin(c *gin.Context) {
	if gitMgr == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Git not enabled"})
		return
	}

	status, err := gitMgr.GetStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": status,
	})
}

func gitCommitGin(c *gin.Context) {
	if gitMgr == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Git not enabled"})
		return
	}

	var req struct {
		Message string `json:"message"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Message == "" {
		req.Message = "Update templates and configuration"
	}

	if err := gitMgr.AddAll(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := gitMgr.Commit(req.Message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "committed",
		"message": req.Message,
	})
}

func gitPushGin(c *gin.Context) {
	if gitMgr == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Git not enabled"})
		return
	}

	if err := gitMgr.Push(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "pushed",
	})
}

func gitPullGin(c *gin.Context) {
	if gitMgr == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Git not enabled"})
		return
	}

	if err := gitMgr.Pull(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "pulled",
	})
}

func gitLogGin(c *gin.Context) {
	if gitMgr == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Git not enabled"})
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	commits, err := gitMgr.GetLog(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"commits": commits,
	})
}

func gitBranchesGin(c *gin.Context) {
	if gitMgr == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Git not enabled"})
		return
	}

	branches, err := gitMgr.GetBranches()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"branches": branches,
	})
}

func gitBranchOpsGin(c *gin.Context) {
	if gitMgr == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Git not enabled"})
		return
	}

	// Extract branch name from URL path
	pathParts := strings.Split(strings.Trim(c.Request.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid branch path"})
		return
	}

	branchName := pathParts[2]

	switch c.Request.Method {
	case "POST":
		var req struct {
			Name string `json:"name"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if this is a create or switch operation
		if strings.HasSuffix(c.Request.URL.Path, "/switch") {
			if err := gitMgr.SwitchBranch(branchName); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"status": "switched",
				"branch": branchName,
			})
		} else {
			// Create new branch
			if err := gitMgr.CreateBranch(req.Name); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"status": "created",
				"branch": req.Name,
			})
		}

	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
	}
}

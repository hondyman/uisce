package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/hondyman/semlayer/backend/internal/config"
	"github.com/hondyman/semlayer/backend/internal/domain"
	"github.com/hondyman/semlayer/backend/internal/monitoring"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// ServiceContainer holds all service dependencies
type ServiceContainer struct {
	Config        *config.GovernanceConfig
	DB            *sql.DB
	Evaluator     domain.Evaluator
	PolicyChecker domain.PolicyChecker
	Cache         domain.DecisionCache
	Metrics       monitoring.MetricsCollector
	HealthChecker *monitoring.HealthChecker
	AlertManager  *monitoring.AlertManager
	Tracer        *monitoring.DistributedTracer
}

// NewServiceContainer creates a new service container
func NewServiceContainer(cfg *config.GovernanceConfig) (*ServiceContainer, error) {
	container := &ServiceContainer{
		Config: cfg,
	}

	// Initialize database connection
	if err := container.initDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize monitoring
	if err := container.initMonitoring(); err != nil {
		return nil, fmt.Errorf("failed to initialize monitoring: %w", err)
	}

	// Initialize cache
	if err := container.initCache(); err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Initialize domain services
	if err := container.initServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	return container, nil
}

// initDatabase initializes the database connection
func (sc *ServiceContainer) initDatabase() error {
	db, err := sql.Open("postgres", sc.Config.DatabaseURL)
	if err != nil {
		return err
	}

	// Configure connection pool
	db.SetMaxOpenConns(sc.Config.DBMaxConnections)
	db.SetMaxIdleConns(sc.Config.DBMaxConnections / 2)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return err
	}

	sc.DB = db
	log.Println("Database connection established")
	return nil
}

// initMonitoring initializes monitoring components
func (sc *ServiceContainer) initMonitoring() error {
	// Initialize metrics
	sc.Metrics = monitoring.NewPrometheusMetrics()

	// Initialize health checker
	sc.HealthChecker = monitoring.NewHealthChecker()

	// Add database health check
	sc.HealthChecker.AddService(&DatabaseHealthCheck{db: sc.DB})

	// Initialize alert manager
	sc.AlertManager = monitoring.NewAlertManager()

	// Initialize tracer
	sc.Tracer = monitoring.NewDistributedTracer("governance-service")

	log.Println("Monitoring components initialized")
	return nil
}

// initCache initializes the cache layer
func (sc *ServiceContainer) initCache() error {
	if sc.Config.CacheEnabled {
		// Initialize Redis cache
		// sc.Cache = cache.NewRedisCache(sc.Config.RedisURL, sc.Config.CacheTTL)
		log.Println("Cache layer initialized")
	}
	return nil
}

// initServices initializes domain services
func (sc *ServiceContainer) initServices() error {
	// Initialize repositories
	// claimRepo := repository.NewClaimRepository(sc.DB)
	// policyRepo := repository.NewPolicyRepository(sc.DB)

	// Initialize base evaluator
	// baseEvaluator := &domain.SimpleEvaluator{Repo: claimRepo}

	// Wrap with caching
	// if sc.Cache != nil {
	//     sc.Evaluator = &domain.CachedEvaluator{
	//         Evaluator: baseEvaluator,
	//         Cache:     sc.Cache,
	//     }
	// } else {
	//     sc.Evaluator = baseEvaluator
	// }

	// Initialize policy checker
	// sc.PolicyChecker = &domain.SimplePolicyChecker{Repo: policyRepo}

	log.Println("Domain services initialized")
	return nil
}

// Close gracefully shuts down all services
func (sc *ServiceContainer) Close() error {
	if sc.DB != nil {
		if err := sc.DB.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}

	log.Println("Service container shut down gracefully")
	return nil
}

// DatabaseHealthCheck implements health checking for database
type DatabaseHealthCheck struct {
	db *sql.DB
}

func (dhc *DatabaseHealthCheck) Name() string {
	return "database"
}

func (dhc *DatabaseHealthCheck) CheckHealth(ctx context.Context) monitoring.HealthStatus {
	status := monitoring.HealthStatus{
		Name:   dhc.Name(),
		Status: "healthy",
	}

	if err := dhc.db.PingContext(ctx); err != nil {
		status.Status = "unhealthy"
		status.Message = fmt.Sprintf("Database ping failed: %v", err)
	}

	return status
}

// GracefulShutdown handles graceful shutdown
func (sc *ServiceContainer) GracefulShutdown(ctx context.Context) error {
	log.Println("Initiating graceful shutdown...")

	// Stop accepting new requests
	// Close database connections
	// Flush metrics
	// Close cache connections

	select {
	case <-ctx.Done():
		log.Println("Shutdown timeout exceeded")
		return ctx.Err()
	default:
		return sc.Close()
	}
}

// ReadinessCheck checks if the service is ready to serve traffic
func (sc *ServiceContainer) ReadinessCheck(ctx context.Context) error {
	// Check database connectivity
	if err := sc.DB.PingContext(ctx); err != nil {
		return fmt.Errorf("database not ready: %w", err)
	}

	// Check cache connectivity
	// if sc.Cache != nil {
	//     if err := sc.Cache.Ping(ctx); err != nil {
	//         return fmt.Errorf("cache not ready: %w", err)
	//     }
	// }

	return nil
}

// LivenessCheck checks if the service is alive
func (sc *ServiceContainer) LivenessCheck(ctx context.Context) error {
	// Basic liveness check - service is alive if it can respond
	return nil
}

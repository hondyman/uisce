package observability

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof" // Import for pprof endpoints
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// ObservabilityManager handles all observability concerns
type ObservabilityManager struct {
	registry      *prometheus.Registry
	httpServer    *http.Server
	metricsServer *http.Server
	shutdownCh    chan struct{}
	wg            sync.WaitGroup

	// RED metrics (Rate, Errors, Duration)
	requestCount    *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	errorCount      *prometheus.CounterVec

	// USE metrics (Utilization, Saturation, Errors)
	cacheUtilization  *prometheus.GaugeVec
	dbConnectionUtil  *prometheus.GaugeVec
	memoryUtilization prometheus.Gauge
	goroutineCount    prometheus.Gauge
	mutexBlockProfile *prometheus.GaugeVec
}

// NewObservabilityManager creates a new observability manager
func NewObservabilityManager() *ObservabilityManager {
	reg := prometheus.NewRegistry()

	om := &ObservabilityManager{
		registry:   reg,
		shutdownCh: make(chan struct{}),
	}

	om.initializeMetrics()
	om.startBackgroundTasks()

	return om
}

// initializeMetrics sets up all Prometheus metrics
func (om *ObservabilityManager) initializeMetrics() {
	// RED Metrics
	om.requestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "semlayer_requests_total",
			Help: "Total number of requests by endpoint and method",
		},
		[]string{"endpoint", "method", "status"},
	)

	om.requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "semlayer_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint", "method"},
	)

	om.errorCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "semlayer_errors_total",
			Help: "Total number of errors by type",
		},
		[]string{"type", "endpoint"},
	)

	// USE Metrics
	om.cacheUtilization = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "semlayer_cache_utilization_ratio",
			Help: "Cache utilization ratio (0-1)",
		},
		[]string{"cache_name", "shard"},
	)

	om.dbConnectionUtil = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "semlayer_db_connections_active",
			Help: "Number of active database connections",
		},
		[]string{"pool_type"},
	)

	om.memoryUtilization = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "semlayer_memory_utilization_bytes",
			Help: "Current memory utilization in bytes",
		},
	)

	om.goroutineCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "semlayer_goroutines_total",
			Help: "Total number of goroutines",
		},
	)

	om.mutexBlockProfile = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "semlayer_mutex_block_seconds",
			Help: "Mutex block time in seconds",
		},
		[]string{"mutex_name"},
	)

	// Register all metrics
	om.registry.MustRegister(
		om.requestCount,
		om.requestDuration,
		om.errorCount,
		om.cacheUtilization,
		om.dbConnectionUtil,
		om.memoryUtilization,
		om.goroutineCount,
		om.mutexBlockProfile,
	)
}

// startBackgroundTasks starts background monitoring tasks
func (om *ObservabilityManager) startBackgroundTasks() {
	om.wg.Add(1)
	go om.systemMetricsCollector()
}

// systemMetricsCollector periodically collects system metrics
func (om *ObservabilityManager) systemMetricsCollector() {
	defer om.wg.Done()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-om.shutdownCh:
			return
		case <-ticker.C:
			om.collectSystemMetrics()
		}
	}
}

// collectSystemMetrics gathers current system metrics
func (om *ObservabilityManager) collectSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	om.memoryUtilization.Set(float64(m.Alloc))
	om.goroutineCount.Set(float64(runtime.NumGoroutine()))
}

// StartMetricsServer starts the Prometheus metrics HTTP server
func (om *ObservabilityManager) StartMetricsServer(port string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(om.registry, promhttp.HandlerOpts{}))

	om.metricsServer = &http.Server{
		Addr:    port,
		Handler: mux,
	}

	logging.GetLogger().Info("Starting metrics server", zap.String("port", port))
	return om.metricsServer.ListenAndServe()
}

// StartPprofServer starts the pprof HTTP server for profiling
func (om *ObservabilityManager) StartPprofServer(port string) error {
	om.httpServer = &http.Server{
		Addr:    port,
		Handler: nil, // Default mux with pprof handlers
	}

	logging.GetLogger().Info("Starting pprof server", zap.String("port", port))
	return om.httpServer.ListenAndServe()
}

// RecordRequest records a request metric
func (om *ObservabilityManager) RecordRequest(endpoint, method, status string, duration time.Duration) {
	om.requestCount.WithLabelValues(endpoint, method, status).Inc()
	om.requestDuration.WithLabelValues(endpoint, method).Observe(duration.Seconds())
}

// RecordError records an error metric
func (om *ObservabilityManager) RecordError(errorType, endpoint string) {
	om.errorCount.WithLabelValues(errorType, endpoint).Inc()
}

// UpdateCacheUtilization updates cache utilization metrics
func (om *ObservabilityManager) UpdateCacheUtilization(cacheName, shard string, utilization float64) {
	om.cacheUtilization.WithLabelValues(cacheName, shard).Set(utilization)
}

// UpdateDBConnections updates database connection metrics
func (om *ObservabilityManager) UpdateDBConnections(poolType string, active int) {
	om.dbConnectionUtil.WithLabelValues(poolType).Set(float64(active))
}

// CaptureProfile captures a runtime profile to file
func (om *ObservabilityManager) CaptureProfile(profileType string, duration time.Duration) error {
	filename := fmt.Sprintf("profile_%s_%d.pprof", profileType, time.Now().Unix())

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create profile file: %w", err)
	}
	defer f.Close()

	switch profileType {
	case "cpu":
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("could not start CPU profile: %w", err)
		}
		time.Sleep(duration)
		pprof.StopCPUProfile()
	case "heap":
		if err := pprof.WriteHeapProfile(f); err != nil {
			return fmt.Errorf("could not write heap profile: %w", err)
		}
	case "mutex":
		if err := pprof.Lookup("mutex").WriteTo(f, 0); err != nil {
			return fmt.Errorf("could not write mutex profile: %w", err)
		}
	case "block":
		if err := pprof.Lookup("block").WriteTo(f, 0); err != nil {
			return fmt.Errorf("could not write block profile: %w", err)
		}
	default:
		return fmt.Errorf("unknown profile type: %s", profileType)
	}

	logging.GetLogger().Info("Profile captured", zap.String("type", profileType), zap.String("file", filename))
	return nil
}

// Shutdown gracefully shuts down the observability manager
func (om *ObservabilityManager) Shutdown(ctx context.Context) error {
	close(om.shutdownCh)

	if om.httpServer != nil {
		if err := om.httpServer.Shutdown(ctx); err != nil {
			logging.GetLogger().Error("Error shutting down pprof server", zap.Error(err))
		}
	}

	if om.metricsServer != nil {
		if err := om.metricsServer.Shutdown(ctx); err != nil {
			logging.GetLogger().Error("Error shutting down metrics server", zap.Error(err))
		}
	}

	om.wg.Wait()
	return nil
}

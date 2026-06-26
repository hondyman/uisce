package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"calendar-service/internal/cache"
	"calendar-service/internal/hasura"

	"github.com/sirupsen/logrus"
	"go.temporal.io/sdk/client"
)

// HealthHandlers handles health check endpoints
type HealthHandlers struct {
	hasuraClient   *hasura.Client
	redisClient    *cache.CalendarCache
	temporalClient client.Client
	logger         *logrus.Entry
}

// NewHealthHandlers creates a new health handlers instance
func NewHealthHandlers(hc *hasura.Client, rc *cache.CalendarCache, tc client.Client, logger *logrus.Entry) *HealthHandlers {
	return &HealthHandlers{
		hasuraClient:   hc,
		redisClient:    rc,
		temporalClient: tc,
		logger:         logger.WithField("handler", "health"),
	}
}

// HealthResponse is the response body for health checks
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    string    `json:"uptime,omitempty"`
}

// ReadinessResponse includes detailed readiness status
type ReadinessResponse struct {
	Status     string                     `json:"status"`
	Ready      bool                       `json:"ready"`
	Components map[string]ComponentStatus `json:"components"`
	Timestamp  time.Time                  `json:"timestamp"`
}

// ComponentStatus represents a component's readiness status
type ComponentStatus struct {
	Ready   bool   `json:"ready"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

var startTime = time.Now()

// Health handles GET /health
// Basic liveness probe - returns 200 if service is running
func (h *HealthHandlers) Health(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(startTime).String()

	resp := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC(),
		Uptime:    uptime,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)

	h.logger.WithField("uptime", uptime).Debug("Health check passed")
}

// Ready handles GET /ready
// Readiness probe - returns 200 only if all dependencies are ready
func (h *HealthHandlers) Ready(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	components := make(map[string]ComponentStatus)
	allReady := true

	// Check Hasura
	hasuraReady := true
	hasuraErr := ""
	if err := h.checkHasura(ctx); err != nil {
		hasuraReady = false
		hasuraErr = err.Error()
		allReady = false
	}
	components["hasura"] = ComponentStatus{
		Ready:   hasuraReady,
		Message: "GraphQL endpoint",
		Error:   hasuraErr,
	}

	// Check Redis
	redisReady := true
	redisErr := ""
	if err := h.checkRedis(ctx); err != nil {
		redisReady = false
		redisErr = err.Error()
		allReady = false
	}
	components["redis"] = ComponentStatus{
		Ready:   redisReady,
		Message: "Cache layer",
		Error:   redisErr,
	}

	// Check Temporal
	temporalReady := true
	temporalErr := ""
	if err := h.checkTemporal(ctx); err != nil {
		temporalReady = false
		temporalErr = err.Error()
		allReady = false
	}
	components["temporal"] = ComponentStatus{
		Ready:   temporalReady,
		Message: "Workflow orchestration",
		Error:   temporalErr,
	}

	status := "ready"
	statusCode := http.StatusOK
	if !allReady {
		status = "not-ready"
		statusCode = http.StatusServiceUnavailable
	}

	resp := ReadinessResponse{
		Status:     status,
		Ready:      allReady,
		Components: components,
		Timestamp:  time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)

	h.logger.WithFields(logrus.Fields{
		"ready":    allReady,
		"hasura":   hasuraReady,
		"redis":    redisReady,
		"temporal": temporalReady,
	}).Debug("Readiness check completed")
}

// CheckReady performs a readiness check and returns the response struct
func (h *HealthHandlers) CheckReady(ctx context.Context) ReadinessResponse {
	components := make(map[string]ComponentStatus)
	allReady := true

	// Check Hasura
	hasuraReady := true
	hasuraErr := ""
	if err := h.checkHasura(ctx); err != nil {
		hasuraReady = false
		hasuraErr = err.Error()
		allReady = false
	}
	components["hasura"] = ComponentStatus{
		Ready:   hasuraReady,
		Message: "GraphQL endpoint",
		Error:   hasuraErr,
	}

	// Check Redis
	redisReady := true
	redisErr := ""
	if err := h.checkRedis(ctx); err != nil {
		redisReady = false
		redisErr = err.Error()
		allReady = false
	}
	components["redis"] = ComponentStatus{
		Ready:   redisReady,
		Message: "Cache layer",
		Error:   redisErr,
	}

	// Check Temporal
	temporalReady := true
	temporalErr := ""
	if err := h.checkTemporal(ctx); err != nil {
		temporalReady = false
		temporalErr = err.Error()
		allReady = false
	}
	components["temporal"] = ComponentStatus{
		Ready:   temporalReady,
		Message: "Workflow orchestration",
		Error:   temporalErr,
	}

	status := "ready"
	if !allReady {
		status = "not-ready"
	}

	return ReadinessResponse{
		Status:     status,
		Ready:      allReady,
		Components: components,
		Timestamp:  time.Now().UTC(),
	}
}

// checkHasura verifies Hasura GraphQL endpoint connectivity
func (h *HealthHandlers) checkHasura(ctx context.Context) error {
	if h.hasuraClient == nil {
		return fmt.Errorf("hasura client not initialized")
	}

	var result struct {
		Typename string `json:"__typename"`
	}
	// Simple introspection query to verify connectivity
	if err := h.hasuraClient.QueryRaw(ctx, "query { __typename }", nil, &result); err != nil {
		return fmt.Errorf("hasura query failed: %w", err)
	}
	return nil
}

// checkRedis verifies Redis connectivity
func (h *HealthHandlers) checkRedis(ctx context.Context) error {
	if h.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return h.redisClient.Ping(ctx)
}

// checkTemporal verifies Temporal Server connectivity
func (h *HealthHandlers) checkTemporal(ctx context.Context) error {
	if h.temporalClient == nil {
		return fmt.Errorf("temporal client not initialized")
	}

	// Pulse check
	_, err := h.temporalClient.CheckHealth(ctx, &client.CheckHealthRequest{})
	if err != nil {
		return fmt.Errorf("temporal health check failed: %w", err)
	}
	return nil
}

// Ping handles GET /ping
// Ultra-lightweight liveness check
func (h *HealthHandlers) Ping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"portfolio-management/internal/backtest"
	"portfolio-management/internal/hierarchy"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// ============================================================================
// Global Services
// ============================================================================

var (
	backtestService  *backtest.Service
	hierarchyService hierarchy.Service
)

// ============================================================================
// Request/Response Types
// ============================================================================

type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Services  map[string]interface{} `json:"services"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// ============================================================================
// Main Application
// ============================================================================

func main() {
	// Database connection (sqlx) used by backtest and optionally by hierarchy
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	sqlxDb, err := sqlx.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to create sqlx connection: %v", err)
	}
	defer sqlxDb.Close()

	if err := sqlxDb.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("✓ Connected to database (sqlx)")

	// Initialize backtest service
	backtestService = backtest.NewService(sqlxDb)
	log.Println("✓ Initialized backtest service")

	// GORM connection (only needed if using GORM-backed hierarchy service)
	gormDb := InitDatabase()

	// Initialize hierarchy service: prefer sqlx implementation when enabled
	if os.Getenv("USE_SQLX_HIERARCHY") == "true" {
		hierarchyService = hierarchy.NewHierarchyServiceSQLX(sqlxDb)
		log.Println("✓ Initialized hierarchy service (sqlx)")
	} else {
		hierarchyService = hierarchy.NewHierarchyService(gormDb)
		log.Println("✓ Initialized hierarchy service (gorm)")
	}

	// Setup HTTP handlers
	setupRoutes()

	// Wrap the default mux with JWT middleware (skip health endpoint)
	jwtMw := jwtmiddleware.NewJWTMiddleware("/health")
	handler := jwtMw.Handler(http.DefaultServeMux)

	// Start HTTP server
	port := os.Getenv("PORTFOLIO_SERVICE_PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Starting portfolio management service on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}

// ============================================================================
// Route Setup
// ============================================================================

func setupRoutes() {
	// Health checks
	http.HandleFunc("/health", handleHealth)

	// Hierarchy Management
	http.HandleFunc("/api/hierarchy/validate", handleValidateHierarchy)
	http.HandleFunc("/api/hierarchy/rules", handleGetHierarchyRules)
	http.HandleFunc("/api/hierarchy/summary", handleGetHierarchySummary)
	http.HandleFunc("/api/hierarchy/tree", handleGetHierarchyTree)
	http.HandleFunc("/api/hierarchy/stats", handleGetHierarchyStats)
	http.HandleFunc("/api/hierarchy/import", handleImportHierarchy)

	// Portfolio Management
	http.HandleFunc("/api/portfolios", handlePortfolios)
	http.HandleFunc("/api/holdings", handleHoldings)
	http.HandleFunc("/api/portfolio-risk-metrics", handleRiskMetrics)

	// Recommendations
	http.HandleFunc("/api/recommendations", handleRecommendations)
	http.HandleFunc("/api/recommendation-status", handleRecommendationDetail)

	// Backtesting
	http.HandleFunc("/api/backtest/run", handleRunBacktest)
	http.HandleFunc("/api/backtest/results", handleGetBacktestResults)
	http.HandleFunc("/api/backtest/compare", handleCompareBacktests)
	http.HandleFunc("/api/backtest-detail", handleBacktestDetail)

	// Rebalancing
	http.HandleFunc("/api/rebalancing/plans", handleRebalancingPlans)
	http.HandleFunc("/api/rebalancing/suggest", handleSuggestRebalancing)
}

// ============================================================================
// Health Check Handlers
// ============================================================================

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services: map[string]interface{}{
			"portfolio_service": "operational",
			"backtest_service":  "operational",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// ============================================================================
// Portfolio Handlers
// ============================================================================

func handlePortfolios(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		handleCreatePortfolio(w, r)
	case http.MethodGet:
		handleListPortfolios(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleCreatePortfolio(w http.ResponseWriter, r *http.Request) {
	var req backtest.CreatePortfolioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// extract claims from context
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		respondWithError(w, http.StatusUnauthorized, "unauthorized", "")
		return
	}
	tenantID := claims.TenantID
	clientID := claims.UserID

	portfolio, err := backtestService.CreatePortfolio(r.Context(), req, tenantID, clientID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create portfolio", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(portfolio)
}

func handleListPortfolios(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"portfolios": []interface{}{},
		"count":      0,
	})
}

func handleHoldings(w http.ResponseWriter, r *http.Request) {
	portfolioID := r.URL.Query().Get("portfolio_id")
	if portfolioID == "" {
		respondWithError(w, http.StatusBadRequest, "Missing portfolio_id parameter", "")
		return
	}

	holdings, err := backtestService.GetHoldings(r.Context(), portfolioID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch holdings", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(holdings)
}

// ============================================================================
// Recommendation Handlers
// ============================================================================

func handleRecommendations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		handleCreateRecommendation(w, r)
	case http.MethodGet:
		respondWithError(w, http.StatusNotImplemented, "Not implemented", "")
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleCreateRecommendation(w http.ResponseWriter, r *http.Request) {
	var req backtest.CreateRecommendationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	portfolioID := r.URL.Query().Get("portfolio_id")
	userID := r.Header.Get("X-User-ID")

	if portfolioID == "" || userID == "" {
		respondWithError(w, http.StatusBadRequest, "Missing portfolio_id or X-User-ID", "")
		return
	}

	rec, err := backtestService.CreateRecommendation(r.Context(), portfolioID, userID, req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create recommendation", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rec)
}

func handleRecommendationDetail(w http.ResponseWriter, r *http.Request) {
	recID := r.URL.Query().Get("id")
	if recID == "" {
		respondWithError(w, http.StatusBadRequest, "Missing id parameter", "")
		return
	}

	switch r.Method {
	case http.MethodGet:
		rec, err := backtestService.GetRecommendation(r.Context(), recID)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "Recommendation not found", err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rec)

	case http.MethodPatch:
		var req backtest.UpdateRecommendationStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request", err.Error())
			return
		}
		err := backtestService.UpdateRecommendationStatus(r.Context(), recID, req.Status, req.Notes)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Update failed", err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ============================================================================
// Backtest Handlers
// ============================================================================

func handleRunBacktest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req backtest.BacktestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// Validate required fields
	if req.RecommendationID == "" || req.PortfolioID == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required fields", "recommendation_id and portfolio_id required")
		return
	}

	// Set default date range
	if req.EndDate.IsZero() {
		req.EndDate = time.Now()
	}
	if req.StartDate.IsZero() {
		req.StartDate = req.EndDate.AddDate(-1, 0, 0)
	}
	if req.SimulationDays == 0 {
		req.SimulationDays = 252 // Trading days
	}

	// Run backtest
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result, err := backtestService.RunBacktest(ctx, req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Backtest failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

func handleGetBacktestResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	portfolioID := r.URL.Query().Get("portfolio_id")
	if portfolioID == "" {
		respondWithError(w, http.StatusBadRequest, "Missing portfolio_id", "")
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	results, err := backtestService.GetBacktestResults(r.Context(), portfolioID, limit)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch results", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func handleBacktestDetail(w http.ResponseWriter, r *http.Request) {
	backtestID := r.URL.Query().Get("id")
	if backtestID == "" {
		respondWithError(w, http.StatusBadRequest, "Missing id parameter", "")
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	result, err := backtestService.GetBacktestByID(r.Context(), backtestID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Backtest not found", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleCompareBacktests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req backtest.ComparisonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// Validate
	if req.PortfolioID == "" || req.RecommendationID1 == "" || req.RecommendationID2 == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required fields", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	comparison, err := backtestService.CompareBacktests(ctx, req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Comparison failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comparison)
}

// ============================================================================
// Risk Metrics Handler
// ============================================================================

func handleRiskMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	portfolioID := r.URL.Query().Get("portfolio_id")
	if portfolioID == "" {
		respondWithError(w, http.StatusBadRequest, "Missing portfolio_id", "")
		return
	}

	metrics, err := backtestService.GetRiskMetrics(r.Context(), portfolioID)
	if err != nil {
		// Return calculated metrics if not cached
		metrics, err = backtestService.CalculateRiskMetrics(r.Context(), portfolioID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to calculate metrics", err.Error())
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// ============================================================================
// Rebalancing Handlers
// ============================================================================

func handleRebalancingPlans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plans": []interface{}{},
	})
}

func handleSuggestRebalancing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	portfolioID := r.URL.Query().Get("portfolio_id")
	if portfolioID == "" {
		respondWithError(w, http.StatusBadRequest, "Missing portfolio_id", "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "analyzing",
		"portfolio_id": portfolioID,
	})
}

// ============================================================================
// Utility Functions
// ============================================================================

func respondWithError(w http.ResponseWriter, code int, message, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	resp := ErrorResponse{
		Error:   message,
		Message: details,
		Code:    code,
	}
	json.NewEncoder(w).Encode(resp)
}

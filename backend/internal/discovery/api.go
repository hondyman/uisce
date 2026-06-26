package discovery

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"

	"github.com/go-chi/chi/v5"
)

// DiscoveryHandler manages HTTP requests for feature discovery
type DiscoveryHandler struct {
	db     *sql.DB
	logger *log.Logger
}

// NewDiscoveryHandler creates a new discovery API handler
func NewDiscoveryHandler(db *sql.DB, logger *log.Logger) *DiscoveryHandler {
	return &DiscoveryHandler{
		db:     db,
		logger: logger,
	}
}

// Request/Response types
type StartDiscoveryRequest struct {
	DatabaseType   string             `json:"database_type"` // "auto" or specific type
	ScanInterval   int                `json:"scan_interval_hours"`
	ScoringWeights map[string]float64 `json:"scoring_weights"`
	UseCase        string             `json:"use_case"` // "forecasting", "classification", etc
}

type DiscoveryRunResponse struct {
	RunID           string     `json:"run_id"`
	Status          string     `json:"status"` // "pending", "running", "complete", "failed"
	StartedAt       time.Time  `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CandidatesFound int        `json:"candidates_found"`
	SourcesScanned  []string   `json:"sources_scanned"`
	Error           string     `json:"error,omitempty"`
}

type CandidateListResponse struct {
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	Candidates []CandidateResponse `json:"candidates"`
}

type CandidateResponse struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	SourceDatabase  string    `json:"source_database"`
	SourceField     string    `json:"source_field"`
	DataType        string    `json:"data_type"`
	Completeness    float64   `json:"completeness"`
	Cardinality     int64     `json:"cardinality"`
	Score           float64   `json:"score"`
	Status          string    `json:"status"`                     // candidate, approved, rejected
	RejectionReason string    `json:"rejection_reason,omitempty"` // Reason for rejection if applicable
	DiscoveredAt    time.Time `json:"discovered_at"`
	Rationale       string    `json:"rationale"` // Why was this scored high?
}

type ApproveRequest struct {
	CandidateID string `json:"candidate_id"`
	FeatureName string `json:"feature_name"` // Optional override
	Notes       string `json:"notes"`
}

type RejectRequest struct {
	CandidateID string `json:"candidate_id"`
	Reason      string `json:"reason"`
}

type DiscoveryStats struct {
	TotalCandidates      int              `json:"total_candidates"`
	ApprovedCount        int              `json:"approved_count"`
	RejectedCount        int              `json:"rejected_count"`
	SourceDistribution   map[string]int   `json:"source_distribution"`
	DataTypeDistribution map[string]int   `json:"data_type_distribution"`
	ScoreDistribution    map[string]int   `json:"score_distribution"` // buckets: 0-0.2, 0.2-0.4, etc
	AvgScore             float64          `json:"avg_score"`
	MedianScore          float64          `json:"median_score"`
	LastDiscoveryRun     DiscoveryRunInfo `json:"last_discovery_run"`
}

type DiscoveryRunInfo struct {
	RunID           string    `json:"run_id"`
	CompletedAt     time.Time `json:"completed_at"`
	Duration        float64   `json:"duration_seconds"`
	CandidatesFound int       `json:"candidates_found"`
}

// RegisterRoutes registers discovery API endpoints
func (dh *DiscoveryHandler) RegisterRoutes(router *chi.Mux) {
	router.Route("/api/v3/discovery", func(r chi.Router) {
		r.Post("/start", dh.StartDiscovery)
		r.Get("/runs/{runid}", dh.GetDiscoveryRun)
		r.Get("/candidates", dh.ListCandidates)
		r.Get("/candidates/{candidateid}", dh.GetCandidate)
		r.Post("/approve", dh.ApproveCandidate)
		r.Post("/reject", dh.RejectCandidate)
		r.Get("/stats", dh.GetDiscoveryStats)
		r.Get("/search", dh.SearchCandidates)
	})
}

// StartDiscovery triggers a new discovery run
func (dh *DiscoveryHandler) StartDiscovery(w http.ResponseWriter, r *http.Request) {
	var req StartDiscoveryRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		dh.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.ScanInterval == 0 {
		req.ScanInterval = 24 // Default: daily
	}
	if req.UseCase == "" {
		req.UseCase = "forecasting" // Default use case
	}

	// Generate run ID
	runID := generateRunID()

	// Create discovery config
	config := models.DiscoveryConfig{
		ScanInterval:      time.Duration(req.ScanInterval) * time.Hour,
		PostgresDatabases: []string{"semlayer", "analytics"},
		TrinoDatabases:    []string{"warehouse"},
		PrometheusURL:     "http://localhost:9090",
		ScoringWeights:    req.ScoringWeights,
	}

	// Insert discovery run record
	now := time.Now()
	err := dh.db.QueryRowContext(
		r.Context(),
		`INSERT INTO discovery_runs (run_id, status, started_at, sources_scanned)
		 VALUES ($1, $2, $3, $4)
		 RETURNING run_id`,
		runID, "pending", now, `[]`,
	).Scan(&runID)

	if err != nil {
		dh.logger.Printf("Failed to insert discovery run: %v", err)
		dh.respondError(w, http.StatusInternalServerError, "Failed to start discovery")
		return
	}

	// Trigger discovery workflow asynchronously
	// In real implementation, would use Temporal client
	go dh.executeDiscovery(runID, config)

	response := DiscoveryRunResponse{
		RunID:     runID,
		Status:    "pending",
		StartedAt: now,
		Error:     "",
	}

	dh.respondJSON(w, http.StatusCreated, response)
}

// GetDiscoveryRun retrieves status of a specific discovery run
func (dh *DiscoveryHandler) GetDiscoveryRun(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runid")

	var (
		status          string
		startedAt       time.Time
		completedAt     *time.Time
		candidatesFound int
		sourceScanned   string
		errorMsg        string
	)

	err := dh.db.QueryRowContext(
		r.Context(),
		`SELECT status, started_at, completed_at, candidates_found, sources_scanned, error_message
		 FROM discovery_runs WHERE run_id = $1`,
		runID,
	).Scan(&status, &startedAt, &completedAt, &candidatesFound, &sourceScanned, &errorMsg)

	if err == sql.ErrNoRows {
		dh.respondError(w, http.StatusNotFound, "Discovery run not found")
		return
	}
	if err != nil {
		dh.respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Parse sources
	var sources []string
	if sourceScanned != "" {
		_ = json.Unmarshal([]byte(sourceScanned), &sources)
	}

	response := DiscoveryRunResponse{
		RunID:           runID,
		Status:          status,
		StartedAt:       startedAt,
		CompletedAt:     completedAt,
		CandidatesFound: candidatesFound,
		SourcesScanned:  sources,
		Error:           errorMsg,
	}

	dh.respondJSON(w, http.StatusOK, response)
}

// ListCandidates returns paginated list of discovered candidates
func (dh *DiscoveryHandler) ListCandidates(w http.ResponseWriter, r *http.Request) {
	// Query parameters
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	status := r.URL.Query().Get("status")      // candidate, approved, rejected
	sourceDB := r.URL.Query().Get("source_db") // postgres, trino, logs, prometheus
	minScore := r.URL.Query().Get("min_score") // 0.0-1.0
	sortBy := r.URL.Query().Get("sort_by")     // "score", "name", "discovered_at"

	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	pageSize := 20
	if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
		if ps > 100 {
			pageSize = 100
		} else {
			pageSize = ps
		}
	}

	// Build query
	query := `SELECT id, name, source_database, source_field, data_type, 
	          completeness, cardinality, business_value, status, discovered_at
			  FROM discovery_candidates WHERE 1=1`

	args := []interface{}{}
	argIndex := 1

	if status != "" {
		query += fmt.Sprintf(` AND status = $%d`, argIndex)
		args = append(args, status)
		argIndex++
	}

	if sourceDB != "" {
		query += fmt.Sprintf(` AND source_database = $%d`, argIndex)
		args = append(args, sourceDB)
		argIndex++
	}

	if minScore != "" {
		if score, err := strconv.ParseFloat(minScore, 64); err == nil {
			query += fmt.Sprintf(` AND business_value >= $%d`, argIndex)
			args = append(args, score)
			argIndex++
		}
	}

	// Sorting
	switch sortBy {
	case "name":
		query += " ORDER BY name ASC"
	case "discovered_at":
		query += " ORDER BY discovered_at DESC"
	default:
		query += " ORDER BY business_value DESC"
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM discovery_candidates WHERE 1=1"
	for i := range args[:min(argIndex-1, len(args))] {
		if i > 0 {
			countQuery += fmt.Sprintf(" AND source_database = $%d", i+1)
		}
	}

	var total int
	_ = dh.db.QueryRowContext(r.Context(), countQuery, args...).Scan(&total)

	// Add pagination
	offset := (page - 1) * pageSize
	query += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	rows, err := dh.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		dh.respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	candidates := []CandidateResponse{}
	for rows.Next() {
		var c CandidateResponse
		err := rows.Scan(
			&c.ID, &c.Name, &c.SourceDatabase, &c.SourceField,
			&c.DataType, &c.Completeness, &c.Cardinality,
			&c.Score, &c.Status, &c.DiscoveredAt,
		)
		if err != nil {
			continue
		}
		c.Rationale = dh.generateRationale(c.Score, c.SourceDatabase, c.DataType)
		candidates = append(candidates, c)
	}

	response := CandidateListResponse{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		Candidates: candidates,
	}

	dh.respondJSON(w, http.StatusOK, response)
}

// GetCandidate returns details of a specific candidate
func (dh *DiscoveryHandler) GetCandidate(w http.ResponseWriter, r *http.Request) {
	candidateID := chi.URLParam(r, "candidateid")

	var c CandidateResponse
	err := dh.db.QueryRowContext(
		r.Context(),
		`SELECT id, name, source_database, source_field, data_type,
		        completeness, cardinality, business_value, status, discovered_at
		 FROM discovery_candidates WHERE id = $1`,
		candidateID,
	).Scan(&c.ID, &c.Name, &c.SourceDatabase, &c.SourceField,
		&c.DataType, &c.Completeness, &c.Cardinality,
		&c.Score, &c.Status, &c.DiscoveredAt)

	if err == sql.ErrNoRows {
		dh.respondError(w, http.StatusNotFound, "Candidate not found")
		return
	}
	if err != nil {
		dh.respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	c.Rationale = dh.generateRationale(c.Score, c.SourceDatabase, c.DataType)

	dh.respondJSON(w, http.StatusOK, c)
}

// ApproveCandidate moves a candidate to the feature catalog
func (dh *DiscoveryHandler) ApproveCandidate(w http.ResponseWriter, r *http.Request) {
	var req ApproveRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		dh.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get candidate details
	var name, dataType string
	err := dh.db.QueryRowContext(
		r.Context(),
		`SELECT name, data_type FROM discovery_candidates WHERE id = $1`,
		req.CandidateID,
	).Scan(&name, &dataType)

	if err == sql.ErrNoRows {
		dh.respondError(w, http.StatusNotFound, "Candidate not found")
		return
	}
	if err != nil {
		dh.respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Use provided name or discovered name
	featureName := req.FeatureName
	if featureName == "" {
		featureName = name
	}

	// Update status
	_, err = dh.db.ExecContext(
		r.Context(),
		`UPDATE discovery_candidates SET status = $1, approved_by = $2, approved_at = $3, notes = $4
		 WHERE id = $5`,
		"approved", r.Header.Get("X-User-ID"), time.Now(), req.Notes, req.CandidateID,
	)

	if err != nil {
		dh.respondError(w, http.StatusInternalServerError, "Failed to approve candidate")
		return
	}

	// TODO: Add to feature catalog
	dh.logger.Printf("Candidate %s approved as feature %s", req.CandidateID, featureName)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "approved",
		"feature": featureName,
		"message": "Candidate approved and added to feature catalog",
	})
}

// RejectCandidate marks a candidate as rejected
func (dh *DiscoveryHandler) RejectCandidate(w http.ResponseWriter, r *http.Request) {
	var req RejectRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		dh.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	_, err := dh.db.ExecContext(
		r.Context(),
		`UPDATE discovery_candidates SET status = $1, rejection_reason = $2
		 WHERE id = $3`,
		"rejected", req.Reason, req.CandidateID,
	)

	if err != nil {
		dh.respondError(w, http.StatusInternalServerError, "Failed to reject candidate")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "rejected",
		"reason": req.Reason,
	})
}

// GetDiscoveryStats returns aggregate statistics
func (dh *DiscoveryHandler) GetDiscoveryStats(w http.ResponseWriter, r *http.Request) {
	var totalCount, approvedCount, rejectedCount int

	// Total count
	_ = dh.db.QueryRowContext(
		r.Context(),
		`SELECT COUNT(*) FROM discovery_candidates`,
	).Scan(&totalCount)

	// Approved
	_ = dh.db.QueryRowContext(
		r.Context(),
		`SELECT COUNT(*) FROM discovery_candidates WHERE status = $1`,
		"approved",
	).Scan(&approvedCount)

	// Rejected
	_ = dh.db.QueryRowContext(
		r.Context(),
		`SELECT COUNT(*) FROM discovery_candidates WHERE status = $1`,
		"rejected",
	).Scan(&rejectedCount)

	// Source distribution
	sourceDistribution := make(map[string]int)
	sourceRows, err := dh.db.QueryContext(
		r.Context(),
		`SELECT source_database, COUNT(*) FROM discovery_candidates GROUP BY source_database`,
	)
	if err != nil {
		dh.logger.Printf("Failed to query source distribution: %v", err)
	} else {
		defer sourceRows.Close()

		for sourceRows.Next() {
			var source string
			var count int
			sourceRows.Scan(&source, &count)
			sourceDistribution[source] = count
		}
	}

	// Data type distribution
	typeDistribution := make(map[string]int)
	typeRows, err := dh.db.QueryContext(
		r.Context(),
		`SELECT data_type, COUNT(*) FROM discovery_candidates GROUP BY data_type`,
	)
	if err != nil {
		dh.logger.Printf("Failed to query type distribution: %v", err)
	} else {
		defer typeRows.Close()

		for typeRows.Next() {
			var dtype string
			var count int
			typeRows.Scan(&dtype, &count)
			typeDistribution[dtype] = count
		}
	}

	// Score distribution (buckets: 0-0.2, 0.2-0.4, etc)
	scoreDistribution := map[string]int{
		"0.0-0.2": 0,
		"0.2-0.4": 0,
		"0.4-0.6": 0,
		"0.6-0.8": 0,
		"0.8-1.0": 0,
	}

	scoreRows, err := dh.db.QueryContext(
		r.Context(),
		`SELECT FLOOR(business_value * 5) * 0.2 as bucket, COUNT(*) FROM discovery_candidates GROUP BY bucket`,
	)
	if err != nil {
		dh.logger.Printf("Failed to query score distribution: %v", err)
	} else {
		defer scoreRows.Close()

		for scoreRows.Next() {
			var bucket float64
			var count int
			scoreRows.Scan(&bucket, &count)
			// Map bucket to label
			if bucket >= 0.8 {
				scoreDistribution["0.8-1.0"] += count
			} else if bucket >= 0.6 {
				scoreDistribution["0.6-0.8"] += count
			} else if bucket >= 0.4 {
				scoreDistribution["0.4-0.6"] += count
			} else if bucket >= 0.2 {
				scoreDistribution["0.2-0.4"] += count
			} else {
				scoreDistribution["0.0-0.2"] += count
			}
		}
	}

	// Average and median score
	var avgScore, medianScore float64
	_ = dh.db.QueryRowContext(
		r.Context(),
		`SELECT AVG(business_value), PERCENTILE_CONT(0.5) WITHIN GROUP(ORDER BY business_value)
		 FROM discovery_candidates`,
	).Scan(&avgScore, &medianScore)

	// Last discovery run
	var lastRunID string
	var completedAt time.Time
	var duration float64
	var candidatesFound int

	_ = dh.db.QueryRowContext(
		r.Context(),
		`SELECT run_id, completed_at, 
		        EXTRACT(EPOCH FROM (completed_at - started_at)), candidates_found
		 FROM discovery_runs WHERE status = $1 ORDER BY completed_at DESC LIMIT 1`,
		"success",
	).Scan(&lastRunID, &completedAt, &duration, &candidatesFound)

	stats := DiscoveryStats{
		TotalCandidates:      totalCount,
		ApprovedCount:        approvedCount,
		RejectedCount:        rejectedCount,
		SourceDistribution:   sourceDistribution,
		DataTypeDistribution: typeDistribution,
		ScoreDistribution:    scoreDistribution,
		AvgScore:             avgScore,
		MedianScore:          medianScore,
		LastDiscoveryRun: DiscoveryRunInfo{
			RunID:           lastRunID,
			CompletedAt:     completedAt,
			Duration:        duration,
			CandidatesFound: candidatesFound,
		},
	}

	dh.respondJSON(w, http.StatusOK, stats)
}

// SearchCandidates performs full-text search on candidates
func (dh *DiscoveryHandler) SearchCandidates(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

	if len(q) < 2 {
		dh.respondError(w, http.StatusBadRequest, "Search query must be at least 2 characters")
		return
	}

	// Escape for LIKE query
	q = "%" + strings.ReplaceAll(q, "%", "\\%") + "%"

	rows, err := dh.db.QueryContext(
		r.Context(),
		`SELECT id, name, source_database, source_field, data_type,
		        completeness, cardinality, business_value, status, discovered_at
		 FROM discovery_candidates WHERE name ILIKE $1 OR source_field ILIKE $1
		 ORDER BY business_value DESC LIMIT 50`,
		q,
	)

	if err != nil {
		dh.respondError(w, http.StatusInternalServerError, "Search failed")
		return
	}
	defer rows.Close()

	candidates := []CandidateResponse{}
	for rows.Next() {
		var c CandidateResponse
		err := rows.Scan(
			&c.ID, &c.Name, &c.SourceDatabase, &c.SourceField,
			&c.DataType, &c.Completeness, &c.Cardinality,
			&c.Score, &c.Status, &c.DiscoveredAt,
		)
		if err != nil {
			continue
		}
		c.Rationale = dh.generateRationale(c.Score, c.SourceDatabase, c.DataType)
		candidates = append(candidates, c)
	}

	dh.respondJSON(w, http.StatusOK, map[string]interface{}{
		"query":      r.URL.Query().Get("q"),
		"results":    len(candidates),
		"candidates": candidates,
	})
}

// Helper functions

func (dh *DiscoveryHandler) executeDiscovery(runID string, config models.DiscoveryConfig) {
	// In real implementation, would:
	// 1. Connect to Temporal client
	// 2. Start DiscoveryWorkflow execution
	// 3. Await completion
	// 4. Update discovery_runs table with results

	dh.logger.Printf("Starting discovery workflow for run %s", runID)

	// Simulate execution
	time.Sleep(2 * time.Second)

	completedAt := time.Now()
	_, err := dh.db.Exec(
		`UPDATE discovery_runs SET status = $1, completed_at = $2, candidates_found = $3
		 WHERE run_id = $4`,
		"success", completedAt, 127, runID,
	)

	if err != nil {
		dh.logger.Printf("Failed to update discovery run %s: %v", runID, err)
	}

	dh.logger.Printf("Discovery workflow completed for run %s", runID)
}

func (dh *DiscoveryHandler) generateRationale(score float64, sourceDB string, dataType string) string {
	reasons := []string{}

	// Score-based reasoning
	if score >= 0.8 {
		reasons = append(reasons, "high predictive value")
	} else if score >= 0.6 {
		reasons = append(reasons, "moderate predictive value")
	}

	// Source-based reasoning
	switch sourceDB {
	case "postgres", "trino":
		reasons = append(reasons, "structured source")
	case "logs":
		reasons = append(reasons, "extracted from logs")
	case "prometheus":
		reasons = append(reasons, "operational metric")
	}

	// Type-based reasoning
	if dataType == "float" || dataType == "number" {
		reasons = append(reasons, "numeric (good for ML)")
	} else if dataType == "categorical" || dataType == "string" {
		reasons = append(reasons, "categorical (requires encoding)")
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "candidate feature")
	}

	return strings.Join(reasons, "; ")
}

func (dh *DiscoveryHandler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (dh *DiscoveryHandler) respondError(w http.ResponseWriter, statusCode int, message string) {
	dh.respondJSON(w, statusCode, map[string]string{
		"error": message,
	})
}

func generateRunID() string {
	return fmt.Sprintf("discovery-%d", time.Now().UnixMicro())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

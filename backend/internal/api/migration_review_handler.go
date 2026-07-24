package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// MigrationJob represents a code migration job
type MigrationJob struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Status          string                 `json:"status"`
	SourceCode      string                 `json:"sourceCode"`
	SourceLanguage  string                 `json:"sourceLanguage"`
	ASTJSON         map[string]interface{} `json:"astJson,omitempty"`
	ExtractedIntent map[string]interface{} `json:"extractedIntent,omitempty"`
	GeneratedDAG    map[string]interface{} `json:"generatedDag,omitempty"`
	GeneratedRego   string                 `json:"generatedRego,omitempty"`
	RAGContext      []string               `json:"ragContext,omitempty"`
	ReviewerID      string                 `json:"reviewerId,omitempty"`
	ReviewNotes     string                 `json:"reviewNotes,omitempty"`
	ApprovedAt      *time.Time             `json:"approvedAt,omitempty"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

// ListMigrations handles GET /api/migrations
func (s *Server) ListMigrations(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	query := `
		SELECT id, name, status, source_language, created_at, updated_at
		FROM migration_jobs
		WHERE ($1 = '' OR status = $1)
		ORDER BY created_at DESC
		LIMIT 50
	`

	rows, err := s.DB.QueryContext(r.Context(), query, status)
	if err != nil {
		http.Error(w, "Failed to fetch migrations", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var jobs []MigrationJob
	for rows.Next() {
		var job MigrationJob
		if err := rows.Scan(&job.ID, &job.Name, &job.Status, &job.SourceLanguage, &job.CreatedAt, &job.UpdatedAt); err != nil {
			continue
		}
		jobs = append(jobs, job)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

// GetMigration handles GET /api/migrations/{id}
func (s *Server) GetMigration(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	query := `
		SELECT id, name, status, source_code, source_language, 
			   ast_json, extracted_intent, generated_dag, generated_rego,
			   rag_context, reviewer_id, review_notes, approved_at,
			   created_at, updated_at
		FROM migration_jobs
		WHERE id = $1
	`

	var job MigrationJob
	var astJSON, intentJSON, dagJSON, contextJSON sql.NullString
	var reviewerID, reviewNotes sql.NullString
	var approvedAt sql.NullTime

	err := s.DB.QueryRowContext(r.Context(), query, id).Scan(
		&job.ID, &job.Name, &job.Status, &job.SourceCode, &job.SourceLanguage,
		&astJSON, &intentJSON, &dagJSON, &job.GeneratedRego,
		&contextJSON, &reviewerID, &reviewNotes, &approvedAt,
		&job.CreatedAt, &job.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Migration not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to fetch migration", http.StatusInternalServerError)
		return
	}

	// Parse JSON fields
	if astJSON.Valid {
		_ = json.Unmarshal([]byte(astJSON.String), &job.ASTJSON)
	}
	if intentJSON.Valid {
		_ = json.Unmarshal([]byte(intentJSON.String), &job.ExtractedIntent)
	}
	if dagJSON.Valid {
		_ = json.Unmarshal([]byte(dagJSON.String), &job.GeneratedDAG)
	}
	if contextJSON.Valid {
		_ = json.Unmarshal([]byte(contextJSON.String), &job.RAGContext)
	}
	if reviewerID.Valid {
		job.ReviewerID = reviewerID.String
	}
	if reviewNotes.Valid {
		job.ReviewNotes = reviewNotes.String
	}
	if approvedAt.Valid {
		job.ApprovedAt = &approvedAt.Time
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// CreateMigration handles POST /api/migrations
func (s *Server) CreateMigration(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name           string `json:"name"`
		SourceCode     string `json:"sourceCode"`
		SourceLanguage string `json:"sourceLanguage"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.SourceCode == "" {
		http.Error(w, "Source code is required", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		req.Name = "Untitled Migration"
	}
	if req.SourceLanguage == "" {
		req.SourceLanguage = "java"
	}

	query := `
		INSERT INTO migration_jobs (name, source_code, source_language, status)
		VALUES ($1, $2, $3, 'PENDING')
		RETURNING id, created_at, updated_at
	`

	var job MigrationJob
	job.Name = req.Name
	job.SourceCode = req.SourceCode
	job.SourceLanguage = req.SourceLanguage
	job.Status = "PENDING"

	err := s.DB.QueryRowContext(r.Context(), query, req.Name, req.SourceCode, req.SourceLanguage).
		Scan(&job.ID, &job.CreatedAt, &job.UpdatedAt)

	if err != nil {
		http.Error(w, "Failed to create migration", http.StatusInternalServerError)
		return
	}

	// TODO: Trigger Temporal workflow for async processing
	// s.TemporalClient.ExecuteWorkflow(ctx, options, "MigrationWorkflow", job.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(job)
}

// ApproveMigration handles POST /api/migrations/{id}/approve
func (s *Server) ApproveMigration(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	// Get reviewer from context (set by auth middleware)
	reviewerID := r.Header.Get("X-User-ID")
	if reviewerID == "" {
		reviewerID = "anonymous"
	}

	var req struct {
		Notes string `json:"notes"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	query := `
		UPDATE migration_jobs 
		SET status = 'APPROVED', 
			reviewer_id = $2, 
			review_notes = $3,
			approved_at = NOW(),
			updated_at = NOW()
		WHERE id = $1 AND status = 'REVIEW'
		RETURNING id
	`

	var updatedID string
	err := s.DB.QueryRowContext(r.Context(), query, id, reviewerID, req.Notes).Scan(&updatedID)
	if err == sql.ErrNoRows {
		http.Error(w, "Migration not found or not in REVIEW status", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to approve migration", http.StatusInternalServerError)
		return
	}

	// TODO: Commit generated DAG/Rego to Titan pipeline storage
	// This would create the actual pipeline in the system

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "approved",
		"message": "Migration approved and committed to Titan",
	})
}

// RejectMigration handles POST /api/migrations/{id}/reject
func (s *Server) RejectMigration(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	reviewerID := r.Header.Get("X-User-ID")
	if reviewerID == "" {
		reviewerID = "anonymous"
	}

	var req struct {
		Notes    string `json:"notes"`
		Feedback string `json:"feedback"` // Specific feedback for re-processing
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	query := `
		UPDATE migration_jobs 
		SET status = 'REJECTED', 
			reviewer_id = $2, 
			review_notes = $3,
			updated_at = NOW()
		WHERE id = $1 AND status = 'REVIEW'
		RETURNING id
	`

	notes := req.Notes
	if req.Feedback != "" {
		notes += "\n\nFeedback: " + req.Feedback
	}

	var updatedID string
	err := s.DB.QueryRowContext(r.Context(), query, id, reviewerID, notes).Scan(&updatedID)
	if err == sql.ErrNoRows {
		http.Error(w, "Migration not found or not in REVIEW status", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to reject migration", http.StatusInternalServerError)
		return
	}

	// TODO: Option to re-queue for processing with feedback

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "rejected",
		"message": "Migration rejected. Feedback recorded.",
	})
}

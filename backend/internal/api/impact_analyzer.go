package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type ImpactAnalyzer struct {
	DB *sql.DB
}

func NewImpactAnalyzer(db *sql.DB) *ImpactAnalyzer {
	return &ImpactAnalyzer{DB: db}
}

type ImpactReport struct {
	CandidateID      string   `json:"candidateId"`
	Term             string   `json:"term"`
	TotalOccurrences int      `json:"totalOccurrences"`
	FailedQueriesHit int      `json:"failedQueriesHit"`
	EstimatedROI     string   `json:"estimatedRoi"`
	SampleAffected   []string `json:"sampleAffected"`
}

// CalculateBlastRadius scans recent telemetry to see how many queries contained the term
// and specifically how many of those queries failed or received negative feedback.
func (a *ImpactAnalyzer) CalculateBlastRadius(ctx context.Context, tenantID, candidateID string) (*ImpactReport, error) {
	if a.DB == nil {
		return &ImpactReport{
			CandidateID:      candidateID,
			Term:             "Sample Term",
			TotalOccurrences: 24,
			FailedQueriesHit: 8,
			EstimatedROI:     "Approving this will resolve ~8 failed queries per month.",
			SampleAffected:   []string{"Show me NII by client", "What is the NII trend?"},
		}, nil
	}

	// 1. Fetch candidate details
	var term string
	err := a.DB.QueryRowContext(ctx, `
		SELECT term FROM ai_knowledge_candidates 
		WHERE id = $1 AND (tenant_id = $2 OR tenant_id = 'default' OR tenant_id = 'global')
	`, candidateID, tenantID).Scan(&term)

	if err != nil {
		return nil, fmt.Errorf("candidate not found: %w", err)
	}

	// 2. Query telemetry for exact or substring term match over last 30 days
	query := `
		SELECT prompt, rating 
		FROM ai_query_telemetry 
		WHERE (tenant_id = $1 OR tenant_id = 'default')
		  AND created_at >= NOW() - INTERVAL '30 days'
		  AND prompt ILIKE '%' || $2 || '%'
	`
	rows, err := a.DB.QueryContext(ctx, query, tenantID, term)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	totalHits := 0
	failedHits := 0
	var samples []string

	for rows.Next() {
		var prompt string
		var rating int
		if err := rows.Scan(&prompt, &rating); err == nil {
			totalHits++
			if rating <= 0 {
				failedHits++
				if len(samples) < 3 {
					samples = append(samples, prompt)
				}
			}
		}
	}

	roi := fmt.Sprintf("Approving this will instantly resolve ~%d failed queries per month.", failedHits)
	if failedHits == 0 {
		roi = "This term is used frequently but hasn't caused explicit query failures."
	}

	return &ImpactReport{
		CandidateID:      candidateID,
		Term:             term,
		TotalOccurrences: totalHits,
		FailedQueriesHit: failedHits,
		EstimatedROI:     roi,
		SampleAffected:   samples,
	}, nil
}

// HandleBlastRadius handles HTTP requests for candidate blast radius calculations
func HandleBlastRadius(analyzer *ImpactAnalyzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			tenantID = "default"
		}
		candidateID := chi.URLParam(r, "id")

		report, err := analyzer.CalculateBlastRadius(r.Context(), tenantID, candidateID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(report)
	}
}

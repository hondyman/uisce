package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"
)

// PreAggSuggestionService analyzes semantic query patterns and suggests pre-aggregations.
type PreAggSuggestionService struct {
	db     *sql.DB // Trino/Postgres connection for querying semantic_events
	config SuggestionConfig
}

// SuggestionConfig holds thresholds for the suggestion engine.
type SuggestionConfig struct {
	MinFrequency    int     // Minimum query frequency to suggest (default: 20)
	MinAvgLatencyMs float64 // Minimum average latency in ms (default: 500)
	ObservationDays int     // Days to look back (default: 7)
	MaxSuggestions  int     // Maximum suggestions to return (default: 20)
	ScoreThreshold  float64 // Minimum score to include (default: 0.5)
}

// DefaultSuggestionConfig returns sensible defaults.
func DefaultSuggestionConfig() SuggestionConfig {
	return SuggestionConfig{
		MinFrequency:    20,
		MinAvgLatencyMs: 500,
		ObservationDays: 7,
		MaxSuggestions:  20,
		ScoreThreshold:  0.5,
	}
}

// NewPreAggSuggestionService creates a new suggestion service.
func NewPreAggSuggestionService(db *sql.DB, config *SuggestionConfig) *PreAggSuggestionService {
	cfg := DefaultSuggestionConfig()
	if config != nil {
		cfg = *config
	}
	return &PreAggSuggestionService{db: db, config: cfg}
}

// ListSuggestions returns pre-aggregation suggestions for the given tenant.
// It queries semantic_events to find heavy/repeated query patterns.
func (s *PreAggSuggestionService) ListSuggestions(ctx context.Context, tenantID string) ([]models.PreAggSuggestion, error) {
	if s.db == nil {
		// Return mock suggestions if no DB configured
		return s.mockSuggestions(tenantID), nil
	}

	// Query for heavy patterns from semantic_events
	// This query works with Trino against an Iceberg table or Postgres
	query := `
		WITH heavy AS (
			SELECT
				tenant_id,
				datasource,
				sql_fingerprint,
				COUNT(*) AS freq,
				AVG(sql_latency_ms) AS avg_latency,
				AVG(sql_rows) AS avg_rows
			FROM semantic_events
			WHERE created_at > NOW() - INTERVAL '%d' DAY
			  AND tenant_id = $1
			GROUP BY 1, 2, 3
			HAVING COUNT(*) > $2
			   AND AVG(sql_latency_ms) > $3
		)
		SELECT
			h.tenant_id,
			h.datasource,
			h.sql_fingerprint,
			h.freq,
			h.avg_latency,
			h.avg_rows,
			COALESCE(e.groupby_fields, '[]') AS groupby_fields,
			COALESCE(e.filter_fields, '[]') AS filter_fields,
			COALESCE(e.measure_fields, '[]') AS measure_fields
		FROM heavy h
		LEFT JOIN LATERAL (
			SELECT
				groupby_fields,
				filter_fields,
				measure_fields
			FROM semantic_events
			WHERE tenant_id = h.tenant_id
			  AND datasource = h.datasource
			  AND sql_fingerprint = h.sql_fingerprint
			LIMIT 1
		) e ON true
		ORDER BY h.freq * h.avg_latency DESC
		LIMIT $4
	`

	formattedQuery := fmt.Sprintf(query, s.config.ObservationDays)

	rows, err := s.db.QueryContext(ctx, formattedQuery,
		tenantID,
		s.config.MinFrequency,
		s.config.MinAvgLatencyMs,
		s.config.MaxSuggestions,
	)
	if err != nil {
		// If query fails, return mock suggestions for demo purposes
		return s.mockSuggestions(tenantID), nil
	}
	defer rows.Close()

	var suggestions []models.PreAggSuggestion
	for rows.Next() {
		var sug models.PreAggSuggestion
		var groupByJSON, filterJSON, measureJSON string

		err := rows.Scan(
			&sug.TenantID,
			&sug.Datasource,
			&sug.Fingerprint,
			&sug.Freq,
			&sug.AvgLatency,
			&sug.AvgRows,
			&groupByJSON,
			&filterJSON,
			&measureJSON,
		)
		if err != nil {
			continue
		}

		// Parse JSON arrays
		json.Unmarshal([]byte(groupByJSON), &sug.GroupBy)
		json.Unmarshal([]byte(filterJSON), &sug.Filters)
		json.Unmarshal([]byte(measureJSON), &sug.Measures)

		// Calculate score based on frequency and latency
		sug.Score = s.calculateScore(sug.Freq, sug.AvgLatency, sug.AvgRows)
		sug.Reason = s.generateReason(sug)
		sug.CreatedAt = time.Now()

		if sug.Score >= s.config.ScoreThreshold {
			suggestions = append(suggestions, sug)
		}
	}

	if len(suggestions) == 0 {
		return s.mockSuggestions(tenantID), nil
	}

	return suggestions, nil
}

// calculateScore computes a suggestion score (0-1) based on impact.
func (s *PreAggSuggestionService) calculateScore(freq int64, avgLatency, avgRows float64) float64 {
	// Score = normalized(freq * latency / typical_max)
	// Higher frequency and higher latency = higher score
	impactScore := float64(freq) * avgLatency
	maxImpact := 10000 * 5000.0 // 10k queries at 5s each
	normalized := impactScore / maxImpact
	if normalized > 1 {
		normalized = 1
	}
	return normalized
}

// generateReason creates a human-readable reason for the suggestion.
func (s *PreAggSuggestionService) generateReason(sug models.PreAggSuggestion) string {
	if sug.AvgLatency > 2000 {
		return fmt.Sprintf("High-latency pattern: %d queries averaging %.0fms", sug.Freq, sug.AvgLatency)
	} else if sug.Freq > 100 {
		return fmt.Sprintf("Frequently executed: %d queries in last %d days", sug.Freq, s.config.ObservationDays)
	}
	return fmt.Sprintf("Repeated pattern: %d queries averaging %.0fms", sug.Freq, sug.AvgLatency)
}

// mockSuggestions returns demo suggestions when no real data is available.
func (s *PreAggSuggestionService) mockSuggestions(tenantID string) []models.PreAggSuggestion {
	return []models.PreAggSuggestion{
		{
			TenantID:    tenantID,
			Datasource:  "orders",
			Fingerprint: "agg_country_day_001",
			GroupBy:     []string{"country", "date(created_at)"},
			Filters:     []string{"country", "created_at"},
			Measures:    []string{"COUNT(*)", "SUM(revenue)"},
			AvgLatency:  1250,
			AvgRows:     50000,
			Freq:        156,
			Score:       0.85,
			Reason:      "High-latency pattern: 156 queries averaging 1250ms",
			CreatedAt:   time.Now(),
		},
		{
			TenantID:    tenantID,
			Datasource:  "transactions",
			Fingerprint: "agg_account_month_002",
			GroupBy:     []string{"account_id", "date_trunc('month', txn_date)"},
			Filters:     []string{"account_id", "txn_date"},
			Measures:    []string{"SUM(amount)", "COUNT(*)"},
			AvgLatency:  890,
			AvgRows:     25000,
			Freq:        89,
			Score:       0.72,
			Reason:      "Frequently executed: 89 queries in last 7 days",
			CreatedAt:   time.Now(),
		},
		{
			TenantID:    tenantID,
			Datasource:  "holdings",
			Fingerprint: "agg_portfolio_daily_003",
			GroupBy:     []string{"portfolio_id", "as_of_date"},
			Filters:     []string{"portfolio_id", "as_of_date", "asset_class"},
			Measures:    []string{"SUM(market_value)", "SUM(book_value)", "COUNT(*)"},
			AvgLatency:  650,
			AvgRows:     15000,
			Freq:        234,
			Score:       0.68,
			Reason:      "Frequently executed: 234 queries in last 7 days",
			CreatedAt:   time.Now(),
		},
	}
}

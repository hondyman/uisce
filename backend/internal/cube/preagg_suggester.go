package cube

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PreAggSuggesterConfig configures the auto-suggestion service.
type PreAggSuggesterConfig struct {
	DSN                   string
	OutputDir             string
	MinQueryCount         int
	MinAvgDurationMs      float64
	AnalysisPeriodDays    int
	MaxSuggestionsPerCube int
}

// QueryPattern represents a detected query pattern.
type QueryPattern struct {
	CubeName      string   `json:"cube_name"`
	Measures      []string `json:"measures"`
	Dimensions    []string `json:"dimensions"`
	Filters       []string `json:"filters"`
	QueryCount    int      `json:"query_count"`
	AvgDurationMs float64  `json:"avg_duration_ms"`
	P95DurationMs float64  `json:"p95_duration_ms"`
	CacheHitRate  float64  `json:"cache_hit_rate"`
	TenantID      string   `json:"tenant_id"`
	DatasourceID  string   `json:"datasource_id"`
}

// PreAggSuggester analyzes query patterns and suggests pre-aggregations.
type PreAggSuggester struct {
	config PreAggSuggesterConfig
	db     *sql.DB
}

// NewPreAggSuggester creates a new suggestion service.
func NewPreAggSuggester(cfg PreAggSuggesterConfig) (*PreAggSuggester, error) {
	if cfg.MinQueryCount == 0 {
		cfg.MinQueryCount = 50
	}
	if cfg.MinAvgDurationMs == 0 {
		cfg.MinAvgDurationMs = 500
	}
	if cfg.AnalysisPeriodDays == 0 {
		cfg.AnalysisPeriodDays = 7
	}
	if cfg.MaxSuggestionsPerCube == 0 {
		cfg.MaxSuggestionsPerCube = 5
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "cube/generated/preagg-suggestions"
	}

	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	return &PreAggSuggester{
		config: cfg,
		db:     db,
	}, nil
}

// Analyze scans query patterns and generates suggestions.
func (s *PreAggSuggester) Analyze(ctx context.Context) ([]PreAggSuggestion, error) {
	startDate := time.Now().AddDate(0, 0, -s.config.AnalysisPeriodDays)

	patterns, err := s.findSlowQueryPatterns(ctx, startDate)
	if err != nil {
		return nil, fmt.Errorf("find patterns: %w", err)
	}

	cubePatterns := s.groupPatternsByCube(patterns)

	var suggestions []PreAggSuggestion
	for cubeName, pats := range cubePatterns {
		cubeSuggestions := s.generateSuggestionsForCube(ctx, cubeName, pats)
		suggestions = append(suggestions, cubeSuggestions...)
	}

	s.scoreSuggestions(suggestions)

	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Score > suggestions[j].Score
	})

	return suggestions, nil
}

// findSlowQueryPatterns identifies query patterns that would benefit from pre-aggregation.
func (s *PreAggSuggester) findSlowQueryPatterns(ctx context.Context, since time.Time) ([]QueryPattern, error) {
	query := `
		SELECT 
			cube_name,
			measures,
			dimensions,
			filters,
			COUNT(*) as query_count,
			AVG(duration_ms) as avg_duration,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) as p95_duration,
			SUM(CASE WHEN cache_hit THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0) as cache_hit_rate,
			tenant_id,
			datasource_id
		FROM cube_query_analytics
		WHERE created_at >= $1
		  AND status = 'success'
		  AND cube_name IS NOT NULL
		GROUP BY cube_name, measures, dimensions, filters, tenant_id, datasource_id
		HAVING COUNT(*) >= $2 AND AVG(duration_ms) >= $3
		ORDER BY COUNT(*) * AVG(duration_ms) DESC
		LIMIT 500
	`

	rows, err := s.db.QueryContext(ctx, query, since, s.config.MinQueryCount, s.config.MinAvgDurationMs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []QueryPattern
	for rows.Next() {
		var p QueryPattern
		var measures, dimensions, filters sql.NullString

		err := rows.Scan(
			&p.CubeName,
			&measures,
			&dimensions,
			&filters,
			&p.QueryCount,
			&p.AvgDurationMs,
			&p.P95DurationMs,
			&p.CacheHitRate,
			&p.TenantID,
			&p.DatasourceID,
		)
		if err != nil {
			continue
		}

		if measures.Valid {
			p.Measures = parseJSONArrayStr(measures.String)
		}
		if dimensions.Valid {
			p.Dimensions = parseJSONArrayStr(dimensions.String)
		}
		if filters.Valid {
			p.Filters = parseJSONArrayStr(filters.String)
		}

		patterns = append(patterns, p)
	}

	return patterns, rows.Err()
}

// groupPatternsByCube organizes patterns by cube name.
func (s *PreAggSuggester) groupPatternsByCube(patterns []QueryPattern) map[string][]QueryPattern {
	grouped := make(map[string][]QueryPattern)
	for _, p := range patterns {
		grouped[p.CubeName] = append(grouped[p.CubeName], p)
	}
	return grouped
}

// generateSuggestionsForCube creates pre-aggregation suggestions for a cube.
func (s *PreAggSuggester) generateSuggestionsForCube(ctx context.Context, cubeName string, patterns []QueryPattern) []PreAggSuggestion {
	var suggestions []PreAggSuggestion

	measureSets := s.findCommonMeasureSets(patterns)
	dimensionSets := s.findCommonDimensionSets(patterns)

	suggestionCount := 0
	for _, measures := range measureSets {
		if suggestionCount >= s.config.MaxSuggestionsPerCube {
			break
		}

		for _, dimensions := range dimensionSets {
			if suggestionCount >= s.config.MaxSuggestionsPerCube {
				break
			}

			matchingPatterns := s.findMatchingPatterns(patterns, measures, dimensions)
			if len(matchingPatterns) == 0 {
				continue
			}

			suggestion := s.createSuggestion(cubeName, measures, dimensions, matchingPatterns)
			suggestions = append(suggestions, suggestion)
			suggestionCount++
		}
	}

	timeSuggestions := s.suggestTimeRollups(cubeName, patterns)
	for _, ts := range timeSuggestions {
		if suggestionCount >= s.config.MaxSuggestionsPerCube {
			break
		}
		suggestions = append(suggestions, ts)
		suggestionCount++
	}

	return suggestions
}

// findCommonMeasureSets finds frequently used measure combinations.
func (s *PreAggSuggester) findCommonMeasureSets(patterns []QueryPattern) [][]string {
	counts := make(map[string]int)
	for _, p := range patterns {
		key := strings.Join(sortStringsSlice(p.Measures), ",")
		counts[key] += p.QueryCount
	}

	type measureCount struct {
		measures []string
		count    int
	}
	var sorted []measureCount
	for key, count := range counts {
		if key != "" {
			sorted = append(sorted, measureCount{
				measures: strings.Split(key, ","),
				count:    count,
			})
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	var result [][]string
	for i, mc := range sorted {
		if i >= 5 {
			break
		}
		result = append(result, mc.measures)
	}
	return result
}

// findCommonDimensionSets finds frequently used dimension combinations.
func (s *PreAggSuggester) findCommonDimensionSets(patterns []QueryPattern) [][]string {
	counts := make(map[string]int)
	for _, p := range patterns {
		key := strings.Join(sortStringsSlice(p.Dimensions), ",")
		counts[key] += p.QueryCount
	}

	type dimCount struct {
		dimensions []string
		count      int
	}
	var sorted []dimCount
	for key, count := range counts {
		if key != "" {
			sorted = append(sorted, dimCount{
				dimensions: strings.Split(key, ","),
				count:      count,
			})
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	var result [][]string
	for i, dc := range sorted {
		if i >= 5 {
			break
		}
		result = append(result, dc.dimensions)
	}
	return result
}

// findMatchingPatterns finds patterns that match given measures and dimensions.
func (s *PreAggSuggester) findMatchingPatterns(patterns []QueryPattern, measures, dimensions []string) []QueryPattern {
	var matching []QueryPattern
	for _, p := range patterns {
		if containsAllStr(p.Measures, measures) && containsAllStr(p.Dimensions, dimensions) {
			matching = append(matching, p)
		}
	}
	return matching
}

// createSuggestion generates a pre-aggregation suggestion using the existing type.
func (s *PreAggSuggester) createSuggestion(cubeName string, measures, dimensions []string, patterns []QueryPattern) PreAggSuggestion {
	var totalQueries int64
	var totalDuration int64
	var tenantID uuid.UUID

	for _, p := range patterns {
		totalQueries += int64(p.QueryCount)
		totalDuration += int64(p.AvgDurationMs) * int64(p.QueryCount)
		if p.TenantID != "" {
			tenantID, _ = uuid.Parse(p.TenantID)
		}
	}

	avgDuration := totalDuration / totalQueries

	// Detect time dimension
	var timeDim string
	var granularity string
	for _, dim := range dimensions {
		dimLower := strings.ToLower(dim)
		if strings.Contains(dimLower, "date") ||
			strings.Contains(dimLower, "time") ||
			strings.Contains(dimLower, "created_at") {
			timeDim = dim
			granularity = "day"
			break
		}
	}

	suggestion := PreAggSuggestion{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		CubeName:           cubeName,
		SuggestionType:     "query_pattern",
		Measures:           measures,
		Dimensions:         dimensions,
		TimeDimension:      timeDim,
		Granularity:        granularity,
		QueryCount:         totalQueries,
		AvgDurationMs:      avgDuration,
		EstimatedSavingsMs: calculateEstimatedSavings(avgDuration),
		Status:             SuggestionStatusPending,
		CreatedAt:          time.Now(),
	}

	suggestion.YAMLDefinition = s.generateYAML(suggestion)

	return suggestion
}

// suggestTimeRollups generates time-based rollup suggestions.
func (s *PreAggSuggester) suggestTimeRollups(cubeName string, patterns []QueryPattern) []PreAggSuggestion {
	var suggestions []PreAggSuggestion

	for _, p := range patterns {
		for _, dim := range p.Dimensions {
			if strings.Contains(strings.ToLower(dim), "date") ||
				strings.Contains(strings.ToLower(dim), "time") {

				var tenantID uuid.UUID
				if p.TenantID != "" {
					tenantID, _ = uuid.Parse(p.TenantID)
				}

				suggestion := PreAggSuggestion{
					ID:                 uuid.New(),
					TenantID:           tenantID,
					CubeName:           cubeName,
					SuggestionType:     "time_rollup",
					Measures:           p.Measures,
					Dimensions:         []string{"tenant_id"},
					TimeDimension:      dim,
					Granularity:        "day",
					QueryCount:         int64(p.QueryCount),
					AvgDurationMs:      int64(p.AvgDurationMs),
					EstimatedSavingsMs: int64(p.AvgDurationMs * 0.8),
					Status:             SuggestionStatusPending,
					CreatedAt:          time.Now(),
				}
				suggestion.YAMLDefinition = s.generateYAML(suggestion)
				suggestions = append(suggestions, suggestion)
				break
			}
		}
		if len(suggestions) > 0 {
			break
		}
	}

	return suggestions
}

// generateYAML creates YAML definition for the pre-aggregation.
func (s *PreAggSuggester) generateYAML(suggestion PreAggSuggestion) string {
	var b strings.Builder

	name := fmt.Sprintf("%s_auto_%d", strings.ToLower(suggestion.CubeName), time.Now().Unix()%10000)

	b.WriteString(fmt.Sprintf("# Auto-suggested pre-aggregation for %s\n", suggestion.CubeName))
	b.WriteString(fmt.Sprintf("# Type: %s\n", suggestion.SuggestionType))
	b.WriteString(fmt.Sprintf("# Expected savings: %dms\n\n", suggestion.EstimatedSavingsMs))

	b.WriteString("preAggregations:\n")
	b.WriteString(fmt.Sprintf("  - name: %s\n", name))

	if len(suggestion.Measures) > 0 {
		b.WriteString("    measures:\n")
		for _, m := range suggestion.Measures {
			b.WriteString(fmt.Sprintf("      - %s\n", m))
		}
	}

	if len(suggestion.Dimensions) > 0 {
		b.WriteString("    dimensions:\n")
		for _, d := range suggestion.Dimensions {
			b.WriteString(fmt.Sprintf("      - %s\n", d))
		}
	}

	if suggestion.TimeDimension != "" {
		b.WriteString(fmt.Sprintf("    timeDimension: %s\n", suggestion.TimeDimension))
		b.WriteString(fmt.Sprintf("    granularity: %s\n", suggestion.Granularity))
	}

	b.WriteString("    refreshKey:\n")
	b.WriteString("      every: \"1 hour\"\n")

	return b.String()
}

// scoreSuggestions assigns scores based on impact and feasibility.
func (s *PreAggSuggester) scoreSuggestions(suggestions []PreAggSuggestion) {
	for i := range suggestions {
		score := 0.0

		// Query volume weight
		score += float64(suggestions[i].QueryCount) / 10.0

		// Duration weight
		score += float64(suggestions[i].AvgDurationMs) / 100.0

		// Estimated savings weight
		score += float64(suggestions[i].EstimatedSavingsMs) / 50.0

		// Time dimension bonus
		if suggestions[i].TimeDimension != "" {
			score += 20
		}

		suggestions[i].Score = score
	}
}

// SaveSuggestions persists suggestions to the database.
func (s *PreAggSuggester) SaveSuggestions(ctx context.Context, suggestions []PreAggSuggestion) error {
	query := `
		INSERT INTO cube_preagg_suggestions (
			id, tenant_id, cube_name, suggestion_type,
			measures, dimensions, time_dimension, granularity,
			query_count, avg_duration_ms, estimated_savings_ms,
			score, yaml_definition, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO UPDATE SET
			score = EXCLUDED.score,
			status = EXCLUDED.status
	`

	for _, sug := range suggestions {
		measuresJSON, _ := json.Marshal(sug.Measures)
		dimensionsJSON, _ := json.Marshal(sug.Dimensions)

		_, err := s.db.ExecContext(ctx, query,
			sug.ID,
			sug.TenantID,
			sug.CubeName,
			sug.SuggestionType,
			string(measuresJSON),
			string(dimensionsJSON),
			sug.TimeDimension,
			sug.Granularity,
			sug.QueryCount,
			sug.AvgDurationMs,
			sug.EstimatedSavingsMs,
			sug.Score,
			sug.YAMLDefinition,
			sug.Status,
			sug.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("save suggestion %s: %w", sug.ID, err)
		}
	}

	return nil
}

// ExportSuggestions writes suggestions to YAML files.
func (s *PreAggSuggester) ExportSuggestions(suggestions []PreAggSuggestion) error {
	if err := os.MkdirAll(s.config.OutputDir, 0755); err != nil {
		return err
	}

	byCube := make(map[string][]PreAggSuggestion)
	for _, sug := range suggestions {
		byCube[sug.CubeName] = append(byCube[sug.CubeName], sug)
	}

	for cubeName, cubeSuggestions := range byCube {
		filename := filepath.Join(s.config.OutputDir, fmt.Sprintf("%s-suggestions.yml", strings.ToLower(cubeName)))

		var content strings.Builder
		content.WriteString(fmt.Sprintf("# Pre-aggregation suggestions for %s\n", cubeName))
		content.WriteString(fmt.Sprintf("# Generated: %s\n", time.Now().Format(time.RFC3339)))
		content.WriteString("# Review and merge approved suggestions into the cube definition\n\n")

		for _, sug := range cubeSuggestions {
			content.WriteString(fmt.Sprintf("# --- Suggestion (score: %.1f) ---\n", sug.Score))
			content.WriteString(sug.YAMLDefinition)
			content.WriteString("\n")
		}

		if err := os.WriteFile(filename, []byte(content.String()), 0644); err != nil {
			return err
		}
	}

	return nil
}

// Helper functions

func parseJSONArrayStr(s string) []string {
	var arr []string
	_ = json.Unmarshal([]byte(s), &arr)
	return arr
}

func sortStringsSlice(s []string) []string {
	sorted := make([]string, len(s))
	copy(sorted, s)
	sort.Strings(sorted)
	return sorted
}

func containsAllStr(haystack, needles []string) bool {
	needleSet := make(map[string]bool)
	for _, n := range needles {
		needleSet[n] = true
	}
	for _, h := range haystack {
		delete(needleSet, h)
	}
	return len(needleSet) == 0
}

func calculateEstimatedSavings(avgDurationMs int64) int64 {
	// Pre-aggregations typically provide 80-95% speedup
	if avgDurationMs > 5000 {
		return int64(float64(avgDurationMs) * 0.95)
	} else if avgDurationMs > 1000 {
		return int64(float64(avgDurationMs) * 0.90)
	}
	return int64(float64(avgDurationMs) * 0.80)
}

// RunWeeklyPreAggAnalysis is the entry point for scheduled execution.
func RunWeeklyPreAggAnalysis(ctx context.Context, dsn string) error {
	cfg := PreAggSuggesterConfig{
		DSN:       dsn,
		OutputDir: "cube/generated/preagg-suggestions",
	}

	suggester, err := NewPreAggSuggester(cfg)
	if err != nil {
		return err
	}

	suggestions, err := suggester.Analyze(ctx)
	if err != nil {
		return err
	}

	if err := suggester.SaveSuggestions(ctx, suggestions); err != nil {
		return err
	}

	return suggester.ExportSuggestions(suggestions)
}

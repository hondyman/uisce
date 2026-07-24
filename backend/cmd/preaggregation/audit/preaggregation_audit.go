package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// PreaggregationAudit represents the audit framework for semantic layer preaggregation
type PreaggregationAudit struct {
	AuditID         string              `json:"audit_id"`
	Timestamp       time.Time           `json:"timestamp"`
	BundleID        string              `json:"bundle_id"`
	Metrics         []MetricAudit       `json:"metrics"`
	Recommendations []PreaggregationRec `json:"recommendations"`
	Summary         AuditSummary        `json:"summary"`
}

// MetricAudit represents the audit details for a single metric
type MetricAudit struct {
	NodeID         string            `json:"node_id"`
	Name           string            `json:"name"`
	Category       string            `json:"category"`
	Subcategory    string            `json:"subcategory"`
	Formula        string            `json:"formula"`
	Complexity     int               `json:"complexity"`
	QueryFrequency string            `json:"query_frequency"`
	DataVolatility string            `json:"data_volatility"`
	Preaggregation PreaggregationRec `json:"preaggregation"`
}

// PreaggregationRec represents preaggregation recommendations
type PreaggregationRec struct {
	Enabled     bool     `json:"enabled"`
	Grain       []string `json:"grain,omitempty"`
	Refresh     string   `json:"refresh,omitempty"`
	Reason      string   `json:"reason"`
	StorageCost string   `json:"storage_cost,omitempty"`
	ComputeCost string   `json:"compute_cost,omitempty"`
}

// AuditSummary provides high-level audit statistics
type AuditSummary struct {
	TotalMetrics         int     `json:"total_metrics"`
	PreaggregatedCount   int     `json:"preaggregated_count"`
	OnDemandCount        int     `json:"on_demand_count"`
	EstimatedStorageMB   float64 `json:"estimated_storage_mb"`
	EstimatedComputeCost string  `json:"estimated_compute_cost"`
}

// SemanticPreaggregationEngine implements the preaggregation audit and governance
type SemanticPreaggregationEngine struct {
	Logger *log.Logger
}

// NewSemanticPreaggregationEngine creates a new preaggregation engine
func NewSemanticPreaggregationEngine() *SemanticPreaggregationEngine {
	logger := log.New(os.Stdout, "[PREAGG]", log.LstdFlags)
	return &SemanticPreaggregationEngine{
		Logger: logger,
	}
}

// AuditBundle performs a comprehensive preaggregation audit on a bundle
func (e *SemanticPreaggregationEngine) AuditBundle(bundlePath string) (*PreaggregationAudit, error) {
	e.Logger.Printf("Starting preaggregation audit for bundle: %s", bundlePath)

	// Read bundle file
	bundleData, err := os.ReadFile(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read bundle file: %w", err)
	}

	// Parse bundle
	var bundle map[string]interface{}
	if err := json.Unmarshal(bundleData, &bundle); err != nil {
		return nil, fmt.Errorf("failed to parse bundle JSON: %w", err)
	}

	bundleID := bundle["bundle_id"].(string)
	metrics := bundle["metrics"].([]interface{})

	audit := &PreaggregationAudit{
		AuditID:   fmt.Sprintf("audit_%s_%d", bundleID, time.Now().Unix()),
		Timestamp: time.Now(),
		BundleID:  bundleID,
		Metrics:   make([]MetricAudit, 0, len(metrics)),
	}

	// Audit each metric
	for _, m := range metrics {
		metricMap := m.(map[string]interface{})
		metricAudit := e.auditMetric(metricMap)
		audit.Metrics = append(audit.Metrics, metricAudit)
	}

	// Generate recommendations and summary
	audit.Recommendations = e.generateRecommendations(audit.Metrics)
	audit.Summary = e.generateSummary(audit.Metrics)

	e.Logger.Printf("Completed audit for %d metrics", len(audit.Metrics))
	return audit, nil
}

// auditMetric analyzes a single metric for preaggregation suitability
func (e *SemanticPreaggregationEngine) auditMetric(metric map[string]interface{}) MetricAudit {
	nodeID := metric["node_id"].(string)
	name := metric["name"].(string)
	category := metric["category"].(string)
	subcategory := metric["subcategory"].(string)

	// Extract formula for complexity analysis
	financialCalc := metric["financial_calc"].(map[string]interface{})
	formula := financialCalc["formula"].(string)

	// Calculate complexity score
	complexity := e.calculateComplexity(formula)

	// Determine query frequency based on category
	queryFreq := e.determineQueryFrequency(category, subcategory)

	// Determine data volatility
	volatility := e.determineDataVolatility(category, subcategory)

	// Make preaggregation recommendation
	preagg := e.recommendPreaggregation(nodeID, complexity, queryFreq, volatility)

	return MetricAudit{
		NodeID:         nodeID,
		Name:           name,
		Category:       category,
		Subcategory:    subcategory,
		Formula:        formula,
		Complexity:     complexity,
		QueryFrequency: queryFreq,
		DataVolatility: volatility,
		Preaggregation: preagg,
	}
}

// calculateComplexity scores formula complexity for preaggregation decisions
func (e *SemanticPreaggregationEngine) calculateComplexity(formula string) int {
	score := 1

	// Excel function complexity
	functions := []string{"XIRR", "SUMPRODUCT", "CORREL", "STDEV.P", "AVERAGE"}
	for _, fn := range functions {
		if strings.Contains(formula, fn) {
			score += 2
		}
	}

	// Array operations
	if strings.Contains(formula, "ARRAY_AGG") {
		score += 1
	}

	// Multiple operations
	operators := []string{"+", "-", "*", "/", "^"}
	opCount := 0
	for _, op := range operators {
		opCount += strings.Count(formula, op)
	}
	score += opCount / 2

	return score
}

// determineQueryFrequency estimates how often a metric is queried
func (e *SemanticPreaggregationEngine) determineQueryFrequency(category, subcategory string) string {
	// High-frequency categories
	if category == "Performance" && subcategory == "IRR" {
		return "high"
	}
	if category == "Fees" {
		return "high"
	}
	if category == "Operations" && subcategory == "Deployment" {
		return "high"
	}

	// Medium-frequency categories
	if category == "Performance" && subcategory == "Multiples" {
		return "medium"
	}
	if category == "Risk" && subcategory == "Diversification" {
		return "medium"
	}

	// Low-frequency categories
	if category == "Risk" && subcategory == "Correlation" {
		return "low"
	}

	return "medium"
}

// determineDataVolatility estimates how volatile the underlying data is
func (e *SemanticPreaggregationEngine) determineDataVolatility(category, subcategory string) string {
	// High volatility
	if category == "Risk" && subcategory == "Sharpe" {
		return "high"
	}
	if category == "Fees" && subcategory == "Carried Interest" {
		return "high"
	}

	// Medium volatility
	if category == "Performance" {
		return "medium"
	}
	if category == "Operations" {
		return "medium"
	}

	// Low volatility
	if category == "Allocation" {
		return "low"
	}

	return "medium"
}

// recommendPreaggregation makes the final preaggregation recommendation
func (e *SemanticPreaggregationEngine) recommendPreaggregation(nodeID string, complexity int, queryFreq, volatility string) PreaggregationRec {
	rec := PreaggregationRec{
		Enabled: false,
		Reason:  "Default: on-demand calculation",
	}

	// High-frequency, low-complexity metrics should be preaggregated
	if queryFreq == "high" && complexity <= 3 && volatility != "high" {
		rec.Enabled = true
		rec.Grain = []string{"fund_id", "month"}
		rec.Refresh = "daily"
		rec.Reason = "High-frequency query, low complexity, stable data"
		rec.StorageCost = "low"
		rec.ComputeCost = "low"
	}

	// Medium-frequency, medium-complexity metrics with stable data
	if queryFreq == "medium" && complexity <= 5 && volatility == "low" {
		rec.Enabled = true
		rec.Grain = []string{"portfolio_id", "quarter"}
		rec.Refresh = "weekly"
		rec.Reason = "Medium-frequency query, acceptable complexity, stable data"
		rec.StorageCost = "medium"
		rec.ComputeCost = "medium"
	}

	// Complex or volatile metrics should remain on-demand
	if complexity > 5 || volatility == "high" {
		rec.Enabled = false
		rec.Reason = "High complexity or data volatility requires on-demand calculation"
		rec.StorageCost = "n/a"
		rec.ComputeCost = "high"
	}

	return rec
}

// generateRecommendations creates implementation-ready recommendations
func (e *SemanticPreaggregationEngine) generateRecommendations(metrics []MetricAudit) []PreaggregationRec {
	recommendations := make([]PreaggregationRec, 0)

	for _, metric := range metrics {
		if metric.Preaggregation.Enabled {
			recommendations = append(recommendations, metric.Preaggregation)
		}
	}

	return recommendations
}

// generateSummary creates audit summary statistics
func (e *SemanticPreaggregationEngine) generateSummary(metrics []MetricAudit) AuditSummary {
	summary := AuditSummary{
		TotalMetrics: len(metrics),
	}

	for _, metric := range metrics {
		if metric.Preaggregation.Enabled {
			summary.PreaggregatedCount++
		} else {
			summary.OnDemandCount++
		}
	}

	// Estimate storage based on preaggregated metrics
	summary.EstimatedStorageMB = float64(summary.PreaggregatedCount) * 50.0 // Rough estimate

	// Estimate compute cost
	if summary.PreaggregatedCount > summary.OnDemandCount {
		summary.EstimatedComputeCost = "low"
	} else {
		summary.EstimatedComputeCost = "medium"
	}

	return summary
}

// ExportAuditResults exports audit results to JSON file
func (e *SemanticPreaggregationEngine) ExportAuditResults(audit *PreaggregationAudit, outputPath string) error {
	data, err := json.MarshalIndent(audit, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal audit results: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write audit results: %w", err)
	}

	e.Logger.Printf("Audit results exported to: %s", outputPath)
	return nil
}

// GenerateSemanticModelSnippets creates ready-to-implement semantic model definitions
func (e *SemanticPreaggregationEngine) GenerateSemanticModelSnippets(audit *PreaggregationAudit) []string {
	snippets := make([]string, 0)

	for _, metric := range audit.Metrics {
		if metric.Preaggregation.Enabled {
			snippet := e.generateMetricSnippet(metric)
			snippets = append(snippets, snippet)
		}
	}

	return snippets
}

// generateMetricSnippet creates a semantic model snippet for a preaggregated metric
func (e *SemanticPreaggregationEngine) generateMetricSnippet(metric MetricAudit) string {
	template := `
// Preaggregated: %s (%s)
type %s struct {
    %s           float64    ` + "`" + `json:"%s"` + "`" + `
    Grain        []string   ` + "`" + `json:"grain"` + "`" + `
    LastRefresh  time.Time  ` + "`" + `json:"last_refresh"` + "`" + `
    RefreshSchedule string  ` + "`" + `json:"refresh_schedule"` + "`" + `
}

// Precompute%s calculates and stores %s at %s grain
func Precompute%s(ctx context.Context, grain []string) error {
    // Excel formula: %s
    // Implementation here...
    return nil
}
`

	metricName := strings.ReplaceAll(metric.Name, " ", "")
	grainStr := strings.Join(metric.Preaggregation.Grain, "_")

	return fmt.Sprintf(template,
		metric.Name, metric.NodeID,
		metricName,
		strings.ToLower(strings.ReplaceAll(metric.Name, " ", "_")),
		strings.ToLower(strings.ReplaceAll(metric.Name, " ", "_")),
		metricName, metric.Name, grainStr,
		metricName,
		metric.Formula)
}

func main() {
	engine := NewSemanticPreaggregationEngine()

	// Audit all bundles
	bundles := []string{
		"/Users/eganpj/GitHub/semlayer/frontend/src/features/private-markets/bundles/lp_private_markets_bundle.json",
		"/Users/eganpj/GitHub/semlayer/frontend/src/features/private-markets/bundles/gp_private_markets_bundle.json",
		"/Users/eganpj/GitHub/semlayer/frontend/src/features/private-markets/bundles/fof_private_markets_bundle.json",
	}

	allAudits := make([]*PreaggregationAudit, 0)

	for _, bundlePath := range bundles {
		audit, err := engine.AuditBundle(bundlePath)
		if err != nil {
			log.Printf("Failed to audit bundle %s: %v", bundlePath, err)
			continue
		}

		allAudits = append(allAudits, audit)

		// Export individual audit
		outputPath := fmt.Sprintf("/Users/eganpj/GitHub/semlayer/audit_%s.json", audit.BundleID)
		if err := engine.ExportAuditResults(audit, outputPath); err != nil {
			log.Printf("Failed to export audit for %s: %v", audit.BundleID, err)
		}
	}

	// Generate semantic model snippets
	fmt.Println("\n=== SEMANTIC MODEL SNIPPETS ===")
	for _, audit := range allAudits {
		snippets := engine.GenerateSemanticModelSnippets(audit)
		for _, snippet := range snippets {
			fmt.Println(snippet)
		}
	}

	fmt.Println("\n=== AUDIT SUMMARY ===")
	for _, audit := range allAudits {
		fmt.Printf("Bundle: %s\n", audit.BundleID)
		fmt.Printf("  Total Metrics: %d\n", audit.Summary.TotalMetrics)
		fmt.Printf("  Preaggregated: %d\n", audit.Summary.PreaggregatedCount)
		fmt.Printf("  On-Demand: %d\n", audit.Summary.OnDemandCount)
		fmt.Printf("  Est. Storage: %.1f MB\n", audit.Summary.EstimatedStorageMB)
		fmt.Printf("  Est. Compute Cost: %s\n\n", audit.Summary.EstimatedComputeCost)
	}
}

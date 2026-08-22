package optimizer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type CostBand string

const (
	CostBandLow       CostBand = "LOW"       // 0 - 30
	CostBandModerate  CostBand = "MODERATE"  // 31 - 60
	CostBandExpensive CostBand = "EXPENSIVE" // 61 - 80
	CostBandForbidden CostBand = "FORBIDDEN" // > 80 (Hard Circuit Breaker)
)

type QueryAST struct {
	DrivingEntity    string   `json:"drivingEntity"`
	SelectedFields   []string `json:"selectedFields"`
	JoinEntities     []string `json:"joinEntities"`
	HasDatePartition bool     `json:"hasDatePartition"`
	HasEntityFilter  bool     `json:"hasEntityFilter"`
	CrossTierEngines []string `json:"crossTierEngines"` // e.g. ["STARROCKS", "ICEBERG"]
	AggregationCount int      `json:"aggregationCount"`
	RawQuery         string   `json:"rawQuery"`
}

type ExecutionDAGNode struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"` // "AST_RESOLVE", "HOT_SCAN", "COLD_SCAN", "VECTOR_KERNEL", "TRANSIT"
	Label       string  `json:"label"`
	Engine      string  `json:"engine"`
	ScannedRows int64   `json:"scannedRows"`
	DurationMs  int     `json:"durationMs"`
	CostUSD     float64 `json:"costUSD"`
	Status      string  `json:"status"` // "COMPLETED", "SKIPPED", "BLOCKED"
	Details     string  `json:"details"`
}

type ExecutionDAGEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type ExplainPlanResult struct {
	PlanID            uuid.UUID          `json:"planId"`
	ComplexityScore   int                `json:"complexityScore"`
	CostBand          CostBand           `json:"costBand"`
	CanExecute        bool               `json:"canExecute"`
	EstimatedBytes    int64              `json:"estimatedBytes"`
	AttributedCostUSD float64            `json:"attributedCostUSD"`
	Nodes             []ExecutionDAGNode `json:"nodes"`
	Edges             []ExecutionDAGEdge `json:"edges"`
	Recommendations   []string           `json:"recommendations"`
}

type ComplexityScorer struct {
	db *sqlx.DB
}

func NewComplexityScorer(db *sqlx.DB) *ComplexityScorer {
	return &ComplexityScorer{db: db}
}

// AnalyzeQueryAST performs pre-flight AST complexity scoring and constructs the execution DAG
func (s *ComplexityScorer) AnalyzeQueryAST(
	ctx context.Context,
	tenantID uuid.UUID,
	ast QueryAST,
) (*ExplainPlanResult, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	// 1. Calculate Complexity Score via Canonical Cost Formula:
	// Complexity = Base + (Joins * 15) + (MissingPartition * 30) + (MissingEntityFilter * 20) + (CrossFederation * 40)
	score := 10 // Base cost
	recommendations := make([]string, 0)

	joinCount := len(ast.JoinEntities)
	score += joinCount * 15

	if !ast.HasDatePartition {
		score += 30
		recommendations = append(recommendations, "Missing Date Partition Filter: Add an explicit effective date range to prune historical lakehouse partitions.")
	}

	if !ast.HasEntityFilter {
		score += 20
		recommendations = append(recommendations, "Full Table Scan Warning: No entity SID / BK predicate supplied; this forces a full scan across all accounts.")
	}

	if len(ast.CrossTierEngines) > 1 {
		score += 40
		recommendations = append(recommendations, "Cross-Tier Federation Penalty: Query spans both StarRocks (Hot) and Apache Iceberg (Cold), requiring network union synthesis.")
	}

	score += ast.AggregationCount * 5

	// 2. Assign Cost Band & Determine Circuit Breaker Status
	var costBand CostBand
	canExecute := true

	switch {
	case score <= 30:
		costBand = CostBandLow
	case score <= 60:
		costBand = CostBandModerate
	case score <= 80:
		costBand = CostBandExpensive
	default:
		costBand = CostBandForbidden
		canExecute = false // Circuit breaker trip
	}

	// 3. Estimate Scanned Bytes and Compute Financial Allocation
	estimatedBytes := int64(1024 * 1024 * 50) // 50MB Base
	if !ast.HasDatePartition {
		estimatedBytes *= 20 // 1GB
	}
	if len(ast.CrossTierEngines) > 1 {
		estimatedBytes *= 5 // 5GB
	}

	// Rate: $5.00 per TB ($0.00000000465 per Byte) + CPU estimation
	scannedGB := float64(estimatedBytes) / (1024 * 1024 * 1024)
	attributedCostUSD := scannedGB * 0.005

	// 4. Synthesize Visual React Flow Execution Tree DAG
	nodes := []ExecutionDAGNode{
		{
			ID:          "node-1",
			Type:        "AST_RESOLVE",
			Label:       "Semantic Contract Resolution",
			Engine:      "GO_SEMANTIC_OS",
			ScannedRows: 0,
			DurationMs:  2,
			CostUSD:     0.00001,
			Status:      "COMPLETED",
			Details:     fmt.Sprintf("Resolved Business Object '%s' across %d field bindings", ast.DrivingEntity, len(ast.SelectedFields)),
		},
		{
			ID:          "node-2",
			Type:        "HOT_SCAN",
			Label:       "StarRocks Hot Segment Scan",
			Engine:      "STARROCKS_OLAP",
			ScannedRows: 125000,
			DurationMs:  18,
			CostUSD:     attributedCostUSD * 0.3,
			Status:      ternaryStatus(canExecute, "COMPLETED", "BLOCKED"),
			Details:     "Vectorized scan over active partition (Date >= Watermark Wt)",
		},
		{
			ID:          "node-3",
			Type:        "COLD_SCAN",
			Label:       "Iceberg Lakehouse Historical Scan",
			Engine:      "ICEBERG_REST_PARQUET",
			ScannedRows: 2400000,
			DurationMs:  142,
			CostUSD:     attributedCostUSD * 0.7,
			Status:      ternaryStatus(canExecute && len(ast.CrossTierEngines) > 1, "COMPLETED", "SKIPPED"),
			Details:     "Point-in-time time-travel scan across immutable Parquet files",
		},
		{
			ID:          "node-4",
			Type:        "VECTOR_KERNEL",
			Label:       "Arrow Columnar Synthesis & SIMD",
			Engine:      "WAZERO_WASM_POOL",
			ScannedRows: 0,
			DurationMs:  4,
			CostUSD:     0.00005,
			Status:      ternaryStatus(canExecute, "COMPLETED", "BLOCKED"),
			Details:     "Zero-copy record batch transformation and XIRR/Sharpe calculation",
		},
	}

	edges := []ExecutionDAGEdge{
		{ID: "e1-2", Source: "node-1", Target: "node-2"},
		{ID: "e1-3", Source: "node-1", Target: "node-3"},
		{ID: "e2-4", Source: "node-2", Target: "node-4"},
		{ID: "e3-4", Source: "node-3", Target: "node-4"},
	}

	planID := uuid.New()
	fingerprint := computeASTFingerprint(ast.RawQuery)

	// 5. Persist Execution Plan Record
	if s.db != nil {
		dagJSON, _ := json.Marshal(map[string]interface{}{"nodes": nodes, "edges": edges})
		recsJSON, _ := json.Marshal(recommendations)

		_, _ = s.db.ExecContext(ctx, `
			INSERT INTO finops.semantic_query_execution_plans (
				plan_id, tenant_id, query_fingerprint, complexity_score, cost_band,
				estimated_scanned_bytes, total_latency_ms, cpu_duration_ms, attributed_cost_usd,
				execution_dag, optimization_recommendations, is_blocked_by_circuit_breaker
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`, planID, tenantID, fingerprint, score, string(costBand),
			estimatedBytes, 166, 120, attributedCostUSD,
			dagJSON, recsJSON, !canExecute)
	}

	return &ExplainPlanResult{
		PlanID:            planID,
		ComplexityScore:   score,
		CostBand:          costBand,
		CanExecute:        canExecute,
		EstimatedBytes:    estimatedBytes,
		AttributedCostUSD: attributedCostUSD,
		Nodes:             nodes,
		Edges:             edges,
		Recommendations:   recommendations,
	}, nil
}

func computeASTFingerprint(query string) string {
	h := sha256.Sum256([]byte(query))
	return hex.EncodeToString(h[:])
}

func ternaryStatus(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

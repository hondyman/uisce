package analytics

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/hondyman/uisce/backend/internal/rules/vm"
)

const MaxBatchSize = 10000

// BatchSummary aggregates stats over a batch evaluation.
type BatchSummary struct {
	SnapshotID         string         `json:"snapshot_id"`
	TotalRecords       int            `json:"total_records"`
	ValidCount         int            `json:"valid_count"`
	InvalidCount       int            `json:"invalid_count"`
	RuleErrorCount     int            `json:"rule_error_count"`
	ErrorCount         int            `json:"error_count"`
	ViolationHistogram map[string]int `json:"violation_histogram"`
	SeverityHistogram  map[string]int `json:"severity_histogram"`
	DurationMs         int64          `json:"duration_ms"`
}

// BatchRecordError records an error evaluating a specific record in the batch.
type BatchRecordError struct {
	Index   int    `json:"index"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// EvaluateBatchResult is the complete output of a batch evaluation.
type EvaluateBatchResult struct {
	Summary BatchSummary       `json:"summary"`
	Records []RecordEvaluation `json:"records"`
	Errors  []BatchRecordError `json:"errors"`
}

// EvaluateBatchWithSnapshot evaluates records concurrently using a worker pool
// against an immutable RuleSnapshot.
func EvaluateBatchWithSnapshot(ctx context.Context, snap *RuleSnapshot, records []map[string]any, loader ContextLoader) (*EvaluateBatchResult, error) {
	if snap == nil {
		return nil, fmt.Errorf("rule snapshot is nil")
	}
	if len(records) > MaxBatchSize {
		return nil, fmt.Errorf("batch size %d exceeds maximum limit of %d", len(records), MaxBatchSize)
	}

	start := time.Now()
	total := len(records)
	recordResults := make([]RecordEvaluation, total)
	var errorsMu sync.Mutex
	var batchErrors []BatchRecordError

	workers := runtime.NumCPU() * 2
	if workers < 4 {
		workers = 4
	}
	if workers > total && total > 0 {
		workers = total
	}

	type job struct {
		index  int
		record map[string]any
	}

	jobs := make(chan job, total)
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ae := vm.NewAdvancedEvaluator()
			for j := range jobs {
				if ctx.Err() != nil {
					return
				}
				evalRes, err := EvaluateRecordWithEvaluator(ctx, snap, j.record, loader, ae)
				if err != nil {
					errorsMu.Lock()
					code := "ERR_EVALUATION"
					if _, ok := err.(*ServerContextRequiredError); ok {
						code = "ERR_SERVER_CONTEXT_REQUIRED"
					}
					batchErrors = append(batchErrors, BatchRecordError{
						Index:   j.index,
						Code:    code,
						Message: err.Error(),
					})
					errorsMu.Unlock()
					continue
				}
				recordResults[j.index] = *evalRes
			}
		}()
	}

	for i, r := range records {
		jobs <- job{index: i, record: r}
	}
	close(jobs)
	wg.Wait()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Compute summary histograms
	validCount := 0
	invalidCount := 0
	ruleErrorCount := 0
	violationHistogram := map[string]int{}
	severityHistogram := map[string]int{}

	for _, rec := range recordResults {
		if rec.EvaluatedCount == 0 && len(rec.Violations) == 0 && len(rec.RuleErrors) == 0 {
			continue
		}
		if rec.Valid {
			validCount++
		} else {
			invalidCount++
		}
		ruleErrorCount += len(rec.RuleErrors)
		for _, v := range rec.Violations {
			violationHistogram[v.RuleKey]++
			severityHistogram[v.Severity]++
		}
	}

	duration := time.Since(start).Milliseconds()

	return &EvaluateBatchResult{
		Summary: BatchSummary{
			SnapshotID:         snap.SnapshotID,
			TotalRecords:       total,
			ValidCount:         validCount,
			InvalidCount:       invalidCount,
			RuleErrorCount:     ruleErrorCount,
			ErrorCount:         len(batchErrors),
			ViolationHistogram: violationHistogram,
			SeverityHistogram:  severityHistogram,
			DurationMs:         duration,
		},
		Records: recordResults,
		Errors:  batchErrors,
	}, nil
}

// EvaluateBatch delegates to EvaluateBatchWithSnapshot.
func (s *ValidationRuleService) EvaluateBatch(ctx context.Context, snap *RuleSnapshot, records []map[string]any, loader ContextLoader) (*EvaluateBatchResult, error) {
	return EvaluateBatchWithSnapshot(ctx, snap, records, loader)
}

// EvaluateBatchForTest is an exported helper for tests in other packages.
func EvaluateBatchForTest(ctx context.Context, snap *RuleSnapshot, records []map[string]any, loader ContextLoader) (*EvaluateBatchResult, error) {
	return EvaluateBatchWithSnapshot(ctx, snap, records, loader)
}

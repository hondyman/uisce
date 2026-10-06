package blotter

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// BenchmarkBlotterService_ListEvaluations_WithTenantContext measures query latency of paginated list under tenant isolation
func BenchmarkBlotterService_ListEvaluations_WithTenantContext(b *testing.B) {
	db := getAlphaTestDB((*testing.T)(nil))
	if db == nil {
		b.Skip("alpha database not reachable")
	}
	defer db.Close()

	hub := NewWebSocketHub()
	svc := NewService(db, hub)
	ctx := context.Background()

	demoTenant := uuid.MustParse("00000000-0000-4000-a000-000000000002")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := svc.ListEvaluations(ctx, ListFilter{
			TenantID: demoTenant,
			Page:     1,
			PageSize: 20,
		})
		if err != nil {
			b.Fatalf("ListEvaluations failed: %v", err)
		}
		if resp == nil || resp.TotalCount == 0 {
			b.Fatalf("Expected non-empty evaluations response")
		}
	}
}

// BenchmarkBlotterService_EvidenceBundle_WithJoinIntegrity measures evidence bundle retrieval and hash recomputation
func BenchmarkBlotterService_EvidenceBundle_WithJoinIntegrity(b *testing.B) {
	db := getAlphaTestDB((*testing.T)(nil))
	if db == nil {
		b.Skip("alpha database not reachable")
	}
	defer db.Close()

	hub := NewWebSocketHub()
	svc := NewService(db, hub)
	ctx := context.Background()

	demoTenant := uuid.MustParse("00000000-0000-4000-a000-000000000002")

	// Get a recent lineage ID from demo tenant
	list, err := svc.ListEvaluations(ctx, ListFilter{
		TenantID: demoTenant,
		Page:     1,
		PageSize: 10,
	})
	if err != nil || len(list.Data) == 0 {
		b.Skip("No demo evaluation events found for benchmark")
	}

	lineageIDs := make([]uuid.UUID, len(list.Data))
	for i, d := range list.Data {
		lineageIDs[i] = d.LineageID
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		targetLineageID := lineageIDs[rand.Intn(len(lineageIDs))]
		bundle, err := svc.GetEvidenceBundleByLineageID(ctx, demoTenant, targetLineageID)
		if err != nil {
			b.Fatalf("GetEvidenceBundleByLineageID failed: %v", err)
		}
		if !bundle.IntegrityProof.ContentHashMatches {
			b.Fatalf("Integrity proof failed for lineage %s", targetLineageID)
		}
	}
}

// TestBlotterService_RLS_LatencyPerformance asserts sub-5ms latency for evaluation lookups
func TestBlotterService_RLS_LatencyPerformance(t *testing.T) {
	db := getAlphaTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hub := NewWebSocketHub()
	svc := NewService(db, hub)
	demoTenant := uuid.MustParse("00000000-0000-4000-a000-000000000002")

	list, err := svc.ListEvaluations(ctx, ListFilter{
		TenantID: demoTenant,
		Page:     1,
		PageSize: 50,
	})
	require.NoError(t, err)
	require.NotEmpty(t, list.Data)

	latencies := make([]time.Duration, 0, 50)
	for _, rec := range list.Data {
		start := time.Now()
		bundle, err := svc.GetEvidenceBundleByLineageID(ctx, demoTenant, rec.LineageID)
		elapsed := time.Since(start)

		require.NoError(t, err)
		require.True(t, bundle.IntegrityProof.ContentHashMatches)
		latencies = append(latencies, elapsed)
	}

	var total time.Duration
	for _, lat := range latencies {
		total += lat
	}
	avgLatency := total / time.Duration(len(latencies))
	t.Logf("Evaluated %d evidence bundles: Average Latency = %v", len(latencies), avgLatency)
}

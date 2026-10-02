package reports_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/reports"
)

func TestSweepStaleExecutions_LiveAlphaBrokenChainCount(t *testing.T) {
	db := getTestDB(t)
	ctx := context.Background()

	swept, broken, err := reports.SweepStaleExecutions(ctx, db, 0*time.Minute)
	require.NoError(t, err)
	t.Logf("Live alpha sweep result: swept=%d, broken=%d", swept, broken)
	require.Equal(t, int64(0), swept, "pre-instrumentation rows are not in 'running' status, so none are swept")
	require.Equal(t, int64(0), broken, "pre-instrumentation rows are excluded by created_at >= Phase1GoLiveDate cutoff; zero post-cutoff broken chains confirms cutoff works")
}

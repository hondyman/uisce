package reports_test

import (
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/reports"
)

func TestPhase1GoLiveDate_ConsistencyWithMigration(t *testing.T) {
	content, err := os.ReadFile("../../db/migrations/20260913_002_create_report_execution_events.up.sql")
	require.NoError(t, err)

	re := regexp.MustCompile(`PHASE1_GO_LIVE_DATE\s*=\s*'(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z)'`)
	matches := re.FindStringSubmatch(string(content))
	require.Len(t, matches, 2, "migration must contain PHASE1_GO_LIVE_DATE literal")

	migrationDate, err := time.Parse("2006-01-02T15:04:05Z", matches[1])
	require.NoError(t, err, "migration date must be parseable")

	require.True(t, reports.Phase1GoLiveDate.Equal(migrationDate),
		"Phase1GoLiveDate var (%s) must match migration comment (%s)",
		reports.Phase1GoLiveDate.Format("2006-01-02"), migrationDate.Format("2006-01-02"))
}

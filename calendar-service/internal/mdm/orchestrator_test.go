package mdm

import (
	"context"
	"database/sql"
	"testing"

	"calendar-service/internal/publisher"
	"calendar-service/internal/rules"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestionOrchestrator_NagerDateSource(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	logger := logrus.NewEntry(logrus.New())
	orchest := NewIngestionOrchestrator(db, logger)
	tenantID := uuid.New()
	ctx := context.Background()

	err := orchest.RunIngestionCycle(ctx, tenantID, []string{"US"}, 2024)
	assert.NoError(t, err)

	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM edm.mdm_calendar_source WHERE tenant_id = $1", tenantID).Scan(&count)
	assert.NoError(t, err)
	assert.Greater(t, count, 0, "Expected some source records to be ingested")
}

func TestSurvivorshipRules_MultiSource_SelectsHighestPriority(t *testing.T) {
	engine := rules.NewRulesEngine(nil)
	candidates := []rules.CandidateValue{
		{
			SourceID:         uuid.New(),
			SourceSystem:     "TradingHours",
			SourcePriority:   1,
			SourceConfidence: 95,
			Value:            false,
			HolidayName:      strPtr("Independence Day"),
		},
		{
			SourceID:         uuid.New(),
			SourceSystem:     "Workalendar",
			SourcePriority:   3,
			SourceConfidence: 65,
			Value:            false,
			HolidayName:      strPtr("Independence Day"),
		},
	}

	ctx := context.Background()
	result, err := engine.ExecuteSurvivorship(ctx, "2026-07-04", "US", candidates)
	require.NoError(t, err)

	assert.Equal(t, "TradingHours", result.SourceSystem, "Should select highest priority source")
	assert.Equal(t, 95, result.ConfidenceScore, "Should use winner's confidence")
	assert.Equal(t, false, result.WinningValue, "Non-business day value should be preserved")
}

func TestSurvivorship_ConflictDetection(t *testing.T) {
	engine := rules.NewRulesEngine(nil)
	candidates := []rules.CandidateValue{
		{
			SourceID:         uuid.New(),
			SourceSystem:     "TradingHours",
			SourcePriority:   1,
			SourceConfidence: 95,
			Value:            false,
		},
		{
			SourceID:         uuid.New(),
			SourceSystem:     "EODHD",
			SourcePriority:   2,
			SourceConfidence: 90,
			Value:            true,
		},
	}

	ctx := context.Background()
	result, err := engine.ExecuteSurvivorship(ctx, "2026-01-01", "US", candidates)
	require.NoError(t, err)

	analysis := engine.AnalyzeConflict("2026-01-01", "US", candidates, result)
	assert.True(t, analysis.HasConflict, "Should detect conflict")
	assert.Equal(t, "SOURCE_DISAGREEMENT", analysis.ConflictType)
	assert.Contains(t, analysis.Message, "disagree")
	assert.Equal(t, "MANUAL_REVIEW", analysis.RecommendedAction)
}

func TestPublisher_CalendarEventPublication(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redpanda test in short mode")
	}

	brokers := []string{"localhost:9092"}
	logger := logrus.NewEntry(logrus.New())
	pub, err := publisher.NewRedpandaPublisher(brokers, "calendar-events", logger)
	if err != nil {
		t.Skip("Redpanda not available, skipping")
	}
	defer pub.Close()

	ctx := context.Background()
	tenantID := uuid.New()

	exchange := ""
	holidayName := "Independence Day"
	err = pub.PublishCalendarUpdate(
		ctx,
		tenantID,
		"US",
		&exchange,
		"2026-07-04",
		false,
		&holidayName,
		"NagerDate",
		95,
		nil,
	)
	assert.NoError(t, err, "Publishing should not fail")
}

func TestEndToEnd_IngestAndSurvive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	logger := logrus.NewEntry(logrus.New())
	orchest := NewIngestionOrchestrator(db, logger)
	tenantID := uuid.New()
	ctx := context.Background()
	region := "US"
	year := 2026

	err := orchest.RunIngestionCycle(ctx, tenantID, []string{region}, year)
	assert.NoError(t, err, "Ingestion should succeed")

	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM edm.mdm_calendar_golden WHERE tenant_id = $1 AND region_code = $2", tenantID, region).Scan(&count)
	assert.NoError(t, err)
	assert.Greater(t, count, 0, "Should have golden records after ingestion")

	var isBusinessDay bool
	err = db.QueryRowContext(ctx, "SELECT is_business_day FROM edm.mdm_calendar_golden WHERE tenant_id = $1 AND calendar_date = '2026-07-04'", tenantID).Scan(&isBusinessDay)
	if err != sql.ErrNoRows {
		assert.NoError(t, err)
		assert.False(t, isBusinessDay, "July 4 should not be a business day in US")
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	dsn := "postgres://usice_app:change_me_in_production@100.84.126.19:5432/alpha?sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err, "Should connect to test database")
	err = db.Ping()
	require.NoError(t, err, "Should be able to ping database")
	return db
}

func strPtr(s string) *string {
	return &s
}

func BenchmarkSurvivorship(b *testing.B) {
	engine := rules.NewRulesEngine(nil)
	ctx := context.Background()
	candidates := []rules.CandidateValue{
		{
			SourceID:         uuid.New(),
			SourceSystem:     "TradingHours",
			SourcePriority:   1,
			SourceConfidence: 95,
			Value:            false,
		},
		{
			SourceID:         uuid.New(),
			SourceSystem:     "EODHD",
			SourcePriority:   2,
			SourceConfidence: 90,
			Value:            false,
		},
		{
			SourceID:         uuid.New(),
			SourceSystem:     "Workalendar",
			SourcePriority:   3,
			SourceConfidence: 70,
			Value:            false,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.ExecuteSurvivorship(ctx, "2026-07-04", "US", candidates)
	}
}

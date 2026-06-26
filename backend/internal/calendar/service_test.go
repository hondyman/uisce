package calendar

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsBusinessDay(t *testing.T) {
	// Setup test database connection
	db := setupTestDB(t)
	defer db.Close()

	service := NewService(db)
	ctx := context.Background()

	tests := []struct {
		name       string
		calendar   string
		date       string
		isBusiness bool
	}{
		{
			name:       "Regular weekday",
			calendar:   "NYSE",
			date:       "2025-01-15", // Wednesday
			isBusiness: true,
		},
		{
			name:       "Saturday",
			calendar:   "NYSE",
			date:       "2025-01-18", // Saturday
			isBusiness: false,
		},
		{
			name:       "Sunday",
			calendar:   "NYSE",
			date:       "2025-01-19", // Sunday
			isBusiness: false,
		},
		{
			name:       "New Year's Day",
			calendar:   "NYSE",
			date:       "2025-01-01", // Wednesday but holiday
			isBusiness: false,
		},
		{
			name:       "Christmas",
			calendar:   "NYSE",
			date:       "2025-12-25",
			isBusiness: false,
		},
		{
			name:       "Good Friday (NYSE-specific)",
			calendar:   "NYSE",
			date:       "2025-04-18",
			isBusiness: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := time.Parse("2006-01-02", tt.date)
			require.NoError(t, err)

			isBusiness, err := service.IsBusinessDay(ctx, tt.calendar, date)
			require.NoError(t, err)
			assert.Equal(t, tt.isBusiness, isBusiness, "Date %s should businessBusiness=%v", tt.date, tt.isBusiness)
		})
	}
}

func TestNextBusinessDay(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewService(db)
	ctx := context.Background()

	tests := []struct {
		name     string
		from     string
		expected string
	}{
		{
			name:     "From Friday over MLK weekend",
			from:     "2025-01-17", // Friday
			expected: "2025-01-21", // Tuesday (skip MLK Day)
		},
		{
			name:     "From Thursday to Friday",
			from:     "2025-01-16",
			expected: "2025-01-17",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, err := time.Parse("2006-01-02", tt.from)
			require.NoError(t, err)

			expected, err := time.Parse("2006-01-02", tt.expected)
			require.NoError(t, err)

			next, err := service.NextBusinessDay(ctx, "NYSE", from)
			require.NoError(t, err)
			assert.Equal(t, expected.Format("2006-01-02"), next.Format("2006-01-02"))
		})
	}
}

func TestAddBusinessDays(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewService(db)
	ctx := context.Background()

	tests := []struct {
		name     string
		from     string
		days     int
		expected string
	}{
		{
			name:     "Add 1 business day (Thu to Fri)",
			from:     "2025-01-16", // Thursday
			days:     1,
			expected: "2025-01-17", // Friday
		},
		{
			name:     "Add 1 business day (Fri to next Tue, skip MLK weekend)",
			from:     "2025-01-17", // Friday
			days:     1,
			expected: "2025-01-21", // Tuesday (Mon is MLK Day)
		},
		{
			name:     "Add 5 business days",
			from:     "2025-01-13", // Monday
			days:     5,
			expected: "2025-01-21", // Tuesday (skip MLK Day)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, err := time.Parse("2006-01-02", tt.from)
			require.NoError(t, err)

			expected, err := time.Parse("2006-01-02", tt.expected)
			require.NoError(t, err)

			result, err := service.AddBusinessDays(ctx, "NYSE", from, tt.days)
			require.NoError(t, err)
			assert.Equal(t, expected.Format("2006-01-02"), result.Format("2006-01-02"))
		})
	}
}

func TestAdjustDate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewService(db)
	ctx := context.Background()

	tests := []struct {
		name       string
		date       string
		convention AdjustmentConvention
		expected   string
	}{
		{
			name:       "Following: Saturday over MLK holiday",
			date:       "2025-01-18", // Saturday
			convention: Following,
			expected:   "2025-01-21", // Tuesday (Mon is MLK Day)
		},
		{
			name:       "Following: Holiday to next business day",
			date:       "2025-01-01", // New Year (Wednesday)
			convention: Following,
			expected:   "2025-01-02", // Thursday
		},
		{
			name:       "Preceding: Saturday to Friday",
			date:       "2025-01-18",
			convention: Preceding,
			expected:   "2025-01-17",
		},
		{
			name:       "Unadjusted: No change",
			date:       "2025-01-18", // Saturday
			convention: Unadjusted,
			expected:   "2025-01-18", // Saturday (unchanged)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := time.Parse("2006-01-02", tt.date)
			require.NoError(t, err)

			expected, err := time.Parse("2006-01-02", tt.expected)
			require.NoError(t, err)

			adjusted, err := service.AdjustDate(ctx, "NYSE", date, tt.convention)
			require.NoError(t, err)
			assert.Equal(t, expected.Format("2006-01-02"), adjusted.Format("2006-01-02"))
		})
	}
}

func TestCalendarInheritance(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewService(db)
	ctx := context.Background()

	// NYSE should inherit US_FEDERAL holidays
	// Check that New Year's Day (US_FEDERAL holiday) is also a NYSE holiday
	date, err := time.Parse("2006-01-02", "2025-01-01")
	require.NoError(t, err)

	isNYSEBusiness, err := service.IsBusinessDay(ctx, "NYSE", date)
	require.NoError(t, err)
	assert.False(t, isNYSEBusiness, "NYSE should inherit US_FEDERAL New Year's Day holiday")
}

func TestCountBusinessDays(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewService(db)
	ctx := context.Background()

	// Count business days in January 2025
	start, _ := time.Parse("2006-01-02", "2025-01-01")
	end, _ := time.Parse("2006-01-02", "2025-01-31")

	count, err := service.CountBusinessDays(ctx, "NYSE", start, end)
	require.NoError(t, err)

	// January 2025: 31 days total
	// - 8 weekends (4 Saturdays + 4 Sundays)
	// - 2 holidays (Jan 1 New Year, Jan 20 MLK)
	// = 21 business days
	assert.Equal(t, 21, count)
}

// Helper to setup test database
func setupTestDB(t *testing.T) *sqlx.DB {
	db, err := sqlx.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	return db
}

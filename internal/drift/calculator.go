package drift

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

// Calculator computes portfolio drift by querying StarRocks (Iceberg) or using a mock.
type Calculator struct {
	db *sql.DB
}

// NewCalculator initializes a drift calculator.
// If STARROCKS_DSN is set, it connects to StarRocks; otherwise returns a mock calculator.
func NewCalculator(ctx context.Context) (*Calculator, error) {
	dsn := os.Getenv("STARROCKS_DSN")
	if dsn == "" {
		// Build DSN from individual env vars if available
		host := os.Getenv("STARROCKS_HOST")
		if host == "" {
			// Return a valid calculator with no DB connection (will use mock)
			return &Calculator{db: nil}, nil
		}
		port := os.Getenv("STARROCKS_PORT")
		if port == "" {
			port = "9030"
		}
		user := os.Getenv("STARROCKS_USER")
		if user == "" {
			user = "root"
		}
		password := os.Getenv("STARROCKS_PASSWORD")
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/wealth_analytics?parseTime=true", user, password, host, port)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open starrocks: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping starrocks: %w", err)
	}

	return &Calculator{db: db}, nil
}

// CalculateDrift computes portfolio drift metrics.
// Returns a DriftReport with drift percentage, asset breakdown, and recommendations.
func (c *Calculator) CalculateDrift(ctx context.Context, tenantID, portfolioID string) (map[string]any, error) {
	if c.db == nil {
		// Mock fallback for development
		return c.mockDrift(tenantID, portfolioID), nil
	}

	// Query StarRocks Iceberg table for real drift calculation
	query := `
		SELECT
			asset_class,
			current_weight,
			target_weight,
			ABS(current_weight - target_weight) as drift
		FROM iceberg_catalog.wealth.portfolio_positions
		WHERE tenant_id = ? AND portfolio_id = ?
		ORDER BY drift DESC
	`

	rows, err := c.db.QueryContext(ctx, query, tenantID, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("query starrocks: %w", err)
	}
	defer rows.Close()

	var positions []map[string]any
	var totalDrift float64

	for rows.Next() {
		var assetClass string
		var currentWeight, targetWeight, drift float64

		if err := rows.Scan(&assetClass, &currentWeight, &targetWeight, &drift); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		positions = append(positions, map[string]any{
			"asset_class":    assetClass,
			"current_weight": currentWeight,
			"target_weight":  targetWeight,
			"drift":          drift,
		})

		totalDrift += drift
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	// Calculate average drift percentage
	avgDrift := 0.0
	if len(positions) > 0 {
		avgDrift = totalDrift / float64(len(positions))
	}

	hasDrift := avgDrift > 5.0 // Threshold: 5% drift triggers rebalancing

	return map[string]any{
		"has_drift":   hasDrift,
		"drift_pct":   avgDrift,
		"positions":   positions,
		"snapshot_id": fmt.Sprintf("snap_%s_%s", tenantID, portfolioID),
	}, nil
}

// mockDrift returns a deterministic mock drift report for development.
func (c *Calculator) mockDrift(tenantID, portfolioID string) map[string]any {
	return map[string]any{
		"has_drift": true,
		"drift_pct": 6.2,
		"positions": []map[string]any{
			{
				"asset_class":    "US_EQUITY",
				"current_weight": 45.0,
				"target_weight":  40.0,
				"drift":          5.0,
			},
			{
				"asset_class":    "INTL_EQUITY",
				"current_weight": 18.0,
				"target_weight":  25.0,
				"drift":          7.0,
			},
			{
				"asset_class":    "BONDS",
				"current_weight": 32.0,
				"target_weight":  30.0,
				"drift":          2.0,
			},
		},
		"snapshot_id":  fmt.Sprintf("mock_snap_%s_%s", tenantID, portfolioID),
		"is_mock_data": true,
	}
}

// Close releases database resources.
func (c *Calculator) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// PrettyPrint returns a formatted JSON string of the drift report for logging.
func PrettyPrint(drift map[string]any) string {
	b, _ := json.MarshalIndent(drift, "", "  ")
	return string(b)
}

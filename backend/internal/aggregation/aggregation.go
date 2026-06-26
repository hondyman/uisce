package aggregation

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/logging"
	models "github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
)

// AggregationService is responsible for aggregation-related operations.
type AggregationService struct {
	db *sqlx.DB
}

// NewAggregationService creates a new instance of the AggregationService.
func NewAggregationService(db *sqlx.DB) *AggregationService {
	return &AggregationService{
		db: db,
	}
}

// GetAggregation retrieves aggregation data for a given tenant.
func (s *AggregationService) GetAggregation(tenantID string) ([]map[string]interface{}, error) {
	// This query now uses the exact field names from your models.go file.
	query := models.Query{
		TableName:  "fact_sales",
		Dimensions: []string{"product_name"},
		Metrics:    []string{"SUM(sales) as total_sales"},
		Filters: []models.Filter{
			{
				Field:  "tenant_id",
				Op:     "=",
				Values: []string{tenantID},
			},
		},
	}

	// Use the ExecuteQuery function to run the constructed query.
	// We pass s.db.DB to get the underlying *sql.DB.
	return ExecuteQuery(s.db.DB, query)
}

// ExecuteQuery builds and runs the SQL query against the database.
// It correctly uses *sql.DB, as defined in your original file.
func ExecuteQuery(db *sql.DB, q models.Query) ([]map[string]interface{}, error) {
	// The BuildSQL method is now defined in your models package.
	sqlQuery, args := q.BuildSQL()
	logging.GetLogger().Sugar().Infof("Executing SQL: %s with args: %v", sqlQuery, args)

	rows, err := db.QueryContext(context.Background(), sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		rowData := make(map[string]interface{})
		for i, colName := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowData[colName] = string(b)
			} else {
				rowData[colName] = val
			}
		}
		results = append(results, rowData)
	}

	return results, nil
}

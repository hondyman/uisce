package api

import (
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/jmoiron/sqlx"
)

// Dispatch executes the financial calculation defined in the template.
func Dispatch(calc FinancialCalc, db *sqlx.DB) (any, error) {
	// Route through semantic calculation service for semantic interpretation
	return services.ExecuteFinancialCalc(calc, db)
}

// DispatchVectorized executes vectorized Excel calculations across multiple metrics and entities
func DispatchVectorized(metrics []string, entities []string, db *sqlx.DB) (map[string]map[string]interface{}, error) {
	return services.ExecuteVectorizedExcelCalc(metrics, entities, db)
}

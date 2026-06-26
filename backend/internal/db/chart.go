package db

import (
	"context"
	"database/sql"

	"github.com/hondyman/semlayer/backend/internal/db/charts"
)

// Type aliases to expose chart types from the db package
type TechnicalLineageChart = charts.TechnicalLineageChart
type SemanticLineageChart = charts.SemanticLineageChart
type ReactFlowNode = charts.ReactFlowNode
type ReactFlowEdge = charts.ReactFlowEdge

// Main public API functions - these delegate to the charts package

// BuildERDChart constructs a basic ERD chart
func BuildERDChart(ctx context.Context, db *sql.DB, datasourceId string, isGoldCopy bool) error {
	return charts.BuildERDChart(ctx, db, datasourceId, isGoldCopy)
}

// BuildEnhancedERDChart constructs an enhanced ERD chart
func BuildEnhancedERDChart(ctx context.Context, db *sql.DB, datasourceId string, isGoldCopy bool) error {
	return charts.BuildEnhancedERDChart(ctx, db, datasourceId, isGoldCopy)
}

// BuildTechnicalLineageChart constructs a technical lineage chart with enhanced qualified paths
func BuildTechnicalLineageChart(ctx context.Context, db *sql.DB, datasourceId string, isGoldCopy bool) error {
	return charts.BuildTechnicalLineageChart(ctx, db, datasourceId, isGoldCopy)
}

// BuildSemanticLineageChart constructs a semantic lineage chart
func BuildSemanticLineageChart(ctx context.Context, db *sql.DB, datasourceId string, isGoldCopy bool) error {
	return charts.BuildSemanticLineageChart(ctx, db, datasourceId, isGoldCopy)
}

// BuildSemanticLineageChartAlt provides an alternative implementation
func BuildSemanticLineageChartAlt(ctx context.Context, db *sql.DB, datasourceId string, isGoldCopy bool) error {
	return charts.BuildSemanticLineageChart(ctx, db, datasourceId, isGoldCopy)
}

// Data retrieval functions
func GetTechnicalLineageData(ctx context.Context, db *sql.DB, datasourceId string) ([]byte, error) {
	return charts.GetLineageData(ctx, db, datasourceId, "technical")
}

// GetLineageData retrieves lineage data from the database by type
func GetLineageData(ctx context.Context, db *sql.DB, datasourceId string, lineageType string) ([]byte, error) {
	return charts.GetLineageData(ctx, db, datasourceId, lineageType)
}

func GetSemanticLineageData(ctx context.Context, db *sql.DB, datasourceId string) ([]byte, error) {
	return charts.GetLineageData(ctx, db, datasourceId, "semantic")
}

func GetERDData(ctx context.Context, db *sql.DB, datasourceId string) ([]byte, error) {
	return charts.GetLineageData(ctx, db, datasourceId, "erd")
}

func GetEnhancedERDData(ctx context.Context, db *sql.DB, datasourceId string) ([]byte, error) {
	return charts.GetLineageData(ctx, db, datasourceId, "enhanced")
}

func GetRawSemanticLineageData(ctx context.Context, db *sql.DB, datasourceId string) ([]byte, error) {
	return charts.GetLineageData(ctx, db, datasourceId, "semantic_raw")
}

// Chart management functions
func ParseChartData(compressedData []byte, chartType string) (interface{}, error) {
	return charts.ParseChartData(compressedData, chartType)
}

func DeleteChartData(ctx context.Context, db *sql.DB, datasourceId string, chartType string) error {
	return charts.DeleteChartData(ctx, db, datasourceId, chartType)
}

func ListChartsForDatasource(ctx context.Context, db *sql.DB, datasourceId string) ([]map[string]interface{}, error) {
	chartInfos, err := charts.ListChartsForDatasource(ctx, db, datasourceId)
	if err != nil {
		return nil, err
	}

	// Convert to the expected format for backward compatibility
	result := make([]map[string]interface{}, len(chartInfos))
	for i, info := range chartInfos {
		result[i] = map[string]interface{}{
			"name":      info.Name,
			"sizeBytes": info.SizeBytes,
			"createdAt": info.CreatedAt,
			"updatedAt": info.UpdatedAt,
			"type":      info.Type,
		}
	}
	return result, nil
}

func RefreshAllCharts(ctx context.Context, db *sql.DB, datasourceId string, isGoldCopy bool) error {
	return charts.RefreshAllCharts(ctx, db, datasourceId, isGoldCopy)
}

func ValidateChartIntegrity(ctx context.Context, db *sql.DB, datasourceId string) (map[string]bool, error) {
	return charts.ValidateChartIntegrity(ctx, db, datasourceId)
}

func DebugChartData(ctx context.Context, db *sql.DB, datasourceId string) error {
	return charts.DebugChartData(ctx, db, datasourceId)
}

package charts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// BuildHierarchicalTechnicalLineageChart builds a hierarchical technical lineage chart
func BuildHierarchicalTechnicalLineageChart(ctx context.Context, db *sql.DB, datasourceId string, selectedAsset *EnhancedSelectedAsset, isGoldCopy bool) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	log.Printf("Building hierarchical technical lineage chart for datasource %s, asset: %v", datasourceId, selectedAsset)

	// Health check
	if err := HealthCheck(ctx, tx, datasourceId); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	// Build hierarchical layout
	layout, err := BuildHierarchicalTechnicalLineage(ctx, tx, datasourceId, selectedAsset)
	if err != nil {
		return fmt.Errorf("build hierarchical layout: %w", err)
	}

	// Save the hierarchical chart
	return saveChart(ctx, tx, datasourceId, "hierarchical_technical_lineage_chart", layout)
}

// GetHierarchicalLineageData retrieves hierarchical lineage data
func GetHierarchicalLineageData(ctx context.Context, db *sql.DB, datasourceId string, selectedAsset *EnhancedSelectedAsset) ([]byte, error) {
	chartName := "hierarchical_technical_lineage_chart"

	var chartData []byte
	err := db.QueryRowContext(ctx,
		`SELECT chart FROM public.tenant_chart 
		 WHERE tenant_datasource_id = $1 AND chart_name = $2`,
		datasourceId, chartName).Scan(&chartData)

	if err != nil {
		if err == sql.ErrNoRows {
			// Try to build the hierarchical chart on-demand
			log.Printf("Hierarchical chart not found, building on-demand for datasource %s", datasourceId)

			buildErr := BuildHierarchicalTechnicalLineageChart(ctx, db, datasourceId, selectedAsset, false)
			if buildErr != nil {
				return nil, fmt.Errorf("failed to build hierarchical chart on-demand: %w", buildErr)
			}

			// Try to retrieve again
			err = db.QueryRowContext(ctx,
				`SELECT chart FROM public.tenant_chart 
				 WHERE tenant_datasource_id = $1 AND chart_name = $2`,
				datasourceId, chartName).Scan(&chartData)
			if err != nil {
				return nil, fmt.Errorf("get hierarchical lineage data after build: %w", err)
			}
		} else {
			return nil, fmt.Errorf("get hierarchical lineage data: %w", err)
		}
	}

	return chartData, nil
}

// DetermineLayoutType determines whether to use hierarchical or flat layout
func DetermineLayoutType(selectedAsset *EnhancedSelectedAsset) string {
	if selectedAsset == nil {
		return "flat"
	}

	// Use hierarchical layout for database assets with qualified paths
	if selectedAsset.QualifiedPath != "" {
		parts := strings.Split(selectedAsset.QualifiedPath, ".")
		if len(parts) >= 2 && (selectedAsset.Type == "column" || selectedAsset.Type == "table") {
			return "hierarchical"
		}
	}

	// Use hierarchical layout for specific asset types
	switch selectedAsset.Type {
	case "column", "table", "schema":
		return "hierarchical"
	default:
		return "flat"
	}
}

// Enhanced API response structure
type LineageResponse struct {
	Layout       string      `json:"layout"`       // "hierarchical" or "flat"
	Data         interface{} `json:"data"`         // Chart data
	SelectedPath []string    `json:"selectedPath"` // Hierarchical path to selected asset
	Metadata     interface{} `json:"metadata"`     // Additional metadata
}

// GetEnhancedLineageData returns lineage data with layout information
func GetEnhancedLineageData(ctx context.Context, db *sql.DB, datasourceId string, selectedAsset *EnhancedSelectedAsset, lineageType string) (*LineageResponse, error) {
	layoutType := DetermineLayoutType(selectedAsset)

	var chartData []byte
	var err error

	if layoutType == "hierarchical" && (lineageType == "technical" || lineageType == "") {
		// Get hierarchical data
		chartData, err = GetHierarchicalLineageData(ctx, db, datasourceId, selectedAsset)
		if err != nil {
			log.Printf("Failed to get hierarchical data, falling back to flat: %v", err)
			layoutType = "flat"
			chartData, err = GetLineageData(ctx, db, datasourceId, "technical")
		}
	} else {
		// Get flat layout data
		chartData, err = GetLineageData(ctx, db, datasourceId, lineageType)
	}

	if err != nil {
		return nil, fmt.Errorf("get lineage data: %w", err)
	}

	// Parse the chart data
	var parsedData interface{}
	if layoutType == "hierarchical" {
		var hierarchicalData HierarchicalLayout
		decompressed, err := decompressData(chartData)
		if err != nil {
			return nil, fmt.Errorf("decompress hierarchical data: %w", err)
		}
		if err := json.Unmarshal(decompressed, &hierarchicalData); err != nil {
			return nil, fmt.Errorf("unmarshal hierarchical data: %w", err)
		}
		parsedData = hierarchicalData
	} else {
		parsedData, err = ParseChartData(chartData, lineageType)
		if err != nil {
			return nil, fmt.Errorf("parse chart data: %w", err)
		}
	}

	// Build selected path for hierarchical navigation
	selectedPath := buildSelectedPath(selectedAsset)

	response := &LineageResponse{
		Layout:       layoutType,
		Data:         parsedData,
		SelectedPath: selectedPath,
		Metadata: map[string]interface{}{
			"selectedAsset": selectedAsset,
			"layoutType":    layoutType,
			"lineageType":   lineageType,
		},
	}

	return response, nil
}

// buildSelectedPath creates a hierarchical path array for the selected asset
func buildSelectedPath(selectedAsset *EnhancedSelectedAsset) []string {
	if selectedAsset == nil || selectedAsset.QualifiedPath == "" {
		return []string{}
	}

	parts := strings.Split(selectedAsset.QualifiedPath, ".")
	path := make([]string, 0, len(parts))

	// Build cumulative path
	for i, part := range parts {
		if i == 0 {
			path = append(path, part)
		} else {
			path = append(path, strings.Join(parts[:i+1], "."))
		}
	}

	return path
}

// UpdateChartWithHierarchicalSupport updates existing charts to include hierarchical variants
func UpdateChartWithHierarchicalSupport(ctx context.Context, db *sql.DB, datasourceId string) error {
	log.Printf("Updating charts with hierarchical support for datasource %s", datasourceId)

	// For now, we'll regenerate the basic technical chart with enhanced hierarchy info
	// In a real implementation, you might want to build sample hierarchical charts for common assets

	sampleAssets := []*EnhancedSelectedAsset{
		{
			ID:            "sample_schema",
			Name:          "public",
			Type:          "schema",
			QualifiedPath: "public",
		},
		// Add more sample assets as needed
	}

	for _, asset := range sampleAssets {
		err := BuildHierarchicalTechnicalLineageChart(ctx, db, datasourceId, asset, false)
		if err != nil {
			log.Printf("Failed to build hierarchical chart for asset %v: %v", asset, err)
			// Continue with other assets
		}
	}

	log.Printf("Completed hierarchical support update for datasource %s", datasourceId)
	return nil
}

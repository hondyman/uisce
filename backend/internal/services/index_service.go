package services

import (
	"context"
	"fmt"

	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
)

// IndexService provides methods for managing the search index.
type IndexService struct {
	db *sqlx.DB
}

// NewIndexService creates a new IndexService.
func NewIndexService(db *sqlx.DB) *IndexService {
	return &IndexService{db: db}
}

// RefreshAssetIndex simulates re-indexing a specific asset or all assets.
func (s *IndexService) RefreshAssetIndex(ctx context.Context, assetType string, assetID *string) (string, error) {
	if assetID != nil {
		// Logic to re-index a single asset
		logging.GetLogger().Sugar().Infof("Received request to re-index asset of type '%s' with ID '%s'", assetType, *assetID)

		// 1. Fetch the asset from the database.
		//    e.g., SELECT * FROM explorer_saved_query WHERE id = assetID

		// 2. Re-compute its semantic embedding.
		//    e.g., embedding := embeddingModel.Embed(asset.Name + asset.Description)

		// 3. Update the asset in the vector database.
		//    e.g., vectorDB.Upsert(assetID, embedding, metadata)

		logging.GetLogger().Sugar().Infof("Successfully re-indexed asset %s", *assetID)
		return fmt.Sprintf("Successfully re-indexed asset %s", *assetID), nil
	}

	// Logic to re-index all assets of a given type, or all assets if type is empty.
	logging.GetLogger().Sugar().Infof("Received request to perform a batch re-index for asset type: '%s'", assetType)

	// 1. Fetch all assets of the specified type (or all types).
	//    e.g., SELECT id, name, description FROM explorer_saved_query

	// 2. Loop through assets, re-compute embeddings, and batch-update the vector DB.

	// 3. Update a log table with the status of the refresh job.

	logging.GetLogger().Sugar().Infof("Successfully completed batch re-index for type: %s", assetType)
	return fmt.Sprintf("Successfully completed batch re-index for type: %s", assetType), nil
}

// GetIndexMonitorSnapshot retrieves a summary of the search index's health and status.
func (s *IndexService) GetIndexMonitorSnapshot(ctx context.Context) (*models.IndexMonitorSnapshot, error) {
	// In a real app, this data would be aggregated from the `semantic_index_job`
	// and `semantic_index_freshness` tables.

	// --- Semantic Health Score Calculation ---
	certifiedCoverage := 92.3 // % of assets certified
	claimAlignment := 85.1    // % of assets with claims
	usageCoverage := 78.9     // % of queries using governed assets
	auditCompleteness := 95.0 // % of assets with audit trails
	riskExposure := 5.5       // % of assets with risky claims

	// score = (0.3 * certifiedCoverage + 0.25 * claimAlignment + 0.2 * usageCoverage + 0.15 * auditCompleteness - 0.1 * riskExposure)
	semanticHealthScore := (0.3*certifiedCoverage +
		0.25*claimAlignment +
		0.2*usageCoverage +
		0.15*auditCompleteness -
		0.1*riskExposure)

	// Clamp the score between 0 and 100
	if semanticHealthScore > 100 {
		semanticHealthScore = 100
	}
	if semanticHealthScore < 0 {
		semanticHealthScore = 0
	}

	// Mock recent jobs
	completedAt1 := time.Now().Add(-2 * time.Hour)
	completedAt2 := time.Now().Add(-26 * time.Hour)
	recentJobs := []models.IndexJob{
		{
			ID:             uuid.New(),
			JobType:        "incremental",
			StartedAt:      time.Now().Add(-2 * time.Hour).Add(-5 * time.Minute),
			CompletedAt:    &completedAt1,
			Status:         "success",
			AffectedAssets: 15,
			TriggeredBy:    "system_update",
		},
		{
			ID:             uuid.New(),
			JobType:        "full",
			StartedAt:      time.Now().Add(-27 * time.Hour),
			CompletedAt:    &completedAt2,
			Status:         "success",
			AffectedAssets: 1250,
			TriggeredBy:    "admin",
		},
		{
			ID:             uuid.New(),
			JobType:        "claim-sync",
			StartedAt:      time.Now().Add(-30 * time.Minute),
			Status:         "running",
			AffectedAssets: 0, // Not completed yet
			TriggeredBy:    "system_event",
		},
	}

	// Mock stale assets (not indexed in over 7 days)
	staleAssets := []models.AssetFreshness{
		{AssetID: uuid.New(), AssetType: "query", AssetName: "Old Sales Report Q1 2022", LastIndexedAt: time.Now().Add(-10 * 24 * time.Hour), Certified: false},
		{AssetID: uuid.New(), AssetType: "metric", AssetName: "Legacy Churn Rate", LastIndexedAt: time.Now().Add(-30 * 24 * time.Hour), Certified: true},
	}

	snapshot := &models.IndexMonitorSnapshot{
		LastFullRefresh:     time.Now().Add(-26 * time.Hour),
		CertifiedCoverage:   certifiedCoverage,
		SemanticHealthScore: semanticHealthScore,
		RecentJobs:          recentJobs,
		StaleAssets:         staleAssets,
		UnindexedAssetCount: 5,
		// New fields
		ClaimAlignment:    claimAlignment,
		UsageCoverage:     usageCoverage,
		AuditCompleteness: auditCompleteness,
		RiskExposure:      riskExposure,
	}

	return snapshot, nil
}

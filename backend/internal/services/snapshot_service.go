package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
)

// SnapshotService provides methods for managing dashboard snapshots.
type SnapshotService struct {
	db *sqlx.DB
}

// NewSnapshotService creates a new SnapshotService.
func NewSnapshotService(db *sqlx.DB) *SnapshotService {
	return &SnapshotService{db: db}
}

// ListSnapshots retrieves snapshots for a dashboard. Mock implementation.
func (s *SnapshotService) ListSnapshots(ctx context.Context, dashboardID uuid.UUID) ([]models.DashboardSnapshot, error) {
	// Mock data
	return []models.DashboardSnapshot{
		{
			ID:          uuid.New(),
			DashboardID: dashboardID,
			Name:        "End of Q2",
			Timestamp:   time.Now().AddDate(0, -3, 0),
			CreatedBy:   "data_team_lead",
			Certified:   true,
		},
		{
			ID:          uuid.New(),
			DashboardID: dashboardID,
			Name:        "Post-Campaign Launch",
			Timestamp:   time.Now().AddDate(0, -1, 0),
			CreatedBy:   "patrick",
			Certified:   false,
		},
	}, nil
}

// CreateSnapshot creates a new snapshot for a dashboard. Mock implementation.
func (s *SnapshotService) CreateSnapshot(ctx context.Context, dashboardID uuid.UUID, name, createdBy string) (*models.DashboardSnapshot, error) {
	// In a real app, you would fetch the current state of the dashboard and store it.
	snapshot := &models.DashboardSnapshot{
		ID:          uuid.New(),
		DashboardID: dashboardID,
		Name:        name,
		Timestamp:   time.Now(),
		CreatedBy:   createdBy,
		Certified:   false,
	}
	// In a real app, this would be inserted into the explorer_dashboard_snapshot table.
	return snapshot, nil
}

// CompareSnapshots generates a diff between two snapshots. Mock implementation.
func (s *SnapshotService) CompareSnapshots(ctx context.Context, snapshotID, compareToID uuid.UUID) (*models.SnapshotDiff, error) {
	// Mock diff data
	diff := &models.SnapshotDiff{
		FiltersDiff: []models.SnapshotDiffItem{
			{Field: "region", Before: "['APAC']", After: "['APAC', 'EMEA']", ChangeType: "modified"},
			{Field: "start_date", Before: "2023-04-01", After: "", ChangeType: "removed"},
		},
		MetricsDiff: []models.SnapshotDiffItem{
			{Field: "avg_order_value", Before: "", After: "SUM(price) / COUNT(DISTINCT order_id)", ChangeType: "added"},
		},
		LayoutDiff: []models.SnapshotDiffItem{
			{Field: "Chart: Regional Sales", Before: "Bar Chart", After: "Line Chart", ChangeType: "modified"},
		},
		SemanticDiff: []models.SnapshotDiffItem{
			{Field: "semantic_view", Before: "sales_v1", After: "sales_v2", ChangeType: "modified"},
		},
	}
	return diff, nil
}

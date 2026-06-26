package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
)

// TourService provides methods for managing interactive tours.
type TourService struct {
	db *sqlx.DB
}

// NewTourService creates a new TourService.
func NewTourService(db *sqlx.DB) *TourService {
	return &TourService{db: db}
}

// ListTours retrieves available tours for a user.
func (s *TourService) ListTours(ctx context.Context, userID string) ([]models.Tour, error) {
	// In a real app, this would filter based on user progress and audience.
	mockTours := []models.Tour{
		{ID: uuid.MustParse("10000000-0000-0000-0000-000000000001"), Name: "Build Your First Query", Description: "Learn the basics of the Explorer by building a simple query.", Audience: []string{"new_user"}},
		{ID: uuid.MustParse("10000000-0000-0000-0000-000000000002"), Name: "Save and Share a Workbook", Description: "Discover how to save your work and collaborate with your team.", Audience: []string{"analyst"}},
	}
	return mockTours, nil
}

// GetTour retrieves the full details and steps for a single tour.
func (s *TourService) GetTour(ctx context.Context, tourID string) (*models.FullTour, error) {
	// Mocking steps for the "Build Your First Query" tour
	if tourID == "10000000-0000-0000-0000-000000000001" {
		return &models.FullTour{
			Tour: models.Tour{ID: uuid.MustParse(tourID), Name: "Build Your First Query"},
			Steps: []models.TourStep{
				{Step: 1, TargetSelector: ".view-list", Title: "Select a View", Content: "Views are curated datasets. Click on a view to get started.", Position: "right"},
				{Step: 2, TargetSelector: ".member-browser", Title: "Choose Measures & Dimensions", Content: "Select the metrics (measures) and attributes (dimensions) you want to analyze.", Position: "right"},
				{Step: 3, TargetSelector: ".query-composer-actions button:last-child", Title: "Run the Query", Content: "Click the 'Execute' button to run your query and see the results.", Position: "bottom"},
				{Step: 4, TargetSelector: ".results-tabs", Title: "Explore the Results", Content: "You can view your data as a grid, a visualization, or inspect the generated SQL.", Position: "bottom"},
			},
		}, nil
	}
	return nil, fmt.Errorf("tour not found")
}

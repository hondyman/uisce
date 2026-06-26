package models

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Tour represents the metadata for a guided tour.
type Tour struct {
	ID          uuid.UUID      `db:"id" json:"id"`
	Name        string         `db:"name" json:"name"`
	Description string         `db:"description" json:"description"`
	Audience    pq.StringArray `db:"audience" json:"audience"`
}

// TourStep defines a single step within a tour.
type TourStep struct {
	TargetSelector string `json:"target_selector"`
	Title          string `json:"title"`
	Content        string `json:"content"`
	Position       string `json:"position"` // e.g., 'bottom', 'right', 'left', 'top'
	Step           int    `json:"step"`
}

// FullTour is the complete tour object including all its steps.
type FullTour struct {
	Tour
	Steps []TourStep `json:"steps"`
}

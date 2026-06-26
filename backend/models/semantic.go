package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/cube"
)

// ModelProvider defines the interface for obtaining the semantic model catalog.
type ModelProvider interface {
	GetActiveCatalog(ctx context.Context, tenantID, datasourceID string) (*cube.Catalog, error)
}

// SemanticViewMeta represents a high-level, business-friendly view of data.
type SemanticViewMeta struct {
	ID          uuid.UUID        `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Owner       string           `json:"owner"`
	Certified   bool             `json:"certified"`
	UpdatedAt   time.Time        `json:"updated_at"`
	Dimensions  []SemanticMember `json:"dimensions"`
	Metrics     []SemanticMember `json:"metrics"`
}

// SemanticQuery represents a query built against a semantic view.
type SemanticQuery struct {
	Dimensions []string  `json:"dimensions"`
	Metrics    []string  `json:"metrics"`
	Filters    []Filter  `json:"filters"` // Reusing existing Filter model
	Order      []OrderBy `json:"order"`   // Reusing existing OrderBy model
	Region     string    `json:"region"`
	Limit      int       `json:"limit"`
}

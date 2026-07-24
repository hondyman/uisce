package marketplace

import (
	"context"

	"github.com/google/uuid"
)

type MarketplaceItem struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // Component, Recipe, Micro
	Version     string    `json:"version"`
	Author      string    `json:"author"`
	Description string    `json:"description"`
	Scope       string    `json:"scope"`  // core, tenant
	Config      string    `json:"config"` // JSON payload
}

type Service struct {
	items []MarketplaceItem
}

func NewService() *Service {
	return &Service{
		items: make([]MarketplaceItem, 0),
	}
}

func (s *Service) List(ctx context.Context, scopeFilter string) ([]MarketplaceItem, error) {
	// Mock return
	return []MarketplaceItem{
		{
			ID:          uuid.New(),
			Name:        "Advanced KPI Card",
			Type:        "Micro",
			Version:     "1.0.0",
			Author:      "Core Team",
			Description: "KPI with trend and tooltip",
			Scope:       "core",
		},
	}, nil
}

func (s *Service) Install(ctx context.Context, itemID uuid.UUID, targetTenant string) error {
	// Logic to copy item to tenant's registry
	return nil
}

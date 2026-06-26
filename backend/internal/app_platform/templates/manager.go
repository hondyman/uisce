package templates

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AppTemplate struct {
	ID           uuid.UUID       `json:"id"`
	Name         string          `json:"name"`
	Version      string          `json:"version"`
	Description  string          `json:"description"`
	Pages        []uuid.UUID     `json:"pages"`
	Workflows    []uuid.UUID     `json:"workflows"`
	APIs         []uuid.UUID     `json:"apis"`
	BOs          []uuid.UUID     `json:"bos"`
	Themes       []uuid.UUID     `json:"themes"`
	SLOBundles   []uuid.UUID     `json:"slo_bundles"`
	DataPolicies []uuid.UUID     `json:"data_policies"`
	FeatureFlags map[string]bool `json:"feature_flags"`
	Navigation   json.RawMessage `json:"navigation"`
	CreatedAt    time.Time       `json:"created_at"`
	CreatedBy    string          `json:"created_by"`
	IsCore       bool            `json:"is_core"`
}

type InstallRequest struct {
	TemplateID uuid.UUID              `json:"template_id"`
	TenantID   string                 `json:"tenant_id"`
	Overrides  map[string]interface{} `json:"overrides,omitempty"`
}

type InstallResult struct {
	AppID          uuid.UUID `json:"app_id"`
	InstalledAt    time.Time `json:"installed_at"`
	ObjectsCreated int       `json:"objects_created"`
}

type TemplateManager struct {
	// DB access would go here
}

func NewTemplateManager() *TemplateManager {
	return &TemplateManager{}
}

func (m *TemplateManager) CreateTemplate(ctx context.Context, template *AppTemplate) error {
	// Mock: Save template to DB
	template.ID = uuid.New()
	template.CreatedAt = time.Now()
	return nil
}

func (m *TemplateManager) ListTemplates(ctx context.Context, tenantID string) ([]AppTemplate, error) {
	// Mock: Return sample templates
	return []AppTemplate{
		{
			ID:          uuid.New(),
			Name:        "Wealth Management App",
			Version:     "1.0.0",
			Description: "Complete wealth management application with positions, accounts, and trading.",
			IsCore:      true,
		},
		{
			ID:          uuid.New(),
			Name:        "KYC Onboarding App",
			Version:     "1.0.0",
			Description: "Regulatory-compliant KYC and client onboarding workflow.",
			IsCore:      true,
		},
	}, nil
}

func (m *TemplateManager) InstallTemplate(ctx context.Context, req *InstallRequest) (*InstallResult, error) {
	// Mock: Install template for tenant
	// Real: Clone all objects, apply tenant overlays, register navigation
	result := &InstallResult{
		AppID:          uuid.New(),
		InstalledAt:    time.Now(),
		ObjectsCreated: 15, // Mock count
	}
	return result, nil
}

func (m *TemplateManager) ExportTemplate(ctx context.Context, appID uuid.UUID) (*AppTemplate, error) {
	// Mock: Export existing app as template
	template := &AppTemplate{
		ID:          uuid.New(),
		Name:        "Exported App",
		Version:     "1.0.0",
		Description: "Exported from existing app",
		Pages:       []uuid.UUID{uuid.New(), uuid.New()},
		APIs:        []uuid.UUID{uuid.New()},
	}
	return template, nil
}

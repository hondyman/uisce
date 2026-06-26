package tenant

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"cube-gonja/config"
	"cube-gonja/internal/render"
)

type Tenant struct {
	ID            string
	Name          string
	TemplateDir   string
	OutputDir     string
	RenderService *render.Service
	DataSources   map[string]struct{}
	APIKey        string
}

type Manager struct {
	mu              sync.RWMutex
	tenants         map[string]*Tenant
	baseDir         string
	baseTemplateDir string
	config          config.Config
}

func NewManager(cfg config.Config) *Manager {
	return &Manager{
		tenants: make(map[string]*Tenant),
		baseDir: cfg.TenantBaseDir,
		config:  cfg,
	}
}

func (m *Manager) SetBaseTemplateDir(baseTemplateDir string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.baseTemplateDir = baseTemplateDir
}

func (m *Manager) InitializeTenant(tenantID, name, apiKey string) (*Tenant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if tenant, exists := m.tenants[tenantID]; exists {
		return tenant, nil
	}

	// Create tenant directories
	tenantDir := filepath.Join(m.baseDir, tenantID)
	templateDir := filepath.Join(tenantDir, "templates")
	outputDir := filepath.Join(tenantDir, "output")

	for _, dir := range []string{tenantDir, templateDir, outputDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create tenant dir %s: %v", dir, err)
		}
	}

	// Initialize render service for tenant
	renderSvc := render.NewService(templateDir, m.baseTemplateDir, outputDir, m.config.AllowedDataSource)

	tenant := &Tenant{
		ID:            tenantID,
		Name:          name,
		TemplateDir:   templateDir,
		OutputDir:     outputDir,
		RenderService: renderSvc,
		DataSources:   make(map[string]struct{}),
		APIKey:        apiKey,
	}

	m.tenants[tenantID] = tenant
	return tenant, nil
}

func (m *Manager) GetTenant(tenantID string) (*Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenant, exists := m.tenants[tenantID]
	if !exists {
		return nil, fmt.Errorf("tenant %s not found", tenantID)
	}
	return tenant, nil
}

func (m *Manager) ListTenants() []*Tenant {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenants := make([]*Tenant, 0, len(m.tenants))
	for _, tenant := range m.tenants {
		tenants = append(tenants, tenant)
	}
	return tenants
}

func (m *Manager) ValidateTenantAccess(tenantID, apiKey string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenant, exists := m.tenants[tenantID]
	if !exists {
		return false
	}

	if m.config.RequireAuth && tenant.APIKey != "" {
		return tenant.APIKey == apiKey
	}

	return true
}

func (m *Manager) GetDefaultTenant() (*Tenant, error) {
	return m.GetTenant(m.config.DefaultTenant)
}

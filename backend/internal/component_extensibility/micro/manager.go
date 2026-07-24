package micro

import (
	"context"
)

type ComponentNode struct {
	Type     string            `json:"type"`
	Props    map[string]string `json:"props"` // Simplified props map
	Children []ComponentNode   `json:"children,omitempty"`
}

type MicroComponent struct {
	ID       string        `json:"id"`
	Type     string        `json:"type"` // "Composite"
	RootNode ComponentNode `json:"root_node"`
	TenantID string        `json:"tenant_id"`
}

type CompositeManager struct {
	components map[string]MicroComponent
}

func NewCompositeManager() *CompositeManager {
	return &CompositeManager{
		components: make(map[string]MicroComponent),
	}
}

func (m *CompositeManager) Save(ctx context.Context, comp MicroComponent) error {
	m.components[comp.ID] = comp
	return nil
}

// Expand resolves specific micro-components into their primitive structure
func (m *CompositeManager) Expand(ctx context.Context, componentID string) (*ComponentNode, error) {
	comp, ok := m.components[componentID]
	if !ok {
		return nil, nil
	}
	return &comp.RootNode, nil
}

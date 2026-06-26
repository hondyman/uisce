package recipes

import (
	"context"
	"encoding/json"
)

type ComponentDef struct {
	Type     string          `json:"type"`
	Required bool            `json:"required"`
	Props    json.RawMessage `json:"props,omitempty"`
}

type LayoutDef struct {
	Root     string   `json:"root"` // row, column
	Children []string `json:"children"`
}

type Recipe struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Components  []ComponentDef    `json:"components"`
	Layout      LayoutDef         `json:"layout"`
	Bindings    map[string]string `json:"bindings"` // "table.rows": "source.data"
	IsCore      bool              `json:"is_core"`
	TenantID    string            `json:"tenant_id,omitempty"`
}

type Registry struct {
	// In-memory or DB
	recipes map[string]Recipe
}

func NewRegistry() *Registry {
	return &Registry{
		recipes: make(map[string]Recipe),
	}
}

func (r *Registry) Register(ctx context.Context, recipe Recipe) error {
	r.recipes[recipe.ID] = recipe
	return nil
}

func (r *Registry) List(ctx context.Context, tenantID string) ([]Recipe, error) {
	list := make([]Recipe, 0)
	for _, v := range r.recipes {
		if v.IsCore || v.TenantID == tenantID {
			list = append(list, v)
		}
	}
	return list, nil
}

func (r *Registry) Get(ctx context.Context, id string) (*Recipe, error) {
	if v, ok := r.recipes[id]; ok {
		return &v, nil
	}
	return nil, nil // Error in real impl
}

// Instantiate creates a page/component structure from a recipe
// This acts as a factory
func (r *Registry) Instantiate(ctx context.Context, recipeID string) (json.RawMessage, error) {
	recipe, _ := r.Get(ctx, recipeID)
	if recipe == nil {
		return nil, nil
	}

	// Mock instantiation logic: just return the recipe as a "component" block for now
	return json.Marshal(map[string]interface{}{
		"type": "RecipeInstance",
		"props": map[string]interface{}{
			"recipeId": recipeID,
			"layout":   recipe.Layout,
		},
	})
}

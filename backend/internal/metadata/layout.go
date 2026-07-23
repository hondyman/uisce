package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ViewDefinition represents a UI layout configuration
type ViewDefinition struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	TenantID    uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	Slug        string          `db:"slug" json:"slug"`
	Version     int             `db:"version" json:"version"`
	Title       string          `db:"title" json:"title"`
	LayoutJSON  json.RawMessage `db:"layout_json" json:"layout_json"`
	AllowedRoles []string       `db:"allowed_roles" json:"allowed_roles"` // Handled as Postgres Array
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
}

type LayoutService struct {
	db *sqlx.DB
}

func NewLayoutService(db *sqlx.DB) *LayoutService {
	return &LayoutService{db: db}
}

// GetLayoutBySlug fetches the latest version of a layout for a given slug
func (s *LayoutService) GetLayoutBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*ViewDefinition, error) {
	var layout ViewDefinition
	// Simple query: get latest version
	query := `
		SELECT * FROM view_definitions 
		WHERE tenant_id = $1 AND slug = $2 
		ORDER BY version DESC 
		LIMIT 1
	`
	err := s.db.GetContext(ctx, &layout, query, tenantID, slug)
	if err != nil {
		return nil, fmt.Errorf("layout not found: %w", err)
	}
	return &layout, nil
}

// CreateLayout persists a new layout definition
func (s *LayoutService) CreateLayout(ctx context.Context, tenantID uuid.UUID, slug, title string, layoutJSON map[string]interface{}) (*ViewDefinition, error) {
	layoutBytes, _ := json.Marshal(layoutJSON)
	
	def := &ViewDefinition{
		TenantID:   tenantID,
		Slug:       slug,
		Version:    1, // Simplified versioning logic
		Title:      title,
		LayoutJSON: layoutBytes,
	}

	query := `
		INSERT INTO view_definitions (tenant_id, slug, version, title, layout_json)
		VALUES (:tenant_id, :slug, :version, :title, :layout_json)
		RETURNING id, created_at, updated_at
	`
	rows, err := s.db.NamedQueryContext(ctx, query, def)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		rows.Scan(&def.ID, &def.CreatedAt, &def.UpdatedAt)
	}

	return def, nil
}

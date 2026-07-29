package agentic

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type ToolEntry struct {
	ToolName     string   `db:"tool_name"`
	DisplayName  string   `db:"display_name"`
	Description  string   `db:"description"`
	AllowedRoles []string `db:"allowed_roles"`
}

type MCPRegistryService struct {
	db *sqlx.DB
}

func NewMCPRegistryService(db *sqlx.DB) *MCPRegistryService {
	return &MCPRegistryService{db: db}
}

func (s *MCPRegistryService) ListToolsForRole(ctx context.Context, tenantID, functionalRole string) ([]ToolEntry, error) {
	var all []ToolEntry
	query := `
		SELECT tool_name, display_name, description, allowed_roles
		FROM mcp_tool_registry
		WHERE is_active = true
		  AND tenant_id IN ($1, '00000000-0000-0000-0000-000000000000')
		ORDER BY tool_name
	`
	rows, err := s.db.QueryxContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t ToolEntry
		if err := rows.StructScan(&t); err != nil {
			return nil, err
		}
		all = append(all, t)
	}

	if functionalRole == "" {
		return all, nil
	}

	var filtered []ToolEntry
	for _, t := range all {
		for _, role := range t.AllowedRoles {
			if role == functionalRole {
				filtered = append(filtered, t)
				break
			}
		}
	}
	return filtered, nil
}

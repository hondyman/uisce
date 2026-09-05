// backend/internal/api/relationship_suggestions.go
package api

import (
	"database/sql"
)

// RelationshipService provides relationship discovery and management
type RelationshipService struct {
	db *sql.DB
}

func NewRelationshipService(db *sql.DB) *RelationshipService {
	return &RelationshipService{db: db}
}

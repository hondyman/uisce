package hierarchy

import "github.com/jmoiron/sqlx"

// NewHierarchyServiceSQLX constructs the sqlx-backed hierarchy Service and returns it as the Service interface.
func NewHierarchyServiceSQLX(db *sqlx.DB) Service {
	return NewHierarchyServiceSQLXImpl(db)
}

// For compatibility with callers expecting the concrete type name, provide a type alias.
type HierarchyServiceSQLX = HierarchySQLXServiceImpl

package graphql

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type ABAC interface {
	Can(ctx context.Context, action, resource string, attrs map[string]interface{}) bool
}

type Resolver struct {
	DB   *sqlx.DB
	ABAC ABAC
}

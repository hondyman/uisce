package api_test

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	httpapi "github.com/hondyman/uisce/backend/internal/api"
)

func newIntegrationValidationRouter(db *sql.DB) http.Handler {
	r := chi.NewRouter()
	httpapi.RegisterValidationRulesRoutes(r, db, nil, &mockResolver{})
	return r
}

// Tests removed as they relied on Starlark execution which is no longer supported via API.
// Future tests should use CUE or ASL integration.

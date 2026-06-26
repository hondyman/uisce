package api_test

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	httpapi "github.com/hondyman/semlayer/backend/internal/api"
	"github.com/hondyman/semlayer/backend/internal/services"
)

func newIntegrationValidationRouter(db *sql.DB) http.Handler {
	r := chi.NewRouter()
	httpapi.RegisterValidationRulesRoutes(r, db, services.NewCueEngine(), nil, &mockResolver{})
	return r
}

// Tests removed as they relied on Starlark execution which is no longer supported via API.
// Future tests should use CUE or ASL integration.

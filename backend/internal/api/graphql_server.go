package api

import (
	"fmt"
	"net/http"

	chi "github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

// RegisterGraphQLAPI registers the GraphQL endpoint
// For now, this creates a simple proxy to Hasura
func RegisterGraphQLAPI(r chi.Router, db *sqlx.DB) {
	// Proxy to Hasura GraphQL endpoint
	r.Post("/graphql", func(w http.ResponseWriter, req *http.Request) {
		// TODO: Implement Hasura proxy
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"error": "GraphQL proxy not yet implemented"}`)
	})
}

// RegisterGraphQLPlayground registers the public GraphQL playground endpoint
func RegisterGraphQLPlayground(r *chi.Mux) {
	r.Get("/playground", func(w http.ResponseWriter, req *http.Request) {
		// Redirect to Hasura playground
		http.Redirect(w, req, "http://hasura:8080/console", http.StatusTemporaryRedirect)
	})
}

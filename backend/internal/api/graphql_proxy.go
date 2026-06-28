package api

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
)

func registerGraphQLProxyRoutesV2(r chi.Router) {
	hasuraURL := os.Getenv("HASURA_URL")
	if hasuraURL == "" {
		hasuraURL = "http://hasura:8080"
	}

	if !strings.HasSuffix(hasuraURL, "/v1/graphql") {
		hasuraURL = strings.TrimSuffix(hasuraURL, "/") + "/v1/graphql"
	}

	target, err := url.Parse(hasuraURL)
	if err != nil {
		log.Printf("Failed to parse Hasura URL %q: %v", hasuraURL, err)
		return
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = target.Path
			req.URL.RawQuery = target.RawQuery
			req.Host = target.Host

			// Attach admin secret server-side if set
			if secret := os.Getenv("HASURA_ADMIN_SECRET"); secret != "" {
				req.Header.Set("X-Hasura-Admin-Secret", secret)
			}

			// Propagate identity for Hasura session variables
			if tenantID := req.Header.Get("X-Tenant-ID"); tenantID != "" {
				req.Header.Set("X-Hasura-Tenant-Id", tenantID)
			}
			if userID := req.Header.Get("X-User-ID"); userID != "" {
				req.Header.Set("X-Hasura-User-Id", userID)
				req.Header.Set("X-Hasura-Role", "user") // Default role, adjust if needed
			}
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("GraphQL proxy error: %v", err)
			http.Error(w, "GraphQL service unavailable", http.StatusServiceUnavailable)
		},
	}

	r.Route("/v1", func(rr chi.Router) {
		rr.Handle("/graphql", proxy)
	})
}

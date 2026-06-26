package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// LayoutSchema represents the JSON schema for the UI layout
type LayoutSchema struct {
	Components []Component `json:"components"`
}

// Component represents a UI component
type Component struct {
	Type  string            `json:"type"`
	Props map[string]string `json:"props"`
}

func main() {
	http.HandleFunc("/layout", resolveIntent)
	log.Println("GenUI API listening on :8080")

	// wrap default mux with JWT middleware (only /layout requires auth)
	jwtMw := jwtmiddleware.NewJWTMiddleware("/health")
	handler := jwtMw.Handler(http.DefaultServeMux)

	log.Fatal(http.ListenAndServe(":8080", handler))
}

func resolveIntent(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	nlQuery := r.URL.Query().Get("q")
	log.Printf("Received query: %s for tenant: %s", nlQuery, tenantID)

	// Mock AI/Intent resolution logic
	layout := generateLayout(nlQuery)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(layout)
}

func generateLayout(query string) LayoutSchema {
	// In a real implementation, this would call an LLM or look up metadata
	if strings.Contains(strings.ToLower(query), "rebalance") {
		return LayoutSchema{
			Components: []Component{
				{Type: "Header", Props: map[string]string{"title": "Portfolio Rebalance"}},
				{Type: "RebalanceForm", Props: map[string]string{"target": "Conservative"}},
			},
		}
	}
	return LayoutSchema{
		Components: []Component{
			{Type: "Header", Props: map[string]string{"title": "Dashboard"}},
			{Type: "SummaryCard", Props: map[string]string{"metric": "AUM"}},
		},
	}
}

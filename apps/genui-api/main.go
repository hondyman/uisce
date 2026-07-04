package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

type LayoutSchema struct {
	Components []Component `json:"components"`
}

type Component struct {
	Type  string            `json:"type"`
	Props map[string]string `json:"props"`
}

func main() {
	http.HandleFunc("/layout", resolveIntent)
	log.Println("GenUI API listening on :8080")

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

	layout := generateLayout(nlQuery)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(layout)
}

func generateLayout(query string) LayoutSchema {
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

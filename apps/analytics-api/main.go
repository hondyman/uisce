package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/query", queryAnalytics)
	log.Println("Analytics API listening on :8082")

	jwtMw := jwtmiddleware.NewJWTMiddleware("/health")
	handler := jwtMw.Handler(http.DefaultServeMux)
	log.Fatal(http.ListenAndServe(":8082", handler))
}

func queryAnalytics(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	sqlQuery := r.URL.Query().Get("sql")
	log.Printf("Executing query for tenant %s: %s", tenantID, sqlQuery)

	// Mock analytics result
	results := []map[string]interface{}{
		{"date": "2023-10-26", "value": 1000},
		{"date": "2023-10-27", "value": 1050},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

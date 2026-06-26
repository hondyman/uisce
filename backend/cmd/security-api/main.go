package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/api"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	securityService := security.NewAccessRuleService(db)
	securityHandler := api.NewSecurityRulesHandler(securityService)

	router := chi.NewRouter()
	router.Route("/api", func(r chi.Router) {
		securityHandler.RegisterRoutes(r)
	})

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	handler := corsMiddleware(router)

	addr := fmt.Sprintf(":%s", port)
	fmt.Printf("Security Rules API Server starting on %s\n", addr)
	fmt.Println("Endpoints:")
	fmt.Println("  GET    /api/security/rules")
	fmt.Println("  POST   /api/security/rules")
	fmt.Println("  GET    /api/security/rules/:id")
	fmt.Println("  PUT    /api/security/rules/:id")
	fmt.Println("  POST   /api/security/rules/validate")
	fmt.Println("  GET    /api/security/rules/:id/impact")

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

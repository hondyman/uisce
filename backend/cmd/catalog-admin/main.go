package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/api"
	"github.com/hondyman/uisce/backend/internal/tenant"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	dbRaw, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open raw sql connection: %v", err)
	}
	defer dbRaw.Close()

	tenantMgr := tenant.NewTenantManager(dbRaw)

	router := chi.NewRouter()
	router.Route("/api", func(r chi.Router) {
		api.RegisterCatalogAdminRoutes(r, dbRaw, tenantMgr)
	})

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	addr := fmt.Sprintf(":%s", port)
	fmt.Printf("Catalog Admin API starting on %s\n", addr)
	fmt.Println("Endpoints:")
	fmt.Println("  POST   /api/catalog/admin/sync-subtypes")

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

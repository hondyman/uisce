package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/hondyman/uisce/backend/internal/api"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" || dbURL == "<VALUE_TO_BE_PROVIDED>" {
		dbURL = os.Getenv("POSTGRES_DSN")
	}
	if dbURL == "" || dbURL == "<VALUE_TO_BE_PROVIDED>" {
		dbURL = "postgresql://postgres:postgres@100.84.50.65:5432/alpha?sslmode=disable"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Connected to database successfully")

	sqlxDB := sqlx.NewDb(db, "postgres")
	_ = sqlxDB

	router := api.SetupRouter(db, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting main Uisce Unified API server on %s...\n", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}

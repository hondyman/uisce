package main

import (
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// getEnv returns the value of the environment variable if set, otherwise returns the default value.
func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

// ConnectSQLX creates a sqlx DB using the same env vars as GORM-based code.
func ConnectSQLX() (*sqlx.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("DB_HOST", "100.84.126.19"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "postgres"),
		getEnv("DB_NAME", "alpha"),
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// Optional: set reasonable defaults
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	// If running in short-lived contexts, you could ping to verify
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Expose DB via env if needed
	_ = os.Setenv("SQLX_CONNECTED", "true")
	return db, nil
}

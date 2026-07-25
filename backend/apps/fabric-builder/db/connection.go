package db

import (
"database/sql"
"fmt"
"os"

_ "github.com/lib/pq"
)

// NewConnection creates a new PostgreSQL database connection
func NewConnection() (*sql.DB, error) {
dsn := os.Getenv("DATABASE_URL")
if dsn == "" {
// Default connection string
dsn = "host=localhost port=5432 user=postgres password=postgres dbname=semlayer sslmode=disable"
}

db, err := sql.Open("postgres", dsn)
if err != nil {
return nil, fmt.Errorf("failed to open database: %w", err)
}

// Test the connection
if err := db.Ping(); err != nil {
return nil, fmt.Errorf("failed to ping database: %w", err)
}

// Set connection pool settings
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)

return db, nil
}

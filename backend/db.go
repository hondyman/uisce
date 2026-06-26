package backend

import (
	"database/sql"
	"fmt"
	"net/url"

	"github.com/hondyman/semlayer/backend/internal/config"
	_ "github.com/lib/pq"
)

// GetAppDBConnection establishes a connection to a specific database using the application's configuration.
// If dbName is empty, it uses the default database from the DSN.
func GetAppDBConnection(dbName string) (*sql.DB, error) {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	dsn := cfg.DSN
	if dbName != "" {
		parsedURL, err := url.Parse(dsn)
		if err != nil {
			return nil, fmt.Errorf("could not parse DSN from config: %w", err)
		}
		parsedURL.Path = "/" + dbName
		dsn = parsedURL.String()
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		pingDbName := dbName
		if pingDbName == "" {
			pingDbName = "default"
		}
		return nil, fmt.Errorf("failed to ping database '%s': %w", pingDbName, err)
	}

	return db, nil
}

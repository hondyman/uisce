package discovery

import (
	"log"
	"os"
	"testing"
)

var testLogger = log.New(os.Stdout, "DISCOVERY_TEST: ", log.LstdFlags)

// Test: Schema scanner initialization
func TestNewSchemaScanner(t *testing.T) {
	config := SchemaScannerConfig{
		PostgresDBs: []string{"semlayer", "analytics"},
		TrinoDBs:    []string{"warehouse"},
		S3Buckets:   []string{"data-lake"},
	}

	scanner := NewSchemaScanner(nil, nil, config, testLogger)

	if scanner == nil {
		t.Fatal("SchemaScanner should not be nil")
	}
	if len(scanner.config.PostgresDBs) != 2 {
		t.Errorf("Expected 2 Postgres DBs, got %d", len(scanner.config.PostgresDBs))
	}
}

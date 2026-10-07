package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"
)

func getAlphaAdminDSN() string {
	home, _ := os.UserHomeDir()
	caPath := filepath.Join(home, ".uisce/certs/ca.crt")
	certPath := filepath.Join(home, ".uisce/certs/postgres-client.crt")
	keyPath := filepath.Join(home, ".uisce/certs/postgres-client.key")

	return fmt.Sprintf("host=100.84.50.65 port=5432 user=postgres password=postgres dbname=alpha sslmode=verify-full sslrootcert=%s sslcert=%s sslkey=%s", caPath, certPath, keyPath)
}

func main() {
	db, err := sql.Open("postgres", getAlphaAdminDSN())
	if err != nil {
		log.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sqlCmds := `
		ALTER ROLE uisce_mcp_app WITH LOGIN PASSWORD 'uisce_mcp_secure_2026' NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
		GRANT USAGE ON SCHEMA compliance TO uisce_mcp_app;
		GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA compliance TO uisce_mcp_app;
		GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA compliance TO uisce_mcp_app;
		GRANT USAGE ON SCHEMA public TO uisce_mcp_app;
		GRANT SELECT ON ALL TABLES IN SCHEMA public TO uisce_mcp_app;
	`
	_, err = db.ExecContext(ctx, sqlCmds)
	if err != nil {
		log.Fatalf("Exec failed: %v", err)
	}
	fmt.Println("Role uisce_mcp_app password and schema grants configured successfully.")
}

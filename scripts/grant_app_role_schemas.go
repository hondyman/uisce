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
		GRANT ALL ON DATABASE alpha TO uisce_mcp_app;
		GRANT ALL ON SCHEMA public, oms, altinv, cash_flow, master, compliance TO uisce_mcp_app;
		GRANT ALL ON ALL TABLES IN SCHEMA public, oms, altinv, cash_flow, master, compliance TO uisce_mcp_app;
		GRANT ALL ON ALL SEQUENCES IN SCHEMA public, oms, altinv, cash_flow, master, compliance TO uisce_mcp_app;
		GRANT ALL ON ALL FUNCTIONS IN SCHEMA public, oms, altinv, cash_flow, master, compliance TO uisce_mcp_app;
	`
	_, err = db.ExecContext(ctx, sqlCmds)
	if err != nil {
		log.Fatalf("Exec failed: %v", err)
	}
	fmt.Println("Grants updated for uisce_mcp_app successfully.")
}

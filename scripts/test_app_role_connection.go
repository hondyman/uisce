package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/lib/pq"
)

func main() {
	home, _ := os.UserHomeDir()
	caPath := filepath.Join(home, ".uisce/certs/ca.crt")
	certPath := filepath.Join(home, ".uisce/certs/postgres-client.crt")
	keyPath := filepath.Join(home, ".uisce/certs/postgres-client.key")

	dsn := fmt.Sprintf("host=100.84.50.65 port=5432 user=uisce_mcp_app password=uisce_mcp_secure_2026 dbname=alpha sslmode=verify-full sslrootcert=%s sslcert=%s sslkey=%s", caPath, certPath, keyPath)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	var currentUser string
	var isSuper, bypassRLS bool
	err = db.QueryRow("SELECT current_user, rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user").Scan(&currentUser, &isSuper, &bypassRLS)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	fmt.Printf("Connected successfully as: %s (super=%v, bypassrls=%v)\n", currentUser, isSuper, bypassRLS)
}

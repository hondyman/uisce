package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		panic("DATABASE_URL environment variable is required")
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check catalog_nodes schema for id type
	var idType string
	err = db.QueryRow("SELECT data_type FROM information_schema.columns WHERE table_name = 'catalog_nodes' AND column_name = 'id'").Scan(&idType)
	if err != nil {
		log.Printf("Error checking catalog_nodes schema: %v", err)
	} else {
		fmt.Printf("catalog_nodes.id type: %s\n", idType)
	}

	// Check catalog_nodes table content
	rows, err := db.Query("SELECT id, name FROM catalog_nodes LIMIT 5")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Sample catalog_nodes:")
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("- %s (%s)\n", id, name)
	}
}

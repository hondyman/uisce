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
		log.Fatal("DATABASE_URL environment variable is required")
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT column_name, data_type, is_nullable FROM information_schema.columns WHERE table_name = 'business_objects' ORDER BY ordinal_position")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Schema of business_objects:")
	count := 0
	for rows.Next() {
		var colName, dataType, isNullable string
		if err := rows.Scan(&colName, &dataType, &isNullable); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("- %s (%s) Nullable: %s\n", colName, dataType, isNullable)
		count++
	}
	if count == 0 {
		fmt.Println("Table business_objects does not exist!")
	}
}

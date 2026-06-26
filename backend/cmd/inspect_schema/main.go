package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:5432/semlayer?sslmode=disable")
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

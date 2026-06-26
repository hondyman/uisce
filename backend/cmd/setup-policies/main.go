package main

import (
	"log"

	"github.com/hondyman/semlayer/backend"
	_ "github.com/lib/pq"
)

// CreatePoliciesTable creates the policies table in the database.
func main() {
	if err := backend.CreatePoliciesTable(); err != nil {
		log.Fatal(err)
	}
}

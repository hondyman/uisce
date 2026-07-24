package main

import (
	"log"

	"github.com/hondyman/uisce/backend"
	_ "github.com/lib/pq"
)

func main() {
	if err := backend.CheckPolicyTables(); err != nil {
		log.Fatal(err)
	}
}

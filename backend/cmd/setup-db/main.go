package main

import (
	"log"

	"github.com/hondyman/uisce/backend"
)

func CreateDB() error {
	return backend.CreateDB()
}

func main() {
	if err := CreateDB(); err != nil {
		log.Fatal(err)
	}
}

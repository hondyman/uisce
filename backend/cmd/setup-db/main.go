package main

import (
	"log"

	"github.com/hondyman/semlayer/backend"
)

func CreateDB() error {
	return backend.CreateDB()
}

func main() {
	if err := CreateDB(); err != nil {
		log.Fatal(err)
	}
}

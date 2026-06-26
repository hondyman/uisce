package main

import (
	"fmt"
	"log"

	"github.com/hondyman/semlayer/backend/internal/config"
)

func main() {
	cfg := config.LoadDefaultConfig()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Config validation failed: %v", err)
	}
	fmt.Println("Config loaded and validated successfully!")
	fmt.Printf("Database URL: %s\n", cfg.DatabaseURL)
	fmt.Printf("HTTP Port: %s\n", cfg.HTTPPort)
	fmt.Printf("Metrics Enabled: %t\n", cfg.MetricsEnabled)
}

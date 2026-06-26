package commands

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hondyman/semlayer/backend/internal/semctl/config"
	"gopkg.in/yaml.v3"
)

func RunInit(args []string) {
	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	// Add flags if needed, e.g. --overwrite
	initCmd.Parse(args)

	// Check if .sem exists
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	semDir := filepath.Join(cwd, ".sem")
	if _, err := os.Stat(semDir); !os.IsNotExist(err) {
		fmt.Println(".sem directory already exists")
		return // Or prompt to overwrite
	}

	if err := os.Mkdir(semDir, 0755); err != nil {
		fmt.Printf("Error creating .sem directory: %v\n", err)
		os.Exit(1)
	}

	// Create default config
	cfg := config.Config{
		Envs: map[string]config.EnvConfig{
			"dev":  {URL: "http://localhost:8080"},
			"prod": {URL: "https://api.yourorg.com"},
		},
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		fmt.Printf("Error marshaling config: %v\n", err)
		os.Exit(1)
	}

	configPath := filepath.Join(semDir, "config.yaml")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		fmt.Printf("Error writing config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Initialized semantic project in %s\n", semDir)
}

package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/hondyman/semlayer/backend/internal/semctl/config"
)

func RunPull(args []string) {
	pullCmd := flag.NewFlagSet("pull", flag.ExitOnError)
	envFlag := pullCmd.String("env", "dev", "Environment to pull from")
	outFlag := pullCmd.String("out", "./semantic", "Output directory")
	pullCmd.Parse(args)

	// Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	envCfg, ok := cfg.Envs[*envFlag]
	if !ok {
		fmt.Printf("Environment '%s' not found in config\n", *envFlag)
		os.Exit(1)
	}
	baseURL := envCfg.URL

	// Create directories
	apiDir := filepath.Join(*outFlag, "apis")
	pageDir := filepath.Join(*outFlag, "pages")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		fmt.Printf("Error creating directory %s: %v\n", apiDir, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(pageDir, 0755); err != nil {
		fmt.Printf("Error creating directory %s: %v\n", pageDir, err)
		os.Exit(1)
	}

	// Pull APIs
	fmt.Printf("Pulling APIs from %s...\n", *envFlag)
	if err := fetchAndSave(baseURL+"/api/api-studio/endpoints?env="+*envFlag, filepath.Join(apiDir, "endpoints.json")); err != nil {
		fmt.Printf("Error pulling APIs: %v\n", err)
	}

	// Pull Pages
	fmt.Printf("Pulling Pages from %s...\n", *envFlag)
	if err := fetchAndSave(baseURL+"/api/page-studio/pages?env="+*envFlag, filepath.Join(pageDir, "pages.json")); err != nil {
		fmt.Printf("Error pulling Pages: %v\n", err)
	}

	fmt.Println("Pull complete.")
}

func fetchAndSave(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Pretty print JSON
	var jsonObj interface{}
	if err := json.Unmarshal(body, &jsonObj); err != nil {
		return fmt.Errorf("invalid json response: %w", err)
	}

	formatted, err := json.MarshalIndent(jsonObj, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(dest, formatted, 0644)
}

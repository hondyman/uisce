package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hondyman/semlayer/backend/internal/apistudio"
	"github.com/hondyman/semlayer/backend/internal/pagestudio"
)

func RunValidate(args []string) {
	validateCmd := flag.NewFlagSet("validate", flag.ExitOnError)
	dirFlag := validateCmd.String("dir", "./semantic", "Local directory to validate")
	validateCmd.Parse(args)

	hasError := false

	// Validate APIs
	apiPath := filepath.Join(*dirFlag, "apis", "endpoints.json")
	if files, err := os.ReadDir(filepath.Join(*dirFlag, "apis")); err == nil && len(files) > 0 {
		fmt.Println("Validating APIs...")
		// Assuming single file or multiple. Pull creates single file.
		if _, err := os.Stat(apiPath); err == nil {
			if err := validateEndpoints(apiPath); err != nil {
				fmt.Printf(" [x] API Validation Failed: %v\n", err)
				hasError = true
			} else {
				fmt.Println(" [v] APIs Valid")
			}
		}
	}

	// Validate Pages
	pagePath := filepath.Join(*dirFlag, "pages", "pages.json")
	if _, err := os.Stat(pagePath); err == nil {
		fmt.Println("Validating Pages...")
		if err := validatePages(pagePath); err != nil {
			fmt.Printf(" [x] Page Validation Failed: %v\n", err)
			hasError = true
		} else {
			fmt.Println(" [v] Pages Valid")
		}
	}

	if hasError {
		os.Exit(1)
	}
}

func validateEndpoints(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var eps []apistudio.APIEndpoint
	if err := json.Unmarshal(data, &eps); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}
	for _, ep := range eps {
		if ep.Name == "" || ep.Path == "" {
			return fmt.Errorf("endpoint missing name or path: %+v", ep)
		}
	}
	return nil
}

func validatePages(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var pages []pagestudio.CorePage
	if err := json.Unmarshal(data, &pages); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}
	for _, p := range pages {
		if p.Slug == "" || p.Name == "" {
			return fmt.Errorf("page missing slug or name: %+v", p)
		}
	}
	return nil
}

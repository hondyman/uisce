package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hondyman/uisce/backend/internal/apistudio"
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
		if _, err := os.Stat(apiPath); err == nil {
			if err := validateEndpoints(apiPath); err != nil {
				fmt.Printf(" [x] API Validation Failed: %v\n", err)
				hasError = true
			} else {
				fmt.Println(" [v] APIs Valid")
			}
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

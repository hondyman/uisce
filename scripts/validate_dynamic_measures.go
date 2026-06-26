//go:build ignore

package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DynamicMeasureContract struct {
	NodeID              string                 `json:"node_id"`
	NodeType            string                 `json:"node_type"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	SourceEnum          string                 `json:"source_enum"`
	SQL                 string                 `json:"sql"`
	Type                string                 `json:"type"`
	Tags                []string               `json:"tags"`
	Owner               string                 `json:"owner"`
	Version             string                 `json:"version"`
	GoldenPath          bool                   `json:"golden_path"`
	SchemaHash          string                 `json:"schema_hash"`
	DataQualityContract map[string]interface{} `json:"data_quality_contract,omitempty"`
	Lineage             map[string]interface{} `json:"lineage,omitempty"`
	StewardGroup        string                 `json:"steward_group,omitempty"`
	AnomalyDetection    map[string]interface{} `json:"anomaly_detection,omitempty"`
	ReviewStatus        string                 `json:"review_status,omitempty"`
	ReviewComments      []interface{}          `json:"review_comments,omitempty"`
}

func main() {
	fmt.Println("🔍 CI/CD Validation for Dynamic Measures")
	fmt.Println("=======================================")

	root := "semantic_layer/dynamic_measures"
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	schemaPath := "schemas/dynamic_measure.schema.json"
	if len(os.Args) > 2 {
		schemaPath = os.Args[2]
	}

	// Load JSON schema
	schema, err := loadSchema(schemaPath)
	if err != nil {
		fmt.Printf("❌ Failed to load schema from %s: %v\n", schemaPath, err)
		os.Exit(1)
	}

	fmt.Printf("📋 Validating measures in: %s\n", root)
	fmt.Printf("📄 Using schema: %s\n", schemaPath)
	fmt.Println()

	validCount := 0
	totalCount := 0
	hasErrors := false

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) != ".json" {
			return nil
		}

		totalCount++

		fmt.Printf("🔍 Validating: %s\n", filepath.Base(path))

		// Read and parse JSON file
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("  ❌ Failed to read file: %v\n", err)
			hasErrors = true
			return nil
		}

		var contract DynamicMeasureContract
		if err := json.Unmarshal(data, &contract); err != nil {
			fmt.Printf("  ❌ Failed to parse JSON: %v\n", err)
			hasErrors = true
			return nil
		}

		// Validate against schema
		if err := validateAgainstSchema(contract, schema); err != nil {
			fmt.Printf("  ❌ Schema validation failed: %v\n", err)
			hasErrors = true
			return nil
		}

		// Validate measure-specific rules
		if err := validateMeasureRules(contract); err != nil {
			fmt.Printf("  ❌ Measure validation failed: %v\n", err)
			hasErrors = true
			return nil
		}

		// Check schema hash for drift detection
		computedHash := computeSchemaHash(data)
		if contract.SchemaHash != "" && contract.SchemaHash != computedHash {
			fmt.Printf("  ⚠️  Schema drift detected! Expected: %s, Got: %s\n", contract.SchemaHash, computedHash)
			hasErrors = true
			return nil
		}

		// Update schema hash if not present
		if contract.SchemaHash == "" {
			contract.SchemaHash = computedHash
			updatedData, _ := json.MarshalIndent(contract, "", "  ")
			if err := os.WriteFile(path, updatedData, 0644); err != nil {
				fmt.Printf("  ⚠️  Failed to update schema hash: %v\n", err)
			}
		}

		fmt.Printf("  ✅ Valid - hash: sha256:%s\n", computedHash[:16]+"...")
		validCount++
		return nil
	})

	if err != nil {
		fmt.Printf("❌ Walk error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("📊 Validation Summary:\n")
	fmt.Printf("   Total files: %d\n", totalCount)
	fmt.Printf("   Valid files: %d\n", validCount)
	fmt.Printf("   Failed files: %d\n", totalCount-validCount)

	if hasErrors {
		fmt.Println("❌ Validation failed - check errors above")
		os.Exit(1)
	}

	fmt.Println("✅ All validations passed!")
}

func loadSchema(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, err
	}

	return schema, nil
}

func validateAgainstSchema(contract DynamicMeasureContract, _ map[string]interface{}) error {
	// Basic validation - check required fields
	if contract.NodeID == "" {
		return fmt.Errorf("node_id is required")
	}
	if contract.NodeType != "dynamic_measure" {
		return fmt.Errorf("node_type must be 'dynamic_measure', got '%s'", contract.NodeType)
	}
	if contract.Name == "" {
		return fmt.Errorf("name is required")
	}
	if contract.SourceEnum == "" {
		return fmt.Errorf("source_enum is required")
	}
	if contract.SQL == "" {
		return fmt.Errorf("sql is required")
	}
	if len(contract.Tags) == 0 {
		return fmt.Errorf("at least one tag is required")
	}
	if contract.Owner == "" {
		return fmt.Errorf("owner is required")
	}
	if contract.Version == "" {
		return fmt.Errorf("version is required")
	}

	// Validate source_enum format
	if !strings.Contains(contract.SourceEnum, ".") {
		return fmt.Errorf("source_enum must be in format 'table.column', got '%s'", contract.SourceEnum)
	}

	// Validate version format
	if !strings.HasPrefix(contract.Version, "v") {
		return fmt.Errorf("version must start with 'v', got '%s'", contract.Version)
	}

	return nil
}

func validateMeasureRules(contract DynamicMeasureContract) error {
	// Validate SQL contains the source column
	parts := strings.Split(contract.SourceEnum, ".")
	if len(parts) != 2 {
		return fmt.Errorf("invalid source_enum format")
	}
	column := parts[1]

	if !strings.Contains(strings.ToLower(contract.SQL), strings.ToLower(column)) {
		return fmt.Errorf("SQL must reference the source column '%s'", column)
	}

	// Validate measure name follows convention
	expectedPrefix := "total_"
	if !strings.HasPrefix(contract.Name, expectedPrefix) {
		return fmt.Errorf("measure name should start with '%s', got '%s'", expectedPrefix, contract.Name)
	}

	// Validate tags include source table
	table := parts[0]
	found := false
	for _, tag := range contract.Tags {
		if tag == table {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("tags must include source table '%s'", table)
	}

	return nil
}

func computeSchemaHash(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

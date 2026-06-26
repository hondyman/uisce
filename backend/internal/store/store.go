package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hondyman/semlayer/backend/internal/policy"
	"github.com/hondyman/semlayer/backend/internal/simulation"
	"github.com/jmoiron/sqlx"
	"gopkg.in/yaml.v3"
)

// LoadPolicy loads a single policy from a YAML file.
func LoadPolicy(path string) (*policy.Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy file %s: %w", path, err)
	}

	var pol policy.Policy
	if err := yaml.Unmarshal(data, &pol); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy file %s: %w", path, err)
	}

	// If the policy ID is not set in the file, use the file name.
	if pol.ID == "" {
		pol.ID = filepath.Base(path)
	}

	return &pol, nil
}

// LoadAllActivePoliciesFromDir loads all active policies from a directory.
func LoadAllActivePoliciesFromDir(dir string) ([]*policy.Policy, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy directory %s: %w", dir, err)
	}

	var policies []*policy.Policy
	for _, file := range files {
		if !file.IsDir() && (filepath.Ext(file.Name()) == ".yaml" || filepath.Ext(file.Name()) == ".yml") {
			policy, err := LoadPolicy(filepath.Join(dir, file.Name()))
			if err != nil {
				return nil, err
			}
			policies = append(policies, policy)
		}
	}

	return policies, nil
}

// LoadPolicyFromDB loads a single policy from the database.
func LoadPolicyFromDB(ctx context.Context, db *sqlx.DB, policyID string) (*policy.Policy, error) {
	// Placeholder implementation
	return &policy.Policy{}, nil
}

// LoadAllActivePoliciesFromDB loads all active policies from the database.
func LoadAllActivePoliciesFromDB(ctx context.Context, db *sqlx.DB) ([]*policy.Policy, error) {
	// Placeholder implementation
	return []*policy.Policy{}, nil
}

// LoadPolicyVersionFromDB loads a specific version of a policy from the database.
func LoadPolicyVersionFromDB(ctx context.Context, db *sqlx.DB, policyID string, version int) (*policy.Policy, error) {
	// Placeholder implementation
	return &policy.Policy{}, nil
}

// LoadHistoryFromDB loads the history of changes from the database.
func LoadHistoryFromDB(ctx context.Context, db *sqlx.DB, from, to time.Time) ([]simulation.ChangeSet, error) {
	// Placeholder implementation
	return []simulation.ChangeSet{}, nil
}

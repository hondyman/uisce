package services

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
)

type apiKeyFileEntry struct {
	Key       string   `json:"key"`
	UserID    string   `json:"user_id"`
	TenantIDs []string `json:"tenant_ids,omitempty"`
	Roles     []string `json:"roles,omitempty"`
}

// LoadAPIKeysFromFile loads API keys from a JSON file and registers them in memory.
func (sm *SecurityManager) LoadAPIKeysFromFile(path string) error {
	if sm == nil || sm.apiKeyManager == nil {
		return nil
	}

	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}

	payload, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	if len(payload) == 0 {
		return nil
	}

	entries := []apiKeyFileEntry{}
	if err := json.Unmarshal(payload, &entries); err != nil {
		return err
	}

	for _, entry := range entries {
		key := strings.TrimSpace(entry.Key)
		if key == "" {
			continue
		}
		sm.RegisterAPIKey(key, strings.TrimSpace(entry.UserID), entry.TenantIDs, entry.Roles)
	}

	return nil
}

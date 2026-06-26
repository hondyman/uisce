package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"strings"

	imodels "github.com/hondyman/semlayer/backend/internal/models"
)

// GetPoliciesForMicroBundle queries the policies table, parses the JSON rules,
// and returns a slice of internal Policy objects that are relevant to the
// provided micro-bundle resource identifier (e.g., "micro_bundle" or "micro_bundle:<id>").
func GetPoliciesForMicroBundle(ctx context.Context, db *sql.DB, resource string) ([]imodels.Policy, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, name, rules, active FROM policies WHERE active = true`)
	if err != nil {
		return nil, fmt.Errorf("failed to query policies: %w", err)
	}
	defer rows.Close()

	var out []imodels.Policy
	for rows.Next() {
		var id, name string
		var rulesRaw []byte
		var active bool
		if err := rows.Scan(&id, &name, &rulesRaw, &active); err != nil {
			continue
		}

		var rules map[string]any
		if err := json.Unmarshal(rulesRaw, &rules); err != nil {
			continue
		}

		// extract basic fields
		effect := "allow"
		if e, ok := rules["effect"].(string); ok {
			effect = e
		}

		// actions
		var actions []string
		if a, ok := rules["action"].(string); ok {
			actions = append(actions, a)
		} else if arr, ok := rules["action"].([]any); ok {
			for _, it := range arr {
				if s, ok := it.(string); ok {
					actions = append(actions, s)
				}
			}
		}

		// resource match: rules.resource may be object containing type or id, or a string
		matched := false
		if rsrc, ok := rules["resource"]; ok {
			switch rr := rsrc.(type) {
			case string:
				if rr == "*" || rr == resource || rr == "micro_bundle" || rr == "micro_bundle:*" {
					matched = true
				}
			case map[string]any:
				// check type or id
				if t, ok := rr["type"].(string); ok && (t == "micro_bundle" || t == resource) {
					matched = true
				}
				if idv, ok := rr["id"].(string); ok && (idv == resource || idv == "*") {
					matched = true
				}
			}
		}

		// simple fallback: include policies that mention "micro_bundle" in rules JSON
		if !matched {
			rawStr := string(rulesRaw)
			if rawStr != "" && (strings.Contains(rawStr, "micro_bundle") || strings.Contains(rawStr, "micro-bundle")) {
				matched = true
			}
		}

		if !matched {
			continue
		}

		p := imodels.Policy{
			ID:          id,
			Effect:      effect,
			Actions:     actions,
			Resources:   []string{resource},
			Description: name,
		}
		out = append(out, p)
	}

	return out, nil
}

// no extras

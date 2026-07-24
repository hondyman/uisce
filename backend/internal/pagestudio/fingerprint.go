package pagestudio

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PageFingerprint represents the semantic dependencies of a page
type PageFingerprint struct {
	GeneratedAt time.Time `json:"generated_at"`

	// BO Usage
	BOName string   `json:"bo_name"`
	Fields []string `json:"fields"` // Field names used in columns, charts, KPIs

	// API Usage
	APIEndpointIDs []uuid.UUID       `json:"api_endpoint_ids"`
	APIArgs        map[string]string `json:"api_args"` // Arg name -> Expected Type
}

// ComputeFingerprint analyzes a page definition to extract semantic dependencies
func ComputeFingerprint(page *CorePage) (*PageFingerprint, error) {
	fp := &PageFingerprint{
		GeneratedAt: time.Now(),
		Fields:      make([]string, 0),
		APIArgs:     make(map[string]string),
	}

	// 1. Analyze Components used
	var components []map[string]interface{}
	if len(page.Components) > 0 {
		if err := json.Unmarshal(page.Components, &components); err == nil {
			for _, comp := range components {
				extractFields(comp, fp)
			}
		}
	}

	// 2. Analyze Data Bindings
	var bindings map[string]interface{}
	if len(page.DataBindings) > 0 {
		if err := json.Unmarshal(page.DataBindings, &bindings); err == nil {
			for _, v := range bindings {
				if bMap, ok := v.(map[string]interface{}); ok {
					// Check Endpoint ID
					if epID, ok := bMap["endpoint_id"].(string); ok {
						if uid, err := uuid.Parse(epID); err == nil {
							fp.APIEndpointIDs = append(fp.APIEndpointIDs, uid)
						}
					}
					// Check Params
					if params, ok := bMap["params"].(map[string]interface{}); ok {
						for pName := range params {
							fp.APIArgs[pName] = "any"
						}
					}
				}
			}
		}
	}

	return fp, nil
}

func extractFields(comp map[string]interface{}, fp *PageFingerprint) {
	for k, v := range comp {
		if k == "field" || k == "dataKey" {
			if s, ok := v.(string); ok {
				fp.Fields = append(fp.Fields, s)
			}
		}

		if nested, ok := v.(map[string]interface{}); ok {
			extractFields(nested, fp)
		}
		if list, ok := v.([]interface{}); ok {
			for _, item := range list {
				if itemMap, ok := item.(map[string]interface{}); ok {
					extractFields(itemMap, fp)
				}
			}
		}
	}
}

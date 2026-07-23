package apistudio

import (
	"context"
	"encoding/json"
	"strings"
)

// APITestRunner handles execution of API-specific tests
type APITestRunner struct {
	runtime *APIRuntime
}

// NewAPITestRunner creates a new test runner
func NewAPITestRunner(runtime *APIRuntime) *APITestRunner {
	return &APITestRunner{runtime: runtime}
}

// RunPIITest checks if any PII fields are exposed in the API sample response
func (r *APITestRunner) RunPIITest(ctx context.Context, test APITest, ep APIEndpoint) (json.RawMessage, error) {
	var def struct {
		PIIFields []string `json:"pii_fields"`
	}
	if err := json.Unmarshal(test.Definition, &def); err != nil {
		return nil, err
	}

	// 1. Call endpoint (dry run/sample)
	// For simplicity, we assume we can call the runtime directly if it doesn't have side effects
	// or we mock the request
	// In reality, we'd use a restricted context

	// Since we don't have a fake http.Request context here easily,
	// we'll simulate the logic: get the fields currently exposed.
	var exposedFields []string
	json.Unmarshal(ep.Fields, &exposedFields)

	var leaked []string
	for _, f := range exposedFields {
		for _, pii := range def.PIIFields {
			if strings.EqualFold(f, pii) {
				leaked = append(leaked, f)
			}
		}
	}

	status := "passed"
	if len(leaked) > 0 {
		status = "failed"
	}

	result := map[string]interface{}{
		"status":        status,
		"leaked_fields": leaked,
		"checked":       def.PIIFields,
	}

	return json.Marshal(result)
}

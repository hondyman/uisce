package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestGatewayRolesEndpoint(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("set RUN_INTEGRATION_TESTS=1 to run integration tests")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	baseURL := os.Getenv("INTEGRATION_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8001"
	}
	url := baseURL + "/api/roles"

	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("failed to GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var arr []interface{}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&arr); err != nil {
		t.Fatalf("failed to decode json array: %v", err)
	}

	if len(arr) == 0 {
		t.Fatalf("expected at least one role in response array")
	}
}

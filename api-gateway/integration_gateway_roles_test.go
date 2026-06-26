package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

// makeTestToken builds a simple HS256 token signed with JWT_SECRET from env.
func makeTestToken() (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "test-secret"
	}
	claims := jwt.MapClaims{
		"user_id":   "integration-test-user",
		"tenant_id": "integration-tenant",
		"exp":       time.Now().Add(1 * time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(secret))
}

func TestGatewayRolesProxy(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration gateway test: RUN_INTEGRATION_TESTS not set")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	url := "http://localhost:8001/api/roles"

	token, err := makeTestToken()
	if err != nil {
		t.Fatalf("failed to create test token: %v", err)
	}

	// Retry briefly in case containers are starting
	var resp *http.Response
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err = client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
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

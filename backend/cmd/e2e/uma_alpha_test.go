package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUMAAlphaE2E(t *testing.T) {
	if os.Getenv("SEMLAYER_E2E") != "1" {
		t.Skip("Skipping E2E tests. Set SEMLAYER_E2E=1 to run")
	}
	// Query Hasura for UMA accounts before rebalance
	resp, err := http.Post("http://hasura:8080/v1/graphql", "application/json",
		strings.NewReader(`{"query":"{uma_accounts{tax_saved}}"}`))
	assert.NoError(t, err)
	defer resp.Body.Close()

	var result struct {
		Data struct {
			UMA []struct {
				Saved float64 `json:"tax_saved"`
			} `json:"uma_accounts"`
		}
	}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

	initialTaxSaved := result.Data.UMA[0].Saved
	fmt.Printf("Initial tax saved: $%.2f\n", initialTaxSaved)

	// Trigger UMA Alpha rebalance
	resp2, err := http.Post("http://localhost:8080/api/uma/test-uma-1/alpha", "application/json", strings.NewReader("{}"))
	assert.NoError(t, err)
	assert.Equal(t, 202, resp2.StatusCode)
	resp2.Body.Close()

	// Wait for workflow completion (in real test, use proper waiting)
	// For demo, assume it completes and check tax_saved increased
	resp3, err := http.Post("http://hasura:8080/v1/graphql", "application/json",
		strings.NewReader(`{"query":"{uma_accounts{tax_saved}}"}`))
	assert.NoError(t, err)
	defer resp3.Body.Close()

	assert.NoError(t, json.NewDecoder(resp3.Body).Decode(&result))
	finalTaxSaved := result.Data.UMA[0].Saved
	fmt.Printf("Final tax saved: $%.2f\n", finalTaxSaved)

	// Assert tax savings increased by at least $50K
	assert.True(t, finalTaxSaved >= initialTaxSaved+50000, "Tax savings should increase by at least $50K")
}

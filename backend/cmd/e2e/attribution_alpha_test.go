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

func TestAttributionAlphaE2E(t *testing.T) {
	if os.Getenv("SEMLAYER_E2E") != "1" {
		t.Skip("Skipping E2E tests. Set SEMLAYER_E2E=1 to run")
	}
	// Query Hasura for portfolios before attribution
	resp, err := http.Post("http://hasura:8080/v1/graphql", "application/json",
		strings.NewReader(`{"query":"{portfolios{alpha}}"}`))
	assert.NoError(t, err)
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Portfolios []struct {
				Alpha float64 `json:"alpha"`
			} `json:"portfolios"`
		}
	}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

	initialAlpha := result.Data.Portfolios[0].Alpha
	fmt.Printf("Initial alpha: %.2f%%\n", initialAlpha)

	// Trigger Attribution Alpha analysis
	resp2, err := http.Post("http://localhost:8080/api/portfolio/test-portfolio-1/attribute", "application/json", strings.NewReader("{}"))
	assert.NoError(t, err)
	assert.Equal(t, 202, resp2.StatusCode)
	resp2.Body.Close()

	// Wait for workflow completion (in real test, use proper waiting)
	// For demo, assume it completes and check alpha increased
	resp3, err := http.Post("http://hasura:8080/v1/graphql", "application/json",
		strings.NewReader(`{"query":"{portfolios{alpha}}"}`))
	assert.NoError(t, err)
	defer resp3.Body.Close()

	assert.NoError(t, json.NewDecoder(resp3.Body).Decode(&result))
	finalAlpha := result.Data.Portfolios[0].Alpha
	fmt.Printf("Final alpha: %.2f%%\n", finalAlpha)

	// Assert alpha increased by at least 1.0%
	assert.True(t, finalAlpha >= initialAlpha+1.0, "Alpha should increase by at least 1.0%%")
}

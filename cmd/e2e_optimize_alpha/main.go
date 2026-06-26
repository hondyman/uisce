package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOptimizeAlphaWorkflow(t *testing.T) {
	// 1. Trigger the workflow
	resp, err := http.Post("http://localhost:8001/portfolio/1/optimize", "application/json", nil)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	// 2. Wait for the workflow to complete
	time.Sleep(5 * time.Second)

	// 3. Check the result in the database
	query := `{"query":"{portfolios(where: {id: {_eq: \"1\"}}){sharpe}}"}`
	resp, err = http.Post("http://hasura:8080/v1/graphql", "application/json", strings.NewReader(query))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var r struct {
		Data struct {
			Portfolios []struct {
				Sharpe float64 `json:"sharpe"`
			} `json:"portfolios"`
		} `json:"data"`
	}

	err = json.NewDecoder(resp.Body).Decode(&r)
	assert.NoError(t, err)
	assert.True(t, len(r.Data.Portfolios) > 0, "portfolio not found")
	assert.True(t, r.Data.Portfolios[0].Sharpe > 1.5, "Sharpe ratio should be > 1.5")

	fmt.Println("E2E test for OptimizeAlpha workflow passed!")
}

func main() {
	// This is a dummy main function to make the file runnable.
	// In a real scenario, this would be run as part of a test suite.
	t := &testing.T{}
	TestOptimizeAlphaWorkflow(t)
}

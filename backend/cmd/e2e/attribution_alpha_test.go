package main

import (
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
	db, err := e2eDB()
	assert.NoError(t, err)
	defer db.Close()

	// Read portfolios before attribution
	initialAlpha, err := queryPortfolioAlpha(db)
	assert.NoError(t, err)
	fmt.Printf("Initial alpha: %.2f%%\n", initialAlpha)

	// Trigger Attribution Alpha analysis
	resp2, err := http.Post("http://localhost:8080/api/portfolio/test-portfolio-1/attribute", "application/json", strings.NewReader("{}"))
	assert.NoError(t, err)
	assert.Equal(t, 202, resp2.StatusCode)
	resp2.Body.Close()

	// Wait for workflow completion (in real test, use proper waiting)
	// For demo, assume it completes and check alpha increased
	finalAlpha, err := queryPortfolioAlpha(db)
	assert.NoError(t, err)
	fmt.Printf("Final alpha: %.2f%%\n", finalAlpha)

	// Assert alpha increased by at least 1.0%
	assert.True(t, finalAlpha >= initialAlpha+1.0, "Alpha should increase by at least 1.0%%")
}

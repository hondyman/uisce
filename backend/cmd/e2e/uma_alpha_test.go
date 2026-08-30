package main

import (
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
	db, err := e2eDB()
	assert.NoError(t, err)
	defer db.Close()

	// Read UMA accounts before rebalance
	initialTaxSaved, err := queryUMATaxSaved(db)
	assert.NoError(t, err)
	fmt.Printf("Initial tax saved: $%.2f\n", initialTaxSaved)

	// Trigger UMA Alpha rebalance
	resp2, err := http.Post("http://localhost:8080/api/uma/test-uma-1/alpha", "application/json", strings.NewReader("{}"))
	assert.NoError(t, err)
	assert.Equal(t, 202, resp2.StatusCode)
	resp2.Body.Close()

	// Wait for workflow completion (in real test, use proper waiting)
	// For demo, assume it completes and check tax_saved increased
	finalTaxSaved, err := queryUMATaxSaved(db)
	assert.NoError(t, err)
	fmt.Printf("Final tax saved: $%.2f\n", finalTaxSaved)

	// Assert tax savings increased by at least $50K
	assert.True(t, finalTaxSaved >= initialTaxSaved+50000, "Tax savings should increase by at least $50K")
}

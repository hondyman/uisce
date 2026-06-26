//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func main() {
	fmt.Println("🧪 Running Direct Indexing Alpha E2E Test...")

	// Test the API endpoint
	resp, err := http.Post("http://localhost:8080/api/index/test-index/alpha", "application/json", strings.NewReader("{}"))
	if err != nil {
		fmt.Printf("❌ API call failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		fmt.Printf("❌ Expected status 202, got %d\n", resp.StatusCode)
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("❌ Failed to decode response: %v\n", err)
		return
	}

	if result["status"] != "alpha optimization initiated" {
		fmt.Printf("❌ Unexpected status: %v\n", result["status"])
		return
	}

	fmt.Println("✅ Direct Indexing Alpha API test passed")

	// Test Hasura subscription for optimization results
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Post("http://hasura:8080/v1/graphql", "application/json", strings.NewReader(`{"query":"{direct_indexes{tax_saved}}"}`))
		if err != nil {
			fmt.Printf("❌ Hasura query failed: %v\n", err)
			time.Sleep(2 * time.Second)
			continue
		}

		var r struct {
			Data struct {
				DirectIndexes []struct {
					TaxSaved float64 `json:"tax_saved"`
				} `json:"direct_indexes"`
			}
		}
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			fmt.Printf("❌ Failed to decode Hasura response: %v\n", err)
			resp.Body.Close()
			time.Sleep(2 * time.Second)
			continue
		}
		resp.Body.Close()

		if len(r.Data.DirectIndexes) > 0 && r.Data.DirectIndexes[0].TaxSaved > 100000 {
			fmt.Printf("✅ Direct Indexing Alpha optimization verified: $%.0f tax saved\n", r.Data.DirectIndexes[0].TaxSaved)
			return
		}

		time.Sleep(2 * time.Second)
	}

	fmt.Println("❌ Timeout waiting for Direct Indexing Alpha optimization results")
}

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

// Tiny eventual polling sample used by CI to assert that Hasura received a row.
func main() {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Post("http://hasura:8080/v1/graphql", "application/json", strings.NewReader(`{"query":"{temporal_workflows{workflow_id}}"}`))
		if err != nil {
			fmt.Println("request failed:", err)
			time.Sleep(1 * time.Second)
			continue
		}
		var r struct {
			Data struct {
				TemporalWorkflows []struct {
					WorkflowID string `json:"workflow_id"`
				} `json:"temporal_workflows"`
			}
		}
		json.NewDecoder(resp.Body).Decode(&r)
		resp.Body.Close()
		if len(r.Data.TemporalWorkflows) > 0 && strings.Contains(r.Data.TemporalWorkflows[0].WorkflowID, "e2e-") {
			fmt.Println("found e2e workflow")
			return
		}
		time.Sleep(1 * time.Second)
	}
	fmt.Println("timeout waiting for e2e workflow")
}

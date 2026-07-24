package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func main() {
	resp, err := http.Post("http://hasura:8080/v1/graphql", "application/json", strings.NewReader(`{"query":"{temporal_workflows{workflow_id}}"}`))
	if err != nil {
		fmt.Println("error querying hasura:", err)
		return
	}
	defer resp.Body.Close()
	var r struct {
		Data map[string][]map[string]string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		fmt.Println("decode error:", err)
		return
	}
	items := r.Data["temporal_workflows"]
	if len(items) == 0 {
		fmt.Println("no workflows found")
		return
	}
	fmt.Println("found workflows:", len(items))
}

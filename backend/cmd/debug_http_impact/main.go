package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func main() {
	// Use the UUID we know has data from our AGE debug script
	nodeID := "14b5e022-3755-4a0a-b53b-c2ab4e392931"
	nodeType := "semantic_term"
	url := fmt.Sprintf("http://localhost:8082/api/impact/graph/%s/%s?depth=1", nodeType, nodeID)

	fmt.Printf("Testing Impact Graph API: %s\n", url)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Tenant-ID", "910638ba-a459-4a3f-bb2d-78391b0595f6") // From previous turn

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to call API: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %s\n", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		var data map[string]interface{}
		json.Unmarshal(body, &data)

		nodes := data["nodes"].([]interface{})
		edges := data["edges"].([]interface{})

		fmt.Printf("Nodes found: %d\n", len(nodes))
		fmt.Printf("Edges found: %d\n", len(edges))

		if len(nodes) > 0 {
			formatted, _ := json.MarshalIndent(data, "", "  ")
			fmt.Println("\nResponse Sample:")
			fmt.Println(string(formatted))
		}
	} else {
		fmt.Printf("Error Response: %s\n", string(body))
	}
}

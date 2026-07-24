package metadata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type HasuraClient struct {
	BaseURL     string
	AdminSecret string
	Client      *http.Client
}

func NewHasuraClient(baseURL, adminSecret string) *HasuraClient {
	return &HasuraClient{
		BaseURL:     baseURL,
		AdminSecret: adminSecret,
		Client:      &http.Client{},
	}
}

// TrackTable instructs Hasura to expose a Postgres table in the GraphQL API
func (c *HasuraClient) TrackTable(tableName string) error {
	payload := map[string]interface{}{
		"type": "pg_track_table",
		"args": map[string]interface{}{
			"source": "default",
			"table": map[string]string{
				"schema": "public",
				"name":   tableName,
			},
		},
	}
	return c.sendMetadataRequest(payload)
}

// AddComputedField adds a SQL function as a computed field on a table
func (c *HasuraClient) AddComputedField(tableName, fieldName, functionName string) error {
	payload := map[string]interface{}{
		"type": "pg_add_computed_field",
		"args": map[string]interface{}{
			"source": "default",
			"table": map[string]string{
				"schema": "public",
				"name":   tableName,
			},
			"name": fieldName,
			"definition": map[string]interface{}{
				"function": map[string]string{
					"schema": "public",
					"name":   functionName,
				},
				"table_argument": "entity_row", // Assumes function signature takes the row
			},
		},
	}
	return c.sendMetadataRequest(payload)
}

// CreateSelectPermission adds a row-level permission rule
func (c *HasuraClient) CreateSelectPermission(tableName, role string, filter map[string]interface{}) error {
	payload := map[string]interface{}{
		"type": "pg_create_select_permission",
		"args": map[string]interface{}{
			"source": "default",
			"table": map[string]string{
				"schema": "public",
				"name":   tableName,
			},
			"role": role,
			"permission": map[string]interface{}{
				"columns": "*",
				"filter":  filter,
			},
		},
	}
	return c.sendMetadataRequest(payload)
}

func (c *HasuraClient) sendMetadataRequest(payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/v1/metadata", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hasura-Admin-Secret", c.AdminSecret)

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hasura metadata api returned status: %d", resp.StatusCode)
	}

	return nil
}

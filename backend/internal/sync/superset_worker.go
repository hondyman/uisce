package sync

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

// SupersetSyncWorker handles synchronization to Apache Superset
type SupersetSyncWorker struct {
	supersetURL string
	username    string
	password    string
	db          *sql.DB
	httpClient  *http.Client
	accessToken string
}

// NewSupersetSyncWorker creates a new Superset sync worker
func NewSupersetSyncWorker(supersetURL, username, password string, db *sql.DB) *SupersetSyncWorker {
	return &SupersetSyncWorker{
		supersetURL: supersetURL,
		username:    username,
		password:    password,
		db:          db,
		httpClient:  &http.Client{},
	}
}

// Login authenticates with Superset and gets access token
func (w *SupersetSyncWorker) Login(ctx context.Context) error {
	payload := map[string]interface{}{
		"username": w.username,
		"password": w.password,
		"provider": "db",
		"refresh":  true,
	}

	jsonData, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", w.supersetURL+"/api/v1/security/login", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	w.accessToken = result["access_token"].(string)
	return nil
}

// SyncRole creates or updates a Superset role
func (w *SupersetSyncWorker) SyncRole(ctx context.Context, roleData map[string]interface{}) error {
	if w.accessToken == "" {
		if err := w.Login(ctx); err != nil {
			return err
		}
	}

	roleName, _ := roleData["role_name"].(string)
	tenantID, _ := roleData["tenant_id"].(string)
	isGlobalAdmin, _ := roleData["is_global_admin"].(bool)

	// Create role in Superset
	roleID, err := w.createRole(ctx, roleName)
	if err != nil {
		return err
	}

	// If not global admin, create RLS rules for all datasets
	if !isGlobalAdmin {
		datasets, err := w.getDatasets(ctx)
		if err != nil {
			return err
		}

		for _, dataset := range datasets {
			if err := w.createRLSRule(ctx, dataset, roleID, tenantID); err != nil {
				return fmt.Errorf("failed to create RLS for dataset %v: %w", dataset["id"], err)
			}
		}
	}

	return nil
}

func (w *SupersetSyncWorker) createRole(ctx context.Context, roleName string) (int, error) {
	payload := map[string]interface{}{
		"name": roleName,
	}

	jsonData, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", w.supersetURL+"/api/v1/security/roles/", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+w.accessToken)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	roleID := int(result["id"].(float64))
	return roleID, nil
}

func (w *SupersetSyncWorker) getDatasets(ctx context.Context) ([]map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", w.supersetURL+"/api/v1/dataset/", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+w.accessToken)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	datasets := result["result"].([]interface{})
	var datasetList []map[string]interface{}
	for _, d := range datasets {
		datasetList = append(datasetList, d.(map[string]interface{}))
	}

	return datasetList, nil
}

func (w *SupersetSyncWorker) createRLSRule(ctx context.Context, dataset map[string]interface{}, roleID int, tenantID string) error {
	datasetID := int(dataset["id"].(float64))
	datasetName := dataset["table_name"].(string)

	payload := map[string]interface{}{
		"name":        fmt.Sprintf("Tenant %s - %s", tenantID, datasetName),
		"description": fmt.Sprintf("Row-level security for tenant %s", tenantID),
		"filter_type": "Regular",
		"tables":      []int{datasetID},
		"roles":       []int{roleID},
		"clause":      fmt.Sprintf("tenant_id = '%s'", tenantID),
	}

	jsonData, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", w.supersetURL+"/api/v1/rowlevelsecurity/", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+w.accessToken)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("superset API error: %v", errResp)
	}

	return nil
}

// AssignUserToRole assigns a user to a Superset role
func (w *SupersetSyncWorker) AssignUserToRole(ctx context.Context, userID, roleID string) error {
	// TODO: Implement user-role assignment via Superset API
	return nil
}

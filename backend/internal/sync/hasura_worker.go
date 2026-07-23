package sync

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

// HasuraSyncWorker handles synchronization to Hasura
type HasuraSyncWorker struct {
	hasuraURL   string
	adminSecret string
	db          *sql.DB
	httpClient  *http.Client
}

// NewHasuraSyncWorker creates a new Hasura sync worker
func NewHasuraSyncWorker(hasuraURL, adminSecret string, db *sql.DB) *HasuraSyncWorker {
	return &HasuraSyncWorker{
		hasuraURL:   hasuraURL,
		adminSecret: adminSecret,
		db:          db,
		httpClient:  &http.Client{},
	}
}

// SyncRole creates or updates Hasura permissions for a role
func (w *HasuraSyncWorker) SyncRole(ctx context.Context, roleData map[string]interface{}) error {
	roleName, _ := roleData["role_name"].(string)
	isGlobalAdmin, _ := roleData["is_global_admin"].(bool)

	// Get all tables that need permissions
	tables := []string{
		"business_objects",
		"processes",
		"validation_rules",
		"catalog_node",
		"catalog_edge",
		"page_layouts",
		"pipelines",
		"users",
		"tenants",
	}

	for _, table := range tables {
		// Create SELECT permission
		if err := w.createPermission(ctx, table, roleName, "select", isGlobalAdmin); err != nil {
			return fmt.Errorf("failed to create select permission for %s: %w", table, err)
		}

		// Create INSERT permission
		if err := w.createPermission(ctx, table, roleName, "insert", isGlobalAdmin); err != nil {
			return fmt.Errorf("failed to create insert permission for %s: %w", table, err)
		}

		// Create UPDATE permission
		if err := w.createPermission(ctx, table, roleName, "update", isGlobalAdmin); err != nil {
			return fmt.Errorf("failed to create update permission for %s: %w", table, err)
		}

		// Create DELETE permission (only for admins)
		if isGlobalAdmin {
			if err := w.createPermission(ctx, table, roleName, "delete", isGlobalAdmin); err != nil {
				return fmt.Errorf("failed to create delete permission for %s: %w", table, err)
			}
		}
	}

	return nil
}

func (w *HasuraSyncWorker) createPermission(ctx context.Context, table, role, operation string, isGlobalAdmin bool) error {
	var filter map[string]interface{}
	var columnPreset map[string]string

	if isGlobalAdmin {
		// Global admins have no filter
		filter = map[string]interface{}{}
	} else {
		// Standard users filtered by tenant_id
		filter = map[string]interface{}{
			"tenant_id": map[string]string{
				"_eq": "X-Hasura-Tenant-Id",
			},
		}

		// For INSERT, auto-set tenant_id
		if operation == "insert" {
			columnPreset = map[string]string{
				"tenant_id": "X-Hasura-Tenant-Id",
			}
		}
	}

	payload := map[string]interface{}{
		"type": fmt.Sprintf("pg_create_%s_permission", operation),
		"args": map[string]interface{}{
			"table": table,
			"role":  role,
			"permission": map[string]interface{}{
				"columns": "*",
				"filter":  filter,
			},
		},
	}

	// Add column preset for INSERT
	if len(columnPreset) > 0 {
		payload["args"].(map[string]interface{})["permission"].(map[string]interface{})["set"] = columnPreset
	}

	return w.callHasuraAPI(ctx, payload)
}

func (w *HasuraSyncWorker) callHasuraAPI(ctx context.Context, payload map[string]interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", w.hasuraURL+"/v1/metadata", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", w.adminSecret)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("hasura API error: %v", errResp)
	}

	return nil
}

// InvalidateUserJWT marks user sessions as invalid, forcing re-login
func (w *HasuraSyncWorker) InvalidateUserJWT(ctx context.Context, userID string) error {
	_, err := w.db.ExecContext(ctx,
		"UPDATE user_sessions SET is_active = false WHERE user_id = $1",
		userID,
	)
	return err
}

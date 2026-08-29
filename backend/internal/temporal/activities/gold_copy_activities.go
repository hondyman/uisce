package activities

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/hondyman/uisce/backend/internal/events"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// connectionTemplateFields are the non-credential columns propagated from a
// gold-copy connections row. Deliberately excludes password, api_key, and
// secret_path — see syncConnectionToTenant's doc comment.
type connectionTemplateFields struct {
	Name     string
	Type     string
	Host     sql.NullString
	Port     sql.NullInt32
	Database sql.NullString
	Schema   sql.NullString
	BaseURL  sql.NullString
	Metadata []byte // raw JSON, for the jsonb column
}

func connectionTemplateFieldsFromData(data map[string]interface{}) (connectionTemplateFields, error) {
	name, _ := data["name"].(string)
	connType, _ := data["type"].(string)
	if name == "" || connType == "" {
		return connectionTemplateFields{}, fmt.Errorf("connection_data missing required name/type")
	}

	f := connectionTemplateFields{Name: name, Type: connType}
	if v, ok := data["host"].(string); ok && v != "" {
		f.Host = sql.NullString{String: v, Valid: true}
	}
	if v, ok := data["port"].(float64); ok { // JSON numbers decode as float64
		f.Port = sql.NullInt32{Int32: int32(v), Valid: true}
	}
	if v, ok := data["database"].(string); ok && v != "" {
		f.Database = sql.NullString{String: v, Valid: true}
	}
	if v, ok := data["schema"].(string); ok && v != "" {
		f.Schema = sql.NullString{String: v, Valid: true}
	}
	if v, ok := data["base_url"].(string); ok && v != "" {
		f.BaseURL = sql.NullString{String: v, Valid: true}
	}

	metadata := map[string]interface{}{}
	if m, ok := data["metadata"].(map[string]interface{}); ok {
		metadata = m
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		return connectionTemplateFields{}, fmt.Errorf("marshal metadata: %w", err)
	}
	f.Metadata = raw

	return f, nil
}

type AuditService interface {
	WriteEvent(ctx context.Context, event events.GoldCopyConnectionEvent) error
}

type GoldCopyActivities struct {
	DB           *sqlx.DB
	Logger       *zap.SugaredLogger
	AuditService AuditService
}

func NewGoldCopyActivities(db *sql.DB, logger *zap.SugaredLogger, auditService AuditService) *GoldCopyActivities {
	return &GoldCopyActivities{
		DB:           sqlx.NewDb(db, "pgx"),
		Logger:       logger,
		AuditService: auditService,
	}
}

func (a *GoldCopyActivities) PropagateConnectionActivity(ctx context.Context, event events.GoldCopyConnectionEvent) error {
	a.Logger.Infof("Propagating connection change: %s (%s)", event.ConnectionID, event.Action)

	// Fetch all downstream tenants (that are not gold copy)
	// We can filter by "subscribed" tenants if that concept exists, or just all active tenants.
	var tenants []string
	err := a.DB.SelectContext(ctx, &tenants, `SELECT id FROM tenants WHERE is_active = true AND gold_copy = false`)
	if err != nil {
		return fmt.Errorf("failed to fetch tenants: %w", err)
	}

	for _, tenantID := range tenants {
		if err := a.syncConnectionToTenant(ctx, tenantID, event); err != nil {
			a.Logger.Errorf("Failed to sync to tenant %s: %v", tenantID, err)
			// Continue to next tenant? Or fail?
			// For robustness, we should probably continue and report errors array, or use child workflows.
			// But for simplicity in Activity, we'll log error and proceed to ensure best-effort propagation.
			// Ideally this should be a workflow iterating activities.
		}
	}

	return nil
}

// syncConnectionToTenant propagates a gold-copy connections row to one
// tenant's copy, linked by core_id = the gold-copy connection's own id.
//
// DELIBERATE SCOPE: this propagates the connection's shape/template fields
// only — name, type, host, port, database, schema, base_url, metadata. It
// NEVER copies credentials (password, api_key, secret_path) and always
// creates the tenant's row with is_active = false. Each tenant configures
// its own real database credentials — a tenant's actual data connection is
// never inherited, only the template describing what kind of connection it
// should be. An UPDATE from gold-copy likewise only touches template fields,
// never a tenant's already-configured credentials or activation state, so a
// gold-copy edit can never silently disable or reconfigure a live connection.
func (a *GoldCopyActivities) syncConnectionToTenant(ctx context.Context, tenantID string, event events.GoldCopyConnectionEvent) error {
	switch event.Action {
	case "INSERT":
		var exists bool
		err := a.DB.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM connections WHERE tenant_id = $1 AND core_id = $2)`, tenantID, event.ConnectionID)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}

		fields, err := connectionTemplateFieldsFromData(event.ConnectionData)
		if err != nil {
			return fmt.Errorf("gold copy connection %s: %w", event.ConnectionID, err)
		}

		_, err = a.DB.ExecContext(ctx, `
			INSERT INTO connections (
				tenant_id, core_id, name, type, host, port, database, schema, base_url, metadata, is_active
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, false
			)
		`, tenantID, event.ConnectionID, fields.Name, fields.Type, fields.Host, fields.Port,
			fields.Database, fields.Schema, fields.BaseURL, fields.Metadata)
		if err != nil {
			return fmt.Errorf("insert connection for tenant %s: %w", tenantID, err)
		}
		a.Logger.Infof("Propagated new connection template %q (core_id=%s) to tenant %s, inactive pending credentials", fields.Name, event.ConnectionID, tenantID)

	case "UPDATE":
		fields, err := connectionTemplateFieldsFromData(event.ConnectionData)
		if err != nil {
			return fmt.Errorf("gold copy connection %s: %w", event.ConnectionID, err)
		}

		res, err := a.DB.ExecContext(ctx, `
			UPDATE connections
			SET name = $3, type = $4, host = $5, port = $6, database = $7, schema = $8, base_url = $9, metadata = $10, updated_at = now()
			WHERE tenant_id = $1 AND core_id = $2
		`, tenantID, event.ConnectionID, fields.Name, fields.Type, fields.Host, fields.Port,
			fields.Database, fields.Schema, fields.BaseURL, fields.Metadata)
		if err != nil {
			return fmt.Errorf("update connection for tenant %s: %w", tenantID, err)
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			a.Logger.Warnf("Gold copy connection %s updated but tenant %s has no linked row (core_id) to update", event.ConnectionID, tenantID)
		}

	case "DELETE":
		_, err := a.DB.ExecContext(ctx, `DELETE FROM connections WHERE tenant_id = $1 AND core_id = $2`, tenantID, event.ConnectionID)
		if err != nil {
			return err
		}
	}

	// Log Audit Event for the Clone
	if a.AuditService != nil {
		cloneEvent := event
		cloneEvent.TenantID = tenantID // Audit this for the specific tenant
		cloneEvent.Action = "CLONE_" + event.Action
		if err := a.AuditService.WriteEvent(ctx, cloneEvent); err != nil {
			a.Logger.Warnf("Failed to audit clone event for tenant %s: %v", tenantID, err)
		}
	}

	return nil
}

func (a *GoldCopyActivities) LogConnectionAuditActivity(ctx context.Context, event events.GoldCopyConnectionEvent) error {
	a.Logger.Infof("Logging connection audit for event %s", event.EventID)

	if a.AuditService == nil {
		a.Logger.Warn("AuditService not initialized, skipping audit log")
		return nil
	}

	if err := a.AuditService.WriteEvent(ctx, event); err != nil {
		a.Logger.Errorf("Failed to write audit log: %v", err)
		return err
	}

	return nil
}

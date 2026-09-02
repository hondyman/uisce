package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/events"
	"github.com/hondyman/uisce/backend/models"
	"github.com/jmoiron/sqlx"
)

// Connection represents a datasource connection from the `connections` table
// (the table tenant_product_datasource.connection_id actually references).
type Connection struct {
	ID         string          `json:"id" db:"id"`
	TenantID   string          `json:"tenant_id" db:"tenant_id"`
	Name       string          `json:"name" db:"name"`
	Type       string          `json:"type" db:"type"`
	Host       *string         `json:"host,omitempty" db:"host"`
	Port       *int            `json:"port,omitempty" db:"port"`
	Database   *string         `json:"database,omitempty" db:"database"`
	Schema     *string         `json:"schema,omitempty" db:"schema"`
	Username   *string         `json:"username,omitempty" db:"username"`
	Password   *string         `json:"password,omitempty" db:"password"`
	SecretPath *string         `json:"secret_path,omitempty" db:"secret_path"`
	BaseURL    *string         `json:"base_url,omitempty" db:"base_url"`
	APIKey     *string         `json:"api_key,omitempty" db:"api_key"`
	Metadata   json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	IsActive   bool            `json:"is_active" db:"is_active"`
	// CoreID is set when this row was propagated from a gold-copy connection
	// (see internal/temporal/activities/gold_copy_activities.go). A tenant's
	// own credentials/is_active are never touched by that propagation.
	CoreID    *string   `json:"core_id,omitempty" db:"core_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	// Origin is derived, not stored: "gold_copy" for the gold-copy tenant's
	// own row, "inherited" when CoreID is set, "custom" otherwise.
	Origin string `json:"origin,omitempty" db:"-"`
}

// ConnectionsService handles unified connection management
type ConnectionsService struct {
	db             *sqlx.DB
	goldCopyEvents *events.KafkaPublisher // nil is fine: publishing becomes a no-op
}

// NewConnectionsService creates a new connections service
func NewConnectionsService(db *sqlx.DB) *ConnectionsService {
	return &ConnectionsService{db: db}
}

// NewConnectionsServiceWithEvents creates a connections service that publishes
// a GoldCopyConnectionChanged event (see internal/events/kafka_publisher.go)
// whenever the gold-copy tenant creates/updates/deletes a connection, so the
// already-registered Temporal pipeline (GoldCopyConnectionPropagation ->
// GoldCopyActivities.syncConnectionToTenant) propagates the connection
// template to every other tenant.
func NewConnectionsServiceWithEvents(db *sqlx.DB, publisher *events.KafkaPublisher) *ConnectionsService {
	return &ConnectionsService{db: db, goldCopyEvents: publisher}
}

// isGoldCopyTenant reports whether tenantID is the tenant flagged gold_copy = true.
func (s *ConnectionsService) isGoldCopyTenant(ctx context.Context, tenantID string) bool {
	var goldCopyID sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT id::text FROM public.tenants WHERE gold_copy = true LIMIT 1`).Scan(&goldCopyID); err != nil {
		return false
	}
	return goldCopyID.Valid && goldCopyID.String == tenantID
}

// publishGoldCopyChange is best-effort: a Kafka/worker outage must never
// block connection CRUD, so failures are logged, not returned.
func (s *ConnectionsService) publishGoldCopyChange(ctx context.Context, action string, conn *Connection) {
	if s.goldCopyEvents == nil || conn == nil {
		return
	}
	if !s.isGoldCopyTenant(ctx, conn.TenantID) {
		return
	}

	data := map[string]interface{}{
		"name": conn.Name,
		"type": conn.Type,
	}
	if conn.Host != nil {
		data["host"] = *conn.Host
	}
	if conn.Port != nil {
		data["port"] = float64(*conn.Port)
	}
	if conn.Database != nil {
		data["database"] = *conn.Database
	}
	if conn.Schema != nil {
		data["schema"] = *conn.Schema
	}
	if conn.BaseURL != nil {
		data["base_url"] = *conn.BaseURL
	}
	if len(conn.Metadata) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(conn.Metadata, &meta); err == nil {
			data["metadata"] = meta
		}
	}

	event := &events.GoldCopyConnectionEvent{
		EventType:      events.GoldCopyConnectionChanged,
		TenantID:       conn.TenantID,
		ConnectionID:   conn.ID,
		Action:         action,
		ConnectionData: data,
		Timestamp:      time.Now(),
	}
	if err := s.goldCopyEvents.PublishGoldCopyConnectionEvent(ctx, event); err != nil {
		log.Printf("[ConnectionsService] failed to publish gold copy connection event (connection=%s action=%s): %v", conn.ID, action, err)
	}
}

func normalizedMetadata(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("{}")
	}
	return raw
}

// CreateConnection creates a new connection
func (s *ConnectionsService) CreateConnection(ctx context.Context, tenantID string, conn *Connection) (*Connection, error) {
	if conn == nil {
		return nil, fmt.Errorf("connection cannot be nil")
	}
	if conn.Name == "" {
		return nil, fmt.Errorf("connection name is required")
	}
	if conn.Type == "" {
		return nil, fmt.Errorf("connection type is required")
	}

	conn.ID = uuid.NewString()
	conn.TenantID = tenantID
	conn.Metadata = normalizedMetadata(conn.Metadata)

	query := `
		INSERT INTO connections
		(id, tenant_id, name, type, host, port, database, schema, username, password,
		 secret_path, base_url, api_key, metadata, is_active, core_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	err := db.WithTenantTransaction(ctx, s.db.DB, tenantID, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, query,
			conn.ID, conn.TenantID, conn.Name, conn.Type, conn.Host, conn.Port, conn.Database, conn.Schema,
			conn.Username, conn.Password, conn.SecretPath, conn.BaseURL, conn.APIKey, conn.Metadata, conn.IsActive, conn.CoreID,
		).Scan(&conn.CreatedAt, &conn.UpdatedAt)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	s.publishGoldCopyChange(ctx, "INSERT", conn)

	return conn, nil
}

// GetConnection retrieves a connection by ID
func (s *ConnectionsService) GetConnection(ctx context.Context, tenantID, connectionID string) (*Connection, error) {
	conn := &Connection{}

	query := `
		SELECT id, tenant_id, name, type, host, port, database, schema, username, password,
		       secret_path, base_url, api_key, COALESCE(metadata, '{}'::jsonb) AS metadata,
		       COALESCE(is_active, true) AS is_active, core_id, created_at, updated_at
		FROM connections
		WHERE id = $1 AND tenant_id = $2
	`

	err := db.WithTenantTransaction(ctx, s.db.DB, tenantID, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, query, connectionID, tenantID).Scan(
			&conn.ID, &conn.TenantID, &conn.Name, &conn.Type, &conn.Host, &conn.Port, &conn.Database, &conn.Schema,
			&conn.Username, &conn.Password, &conn.SecretPath, &conn.BaseURL, &conn.APIKey, &conn.Metadata,
			&conn.IsActive, &conn.CoreID, &conn.CreatedAt, &conn.UpdatedAt,
		)
	})

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("connection not found")
		}
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	s.setOrigin(ctx, conn)
	return conn, nil
}

// setOrigin derives Connection.Origin: "gold_copy" for the gold-copy
// tenant's own row, "inherited" when propagated from one (CoreID set),
// "custom" otherwise. Not persisted — computed for API responses so the
// frontend doesn't need to know the gold-copy tenant ID itself.
func (s *ConnectionsService) setOrigin(ctx context.Context, conn *Connection) {
	if conn == nil {
		return
	}
	if conn.CoreID != nil {
		conn.Origin = "inherited"
		return
	}
	if s.isGoldCopyTenant(ctx, conn.TenantID) {
		conn.Origin = "gold_copy"
		return
	}
	conn.Origin = "custom"
}

// ListConnections retrieves all connections for a tenant
func (s *ConnectionsService) ListConnections(ctx context.Context, tenantID string) ([]*Connection, error) {
	query := `
		SELECT id, tenant_id, name, type, host, port, database, schema, username, password,
		       secret_path, base_url, api_key, COALESCE(metadata, '{}'::jsonb) AS metadata,
		       COALESCE(is_active, true) AS is_active, core_id, created_at, updated_at
		FROM connections
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	isGoldCopy := s.isGoldCopyTenant(ctx, tenantID)

	var connections []*Connection
	err := db.WithTenantTransaction(ctx, s.db.DB, tenantID, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, query, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			conn := &Connection{}

			if err := rows.Scan(
				&conn.ID, &conn.TenantID, &conn.Name, &conn.Type, &conn.Host, &conn.Port, &conn.Database, &conn.Schema,
				&conn.Username, &conn.Password, &conn.SecretPath, &conn.BaseURL, &conn.APIKey, &conn.Metadata,
				&conn.IsActive, &conn.CoreID, &conn.CreatedAt, &conn.UpdatedAt,
			); err != nil {
				return fmt.Errorf("failed to scan connection: %w", err)
			}

			if conn.CoreID != nil {
				conn.Origin = "inherited"
			} else if isGoldCopy {
				conn.Origin = "gold_copy"
			} else {
				conn.Origin = "custom"
			}

			connections = append(connections, conn)
		}

		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list connections: %w", err)
	}

	return connections, nil
}

// UpdateConnection updates an existing connection. Only fields set on conn
// (i.e. Name/Type, and any non-nil pointer field) are written — callers
// currently resend the full form on every save, matching PUT/PATCH-as-replace
// semantics for the whole editable field set.
func (s *ConnectionsService) UpdateConnection(ctx context.Context, tenantID string, conn *Connection) (*Connection, error) {
	if conn == nil || conn.ID == "" {
		return nil, fmt.Errorf("connection and ID cannot be nil/empty")
	}

	query := `
		UPDATE connections
		SET name = $1, type = $2, host = $3, port = $4, database = $5, schema = $6,
		    username = $7, password = COALESCE($8, password), secret_path = $9,
		    base_url = $10, api_key = $11, metadata = $12, is_active = $13, updated_at = NOW()
		WHERE id = $14 AND tenant_id = $15
		RETURNING created_at, updated_at
	`

	err := db.WithTenantTransaction(ctx, s.db.DB, tenantID, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, query,
			conn.Name, conn.Type, conn.Host, conn.Port, conn.Database, conn.Schema,
			conn.Username, conn.Password, conn.SecretPath, conn.BaseURL, conn.APIKey,
			normalizedMetadata(conn.Metadata), conn.IsActive,
			conn.ID, tenantID,
		).Scan(&conn.CreatedAt, &conn.UpdatedAt)
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("connection not found")
		}
		return nil, fmt.Errorf("failed to update connection: %w", err)
	}

	conn.TenantID = tenantID
	s.publishGoldCopyChange(ctx, "UPDATE", conn)
	return conn, nil
}

// DeleteConnection deletes a connection
func (s *ConnectionsService) DeleteConnection(ctx context.Context, tenantID, connectionID string) error {
	query := `DELETE FROM connections WHERE id = $1 AND tenant_id = $2`

	var affected int64
	err := db.WithTenantTransaction(ctx, s.db.DB, tenantID, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, query, connectionID, tenantID)
		if err != nil {
			return fmt.Errorf("failed to delete connection: %w", err)
		}
		affected, err = result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if affected == 0 {
		return fmt.Errorf("connection not found")
	}

	s.publishGoldCopyChange(ctx, "DELETE", &Connection{ID: connectionID, TenantID: tenantID})

	return nil
}

func (s *ConnectionsService) LinkConnectionToDatasource(ctx context.Context, tenantID, datasourceID, connectionID string) error {
	if err := db.RequireVerifiedTenantFromCtx(ctx); err != nil {
		return fmt.Errorf("security: %w", err)
	}
	var result sql.Result
	err := db.WithTenantTransaction(ctx, s.db.DB, tenantID, func(tx *sql.Tx) error {
		query := `
			UPDATE tenant_product_datasource
			SET connection_id = $1, updated_at = NOW()
			WHERE id = $2
			  AND tenant_product_id IN (
				SELECT id FROM tenant_product WHERE datasource_id IN (
					SELECT id FROM tenant_instance WHERE tenant_id = $3
				)
			  )
			  AND EXISTS (
				SELECT 1 FROM connections
				WHERE id = $1 AND tenant_id = $3
			  )
		`
		var err2 error
		result, err2 = tx.ExecContext(ctx, query, connectionID, datasourceID, tenantID)
		return err2
	})
	if err != nil {
		return fmt.Errorf("failed to link connection to datasource: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("datasource not found")
	}
	return nil
}

func (s *ConnectionsService) UnlinkConnectionFromDatasource(ctx context.Context, tenantID, datasourceID string) error {
	if err := db.RequireVerifiedTenantFromCtx(ctx); err != nil {
		return fmt.Errorf("security: %w", err)
	}
	var result sql.Result
	err := db.WithTenantTransaction(ctx, s.db.DB, tenantID, func(tx *sql.Tx) error {
		query := `
			UPDATE tenant_product_datasource
			SET connection_id = NULL, updated_at = NOW()
			WHERE id = $1 AND tenant_product_id IN (
				SELECT id FROM tenant_product WHERE datasource_id IN (
					SELECT id FROM tenant_instance WHERE tenant_id = $2
				)
			)
		`
		var err2 error
		result, err2 = tx.ExecContext(ctx, query, datasourceID, tenantID)
		return err2
	})
	if err != nil {
		return fmt.Errorf("failed to unlink connection from datasource: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("datasource not found")
	}
	return nil
}

// CreateTenantProductDatasource creates a tenant_product_datasource row,
// scoped to the tenant via WithTenantTransaction/RLS. input.TenantProductID
// must reference a tenant_product belonging to a tenant_instance owned by
// tenantID, and (if set) input.ConnectionID must reference a connection
// owned by tenantID; both are enforced by the WHERE clauses below rather
// than relying solely on RLS, since tenant_product_datasource itself has no
// tenant_id column.
func (s *ConnectionsService) CreateTenantProductDatasource(ctx context.Context, tenantID string, input *models.TenantProductDatasourceInput) (*models.TenantProductDatasource, error) {
	if err := db.RequireVerifiedTenantFromCtx(ctx); err != nil {
		return nil, fmt.Errorf("security: %w", err)
	}
	if input == nil || input.TenantProductID == uuid.Nil {
		return nil, fmt.Errorf("tenant_product_id is required")
	}
	if input.AlphaDatasourceID == uuid.Nil {
		return nil, fmt.Errorf("alpha_datasource_id is required")
	}

	var created models.TenantProductDatasource
	err := db.WithTenantTransaction(ctx, s.db.DB, tenantID, func(tx *sql.Tx) error {
		var ownsTenantProduct bool
		ownsQuery := `
			SELECT EXISTS (
				SELECT 1 FROM tenant_product tp
				JOIN tenant_instance ti ON ti.id = tp.datasource_id
				WHERE tp.id = $1 AND ti.tenant_id = $2
			)
		`
		if err := tx.QueryRowContext(ctx, ownsQuery, input.TenantProductID, tenantID).Scan(&ownsTenantProduct); err != nil {
			return fmt.Errorf("failed to verify tenant product ownership: %w", err)
		}
		if !ownsTenantProduct {
			return fmt.Errorf("tenant product not found")
		}

		if input.ConnectionID != nil {
			var ownsConnection bool
			connQuery := `SELECT EXISTS (SELECT 1 FROM connections WHERE id = $1 AND tenant_id = $2)`
			if err := tx.QueryRowContext(ctx, connQuery, *input.ConnectionID, tenantID).Scan(&ownsConnection); err != nil {
				return fmt.Errorf("failed to verify connection ownership: %w", err)
			}
			if !ownsConnection {
				return fmt.Errorf("connection not found")
			}
		}

		row, err := db.CreateTenantProductDatasource(ctx, tx, input)
		if err != nil {
			return err
		}
		created = row
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

// UpdateTenantProductDatasource updates the inline config/name/active-state
// of a tenant_product_datasource row (used for datasources that store their
// connection details inline rather than through a shared `connections` row).
func (s *ConnectionsService) UpdateTenantProductDatasource(ctx context.Context, tenantID, id string, input *models.TenantProductDatasourceInput) (*models.TenantProductDatasource, error) {
	if err := db.RequireVerifiedTenantFromCtx(ctx); err != nil {
		return nil, fmt.Errorf("security: %w", err)
	}
	if input == nil {
		return nil, fmt.Errorf("input is required")
	}

	var updated models.TenantProductDatasource
	err := db.WithTenantTransaction(ctx, s.db.DB, tenantID, func(tx *sql.Tx) error {
		query := `
			UPDATE tenant_product_datasource
			SET source_name = $1, config = $2, is_active = $3, connection_id = $4, updated_at = NOW()
			WHERE id = $5 AND tenant_id = $6
			RETURNING id, tenant_product_id, tenant_id, COALESCE(alpha_datasource_id, '00000000-0000-0000-0000-000000000000'),
			          COALESCE(source_name, ''), config, is_active, connection_id
		`
		row := tx.QueryRowContext(ctx, query,
			input.SourceName, normalizedMetadata(input.Config), input.IsActive, input.ConnectionID,
			id, tenantID,
		)
		return row.Scan(
			&updated.ID, &updated.TenantProductID, &updated.TenantID, &updated.AlphaDatasourceID,
			&updated.Name, &updated.Config, &updated.IsActive, &updated.ConnectionID,
		)
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("datasource not found")
		}
		return nil, fmt.Errorf("failed to update datasource: %w", err)
	}
	return &updated, nil
}

// DeleteTenantProductDatasource deletes a tenant_product_datasource row
// (a datasource with inline config that isn't backed by a `connections` row).
func (s *ConnectionsService) DeleteTenantProductDatasource(ctx context.Context, tenantID, id string) error {
	if err := db.RequireVerifiedTenantFromCtx(ctx); err != nil {
		return fmt.Errorf("security: %w", err)
	}

	var affected int64
	err := db.WithTenantTransaction(ctx, s.db.DB, tenantID, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `DELETE FROM tenant_product_datasource WHERE id = $1 AND tenant_id = $2`, id, tenantID)
		if err != nil {
			return fmt.Errorf("failed to delete datasource: %w", err)
		}
		affected, err = result.RowsAffected()
		return err
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("datasource not found")
	}
	return nil
}

func (s *ConnectionsService) GetDatasourcesForConnection(ctx context.Context, tenantID, connectionID string) ([]string, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("security: tenant context required")
	}
	var datasourceIDs []string
	err := db.WithTenantTransaction(ctx, s.db.DB, tenantID, func(tx *sql.Tx) error {
		query := `
			SELECT id FROM tenant_product_datasource
			WHERE connection_id = $1 AND tenant_product_id IN (
				SELECT id FROM tenant_product WHERE datasource_id IN (
					SELECT id FROM tenant_instance WHERE tenant_id = $2
				)
			)
		`
		rows, err := tx.QueryContext(ctx, query, connectionID, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			datasourceIDs = append(datasourceIDs, id)
		}
		return rows.Err()
	})
	return datasourceIDs, err
}

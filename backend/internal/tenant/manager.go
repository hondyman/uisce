package tenant

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/audit"
)

// Tenant represents a client or household in the system
type Tenant struct {
	TenantID   uuid.UUID              `json:"tenant_id"`
	TenantCode string                 `json:"tenant_code"`
	TenantName string                 `json:"tenant_name"`
	SchemaName string                 `json:"schema_name"`
	Status     string                 `json:"status"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"created_at"`
}

// TenantManager handles tenant lifecycle and database isolation
type TenantManager struct {
	controlDB    *sql.DB
	auditService *audit.TrinoAuditService
}

// NewTenantManager creates a new TenantManager instance
func NewTenantManager(controlDB *sql.DB, auditService *audit.TrinoAuditService) *TenantManager {
	return &TenantManager{
		controlDB:    controlDB,
		auditService: auditService,
	}
}

// CreateTenant provisions a new tenant with an isolated schema and necessary tables
func (tm *TenantManager) CreateTenant(ctx context.Context, tenantCode, tenantName string) (*Tenant, error) {
	tenant := &Tenant{
		TenantID:   uuid.New(),
		TenantCode: tenantCode,
		TenantName: tenantName,
		SchemaName: fmt.Sprintf("tenant_%s", tenantCode),
		Status:     "active",
		Metadata:   make(map[string]interface{}),
		CreatedAt:  time.Now(),
	}

	tx, err := tm.controlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1. Insert tenant record into control database
	// Try the preferred insert which includes schema_name, but be tolerant of older schemas
	// that may not have that column present. TODO: Replace with Hasura GraphQL mutation.
	_, err = tx.ExecContext(ctx, `
		INSERT INTO public.tenants (id, tenant_code, name, display_name, schema_name, status, metadata)
		VALUES ($1, $2, $3, $3, $4, $5, $6)
	`, tenant.TenantID, tenant.TenantCode, tenant.TenantName, tenant.SchemaName, tenant.Status, "{}")
	if err != nil {
		// Some legacy databases define a different tenants shape (e.g., no schema_name column).
		// Attempt progressively simpler fallbacks. Each attempted Exec may leave the
		// transaction in an aborted state, so rollback and start a fresh transaction
		// before each fallback attempt.
		if strings.Contains(err.Error(), "schema_name") || strings.Contains(err.Error(), "column") {
			// Fallback 1: omit schema_name but include metadata
			_ = tx.Rollback()
			tx, err = tm.controlDB.BeginTx(ctx, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to begin fallback transaction: %w", err)
			}

			_, err = tx.ExecContext(ctx, `
				INSERT INTO public.tenants (id, tenant_code, name, display_name, status, metadata)
				VALUES ($1, $2, $3, $3, $4, $5)
			`, tenant.TenantID, tenant.TenantCode, tenant.TenantName, tenant.Status, "{}")
			if err == nil {
				// continue with this transaction
			} else {
				// If metadata column also missing, try the minimal insert (with display_name)
				_ = tx.Rollback()
				tx, err = tm.controlDB.BeginTx(ctx, nil)
				if err != nil {
					return nil, fmt.Errorf("failed to begin second fallback transaction: %w", err)
				}

				_, err = tx.ExecContext(ctx, `
					INSERT INTO public.tenants (id, tenant_code, name, display_name)
					VALUES ($1, $2, $3, $3)
				`, tenant.TenantID, tenant.TenantCode, tenant.TenantName)
				if err != nil {
					_ = tx.Rollback()
					return nil, fmt.Errorf("failed to insert tenant record (minimal fallback): %w", err)
				}
			}
		} else {
			return nil, fmt.Errorf("failed to insert tenant record: %w", err)
		}
	}

	// 2. Create dedicated schema for physical isolation
	// TODO: DDL operation - keep SQL (CREATE SCHEMA requires superuser/database-level privileges)
	//   Not suitable for Hasura GraphQL (DDL operations should remain in migration/provisioning layer)
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, tenant.SchemaName))
	if err != nil {
		return nil, fmt.Errorf("failed to create schema %s: %w", tenant.SchemaName, err)
	}

	// 3. Enable pgvector extension in the new schema
	// If the DB or environment does not allow creating extensions, warn and continue
	// so tenant creation does not fail in CI/dev envs lacking the extension.
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`CREATE EXTENSION IF NOT EXISTS vector SCHEMA %s`, tenant.SchemaName))
	if err != nil {
		log.Printf("Warning: failed to enable vector extension in schema %s: %v", tenant.SchemaName, err)
	}

	// 4. Create tenant-specific tables
	// To be resilient against earlier non-fatal errors (e.g., extension creation
	// failures that may leave the transaction in an aborted state), commit the
	// current transaction (which contains the tenant record and schema creation)
	// and perform table creation in a fresh transaction.
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Create tables in a new transaction to avoid carrying any aborted state.
	tx2, err := tm.controlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tenant tables transaction: %w", err)
	}
	if err := tm.createTenantTables(ctx, tx2, tenant.SchemaName); err != nil {
		_ = tx2.Rollback()
		return nil, fmt.Errorf("failed to create tenant tables: %w", err)
	}

	if err := tx2.Commit(); err != nil {
		return nil, err
	}

	log.Printf("Successfully created tenant: %s (ID: %s, Schema: %s)", tenant.TenantName, tenant.TenantID, tenant.SchemaName)

	// Audit Log
	if tm.auditService != nil {
		err := tm.auditService.LogEvent(ctx, tenant.TenantID.String(), "system", "", "System", "create", "tenant", tenant.TenantID.String(), map[string]interface{}{
			"code": tenantCode,
			"name": tenantName,
		})
		if err != nil {
			log.Printf("Failed to log audit event: %v", err)
		}
	}

	return tenant, nil
}

// createTenantTables creates the necessary tables within the tenant's schema
func (tm *TenantManager) createTenantTables(ctx context.Context, tx *sql.Tx, schema string) error {
	tables := []string{
		// Documents table
		fmt.Sprintf(`
			CREATE TABLE %s.documents (
				document_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				source_path TEXT NOT NULL,
				document_type VARCHAR(50) NOT NULL,
				title TEXT,
				upload_date TIMESTAMPTZ DEFAULT NOW(),
				file_hash VARCHAR(64),
				file_size_bytes BIGINT,
				metadata JSONB DEFAULT '{}'::jsonb,
				iceberg_table_path TEXT,
				status VARCHAR(20) DEFAULT 'processing',
				updated_at TIMESTAMPTZ DEFAULT NOW(),
				UNIQUE(file_hash)
			)
		`, schema),

		// Document chunks with embeddings
		fmt.Sprintf(`
			CREATE TABLE %s.document_chunks (
				chunk_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				document_id UUID REFERENCES %s.documents(document_id) ON DELETE CASCADE,
				chunk_index INTEGER NOT NULL,
				chunk_type VARCHAR(50),
				content TEXT NOT NULL,
				content_hash VARCHAR(64),
				embedding vector(1536),
				token_count INTEGER,
				metadata JSONB DEFAULT '{}'::jsonb,
				created_at TIMESTAMPTZ DEFAULT NOW(),
				UNIQUE(document_id, chunk_index)
			)
		`, schema, schema),

		// Query logs (per-tenant view)
		fmt.Sprintf(`
			CREATE TABLE %s.query_logs (
				query_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				user_id VARCHAR(255),
				query_text TEXT NOT NULL,
				query_embedding vector(1536),
				retrieved_chunks UUID[],
				response_text TEXT,
				feedback_score INTEGER,
				created_at TIMESTAMPTZ DEFAULT NOW()
			)
		`, schema),
	}

	// TODO: DDL operation - keep SQL (CREATE TABLE requires schema modification privileges)
	//   Not suitable for Hasura GraphQL (table provisioning should remain in migration layer)
	//   Consider moving to separate migration system or Hasura metadata API for table tracking
	for _, tableSQL := range tables {
		if _, err := tx.ExecContext(ctx, tableSQL); err != nil {
			return err
		}
	}

	// Create indexes for performance
	indexes := []string{
		// Vector index for similarity search
		fmt.Sprintf(`CREATE INDEX idx_chunks_embedding ON %s.document_chunks USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100)`, schema),
		// Standard indexes
		fmt.Sprintf(`CREATE INDEX idx_chunks_document ON %s.document_chunks(document_id)`, schema),
		fmt.Sprintf(`CREATE INDEX idx_chunks_metadata ON %s.document_chunks USING gin(metadata)`, schema),
		fmt.Sprintf(`CREATE INDEX idx_documents_type ON %s.documents(document_type)`, schema),
	}

	// TODO: DDL operation - keep SQL (CREATE INDEX requires schema modification privileges)
	//   Not suitable for Hasura GraphQL (index management should remain at database level)
	//   pgvector IVFFLAT index requires native SQL for performance tuning
	for _, indexSQL := range indexes {
		if _, err := tx.ExecContext(ctx, indexSQL); err != nil {
			return err
		}
	}

	return nil
}

// GetTenantConnection returns a database connection scoped to the tenant's schema
// This enforces physical isolation at the database session level
func (tm *TenantManager) GetTenantConnection(ctx context.Context, tenantID uuid.UUID) (*sql.Conn, error) {
	// 1. Resolve Tenant ID to Schema Name
	// TODO: Replace with Hasura GraphQL query:
	//   query { tenants_by_pk(id: $tenantID) { schema_name } } where status = 'active'
	var schemaName string
	err := tm.controlDB.QueryRowContext(ctx, `
		SELECT schema_name FROM public.tenants WHERE id = $1 AND status = 'active'
	`, tenantID).Scan(&schemaName)
	if err != nil {
		// If the tenants table does not have a schema_name column, derive the
		// expected schema name from the tenant_code column (legacy schemas).
		if strings.Contains(err.Error(), "schema_name") || strings.Contains(err.Error(), "column") {
			var tenantCode string
			if err2 := tm.controlDB.QueryRowContext(ctx, `SELECT tenant_code FROM public.tenants WHERE id = $1`, tenantID).Scan(&tenantCode); err2 != nil {
				return nil, fmt.Errorf("failed to resolve tenant schema (fallback): %w", err2)
			}
			schemaName = fmt.Sprintf("tenant_%s", tenantCode)
		} else {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("tenant not found or inactive: %s", tenantID)
			}
			return nil, fmt.Errorf("failed to resolve tenant schema: %w", err)
		}
	}

	// 2. Acquire a connection from the pool
	conn, err := tm.controlDB.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire database connection: %w", err)
	}

	// 3. Set search_path to enforce isolation
	// This ensures all subsequent queries on this connection are limited to the tenant's schema
	// TODO: Session-level SQL - keep as-is (SET search_path is connection-scoped, not data operation)
	//   Not suitable for Hasura GraphQL (connection pooling managed at database/middleware level)
	//   Consider Hasura's role-based schema access or custom JWT claims for tenant isolation
	_, err = conn.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s, public", schemaName))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to set search_path to %s, public: %w", schemaName, err)
	}

	return conn, nil
}

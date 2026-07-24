package platform

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// TenantDBManager manages database connections for tenants
type TenantDBManager struct {
	centralDB *sql.DB
	pools     map[string]*sql.DB
	mu        sync.RWMutex
}

// NewTenantDBManager creates a new TenantDBManager
func NewTenantDBManager(centralDB *sql.DB) *TenantDBManager {
	return &TenantDBManager{
		centralDB: centralDB,
		pools:     make(map[string]*sql.DB),
	}
}

// GetConnection retrieves or creates a database connection for a specific tenant
func (m *TenantDBManager) GetConnection(tenantID string) (*sql.DB, error) {
	// 1. Check if we already have a connection pool for this tenant
	m.mu.RLock()
	if db, ok := m.pools[tenantID]; ok {
		m.mu.RUnlock()
		// Validate connection is still alive
		if err := db.Ping(); err == nil {
			return db, nil
		}
		// If ping fails, we'll try to reconnect below
	}
	m.mu.RUnlock()

	// 2. We need to create a new connection. Lock for writing.
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check in case another goroutine created it while we were waiting for the lock
	if db, ok := m.pools[tenantID]; ok {
		if err := db.Ping(); err == nil {
			return db, nil
		}
		// Close bad connection if it exists
		db.Close()
		delete(m.pools, tenantID)
	}

	// 3. Fetch connection string from central DB
	var connStr string
	var hasDedicatedDB bool
	query := `SELECT db_connection_string, has_dedicated_db FROM platform.tenants WHERE tenant_id = $1`
	err := m.centralDB.QueryRow(query, tenantID).Scan(&connStr, &hasDedicatedDB)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found: %s", tenantID)
		}
		return nil, fmt.Errorf("failed to fetch tenant config: %v", err)
	}

	// 4. If no dedicated DB, fallback to central DB (if that's the policy) or error
	// For this architecture, we assume dedicated DB is required for wealth data.
	// However, if has_dedicated_db is false, maybe we return the central DB?
	// The user requirement is "separate physical database per client".
	// If connStr is empty, we can't connect.
	if connStr == "" {
		return nil, fmt.Errorf("no database connection string configured for tenant %s", tenantID)
	}

	// 5. Open new connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open tenant database: %v", err)
	}

	// Configure pool settings (could be configurable per tenant too)
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping tenant database: %v", err)
	}

	// 6. Store in map
	m.pools[tenantID] = db

	return db, nil
}

// CloseAll closes all tenant connections
func (m *TenantDBManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, db := range m.pools {
		db.Close()
		delete(m.pools, id)
	}
}

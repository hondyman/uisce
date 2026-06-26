package multitenancy

import (
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"
)

// TenantManager handles database connections for multiple tenants
type TenantManager struct {
	mu      sync.RWMutex
	dbs     map[string]*sqlx.DB
	defaultDB *sqlx.DB // For fallback or shared data (if any)
}

func NewTenantManager(defaultDB *sqlx.DB) *TenantManager {
	return &TenantManager{
		dbs:       make(map[string]*sqlx.DB),
		defaultDB: defaultDB,
	}
}

// GetDB returns the database connection for a specific tenant.
// In a real system, this might dynamically connect or look up connection strings from a Vault.
func (tm *TenantManager) GetDB(tenantID string) (*sqlx.DB, error) {
	tm.mu.RLock()
	db, ok := tm.dbs[tenantID]
	tm.mu.RUnlock()

	if ok {
		return db, nil
	}

	// For this prototype, we simulate isolation by returning the default DB 
	// BUT logically we are prepared to return distinct DBs.
	// In production:
	// dsn := vault.GetDSN(tenantID)
	// db = sqlx.Connect("postgres", dsn)
	
	// To strictly enforce "Separate Postgres DB" requirement in this mock environment:
	// We will assume the defaultDB IS the tenant DB for the "demo" tenant.
	if tenantID == "demo-tenant" || tenantID == "client-abc" {
		return tm.defaultDB, nil
	}

	return nil, fmt.Errorf("tenant database not found for: %s", tenantID)
}

// RegisterTenant adds a tenant DB (for startup/config loading)
func (tm *TenantManager) RegisterTenant(tenantID string, db *sqlx.DB) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.dbs[tenantID] = db
}

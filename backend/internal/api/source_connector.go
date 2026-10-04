package api

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"

	uiscedb "github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/sourceconn"
)

// newSourceConnector builds the one audited opener of tenant SOURCE databases (ADR-030). The caller
// puts the verified tenant in the context (db.WithTenantContextToCtx) before every call.
func newSourceConnector(alpha *sql.DB) (*sourceconn.Connector, error) {
	return sourceconn.New(sourceconn.Config{
		Registry:        &sourceconn.AlphaRegistry{DB: alpha, Resolver: security.NewDBDatasourceResolver(sqlx.NewDb(alpha, "pgx"))},
		Credentials:     sourceconn.DSCreds{},
		CallerTenant:    uiscedb.GetTenantIDFromCtx,
		MaxPools:        64,
		MaxConnsPerPool: 4,
		IdleTTL:         10 * time.Minute,
		DialTimeout:     10 * time.Second,
		AuthTTL:         sourceconn.DefaultAuthTTL,
		Warn:            func(msg string) { logging.GetLogger().Sugar().Warn(msg) },
	})
}

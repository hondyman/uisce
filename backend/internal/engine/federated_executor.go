package engine

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hondyman/uisce/backend/internal/domain"
)

// FederatedExecutor routes queries to the appropriate database pool (PostgreSQL vs StarRocks/Trino)
type FederatedExecutor struct {
	postgresPool  *sql.DB
	starRocksPool *sql.DB
}

// NewFederatedExecutor constructs a FederatedExecutor given Postgres and StarRocks/Trino connections
func NewFederatedExecutor(pg *sql.DB, sr *sql.DB) *FederatedExecutor {
	return &FederatedExecutor{
		postgresPool:  pg,
		starRocksPool: sr,
	}
}

// Execute enforces Rule 7 security checks and executes the query against the target database pool
func (e *FederatedExecutor) Execute(ctx context.Context, req domain.ExecutionRequest) (*sql.Rows, error) {
	// Rule 7 check: context MUST contain tenant_id
	tenantID := ctx.Value("tenant_id")
	if tenantID == nil || tenantID == "" {
		return nil, fmt.Errorf("FATAL: execution blocked, missing tenant_id in context")
	}

	pool := e.getPool(req.TargetEngine)
	if pool == nil {
		return nil, fmt.Errorf("no database connection pool available for engine: %s", req.TargetEngine)
	}

	return pool.QueryContext(ctx, req.GeneratedSQL, req.Args...)
}

// Explain executes an EXPLAIN command against the target database engine
func (e *FederatedExecutor) Explain(ctx context.Context, req domain.ExecutionRequest) (string, error) {
	// Rule 7 check: context MUST contain tenant_id
	tenantID := ctx.Value("tenant_id")
	if tenantID == nil || tenantID == "" {
		return "", fmt.Errorf("FATAL: explain execution blocked, missing tenant_id in context")
	}

	pool := e.getPool(req.TargetEngine)
	if pool == nil {
		return "", fmt.Errorf("no database connection pool available for engine: %s", req.TargetEngine)
	}

	explainSQL := fmt.Sprintf("EXPLAIN %s", req.GeneratedSQL)
	rows, err := pool.QueryContext(ctx, explainSQL, req.Args...)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var plan string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return "", err
		}
		plan += line + "\n"
	}
	return plan, nil
}

func (e *FederatedExecutor) getPool(engineType string) *sql.DB {
	if engineType == "STARROCKS_FEDERATED" || engineType == "TRINO_COLD" {
		if e.starRocksPool != nil {
			return e.starRocksPool
		}
	}
	return e.postgresPool
}

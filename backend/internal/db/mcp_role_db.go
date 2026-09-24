package db

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

var safeRoleName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// OpenMCPAppDB opens a sqlx DB for the MCP server under staged MCP-first cutover.
//
// Prefer UISCE_APP_DSN when the role can TCP-login. Otherwise open parentDSN
// (typically DATABASE_URL as postgres) and SET ROLE uisce_mcp_app on every
// new connection so FORCE RLS binds without flipping the whole backend pool.
//
// Returns (db, mode, err) where mode is "uisce-app-dsn" or "set-role:"+role.
func OpenMCPAppDB(parentDSN string) (*sqlx.DB, string, error) {
	role := os.Getenv("UISCE_MCP_DB_ROLE")
	if role == "" {
		role = "uisce_mcp_app"
	}
	if !safeRoleName.MatchString(role) {
		return nil, "", fmt.Errorf("OpenMCPAppDB: invalid role name %q", role)
	}

	dsn := os.Getenv("UISCE_APP_DSN")
	mode := "uisce-app-dsn"
	useSetRole := false
	if dsn == "" {
		dsn = parentDSN
		mode = "set-role:" + role
		useSetRole = true
	}
	if dsn == "" {
		return nil, "", fmt.Errorf("OpenMCPAppDB: no DSN (UISCE_APP_DSN or parent)")
	}

	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, "", fmt.Errorf("OpenMCPAppDB: parse dsn: %w", err)
	}
	if useSetRole {
		r := role
		// pgx v5: AfterConnect lives on the nested pgconn.Config, not on pgx.ConnConfig.
		// pgconn.PgConn.Exec runs a simple protocol command — correct for SET ROLE.
		cfg.Config.AfterConnect = func(ctx context.Context, c *pgconn.PgConn) error {
			_, err := c.Exec(ctx, "SET ROLE "+r).ReadAll()
			return err
		}
	}

	sqlDB := stdlib.OpenDB(*cfg)
	sqlDB.SetMaxOpenConns(8)
	sqlDB.SetMaxIdleConns(4)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var user string
	if err := sqlDB.QueryRowContext(ctx, "SELECT current_user").Scan(&user); err != nil {
		sqlDB.Close()
		if useSetRole {
			return nil, "", fmt.Errorf("OpenMCPAppDB: ping/set-role failed: %w", err)
		}
		// Direct app DSN failed (e.g. pg_hba) — fall back to set-role on parent.
		return OpenMCPAppDBWithSetRole(parentDSN, role)
	}
	if useSetRole && user != role {
		sqlDB.Close()
		return nil, "", fmt.Errorf("OpenMCPAppDB: expected current_user=%s after SET ROLE, got %s", role, user)
	}
	return sqlx.NewDb(sqlDB, "pgx"), mode, nil
}

// OpenMCPAppDBWithSetRole forces parentDSN + SET ROLE (used when UISCE_APP_DSN TCP fails).
func OpenMCPAppDBWithSetRole(parentDSN, role string) (*sqlx.DB, string, error) {
	if parentDSN == "" {
		return nil, "", fmt.Errorf("OpenMCPAppDBWithSetRole: empty parent DSN")
	}
	if !safeRoleName.MatchString(role) {
		return nil, "", fmt.Errorf("OpenMCPAppDBWithSetRole: invalid role %q", role)
	}
	cfg, err := pgx.ParseConfig(parentDSN)
	if err != nil {
		return nil, "", err
	}
	// pgx v5: AfterConnect on nested pgconn.Config.
	cfg.Config.AfterConnect = func(ctx context.Context, c *pgconn.PgConn) error {
		_, err := c.Exec(ctx, "SET ROLE "+role).ReadAll()
		return err
	}
	sqlDB := stdlib.OpenDB(*cfg)
	sqlDB.SetMaxOpenConns(8)
	sqlDB.SetMaxIdleConns(4)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var user string
	if err := sqlDB.QueryRowContext(ctx, "SELECT current_user").Scan(&user); err != nil {
		sqlDB.Close()
		return nil, "", err
	}
	if user != role {
		sqlDB.Close()
		return nil, "", fmt.Errorf("SET ROLE ineffective: current_user=%s want=%s", user, role)
	}
	_ = user
	return sqlx.NewDb(sqlDB, "pgx"), "set-role:" + role, nil
}

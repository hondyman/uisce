package tenantdbcreds

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// PgxConnector opens admin connections over TLS. The maintenance database is
// "postgres", because CREATE ROLE is cluster-wide.
type PgxConnector struct{}

// Connect dials the region's Postgres as the admin. The credential is escaped
// through net/url, so reserved characters in the password are safe.
func (PgxConnector) Connect(ctx context.Context, host string, port int, adminUser, adminPassword string) (RoleAdmin, error) {
	dsn := (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(adminUser, adminPassword),
		Host:     net.JoinHostPort(host, strconv.Itoa(port)),
		Path:     "/postgres",
		RawQuery: "sslmode=require",
	}).String()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return &pgxRoleAdmin{conn: conn}, nil
}

type pgxRoleAdmin struct {
	conn *pgx.Conn
}

func (p *pgxRoleAdmin) RoleExists(ctx context.Context, role string) (bool, error) {
	var exists bool
	err := p.conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, role).Scan(&exists)
	return exists, err
}

// CreateRole creates a login role. CREATE ROLE takes no bind parameters, so the
// password is embedded. It is safe only because the caller passes a value from
// the restricted generator alphabet, which is checked here as well.
func (p *pgxRoleAdmin) CreateRole(ctx context.Context, role, password string) error {
	if !safePassword.MatchString(password) {
		return errors.New("password contains characters that cannot be embedded safely")
	}
	stmt := fmt.Sprintf("CREATE ROLE %s LOGIN PASSWORD '%s'", pgx.Identifier{role}.Sanitize(), password)
	_, err := p.conn.Exec(ctx, stmt)
	return err
}

func (p *pgxRoleAdmin) GrantConnect(ctx context.Context, role, database string) error {
	stmt := fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s", pgx.Identifier{database}.Sanitize(), pgx.Identifier{role}.Sanitize())
	_, err := p.conn.Exec(ctx, stmt)
	return err
}

func (p *pgxRoleAdmin) DropRole(ctx context.Context, role string) error {
	_, err := p.conn.Exec(ctx, "DROP ROLE IF EXISTS "+pgx.Identifier{role}.Sanitize())
	return err
}

func (p *pgxRoleAdmin) Close() error {
	return p.conn.Close(context.Background())
}

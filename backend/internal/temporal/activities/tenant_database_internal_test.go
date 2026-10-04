package activities

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestClassifyConnectError(t *testing.T) {
	pg := func(code string) error { return fmt.Errorf("failed to connect: %w", &pgconn.PgError{Code: code}) }
	for name, tc := range map[string]struct {
		err  error
		want connectOutcome
	}{
		"CONNECT denied":                     {pg("42501"), connectDenied},
		"the database was dropped meanwhile": {pg("3D000"), connectGone},
		"too many connections for database":  {pg("53300"), connectUnknown},
		"database not accepting connections": {pg("57P03"), connectUnknown},
		"invalid password":                   {pg("28P01"), connectUnknown},
		"invalid authorization":              {pg("28000"), connectUnknown},
		"server shutting down":               {pg("57P01"), connectUnknown},
		"a network failure":                  {errors.New("dial tcp: connection refused"), connectUnknown},
		"nil-ish wrapped non-pg error":       {fmt.Errorf("wrapped: %w", errors.New("x")), connectUnknown},
	} {
		require.Equal(t, tc.want, classifyConnectError(tc.err), name)
	}
}

// The order is the guarantee: never a moment where the database is connectable AND open to PUBLIC.
func TestCreateTenantDatabaseStatements_NeverConnectableWhileOpenToPublic(t *testing.T) {
	s := createTenantDatabaseStatements("tenant_acme")
	require.Contains(t, s.create, "ALLOW_CONNECTIONS false", "it must be created with connections disabled")
	require.True(t, strings.HasPrefix(s.create, "CREATE DATABASE"))
	require.Contains(t, s.closePublic, "REVOKE CONNECT")
	require.Contains(t, s.closePublic, "FROM PUBLIC")
	require.Contains(t, s.allowConnections, "ALLOW_CONNECTIONS true")
	for _, stmt := range []string{s.create, s.closePublic, s.allowConnections} {
		require.Contains(t, stmt, `"tenant_acme"`, "the identifier is quoted")
	}
}

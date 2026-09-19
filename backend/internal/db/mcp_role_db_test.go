package db

import (
	"os"
	"testing"
)

func TestOpenMCPAppDB_RequiresDSN(t *testing.T) {
	os.Unsetenv("UISCE_APP_DSN")
	_, _, err := OpenMCPAppDB("")
	if err == nil {
		t.Fatal("expected error for empty DSN")
	}
}

func TestOpenMCPAppDB_RejectsBadRole(t *testing.T) {
	t.Setenv("UISCE_MCP_DB_ROLE", "bad;role")
	t.Setenv("UISCE_APP_DSN", "")
	_, _, err := OpenMCPAppDB("postgres://u:p@localhost/db")
	if err == nil {
		t.Fatal("expected invalid role error")
	}
}

func TestOpenMCPAppDB_LiveSetRole(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("POSTGRES_DSN")
	}
	if dsn == "" {
		t.Skip("no DATABASE_URL")
	}
	t.Setenv("UISCE_APP_DSN", "") // force SET ROLE path (direct TCP often pg_hba-blocked)
	t.Setenv("UISCE_MCP_DB_ROLE", "uisce_mcp_app")
	xdb, mode, err := OpenMCPAppDB(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer xdb.Close()
	var user string
	if err := xdb.QueryRow("SELECT current_user").Scan(&user); err != nil {
		t.Fatal(err)
	}
	if user != "uisce_mcp_app" {
		t.Fatalf("current_user=%s want uisce_mcp_app mode=%s", user, mode)
	}
	t.Logf("MCP pool live: mode=%s user=%s", mode, user)
}

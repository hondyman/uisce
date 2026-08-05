package testutil

import (
	"os"
	"testing"
)

func GoldCopyTenantID(t *testing.T) string {
	t.Helper()
	v := os.Getenv("GOLD_COPY_TENANT_ID")
	if v == "" {
		t.Skip("GOLD_COPY_TENANT_ID env not set — run `export GOLD_COPY_TENANT_ID=$(psql ... -t -c \"SELECT id FROM tenants WHERE gold_copy=true LIMIT 1\")` first")
	}
	return v
}

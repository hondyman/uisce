package provisioning

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// tenantCodeRe is the strict allowlist for tenant codes. A tenant code is
	// interpolated into a database name, a Lakekeeper namespace and a workflow
	// ID, so it must never contain anything but lowercase letters, digits and
	// underscores.
	tenantCodeRe = regexp.MustCompile(`^[a-z][a-z0-9_]{1,40}$`)

	// databaseNameRe is the allowlist for a Postgres database name handed to
	// DDL or to pg_dump/psql. Tenant databases are always lowercase; the gold
	// copy database is allowed mixed case. Postgres caps identifiers at 63 bytes.
	databaseNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,62}$`)
)

// ValidateTenantCode reports whether code is acceptable as a tenant code.
func ValidateTenantCode(code string) error {
	if !tenantCodeRe.MatchString(code) {
		return fmt.Errorf("invalid tenant code: must match %s", tenantCodeRe.String())
	}
	return nil
}

// ValidateDatabaseName reports whether name is safe to use as a database
// identifier or as a pg_dump/psql database argument.
func ValidateDatabaseName(name string) error {
	if !databaseNameRe.MatchString(name) {
		return fmt.Errorf("invalid database name: must match %s", databaseNameRe.String())
	}
	return nil
}

// TenantDatabaseName derives the tenant database name from a tenant code,
// validating the code first.
func TenantDatabaseName(code string) (string, error) {
	if err := ValidateTenantCode(code); err != nil {
		return "", err
	}
	return "tenant_" + strings.ToLower(code), nil
}

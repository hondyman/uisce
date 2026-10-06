package querybuilder

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/hondyman/uisce/backend/internal/iceberg"
)

// IcebergCatalogTokenRefresher mints a Keycloak client-credentials token and
// writes it onto a StarRocks external Iceberg catalog property. Used because
// StarRocks 3.3.22 oauth2.credential auto-refresh is broken for Lakekeeper in
// this stack (Track B / CUBE cold path).
type IcebergCatalogTokenRefresher interface {
	Ensure(ctx context.Context, catalog string) error
}

// icebergTokenRefreshEnabled is the process gate for ApplyCold token refresh.
// Off by default so unit tests stay hermetic without Keycloak.
func icebergTokenRefreshEnabled() bool {
	v := strings.TrimSpace(os.Getenv("CUBE_ICEBERG_TOKEN_REFRESH"))
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// newDefaultIcebergCatalogTokenRefresher returns a refresher when the env gate
// is on and a StarRocks DB is available. Returns nil when refresh is disabled
// (ApplyCold skips). Misconfigured secrets surface as Ensure errors, not as a
// silent nil, once the gate is on.
func newDefaultIcebergCatalogTokenRefresher(db *sql.DB) IcebergCatalogTokenRefresher {
	if db == nil || !icebergTokenRefreshEnabled() {
		return nil
	}
	tokenURL := strings.TrimSpace(os.Getenv("LAKEKEEPER_TOKEN_URL"))
	clientID := strings.TrimSpace(os.Getenv("LAKEKEEPER_CLIENT_ID"))
	if clientID == "" {
		clientID = "uisce-provisioner"
	}
	clientSecret := strings.TrimSpace(os.Getenv("LAKEKEEPER_CLIENT_SECRET"))
	var tm *iceberg.TokenManager
	if tokenURL != "" && clientSecret != "" {
		tm = iceberg.NewTokenManager(tokenURL, clientID, clientSecret)
	}
	return &starRocksCatalogTokenRefresher{db: db, tm: tm}
}

type starRocksCatalogTokenRefresher struct {
	db *sql.DB
	tm *iceberg.TokenManager
}

func (r *starRocksCatalogTokenRefresher) Ensure(ctx context.Context, catalog string) error {
	ident := sanitizeIdentifier(catalog)
	if ident == "" {
		return fmt.Errorf("iceberg catalog token refresh: empty catalog name")
	}
	if r == nil || r.db == nil {
		return fmt.Errorf("iceberg catalog token refresh: starrocks connection is not configured")
	}
	if r.tm == nil {
		return fmt.Errorf("iceberg catalog token refresh: LAKEKEEPER_TOKEN_URL and LAKEKEEPER_CLIENT_SECRET are required when CUBE_ICEBERG_TOKEN_REFRESH is enabled")
	}
	tok, err := r.tm.Token(ctx)
	if err != nil {
		return fmt.Errorf("iceberg catalog token refresh: mint token: %w", err)
	}
	// StarRocks docs: ALTER CATALOG name SET PROPERTIES (...).
	q := fmt.Sprintf(
		`ALTER CATALOG %s SET PROPERTIES ("iceberg.catalog.token" = %s)`,
		quoteStarRocksIdent(ident),
		quoteStarRocksString(tok),
	)
	if _, err := r.db.ExecContext(ctx, q); err != nil {
		return fmt.Errorf("iceberg catalog token refresh: ALTER CATALOG %s: %w", ident, redactCatalogAuthSecrets(err, tok))
	}
	return nil
}

// redactCatalogAuthSecrets strips bearer tokens from error text so Temporal /
// logs never retain a live Keycloak access token.
func redactCatalogAuthSecrets(err error, secrets ...string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	for _, s := range secrets {
		if s != "" && strings.Contains(msg, s) {
			msg = strings.ReplaceAll(msg, s, "***")
		}
	}
	return fmt.Errorf("%s", msg)
}

package tenantplatform

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// SecretReader reads one secret map. secrets.Provider satisfies it.
type SecretReader interface {
	GetMap(ctx context.Context, key string) (map[string]string, error)
}

const (
	databaseURLKey       = "database_url"
	databaseProbeTimeout = 5 * time.Second
)

// PgDatabaseProbe runs SELECT 1 against the database at an instance path. It reads
// the URL SeedInstanceSecrets wrote, so the check uses the seeded tenant credential
// and the endpoint the saga provisioned. The URL never leaves this function.
type PgDatabaseProbe struct {
	Secrets SecretReader
}

// Ping returns nil only when the database answers SELECT 1. Errors never carry
// the URL, because driver errors can quote it.
func (p *PgDatabaseProbe) Ping(ctx context.Context, instancePath string) error {
	if p.Secrets == nil {
		return errors.New("database probe has no secret store")
	}
	values, err := p.Secrets.GetMap(ctx, instancePath)
	if err != nil {
		return errors.New("read database URL: the secret store could not be read")
	}
	url := values[databaseURLKey]
	if url == "" {
		return errors.New("database URL is missing from the instance path")
	}
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		// The parse error can quote the URL, so it is not wrapped.
		return errors.New("database URL could not be parsed")
	}
	ctx, cancel := context.WithTimeout(ctx, databaseProbeTimeout)
	defer cancel()
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return errors.New("database connection failed")
	}
	defer conn.Close(context.Background())
	var one int
	if err := conn.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil || one != 1 {
		return errors.New("database did not answer SELECT 1")
	}
	return nil
}

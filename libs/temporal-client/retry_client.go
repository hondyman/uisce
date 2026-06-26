package temporalclient

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"go.temporal.io/sdk/client"
)

// NewClientWithRetry creates a Temporal client using env vars and retries until
// a connection is established or attempts are exhausted.
// Env supported:
//   - TEMPORAL_HOST or TEMPORAL_ADDRESS or TEMPORAL_HOSTPORT: host:port
//   - TEMPORAL_RETRY_ATTEMPTS: integer attempts (default 40)
//   - TEMPORAL_RETRY_DELAY_SECONDS: delay in seconds between attempts (default 3)
func NewClientWithRetry() (client.Client, error) {
	host := os.Getenv("TEMPORAL_HOST")
	if host == "" {
		host = os.Getenv("TEMPORAL_ADDRESS")
	}
	if host == "" {
		host = os.Getenv("TEMPORAL_HOSTPORT")
	}
	if host == "" {
		// Default to the compose service name so services running inside Docker
		// can resolve Temporal by service name. Local dev can override via env.
		host = "temporal:7233"
	}

	attempts := 40
	if v := os.Getenv("TEMPORAL_RETRY_ATTEMPTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			attempts = n
		}
	}
	delay := 3 * time.Second
	if v := os.Getenv("TEMPORAL_RETRY_DELAY_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			delay = time.Duration(n) * time.Second
		}
	}

	var lastErr error
	for i := 1; i <= attempts; i++ {
		c, err := client.Dial(client.Options{HostPort: host})
		if err == nil {
			return c, nil
		}
		lastErr = err
		time.Sleep(delay)
	}
	return nil, fmt.Errorf("failed to connect to Temporal at %s after %d attempts: %w", host, attempts, lastErr)
}

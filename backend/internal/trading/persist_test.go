package trading

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

// The ORM database is never guessed. These need no database: the open must be refused before any
// connection is attempted.
func TestOpenCRIMS_RequiresItsOwnDSNAndNeverGuesses(t *testing.T) {
	// A control-plane DSN that a guess would have turned into a crims connection.
	t.Setenv("DATABASE_URL", "postgresql://u:p@127.0.0.1:1/alpha?sslmode=disable")
	t.Setenv("POSTGRES_DSN", "postgresql://u:p@127.0.0.1:1/alpha?sslmode=disable")
	t.Setenv("CRIMS_ORM_DSN", "")

	db, err := OpenCRIMS(context.Background())
	if db != nil {
		db.Close()
		t.Fatal("OpenCRIMS must not return a connection when CRIMS_ORM_DSN is unset")
	}
	if !errors.Is(err, ErrCRIMSNotConfigured) {
		t.Fatalf("want ErrCRIMSNotConfigured, got %v", err)
	}
	if !strings.Contains(err.Error(), "never derived from DATABASE_URL") {
		t.Fatalf("the error must say what to set and that nothing is derived: %v", err)
	}
}

func TestOpenCRIMS_AnUnreachableDSNIsAnErrorNotAFallback(t *testing.T) {
	t.Setenv("CRIMS_ORM_DSN", "postgresql://u:p@127.0.0.1:1/crims?sslmode=disable&connect_timeout=1")
	if db, err := OpenCRIMS(context.Background()); err == nil {
		db.Close()
		t.Fatal("an unreachable CRIMS_ORM_DSN must be an error")
	}
}

// persist.go must not grow a way to derive the database from another setting again.
func TestPersistDoesNotDeriveTheCRIMSDatabase(t *testing.T) {
	src, err := os.ReadFile("persist.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(src), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		for _, banned := range []string{`Getenv("DATABASE_URL")`, `Getenv("POSTGRES_DSN")`, "swapDBName", "replaceKV"} {
			if strings.Contains(line, banned) {
				t.Fatalf("persist.go must not use %s to find the ORM database: %s", banned, strings.TrimSpace(line))
			}
		}
	}
}

package trading

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	platformdb "github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/tenantdb"
)

// The ORM database is no longer opened from a DSN at all. It is resolved from
// the registry for the CALLING tenant (ADR-030), so what these tests pin is the
// refusal behaviour: nothing is guessed, nothing falls back, and an uninstalled
// or refusing resolver is an error rather than a shared database.

type fakeResolver struct {
	gotApp   string
	gotCtx   context.Context
	pool     *tenantdb.Pool
	err      error
	resolved int
}

func (f *fakeResolver) ResolveApp(ctx context.Context, app string) (*tenantdb.Pool, error) {
	f.gotApp, f.gotCtx, f.resolved = app, ctx, f.resolved+1
	return f.pool, f.err
}

func withResolver(t *testing.T, r Resolver) {
	t.Helper()
	prev := resolver
	SetResolver(r)
	t.Cleanup(func() { resolver = prev })
}

func TestNoResolverFailsClosed(t *testing.T) {
	withResolver(t, nil)

	_, err := ormDB(context.Background(), uuid.New())
	if !errors.Is(err, ErrNoResolver) {
		t.Fatalf("without a resolver every ORM call must fail closed with ErrNoResolver, got %v", err)
	}
	if !strings.Contains(err.Error(), "ADR-030") {
		t.Fatalf("the error should name the decision that makes it closed: %v", err)
	}
}

func TestAnEmptyTenantIsRefusedBeforeResolving(t *testing.T) {
	f := &fakeResolver{}
	withResolver(t, f)

	_, err := ormDB(context.Background(), uuid.Nil)
	if err == nil {
		t.Fatal("an activity with no tenant id must not resolve anything")
	}
	if f.resolved != 0 {
		t.Fatalf("the resolver was called for an empty tenant (%d calls); the refusal must come first", f.resolved)
	}
}

func TestResolvesTheORMAppNotCore(t *testing.T) {
	f := &fakeResolver{err: errors.New("stop here")}
	withResolver(t, f)

	tenant := uuid.New()
	_, _ = ormDB(context.Background(), tenant)

	// The wrong app code is a runtime refusal, not a compile error, so pin it.
	if f.gotApp != "orm" {
		t.Fatalf("resolved app %q; a tenant's ORM database is app \"orm\", not \"core\"", f.gotApp)
	}
	if f.gotApp == "core" {
		t.Fatal("resolved the wealth/core database for an ORM call")
	}
}

func TestTheActivityTenantIsPlacedInTheContext(t *testing.T) {
	// The Temporal activities have no request context, so the tenant comes from
	// the activity INPUT and must reach the router as the caller tenant. If it
	// did not, the router would refuse every call with no tenant in context, and
	// if it carried the wrong tenant the write would land in another tenant's
	// database.
	f := &fakeResolver{err: errors.New("stop here")}
	withResolver(t, f)

	tenant := uuid.New()
	_, _ = ormDB(context.Background(), tenant)

	got, err := platformdb.GetTenantIDFromCtx(f.gotCtx)
	if err != nil {
		t.Fatalf("the router's context carries no tenant: %v", err)
	}
	if got != tenant.String() {
		t.Fatalf("context tenant %q, want %q", got, tenant)
	}
}

func TestARefusedResolverIsPropagatedNotSwallowed(t *testing.T) {
	boom := errors.New("datasource belongs to another tenant")
	f := &fakeResolver{err: boom}
	withResolver(t, f)

	_, err := ormDB(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("a refused resolution must be an error")
	}
	if !strings.Contains(err.Error(), "another tenant") {
		t.Fatalf("the reason must survive to the caller: %v", err)
	}
	if !strings.Contains(err.Error(), "orm database for tenant") {
		t.Fatalf("the error should say what was being resolved: %v", err)
	}
}

// The old invariant still holds, in a stronger form: there is no DSN to derive
// from at all, so the old "never guess from DATABASE_URL" guard is now
// structural. This test keeps the file honest about that.
func TestPersistDoesNotOpenADatabaseOrReadADSN(t *testing.T) {
	src, err := os.ReadFile("persist.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	for _, banned := range []string{
		`Getenv("DATABASE_URL")`, `Getenv("POSTGRES_DSN")`, `Getenv("CRIMS_ORM_DSN")`,
		"sql.Open", "swapDBName", "replaceKV",
	} {
		if strings.Contains(body, banned) {
			t.Fatalf("persist.go must not use %s: the ORM is reached only through tenantdb (ADR-030)", banned)
		}
	}
}

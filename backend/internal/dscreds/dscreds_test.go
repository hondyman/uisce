package dscreds

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/hondyman/uisce/backend/internal/secrets"
)

const (
	tenantA = "99e99e99-99e9-49e9-89e9-99e99e99e999"
	tenantB = "11111111-1111-4111-8111-111111111111"
	dsID    = "441f62c9-aad1-481d-9aab-62943fa11cd3"
)

// countingProvider records GetMap calls so tests can prove the store was (or
// was not) consulted.
type countingProvider struct {
	*secrets.MemoryProvider
	gets int
}

func (c *countingProvider) GetMap(ctx context.Context, key string) (map[string]string, error) {
	c.gets++
	return c.MemoryProvider.GetMap(ctx, key)
}

func newStore(t *testing.T) *countingProvider {
	t.Helper()
	p := &countingProvider{MemoryProvider: secrets.NewMemoryProvider()}
	path, _ := CanonicalPath(KindDatasource, tenantA, dsID)
	if err := p.PutMap(context.Background(), path, map[string]string{
		KeyUsername:   "store-user",
		KeyPassword:   "store-pass",
		KeyPrivateKey: "-----BEGIN PRIVATE KEY-----store-----END PRIVATE KEY-----",
	}); err != nil {
		t.Fatal(err)
	}
	return p
}

func decode(t *testing.T, b []byte) map[string]any {
	t.Helper()
	m := map[string]any{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("decode %s: %v", b, err)
	}
	return m
}

func basicOf(m map[string]any) map[string]any {
	auth, _ := m["auth"].(map[string]any)
	basic, _ := auth["basic"].(map[string]any)
	return basic
}

func TestCanonicalPath(t *testing.T) {
	got, err := CanonicalPath(KindDatasource, tenantA, strings.ToUpper(dsID))
	if err != nil || got != "/datasources/"+tenantA+"/"+dsID {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, bad := range []struct {
		kind       Kind
		tenant, id string
	}{
		{KindDatasource, "../" + tenantA, dsID},
		{KindDatasource, tenantA, dsID + "/../x"},
		{KindDatasource, "", dsID},
		{Kind("tenants"), tenantA, dsID},
	} {
		if _, err := CanonicalPath(bad.kind, bad.tenant, bad.id); err == nil {
			t.Errorf("CanonicalPath(%q,%q,%q) accepted", bad.kind, bad.tenant, bad.id)
		}
	}
}

func TestHydrate_NoRefPassesThrough(t *testing.T) {
	p := newStore(t)
	r := NewResolver(p)
	in := []byte(`{"host":"db","auth":{"basic":{"username":"u","password":"inline"}}}`)
	out, err := r.Hydrate(context.Background(), KindDatasource, tenantA, dsID, in)
	if err != nil || string(out) != string(in) {
		t.Fatalf("legacy config changed: %s, %v", out, err)
	}
	if p.gets != 0 {
		t.Fatal("store consulted for a config with no secret_path")
	}
}

func TestHydrate_StrictModeRefusesPlaintext(t *testing.T) {
	r := NewResolver(newStore(t), WithRequireRef(true))
	_, err := r.Hydrate(context.Background(), KindDatasource, tenantA, dsID,
		[]byte(`{"host":"db","auth":{"basic":{"username":"u","password":"inline"}}}`))
	if !errors.Is(err, ErrPlaintextCredentials) {
		t.Fatalf("want ErrPlaintextCredentials, got %v", err)
	}
	// A config with no credentials at all is fine in strict mode.
	if _, err := r.Hydrate(context.Background(), KindDatasource, tenantA, dsID, []byte(`{"host":"db"}`)); err != nil {
		t.Fatalf("credential-free config refused: %v", err)
	}
}

func TestHydrate_FromStoreOverridesStaleInline(t *testing.T) {
	r := NewResolver(newStore(t))
	in := []byte(`{"host":"db","port":5432,"ca_cert":"CA","secret_path":"/datasources/` + tenantA + `/` + dsID + `",
		"private_key":"stale-key","auth":{"basic":{"username":"old-user","password":"stale-pass"}}}`)
	out, err := r.Hydrate(context.Background(), KindDatasource, tenantA, dsID, in)
	if err != nil {
		t.Fatal(err)
	}
	m := decode(t, out)
	b := basicOf(m)
	if b["password"] != "store-pass" || b["username"] != "store-user" {
		t.Fatalf("basic auth not from store: %v", b)
	}
	if !strings.Contains(m["private_key"].(string), "store") {
		t.Fatal("private_key not from store")
	}
	if m["host"] != "db" || m["ca_cert"] != "CA" {
		t.Fatalf("non-secret fields lost: %v", m)
	}
}

func TestHydrate_KeepsDBUsernameWhenStoreHasNone(t *testing.T) {
	p := &countingProvider{MemoryProvider: secrets.NewMemoryProvider()}
	path, _ := CanonicalPath(KindDatasource, tenantA, dsID)
	_ = p.PutMap(context.Background(), path, map[string]string{KeyPassword: "pw"})
	out, err := NewResolver(p).Hydrate(context.Background(), KindDatasource, tenantA, dsID,
		[]byte(`{"secret_path":"`+path+`","auth":{"basic":{"username":"db-user"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if b := basicOf(decode(t, out)); b["username"] != "db-user" || b["password"] != "pw" {
		t.Fatalf("got %v", b)
	}
}

// A config pointing at another tenant's secret (e.g. copied from the gold copy,
// or edited) must be refused without ever reading that secret.
func TestHydrate_RefusesOtherTenantsSecret(t *testing.T) {
	p := newStore(t)
	r := NewResolver(p)
	ref := "/datasources/" + tenantA + "/" + dsID
	_, err := r.Hydrate(context.Background(), KindDatasource, tenantB, dsID, []byte(`{"secret_path":"`+ref+`"}`))
	if !errors.Is(err, ErrRefMismatch) {
		t.Fatalf("want ErrRefMismatch, got %v", err)
	}
	// Same tenant, different datasource id.
	_, err = r.Hydrate(context.Background(), KindDatasource, tenantA, "c2c00000-0000-4000-9000-0000000000c3", []byte(`{"secret_path":"`+ref+`"}`))
	if !errors.Is(err, ErrRefMismatch) {
		t.Fatalf("want ErrRefMismatch for another datasource, got %v", err)
	}
	// Right ids, wrong kind.
	_, err = r.Hydrate(context.Background(), KindConnection, tenantA, dsID, []byte(`{"secret_path":"`+ref+`"}`))
	if !errors.Is(err, ErrRefMismatch) {
		t.Fatalf("want ErrRefMismatch for another kind, got %v", err)
	}
	if p.gets != 0 {
		t.Fatal("store consulted for a mismatched reference")
	}
}

func TestHydrate_FailsClosed(t *testing.T) {
	ref := "/datasources/" + tenantA + "/" + dsID
	cfg := []byte(`{"secret_path":"` + ref + `","auth":{"basic":{"password":"inline"}}}`)

	// No provider: no fallback to the inline password.
	if _, err := NewResolver(nil).Hydrate(context.Background(), KindDatasource, tenantA, dsID, cfg); !errors.Is(err, ErrNoProvider) {
		t.Fatalf("want ErrNoProvider, got %v", err)
	}
	// Folder missing.
	if _, err := NewResolver(secrets.NewMemoryProvider()).Hydrate(context.Background(), KindDatasource, tenantA, dsID, cfg); !errors.Is(err, secrets.ErrSecretNotFound) {
		t.Fatalf("want ErrSecretNotFound, got %v", err)
	}
	// Folder present but holds no credential.
	p := secrets.NewMemoryProvider()
	_ = p.PutMap(context.Background(), ref, map[string]string{KeyUsername: "u"})
	if _, err := NewResolver(p).Hydrate(context.Background(), KindDatasource, tenantA, dsID, cfg); !errors.Is(err, secrets.ErrSecretNotFound) {
		t.Fatalf("want ErrSecretNotFound for empty folder, got %v", err)
	}
}

func TestHydrate_ErrorsNeverContainSecretValues(t *testing.T) {
	ref := "/datasources/" + tenantA + "/" + dsID
	cfg := []byte(`{"secret_path":"` + ref + `","auth":{"basic":{"password":"inline-hunter2"}}}`)
	_, err := NewResolver(nil).Hydrate(context.Background(), KindDatasource, tenantB, dsID, cfg)
	if err == nil || strings.Contains(err.Error(), "hunter2") {
		t.Fatalf("error leaks or missing: %v", err)
	}
}

func TestHydrate_CachesAndInvalidates(t *testing.T) {
	p := newStore(t)
	r := NewResolver(p)
	ref := "/datasources/" + tenantA + "/" + dsID
	cfg := []byte(`{"secret_path":"` + ref + `"}`)
	for i := 0; i < 3; i++ {
		if _, err := r.Hydrate(context.Background(), KindDatasource, tenantA, dsID, cfg); err != nil {
			t.Fatal(err)
		}
	}
	if p.gets != 1 {
		t.Fatalf("want 1 store read, got %d", p.gets)
	}
	r.Invalidate(ref)
	_, _ = r.Hydrate(context.Background(), KindDatasource, tenantA, dsID, cfg)
	if p.gets != 2 {
		t.Fatalf("want re-read after Invalidate, got %d", p.gets)
	}
	nc := NewResolver(p, WithCacheTTL(0))
	_, _ = nc.Hydrate(context.Background(), KindDatasource, tenantA, dsID, cfg)
	_, _ = nc.Hydrate(context.Background(), KindDatasource, tenantA, dsID, cfg)
	if p.gets != 4 {
		t.Fatalf("TTL 0 must not cache, got %d reads", p.gets)
	}
}

func TestStripForClone(t *testing.T) {
	in := []byte(`{"host":"db","port":5432,"ca_cert":"CA","client_cert":"CERT","private_key":"KEY","api_key":"AK",
		"password":"p","dsn":"postgres://u:p@h/d","secret_path":"/datasources/x/y",
		"auth":{"basic":{"username":"u","password":"p"}}}`)
	m := decode(t, StripForClone(in))
	for _, k := range []string{"private_key", "api_key", "password", "dsn", "secret_path"} {
		if _, ok := m[k]; ok {
			t.Errorf("%s survived clone", k)
		}
	}
	if _, ok := basicOf(m)["password"]; ok {
		t.Error("auth.basic.password survived clone")
	}
	if m["host"] != "db" || m["ca_cert"] != "CA" || basicOf(m)["username"] != "u" {
		t.Errorf("non-secret fields lost: %v", m)
	}
	if HasInlineSecrets(m) {
		t.Error("HasInlineSecrets true after strip")
	}
	if string(StripForClone([]byte("not json"))) != "{}" {
		t.Error("invalid JSON must clone as {}")
	}
}

func TestExtractInline(t *testing.T) {
	got, err := ExtractInline(map[string]any{
		"username":    "u",
		"private_key": "K",
		"auth":        map[string]any{"basic": map[string]any{"username": "u", "password": "p"}},
	})
	if err != nil || got[KeyUsername] != "u" || got[KeyPassword] != "p" || got[KeyPrivateKey] != "K" || len(got) != 3 {
		t.Fatalf("got %v, %v", got, err)
	}
	_, err = ExtractInline(map[string]any{
		"password": "flat-secret",
		"auth":     map[string]any{"basic": map[string]any{"password": "nested-secret"}},
	})
	if !errors.Is(err, ErrConflictingCredentials) || strings.Contains(err.Error(), "secret\"") || strings.Contains(err.Error(), "flat-secret") {
		t.Fatalf("want redacted conflict error, got %v", err)
	}
}

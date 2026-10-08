package tenantsecrets

import (
	"context"
	"errors"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

const (
	testUser     = "tenant_acme_app"
	testPassword = "p@ss:w/rd?#%&"
)

type fakeStore struct {
	stored   map[string]map[string]string
	getErr   error
	putErr   error
	puts     int
	putPath  string
	putValue map[string]string
}

func (f *fakeStore) GetMap(_ context.Context, key string) (map[string]string, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.stored[key], nil
}

func (f *fakeStore) PutMap(_ context.Context, key string, values map[string]string) error {
	f.puts++
	f.putPath = key
	f.putValue = values
	return f.putErr
}

func storeWithCredentials(code, env string) *fakeStore {
	path, _ := CredentialPath(code, env)
	return &fakeStore{stored: map[string]map[string]string{
		path: {CredentialKeyUsername: testUser, CredentialKeyPassword: testPassword},
	}}
}

func validInput() Input {
	return Input{
		TenantCode:   "acme",
		Environment:  "dev",
		DatabaseName: "tenant_acme",
		DatabaseHost: "pg.us-east-1.internal",
		DatabasePort: 5432,
	}
}

func TestSeedBuildsURLFromStoredCredentials(t *testing.T) {
	st := storeWithCredentials("acme", "dev")
	res, err := (&Activities{Secrets: st}).SeedInstanceSecrets(context.Background(), validInput())
	if err != nil {
		t.Fatalf("SeedInstanceSecrets: %v", err)
	}
	if st.putPath != "tenants/acme/dev" || res.Path != "tenants/acme/dev" {
		t.Fatalf("path = %q / %q", st.putPath, res.Path)
	}
	if st.putValue[DatabaseNameKey] != "tenant_acme" {
		t.Fatalf("database name = %q", st.putValue[DatabaseNameKey])
	}
	// Round-trip the URL: the password with reserved characters must survive intact.
	u, err := url.Parse(st.putValue[URLKey])
	if err != nil {
		t.Fatalf("written URL does not parse: %v", err)
	}
	pw, _ := u.User.Password()
	if u.Scheme != "postgres" || u.User.Username() != testUser || pw != testPassword {
		t.Fatalf("URL round trip lost data: scheme=%q user=%q", u.Scheme, u.User.Username())
	}
	if u.Host != "pg.us-east-1.internal:5432" || u.Path != "/tenant_acme" || u.Query().Get("sslmode") != "require" {
		t.Fatalf("URL = %q", st.putValue[URLKey])
	}
	if !reflect.DeepEqual(res.Keys, []string{URLKey, DatabaseNameKey}) {
		t.Fatalf("keys = %v", res.Keys)
	}
}

func TestSeedReadsCredentialPathNotCallerInput(t *testing.T) {
	st := storeWithCredentials("acme", "uat")
	in := validInput()
	in.Environment = "uat"
	if _, err := (&Activities{Secrets: st}).SeedInstanceSecrets(context.Background(), in); err != nil {
		t.Fatalf("SeedInstanceSecrets: %v", err)
	}
	if st.putPath != "tenants/acme/uat" {
		t.Fatalf("wrote to %q", st.putPath)
	}
}

func TestSeedRejectsBadInputBeforeAnyWrite(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Input)
	}{
		{"tenant code injection", func(i *Input) { i.TenantCode = "x; DROP DATABASE alpha;--" }},
		{"tenant code uppercase", func(i *Input) { i.TenantCode = "Acme" }},
		{"tenant code empty", func(i *Input) { i.TenantCode = "" }},
		{"tenant code dotdot", func(i *Input) { i.TenantCode = ".." }},
		{"tenant code slash", func(i *Input) { i.TenantCode = "acme/prod" }},
		{"environment empty", func(i *Input) { i.Environment = "" }},
		{"environment traversal", func(i *Input) { i.Environment = "../prod" }},
		{"environment unknown", func(i *Input) { i.Environment = "staging" }},
		{"environment case", func(i *Input) { i.Environment = "DEV" }},
		{"environment absolute", func(i *Input) { i.Environment = "/dev" }},
		{"environment nested", func(i *Input) { i.Environment = "dev/../prod" }},
		{"environment null byte", func(i *Input) { i.Environment = "dev\x00" }},
		{"environment overlong", func(i *Input) { i.Environment = strings.Repeat("d", 200) }},
		{"database name quote", func(i *Input) { i.DatabaseName = `tenant"x` }},
		{"database name semicolon", func(i *Input) { i.DatabaseName = "a;drop" }},
		{"database name unicode", func(i *Input) { i.DatabaseName = "tenant_é" }},
		{"database name overlong", func(i *Input) { i.DatabaseName = "d" + strings.Repeat("x", 63) }},
		{"host semicolon", func(i *Input) { i.DatabaseHost = "pg;evil" }},
		{"host empty", func(i *Input) { i.DatabaseHost = "" }},
		{"host with userinfo", func(i *Input) { i.DatabaseHost = "u:p@evil" }},
		{"port zero", func(i *Input) { i.DatabasePort = 0 }},
		{"port too large", func(i *Input) { i.DatabasePort = 70000 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := storeWithCredentials("acme", "dev")
			in := validInput()
			tc.mutate(&in)
			if _, err := (&Activities{Secrets: st}).SeedInstanceSecrets(context.Background(), in); err == nil {
				t.Fatal("accepted invalid input")
			}
			if st.puts != 0 {
				t.Fatal("store written for invalid input")
			}
		})
	}
}

func TestSeedRejectsBadStoredCredentials(t *testing.T) {
	tests := []struct {
		name  string
		creds map[string]string
	}{
		{"missing user", map[string]string{CredentialKeyPassword: "x"}},
		{"injected user", map[string]string{CredentialKeyUsername: "a;drop", CredentialKeyPassword: "x"}},
		{"missing password", map[string]string{CredentialKeyUsername: testUser}},
		{"overlong password", map[string]string{CredentialKeyUsername: testUser, CredentialKeyPassword: strings.Repeat("p", maxPasswordLen+1)}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path, _ := CredentialPath("acme", "dev")
			st := &fakeStore{stored: map[string]map[string]string{path: tc.creds}}
			if _, err := (&Activities{Secrets: st}).SeedInstanceSecrets(context.Background(), validInput()); err == nil {
				t.Fatal("accepted bad stored credentials")
			}
			if st.puts != 0 {
				t.Fatal("store written with bad credentials")
			}
		})
	}
}

func TestSeedErrorsNeverEchoSecrets(t *testing.T) {
	st := &fakeStore{getErr: errors.New("store echoed " + testPassword + " at tenants/acme/dev/db_credentials")}
	_, err := (&Activities{Secrets: st}).SeedInstanceSecrets(context.Background(), validInput())
	if err == nil {
		t.Fatal("store read failure not reported")
	}
	if strings.Contains(err.Error(), testPassword) || strings.Contains(err.Error(), "db_credentials") {
		t.Fatalf("store read failure leaked detail: %v", err)
	}

	st = storeWithCredentials("acme", "dev")
	st.putErr = errors.New("store echoed " + testPassword)
	_, err = (&Activities{Secrets: st}).SeedInstanceSecrets(context.Background(), validInput())
	if err == nil || strings.Contains(err.Error(), testPassword) {
		t.Fatalf("store write failure leaked detail: %v", err)
	}
}

func TestSeedWithoutStoreFailsClosed(t *testing.T) {
	if _, err := (&Activities{}).SeedInstanceSecrets(context.Background(), validInput()); err == nil {
		t.Fatal("ran without a secret store")
	}
}

func TestPathRejectsTraversalAndAcceptsLayout(t *testing.T) {
	bad := []struct{ code, env string }{
		{"..", "dev"}, {"acme", ".."}, {"acme", "/dev"}, {"acme", "dev/x"},
		{"acme/x", "dev"}, {"/acme", "dev"}, {"", "dev"}, {"acme", ""},
		{"acme", strings.Repeat("d", 200)}, {"a" + strings.Repeat("b", 60), "dev"},
	}
	for _, tc := range bad {
		if p, err := Path(tc.code, tc.env); err == nil {
			t.Errorf("Path(%q, %q) = %q, want error", tc.code, tc.env, p)
		}
		if p, err := CredentialPath(tc.code, tc.env); err == nil {
			t.Errorf("CredentialPath(%q, %q) = %q, want error", tc.code, tc.env, p)
		}
	}
	for _, env := range []string{"dev", "uat", "prod"} {
		p, err := Path("acme", env)
		if err != nil || p != "tenants/acme/"+env {
			t.Errorf("Path(acme, %s) = %q, %v", env, p, err)
		}
	}
}

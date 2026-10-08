package tenantsecrets

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeStore struct {
	calls  int
	path   string
	values map[string]string
	err    error
}

func (f *fakeStore) PutMap(_ context.Context, path string, values map[string]string) error {
	f.calls++
	f.path = path
	f.values = values
	return f.err
}

const goodURL = "postgres://app_user:s3cret-pass@db.internal:5432/tenant_acme?sslmode=require"

func validInput() Input {
	return Input{
		TenantCode:   "acme",
		Environment:  "dev",
		DatabaseName: "tenant_acme",
		DatabaseURL:  goodURL,
	}
}

func TestSeedInstanceSecretsWritesLayoutPath(t *testing.T) {
	st := &fakeStore{}
	res, err := (&Activities{Secrets: st}).SeedInstanceSecrets(context.Background(), validInput())
	if err != nil {
		t.Fatalf("SeedInstanceSecrets: %v", err)
	}
	if st.path != "tenants/acme/dev" || res.Path != "tenants/acme/dev" {
		t.Fatalf("path = %q / %q, want tenants/acme/dev", st.path, res.Path)
	}
	if st.values["database_url"] != goodURL || st.values["database_name"] != "tenant_acme" {
		t.Fatalf("unexpected values: %v", st.values)
	}
	if len(res.Keys) != 2 {
		t.Fatalf("keys = %v", res.Keys)
	}
}

func TestSeedInstanceSecretsEnvironmentsFollowLayout(t *testing.T) {
	for _, env := range []string{"dev", "uat", "prod"} {
		st := &fakeStore{}
		in := validInput()
		in.Environment = env
		res, err := (&Activities{Secrets: st}).SeedInstanceSecrets(context.Background(), in)
		if err != nil {
			t.Fatalf("env %s: %v", env, err)
		}
		if res.Path != "tenants/acme/"+env {
			t.Fatalf("env %s: path %q", env, res.Path)
		}
	}
}

func TestSeedInstanceSecretsRejectsBadInputBeforeStore(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Input)
	}{
		{"tenant code injection", func(i *Input) { i.TenantCode = "x; DROP DATABASE alpha;--" }},
		{"tenant code uppercase", func(i *Input) { i.TenantCode = "Acme" }},
		{"tenant code empty", func(i *Input) { i.TenantCode = "" }},
		{"environment empty", func(i *Input) { i.Environment = "" }},
		{"environment traversal", func(i *Input) { i.Environment = "../prod" }},
		{"environment unknown", func(i *Input) { i.Environment = "staging" }},
		{"environment case", func(i *Input) { i.Environment = "DEV" }},
		{"database name quote", func(i *Input) { i.DatabaseName = `tenant"x` }},
		{"database name semicolon", func(i *Input) { i.DatabaseName = "a;drop" }},
		{"database name unicode", func(i *Input) { i.DatabaseName = "tenant_é" }},
		{"database name overlong", func(i *Input) { i.DatabaseName = "d" + strings.Repeat("x", 63) }},
		{"database url empty", func(i *Input) { i.DatabaseURL = "" }},
		{"database url wrong scheme", func(i *Input) { i.DatabaseURL = "mysql://u:p@h/db" }},
		{"database url no host", func(i *Input) { i.DatabaseURL = "postgres:///db" }},
		{"database url malformed", func(i *Input) { i.DatabaseURL = "postgres://u:p@h\n/db" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := &fakeStore{}
			in := validInput()
			tc.mutate(&in)
			if _, err := (&Activities{Secrets: st}).SeedInstanceSecrets(context.Background(), in); err == nil {
				t.Fatal("accepted invalid input")
			}
			if st.calls != 0 {
				t.Fatal("store written for invalid input")
			}
		})
	}
}

func TestSeedInstanceSecretsErrorsNeverEchoSecrets(t *testing.T) {
	// Malformed URL: the parse error would quote the password.
	in := validInput()
	in.DatabaseURL = "postgres://app_user:s3cret-pass@h\n/db"
	_, err := (&Activities{Secrets: &fakeStore{}}).SeedInstanceSecrets(context.Background(), in)
	if err == nil || strings.Contains(err.Error(), "s3cret-pass") {
		t.Fatalf("error %v leaks or is missing", err)
	}

	// Store failure: the store error may echo the request, so it is not surfaced.
	st := &fakeStore{err: errors.New("store echoed " + goodURL)}
	_, err = (&Activities{Secrets: st}).SeedInstanceSecrets(context.Background(), validInput())
	if err == nil {
		t.Fatal("store failure not reported")
	}
	if strings.Contains(err.Error(), "s3cret-pass") {
		t.Fatalf("store failure leaked the URL: %v", err)
	}
}

func TestSeedInstanceSecretsWithoutStoreFailsClosed(t *testing.T) {
	if _, err := (&Activities{}).SeedInstanceSecrets(context.Background(), validInput()); err == nil {
		t.Fatal("ran without a secret store")
	}
}

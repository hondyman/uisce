package tenantidentity

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeKeycloak struct {
	calls          []string
	realmExists    bool
	failClient     error
	failLDAP       error
	clientSecret   string
	createdRealms  []string
	deletedRealms  []string
	ldapRealms     []string
	ldapConfigs    []LDAPConfig
	createRealmErr error
}

func (f *fakeKeycloak) CreateRealm(_ context.Context, realm string) error {
	f.calls = append(f.calls, "CreateRealm")
	if f.createRealmErr != nil {
		return f.createRealmErr
	}
	if f.realmExists {
		return ErrRealmExists
	}
	f.createdRealms = append(f.createdRealms, realm)
	return nil
}

func (f *fakeKeycloak) DeleteRealm(_ context.Context, realm string) error {
	f.calls = append(f.calls, "DeleteRealm")
	f.deletedRealms = append(f.deletedRealms, realm)
	return nil
}

func (f *fakeKeycloak) CreatePlatformClient(_ context.Context, realm string) (string, error) {
	f.calls = append(f.calls, "CreatePlatformClient")
	if f.failClient != nil {
		return "", f.failClient
	}
	return f.clientSecret, nil
}

func (f *fakeKeycloak) AddLDAPFederation(_ context.Context, realm string, ldap LDAPConfig) error {
	f.calls = append(f.calls, "AddLDAPFederation")
	if f.failLDAP != nil {
		return f.failLDAP
	}
	f.ldapRealms = append(f.ldapRealms, realm)
	f.ldapConfigs = append(f.ldapConfigs, ldap)
	return nil
}

type fakeSecrets struct {
	calls  int
	key    string
	values map[string]string
	err    error
}

func (f *fakeSecrets) PutMap(_ context.Context, key string, values map[string]string) error {
	f.calls++
	f.key = key
	f.values = values
	return f.err
}

type fakeIssuers struct {
	calls  int
	code   string
	issuer string
	err    error
}

func (f *fakeIssuers) SetIssuer(_ context.Context, code, issuer string) error {
	f.calls++
	f.code = code
	f.issuer = issuer
	return f.err
}

const testBase = "https://keycloak.example.internal"

func newActivities(kc *fakeKeycloak, sec *fakeSecrets, iss *fakeIssuers) *Activities {
	return &Activities{Keycloak: kc, Secrets: sec, Issuers: iss, PublicBaseURL: testBase}
}

func validLDAP() *LDAPConfig {
	return &LDAPConfig{
		Host:         "ldap.corp.example.internal",
		Port:         636,
		BaseDN:       "dc=corp,dc=example",
		BindDN:       "cn=svc-uisce,ou=services,dc=corp,dc=example",
		BindPassword: "bind-secret-value",
	}
}

func TestConfigureTenantIdentityLocalUsers(t *testing.T) {
	kc := &fakeKeycloak{clientSecret: "client-secret-value"}
	sec := &fakeSecrets{}
	iss := &fakeIssuers{}
	res, err := newActivities(kc, sec, iss).ConfigureTenantIdentity(context.Background(), Input{TenantCode: "acme"})
	if err != nil {
		t.Fatalf("ConfigureTenantIdentity: %v", err)
	}
	if !res.RealmCreated || !res.ClientCreated || !res.SecretsWritten {
		t.Fatalf("result flags = %+v", res)
	}
	if res.Issuer != testBase+"/realms/acme" || iss.issuer != res.Issuer || iss.code != "acme" {
		t.Fatalf("issuer = %q, recorded %q for %q", res.Issuer, iss.issuer, iss.code)
	}
	if sec.key != "tenants/acme/identity" {
		t.Fatalf("secret path = %q", sec.key)
	}
	if _, has := sec.values["ldap_bind_password"]; has {
		t.Fatal("bind password stored without LDAP")
	}
	if sec.values["client_secret"] != "client-secret-value" {
		t.Fatalf("client secret not stored: %v", sec.values)
	}
	if len(kc.ldapConfigs) != 0 {
		t.Fatal("LDAP federation added without LDAP input")
	}
}

func TestConfigureTenantIdentityWithLDAP(t *testing.T) {
	kc := &fakeKeycloak{clientSecret: "client-secret-value"}
	sec := &fakeSecrets{}
	iss := &fakeIssuers{}
	if _, err := newActivities(kc, sec, iss).ConfigureTenantIdentity(context.Background(), Input{TenantCode: "acme", LDAP: validLDAP()}); err != nil {
		t.Fatalf("ConfigureTenantIdentity: %v", err)
	}
	if len(kc.ldapConfigs) != 1 || kc.ldapConfigs[0].Host != "ldap.corp.example.internal" {
		t.Fatalf("LDAP federation not added: %+v", kc.ldapConfigs)
	}
	if sec.values["ldap_bind_password"] != "bind-secret-value" {
		t.Fatal("LDAP bind password not stored in secrets")
	}
}

func TestConfigureTenantIdentityRejectsBadInputBeforeKeycloak(t *testing.T) {
	tests := []struct {
		name string
		in   Input
	}{
		{"empty code", Input{TenantCode: ""}},
		{"semicolon injection", Input{TenantCode: "x; DROP DATABASE alpha;--"}},
		{"quote", Input{TenantCode: `x"y`}},
		{"space", Input{TenantCode: "ac me"}},
		{"uppercase", Input{TenantCode: "Acme"}},
		{"leading digit", Input{TenantCode: "1acme"}},
		{"unicode", Input{TenantCode: "acmé"}},
		{"overlong", Input{TenantCode: "a" + strings.Repeat("b", 60)}},
		{"ldap host semicolon", Input{TenantCode: "acme", LDAP: withHost(validLDAP(), "ldap;evil")}},
		{"ldap host empty", Input{TenantCode: "acme", LDAP: withHost(validLDAP(), "")}},
		{"ldap host overlong", Input{TenantCode: "acme", LDAP: withHost(validLDAP(), strings.Repeat("a", 300))}},
		{"ldap port zero", Input{TenantCode: "acme", LDAP: withPort(validLDAP(), 0)}},
		{"ldap port too large", Input{TenantCode: "acme", LDAP: withPort(validLDAP(), 70000)}},
		{"ldap base dn newline", Input{TenantCode: "acme", LDAP: withBaseDN(validLDAP(), "dc=a\nb")}},
		{"ldap bind dn quote", Input{TenantCode: "acme", LDAP: withBindDN(validLDAP(), `cn="x"`)}},
		{"ldap empty password", Input{TenantCode: "acme", LDAP: withPassword(validLDAP(), "")}},
		{"ldap overlong password", Input{TenantCode: "acme", LDAP: withPassword(validLDAP(), strings.Repeat("p", 257))}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kc := &fakeKeycloak{}
			sec := &fakeSecrets{}
			iss := &fakeIssuers{}
			_, err := newActivities(kc, sec, iss).ConfigureTenantIdentity(context.Background(), tc.in)
			if err == nil {
				t.Fatal("accepted invalid input")
			}
			if len(kc.calls) != 0 {
				t.Fatalf("Keycloak called for invalid input: %v", kc.calls)
			}
		})
	}
}

func TestConfigureTenantIdentityRejectsBadBaseURL(t *testing.T) {
	for _, base := range []string{"", "ftp://kc.example.internal", "javascript:alert(1)", "https://"} {
		kc := &fakeKeycloak{}
		a := &Activities{Keycloak: kc, Secrets: &fakeSecrets{}, Issuers: &fakeIssuers{}, PublicBaseURL: base}
		if _, err := a.ConfigureTenantIdentity(context.Background(), Input{TenantCode: "acme"}); err == nil {
			t.Errorf("accepted base URL %q", base)
		}
		if len(kc.calls) != 0 {
			t.Errorf("Keycloak called with base URL %q", base)
		}
	}
}

func TestConfigureTenantIdentityRealmAlreadyExistsIsNotOwned(t *testing.T) {
	kc := &fakeKeycloak{realmExists: true}
	res, err := newActivities(kc, &fakeSecrets{}, &fakeIssuers{}).ConfigureTenantIdentity(context.Background(), Input{TenantCode: "acme"})
	if err == nil {
		t.Fatal("expected realm-exists error")
	}
	if res.RealmCreated {
		t.Fatal("pre-existing realm reported as created")
	}
	if len(kc.calls) != 1 {
		t.Fatalf("expected only CreateRealm, got %v", kc.calls)
	}
}

func TestConfigureTenantIdentityPartialFailureReportsCreatedRealm(t *testing.T) {
	kc := &fakeKeycloak{failClient: errors.New("keycloak 503")}
	res, err := newActivities(kc, &fakeSecrets{}, &fakeIssuers{}).ConfigureTenantIdentity(context.Background(), Input{TenantCode: "acme"})
	if err == nil {
		t.Fatal("expected client error")
	}
	if !res.RealmCreated || res.ClientCreated {
		t.Fatalf("partial result = %+v, want realm created and client not", res)
	}
}

func TestConfigureTenantIdentitySecretsFailureReportsCreatedClient(t *testing.T) {
	kc := &fakeKeycloak{clientSecret: "client-secret-value"}
	sec := &fakeSecrets{err: errors.New("infisical unavailable")}
	iss := &fakeIssuers{}
	res, err := newActivities(kc, sec, iss).ConfigureTenantIdentity(context.Background(), Input{TenantCode: "acme"})
	if err == nil {
		t.Fatal("expected secrets error")
	}
	if !res.RealmCreated || !res.ClientCreated || res.SecretsWritten {
		t.Fatalf("partial result = %+v", res)
	}
	if iss.calls != 0 {
		t.Fatal("issuer recorded after a failed secrets write")
	}
}

func TestErrorsNeverContainSecrets(t *testing.T) {
	kc := &fakeKeycloak{failLDAP: errors.New("ldap refused")}
	_, err := newActivities(kc, &fakeSecrets{}, &fakeIssuers{}).ConfigureTenantIdentity(context.Background(), Input{TenantCode: "acme", LDAP: validLDAP()})
	if err == nil {
		t.Fatal("expected LDAP error")
	}
	if strings.Contains(err.Error(), "bind-secret-value") {
		t.Fatal("bind password leaked into error text")
	}
}

func TestConfigureTenantIdentityWithoutDependenciesFailsClosed(t *testing.T) {
	a := &Activities{PublicBaseURL: testBase}
	if _, err := a.ConfigureTenantIdentity(context.Background(), Input{TenantCode: "acme"}); err == nil {
		t.Fatal("ran without dependencies")
	}
}

func TestRollbackDeletesOnlyWhatIsPassedAndValidatesCode(t *testing.T) {
	kc := &fakeKeycloak{}
	a := &Activities{Keycloak: kc, Secrets: &fakeSecrets{}, Issuers: &fakeIssuers{}, PublicBaseURL: testBase}
	if err := a.RollbackConfigureTenantIdentity(context.Background(), "acme"); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if len(kc.deletedRealms) != 1 || kc.deletedRealms[0] != "acme" {
		t.Fatalf("deleted = %v", kc.deletedRealms)
	}
	if err := a.RollbackConfigureTenantIdentity(context.Background(), "x; drop"); err == nil {
		t.Fatal("rollback accepted a hostile code")
	}
}

func withHost(c *LDAPConfig, v string) *LDAPConfig     { c.Host = v; return c }
func withPort(c *LDAPConfig, v int) *LDAPConfig        { c.Port = v; return c }
func withBaseDN(c *LDAPConfig, v string) *LDAPConfig   { c.BaseDN = v; return c }
func withBindDN(c *LDAPConfig, v string) *LDAPConfig   { c.BindDN = v; return c }
func withPassword(c *LDAPConfig, v string) *LDAPConfig { c.BindPassword = v; return c }

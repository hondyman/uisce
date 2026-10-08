package tenantidentity

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type fakeKeycloak struct {
	calls         []string
	realmExists   bool
	failClient    error
	failLDAP      error
	clientSecret  string
	deletedRealms []string
	ldapPasswords []string
	ldapConfigs   []LDAPConfig
}

func (f *fakeKeycloak) CreateRealm(_ context.Context, realm string) error {
	f.calls = append(f.calls, "CreateRealm")
	if f.realmExists {
		return ErrRealmExists
	}
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

func (f *fakeKeycloak) AddLDAPFederation(_ context.Context, realm string, ldap LDAPConfig, bindPassword string) error {
	f.calls = append(f.calls, "AddLDAPFederation")
	if f.failLDAP != nil {
		return f.failLDAP
	}
	f.ldapConfigs = append(f.ldapConfigs, ldap)
	f.ldapPasswords = append(f.ldapPasswords, bindPassword)
	return nil
}

type fakeSecrets struct {
	stored  map[string]map[string]string
	putKeys []string
	putPath string
	putErr  error
	getErr  error
}

func (f *fakeSecrets) PutMap(_ context.Context, key string, values map[string]string) error {
	f.putPath = key
	for k := range values {
		f.putKeys = append(f.putKeys, k)
	}
	if f.stored == nil {
		f.stored = map[string]map[string]string{}
	}
	if f.stored[key] == nil {
		f.stored[key] = map[string]string{}
	}
	for k, v := range values {
		f.stored[key][k] = v
	}
	return f.putErr
}

func (f *fakeSecrets) GetMap(_ context.Context, key string) (map[string]string, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.stored[key], nil
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

const (
	testBase         = "https://keycloak.example.internal"
	testBindPassword = "bind-secret-value"
	testClientSecret = "client-secret-value"
)

func newActivities(kc *fakeKeycloak, sec *fakeSecrets, iss *fakeIssuers) *Activities {
	return &Activities{Keycloak: kc, Secrets: sec, Issuers: iss, PublicBaseURL: testBase}
}

// seededSecrets returns a store that already holds the LDAP bind password at the
// identity path. The wizard handler writes it there before the workflow starts.
func seededSecrets(t *testing.T, code string) *fakeSecrets {
	t.Helper()
	path, err := IdentityPath(code)
	if err != nil {
		t.Fatalf("IdentityPath: %v", err)
	}
	return &fakeSecrets{stored: map[string]map[string]string{
		path: {LDAPBindPasswordKey: testBindPassword},
	}}
}

func validLDAP() *LDAPConfig {
	return &LDAPConfig{
		Host:   "ldap.corp.example.internal",
		Port:   636,
		BaseDN: "dc=corp,dc=example",
		BindDN: "cn=svc-uisce,ou=services,dc=corp,dc=example",
	}
}

func TestConfigureTenantIdentityLocalUsers(t *testing.T) {
	kc := &fakeKeycloak{clientSecret: testClientSecret}
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
	if sec.putPath != "tenants/acme/identity" {
		t.Fatalf("secret path = %q", sec.putPath)
	}
	if !reflect.DeepEqual(sec.putKeys, []string{ClientSecretKey}) {
		t.Fatalf("keys written = %v, want only the client secret", sec.putKeys)
	}
	if len(kc.ldapConfigs) != 0 {
		t.Fatal("LDAP federation added without LDAP input")
	}
}

func TestConfigureTenantIdentityWithLDAPReadsBindPasswordFromStore(t *testing.T) {
	kc := &fakeKeycloak{clientSecret: testClientSecret}
	sec := seededSecrets(t, "acme")
	iss := &fakeIssuers{}
	if _, err := newActivities(kc, sec, iss).ConfigureTenantIdentity(context.Background(), Input{TenantCode: "acme", LDAP: validLDAP()}); err != nil {
		t.Fatalf("ConfigureTenantIdentity: %v", err)
	}
	if len(kc.ldapPasswords) != 1 || kc.ldapPasswords[0] != testBindPassword {
		t.Fatalf("bind password passed to Keycloak = %v", kc.ldapPasswords)
	}
	// The bind password was already in the store, so the activity must not
	// rewrite it. Only the client secret is written.
	if !reflect.DeepEqual(sec.putKeys, []string{ClientSecretKey}) {
		t.Fatalf("keys written = %v", sec.putKeys)
	}
}

func TestMissingBindPasswordFailsBeforeAnythingIsCreated(t *testing.T) {
	kc := &fakeKeycloak{clientSecret: testClientSecret}
	sec := &fakeSecrets{} // no bind password in the store
	iss := &fakeIssuers{}
	_, err := newActivities(kc, sec, iss).ConfigureTenantIdentity(context.Background(), Input{TenantCode: "acme", LDAP: validLDAP()})
	if err == nil {
		t.Fatal("ran LDAP setup without a bind password")
	}
	if len(kc.calls) != 0 {
		t.Fatalf("Keycloak called despite missing bind password: %v", kc.calls)
	}
}

func TestUnreadableStoreFailsBeforeAnythingIsCreated(t *testing.T) {
	kc := &fakeKeycloak{}
	sec := &fakeSecrets{getErr: errors.New("infisical 503 with detail")}
	_, err := newActivities(kc, sec, &fakeIssuers{}).ConfigureTenantIdentity(context.Background(), Input{TenantCode: "acme", LDAP: validLDAP()})
	if err == nil {
		t.Fatal("ran LDAP setup with an unreadable store")
	}
	if len(kc.calls) != 0 {
		t.Fatalf("Keycloak called with unreadable store: %v", kc.calls)
	}
	if strings.Contains(err.Error(), "detail") {
		t.Fatalf("store error detail leaked: %v", err)
	}
}

func TestBindPasswordOverlongIsRejected(t *testing.T) {
	kc := &fakeKeycloak{}
	sec := &fakeSecrets{stored: map[string]map[string]string{
		"tenants/acme/identity": {LDAPBindPasswordKey: strings.Repeat("p", maxBindPasswordLen+1)},
	}}
	if _, err := newActivities(kc, sec, &fakeIssuers{}).ConfigureTenantIdentity(context.Background(), Input{TenantCode: "acme", LDAP: validLDAP()}); err == nil {
		t.Fatal("accepted an overlong bind password")
	}
	if len(kc.calls) != 0 {
		t.Fatal("Keycloak called for an overlong bind password")
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
		{"dotdot path", Input{TenantCode: ".."}},
		{"slash path", Input{TenantCode: "acme/prod"}},
		{"overlong", Input{TenantCode: "a" + strings.Repeat("b", 60)}},
		{"ldap host semicolon", Input{TenantCode: "acme", LDAP: withHost(validLDAP(), "ldap;evil")}},
		{"ldap host empty", Input{TenantCode: "acme", LDAP: withHost(validLDAP(), "")}},
		{"ldap host overlong", Input{TenantCode: "acme", LDAP: withHost(validLDAP(), strings.Repeat("a", 300))}},
		{"ldap port zero", Input{TenantCode: "acme", LDAP: withPort(validLDAP(), 0)}},
		{"ldap port too large", Input{TenantCode: "acme", LDAP: withPort(validLDAP(), 70000)}},
		{"ldap base dn newline", Input{TenantCode: "acme", LDAP: withBaseDN(validLDAP(), "dc=a\nb")}},
		{"ldap bind dn quote", Input{TenantCode: "acme", LDAP: withBindDN(validLDAP(), `cn="x"`)}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kc := &fakeKeycloak{}
			sec := seededSecrets(t, "acme")
			if _, err := newActivities(kc, sec, &fakeIssuers{}).ConfigureTenantIdentity(context.Background(), tc.in); err == nil {
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
	kc := &fakeKeycloak{clientSecret: testClientSecret}
	sec := &fakeSecrets{putErr: errors.New("infisical unavailable")}
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

// The client secret is generated in Keycloak and written to the store. It must
// not appear in the result or in any error text.
func TestClientSecretNeverLeavesTheActivity(t *testing.T) {
	kc := &fakeKeycloak{clientSecret: testClientSecret}
	res, err := newActivities(kc, &fakeSecrets{}, &fakeIssuers{}).ConfigureTenantIdentity(context.Background(), Input{TenantCode: "acme"})
	if err != nil {
		t.Fatalf("ConfigureTenantIdentity: %v", err)
	}
	fields := []string{res.Realm, res.Issuer, res.SecretPath}
	for _, f := range fields {
		if strings.Contains(f, testClientSecret) {
			t.Fatalf("client secret present in result field %q", f)
		}
	}
}

func TestErrorsNeverContainBindPassword(t *testing.T) {
	kc := &fakeKeycloak{failLDAP: errors.New("ldap refused")}
	_, err := newActivities(kc, seededSecrets(t, "acme"), &fakeIssuers{}).ConfigureTenantIdentity(context.Background(), Input{TenantCode: "acme", LDAP: validLDAP()})
	if err == nil {
		t.Fatal("expected LDAP error")
	}
	if strings.Contains(err.Error(), testBindPassword) {
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

func withHost(c *LDAPConfig, v string) *LDAPConfig   { c.Host = v; return c }
func withPort(c *LDAPConfig, v int) *LDAPConfig      { c.Port = v; return c }
func withBaseDN(c *LDAPConfig, v string) *LDAPConfig { c.BaseDN = v; return c }
func withBindDN(c *LDAPConfig, v string) *LDAPConfig { c.BindDN = v; return c }

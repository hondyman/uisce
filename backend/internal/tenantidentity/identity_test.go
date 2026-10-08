package tenantidentity

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

const (
	testTenantID     = "11111111-1111-1111-1111-111111111111"
	testOtherTenant  = "22222222-2222-2222-2222-222222222222"
	testBase         = "https://keycloak.example.internal"
	testBindPassword = "bind-secret-value"
	testClientSecret = "client-secret-value"
)

type fakeKeycloak struct {
	calls         []string
	realms        map[string]string // realm -> owner tenant ID ("" = unowned)
	failClient    error
	failClientOne bool // fail the next CreatePlatformClient only
	failLDAP      error
	clientSecret  string
	deletedRealms []string
	ldapPasswords []string
	ldapConfigs   []LDAPConfig
}

func (f *fakeKeycloak) CreateRealm(_ context.Context, realm, tenantID string) error {
	f.calls = append(f.calls, "CreateRealm")
	if f.realms == nil {
		f.realms = map[string]string{}
	}
	if _, exists := f.realms[realm]; exists {
		return ErrRealmExists
	}
	f.realms[realm] = tenantID
	return nil
}

func (f *fakeKeycloak) RealmOwner(_ context.Context, realm string) (string, error) {
	f.calls = append(f.calls, "RealmOwner")
	return f.realms[realm], nil
}

func (f *fakeKeycloak) DeleteRealm(_ context.Context, realm string) error {
	f.calls = append(f.calls, "DeleteRealm")
	f.deletedRealms = append(f.deletedRealms, realm)
	return nil
}

func (f *fakeKeycloak) CreatePlatformClient(_ context.Context, realm string) (string, error) {
	f.calls = append(f.calls, "CreatePlatformClient")
	if f.failClientOne {
		f.failClientOne = false
		return "", errors.New("keycloak 503")
	}
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

func (f *fakeIssuers) SetIssuer(_ context.Context, tenantID, issuer string) error {
	f.calls++
	f.code = tenantID
	f.issuer = issuer
	return f.err
}

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
	res, err := newActivities(kc, sec, iss).ConfigureTenantIdentity(context.Background(), Input{TenantID: testTenantID, TenantCode: "acme"})
	if err != nil {
		t.Fatalf("ConfigureTenantIdentity: %v", err)
	}
	if !res.RealmCreated || !res.ClientCreated || !res.SecretsWritten {
		t.Fatalf("result flags = %+v", res)
	}
	if res.Issuer != testBase+"/realms/acme" || iss.issuer != res.Issuer || iss.code != testTenantID {
		t.Fatalf("issuer = %q, recorded %q for tenant %q", res.Issuer, iss.issuer, iss.code)
	}
	if kc.realms["acme"] != testTenantID {
		t.Fatalf("realm owner = %q, want the tenant ID", kc.realms["acme"])
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
	if _, err := newActivities(kc, sec, iss).ConfigureTenantIdentity(context.Background(), Input{TenantID: testTenantID, TenantCode: "acme", LDAP: validLDAP()}); err != nil {
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
	_, err := newActivities(kc, sec, &fakeIssuers{}).ConfigureTenantIdentity(context.Background(), Input{TenantID: testTenantID, TenantCode: "acme", LDAP: validLDAP()})
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
	_, err := newActivities(kc, sec, &fakeIssuers{}).ConfigureTenantIdentity(context.Background(), Input{TenantID: testTenantID, TenantCode: "acme", LDAP: validLDAP()})
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
	if _, err := newActivities(kc, sec, &fakeIssuers{}).ConfigureTenantIdentity(context.Background(), Input{TenantID: testTenantID, TenantCode: "acme", LDAP: validLDAP()}); err == nil {
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
		{"empty code", Input{TenantID: testTenantID, TenantCode: ""}},
		{"semicolon injection", Input{TenantID: testTenantID, TenantCode: "x; DROP DATABASE alpha;--"}},
		{"quote", Input{TenantID: testTenantID, TenantCode: `x"y`}},
		{"space", Input{TenantID: testTenantID, TenantCode: "ac me"}},
		{"uppercase", Input{TenantID: testTenantID, TenantCode: "Acme"}},
		{"leading digit", Input{TenantID: testTenantID, TenantCode: "1acme"}},
		{"unicode", Input{TenantID: testTenantID, TenantCode: "acmé"}},
		{"dotdot path", Input{TenantID: testTenantID, TenantCode: ".."}},
		{"slash path", Input{TenantID: testTenantID, TenantCode: "acme/prod"}},
		{"overlong", Input{TenantID: testTenantID, TenantCode: "a" + strings.Repeat("b", 60)}},
		{"ldap host semicolon", Input{TenantID: testTenantID, TenantCode: "acme", LDAP: withHost(validLDAP(), "ldap;evil")}},
		{"ldap host empty", Input{TenantID: testTenantID, TenantCode: "acme", LDAP: withHost(validLDAP(), "")}},
		{"ldap host overlong", Input{TenantID: testTenantID, TenantCode: "acme", LDAP: withHost(validLDAP(), strings.Repeat("a", 300))}},
		{"ldap port zero", Input{TenantID: testTenantID, TenantCode: "acme", LDAP: withPort(validLDAP(), 0)}},
		{"ldap port too large", Input{TenantID: testTenantID, TenantCode: "acme", LDAP: withPort(validLDAP(), 70000)}},
		{"ldap base dn newline", Input{TenantID: testTenantID, TenantCode: "acme", LDAP: withBaseDN(validLDAP(), "dc=a\nb")}},
		{"ldap bind dn quote", Input{TenantID: testTenantID, TenantCode: "acme", LDAP: withBindDN(validLDAP(), `cn="x"`)}},
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

func TestInvalidTenantIDRejectedBeforeKeycloak(t *testing.T) {
	for _, id := range []string{"", "not-a-uuid", testTenantID + "; drop"} {
		kc := &fakeKeycloak{}
		if _, err := newActivities(kc, &fakeSecrets{}, &fakeIssuers{}).ConfigureTenantIdentity(context.Background(), Input{TenantID: id, TenantCode: "acme"}); err == nil {
			t.Errorf("accepted tenant ID %q", id)
		}
		if len(kc.calls) != 0 {
			t.Errorf("Keycloak called for tenant ID %q", id)
		}
	}
}

func TestConfigureTenantIdentityRejectsBadBaseURL(t *testing.T) {
	for _, base := range []string{"", "ftp://kc.example.internal", "javascript:alert(1)", "https://"} {
		kc := &fakeKeycloak{}
		a := &Activities{Keycloak: kc, Secrets: &fakeSecrets{}, Issuers: &fakeIssuers{}, PublicBaseURL: base}
		if _, err := a.ConfigureTenantIdentity(context.Background(), Input{TenantID: testTenantID, TenantCode: "acme"}); err == nil {
			t.Errorf("accepted base URL %q", base)
		}
		if len(kc.calls) != 0 {
			t.Errorf("Keycloak called with base URL %q", base)
		}
	}
}

// A realm owned by another tenant is a conflict. This run must not adopt it,
// configure it, or delete it on rollback.
func TestForeignRealmIsConflictAndNotOwned(t *testing.T) {
	kc := &fakeKeycloak{realms: map[string]string{"acme": testOtherTenant}}
	res, err := newActivities(kc, &fakeSecrets{}, &fakeIssuers{}).ConfigureTenantIdentity(context.Background(), Input{TenantID: testTenantID, TenantCode: "acme"})
	if err == nil {
		t.Fatal("took over a realm owned by another tenant")
	}
	if res.RealmCreated {
		t.Fatal("foreign realm reported as owned")
	}
	for _, c := range kc.calls {
		if c == "CreatePlatformClient" || c == "AddLDAPFederation" {
			t.Fatalf("configured a foreign realm: %v", kc.calls)
		}
	}
}

// An unowned realm that already exists is not this run's either.
func TestUnownedPreexistingRealmIsConflict(t *testing.T) {
	kc := &fakeKeycloak{realms: map[string]string{"acme": ""}}
	res, err := newActivities(kc, &fakeSecrets{}, &fakeIssuers{}).ConfigureTenantIdentity(context.Background(), Input{TenantID: testTenantID, TenantCode: "acme"})
	if err == nil || res.RealmCreated {
		t.Fatalf("unowned realm adopted: err=%v created=%v", err, res.RealmCreated)
	}
}

// A retry after a partial first attempt must converge. The first attempt
// creates the realm and fails at client creation. The retry finds the realm
// owned by this tenant and completes.
func TestRetryAfterPartialFirstAttemptConverges(t *testing.T) {
	kc := &fakeKeycloak{clientSecret: testClientSecret, failClientOne: true}
	iss := &fakeIssuers{}
	a := newActivities(kc, &fakeSecrets{}, iss)
	in := Input{TenantID: testTenantID, TenantCode: "acme"}

	first, err := a.ConfigureTenantIdentity(context.Background(), in)
	if err == nil {
		t.Fatal("first attempt should fail at client creation")
	}
	if !first.RealmCreated || first.ClientCreated {
		t.Fatalf("first attempt result = %+v", first)
	}

	second, err := a.ConfigureTenantIdentity(context.Background(), in)
	if err != nil {
		t.Fatalf("retry did not converge: %v", err)
	}
	if !second.RealmCreated || !second.ClientCreated || !second.SecretsWritten {
		t.Fatalf("retry result = %+v", second)
	}
	if iss.issuer != testBase+"/realms/acme" {
		t.Fatalf("issuer after retry = %q", iss.issuer)
	}
}

// A realm owned by this tenant from an earlier attempt is reported as owned, so
// compensation can delete it if the run fails later.
func TestOwnedRealmFromEarlierAttemptIsReportedForCompensation(t *testing.T) {
	kc := &fakeKeycloak{realms: map[string]string{"acme": testTenantID}}
	res, err := newActivities(kc, &fakeSecrets{}, &fakeIssuers{}).ConfigureTenantIdentity(context.Background(), Input{TenantID: testTenantID, TenantCode: "acme"})
	if err != nil {
		t.Fatalf("owned realm should continue: %v", err)
	}
	if !res.RealmCreated {
		t.Fatal("owned realm not reported for compensation")
	}
}

func TestConfigureTenantIdentityPartialFailureReportsCreatedRealm(t *testing.T) {
	kc := &fakeKeycloak{failClient: errors.New("keycloak 503")}
	res, err := newActivities(kc, &fakeSecrets{}, &fakeIssuers{}).ConfigureTenantIdentity(context.Background(), Input{TenantID: testTenantID, TenantCode: "acme"})
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
	res, err := newActivities(kc, sec, iss).ConfigureTenantIdentity(context.Background(), Input{TenantID: testTenantID, TenantCode: "acme"})
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
	res, err := newActivities(kc, &fakeSecrets{}, &fakeIssuers{}).ConfigureTenantIdentity(context.Background(), Input{TenantID: testTenantID, TenantCode: "acme"})
	if err != nil {
		t.Fatalf("ConfigureTenantIdentity: %v", err)
	}
	for _, f := range []string{res.Realm, res.Issuer, res.SecretPath} {
		if strings.Contains(f, testClientSecret) {
			t.Fatalf("client secret present in result field %q", f)
		}
	}
}

func TestErrorsNeverContainBindPassword(t *testing.T) {
	kc := &fakeKeycloak{failLDAP: errors.New("ldap refused")}
	_, err := newActivities(kc, seededSecrets(t, "acme"), &fakeIssuers{}).ConfigureTenantIdentity(context.Background(), Input{TenantID: testTenantID, TenantCode: "acme", LDAP: validLDAP()})
	if err == nil {
		t.Fatal("expected LDAP error")
	}
	if strings.Contains(err.Error(), testBindPassword) {
		t.Fatal("bind password leaked into error text")
	}
}

func TestConfigureTenantIdentityWithoutDependenciesFailsClosed(t *testing.T) {
	a := &Activities{PublicBaseURL: testBase}
	if _, err := a.ConfigureTenantIdentity(context.Background(), Input{TenantID: testTenantID, TenantCode: "acme"}); err == nil {
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

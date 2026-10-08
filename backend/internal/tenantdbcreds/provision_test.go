package tenantdbcreds

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const testRegion = "us-east-1"

type fakeSecrets struct {
	stored  map[string]map[string]string
	putPath string
	putVals map[string]string
	puts    int
	putErr  error
	getErr  error
}

func (f *fakeSecrets) GetMap(_ context.Context, key string) (map[string]string, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.stored[key], nil
}

func (f *fakeSecrets) PutMap(_ context.Context, key string, values map[string]string) error {
	f.puts++
	f.putPath = key
	f.putVals = values
	return f.putErr
}

type fakeAdmin struct {
	rec         *[]string
	roleExists  bool
	createErr   error
	grantErr    error
	failRoleQry error
	createdPW   string
}

func (f *fakeAdmin) RoleExists(_ context.Context, role string) (bool, error) {
	*f.rec = append(*f.rec, "RoleExists")
	return f.roleExists, f.failRoleQry
}

func (f *fakeAdmin) CreateRole(_ context.Context, role, password string) error {
	*f.rec = append(*f.rec, "CreateRole")
	f.createdPW = password
	return f.createErr
}

func (f *fakeAdmin) GrantConnect(_ context.Context, role, database string) error {
	*f.rec = append(*f.rec, "GrantConnect")
	return f.grantErr
}

func (f *fakeAdmin) DropRole(_ context.Context, role string) error {
	*f.rec = append(*f.rec, "DropRole")
	return nil
}

func (f *fakeAdmin) Close() error { return nil }

type fakeConnector struct {
	rec      []string
	admin    *fakeAdmin
	connects int
	err      error
	gotUser  string
	gotPass  string
}

func (f *fakeConnector) Connect(_ context.Context, host string, port int, user, pass string) (RoleAdmin, error) {
	f.connects++
	f.gotUser, f.gotPass = user, pass
	if f.err != nil {
		return nil, f.err
	}
	f.admin.rec = &f.rec
	return f.admin, nil
}

func platformAdmin() map[string]string {
	return map[string]string{AdminUsernameKey: "platform_admin", AdminPasswordKey: "admin-secret-value"}
}

// newHarness returns a store with the admin credential present, a connector
// whose admin records calls, and a deterministic random source.
func newHarness() (*fakeSecrets, *fakeConnector, *Activities) {
	st := &fakeSecrets{stored: map[string]map[string]string{
		"platform/postgres/us-east-1/admin": platformAdmin(),
	}}
	conn := &fakeConnector{admin: &fakeAdmin{}}
	a := &Activities{Secrets: st, Connector: conn, Random: func(b []byte) (int, error) {
		for i := range b {
			b[i] = byte(i)
		}
		return len(b), nil
	}}
	return st, conn, a
}

func validInput() Input {
	return Input{
		TenantCode: "acme", Environment: "dev", Region: testRegion,
		DatabaseName: "tenant_acme", DatabaseHost: "pg.us-east-1.internal", DatabasePort: 5432,
	}
}

func TestProvisionCreatesRoleAndStoresCredentials(t *testing.T) {
	st, conn, a := newHarness()
	res, err := a.ProvisionDbCredentials(context.Background(), validInput())
	if err != nil {
		t.Fatalf("ProvisionDbCredentials: %v", err)
	}
	if !res.Created || res.Skipped || res.Role != "tenant_acme_dev" || res.Path != "tenants/acme/dev/db_credentials" {
		t.Fatalf("result = %+v", res)
	}
	want := []string{"RoleExists", "CreateRole", "GrantConnect"}
	if strings.Join(conn.rec, ",") != strings.Join(want, ",") {
		t.Fatalf("role operations = %v, want %v", conn.rec, want)
	}
	if conn.gotUser != "platform_admin" || conn.gotPass != "admin-secret-value" {
		t.Fatal("admin credential not read from the platform path")
	}
	if st.putPath != "tenants/acme/dev/db_credentials" || st.putVals["username"] != "tenant_acme_dev" {
		t.Fatalf("stored = %q %v", st.putPath, st.putVals)
	}
	if st.putVals["password"] == "" || st.putVals["password"] != conn.admin.createdPW {
		t.Fatal("stored password does not match the role password")
	}
}

func TestExistingCredentialsAreSkippedNeverRotated(t *testing.T) {
	st, conn, a := newHarness()
	st.stored["tenants/acme/dev/db_credentials"] = map[string]string{"username": "tenant_acme_dev", "password": "already-set"}
	res, err := a.ProvisionDbCredentials(context.Background(), validInput())
	if err != nil {
		t.Fatalf("ProvisionDbCredentials: %v", err)
	}
	if !res.Skipped || res.Created {
		t.Fatalf("result = %+v, want skipped", res)
	}
	if conn.connects != 0 || st.puts != 0 {
		t.Fatalf("connected or wrote despite existing credentials: connects=%d puts=%d", conn.connects, st.puts)
	}
}

func TestRoleWithoutStoredCredentialsIsAHumanProblem(t *testing.T) {
	_, conn, a := newHarness()
	conn.admin.roleExists = true
	_, err := a.ProvisionDbCredentials(context.Background(), validInput())
	if err == nil {
		t.Fatal("reused a role whose password is unrecoverable")
	}
	for _, op := range conn.rec {
		if op == "CreateRole" {
			t.Fatal("created a role over an existing one")
		}
	}
}

func TestMissingAdminCredentialFailsBeforeConnecting(t *testing.T) {
	st, conn, a := newHarness()
	delete(st.stored, "platform/postgres/us-east-1/admin")
	if _, err := a.ProvisionDbCredentials(context.Background(), validInput()); err == nil {
		t.Fatal("ran without an admin credential")
	}
	if conn.connects != 0 {
		t.Fatal("connected without an admin credential")
	}
}

func TestGrantFailureUndoesTheRole(t *testing.T) {
	st, conn, a := newHarness()
	conn.admin.grantErr = errors.New("permission denied")
	if _, err := a.ProvisionDbCredentials(context.Background(), validInput()); err == nil {
		t.Fatal("grant failure not reported")
	}
	if !contains(conn.rec, "DropRole") {
		t.Fatalf("role not undone after grant failure: %v", conn.rec)
	}
	if st.puts != 0 {
		t.Fatal("credentials stored after a failed grant")
	}
}

func TestStoreFailureUndoesTheRole(t *testing.T) {
	st, conn, a := newHarness()
	st.putErr = errors.New("infisical 503")
	if _, err := a.ProvisionDbCredentials(context.Background(), validInput()); err == nil {
		t.Fatal("store failure not reported")
	}
	if !contains(conn.rec, "DropRole") {
		t.Fatalf("role not undone after store failure: %v", conn.rec)
	}
}

// The password is the most sensitive value in this step. It must not appear in
// the result, in any error, or in the path that the store is asked for.
func TestPasswordNeverAppearsInErrorsOrResult(t *testing.T) {
	st, conn, a := newHarness()
	st.putErr = errors.New("store echoed the password")
	conn.admin.grantErr = nil
	res, err := a.ProvisionDbCredentials(context.Background(), validInput())
	if err == nil {
		t.Fatal("expected failure")
	}
	pw := conn.admin.createdPW
	if pw == "" {
		t.Fatal("test did not capture the generated password")
	}
	if strings.Contains(err.Error(), pw) || strings.Contains(res.Role+res.Path, pw) {
		t.Fatalf("password leaked: %v", err)
	}
}

func TestPasswordIsRestrictedAlphabetAndLength(t *testing.T) {
	_, _, a := newHarness()
	pw, err := a.generatePassword()
	if err != nil {
		t.Fatalf("generatePassword: %v", err)
	}
	if len(pw) < 40 || !safePassword.MatchString(pw) {
		t.Fatalf("password %q is not restricted-alphabet, long enough", pw)
	}
}

func TestRandomSourceFailureIsReported(t *testing.T) {
	_, _, a := newHarness()
	a.Random = func([]byte) (int, error) { return 0, errors.New("entropy exhausted") }
	if _, err := a.generatePassword(); err == nil {
		t.Fatal("short or failed entropy read accepted")
	}
	a.Random = func(b []byte) (int, error) { return len(b) - 1, nil }
	if _, err := a.generatePassword(); err == nil {
		t.Fatal("partial entropy read accepted")
	}
}

func TestRejectsBadInputBeforeAnyConnection(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Input)
	}{
		{"tenant code injection", func(i *Input) { i.TenantCode = "x; DROP DATABASE alpha;--" }},
		{"tenant code dotdot", func(i *Input) { i.TenantCode = ".." }},
		{"environment unknown", func(i *Input) { i.Environment = "staging" }},
		{"region injection", func(i *Input) { i.Region = "us-east-1; drop" }},
		{"region uppercase", func(i *Input) { i.Region = "US-EAST-1" }},
		{"database name quote", func(i *Input) { i.DatabaseName = `a"b` }},
		{"database name overlong", func(i *Input) { i.DatabaseName = "d" + strings.Repeat("x", 63) }},
		{"host semicolon", func(i *Input) { i.DatabaseHost = "pg;evil" }},
		{"host empty", func(i *Input) { i.DatabaseHost = "" }},
		{"port zero", func(i *Input) { i.DatabasePort = 0 }},
		{"port too large", func(i *Input) { i.DatabasePort = 70000 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st, conn, a := newHarness()
			in := validInput()
			tc.mutate(&in)
			if _, err := a.ProvisionDbCredentials(context.Background(), in); err == nil {
				t.Fatal("accepted invalid input")
			}
			if conn.connects != 0 || st.puts != 0 {
				t.Fatalf("connected or wrote for invalid input: connects=%d puts=%d", conn.connects, st.puts)
			}
		})
	}
}

func TestWithoutDependenciesFailsClosed(t *testing.T) {
	if _, err := (&Activities{}).ProvisionDbCredentials(context.Background(), validInput()); err == nil {
		t.Fatal("ran without dependencies")
	}
}

// CREATE ROLE takes no bind parameters, so the password is embedded. The adapter
// must refuse anything outside the restricted alphabet before it touches the
// connection. A nil connection makes any attempt to proceed panic, which fails
// the test.
func TestAdapterRefusesUnsafePasswordBeforeTouchingConnection(t *testing.T) {
	p := &pgxRoleAdmin{conn: nil}
	for _, pw := range []string{"x' ; DROP ROLE platform_admin; --", "has space", "quote'", ""} {
		if err := p.CreateRole(context.Background(), "tenant_acme_dev", pw); err == nil {
			t.Errorf("CreateRole accepted password %q", pw)
		}
	}
}

func TestAdminPathValidatesRegion(t *testing.T) {
	if p, err := AdminPath("eu-west-1"); err != nil || p != "platform/postgres/eu-west-1/admin" {
		t.Fatalf("AdminPath = %q, %v", p, err)
	}
	for _, r := range []string{"", "us-east-1/../x", "us east 1", strings.Repeat("a", 40)} {
		if _, err := AdminPath(r); err == nil {
			t.Errorf("AdminPath accepted %q", r)
		}
	}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

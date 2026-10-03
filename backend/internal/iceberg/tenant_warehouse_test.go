package iceberg

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
)

// fakeLakekeeper is just enough of the management API for warehouse
// provisioning: list-by-name and create.
type fakeLakekeeper struct {
	mu         sync.Mutex
	warehouses map[string]string // name -> id
	posts      []map[string]interface{}
	failCreate bool // simulate losing a race: warehouse appears, POST errors
	dropCreate bool // POST "succeeds" but the warehouse never appears
}

func (f *fakeLakekeeper) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/management/v1/warehouse":
			name := r.URL.Query().Get("name")
			out := map[string]interface{}{"warehouses": []map[string]string{}}
			if id, ok := f.warehouses[name]; ok {
				out["warehouses"] = []map[string]string{{"id": id, "name": name}}
			}
			_ = json.NewEncoder(w).Encode(out)
		case r.Method == http.MethodPost && r.URL.Path == "/management/v1/warehouse":
			b, _ := io.ReadAll(r.Body)
			var payload map[string]interface{}
			_ = json.Unmarshal(b, &payload)
			f.posts = append(f.posts, payload)
			name, _ := payload["warehouse-name"].(string)
			if f.dropCreate {
				w.WriteHeader(http.StatusCreated)
				return
			}
			f.warehouses[name] = "wh-" + name
			if f.failCreate {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte("already exists"))
				return
			}
			w.WriteHeader(http.StatusCreated)
		default:
			http.NotFound(w, r)
		}
	})
}

func newProvisioner(t *testing.T, f *fakeLakekeeper) *LakekeeperProvisioner {
	t.Helper()
	srv := httptest.NewServer(f.handler())
	t.Cleanup(srv.Close)
	// s3Bucket is the SHARED bucket; the tests assert it is never used.
	return &LakekeeperProvisioner{baseURL: srv.URL, s3Bucket: "shared-bucket", httpClient: srv.Client()}
}

func spec(tenant uuid.UUID) TenantWarehouseSpec {
	return TenantWarehouseSpec{
		TenantID:        tenant,
		Endpoint:        "http://minio:9000",
		AccessKeyID:     "tenant-key",
		SecretAccessKey: "tenant-secret",
	}
}

func TestTenantWarehouseName(t *testing.T) {
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	got, err := TenantWarehouseName(id)
	if err != nil {
		t.Fatal(err)
	}
	if want := "ivy-t-11111111222233334444555555555555"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if len(got) > 63 {
		t.Fatalf("name %q is %d chars, over the 63-char bucket limit", got, len(got))
	}
	if _, err := TenantWarehouseName(uuid.Nil); err == nil {
		t.Fatal("nil tenant must be rejected")
	}
}

func TestEnsureTenantWarehouse_CreatesInTenantsOwnBucket(t *testing.T) {
	f := &fakeLakekeeper{warehouses: map[string]string{}}
	p := newProvisioner(t, f)
	tenant := uuid.New()

	got, err := p.EnsureTenantWarehouse(context.Background(), spec(tenant))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := TenantWarehouseName(tenant)
	if got.Name != want || got.Bucket != want || got.ID == "" || !got.Created {
		t.Fatalf("unexpected result %+v", got)
	}
	if len(f.posts) != 1 {
		t.Fatalf("want 1 create, got %d", len(f.posts))
	}
	profile := f.posts[0]["storage-profile"].(map[string]interface{})
	if profile["bucket"] != want {
		t.Fatalf("bucket = %v, want the tenant's own %q", profile["bucket"], want)
	}
	if strings.Contains(profile["bucket"].(string), "shared-bucket") {
		t.Fatal("the provisioner's shared bucket must never be used")
	}
	cred := f.posts[0]["storage-credential"].(map[string]interface{})
	if cred["access-key-id"] != "tenant-key" || cred["secret-access-key"] != "tenant-secret" {
		t.Fatalf("the tenant's own credential was not used: %v", cred)
	}
}

func TestEnsureTenantWarehouse_Idempotent(t *testing.T) {
	f := &fakeLakekeeper{warehouses: map[string]string{}}
	p := newProvisioner(t, f)
	tenant := uuid.New()

	first, err := p.EnsureTenantWarehouse(context.Background(), spec(tenant))
	if err != nil {
		t.Fatal(err)
	}
	second, err := p.EnsureTenantWarehouse(context.Background(), spec(tenant))
	if err != nil {
		t.Fatal(err)
	}
	if second.Created || second.ID != first.ID {
		t.Fatalf("second call must return the existing warehouse: %+v vs %+v", second, first)
	}
	if len(f.posts) != 1 {
		t.Fatalf("want exactly 1 create across two calls, got %d", len(f.posts))
	}
}

func TestEnsureTenantWarehouse_TwoTenantsGetTwoWarehouses(t *testing.T) {
	f := &fakeLakekeeper{warehouses: map[string]string{}}
	p := newProvisioner(t, f)
	a, err := p.EnsureTenantWarehouse(context.Background(), spec(uuid.New()))
	if err != nil {
		t.Fatal(err)
	}
	b, err := p.EnsureTenantWarehouse(context.Background(), spec(uuid.New()))
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == b.ID || a.Bucket == b.Bucket {
		t.Fatalf("tenants must not share a warehouse or bucket: %+v %+v", a, b)
	}
}

func TestEnsureTenantWarehouse_LostRaceIsNotAnError(t *testing.T) {
	f := &fakeLakekeeper{warehouses: map[string]string{}, failCreate: true}
	p := newProvisioner(t, f)

	got, err := p.EnsureTenantWarehouse(context.Background(), spec(uuid.New()))
	if err != nil {
		t.Fatalf("a create that lost a race must resolve to the existing warehouse: %v", err)
	}
	if got.ID == "" || got.Created {
		t.Fatalf("want existing warehouse, Created=false; got %+v", got)
	}
}

func TestEnsureTenantWarehouse_FailsWhenWarehouseNeverAppears(t *testing.T) {
	f := &fakeLakekeeper{warehouses: map[string]string{}, dropCreate: true}
	p := newProvisioner(t, f)
	if _, err := p.EnsureTenantWarehouse(context.Background(), spec(uuid.New())); err == nil {
		t.Fatal("a create that leaves no warehouse behind must not report success")
	}
}

func TestEnsureTenantWarehouse_RejectsIncompleteSpec(t *testing.T) {
	f := &fakeLakekeeper{warehouses: map[string]string{}}
	p := newProvisioner(t, f)
	good := spec(uuid.New())

	cases := map[string]func(*TenantWarehouseSpec){
		"nil tenant":    func(s *TenantWarehouseSpec) { s.TenantID = uuid.Nil },
		"no access key": func(s *TenantWarehouseSpec) { s.AccessKeyID = "" },
		"no secret":     func(s *TenantWarehouseSpec) { s.SecretAccessKey = "" },
		"no endpoint":   func(s *TenantWarehouseSpec) { s.Endpoint = "" },
	}
	for name, mutate := range cases {
		s := good
		mutate(&s)
		if _, err := p.EnsureTenantWarehouse(context.Background(), s); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
	if len(f.posts) != 0 {
		t.Fatalf("an invalid spec must not reach Lakekeeper; got %d creates", len(f.posts))
	}
}

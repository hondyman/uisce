package provisioning

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeCatalog struct {
	regions  []Region
	products []Product
	taken    map[string]bool
	err      error
}

func (f fakeCatalog) Regions(context.Context) ([]Region, error)   { return f.regions, f.err }
func (f fakeCatalog) Products(context.Context) ([]Product, error) { return f.products, f.err }
func (f fakeCatalog) DatabaseNameTaken(_ context.Context, n string) (bool, error) {
	return f.taken[n], f.err
}

func testCatalog() fakeCatalog {
	return fakeCatalog{
		regions: []Region{
			{Code: "us-east-1", Name: "US East (N. Virginia)", ClusterHost: "100.84.50.65", ClusterPort: 5432},
			{Code: "us-west-2", Name: "US West (Oregon)"},
			{Code: "eu-west-1", Name: "Europe (Ireland)"},
		},
		products: []Product{{Code: "ORM", Name: "Order and risk management"}, {Code: "MDM", Name: "Master data"}},
		taken:    map[string]bool{"taken_orm": true},
	}
}

func TestDatabaseNameFor(t *testing.T) {
	cases := []struct {
		label, product, want string
		wantErr              bool
	}{
		{"ABC", "ORM", "abc_orm", false},
		{"  Xyz_Inv ", "orm", "xyz_inv_orm", false},
		{"1abc", "orm", "", true},
		{"a-b", "orm", "", true},
		{"a b", "orm", "", true},
		{"", "orm", "", true},
		{"abc", "", "", true},
		{"abc;drop", "orm", "", true},
		{strings.Repeat("a", 33), "orm", "", true},
		{strings.Repeat("a", 32), "orm", strings.Repeat("a", 32) + "_orm", false},
	}
	for _, c := range cases {
		got, err := DatabaseNameFor(c.label, c.product)
		if (err != nil) != c.wantErr || got != c.want {
			t.Errorf("DatabaseNameFor(%q,%q) = %q, %v; want %q, err=%v", c.label, c.product, got, err, c.want, c.wantErr)
		}
	}
}

func TestDatabaseNameFor_RoleFits(t *testing.T) {
	// The longest label and product still leave room for "_app" in 63 bytes.
	name, err := DatabaseNameFor(strings.Repeat("a", 32), strings.Repeat("b", 16))
	if err == nil && len(name)+len("_app") > 63 {
		t.Fatalf("%q plus the role suffix exceeds 63 bytes", name)
	}
}

func TestResolveRegion(t *testing.T) {
	regions := testCatalog().regions
	cases := []struct {
		in      string
		want    string
		wantErr error
	}{
		{"us-east-1", "us-east-1", nil},
		{"US-EAST-1", "us-east-1", nil},
		{"US East", "us-east-1", nil},
		{"us east", "us-east-1", nil},
		{"US East (N. Virginia)", "us-east-1", nil},
		{"Europe", "eu-west-1", nil},
		{"us", "", ErrRegionAmbiguous},
		{"mars", "", ErrRegionUnknown},
		{"", "", ErrRegionUnknown},
		{"east", "", ErrRegionUnknown},
	}
	for _, c := range cases {
		got, _, err := ResolveRegion(c.in, regions)
		if !errors.Is(err, c.wantErr) || got.Code != c.want {
			t.Errorf("ResolveRegion(%q) = %q, %v; want %q, %v", c.in, got.Code, err, c.want, c.wantErr)
		}
	}
	_, cands, _ := ResolveRegion("us", regions)
	if len(cands) != 2 {
		t.Errorf("ambiguous candidates = %d, want 2", len(cands))
	}
}

func TestDescribe_CompleteRequest(t *testing.T) {
	req := ProvisionTenantRequest{
		TenantName: "XYZ Investments", Region: "US East",
		Products: []ProductRequest{{Product: "ORM", Label: "ABC"}},
	}
	got, err := Describe(context.Background(), req, testCatalog())
	if err != nil {
		t.Fatal(err)
	}
	if !got.Complete || got.Plan == nil || got.Normalized == nil {
		t.Fatalf("not complete: %+v", got)
	}
	p := got.Plan
	if p.Region != "us-east-1" || p.Host != "100.84.50.65" || p.Port != 5432 {
		t.Errorf("region/host = %s %s:%d", p.Region, p.Host, p.Port)
	}
	if len(p.Databases) != 1 || p.Databases[0].Database != "abc_orm" || p.Databases[0].Role != "abc_orm_app" {
		t.Errorf("databases = %+v", p.Databases)
	}
	if got.Normalized.Region != "us-east-1" || got.Normalized.InstanceName != DefaultInstanceName ||
		got.Normalized.Products[0] != (ProductRequest{Product: "orm", Label: "abc"}) {
		t.Errorf("normalized = %+v", got.Normalized)
	}
	if got.Normalized.TenantCode != "xyz_i" {
		t.Errorf("derived code = %q", got.Normalized.TenantCode)
	}
}

func fields(is []Issue) map[string]Issue {
	m := map[string]Issue{}
	for _, i := range is {
		m[i.Field] = i
	}
	return m
}

func TestDescribe_EmptyRequestAsksForEverything(t *testing.T) {
	got, err := Describe(context.Background(), ProvisionTenantRequest{}, testCatalog())
	if err != nil {
		t.Fatal(err)
	}
	if got.Complete || got.Plan != nil {
		t.Fatalf("an empty request is complete: %+v", got)
	}
	m := fields(got.Missing)
	for _, f := range []string{"tenant_name", "region", "products"} {
		if _, ok := m[f]; !ok {
			t.Errorf("%s is not asked for: %+v", f, got.Missing)
		}
	}
	if len(m["region"].Allowed) != 3 {
		t.Errorf("regions offered = %v", m["region"].Allowed)
	}
	if len(m["products"].Allowed) != 2 || m["products"].Allowed[0] != "mdm" {
		t.Errorf("products offered = %v", m["products"].Allowed)
	}
}

func TestDescribe_AsksForTheLabel(t *testing.T) {
	req := ProvisionTenantRequest{TenantName: "XYZ Investments", Region: "us-east-1", Products: []ProductRequest{{Product: "orm"}}}
	got, _ := Describe(context.Background(), req, testCatalog())
	if got.Complete {
		t.Fatal("complete without a label")
	}
	if _, ok := fields(got.Missing)["products[0].label"]; !ok {
		t.Errorf("label not asked for: %+v", got.Missing)
	}
}

func TestDescribe_Refusals(t *testing.T) {
	base := func() ProvisionTenantRequest {
		return ProvisionTenantRequest{TenantName: "XYZ Investments", Region: "us-east-1", Products: []ProductRequest{{Product: "orm", Label: "abc"}}}
	}
	cases := []struct {
		name  string
		mod   func(*ProvisionTenantRequest)
		field string
		msg   string
	}{
		{"unknown region", func(r *ProvisionTenantRequest) { r.Region = "mars" }, "region", "not configured"},
		{"ambiguous region", func(r *ProvisionTenantRequest) { r.Region = "us" }, "region", "more than one"},
		{"region without a cluster", func(r *ProvisionTenantRequest) { r.Region = "eu-west-1" }, "region", "no Postgres cluster"},
		{"unknown product", func(r *ProvisionTenantRequest) { r.Products[0].Product = "nope" }, "products[0].product", "not available"},
		{"bad label", func(r *ProvisionTenantRequest) { r.Products[0].Label = "a;b" }, "products[0].label", "label must"},
		{"taken database", func(r *ProvisionTenantRequest) { r.Products[0].Label = "taken" }, "products[0].label", "already exists"},
		{"two products", func(r *ProvisionTenantRequest) {
			r.Products = append(r.Products, ProductRequest{Product: "mdm", Label: "abc"})
		}, "products", "one product per request"},
		{"bad code", func(r *ProvisionTenantRequest) { r.TenantCode = "Bad Code" }, "tenant_code", "must match"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := base()
			c.mod(&req)
			got, err := Describe(context.Background(), req, testCatalog())
			if err != nil {
				t.Fatal(err)
			}
			if got.Complete || got.Plan != nil {
				t.Fatalf("complete despite %s", c.name)
			}
			is, ok := fields(got.Invalid)[c.field]
			if !ok || !strings.Contains(is.Message, c.msg) {
				t.Errorf("want invalid %s containing %q, got %+v", c.field, c.msg, got.Invalid)
			}
		})
	}
}

func TestDescribe_DuplicateDatabaseInOneRequest(t *testing.T) {
	// Two entries that name the same database would share it.
	req := ProvisionTenantRequest{TenantName: "XYZ", Region: "us-east-1",
		Products: []ProductRequest{{Product: "orm", Label: "abc"}, {Product: "ORM", Label: "ABC"}}}
	got, _ := Describe(context.Background(), req, testCatalog())
	if got.Complete {
		t.Fatal("complete with the same database named twice")
	}
}

func TestDescribe_CatalogErrorFailsClosed(t *testing.T) {
	cat := testCatalog()
	cat.err = errors.New("db down")
	got, err := Describe(context.Background(), ProvisionTenantRequest{TenantName: "X", Region: "us-east-1"}, cat)
	if err == nil || got.Complete {
		t.Fatalf("a catalog error must be an error, got %+v, %v", got, err)
	}
}

func TestDeterministicIDs(t *testing.T) {
	n := ProvisionTenantRequest{TenantName: "XYZ Investments", TenantCode: "xyz_i", Region: "us-east-1", InstanceName: "primary",
		Products: []ProductRequest{{Product: "orm", Label: "abc"}}}
	w1, t1, i1 := deterministicIDs("tenant-provisioning", n)
	w2, t2, i2 := deterministicIDs("tenant-provisioning", n)
	if w1 != w2 || t1 != t2 || i1 != i2 {
		t.Fatal("the same request must give the same ids")
	}
	if t1 == i1 {
		t.Error("tenant and instance ids must differ")
	}
	if !strings.HasPrefix(w1, "tenant-provisioning-xyz_i-") {
		t.Errorf("workflow id = %q", w1)
	}
	n.Products[0].Label = "def"
	if w3, t3, _ := deterministicIDs("tenant-provisioning", n); w3 == w1 || t3 == t1 {
		t.Error("a different label is a different request")
	}
}

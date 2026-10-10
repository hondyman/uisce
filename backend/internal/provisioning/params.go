package provisioning

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// This file is the parameter model for "create tenant X in region R, register product P with label L".
// It is pure: names and regions are normalized and checked here, against a Catalog that reads the
// regions, the cluster each one runs on and the products. The describe endpoint and the create
// handler share it, so the wizard prompts for exactly what create would refuse.

// maxDatabaseNameLen leaves room for the "_app" suffix of the tenant role (63 bytes in all).
const maxDatabaseNameLen = 59

// DefaultInstanceName names the one instance of a tenant created from a product list.
const DefaultInstanceName = "primary"

var (
	// labelPattern: a short lowercase identifier. The label is the tenant's own name for a product
	// install ("ABC"), and it is the front of the database name, so it is never free text.
	labelPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)
	// productPattern is the normalized product code ("ORM" -> "orm").
	productPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,15}$`)
	nonAlnum       = regexp.MustCompile(`[^a-z0-9]+`)
)

// ProductRequest is one product to register for the tenant, with the label that names its database.
type ProductRequest struct {
	Product string `json:"product"`
	Label   string `json:"label"`
}

// NormalizeLabel lowercases and validates a label. "ABC" is "abc".
func NormalizeLabel(s string) (string, error) {
	l := strings.ToLower(strings.TrimSpace(s))
	if !labelPattern.MatchString(l) {
		return "", errors.New("label must start with a letter and use only letters, digits and underscores (at most 32)")
	}
	return l, nil
}

// NormalizeProductCode lowercases and validates a product code.
func NormalizeProductCode(s string) (string, error) {
	p := strings.ToLower(strings.TrimSpace(s))
	if !productPattern.MatchString(p) {
		return "", errors.New("product must start with a letter and use only letters, digits and underscores (at most 16)")
	}
	return p, nil
}

// DatabaseNameFor is the tenant database of a product install: <label>_<product>, lowercased, so
// label ABC and product ORM give abc_orm. It must leave room for the role suffix.
func DatabaseNameFor(label, product string) (string, error) {
	l, err := NormalizeLabel(label)
	if err != nil {
		return "", err
	}
	p, err := NormalizeProductCode(product)
	if err != nil {
		return "", err
	}
	name := l + "_" + p
	if len(name) > maxDatabaseNameLen {
		return "", fmt.Errorf("database name %q is longer than %d characters; use a shorter label", name, maxDatabaseNameLen)
	}
	return name, nil
}

// Region is a configured region and, when one is registered, the Postgres cluster it runs on.
type Region struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	ClusterHost string `json:"-"`
	ClusterPort int    `json:"-"`
}

// HasCluster reports whether tenant databases can be created in the region.
func (r Region) HasCluster() bool { return r.ClusterHost != "" && r.ClusterPort > 0 }

// Product is a product tenants can register.
type Product struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// Catalog is what the parameter checks read. Every method fails closed: an error is an error, never
// "none".
type Catalog interface {
	// Regions are the active regions.
	Regions(ctx context.Context) ([]Region, error)
	// Products are the active products.
	Products(ctx context.Context) ([]Product, error)
	// DatabaseNameTaken reports whether a database of this name exists on the control cluster or is
	// already bound to a datasource.
	DatabaseNameTaken(ctx context.Context, name string) (bool, error)
}

// words lowercases and splits on anything that is not a letter or digit, so "US East (N. Virginia)",
// "us-east-1" and "US East" compare by their words.
func words(s string) []string {
	return strings.Fields(nonAlnum.ReplaceAllString(strings.ToLower(s), " "))
}

func hasWordPrefix(full, prefix []string) bool {
	if len(prefix) == 0 || len(prefix) > len(full) {
		return false
	}
	for i := range prefix {
		if full[i] != prefix[i] {
			return false
		}
	}
	return true
}

// ErrRegionUnknown and ErrRegionAmbiguous classify ResolveRegion's refusals.
var (
	ErrRegionUnknown   = errors.New("region is not configured")
	ErrRegionAmbiguous = errors.New("region matches more than one configured region")
)

// ResolveRegion finds the configured region a person means by "us-east-1", "US East" or the full
// name. An exact code or name wins; otherwise the words must begin a region's code or name, and
// exactly one region may match. On ErrRegionAmbiguous the candidates are returned.
func ResolveRegion(input string, regions []Region) (Region, []Region, error) {
	in := words(input)
	if len(in) == 0 {
		return Region{}, nil, ErrRegionUnknown
	}
	for _, r := range regions {
		if strings.EqualFold(strings.TrimSpace(input), r.Code) || strings.EqualFold(strings.TrimSpace(input), r.Name) {
			return r, nil, nil
		}
	}
	var hits []Region
	for _, r := range regions {
		if hasWordPrefix(words(r.Code), in) || hasWordPrefix(words(r.Name), in) {
			hits = append(hits, r)
		}
	}
	switch len(hits) {
	case 0:
		return Region{}, nil, ErrRegionUnknown
	case 1:
		return hits[0], nil, nil
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Code < hits[j].Code })
	return Region{}, hits, ErrRegionAmbiguous
}

// Issue is one thing the wizard must ask about or the caller must fix.
type Issue struct {
	Field   string   `json:"field"`
	Message string   `json:"message"`
	Allowed []string `json:"allowed,omitempty"`
}

// PlannedDatabase is one database the request will create.
type PlannedDatabase struct {
	Product  string `json:"product"`
	Label    string `json:"label"`
	Database string `json:"database"`
	Role     string `json:"role"`
}

// Plan is what a complete request will do. It is shown for confirmation before anything is created.
type Plan struct {
	TenantName string            `json:"tenant_name"`
	TenantCode string            `json:"tenant_code"`
	Region     string            `json:"region"`
	RegionName string            `json:"region_name"`
	Host       string            `json:"host"`
	Port       int               `json:"port"`
	Databases  []PlannedDatabase `json:"databases"`
}

// DescribeResponse answers "what is still needed". Complete is true only when nothing is missing or
// invalid, and then Plan is set.
type DescribeResponse struct {
	Complete bool    `json:"complete"`
	Missing  []Issue `json:"missing,omitempty"`
	Invalid  []Issue `json:"invalid,omitempty"`
	Plan     *Plan   `json:"plan,omitempty"`
	// Normalized is the request with the region resolved and the names normalized, so a client can
	// send it back as it is.
	Normalized *ProvisionTenantRequest `json:"normalized,omitempty"`
}

// Describe checks a possibly partial request. It reads the catalog and creates nothing.
func Describe(ctx context.Context, req ProvisionTenantRequest, cat Catalog) (DescribeResponse, error) {
	var out DescribeResponse
	norm := req
	norm.Products = nil
	if strings.TrimSpace(norm.InstanceName) == "" {
		// A tenant created from a product list has one instance, which the caller need not name.
		norm.InstanceName = DefaultInstanceName
	}

	if strings.TrimSpace(req.TenantName) == "" {
		out.Missing = append(out.Missing, Issue{Field: "tenant_name", Message: "the tenant's name is required"})
	}
	if req.TenantCode != "" && !codePattern.MatchString(req.TenantCode) {
		out.Invalid = append(out.Invalid, Issue{Field: "tenant_code", Message: "tenant_code must match ^[a-z][a-z0-9_]{0,40}$"})
	}
	if norm.TenantCode == "" && strings.TrimSpace(req.TenantName) != "" {
		norm.TenantCode = GenerateTenantCode(req.TenantName)
		if !codePattern.MatchString(norm.TenantCode) {
			out.Missing = append(out.Missing, Issue{Field: "tenant_code", Message: "a code could not be derived from the name; give one"})
		}
	}

	regions, err := cat.Regions(ctx)
	if err != nil {
		return DescribeResponse{}, fmt.Errorf("read regions: %w", err)
	}
	var region Region
	allowedRegions := make([]string, 0, len(regions))
	for _, r := range regions {
		allowedRegions = append(allowedRegions, r.Code+" ("+r.Name+")")
	}
	if strings.TrimSpace(req.Region) == "" {
		out.Missing = append(out.Missing, Issue{Field: "region", Message: "the region to create the tenant in", Allowed: allowedRegions})
	} else {
		var cands []Region
		region, cands, err = ResolveRegion(req.Region, regions)
		switch {
		case errors.Is(err, ErrRegionAmbiguous):
			var allowed []string
			for _, c := range cands {
				allowed = append(allowed, c.Code+" ("+c.Name+")")
			}
			out.Invalid = append(out.Invalid, Issue{Field: "region", Message: "that matches more than one region; pick one", Allowed: allowed})
		case err != nil:
			out.Invalid = append(out.Invalid, Issue{Field: "region", Message: "that region is not configured", Allowed: allowedRegions})
		case !region.HasCluster():
			out.Invalid = append(out.Invalid, Issue{Field: "region", Message: region.Code + " has no Postgres cluster registered, so databases cannot be created there"})
		default:
			norm.Region = region.Code
		}
	}

	products, err := cat.Products(ctx)
	if err != nil {
		return DescribeResponse{}, fmt.Errorf("read products: %w", err)
	}
	known := map[string]Product{}
	var allowedProducts []string
	for _, p := range products {
		if code, e := NormalizeProductCode(p.Code); e == nil {
			known[code] = p
			allowedProducts = append(allowedProducts, code)
		}
	}
	sort.Strings(allowedProducts)

	// A legacy request names an app and no products: it is not the product path.
	if len(req.Products) == 0 && req.App == "" {
		out.Missing = append(out.Missing, Issue{Field: "products", Message: "at least one product to register, with a label for its database", Allowed: allowedProducts})
	}
	// The saga builds one tenant database per run today. The shape of the request is a list so that
	// more databases fit without changing it.
	if len(req.Products) > 1 {
		out.Invalid = append(out.Invalid, Issue{Field: "products", Message: "one product per request is supported for now"})
	}
	seenDB := map[string]bool{}
	var dbs []PlannedDatabase
	for i, p := range req.Products {
		field := fmt.Sprintf("products[%d]", i)
		code, perr := NormalizeProductCode(p.Product)
		if strings.TrimSpace(p.Product) == "" {
			out.Missing = append(out.Missing, Issue{Field: field + ".product", Message: "the product to register", Allowed: allowedProducts})
			continue
		}
		if perr != nil {
			out.Invalid = append(out.Invalid, Issue{Field: field + ".product", Message: perr.Error(), Allowed: allowedProducts})
			continue
		}
		if _, ok := known[code]; !ok {
			out.Invalid = append(out.Invalid, Issue{Field: field + ".product", Message: "that product is not available", Allowed: allowedProducts})
			continue
		}
		if strings.TrimSpace(p.Label) == "" {
			out.Missing = append(out.Missing, Issue{Field: field + ".label", Message: "a label for " + code + "; the database is <label>_" + code})
			continue
		}
		name, nerr := DatabaseNameFor(p.Label, code)
		if nerr != nil {
			out.Invalid = append(out.Invalid, Issue{Field: field + ".label", Message: nerr.Error()})
			continue
		}
		if seenDB[name] {
			out.Invalid = append(out.Invalid, Issue{Field: field + ".label", Message: "database " + name + " is named twice in this request"})
			continue
		}
		seenDB[name] = true
		taken, terr := cat.DatabaseNameTaken(ctx, name)
		if terr != nil {
			return DescribeResponse{}, fmt.Errorf("check database name: %w", terr)
		}
		if taken {
			out.Invalid = append(out.Invalid, Issue{Field: field + ".label", Message: "database " + name + " already exists; choose another label"})
			continue
		}
		label, _ := NormalizeLabel(p.Label)
		norm.Products = append(norm.Products, ProductRequest{Product: code, Label: label})
		dbs = append(dbs, PlannedDatabase{Product: code, Label: label, Database: name, Role: name + "_app"})
	}

	if len(out.Missing) == 0 && len(out.Invalid) == 0 {
		out.Complete = true
		out.Plan = &Plan{
			TenantName: strings.TrimSpace(req.TenantName), TenantCode: norm.TenantCode,
			Region: region.Code, RegionName: region.Name, Host: region.ClusterHost, Port: region.ClusterPort,
			Databases: dbs,
		}
		norm.TenantName = strings.TrimSpace(req.TenantName)
		out.Normalized = &norm
	}
	return out, nil
}

// requestKey is a stable fingerprint of the facts a run is for. Two requests with the same key are
// the same request: the second must find the first's run, not start another.
func requestKey(n ProvisionTenantRequest) string {
	parts := []string{strings.ToLower(strings.TrimSpace(n.TenantName)), n.TenantCode, n.Region, n.InstanceName}
	prods := make([]string, 0, len(n.Products))
	for _, p := range n.Products {
		prods = append(prods, p.Product+":"+p.Label)
	}
	sort.Strings(prods)
	parts = append(parts, prods...)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:])
}

// idNamespace seeds the ids derived from a request key. It is a constant, so the same request
// always yields the same ids; changing it would orphan retries of runs already started.
var idNamespace = uuid.MustParse("6f2b3a64-61a0-4c1e-9a47-3b7c1c9a2e10")

// deterministicIDs returns the workflow id and the tenant and instance ids for a request, so that a
// retried request names the same run and the same rows.
func deterministicIDs(base string, n ProvisionTenantRequest) (workflowID, tenantID, instanceID string) {
	key := requestKey(n)
	return fmt.Sprintf("%s-%s-%s", base, n.TenantCode, key[:8]),
		uuid.NewSHA1(idNamespace, []byte("tenant:"+key)).String(),
		uuid.NewSHA1(idNamespace, []byte("instance:"+key)).String()
}

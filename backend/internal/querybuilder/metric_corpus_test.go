package querybuilder

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// corpusVersion is the corpus's own version, independent of the bundle schema.
// It moves when the CORPUS changes shape; MetricBundleSchemaVersion moves when
// the wire format does. C3's byte-identical gate pins both.
const corpusVersion = "uisce.metric-corpus/1"

const corpusDir = "testdata/metric_corpus/v1"

var updateCorpus = flag.Bool("update-corpus", false,
	"rewrite the metric corpus from current compiler behaviour (review the diff before committing)")

// corpusCase is one metric's expected compilation, recorded byte-for-byte.
type corpusExpectation struct {
	MetricID string   `json:"metricId"`
	SQLExpr  string   `json:"sqlExpr"`
	Args     []string `json:"args"`
	// KnownDefect marks an expectation that pins behaviour believed to be WRONG.
	// The freeze records what the code does; a defect pinned this way has to be
	// flipped deliberately, which is the point. An unmarked defect would let the
	// corpus quietly become the defence of a bug.
	KnownDefect string `json:"knownDefect,omitempty"`
}

type corpusManifest struct {
	Version      string   `json:"version"`
	BundleSchema string   `json:"bundleSchema"`
	Tenants      []string `json:"tenants"`
	Cases        int      `json:"cases"`
}

// corpusMetrics builds the corpus. It spans:
//   - core (master-tenant, IsCore) and custom (own-tenant) metrics
//   - aggregation, formula, and derived - ratio AND n-operand-sum
//   - operand-explicit ratios in BOTH directions, proving direction is data
//   - a calc-fed metric, pinned as a known defect
//   - two tenants, so nothing in the corpus can quietly assume a single tenant
func corpusMetrics() []MetricDefinition {
	gold := "00000000-0000-4000-8000-000000000001"
	acme := "00000000-0000-4000-8000-000000000002"

	core := func(id, name, fn, term string) MetricDefinition {
		return MetricDefinition{
			ID: id, TenantID: gold, Name: name, BOID: "bo_sales",
			Expression: MetricExpression{Kind: "aggregation", Fn: fn, TermNodeID: term},
			GrainAllowlist: []string{"country", "product", "order_date"},
			IsCore: true, Status: "active", Decomposable: true,
		}
	}
	custom := func(id, name, term string) MetricDefinition {
		return MetricDefinition{
			ID: id, TenantID: acme, Name: name, BOID: "bo_sales",
			Expression: MetricExpression{Kind: "aggregation", Fn: "sum", TermNodeID: term},
			GrainAllowlist: []string{"country", "product", "order_date"},
			Status: "active", Decomposable: true,
		}
	}

	return []MetricDefinition{
		// --- core (gold/master tenant) -------------------------------------
		core("m_rev_core", "Revenue", "sum", "revenue"),
		core("m_cost_core", "Cost", "sum", "cost"),
		{
			ID: "m_margin_core", TenantID: gold, Name: "Margin Ratio", BOID: "bo_sales",
			Expression: MetricExpression{
				Kind: "derived", BaseMetricIDs: []string{"m_rev_core", "m_cost_core"},
				NumeratorID: "m_rev_core", DenominatorID: "m_cost_core",
			},
			GrainAllowlist: []string{"country", "product", "order_date"},
			IsCore:        true, Status: "active", Decomposable: true,
		},
		{
			// KNOWN DEFECT. A calc-fed metric: TermNodeID names a calc term, so
			// this compiles to SUM(t0.<calcTermId>) - a column that does not
			// exist. There is no calcTermID field on MetricExpression; routing a
			// metric through a calc term is exactly the C1/C2 gap. Pinned so C2's
			// fix is a visible, deliberate flip rather than a silent one.
			ID: "m_calc_fed_core", TenantID: gold, Name: "Net Interest Income", BOID: "bo_banking",
			Expression: MetricExpression{
				Kind: "aggregation", Fn: "sum", TermNodeID: "calc_term_net_interest_income",
			},
			GrainAllowlist: []string{"account", "period"},
			IsCore:         true, Status: "active", Decomposable: true,
		},

		// --- custom (own-tenant) -------------------------------------------
		custom("m_units_acme", "Units Sold", "units_sold"),
		custom("m_returns_acme", "Returns", "returns"),
		{
			// Literal-only formula: materializable, so it must survive the cube
			// DDL refusal. A parameterised formula is NOT in the corpus on
			// purpose - it is rejected at cube deploy, so there is no stable
			// compiled output to protect.
			ID: "m_gross_acme", TenantID: acme, Name: "Gross", BOID: "bo_sales",
			Expression: MetricExpression{Kind: "formula", Formula: "SUM(t0.revenue) - SUM(t0.returns)"},
			GrainAllowlist: []string{"country", "product", "order_date"},
			Status:         "active", Decomposable: true,
		},
		{
			ID: "m_return_rate_acme", TenantID: acme, Name: "Return Rate", BOID: "bo_sales",
			Expression: MetricExpression{
				Kind: "derived", BaseMetricIDs: []string{"m_returns_acme", "m_units_acme"},
				NumeratorID: "m_returns_acme", DenominatorID: "m_units_acme",
			},
			GrainAllowlist: []string{"country", "product", "order_date"},
			Status:         "active", Decomposable: true,
		},
		{
			// The SAME two operands as m_margin_core, named the other way round.
			// If direction were ever inferred from anything but the named
			// operands, these two would collapse together.
			ID: "m_cost_ratio_core", TenantID: gold, Name: "Cost Ratio", BOID: "bo_sales",
			Expression: MetricExpression{
				Kind: "derived", BaseMetricIDs: []string{"m_cost_core", "m_rev_core"},
				NumeratorID: "m_cost_core", DenominatorID: "m_rev_core",
			},
			GrainAllowlist: []string{"country", "product", "order_date"},
			IsCore:        true, Status: "active", Decomposable: true,
		},
		{
			// N-operand sum: order cannot change the value, so it stays sorted.
			// Present so the corpus covers the one derived shape that is NOT a ratio.
			ID: "m_triple_core", TenantID: gold, Name: "Rev + Cost + Units", BOID: "bo_sales",
			Expression: MetricExpression{
				Kind:          "derived",
				BaseMetricIDs: []string{"m_units_acme", "m_cost_core", "m_rev_core"},
			},
			GrainAllowlist: []string{"country", "product", "order_date"},
			IsCore:         true, Status: "active", Decomposable: true,
		},
	}
}

func TestMetricCorpus(t *testing.T) {
	metrics := corpusMetrics()

	// --- ingest through the real boundary, one bundle per tenant -----------
	byTenant := map[string][]MetricDefinition{}
	for _, m := range metrics {
		byTenant[m.TenantID] = append(byTenant[m.TenantID], m)
	}
	tenants := make([]string, 0, len(byTenant))
	for t2 := range byTenant {
		tenants = append(tenants, t2)
	}
	sort.Strings(tenants)

	// Cross-tenant references (m_triple_core names an acme metric) resolve
	// through the existing set, exactly as a real import would. The set must
	// hold only what THIS bundle does not restate, or every metric collides
	// with itself and the duplicate-ID guard fires.
	inBundle := func(id string, ms []MetricDefinition) bool {
		for _, m := range ms {
			if m.ID == id {
				return true
			}
		}
		return false
	}

	imported := map[string]MetricDefinition{}
	var bundles []struct {
		tenant string
		data   []byte
	}
	for _, tenant := range tenants {
		bundle, err := ExportMetricBundle(byTenant[tenant])
		require.NoError(t, err)
		// A corpus that carries a wall-clock export time changes on every run
		// and cannot be reviewed as a diff. Zero it; the schema version is the
		// compatibility signal, not the moment it was cut.
		bundle.ExportedAt = time.Time{}

		existing := map[string]MetricDefinition{}
		for _, m := range metrics {
			if !inBundle(m.ID, byTenant[tenant]) {
				existing[m.ID] = m
			}
		}

		got, err := ImportMetricBundle(bundle, existing)
		require.NoErrorf(t, err, "tenant %s must import cleanly", tenant)
		for _, m := range got {
			imported[m.ID] = m
		}
		data, err := json.MarshalIndent(bundle, "", "  ")
		require.NoError(t, err)
		bundles = append(bundles, struct {
			tenant string
			data   []byte
		}{tenant, append(data, '\n')})
	}

	// --- compile and record the expected output ---------------------------
	mc := NewMetricCompiler(nil)
	var expectations []corpusExpectation
	for _, m := range metrics {
		lookup := map[string]MetricDefinition{}
		for id, def := range imported {
			lookup[id] = def
		}
		res, err := mc.CompileMetric(m, nil, lookup)
		require.NoErrorf(t, err, "metric %s must compile", m.ID)
		exp := corpusExpectation{MetricID: m.ID, SQLExpr: res.SQLExpr}
		if m.ID == "m_calc_fed_core" {
			exp.KnownDefect = "calc-fed metric compiles to SUM(t0.<calcTermId>), a column that does not " +
				"exist; no calcTermID field on MetricExpression yet. See ADR-026 and the C1/C2 stream."
		}
		expectations = append(expectations, exp)
	}
	sort.Slice(expectations, func(i, j int) bool { return expectations[i].MetricID < expectations[j].MetricID })

	manifest := corpusManifest{
		Version: corpusVersion, BundleSchema: MetricBundleSchemaVersion,
		Tenants: tenants, Cases: len(expectations),
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	require.NoError(t, err)
	expectData, err := json.MarshalIndent(expectations, "", "  ")
	require.NoError(t, err)

	files := map[string][]byte{
		"manifest.json":     append(manifestData, '\n'),
		"expectations.json": append(expectData, '\n'),
	}
	for _, b := range bundles {
		files[filepath.Join("bundles", b.tenant+".json")] = b.data
	}

	if *updateCorpus {
		for name, data := range files {
			p := filepath.Join(corpusDir, name)
			require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
			require.NoError(t, os.WriteFile(p, data, 0o644))
		}
		t.Logf("corpus rewritten (%d files)", len(files))
		return
	}

	// --- compare byte-for-byte -------------------------------------------
	for name := range files {
		p := filepath.Join(corpusDir, name)
		committed, err := os.ReadFile(p)
		require.NoErrorf(t, err, "corpus file %s is missing - regenerate with -update-corpus", name)
		assert.Equalf(t, string(committed), string(files[name]),
			"corpus file %s differs from current behaviour. If the change is intended, "+
				"regenerate with -update-corpus and read the diff: it is a change to the "+
				"semantics this corpus exists to protect.", name)
	}
}

// TestMetricCorpus_RejectsAmbiguousRatioAtIngestion is the corpus's own
// evidence that it is a real boundary test and not a pile of fixtures: the
// corpus's ratio cases carry explicit operands, and the same shape without them
// is refused on the way in.
func TestMetricCorpus_RejectsAmbiguousRatioAtIngestion(t *testing.T) {
	for _, m := range corpusMetrics() {
		if m.Expression.Kind != "derived" || len(m.Expression.BaseMetricIDs) != 2 {
			continue
		}
		stripped := m
		stripped.Expression.NumeratorID = ""
		stripped.Expression.DenominatorID = ""
		bundle, err := ExportMetricBundle([]MetricDefinition{stripped})
		require.NoError(t, err)
		_, err = ImportMetricBundle(bundle, nil)
		require.Errorf(t, err, "corpus ratio %s must be rejected when its operands are stripped", m.ID)
	}
}

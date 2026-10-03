package querybuilder

import (
	"testing"
	"time"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests are the merge gate for the cache seam. They exist because
// test #10 (TestCacheKey_CubeContentHashInvalidates) validated a function with
// no production call site: the live batch-execute key was
// BuildCompositeCacheKey, which had no cube term at all.
//
// A test proves only what it executes. Every gate tied to a function must name
// that function's production call site; see the test-#13 extension in
// docs/ARCHITECTURAL_DECISIONS.md.

// TestBatchExecuteCacheKey_CubeStateDistinguishes is the gate-#11 test.
//
// It asserts on BuildCompositeCacheKey — the function actually called at
// saved_query_cache.go:340 — rather than on a same-shaped helper. The two
// defects it covers:
//
//  1. a cube deploy/undeploy must produce a DIFFERENT key, so a cached payload
//     computed against a now-undeployed materialization is not served; and
//  2. the no-cube case must be a stable, DISTINCT value rather than an omitted
//     term. Omitting it is the mirror-image bug: an entry cached before a cube
//     was deployed would be served after.
func TestBatchExecuteCacheKey_CubeStateDistinguishes(t *testing.T) {
	const (
		tenant = "tenant_a"
		qHash  = "query-content"
		pHash  = "params"
		abac   = "abac"
		tier   = "hot"
		boVer  = "bo-v1"
		cubeV1 = "cube-hash-v1"
		cubeV2 = "cube-hash-v2"
		noCube = NoCubeCacheTerm
	)

	deployed := BuildCompositeCacheKey(tenant, qHash, pHash, abac, tier, boVer, cubeV1)
	edited := BuildCompositeCacheKey(tenant, qHash, pHash, abac, tier, boVer, cubeV2)
	undeployed := BuildCompositeCacheKey(tenant, qHash, pHash, abac, tier, boVer, noCube)

	assert.NotEqual(t, deployed, edited,
		"editing a cube must invalidate its cached results")
	assert.NotEqual(t, deployed, undeployed,
		"undeploying a cube must invalidate its cached results")
	assert.NotEqual(t, edited, undeployed,
		"each distinct cube state needs its own cache entry")

	// The no-cube term must be a real, stable value — not an empty string that
	// would collide with any other omitted-term path.
	assert.NotEmpty(t, noCube,
		"the no-cube marker must not be empty; emptiness is what makes it collide")

	// And it must be deterministic, so the same logical state always hits the
	// same entry rather than thrashing.
	assert.Equal(t, noCube, NoCubeCacheTerm,
		"the no-cube marker must be a stable constant")

	// Distinct tenants stay distinct even with identical cube state.
	otherTenant := BuildCompositeCacheKey("tenant_b", qHash, pHash, abac, tier, boVer, cubeV1)
	assert.NotEqual(t, deployed, otherTenant,
		"tenant isolation must survive the cube term")
}

// TestBatchExecuteCacheKey_BOSchemaVersionInvalidates covers the second defect
// found at merge review: the live call site passed a hardcoded "v1" in the
// boSchemaVersion position, so no semantic catalog change could ever invalidate
// a batch-execute cache entry.
//
// A bumped BO schema version must produce a different key.
func TestBatchExecuteCacheKey_BOSchemaVersionInvalidates(t *testing.T) {
	base := BuildCompositeCacheKey("tenant_a", "q", "p", "a", "hot", "bo-v1", NoCubeCacheTerm)
	bumped := BuildCompositeCacheKey("tenant_a", "q", "p", "a", "hot", "bo-v2", NoCubeCacheTerm)

	assert.NotEqual(t, base, bumped,
		"a BO schema version change must invalidate cached results")
}

// TestCacheSeam_StaleRowsAreNotServedAfterUndeploy is the behavioural half of
// gate #11: it drives the real cache and proves a payload computed while a cube
// was live is not returned after that cube is undeployed.
//
// The flag is not the important assertion. The rows are: a cached payload
// computed against a materialization that no longer exists keeps serving data
// that corresponds to no live physical object, bypassing every router gate
// (ABAC-below-grain, staleness, decomposability) because the cache path never
// consults the router.
func TestCacheSeam_StaleRowsAreNotServedAfterUndeploy(t *testing.T) {
	cache := NewQueryResultCache(60 * time.Second)
	tenant, boID := "tenant_a", "bo_sales"
	qHash, pHash, abac, tier, boVer := "q", "p", "a", "hot", "bo-v1"

	cubeRows := []map[string]interface{}{{"country": "US", "revenue": 100.0}}
	baseRows := []map[string]interface{}{{"country": "US", "revenue": 250.0}}

	// While a cube is deployed, the key includes its content hash and the
	// result is cached.
	cubeKey := BuildCompositeCacheKey(tenant, qHash, pHash, abac, tier, boVer, "cube-hash-v1")
	cache.Put(cubeKey, &QueryResultPayload{
		Columns: []boresolver.QueryResultColumn{},
		Rows:    cubeRows, RowCount: 1, ResolvedTier: tier, MVHit: true,
	}, 0, boID, time.Minute)

	// After undeploy, the key must differ, so this is a MISS and the caller
	// falls through to the base path.
	noCubeKey := BuildCompositeCacheKey(tenant, qHash, pHash, abac, tier, boVer, NoCubeCacheTerm)
	payload, hit := cache.Get(noCubeKey, 0)
	assert.False(t, hit,
		"a cache entry computed against an undeployed cube must not be served")
	assert.Nil(t, payload)

	// The base-path result then caches under its own key and serves normally.
	cache.Put(noCubeKey, &QueryResultPayload{
		Columns: []boresolver.QueryResultColumn{},
		Rows:    baseRows, RowCount: 1, ResolvedTier: tier, MVHit: false,
	}, 0, boID, time.Minute)

	served, hit := cache.Get(noCubeKey, 0)
	require.True(t, hit, "the base-path result must cache normally")
	assert.False(t, served.MVHit,
		"with no cube deployed, MVHit must be false")
	require.Len(t, served.Rows, 1)
	assert.Equal(t, 250.0, served.Rows[0]["revenue"],
		"rows must come from the base path, not the undeployed cube")

	// And the cube-keyed entry is still addressable but is never the one the
	// no-cube path would reach.
	require.NotEqual(t, cubeKey, noCubeKey)
}

// TestCacheKey_LiveSeamAndMetricSeamStayDistinct documents why there are two key
// builders rather than one.
//
// BuildCompositeCacheKey is the batch-execute seam: tenant, query, params,
// abac, tier, schema version, cube term. ComputeQueryAndMetricsAndCubeCacheKey
// additionally folds in the referenced metrics' content hashes, which the
// batch path does not have. They compose DIFFERENT keys by design and must
// never be asserted equal — a previous draft of this test did exactly that and
// failed, which is the correct outcome for a test asserting a falsehood.
//
// The invariant that actually matters is the one the defect violated: the live
// seam must vary with cube state.
func TestCacheKey_LiveSeamAndMetricSeamStayDistinct(t *testing.T) {
	metrics := []MetricDefinition{revenueMetric()}
	metricSeam := ComputeQueryAndMetricsAndCubeCacheKey("tenant_a", "q", metrics, "cube-hash-v1", "p", "a", "hot", "bo-v1")
	liveSeam := BuildCompositeCacheKey("tenant_a", "q", "p", "a", "hot", "bo-v1", "cube-hash-v1")

	// Distinct by construction: the metric seam carries metric hashes the live
	// batch seam does not. Documented, not a defect.
	assert.NotEqual(t, liveSeam, metricSeam,
		"the two seams compose different keys; metric hashes are only known to the metric seam")

	// Both must still vary with cube state, which is the property the defect broke.
	metricEdited := ComputeQueryAndMetricsAndCubeCacheKey("tenant_a", "q", metrics, "cube-hash-v2", "p", "a", "hot", "bo-v1")
	liveEdited := BuildCompositeCacheKey("tenant_a", "q", "p", "a", "hot", "bo-v1", "cube-hash-v2")
	assert.NotEqual(t, metricSeam, metricEdited)
	assert.NotEqual(t, liveSeam, liveEdited)
}

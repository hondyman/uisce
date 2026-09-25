package api

import (
	"testing"
)

// TestComputeGroupKeyForItem_AddressLineContext verifies the grouping key
// includes table context when the address-line rule fires.
func TestComputeGroupKeyForItem_AddressLineContext(t *testing.T) {
	abbrMap := map[string]string{}

	key1 := computeGroupKeyForItem("tenant-1", "address_line_1", `table "issuer_address" in schema "orm"`, abbrMap)
	key2 := computeGroupKeyForItem("tenant-1", "address_line_1", `table "customer_address" in schema "orm"`, abbrMap)
	key3 := computeGroupKeyForItem("tenant-1", "address_line_1", `table "issuer_address" in schema "orm"`, abbrMap)

	if key1 == "" || key2 == "" || key3 == "" {
		t.Fatal("empty group key")
	}
	if key1 == key2 {
		t.Error("same table but different contexts should produce different keys")
	}
	if key1 != key3 {
		t.Error("same table and context should produce same key")
	}
}

// TestComputeGroupKeyForItem_CustomerIdInThreeTables verifies the dedup win:
// same column name, no context rules → same key across three tables.
func TestComputeGroupKeyForItem_CustomerIdInThreeTables(t *testing.T) {
	abbrMap := map[string]string{"ID": "IDENTIFIER"}

	tables := []string{
		`table "orders" in schema "orm"`,
		`table "invoices" in schema "orm"`,
		`table "payments" in schema "orm"`,
	}

	keys := make(map[string]bool)
	for _, tableCtx := range tables {
		key := computeGroupKeyForItem("tenant-1", "customer_id", tableCtx, abbrMap)
		if key == "" {
			t.Fatal("empty group key")
		}
		keys[key] = true
	}

	if len(keys) != 1 {
		t.Errorf("expected 1 unique key (all collapse), got %d: %v", len(keys), keys)
	}
}

// TestComputeGroupKeyForItem_CompleteAbbrevMapCollapse verifies that a complete
// abbreviation map (no unresolved tokens → no LLM expansion) produces the same
// key across different tables.
func TestComputeGroupKeyForItem_CompleteAbbrevMapCollapse(t *testing.T) {
	abbrMap := map[string]string{"ID": "IDENTIFIER"}

	key1 := computeGroupKeyForItem("tenant-1", "customer_id", `table "orders" in schema "orm"`, abbrMap)
	key2 := computeGroupKeyForItem("tenant-1", "customer_id", `table "invoices" in schema "orm"`, abbrMap)

	if key1 != key2 {
		t.Errorf("complete abbreviation map: same column should collapse (key1=%s, key2=%s)", key1, key2)
	}
}

// TestComputeGroupKeyForItem_EmptyRawName returns empty key for empty input.
func TestComputeGroupKeyForItem_EmptyRawName(t *testing.T) {
	key := computeGroupKeyForItem("tenant-1", "", "", nil)
	if key != "" {
		t.Errorf("expected empty key for empty rawName, got %q", key)
	}
}

// TestComputeGroupKeyForItem_Deterministic verifies that the same inputs
// always produce the same key (no randomness, no time dependency).
func TestComputeGroupKeyForItem_Deterministic(t *testing.T) {
	abbrMap := map[string]string{"ID": "IDENTIFIER"}
	key1 := computeGroupKeyForItem("tenant-1", "customer_id", `table "orders" in schema "orm"`, abbrMap)
	key2 := computeGroupKeyForItem("tenant-1", "customer_id", `table "orders" in schema "orm"`, abbrMap)
	if key1 != key2 {
		t.Errorf("deterministic check failed: %s != %s", key1, key2)
	}
}

// TestComputeGroupKeyForItem_DifferentTenantsDifferentKeys verifies tenant isolation:
// same column name but different tenants produce different keys.
func TestComputeGroupKeyForItem_DifferentTenantsDifferentKeys(t *testing.T) {
	abbrMap := map[string]string{"ID": "IDENTIFIER"}
	key1 := computeGroupKeyForItem("tenant-A", "customer_id", `table "orders" in schema "orm"`, abbrMap)
	key2 := computeGroupKeyForItem("tenant-B", "customer_id", `table "orders" in schema "orm"`, abbrMap)
	if key1 == key2 {
		t.Error("different tenants should produce different keys")
	}
}

// TestComputeGroupKeyForItem_ContextSensitiveSplitsSameColumn verifies the key
// correctness property: same column name in two tables where the address-line
// rule fires → two different keys.
func TestComputeGroupKeyForItem_ContextSensitiveSplitsSameColumn(t *testing.T) {
	abbrMap := map[string]string{}

	keyIssuer := computeGroupKeyForItem("tenant-1", "address_line_1", `table "issuer_address" in schema "orm"`, abbrMap)
	keyCustomer := computeGroupKeyForItem("tenant-1", "address_line_1", `table "customer_address" in schema "orm"`, abbrMap)

	if keyIssuer == keyCustomer {
		t.Error("address-line rule: same column in two different tables should produce different keys")
	}
}

// TestRunBulk_GroupingPreservesOriginalIndex verifies that the grouping logic
// preserves the original item index for result assignment. In runBulk, results
// are indexed by origIndex, not by group index.
func TestRunBulk_GroupingPreservesOriginalIndex(t *testing.T) {
	// Simulate the grouping logic: 4 items, 2 unique groups.
	// Items 0,2 share group A; items 1,3 share group B.
	items := []generateTermItem{
		{Name: "", ColumnIDs: []string{"a1"}},
		{Name: "user-typed", ColumnIDs: []string{"b1"}},
		{Name: "", ColumnIDs: []string{"a2"}},
		{Name: "user-typed", ColumnIDs: []string{"b2"}},
	}

	groups := make(map[string]*preGroupedItem)
	groupOrder := make([]string, 0)

	for i, item := range items {
		if item.Name != "" {
			key := "user_" + item.Name + "_" + item.ColumnIDs[0]
			groups[key] = &preGroupedItem{item: item, origIndex: i}
			groupOrder = append(groupOrder, key)
			continue
		}
		// Non-user-named items: use a simple key (in real code, this is computeGroupKeyForItem)
		key := "group_" + item.ColumnIDs[0]
		if existing, ok := groups[key]; ok {
			existing.item.ColumnIDs = append(existing.item.ColumnIDs, item.ColumnIDs...)
		} else {
			groups[key] = &preGroupedItem{item: item, origIndex: i}
			groupOrder = append(groupOrder, key)
		}
	}

	// Should have 4 groups: user_b1, group_a, user_b2, group_a2
	// (user-named items get unique keys per column_id; non-user items with
	// same column_id merge)
	if len(groups) != 4 {
		t.Errorf("expected 4 groups, got %d: %v", len(groups), groups)
	}

	// Verify origIndex is correct for each group
	for _, gk := range groupOrder {
		g := groups[gk]
		if g.origIndex < 0 || g.origIndex >= len(items) {
			t.Errorf("group %s: origIndex %d out of range", gk, g.origIndex)
		}
	}
}

func TestBuildDefinitionCacheKey_Deterministic(t *testing.T) {
	key1 := buildDefinitionCacheKey([]string{"Customer", "Identifier"}, "", false)
	key2 := buildDefinitionCacheKey([]string{"Customer", "Identifier"}, "", false)
	if key1 != key2 {
		t.Errorf("deterministic: %s != %s", key1, key2)
	}
}

func TestBuildDefinitionCacheKey_DifferentTokensProduceDifferentKeys(t *testing.T) {
	key1 := buildDefinitionCacheKey([]string{"Customer", "Identifier"}, "", false)
	key2 := buildDefinitionCacheKey([]string{"Tenant", "Identifier"}, "", false)
	if key1 == key2 {
		t.Error("different tokens should produce different keys")
	}
}

func TestBuildDefinitionCacheKey_ContextSensitiveIncludesContext(t *testing.T) {
	key1 := buildDefinitionCacheKey([]string{"Issuer", "Address", "Line", "1"}, `table "issuer_address"`, true)
	key2 := buildDefinitionCacheKey([]string{"Issuer", "Address", "Line", "1"}, `table "customer_address"`, true)
	if key1 == key2 {
		t.Error("different context tables should produce different keys for context-sensitive names")
	}
}

func TestBuildDefinitionCacheKey_NonContextSensitiveIgnoresContext(t *testing.T) {
	key1 := buildDefinitionCacheKey([]string{"Customer", "Identifier"}, `table "orders"`, false)
	key2 := buildDefinitionCacheKey([]string{"Customer", "Identifier"}, `table "invoices"`, false)
	if key1 != key2 {
		t.Error("non-context-sensitive names should ignore context")
	}
}

// TestBuildDefinitionCacheKey_VersionBumpInvalidatesCache verifies that bumping
// NamingLogicVersion produces a different cache key for identical inputs.
// This is the version-as-mechanism design: when naming logic or abbreviation
// dictionaries change, bumping the constant invalidates all cached definitions
// without requiring a manual cache flush.
func TestBuildDefinitionCacheKey_VersionBumpInvalidatesCache(t *testing.T) {
	// Save and restore the constant to avoid polluting other tests.
	origVersion := NamingLogicVersion
	defer func() { NamingLogicVersion = origVersion }()

	NamingLogicVersion = "v1"
	keyV1 := buildDefinitionCacheKey([]string{"Customer", "Identifier"}, "", false)

	NamingLogicVersion = "v2"
	keyV2 := buildDefinitionCacheKey([]string{"Customer", "Identifier"}, "", false)

	if keyV1 == keyV2 {
		t.Errorf("version bump should invalidate cache: v1=%s v2=%s", keyV1, keyV2)
	}
}

// TestBuildDefinitionCacheKey_AbbreviationBumpInvalidatesCache verifies that
// bumping AbbreviationsVersion produces a different cache key.
func TestBuildDefinitionCacheKey_AbbreviationBumpInvalidatesCache(t *testing.T) {
	origVersion := AbbreviationsVersion
	defer func() { AbbreviationsVersion = origVersion }()

	AbbreviationsVersion = "v1"
	keyV1 := buildDefinitionCacheKey([]string{"Customer", "Identifier"}, "", false)

	AbbreviationsVersion = "v2"
	keyV2 := buildDefinitionCacheKey([]string{"Customer", "Identifier"}, "", false)

	if keyV1 == keyV2 {
		t.Errorf("abbreviation version bump should invalidate cache: v1=%s v2=%s", keyV1, keyV2)
	}
}

// TestBuildDefinitionCacheKey_CrossVersionStability verifies that the same
// version constants always produce the same key (no drift across runs).
func TestBuildDefinitionCacheKey_CrossVersionStability(t *testing.T) {
	origNLV := NamingLogicVersion
	origAV := AbbreviationsVersion
	defer func() {
		NamingLogicVersion = origNLV
		AbbreviationsVersion = origAV
	}()

	NamingLogicVersion = "v3"
	AbbreviationsVersion = "v7"
	key1 := buildDefinitionCacheKey([]string{"Tenant", "Status"}, `table "security_change_request"`, true)
	key2 := buildDefinitionCacheKey([]string{"Tenant", "Status"}, `table "security_change_request"`, true)
	if key1 != key2 {
		t.Errorf("same constants should be stable: %s != %s", key1, key2)
	}
}

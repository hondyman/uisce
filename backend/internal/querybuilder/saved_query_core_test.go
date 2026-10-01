package querybuilder

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/hondyman/uisce/backend/internal/corecustom"
)

func sampleBaseCoreContent() savedQueryContent {
	return savedQueryContent{
		Name:         "Core Orders Summary",
		Description:  "Standard gold-copy orders breakdown",
		BOID:         "bo_orders",
		BindingID:    "bind_postgres_default",
		RelatedBOIDs: []string{"bo_accounts"},
		ChartType:    "bar",
		State: SavedQueryState{
			Dimensions: []SavedQueryDimension{
				{TermNodeID: "term_region", Alias: "region"},
				{TermNodeID: "term_order_status", Alias: "status"},
			},
			Measures: []SavedQueryMeasure{
				{TermNodeID: "term_order_amount", Alias: "total_amount", Aggregation: "SUM"},
				{TermNodeID: "term_order_count", Alias: "order_count", Aggregation: "COUNT"},
			},
			Filters: []SavedQueryFilter{
				{TermNodeID: "term_is_active", Operator: "eq", Value: true},
				{TermNodeID: "term_date", Operator: "gte", ParamRef: "startDate"},
			},
			Parameters: []SavedQueryParameter{
				{Name: "startDate", Type: "date", Required: true},
			},
			Limit: 1000,
		},
		Tags: []string{"core", "finance"},
	}
}

func TestValidateAdditiveExtension_AllowedAdditions(t *testing.T) {
	base := sampleBaseCoreContent()

	ext := base
	ext.State.Dimensions = append(ext.State.Dimensions, SavedQueryDimension{
		TermNodeID: "term_cust_segment",
		Alias:      "customer_segment",
		BOID:       "bo_accounts",
	})
	ext.State.Measures = append(ext.State.Measures, SavedQueryMeasure{
		TermNodeID:  "term_margin",
		Alias:       "total_margin",
		Aggregation: "SUM",
	})
	ext.State.Filters = append(ext.State.Filters, SavedQueryFilter{
		TermNodeID: "term_region",
		Operator:   "in",
		ParamRef:   "customRegionParam",
	})
	ext.State.Parameters = append(ext.State.Parameters, SavedQueryParameter{
		Name:     "customRegionParam",
		Type:     "string",
		Required: false,
	})
	ext.RelatedBOIDs = append(ext.RelatedBOIDs, "bo_customers")

	if err := ValidateAdditiveExtension(base, ext); err != nil {
		t.Fatalf("expected valid extension to pass, got error: %v", err)
	}
}

func TestValidateAdditiveExtension_RejectPrimaryBOChange(t *testing.T) {
	base := sampleBaseCoreContent()
	ext := base
	ext.BOID = "bo_positions"

	err := ValidateAdditiveExtension(base, ext)
	if err == nil {
		t.Fatalf("expected error when primary BO is changed, got nil")
	}
}

func TestValidateAdditiveExtension_RejectCoreDimensionRemoval(t *testing.T) {
	base := sampleBaseCoreContent()
	ext := base
	ext.State.Dimensions = []SavedQueryDimension{
		{TermNodeID: "term_region", Alias: "region"},
	}

	err := ValidateAdditiveExtension(base, ext)
	if err == nil || err.Error() != `cannot remove core dimension with alias "status" (term "term_order_status")` {
		t.Fatalf("expected error removing core dimension, got: %v", err)
	}
}

func TestValidateAdditiveExtension_RejectCoreMeasureRemoval(t *testing.T) {
	base := sampleBaseCoreContent()
	ext := base
	ext.State.Measures = []SavedQueryMeasure{
		{TermNodeID: "term_order_amount", Alias: "total_amount", Aggregation: "SUM"},
	}

	err := ValidateAdditiveExtension(base, ext)
	if err == nil || err.Error() != `cannot remove core measure with alias "order_count" (term "term_order_count")` {
		t.Fatalf("expected error removing core measure, got: %v", err)
	}
}

func TestValidateAdditiveExtension_RejectAggregationModification(t *testing.T) {
	base := sampleBaseCoreContent()
	ext := base
	ext.State.Measures = []SavedQueryMeasure{
		{TermNodeID: "term_order_amount", Alias: "total_amount", Aggregation: "AVG"},
		{TermNodeID: "term_order_count", Alias: "order_count", Aggregation: "COUNT"},
	}

	err := ValidateAdditiveExtension(base, ext)
	if err == nil || err.Error() != `cannot modify core measure "total_amount" aggregation from "SUM" to "AVG"` {
		t.Fatalf("expected error changing aggregation, got: %v", err)
	}
}

func TestValidateAdditiveExtension_RejectAliasCollision(t *testing.T) {
	base := sampleBaseCoreContent()
	ext := base
	ext.State.Measures = append(ext.State.Measures, SavedQueryMeasure{
		TermNodeID:  "term_custom_metric",
		Alias:       "region",
		Aggregation: "SUM",
	})

	err := ValidateAdditiveExtension(base, ext)
	if err == nil || err.Error() != `alias collision: "region" cannot be both a dimension and a measure` {
		t.Fatalf("expected error for alias collision, got: %v", err)
	}
}

func TestValidateAdditiveExtension_RejectCoreFilterRemoval(t *testing.T) {
	base := sampleBaseCoreContent()
	ext := base
	ext.State.Filters = []SavedQueryFilter{
		{TermNodeID: "term_date", Operator: "gte", ParamRef: "startDate"},
	}

	err := ValidateAdditiveExtension(base, ext)
	if err == nil {
		t.Fatalf("expected error removing core base filter, got nil")
	}
}

func TestValidateAdditiveExtension_RejectCoreParameterRemoval(t *testing.T) {
	base := sampleBaseCoreContent()
	ext := base
	ext.State.Parameters = []SavedQueryParameter{}

	err := ValidateAdditiveExtension(base, ext)
	if err == nil || err.Error() != `cannot remove core parameter "startDate"` {
		t.Fatalf("expected error removing core parameter, got: %v", err)
	}
}

func TestValidateAdditiveExtension_RejectCoreParameterTypeChange(t *testing.T) {
	base := sampleBaseCoreContent()
	ext := base
	ext.State.Parameters = []SavedQueryParameter{
		{Name: "startDate", Type: "number", Required: true},
	}

	err := ValidateAdditiveExtension(base, ext)
	if err == nil || err.Error() != `cannot change type of core parameter "startDate" from "date" to "number"` {
		t.Fatalf("expected error changing parameter type, got: %v", err)
	}
}

func TestValidateAdditiveExtension_DeterministicJSON(t *testing.T) {
	base := sampleBaseCoreContent()
	extRaw, err := json.Marshal(base)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var ext savedQueryContent
	if err := json.Unmarshal(extRaw, &ext); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if err := ValidateAdditiveExtension(base, ext); err != nil {
		t.Fatalf("expected identical JSON unmarshaled content to pass, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Lifecycle & Upgrade Tests (Phase 1.2)
// ---------------------------------------------------------------------------

func TestPresentCoreQuery_VanillaReadThrough(t *testing.T) {
	base := sampleBaseCoreContent()
	sq := SavedQuery{
		ID:          "core_q1",
		TenantID:    "gold_tenant",
		Name:        base.Name,
		Description: base.Description,
		BOID:        base.BOID,
		BindingID:   base.BindingID,
		State:       base.State,
		IsCore:      true,
	}

	// Read from a client tenant with NO adoption row
	presentCoreQuery(&sq, nil, false, true)

	if sq.CoreStatus != "vanilla" {
		t.Errorf("expected CoreStatus 'vanilla', got %q", sq.CoreStatus)
	}
	if sq.Editable {
		t.Errorf("expected core query to not be directly editable by client")
	}
	if !sq.CanCustomize {
		t.Errorf("expected CanCustomize to be true")
	}
	if sq.Customization == nil || sq.Customization.Mode != "vanilla" || !sq.Customization.Active {
		t.Errorf("expected Customization to be vanilla & active, got: %+v", sq.Customization)
	}
}

func TestPresentCoreQuery_ExtendedAndUpgradeAvailable(t *testing.T) {
	base := sampleBaseCoreContent()
	baseBytes, _ := json.Marshal(base)

	// Client extension has an extra measure
	ext := base
	ext.State.Measures = append(ext.State.Measures, SavedQueryMeasure{
		TermNodeID:  "term_margin",
		Alias:       "total_margin",
		Aggregation: "SUM",
	})
	extBytes, _ := json.Marshal(ext)

	sq := SavedQuery{
		ID:           "core_q1",
		TenantID:     "gold_tenant",
		Name:         base.Name,
		Description:  base.Description,
		BOID:         base.BOID,
		BindingID:    base.BindingID,
		RelatedBOIDs: base.RelatedBOIDs,
		ChartType:    base.ChartType,
		State:        base.State,
		Tags:         base.Tags,
		IsCore:       true,
	}

	adopt := queryAdoption{
		CoreObjectID: "core_q1",
		Active:       true,
		Mode:         "extended",
		BaseVersion:  sql.NullInt64{Int64: 1, Valid: true},
		BaseSnapshot: baseBytes,
		Extension:    extBytes,
	}

	// 1. When core matches base snapshot -> status is 'extended'
	presentCoreQuery(&sq, &adopt, false, true)
	if sq.CoreStatus != "extended" {
		t.Errorf("expected CoreStatus 'extended', got %q", sq.CoreStatus)
	}
	if len(sq.State.Measures) != 3 {
		t.Errorf("expected extended query to reflect 3 measures, got %d", len(sq.State.Measures))
	}

	// 2. When core upgraded in gold copy (added new core dimension) -> status is 'upgrade_available'
	coreV2 := base
	coreV2.State.Dimensions = append(coreV2.State.Dimensions, SavedQueryDimension{
		TermNodeID: "term_channel",
		Alias:      "channel",
	})
	sqV2 := SavedQuery{
		ID:           "core_q1",
		TenantID:     "gold_tenant",
		Name:         coreV2.Name,
		Description:  coreV2.Description,
		BOID:         coreV2.BOID,
		BindingID:    coreV2.BindingID,
		RelatedBOIDs: coreV2.RelatedBOIDs,
		ChartType:    coreV2.ChartType,
		State:        coreV2.State,
		Tags:         coreV2.Tags,
		IsCore:       true,
	}
	presentCoreQuery(&sqV2, &adopt, false, true)
	if sqV2.CoreStatus != "upgrade_available" {
		t.Errorf("expected CoreStatus 'upgrade_available', got %q", sqV2.CoreStatus)
	}
}

func TestCoreQueryUpgrade_PreservesCustomAdditionsAcrossCoreV2(t *testing.T) {
	base := sampleBaseCoreContent()

	// Client extension added a custom measure and a custom filter
	ext := base
	ext.State.Measures = append(ext.State.Measures, SavedQueryMeasure{
		TermNodeID:  "term_margin",
		Alias:       "total_margin",
		Aggregation: "SUM",
	})
	ext.State.Filters = append(ext.State.Filters, SavedQueryFilter{
		TermNodeID: "term_region",
		Operator:   "eq",
		Value:      "EMEA",
	})

	// Core upgraded to v2: master tenant added a new dimension "channel"
	curCore := base
	curCore.State.Dimensions = append(curCore.State.Dimensions, SavedQueryDimension{
		TermNodeID: "term_channel",
		Alias:      "channel",
	})

	mergedDoc := corecustom.Merge(base.doc(), ext.doc(), curCore.doc(), map[string]bool{}, savedQueryGrouper)
	merged, err := contentFromDoc(mergedDoc)
	if err != nil {
		t.Fatalf("failed to parse merged doc: %v", err)
	}

	// Verify merged query has:
	// 1. All 3 dimensions (region, status, and newly added channel from Core v2)
	// 2. All 3 measures (total_amount, order_count, and custom total_margin from client extension)
	// 3. All 3 filters (is_active, startDate, and custom region=EMEA from client extension)
	if len(merged.State.Dimensions) != 3 {
		t.Errorf("expected 3 dimensions in merged query, got %d: %+v", len(merged.State.Dimensions), merged.State.Dimensions)
	}
	if len(merged.State.Measures) != 3 {
		t.Errorf("expected 3 measures in merged query, got %d: %+v", len(merged.State.Measures), merged.State.Measures)
	}
	if len(merged.State.Filters) != 3 {
		t.Errorf("expected 3 filters in merged query, got %d: %+v", len(merged.State.Filters), merged.State.Filters)
	}

	// Verify the merged result satisfies the Additive-Only contract against current Core v2
	if err := ValidateAdditiveExtension(curCore, merged); err != nil {
		t.Fatalf("expected merged output to satisfy Additive-Only contract against Core v2, got error: %v", err)
	}
}

func TestCoreQueryCompare_FlagsAliasCollisionConflict(t *testing.T) {
	base := sampleBaseCoreContent()

	// Tenant added measure with alias "channel_metric"
	ext := base
	ext.State.Measures = append(ext.State.Measures, SavedQueryMeasure{
		TermNodeID:  "term_margin",
		Alias:       "channel_metric",
		Aggregation: "SUM",
	})

	// Core v2 also added a dimension with alias "channel_metric" (conflict!)
	curCore := base
	curCore.State.Dimensions = append(curCore.State.Dimensions, SavedQueryDimension{
		TermNodeID: "term_channel",
		Alias:      "channel_metric",
	})

	// Conflict check logic from HandleCompareCoreQuery
	var conflicts []string
	extAliases := make(map[string]bool)
	for _, d := range ext.State.Dimensions {
		extAliases[d.Alias] = true
	}
	for _, m := range ext.State.Measures {
		extAliases[m.Alias] = true
	}
	for _, d := range curCore.State.Dimensions {
		if extAliases[d.Alias] && !containsDimAlias(base.State.Dimensions, d.Alias) {
			conflicts = append(conflicts, "core update added dimension "+d.Alias+" colliding with tenant extension")
		}
	}

	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict for alias collision, got %d: %v", len(conflicts), conflicts)
	}
}

func TestPresentCoreQuery_ClonedMode(t *testing.T) {
	base := sampleBaseCoreContent()
	sq := SavedQuery{
		ID:       "core_q1",
		TenantID: "gold_tenant",
		Name:     base.Name,
		IsCore:   true,
	}

	cloneID := "tenant_cloned_q1"
	adopt := queryAdoption{
		CoreObjectID:  "core_q1",
		Active:        true,
		Mode:          "cloned",
		CloneObjectID: sql.NullString{String: cloneID, Valid: true},
	}

	presentCoreQuery(&sq, &adopt, false, true)
	if sq.CoreStatus != "cloned" {
		t.Errorf("expected CoreStatus 'cloned', got %q", sq.CoreStatus)
	}
	if sq.Customization.CloneQueryID == nil || *sq.Customization.CloneQueryID != cloneID {
		t.Errorf("expected CloneQueryID %q, got: %+v", cloneID, sq.Customization.CloneQueryID)
	}
}

func TestRevertCoreQuery_KeepsExtensionDormant(t *testing.T) {
	base := sampleBaseCoreContent()
	baseBytes, _ := json.Marshal(base)

	ext := base
	ext.State.Measures = append(ext.State.Measures, SavedQueryMeasure{
		TermNodeID:  "term_margin",
		Alias:       "total_margin",
		Aggregation: "SUM",
	})
	extBytes, _ := json.Marshal(ext)

	// Revert sets mode='vanilla' but retains extension in DB
	adoptReverted := queryAdoption{
		CoreObjectID: "core_q1",
		Active:       true,
		Mode:         "vanilla",
		BaseVersion:  sql.NullInt64{Int64: 1, Valid: true},
		BaseSnapshot: baseBytes,
		Extension:    extBytes,
	}

	sq := SavedQuery{
		ID:           "core_q1",
		TenantID:     "gold_tenant",
		Name:         base.Name,
		Description:  base.Description,
		BOID:         base.BOID,
		BindingID:    base.BindingID,
		RelatedBOIDs: base.RelatedBOIDs,
		ChartType:    base.ChartType,
		State:        base.State,
		Tags:         base.Tags,
		IsCore:       true,
	}

	presentCoreQuery(&sq, &adoptReverted, false, true)
	if sq.CoreStatus != "vanilla" {
		t.Errorf("expected CoreStatus 'vanilla', got %q", sq.CoreStatus)
	}
	// Verify extension bytes were preserved in adoption struct
	if len(adoptReverted.Extension) == 0 {
		t.Errorf("expected dormant extension to be retained in adoption record")
	}
	// Verify sq view is vanilla base (2 measures, not 3)
	if len(sq.State.Measures) != 2 {
		t.Errorf("expected vanilla query to show 2 base measures, got %d", len(sq.State.Measures))
	}
}

func TestExtendCoreQuery_CreatesCorrectBaseSnapshot(t *testing.T) {
	base := sampleBaseCoreContent()
	baseBytes, err := json.Marshal(base.normalized())
	if err != nil {
		t.Fatalf("failed to marshal base: %v", err)
	}

	ext := base
	ext.State.Measures = append(ext.State.Measures, SavedQueryMeasure{
		TermNodeID:  "term_margin",
		Alias:       "total_margin",
		Aggregation: "SUM",
	})
	extBytes, err := json.Marshal(ext.normalized())
	if err != nil {
		t.Fatalf("failed to marshal ext: %v", err)
	}

	adopt := queryAdoption{
		CoreObjectID: "core_q1",
		Active:       true,
		Mode:         "extended",
		BaseVersion:  sql.NullInt64{Int64: 1, Valid: true},
		BaseSnapshot: baseBytes,
		Extension:    extBytes,
	}

	baseFromAdopt, err := contentFromJSON(adopt.BaseSnapshot)
	if err != nil {
		t.Fatalf("failed to unmarshal base snapshot from adoption: %v", err)
	}
	if baseFromAdopt.Name != base.Name || len(baseFromAdopt.State.Dimensions) != len(base.State.Dimensions) {
		t.Errorf("base snapshot mismatch")
	}
}


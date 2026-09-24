package api

import (
	"testing"
)

// Tests for validateReportGenerationSpec - GenerateReportSpec's
// anti-hallucination pass, extracted so it's testable without a live (or
// mocked) Gemini call. See HANDOFF_REPORT_BUILDER_SPINE_PLAN.md Phase 6.1.

func testFields() []ReportGenerationField {
	return []ReportGenerationField{
		{TermNodeID: "term-region", Key: "region", DisplayName: "Region", DataType: "string", Role: "DIMENSION"},
		{TermNodeID: "term-status", Key: "status", DisplayName: "Status", DataType: "string", Role: "DIMENSION"},
		{TermNodeID: "term-date", Key: "date", DisplayName: "Date", DataType: "date", Role: "DIMENSION"},
		{TermNodeID: "term-owner", Key: "owner", DisplayName: "Owner", DataType: "string", Role: "DIMENSION"},
		{TermNodeID: "term-revenue", Key: "revenue", DisplayName: "Revenue", DataType: "number", Role: "MEASURE"},
		{TermNodeID: "term-qty", Key: "qty", DisplayName: "Quantity", DataType: "number", Role: "MEASURE"},
	}
}

func testRelatedBOs() []ReportGenerationRelatedBO {
	return []ReportGenerationRelatedBO{
		{
			BOID: "rel-1", BOKey: "order_allocation", DisplayName: "Order Allocation",
			RelationshipType: "has_many", Cardinality: "1:N",
			Fields: []ReportGenerationField{
				{TermNodeID: "term-alloc-qty", Key: "alloc_qty", DisplayName: "Allocated Qty", DataType: "number", Role: "MEASURE"},
			},
		},
	}
}

func TestValidateReportGenerationSpec_HallucinatedBOKeyRemapsToPrimary(t *testing.T) {
	spec := &ReportGenerationSpec{
		Elements: []ReportGenerationElement{
			{BOKey: "not_a_real_bo", Type: "table", Dimensions: []string{"term-region"}},
		},
	}
	out := validateReportGenerationSpec(spec, "Order", "dashboard", testFields(), testRelatedBOs())
	if len(out.Elements) != 1 {
		t.Fatalf("expected 1 element to survive, got %d", len(out.Elements))
	}
	if out.Elements[0].BOKey != "" {
		t.Errorf("expected hallucinated boKey to remap to primary (\"\"), got %q", out.Elements[0].BOKey)
	}
}

func TestValidateReportGenerationSpec_HallucinatedTermNodeIDDropped(t *testing.T) {
	spec := &ReportGenerationSpec{
		Elements: []ReportGenerationElement{
			{BOKey: "", Type: "table", Dimensions: []string{"term-region", "term-invented-nonsense"}, Measures: []string{"term-revenue"}},
		},
	}
	out := validateReportGenerationSpec(spec, "Order", "dashboard", testFields(), testRelatedBOs())
	dims := out.Elements[0].Dimensions
	if len(dims) != 1 || dims[0] != "term-region" {
		t.Errorf("expected only the real termNodeId to survive, got %v", dims)
	}
	if len(out.Elements[0].Measures) != 1 || out.Elements[0].Measures[0] != "term-revenue" {
		t.Errorf("expected the real measure to survive untouched, got %v", out.Elements[0].Measures)
	}
}

func TestValidateReportGenerationSpec_RelatedBOTermsValidatedAgainstThatBOsOwnFieldSet(t *testing.T) {
	spec := &ReportGenerationSpec{
		Elements: []ReportGenerationElement{
			// A related-BO element trying to use the PRIMARY's termNodeId -
			// must be dropped, since term-revenue isn't in order_allocation's
			// own field set.
			{BOKey: "order_allocation", Type: "table", Measures: []string{"term-revenue", "term-alloc-qty"}},
		},
	}
	out := validateReportGenerationSpec(spec, "Order", "dashboard", testFields(), testRelatedBOs())
	measures := out.Elements[0].Measures
	if len(measures) != 1 || measures[0] != "term-alloc-qty" {
		t.Errorf("expected only order_allocation's own real measure to survive, got %v", measures)
	}
}

func TestValidateReportGenerationSpec_OverCountDimensionsTruncatedToPerTypeCap(t *testing.T) {
	spec := &ReportGenerationSpec{
		Elements: []ReportGenerationElement{
			// table/matrix/list cap at 4 dimensions - only 4 real dims exist
			// in testFields() anyway, so also assert measures cap at 2.
			{BOKey: "", Type: "table", Dimensions: []string{"term-region", "term-status", "term-date", "term-owner"}, Measures: []string{"term-revenue", "term-qty"}},
			// slicer caps at 1 dimension, 0 measures.
			{BOKey: "", Type: "slicer", Dimensions: []string{"term-region", "term-status"}, Measures: []string{"term-revenue"}},
		},
	}
	out := validateReportGenerationSpec(spec, "Order", "dashboard", testFields(), nil)
	if len(out.Elements[0].Dimensions) != 4 {
		t.Errorf("expected table to keep all 4 dimensions (at cap), got %d", len(out.Elements[0].Dimensions))
	}
	if len(out.Elements[0].Measures) != 2 {
		t.Errorf("expected table to keep both measures (at cap), got %d", len(out.Elements[0].Measures))
	}
	slicer := out.Elements[1]
	if len(slicer.Dimensions) != 1 {
		t.Errorf("expected slicer capped to 1 dimension, got %d", len(slicer.Dimensions))
	}
	if len(slicer.Measures) != 0 {
		t.Errorf("expected slicer to have 0 measures, got %d", len(slicer.Measures))
	}
}

func TestValidateReportGenerationSpec_GaugePrefersMeasureOverDimension(t *testing.T) {
	spec := &ReportGenerationSpec{
		Elements: []ReportGenerationElement{
			{BOKey: "", Type: "gauge", Dimensions: []string{"term-region"}, Measures: []string{"term-revenue", "term-qty"}},
		},
	}
	out := validateReportGenerationSpec(spec, "Order", "dashboard", testFields(), nil)
	g := out.Elements[0]
	if len(g.Measures) != 1 || g.Measures[0] != "term-revenue" {
		t.Errorf("expected gauge to keep exactly 1 measure, got %v", g.Measures)
	}
	if len(g.Dimensions) != 0 {
		t.Errorf("expected gauge to drop dimensions when a measure is present, got %v", g.Dimensions)
	}
}

func TestValidateReportGenerationSpec_GaugeFallsBackToDimensionWhenNoMeasure(t *testing.T) {
	spec := &ReportGenerationSpec{
		Elements: []ReportGenerationElement{
			{BOKey: "", Type: "gauge", Dimensions: []string{"term-region", "term-status"}},
		},
	}
	out := validateReportGenerationSpec(spec, "Order", "dashboard", testFields(), nil)
	g := out.Elements[0]
	if len(g.Dimensions) != 1 || g.Dimensions[0] != "term-region" {
		t.Errorf("expected gauge to fall back to exactly 1 dimension, got %v", g.Dimensions)
	}
}

func TestValidateReportGenerationSpec_UnknownTypeDropped(t *testing.T) {
	spec := &ReportGenerationSpec{
		Elements: []ReportGenerationElement{
			{BOKey: "", Type: "textbox", Title: "not data-bound, should be dropped"},
			{BOKey: "", Type: "table", Dimensions: []string{"term-region"}},
		},
	}
	out := validateReportGenerationSpec(spec, "Order", "dashboard", testFields(), nil)
	if len(out.Elements) != 1 {
		t.Fatalf("expected the unknown type to be dropped, leaving 1 element, got %d", len(out.Elements))
	}
	if out.Elements[0].Type != "table" {
		t.Errorf("expected the surviving element to be the table, got %q", out.Elements[0].Type)
	}
}

func TestValidateReportGenerationSpec_ElementCountCappedAtSix(t *testing.T) {
	elements := make([]ReportGenerationElement, 0, 10)
	for i := 0; i < 10; i++ {
		elements = append(elements, ReportGenerationElement{BOKey: "", Type: "table", Dimensions: []string{"term-region"}})
	}
	spec := &ReportGenerationSpec{Elements: elements}
	out := validateReportGenerationSpec(spec, "Order", "dashboard", testFields(), nil)
	if len(out.Elements) != 6 {
		t.Errorf("expected element count capped at 6, got %d", len(out.Elements))
	}
}

func TestValidateReportGenerationSpec_DefaultsTitleAndReportKind(t *testing.T) {
	spec := &ReportGenerationSpec{
		Elements: []ReportGenerationElement{{BOKey: "", Type: "table", Dimensions: []string{"term-region"}}},
	}
	out := validateReportGenerationSpec(spec, "Order", "list", testFields(), nil)
	if out.Title != "Order" {
		t.Errorf("expected empty title to default to boName, got %q", out.Title)
	}
	if out.ReportKind != "list" {
		t.Errorf("expected empty/invalid reportKind to default to the requested reportKind, got %q", out.ReportKind)
	}
}

func TestValidateReportGenerationSpec_InvalidReportKindFallsBackToDashboard(t *testing.T) {
	spec := &ReportGenerationSpec{
		ReportKind: "not-a-real-kind",
		Elements:   []ReportGenerationElement{{BOKey: "", Type: "table", Dimensions: []string{"term-region"}}},
	}
	out := validateReportGenerationSpec(spec, "Order", "also-not-real", testFields(), nil)
	if out.ReportKind != "dashboard" {
		t.Errorf("expected invalid reportKind (with no valid fallback) to default to dashboard, got %q", out.ReportKind)
	}
}

// Tests for deterministicReportSpec - the zero-LLM-calls fallback used
// whenever geminiClient is nil, errors, or returns zero usable elements.

func TestDeterministicReportSpec_List(t *testing.T) {
	fields := []reportGenerationField{
		{TermNodeID: "t1", Role: "DIMENSION"},
		{TermNodeID: "t2", Role: "MEASURE"},
	}
	related := []reportGenerationRelatedBO{{BOKey: "order_allocation", DisplayName: "Order Allocation"}}
	title, elements := deterministicReportSpec("list", "Order", fields, related)
	if title != "Order List" {
		t.Errorf("expected title 'Order List', got %q", title)
	}
	if len(elements) == 0 {
		t.Fatal("expected at least one element")
	}
	if elements[0].Type != "table" {
		t.Errorf("expected first element to be a table, got %q", elements[0].Type)
	}
	foundSlicer := false
	foundChild := false
	for _, e := range elements {
		if e.Type == "slicer" {
			foundSlicer = true
		}
		if e.BOKey == "order_allocation" {
			foundChild = true
		}
	}
	if !foundSlicer {
		t.Error("expected a slicer element since dimensions exist")
	}
	if !foundChild {
		t.Error("expected a related-BO child table")
	}
}

func TestDeterministicReportSpec_Detail(t *testing.T) {
	title, elements := deterministicReportSpec("detail", "Order", nil, nil)
	if title != "Order Detail" {
		t.Errorf("expected title 'Order Detail', got %q", title)
	}
	if len(elements) != 1 || elements[0].Type != "form" {
		t.Errorf("expected exactly one form element, got %+v", elements)
	}
}

func TestDeterministicReportSpec_MasterDetail(t *testing.T) {
	_, elements := deterministicReportSpec("master-detail", "Order", nil, nil)
	if len(elements) != 2 {
		t.Fatalf("expected exactly 2 elements (table + form), got %d", len(elements))
	}
	if elements[0].Type != "table" || elements[1].Type != "form" {
		t.Errorf("expected table then form, got %q then %q", elements[0].Type, elements[1].Type)
	}
}

func TestDeterministicReportSpec_Dashboard(t *testing.T) {
	fields := []reportGenerationField{
		{TermNodeID: "t1", Role: "DIMENSION"},
		{TermNodeID: "t2", Role: "MEASURE"},
	}
	title, elements := deterministicReportSpec("dashboard", "Order", fields, nil)
	if title != "Order" {
		t.Errorf("expected bare boName as title, got %q", title)
	}
	types := make([]string, len(elements))
	for i, e := range elements {
		types[i] = e.Type
	}
	hasGauge, hasChart, hasTable := false, false, false
	for _, ty := range types {
		switch ty {
		case "gauge":
			hasGauge = true
		case "chart":
			hasChart = true
		case "table":
			hasTable = true
		}
	}
	if !hasGauge || !hasChart || !hasTable {
		t.Errorf("expected gauge+chart+table mix when measures and dimensions both exist, got %v", types)
	}
}

func TestDeterministicReportSpec_DashboardNoMeasures(t *testing.T) {
	fields := []reportGenerationField{{TermNodeID: "t1", Role: "DIMENSION"}}
	_, elements := deterministicReportSpec("dashboard", "Order", fields, nil)
	for _, e := range elements {
		if e.Type == "gauge" || e.Type == "chart" {
			t.Errorf("expected no gauge/chart without measures, got %q", e.Type)
		}
	}
}

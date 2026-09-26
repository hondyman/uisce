package attribute

import (
	"strings"
	"testing"
)

func TestNormalizeFieldCd(t *testing.T) {
	got, err := normalizeFieldCd("SFDR Article", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "sfdr_article" {
		t.Fatalf("got %q", got)
	}
	got, err = normalizeFieldCd("X", "tax_alignment_pct")
	if err != nil || got != "tax_alignment_pct" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := normalizeFieldCd("X", "Bad-Code"); err == nil {
		t.Fatal("expected error for unsafe field_cd")
	}
}

func TestSanitizeTableRef(t *testing.T) {
	schema, table, err := sanitizeTableRef("crims.mdm.product")
	if err != nil {
		t.Fatal(err)
	}
	if schema != "mdm" || table != "product" {
		t.Fatalf("got %s.%s", schema, table)
	}
	if _, _, err := sanitizeTableRef("product"); err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildCustomColumnExpr(t *testing.T) {
	expr := buildCustomColumnExpr("t", AttributeDef{FieldCd: "sfdr_article", DataType: "integer"})
	if !strings.Contains(expr, "mdm.safe_int") || !strings.Contains(expr, "'sfdr_article'") {
		t.Fatalf("unexpected expr: %s", expr)
	}
}

func TestBuildPreviewQuery(t *testing.T) {
	q, args, err := buildPreviewQuery(
		PreviewRequest{TenantID: [16]byte{}, Limit: 10, Offset: 0},
		"mdm.product",
		[]AttributeDef{{FieldCd: "sfdr_article", DataType: "integer", Name: "SFDR"}},
		[]CoreColumn{{Name: "id", DataType: "uuid", Label: "Id"}, {Name: "product_cd", DataType: "text", Label: "Product Cd"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(q, `"mdm"."product"`) {
		t.Fatalf("missing table: %s", q)
	}
	if !strings.Contains(q, "safe_int") {
		t.Fatalf("missing cast: %s", q)
	}
	if len(args) != 3 {
		t.Fatalf("args=%d", len(args))
	}
}

func TestNormalizeCatalogTablePath(t *testing.T) {
	ref, schema, table := normalizeCatalogTablePath("crims.mdm.product")
	if ref != "mdm.product" || schema != "mdm" || table != "product" {
		t.Fatalf("got %s %s %s", ref, schema, table)
	}
}

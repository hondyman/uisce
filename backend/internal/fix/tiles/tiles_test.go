package tiles

import (
	"context"
	"encoding/base64"
	"reflect"
	"testing"
)

func TestFixDecode_ParsesValidMessage(t *testing.T) {
	raw := "8=FIX.4.4\x019=148\x0135=D\x0149=BUYER\x0156=SELLER\x0134=2\x0152=20260913-10:00:00.000\x0110=200\x01"
	rec := Record{
		"raw_bytes": base64.StdEncoding.EncodeToString([]byte(raw)),
		"tenant_id": "00000000-0000-0000-0000-000000000001",
	}
	out, errs, err := FixDecode(context.Background(), []Record{rec})
	if err != nil {
		t.Fatalf("FixDecode returned error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("FixDecode emitted errors: %v", errs)
	}
	if len(out) != 1 {
		t.Fatalf("FixDecode returned %d records, want 1", len(out))
	}
	tags, ok := out[0]["tags"].(map[string]string)
	if !ok {
		t.Fatalf("tags missing or wrong type: %T", out[0]["tags"])
	}
	if tags["35"] != "D" {
		t.Fatalf("tag 35 (MsgType) = %q, want D", tags["35"])
	}
	if out[0]["msg_type"] != "D" {
		t.Fatalf("msg_type = %v, want D", out[0]["msg_type"])
	}
}

func TestFixDecode_RejectsMissingRequiredTags(t *testing.T) {
	raw := "8=FIX.4.4\x019=99\x0135=D\x0149=BUYER\x0156=SELLER\x0134=2\x0152=20260913-10:00:00.000\x01"
	rec := Record{"raw_bytes": base64.StdEncoding.EncodeToString([]byte(raw))}
	_, errs, err := FixDecode(context.Background(), []Record{rec})
	if err != nil {
		t.Fatalf("FixDecode returned error: %v", err)
	}
	if len(errs) == 0 {
		t.Fatalf("FixDecode accepted message missing CheckSum; expected errors")
	}
}

func TestFixTagMap_AppliesTenantMappings(t *testing.T) {
	loader := &FakeTagMappingLoader{
		Mappings: map[string][]TagMapping{
			"FIX.4.4|D": {
				{FixTag: 11, SemanticField: "external_order_id", Required: true},
				{FixTag: 55, SemanticField: "symbol", Required: true, TransformFn: "upper"},
				{FixTag: 54, SemanticField: "side"},
				{FixTag: 38, SemanticField: "quantity", TransformFn: "numeric"},
				{FixTag: 44, SemanticField: "price", TransformFn: "numeric"},
			},
		},
	}

	records := []Record{
		{
			"msg_type": "D",
			"tags": map[string]string{
				"11": "ORD-001",
				"55": "aapl",
				"54": "1",
				"38": "100",
				"44": "178.50",
			},
		},
	}

	ctx := WithTenant(context.Background(), TenantContext{TenantID: "t1", BrokerID: "b1"})
	out, errs, err := FixTagMap(loader)(ctx, records)
	if err != nil {
		t.Fatalf("FixTagMap: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("FixTagMap errors: %v", errs)
	}
	if len(out) != 1 {
		t.Fatalf("FixTagMap returned %d records, want 1", len(out))
	}
	sem, ok := out[0]["semantic"].(map[string]any)
	if !ok {
		t.Fatalf("semantic field missing")
	}
	if sem["external_order_id"] != "ORD-001" {
		t.Fatalf("external_order_id: %v", sem["external_order_id"])
	}
	if sem["symbol"] != "AAPL" {
		t.Fatalf("symbol (upper): %v", sem["symbol"])
	}
	if sem["quantity"] != 100.0 {
		t.Fatalf("quantity (numeric): %v", sem["quantity"])
	}
}

func TestFixTagMap_TenantOverridesGoldCopy(t *testing.T) {
	loader := &FakeTagMappingLoader{
		Mappings: map[string][]TagMapping{
			"FIX.4.4|D": {
				{FixTag: 11, SemanticField: "external_order_id", TenantOwned: false},
				{FixTag: 11, SemanticField: "client_order_ref", TenantOwned: true},
			},
		},
	}

	ctx := WithTenant(context.Background(), TenantContext{TenantID: "t1", BrokerID: "b1"})
	records := []Record{
		{"msg_type": "D", "tags": map[string]string{"11": "ORD-001"}},
	}
	out, _, _ := FixTagMap(loader)(ctx, records)
	sem := out[0]["semantic"].(map[string]any)
	if sem["client_order_ref"] != "ORD-001" {
		t.Fatalf("tenant override not applied; semantic: %+v", sem)
	}
	if _, has := sem["external_order_id"]; has {
		t.Fatalf("gold-copy mapping survived; should have been overridden: %+v", sem)
	}
}

func TestFixOrderEmit_ReverseMapping(t *testing.T) {
	loader := &FakeTagMappingLoader{
		Mappings: map[string][]TagMapping{
			"FIX.4.4|D": {
				{FixTag: 11, SemanticField: "external_order_id"},
				{FixTag: 55, SemanticField: "symbol"},
				{FixTag: 38, SemanticField: "quantity"},
			},
		},
	}

	records := []Record{
		{
			"semantic": map[string]any{
				"external_order_id": "ORD-001",
				"symbol":            "AAPL",
				"quantity":          float64(100),
			},
		},
	}

	ctx := WithTenant(context.Background(), TenantContext{TenantID: "t1", BrokerID: "b1"})
	out, errs, err := FixOrderEmit(loader, "D")(ctx, records)
	if err != nil {
		t.Fatalf("FixOrderEmit: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("FixOrderEmit errors: %v", errs)
	}
	if len(out) != 1 {
		t.Fatalf("FixOrderEmit returned %d records, want 1", len(out))
	}
	tags, ok := out[0]["tags"].(map[string]string)
	if !ok {
		t.Fatalf("tags missing")
	}
	want := map[string]string{"11": "ORD-001", "55": "AAPL", "38": "100"}
	if !reflect.DeepEqual(tags, want) {
		t.Fatalf("reverse mapping mismatch: got %+v, want %+v", tags, want)
	}
	if out[0]["direction"] != "outbound" {
		t.Fatalf("direction: %v", out[0]["direction"])
	}
}

func TestFixOrderEmit_FallbackIdempotencyKey(t *testing.T) {
	loader := &FakeTagMappingLoader{
		Mappings: map[string][]TagMapping{
			"FIX.4.4|D": {{FixTag: 55, SemanticField: "symbol"}},
		},
	}

	records := []Record{
		{"semantic": map[string]any{"symbol": "AAPL"}},
	}
	ctx := WithTenant(context.Background(), TenantContext{TenantID: "t1", BrokerID: "b1"})
	out, _, _ := FixOrderEmit(loader, "D")(ctx, records)

	key, ok := out[0]["cl_ord_id"].(string)
	if !ok || len(key) != 16 {
		t.Fatalf("expected 16-char SHA-256 fallback, got %v", out[0]["cl_ord_id"])
	}
}


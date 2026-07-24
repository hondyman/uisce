package starlib

import "testing"

func TestProjectRecord_EmptyPaths_ReturnsOriginal(t *testing.T) {
	in := map[string]interface{}{"a": 1}
	out := ProjectRecord(in, nil)
	out["b"] = 2
	if in["b"] != 2 {
		t.Fatalf("expected same map instance when no paths")
	}
}

func TestProjectRecord_SubsetNested(t *testing.T) {
	in := map[string]interface{}{
		"page": map[string]interface{}{
			"x": 1,
			"y": 2,
		},
		"account": map[string]interface{}{
			"account_type": "ADVISORY",
			"aum":          123.45,
		},
		"positions": []interface{}{map[string]interface{}{"id": 1}},
	}

	out := ProjectRecord(in, []string{"account.account_type", "page.y"})

	acct, ok := out["account"].(map[string]interface{})
	if !ok {
		t.Fatalf("account missing or wrong type: %#v", out["account"])
	}
	if acct["account_type"] != "ADVISORY" {
		t.Fatalf("account.account_type=%v", acct["account_type"])
	}
	if _, ok := acct["aum"]; ok {
		t.Fatalf("expected account.aum to be omitted")
	}

	page, ok := out["page"].(map[string]interface{})
	if !ok {
		t.Fatalf("page missing or wrong type: %#v", out["page"])
	}
	if page["y"] != 2 {
		t.Fatalf("page.y=%v", page["y"])
	}
	if _, ok := page["x"]; ok {
		t.Fatalf("expected page.x to be omitted")
	}

	if _, ok := out["positions"]; ok {
		t.Fatalf("expected positions to be omitted")
	}
}

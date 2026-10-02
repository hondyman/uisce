package corecustom

import (
	"encoding/json"
	"testing"
)

func doc(t *testing.T, s string) any {
	t.Helper()
	v, err := Decode([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func js(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// byElement groups by the first element-ish step: components.<id>,
// tabs[<id>], or a member id.
func byElement(p Path, _ ...any) (string, string, string) {
	for i, s := range p {
		if s.Kind == SegMember || s.Kind == SegElem {
			return s.Value, "elem", s.Value
		}
		if s.Kind == SegKey && i == 1 && p[0].Value == "components" {
			return s.Value, "component", s.Value
		}
	}
	return p.String(), "other", p.String()
}

const base = `{
  "name": "Mastering",
  "components": {"a": {"type": "DataGrid", "props": {"title": "Golden", "size": 1}}},
  "layout": {"root": "r", "nodes": {"r": {"id": "r", "type": "Row", "children": ["a"]}}},
  "tabs": [{"id": "t1", "label": "One"}, {"id": "t2", "label": "Two"}]
}`

func TestDiff_NumbersCompareByValue(t *testing.T) {
	if ch := Diff(doc(t, `{"n": 1, "f": 1.50}`), doc(t, `{"n": 1, "f": 1.50}`)); len(ch) != 0 {
		t.Fatalf("expected no changes, got %s", js(ch))
	}
}

func TestDiff_AddedWidgetIsTwoChangesNotAnIndexShift(t *testing.T) {
	tenant := `{
	  "name": "Mastering",
	  "components": {"a": {"type": "DataGrid", "props": {"title": "Golden", "size": 1}}, "b": {"type": "Text"}},
	  "layout": {"root": "r", "nodes": {"r": {"id": "r", "type": "Row", "children": ["a", "b"]}}},
	  "tabs": [{"id": "t1", "label": "One"}, {"id": "t2", "label": "Two"}]
	}`
	ch := Diff(doc(t, base), doc(t, tenant))
	if len(ch) != 2 {
		t.Fatalf("want 2 changes, got %s", js(ch))
	}
	got := map[string]Op{}
	for _, c := range ch {
		got[c.Path.String()] = c.Op
	}
	if got["components.b"] != OpAdd || got["layout.nodes.r.children{b}"] != OpAdd {
		t.Fatalf("unexpected changes %v", got)
	}
}

// Upgrade: the core changed one widget and added another; the tenant added
// their own. Keeping everything yields the new core plus the tenant widget,
// placed after the sibling it followed.
func TestMerge_CarriesCustomizationOntoNewCore(t *testing.T) {
	tenant := `{
	  "name": "Mastering",
	  "components": {"a": {"type": "DataGrid", "props": {"title": "Golden", "size": 1}}, "mine": {"type": "Text"}},
	  "layout": {"root": "r", "nodes": {"r": {"id": "r", "type": "Row", "children": ["a", "mine"]}}},
	  "tabs": [{"id": "t1", "label": "One"}, {"id": "t2", "label": "Two"}]
	}`
	core := `{
	  "name": "Mastering",
	  "components": {"a": {"type": "DataGrid", "props": {"title": "Golden records", "size": 1}}, "c": {"type": "Chart"}},
	  "layout": {"root": "r", "nodes": {"r": {"id": "r", "type": "Row", "children": ["a", "c"]}}},
	  "tabs": [{"id": "t1", "label": "One"}, {"id": "t2", "label": "Two"}]
	}`
	b, tn, c := doc(t, base), doc(t, tenant), doc(t, core)
	rep := Compare(b, tn, c, byElement)
	if len(rep.Customizations) != 1 || rep.Customizations[0].ID != "mine" || rep.Customizations[0].Conflict {
		t.Fatalf("customizations: %s", js(rep.Customizations))
	}
	if rep.Customizations[0].Summary != "added" {
		t.Fatalf("summary %q", rep.Customizations[0].Summary)
	}
	if len(rep.CoreUpdates) != 2 {
		t.Fatalf("core updates: %s", js(rep.CoreUpdates))
	}

	kept := Merge(b, tn, c, nil, byElement)
	want := doc(t, `{
	  "name": "Mastering",
	  "components": {"a": {"type": "DataGrid", "props": {"title": "Golden records", "size": 1}}, "c": {"type": "Chart"}, "mine": {"type": "Text"}},
	  "layout": {"root": "r", "nodes": {"r": {"id": "r", "type": "Row", "children": ["a", "mine", "c"]}}},
	  "tabs": [{"id": "t1", "label": "One"}, {"id": "t2", "label": "Two"}]
	}`)
	if !Equal(kept, want) {
		t.Fatalf("kept:\n%s\nwant:\n%s", js(kept), js(want))
	}

	removed := Merge(b, tn, c, map[string]bool{"mine": true}, byElement)
	if !Equal(removed, c) {
		t.Fatalf("removing every customization must give the core: %s", js(removed))
	}
}

func TestCompare_ConflictAndWhoWins(t *testing.T) {
	tenant := `{"components": {"a": {"props": {"title": "Mine"}}}}`
	core := `{"components": {"a": {"props": {"title": "Theirs"}}}}`
	b, tn, c := doc(t, `{"components": {"a": {"props": {"title": "Base"}}}}`), doc(t, tenant), doc(t, core)
	rep := Compare(b, tn, c, byElement)
	if len(rep.Customizations) != 1 || !rep.Customizations[0].Conflict {
		t.Fatalf("expected a conflict: %s", js(rep))
	}
	if got := js(Merge(b, tn, c, nil, byElement)); got != `{"components":{"a":{"props":{"title":"Mine"}}}}` {
		t.Fatalf("keep: tenant wins, got %s", got)
	}
	if got := js(Merge(b, tn, c, map[string]bool{"a": true}, byElement)); got != core2(core) {
		t.Fatalf("remove: core wins, got %s", got)
	}
}

func core2(s string) string {
	v, _ := Decode([]byte(s))
	return js(v)
}

// The core deleted a widget the tenant had restyled; keeping the
// customization brings back the tenant's whole widget, not a fragment.
func TestMerge_CoreRemovedCustomizedElement(t *testing.T) {
	b := doc(t, `{"components": {"a": {"type": "Grid", "props": {"title": "Base"}}}}`)
	tn := doc(t, `{"components": {"a": {"type": "Grid", "props": {"title": "Mine"}}}}`)
	c := doc(t, `{"components": {}}`)
	rep := Compare(b, tn, c, byElement)
	if !rep.Customizations[0].Conflict {
		t.Fatalf("expected conflict: %s", js(rep))
	}
	if got := js(Merge(b, tn, c, nil, byElement)); got != `{"components":{"a":{"props":{"title":"Mine"},"type":"Grid"}}}` {
		t.Fatalf("got %s", got)
	}
}

func TestCompare_CustomizationAdoptedByCore(t *testing.T) {
	b := doc(t, `{"tabs": [{"id": "t1", "label": "One"}]}`)
	tn := doc(t, `{"tabs": [{"id": "t1", "label": "Uno"}]}`)
	c := doc(t, `{"tabs": [{"id": "t1", "label": "Uno"}]}`)
	rep := Compare(b, tn, c, byElement)
	if !rep.Customizations[0].InCore || rep.Customizations[0].Conflict {
		t.Fatalf("expected inCore without conflict: %s", js(rep))
	}
}

func TestMerge_KeyedArrayAddAndReorder(t *testing.T) {
	b := doc(t, `{"tabs": [{"id": "t1"}, {"id": "t2"}]}`)
	tn := doc(t, `{"tabs": [{"id": "t2"}, {"id": "t1"}, {"id": "mine"}]}`)
	c := doc(t, `{"tabs": [{"id": "t1"}, {"id": "t2"}, {"id": "t3"}]}`)
	got := js(Merge(b, tn, c, nil, byElement))
	if got != `{"tabs":[{"id":"t2"},{"id":"t1"},{"id":"mine"},{"id":"t3"}]}` {
		t.Fatalf("got %s", got)
	}
}

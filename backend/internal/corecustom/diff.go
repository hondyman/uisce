// Package corecustom tracks how a tenant has customized a core (gold-copy)
// object and carries those customizations across core upgrades.
//
// A core object is a JSON document. A tenant that extends it keeps three
// documents: the core as it was when they extended it (base), their own
// extended copy (tenant), and whatever the core is now (core). Diff(base,
// tenant) is the tenant's customizations; Diff(base, core) is what the
// gold copy changed since. Compare pairs the two up and flags conflicts;
// Merge replays the customizations the tenant keeps onto the new core.
//
// Nothing here knows about pages. Arrays of objects are matched by a
// stable key (id, then key, then name) and arrays of scalars by
// membership, so "added a widget" is one change rather than a rewrite of
// every later array index. A Grouper supplied by the object type turns
// low-level changes into the units a person decides about.
package corecustom

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// SegKind says how a path segment addresses its parent.
type SegKind string

const (
	SegKey    SegKind = "key"    // object field
	SegElem   SegKind = "elem"   // keyed-array element, by its key value
	SegMember SegKind = "member" // scalar-array member, by its JSON value
	SegOrder  SegKind = "order"  // the order of an array's elements
)

// Seg is one step of a Path.
type Seg struct {
	Kind  SegKind `json:"kind"`
	Value string  `json:"value,omitempty"`
}

// Path addresses a value inside a document.
type Path []Seg

func (p Path) String() string {
	var b strings.Builder
	for _, s := range p {
		switch s.Kind {
		case SegKey:
			if b.Len() > 0 {
				b.WriteByte('.')
			}
			b.WriteString(s.Value)
		case SegElem:
			b.WriteString("[" + s.Value + "]")
		case SegMember:
			b.WriteString("{" + s.Value + "}")
		case SegOrder:
			b.WriteString("<order>")
		}
	}
	return b.String()
}

// Op is what a change does at its path.
type Op string

const (
	OpAdd     Op = "add"
	OpRemove  Op = "remove"
	OpChange  Op = "change"
	OpReorder Op = "reorder"
)

// Change is one difference between two documents.
type Change struct {
	Path Path `json:"path"`
	Op   Op   `json:"op"`
	Old  any  `json:"old,omitempty"`
	New  any  `json:"new,omitempty"`
}

// Decode parses a JSON document keeping numbers exact, so a diff never
// reports 1 vs 1.0 as a change.
func Decode(raw []byte) (any, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// Diff lists the changes that turn base into other.
func Diff(base, other any) []Change {
	var out []Change
	diff(nil, base, other, &out)
	return out
}

func diff(path Path, a, b any, out *[]Change) {
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok {
			break
		}
		for _, k := range unionKeys(av, bv) {
			x, inA := av[k]
			y, inB := bv[k]
			p := with(path, Seg{SegKey, k})
			switch {
			case !inA:
				*out = append(*out, Change{Path: p, Op: OpAdd, New: y})
			case !inB:
				*out = append(*out, Change{Path: p, Op: OpRemove, Old: x})
			default:
				diff(p, x, y, out)
			}
		}
		return
	case []any:
		bv, ok := b.([]any)
		if !ok {
			break
		}
		if field := arrayKeyField(av, bv); field != "" {
			diffKeyed(path, field, av, bv, out)
			return
		}
		if scalarSet(av) && scalarSet(bv) {
			diffMembers(path, av, bv, out)
			return
		}
	}
	if !Equal(a, b) {
		*out = append(*out, Change{Path: path, Op: OpChange, Old: a, New: b})
	}
}

func diffKeyed(path Path, field string, a, b []any, out *[]Change) {
	am, aOrder := index(a, field)
	bm, bOrder := index(b, field)
	for _, id := range aOrder {
		p := with(path, Seg{SegElem, id})
		if y, ok := bm[id]; ok {
			diff(p, am[id], y, out)
		} else {
			*out = append(*out, Change{Path: p, Op: OpRemove, Old: am[id]})
		}
	}
	// Reorder before additions: an added element is then placed next to
	// its sibling's final position.
	reorder(path, aOrder, bOrder, out)
	for _, id := range bOrder {
		if _, ok := am[id]; !ok {
			*out = append(*out, Change{Path: with(path, Seg{SegElem, id}), Op: OpAdd, New: bm[id]})
		}
	}
}

func diffMembers(path Path, a, b []any, out *[]Change) {
	aKeys, bKeys := memberKeys(a), memberKeys(b)
	inA, inB := set(aKeys), set(bKeys)
	for i, k := range aKeys {
		if !inB[k] {
			*out = append(*out, Change{Path: with(path, Seg{SegMember, k}), Op: OpRemove, Old: a[i]})
		}
	}
	reorder(path, aKeys, bKeys, out)
	for i, k := range bKeys {
		if !inA[k] {
			*out = append(*out, Change{Path: with(path, Seg{SegMember, k}), Op: OpAdd, New: b[i]})
		}
	}
}

// reorder records a change of relative order among the elements both sides
// share (additions and removals are their own changes).
func reorder(path Path, a, b []string, out *[]Change) {
	inA, inB := set(a), set(b)
	var ca, cb []string
	for _, k := range a {
		if inB[k] {
			ca = append(ca, k)
		}
	}
	for _, k := range b {
		if inA[k] {
			cb = append(cb, k)
		}
	}
	if !reflect.DeepEqual(ca, cb) {
		*out = append(*out, Change{Path: with(path, Seg{Kind: SegOrder}), Op: OpReorder, Old: ca, New: cb})
	}
}

// keyFields are tried in order; an array is keyed by the first one every
// element carries as a unique string.
var keyFields = []string{"id", "key", "name"}

func arrayKeyField(a, b []any) string {
	if len(a) == 0 && len(b) == 0 {
		return ""
	}
	for _, f := range keyFields {
		if keyedBy(a, f) && keyedBy(b, f) {
			return f
		}
	}
	return ""
}

func keyedBy(arr []any, field string) bool {
	seen := map[string]bool{}
	for _, e := range arr {
		m, ok := e.(map[string]any)
		if !ok {
			return false
		}
		s, ok := m[field].(string)
		if !ok || s == "" || seen[s] {
			return false
		}
		seen[s] = true
	}
	return true
}

func elemKey(e any, field string) string {
	m, _ := e.(map[string]any)
	s, _ := m[field].(string)
	return s
}

func index(arr []any, field string) (map[string]any, []string) {
	m := make(map[string]any, len(arr))
	order := make([]string, 0, len(arr))
	for _, e := range arr {
		k := elemKey(e, field)
		m[k] = e
		order = append(order, k)
	}
	return m, order
}

func scalarSet(arr []any) bool {
	seen := map[string]bool{}
	for _, e := range arr {
		switch e.(type) {
		case string, json.Number, float64, bool:
		default:
			return false
		}
		k := memberKey(e)
		if seen[k] {
			return false
		}
		seen[k] = true
	}
	return true
}

// memberKey is a scalar's identity inside a membership set. Strings stay
// bare (they are almost always ids, and read well in a path); other
// scalars are JSON-encoded so "1" and 1 stay distinct.
func memberKey(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return "#" + string(b)
}

func memberKeys(arr []any) []string {
	out := make([]string, len(arr))
	for i, e := range arr {
		out[i] = memberKey(e)
	}
	return out
}

func set(keys []string) map[string]bool {
	m := make(map[string]bool, len(keys))
	for _, k := range keys {
		m[k] = true
	}
	return m
}

func unionKeys(a, b map[string]any) []string {
	keys := make([]string, 0, len(a)+len(b))
	for k := range a {
		keys = append(keys, k)
	}
	for k := range b {
		if _, ok := a[k]; !ok {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

func with(p Path, s Seg) Path {
	out := make(Path, len(p), len(p)+1)
	copy(out, p)
	return append(out, s)
}

// Equal compares two decoded JSON values by value.
func Equal(a, b any) bool {
	ab, err1 := json.Marshal(a)
	bb, err2 := json.Marshal(b)
	return err1 == nil && err2 == nil && bytes.Equal(ab, bb)
}

func clone(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("corecustom: clone: %v", err))
	}
	out, _ := Decode(b)
	return out
}

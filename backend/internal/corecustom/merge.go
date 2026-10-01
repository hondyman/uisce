package corecustom

// Get returns the value at path, if present.
func Get(doc any, path Path) (any, bool) {
	cur := doc
	for _, s := range path {
		switch s.Kind {
		case SegKey:
			m, ok := cur.(map[string]any)
			if !ok {
				return nil, false
			}
			if cur, ok = m[s.Value]; !ok {
				return nil, false
			}
		case SegElem:
			arr, _ := cur.([]any)
			i := findElem(arr, s.Value)
			if i < 0 {
				return nil, false
			}
			cur = arr[i]
		case SegMember:
			arr, _ := cur.([]any)
			i := findMember(arr, s.Value)
			if i < 0 {
				return nil, false
			}
			cur = arr[i]
		default:
			return nil, false
		}
	}
	return cur, true
}

// Apply replays one change (taken from Diff(base, tenant)) onto target and
// returns the result. tenant is the document the change came from: when
// the target no longer has the element a change sits inside (the core
// removed it), the tenant's whole copy of that element is restored, since
// "keep my customization" means keep the thing it customizes.
func Apply(target any, ch Change, tenant any) any {
	if len(ch.Path) == 0 {
		if ch.Op == OpRemove {
			return nil
		}
		return clone(ch.New)
	}
	return applyAt(target, ch.Path, 0, ch, tenant)
}

func applyAt(cur any, full Path, depth int, ch Change, tenant any) any {
	seg := full[depth]
	last := depth == len(full)-1
	switch seg.Kind {
	case SegKey:
		m, ok := cur.(map[string]any)
		if !ok {
			return cur
		}
		if last {
			if ch.Op == OpRemove {
				delete(m, seg.Value)
			} else {
				m[seg.Value] = clone(ch.New)
			}
			return m
		}
		child, ok := m[seg.Value]
		if !ok {
			if tv, has := Get(tenant, full[:depth+1]); has && ch.Op != OpRemove {
				m[seg.Value] = clone(tv)
			}
			return m
		}
		m[seg.Value] = applyAt(child, full, depth+1, ch, tenant)
		return m

	case SegElem:
		arr, ok := cur.([]any)
		if !ok {
			return cur
		}
		i := findElem(arr, seg.Value)
		if last {
			switch {
			case ch.Op == OpRemove:
				if i >= 0 {
					arr = append(arr[:i], arr[i+1:]...)
				}
			case i >= 0:
				arr[i] = clone(ch.New)
			default:
				arr = insertLike(arr, clone(ch.New), seg.Value, tenant, full[:depth], elemKeyOf)
			}
			return arr
		}
		if i < 0 {
			if tv, has := Get(tenant, full[:depth+1]); has && ch.Op != OpRemove {
				arr = insertLike(arr, clone(tv), seg.Value, tenant, full[:depth], elemKeyOf)
			}
			return arr
		}
		arr[i] = applyAt(arr[i], full, depth+1, ch, tenant)
		return arr

	case SegMember:
		arr, ok := cur.([]any)
		if !ok {
			if ch.Op == OpAdd && cur == nil {
				return []any{clone(ch.New)}
			}
			return cur
		}
		i := findMember(arr, seg.Value)
		if ch.Op == OpRemove {
			if i >= 0 {
				arr = append(arr[:i], arr[i+1:]...)
			}
			return arr
		}
		if i < 0 {
			arr = insertLike(arr, clone(ch.New), seg.Value, tenant, full[:depth], func(v any) string { return memberKey(v) })
		}
		return arr

	case SegOrder:
		arr, ok := cur.([]any)
		if !ok {
			return cur
		}
		return reorderTo(arr, orderKeys(ch.New))
	}
	return cur
}

// insertLike puts item into arr where the tenant has it: after the nearest
// earlier tenant sibling the target also has, else before the nearest later
// one, else at the end.
func insertLike(arr []any, item any, key string, tenant any, arrPath Path, keyOf func(any) string) []any {
	tv, _ := Get(tenant, arrPath)
	tArr, _ := tv.([]any)
	pos := -1
	for i, e := range tArr {
		if keyOf(e) == key {
			pos = i
			break
		}
	}
	at := len(arr)
	if pos >= 0 {
		found := false
		for j := pos - 1; j >= 0 && !found; j-- {
			if k := indexByKey(arr, keyOf(tArr[j]), keyOf); k >= 0 {
				at, found = k+1, true
			}
		}
		for j := pos + 1; j < len(tArr) && !found; j++ {
			if k := indexByKey(arr, keyOf(tArr[j]), keyOf); k >= 0 {
				at, found = k, true
			}
		}
	}
	arr = append(arr, nil)
	copy(arr[at+1:], arr[at:])
	arr[at] = item
	return arr
}

// reorderTo puts the elements named in order into that relative order,
// using the slots they already occupy; everything else stays put.
func reorderTo(arr []any, order []string) []any {
	rank := make(map[string]int, len(order))
	for i, k := range order {
		rank[k] = i
	}
	keyOf := arrayKeyOf(arr)
	var slots []int
	var items []any
	for i, e := range arr {
		if _, ok := rank[keyOf(e)]; ok {
			slots = append(slots, i)
			items = append(items, e)
		}
	}
	sortByRank(items, func(e any) int { return rank[keyOf(e)] })
	for n, i := range slots {
		arr[i] = items[n]
	}
	return arr
}

func sortByRank(items []any, rank func(any) int) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && rank(items[j]) < rank(items[j-1]); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

func arrayKeyOf(arr []any) func(any) string {
	if f := arrayKeyField(arr, arr); f != "" {
		return func(e any) string { return elemKey(e, f) }
	}
	return memberKey
}

func orderKeys(v any) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, e := range x {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// elemKeyOf is an object element's key, trying the key fields in order.
func elemKeyOf(e any) string {
	for _, f := range keyFields {
		if k := elemKey(e, f); k != "" {
			return k
		}
	}
	return ""
}

func findElem(arr []any, key string) int {
	for _, f := range keyFields {
		for i, e := range arr {
			if elemKey(e, f) == key {
				return i
			}
		}
	}
	return -1
}

func findMember(arr []any, key string) int {
	return indexByKey(arr, key, memberKey)
}

func indexByKey(arr []any, key string, keyOf func(any) string) int {
	for i, e := range arr {
		if keyOf(e) == key {
			return i
		}
	}
	return -1
}

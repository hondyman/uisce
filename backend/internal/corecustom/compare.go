package corecustom

import "sort"

// Group is the unit a person keeps or removes: every change that belongs
// to one element (a widget, a tab, a data source...). ID is stable across
// calls for the same documents, so a decision can be sent back by ID.
type Group struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
	// Summary is "added" / "removed" / "changed": what the group does to
	// the element as a whole.
	Summary string       `json:"summary"`
	Changes []ChangeView `json:"changes"`
	// Conflict: the core also changed this element since the tenant's base.
	// Keeping the customization means the tenant's version wins.
	Conflict bool `json:"conflict"`
	// InCore: the core now contains exactly this customization, so it
	// carries forward whether kept or not.
	InCore bool `json:"inCore"`
}

// ChangeView is a Change with its path rendered for display.
type ChangeView struct {
	Path string `json:"path"`
	Op   Op     `json:"op"`
	Old  any    `json:"old,omitempty"`
	New  any    `json:"new,omitempty"`
}

// Grouper maps a change to the element it belongs to. docs are base,
// tenant and core, for looking up a readable label.
type Grouper func(p Path, docs ...any) (id, kind, label string)

// Report is the comparison of a tenant's extension with the core.
type Report struct {
	// Customizations: what the tenant changed relative to their base.
	Customizations []Group `json:"customizations"`
	// CoreUpdates: what the gold copy changed since the tenant's base -
	// taken as-is on upgrade except where a kept customization conflicts.
	CoreUpdates []Group `json:"coreUpdates"`
}

// Compare pairs the tenant's customizations with the core's changes.
func Compare(base, tenant, core any, g Grouper) Report {
	tc := Diff(base, tenant)
	cc := Diff(base, core)
	custom := group(tc, g, base, tenant, core)
	updates := group(cc, g, base, tenant, core)
	for i := range custom {
		changes := changesOf(tc, custom[i].ID, g, base, tenant, core)
		inCore := true
		for _, t := range changes {
			for _, c := range cc {
				if overlaps(t.Path, c.Path) {
					custom[i].Conflict = true
				}
			}
			if !reflectsIn(t, core) {
				inCore = false
			}
		}
		custom[i].InCore = inCore
		if inCore {
			custom[i].Conflict = false
		}
	}
	return Report{Customizations: custom, CoreUpdates: updates}
}

// Merge replays onto core every customization whose group is not in
// remove. A kept customization wins where it conflicts with the core.
func Merge(base, tenant, core any, remove map[string]bool, g Grouper) any {
	out := clone(core)
	for _, ch := range Diff(base, tenant) {
		id, _, _ := g(ch.Path, base, tenant, core)
		if remove[id] {
			continue
		}
		out = Apply(out, ch, tenant)
	}
	return out
}

func group(changes []Change, g Grouper, docs ...any) []Group {
	byID := map[string]*Group{}
	var order []string
	for _, ch := range changes {
		id, kind, label := g(ch.Path, docs...)
		grp, ok := byID[id]
		if !ok {
			grp = &Group{ID: id, Kind: kind, Label: label}
			byID[id] = grp
			order = append(order, id)
		}
		grp.Changes = append(grp.Changes, ChangeView{Path: ch.Path.String(), Op: ch.Op, Old: ch.Old, New: ch.New})
	}
	out := make([]Group, 0, len(order))
	for _, id := range order {
		grp := byID[id]
		grp.Summary = summarize(grp.Changes)
		out = append(out, *grp)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Kind < out[j].Kind })
	return out
}

func changesOf(changes []Change, id string, g Grouper, docs ...any) []Change {
	var out []Change
	for _, ch := range changes {
		if gid, _, _ := g(ch.Path, docs...); gid == id {
			out = append(out, ch)
		}
	}
	return out
}

// summarize reads the group's net effect off its shallowest change: a
// group whose element was added or removed whole is "added"/"removed".
func summarize(changes []ChangeView) string {
	shallow := changes[0]
	for _, c := range changes[1:] {
		if len(c.Path) < len(shallow.Path) {
			shallow = c
		}
	}
	switch shallow.Op {
	case OpAdd:
		return "added"
	case OpRemove:
		return "removed"
	}
	return "changed"
}

// overlaps: one path is inside (or equal to) the other. Two membership or
// order changes of the same array touch different members, so they don't
// overlap unless they name the same member.
func overlaps(a, b Path) bool {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return false
		}
	}
	if len(a) != len(b) {
		// a strict prefix: the shorter one's last segment addresses the
		// whole element the longer one sits in - unless it's a member or
		// order step, which is a leaf of the array, not a container.
		short := a
		if len(b) < len(a) {
			short = b
		}
		k := short[len(short)-1].Kind
		return k == SegKey || k == SegElem
	}
	return true
}

// reflectsIn: the core already has this change's outcome.
func reflectsIn(ch Change, core any) bool {
	v, ok := Get(core, ch.Path)
	switch ch.Op {
	case OpRemove:
		return !ok
	case OpReorder:
		return false
	}
	return ok && Equal(v, ch.New)
}

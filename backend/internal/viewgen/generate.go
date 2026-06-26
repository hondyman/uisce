package viewgen

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/cube"
	"github.com/hondyman/semlayer/backend/internal/viewmodel"
)

// Options controls view generation behavior.
type Options struct {
	MaxDepth       int
	DefaultPublic  any
	PiiMetaKey     string
	AdminViewRoles []string
	// When true (or when AdminViewRoles/AdminPublic provided), also emit an admin variant
	// named <base>_admin_view with broader access and fewer excludes.
	EnableAdminVariant bool
	// If set, use this as Public for admin variant; otherwise, if AdminViewRoles set,
	// generator will synthesize a simple expression string like "role in (...)".
	AdminPublic         any
	PreferPrefixedJoins bool
	ExcludeFields       []string
}

type Result struct {
	Views    []viewmodel.View
	Warnings []string
}

// GenerateViews builds views for custom/extension cubes using merged runtime cubes.
func GenerateViews(cubes []cube.Cube, opt Options) Result {
	if opt.MaxDepth <= 0 {
		opt.MaxDepth = 2
	}
	if opt.PiiMetaKey == "" {
		opt.PiiMetaKey = "pii"
	}

	graph := buildGraph(cubes)
	idx := indexCubes(cubes)

	var out []viewmodel.View
	var warns []string

	for _, c := range cubes {
		if isCustomCube(c) {
			v := buildRootView(c, cubes, graph, idx, opt)
			out = append(out, v)
			// Optionally emit admin variant
			if opt.EnableAdminVariant || len(opt.AdminViewRoles) > 0 || opt.AdminPublic != nil {
				admin := makeAdminVariant(v, idx, opt)
				out = append(out, admin)
			}
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return Result{Views: out, Warnings: warns}
}

// ---- helpers ----

func buildGraph(cubes []cube.Cube) map[string][]string {
	g := map[string][]string{}
	for _, c := range cubes {
		for jn := range c.Joins {
			if jn == "" {
				continue
			}
			g[c.Name] = append(g[c.Name], jn)
		}
	}
	for k := range g {
		g[k] = uniq(g[k])
	}
	return g
}

func indexCubes(cubes []cube.Cube) map[string]cube.Cube {
	m := make(map[string]cube.Cube, len(cubes))
	for _, c := range cubes {
		m[c.Name] = c
	}
	return m
}

func uniq(ss []string) []string {
	m := map[string]struct{}{}
	var out []string
	for _, s := range ss {
		if _, ok := m[s]; ok {
			continue
		}
		m[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func isCustomCube(c cube.Cube) bool {
	// Prefer explicit metadata from merge pipeline
	if c.Metadata != nil {
		if _, ok := c.Metadata["extension_changes"]; ok {
			return true
		}
		if _, ok := c.Metadata["inherits_from"]; ok {
			return true
		}
		if v, ok := c.Metadata["custom"].(bool); ok && v {
			return true
		}
	}
	if c.Meta != nil {
		if v, ok := c.Meta["custom"].(bool); ok && v {
			return true
		}
	}
	if strings.Contains(c.Title, "(Custom)") {
		return true
	}
	return false
}

func buildRootView(root cube.Cube, cubes []cube.Cube, graph map[string][]string, idx map[string]cube.Cube, opt Options) viewmodel.View {
	viewName := fmt.Sprintf("%s_view", root.Name)
	title := root.Title
	if title == "" {
		title = humanize(root.Name)
	}
	if !strings.Contains(strings.ToLower(title), "view") {
		title += " View"
	}

	v := viewmodel.View{
		Name:        viewName,
		Title:       title,
		Description: defaultViewDescription(root),
		Public:      opt.DefaultPublic,
		Meta: map[string]any{
			"generated_at": time.Now().UTC().Format(time.RFC3339),
			"root_cube":    root.Name,
		},
	}
	// Inherit access policy from root if present and not explicitly set
	if root.AccessPolicy != nil && v.AccessPolicy == nil {
		v.AccessPolicy = &viewmodel.AccessPolicy{Rules: root.AccessPolicy}
	}

	rootBlock := viewmodel.ViewCube{
		JoinPath: root.Name,
		Includes: decideRootIncludes(root, opt),
		Excludes: decideExcludes(root, opt),
	}
	v.Cubes = append(v.Cubes, rootBlock)

	type path struct{ nodes []string }
	queue := []path{{nodes: []string{root.Name}}}
	seenPaths := map[string]struct{}{root.Name: {}}

	for depth := 0; depth < opt.MaxDepth; depth++ {
		var next []path
		for _, p := range queue {
			cur := p.nodes[len(p.nodes)-1]
			for _, nbr := range graph[cur] {
				if contains(p.nodes, nbr) {
					continue
				}
				newPath := append(safeCopy(p.nodes), nbr)
				joinPath := strings.Join(newPath, ".")
				if _, ok := seenPaths[joinPath]; ok {
					continue
				}
				seenPaths[joinPath] = struct{}{}
				next = append(next, path{nodes: newPath})

				target := idx[nbr]
				block := viewmodel.ViewCube{
					JoinPath: joinPath,
					Prefix:   opt.PreferPrefixedJoins,
					Includes: decideJoinedIncludes(target, opt),
					Excludes: decideExcludes(target, opt),
				}
				v.Cubes = append(v.Cubes, block)
			}
		}
		queue = next
	}

	v.Folders = autoFolders(v, idx, opt)
	return v
}

func safeCopy[T any](in []T) []T { out := make([]T, len(in)); copy(out, in); return out }
func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func humanize(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func defaultViewDescription(root cube.Cube) string {
	if root.Description != "" {
		return root.Description
	}
	return "Auto-generated view over " + root.Name
}

// makeAdminVariant creates a parallel admin view with relaxed excludes and role/expr public.
func makeAdminVariant(base viewmodel.View, idx map[string]cube.Cube, opt Options) viewmodel.View {
	v := base
	// Name and title tweaks
	if strings.HasSuffix(v.Name, "_view") {
		v.Name = strings.TrimSuffix(v.Name, "_view") + "_admin_view"
	} else {
		v.Name = v.Name + "_admin"
	}
	if v.Title == "" {
		v.Title = humanize(v.Name)
	}
	if !strings.Contains(strings.ToLower(v.Title), "admin") {
		v.Title += " (Admin)"
	}
	// Public override
	if opt.AdminPublic != nil {
		v.Public = opt.AdminPublic
	} else if len(opt.AdminViewRoles) > 0 {
		v.Public = fmt.Sprintf("role in (%s)", strings.Join(opt.AdminViewRoles, ","))
	}
	// Meta hint
	if v.Meta == nil {
		v.Meta = map[string]any{}
	}
	v.Meta["admin_variant"] = true
	// Relax excludes: remove tenant_id and PII exclusions
	for i := range v.Cubes {
		blk := &v.Cubes[i]
		// Remove tenant_id from excludes
		var ex []string
		for _, e := range blk.Excludes {
			if strings.EqualFold(e, "tenant_id") {
				continue
			}
			ex = append(ex, e)
		}
		// Remove PII excludes if they are present by name
		// Identify leaf cube
		parts := strings.Split(blk.JoinPath, ".")
		leaf := parts[len(parts)-1]
		if c, ok := idx[leaf]; ok {
			// Build set of PII dimension names
			pii := map[string]struct{}{}
			for n, d := range c.Dimensions {
				if isPII(d, opt) {
					pii[n] = struct{}{}
				}
			}
			tmp := ex[:0]
			for _, e := range ex {
				if _, ok := pii[e]; ok {
					continue
				}
				tmp = append(tmp, e)
			}
			ex = tmp
		}
		blk.Excludes = ex
	}
	return v
}

// include/exclude policies

func decideRootIncludes(c cube.Cube, opt Options) []viewmodel.IncludeItem {
	var items []viewmodel.IncludeItem
	// dimensions: PK, business keys, descriptors, time
	for n, d := range c.Dimensions {
		pk := asBool(d["primary_key"])
		if pk || isBusinessKey(n, d) || isDescriptor(n) || isTime(n, d) {
			if shouldExclude(n, opt) || isPII(d, opt) {
				continue
			}
			it := viewmodel.IncludeItem{Name: n}
			// Tag PII propagation in meta if dimension is marked sensitive
			if isPII(d, opt) {
				it.Meta = map[string]any{"pii": true}
			}
			items = append(items, it)
		}
	}
	// measures: core set
	for n, m := range c.Measures {
		if isCoreMeasure(m) {
			items = append(items, viewmodel.IncludeItem{Name: n})
		}
	}
	if len(items) == 0 {
		return []viewmodel.IncludeItem{{Name: "*"}}
	}
	// deterministic order
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return applyOverrides(c, items)
}

func decideJoinedIncludes(c cube.Cube, opt Options) []viewmodel.IncludeItem {
	return []viewmodel.IncludeItem{{Name: "*"}}
}

func decideExcludes(c cube.Cube, opt Options) []string {
	ex := append([]string{}, opt.ExcludeFields...)
	// exclude pii
	for n, d := range c.Dimensions {
		if isPII(d, opt) {
			ex = append(ex, n)
		}
	}
	// exclude tenant_id by default
	if !inSlice(ex, "tenant_id") {
		if _, ok := c.Dimensions["tenant_id"]; ok {
			ex = append(ex, "tenant_id")
		}
	}
	ex = uniq(ex)
	sort.Strings(ex)
	return ex
}

func asBool(v any) bool { b, _ := v.(bool); return b }
func inSlice(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func isPII(dim map[string]any, opt Options) bool {
	if dim == nil {
		return false
	}
	if dim["meta"] == nil {
		return false
	}
	if m, ok := dim["meta"].(map[string]any); ok {
		if v, ok := m[opt.PiiMetaKey]; ok {
			if b, okb := v.(bool); okb {
				return b
			}
		}
	}
	return false
}

func isBusinessKey(name string, dim map[string]any) bool {
	if asBool(dim["primary_key"]) {
		return false
	}
	if strings.HasSuffix(name, "_id") {
		return true
	}
	if m, ok := dim["meta"].(map[string]any); ok {
		if t, ok := m["key_type"].(string); ok && t == "business" {
			return true
		}
	}
	return false
}

func isDescriptor(name string) bool {
	n := strings.ToLower(name)
	return strings.Contains(n, "name") || strings.Contains(n, "title") || strings.Contains(n, "slug") || strings.Contains(n, "sku") || strings.Contains(n, "code") || strings.Contains(n, "country") || strings.Contains(n, "region")
}

func isTime(name string, dim map[string]any) bool {
	if t, ok := dim["type"].(string); ok && t == "time" {
		return true
	}
	n := strings.ToLower(name)
	return strings.HasSuffix(n, "_at") || strings.Contains(n, "date")
}

func isCoreMeasure(mea map[string]any) bool {
	if t, ok := mea["type"].(string); ok {
		if t == "count" {
			return true
		}
		if t == "sum" || t == "avg" || t == "min" || t == "max" {
			return true
		}
	}
	return false
}

// autoFolders creates simple folders based on root and related blocks.
func autoFolders(v viewmodel.View, idx map[string]cube.Cube, opt Options) []viewmodel.Folder {
	var folders []viewmodel.Folder
	// Basic folders for root
	var basic []viewmodel.FolderItem
	var metrics []viewmodel.FolderItem
	var times []viewmodel.FolderItem
	if len(v.Cubes) > 0 {
		root := v.Cubes[0]
		if c, ok := idx[root.JoinPath]; ok {
			for n, d := range c.Dimensions {
				if shouldExclude(n, opt) || isPII(d, opt) {
					continue
				}
				if isTime(n, d) {
					times = append(times, viewmodel.FolderItem{Member: n})
				} else if isDescriptor(n) || isBusinessKey(n, d) || asBool(d["primary_key"]) {
					basic = append(basic, viewmodel.FolderItem{Member: n})
				}
			}
			for n := range c.Measures {
				if isCoreMeasure(c.Measures[n]) {
					metrics = append(metrics, viewmodel.FolderItem{Member: n})
				}
			}
		}
	}
	if len(basic) > 0 {
		folders = append(folders, viewmodel.Folder{Name: "Basic Details", Includes: basic})
	}
	if len(metrics) > 0 {
		folders = append(folders, viewmodel.Folder{Name: "Metrics", Includes: metrics})
	}
	if len(times) > 0 {
		folders = append(folders, viewmodel.Folder{Name: "Time", Includes: times})
	}

	// Related folders per joined block
	var relatedItems []viewmodel.FolderItem
	for i := 1; i < len(v.Cubes); i++ {
		blk := v.Cubes[i]
		// Name from last segment of join_path
		parts := strings.Split(blk.JoinPath, ".")
		label := humanize(parts[len(parts)-1])
		// Simulate a couple representative members: if prefixed, show two placeholders
		// In a real implementation, we would map includes to rendered names. Here we just suggest by cube dims.
		if c, ok := idx[parts[len(parts)-1]]; ok {
			var items []viewmodel.FolderItem
			count := 0
			for n, d := range c.Dimensions {
				if shouldExclude(n, opt) || isPII(d, opt) {
					continue
				}
				items = append(items, viewmodel.FolderItem{Member: renderName(blk, n)})
				count++
				if count >= 3 {
					break
				}
			}
			if len(items) > 0 {
				relatedItems = append(relatedItems, viewmodel.FolderItem{Nested: &viewmodel.Folder{Name: label, Includes: items}})
			}
		}
	}
	if len(relatedItems) > 0 {
		folders = append(folders, viewmodel.Folder{Name: "Related", Includes: relatedItems})
	}
	return folders
}

func shouldExclude(name string, opt Options) bool { return inSlice(opt.ExcludeFields, name) }

func renderName(blk viewmodel.ViewCube, member string) string {
	if blk.Prefix {
		parts := strings.Split(blk.JoinPath, ".")
		last := parts[len(parts)-1]
		pfx := blk.Alias
		if pfx == "" {
			pfx = last
		}
		return pfx + "_" + member
	}
	return member
}

// applyOverrides maps cube-level member overrides (if any) into include items.
// Expectation: c.Meta["member_overrides"] is map[string]map[string]any keyed by member name.
func applyOverrides(c cube.Cube, items []viewmodel.IncludeItem) []viewmodel.IncludeItem {
	if c.Meta == nil {
		return items
	}
	raw, ok := c.Meta["member_overrides"]
	if !ok {
		return items
	}
	ov, ok := raw.(map[string]any)
	if !ok {
		return items
	}
	out := make([]viewmodel.IncludeItem, 0, len(items))
	for _, it := range items {
		if mraw, ok := ov[it.Name]; ok {
			if m, ok := mraw.(map[string]any); ok {
				if v, ok := m["alias"].(string); ok {
					it.Alias = v
				}
				if v, ok := m["title"].(string); ok {
					it.Title = v
				}
				if v, ok := m["description"].(string); ok {
					it.Description = v
				}
				if v, ok := m["format"].(string); ok {
					it.Format = v
				}
				if v, ok := m["meta"].(map[string]any); ok {
					it.Meta = v
				}
			}
		}
		out = append(out, it)
	}
	return out
}

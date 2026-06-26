package viewmerge

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/hondyman/semlayer/backend/internal/viewmodel"
)

type Change struct {
	Type    string         `json:"type"`
	Target  string         `json:"target,omitempty"`
	Old     any            `json:"old,omitempty"`
	New     any            `json:"new,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

type Stores struct {
	Core       []viewmodel.View
	Extensions []viewmodel.View
}

type Options struct {
	CoreVersion string
}

type Result struct {
	Merged   []viewmodel.View
	Warnings []string
}

func MergeViews(st Stores, opt Options) (Result, error) {
	coreIdx := indexByName(st.Core)
	extIdx := indexByExtends(st.Extensions)

	var out []viewmodel.View
	var warnings []string

	for _, core := range st.Core {
		merged := deepCopy(core)
		var changes []Change

		if ext, ok := extIdx[core.Name]; ok {
			// Safe overrides
			if ext.Title != "" && ext.Title != merged.Title {
				changes = append(changes, Change{Type: "override_title", Old: merged.Title, New: ext.Title})
				merged.Title = ext.Title
			}
			if ext.Description != "" && ext.Description != merged.Description {
				changes = append(changes, Change{Type: "override_description", Old: merged.Description, New: ext.Description})
				merged.Description = ext.Description
			}
			if ext.Public != nil {
				changes = append(changes, Change{Type: "override_public", Old: merged.Public, New: ext.Public})
				merged.Public = ext.Public
			}
			if len(ext.Meta) > 0 {
				if merged.Meta == nil {
					merged.Meta = map[string]any{}
				}
				maps.Copy(merged.Meta, ext.Meta)
				changes = append(changes, Change{Type: "merge_meta", Details: ext.Meta})
			}

			// Merge cube blocks by join_path
			for _, cb := range ext.Cubes {
				i := findCubeBlock(merged.Cubes, cb.JoinPath)
				if i < 0 {
					// Additive: allow adding new blocks, but not removing base ones.
					merged.Cubes = append(merged.Cubes, cb)
					changes = append(changes, Change{Type: "add_cube_block", Target: cb.JoinPath})
					continue
				}
				// Merge fields on existing block, disallow changing join_path target
				// Prefix/Alias are safe refinements
				if cb.Alias != "" {
					merged.Cubes[i].Alias = cb.Alias
				}
				if cb.Prefix {
					merged.Cubes[i].Prefix = cb.Prefix
				}
				// Includes: merge with overrides for matching names
				if len(cb.Includes) > 0 {
					before := merged.Cubes[i].Includes
					merged.Cubes[i].Includes = mergeIncludes(before, cb.Includes, &changes, cb.JoinPath)
				}
				// Excludes: additive unique
				if len(cb.Excludes) > 0 {
					merged.Cubes[i].Excludes = mergeStrings(merged.Cubes[i].Excludes, cb.Excludes)
				}
			}

			// Merge folders additively by name
			for _, f := range ext.Folders {
				j := findFolder(merged.Folders, f.Name)
				if j < 0 {
					merged.Folders = append(merged.Folders, f)
					changes = append(changes, Change{Type: "add_folder", Target: f.Name})
				} else {
					merged.Folders[j].Includes = mergeFolderIncludes(merged.Folders[j].Includes, f.Includes)
				}
			}
		}

		// Audit meta
		if merged.Meta == nil {
			merged.Meta = map[string]any{}
		}
		if opt.CoreVersion != "" {
			merged.Meta["core_version"] = opt.CoreVersion
		}
		merged.Meta["last_merged"] = time.Now().UTC().Format(time.RFC3339)
		if len(changes) > 0 {
			// store as generic list
			merged.Meta["extension_changes"] = changes
		}

		out = append(out, merged)
	}

	// Orphan extensions (or standalone overrides): include them as standalone views
	// This ensures tenant overrides show up even when there are no published core views.
	// Avoid duplicates when an override's Name already matches a core view we added above.
	for _, ext := range st.Extensions {
		if _, ok := coreIdx[ext.Extends]; !ok {
			warnings = append(warnings, fmt.Sprintf("view extension '%s' has no matching core view '%s'", ext.Name, ext.Extends))
			if _, nameClash := coreIdx[ext.Name]; nameClash {
				// Skip adding a duplicate by name; a core with this name already exists.
				continue
			}
			out = append(out, ext)
		}
	}

	slices.SortFunc(out, func(a, b viewmodel.View) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})

	return Result{Merged: out, Warnings: warnings}, nil
}

// --- helpers ---
func indexByName(vs []viewmodel.View) map[string]viewmodel.View {
	m := make(map[string]viewmodel.View, len(vs))
	for _, v := range vs {
		m[v.Name] = v
	}
	return m
}

func indexByExtends(vs []viewmodel.View) map[string]viewmodel.View {
	m := map[string]viewmodel.View{}
	for _, v := range vs {
		if v.Extends != "" {
			m[v.Extends] = v
		}
	}
	return m
}

func deepCopy[T any](v T) T {
	b, _ := json.Marshal(v)
	var out T
	_ = json.Unmarshal(b, &out)
	return out
}

func findCubeBlock(blocks []viewmodel.ViewCube, joinPath string) int {
	for i := range blocks {
		if blocks[i].JoinPath == joinPath {
			return i
		}
	}
	return -1
}

func mergeIncludes(base, extra []viewmodel.IncludeItem, changes *[]Change, joinPath string) []viewmodel.IncludeItem {
	// Index base by name
	idx := map[string]int{}
	for i, inc := range base {
		idx[inc.Name] = i
	}

	for _, inc := range extra {
		if inc.Name == "*" {
			// If user supplies '*', prefer exactly their list (replace), note change
			*changes = append(*changes, Change{Type: "override_includes", Target: joinPath, Details: map[string]any{"mode": "star"}})
			return extra
		}
		if j, ok := idx[inc.Name]; ok {
			// Apply safe member-level overrides
			b := base[j]
			if inc.Alias != "" {
				b.Alias = inc.Alias
			}
			if inc.Title != "" {
				b.Title = inc.Title
			}
			if inc.Description != "" {
				b.Description = inc.Description
			}
			if inc.Format != "" {
				b.Format = inc.Format
			}
			if inc.Meta != nil {
				b.Meta = inc.Meta
			}
			base[j] = b
		} else {
			base = append(base, inc)
			*changes = append(*changes, Change{Type: "add_include", Target: joinPath, New: inc.Name})
		}
	}
	// deterministic order by Name
	slices.SortFunc(base, func(a, b viewmodel.IncludeItem) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return base
}

func mergeStrings(base, extra []string) []string {
	m := map[string]struct{}{}
	for _, s := range base {
		m[s] = struct{}{}
	}
	for _, s := range extra {
		m[s] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for s := range m {
		out = append(out, s)
	}
	slices.Sort(out)
	return out
}

func findFolder(folders []viewmodel.Folder, name string) int {
	for i := range folders {
		if folders[i].Name == name {
			return i
		}
	}
	return -1
}

func mergeFolderIncludes(base, extra []viewmodel.FolderItem) []viewmodel.FolderItem {
	// naive append; could dedupe by member name
	return append(base, extra...)
}

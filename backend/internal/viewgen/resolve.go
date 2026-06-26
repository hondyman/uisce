package viewgen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/viewmodel"
)

// ReadViewsDir loads all *.json view files from a directory.
func ReadViewsDir(dir string) (map[string]viewmodel.View, error) {
	out := map[string]viewmodel.View{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		var v viewmodel.View
		if err := json.Unmarshal(b, &v); err != nil {
			return nil, fmt.Errorf("unmarshal %s: %w", e.Name(), err)
		}
		out[v.Name] = v
	}
	return out, nil
}

// ResolveViewExtends merges user-authored overrides into generated views using a shallow, safe strategy.
// - title/description/public/meta: override if set on override
// - cubes: match by join_path, merge includes/excludes (additive), keep explicit overrides for IncludeItem fields
// - folders: merge by name (additive)
func ResolveViewExtends(generated map[string]viewmodel.View, overrides map[string]viewmodel.View) map[string]viewmodel.View {
	// Start from generated; apply overrides whose Extends points to a generated base.
	resolved := make(map[string]viewmodel.View, len(generated))
	for k, v := range generated {
		resolved[k] = v
	}
	for name, ov := range overrides {
		baseName := ov.Extends
		if baseName == "" {
			// If no extends, just take override as-is (user-created standalone view)
			resolved[name] = ov
			continue
		}
		base, ok := generated[baseName]
		if !ok {
			// Skip overrides extending unknown base
			continue
		}
		res := base
		// Override metadata
		if ov.Title != "" {
			res.Title = ov.Title
		}
		if ov.Description != "" {
			res.Description = ov.Description
		}
		if ov.Public != nil {
			res.Public = ov.Public
		}
		if ov.Meta != nil {
			if res.Meta == nil {
				res.Meta = map[string]any{}
			}
			for k, v := range ov.Meta {
				res.Meta[k] = v
			}
		}
		if ov.AccessPolicy != nil {
			res.AccessPolicy = ov.AccessPolicy
		}
		// Merge cubes by join_path
		if len(ov.Cubes) > 0 {
			// index base
			byJP := map[string]viewmodel.ViewCube{}
			order := []string{}
			for _, b := range res.Cubes {
				byJP[b.JoinPath] = b
				order = append(order, b.JoinPath)
			}
			for _, cb := range ov.Cubes {
				b, ok := byJP[cb.JoinPath]
				if !ok {
					// add new block
					byJP[cb.JoinPath] = cb
					order = append(order, cb.JoinPath)
					continue
				}
				// merge fields
				if cb.Alias != "" {
					b.Alias = cb.Alias
				}
				if cb.Prefix {
					b.Prefix = cb.Prefix
				}
				// includes: if override has explicit list, prefer override items; else keep base
				if len(cb.Includes) > 0 {
					b.Includes = mergeIncludeItems(b.Includes, cb.Includes)
				}
				// excludes: additive unique
				if len(cb.Excludes) > 0 {
					b.Excludes = uniqStrings(append(b.Excludes, cb.Excludes...))
				}
				byJP[cb.JoinPath] = b
			}
			// rebuild ordered list
			merged := make([]viewmodel.ViewCube, 0, len(byJP))
			for _, jp := range order {
				merged = append(merged, byJP[jp])
			}
			// Any newly added blocks not in order yet (from overrides only)
			for jp, b := range byJP {
				found := false
				for _, o := range order {
					if o == jp {
						found = true
						break
					}
				}
				if !found {
					merged = append(merged, b)
				}
			}
			res.Cubes = merged
		}
		// Merge folders by name (additive)
		if len(ov.Folders) > 0 {
			byName := map[string]viewmodel.Folder{}
			order := []string{}
			for _, f := range res.Folders {
				byName[f.Name] = f
				order = append(order, f.Name)
			}
			for _, f := range ov.Folders {
				if b, ok := byName[f.Name]; ok {
					b.Includes = append(b.Includes, f.Includes...)
					byName[f.Name] = b
				} else {
					byName[f.Name] = f
					order = append(order, f.Name)
				}
			}
			merged := make([]viewmodel.Folder, 0, len(byName))
			for _, n := range order {
				merged = append(merged, byName[n])
			}
			// Add any new ones
			for n, f := range byName {
				found := false
				for _, o := range order {
					if o == n {
						found = true
						break
					}
				}
				if !found {
					merged = append(merged, f)
				}
			}
			res.Folders = merged
		}
		// Use override view Name
		res.Name = name
		resolved[name] = res
	}
	return resolved
}

func mergeIncludeItems(base, ov []viewmodel.IncludeItem) []viewmodel.IncludeItem {
	// If override contains "*", return exactly override list, trusting user intent.
	for _, it := range ov {
		if it.Name == "*" {
			return ov
		}
	}
	// Otherwise merge by member name, applying alias/title/description/format/meta
	idx := map[string]int{}
	out := make([]viewmodel.IncludeItem, len(base))
	copy(out, base)
	for i, it := range base {
		idx[it.Name] = i
	}
	for _, it := range ov {
		if j, ok := idx[it.Name]; ok {
			b := out[j]
			if it.Alias != "" {
				b.Alias = it.Alias
			}
			if it.Title != "" {
				b.Title = it.Title
			}
			if it.Description != "" {
				b.Description = it.Description
			}
			if it.Format != "" {
				b.Format = it.Format
			}
			if it.Meta != nil {
				b.Meta = it.Meta
			}
			out[j] = b
		} else {
			out = append(out, it)
		}
	}
	return out
}

func uniqStrings(ss []string) []string {
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

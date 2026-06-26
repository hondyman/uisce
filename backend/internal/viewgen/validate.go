package viewgen

import (
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/cube"
	"github.com/hondyman/semlayer/backend/internal/viewmodel"
)

// ValidateViews runs structural validation on generated or user-authored views.
func ValidateViews(cubes []cube.Cube, views []viewmodel.View) []cube.ValidationIssue {
	var issues []cube.ValidationIssue
	graph := buildGraph(cubes)
	idx := indexCubes(cubes)

	// View name uniqueness
	seen := map[string]struct{}{}
	for _, v := range views {
		if _, ok := seen[v.Name]; ok {
			issues = append(issues, vi("error", "DUPLICATE_VIEW_NAME", fmt.Sprintf("duplicate view name '%s'", v.Name)))
		}
		seen[v.Name] = struct{}{}
	}

	for _, v := range views {
		// Validate join paths and members
		flat := flattenMembers(v, idx)
		// Tenant isolation enforcement: ensure tenant_id present in root or via implicit policy
		if !hasTenantGuard(v, idx) {
			issues = append(issues, vi("warning", "TENANT_GUARD_MISSING", fmt.Sprintf("view %s: tenant isolation guard not detected; ensure access_policy or tenant_id filters exist", v.Name)))
		}
		for _, blk := range v.Cubes {
			// join path must be resolvable
			if !validateJoinPath(graph, blk.JoinPath) {
				issues = append(issues, vi("error", "JOIN_PATH_INVALID", fmt.Sprintf("view %s: invalid join_path '%s'", v.Name, blk.JoinPath)))
				continue
			}
			leaf := lastSegment(blk.JoinPath)
			c, ok := idx[leaf]
			if !ok {
				issues = append(issues, vi("error", "JOIN_TARGET_MISSING", fmt.Sprintf("view %s: target cube '%s' not found", v.Name, leaf)))
				continue
			}
			// includes/excludes
			if hasStar(blk.Includes) && len(blk.Includes) > 1 {
				issues = append(issues, vi("warning", "REDUNDANT_INCLUDES", fmt.Sprintf("view %s: includes contains '*' with other members in block %s", v.Name, blk.JoinPath)))
			}
			allowed := membersSet(c)
			for _, it := range blk.Includes {
				if it.Name == "*" {
					continue
				}
				if _, ok := allowed[it.Name]; !ok {
					issues = append(issues, vi("error", "UNKNOWN_MEMBER", fmt.Sprintf("view %s: include '%s' not found at %s", v.Name, it.Name, blk.JoinPath)))
				}
				// PII propagation: if source member is PII, require meta flag or warn
				if isDimPII(idx[leaf], it.Name) && (it.Meta == nil || it.Meta["pii"] != true) {
					issues = append(issues, vi("warning", "PII_PROPAGATION", fmt.Sprintf("view %s: member '%s' appears PII; mark meta.pii=true or exclude", v.Name, it.Name)))
				}
			}
			for _, ex := range blk.Excludes {
				if _, ok := allowed[ex]; !ok {
					// soft warning for excludes that don't match anything
					issues = append(issues, vi("warning", "UNKNOWN_EXCLUDE", fmt.Sprintf("view %s: exclude '%s' not found at %s", v.Name, ex, blk.JoinPath)))
				}
			}
		}
		// Folders reference members that exist in flattened set and are acyclic
		if cyc := foldersCyclic(v.Folders, map[string]struct{}{}); cyc {
			issues = append(issues, vi("error", "FOLDER_CYCLE", fmt.Sprintf("view %s: folder hierarchy has a cycle", v.Name)))
		}
		for _, f := range v.Folders {
			validateFolderMembers(v.Name, f, flat, &issues)
		}
	}

	return issues
}

func vi(level, code, msg string) cube.ValidationIssue {
	return cube.ValidationIssue{Level: level, Code: code, Message: msg}
}

func validateJoinPath(graph map[string][]string, path string) bool {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return false
	}
	// root exists implicitly; verify edges
	for i := 0; i < len(parts)-1; i++ {
		cur := parts[i]
		nxt := parts[i+1]
		if !inSlice(graph[cur], nxt) {
			return false
		}
	}
	return true
}

func lastSegment(p string) string {
	parts := strings.Split(p, ".")
	return parts[len(parts)-1]
}

func hasStar(incl []viewmodel.IncludeItem) bool {
	for _, it := range incl {
		if it.Name == "*" {
			return true
		}
	}
	return false
}

func membersSet(c cube.Cube) map[string]struct{} {
	m := map[string]struct{}{}
	for n := range c.Dimensions {
		m[n] = struct{}{}
	}
	for n := range c.Measures {
		m[n] = struct{}{}
	}
	return m
}

func flattenMembers(v viewmodel.View, idx map[string]cube.Cube) map[string]struct{} {
	out := map[string]struct{}{}
	for _, blk := range v.Cubes {
		leaf := lastSegment(blk.JoinPath)
		c, ok := idx[leaf]
		if !ok {
			continue
		}
		allowed := membersSet(c)
		// compute block members
		if hasStar(blk.Includes) {
			for n := range allowed {
				name := n
				if blk.Prefix {
					name = renderName(blk, n)
				}
				if !inSlice(blk.Excludes, n) {
					out[name] = struct{}{}
				}
			}
		} else {
			for _, it := range blk.Includes {
				if _, ok := allowed[it.Name]; !ok {
					continue
				}
				name := it.Name
				if blk.Prefix {
					name = renderName(blk, name)
				}
				if !inSlice(blk.Excludes, it.Name) {
					out[name] = struct{}{}
				}
			}
		}
	}
	return out
}

func hasTenantGuard(v viewmodel.View, idx map[string]cube.Cube) bool {
	// If access policy specified assume tenant guard is handled
	if v.AccessPolicy != nil && v.AccessPolicy.Rules != nil {
		return true
	}
	// Otherwise check for tenant_id presence in root block includes (either explicit or star without exclude)
	if len(v.Cubes) == 0 {
		return false
	}
	root := v.Cubes[0]
	leaf := lastSegment(root.JoinPath)
	if _, ok := idx[leaf]; !ok {
		return false
	}
	if hasStar(root.Includes) {
		// Ensure tenant_id is not explicitly excluded
		return !inSlice(root.Excludes, "tenant_id")
	}
	for _, it := range root.Includes {
		if strings.EqualFold(it.Name, "tenant_id") {
			return true
		}
	}
	return false
}

func isDimPII(c cube.Cube, name string) bool {
	if d, ok := c.Dimensions[name]; ok {
		return isPII(d, Options{PiiMetaKey: "pii"})
	}
	return false
}

func foldersCyclic(folders []viewmodel.Folder, seen map[string]struct{}) bool {
	for _, f := range folders {
		if _, ok := seen[f.Name]; ok {
			return true
		}
		seen[f.Name] = struct{}{}
		for _, it := range f.Includes {
			if it.Nested != nil {
				if foldersCyclic([]viewmodel.Folder{*it.Nested}, seen) {
					return true
				}
			}
		}
		delete(seen, f.Name)
	}
	return false
}

func validateFolderMembers(viewName string, f viewmodel.Folder, members map[string]struct{}, out *[]cube.ValidationIssue) {
	for _, it := range f.Includes {
		if it.Nested != nil {
			validateFolderMembers(viewName, *it.Nested, members, out)
			continue
		}
		if it.Member == "" {
			continue
		}
		if _, ok := members[it.Member]; !ok {
			*out = append(*out, vi("warning", "FOLDER_MEMBER_UNKNOWN", fmt.Sprintf("view %s: folder '%s' refers to unknown member '%s'", viewName, f.Name, it.Member)))
		}
	}
}

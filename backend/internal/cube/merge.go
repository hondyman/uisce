package cube

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ValidationIssue captures problems or warnings when merging/validating core+extension cubes.
type ValidationIssue struct {
	Level   string         `json:"level"` // "error" | "warning"
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// MergeCube produces a merged cube view by applying extension overrides on top of a base core cube.
// - Safe overrides: title, description, meta, formatting-like fields in dimensions/measures (title, description, meta).
// - Risky overrides: sql in dimensions/measures/joins -> allowed but emits a warning.
// - Disallowed: changing primary_key state of any core dimension.
func MergeCube(base Cube, ext Cube) (Cube, []ValidationIssue) {
	issues := []ValidationIssue{}

	// Start with a deep copy of base
	out := cloneCube(base)

	// Track audit of what changed in the extension vs base
	changes := map[string]any{}
	cubeChanged := []string{}

	// Cube-level simple overrides
	if ext.Title != "" {
		out.Title = ext.Title
		cubeChanged = append(cubeChanged, "title")
	}
	if ext.Description != "" {
		out.Description = ext.Description
		cubeChanged = append(cubeChanged, "description")
	}
	if ext.Public != nil {
		out.Public = ext.Public
		cubeChanged = append(cubeChanged, "public")
	}
	if len(ext.Tags) > 0 {
		out.Tags = append([]string{}, ext.Tags...)
		cubeChanged = append(cubeChanged, "tags")
	}
	if len(ext.Meta) > 0 {
		out.Meta = mergeStringAny(out.Meta, ext.Meta)
		cubeChanged = append(cubeChanged, "meta")
	}
	if len(ext.AccessPolicy) > 0 {
		out.AccessPolicy = mergeStringAny(out.AccessPolicy, ext.AccessPolicy)
		cubeChanged = append(cubeChanged, "access_policy")
	}
	if len(ext.RefreshKey) > 0 {
		out.RefreshKey = mergeStringAny(out.RefreshKey, ext.RefreshKey)
		cubeChanged = append(cubeChanged, "refresh_key")
	}
	if len(ext.PreAggregations) > 0 {
		out.PreAggregations = mergeNestedMap(out.PreAggregations, ext.PreAggregations)
		cubeChanged = append(cubeChanged, "pre_aggregations")
	}
	if ext.SQLAlias != "" {
		out.SQLAlias = ext.SQLAlias
		cubeChanged = append(cubeChanged, "sql_alias")
	}
	if ext.DataSource != "" {
		out.DataSource = ext.DataSource
		cubeChanged = append(cubeChanged, "data_source")
	}
	// Note: Name and SQL/SQLTable remain from base; ext.Name is treated as the published name if desired by caller.
	if ext.SQL != "" || ext.SQLTable != "" {
		// These are intentionally ignored; record a warning for visibility
		issues = append(issues, issue("warning", "CUBE_SQL_OVERRIDE_IGNORED", "cube-level SQL/SQL_TABLE overrides are ignored for extensions", nil))
	}

	// Dimensions: add new, override existing with rules
	dimAdded := []string{}
	dimOverridden := map[string][]string{}
	for name, extDim := range ext.Dimensions {
		if baseDim, ok := out.Dimensions[name]; ok {
			// Protect primary_key
			if changedPrimaryKey(baseDim, extDim) {
				issues = append(issues, issue("error", "DISALLOWED_PK_OVERRIDE", fmt.Sprintf("dimension '%s': primary_key cannot be changed by extension", name), nil))
				// Remove attempted override
				delete(extDim, "primary_key")
			}
			risky := riskyKeys(extDim)
			if len(risky) > 0 {
				issues = append(issues, issue("warning", "RISKY_DIMENSION_OVERRIDE", fmt.Sprintf("dimension '%s': overriding %v", name, risky), map[string]any{"keys": risky}))
			}
			out.Dimensions[name] = mergeStringAnyMap(baseDim, extDim)
			// Track which keys were provided by extension for this dimension
			dimOverridden[name] = sortedKeys(extDim)
		} else {
			// New dimension
			out.Dimensions[name] = cloneStringAnyMap(extDim)
			dimAdded = append(dimAdded, name)
		}
	}

	// Measures: similar to dimensions
	meaAdded := []string{}
	meaOverridden := map[string][]string{}
	for name, extMea := range ext.Measures {
		if baseMea, ok := out.Measures[name]; ok {
			risky := riskyKeys(extMea)
			if len(risky) > 0 {
				issues = append(issues, issue("warning", "RISKY_MEASURE_OVERRIDE", fmt.Sprintf("measure '%s': overriding %v", name, risky), map[string]any{"keys": risky}))
			}
			out.Measures[name] = mergeStringAnyMap(baseMea, extMea)
			meaOverridden[name] = sortedKeys(extMea)
		} else {
			out.Measures[name] = cloneStringAnyMap(extMea)
			meaAdded = append(meaAdded, name)
		}
	}

	// Joins: add new, override existing (risky if overriding sql)
	joinAdded := []string{}
	joinOverridden := map[string][]string{}
	if out.Joins == nil && (len(base.Joins) > 0 || len(ext.Joins) > 0) {
		out.Joins = map[string]map[string]any{}
	}
	for name, extJoin := range ext.Joins {
		if baseJoin, ok := out.Joins[name]; ok {
			risky := riskyKeys(extJoin)
			if len(risky) > 0 {
				issues = append(issues, issue("warning", "RISKY_JOIN_OVERRIDE", fmt.Sprintf("join '%s': overriding %v", name, risky), map[string]any{"keys": risky}))
			}
			out.Joins[name] = mergeStringAnyMap(baseJoin, extJoin)
			joinOverridden[name] = sortedKeys(extJoin)
		} else {
			out.Joins[name] = cloneStringAnyMap(extJoin)
			joinAdded = append(joinAdded, name)
		}
	}

	// Hierarchies/Segments: naïve union (dedupe by pointer equality is fine since they are maps, but we'll append)
	if len(ext.Hierarchies) > 0 {
		out.Hierarchies = append([]map[string]any{}, base.Hierarchies...)
		out.Hierarchies = append(out.Hierarchies, cloneSliceMap(ext.Hierarchies)...)
		cubeChanged = append(cubeChanged, "hierarchies")
	}
	if len(ext.Segments) > 0 {
		out.Segments = mergeNestedMap(out.Segments, ext.Segments)
		cubeChanged = append(cubeChanged, "segments")
	}

	if ext.DrillMembers != nil {
		out.DrillMembers = append([]string{}, ext.DrillMembers...)
		cubeChanged = append(cubeChanged, "drill_members")
	}

	// Record inheritance metadata
	if out.Metadata == nil {
		out.Metadata = map[string]any{}
	}
	out.Metadata["inherits_from"] = base.Name
	// ext can pass core_version via Metadata
	if v, ok := ext.Metadata["core_version"]; ok {
		out.Metadata["core_version"] = v
	}

	// Capture inheritance chain for downstream consumers
	chain := collectInheritanceChainFromMetadata(base.Metadata)
	if len(chain) == 0 {
		chain = []string{base.Name}
	}
	pubName := ext.Name
	if pubName == "" {
		pubName = base.Name + "_ext"
	}
	chain = append(chain, pubName)
	out.Metadata["inheritance_chain"] = chain

	// Finalize changes audit
	if len(cubeChanged) > 0 {
		changes["cube_fields"] = cubeChanged
	}
	if len(dimAdded) > 0 {
		sort.Strings(dimAdded)
		changes["dimensions_added"] = dimAdded
	}
	if len(dimOverridden) > 0 {
		changes["dimensions_overridden"] = dimOverridden
	}
	if len(meaAdded) > 0 {
		sort.Strings(meaAdded)
		changes["measures_added"] = meaAdded
	}
	if len(meaOverridden) > 0 {
		changes["measures_overridden"] = meaOverridden
	}
	if len(joinAdded) > 0 {
		sort.Strings(joinAdded)
		changes["joins_added"] = joinAdded
	}
	if len(joinOverridden) > 0 {
		changes["joins_overridden"] = joinOverridden
	}
	if len(changes) > 0 {
		out.Metadata["extension_changes"] = changes
	}

	return out, issues
}

// ValidateExtension checks that overrides refer to existing core attributes and disallowed changes are not attempted.
func ValidateExtension(base Cube, ext Cube) []ValidationIssue {
	issues := []ValidationIssue{}
	// Warn on cube-level SQL overrides (ignored by MergeCube)
	if ext.SQL != "" || ext.SQLTable != "" {
		issues = append(issues, issue("warning", "CUBE_SQL_OVERRIDE_IGNORED", "cube-level SQL/SQL_TABLE overrides are ignored for extensions", nil))
	}
	// Determine tenant-scoped behavior from base
	tenantScoped := false
	if base.Dimensions != nil {
		if _, ok := base.Dimensions["tenant_id"]; ok {
			tenantScoped = true
		}
	}
	if !tenantScoped && base.AccessPolicy != nil {
		if v, ok := base.AccessPolicy["tenant_scoped"].(bool); ok && v {
			tenantScoped = true
		}
	}
	if !tenantScoped && base.Metadata != nil {
		if v, ok := base.Metadata["tenant_scoped"].(bool); ok && v {
			tenantScoped = true
		}
	}

	// Dimensions
	for name, extDim := range ext.Dimensions {
		if baseDim, ok := base.Dimensions[name]; ok {
			if changedPrimaryKey(baseDim, extDim) {
				issues = append(issues, issue("error", "DISALLOWED_PK_OVERRIDE", fmt.Sprintf("dimension '%s': primary_key cannot be changed by extension", name), nil))
			}
			// Tenant dimension must not change sql/type in tenant-scoped cubes
			if tenantScoped && name == "tenant_id" {
				if _, has := extDim["sql"]; has && !reflect.DeepEqual(baseDim["sql"], extDim["sql"]) {
					issues = append(issues, issue("error", "DISALLOWED_TENANT_ID_CHANGE", "tenant_id dimension sql cannot be changed in tenant-scoped models", nil))
				}
				if _, has := extDim["type"]; has && !reflect.DeepEqual(baseDim["type"], extDim["type"]) {
					issues = append(issues, issue("error", "DISALLOWED_TENANT_ID_CHANGE", "tenant_id dimension type cannot be changed in tenant-scoped models", nil))
				}
			}
		} else {
			// OK to add a new dimension
			if tenantScoped && name == "tenant_id" {
				// Adding tenant_id where base didn't have it: warn for visibility
				issues = append(issues, issue("warning", "TENANT_ID_DIMENSION_ADDED", "tenant_id dimension added by extension; ensure join guards and filters are consistent", nil))
			}
		}
	}
	// Measures: overriding is fine; adding new is fine
	// Joins: overriding is allowed but warn on sql changes
	baseHasTenant := false
	if base.Dimensions != nil {
		if _, ok := base.Dimensions["tenant_id"]; ok {
			baseHasTenant = true
		}
	}
	for name, extJoin := range ext.Joins {
		if baseJoin, ok := base.Joins[name]; ok {
			if _, has := extJoin["sql"]; has {
				issues = append(issues, issue("warning", "JOIN_SQL_OVERRIDE", fmt.Sprintf("join '%s': sql override may impact correctness", name), nil))
				// Tenant guard heuristic
				if baseHasTenant || tenantScoped {
					if s, ok := extJoin["sql"].(string); ok && !containsTenantGuard(s) {
						issues = append(issues, issue("warning", "TENANT_GUARD_MISSING", fmt.Sprintf("join '%s': sql appears to lack tenant_id guard", name), nil))
					}
				}
			}
			// Relationship change warning
			if r, ok := extJoin["relationship"]; ok {
				if br, ok2 := baseJoin["relationship"]; ok2 && !reflect.DeepEqual(r, br) {
					issues = append(issues, issue("warning", "GOVERNANCE_RELATIONSHIP_CHANGE", fmt.Sprintf("join '%s': relationship changed from %v to %v", name, br, r), nil))
				}
			}
		} else {
			// New join governance checks
			if baseHasTenant || tenantScoped {
				if s, ok := extJoin["sql"].(string); ok && !containsTenantGuard(s) {
					issues = append(issues, issue("warning", "TENANT_GUARD_MISSING", fmt.Sprintf("join '%s': sql appears to lack tenant_id guard", name), nil))
				}
			}
		}
	}
	// Access policy row filters must include tenant guard when tenant-scoped
	if tenantScoped && ext.AccessPolicy != nil {
		if rf, ok := ext.AccessPolicy["row_filter"].(string); ok && rf != "" && !containsTenantGuard(rf) {
			issues = append(issues, issue("warning", "TENANT_ROW_FILTER_MISSING", "access_policy.row_filter appears to lack tenant_id guard", nil))
		}
	}
	return issues
}

// ComposeCatalog merges core and extension cubes into a final map of cubes.
// - coreCubes: authoratative generated cubes, marked read_only in Metadata if desired.
// - extCubes: user-defined extensions, each should set Extends to a string of the base cube name.
// Returns final cubes ready for query consumption (merged), sorted by name in deterministic order.
func ComposeCatalog(coreCubes map[string]Cube, extCubes map[string]Cube) (map[string]Cube, []ValidationIssue) {
	out := map[string]Cube{}
	issues := []ValidationIssue{}

	if coreCubes == nil {
		coreCubes = map[string]Cube{}
	}
	if extCubes == nil {
		extCubes = map[string]Cube{}
	}

	// Index core cubes for fast lookup
	uuidIndex := map[uuid.UUID]string{}
	outByNormalizedName := map[string]string{}
	outByModelKey := map[string]string{}

	for name, c := range coreCubes {
		clone := cloneCube(c)
		out[name] = clone
		norm := normalizeCubeName(name)
		if norm != "" {
			outByNormalizedName[norm] = name
		}
		if mk := extractModelKey(clone.Metadata); mk != "" {
			outByModelKey[normalizeCubeName(mk)] = name
		}
		if clone.FabricDefnID != nil {
			uuidIndex[*clone.FabricDefnID] = name
		}
	}

	// Prepare extension indexes
	pending := map[string]Cube{}
	pendingByNormalizedName := map[string]string{}
	pendingByModelKey := map[string]string{}
	for name, ext := range extCubes {
		pending[name] = ext
		norm := normalizeCubeName(name)
		if norm != "" {
			pendingByNormalizedName[norm] = name
		}
		if mk := extractModelKey(ext.Metadata); mk != "" {
			pendingByModelKey[normalizeCubeName(mk)] = name
		}
	}

	// Resolve extensions; allow multi-pass to satisfy dependency ordering
	for len(pending) > 0 {
		progress := false
		for name, ext := range pending {
			base, found, baseHint, waiting, extraIssues := resolveBaseForExtension(ext, out, uuidIndex, outByNormalizedName, outByModelKey, pending, pendingByNormalizedName, pendingByModelKey)
			if len(extraIssues) > 0 {
				issues = append(issues, extraIssues...)
			}
			if !found {
				if waiting {
					// Dependency not ready yet; defer to later pass
					continue
				}
				hint := baseHint
				if hint == "" && ext.Extends != nil {
					if s, ok := ext.Extends.(string); ok {
						hint = s
					}
				}
				issues = append(issues, issue("error", "BASE_NOT_FOUND", fmt.Sprintf("extension '%s' references missing base '%s'", ext.Name, hint), map[string]any{"cube": ext.Name}))
				// Drop to avoid infinite loop
				normName := normalizeCubeName(name)
				delete(pending, name)
				if normName != "" {
					delete(pendingByNormalizedName, normName)
				}
				if mk := extractModelKey(ext.Metadata); mk != "" {
					delete(pendingByModelKey, normalizeCubeName(mk))
				}
				progress = true
				continue
			}
			// Validate + merge
			if errs := ValidateExtension(base, ext); len(errs) > 0 {
				issues = append(issues, errs...)
			}
			merged, warns := MergeCube(base, ext)
			if len(warns) > 0 {
				issues = append(issues, warns...)
			}
			pubName := ext.Name
			if pubName == "" {
				pubName = base.Name + "_ext"
			}
			merged.Name = pubName
			if merged.Metadata == nil {
				merged.Metadata = map[string]any{}
			}
			merged.Metadata["last_merged"] = time.Now().UTC().Format(time.RFC3339)
			out[pubName] = merged
			normPub := normalizeCubeName(pubName)
			if normPub != "" {
				outByNormalizedName[normPub] = pubName
			}
			if mk := extractModelKey(merged.Metadata); mk != "" {
				outByModelKey[normalizeCubeName(mk)] = pubName
			}
			if merged.FabricDefnID != nil {
				uuidIndex[*merged.FabricDefnID] = pubName
			}
			// Remove from pending structures
			normName := normalizeCubeName(name)
			delete(pending, name)
			if normName != "" {
				delete(pendingByNormalizedName, normName)
			}
			if mk := extractModelKey(ext.Metadata); mk != "" {
				delete(pendingByModelKey, normalizeCubeName(mk))
			}
			progress = true
		}
		if !progress {
			// Remaining items could not be resolved due to cycles or missing bases
			for name, ext := range pending {
				names, models, _ := collectExtendsCandidates(ext)
				hint := ""
				if len(names) > 0 {
					hint = names[0]
				} else if len(models) > 0 {
					hint = models[0]
				}
				issues = append(issues, issue("error", "EXTENDS_RESOLUTION_FAILED", fmt.Sprintf("extension '%s' could not resolve base '%s' (possible cycle or missing model)", name, hint), map[string]any{"cube": name}))
			}
			break
		}
	}

	// Make deterministic: rebuild into sorted map order by name
	if len(out) > 0 {
		names := make([]string, 0, len(out))
		for n := range out {
			names = append(names, n)
		}
		sort.Strings(names)
		stable := map[string]Cube{}
		for _, n := range names {
			stable[n] = out[n]
		}
		out = stable
	}
	return out, issues
}

func resolveBaseForExtension(ext Cube, out map[string]Cube, uuidIndex map[uuid.UUID]string, outByNormalizedName map[string]string, outByModelKey map[string]string, pending map[string]Cube, pendingByNormalizedName map[string]string, pendingByModelKey map[string]string) (Cube, bool, string, bool, []ValidationIssue) {
	names, modelKeys, uuids := collectExtendsCandidates(ext)
	issues := []ValidationIssue{}
	baseHint := ""
	if len(names) > 0 {
		baseHint = names[0]
	} else if len(modelKeys) > 0 {
		baseHint = modelKeys[0]
	}

	// UUID resolution
	for _, id := range uuids {
		if id == nil {
			continue
		}
		if baseName, ok := uuidIndex[*id]; ok {
			if baseCube, ok2 := out[baseName]; ok2 {
				return baseCube, true, baseName, false, issues
			}
		}
	}

	// Direct / normalized name lookups
	for _, candidate := range names {
		if candidate == "" {
			continue
		}
		if baseCube, ok := out[candidate]; ok {
			return baseCube, true, candidate, false, issues
		}
		norm := normalizeCubeName(candidate)
		if actualName, ok := outByNormalizedName[norm]; ok {
			return out[actualName], true, actualName, false, issues
		}
		if _, ok := pending[candidate]; ok {
			return Cube{}, false, candidate, true, issues
		}
		if pendingName, ok := pendingByNormalizedName[norm]; ok {
			return Cube{}, false, pendingName, true, issues
		}
	}

	// Model key lookups
	for _, mk := range modelKeys {
		if mk == "" {
			continue
		}
		norm := normalizeCubeName(mk)
		if actualName, ok := outByModelKey[norm]; ok {
			return out[actualName], true, actualName, false, issues
		}
		if pendingName, ok := pendingByModelKey[norm]; ok {
			return Cube{}, false, pendingName, true, issues
		}
	}

	return Cube{}, false, baseHint, false, issues
}

func collectExtendsCandidates(ext Cube) (names []string, modelKeys []string, uuids []*uuid.UUID) {
	nameSeen := map[string]struct{}{}
	modelSeen := map[string]struct{}{}
	var uuidSeen []uuid.UUID

	var addName func(string)
	addModelKey := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := modelSeen[s]; !ok {
			modelSeen[s] = struct{}{}
			modelKeys = append(modelKeys, s)
		}
	}

	addName = func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := nameSeen[s]; !ok {
			nameSeen[s] = struct{}{}
			names = append(names, s)
		}
		trim := strings.TrimPrefix(s, "/")
		trim = strings.TrimSpace(trim)
		if trim != "" {
			addModelKey(trim)
			if idx := strings.LastIndex(trim, "."); idx >= 0 && idx+1 < len(trim) {
				addModelKey(trim[idx+1:])
			}
			if idx := strings.LastIndex(trim, "/"); idx >= 0 && idx+1 < len(trim) {
				addModelKey(trim[idx+1:])
			}
		}
	}

	addUUID := func(val any) {
		s, ok := val.(string)
		if !ok {
			return
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if parsed, err := uuid.Parse(s); err == nil {
			for _, existing := range uuidSeen {
				if existing == parsed {
					return
				}
			}
			uuidSeen = append(uuidSeen, parsed)
			parsedCopy := parsed
			uuids = append(uuids, &parsedCopy)
		}
	}

	if ext.Extends != nil {
		switch v := ext.Extends.(type) {
		case string:
			addName(v)
		case []any:
			for _, item := range v {
				switch t := item.(type) {
				case string:
					addName(t)
				case map[string]any:
					if name, ok := t["name"].(string); ok {
						addName(name)
					}
					if mk, ok := t["model_key"].(string); ok {
						addModelKey(mk)
					}
					if mk, ok := t["model"].(string); ok {
						addModelKey(mk)
					}
					addUUID(t["fabric_defn_id"])
				}
			}
		case map[string]any:
			if name, ok := v["name"].(string); ok {
				addName(name)
			}
			if mk, ok := v["model_key"].(string); ok {
				addModelKey(mk)
			}
			if mk, ok := v["model"].(string); ok {
				addModelKey(mk)
			}
			addUUID(v["fabric_defn_id"])
		}
	}

	if ext.Metadata != nil {
		if name, ok := ext.Metadata["inherits_from"].(string); ok {
			addName(name)
		}
		if name, ok := ext.Metadata["base_cube_name"].(string); ok {
			addName(name)
		}
		if mk, ok := ext.Metadata["parent_model_key"].(string); ok {
			addModelKey(mk)
		}
		if mk, ok := ext.Metadata["base_model_key"].(string); ok {
			addModelKey(mk)
		}
		if mk, ok := ext.Metadata["extends_model_key"].(string); ok {
			addModelKey(mk)
		}
	}

	if ext.Meta != nil {
		if mk, ok := ext.Meta["parent_model_key"].(string); ok {
			addModelKey(mk)
		}
	}

	return names, modelKeys, uuids
}

func normalizeCubeName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.TrimPrefix(name, "/")
	name = strings.TrimPrefix(name, "cube:")
	return name
}

func extractModelKey(meta map[string]any) string {
	if meta == nil {
		return ""
	}
	if mk, ok := meta["model_key"].(string); ok && mk != "" {
		return mk
	}
	if mk, ok := meta["parent_model_key"].(string); ok && mk != "" {
		return mk
	}
	if mk, ok := meta["base_model_key"].(string); ok && mk != "" {
		return mk
	}
	if mk, ok := meta["extends_model_key"].(string); ok && mk != "" {
		return mk
	}
	if mk, ok := meta["model"].(string); ok && mk != "" {
		return mk
	}
	return ""
}

func collectInheritanceChainFromMetadata(meta map[string]any) []string {
	if meta == nil {
		return nil
	}
	if chain, ok := meta["inheritance_chain"].([]string); ok {
		return append([]string{}, chain...)
	}
	if chainAny, ok := meta["inheritance_chain"].([]any); ok {
		out := make([]string, 0, len(chainAny))
		for _, v := range chainAny {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// Helpers

func issue(level, code, msg string, details map[string]any) ValidationIssue {
	return ValidationIssue{Level: level, Code: code, Message: msg, Details: details}
}

func cloneCube(in Cube) Cube {
	out := in
	out.Dimensions = cloneNestedMap(in.Dimensions)
	out.Measures = cloneNestedMap(in.Measures)
	out.Joins = cloneNestedMap(in.Joins)
	out.Segments = cloneNestedMap(in.Segments)
	out.PreAggregations = cloneNestedMap(in.PreAggregations)
	out.Meta = cloneStringAnyMap(in.Meta)
	out.AccessPolicy = cloneStringAnyMap(in.AccessPolicy)
	out.RefreshKey = cloneStringAnyMap(in.RefreshKey)
	out.Hierarchies = cloneSliceMap(in.Hierarchies)
	out.Metadata = cloneStringAnyMap(in.Metadata)
	if in.Public != nil {
		p := *in.Public
		out.Public = &p
	}
	out.Tags = append([]string{}, in.Tags...)
	return out
}

func cloneNestedMap(in map[string]map[string]any) map[string]map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]map[string]any, len(in))
	for k, v := range in {
		out[k] = cloneStringAnyMap(v)
	}
	return out
}

func cloneSliceMap(in []map[string]any) []map[string]any {
	if in == nil {
		return nil
	}
	out := make([]map[string]any, 0, len(in))
	for _, m := range in {
		out = append(out, cloneStringAnyMap(m))
	}
	return out
}

func cloneStringAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = deepCopy(v)
	}
	return out
}

func deepCopy(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return cloneStringAnyMap(t)
	case []any:
		out := make([]any, len(t))
		for i := range t {
			out[i] = deepCopy(t[i])
		}
		return out
	default:
		return t
	}
}

func mergeStringAny(a, b map[string]any) map[string]any {
	if a == nil && b == nil {
		return nil
	}
	if a == nil {
		return cloneStringAnyMap(b)
	}
	if b == nil {
		return cloneStringAnyMap(a)
	}
	out := cloneStringAnyMap(a)
	for k, v := range b {
		out[k] = deepCopy(v)
	}
	return out
}

func mergeNestedMap(a, b map[string]map[string]any) map[string]map[string]any {
	if a == nil && b == nil {
		return nil
	}
	if a == nil {
		return cloneNestedMap(b)
	}
	if b == nil {
		return cloneNestedMap(a)
	}
	out := cloneNestedMap(a)
	for k, v := range b {
		if base, ok := out[k]; ok {
			out[k] = mergeStringAnyMap(base, v)
		} else {
			out[k] = cloneStringAnyMap(v)
		}
	}
	return out
}

func mergeStringAnyMap(a, b map[string]any) map[string]any {
	out := cloneStringAnyMap(a)
	for k, v := range b {
		av, ok := out[k]
		if ok && isMap(av) && isMap(v) {
			out[k] = mergeStringAny(av.(map[string]any), v.(map[string]any))
		} else {
			out[k] = deepCopy(v)
		}
	}
	return out
}

func isMap(v any) bool { _, ok := v.(map[string]any); return ok }

func changedPrimaryKey(base map[string]any, ext map[string]any) bool {
	_, baseHas := base["primary_key"]
	_, extHas := ext["primary_key"]
	if !baseHas && !extHas {
		return false
	}
	if baseHas != extHas {
		return true
	}
	// If both have, compare truthiness
	return !reflect.DeepEqual(base["primary_key"], ext["primary_key"])
}

func riskyKeys(m map[string]any) []string {
	keys := []string{}
	for _, k := range []string{"sql", "type", "relationship"} {
		if _, ok := m[k]; ok {
			keys = append(keys, k)
		}
	}
	return keys
}

func sortedKeys(m map[string]any) []string {
	if m == nil {
		return nil
	}
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func containsTenantGuard(sql string) bool {
	s := strings.ToLower(sql)
	return strings.Contains(s, "tenant_id")
}

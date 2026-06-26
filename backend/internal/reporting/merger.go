package reporting

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DefinitionMerger handles merging core definitions with extensions
type DefinitionMerger struct{}

// NewDefinitionMerger creates a new merger
func NewDefinitionMerger() *DefinitionMerger {
	return &DefinitionMerger{}
}

// Merge combines a core report definition with an extension
func (m *DefinitionMerger) Merge(core *ReportLayout, ext *ReportExtension) (*ReportLayout, error) {
	if core == nil {
		return nil, fmt.Errorf("core definition is required")
	}

	// If no extension, return core as-is
	if ext == nil {
		return core, nil
	}

	// Deep copy the core definition
	merged, err := deepCopy(core)
	if err != nil {
		return nil, fmt.Errorf("failed to copy core definition: %w", err)
	}

	// Apply overrides
	if len(ext.Overrides) > 0 {
		if err := m.applyOverrides(merged, ext.Overrides); err != nil {
			return nil, fmt.Errorf("failed to apply overrides: %w", err)
		}
	}

	// Apply additions
	if len(ext.Additions) > 0 {
		if err := m.applyAdditions(merged, ext.Additions); err != nil {
			return nil, fmt.Errorf("failed to apply additions: %w", err)
		}
	}

	// Apply removals
	if len(ext.Removals) > 0 {
		if err := m.applyRemovals(merged, ext.Removals); err != nil {
			return nil, fmt.Errorf("failed to apply removals: %w", err)
		}
	}

	// Apply parameter defaults
	if len(ext.ParameterDefaults) > 0 {
		if err := m.applyParameterDefaults(merged, ext.ParameterDefaults); err != nil {
			return nil, fmt.Errorf("failed to apply parameter defaults: %w", err)
		}
	}

	return merged, nil
}

// applyOverrides applies field-level overrides using dot notation paths
func (m *DefinitionMerger) applyOverrides(layout *ReportLayout, overrides json.RawMessage) error {
	var overrideMap map[string]interface{}
	if err := json.Unmarshal(overrides, &overrideMap); err != nil {
		return err
	}

	// Convert layout to map for easy path-based updates
	layoutJSON, err := json.Marshal(layout)
	if err != nil {
		return err
	}

	var layoutMap map[string]interface{}
	if err := json.Unmarshal(layoutJSON, &layoutMap); err != nil {
		return err
	}

	// Apply each override
	for path, value := range overrideMap {
		if err := setNestedValue(layoutMap, path, value); err != nil {
			return fmt.Errorf("failed to set %s: %w", path, err)
		}
	}

	// Convert back to ReportLayout
	updatedJSON, err := json.Marshal(layoutMap)
	if err != nil {
		return err
	}

	return json.Unmarshal(updatedJSON, layout)
}

// applyAdditions adds new sections/elements to the layout
func (m *DefinitionMerger) applyAdditions(layout *ReportLayout, additions json.RawMessage) error {
	var addMap map[string]interface{}
	if err := json.Unmarshal(additions, &addMap); err != nil {
		return err
	}

	// Handle section additions
	if sectionsRaw, ok := addMap["layout.body.sections"]; ok {
		sectionsJSON, err := json.Marshal(sectionsRaw)
		if err != nil {
			return err
		}

		var newSections []ReportSection
		if err := json.Unmarshal(sectionsJSON, &newSections); err != nil {
			return err
		}

		// Insert sections at appropriate positions
		for _, newSection := range newSections {
			inserted := false

			if newSection.InsertAfter != "" {
				// Find position to insert after
				for i, existing := range layout.Layout.Body.Sections {
					if existing.ID == newSection.InsertAfter {
						// Insert after this position
						layout.Layout.Body.Sections = insertSection(layout.Layout.Body.Sections, i+1, newSection)
						inserted = true
						break
					}
				}
			}

			if !inserted {
				// Append to end
				layout.Layout.Body.Sections = append(layout.Layout.Body.Sections, newSection)
			}
		}
	}

	// Handle parameter additions
	if paramsRaw, ok := addMap["parameters"]; ok {
		paramsJSON, err := json.Marshal(paramsRaw)
		if err != nil {
			return err
		}

		var newParams []Parameter
		if err := json.Unmarshal(paramsJSON, &newParams); err != nil {
			return err
		}

		layout.Parameters = append(layout.Parameters, newParams...)
	}

	// Handle data binding additions
	if bindingsRaw, ok := addMap["dataBindings"]; ok {
		bindingsJSON, err := json.Marshal(bindingsRaw)
		if err != nil {
			return err
		}

		var newBindings map[string]DataBinding
		if err := json.Unmarshal(bindingsJSON, &newBindings); err != nil {
			return err
		}

		if layout.DataBindings == nil {
			layout.DataBindings = make(map[string]DataBinding)
		}

		for key, binding := range newBindings {
			layout.DataBindings[key] = binding
		}
	}

	return nil
}

// applyRemovals removes sections/elements from the layout
func (m *DefinitionMerger) applyRemovals(layout *ReportLayout, removals json.RawMessage) error {
	var removeMap map[string]interface{}
	if err := json.Unmarshal(removals, &removeMap); err != nil {
		return err
	}

	// Handle section removals
	if sectionsRaw, ok := removeMap["layout.body.sections"]; ok {
		var sectionIDs []string

		switch v := sectionsRaw.(type) {
		case []interface{}:
			for _, id := range v {
				if strID, ok := id.(string); ok {
					sectionIDs = append(sectionIDs, strID)
				}
			}
		case []string:
			sectionIDs = v
		}

		// Filter out removed sections
		filtered := make([]ReportSection, 0)
		for _, section := range layout.Layout.Body.Sections {
			if !contains(sectionIDs, section.ID) {
				filtered = append(filtered, section)
			}
		}
		layout.Layout.Body.Sections = filtered
	}

	// Handle parameter removals
	if paramsRaw, ok := removeMap["parameters"]; ok {
		var paramNames []string

		switch v := paramsRaw.(type) {
		case []interface{}:
			for _, name := range v {
				if strName, ok := name.(string); ok {
					paramNames = append(paramNames, strName)
				}
			}
		case []string:
			paramNames = v
		}

		// Filter out removed parameters
		filtered := make([]Parameter, 0)
		for _, param := range layout.Parameters {
			if !contains(paramNames, param.Name) {
				filtered = append(filtered, param)
			}
		}
		layout.Parameters = filtered
	}

	return nil
}

// applyParameterDefaults applies default value overrides for parameters
func (m *DefinitionMerger) applyParameterDefaults(layout *ReportLayout, defaults json.RawMessage) error {
	var defaultMap map[string]interface{}
	if err := json.Unmarshal(defaults, &defaultMap); err != nil {
		return err
	}

	for i := range layout.Parameters {
		if newDefault, ok := defaultMap[layout.Parameters[i].Name]; ok {
			layout.Parameters[i].Default = newDefault
		}
	}

	return nil
}

// ============================================================================
// HELPERS
// ============================================================================

// deepCopy creates a deep copy of a ReportLayout
func deepCopy(src *ReportLayout) (*ReportLayout, error) {
	data, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}

	var dst ReportLayout
	if err := json.Unmarshal(data, &dst); err != nil {
		return nil, err
	}

	return &dst, nil
}

// setNestedValue sets a value at a dot-notation path in a map
func setNestedValue(m map[string]interface{}, path string, value interface{}) error {
	parts := strings.Split(path, ".")
	current := m

	for i, part := range parts[:len(parts)-1] {
		// Handle array notation like "elements[0]"
		if bracketIdx := strings.Index(part, "["); bracketIdx != -1 {
			key := part[:bracketIdx]
			indexStr := part[bracketIdx+1 : len(part)-1]
			var index int
			fmt.Sscanf(indexStr, "%d", &index)

			if arr, ok := current[key].([]interface{}); ok {
				if index < len(arr) {
					if nextMap, ok := arr[index].(map[string]interface{}); ok {
						current = nextMap
						continue
					}
				}
			}
			return fmt.Errorf("invalid path at %s", strings.Join(parts[:i+1], "."))
		}

		next, ok := current[part]
		if !ok {
			// Create nested map if it doesn't exist
			current[part] = make(map[string]interface{})
			next = current[part]
		}

		if nextMap, ok := next.(map[string]interface{}); ok {
			current = nextMap
		} else {
			return fmt.Errorf("path %s is not a map at %s", path, part)
		}
	}

	// Handle array notation in final part
	finalPart := parts[len(parts)-1]
	if bracketIdx := strings.Index(finalPart, "["); bracketIdx != -1 {
		key := finalPart[:bracketIdx]
		indexStr := finalPart[bracketIdx+1 : len(finalPart)-1]
		var index int
		fmt.Sscanf(indexStr, "%d", &index)

		if arr, ok := current[key].([]interface{}); ok {
			if index < len(arr) {
				arr[index] = value
				return nil
			}
		}
		return fmt.Errorf("invalid array index in path %s", path)
	}

	current[finalPart] = value
	return nil
}

// insertSection inserts a section at the given index
func insertSection(sections []ReportSection, index int, section ReportSection) []ReportSection {
	if index >= len(sections) {
		return append(sections, section)
	}

	sections = append(sections[:index+1], sections[index:]...)
	sections[index] = section
	return sections
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// MergedReport represents a fully merged report ready for rendering
type MergedReport struct {
	Definition     *ReportLayout          `json:"definition"`
	CoreVersion    int                    `json:"core_version"`
	ExtensionKey   string                 `json:"extension_key,omitempty"`
	ExtVersion     int                    `json:"extension_version,omitempty"`
	AppliedChanges map[string]interface{} `json:"applied_changes,omitempty"`
}

// GetMergedReport returns a merged report with metadata about what was changed
func (m *DefinitionMerger) GetMergedReport(core *ReportLayout, coreVersion int, ext *ReportExtension) (*MergedReport, error) {
	merged, err := m.Merge(core, ext)
	if err != nil {
		return nil, err
	}

	result := &MergedReport{
		Definition:  merged,
		CoreVersion: coreVersion,
	}

	if ext != nil {
		result.ExtensionKey = ext.ExtensionKey
		result.ExtVersion = ext.Version

		// Document what changed
		changes := make(map[string]interface{})
		if len(ext.Overrides) > 0 {
			var overrides map[string]interface{}
			json.Unmarshal(ext.Overrides, &overrides)
			changes["overrides"] = overrides
		}
		if len(ext.Additions) > 0 {
			var additions map[string]interface{}
			json.Unmarshal(ext.Additions, &additions)
			changes["additions"] = additions
		}
		if len(ext.Removals) > 0 {
			var removals map[string]interface{}
			json.Unmarshal(ext.Removals, &removals)
			changes["removals"] = removals
		}

		if len(changes) > 0 {
			result.AppliedChanges = changes
		}
	}

	return result, nil
}

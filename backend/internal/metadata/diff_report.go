package metadata

// DiffReport provides a structured summary of metadata changes between versions
type DiffReport struct {
	Summary   DiffSummary `json:"summary"`
	Changes   []Change    `json:"changes"`
	Risks     []Risk      `json:"risks"`
	Generated string      `json:"generated_at"`
}

// DiffSummary aggregates change statistics
type DiffSummary struct {
	TotalChanges    int `json:"total_changes"`
	AdditiveChanges int `json:"additive_changes"`
	BreakingChanges int `json:"breaking_changes"`
	SafeChanges     int `json:"safe_changes"`
}

// Change represents a single metadata modification
type Change struct {
	Type     ChangeType     `json:"type"`
	Path     string         `json:"path"`       // e.g., "BusinessObject.ClientProfile.attributes.risk_score"
	Severity ChangeSeverity `json:"severity"`
	OldValue interface{}    `json:"old_value,omitempty"`
	NewValue interface{}    `json:"new_value,omitempty"`
	Impact   string         `json:"impact"` // Human-readable explanation
}

// ChangeSeverity indicates the risk level of a change
type ChangeSeverity string

const (
	ChangeSeverityAdditive ChangeSeverity = "ADDITIVE" // Safe, backwards-compatible
	ChangeSeverityBreaking ChangeSeverity = "BREAKING" // May require manual intervention
	ChangeSeveritySafe     ChangeSeverity = "SAFE"     // No impact on existing data
)

// Risk represents a potential upgrade hazard
type Risk struct {
	Severity       RiskLevel `json:"severity"`
	Description    string    `json:"description"`
	Mitigation     string    `json:"mitigation"`
	RelatedChanges []string  `json:"related_changes,omitempty"` // Paths to changes
}

// RiskLevel categorizes risk severity
type RiskLevel string

const (
	RiskLevelHigh   RiskLevel = "HIGH"
	RiskLevelMedium RiskLevel = "MEDIUM"
	RiskLevelLow    RiskLevel = "LOW"
)

// GenerateDiffReport compares two business object versions and produces a structured report
func GenerateDiffReport(oldBO, newBO BusinessObject) *DiffReport {
	changes := detectChanges(oldBO, newBO)
	risks := assessRisks(changes)

	summary := DiffSummary{
		TotalChanges: len(changes),
	}

	for _, change := range changes {
		switch change.Severity {
		case ChangeSeverityAdditive:
			summary.AdditiveChanges++
		case ChangeSeverityBreaking:
			summary.BreakingChanges++
		case ChangeSeveritySafe:
			summary.SafeChanges++
		}
	}

	return &DiffReport{
		Summary: summary,
		Changes: changes,
		Risks:   risks,
	}
}

// detectChanges compares two business objects and identifies differences
func detectChanges(oldBO, newBO BusinessObject) []Change {
	var changes []Change

	// Compare attributes
	oldAttrsMap := make(map[string]BOAttribute)
	for _, attr := range oldBO.Attributes {
		oldAttrsMap[attr.Name] = attr
	}

	newAttrsMap := make(map[string]BOAttribute)
	for _, attr := range newBO.Attributes {
		newAttrsMap[attr.Name] = attr
	}

	// Detect added attributes
	for name, newAttr := range newAttrsMap {
		if _, exists := oldAttrsMap[name]; !exists {
			severity := ChangeSeverityAdditive
			if newAttr.Required {
				severity = ChangeSeverityBreaking
			}
			changes = append(changes, Change{
				Type:     ChangeTypeAttributeAdded,
				Path:     "BusinessObject." + oldBO.Meta.Name + ".attributes." + name,
				Severity: severity,
				NewValue: newAttr,
				Impact:   "New attribute added. Existing queries unaffected unless required.",
			})
		}
	}

	// Detect removed attributes
	for name, oldAttr := range oldAttrsMap {
		if _, exists := newAttrsMap[name]; !exists {
			changes = append(changes, Change{
				Type:     ChangeTypeAttributeRemoved,
				Path:     "BusinessObject." + oldBO.Meta.Name + ".attributes." + name,
				Severity: ChangeSeverityBreaking,
				OldValue: oldAttr,
				Impact:   "Attribute removed. Queries referencing this attribute will fail.",
			})
		}
	}

	// Detect modified attributes
	for name, oldAttr := range oldAttrsMap {
		if newAttr, exists := newAttrsMap[name]; exists {
			if oldAttr.Type != newAttr.Type {
				changes = append(changes, Change{
					Type:     ChangeTypeAttributeChanged,
					Path:     "BusinessObject." + oldBO.Meta.Name + ".attributes." + name + ".type",
					Severity: ChangeSeverityBreaking,
					OldValue: oldAttr.Type,
					NewValue: newAttr.Type,
					Impact:   "Data type changed. May require data migration.",
				})
			}

			// Compare validations
			if oldAttr.Validation != newAttr.Validation {
				oldVal := ""
				if oldAttr.Validation != nil {
					oldVal = *oldAttr.Validation
				}
				newVal := ""
				if newAttr.Validation != nil {
					newVal = *newAttr.Validation
				}

				if oldVal != newVal {
					changes = append(changes, Change{
						Type:     ChangeTypeAttributeChanged,
						Path:     "BusinessObject." + oldBO.Meta.Name + ".attributes." + name + ".validation",
						Severity: ChangeSeverityBreaking,
						OldValue: oldVal,
						NewValue: newVal,
						Impact:   "Validation rules changed. Existing data may become invalid.",
					})
				}
			}
		}
	}

	return changes
}

// assessRisks evaluates changes and identifies potential upgrade hazards
func assessRisks(changes []Change) []Risk {
	var risks []Risk

	breakingChangeCount := 0
	var breakingChangePaths []string

	for _, change := range changes {
		if change.Severity == ChangeSeverityBreaking {
			breakingChangeCount++
			breakingChangePaths = append(breakingChangePaths, change.Path)
		}
	}

	if breakingChangeCount > 0 {
		risks = append(risks, Risk{
			Severity:       RiskLevelHigh,
			Description:    "Breaking changes detected",
			Mitigation:     "Review all breaking changes and ensure backward compatibility or implement migration scripts.",
			RelatedChanges: breakingChangePaths,
		})
	}

	if breakingChangeCount > 5 {
		risks = append(risks, Risk{
			Severity:       RiskLevelHigh,
			Description:    "Large number of breaking changes may indicate significant refactoring",
			Mitigation:     "Consider phased rollout or provide comprehensive migration documentation.",
			RelatedChanges: breakingChangePaths,
		})
	}

	return risks
}

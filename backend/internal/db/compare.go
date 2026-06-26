package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nsf/jsondiff"
)

// ChangeSeverity defines the impact level of a change.
type ChangeSeverity string

const (
	// SeverityLow indicates a non-disruptive, additive change.
	SeverityLow ChangeSeverity = "low"
	// SeverityMedium indicates a modification that may require attention.
	SeverityMedium ChangeSeverity = "medium"
	// SeverityBreaking indicates a destructive change that will likely break downstream dependencies.
	SeverityBreaking ChangeSeverity = "breaking"
)

// ChangeImpact defines the domain of a change's effect.
type ChangeImpact string

const (
	ImpactSchema      ChangeImpact = "schema"
	ImpactData        ChangeImpact = "data"
	ImpactBehavior    ChangeImpact = "behavior"
	ImpactPerformance ChangeImpact = "performance"
)

// ChangeMetadata holds specific details about a modification for richer explanations.
type ChangeMetadata struct {
	OldType       string `json:"old_type,omitempty"`
	NewType       string `json:"new_type,omitempty"`
	OldConstraint string `json:"old_constraint,omitempty"`
	NewConstraint string `json:"new_constraint,omitempty"`
	TriggerBody   string `json:"trigger_body,omitempty"`
}

// Change represents a single difference between two model snapshots.
type Change struct {
	NodeType      string         `json:"node_type"`
	QualifiedPath string         `json:"qualified_path"`
	ChangeType    string         `json:"change_type"` // "added", "removed", "modified"
	Severity      ChangeSeverity `json:"severity"`
	Details       string         `json:"details"`
	PropDiff      string         `json:"prop_diff,omitempty"`
	EdgeDiff      string         `json:"edge_diff,omitempty"`
	Impacts       []ChangeImpact `json:"impacts,omitempty"`
	Explanation   string         `json:"explanation,omitempty"`
	Metadata      ChangeMetadata `json:"metadata,omitempty"`
}

// ToolInfo captures the build information of the CLI tool.
type ToolInfo struct {
	Version   string `json:"version,omitempty"`
	Commit    string `json:"commit,omitempty"`
	BuildDate string `json:"build_date,omitempty"`
}

// RunInfo captures details about the specific execution, including CLI arguments.
type RunInfo struct {
	Args     []string          `json:"args"`
	CWD      string            `json:"cwd"`
	EnvFlags map[string]string `json:"env_flags,omitempty"`
}

// ChangeReport wraps the list of changes with a summary and groups for easier consumption.
type ChangeReport struct {
	Summary     map[ChangeSeverity]int      `json:"summary"`
	Groups      map[ChangeSeverity][]Change `json:"groups"`
	AllChanges  []Change                    `json:"all_changes,omitempty"`
	GeneratedAt time.Time                   `json:"generated_at"`
	Tool        ToolInfo                    `json:"tool"`
	Run         RunInfo                     `json:"run"`
	SchemaHash  string                      `json:"schema_hash,omitempty"`
}

// jsonDiff compares two JSON raw messages and returns a human-readable diff report.
func jsonDiff(a, b json.RawMessage) string {
	// Ensure nil slices are treated as empty JSON "null" for comparison.
	if a == nil {
		a = []byte("null")
	}
	if b == nil {
		b = []byte("null")
	}

	opts := jsondiff.DefaultConsoleOptions()
	diff, report := jsondiff.Compare(a, b, &opts)

	if diff == jsondiff.FullMatch {
		return ""
	}
	return report
}

// HashSchemaState calculates a deterministic hash of a given schema snapshot.
func HashSchemaState(snapshot []Node) string {
	// Sort nodes to ensure a canonical representation.
	// This makes the hash insensitive to the order of nodes in the snapshot.
	sort.Slice(snapshot, func(i, j int) bool {
		if snapshot[i].TypeName == snapshot[j].TypeName {
			return snapshot[i].QualifiedPath < snapshot[j].QualifiedPath
		}
		return snapshot[i].TypeName < snapshot[j].TypeName
	})

	// Marshal the canonical snapshot to JSON.
	buf, err := json.Marshal(snapshot)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal snapshot for hashing: %v", err))
	}

	sum := sha256.Sum256(buf)
	// Return a shortened fingerprint as requested.
	return hex.EncodeToString(sum[:])[:16]
}

// HashNode calculates a deterministic hash for a node's content.
func HashNode(n Node) string {
	// The hash should be based on content, not identity (like ID) or the hash itself.
	type HashableNode struct {
		NodeName      string          `json:"node_name"`
		Properties    json.RawMessage `json:"properties"`
		Edges         json.RawMessage `json:"edges"`
		TypeName      string          `json:"type_name"`
		QualifiedPath string          `json:"qualified_path"`
	}

	h := HashableNode{
		NodeName:      n.NodeName,
		Properties:    n.Properties,
		Edges:         n.Edges,
		TypeName:      n.TypeName,
		QualifiedPath: n.QualifiedPath,
	}

	// Marshal for hashing. In a production system, you might use a canonical JSON library.
	data, err := json.Marshal(h)
	if err != nil {
		// This should ideally not happen with a controlled struct.
		return ""
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// MergeOverlay combines a core snapshot with a tenant's custom snapshot.
// Tenant nodes with a `core_id` will override the corresponding core node.
func MergeOverlay(coreSnap, tenantSnap []Node) []Node {
	// Index core nodes by their ID for quick lookups.
	coreIndex := make(map[uuid.UUID]Node)
	for _, n := range coreSnap {
		coreIndex[n.ID] = n
	}

	// Start with the core snapshot as the base.
	effective := make(map[uuid.UUID]Node)
	for _, n := range coreSnap {
		effective[n.ID] = n
	}

	// Apply tenant overrides and add purely custom nodes.
	for _, tn := range tenantSnap {
		if tn.CoreID != nil {
			// This is an overlay on a core node.
			if base, ok := coreIndex[*tn.CoreID]; ok {
				merged := base // Start with the core node's properties.
				// Apply tenant-specific overrides.
				if tn.NodeName != "" {
					merged.NodeName = tn.NodeName
				}
				if len(tn.Properties) > 0 {
					merged.Properties = tn.Properties
				}
				if len(tn.Edges) > 0 {
					merged.Edges = tn.Edges
				}
				merged.CanonicalHash = HashNode(merged) // Recalculate hash after merging.
				effective[base.ID] = merged
				continue
			}
		}
		// This is a purely custom node (not linked to a core node).
		effective[tn.ID] = tn
	}

	// Flatten the map back to a slice for consistent processing.
	var out []Node
	for _, n := range effective {
		out = append(out, n)
	}

	// Sort for deterministic output.
	sort.Slice(out, func(i, j int) bool {
		if out[i].TypeName == out[j].TypeName {
			return out[i].QualifiedPath < out[j].QualifiedPath
		}
		return out[i].TypeName < out[j].TypeName
	})
	return out
}

// classifyModification determines the severity of a change between two versions of a node.
func classifyModification(oldNode, newNode Node) ChangeSeverity {
	// This is a simple classification. You can expand this with more detailed logic,
	// for example, by performing a deep diff on the Properties JSON.
	if string(oldNode.Properties) != string(newNode.Properties) {
		return SeverityMedium
	}
	if len(oldNode.Edges) != len(newNode.Edges) {
		return SeverityMedium
	}
	// If only minor fields changed, it might be low severity.
	return SeverityLow
}

// classifyImpact adds impact domains to a change based on its type and content.
func classifyImpact(ch *Change) {
	// A simple rule-based classification.
	if strings.Contains(ch.NodeType, "table") || strings.Contains(ch.NodeType, "column") {
		ch.Impacts = append(ch.Impacts, ImpactSchema)
	}
	if ch.ChangeType == "modified" && ch.PropDiff != "" {
		// Property changes can affect data representation or defaults.
		ch.Impacts = append(ch.Impacts, ImpactData)
	}
	if strings.Contains(ch.NodeType, "join") || (ch.ChangeType == "modified" && ch.EdgeDiff != "") {
		// Joins and edge changes affect behavior.
		ch.Impacts = append(ch.Impacts, ImpactBehavior)
	}

	// Ensure at least one impact is present, default to schema for structural changes.
	if len(ch.Impacts) == 0 && (ch.ChangeType == "added" || ch.ChangeType == "removed") {
		ch.Impacts = append(ch.Impacts, ImpactSchema)
	}
}

// DiffSnapshots compares two snapshots and returns a list of classified changes.
func DiffSnapshots(oldSnap, newSnap []Node) []Change {
	oldIndex := make(map[string]Node)
	for _, n := range oldSnap {
		key := n.TypeName + "::" + n.QualifiedPath
		oldIndex[key] = n
	}

	newIndex := make(map[string]Node)
	for _, n := range newSnap {
		key := n.TypeName + "::" + n.QualifiedPath
		newIndex[key] = n
	}

	var changes []Change

	// Check for removals and modifications.
	for key, oldNode := range oldIndex {
		if newNode, ok := newIndex[key]; !ok {
			change := Change{
				NodeType:      oldNode.TypeName,
				QualifiedPath: oldNode.QualifiedPath,
				ChangeType:    "removed",
				Severity:      SeverityBreaking,
				Details:       "Node removed from model",
			}
			classifyImpact(&change)
			changes = append(changes, change)
		} else if oldNode.CanonicalHash != newNode.CanonicalHash {
			change := Change{
				NodeType:      oldNode.TypeName,
				QualifiedPath: oldNode.QualifiedPath,
				ChangeType:    "modified",
				Severity:      classifyModification(oldNode, newNode),
				Details:       "Node properties or edges were modified.",
				PropDiff:      jsonDiff(oldNode.Properties, newNode.Properties),
				EdgeDiff:      jsonDiff(oldNode.Edges, newNode.Edges),
			}

			// Populate metadata for context-aware explanations
			var oldProps, newProps map[string]interface{}
			_ = json.Unmarshal(oldNode.Properties, &oldProps)
			_ = json.Unmarshal(newNode.Properties, &newProps)

			if oldVal, ok := oldProps["data_type"].(string); ok {
				change.Metadata.OldType = oldVal
			}
			if newVal, ok := newProps["data_type"].(string); ok {
				change.Metadata.NewType = newVal
			}
			if oldVal, ok := oldProps["constraint_definition"].(string); ok {
				change.Metadata.OldConstraint = oldVal
			}
			if newVal, ok := newProps["constraint_definition"].(string); ok {
				change.Metadata.NewConstraint = newVal
			}
			if newVal, ok := newProps["trigger_body"].(string); ok {
				change.Metadata.TriggerBody = newVal
			}

			classifyImpact(&change)
			changes = append(changes, change)
		}
	}

	// Check for additions.
	for key, newNode := range newIndex {
		if _, ok := oldIndex[key]; !ok {
			change := Change{
				NodeType:      newNode.TypeName,
				QualifiedPath: newNode.QualifiedPath,
				ChangeType:    "added",
				Severity:      SeverityLow,
				Details:       "New node added",
			}
			classifyImpact(&change)
			changes = append(changes, change)
		}
	}

	return changes
}

// containsImpact checks if a change has a specific impact.
func containsImpact(ch *Change, imp ChangeImpact) bool {
	for _, i := range ch.Impacts {
		if i == imp {
			return true
		}
	}
	return false
}

// snippet returns a shortened version of a string with an ellipsis.
func snippet(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// GenerateExplanation creates a human-readable explanation for a change and assigns it.
func GenerateExplanation(ch *Change) {
	switch {
	case ch.Severity == SeverityBreaking && containsImpact(ch, ImpactSchema) && ch.Metadata.OldType != "" && ch.Metadata.NewType != "" && ch.Metadata.OldType != ch.Metadata.NewType:
		ch.Explanation = fmt.Sprintf("Column %s type changed from %s to %s. This may require data migration or application code updates.", ch.QualifiedPath, ch.Metadata.OldType, ch.Metadata.NewType)
	case containsImpact(ch, ImpactBehavior) && ch.Metadata.TriggerBody != "":
		ch.Explanation = fmt.Sprintf("Trigger %s logic changed. Review new body for side effects:\n%s", ch.QualifiedPath, snippet(ch.Metadata.TriggerBody, 120))
	case containsImpact(ch, ImpactPerformance) && ch.Metadata.NewConstraint != "":
		ch.Explanation = fmt.Sprintf("Constraint/index change on %s may alter query performance: %s", ch.QualifiedPath, ch.Metadata.NewConstraint)
	case ch.Severity == SeverityBreaking && containsImpact(ch, ImpactSchema):
		ch.Explanation = fmt.Sprintf("Breaking schema change. Dropping or altering '%s' may break existing queries or application logic.", ch.QualifiedPath)
	case ch.Severity == SeverityMedium && containsImpact(ch, ImpactData):
		ch.Explanation = "A change to properties may alter data representation or defaults, affecting new or existing records."
	case ch.Severity == SeverityMedium && containsImpact(ch, ImpactBehavior):
		ch.Explanation = fmt.Sprintf("A change to edges for '%s' can alter join behavior or data relationships.", ch.QualifiedPath)
	case ch.Severity == SeverityLow && ch.ChangeType == "added":
		ch.Explanation = fmt.Sprintf("Adding the new node '%s' is generally safe, but downstream consumers may need to be updated to use it.", ch.QualifiedPath)
	default:
		ch.Explanation = "Review this change for potential downstream effects on dependent systems."
	}
}

// FilterByImpact returns a new ChangeReport containing only changes with allowed impacts.
func FilterByImpact(report ChangeReport, allowed map[ChangeImpact]bool) ChangeReport {
	if len(allowed) == 0 {
		return report
	}

	filteredReport := ChangeReport{
		Summary:     make(map[ChangeSeverity]int),
		Groups:      make(map[ChangeSeverity][]Change),
		GeneratedAt: report.GeneratedAt,
		Tool:        report.Tool,
		Run:         report.Run,
		SchemaHash:  report.SchemaHash,
	}
	// Initialize maps
	for _, sev := range []ChangeSeverity{SeverityLow, SeverityMedium, SeverityBreaking} {
		filteredReport.Groups[sev] = []Change{}
		filteredReport.Summary[sev] = 0
	}

	for sev, group := range report.Groups {
		for _, ch := range group {
			for _, impact := range ch.Impacts {
				if allowed[impact] {
					filteredReport.Groups[sev] = append(filteredReport.Groups[sev], ch)
					filteredReport.Summary[sev]++
					break // Found a matching impact, add change and move to next change
				}
			}
		}
	}
	return filteredReport
}

// FilterBySeverities returns a new slice of changes containing only those with the specified severities.
func FilterBySeverities(changes []Change, allowed map[ChangeSeverity]bool) []Change {
	// If no filter is provided, return the original slice.
	if len(allowed) == 0 {
		return changes
	}

	var filtered []Change
	for _, ch := range changes {
		if allowed[ch.Severity] {
			filtered = append(filtered, ch)
		}
	}
	return filtered
}

// GroupChanges takes a flat list of changes and organizes them into a structured report.
func GroupChanges(changes []Change) ChangeReport {
	report := ChangeReport{
		Summary:     make(map[ChangeSeverity]int),
		Groups:      make(map[ChangeSeverity][]Change),
		AllChanges:  changes,
		GeneratedAt: time.Now().UTC(),
	}

	// Initialize maps to ensure all severity levels are present in the output, even if empty.
	for _, sev := range []ChangeSeverity{SeverityLow, SeverityMedium, SeverityBreaking} {
		report.Groups[sev] = []Change{}
		report.Summary[sev] = 0
	}

	for _, ch := range changes {
		report.Summary[ch.Severity]++
		report.Groups[ch.Severity] = append(report.Groups[ch.Severity], ch)
	}

	// Sort each group for deterministic output.
	for sev := range report.Groups {
		sort.Slice(report.Groups[sev], func(i, j int) bool {
			if report.Groups[sev][i].NodeType == report.Groups[sev][j].NodeType {
				return report.Groups[sev][i].QualifiedPath < report.Groups[sev][j].QualifiedPath
			}
			return report.Groups[sev][i].NodeType < report.Groups[sev][j].NodeType
		})
	}

	return report
}

// CompareTenantToCore orchestrates the full comparison process.
func CompareTenantToCore(ctx context.Context, db *sqlx.DB, tenantDS, oldCoreDS, newCoreDS uuid.UUID) (ChangeReport, error) {
	// 1. Extract all three snapshots.
	oldCoreSnap, err := BuildSnapshot(ctx, db, oldCoreDS)
	if err != nil {
		return ChangeReport{}, fmt.Errorf("failed to build old core snapshot: %w", err)
	}
	newCoreSnap, err := BuildSnapshot(ctx, db, newCoreDS)
	if err != nil {
		return ChangeReport{}, fmt.Errorf("failed to build new core snapshot: %w", err)
	}
	tenantSnap, err := BuildSnapshot(ctx, db, tenantDS)
	if err != nil {
		return ChangeReport{}, fmt.Errorf("failed to build tenant snapshot: %w", err)
	}

	// 2. Create effective models by overlaying tenant customizations.
	oldEffective := MergeOverlay(oldCoreSnap, tenantSnap)
	newEffective := MergeOverlay(newCoreSnap, tenantSnap)

	// 3. Diff the two effective models, group them, and return the report.
	changes := DiffSnapshots(oldEffective, newEffective)
	return GroupChanges(changes), nil
}

// Code generated from JSON Schema using quicktype. DO NOT EDIT.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    upgradeArtifacts, err := UnmarshalUpgradeArtifacts(bytes)
//    bytes, err = upgradeArtifacts.Marshal()

package types

import "encoding/json"

func UnmarshalUpgradeArtifacts(data []byte) (UpgradeArtifacts, error) {
	var r UpgradeArtifacts
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *UpgradeArtifacts) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type UpgradeArtifacts struct {
	Schema        string                     `json:"$schema"`
	ID            string                     `json:"$id"`
	Title         string                     `json:"title"`
	SchemaVersion string                     `json:"schema_version"`
	Type          Type                       `json:"type"`
	Properties    UpgradeArtifactsProperties `json:"properties"`
	Required      []string                   `json:"required"`
	Definitions   Definitions                `json:"definitions"`
}

type Definitions struct {
	DiffReport         DiffReport         `json:"DiffReport"`
	DiffSummary        DiffSummary        `json:"DiffSummary"`
	CubeDiff           Diff               `json:"CubeDiff"`
	ViewDiff           Diff               `json:"ViewDiff"`
	GovernanceDiff     GovernanceDiff     `json:"GovernanceDiff"`
	PreAggregationDiff PreAggregationDiff `json:"PreAggregationDiff"`
	Change             Change             `json:"Change"`
	AliasMap           AliasMap           `json:"AliasMap"`
	AliasEntry         AliasEntry         `json:"AliasEntry"`
	AliasMeta          AliasMeta          `json:"AliasMeta"`
}

type AliasEntry struct {
	Type       Type                 `json:"type"`
	Properties AliasEntryProperties `json:"properties"`
	Required   []string             `json:"required"`
}

type AliasEntryProperties struct {
	Scope      MemberType  `json:"scope"`
	Name       Name        `json:"name"`
	MemberType MemberType  `json:"member_type"`
	OldName    Name        `json:"old_name"`
	NewName    Name        `json:"new_name"`
	Status     MemberType  `json:"status"`
	Meta       ReportClass `json:"meta"`
}

type MemberType struct {
	Type Type     `json:"type"`
	Enum []string `json:"enum"`
}

type ReportClass struct {
	Ref string `json:"$ref"`
}

type Name struct {
	Type Type `json:"type"`
}

type AliasMap struct {
	Type       Type               `json:"type"`
	Properties AliasMapProperties `json:"properties"`
	Required   []string           `json:"required"`
}

type AliasMapProperties struct {
	CoreVersion     Name         `json:"core_version"`
	PreviousVersion Name         `json:"previous_version"`
	GeneratedAt     GeneratedAt  `json:"generated_at"`
	Aliases         ChangesClass `json:"aliases"`
}

type ChangesClass struct {
	Type  string      `json:"type"`
	Items ReportClass `json:"items"`
}

type GeneratedAt struct {
	Type   Type   `json:"type"`
	Format string `json:"format"`
}

type AliasMeta struct {
	Type       Type                `json:"type"`
	Properties AliasMetaProperties `json:"properties"`
	Required   []string            `json:"required"`
}

type AliasMetaProperties struct {
	Reason               Name `json:"reason"`
	AutoRewrite          Name `json:"auto_rewrite"`
	SuggestedReplacement Name `json:"suggested_replacement"`
	BreakingChange       Name `json:"breaking_change"`
}

type Change struct {
	Type       Type             `json:"type"`
	Properties ChangeProperties `json:"properties"`
	Required   []string         `json:"required"`
}

type ChangeProperties struct {
	Type     Name `json:"type"`
	Name     Name `json:"name"`
	JoinPath Name `json:"join_path"`
	Member   Name `json:"member"`
	Old      New  `json:"old"`
	New      New  `json:"new"`
	Details  Name `json:"details"`
}

type New struct {
}

type Diff struct {
	Type       Type               `json:"type"`
	Properties CubeDiffProperties `json:"properties"`
	Required   []string           `json:"required"`
}

type CubeDiffProperties struct {
	Name    Name         `json:"name"`
	Status  MemberType   `json:"status"`
	Changes ChangesClass `json:"changes"`
}

type DiffReport struct {
	Type       Type                 `json:"type"`
	Properties DiffReportProperties `json:"properties"`
	Required   []string             `json:"required"`
}

type DiffReportProperties struct {
	CoreVersion     Name         `json:"core_version"`
	PreviousVersion Name         `json:"previous_version"`
	GeneratedAt     GeneratedAt  `json:"generated_at"`
	SchemaHash      Name         `json:"schema_hash"`
	Summary         ReportClass  `json:"summary"`
	Cubes           ChangesClass `json:"cubes"`
	Views           ChangesClass `json:"views"`
	Governance      ChangesClass `json:"governance"`
	PreAggregations ChangesClass `json:"pre_aggregations"`
	Warnings        Warnings     `json:"warnings"`
}

type Warnings struct {
	Type  string `json:"type"`
	Items Name   `json:"items"`
}

type DiffSummary struct {
	Type       Type                  `json:"type"`
	Properties DiffSummaryProperties `json:"properties"`
	Required   []string              `json:"required"`
}

type DiffSummaryProperties struct {
	CubesAdded      Name `json:"cubes_added"`
	CubesRemoved    Name `json:"cubes_removed"`
	CubesChanged    Name `json:"cubes_changed"`
	ViewsAdded      Name `json:"views_added"`
	ViewsRemoved    Name `json:"views_removed"`
	ViewsChanged    Name `json:"views_changed"`
	BreakingChanges Name `json:"breaking_changes"`
	Warnings        Name `json:"warnings"`
}

type GovernanceDiff struct {
	Type       Type                     `json:"type"`
	Properties GovernanceDiffProperties `json:"properties"`
	Required   []string                 `json:"required"`
}

type GovernanceDiffProperties struct {
	Scope   MemberType   `json:"scope"`
	Name    Name         `json:"name"`
	Changes ChangesClass `json:"changes"`
}

type PreAggregationDiff struct {
	Type       Type                         `json:"type"`
	Properties PreAggregationDiffProperties `json:"properties"`
	Required   []string                     `json:"required"`
}

type PreAggregationDiffProperties struct {
	Cube   Name       `json:"cube"`
	Name   Name       `json:"name"`
	Status MemberType `json:"status"`
	Reason Name       `json:"reason"`
}

type UpgradeArtifactsProperties struct {
	SchemaVersion SchemaVersion `json:"schema_version"`
	Changelog     Changelog     `json:"changelog"`
	Report        ReportClass   `json:"report"`
	Aliases       ReportClass   `json:"aliases"`
}

type Changelog struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Items       Items  `json:"items"`
}

type Items struct {
	Type       Type            `json:"type"`
	Properties ItemsProperties `json:"properties"`
	Required   []string        `json:"required"`
}

type ItemsProperties struct {
	Version     Name        `json:"version"`
	Date        GeneratedAt `json:"date"`
	Description Name        `json:"description"`
}

type SchemaVersion struct {
	Type        Type   `json:"type"`
	Description string `json:"description"`
}

type Type string

const (
	Boolean Type = "boolean"
	Integer Type = "integer"
	Object  Type = "object"
	String  Type = "string"
)

// Upgrade Overview Response Types
type UpgradeOverviewResponse struct {
	SchemaVersion string           `json:"schema_version"`
	Changelog     []ChangelogEntry `json:"changelog,omitempty"`
	Report        DiffReport       `json:"report"`
	Aliases       AliasMap         `json:"aliases"`
	Status        UpgradeStatus    `json:"status"`
	UIHints       UIHints          `json:"ui_hints,omitempty"`
}

type ChangelogEntry struct {
	Version     string `json:"version"`
	Date        string `json:"date"`
	Description string `json:"description"`
}

type UpgradeStatus struct {
	CoreVersion string   `json:"core_version"`
	Status      string   `json:"status"` // pending | ready | canary | active | rolled_back
	Warnings    []string `json:"warnings"`
	Blockers    []string `json:"blockers"`
}

type UIHints struct {
	NeedsDiffReview   bool `json:"needs_diff_review"`
	NeedsExtensionFix bool `json:"needs_extension_fix"`
	NeedsQueryRun     bool `json:"needs_query_run"`
}

type MultiUpgradeOverviewResponse struct {
	Versions []UpgradeOverviewResponse `json:"versions"`
}

// UpgradeArtifactsData represents the actual upgrade artifacts data (not the schema)
type UpgradeArtifactsData struct {
	SchemaVersion string           `json:"schema_version"`
	Changelog     []ChangelogEntry `json:"changelog,omitempty"`
	Report        DiffReport       `json:"report"`
	Aliases       AliasMap         `json:"aliases"`
}

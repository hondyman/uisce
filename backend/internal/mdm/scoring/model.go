package scoring

import (
	"time"

	"github.com/google/uuid"
)

// MatchType defines how two values are compared within tolerance.
type MatchType string

const (
	MatchExact      MatchType = "EXACT"
	MatchFuzzyJaro  MatchType = "FUZZY_JARO"
	MatchNumericBP  MatchType = "NUMERIC_BP"
	MatchNumericPct MatchType = "NUMERIC_PCT"
	MatchDateLag    MatchType = "DATE_LAG"
)

// AttributeTolerance defines matching criteria and weighting for a single attribute.
type AttributeTolerance struct {
	AttributeCode string    `json:"attribute_code" db:"attribute_code"`
	Tier          int       `json:"tier" db:"tier"` // 1 = Critical/Blocking, 2 = Reporting, 3 = Enrichment
	MatchType     MatchType `json:"match_type" db:"match_type"`
	ToleranceVal  float64   `json:"tolerance_val" db:"tolerance_val"`
	TierWeight    float64   `json:"tier_weight" db:"tier_weight"`
	Description   string    `json:"description" db:"description"`
}

// VendorCandidate represents a candidate source value for an entity attribute.
type VendorCandidate struct {
	EntityID        int64     `json:"entity_id"`
	AttributeCode   string    `json:"attribute_code"`
	VendorID        string    `json:"vendor_id"`
	RawValue        string    `json:"raw_value"`
	NormalizedValue string    `json:"normalized_value"`
	FormatOK        bool      `json:"format_ok"`
	RangeOK         bool      `json:"range_ok"`
	RefIntegrityOK  bool      `json:"ref_integrity_ok"`
	AsOfDate        time.Time `json:"as_of_date"`
}

// IsValid checks whether the candidate payload passed syntax, range, and reference checks.
func (vc VendorCandidate) IsValid() bool {
	return vc.NormalizedValue != "" && vc.FormatOK && vc.RangeOK && vc.RefIntegrityOK
}

// GoldenRecord represents the winning survivor value for an entity attribute.
type GoldenRecord struct {
	TenantID        uuid.UUID `json:"tenant_id"`
	EntityID        int64     `json:"entity_id"`
	AttributeCode   string    `json:"attribute_code"`
	GoldenValue     string    `json:"golden_value"`
	WinningVendorID string    `json:"winning_vendor_id"`
	RuleApplied     string    `json:"rule_applied"`
	AsOfDate        time.Time `json:"as_of_date"`
}

// ValueOverrideRecord captures steward manual override actions.
type ValueOverrideRecord struct {
	OverrideID          int64     `json:"override_id"`
	TenantID            uuid.UUID `json:"tenant_id"`
	EntityID            int64     `json:"entity_id"`
	AttributeCode       string    `json:"attribute_code"`
	PriorGoldenValue    string    `json:"prior_golden_value"`
	PriorVendorID       string    `json:"prior_vendor_id"`
	OverriddenValue     string    `json:"overridden_value"`
	EndorsementVendorID string    `json:"endorsement_vendor_id"`
	StewardID           string    `json:"steward_id"`
	DefectReason        string    `json:"defect_reason"`
	OverrideTS          time.Time `json:"override_ts"`
}

// EvaluationState captures the relationship between a candidate value and the golden value.
type EvaluationState string

const (
	StateAbsent       EvaluationState = "ABSENT"
	StateInvalid      EvaluationState = "INVALID"
	StateValidMatch   EvaluationState = "VALID_MATCH"
	StateValidDiffers EvaluationState = "VALID_DIFFERS"
)

// SubstitutionScore represents sufficiency and contribution metrics for a vendor on a specific attribute.
type SubstitutionScore struct {
	VendorID                string  `json:"vendor_id"`
	AttributeCode           string  `json:"attribute_code"`
	Tier                    int     `json:"tier"`
	InScope                 int     `json:"in_scope"`
	AvailableCount          int     `json:"available_count"`
	ValidMatchCount         int     `json:"valid_match_count"`
	CoveragePct             float64 `json:"coverage_pct"`
	SufficiencyRatePct      float64 `json:"sufficiency_rate_pct"`
	ConditionalSufficiency  float64 `json:"conditional_sufficiency_pct"`
	SoloRatePct             float64 `json:"solo_rate_pct"`
	SoloRecordsCount        int     `json:"solo_records_count"`
	ContributionSharePct    float64 `json:"contribution_share_pct"`
	ExceptionsCount         int     `json:"exceptions_count"`
	OverrideEndorsementRate float64 `json:"override_endorsement_rate_pct"`
}

// TierDisplacementSummary aggregates displacement impact by attribute tier.
type TierDisplacementSummary struct {
	Tier              int     `json:"tier"`
	AttributesCount   int     `json:"attributes_count"`
	AverageInScope    int     `json:"average_in_scope"`
	UnchangedPct      float64 `json:"unchanged_pct"` // another source matched
	ChangedPct        float64 `json:"changed_pct"`   // alternative value used
	NowNullPct        float64 `json:"now_null_pct"`  // sole source lost
}

// ResidualGap isolates attributes with non-zero solo dependencies.
type ResidualGap struct {
	AttributeCode string  `json:"attribute_code"`
	Tier          int     `json:"tier"`
	SoloRatePct   float64 `json:"solo_rate_pct"`
	RecordsLost   int     `json:"records_lost"`
}

// VendorDisplacementResult captures the simulation output if a vendor is removed.
type VendorDisplacementResult struct {
	DroppedVendorID       string                    `json:"dropped_vendor_id"`
	DroppedVendorName     string                    `json:"dropped_vendor_name"`
	AnnualCost            float64                   `json:"annual_cost"`
	DisplacementReadiness float64                   `json:"displacement_readiness_pct"`
	TierSummaries         []TierDisplacementSummary `json:"tier_summaries"`
	ResidualGaps          []ResidualGap             `json:"residual_gaps"`
	TotalNullValues       int                       `json:"total_null_values"`
	RemediationCostEst    float64                   `json:"remediation_cost_est"`
	NetFirstYearBenefit   float64                   `json:"net_first_year_benefit"`
	CostPerUniqueValue    float64                   `json:"cost_per_unique_value"`
}

// ValueForMoneyPoint represents a vendor's coordinates on the cost/quality frontier.
type ValueForMoneyPoint struct {
	VendorID             string  `json:"vendor_id"`
	VendorName           string  `json:"vendor_name"`
	AnnualCost           float64 `json:"annual_cost"`
	QualityIndex         float64 `json:"quality_index"`
	MarginalQualityUplift float64 `json:"marginal_quality_uplift"`
	CostPerQualityPoint  float64 `json:"cost_per_quality_point"`
	IsOnFrontier         bool    `json:"is_on_frontier"`
}

// VendorScorecardReport aggregates the full executive pack for a set of vendors.
type VendorScorecardReport struct {
	AsOfDate               string                        `json:"as_of_date"`
	UniverseSize           int                           `json:"universe_size"`
	TiersTracked           int                           `json:"tiers_tracked"`
	AnnualSpendTotal       float64                       `json:"annual_spend_total"`
	BestAltSufficiencyT1   float64                       `json:"best_alt_sufficiency_t1"`
	PremiumSoloShareT1     float64                       `json:"premium_solo_share_t1"`
	DisplacementReadiness  float64                       `json:"displacement_readiness"`
	SubstitutionMatrix     []SubstitutionScore           `json:"substitution_matrix"`
	FrontierPoints         []ValueForMoneyPoint          `json:"frontier_points"`
	DisplacementScenarios  []VendorDisplacementResult    `json:"displacement_scenarios"`
}

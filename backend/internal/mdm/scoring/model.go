package scoring

import (
	"encoding/json"
	"fmt"
	"math"
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
	EntityDomain            string  `json:"entity_domain"` // "security", "ratings", "benchmarks", "pricing", "party"
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
	FrictionSavings       float64                   `json:"friction_savings"`
	ForfeitedSLACredits   float64                   `json:"forfeited_sla_credits"`
	NetTCOBenefit         float64                   `json:"net_tco_benefit"`
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

// VendorEntityBreakdown captures quality, cost, and standing for a vendor within a specific entity domain.
type VendorEntityBreakdown struct {
	VendorID            string  `json:"vendor_id"`
	VendorName          string  `json:"vendor_name"`
	EntityDomain        string  `json:"entity_domain"` // "security", "ratings", "benchmarks", "pricing", "party"
	AnnualCost          float64 `json:"annual_cost"`
	QualityIndex        float64 `json:"quality_index"`
	CostPerQualityPoint float64 `json:"cost_per_quality_point"`
	SufficiencyRatePct  float64 `json:"sufficiency_rate_pct"`
	CoveragePct         float64 `json:"coverage_pct"`
	SoloRatePct         float64 `json:"solo_rate_pct"`
	IsOnFrontier        bool    `json:"is_on_frontier"`
	AttributeCount      int     `json:"attribute_count"`
}

// UpdateSpendRequest models a payload for updating vendor annual spend.
type UpdateSpendRequest struct {
	VendorID     string  `json:"vendor_id"`
	AnnualCost   float64 `json:"annual_cost"`
	EntityDomain string  `json:"entity_domain,omitempty"`
}

// VendorScorecardReport aggregates the full executive pack for a set of vendors.
type VendorScorecardReport struct {
	ScoringVersion        string                     `json:"scoring_version"`
	TenantID              string                     `json:"tenant_id,omitempty"`
	AsOfDate              string                     `json:"as_of_date"`
	UniverseSize          int                        `json:"universe_size"`
	TiersTracked          int                        `json:"tiers_tracked"`
	AnnualSpendTotal      float64                    `json:"annual_spend_total"`
	BestAltSufficiencyT1  float64                    `json:"best_alt_sufficiency_t1"`
	PremiumSoloShareT1    float64                    `json:"premium_solo_share_t1"`
	DisplacementReadiness float64                    `json:"displacement_readiness"`
	SubstitutionMatrix    []SubstitutionScore        `json:"substitution_matrix"`
	Matrix                []SubstitutionScore        `json:"matrix,omitempty"`
	FrontierPoints        []ValueForMoneyPoint       `json:"frontier_points"`
	EntityBreakdowns      []VendorEntityBreakdown    `json:"entity_breakdowns"`
	EntityDomains         []string                   `json:"entity_domains"`
	DisplacementScenarios []VendorDisplacementResult `json:"displacement_scenarios"`
	WeightProfileID       int64                      `json:"weight_profile_id,omitempty"`
	ActiveWeightProfile   *WeightProfile             `json:"active_weight_profile,omitempty"`
	DimensionProfiles     []VendorDimensionProfile   `json:"dimension_profiles,omitempty"`
}

// ScoringSettings defines tenant-level parameters for scoring evaluation.
type ScoringSettings struct {
	TenantID                 uuid.UUID `json:"tenant_id" db:"tenant_id"`
	HourlyLaborRate          float64   `json:"hourly_labor_rate" db:"hourly_labor_rate"`
	StabilityDecayK          float64   `json:"stability_decay_k" db:"stability_decay_k"`
	FrictionBudget           float64   `json:"friction_budget" db:"friction_budget"`
	ColdStartDays            int       `json:"cold_start_days" db:"cold_start_days"`
	WatermarkHotDays         int       `json:"watermark_hot_days" db:"watermark_hot_days"`
	WatermarkWarmDays        int       `json:"watermark_warm_days" db:"watermark_warm_days"`
	QualityZoneHighThreshold float64   `json:"quality_zone_high_threshold" db:"quality_zone_high_threshold"`
	QualityZoneMidThreshold  float64   `json:"quality_zone_mid_threshold" db:"quality_zone_mid_threshold"`
	UpdatedAt                time.Time `json:"updated_at" db:"updated_at"`
}

// WeightProfile defines the 6-pillar weighting configuration for composite scoring.
type WeightProfile struct {
	ProfileID    int64     `json:"profile_id" db:"profile_id"`
	TenantID     uuid.UUID `json:"tenant_id" db:"tenant_id"`
	ProfileName  string    `json:"profile_name" db:"profile_name"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	WeightSuff   float64   `json:"weight_suff" db:"weight_suff"`
	WeightCov    float64   `json:"weight_cov" db:"weight_cov"`
	WeightSLA    float64   `json:"weight_sla" db:"weight_sla"`
	WeightStab   float64   `json:"weight_stab" db:"weight_stab"`
	WeightOER    float64   `json:"weight_oer" db:"weight_oer"`
	WeightLic    float64   `json:"weight_lic" db:"weight_lic"`
	CreatedBy    string    `json:"created_by" db:"created_by"`
	ChangeReason string    `json:"change_reason" db:"change_reason"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// VendorFeedLog tracks feed arrival timeliness and SLA cutoff compliance.
type VendorFeedLog struct {
	LogID           int64     `json:"log_id" db:"log_id"`
	TenantID        uuid.UUID `json:"tenant_id" db:"tenant_id"`
	VendorID        string    `json:"vendor_id" db:"vendor_id"`
	EntityDomain    string    `json:"entity_domain" db:"entity_domain"`
	FeedName        string    `json:"feed_name" db:"feed_name"`
	AsOfDate        time.Time `json:"as_of_date" db:"as_of_date"`
	ArrivalTS       time.Time `json:"arrival_ts" db:"arrival_ts"`
	SLACutoffTS     time.Time `json:"sla_cutoff_ts" db:"sla_cutoff_ts"`
	DeliveryLagMins int       `json:"delivery_lag_mins" db:"delivery_lag_mins"`
	SLABreached     bool      `json:"sla_breached" db:"sla_breached"`
	RecordsReceived int       `json:"records_received" db:"records_received"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// VendorRevisionLog captures vendor-initiated restatements and changes.
type VendorRevisionLog struct {
	RevisionID      int64     `json:"revision_id" db:"revision_id"`
	TenantID        uuid.UUID `json:"tenant_id" db:"tenant_id"`
	VendorID        string    `json:"vendor_id" db:"vendor_id"`
	EntityID        int64     `json:"entity_id" db:"entity_id"`
	AttributeCode   string    `json:"attribute_code" db:"attribute_code"`
	EntityDomain    string    `json:"entity_domain" db:"entity_domain"`
	AsOfDate        time.Time `json:"as_of_date" db:"as_of_date"`
	OriginalValue   string    `json:"original_value" db:"original_value"`
	RevisedValue    string    `json:"revised_value" db:"revised_value"`
	PctChange       float64   `json:"pct_change" db:"pct_change"`
	HoursToRevision float64   `json:"hours_to_revision" db:"hours_to_revision"`
	RevisionTS      time.Time `json:"revision_ts" db:"revision_ts"`
}

// VendorOperationalFriction records steward investigation hours, ticket volume, and SLA credit offsets.
type VendorOperationalFriction struct {
	MetricID               int64     `json:"metric_id" db:"metric_id"`
	TenantID               uuid.UUID `json:"tenant_id" db:"tenant_id"`
	VendorID               string    `json:"vendor_id" db:"vendor_id"`
	AsOfDate               time.Time `json:"as_of_date" db:"as_of_date"`
	DefectTicketsCount     int       `json:"defect_tickets_count" db:"defect_tickets_count"`
	InvestigationHours     float64   `json:"investigation_hours" db:"investigation_hours"`
	AvgMTTRHours           float64   `json:"avg_mttr_hours" db:"avg_mttr_hours"`
	ContractSLACredits     float64   `json:"contract_sla_credits" db:"contract_sla_credits"`
	CalculatedFrictionCost float64   `json:"calculated_friction_cost" db:"calculated_friction_cost"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
}

// VendorContractRights captures commercial usage rights, unbundled access, and termination flexibility.
type VendorContractRights struct {
	ContractID             int64     `json:"contract_id" db:"contract_id"`
	TenantID               uuid.UUID `json:"tenant_id" db:"tenant_id"`
	VendorID               string    `json:"vendor_id" db:"vendor_id"`
	DerivedDataRights      int       `json:"derived_data_rights" db:"derived_data_rights"`
	ClientRedistribution   int       `json:"client_redistribution" db:"client_redistribution"`
	ExternalWebRights      int       `json:"external_web_rights" db:"external_web_rights"`
	UnbundledAPIAccess     bool      `json:"unbundled_api_access" db:"unbundled_api_access"`
	CancellationNoticeDays int       `json:"cancellation_notice_days" db:"cancellation_notice_days"`
	ContractStartDate      time.Time `json:"contract_start_date" db:"contract_start_date"`
	ContractEndDate        time.Time `json:"contract_end_date" db:"contract_end_date"`
	CompositeRightsScore   float64   `json:"composite_rights_score" db:"composite_rights_score"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
}

// QualityComponents holds normalized sub-scores in [0.0, 1.0] across all 6 pillars.
type QualityComponents struct {
	Sufficiency float64 `json:"sufficiency"` // SR_v
	Coverage    float64 `json:"coverage"`    // Cov_v
	SLA         float64 `json:"sla"`         // SLA_v
	Stability   float64 `json:"stability"`   // Stab_v = exp(-k * r_v)
	Friction    float64 `json:"friction"`    // max(0, 1 - cost / budget)
	Licensing   float64 `json:"licensing"`   // Lic_v
}

// VendorDimensionProfile provides a 360-degree quality profile for a vendor.
type VendorDimensionProfile struct {
	VendorID            string            `json:"vendor_id"`
	VendorName          string            `json:"vendor_name"`
	AnnualSpend         float64           `json:"annual_spend"`
	Components          QualityComponents `json:"components"`
	CompositeQuality    float64           `json:"composite_quality"` // Q_v in [0.0, 1.0]
	CostPerQualityPoint float64           `json:"cost_per_quality_point"`
	AvgDeliveryLagMins  float64           `json:"avg_delivery_lag_mins"`
	SLABreachCount      int               `json:"sla_breach_count"`
	TotalFeedsReceived  int               `json:"total_feeds_received"`
	TotalRevisions      int               `json:"total_revisions"`
	RevisionRatePct     float64           `json:"revision_rate_pct"`
	DefectTicketsCount  int               `json:"defect_tickets_count"`
	InvestigationHours  float64           `json:"investigation_hours"`
	FrictionCost        float64           `json:"friction_cost"`
	SLACredits          float64           `json:"sla_credits"`
	RightsScore         float64           `json:"rights_score"`
	IsBaselineSeeded    bool              `json:"is_baseline_seeded"`
}

// OptimalBundleRequest specifies constraints for portfolio set-cover optimization.
type OptimalBundleRequest struct {
	TargetT1Coverage float64  `json:"target_t1_coverage"` // e.g. 0.995 (99.5%)
	TargetT2Coverage float64  `json:"target_t2_coverage"` // e.g. 0.950 (95.0%)
	TargetT3Coverage float64  `json:"target_t3_coverage"` // e.g. 0.850 (85.0%)
	MaxBudget        float64  `json:"max_budget,omitempty"`
	MandatoryVendors []string `json:"mandatory_vendors,omitempty"`
	ExcludedVendors  []string `json:"excluded_vendors,omitempty"`
	UniverseSize     int      `json:"universe_size,omitempty"`
}

// ResidualCoverageGap reports coverage deficits when an exact target cannot be fulfilled.
type ResidualCoverageGap struct {
	Tier               int     `json:"tier"`
	TargetCoverage     float64 `json:"target_coverage"`
	AchievedCoverage   float64 `json:"achieved_coverage"`
	MissingEntityCount int     `json:"missing_entity_count"`
	Notes              string  `json:"notes"`
}

// OptimalBundleResult holds the optimal vendor subset and efficiency metrics.
type OptimalBundleResult struct {
	SelectedVendors     []string              `json:"selected_vendors"`
	SelectedVendorNames []string              `json:"selected_vendor_names"`
	TotalAnnualCost     float64               `json:"total_annual_cost"`
	PreviousTotalCost   float64               `json:"previous_total_cost"`
	AnnualSavings       float64               `json:"annual_savings"`
	SavingsPct          float64               `json:"savings_pct"`
	T1CoverageAchieved  float64               `json:"t1_coverage_achieved"`
	T2CoverageAchieved  float64               `json:"t2_coverage_achieved"`
	T3CoverageAchieved  float64               `json:"t3_coverage_achieved"`
	WasRelaxed          bool                  `json:"was_relaxed"`
	ResidualGaps        []ResidualCoverageGap `json:"residual_gaps,omitempty"`
	SolverExecutionMs   float64               `json:"solver_execution_ms"`
	SolverStrategy      string                `json:"solver_strategy"`
	SolverPartial       bool                  `json:"solver_partial,omitempty"`
}

// CanonicalEntityDomains lists all evaluated master data domains.
var CanonicalEntityDomains = []string{"security", "ratings", "benchmarks", "pricing", "party"}

// AttributeCodeToDomain maps an attribute code to its master data entity domain.
func AttributeCodeToDomain(attrCode string) string {
	switch attrCode {
	case "ISIN", "SHARES_OUTSTANDING", "MARKET_CAP", "SEDOL", "CUSIP":
		return "security"
	case "COMPOSITE_RATING", "SANCTIONS_FLAG", "COUNTRY_OF_RISK", "CREDIT_WATCH":
		return "ratings"
	case "GICS_SECTOR", "NAICS_INDUSTRY", "INDEX_WEIGHT", "BENCHMARK_FAMILY":
		return "benchmarks"
	case "CLOSING_PRICE", "BID_ASK_SPREAD", "YIELD_TO_MATURITY", "NAV":
		return "pricing"
	case "LEI", "LEGAL_NAME", "DOMICILE", "PARENT_SUBSIDIARY", "EMPLOYEE_COUNT", "WEBSITE", "YEAR_FOUNDED":
		return "party"
	default:
		return "security"
	}
}

// MultiVendorDisplacementRequest defines inputs for multi-vendor removal simulation.
type MultiVendorDisplacementRequest struct {
	TenantID             string   `json:"tenant_id,omitempty"`
	DroppedVendorIDs     []string `json:"dropped_vendor_ids"`
	ReplacementVendorIDs []string `json:"replacement_vendor_ids,omitempty"`
	TargetT1Coverage     float64  `json:"target_t1_coverage,omitempty"`
	TargetT2Coverage     float64  `json:"target_t2_coverage,omitempty"`
	TargetT3Coverage     float64  `json:"target_t3_coverage,omitempty"`
	UniverseSize         int      `json:"universe_size,omitempty"`
}

// CombinedTCOBreakdown summarizes financial impact of multi-vendor substitution.
type CombinedTCOBreakdown struct {
	LicenseSavingsTotal    float64 `json:"license_savings_total"`
	FrictionDeltaTotal     float64 `json:"friction_delta_total"`       // positive = friction cost increase
	ForfeitedSLACredits    float64 `json:"forfeited_sla_credits_total"` // penalty credits lost
	RemediationDragTotal   float64 `json:"remediation_drag_total"`     // quality gap cost
	ReplacementCostDelta   float64 `json:"replacement_cost_delta"`     // new spend on replacements
	NetFirstYearTCOBenefit float64 `json:"net_first_year_tco_benefit"` // first-year net benefit
	NetAnnualTCOBenefit    float64 `json:"net_annual_tco_benefit"`     // recurring ongoing benefit
	PaybackMonths          float64 `json:"payback_months"`             // months to recoup one-time drag
}

// SingleVendorTCO captures per-vendor financial and readiness metrics within multi-displacement.
type SingleVendorTCO struct {
	VendorID             string  `json:"vendor_id"`
	VendorName           string  `json:"vendor_name"`
	AnnualSavings        float64 `json:"annual_savings"`
	FrictionChange       float64 `json:"friction_change"`
	SLACreditsLost       float64 `json:"sla_credits_lost"`
	RemediationCost      float64 `json:"remediation_cost"`
	NetTCOBenefit        float64 `json:"net_tco_benefit"`
	DisplacementReadyPct float64 `json:"displacement_readiness_pct"`
}

// TierCoverageDelta represents coverage shifts in each attribute tier before and after displacement.
type TierCoverageDelta struct {
	Tier           int     `json:"tier"`            // 1, 2, 3
	CoverageBefore float64 `json:"coverage_before"` // with incumbent full stack
	CoverageAfter  float64 `json:"coverage_after"`  // with dropped vendors removed and replacements added
	DeltaPct       float64 `json:"delta_pct"`       // signed change percentage
	MeetsThreshold bool    `json:"meets_threshold"`
	Threshold      float64 `json:"threshold"`
}

// ResidualGapAttribute details an orphaned attribute and affected entities.
type ResidualGapAttribute struct {
	AttributeCode           string  `json:"attribute_code"`
	EntityDomain            string  `json:"entity_domain"`
	Tier                    int     `json:"tier"`
	EntitiesAffected        int     `json:"entities_affected"`
	SoleSourceVendorID      string  `json:"sole_source_vendor_id"`
	SuggestedSubstituteID   string  `json:"suggested_substitute_id,omitempty"`
	EstimatedSubLicenseCost float64 `json:"estimated_sub_license_cost"`
	IsTier1Critical         bool    `json:"is_tier1_critical"`
}

// SubLicenseProposal recommends targeted unbundled data carve-outs to preserve 100% coverage.
type SubLicenseProposal struct {
	VendorID            string   `json:"vendor_id"`
	VendorName          string   `json:"vendor_name"`
	AttributesCovered   []string `json:"attributes_covered"`
	EntitiesCovered     int      `json:"entities_covered"`
	EstimatedAnnualCost float64  `json:"estimated_annual_cost"`
	TierPriority        int      `json:"tier_priority"`
}

// DisplacementGapReport aggregates attribute-level gaps and negotiation leverage.
type DisplacementGapReport struct {
	TotalEntitiesAffected   int                    `json:"total_entities_affected"`
	Tier1Gaps               []ResidualGapAttribute `json:"tier1_gaps"`
	Tier2Gaps               []ResidualGapAttribute `json:"tier2_gaps"`
	Tier3Gaps               []ResidualGapAttribute `json:"tier3_gaps"`
	RecommendedSubLicenses  []SubLicenseProposal   `json:"recommended_sub_licenses"`
	EstimatedGapRemediation float64                `json:"estimated_gap_remediation_cost"`
	NetNegotiationLeverage  float64                `json:"net_negotiation_leverage"` // 1 - (SumSubLicenses / FullBundleCost)
}

// MultiVendorDisplacementResult delivers the comprehensive multi-vendor displacement TCO tearsheet.
type MultiVendorDisplacementResult struct {
	DroppedVendorIDs     []string              `json:"dropped_vendor_ids"`
	ReplacementVendorIDs []string              `json:"replacement_vendor_ids"`
	CombinedTCO          CombinedTCOBreakdown  `json:"combined_tco"`
	PerVendorBreakdown   []SingleVendorTCO     `json:"per_vendor_breakdown"`
	TierCoverageDeltas   []TierCoverageDelta   `json:"tier_coverage_deltas"`
	ResidualGaps         []ResidualGap         `json:"residual_gaps"`
	GapReport            DisplacementGapReport `json:"gap_report"`
	SolverPartial        bool                  `json:"solver_partial,omitempty"`
	GeneratedAt          time.Time             `json:"generated_at"`
}

// ShadowStabilityState classifies the trust level of a vendor's rank position.
type ShadowStabilityState string

const (
	ShadowStable   ShadowStabilityState = "STABLE"    // Rank delta == 0 and gap to neighbors >= 0.5 pts
	ShadowTieBound ShadowStabilityState = "TIE_BOUND" // Gap to nearest competitor < 0.5 pts regardless of rank
	ShadowMoved    ShadowStabilityState = "MOVED"     // Real displacement with separated rank (delta != 0 and gap >= 0.5 pts)
)

// ShadowAttribution distinguishes whether a score delta is driven by data drift or model changes.
type ShadowAttribution string

const (
	AttributionComparable   ShadowAttribution = "COMPARABLE"    // Delta is pure input drift (weights and scale identical)
	AttributionScaled       ShadowAttribution = "SCALED"        // Score scale changed; normalized to 0-100 basis
	AttributionModelChanged ShadowAttribution = "MODEL_CHANGED" // Weights changed; see ModelEffect
	AttributionUnavailable  ShadowAttribution = "UNAVAILABLE"   // Old basis not available for decomposition
)

// ShadowBasis documents the provenance of the old vs new scoring basis.
type ShadowBasis struct {
	WeightsChanged bool    `json:"weights_changed"`
	ScaleFactor    float64 `json:"scale_factor"`
	Decomposition  string  `json:"decomposition"` // "EXACT_MIDPOINT" or "UNAVAILABLE"
	OldRanking     string  `json:"old_ranking"`   // "VALID" or "DEGENERATE"
	OldSpread      float64 `json:"old_spread"`    // max(old) - min(old)
	InputBasis     string  `json:"input_basis"`   // "SAME_SNAPSHOT_RULER_ONLY" or "HISTORICAL_SNAPSHOT"
}

// DroppedVendorInfo details why an analytical candidate or test fixture was excluded from shadow comparison.
type DroppedVendorInfo struct {
	VendorID string `json:"vendor_id"`
	Reason   string `json:"reason"`
}

// ShadowVendorComparison represents the side-by-side legacy vs multi-dimensional scoring of a single vendor.
type ShadowVendorComparison struct {
	VendorID           string               `json:"vendor_id"`
	VendorName         string               `json:"vendor_name"`
	OldScore           float64              `json:"old_score"` // raw, as originally published
	NewScore           float64              `json:"new_score"` // raw current composite
	OldScoreNormalized float64              `json:"old_score_normalized"` // both on the current 0-100 basis
	NewScoreNormalized float64              `json:"new_score_normalized"`
	OldRank            int                  `json:"old_rank"`
	NewRank            int                  `json:"new_rank"`
	RankDelta          *int                 `json:"rank_delta"` // nil when old ranking is DEGENERATE (old spread < 0.5 pts)
	ScoreDelta         float64              `json:"score_delta"` // NormalizedNew - NormalizedOld == ModelEffect + InputEffect
	ModelEffect        float64              `json:"model_effect"` // The ruler moved: weight/scale shift
	InputEffect        float64              `json:"input_effect"` // The data moved: dimension scores
	ScaleFactor        float64              `json:"scale_factor"` // Old -> New basis multiplier (1.0 if unchanged)
	Attribution        ShadowAttribution    `json:"attribution"`
	NeighborGap        *float64             `json:"neighbor_gap"` // nil when there is no adjacent competitor
	Stability          ShadowStabilityState `json:"stability"`    // STABLE, TIE_BOUND, MOVED
	SafetyAlert        string               `json:"safety_alert,omitempty"`
}

// ShadowValidationReport delivers the full dual-run comparator results and stability metrics.
type ShadowValidationReport struct {
	AsOfDate          time.Time                `json:"as_of_date"`
	TenantID          string                   `json:"tenant_id"`
	WeightProfileName string                   `json:"weight_profile_name"`
	Basis             ShadowBasis              `json:"basis"`
	IsStable          bool                     `json:"is_stable"`
	MaxRankDelta      int                      `json:"max_rank_delta"`
	PairwiseAgreement float64                  `json:"pairwise_agreement_pct"`
	RankInversionRate float64                  `json:"rank_inversion_rate_pct"`
	Comparisons       []ShadowVendorComparison `json:"comparisons"`
	DroppedVendors    []DroppedVendorInfo      `json:"dropped_vendors,omitempty"`
	RecentRunHistory  []ShadowRunLogEntry      `json:"recent_run_history,omitempty"`
}

// ShadowRunLogEntry maps directly to mdm_eval.shadow_run_log.
type ShadowRunLogEntry struct {
	RunID                int64     `json:"run_id" db:"run_id"`
	TenantID             string    `json:"tenant_id" db:"tenant_id"`
	ExecutedAt           time.Time `json:"executed_at" db:"executed_at"`
	CandidateVendorIDs   []string  `json:"candidate_vendor_ids" db:"candidate_vendor_ids"`
	DroppedVendorIDs     []string  `json:"dropped_vendor_ids" db:"dropped_vendor_ids"`
	ReplacementVendorIDs []string  `json:"replacement_vendor_ids" db:"replacement_vendor_ids"`
	UniverseSize         int       `json:"universe_size" db:"universe_size"`
	T1ConcordanceDelta   float64   `json:"t1_concordance_delta" db:"t1_concordance_delta"`
	T2ConcordanceDelta   float64   `json:"t2_concordance_delta" db:"t2_concordance_delta"`
	T3ConcordanceDelta   float64   `json:"t3_concordance_delta" db:"t3_concordance_delta"`
	GrossAnnualSavings   float64   `json:"gross_annual_savings" db:"gross_annual_savings"`
	NetTCOBenefit        float64   `json:"net_tco_benefit" db:"net_tco_benefit"`
	PaybackMonths        float64   `json:"payback_months" db:"payback_months"`
	SolverLatencyMs      float64   `json:"solver_latency_ms" db:"solver_latency_ms"`
	SolverStrategy       string    `json:"solver_strategy" db:"solver_strategy"`
	SolverPartial        bool      `json:"solver_partial" db:"solver_partial"`
	Status               string    `json:"status" db:"status"`
	MetadataJSON         string    `json:"metadata,omitempty" db:"metadata"`
}

// BoundaryConflict records a detected value divergence between two storage tiers at an overlapping boundary date.
type BoundaryConflict struct {
	VendorID string    `json:"vendor_id"`
	Date     time.Time `json:"date"`
	TierA    string    `json:"tier_a"` // Winner tier
	TierB    string    `json:"tier_b"` // Overwritten tier
	ValueA   float64   `json:"value_a"`
	ValueB   float64   `json:"value_b"`
}

// TrendPoint represents a single data point in a historical trend series.
type TrendPoint struct {
	AsOfDate       time.Time `json:"as_of_date"`
	VendorID       string    `json:"vendor_id"`
	VendorName     string    `json:"vendor_name"`
	Dimension      string    `json:"dimension"`
	ScoreValue     float64   `json:"score_value"` // Standardized 0-100 score
	RawMetricValue *float64  `json:"raw_metric_value,omitempty"`
	Tier1Coverage  float64   `json:"tier1_coverage,omitempty"`
	Tier2Coverage  float64   `json:"tier2_coverage,omitempty"`
	Tier3Coverage  float64   `json:"tier3_coverage,omitempty"`
	RankPosition   int       `json:"rank_position,omitempty"`
	StorageTier    string    `json:"storage_tier"` // HOT, WARM, COLD
}

// VendorTrendSeries holds the continuous time series for a single vendor along a specific dimension.
type VendorTrendSeries struct {
	VendorID         string            `json:"vendor_id"`
	VendorName       string            `json:"vendor_name"`
	Dimension        string            `json:"dimension"`
	Points           []TrendPoint      `json:"points"`
	MinScore         float64           `json:"min_score"`
	MaxScore         float64           `json:"max_score"`
	AvgScore         float64           `json:"avg_score"`          // Unweighted arithmetic average
	AvgScoreWeighted float64           `json:"avg_score_weighted"` // Interval-weighted average (density-safe)
	SlopePer30d      float64           `json:"slope_per_30d"`      // Weighted least-squares slope in pts/30d
	TrendDirection   string            `json:"trend_direction"`    // IMPROVING, DECLINING, STABLE, INSUFFICIENT_DATA
	TrendByTier      map[string]string `json:"trend_by_tier"`      // Per-tier trend classification
}

// WatermarkBoundaries details the time ranges routed to each storage tier.
type WatermarkBoundaries struct {
	HotWindowDays  int        `json:"hot_window_days"`
	WarmWindowDays int        `json:"warm_window_days"`
	HotStart       *time.Time `json:"hot_start,omitempty"`
	HotEnd         *time.Time `json:"hot_end,omitempty"`
	WarmStart      *time.Time `json:"warm_start,omitempty"`
	WarmEnd        *time.Time `json:"warm_end,omitempty"`
	ColdStart      *time.Time `json:"cold_start,omitempty"`
	ColdEnd        *time.Time `json:"cold_end,omitempty"`
}

// TrendAnalysisReport is the comprehensive historical trend analysis payload.
type TrendAnalysisReport struct {
	TenantID                 string              `json:"tenant_id"`
	Dimension                string              `json:"dimension"`
	DateFrom                 time.Time           `json:"date_from"`
	DateTo                   time.Time           `json:"date_to"`
	TiersPlanned             []string            `json:"tiers_planned"`
	TiersHit                 []string            `json:"tiers_hit"`
	EmptyTiers               []string            `json:"empty_tiers"`
	TiersQueried             []string            `json:"tiers_queried"` // Retained for backward-compat
	BoundaryConflicts        []BoundaryConflict  `json:"boundary_conflicts"`
	BoundaryConflictCount    int                 `json:"boundary_conflict_count"`
	Series                   []VendorTrendSeries `json:"series"`
	Watermarks               WatermarkBoundaries `json:"watermarks"`
	QualityZoneHighThreshold float64             `json:"quality_zone_high_threshold"`
	QualityZoneMidThreshold  float64             `json:"quality_zone_mid_threshold"`
	GeneratedAt              time.Time           `json:"generated_at"`
}

// TrendQueryRequest defines the parameters for historical trend queries.
type TrendQueryRequest struct {
	TenantID     string    `json:"tenant_id"`
	VendorIDs    []string  `json:"vendor_ids"`
	Dimension    string    `json:"dimension"` // Whitelisted
	DateFrom     time.Time `json:"date_from"`
	DateTo       time.Time `json:"date_to"`
	EntityDomain string    `json:"entity_domain"`
}

// ProfileWeights defines the 6-pillar weights for custom or resolved profile evaluation.
type ProfileWeights struct {
	Suff float64 `json:"suff"`
	Cov  float64 `json:"cov"`
	SLA  float64 `json:"sla"`
	Stab float64 `json:"stab"`
	OER  float64 `json:"oer"`
	Lic  float64 `json:"lic"`
}

// Validate ensures weights are non-negative and sum to 1.000 within 0.001 tolerance.
func (w *ProfileWeights) UnmarshalJSON(data []byte) error {
	var raw struct {
		Suff float64 `json:"suff"`
		Cov  float64 `json:"cov"`
		SLA  float64 `json:"sla"`
		Stab float64 `json:"stab"`
		OER  float64 `json:"oer"`
		Lic  float64 `json:"lic"`

		WeightSuff float64 `json:"weight_sufficiency"`
		WeightCov  float64 `json:"weight_coverage"`
		WeightSLA  float64 `json:"weight_sla"`
		WeightStab float64 `json:"weight_stability"`
		WeightOER  float64 `json:"weight_friction"`
		WeightOER2 float64 `json:"weight_oer"`
		WeightLic  float64 `json:"weight_licensing"`
	}
	type Alias ProfileWeights
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.Suff != 0 {
		w.Suff = raw.Suff
	} else {
		w.Suff = raw.WeightSuff
	}
	if raw.Cov != 0 {
		w.Cov = raw.Cov
	} else {
		w.Cov = raw.WeightCov
	}
	if raw.SLA != 0 {
		w.SLA = raw.SLA
	} else {
		w.SLA = raw.WeightSLA
	}
	if raw.Stab != 0 {
		w.Stab = raw.Stab
	} else {
		w.Stab = raw.WeightStab
	}
	if raw.OER != 0 {
		w.OER = raw.OER
	} else if raw.WeightOER != 0 {
		w.OER = raw.WeightOER
	} else {
		w.OER = raw.WeightOER2
	}
	if raw.Lic != 0 {
		w.Lic = raw.Lic
	} else {
		w.Lic = raw.WeightLic
	}
	return nil
}

func (w ProfileWeights) Validate() error {
	sum := w.Suff + w.Cov + w.SLA + w.Stab + w.OER + w.Lic
	if math.Abs(sum-1.000) > 0.001 {
		return fmt.Errorf("weights must sum to 1.000 (got %.3f)", sum)
	}
	if w.Suff < 0 || w.Cov < 0 || w.SLA < 0 || w.Stab < 0 || w.OER < 0 || w.Lic < 0 {
		return fmt.Errorf("all weights must be non-negative")
	}
	return nil
}

// ToWeightProfile converts ProfileWeights to a WeightProfile model struct.
func (w ProfileWeights) ToWeightProfile(id int64, name string) WeightProfile {
	return WeightProfile{
		ProfileID:   id,
		ProfileName: name,
		WeightSuff:  w.Suff,
		WeightCov:   w.Cov,
		WeightSLA:   w.SLA,
		WeightStab:  w.Stab,
		WeightOER:   w.OER,
		WeightLic:   w.Lic,
	}
}

// ProfileSimSpec specifies a profile for simulation: either by ProfileID or inline Weights.
type ProfileSimSpec struct {
	ProfileID *int64          `json:"profile_id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Weights   *ProfileWeights `json:"weights,omitempty"`
}

// ProfileSimulationRequest is the payload for POST /api/mdm/scoring/simulate-profiles.
type ProfileSimulationRequest struct {
	TenantID     string         `json:"tenant_id,omitempty"`
	VendorIDs    []string       `json:"vendor_ids,omitempty"`
	EntityDomain string         `json:"entity_domain,omitempty"`
	ProfileA     ProfileSimSpec `json:"profile_a"`
	ProfileB     ProfileSimSpec `json:"profile_b"`
	UniverseSize int            `json:"universe_size,omitempty"`
}

// RadarScores holds the 6 normalized pillar scores for radar visualization.
type RadarScores struct {
	SufficiencyRate     float64 `json:"sufficiency_rate"`
	CoverageRate        float64 `json:"coverage_rate"`
	SLAComplianceRate   float64 `json:"sla_compliance_rate"`
	StabilityScore      float64 `json:"stability_score"`
	StewardFrictionCost float64 `json:"steward_friction_cost"`
	RightsScore         float64 `json:"rights_score"`
}

// VendorRankingEntry is a vendor's quality score and rank under a specific profile.
type VendorRankingEntry struct {
	VendorID              string      `json:"vendor_id"`
	VendorName            string      `json:"vendor_name"`
	Rank                  int         `json:"rank"`
	CompositeQualityScore float64     `json:"composite_quality_score"`
	Radar                 RadarScores `json:"radar"`
	AnnualSpend           float64     `json:"annual_spend"`
	CostPerQualityPoint   float64     `json:"cost_per_quality_point"`
}

// ProfileEvaluationResult contains rankings and metadata for a single profile in a simulation.
type ProfileEvaluationResult struct {
	ProfileID   int64                `json:"profile_id"`
	ProfileName string               `json:"profile_name"`
	Weights     ProfileWeights       `json:"weights"`
	Rankings    []VendorRankingEntry `json:"rankings"`
}

// VendorRankShift captures rank and score movement between Profile A and Profile B.
type VendorRankShift struct {
	VendorID   string  `json:"vendor_id"`
	VendorName string  `json:"vendor_name"`
	RankA      int     `json:"rank_a"`
	RankB      int     `json:"rank_b"`
	RankDelta  int     `json:"rank_delta"` // rank_a - rank_b (positive = higher in B)
	ScoreA     float64 `json:"score_a"`
	ScoreB     float64 `json:"score_b"`
	ScoreDelta float64 `json:"score_delta"` // score_b - score_a
}

// BundleOptimalSummary summarizes the optimal bundle solved under a profile.
type BundleOptimalSummary struct {
	Vendors              []string `json:"vendors"`
	Cost                 float64  `json:"cost"`
	CompositeCoveragePct float64  `json:"composite_coverage_pct"`
}

// BundleSimulationImpact captures optimal bundle cost delta and rationale.
type BundleSimulationImpact struct {
	ProfileAOptimal BundleOptimalSummary `json:"profile_a_optimal"`
	ProfileBOptimal BundleOptimalSummary `json:"profile_b_optimal"`
	BundleDeltaCost float64              `json:"bundle_delta_cost"` // b_cost - a_cost
	Insight         string               `json:"insight"`
}

// ProfileSimulationResponse is the return payload for POST /api/mdm/scoring/simulate-profiles.
type ProfileSimulationResponse struct {
	AsOfDate     time.Time               `json:"as_of_date"`
	TenantID     string                  `json:"tenant_id"`
	ProfileA     ProfileEvaluationResult `json:"profile_a"`
	ProfileB     ProfileEvaluationResult `json:"profile_b"`
	RankShifts   []VendorRankShift       `json:"rank_shifts"`
	BundleImpact BundleSimulationImpact  `json:"bundle_impact"`
}





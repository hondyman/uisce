package wealth

import (
	"time"

	"github.com/shopspring/decimal"
)

// ============================================================================
// CORE DOMAIN TYPES
// ============================================================================

// FamilyOffice represents a multi-generational family office entity
type FamilyOffice struct {
	FamilyID               string                 `json:"family_id" db:"family_id"`
	TenantID               string                 `json:"tenant_id" db:"tenant_id"`
	FamilyName             string                 `json:"family_name" db:"family_name"`
	LegalEntityName        *string                `json:"legal_entity_name,omitempty" db:"legal_entity_name"`
	PrimaryAdvisorID       *string                `json:"primary_advisor_id,omitempty" db:"primary_advisor_id"`
	BackupAdvisorID        *string                `json:"backup_advisor_id,omitempty" db:"backup_advisor_id"`
	TotalEstimatedNetworth decimal.Decimal        `json:"total_estimated_networth" db:"total_estimated_networth"`
	TotalLiquidAssets      decimal.Decimal        `json:"total_liquid_assets" db:"total_liquid_assets"`
	TotalIlliquidAssets    decimal.Decimal        `json:"total_illiquid_assets" db:"total_illiquid_assets"`
	TotalLiabilities       decimal.Decimal        `json:"total_liabilities" db:"total_liabilities"`
	EstatePlanStatus       string                 `json:"estate_plan_status" db:"estate_plan_status"`
	LastPlanReviewDate     *time.Time             `json:"last_plan_review_date,omitempty" db:"last_plan_review_date"`
	NextPlanReviewDate     *time.Time             `json:"next_plan_review_date,omitempty" db:"next_plan_review_date"`
	HasFamilyConstitution  bool                   `json:"has_family_constitution" db:"has_family_constitution"`
	GovernanceStructure    map[string]interface{} `json:"governance_structure,omitempty" db:"governance_structure"`
	PatriarchID            *string                `json:"patriarch_id,omitempty" db:"patriarch_id"`
	MatriarchID            *string                `json:"matriarch_id,omitempty" db:"matriarch_id"`
	GenerationCount        int                    `json:"generation_count" db:"generation_count"`
	CreatedAt              time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time              `json:"updated_at" db:"updated_at"`
	CreatedBy              *string                `json:"created_by,omitempty" db:"created_by"`
	DeletedAt              *time.Time             `json:"deleted_at,omitempty" db:"deleted_at"`
}

// FamilyMember represents an individual family member
type FamilyMember struct {
	MemberID                   string                 `json:"member_id" db:"member_id"`
	FamilyID                   string                 `json:"family_id" db:"family_id"`
	LegalFirstName             string                 `json:"legal_first_name" db:"legal_first_name"`
	LegalMiddleName            *string                `json:"legal_middle_name,omitempty" db:"legal_middle_name"`
	LegalLastName              string                 `json:"legal_last_name" db:"legal_last_name"`
	PreferredName              *string                `json:"preferred_name,omitempty" db:"preferred_name"`
	Suffix                     *string                `json:"suffix,omitempty" db:"suffix"`
	DateOfBirth                time.Time              `json:"date_of_birth" db:"date_of_birth"`
	SSNEncrypted               *string                `json:"ssn_encrypted,omitempty" db:"ssn_encrypted"`
	Citizenship                []string               `json:"citizenship" db:"citizenship"`
	PrimaryStateResidency      string                 `json:"primary_state_residency" db:"primary_state_residency"`
	SecondaryResidences        map[string]interface{} `json:"secondary_residences,omitempty" db:"secondary_residences"`
	DomicileState              string                 `json:"domicile_state" db:"domicile_state"`
	Generation                 int                    `json:"generation" db:"generation"`
	ParentMemberIDs            []string               `json:"parent_member_ids,omitempty" db:"parent_member_ids"`
	SpouseMemberID             *string                `json:"spouse_member_id,omitempty" db:"spouse_member_id"`
	ChildrenMemberIDs          []string               `json:"children_member_ids,omitempty" db:"children_member_ids"`
	SeparateNetworth           decimal.Decimal        `json:"separate_networth" db:"separate_networth"`
	AnnualIncome               *decimal.Decimal       `json:"annual_income,omitempty" db:"annual_income"`
	EmploymentStatus           *string                `json:"employment_status,omitempty" db:"employment_status"`
	Occupation                 *string                `json:"occupation,omitempty" db:"occupation"`
	RiskToleranceScore         *decimal.Decimal       `json:"risk_tolerance_score,omitempty" db:"risk_tolerance_score"`
	InvestmentPhilosophy       *string                `json:"investment_philosophy,omitempty" db:"investment_philosophy"`
	ESGPreferences             map[string]interface{} `json:"esg_preferences,omitempty" db:"esg_preferences"`
	FinancialLiteracyScore     *decimal.Decimal       `json:"financial_literacy_score,omitempty" db:"financial_literacy_score"`
	LiteracyAssessmentDate     *time.Time             `json:"literacy_assessment_date,omitempty" db:"literacy_assessment_date"`
	LiteracyAssessmentMethod   *string                `json:"literacy_assessment_method,omitempty" db:"literacy_assessment_method"`
	MaritalStatus              string                 `json:"marital_status" db:"marital_status"`
	MarriageDate               *time.Time             `json:"marriage_date,omitempty" db:"marriage_date"`
	PrenuptialAgreement        bool                   `json:"prenuptial_agreement" db:"prenuptial_agreement"`
	PrenupDocumentID           *string                `json:"prenup_document_id,omitempty" db:"prenup_document_id"`
	ChildrenCount              int                    `json:"children_count" db:"children_count"`
	HasSpecialNeedsDependents  bool                   `json:"has_special_needs_dependents" db:"has_special_needs_dependents"`
	SpecialNeedsDetails        map[string]interface{} `json:"special_needs_details,omitempty" db:"special_needs_details"`
	EducationLevel             *string                `json:"education_level,omitempty" db:"education_level"`
	CurrentStudent             bool                   `json:"current_student" db:"current_student"`
	StudentLoanBalance         *decimal.Decimal       `json:"student_loan_balance,omitempty" db:"student_loan_balance"`
	HasChronicHealthConditions *bool                  `json:"has_chronic_health_conditions,omitempty" db:"has_chronic_health_conditions"`
	LifeExpectancyEstimate     *int                   `json:"life_expectancy_estimate,omitempty" db:"life_expectancy_estimate"`
	LongTermCareInsurance      *bool                  `json:"long_term_care_insurance,omitempty" db:"long_term_care_insurance"`
	PlatformUserID             *string                `json:"platform_user_id,omitempty" db:"platform_user_id"`
	OnboardingStatus           string                 `json:"onboarding_status" db:"onboarding_status"`
	InvitationSentDate         *time.Time             `json:"invitation_sent_date,omitempty" db:"invitation_sent_date"`
	FirstLoginDate             *time.Time             `json:"first_login_date,omitempty" db:"first_login_date"`
	LastLoginDate              *time.Time             `json:"last_login_date,omitempty" db:"last_login_date"`
	EngagementScore            *decimal.Decimal       `json:"engagement_score,omitempty" db:"engagement_score"`
	EngagementLastCalculated   *time.Time             `json:"engagement_last_calculated,omitempty" db:"engagement_last_calculated"`
	CommunicationPreferences   map[string]interface{} `json:"communication_preferences,omitempty" db:"communication_preferences"`
	AnticipatedMajorExpenses   []MajorExpense         `json:"anticipated_major_expenses,omitempty" db:"anticipated_major_expenses"`
	RetirementTargetAge        *int                   `json:"retirement_target_age,omitempty" db:"retirement_target_age"`
	RetirementTargetDate       *time.Time             `json:"retirement_target_date,omitempty" db:"retirement_target_date"`
	CreatedAt                  time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt                  time.Time              `json:"updated_at" db:"updated_at"`
	CreatedBy                  *string                `json:"created_by,omitempty" db:"created_by"`
	DeletedAt                  *time.Time             `json:"deleted_at,omitempty" db:"deleted_at"`
}

// MajorExpense represents an anticipated future expense
type MajorExpense struct {
	Type          string          `json:"type"`
	ChildName     *string         `json:"child_name,omitempty"`
	StartYear     int             `json:"start_year"`
	EstimatedCost decimal.Decimal `json:"estimated_cost"`
}

// FamilyAsset represents an asset owned by the family
type FamilyAsset struct {
	AssetID                      string           `json:"asset_id" db:"asset_id"`
	FamilyID                     string           `json:"family_id" db:"family_id"`
	AssetClass                   string           `json:"asset_class" db:"asset_class"`
	AssetName                    string           `json:"asset_name" db:"asset_name"`
	AssetDescription             *string          `json:"asset_description,omitempty" db:"asset_description"`
	AssetIdentifier              *string          `json:"asset_identifier,omitempty" db:"asset_identifier"`
	CustodianName                *string          `json:"custodian_name,omitempty" db:"custodian_name"`
	CustodianAccountNumber       *string          `json:"custodian_account_number,omitempty" db:"custodian_account_number"`
	PhysicalLocation             *string          `json:"physical_location,omitempty" db:"physical_location"`
	CurrentValuation             decimal.Decimal  `json:"current_valuation" db:"current_valuation"`
	ValuationDate                time.Time        `json:"valuation_date" db:"valuation_date"`
	ValuationMethod              string           `json:"valuation_method" db:"valuation_method"`
	ValuationFirm                *string          `json:"valuation_firm,omitempty" db:"valuation_firm"`
	AppraisalDocumentID          *string          `json:"appraisal_document_id,omitempty" db:"appraisal_document_id"`
	CostBasis                    *decimal.Decimal `json:"cost_basis,omitempty" db:"cost_basis"`
	AcquisitionDate              *time.Time       `json:"acquisition_date,omitempty" db:"acquisition_date"`
	UnrealizedGainLoss           decimal.Decimal  `json:"unrealized_gain_loss" db:"unrealized_gain_loss"`
	SteppedUpBasisEligible       bool             `json:"stepped_up_basis_eligible" db:"stepped_up_basis_eligible"`
	DepreciationEligible         bool             `json:"depreciation_eligible" db:"depreciation_eligible"`
	AnnualDepreciation           *decimal.Decimal `json:"annual_depreciation,omitempty" db:"annual_depreciation"`
	IncludedInGrossEstate        bool             `json:"included_in_gross_estate" db:"included_in_gross_estate"`
	EstateTaxDiscountPct         decimal.Decimal  `json:"estate_tax_discount_pct" db:"estate_tax_discount_pct"`
	AdjustedEstateValue          decimal.Decimal  `json:"adjusted_estate_value" db:"adjusted_estate_value"`
	OwnershipStructure           []OwnershipShare `json:"ownership_structure" db:"ownership_structure"`
	Illiquid                     bool             `json:"illiquid" db:"illiquid"`
	EstimatedTimeToLiquidateDays *int             `json:"estimated_time_to_liquidate_days,omitempty" db:"estimated_time_to_liquidate_days"`
	EstimatedLiquidationCostPct  *decimal.Decimal `json:"estimated_liquidation_cost_pct,omitempty" db:"estimated_liquidation_cost_pct"`
	GeneratesIncome              bool             `json:"generates_income" db:"generates_income"`
	AnnualIncomeGenerated        *decimal.Decimal `json:"annual_income_generated,omitempty" db:"annual_income_generated"`
	IncomeType                   *string          `json:"income_type,omitempty" db:"income_type"`
	HasDebt                      bool             `json:"has_debt" db:"has_debt"`
	OutstandingDebtBalance       *decimal.Decimal `json:"outstanding_debt_balance,omitempty" db:"outstanding_debt_balance"`
	DebtInterestRate             *decimal.Decimal `json:"debt_interest_rate,omitempty" db:"debt_interest_rate"`
	DebtMaturityDate             *time.Time       `json:"debt_maturity_date,omitempty" db:"debt_maturity_date"`
	HasTransferRestrictions      bool             `json:"has_transfer_restrictions" db:"has_transfer_restrictions"`
	TransferRestrictionDetails   *string          `json:"transfer_restriction_details,omitempty" db:"transfer_restriction_details"`
	RightOfFirstRefusal          *bool            `json:"right_of_first_refusal,omitempty" db:"right_of_first_refusal"`
	BuySellAgreementExists       *bool            `json:"buy_sell_agreement_exists,omitempty" db:"buy_sell_agreement_exists"`
	BuySellAgreementDocumentID   *string          `json:"buy_sell_agreement_document_id,omitempty" db:"buy_sell_agreement_document_id"`
	CreatedAt                    time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt                    time.Time        `json:"updated_at" db:"updated_at"`
	CreatedBy                    *string          `json:"created_by,omitempty" db:"created_by"`
	DeletedAt                    *time.Time       `json:"deleted_at,omitempty" db:"deleted_at"`
}

// OwnershipShare represents a fractional ownership stake
type OwnershipShare struct {
	OwnerType    string          `json:"owner_type"` // INDIVIDUAL, TRUST
	OwnerID      string          `json:"owner_id"`
	OwnershipPct decimal.Decimal `json:"ownership_pct"`
}

// EstateEntity represents a trust, LLC, or foundation
type EstateEntity struct {
	EntityID                                string                 `json:"entity_id" db:"entity_id"`
	FamilyID                                string                 `json:"family_id" db:"family_id"`
	EntityType                              string                 `json:"entity_type" db:"entity_type"`
	EntityName                              string                 `json:"entity_name" db:"entity_name"`
	EntityLegalName                         *string                `json:"entity_legal_name,omitempty" db:"entity_legal_name"`
	FormationDate                           time.Time              `json:"formation_date" db:"formation_date"`
	FormationState                          string                 `json:"formation_state" db:"formation_state"`
	SitusState                              *string                `json:"situs_state,omitempty" db:"situs_state"`
	GoverningLawState                       *string                `json:"governing_law_state,omitempty" db:"governing_law_state"`
	TaxID                                   *string                `json:"tax_id,omitempty" db:"tax_id"`
	TaxIDApplicationDate                    *time.Time             `json:"tax_id_application_date,omitempty" db:"tax_id_application_date"`
	TaxClassification                       *string                `json:"tax_classification,omitempty" db:"tax_classification"`
	FormationDocumentID                     *string                `json:"formation_document_id,omitempty" db:"formation_document_id"`
	TrustAgreementDocumentID                *string                `json:"trust_agreement_document_id,omitempty" db:"trust_agreement_document_id"`
	OperatingAgreementDocumentID            *string                `json:"operating_agreement_document_id,omitempty" db:"operating_agreement_document_id"`
	AmendmentDocumentIDs                    []string               `json:"amendment_document_ids,omitempty" db:"amendment_document_ids"`
	GrantorMemberIDs                        []string               `json:"grantor_member_ids" db:"grantor_member_ids"`
	TrusteeMemberIDs                        []string               `json:"trustee_member_ids,omitempty" db:"trustee_member_ids"`
	TrusteeEntityIDs                        []string               `json:"trustee_entity_ids,omitempty" db:"trustee_entity_ids"`
	SuccessorTrusteeIDs                     []string               `json:"successor_trustee_ids,omitempty" db:"successor_trustee_ids"`
	BeneficiaryMemberIDs                    []string               `json:"beneficiary_member_ids" db:"beneficiary_member_ids"`
	ContingentBeneficiaryMemberIDs          []string               `json:"contingent_beneficiary_member_ids,omitempty" db:"contingent_beneficiary_member_ids"`
	Terms                                   map[string]interface{} `json:"terms,omitempty" db:"terms"`
	TerminationDate                         *time.Time             `json:"termination_date,omitempty" db:"termination_date"`
	TerminationEvent                        *string                `json:"termination_event,omitempty" db:"termination_event"`
	CurrentTotalValue                       decimal.Decimal        `json:"current_total_value" db:"current_total_value"`
	AssetAllocation                         map[string]interface{} `json:"asset_allocation,omitempty" db:"asset_allocation"`
	GRATAnnuityAmount                       *decimal.Decimal       `json:"grat_annuity_amount,omitempty" db:"grat_annuity_amount"`
	GRATAnnuityFrequency                    *string                `json:"grat_annuity_frequency,omitempty" db:"grat_annuity_frequency"`
	GRATTermYears                           *int                   `json:"grat_term_years,omitempty" db:"grat_term_years"`
	GRATRemainderBeneficiaries              []string               `json:"grat_remainder_beneficiaries,omitempty" db:"grat_remainder_beneficiaries"`
	ILITLifeInsurancePolicyID               *string                `json:"ilit_life_insurance_policy_id,omitempty" db:"ilit_life_insurance_policy_id"`
	ILITCrummeyWithdrawalRights             *bool                  `json:"ilit_crummey_withdrawal_rights,omitempty" db:"ilit_crummey_withdrawal_rights"`
	DynastyPerpetual                        bool                   `json:"dynasty_perpetual" db:"dynasty_perpetual"`
	DynastyGenerationLimit                  *int                   `json:"dynasty_generation_limit,omitempty" db:"dynasty_generation_limit"`
	FoundationAnnualDistributionRequirement *decimal.Decimal       `json:"foundation_annual_distribution_requirement,omitempty" db:"foundation_annual_distribution_requirement"`
	FoundationTaxYearEnd                    *time.Time             `json:"foundation_tax_year_end,omitempty" db:"foundation_tax_year_end"`
	FoundationIRSDeterminationLetterID      *string                `json:"foundation_irs_determination_letter_id,omitempty" db:"foundation_irs_determination_letter_id"`
	AnnualTaxFilingRequired                 bool                   `json:"annual_tax_filing_required" db:"annual_tax_filing_required"`
	LastTaxFilingDate                       *time.Time             `json:"last_tax_filing_date,omitempty" db:"last_tax_filing_date"`
	NextTaxFilingDueDate                    *time.Time             `json:"next_tax_filing_due_date,omitempty" db:"next_tax_filing_due_date"`
	RequiresStateRegistration               *bool                  `json:"requires_state_registration,omitempty" db:"requires_state_registration"`
	StateRegistrationNumber                 *string                `json:"state_registration_number,omitempty" db:"state_registration_number"`
	BankAccountInfo                         map[string]interface{} `json:"bank_account_info,omitempty" db:"bank_account_info"`
	EntityStatus                            string                 `json:"entity_status" db:"entity_status"`
	TerminationDateActual                   *time.Time             `json:"termination_date_actual,omitempty" db:"termination_date_actual"`
	CreatedAt                               time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt                               time.Time              `json:"updated_at" db:"updated_at"`
	CreatedBy                               *string                `json:"created_by,omitempty" db:"created_by"`
	DeletedAt                               *time.Time             `json:"deleted_at,omitempty" db:"deleted_at"`
}

// GiftHistory represents a gift transaction for exemption tracking
type GiftHistory struct {
	GiftID                    string          `json:"gift_id" db:"gift_id"`
	FamilyID                  string          `json:"family_id" db:"family_id"`
	DonorMemberID             string          `json:"donor_member_id" db:"donor_member_id"`
	RecipientMemberID         *string         `json:"recipient_member_id,omitempty" db:"recipient_member_id"`
	RecipientEntityID         *string         `json:"recipient_entity_id,omitempty" db:"recipient_entity_id"`
	GiftDate                  time.Time       `json:"gift_date" db:"gift_date"`
	GiftType                  string          `json:"gift_type" db:"gift_type"`
	AssetID                   *string         `json:"asset_id,omitempty" db:"asset_id"`
	AssetDescription          string          `json:"asset_description" db:"asset_description"`
	FairMarketValue           decimal.Decimal `json:"fair_market_value" db:"fair_market_value"`
	ValuationMethod           string          `json:"valuation_method" db:"valuation_method"`
	ValuationDocumentID       *string         `json:"valuation_document_id,omitempty" db:"valuation_document_id"`
	ValuationDiscountPct      decimal.Decimal `json:"valuation_discount_pct" db:"valuation_discount_pct"`
	NetGiftValue              decimal.Decimal `json:"net_gift_value" db:"net_gift_value"`
	AnnualExclusionUtilized   decimal.Decimal `json:"annual_exclusion_utilized" db:"annual_exclusion_utilized"`
	LifetimeExemptionUtilized decimal.Decimal `json:"lifetime_exemption_utilized" db:"lifetime_exemption_utilized"`
	GSTExemptionUtilized      decimal.Decimal `json:"gst_exemption_utilized" db:"gst_exemption_utilized"`
	SpousalSplitElection      bool            `json:"spousal_split_election" db:"spousal_split_election"`
	SpouseMemberID            *string         `json:"spouse_member_id,omitempty" db:"spouse_member_id"`
	RequiresGiftTaxReturn     bool            `json:"requires_gift_tax_return" db:"requires_gift_tax_return"`
	Form709Filed              bool            `json:"form_709_filed" db:"form_709_filed"`
	Form709FilingDate         *time.Time      `json:"form_709_filing_date,omitempty" db:"form_709_filing_date"`
	Form709DocumentID         *string         `json:"form_709_document_id,omitempty" db:"form_709_document_id"`
	Form709DueDate            *time.Time      `json:"form_709_due_date,omitempty" db:"form_709_due_date"`
	IsGenerationSkipping      bool            `json:"is_generation_skipping" db:"is_generation_skipping"`
	GenerationSkipCount       *int            `json:"generation_skip_count,omitempty" db:"generation_skip_count"`
	GiftStructure             *string         `json:"gift_structure,omitempty" db:"gift_structure"`
	GiftRestrictions          *string         `json:"gift_restrictions,omitempty" db:"gift_restrictions"`
	GiftPurpose               *string         `json:"gift_purpose,omitempty" db:"gift_purpose"`
	AdvisorNotes              *string         `json:"advisor_notes,omitempty" db:"advisor_notes"`
	CreatedAt                 time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt                 time.Time       `json:"updated_at" db:"updated_at"`
	CreatedBy                 *string         `json:"created_by,omitempty" db:"created_by"`
	DeletedAt                 *time.Time      `json:"deleted_at,omitempty" db:"deleted_at"`
}

// EstatePlanScenario represents an AI-generated estate planning scenario
type EstatePlanScenario struct {
	ScenarioID                   string                 `json:"scenario_id" db:"scenario_id"`
	FamilyID                     string                 `json:"family_id" db:"family_id"`
	ScenarioName                 string                 `json:"scenario_name" db:"scenario_name"`
	ScenarioDescription          *string                `json:"scenario_description,omitempty" db:"scenario_description"`
	StrategyType                 string                 `json:"strategy_type" db:"strategy_type"`
	StrategiesUsed               []string               `json:"strategies_used,omitempty" db:"strategies_used"`
	BaselineEstateTax            decimal.Decimal        `json:"baseline_estate_tax" db:"baseline_estate_tax"`
	ProjectedEstateTax           decimal.Decimal        `json:"projected_estate_tax" db:"projected_estate_tax"`
	TaxSavings                   decimal.Decimal        `json:"tax_savings" db:"tax_savings"`
	TaxSavingsPct                decimal.Decimal        `json:"tax_savings_pct" db:"tax_savings_pct"`
	BaselineNetToHeirs           decimal.Decimal        `json:"baseline_net_to_heirs" db:"baseline_net_to_heirs"`
	ProjectedNetToHeirs          decimal.Decimal        `json:"projected_net_to_heirs" db:"projected_net_to_heirs"`
	AdditionalWealthTransferred  decimal.Decimal        `json:"additional_wealth_transferred" db:"additional_wealth_transferred"`
	GenerationCount              int                    `json:"generation_count" db:"generation_count"`
	CompoundedBenefit30yr        *decimal.Decimal       `json:"compounded_benefit_30yr,omitempty" db:"compounded_benefit_30yr"`
	DynastyTrustPerpetualBenefit *decimal.Decimal       `json:"dynasty_trust_perpetual_benefit,omitempty" db:"dynasty_trust_perpetual_benefit"`
	ComplexityScore              int                    `json:"complexity_score" db:"complexity_score"`
	ImplementationTimeWeeks      int                    `json:"implementation_time_weeks" db:"implementation_time_weeks"`
	EstimatedImplementationCost  *decimal.Decimal       `json:"estimated_implementation_cost,omitempty" db:"estimated_implementation_cost"`
	AnnualMaintenanceCost        *decimal.Decimal       `json:"annual_maintenance_cost,omitempty" db:"annual_maintenance_cost"`
	RequiresSpousalCooperation   bool                   `json:"requires_spousal_cooperation" db:"requires_spousal_cooperation"`
	RequiresGiftTaxFiling        bool                   `json:"requires_gift_tax_filing" db:"requires_gift_tax_filing"`
	RequiresAppraisal            bool                   `json:"requires_appraisal" db:"requires_appraisal"`
	RequiresLifeInsurance        bool                   `json:"requires_life_insurance" db:"requires_life_insurance"`
	MinimumNetworthRequired      *decimal.Decimal       `json:"minimum_networth_required,omitempty" db:"minimum_networth_required"`
	EntitiesToCreate             []EntityToCreate       `json:"entities_to_create,omitempty" db:"entities_to_create"`
	AnnualGiftsTotal             *decimal.Decimal       `json:"annual_gifts_total,omitempty" db:"annual_gifts_total"`
	LifetimeExemptionUtilized    *decimal.Decimal       `json:"lifetime_exemption_utilized,omitempty" db:"lifetime_exemption_utilized"`
	GSTExemptionUtilized         *decimal.Decimal       `json:"gst_exemption_utilized,omitempty" db:"gst_exemption_utilized"`
	IRSAuditRisk                 *string                `json:"irs_audit_risk,omitempty" db:"irs_audit_risk"`
	ValuationChallengeRisk       *string                `json:"valuation_challenge_risk,omitempty" db:"valuation_challenge_risk"`
	LegislativeChangeRisk        *string                `json:"legislative_change_risk,omitempty" db:"legislative_change_risk"`
	ConfidenceScore              *decimal.Decimal       `json:"confidence_score,omitempty" db:"confidence_score"`
	ConfidenceFactors            map[string]interface{} `json:"confidence_factors,omitempty" db:"confidence_factors"`
	SuitableForRiskTolerance     []string               `json:"suitable_for_risk_tolerance,omitempty" db:"suitable_for_risk_tolerance"`
	SuitableForAgeRange          map[string]interface{} `json:"suitable_for_age_range,omitempty" db:"suitable_for_age_range"`
	SuitableForNetworthRange     map[string]interface{} `json:"suitable_for_networth_range,omitempty" db:"suitable_for_networth_range"`
	AssumedGrowthRate            decimal.Decimal        `json:"assumed_growth_rate" db:"assumed_growth_rate"`
	AssumedTaxLawChanges         map[string]interface{} `json:"assumed_tax_law_changes,omitempty" db:"assumed_tax_law_changes"`
	AssumedLifeExpectancy        *int                   `json:"assumed_life_expectancy,omitempty" db:"assumed_life_expectancy"`
	RankByTaxSavings             *int                   `json:"rank_by_tax_savings,omitempty" db:"rank_by_tax_savings"`
	RankBySimplicity             *int                   `json:"rank_by_simplicity,omitempty" db:"rank_by_simplicity"`
	RankByOverallScore           *int                   `json:"rank_by_overall_score,omitempty" db:"rank_by_overall_score"`
	NarrativeExplanation         *string                `json:"narrative_explanation,omitempty" db:"narrative_explanation"`
	ImplementationChecklist      []ChecklistItem        `json:"implementation_checklist,omitempty" db:"implementation_checklist"`
	CreatedAt                    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt                    time.Time              `json:"updated_at" db:"updated_at"`
	CreatedBy                    *string                `json:"created_by,omitempty" db:"created_by"`
}

// EntityToCreate represents a trust or entity to be established
type EntityToCreate struct {
	Type          string          `json:"type"`
	FundingAmount decimal.Decimal `json:"funding_amount"`
	Beneficiaries []string        `json:"beneficiaries"`
}

// ChecklistItem represents an implementation step
type ChecklistItem struct {
	Step        int    `json:"step"`
	Description string `json:"description"`
	Responsible string `json:"responsible"`
	Deadline    string `json:"deadline,omitempty"`
	Completed   bool   `json:"completed"`
}

// TaxJurisdiction represents configurable tax rates
type TaxJurisdiction struct {
	JurisdictionID        string                 `json:"jurisdiction_id" db:"jurisdiction_id"`
	JurisdictionCode      string                 `json:"jurisdiction_code" db:"jurisdiction_code"`
	JurisdictionName      string                 `json:"jurisdiction_name" db:"jurisdiction_name"`
	JurisdictionType      string                 `json:"jurisdiction_type" db:"jurisdiction_type"`
	EstateTaxApplies      bool                   `json:"estate_tax_applies" db:"estate_tax_applies"`
	EstateTaxExemption    *decimal.Decimal       `json:"estate_tax_exemption,omitempty" db:"estate_tax_exemption"`
	EstateTaxRateSchedule []TaxBracket           `json:"estate_tax_rate_schedule,omitempty" db:"estate_tax_rate_schedule"`
	GiftTaxApplies        bool                   `json:"gift_tax_applies" db:"gift_tax_applies"`
	AnnualGiftExclusion   *decimal.Decimal       `json:"annual_gift_exclusion,omitempty" db:"annual_gift_exclusion"`
	LifetimeGiftExemption *decimal.Decimal       `json:"lifetime_gift_exemption,omitempty" db:"lifetime_gift_exemption"`
	GSTTaxApplies         bool                   `json:"gst_tax_applies" db:"gst_tax_applies"`
	GSTTaxExemption       *decimal.Decimal       `json:"gst_tax_exemption,omitempty" db:"gst_tax_exemption"`
	GSTTaxRate            *decimal.Decimal       `json:"gst_tax_rate,omitempty" db:"gst_tax_rate"`
	EffectiveDate         time.Time              `json:"effective_date" db:"effective_date"`
	ExpirationDate        *time.Time             `json:"expiration_date,omitempty" db:"expiration_date"`
	SunsetProvisions      map[string]interface{} `json:"sunset_provisions,omitempty" db:"sunset_provisions"`
	Notes                 *string                `json:"notes,omitempty" db:"notes"`
	SourceURL             *string                `json:"source_url,omitempty" db:"source_url"`
	LastVerifiedDate      *time.Time             `json:"last_verified_date,omitempty" db:"last_verified_date"`
	CreatedAt             time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at" db:"updated_at"`
}

// TaxBracket represents a progressive tax rate tier
type TaxBracket struct {
	Threshold decimal.Decimal `json:"threshold"`
	Rate      decimal.Decimal `json:"rate"`
}

// ============================================================================
// TAX OPTIMIZATION TYPES (WealthVision Phase 1)
// ============================================================================

// TaxStrategy represents a tax optimization strategy
type TaxStrategy struct {
	StrategyID          string                 `json:"strategy_id" db:"strategy_id"`
	FamilyID            string                 `json:"family_id" db:"family_id"`
	StrategyType        string                 `json:"strategy_type" db:"strategy_type"` // STATE_RESIDENCY, NIIT, CHARITABLE_BUNCHING, SALT, FTC
	AnalysisDate        time.Time              `json:"analysis_date" db:"analysis_date"`
	BaselineTax         decimal.Decimal        `json:"baseline_tax" db:"baseline_tax"`
	OptimizedTax        decimal.Decimal        `json:"optimized_tax" db:"optimized_tax"`
	TaxSavings          decimal.Decimal        `json:"tax_savings" db:"tax_savings"`
	TaxSavingsPct       decimal.Decimal        `json:"tax_savings_pct" db:"tax_savings_pct"`
	Recommendations     map[string]interface{} `json:"recommendations" db:"recommendations"`
	ImplementationSteps []ImplementationStep   `json:"implementation_steps" db:"implementation_steps"`
	Complexity          string                 `json:"complexity" db:"complexity"` // LOW, MEDIUM, HIGH
	TimeToImplementDays int                    `json:"time_to_implement_days" db:"time_to_implement_days"`
	EstimatedCost       *decimal.Decimal       `json:"estimated_cost,omitempty" db:"estimated_cost"`
	RiskLevel           string                 `json:"risk_level" db:"risk_level"` // LOW, MEDIUM, HIGH
	CreatedAt           time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at" db:"updated_at"`
	CreatedBy           *string                `json:"created_by,omitempty" db:"created_by"`
}

// ImplementationStep represents an action item for implementing a tax strategy
type ImplementationStep struct {
	Step        int    `json:"step"`
	Action      string `json:"action"`
	Responsible string `json:"responsible"` // ADVISOR, CLIENT, CPA, ATTORNEY
	Deadline    string `json:"deadline,omitempty"`
	Status      string `json:"status"` // PENDING, IN_PROGRESS, COMPLETED
}

// StateResidencyComparison compares tax implications of different state residencies
type StateResidencyComparison struct {
	FamilyID                 string                 `json:"family_id"`
	AnalysisDate             time.Time              `json:"analysis_date"`
	CurrentState             string                 `json:"current_state"`
	GrossIncome              decimal.Decimal        `json:"gross_income"`
	InvestmentIncome         decimal.Decimal        `json:"investment_income"`
	CapitalGains             decimal.Decimal        `json:"capital_gains"`
	EstateValue              decimal.Decimal        `json:"estate_value"`
	StateComparisons         []StateResidencyDetail `json:"state_comparisons"`
	TopRecommendations       []StateResidencyDetail `json:"top_recommendations"`
	AdditionalConsiderations []string               `json:"additional_considerations"`
}

// StateResidencyDetail represents tax details for a specific state
type StateResidencyDetail struct {
	StateCode                string           `json:"state_code"`
	StateName                string           `json:"state_name"`
	IncomeTax                decimal.Decimal  `json:"income_tax"`
	IncomeTaxRate            decimal.Decimal  `json:"income_tax_rate"`
	CapitalGainsTax          decimal.Decimal  `json:"capital_gains_tax"`
	CapitalGainsTaxRate      decimal.Decimal  `json:"capital_gains_tax_rate"`
	PropertyTax              decimal.Decimal  `json:"property_tax"`
	SalesTax                 decimal.Decimal  `json:"sales_tax"`
	EstateTax                decimal.Decimal  `json:"estate_tax"`
	TotalAnnualTax           decimal.Decimal  `json:"total_annual_tax"`
	AnnualSavingsVsCurrent   decimal.Decimal  `json:"annual_savings_vs_current"`
	LifetimeSavings30yr      decimal.Decimal  `json:"lifetime_savings_30yr"`
	HasIncomeTax             bool             `json:"has_income_tax"`
	HasEstateTax             bool             `json:"has_estate_tax"`
	ResidencyRequirementDays int              `json:"residency_requirement_days"`
	QualityOfLifeScore       *decimal.Decimal `json:"quality_of_life_score,omitempty"`
	Notes                    string           `json:"notes,omitempty"`
}

// NIITCalculation represents Net Investment Income Tax (3.8%) calculation
type NIITCalculation struct {
	FamilyID                   string                    `json:"family_id"`
	MemberID                   string                    `json:"member_id"`
	TaxYear                    int                       `json:"tax_year"`
	FilingStatus               string                    `json:"filing_status"` // SINGLE, MARRIED_JOINT, MARRIED_SEPARATE, HEAD_OF_HOUSEHOLD
	ModifiedAGI                decimal.Decimal           `json:"modified_agi"`
	NIITThreshold              decimal.Decimal           `json:"niit_threshold"`
	NetInvestmentIncome        decimal.Decimal           `json:"net_investment_income"`
	InvestmentIncomeComponents InvestmentIncomeBreakdown `json:"investment_income_components"`
	ExcessOverThreshold        decimal.Decimal           `json:"excess_over_threshold"`
	TaxableNII                 decimal.Decimal           `json:"taxable_nii"`
	NIITax                     decimal.Decimal           `json:"nii_tax"`
	EffectiveNIITRate          decimal.Decimal           `json:"effective_niit_rate"`
	MitigationStrategies       []MitigationStrategy      `json:"mitigation_strategies"`
}

// InvestmentIncomeBreakdown breaks down net investment income components
type InvestmentIncomeBreakdown struct {
	Interest            decimal.Decimal `json:"interest"`
	Dividends           decimal.Decimal `json:"dividends"`
	CapitalGains        decimal.Decimal `json:"capital_gains"`
	RentalIncome        decimal.Decimal `json:"rental_income"`
	PassiveIncome       decimal.Decimal `json:"passive_income"`
	AnnuityIncome       decimal.Decimal `json:"annuity_income"`
	RoyaltyIncome       decimal.Decimal `json:"royalty_income"`
	TotalGrossIncome    decimal.Decimal `json:"total_gross_income"`
	Deductions          decimal.Decimal `json:"deductions"`
	NetInvestmentIncome decimal.Decimal `json:"net_investment_income"`
}

// MitigationStrategy represents a strategy to reduce NIIT
type MitigationStrategy struct {
	StrategyName       string          `json:"strategy_name"`
	EstimatedReduction decimal.Decimal `json:"estimated_reduction"`
	ImplementationCost decimal.Decimal `json:"implementation_cost"`
	NetBenefit         decimal.Decimal `json:"net_benefit"`
	Difficulty         string          `json:"difficulty"` // LOW, MEDIUM, HIGH
	Description        string          `json:"description"`
	RequiresCPAConsult bool            `json:"requires_cpa_consult"`
}

// CharitableBunchingAnalysis analyzes bunching charitable contributions
type CharitableBunchingAnalysis struct {
	FamilyID               string                   `json:"family_id"`
	MemberID               string                   `json:"member_id"`
	AnalysisYears          int                      `json:"analysis_years"` // e.g., 5 years
	AnnualCharitableGiving decimal.Decimal          `json:"annual_charitable_giving"`
	StandardDeduction      decimal.Decimal          `json:"standard_deduction"`
	ItemizedDeductions     decimal.Decimal          `json:"itemized_deductions"`
	MarginalTaxRate        decimal.Decimal          `json:"marginal_tax_rate"`
	BaselineScenario       CharitableGivingScenario `json:"baseline_scenario"`
	BunchingScenario       CharitableGivingScenario `json:"bunching_scenario"`
	RecommendedStrategy    string                   `json:"recommended_strategy"` // ANNUAL, BUNCHING_2YR, BUNCHING_3YR
	EstimatedTaxSavings    decimal.Decimal          `json:"estimated_tax_savings"`
	DAFRecommendation      *DAFRecommendation       `json:"daf_recommendation,omitempty"`
}

// CharitableGivingScenario represents a scenario for charitable giving
type CharitableGivingScenario struct {
	ScenarioName           string                 `json:"scenario_name"`
	YearByYearBreakdown    []CharitableYearDetail `json:"year_by_year_breakdown"`
	TotalContributions     decimal.Decimal        `json:"total_contributions"`
	TotalDeductions        decimal.Decimal        `json:"total_deductions"`
	TotalTaxSavings        decimal.Decimal        `json:"total_tax_savings"`
	EffectiveDeductionRate decimal.Decimal        `json:"effective_deduction_rate"`
}

// CharitableYearDetail represents giving details for one year
type CharitableYearDetail struct {
	Year              int             `json:"year"`
	Contribution      decimal.Decimal `json:"contribution"`
	ItemizesDeduction bool            `json:"itemizes_deduction"`
	DeductionTaken    decimal.Decimal `json:"deduction_taken"`
	TaxSavings        decimal.Decimal `json:"tax_savings"`
}

// DAFRecommendation represents donor-advised fund recommendation
type DAFRecommendation struct {
	RecommendedProvider string          `json:"recommended_provider"`
	InitialContribution decimal.Decimal `json:"initial_contribution"`
	ProjectedGrowth     decimal.Decimal `json:"projected_growth"`
	AnnualGrantBudget   decimal.Decimal `json:"annual_grant_budget"`
	EstimatedFees       decimal.Decimal `json:"estimated_fees"`
	NetCharitableImpact decimal.Decimal `json:"net_charitable_impact"`
}

// SALTCapStrategy represents SALT cap workaround strategy
type SALTCapStrategy struct {
	FamilyID                    string          `json:"family_id"`
	TaxYear                     int             `json:"tax_year"`
	StateCode                   string          `json:"state_code"`
	CurrentSALTDeductions       decimal.Decimal `json:"current_salt_deductions"`
	SALTCap                     decimal.Decimal `json:"salt_cap"` // $10,000
	LostDeduction               decimal.Decimal `json:"lost_deduction"`
	LostTaxBenefit              decimal.Decimal `json:"lost_tax_benefit"`
	PTETAvailable               bool            `json:"ptet_available"`
	PTETEstimatedBenefit        decimal.Decimal `json:"ptet_estimated_benefit"`
	PTETBusinessEntities        []string        `json:"ptet_business_entities,omitempty"`
	OtherStrategies             []string        `json:"other_strategies,omitempty"`
	RecommendedAction           string          `json:"recommended_action"`
	EstimatedImplementationCost decimal.Decimal `json:"estimated_implementation_cost"`
}

// ForeignTaxCreditOptimization represents FTC optimization
type ForeignTaxCreditOptimization struct {
	FamilyID                string          `json:"family_id"`
	TaxYear                 int             `json:"tax_year"`
	ForeignIncomeTotal      decimal.Decimal `json:"foreign_income_total"`
	ForeignTaxesPaid        decimal.Decimal `json:"foreign_taxes_paid"`
	USIncomeTotal           decimal.Decimal `json:"us_income_total"`
	FTCLimit                decimal.Decimal `json:"ftc_limit"`
	FTCAvailable            decimal.Decimal `json:"ftc_available"`
	FTCUtilized             decimal.Decimal `json:"ftc_utilized"`
	FTCCarryforward         decimal.Decimal `json:"ftc_carryforward"`
	StrategyType            string          `json:"strategy_type"` // CREDIT, DEDUCTION, MIXED
	OptimizedFTCUtilization decimal.Decimal `json:"optimized_ftc_utilization"`
	EstimatedTaxSavings     decimal.Decimal `json:"estimated_tax_savings"`
}

// ============================================================================
// TAX CALC ENGINE ADAPTER TYPES
// ============================================================================

// FederalEstateTaxInput represents input for federal estate tax calculation
type FederalEstateTaxInput struct {
	GrossEstateValue     decimal.Decimal `json:"gross_estate_value"`
	PriorLifetimeGifts   decimal.Decimal `json:"prior_lifetime_gifts"`
	CharitableDeductions decimal.Decimal `json:"charitable_deductions"`
	MaritalDeductions    decimal.Decimal `json:"marital_deductions"`
	TaxYear              int             `json:"tax_year"`
}

// FederalEstateTaxResult represents the result of federal estate tax calculation
type FederalEstateTaxResult struct {
	GrossEstateValue  decimal.Decimal          `json:"gross_estate_value"`
	TotalDeductions   decimal.Decimal          `json:"total_deductions"`
	ExemptionAmount   decimal.Decimal          `json:"exemption_amount"`
	TaxableAmount     decimal.Decimal          `json:"taxable_amount"`
	TaxOwed           decimal.Decimal          `json:"tax_owed"`
	EffectiveTaxRate  float64                  `json:"effective_tax_rate"`
	BracketDetails    []TaxBracketDetail       `json:"bracket_details"`
	CalculationMethod string                   `json:"calculation_method"`
	Sources           []map[string]interface{} `json:"sources,omitempty"`
}

// TaxBracketDetail represents enhanced tax bracket details
type TaxBracketDetail struct {
	MinAmount    decimal.Decimal `json:"min_amount"`
	MaxAmount    decimal.Decimal `json:"max_amount"`
	Rate         float64         `json:"rate"`
	TaxInBracket decimal.Decimal `json:"tax_in_bracket"`
}

// StateTaxInput represents input for state tax calculation
type StateTaxInput struct {
	GrossEstateValue decimal.Decimal `json:"gross_estate_value"`
	StateCode        string          `json:"state_code"`
	TaxYear          int             `json:"tax_year"`
}

// StateTaxResult represents the result of state tax calculation
type StateTaxResult struct {
	StateCode        string                   `json:"state_code"`
	GrossEstateValue decimal.Decimal          `json:"gross_estate_value"`
	StateExemption   decimal.Decimal          `json:"state_exemption"`
	TaxableAmount    decimal.Decimal          `json:"taxable_amount"`
	TaxOwed          decimal.Decimal          `json:"tax_owed"`
	EffectiveTaxRate float64                  `json:"effective_tax_rate"`
	Sources          []map[string]interface{} `json:"sources,omitempty"`
}

// GiftTaxInput represents input for gift tax calculation
type GiftTaxInput struct {
	GiftValue                  decimal.Decimal `json:"gift_value"`
	AnnualExclusionAvailable   decimal.Decimal `json:"annual_exclusion_available"`
	LifetimeExemptionAvailable decimal.Decimal `json:"lifetime_exemption_available"`
	SpousalSplitElection       bool            `json:"spousal_split_election"`
	TaxYear                    int             `json:"tax_year"`
}

// GiftTaxResult represents the result of gift tax calculation
type GiftTaxResult struct {
	GiftValue                 decimal.Decimal          `json:"gift_value"`
	AnnualExclusionUtilized   decimal.Decimal          `json:"annual_exclusion_utilized"`
	LifetimeExemptionUtilized decimal.Decimal          `json:"lifetime_exemption_utilized"`
	TaxableGift               decimal.Decimal          `json:"taxable_gift"`
	GiftTaxOwed               decimal.Decimal          `json:"gift_tax_owed"`
	RequiresForm709           bool                     `json:"requires_form_709"`
	CalculationMethod         string                   `json:"calculation_method"`
	Sources                   []map[string]interface{} `json:"sources,omitempty"`
}

// Form709 represents a completed IRS Form 709
type Form709 struct {
	FormID        string `json:"form_id"`
	FamilyID      string `json:"family_id"`
	DonorMemberID string `json:"donor_member_id"`
	DonorName     string `json:"donor_name"`
	DonorSSN      string `json:"donor_ssn"`
	SpouseName    string `json:"spouse_name,omitempty"`
	SpouseSSN     string `json:"spouse_ssn,omitempty"`
	TaxYear       int    `json:"tax_year"`

	// Part 1: Computation of Taxable Gifts
	TotalGiftsMade   float64 `json:"total_gifts_made"`
	AnnualExclusions float64 `json:"annual_exclusions"`
	DeductibleGifts  float64 `json:"deductible_gifts"`
	TaxableGifts     float64 `json:"taxable_gifts"`

	// Part 2: Tax Computation
	LifetimeExemptionUsed      float64 `json:"lifetime_exemption_used"`
	LifetimeExemptionRemaining float64 `json:"lifetime_exemption_remaining"`
	TotalGiftTax               float64 `json:"total_gift_tax"`

	// Part 3: GST Tax
	GSTTransfers     []GSTTransfer `json:"gst_transfers"`
	TotalGSTTax      float64       `json:"total_gst_tax"`
	GSTExemptionUsed float64       `json:"gst_exemption_used"`

	// Totals
	TotalTaxDue float64 `json:"total_tax_due"`

	// Filing info
	FilingStatus       string `json:"filing_status"` // PREPARED, FILED, FAILED
	FilingDate         string `json:"filing_date,omitempty"`
	FilingConfirmation string `json:"filing_confirmation,omitempty"`
	FilingError        string `json:"filing_error,omitempty"`
	PDFPath            string `json:"pdf_path,omitempty"`
}

// GSTTransfer represents a generation-skipping transfer
type GSTTransfer struct {
	RecipientName      string  `json:"recipient_name"`
	TransferValue      float64 `json:"transfer_value"`
	ExemptionAllocated float64 `json:"exemption_allocated"`
	TaxableAmount      float64 `json:"taxable_amount"`
	GSTTax             float64 `json:"gst_tax"`
}

// GiftForFiling represents a gift that requires filing
type GiftForFiling struct {
	GiftID                    string  `json:"gift_id"`
	DonorMemberID             string  `json:"donor_member_id"`
	RecipientName             string  `json:"recipient_name"`
	GiftDate                  string  `json:"gift_date"`
	GiftValue                 float64 `json:"gift_value"`
	AnnualExclusionUtilized   float64 `json:"annual_exclusion_utilized"`
	LifetimeExemptionUtilized float64 `json:"lifetime_exemption_utilized"`
	IsGenerationSkipping      bool    `json:"is_generation_skipping"`
	GenerationSkipCount       int     `json:"generation_skip_count"`
}

// TaxLawChange represents a tax law change
type TaxLawChange struct {
	JurisdictionCode string `json:"jurisdiction_code"`
	Description      string `json:"description"`
	EffectiveDate    string `json:"effective_date"`
	ImpactLevel      string `json:"impact_level"` // LOW, MEDIUM, HIGH
}

// FamilyProfile is a convenience struct for the recommender
type FamilyProfile struct {
	FamilyID            string
	FamilyName          string
	TotalNetworth       decimal.Decimal
	LiquidAssetPct      decimal.Decimal
	BusinessInterestPct decimal.Decimal
	MarriedCouple       bool
	HasChildren         bool
	HasGrandchildren    bool
	GenerationCount     int
	OldestMemberAge     int
	Members             []FamilyMember
	Assets              []FamilyAsset // Added missing field
	PrimaryState        string
}

// EstatePlanGenerationInput input for plan generation workflow
type EstatePlanGenerationInput struct {
	FamilyID           string `json:"family_id"`
	MaxScenarios       int    `json:"max_scenarios"`
	GenerateNarratives bool   `json:"generate_narratives"`
}

// EstatePlanGenerationResult result from plan generation workflow
type EstatePlanGenerationResult struct {
	FamilyID           string               `json:"family_id"`
	ScenariosGenerated int                  `json:"scenarios_generated"`
	Scenarios          []EstatePlanScenario `json:"scenarios"`
}

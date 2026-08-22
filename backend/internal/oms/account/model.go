package account

import (
	"time"

	"github.com/google/uuid"
)

type AccountSubtype string

const (
	SubtypeInstitutional      AccountSubtype = "institutional"
	SubtypeRetailWealth       AccountSubtype = "retail_wealth"
	SubtypeSMA                AccountSubtype = "sma"
	SubtypeTrustEstate        AccountSubtype = "trust_estate"
	SubtypeQualifiedRetirement AccountSubtype = "qualified_retirement"
	SubtypeCorporateTreasury  AccountSubtype = "corporate_treasury"
)

var validAccountSubtypes = map[AccountSubtype]bool{
	SubtypeInstitutional:       true,
	SubtypeRetailWealth:        true,
	SubtypeSMA:                 true,
	SubtypeTrustEstate:         true,
	SubtypeQualifiedRetirement: true,
	SubtypeCorporateTreasury:   true,
}

type AccountRecord struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	TenantID        uuid.UUID `db:"tenant_id" json:"tenant_id"`
	AccountNumber   string    `db:"account_number" json:"account_number"`
	AccountName     string    `db:"account_name" json:"account_name"`
	BaseCurrency    string    `db:"base_currency" json:"base_currency"`
	Status          string    `db:"status" json:"status"`
	SubtypeCode     string    `db:"subtype_code" json:"subtype_code"`

	SponsorID              *uuid.UUID `db:"sponsor_id" json:"sponsor_id,omitempty"`
	MandateType            *string    `db:"mandate_type" json:"mandate_type,omitempty"`
	ErisaFlag              *bool      `db:"erisa_flag" json:"erisa_flag,omitempty"`
	FeeScheduleCode         *string    `db:"fee_schedule_code" json:"fee_schedule_code,omitempty"`
	TaxIDType              *string    `db:"tax_id_type" json:"tax_id_type,omitempty"`
	Citizenship            *string    `db:"citizenship" json:"citizenship,omitempty"`
	MarginAgreementFlag    *bool      `db:"margin_agreement_flag" json:"margin_agreement_flag,omitempty"`
	AccreditedInvestorStatus *string   `db:"accredited_investor_status" json:"accredited_investor_status,omitempty"`
	SponsorFirm            *string    `db:"sponsor_firm" json:"sponsor_firm,omitempty"`
	ModelStrategyID        *uuid.UUID `db:"model_strategy_id" json:"model_strategy_id,omitempty"`
	OverlayManagerID       *uuid.UUID `db:"overlay_manager_id" json:"overlay_manager_id,omitempty"`
	RebalanceFrequency     *string    `db:"rebalance_frequency" json:"rebalance_frequency,omitempty"`
	TrustType              *string    `db:"trust_type" json:"trust_type,omitempty"`
	GrantorName            *string    `db:"grantor_name" json:"grantor_name,omitempty"`
	TrusteeSignatoryID     *uuid.UUID `db:"trustee_signatory_id" json:"trustee_signatory_id,omitempty"`
	DissolutionDate        *time.Time `db:"dissolution_date" json:"dissolution_date,omitempty"`
	PlanType               *string    `db:"plan_type" json:"plan_type,omitempty"`
	VestingScheduleCode    *string    `db:"vesting_schedule_code" json:"vesting_schedule_code,omitempty"`
	RMDEligibleFlag        *bool      `db:"rmd_eligible_flag" json:"rmd_eligible_flag,omitempty"`
	CustodianBankID        *uuid.UUID `db:"custodian_bank_id" json:"custodian_bank_id,omitempty"`
	CorporateEntityID      *uuid.UUID `db:"corporate_entity_id" json:"corporate_entity_id,omitempty"`
	TreasurySignatoryGroup *string    `db:"treasury_signatory_group" json:"treasury_signatory_group,omitempty"`
	WireLimitDaily         *float64   `db:"wire_limit_daily" json:"wire_limit_daily,omitempty"`

	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
	ValidFrom   time.Time  `db:"valid_from" json:"valid_from"`
	ValidTo     *time.Time `db:"valid_to" json:"valid_to,omitempty"`
}

func (a AccountRecord) Validate() error {
	if !validAccountSubtypes[AccountSubtype(a.SubtypeCode)] {
		return ErrInvalidSubtype
	}
	if a.SubtypeCode == string(SubtypeInstitutional) && a.SponsorID == nil {
		return ErrRequiresSponsorID
	}
	if a.SubtypeCode == string(SubtypeQualifiedRetirement) && a.ErisaFlag != nil && *a.ErisaFlag && a.PlanType == nil {
		return ErrRequiresPlanType
	}
	return nil
}

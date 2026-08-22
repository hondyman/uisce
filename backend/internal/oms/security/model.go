package security

import (
	"time"

	"github.com/google/uuid"
)

type SecuritySubtype string

const (
	SubtypeEquity          SecuritySubtype = "equity"
	SubtypeSovereignDebt   SecuritySubtype = "sovereign_debt"
	SubtypeCorporateDebt   SecuritySubtype = "corporate_debt"
	SubtypeStructuredABS   SecuritySubtype = "structured_abs_mbs"
	SubtypeETDDerivative   SecuritySubtype = "etd_derivative"
	SubtypeOTCDerivative   SecuritySubtype = "otc_derivative"
)

var validSecuritySubtypes = map[SecuritySubtype]bool{
	SubtypeEquity:        true,
	SubtypeSovereignDebt: true,
	SubtypeCorporateDebt: true,
	SubtypeStructuredABS:  true,
	SubtypeETDDerivative: true,
	SubtypeOTCDerivative: true,
}

type SecurityRecord struct {
	ID               uuid.UUID  `db:"id" json:"id"`
	TenantID         uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	SecurityName     string     `db:"security_name" json:"security_name"`
	IdentifierType   string     `db:"identifier_type" json:"identifier_type"`
	IdentifierValue  string     `db:"identifier_value" json:"identifier_value"`
	SubtypeCode      string     `db:"subtype_code" json:"subtype_code"`

	Ticker              *string    `db:"ticker" json:"ticker,omitempty"`
	ISIN                *string    `db:"isin" json:"isin,omitempty"`
	VotingRightsType    *string    `db:"voting_rights_type" json:"voting_rights_type,omitempty"`
	DividendCurrency     *string    `db:"dividend_currency" json:"dividend_currency,omitempty"`
	CouponRate          *float64   `db:"coupon_rate" json:"coupon_rate,omitempty"`
	MaturityDate        *time.Time `db:"maturity_date" json:"maturity_date,omitempty"`
	DayCountConvention  *string    `db:"day_count_convention" json:"day_count_convention,omitempty"`
	InflationProtectedFlag *bool    `db:"inflation_protected_flag" json:"inflation_protected_flag,omitempty"`
	CreditRatingSP      *string    `db:"credit_rating_sp" json:"credit_rating_sp,omitempty"`
	CallDate            *time.Time `db:"call_date" json:"call_date,omitempty"`
	ConversionRatio     *float64   `db:"conversion_ratio" json:"conversion_ratio,omitempty"`
	SeniorityLevel      *string    `db:"seniority_level" json:"seniority_level,omitempty"`
	PoolNumber          *string    `db:"pool_number" json:"pool_number,omitempty"`
	FactorCurrent       *float64   `db:"factor_current" json:"factor_current,omitempty"`
	PrepaymentSpeedCPR  *float64   `db:"prepayment_speed_cpr" json:"prepayment_speed_cpr,omitempty"`
	TrancheTier         *string    `db:"tranche_tier" json:"tranche_tier,omitempty"`
	ContractSize        *float64   `db:"contract_size" json:"contract_size,omitempty"`
	StrikePrice         *float64   `db:"strike_price" json:"strike_price,omitempty"`
	PutCallIndicator    *string    `db:"put_call_indicator" json:"put_call_indicator,omitempty"`
	ExchangeMIC         *string    `db:"exchange_mic" json:"exchange_mic,omitempty"`
	ISDAAgreementID     *uuid.UUID `db:"isda_agreement_id" json:"isda_agreement_id,omitempty"`
	FixedRate           *float64   `db:"fixed_rate" json:"fixed_rate,omitempty"`
	FloatingIndexName   *string    `db:"floating_index_name" json:"floating_index_name,omitempty"`
	CounterpartyLEI     *string    `db:"counterparty_lei" json:"counterparty_lei,omitempty"`

	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	ValidFrom time.Time  `db:"valid_from" json:"valid_from"`
	ValidTo   *time.Time `db:"valid_to" json:"valid_to,omitempty"`
}

func (s SecurityRecord) Validate() error {
	if !validSecuritySubtypes[SecuritySubtype(s.SubtypeCode)] {
		return ErrInvalidSubtype
	}
	return nil
}

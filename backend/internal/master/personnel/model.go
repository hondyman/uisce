package personnel

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PersonnelSubtype string

const (
	SubtypePortfolioManager     PersonnelSubtype = "portfolio_manager"
	SubtypeTradeExecution      PersonnelSubtype = "trade_execution"
	SubtypeComplianceOfficer   PersonnelSubtype = "compliance_officer"
	SubtypeClientAdvisor       PersonnelSubtype = "client_advisor"
)

var validPersonnelSubtypes = map[PersonnelSubtype]bool{
	SubtypePortfolioManager:   true,
	SubtypeTradeExecution:     true,
	SubtypeComplianceOfficer:  true,
	SubtypeClientAdvisor:      true,
}

type PersonnelRecord struct {
	ID                       uuid.UUID        `db:"id" json:"id"`
	TenantID                 uuid.UUID        `db:"tenant_id" json:"tenant_id"`
	FullName                 string           `db:"full_name" json:"full_name"`
	Email                    string           `db:"email" json:"email"`
	SubtypeCode              string           `db:"subtype_code" json:"subtype_code"`
	CRDNumber                *string          `db:"crd_number" json:"crd_number,omitempty"`
	SeriesLicensesHeld       *[]string        `db:"series_licenses_held" json:"series_licenses_held,omitempty"`
	SupervisoryID            *uuid.UUID       `db:"supervisory_id" json:"supervisory_id,omitempty"`
	DiscretionaryAuthorityLimit *decimal.Decimal `db:"discretionary_authority_limit" json:"discretionary_authority_limit,omitempty"`

	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	ValidFrom time.Time  `db:"valid_from" json:"valid_from"`
	ValidTo   *time.Time `db:"valid_to" json:"valid_to,omitempty"`
}

func (p PersonnelRecord) Validate() error {
	if !validPersonnelSubtypes[PersonnelSubtype(p.SubtypeCode)] {
		return ErrInvalidSubtype
	}
	return nil
}

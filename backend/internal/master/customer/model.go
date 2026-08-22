package customer

import (
	"time"

	"github.com/google/uuid"
)

type CustomerSubtype string

const (
	SubtypeInstitutionalClient CustomerSubtype = "institutional_client"
	SubtypePrivateWealth      CustomerSubtype = "private_wealth"
	SubtypeBrokerDealer       CustomerSubtype = "broker_dealer"
	SubtypeCorporateTreasury  CustomerSubtype = "corporate_treasury"
)

var validCustomerSubtypes = map[CustomerSubtype]bool{
	SubtypeInstitutionalClient: true,
	SubtypePrivateWealth:       true,
	SubtypeBrokerDealer:        true,
	SubtypeCorporateTreasury:  true,
}

type CustomerRecord struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	TenantID        uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	CustomerName    string     `db:"customer_name" json:"customer_name"`
	SubtypeCode     string     `db:"subtype_code" json:"subtype_code"`
	LEICode         *string    `db:"lei_code" json:"lei_code,omitempty"`
	KYCStatus       string     `db:"kyc_status" json:"kyc_status"`
	SuitabilityProfile *string `db:"suitability_profile" json:"suitability_profile,omitempty"`
	RelationshipTier *string   `db:"relationship_tier" json:"relationship_tier,omitempty"`
	ParentGroupID   *uuid.UUID `db:"parent_group_id" json:"parent_group_id,omitempty"`

	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	ValidFrom time.Time  `db:"valid_from" json:"valid_from"`
	ValidTo   *time.Time `db:"valid_to" json:"valid_to,omitempty"`
}

func (c CustomerRecord) Validate() error {
	if !validCustomerSubtypes[CustomerSubtype(c.SubtypeCode)] {
		return ErrInvalidSubtype
	}
	return nil
}

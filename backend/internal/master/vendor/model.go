package vendor

import (
	"time"

	"github.com/google/uuid"
)

type VendorSubtype string

const (
	SubtypeCustodianPrimeBroker VendorSubtype = "custodian_prime_broker"
	SubtypeMarketData           VendorSubtype = "market_data"
	SubtypeFundAdmin            VendorSubtype = "fund_admin"
	SubtypeCloudTech            VendorSubtype = "cloud_tech"
)

var validVendorSubtypes = map[VendorSubtype]bool{
	SubtypeCustodianPrimeBroker: true,
	SubtypeMarketData:           true,
	SubtypeFundAdmin:            true,
	SubtypeCloudTech:            true,
}

type VendorRecord struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	TenantID        uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	VendorName      string     `db:"vendor_name" json:"vendor_name"`
	SubtypeCode     string     `db:"subtype_code" json:"subtype_code"`
	VendorCategory  *string    `db:"vendor_category" json:"vendor_category,omitempty"`
	SLATier         *string    `db:"sla_tier" json:"sla_tier,omitempty"`
	SOC2CertDate    *time.Time `db:"soc2_certification_date" json:"soc2_certification_date,omitempty"`
	SOC1Type2OnFile *bool     `db:"soc1_type2_on_file" json:"soc1_type2_on_file,omitempty"`
	BillingCycle    *string    `db:"billing_cycle" json:"billing_cycle,omitempty"`

	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	ValidFrom time.Time  `db:"valid_from" json:"valid_from"`
	ValidTo   *time.Time `db:"valid_to" json:"valid_to,omitempty"`
}

func (v VendorRecord) Validate() error {
	if !validVendorSubtypes[VendorSubtype(v.SubtypeCode)] {
		return ErrInvalidSubtype
	}
	return nil
}

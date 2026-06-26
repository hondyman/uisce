package household

import (
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx/types"
)

type EntityType string

const (
	EntityIndividual EntityType = "INDIVIDUAL"
	EntityJoint      EntityType = "JOINT"
	EntityTrust      EntityType = "TRUST"
	EntityLLC        EntityType = "LLC"
	EntityFoundation EntityType = "FOUNDATION"
	EntityEstate     EntityType = "ESTATE"
)

type Entity struct {
	EntityID    uuid.UUID  `db:"entity_id" json:"entity_id"`
	EntityType  EntityType `db:"entity_type" json:"entity_type"`
	EntityName  string     `db:"entity_name" json:"entity_name"`
	TaxID       *string    `db:"tax_id" json:"tax_id,omitempty"`
	ParentID    *uuid.UUID `db:"parent_entity_id" json:"parent_entity_id,omitempty"`
	HouseholdID uuid.UUID  `db:"household_id" json:"household_id"`

	// Trust Specific
	TrustType        *string     `db:"trust_type" json:"trust_type,omitempty"`
	TrusteeIDs       []uuid.UUID `db:"trustee_ids" json:"trustee_ids,omitempty"` // Requires pq.Array handling in scan
	GrantorID        *uuid.UUID  `db:"grantor_id" json:"grantor_id,omitempty"`
	BeneficiaryIDs   []uuid.UUID `db:"beneficiary_ids" json:"beneficiary_ids,omitempty"`
	TrustTermination *time.Time  `db:"trust_termination_date" json:"trust_termination_date,omitempty"`

	// Foundation Specific
	FoundationType *string  `db:"foundation_type" json:"foundation_type,omitempty"`
	AnnualDistReq  *float64 `db:"annual_distribution_requirement" json:"annual_distribution_requirement,omitempty"`

	// LLC Specific
	OwnershipStructure types.JSONText `db:"ownership_structure" json:"ownership_structure,omitempty"`
	OperatingAgreement *string        `db:"operating_agreement_url" json:"operating_agreement_url,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type Household struct {
	HouseholdID   uuid.UUID `db:"household_id" json:"household_id"`
	HouseholdName string    `db:"household_name" json:"household_name"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

type InterEntityTransfer struct {
	TransferID       uuid.UUID `db:"transfer_id" json:"transfer_id"`
	FromEntityID     uuid.UUID `db:"from_entity_id" json:"from_entity_id"`
	ToEntityID       uuid.UUID `db:"to_entity_id" json:"to_entity_id"`
	TransferDate     time.Time `db:"transfer_date" json:"transfer_date"`
	Amount           float64   `db:"amount" json:"amount"`
	AssetDescription *string   `db:"asset_description" json:"asset_description,omitempty"`
	TransferReason   string    `db:"transfer_reason" json:"transfer_reason"`
	GiftTaxRequired  bool      `db:"gift_tax_return_required" json:"gift_tax_return_required"`
	GST              bool      `db:"generation_skipping_transfer" json:"generation_skipping_transfer"`
	AdvisorNotes     *string   `db:"advisor_notes" json:"advisor_notes,omitempty"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

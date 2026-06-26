package wealth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// TrustEntityService handles trust and estate entity management
type TrustEntityService struct {
	db *pgxpool.Pool
}

// NewTrustEntityService creates a new trust entity service
func NewTrustEntityService(db *pgxpool.Pool) *TrustEntityService {
	return &TrustEntityService{
		db: db,
	}
}

// CreateTrustInput represents input for creating a trust
type CreateTrustInput struct {
	FamilyID             string                 `json:"family_id"`
	EntityType           string                 `json:"entity_type"`
	EntityName           string                 `json:"entity_name"`
	EntityLegalName      *string                `json:"entity_legal_name,omitempty"`
	FormationDate        time.Time              `json:"formation_date"`
	FormationState       string                 `json:"formation_state"`
	GrantorMemberIDs     []string               `json:"grantor_member_ids"`
	TrusteeMemberIDs     []string               `json:"trustee_member_ids,omitempty"`
	BeneficiaryMemberIDs []string               `json:"beneficiary_member_ids"`
	Terms                map[string]interface{} `json:"terms,omitempty"`
	CreatedBy            *string                `json:"created_by,omitempty"`
}

// CreateTrust creates a new trust or estate entity
func (s *TrustEntityService) CreateTrust(ctx context.Context, input CreateTrustInput) (*EstateEntity, error) {
	entityID := uuid.New().String()
	now := time.Now()

	termsJSON, _ := json.Marshal(input.Terms)

	query := `
		INSERT INTO estate_entities (
			entity_id, family_id, entity_type, entity_name, entity_legal_name,
			formation_date, formation_state,
			grantor_member_ids, trustee_member_ids, beneficiary_member_ids,
			terms, entity_status, created_at, updated_at, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING entity_id, family_id, entity_type, entity_name, formation_date,
			entity_status, created_at
	`

	var entity EstateEntity
	err := s.db.QueryRow(ctx, query,
		entityID,
		input.FamilyID,
		input.EntityType,
		input.EntityName,
		input.EntityLegalName,
		input.FormationDate,
		input.FormationState,
		input.GrantorMemberIDs,
		input.TrusteeMemberIDs,
		input.BeneficiaryMemberIDs,
		termsJSON,
		"ACTIVE",
		now,
		now,
		input.CreatedBy,
	).Scan(
		&entity.EntityID,
		&entity.FamilyID,
		&entity.EntityType,
		&entity.EntityName,
		&entity.FormationDate,
		&entity.EntityStatus,
		&entity.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create trust: %w", err)
	}

	return &entity, nil
}

// GetTrust retrieves a trust by ID
func (s *TrustEntityService) GetTrust(ctx context.Context, entityID string) (*EstateEntity, error) {
	query := `
		SELECT entity_id, family_id, entity_type, entity_name, entity_legal_name,
			formation_date, formation_state, situs_state, governing_law_state,
			tax_id, tax_classification,
			grantor_member_ids, trustee_member_ids, beneficiary_member_ids,
			successor_trustee_ids, contingent_beneficiary_member_ids,
			terms, current_total_value, entity_status,
			annual_tax_filing_required, last_tax_filing_date, next_tax_filing_due_date,
			created_at, updated_at
		FROM estate_entities
		WHERE entity_id = $1 AND deleted_at IS NULL
	`

	var entity EstateEntity
	var termsJSON []byte

	err := s.db.QueryRow(ctx, query, entityID).Scan(
		&entity.EntityID,
		&entity.FamilyID,
		&entity.EntityType,
		&entity.EntityName,
		&entity.EntityLegalName,
		&entity.FormationDate,
		&entity.FormationState,
		&entity.SitusState,
		&entity.GoverningLawState,
		&entity.TaxID,
		&entity.TaxClassification,
		&entity.GrantorMemberIDs,
		&entity.TrusteeMemberIDs,
		&entity.BeneficiaryMemberIDs,
		&entity.SuccessorTrusteeIDs,
		&entity.ContingentBeneficiaryMemberIDs,
		&termsJSON,
		&entity.CurrentTotalValue,
		&entity.EntityStatus,
		&entity.AnnualTaxFilingRequired,
		&entity.LastTaxFilingDate,
		&entity.NextTaxFilingDueDate,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get trust: %w", err)
	}

	// Parse terms JSON
	if len(termsJSON) > 0 {
		if err := json.Unmarshal(termsJSON, &entity.Terms); err != nil {
			return nil, fmt.Errorf("failed to parse terms: %w", err)
		}
	}

	return &entity, nil
}

// ListTrusts lists all trusts for a family
func (s *TrustEntityService) ListTrusts(ctx context.Context, familyID string) ([]EstateEntity, error) {
	query := `
		SELECT entity_id, family_id, entity_type, entity_name,
			formation_date, current_total_value, entity_status,
			annual_tax_filing_required, next_tax_filing_due_date,
			created_at
		FROM estate_entities
		WHERE family_id = $1 AND deleted_at IS NULL
		ORDER BY formation_date DESC
	`

	rows, err := s.db.Query(ctx, query, familyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entities := []EstateEntity{}
	for rows.Next() {
		var e EstateEntity
		err := rows.Scan(
			&e.EntityID,
			&e.FamilyID,
			&e.EntityType,
			&e.EntityName,
			&e.FormationDate,
			&e.CurrentTotalValue,
			&e.EntityStatus,
			&e.AnnualTaxFilingRequired,
			&e.NextTaxFilingDueDate,
			&e.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}

	return entities, nil
}

// ValidateTrustCompliance checks trust compliance requirements
func (s *TrustEntityService) ValidateTrustCompliance(ctx context.Context, entityID string) ([]ComplianceIssue, error) {
	entity, err := s.GetTrust(ctx, entityID)
	if err != nil {
		return nil, err
	}

	issues := []ComplianceIssue{}

	// Check if tax filing is overdue
	if entity.AnnualTaxFilingRequired && entity.NextTaxFilingDueDate != nil {
		if entity.NextTaxFilingDueDate.Before(time.Now()) {
			issues = append(issues, ComplianceIssue{
				Severity:    "HIGH",
				IssueType:   "TAX_FILING_OVERDUE",
				Description: fmt.Sprintf("Tax filing due date %s has passed", entity.NextTaxFilingDueDate.Format("2006-01-02")),
				Remediation: "File trust tax return immediately to avoid penalties",
			})
		}
	}

	// Check if trustee is assigned
	if len(entity.TrusteeMemberIDs) == 0 && len(entity.TrusteeEntityIDs) == 0 {
		issues = append(issues, ComplianceIssue{
			Severity:    "HIGH",
			IssueType:   "NO_TRUSTEE",
			Description: "Trust has no assigned trustee",
			Remediation: "Appoint a qualified trustee",
		})
	}

	// Check if beneficiaries are defined
	if len(entity.BeneficiaryMemberIDs) == 0 {
		issues = append(issues, ComplianceIssue{
			Severity:    "MEDIUM",
			IssueType:   "NO_BENEFICIARIES",
			Description: "Trust has no defined beneficiaries",
			Remediation: "Define beneficiaries in trust terms",
		})
	}

	// Check if tax ID is assigned (for irrevocable trusts)
	isIrrevocable := entity.EntityType != "REVOCABLE_TRUST"
	if isIrrevocable && (entity.TaxID == nil || *entity.TaxID == "") {
		issues = append(issues, ComplianceIssue{
			Severity:    "MEDIUM",
			IssueType:   "NO_TAX_ID",
			Description: "Irrevocable trust requires its own Tax ID (EIN)",
			Remediation: "Apply for EIN with IRS Form SS-4",
		})
	}

	return issues, nil
}

// ComplianceIssue represents a trust compliance issue
type ComplianceIssue struct {
	Severity    string `json:"severity"`
	IssueType   string `json:"issue_type"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

// CalculateTrustValue calculates total value of assets held by trust
func (s *TrustEntityService) CalculateTrustValue(ctx context.Context, entityID string) (decimal.Decimal, error) {
	query := `
		SELECT COALESCE(SUM(
			fa.current_valuation * (ownership->>'ownership_pct')::DECIMAL / 100.0
		), 0)
		FROM family_assets fa,
		jsonb_array_elements(fa.ownership_structure) as ownership
		WHERE ownership->>'owner_id' = $1
			AND ownership->>'owner_type' = 'TRUST'
			AND fa.deleted_at IS NULL
	`

	var totalValue decimal.Decimal
	err := s.db.QueryRow(ctx, query, entityID).Scan(&totalValue)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to calculate trust value: %w", err)
	}

	// Update entity's current_total_value
	updateQuery := `
		UPDATE estate_entities
		SET current_total_value = $1,
			updated_at = NOW()
		WHERE entity_id = $2
	`

	_, err = s.db.Exec(ctx, updateQuery, totalValue, entityID)
	if err != nil {
		return totalValue, fmt.Errorf("failed to update trust value: %w", err)
	}

	return totalValue, nil
}

// UpdateTaxFilingStatus updates trust tax filing status
func (s *TrustEntityService) UpdateTaxFilingStatus(ctx context.Context, entityID string, filingDate time.Time) error {
	// Set next filing due date (1 year from filing + extension period)
	nextDueDate := filingDate.AddDate(1, 0, 0)

	query := `
		UPDATE estate_entities
		SET last_tax_filing_date = $1,
			next_tax_filing_due_date = $2,
			updated_at = NOW()
		WHERE entity_id = $3
	`

	_, err := s.db.Exec(ctx, query, filingDate, nextDueDate, entityID)
	if err != nil {
		return fmt.Errorf("failed to update tax filing status: %w", err)
	}

	return nil
}

// TerminateTrust terminates a trust
func (s *TrustEntityService) TerminateTrust(ctx context.Context, entityID string, terminationDate time.Time, reason string) error {
	query := `
		UPDATE estate_entities
		SET entity_status = 'TERMINATED',
			termination_date_actual = $1,
			termination_event = $2,
			updated_at = NOW()
		WHERE entity_id = $3
	`

	_, err := s.db.Exec(ctx, query, terminationDate, reason, entityID)
	if err != nil {
		return fmt.Errorf("failed to terminate trust: %w", err)
	}

	return nil
}

// GetTrustsByType retrieves trusts filtered by type
func (s *TrustEntityService) GetTrustsByType(ctx context.Context, familyID string, entityType string) ([]EstateEntity, error) {
	query := `
		SELECT entity_id, family_id, entity_type, entity_name,
			formation_date, current_total_value, entity_status,
			created_at
		FROM estate_entities
		WHERE family_id = $1
			AND entity_type = $2
			AND deleted_at IS NULL
		ORDER BY formation_date DESC
	`

	rows, err := s.db.Query(ctx, query, familyID, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entities := []EstateEntity{}
	for rows.Next() {
		var e EstateEntity
		err := rows.Scan(
			&e.EntityID,
			&e.FamilyID,
			&e.EntityType,
			&e.EntityName,
			&e.FormationDate,
			&e.CurrentTotalValue,
			&e.EntityStatus,
			&e.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}

	return entities, nil
}

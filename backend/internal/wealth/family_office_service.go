package wealth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// FamilyOfficeService handles operations for family offices and members
type FamilyOfficeService struct {
	db *pgxpool.Pool
}

// NewFamilyOfficeService creates a new family office service
func NewFamilyOfficeService(db *pgxpool.Pool) *FamilyOfficeService {
	return &FamilyOfficeService{
		db: db,
	}
}

// CreateFamilyOfficeInput represents input for creating a family office
type CreateFamilyOfficeInput struct {
	TenantID               string                 `json:"tenant_id"`
	FamilyName             string                 `json:"family_name"`
	LegalEntityName        *string                `json:"legal_entity_name,omitempty"`
	PrimaryAdvisorID       *string                `json:"primary_advisor_id,omitempty"`
	BackupAdvisorID        *string                `json:"backup_advisor_id,omitempty"`
	TotalEstimatedNetworth decimal.Decimal        `json:"total_estimated_networth"`
	GovernanceStructure    map[string]interface{} `json:"governance_structure,omitempty"`
	CreatedBy              *string                `json:"created_by,omitempty"`
}

// CreateFamilyOffice creates a new family office
func (s *FamilyOfficeService) CreateFamilyOffice(ctx context.Context, input CreateFamilyOfficeInput) (*FamilyOffice, error) {
	familyID := uuid.New().String()
	now := time.Now()

	governanceJSON, _ := json.Marshal(input.GovernanceStructure)

	query := `
		INSERT INTO family_offices (
			family_id, tenant_id, family_name, legal_entity_name,
			primary_advisor_id, backup_advisor_id, total_estimated_networth,
			governance_structure, created_at, updated_at, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING *
	`

	var family FamilyOffice
	err := s.db.QueryRow(ctx, query,
		familyID,
		input.TenantID,
		input.FamilyName,
		input.LegalEntityName,
		input.PrimaryAdvisorID,
		input.BackupAdvisorID,
		input.TotalEstimatedNetworth,
		governanceJSON,
		now,
		now,
		input.CreatedBy,
	).Scan(
		&family.FamilyID,
		&family.TenantID,
		&family.FamilyName,
		&family.LegalEntityName,
		&family.PrimaryAdvisorID,
		&family.BackupAdvisorID,
		&family.TotalEstimatedNetworth,
		&family.TotalLiquidAssets,
		&family.TotalIlliquidAssets,
		&family.TotalLiabilities,
		&family.EstatePlanStatus,
		&family.LastPlanReviewDate,
		&family.NextPlanReviewDate,
		&family.HasFamilyConstitution,
		&family.GovernanceStructure,
		&family.PatriarchID,
		&family.MatriarchID,
		&family.GenerationCount,
		&family.CreatedAt,
		&family.UpdatedAt,
		&family.CreatedBy,
		&family.DeletedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create family office: %w", err)
	}

	return &family, nil
}

// GetFamilyOffice retrieves a family office by ID
func (s *FamilyOfficeService) GetFamilyOffice(ctx context.Context, familyID string) (*FamilyOffice, error) {
	query := `
		SELECT family_id, tenant_id, family_name, legal_entity_name,
			primary_advisor_id, backup_advisor_id, total_estimated_networth,
			total_liquid_assets, total_illiquid_assets, total_liabilities,
			estate_plan_status, last_plan_review_date, next_plan_review_date,
			has_family_constitution, governance_structure, patriarch_id,
			matriarch_id, generation_count, created_at, updated_at, created_by, deleted_at
		FROM family_offices
		WHERE family_id = $1 AND deleted_at IS NULL
	`

	var family FamilyOffice
	err := s.db.QueryRow(ctx, query, familyID).Scan(
		&family.FamilyID,
		&family.TenantID,
		&family.FamilyName,
		&family.LegalEntityName,
		&family.PrimaryAdvisorID,
		&family.BackupAdvisorID,
		&family.TotalEstimatedNetworth,
		&family.TotalLiquidAssets,
		&family.TotalIlliquidAssets,
		&family.TotalLiabilities,
		&family.EstatePlanStatus,
		&family.LastPlanReviewDate,
		&family.NextPlanReviewDate,
		&family.HasFamilyConstitution,
		&family.GovernanceStructure,
		&family.PatriarchID,
		&family.MatriarchID,
		&family.GenerationCount,
		&family.CreatedAt,
		&family.UpdatedAt,
		&family.CreatedBy,
		&family.DeletedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("family office not found: %s", familyID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get family office: %w", err)
	}

	return &family, nil
}

// AddFamilyMemberInput represents input for adding a family member
type AddFamilyMemberInput struct {
	FamilyID              string           `json:"family_id"`
	LegalFirstName        string           `json:"legal_first_name"`
	LegalMiddleName       *string          `json:"legal_middle_name,omitempty"`
	LegalLastName         string           `json:"legal_last_name"`
	PreferredName         *string          `json:"preferred_name,omitempty"`
	DateOfBirth           time.Time        `json:"date_of_birth"`
	PrimaryStateResidency string           `json:"primary_state_residency"`
	DomicileState         string           `json:"domicile_state"`
	Generation            int              `json:"generation"`
	ParentMemberIDs       []string         `json:"parent_member_ids,omitempty"`
	SpouseMemberID        *string          `json:"spouse_member_id,omitempty"`
	ChildrenMemberIDs     []string         `json:"children_member_ids,omitempty"`
	SeparateNetworth      decimal.Decimal  `json:"separate_networth"`
	AnnualIncome          *decimal.Decimal `json:"annual_income,omitempty"`
	MaritalStatus         string           `json:"marital_status"`
	CreatedBy             *string          `json:"created_by,omitempty"`
}

// AddFamilyMember adds a member to a family
func (s *FamilyOfficeService) AddFamilyMember(ctx context.Context, input AddFamilyMemberInput) (*FamilyMember, error) {
	memberID := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO family_members (
			member_id, family_id, legal_first_name, legal_middle_name, legal_last_name, 
			preferred_name, date_of_birth, primary_state_residency, domicile_state,
			generation, parent_member_ids, spouse_member_id, children_member_ids,
			separate_networth, annual_income, marital_status,
			created_at, updated_at, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING member_id, family_id, legal_first_name, legal_last_name, date_of_birth,
			generation, primary_state_residency, domicile_state, separate_networth,
			marital_status, created_at, updated_at
	`

	var member FamilyMember
	err := s.db.QueryRow(ctx, query,
		memberID,
		input.FamilyID,
		input.LegalFirstName,
		input.LegalMiddleName,
		input.LegalLastName,
		input.PreferredName,
		input.DateOfBirth,
		input.PrimaryStateResidency,
		input.DomicileState,
		input.Generation,
		input.ParentMemberIDs,
		input.SpouseMemberID,
		input.ChildrenMemberIDs,
		input.SeparateNetworth,
		input.AnnualIncome,
		input.MaritalStatus,
		now,
		now,
		input.CreatedBy,
	).Scan(
		&member.MemberID,
		&member.FamilyID,
		&member.LegalFirstName,
		&member.LegalLastName,
		&member.DateOfBirth,
		&member.Generation,
		&member.PrimaryStateResidency,
		&member.DomicileState,
		&member.SeparateNetworth,
		&member.MaritalStatus,
		&member.CreatedAt,
		&member.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to add family member: %w", err)
	}

	// Trigger will automatically update family aggregates
	return &member, nil
}

// GetFamilyMembers retrieves all members of a family
func (s *FamilyOfficeService) GetFamilyMembers(ctx context.Context, familyID string) ([]FamilyMember, error) {
	query := `
		SELECT member_id, family_id, legal_first_name, legal_middle_name, legal_last_name,
			preferred_name, suffix, date_of_birth, primary_state_residency, domicile_state,
			generation, parent_member_ids, spouse_member_id, children_member_ids,
			separate_networth, annual_income, employment_status, marital_status,
			children_count, engagement_score, onboarding_status,
			created_at, updated_at
		FROM family_members
		WHERE family_id = $1 AND deleted_at IS NULL
		ORDER BY generation, date_of_birth
	`

	rows, err := s.db.Query(ctx, query, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query family members: %w", err)
	}
	defer rows.Close()

	members := []FamilyMember{}
	for rows.Next() {
		var m FamilyMember
		err := rows.Scan(
			&m.MemberID,
			&m.FamilyID,
			&m.LegalFirstName,
			&m.LegalMiddleName,
			&m.LegalLastName,
			&m.PreferredName,
			&m.Suffix,
			&m.DateOfBirth,
			&m.PrimaryStateResidency,
			&m.DomicileState,
			&m.Generation,
			&m.ParentMemberIDs,
			&m.SpouseMemberID,
			&m.ChildrenMemberIDs,
			&m.SeparateNetworth,
			&m.AnnualIncome,
			&m.EmploymentStatus,
			&m.MaritalStatus,
			&m.ChildrenCount,
			&m.EngagementScore,
			&m.OnboardingStatus,
			&m.CreatedAt,
			&m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan family member: %w", err)
		}
		members = append(members, m)
	}

	return members, nil
}

// FamilyTreeNode represents a node in the family tree
type FamilyTreeNode struct {
	Member   FamilyMember     `json:"member"`
	Spouse   *FamilyMember    `json:"spouse,omitempty"`
	Children []FamilyTreeNode `json:"children,omitempty"`
	Parents  []FamilyMember   `json:"parents,omitempty"`
}

// GetFamilyTree builds a hierarchical family tree
func (s *FamilyOfficeService) GetFamilyTree(ctx context.Context, familyID string) (*FamilyTreeNode, error) {
	// Get all members
	members, err := s.GetFamilyMembers(ctx, familyID)
	if err != nil {
		return nil, err
	}

	if len(members) == 0 {
		return nil, fmt.Errorf("no members found for family: %s", familyID)
	}

	// Build a map for quick lookups
	memberMap := make(map[string]FamilyMember)
	for _, m := range members {
		memberMap[m.MemberID] = m
	}

	// Find the patriarch/matriarch (generation 1, oldest)
	family, err := s.GetFamilyOffice(ctx, familyID)
	if err != nil {
		return nil, err
	}

	var root FamilyMember
	if family.PatriarchID != nil {
		root = memberMap[*family.PatriarchID]
	} else {
		// Find oldest gen 1 member
		for _, m := range members {
			if m.Generation == 1 {
				root = m
				break
			}
		}
	}

	// Build tree recursively
	tree := s.buildFamilyTreeNode(root, memberMap)
	return tree, nil
}

func (s *FamilyOfficeService) buildFamilyTreeNode(member FamilyMember, memberMap map[string]FamilyMember) *FamilyTreeNode {
	node := &FamilyTreeNode{
		Member:   member,
		Children: []FamilyTreeNode{},
	}

	// Add spouse
	if member.SpouseMemberID != nil {
		if spouse, ok := memberMap[*member.SpouseMemberID]; ok {
			node.Spouse = &spouse
		}
	}

	// Add children recursively
	if member.ChildrenMemberIDs != nil {
		for _, childID := range member.ChildrenMemberIDs {
			if child, ok := memberMap[childID]; ok {
				childNode := s.buildFamilyTreeNode(child, memberMap)
				node.Children = append(node.Children, *childNode)
			}
		}
	}

	return node
}

// CalculateFamilyNetworth recalculates total networth from all members and assets
func (s *FamilyOfficeService) CalculateFamilyNetworth(ctx context.Context, familyID string) (decimal.Decimal, error) {
	query := `
		UPDATE family_offices
		SET total_estimated_networth = (
			SELECT COALESCE(SUM(separate_networth), 0)
			FROM family_members
			WHERE family_id = $1 AND deleted_at IS NULL
		),
		updated_at = NOW()
		WHERE family_id = $1
		RETURNING total_estimated_networth
	`

	var networth decimal.Decimal
	err := s.db.QueryRow(ctx, query, familyID).Scan(&networth)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to calculate networth: %w", err)
	}

	return networth, nil
}

// UpdateEngagementScores calculates engagement scores for all family members
func (s *FamilyOfficeService) UpdateEngagementScores(ctx context.Context) error {
	// AI-driven engagement scoring based on platform activity
	query := `
		UPDATE family_members
		SET engagement_score = CASE
			WHEN last_login_date IS NULL THEN 0.0
			WHEN last_login_date > NOW() - INTERVAL '7 days' THEN 0.9
			WHEN last_login_date > NOW() - INTERVAL '30 days' THEN 0.7
			WHEN last_login_date > NOW() - INTERVAL '90 days' THEN 0.5
			WHEN last_login_date > NOW() - INTERVAL '180 days' THEN 0.3
			ELSE 0.1
		END,
		engagement_last_calculated = NOW()
		WHERE deleted_at IS NULL
	`

	_, err := s.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to update engagement scores: %w", err)
	}

	return nil
}

// ListFamilyOffices lists all family offices for a tenant
func (s *FamilyOfficeService) ListFamilyOffices(ctx context.Context, tenantID string) ([]FamilyOffice, error) {
	query := `
		SELECT family_id, tenant_id, family_name, total_estimated_networth,
			estate_plan_status, last_plan_review_date, next_plan_review_date,
			generation_count, created_at, updated_at
		FROM family_offices
		WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY family_name
	`

	rows, err := s.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query family offices: %w", err)
	}
	defer rows.Close()

	families := []FamilyOffice{}
	for rows.Next() {
		var f FamilyOffice
		err := rows.Scan(
			&f.FamilyID,
			&f.TenantID,
			&f.FamilyName,
			&f.TotalEstimatedNetworth,
			&f.EstatePlanStatus,
			&f.LastPlanReviewDate,
			&f.NextPlanReviewDate,
			&f.GenerationCount,
			&f.CreatedAt,
			&f.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan family office: %w", err)
		}
		families = append(families, f)
	}

	return families, nil
}

// UpdateEstatePlanStatus updates the estate planning status
func (s *FamilyOfficeService) UpdateEstatePlanStatus(ctx context.Context, familyID string, status string, reviewDate *time.Time) error {
	query := `
		UPDATE family_offices
		SET estate_plan_status = $1,
			last_plan_review_date = $2,
			next_plan_review_date = $2 + INTERVAL '1 year',
			updated_at = NOW()
		WHERE family_id = $3
	`

	_, err := s.db.Exec(ctx, query, status, reviewDate, familyID)
	if err != nil {
		return fmt.Errorf("failed to update estate plan status: %w", err)
	}

	return nil
}

// GetFamilyProfile builds a comprehensive profile for AI processing
func (s *FamilyOfficeService) GetFamilyProfile(ctx context.Context, familyID string) (*FamilyProfile, error) {
	family, err := s.GetFamilyOffice(ctx, familyID)
	if err != nil {
		return nil, err
	}

	members, err := s.GetFamilyMembers(ctx, familyID)
	if err != nil {
		return nil, err
	}

	// Get assets
	assets, err := s.getFamilyAssets(ctx, familyID)
	if err != nil {
		return nil, err
	}

	// Build profile
	profile := &FamilyProfile{
		FamilyID:            family.FamilyID,
		FamilyName:          family.FamilyName,
		TotalNetworth:       family.TotalEstimatedNetworth,
		GenerationCount:     family.GenerationCount,
		PrimaryState:        members[0].PrimaryStateResidency, // Use first member's state
		Members:             members,
		Assets:              assets,
		MarriedCouple:       s.checkMarriedCouple(members),
		HasChildren:         s.checkHasChildren(members),
		HasGrandchildren:    s.checkHasGrandchildren(members),
		OldestMemberAge:     s.calculateOldestAge(members),
		LiquidAssetPct:      s.calculateLiquidPct(assets),
		BusinessInterestPct: s.calculateBusinessPct(assets),
	}

	return profile, nil
}

// FamilyProfile definition moved to types.go

func (s *FamilyOfficeService) getFamilyAssets(ctx context.Context, familyID string) ([]FamilyAsset, error) {
	query := `SELECT asset_id, family_id, asset_class, asset_name, current_valuation, illiquid 
	          FROM family_assets WHERE family_id = $1 AND deleted_at IS NULL`

	rows, err := s.db.Query(ctx, query, familyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assets := []FamilyAsset{}
	for rows.Next() {
		var a FamilyAsset
		err := rows.Scan(&a.AssetID, &a.FamilyID, &a.AssetClass, &a.AssetName, &a.CurrentValuation, &a.Illiquid)
		if err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}
	return assets, nil
}

func (s *FamilyOfficeService) checkMarriedCouple(members []FamilyMember) bool {
	for _, m := range members {
		if m.MaritalStatus == "MARRIED" && m.Generation == 1 {
			return true
		}
	}
	return false
}

func (s *FamilyOfficeService) checkHasChildren(members []FamilyMember) bool {
	for _, m := range members {
		if m.Generation == 2 {
			return true
		}
	}
	return false
}

func (s *FamilyOfficeService) checkHasGrandchildren(members []FamilyMember) bool {
	for _, m := range members {
		if m.Generation == 3 {
			return true
		}
	}
	return false
}

func (s *FamilyOfficeService) calculateOldestAge(members []FamilyMember) int {
	var oldestAge int
	for _, m := range members {
		age := time.Now().Year() - m.DateOfBirth.Year()
		if age > oldestAge {
			oldestAge = age
		}
	}
	return oldestAge
}

func (s *FamilyOfficeService) calculateLiquidPct(assets []FamilyAsset) decimal.Decimal {
	var total, liquid decimal.Decimal
	for _, a := range assets {
		total = total.Add(a.CurrentValuation)
		if !a.Illiquid {
			liquid = liquid.Add(a.CurrentValuation)
		}
	}
	if total.IsZero() {
		return decimal.Zero
	}
	return liquid.Div(total).Mul(decimal.NewFromInt(100))
}

func (s *FamilyOfficeService) calculateBusinessPct(assets []FamilyAsset) decimal.Decimal {
	var total, business decimal.Decimal
	for _, a := range assets {
		total = total.Add(a.CurrentValuation)
		if a.AssetClass == "BUSINESS_INTEREST" {
			business = business.Add(a.CurrentValuation)
		}
	}
	if total.IsZero() {
		return decimal.Zero
	}
	return business.Div(total).Mul(decimal.NewFromInt(100))
}

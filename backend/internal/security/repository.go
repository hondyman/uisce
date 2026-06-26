package security

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

// AccessRuleRepository handles database operations for access rules.
type AccessRuleRepository struct {
	db *sqlx.DB
}

// NewAccessRuleRepository creates a new repository.
func NewAccessRuleRepository(db *sqlx.DB) *AccessRuleRepository {
	return &AccessRuleRepository{db: db}
}

// List retrieves access rules with optional filters.
func (r *AccessRuleRepository) List(ctx context.Context, tenantID, businessObjectID, status string) ([]*models.AccessRule, error) {
	query := `
		SELECT 
			id, tenant_id, business_object_id, group_dn, 
			access_level, status, row_filter_dsl, column_masks,
			applies_to_apis, applies_to_bi, applies_to_ai,
			created_by, created_at, updated_by, updated_at, version, description
		FROM access_rule
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 1

	if tenantID != "" {
		query += fmt.Sprintf(" AND tenant_id = $%d", argCount)
		args = append(args, tenantID)
		argCount++
	}
	if businessObjectID != "" {
		query += fmt.Sprintf(" AND business_object_id = $%d", argCount)
		args = append(args, businessObjectID)
		argCount++
	}
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list access rules: %w", err)
	}
	defer rows.Close()

	var rules []*models.AccessRule
	for rows.Next() {
		rule, err := scanAccessRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

// Get retrieves a single access rule by ID.
func (r *AccessRuleRepository) Get(ctx context.Context, ruleID string) (*models.AccessRule, error) {
	query := `
		SELECT 
			id, tenant_id, business_object_id, group_dn, 
			access_level, status, row_filter_dsl, column_masks,
			applies_to_apis, applies_to_bi, applies_to_ai,
			created_by, created_at, updated_by, updated_at, version, description
		FROM access_rule
		WHERE id = $1
	`

	row := r.db.QueryRowxContext(ctx, query, ruleID)
	return scanAccessRule(row)
}

// Create inserts a new access rule.
func (r *AccessRuleRepository) Create(ctx context.Context, rule *models.AccessRule) (*models.AccessRule, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	created, err := r.CreateTx(ctx, tx, rule)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return created, nil
}

// CreateTx inserts a new access rule within a transaction.
func (r *AccessRuleRepository) CreateTx(ctx context.Context, tx *sqlx.Tx, rule *models.AccessRule) (*models.AccessRule, error) {
	if rule.RuleID == "" {
		rule.RuleID = uuid.New().String()
	}
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	rule.Version = 1

	columnMasksJSON, err := json.Marshal(rule.ColumnMasks)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal column masks: %w", err)
	}

	query := `
		INSERT INTO access_rule (
			id, tenant_id, business_object_id, group_dn,
			access_level, status, row_filter_dsl, column_masks,
			applies_to_apis, applies_to_bi, applies_to_ai,
			created_by, created_at, updated_by, updated_at, version, description
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id
	`

	err = tx.QueryRowContext(
		ctx, query,
		rule.RuleID, rule.TenantID, rule.BusinessObjectID, rule.GroupDn,
		rule.AccessLevel, rule.Status, rule.RowFilterDsl, columnMasksJSON,
		rule.AppliesToApis, rule.AppliesToBi, rule.AppliesToAi,
		rule.CreatedBy, rule.CreatedAt, rule.UpdatedBy, rule.UpdatedAt, rule.Version, rule.Description,
	).Scan(&rule.RuleID)

	if err != nil {
		return nil, fmt.Errorf("failed to create access rule: %w", err)
	}

	return rule, nil
}

// Update modifies an existing access rule.
func (r *AccessRuleRepository) Update(ctx context.Context, ruleID string, rule *models.AccessRule) (*models.AccessRule, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	updated, err := r.UpdateTx(ctx, tx, ruleID, rule)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return updated, nil
}

// UpdateTx modifies an existing access rule within a transaction.
func (r *AccessRuleRepository) UpdateTx(ctx context.Context, tx *sqlx.Tx, ruleID string, rule *models.AccessRule) (*models.AccessRule, error) {
	rule.UpdatedAt = time.Now()
	rule.Version++

	columnMasksJSON, err := json.Marshal(rule.ColumnMasks)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal column masks: %w", err)
	}

	query := `
		UPDATE access_rule SET
			business_object_id = $2,
			group_dn = $3,
			access_level = $4,
			status = $5,
			row_filter_dsl = $6,
			column_masks = $7,
			applies_to_apis = $8,
			applies_to_bi = $9,
			applies_to_ai = $10,
			updated_by = $11,
			updated_at = $12,
			version = $13,
			description = $14
		WHERE id = $1
		RETURNING id
	`

	err = tx.QueryRowContext(
		ctx, query,
		ruleID,
		rule.BusinessObjectID, rule.GroupDn, rule.AccessLevel, rule.Status,
		rule.RowFilterDsl, columnMasksJSON,
		rule.AppliesToApis, rule.AppliesToBi, rule.AppliesToAi,
		rule.UpdatedBy, rule.UpdatedAt, rule.Version, rule.Description,
	).Scan(&rule.RuleID)

	if err != nil {
		return nil, fmt.Errorf("failed to update access rule: %w", err)
	}

	return rule, nil
}

// GetByBusinessObjectAndGroups retrieves all rules for a BO and set of groups.
func (r *AccessRuleRepository) GetByBusinessObjectAndGroups(ctx context.Context, businessObjectID string, groups []string) ([]*models.AccessRule, error) {
	query := `
		SELECT 
			id, tenant_id, business_object_id, group_dn, 
			access_level, status, row_filter_dsl, column_masks,
			applies_to_apis, applies_to_bi, applies_to_ai,
			created_by, created_at, updated_by, updated_at, version, description
		FROM access_rule
		WHERE business_object_id = $1 
		  AND group_dn = ANY($2)
		  AND status = 'APPROVED'
	`

	rows, err := r.db.QueryxContext(ctx, query, businessObjectID, groups)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules by BO and groups: %w", err)
	}
	defer rows.Close()

	var rules []*models.AccessRule
	for rows.Next() {
		rule, err := scanAccessRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

// scanAccessRule is a helper to scan a row into an AccessRule struct.
func scanAccessRule(row interface{ Scan(...interface{}) error }) (*models.AccessRule, error) {
	var rule models.AccessRule
	var columnMasksJSON []byte

	err := row.Scan(
		&rule.RuleID, &rule.TenantID, &rule.BusinessObjectID, &rule.GroupDn,
		&rule.AccessLevel, &rule.Status, &rule.RowFilterDsl, &columnMasksJSON,
		&rule.AppliesToApis, &rule.AppliesToBi, &rule.AppliesToAi,
		&rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedBy, &rule.UpdatedAt, &rule.Version, &rule.Description,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("access rule not found")
		}
		return nil, fmt.Errorf("failed to scan access rule: %w", err)
	}

	if columnMasksJSON != nil {
		if err := json.Unmarshal(columnMasksJSON, &rule.ColumnMasks); err != nil {
			return nil, fmt.Errorf("failed to unmarshal column masks: %w", err)
		}
	}

	return &rule, nil
}

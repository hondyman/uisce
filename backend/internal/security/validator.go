package security

import (
	"context"
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/security/dsl"
	"github.com/jmoiron/sqlx"
)

// DslValidator validates row filter DSL expressions against catalog schema.
type DslValidator struct {
	db     *sqlx.DB
	parser *dsl.Parser
}

// NewDslValidator creates a new validator.
func NewDslValidator(db *sqlx.DB) *DslValidator {
	return &DslValidator{
		db:     db,
		parser: dsl.NewParser(),
	}
}

// Validate parses and validates a DSL expression for a given business object.
func (v *DslValidator) Validate(ctx context.Context, businessObjectID, dslExpr string) (*DslValidationResult, error) {
	// Parse the DSL
	ast, err := v.parser.Parse(dslExpr)
	if err != nil {
		errMsg := fmt.Sprintf("Syntax error: %s", err.Error())
		return &DslValidationResult{
			Valid:        false,
			ErrorMessage: &errMsg,
		}, nil
	}

	// Fetch allowed fields from catalog for this BO
	allowedFields, err := v.getAllowedFields(ctx, businessObjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch allowed fields: %w", err)
	}

	// Validate referenced fields exist
	referencedFields := ast.GetReferencedFields()
	for _, field := range referencedFields {
		if !contains(allowedFields, field) {
			errMsg := fmt.Sprintf("Field not allowed: '%s'. Allowed fields: %s", field, strings.Join(allowedFields, ", "))
			return &DslValidationResult{
				Valid:        false,
				ErrorMessage: &errMsg,
			}, nil
		}
	}

	// Generate SQL predicate
	sqlPredicate := ast.ToSQL()

	// logging.Info("DSL validation successful", "businessObjectId", businessObjectID, "dsl", dslExpr, "sql", sqlPredicate)

	return &DslValidationResult{
		Valid:        true,
		SqlPredicate: &sqlPredicate,
	}, nil
}

// getAllowedFields retrieves the list of fields/terms for a business object from the catalog.
func (v *DslValidator) getAllowedFields(ctx context.Context, businessObjectID string) ([]string, error) {
	query := `
		SELECT DISTINCT st.term_name
		FROM semantic_term st
		JOIN business_object_term bot ON st.id = bot.term_id
		WHERE bot.business_object_id = $1
	`

	rows, err := v.db.QueryContext(ctx, query, businessObjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fields []string
	for rows.Next() {
		var field string
		if err := rows.Scan(&field); err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}

	return fields, rows.Err()
}

// contains checks if a slice contains a string (case-insensitive).
func contains(slice []string, item string) bool {
	itemLower := strings.ToLower(item)
	for _, s := range slice {
		if strings.ToLower(s) == itemLower {
			return true
		}
	}
	return false
}

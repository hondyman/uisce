package boresolver

import (
	"fmt"
	"regexp"
	"strings"
)

// sqlIdentRE allows schema.table or bare identifiers (letters, digits, underscore).
// Rejects injection-shaped strings (quotes, semicolons, comments, spaces).
var sqlIdentRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$`)

// ValidateSQLIdentifier rejects identifiers that must never reach fmt.Sprintf SQL.
// Under the MCP invariant, contract-registry names should be the only callers;
// this is the last-line fence for transitional string compilation.
func ValidateSQLIdentifier(kind, value string) error {
	v := strings.TrimSpace(value)
	if v == "" {
		return fmt.Errorf("invalid %s: empty", kind)
	}
	if !sqlIdentRE.MatchString(v) {
		return fmt.Errorf("invalid %s %q: not an allowlisted SQL identifier", kind, value)
	}
	return nil
}

func validateBitemporalIdentifiers(req BitemporalRangeRequest) error {
	if err := ValidateSQLIdentifier("hot_table_name", req.HotTableName); err != nil {
		return err
	}
	if err := ValidateSQLIdentifier("cold_table_name", req.ColdTableName); err != nil {
		return err
	}
	if req.TemporalColumn != "" {
		if err := ValidateSQLIdentifier("temporal_column", req.TemporalColumn); err != nil {
			return err
		}
	}
	for _, c := range req.BusinessKeyColumns {
		if err := ValidateSQLIdentifier("business_key_column", c); err != nil {
			return err
		}
	}
	for _, c := range req.SelectedColumns {
		if err := ValidateSQLIdentifier("selected_column", c); err != nil {
			return err
		}
	}
	return nil
}

package boresolver

import (
	"fmt"
	"regexp"
	"strings"
)

// sqlIdentRE accepts one or more dot-separated identifier parts, each
// `[A-Za-z_][A-Za-z0-9_]*` (identifier-shaped: leading letter or
// underscore, then letters/digits/underscores). Per-part validation
// preserves the anti-injection guarantee (no quotes, semicolons,
// comments, whitespace) — the part count has no upper bound because
// every part is independently identifier-shaped, so a 50-part
// qualified name is harmless.
//
// Production convention is `<engine>.<tenant-shorthand>.<table>` (3
// parts, e.g. `starrocks.t_99e99e99.portfolio_positions_realtime`) or
// `<catalog>.<schema>.<table>` (3 parts, e.g. Iceberg), in addition
// to 2-part `schema.table` (e.g. `oms.orm_execution`) and 1-part bare
// identifiers. The previous `?` quantifier on the dot-group capped
// accepted input at 2 parts and rejected the 3-part production
// bitemporal fixtures, breaking TestCompileRangeQuery_PureCold /
// _PureHot / _SplitAndStitchWithDeduplication with "not an allowlisted
// SQL identifier." This validator was introduced by 37edaf7ee
// (PR #82, `fix(glossary): semantic term generation`); the relaxation
// is conservative: per-part validation is unchanged, only the
// part-count cap is relaxed.
var sqlIdentRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$`)

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

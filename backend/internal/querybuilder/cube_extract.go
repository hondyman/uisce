package querybuilder

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
)

// federationExtractEnabled gates Track C Extract-N (CUBE federated staging).
// Off by default so hermetic tests and the proven JDBC CUBE-2.5 path stay stable.
func federationExtractEnabled() bool {
	v := strings.TrimSpace(os.Getenv("CUBE_FEDERATION_EXTRACT"))
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// CubeExtractStagingTableName builds an attempt-scoped StarRocks staging table
// bare name: cube_ext_{grain8}_{attempt8}_{alias}.
func CubeExtractStagingTableName(grainHash, attemptID, alias string) string {
	gh := sanitizeIdentifier(grainHash)
	if len(gh) > 8 {
		gh = gh[:8]
	}
	att := sanitizeIdentifier(attemptID)
	att = strings.ReplaceAll(att, "-", "")
	if len(att) > 8 {
		att = att[:8]
	}
	al := sanitizeIdentifier(strings.ToLower(strings.TrimSpace(alias)))
	if gh == "" {
		gh = "grain"
	}
	if att == "" {
		att = "attempt"
	}
	if al == "" {
		al = "src"
	}
	return fmt.Sprintf("cube_ext_%s_%s_%s", gh, att, al)
}

// QualifyCubeExtractStagingTable returns db.`table` for StarRocks DDL.
func QualifyCubeExtractStagingTable(targetDB, bareName string) string {
	return fmt.Sprintf("%s.%s",
		quoteStarRocksIdent(sanitizeIdentifier(targetDB)),
		quoteStarRocksIdent(sanitizeIdentifier(bareName)),
	)
}

// RewriteFederationFromSQLForStaging replaces each source DrivingTable with its
// staging qualified name while preserving aliases and ON predicates.
func RewriteFederationFromSQLForStaging(fromSQL string, sources []FederationSourcePlan, stagingByAlias map[string]string) (string, error) {
	fromSQL = strings.TrimSpace(fromSQL)
	if fromSQL == "" {
		return "", fmt.Errorf("cube extract: empty federation FROM SQL")
	}
	if len(sources) == 0 {
		return "", fmt.Errorf("cube extract: no federation sources")
	}
	out := fromSQL
	// Longest driving tables first so shorter prefixes cannot clobber longer names.
	ordered := append([]FederationSourcePlan(nil), sources...)
	sort.Slice(ordered, func(i, j int) bool {
		return len(ordered[i].DrivingTable) > len(ordered[j].DrivingTable)
	})
	for _, src := range ordered {
		alias := strings.ToLower(strings.TrimSpace(src.Alias))
		staging, ok := stagingByAlias[alias]
		if !ok || strings.TrimSpace(staging) == "" {
			return "", fmt.Errorf("cube extract: missing staging table for alias %q", alias)
		}
		drive := strings.TrimSpace(src.DrivingTable)
		if drive == "" {
			return "", fmt.Errorf("cube extract: empty driving table for alias %q", alias)
		}
		// Patterns emitted by CompileFederationJoinSQL.
		oldLead := fmt.Sprintf("%s AS %s", drive, alias)
		newLead := fmt.Sprintf("%s AS %s", staging, alias)
		oldJoin := fmt.Sprintf("INNER JOIN %s AS %s", drive, alias)
		newJoin := fmt.Sprintf("INNER JOIN %s AS %s", staging, alias)
		if !strings.Contains(out, oldLead) && !strings.Contains(out, oldJoin) {
			return "", fmt.Errorf("cube extract: driving table %q for alias %q not found in FROM SQL", drive, alias)
		}
		out = strings.ReplaceAll(out, oldLead, newLead)
		out = strings.ReplaceAll(out, oldJoin, newJoin)
	}
	return out, nil
}

// RewriteCubeDDLFromClause swaps the FROM body inside cube MV DDL.
func RewriteCubeDDLFromClause(ddl, oldFrom, newFrom string) (string, error) {
	ddl = strings.TrimSpace(ddl)
	oldFrom = strings.TrimSpace(oldFrom)
	newFrom = strings.TrimSpace(newFrom)
	if ddl == "" || oldFrom == "" || newFrom == "" {
		return "", fmt.Errorf("cube extract: ddl/from rewrite requires non-empty inputs")
	}
	needle := "FROM " + oldFrom
	if !strings.Contains(ddl, needle) {
		// DDL pretty-prints FROM on its own line; tolerate newline after FROM.
		alt := "FROM\n" + oldFrom
		if strings.Contains(ddl, alt) {
			return strings.Replace(ddl, alt, "FROM\n"+newFrom, 1), nil
		}
		return "", fmt.Errorf("cube extract: FROM clause not found in DDL for rewrite")
	}
	return strings.Replace(ddl, needle, "FROM "+newFrom, 1), nil
}

// CubeExtractResult is returned by ExtractSources for Temporal + cleanup.
type CubeExtractResult struct {
	Plan          *CubeMaterializePlan `json:"plan"`
	StagingTables []string             `json:"staging_tables"`
	RowCounts     map[string]int64     `json:"row_counts,omitempty"`
}

// ExtractSources CTAS each federation driving table into attempt-scoped
// StarRocks staging, then rewrites plan.SourceTable + plan.DDL onto staging.
func (m *CubeMaterializer) ExtractSources(ctx context.Context, plan *CubeMaterializePlan) (*CubeExtractResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("cube extract: plan is required")
	}
	if !plan.ExtractEnabled {
		return nil, fmt.Errorf("cube extract: ExtractEnabled is false")
	}
	if len(plan.FederationSources) == 0 {
		return nil, fmt.Errorf("cube extract: no federation sources on plan")
	}
	if m == nil || m.starrocksDB == nil {
		return nil, fmt.Errorf("cube extract: starrocks connection is not available")
	}

	if _, err := m.starrocksDB.ExecContext(ctx,
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", quoteStarRocksIdent(plan.TargetDatabase)),
	); err != nil {
		return nil, fmt.Errorf("cube extract: ensure database %q: %w", plan.TargetDatabase, err)
	}

	stagingByAlias := make(map[string]string, len(plan.FederationSources))
	stagingList := make([]string, 0, len(plan.FederationSources))
	rowCounts := make(map[string]int64, len(plan.FederationSources))

	cleanup := func() {
		_ = m.DropStagingTables(ctx, stagingList)
	}

	for _, src := range plan.FederationSources {
		alias := strings.ToLower(strings.TrimSpace(src.Alias))
		bare := CubeExtractStagingTableName(plan.GrainHash, plan.AttemptID, alias)
		qualified := QualifyCubeExtractStagingTable(plan.TargetDatabase, bare)
		drive := strings.TrimSpace(src.DrivingTable)
		if drive == "" {
			cleanup()
			return nil, fmt.Errorf("cube extract: empty driving table for alias %q", alias)
		}
		if err := assertSafeSQLIdent(drive); err != nil {
			cleanup()
			return nil, fmt.Errorf("cube extract: driving table %q: %w", drive, err)
		}

		dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", qualified)
		if _, err := m.starrocksDB.ExecContext(ctx, dropSQL); err != nil {
			cleanup()
			return nil, fmt.Errorf("cube extract: drop staging %s: %w", qualified, err)
		}
		ctas := fmt.Sprintf("CREATE TABLE %s AS SELECT * FROM %s", qualified, drive)
		if _, err := m.starrocksDB.ExecContext(ctx, ctas); err != nil {
			cleanup()
			return nil, fmt.Errorf("cube extract: CTAS %s from %s: %w", qualified, drive, err)
		}
		stagingByAlias[alias] = qualified
		stagingList = append(stagingList, qualified)

		var n int64
		countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s", qualified)
		if err := m.starrocksDB.QueryRowContext(ctx, countSQL).Scan(&n); err == nil {
			rowCounts[alias] = n
		}
	}

	newFrom, err := RewriteFederationFromSQLForStaging(plan.SourceTable, plan.FederationSources, stagingByAlias)
	if err != nil {
		cleanup()
		return nil, err
	}
	newDDL, err := RewriteCubeDDLFromClause(plan.DDL, plan.SourceTable, newFrom)
	if err != nil {
		cleanup()
		return nil, err
	}

	outPlan := *plan
	outPlan.SourceTable = newFrom
	outPlan.DDL = newDDL
	outPlan.StagingTables = append([]string(nil), stagingList...)
	outPlan.ExtractApplied = true

	return &CubeExtractResult{
		Plan:          &outPlan,
		StagingTables: outPlan.StagingTables,
		RowCounts:     rowCounts,
	}, nil
}

// DropStagingTables drops attempt-scoped extract tables (best-effort per table).
func (m *CubeMaterializer) DropStagingTables(ctx context.Context, tables []string) error {
	if m == nil || m.starrocksDB == nil || len(tables) == 0 {
		return nil
	}
	var first error
	for _, t := range tables {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, err := m.starrocksDB.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", t)); err != nil && first == nil {
			first = fmt.Errorf("cube extract: drop staging %s: %w", t, err)
		}
	}
	return first
}

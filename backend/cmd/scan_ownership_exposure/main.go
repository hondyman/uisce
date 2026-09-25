// scan_ownership_exposure assembles analytics.ScannedColumn rows from real
// saved queries (data_explorer.saved_query) and runs them through
// analytics.ScanForSharedOwnership, so someone with database access can
// answer the fan-out cardinality trace's exposure question - "is shared
// root ownership a latent hazard, or is it live in a real saved query
// right now" - without needing to design the assembly boundary
// themselves. It calls the exact same ScanForSharedOwnership the ownership
// scanner tests exercise; it does not reimplement anything.
//
// Usage:
//
//	DATABASE_URL=postgres://... go run ./cmd/scan_ownership_exposure [-json]
//
// Exit code is 0 whether or not findings are found - this is a report,
// not a gate. Pipe -json output into `jq` or a file for further triage.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/querybuilder"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	jsonOut := flag.Bool("json", false, "emit findings as JSON instead of a text table")
	flag.Parse()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	boRepo := boresolver.NewPostgresBORepository(db)
	relationships := analytics.NewRelationshipInferenceService(db)

	columns, unresolvedCount, err := assembleScannedColumns(db, boRepo, relationships)
	if err != nil {
		log.Fatalf("assembling scanned columns: %v", err)
	}

	findings := analytics.ScanForSharedOwnership(columns)

	if *jsonOut {
		if err := json.NewEncoder(os.Stdout).Encode(findings); err != nil {
			log.Fatalf("encoding findings: %v", err)
		}
		return
	}

	fmt.Printf("Scanned %d columns across saved queries (%d could not be resolved to a catalog relationship - see \"unresolved\" rows below, not skipped).\n\n", len(columns), unresolvedCount)
	if len(findings) == 0 {
		fmt.Println("No ownership exposure found.")
		return
	}

	fmt.Printf("%-36s  %-24s  %-10s  %-6s  %-10s  %s\n", "REPORT ID", "COLUMN", "OWNERSHIP", "ATRISK", "CARDINALITY", "MEASURE")
	for _, f := range findings {
		fmt.Printf("%-36s  %-24s  %-10s  %-6v  %-10s  %s\n", f.ReportID, f.ColumnName, f.Ownership, f.AtRisk, f.Cardinality, f.Measure)
	}

	atRisk := 0
	for _, f := range findings {
		if f.AtRisk {
			atRisk++
		}
	}
	fmt.Printf("\n%d finding(s), %d at risk (ownership hazard combined with a measure that actually produces a wrong number).\n", len(findings), atRisk)
}

// assembleScannedColumns reads every saved query's dimensions and measures,
// resolves each related-BO column's join path via the SAME relationship
// resolver the query builder itself uses (never re-deriving cardinality by
// hand), and returns one analytics.ScannedColumn per related-BO column.
//
// Per the scanner's own contract: a column whose path can't be resolved to
// a catalog relationship at all is NOT skipped. It's represented with a
// JoinPath carrying one step of empty Cardinality, so
// JoinPath.RootOwnership() classifies it "unresolved" (never silently
// "unique") and it flows through ScanForSharedOwnership exactly like any
// other column - skipping it would under-report exposure precisely where
// visibility is already weakest.
func assembleScannedColumns(db *sqlx.DB, boRepo *boresolver.PostgresBORepository, relationships *analytics.RelationshipInferenceService) ([]analytics.ScannedColumn, int, error) {
	type row struct {
		ID         string `db:"id"`
		Name       string `db:"name"`
		SourceID   string `db:"source_id"`
		QueryState []byte `db:"query_state"`
	}
	var rows []row
	if err := db.Select(&rows, `SELECT id, name, source_id, query_state FROM data_explorer.saved_query`); err != nil {
		return nil, 0, fmt.Errorf("querying data_explorer.saved_query: %w", err)
	}

	var columns []analytics.ScannedColumn
	unresolvedCount := 0

	for _, r := range rows {
		var state querybuilder.SavedQueryState
		if err := json.Unmarshal(r.QueryState, &state); err != nil {
			// A saved query whose state can't even be parsed is its own
			// finding, not something to silently drop - same "don't skip"
			// principle as an unresolvable join path.
			columns = append(columns, analytics.ScannedColumn{
				ReportID:   reportLabel(r.ID, r.Name),
				ColumnName: "<unparseable query_state>",
				Path:       unresolvedPath(),
			})
			unresolvedCount++
			continue
		}

		primaryBO, err := boRepo.GetBODefinition(r.SourceID)
		if err != nil || primaryBO == nil {
			columns = append(columns, analytics.ScannedColumn{
				ReportID:   reportLabel(r.ID, r.Name),
				ColumnName: "<primary BO unresolvable: " + r.SourceID + ">",
				Path:       unresolvedPath(),
			})
			unresolvedCount++
			continue
		}

		datasourceUUID, err := uuid.Parse(primaryBO.DatasourceID)
		if err != nil {
			columns = append(columns, analytics.ScannedColumn{
				ReportID:   reportLabel(r.ID, r.Name),
				ColumnName: "<invalid datasource id on primary BO>",
				Path:       unresolvedPath(),
			})
			unresolvedCount++
			continue
		}
		fromBOUUID, err := uuid.Parse(r.SourceID)
		if err != nil {
			columns = append(columns, analytics.ScannedColumn{
				ReportID:   reportLabel(r.ID, r.Name),
				ColumnName: "<invalid primary BO id>",
				Path:       unresolvedPath(),
			})
			unresolvedCount++
			continue
		}

		resolvePath := func(toBOID string) *analytics.JoinPath {
			toBOUUID, err := uuid.Parse(toBOID)
			if err != nil {
				unresolvedCount++
				return unresolvedPath()
			}
			path, err := relationships.ResolveJoinPathBetweenBOs(context.Background(), datasourceUUID, fromBOUUID, toBOUUID)
			if err != nil || path == nil {
				unresolvedCount++
				return unresolvedPath()
			}
			return path
		}

		for _, d := range state.Dimensions {
			if d.BOID == "" || d.BOID == r.SourceID {
				continue // primary BO's own column - trivially unique, not a related-BO join
			}
			columns = append(columns, analytics.ScannedColumn{
				ReportID:   reportLabel(r.ID, r.Name),
				ColumnName: columnLabel(d.Alias, d.TermNodeID),
				Path:       resolvePath(d.BOID),
			})
		}
		for _, m := range state.Measures {
			if m.BOID == "" || m.BOID == r.SourceID {
				continue
			}
			columns = append(columns, analytics.ScannedColumn{
				ReportID:    reportLabel(r.ID, r.Name),
				ColumnName:  columnLabel(m.Alias, m.TermNodeID),
				Aggregation: m.Aggregation,
				Path:        resolvePath(m.BOID),
			})
		}
	}

	return columns, unresolvedCount, nil
}

func reportLabel(id, name string) string {
	if name != "" {
		return name + " (" + id + ")"
	}
	return id
}

func columnLabel(alias, termNodeID string) string {
	if alias != "" {
		return alias
	}
	return termNodeID
}

// unresolvedPath is a JoinPath with one step of unrecognized Cardinality,
// so JoinPath.RootOwnership() classifies it "unresolved" - see that
// function's doc comment on why absence of information must never read
// as "unique".
func unresolvedPath() *analytics.JoinPath {
	return &analytics.JoinPath{Steps: []analytics.JoinPathStep{{Cardinality: ""}}}
}

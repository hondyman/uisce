// Live proof for Tier 2a of the calc-library build playbook (plain
// statistical aggregates): a measure using a newly-registered pushdownable
// function (STDEV_S) compiles through the real ResolveSemanticFieldMap /
// vm.CompileToSQL chain - the same chain cmd/verify_calc_measure proved for
// SUM - and produces a real computed value in a live StarRocks materialized
// view. One live proof per tier, per the playbook; the per-function
// correctness is library_tier2a_test.go's job, not this script's.
//
// Creates a real order -> placement -> execution chain in the platform-local
// orm schema (Postgres, alpha database - see docs/unified-rule-engine-handoff.md
// item 11) with several executions at varying exec_price under one
// placement, so STDEV_S(ExecPrice) has a real, non-degenerate sample to
// compute over.
//
// Does NOT wait for Debezium CDC to mirror these rows - checked this
// session (docs/orm-oms-connector.md) and confirmed the real
// orm-oms-connector publication is CREATE PUBLICATION ... FOR TABLES IN
// SCHEMA orm on the *crims* database, not alpha.orm (the platform-local
// schema item 11 created purely to unblock the validation-engine write
// path). alpha.orm rows will never reach StarRocks via that pipeline -
// this is the still-open "canonical OMS stratum" question (handoff item
// 10), not a bug introduced here. So this proof seeds the identical rows
// directly into StarRocks's oms.orm_execution hot-tier table instead -
// exactly the same "prove GenerateDDL's compiled SQL against real,
// resident StarRocks data" shape as verify_calc_measure's original proof,
// which itself ran against a single pre-existing StarRocks row with no
// live CDC insert that session either. The Postgres insert is kept
// anyway, both as the source-of-truth record and because a future session
// resolving item 10 may make it CDC-reachable.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const (
	tenantID       = "99e99e99-99e9-49e9-89e9-99e99e99e999"
	executionBOKey = "execution"
)

func main() {
	db, err := sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()

	// 1. Real order -> placement -> several executions, one placement,
	// varying exec_price (99.50, 100.25, 101.75, 98.90) so STDEV_S over
	// them is a real, non-degenerate number, not a single-point edge case.
	// Mirrored directly into StarRocks (see header comment: CDC doesn't
	// reach alpha.orm today).
	placementID, prices := seedExecutions(ctx, db)
	seedStarRocksMirror(ctx, placementID, prices)
	fmt.Printf("Seeded 1 order/placement + %d executions under PlacementID=%s, exec_price=%v (in both Postgres orm.execution and StarRocks oms.orm_execution directly)\n\n", len(prices), placementID, prices)

	// 2. Author the calculated term against the new STDEV_S function.
	termID := authorStdevTerm(ctx, db)
	fmt.Printf("Authored calculated term \"Exec Price StdDev\": catalog_node id=%s, config.rule_ast = STDEV_S(ExecPrice)\n", termID)

	// 3. Register the pre-aggregation.
	svc := analytics.NewPreAggregationService(db, nil, nil)
	_, err = svc.UpsertPreAggregation(ctx, models.UpsertPreAggRequest{
		TenantID:     tenantID,
		BOName:       executionBOKey,
		Name:         "execution_price_stdev_rollup",
		Description:  "STDEV_S(ExecPrice) per placement - Tier 2a live proof (docs/unified-rule-engine-handoff.md)",
		Terms:        []string{"PlacementID"},
		Calculations: []string{"Exec Price StdDev"},
		GroupBy:      []string{"PlacementID"},
		Materialization: models.MaterializationConfig{
			Type:       "materialized_view",
			TargetName: "mv_execution_price_stdev",
		},
		RefreshStrategy: "manual",
	})
	if err != nil {
		log.Fatalf("UpsertPreAggregation: %v", err)
	}
	// Same UpsertPreAggregation id-on-conflict workaround as
	// verify_calc_measure (docs/unified-rule-engine-handoff.md item 27) -
	// re-resolve the real id by node_name rather than trusting the
	// returned descriptor.
	var preAggID uuid.UUID
	if err := db.GetContext(ctx, &preAggID, `SELECT id FROM catalog_node WHERE node_name = $1 AND tenant_id = $2::uuid`, "execution_price_stdev_rollup", tenantID); err != nil {
		log.Fatalf("re-resolve pre-aggregation id: %v", err)
	}
	fmt.Printf("Registered pre-aggregation: id=%s\n\n", preAggID)

	// 4. Generate DDL - STDEV_S must compile to real SQL (STDDEV_SAMP).
	ddl, err := svc.GenerateDDL(ctx, preAggID, "starrocks")
	if err != nil {
		log.Fatalf("GenerateDDL: %v", err)
	}
	fmt.Println("=== Generated DDL ===")
	fmt.Println(ddl)
	if containsNullPlaceholder(ddl) {
		log.Fatalf("STDEV_S DID NOT COMPILE - Tier 2a's pushdown claim is not backed by this proof")
	}
	if !strings.Contains(ddl, "STDDEV_SAMP") {
		log.Fatalf("expected STDDEV_SAMP in generated DDL, got:\n%s", ddl)
	}
	fmt.Println("Confirmed: STDEV_S compiled to STDDEV_SAMP(exec_price), no NULL /* TODO */ placeholder.")

	// 5. Apply to real StarRocks.
	if err := svc.ApplyMaterialization(ctx, preAggID); err != nil {
		log.Fatalf("ApplyMaterialization (real StarRocks): %v", err)
	}
	fmt.Println("Applied to live StarRocks - materialized view created.")

	// 6. Query the real materialized view.
	srDB, err := sql.Open("mysql", starrocksDSN())
	if err != nil {
		log.Fatalf("connect to StarRocks: %v", err)
	}
	defer srDB.Close()

	targetDB := fmt.Sprintf("tenant_%s", tenantID)
	query := fmt.Sprintf("SELECT PlacementID, `Exec Price StdDev` FROM `%s`.mv_execution_price_stdev WHERE PlacementID = '%s'", targetDB, placementID)

	type row struct {
		PlacementID string
		Stdev       float64
	}
	var results []row
	// CDC hop already happened before ApplyMaterialization (step 1 waits
	// for it); this loop is just REFRESH ASYNC's own population lag, same
	// as verify_calc_measure.
	for attempt := 0; attempt < 15; attempt++ {
		results = nil
		rows, err := srDB.QueryContext(ctx, query)
		if err != nil {
			log.Fatalf("query materialized view: %v", err)
		}
		for rows.Next() {
			var pid sql.NullString
			var stdev sql.NullFloat64
			if err := rows.Scan(&pid, &stdev); err != nil {
				rows.Close()
				log.Fatalf("scan: %v", err)
			}
			results = append(results, row{pid.String, stdev.Float64})
		}
		rows.Close()
		if len(results) > 0 {
			break
		}
		time.Sleep(1500 * time.Millisecond)
	}

	fmt.Println("=== Real computed value from the live StarRocks hot tier ===")
	if len(results) == 0 {
		log.Fatalf("materialized view returned zero rows for PlacementID=%s - either CDC hasn't caught up or the DDL is silently wrong", placementID)
	}
	for _, r := range results {
		fmt.Printf("  PlacementID=%s Exec Price StdDev=%v\n", r.PlacementID, r.Stdev)
	}

	want := sampleStdev(prices)
	got := results[0].Stdev
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	if diff > 1e-3 {
		log.Fatalf("StarRocks STDDEV_SAMP(%v) = %v, independently computed sample stdev = %v (diff %v) - pushdown and native disagree", prices, got, want, diff)
	}
	fmt.Printf("\nAgrees with an independently computed sample standard deviation of the same %d prices (%v) to within 1e-3 - Tier 2a's pushdown claim is proven live, not just unit-tested.\n", len(prices), want)
}

// sampleStdev is an independent, from-scratch reimplementation (not a call
// into internal/rules/vm) - the whole point of this check is confirming
// StarRocks's STDDEV_SAMP agrees with a real second computation, not with
// itself.
func sampleStdev(vals []float64) float64 {
	n := float64(len(vals))
	mean := 0.0
	for _, v := range vals {
		mean += v
	}
	mean /= n
	sumSq := 0.0
	for _, v := range vals {
		d := v - mean
		sumSq += d * d
	}
	variance := sumSq / (n - 1)
	// sqrt via Newton's method - avoids importing math for one call in a
	// script this small.
	x := variance
	for i := 0; i < 50; i++ {
		x = 0.5 * (x + variance/x)
	}
	return x
}

func seedExecutions(ctx context.Context, db *sqlx.DB) (placementID string, prices []float64) {
	orderID := uuid.New().String()
	placementID = uuid.New().String()
	prices = []float64{99.50, 100.25, 101.75, 98.90}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO orm."order" (id, sec_id, side, order_type, status, target_qty, executed_qty, leaves_qty, trade_date)
		VALUES ($1, 1234, 'BUY', 'MARKET', 'FILLED', 20000, 20000, 0, CURRENT_DATE)
	`, orderID); err != nil {
		log.Fatalf("insert order: %v", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO orm.placement (id, order_id, broker_id, routed_qty, executed_qty, leaves_qty, status)
		VALUES ($1, $2, 'BRK1', 20000, 20000, 0, 'FILLED')
	`, placementID, orderID); err != nil {
		log.Fatalf("insert placement: %v", err)
	}
	now := time.Now().UTC()
	for i, p := range prices {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO orm.execution (id, placement_id, order_id, exec_qty, exec_price, exec_time, transact_time, status)
			VALUES ($1, $2, $3, 5000, $4, $5, $5, 'FILLED')
		`, uuid.New().String(), placementID, orderID, p, now.Add(time.Duration(i)*time.Second)); err != nil {
			log.Fatalf("insert execution %d: %v", i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Fatalf("commit: %v", err)
	}
	return placementID, prices
}

// seedStarRocksMirror inserts the identical rows directly into the
// StarRocks hot-tier table CDC would otherwise populate - see the header
// comment for why the real pipeline doesn't reach this session's data.
func seedStarRocksMirror(ctx context.Context, placementID string, prices []float64) {
	srDB, err := sql.Open("mysql", starrocksDSN())
	if err != nil {
		log.Fatalf("connect to StarRocks to seed hot-tier mirror: %v", err)
	}
	defer srDB.Close()

	orderID := uuid.New().String()
	now := time.Now().UTC()
	for i, p := range prices {
		// StarRocks's prepared-statement protocol rejects plain INSERT
		// (Error 1295) via this driver - build the literal statement
		// instead. Safe here: every value is either a freshly-generated
		// UUID or a float this program constructed, not external input.
		stmt := fmt.Sprintf(
			"INSERT INTO oms.orm_execution (id, placement_id, order_id, exec_qty, exec_price, exec_time, created_at) VALUES ('%s', '%s', '%s', 5000, %v, '%s', '%s')",
			uuid.New().String(), placementID, orderID, p,
			now.Add(time.Duration(i)*time.Second).Format(time.RFC3339), now.Format(time.RFC3339),
		)
		if _, err := srDB.ExecContext(ctx, stmt); err != nil {
			log.Fatalf("insert into StarRocks oms.orm_execution: %v", err)
		}
	}
}

func authorStdevTerm(ctx context.Context, db *sqlx.DB) uuid.UUID {
	ruleAST := json.RawMessage(`{
		"root": {
			"func": "STDEV_S",
			"args": [{"path": "ExecPrice"}]
		}
	}`)
	properties := map[string]interface{}{
		"tags":         []string{"Statistics", "Execution Quality"},
		"type":         "calculated",
		"term_type":    "calculated",
		"data_type":    "number",
		"return_type":  "decimal",
		"display_name": "Exec Price StdDev",
	}
	propsJSON, _ := json.Marshal(properties)
	config := map[string]interface{}{"rule_ast": json.RawMessage(ruleAST)}
	configJSON, _ := json.Marshal(config)

	var nodeTypeID string
	if err := db.GetContext(ctx, &nodeTypeID, `
		SELECT node_type_id FROM catalog_node WHERE node_name = 'Excel NPV' LIMIT 1
	`); err != nil {
		log.Fatalf("resolve semantic_term node_type_id: %v", err)
	}

	id := uuid.New()
	qualifiedPath := fmt.Sprintf("semantic_term/Statistics.exec_price_stdev_%s", id.String()[:8])
	if err := db.GetContext(ctx, &id, `
		INSERT INTO catalog_node (id, node_name, node_type_id, tenant_id, qualified_path, properties, config, created_at, updated_at)
		VALUES ($1, 'Exec Price StdDev', $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (tenant_id, qualified_path) DO UPDATE SET config = EXCLUDED.config, properties = EXCLUDED.properties, updated_at = NOW()
		RETURNING id
	`, id, nodeTypeID, tenantID, qualifiedPath, propsJSON, configJSON); err != nil {
		log.Fatalf("insert Exec Price StdDev catalog_node: %v", err)
	}
	return id
}

func containsNullPlaceholder(ddl string) bool {
	return strings.Contains(ddl, "NULL /* TODO")
}

func starrocksDSN() string {
	host := getEnv("STARROCKS_HOST", "100.84.50.65")
	port := getEnv("STARROCKS_PORT", "9030")
	user := getEnv("STARROCKS_USER", "root")
	password := getEnv("STARROCKS_PASSWORD", "")
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/?parseTime=true", user, password, host, port)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

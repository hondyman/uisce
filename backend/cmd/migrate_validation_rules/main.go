// migrate_validation_rules translates catalog_validation_rules.condition_json
// into internal/rules/vm.RuleNode-shaped rule_ast, for the subset of rows
// that are mechanically translatable, and produces a categorized report
// for every row - converted, needs-manual, or field-unresolvable.
//
// It never touches condition_json and never activates anything: rule_ast
// is written alongside the original column, and nothing in the running
// server currently reads rule_ast for these rows (see the Phase 0
// investigation - the evaluator that would read condition_json,
// internal/validation.TriggerValidationEngine, isn't wired into api.go
// either, so this table's rules have never been evaluated by anything).
// Wiring the vm engine to actually run these is a separate, later step
// that needs shadow mode first - this program only translates and reports.
//
// Usage:
//
//	go run . -dsn "$DATABASE_URL" [-dry-run] [-report report.json]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// ---- input row shape ------------------------------------------------------

type ruleRow struct {
	ID            string `db:"id"`
	RuleName      string `db:"rule_name"`
	RuleType      string `db:"rule_type"`
	TargetEntity  string `db:"target_entity"`
	ConditionJSON []byte `db:"condition_json"`
}

type conditionEnvelope struct {
	// SchemaVersion is stored as either a JSON string or a JSON number
	// across existing rows (the DB check constraint only requires it
	// text-represent digits, via ->>'schema_version' ~ '^[0-9]+$') -
	// json.RawMessage rather than string so a numeric encoding doesn't
	// fail the whole envelope's unmarshal. Not otherwise used here.
	SchemaVersion json.RawMessage `json:"schema_version"`
	AuthoredMode  string          `json:"authored_mode"`
	Payload       json.RawMessage `json:"payload"`
}

// ---- BO/field catalog for field-ref resolution -----------------------------

type boCatalog struct {
	// boByLowerName maps a lowercased bo_key or bo_name to its bo_id.
	boByLowerName map[string]string
	// fieldsByBO maps bo_id -> set of lowercased field_name.
	fieldsByBO map[string]map[string]bool
}

func loadBOCatalog(ctx context.Context, db *sqlx.DB) (*boCatalog, error) {
	cat := &boCatalog{
		boByLowerName: map[string]string{},
		fieldsByBO:    map[string]map[string]bool{},
	}

	type boRow struct {
		ID     string `db:"id"`
		BOKey  string `db:"bo_key"`
		BOName string `db:"bo_name"`
	}
	var bos []boRow
	if err := db.SelectContext(ctx, &bos, `SELECT id, bo_key, bo_name FROM business_objects`); err != nil {
		return nil, fmt.Errorf("loading business_objects: %w", err)
	}
	for _, bo := range bos {
		cat.boByLowerName[strings.ToLower(bo.BOKey)] = bo.ID
		cat.boByLowerName[strings.ToLower(bo.BOName)] = bo.ID
	}

	type fieldRow struct {
		BOID      string `db:"bo_id"`
		FieldName string `db:"field_name"`
	}
	var fields []fieldRow
	if err := db.SelectContext(ctx, &fields, `SELECT bo_id, field_name FROM business_object_fields`); err != nil {
		return nil, fmt.Errorf("loading business_object_fields: %w", err)
	}
	for _, f := range fields {
		if cat.fieldsByBO[f.BOID] == nil {
			cat.fieldsByBO[f.BOID] = map[string]bool{}
		}
		cat.fieldsByBO[f.BOID][strings.ToLower(f.FieldName)] = true
	}

	return cat, nil
}

// resolveBO finds a BO by case-insensitive match against bo_key/bo_name.
func (c *boCatalog) resolveBO(targetEntity string) (boID string, ok bool) {
	id, ok := c.boByLowerName[strings.ToLower(targetEntity)]
	return id, ok
}

// resolveField checks whether fieldName exists (case-insensitive) on boID.
func (c *boCatalog) resolveField(boID, fieldName string) bool {
	fields, ok := c.fieldsByBO[boID]
	if !ok {
		return false
	}
	return fields[strings.ToLower(fieldName)]
}

// ---- flat condition payload shape ------------------------------------------

type flatPayload struct {
	Field       string      `json:"field"`
	Operator    string      `json:"operator"`
	Value       interface{} `json:"value"`
	TargetField string      `json:"target_field"`
	MaxLength   *float64    `json:"max_length"`
}

// comparisonOps map DB operator names to (ConditionEvaluator long-form,
// BinaryExpr symbol). Both forms are needed: Condition nodes (single field
// vs literal) use the long-form name vm.ConditionEvaluator.compareValues
// accepts; BinaryExpr nodes (field vs field, for target_field rows) use
// the symbol form evalBinaryExpr accepts. These vocabularies are already
// slightly different in the existing codebase (e.g. "greater_equal" vs
// ">=") - this table is the single place that reconciles it for this
// migration rather than guessing per-callsite.
var comparisonOps = map[string]struct {
	conditionForm string
	symbolForm    string
	inRulefabric  bool
}{
	"equals":                 {"equals", "==", true},
	"not_equals":             {"not_equals", "!=", true},
	"greater_than":           {"greater_than", ">", true},
	"less_than":              {"less_than", "<", true},
	"greater_than_or_equal":  {"greater_equal", ">=", false}, // rulefabric spells it greater_than_or_equalS
	"less_than_or_equal":     {"less_equal", "<=", false},    // rulefabric spells it less_than_or_equalS
	"greater_than_or_equals": {"greater_equal", ">=", true},
	"less_than_or_equals":    {"less_equal", "<=", true},
}

// formatPredicates map DB operator names to the FuncCall predicate name
// added to internal/rules/vm/advanced_evaluator.go for this migration.
// None of these existed in rulefabric.OperatorRegistry before this
// migration - they're net-new to the unified function registry.
var formatPredicates = map[string]string{
	"not_empty":   "NOT_EMPTY",
	"is_integer":  "IS_INTEGER",
	"is_number":   "IS_NUMBER",
	"is_boolean":  "IS_BOOLEAN",
	"is_uuid":     "IS_UUID",
	"is_date":     "IS_DATE",
	"is_datetime": "IS_DATETIME",
	"is_json":     "IS_JSON",
	// max_length is handled separately below - it takes a second literal
	// arg (payload.max_length), not just the field.
}

// ---- report shapes ----------------------------------------------------------

type ruleReportEntry struct {
	ID           string `json:"id"`
	RuleName     string `json:"rule_name"`
	RuleType     string `json:"rule_type"` // field_format | business_logic | ...
	Category     string `json:"category"`  // type-validation | business-logic
	TargetEntity string `json:"target_entity"`
	Bucket       string `json:"bucket"` // converted | needs-manual | field-unresolvable
	Reason       string `json:"reason,omitempty"`
	Shape        string `json:"shape"` // flat-condition | code-expression | node-graph | script-dsl | unknown
	Operator     string `json:"operator,omitempty"`
	UsesTarget   bool   `json:"uses_target_field"`
	RuleAST      string `json:"rule_ast,omitempty"` // compact JSON, only set for converted rows
}

type report struct {
	Total             int               `json:"total"`
	Buckets           map[string]int    `json:"buckets"`
	Shapes            map[string]int    `json:"shapes"`
	Categories        map[string]int    `json:"categories"`
	ValueVsTargetSplit map[string]int   `json:"value_vs_target_field_split"`
	OperatorUsage     map[string]int    `json:"operator_usage"`
	NetNewOperators   []string          `json:"net_new_operators"`   // format predicates, no rulefabric precedent
	ExistingOperators []string          `json:"existing_operators"`  // comparison ops already in rulefabric.OperatorRegistry
	Entries           []ruleReportEntry `json:"entries"`
	DryRun            bool              `json:"dry_run"`
}

func classifyCategory(ruleType string) string {
	if ruleType == "field_format" {
		return "type-validation"
	}
	return "business-logic"
}

func main() {
	dsn := flag.String("dsn", os.Getenv("DATABASE_URL"), "Postgres connection string")
	dryRun := flag.Bool("dry-run", false, "don't write rule_ast, only produce the report")
	reportPath := flag.String("report", "validation_rules_migration_report.json", "path to write the JSON report")
	flag.Parse()

	if *dsn == "" {
		log.Fatal("dsn is required (flag or DATABASE_URL)")
	}

	db, err := sqlx.Connect("postgres", *dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	catalog, err := loadBOCatalog(ctx, db)
	if err != nil {
		log.Fatalf("loading BO catalog: %v", err)
	}

	var rows []ruleRow
	if err := db.SelectContext(ctx, &rows, `
		SELECT id, rule_name, rule_type, target_entity, condition_json
		FROM catalog_validation_rules
		ORDER BY rule_name
	`); err != nil {
		log.Fatalf("loading catalog_validation_rules: %v", err)
	}

	rep := report{
		Buckets:            map[string]int{},
		Shapes:             map[string]int{},
		Categories:         map[string]int{},
		ValueVsTargetSplit: map[string]int{},
		OperatorUsage:      map[string]int{},
		DryRun:             *dryRun,
	}
	netNewSet := map[string]bool{}
	existingSet := map[string]bool{}

	var toWrite []struct {
		id      string
		ruleAST string
	}

	for _, row := range rows {
		entry := ruleReportEntry{
			ID:           row.ID,
			RuleName:     row.RuleName,
			RuleType:     row.RuleType,
			Category:     classifyCategory(row.RuleType),
			TargetEntity: row.TargetEntity,
		}
		rep.Categories[entry.Category]++

		var env conditionEnvelope
		if err := json.Unmarshal(row.ConditionJSON, &env); err != nil {
			entry.Bucket = "needs-manual"
			entry.Shape = "unknown"
			entry.Reason = fmt.Sprintf("condition_json did not parse: %v", err)
			rep.Entries = append(rep.Entries, entry)
			rep.Buckets[entry.Bucket]++
			rep.Shapes[entry.Shape]++
			continue
		}

		var payloadMap map[string]json.RawMessage
		_ = json.Unmarshal(env.Payload, &payloadMap)

		switch {
		case has(payloadMap, "designerNodes"), has(payloadMap, "designerEdges"):
			entry.Shape = "node-graph"
			entry.Bucket = "needs-manual"
			entry.Reason = "visual node-graph payload, no mechanical translation"

		case has(payloadMap, "script_content"):
			entry.Shape = "script-dsl"
			entry.Bucket = "needs-manual"
			entry.Reason = "embedded script_content rule-block DSL, no mechanical translation"

		case has(payloadMap, "logic"):
			entry.Shape = "code-expression"
			entry.Bucket = "needs-manual"
			entry.Reason = "free-text expression (3 rows total) - hand-migrate, not worth a parser"

		case has(payloadMap, "field"):
			entry.Shape = "flat-condition"
			var fp flatPayload
			if err := json.Unmarshal(env.Payload, &fp); err != nil {
				entry.Bucket = "needs-manual"
				entry.Reason = fmt.Sprintf("flat payload did not parse: %v", err)
				break
			}
			entry.Operator = fp.Operator
			entry.UsesTarget = fp.TargetField != ""
			if entry.UsesTarget {
				rep.ValueVsTargetSplit["target_field"]++
			} else {
				rep.ValueVsTargetSplit["value"]++
			}
			rep.OperatorUsage[fp.Operator]++

			ast, reason := translateFlatCondition(catalog, row.TargetEntity, fp, netNewSet, existingSet)
			if ast != nil {
				entry.Bucket = "converted"
				b, _ := json.Marshal(ast)
				entry.RuleAST = string(b)
				toWrite = append(toWrite, struct {
					id      string
					ruleAST string
				}{row.ID, string(b)})
			} else {
				entry.Bucket = "field-unresolvable"
				entry.Reason = reason
			}

		default:
			entry.Shape = "unknown"
			entry.Bucket = "needs-manual"
			entry.Reason = "no recognized payload shape (no field/logic/script_content/designerNodes)"
		}

		rep.Buckets[entry.Bucket]++
		rep.Shapes[entry.Shape]++
		rep.Entries = append(rep.Entries, entry)
	}

	rep.Total = len(rows)
	// Classify every *observed* operator against the mapping tables,
	// independent of whether any row using it actually converted -
	// vocabulary classification (does rulefabric already have this
	// operator?) shouldn't depend on an unrelated failure (BO/field
	// resolution) further down the same row's translation.
	for op := range rep.OperatorUsage {
		if forms, ok := comparisonOps[op]; ok {
			if forms.inRulefabric {
				existingSet[op] = true
			} else {
				netNewSet[op] = true
			}
			continue
		}
		if fnName, ok := formatPredicates[strings.ToLower(op)]; ok {
			netNewSet[fnName] = true
			continue
		}
		if strings.ToLower(op) == "max_length" {
			netNewSet["MAX_LENGTH"] = true
		}
	}
	for op := range netNewSet {
		rep.NetNewOperators = append(rep.NetNewOperators, op)
	}
	for op := range existingSet {
		rep.ExistingOperators = append(rep.ExistingOperators, op)
	}
	sort.Strings(rep.NetNewOperators)
	sort.Strings(rep.ExistingOperators)
	sort.Slice(rep.Entries, func(i, j int) bool { return rep.Entries[i].RuleName < rep.Entries[j].RuleName })

	if !*dryRun {
		tx, err := db.BeginTxx(ctx, nil)
		if err != nil {
			log.Fatalf("begin tx: %v", err)
		}
		for _, w := range toWrite {
			if _, err := tx.ExecContext(ctx,
				`UPDATE catalog_validation_rules SET rule_ast = $1::jsonb WHERE id = $2::uuid`,
				w.ruleAST, w.id,
			); err != nil {
				tx.Rollback()
				log.Fatalf("writing rule_ast for %s: %v", w.id, err)
			}
		}
		if err := tx.Commit(); err != nil {
			log.Fatalf("commit: %v", err)
		}
		fmt.Printf("Wrote rule_ast for %d rows.\n", len(toWrite))
	} else {
		fmt.Printf("Dry run: would write rule_ast for %d rows.\n", len(toWrite))
	}

	f, err := os.Create(*reportPath)
	if err != nil {
		log.Fatalf("creating report file: %v", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rep); err != nil {
		log.Fatalf("writing report: %v", err)
	}

	fmt.Printf("\n=== Migration report (%s) ===\n", *reportPath)
	fmt.Printf("Total rules: %d\n", rep.Total)
	fmt.Printf("Buckets: %+v\n", rep.Buckets)
	fmt.Printf("Shapes: %+v\n", rep.Shapes)
	fmt.Printf("Categories: %+v\n", rep.Categories)
	fmt.Printf("value/target_field split: %+v\n", rep.ValueVsTargetSplit)
	fmt.Printf("Operator usage: %+v\n", rep.OperatorUsage)
	fmt.Printf("Net-new operators (added to vm this migration): %v\n", rep.NetNewOperators)
	fmt.Printf("Pre-existing operators (already in rulefabric.OperatorRegistry): %v\n", rep.ExistingOperators)
}

func has(m map[string]json.RawMessage, key string) bool {
	_, ok := m[key]
	return ok
}

// translateFlatCondition builds a vm.RuleNode for a single flat condition
// row, resolving field/target_field against the BO catalog first. Returns
// (nil, reason) if the BO or any referenced field can't be resolved -
// never fabricates a rule_ast against an unresolved field.
func translateFlatCondition(cat *boCatalog, targetEntity string, fp flatPayload, netNew, existing map[string]bool) (*vm.RuleNode, string) {
	boID, ok := cat.resolveBO(targetEntity)
	if !ok {
		return nil, fmt.Sprintf("target_entity %q does not match any business_objects.bo_key/bo_name", targetEntity)
	}
	if fp.Field == "" {
		return nil, "payload has no field"
	}
	if !cat.resolveField(boID, fp.Field) {
		return nil, fmt.Sprintf("field %q not found on resolved BO for target_entity %q", fp.Field, targetEntity)
	}
	if fp.TargetField != "" && !cat.resolveField(boID, fp.TargetField) {
		return nil, fmt.Sprintf("target_field %q not found on resolved BO for target_entity %q", fp.TargetField, targetEntity)
	}

	op := strings.ToLower(fp.Operator)

	// max_length: FuncCall(field, literal max)
	if op == "max_length" {
		if fp.MaxLength == nil {
			return nil, "max_length operator with no payload.max_length value"
		}
		netNew["MAX_LENGTH"] = true
		return &vm.RuleNode{
			Type: vm.NodeTypeExpression,
			Expression: &vm.Expression{
				Root: &vm.FuncCall{
					Name: "MAX_LENGTH",
					Args: []vm.ExprNode{
						&vm.FieldRef{Path: fp.Field},
						&vm.Literal{Value: *fp.MaxLength},
					},
				},
			},
		}, ""
	}

	// format predicates: FuncCall(field)
	if fnName, ok := formatPredicates[op]; ok {
		netNew[fnName] = true
		return &vm.RuleNode{
			Type: vm.NodeTypeExpression,
			Expression: &vm.Expression{
				Root: &vm.FuncCall{
					Name: fnName,
					Args: []vm.ExprNode{&vm.FieldRef{Path: fp.Field}},
				},
			},
		}, ""
	}

	// comparison ops
	if forms, ok := comparisonOps[op]; ok {
		if forms.inRulefabric {
			existing[op] = true
		} else {
			netNew[op] = true
		}

		if fp.TargetField != "" {
			// cross-field: BinaryExpr(field, target_field)
			return &vm.RuleNode{
				Type: vm.NodeTypeExpression,
				Expression: &vm.Expression{
					Root: &vm.BinaryExpr{
						Op:    forms.symbolForm,
						Left:  &vm.FieldRef{Path: fp.Field},
						Right: &vm.FieldRef{Path: fp.TargetField},
					},
				},
			}, ""
		}

		// field vs literal: Condition node
		return &vm.RuleNode{
			Type: vm.NodeTypeCondition,
			Condition: &vm.RuleCondition{
				Field:    fp.Field,
				Operator: forms.conditionForm,
				Value:    fp.Value,
			},
		}, ""
	}

	return nil, fmt.Sprintf("unrecognized operator %q", fp.Operator)
}

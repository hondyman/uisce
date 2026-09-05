package datapipeline

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/rules"
	"github.com/hondyman/uisce/backend/pkg/workflows"
)

// Transformer defines an operator step in the data pipeline
type Transformer interface {
	Transform(ctx context.Context, input []PipelineRecord) ([]PipelineRecord, []string, error)
}

// ColumnMapper handles field renaming and type casting
type ColumnMapper struct {
	Mappings map[string]string // TargetKey -> SourceKey
	Types    map[string]string // TargetKey -> "string" | "int" | "float" | "bool" | "date" | "uuid"
	Move     bool             // If true, delete source key after rename (move semantics). Default false (copy semantics).
}

func (m *ColumnMapper) Transform(ctx context.Context, input []PipelineRecord) ([]PipelineRecord, []string, error) {
	output := make([]PipelineRecord, 0, len(input))
	var errs []string

	for i, record := range input {
		transformed := make(PipelineRecord)
		// Default copy
		for k, v := range record {
			transformed[k] = v
		}

		// Apply field renames
		for targetKey, srcKey := range m.Mappings {
			if val, exists := record[srcKey]; exists {
				transformed[targetKey] = val
				if m.Move && targetKey != srcKey {
					delete(transformed, srcKey)
				}
			}
		}

		// Apply type casting
		for field, targetType := range m.Types {
			if rawVal, exists := transformed[field]; exists && rawVal != nil {
				casted, err := castValue(rawVal, targetType)
				if err != nil {
					errs = append(errs, fmt.Sprintf("row %d: field '%s' cast error: %v", i+1, field, err))
				} else {
					transformed[field] = casted
				}
			}
		}
		output = append(output, transformed)
	}
	return output, errs, nil
}

func castValue(val interface{}, targetType string) (interface{}, error) {
	str := fmt.Sprintf("%v", val)
	switch strings.ToLower(targetType) {
	case "string":
		return str, nil
	case "int", "integer", "int64":
		return strconv.ParseInt(strings.TrimSpace(str), 10, 64)
	case "float", "float64", "double", "number":
		return strconv.ParseFloat(strings.TrimSpace(str), 64)
	case "bool", "boolean":
		return strconv.ParseBool(strings.TrimSpace(str))
	case "uuid":
		u, err := uuid.Parse(strings.TrimSpace(str))
		if err != nil {
			return nil, err
		}
		return u.String(), nil
	case "date", "datetime", "timestamp":
		formats := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02", "01/02/2006"}
		for _, f := range formats {
			if t, err := time.Parse(f, strings.TrimSpace(str)); err == nil {
				return t.Format(time.RFC3339), nil
			}
		}
		return str, nil
	default:
		return val, nil
	}
}

// FilterTransformer filters records based on conditions
type FilterTransformer struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // "eq", "neq", "gt", "lt", "contains", "not_null"
	Value    interface{} `json:"value"`
}

func (f *FilterTransformer) Transform(ctx context.Context, input []PipelineRecord) ([]PipelineRecord, []string, error) {
	output := make([]PipelineRecord, 0, len(input))
	for _, record := range input {
		val := record[f.Field]
		if f.matches(val) {
			output = append(output, record)
		}
	}
	return output, nil, nil
}

func (f *FilterTransformer) matches(val interface{}) bool {
	if f.Operator == "not_null" {
		return val != nil && fmt.Sprintf("%v", val) != ""
	}
	if val == nil {
		return false
	}
	strVal := fmt.Sprintf("%v", val)
	targetStr := fmt.Sprintf("%v", f.Value)

	switch f.Operator {
	case "eq", "==":
		return strings.EqualFold(strVal, targetStr)
	case "neq", "!=":
		return !strings.EqualFold(strVal, targetStr)
	case "contains":
		return strings.Contains(strings.ToLower(strVal), strings.ToLower(targetStr))
	case "gt", ">":
		vNum, err1 := strconv.ParseFloat(strVal, 64)
		tNum, err2 := strconv.ParseFloat(targetStr, 64)
		if err1 == nil && err2 == nil {
			return vNum > tNum
		}
		return strVal > targetStr
	case "lt", "<":
		vNum, err1 := strconv.ParseFloat(strVal, 64)
		tNum, err2 := strconv.ParseFloat(targetStr, 64)
		if err1 == nil && err2 == nil {
			return vNum < tNum
		}
		return strVal < targetStr
	default:
		return true
	}
}

// AllowlistEnforcer ensures records for an STI model only contain columns registered for that subtype
type AllowlistEnforcer struct {
	Allowlists map[string][]string // SubtypeCode -> AllowedFields
}

func (a *AllowlistEnforcer) Transform(ctx context.Context, input []PipelineRecord) ([]PipelineRecord, []string, error) {
	output := make([]PipelineRecord, 0, len(input))
	var errs []string

	for i, record := range input {
		subtype, _ := record["subtype_code"].(string)
		allowedFields, exists := a.Allowlists[subtype]
		if !exists || len(allowedFields) == 0 {
			// If no allowlist registered, pass record through
			output = append(output, record)
			continue
		}

		allowedMap := make(map[string]bool)
		allowedMap["id"] = true
		allowedMap["tenant_id"] = true
		allowedMap["subtype_code"] = true
		allowedMap["created_at"] = true
		allowedMap["valid_from"] = true
		allowedMap["valid_to"] = true

		for _, f := range allowedFields {
			allowedMap[strings.ToLower(f)] = true
		}

		cleaned := make(PipelineRecord)
		for k, v := range record {
			kLower := strings.ToLower(k)
			if allowedMap[kLower] {
				cleaned[k] = v
			} else {
				errs = append(errs, fmt.Sprintf("row %d: stripped disallowed column '%s' for subtype '%s'", i+1, k, subtype))
			}
		}
		output = append(output, cleaned)
	}
	return output, errs, nil
}

// GraphSynthesizer converts tabular schema records into catalog graph nodes & relationship edges
type GraphSynthesizer struct {
	ParentPathField string // e.g. "table_name"
	ChildNameField  string // e.g. "column_name"
	DataTypeField   string // e.g. "data_type"
	EdgePredicate   string // e.g. "ATTRIBUTE_OF" or "COLUMN_OF"
}

func (g *GraphSynthesizer) Transform(ctx context.Context, input []PipelineRecord) ([]PipelineRecord, []string, error) {
	var output []PipelineRecord
	for _, r := range input {
		parent := fmt.Sprintf("%v", r[g.ParentPathField])
		child := fmt.Sprintf("%v", r[g.ChildNameField])
		dType := fmt.Sprintf("%v", r[g.DataTypeField])

		if parent == "" || child == "" {
			continue
		}

		// Synthesize parent TABLE node
		parentNode := PipelineRecord{
			"__graph_type":   "node",
			"node_name":      parent,
			"qualified_path": parent,
			"catalog_type":   "TABLE",
			"description":    fmt.Sprintf("Synthesized Table %s", parent),
		}

		// Synthesize child ATTRIBUTE node
		childNode := PipelineRecord{
			"__graph_type":   "node",
			"node_name":      child,
			"qualified_path": fmt.Sprintf("%s/%s", parent, child),
			"catalog_type":   "ATTRIBUTE",
			"description":    fmt.Sprintf("Synthesized Column %s (%s)", child, dType),
			"properties":     map[string]interface{}{"data_type": dType},
		}

		// Synthesize relationship edge
		pred := g.EdgePredicate
		if pred == "" {
			pred = "COLUMN_OF"
		}
		edge := PipelineRecord{
			"__graph_type":     "edge",
			"edge_type_name":   pred,
			"subject_path":     fmt.Sprintf("%s/%s", parent, child),
			"object_path":      parent,
			"description":      fmt.Sprintf("%s %s %s", child, pred, parent),
		}

		output = append(output, parentNode, childNode, edge)
	}
	return output, nil, nil
}

// WorkflowCallerTransformer triggers an existing Flow Builder / Temporal
// workflow (workflows.RunStoredWorkflow) for each record. TemporalClient
// must be set to actually dispatch; when nil it falls back to a mock
// dispatch (for tests / TestStep sandbox calls where no live Temporal
// connection is available), same as before this transformer was made real.
type WorkflowCallerTransformer struct {
	WorkflowID     string `json:"workflow_id"`
	WorkflowName   string `json:"workflow_name"`
	Mode           string `json:"mode"` // "sync", "async"
	PayloadKey     string `json:"payload_key"`
	TemporalClient client.Client
	TaskQueue      string
}

func (w *WorkflowCallerTransformer) Transform(ctx context.Context, input []PipelineRecord) ([]PipelineRecord, []string, error) {
	output := make([]PipelineRecord, 0, len(input))
	var errs []string

	taskQueue := w.TaskQueue
	if taskQueue == "" {
		taskQueue = workflows.DeployedBPTaskQueue
	}

	for i, record := range input {
		recCopy := make(PipelineRecord)
		for k, v := range record {
			recCopy[k] = v
		}

		payload := map[string]interface{}(recCopy)
		if w.PayloadKey != "" {
			if v, ok := recCopy[w.PayloadKey]; ok {
				if m, ok := v.(map[string]interface{}); ok {
					payload = m
				}
			}
		}

		if w.TemporalClient == nil {
			// No live Temporal connection configured (e.g. TestStep sandbox
			// evaluation): mock the dispatch rather than failing the record.
			wfRunID := fmt.Sprintf("wf-run-%s-%d", uuid.New().String()[:8], time.Now().UnixNano()%100000)
			recCopy["_workflow_run_id"] = wfRunID
			recCopy["_workflow_status"] = "dispatched_mock"
			recCopy["_workflow_name"] = w.WorkflowName
			output = append(output, recCopy)
			continue
		}

		opts := client.StartWorkflowOptions{
			ID:        fmt.Sprintf("pipeline-wfcall-%s-%d", uuid.New().String()[:8], time.Now().UnixNano()),
			TaskQueue: taskQueue,
		}
		run, err := w.TemporalClient.ExecuteWorkflow(ctx, opts, "RunStoredWorkflow", workflows.InterpreterInput{
			WorkflowID:  w.WorkflowID,
			InitialData: payload,
		})
		if err != nil {
			errs = append(errs, fmt.Sprintf("row %d: failed to start workflow '%s': %v", i+1, w.WorkflowName, err))
			if w.Mode == "sync" {
				continue
			}
			// async mode: still emit the record, tagged as failed to dispatch
			recCopy["_workflow_status"] = "dispatch_failed"
			output = append(output, recCopy)
			continue
		}

		recCopy["_workflow_run_id"] = run.GetID()
		recCopy["_workflow_run_run_id"] = run.GetRunID()
		recCopy["_workflow_name"] = w.WorkflowName

		if w.Mode == "sync" {
			var result workflows.WorkflowResult
			if err := run.Get(ctx, &result); err != nil {
				errs = append(errs, fmt.Sprintf("row %d: workflow '%s' failed: %v", i+1, w.WorkflowName, err))
				recCopy["_workflow_status"] = "failed"
			} else {
				recCopy["_workflow_status"] = result.Status
				recCopy["_workflow_result"] = result.FinalState
			}
		} else {
			recCopy["_workflow_status"] = "dispatched"
		}

		output = append(output, recCopy)
	}
	return output, errs, nil
}

// BOCrudTransformer executes full CRUD operations on STI business objects
type BOCrudTransformer struct {
	Driver    *BODriver
	TenantID  uuid.UUID
	Table     string
	Operation string // INSERT, READ, UPDATE, DELETE
}

func (b *BOCrudTransformer) Transform(ctx context.Context, input []PipelineRecord) ([]PipelineRecord, []string, error) {
	if b.Driver == nil {
		return input, nil, nil
	}
	res, err := b.Driver.ExecuteCRUD(ctx, b.TenantID, b.Operation, b.Table, input, "")
	if err != nil {
		return input, []string{err.Error()}, err
	}
	return res, nil, nil
}

// BloombergFieldsMapper converts raw bb_fields.csv records into BLOOMBERG_FIELD catalog nodes & properties
type BloombergFieldsMapper struct {
	CategoryPrefix string `json:"category_prefix"`
}

func (b *BloombergFieldsMapper) Transform(ctx context.Context, input []PipelineRecord) ([]PipelineRecord, []string, error) {
	output := make([]PipelineRecord, 0, len(input))
	var errs []string

	for i, r := range input {
		mnemonic := strings.TrimSpace(fmt.Sprintf("%v", r["FieldMnemonic"]))
		if mnemonic == "" || mnemonic == "<nil>" {
			mnemonic = strings.TrimSpace(fmt.Sprintf("%v", r["mnemonic"]))
		}
		if mnemonic == "" || mnemonic == "<nil>" {
			mnemonic = strings.TrimSpace(fmt.Sprintf("%v", r["FieldID"]))
		}
		if mnemonic == "" || mnemonic == "<nil>" {
			errs = append(errs, fmt.Sprintf("row %d: missing FieldMnemonic or FieldID", i+1))
			continue
		}

		fieldID := strings.TrimSpace(fmt.Sprintf("%v", r["FieldID"]))
		desc := strings.TrimSpace(fmt.Sprintf("%v", r["Description"]))
		definition := strings.TrimSpace(fmt.Sprintf("%v", r["Definition"]))
		dataLicCat := strings.TrimSpace(fmt.Sprintf("%v", r["DataLicenseCategory"]))
		cat := strings.TrimSpace(fmt.Sprintf("%v", r["Category"]))
		fieldType := strings.TrimSpace(fmt.Sprintf("%v", r["FieldType"]))
		backOffice := strings.TrimSpace(fmt.Sprintf("%v", r["BackOffice"]))
		prodDate := strings.TrimSpace(fmt.Sprintf("%v", r["ProductionDate"]))

		// Parse Sector Flags
		sectors := map[string]bool{
			"comdty": isTruthy(r["Comdty"]),
			"equity": isTruthy(r["Equity"]),
			"muni":   isTruthy(r["Muni"]),
			"pfd":    isTruthy(r["Pfd"]),
			"mmkt":   isTruthy(r["MMkt"]) || isTruthy(r["M-Mkt"]),
			"govt":   isTruthy(r["Govt"]),
			"corp":   isTruthy(r["Corp"]),
			"index":  isTruthy(r["Index"]),
			"curncy": isTruthy(r["Curncy"]),
			"mtge":   isTruthy(r["Mtge"]),
		}

		// Width and Decimals
		width, _ := strconv.Atoi(fmt.Sprintf("%v", r["StandardWidth"]))
		dec, _ := strconv.Atoi(fmt.Sprintf("%v", r["StandardDecimalPlaces"]))
		currMaxW, _ := strconv.Atoi(fmt.Sprintf("%v", r["CurrentMaximumWidth"]))
		heldSecOrder, _ := strconv.Atoi(fmt.Sprintf("%v", r["HeldSecuritiesOrder"]))
		heldSec := isTruthy(r["HeldSecurities"])

		properties := map[string]interface{}{
			"field_id":                 fieldID,
			"mnemonic":                 mnemonic,
			"description":              desc,
			"definition":               definition,
			"data_license_category":    dataLicCat,
			"category":                 cat,
			"field_type":               fieldType,
			"back_office":              backOffice,
			"production_date":          prodDate,
			"standard_width":           width,
			"standard_decimal_places":  dec,
			"current_maximum_width":    currMaxW,
			"held_securities":          heldSec,
			"held_securities_order":    heldSecOrder,
			"market_sectors":           sectors,
			"source":                   "bb_fields.csv",
			"vendor":                   "Bloomberg Data License",
		}

		qualifiedPath := fmt.Sprintf("bloomberg.fields/%s", mnemonic)
		nodeName := mnemonic
		if desc != "" {
			nodeName = fmt.Sprintf("%s (%s)", mnemonic, desc)
		}

		nodeRecord := PipelineRecord{
			"__graph_type":   "node",
			"node_name":      nodeName,
			"qualified_path": qualifiedPath,
			"catalog_type":   "BLOOMBERG_FIELD",
			"description":    definition,
			"properties":     properties,
			"is_active":      true,
		}

		output = append(output, nodeRecord)
	}

	return output, errs, nil
}

// RuleValidatorRule is a single named CEL validation rule evaluated per record.
type RuleValidatorRule struct {
	ID         string `json:"id"`
	Expression string `json:"expression"` // CEL expression evaluated against `input`; must return bool
	Message    string `json:"message,omitempty"`
}

// RuleValidatorTransformer validates each pipeline record against real
// business rules via internal/rules.RuleEngine (the same CEL/VM expression
// engine used by BO CRUD validation), instead of the previous "validator"
// node type stub/no-op. Failing records are reported through the errors
// slice (row-level error convention shared with AllowlistEnforcer /
// FilterTransformer) rather than silently dropped from the output.
//
// Config carries the rules to evaluate either as:
//   - `rules`: []RuleValidatorRule — literal CEL expressions (evaluated via
//     RuleEngine.EvaluateCEL against the record, optionally BO-context
//     enriched via BuildContextFromBORow when BO is set), or
//   - `rule_ids`/`core_rule_id`: pre-resolved *rules.RuleWithMetadata
//     (Rules field) evaluated via RuleEngine.EvaluateRule/EvaluateBatch —
//     for callers that have already loaded compiled rule nodes from the
//     rule repository (RuleValidatorTransformer itself does not resolve
//     IDs to compiled RuleNodes; that compilation step lives with the core
//     rule compiler/repository and must be done by the caller ahead of
//     time, e.g. in engine.go's node dispatch).
type RuleValidatorTransformer struct {
	Engine   *rules.RuleEngine
	TenantID string

	// CELRules are literal per-record CEL expressions to evaluate.
	CELRules []RuleValidatorRule

	// Rules are pre-resolved compiled rules (e.g. loaded by ID from the
	// rule repository ahead of time) evaluated via EvaluateRule/EvaluateBatch.
	Rules []*rules.RuleWithMetadata

	// BO optionally maps raw record fields to Business Object semantic
	// field names via BuildContextFromBORow before evaluation, matching the
	// vocabulary rules are normally authored against.
	BO *boresolver.BODefinition
}

func (v *RuleValidatorTransformer) Transform(ctx context.Context, input []PipelineRecord) ([]PipelineRecord, []string, error) {
	if v.Engine == nil {
		// No rule engine wired: pass records through unchanged rather than
		// silently failing every record.
		return input, nil, nil
	}

	output := make([]PipelineRecord, 0, len(input))
	var errs []string

	for i, record := range input {
		var evalCtx map[string]interface{}
		if v.BO != nil {
			evalCtx = rules.BuildContextFromBORow(v.BO, record)
		} else {
			evalCtx = map[string]interface{}(record)
		}

		recordFailed := false

		for _, r := range v.CELRules {
			if r.Expression == "" {
				continue
			}
			passed, err := v.Engine.EvaluateCEL(ctx, r.Expression, evalCtx)
			if err != nil {
				errs = append(errs, fmt.Sprintf("row %d: rule '%s' evaluation error: %v", i+1, r.ID, err))
				recordFailed = true
				continue
			}
			if !passed {
				msg := r.Message
				if msg == "" {
					msg = fmt.Sprintf("failed rule '%s'", r.ID)
				}
				errs = append(errs, fmt.Sprintf("row %d: %s", i+1, msg))
				recordFailed = true
			}
		}

		if len(v.Rules) > 0 {
			batch := v.Engine.EvaluateBatch(ctx, v.TenantID, v.Rules, evalCtx)
			if batch != nil && !batch.PassedAll {
				for _, res := range batch.Results {
					if res == nil || res.Passed {
						continue
					}
					if len(res.FailureReasons) > 0 {
						for _, reason := range res.FailureReasons {
							errs = append(errs, fmt.Sprintf("row %d: rule '%s' failed: %s", i+1, res.RuleID, reason))
						}
					} else {
						errs = append(errs, fmt.Sprintf("row %d: rule '%s' failed", i+1, res.RuleID))
					}
					recordFailed = true
				}
			}
		}

		if !recordFailed {
			output = append(output, record)
		}
	}

	return output, errs, nil
}

func isTruthy(val interface{}) bool {
	if val == nil {
		return false
	}
	s := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", val)))
	return s != "" && s != "<nil>" && s != "false" && s != "0" && s != "n" && s != "no"
}

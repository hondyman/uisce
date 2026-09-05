package datapipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/client"

	"github.com/hondyman/uisce/backend/internal/apistudio"
	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/rules"
	"github.com/hondyman/uisce/backend/internal/secrets"
	"github.com/hondyman/uisce/backend/pkg/workflows"
)

// PipelineEngine coordinates the compilation and parallel execution of data pipelines
type PipelineEngine struct {
	db            *sqlx.DB
	boDriver      *BODriver
	catalogDriver *CatalogDriver
	runRepo       *RunRepository

	// ruleEngine backs the "validator" node type (RuleValidatorTransformer)
	// with real CEL/VM rule evaluation. May be nil in tests/mock setups, in
	// which case validator nodes pass records through unchanged.
	ruleEngine *rules.RuleEngine

	// temporalClient backs durable, Temporal-workflow-based execution
	// (ExecuteRunAsWorkflow) and WorkflowCallerTransformer. May be nil when
	// only synchronous in-process execution (ExecuteRun) is needed.
	temporalClient client.Client

	runsMut     sync.RWMutex
	activeRuns  map[string]*PipelineExecutionRun
	subscribers map[string][]chan PipelineExecutionRun

	// telemetryBus broadcasts run events via Postgres LISTEN/NOTIFY for
	// external consumers (e.g. SSE bridges). May be nil.
	telemetryBus *TelemetryBus

	// apiRepo backs APICallerTransformer (api_caller node type). Provides
	// tenant-scoped endpoint lookup and telemetry logging. May be nil in tests
	// (api_caller nodes return an error in that case).
	apiRepo  *apistudio.Repository
	secrets  secrets.Provider
}

// NewPipelineEngine creates a new pipeline execution engine. ruleEngine and
// temporalClient may both be nil (e.g. in unit tests) — validator nodes and
// durable/workflow-calling execution degrade gracefully when unset.
func NewPipelineEngine(db *sqlx.DB, ruleEngine *rules.RuleEngine, temporalClient client.Client) *PipelineEngine {
	boDriver := NewBODriver(db)
	catalogDriver := NewCatalogDriver(db)
	runRepo := NewRunRepository(db)

	return &PipelineEngine{
		db:             db,
		boDriver:       boDriver,
		catalogDriver:  catalogDriver,
		runRepo:        runRepo,
		ruleEngine:     ruleEngine,
		temporalClient: temporalClient,
		activeRuns:     make(map[string]*PipelineExecutionRun),
		subscribers:    make(map[string][]chan PipelineExecutionRun),
	}
}

// AttachTelemetryBus wires a Postgres-backed TelemetryBus into the engine.
// Calls NotifyRun on every subscriber notification, enabling SSE bridges and
// other out-of-process consumers to receive real-time run events. May be nil
// (e.g. in unit tests) — NotifyRun is a no-op when bus is unset.
func (e *PipelineEngine) AttachTelemetryBus(bus *TelemetryBus) {
	e.telemetryBus = bus
}

// SetAPIEndpointRepository wires the API Studio repository and secrets provider
// into the engine for use by the api_caller transformer. Backward-compatible:
// if apiRepo is nil, api_caller nodes return a clear error at Transform time
// rather than silently stubbing.
func (e *PipelineEngine) SetAPIEndpointRepository(apiRepo *apistudio.Repository, sp secrets.Provider) {
	e.apiRepo = apiRepo
	e.secrets = sp
}

// SubscribeRun attaches a channel to stream live execution progress updates
func (e *PipelineEngine) SubscribeRun(runID string) (chan PipelineExecutionRun, func()) {
	e.runsMut.Lock()
	defer e.runsMut.Unlock()

	ch := make(chan PipelineExecutionRun, 10)
	e.subscribers[runID] = append(e.subscribers[runID], ch)

	cleanup := func() {
		e.runsMut.Lock()
		defer e.runsMut.Unlock()
		subs := e.subscribers[runID]
		for i, sub := range subs {
			if sub == ch {
				e.subscribers[runID] = append(subs[:i], subs[i+1:]...)
				close(ch)
				break
			}
		}
	}
	return ch, cleanup
}

func (e *PipelineEngine) notifySubscribers(run PipelineExecutionRun, nodeID string) {
	e.runsMut.RLock()
	subs, ok := e.subscribers[run.RunID.String()]
	e.runsMut.RUnlock()

	if ok {
		for _, ch := range subs {
			select {
			case ch <- run:
			default:
			}
		}
	}

	if e.telemetryBus == nil {
		return
	}

	if nodeID == "" {
		_ = e.telemetryBus.NotifyCompletion(context.Background(), run)
	} else {
		_ = e.telemetryBus.NotifyStep(context.Background(), run, nodeID)
	}
}

// GetRun returns the execution run state
func (e *PipelineEngine) GetRun(runID string) (*PipelineExecutionRun, bool) {
	e.runsMut.RLock()
	defer e.runsMut.RUnlock()
	run, ok := e.activeRuns[runID]
	return run, ok
}

func (e *PipelineEngine) GetRunWithFallback(ctx context.Context, runID string) (*PipelineExecutionRun, bool) {
	if run, ok := e.GetRun(runID); ok {
		return run, true
	}
	if e.runRepo == nil {
		return nil, false
	}
	uid, err := uuid.Parse(runID)
	if err != nil {
		return nil, false
	}
	run, err := e.runRepo.GetRun(ctx, uid)
	if err != nil {
		return nil, false
	}
	return run, true
}

// CompileDAG parses raw JSON DAG into a PipelineDAG
func (e *PipelineEngine) CompileDAG(rawJSON json.RawMessage) (*PipelineDAG, error) {
	var dag PipelineDAG
	if err := json.Unmarshal(rawJSON, &dag); err != nil {
		return nil, fmt.Errorf("invalid pipeline DAG JSON: %w", err)
	}
	if len(dag.Nodes) == 0 {
		return nil, fmt.Errorf("pipeline DAG contains no nodes")
	}
	return &dag, nil
}

// ExecuteRun executes a pipeline DAG in parallel with full telemetry.
// RunID is auto-generated via uuid.New().
func (e *PipelineEngine) ExecuteRun(ctx context.Context, tenantID uuid.UUID, def PipelineDefinition, inputRecords []PipelineRecord, isDryRun bool) (*PipelineExecutionRun, error) {
	return e.executeRunWithRunID(ctx, tenantID, uuid.New(), def, inputRecords, isDryRun)
}

// executeRunWithRunID is the internal implementation. runID may be pre-allocated
// by the caller (e.g. ExecuteRunAsWorkflow) so the runID is known before
// execution begins, enabling SSE subscription before the run starts.
func (e *PipelineEngine) executeRunWithRunID(ctx context.Context, tenantID uuid.UUID, runID uuid.UUID, def PipelineDefinition, inputRecords []PipelineRecord, isDryRun bool) (*PipelineExecutionRun, error) {
	dag, err := e.CompileDAG(def.DAGJSON)
	if err != nil {
		return nil, err
	}

	concurrency := def.Concurrency
	if concurrency <= 0 {
		concurrency = 8
	}
	batchSize := def.BatchSize
	if batchSize <= 0 {
		batchSize = 2000
	}

	run := &PipelineExecutionRun{
		RunID:         runID,
		PipelineID:    def.ID,
		TenantID:      tenantID,
		Status:        "running",
		StartTime:     time.Now().UTC(),
		StepTelemetry: make(map[string]StepMetrics),
		StepOrder:     make([]string, 0),
	}
	if isDryRun {
		run.Status = "simulated"
	}

	e.runsMut.Lock()
	e.activeRuns[runID.String()] = run
	e.runsMut.Unlock()

	if e.runRepo != nil {
		if err := e.runRepo.CreateRun(ctx, run, def.DAGJSON); err != nil {
			// Log but don't fail the run — DB write is best-effort
			fmt.Printf("WARNING: failed to persist run start: %v\n", err)
		}
	}

	e.notifySubscribers(*run, "")

	currentRecords := inputRecords

	orderedNodes, err := topologicalOrder(*dag)
	if err != nil {
		return nil, fmt.Errorf("executeRunWithRunID: invalid DAG: %w", err)
	}

	for _, node := range orderedNodes {
		stepStart := time.Now()
		stepMetric := StepMetrics{
			NodeID:    node.ID,
			NodeLabel: node.Label,
			NodeType:  node.Type,
			Status:    "running",
		}

		var stepOut []PipelineRecord
		var stepErrs []string
		var stepErr error

		switch node.Type {
		case "source", "reader":
			stepOut, stepErr = e.executeSource(ctx, tenantID, node)
		case "transform", "validator", "filter", "graph_synthesizer":
			stepOut, stepErrs, stepErr = e.executeTransform(ctx, tenantID, node, currentRecords, concurrency)
		case "loader", "writer", "sink":
			if isDryRun {
				stepOut = currentRecords
			} else {
				stepOut, stepErr = e.executeLoader(ctx, tenantID, node, currentRecords, concurrency, batchSize)
			}
		default:
			stepOut = currentRecords
		}

		stepDuration := time.Since(stepStart)
		stepMetric.Duration = stepDuration
		stepMetric.RecordsIn = int64(len(currentRecords))
		stepMetric.RecordsOut = int64(len(stepOut))
		stepMetric.RecordsError = int64(len(stepErrs))
		if stepDuration.Seconds() > 0 {
			stepMetric.RowsPerSec = float64(stepMetric.RecordsOut) / stepDuration.Seconds()
		}

		if stepErr != nil {
			stepMetric.Status = "failed"
			stepMetric.ErrorMessage = stepErr.Error()
			run.ErrorDetails = append(run.ErrorDetails, fmt.Sprintf("Node [%s] error: %v", node.Label, stepErr))
			if def.ErrorPolicy == "fail_fast" {
				run.Status = "failed"
				now := time.Now().UTC()
				run.EndTime = &now
				run.StepTelemetry[node.ID] = stepMetric
				if e.runRepo != nil {
					e.runRepo.UpsertStepTelemetry(ctx, runID, node.ID, stepMetric, len(run.StepOrder))
					e.runRepo.UpdateRunCompletion(ctx, run)
				}
				e.notifySubscribers(*run, node.ID)
				return run, stepErr
			}
		} else {
			stepMetric.Status = "completed"
		}

		run.StepTelemetry[node.ID] = stepMetric
		run.StepOrder = append(run.StepOrder, node.ID)
		currentRecords = stepOut

		if stepMetric.RowsPerSec > run.PeakThroughput {
			run.PeakThroughput = stepMetric.RowsPerSec
		}

		if e.runRepo != nil {
			if err := e.runRepo.UpsertStepTelemetry(ctx, runID, node.ID, stepMetric, len(run.StepOrder)-1); err != nil {
				fmt.Printf("WARNING: failed to persist step telemetry: %v\n", err)
			}
		}

		e.notifySubscribers(*run, node.ID)
	}

	now := time.Now().UTC()
	run.EndTime = &now
	if run.Status != "failed" {
		if isDryRun {
			run.Status = "simulated"
		} else {
			run.Status = "completed"
		}
	}

	if len(inputRecords) > 0 {
		run.TotalRecordsIn = int64(len(inputRecords))
	} else if len(run.StepTelemetry) > 0 {
		for _, s := range run.StepTelemetry {
			if s.NodeType == "source" || s.NodeType == "reader" {
				run.TotalRecordsIn = s.RecordsOut
				break
			}
		}
	}
	run.TotalRecordsOut = int64(len(currentRecords))

	if len(currentRecords) > 10 {
		run.SampleOutput = currentRecords[:10]
	} else {
		run.SampleOutput = currentRecords
	}

	if e.runRepo != nil {
		if err := e.runRepo.UpdateRunCompletion(ctx, run); err != nil {
			fmt.Printf("WARNING: failed to persist run completion: %v\n", err)
		}
	}

	e.notifySubscribers(*run, "")
	return run, nil
}

// RunPipelineSync loads the pipeline definition by ID and executes it
// synchronously (in-process, no Temporal) for a single trigger-supplied
// record. It implements validation.PipelineTriggerExecutor, letting BO CRUD
// "sync" DispatchMode triggers (internal/validation.TriggerValidationEngine)
// invoke a pipeline without internal/validation importing this package
// directly.
func (e *PipelineEngine) RunPipelineSync(ctx context.Context, tenantID uuid.UUID, pipelineID uuid.UUID, record map[string]interface{}) error {
	def, err := e.loadPipelineDefinition(ctx, tenantID, pipelineID)
	if err != nil {
		return err
	}
	run, err := e.ExecuteRun(ctx, tenantID, *def, []PipelineRecord{record}, false)
	if err != nil {
		return err
	}
	if run.Status == "failed" {
		if len(run.ErrorDetails) > 0 {
			return fmt.Errorf("pipeline run failed: %s", run.ErrorDetails[0])
		}
		return fmt.Errorf("pipeline run failed")
	}
	return nil
}

func (e *PipelineEngine) loadPipelineDefinition(ctx context.Context, tenantID uuid.UUID, pipelineID uuid.UUID) (*PipelineDefinition, error) {
	if e.db == nil {
		return nil, fmt.Errorf("pipeline engine has no database configured")
	}
	var def PipelineDefinition
	query := `
		SELECT id, tenant_id, name, description, mode, target_entity, dag_json,
		       concurrency, batch_size, error_policy, is_active, created_by, created_at, last_modified_at
		FROM data_pipeline_definitions
		WHERE id = $1 AND tenant_id = $2 AND is_active = true
	`
	if err := e.db.GetContext(ctx, &def, query, pipelineID, tenantID); err != nil {
		return nil, fmt.Errorf("load pipeline '%s': %w", pipelineID, err)
	}
	return &def, nil
}

// ExecuteRunAsWorkflow starts a durable, Temporal-backed execution of the
// given pipeline definition and returns immediately with the started
// workflow's ID. The actual DAG walk (PipelineEngine.ExecuteRun) runs
// inside RunPipelineDAGWorkflow/ActivityExecutePipelineDAG on the deployed
// worker (cmd/worker), which polls workflows.DeployedBPTaskQueue
// ("bp_queue") — NOT workflows.BPTaskQueue ("bp-framework-queue"), which no
// deployed worker polls.
// ExecuteRunAsWorkflow starts a durable, Temporal-backed execution of the
// given pipeline definition and returns both the workflowID and the pre-allocated
// runID. The runID is known before the activity starts so callers (e.g. SSE
// subscription handlers) can subscribe to telemetry before execution begins.
func (e *PipelineEngine) ExecuteRunAsWorkflow(ctx context.Context, tenantID uuid.UUID, def PipelineDefinition, inputRecords []PipelineRecord) (workflowID string, runID uuid.UUID, err error) {
	if e.temporalClient == nil {
		return "", uuid.Nil, fmt.Errorf("pipeline engine has no temporal client configured")
	}

	runID = uuid.New()
	workflowID = fmt.Sprintf("data-pipeline-%s-%d", def.ID, time.Now().UnixNano())
	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: workflows.DeployedBPTaskQueue,
	}

	input := PipelineWorkflowInput{
		TenantID:     tenantID,
		Definition:   def,
		InputRecords: inputRecords,
		RunID:        runID,
	}

	run, err := e.temporalClient.ExecuteWorkflow(ctx, options, RunPipelineDAGWorkflow, input)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("failed to start durable pipeline workflow: %w", err)
	}
	return run.GetID(), runID, nil
}

func (e *PipelineEngine) executeSource(ctx context.Context, tenantID uuid.UUID, node PipelineNode) ([]PipelineRecord, error) {
	subType, _ := node.Config["sourceType"].(string)
	if subType == "" {
		subType = node.SubType
	}

	switch subType {
	case "bo_reader", "business_object":
		table, _ := node.Config["table"].(string)
		if table == "" {
			table = "oms.account"
		}
		subtypeFilter, _ := node.Config["subtype_code"].(string)
		limit := 1000
		if l, ok := node.Config["limit"].(float64); ok && l > 0 {
			limit = int(l)
		}
		return e.boDriver.ExtractSTI(ctx, tenantID, table, subtypeFilter, limit, 0)

	case "catalog_reader", "catalog_graph":
		catType, _ := node.Config["catalog_type"].(string)
		limit := 1000
		if l, ok := node.Config["limit"].(float64); ok && l > 0 {
			limit = int(l)
		}
		return e.catalogDriver.ExtractCatalogNodes(ctx, tenantID, catType, limit, 0)

	case "raw_json", "mock":
		if rawData, ok := node.Config["raw_data"].([]interface{}); ok {
			var records []PipelineRecord
			for _, item := range rawData {
				if rMap, ok := item.(map[string]interface{}); ok {
					records = append(records, rMap)
				}
			}
			return records, nil
		}
		return []PipelineRecord{
			{"id": uuid.New().String(), "account_number": "ACC-TEST-001", "account_name": "Prime Institutional Fund", "subtype_code": "institutional", "base_currency": "USD", "status": "active"},
			{"id": uuid.New().String(), "account_number": "ACC-TEST-002", "account_name": "Omega SMA Strategy", "subtype_code": "sma", "base_currency": "USD", "status": "active"},
		}, nil

	default:
		return nil, nil
	}
}

func (e *PipelineEngine) executeTransform(ctx context.Context, tenantID uuid.UUID, node PipelineNode, records []PipelineRecord, concurrency int) ([]PipelineRecord, []string, error) {
	subType := node.SubType
	if subType == "" {
		subType, _ = node.Config["transformType"].(string)
	}
	// A bare "validator" node type with no more specific subType routes to
	// real rule validation by default (replaces the previous no-op stub).
	if node.Type == "validator" && (subType == "" || subType == "validate") {
		subType = "rule_validator"
	}

	switch subType {
	case "rule_validator":
		var celRules []RuleValidatorRule
		if rawRules, ok := node.Config["rules"].([]interface{}); ok {
			for _, rr := range rawRules {
				m, ok := rr.(map[string]interface{})
				if !ok {
					continue
				}
				id, _ := m["id"].(string)
				expr, _ := m["expression"].(string)
				msg, _ := m["message"].(string)
				if expr == "" {
					continue
				}
				celRules = append(celRules, RuleValidatorRule{ID: id, Expression: expr, Message: msg})
			}
		}
		// Convenience: a single "expression" key on the node config, for
		// simple single-rule validator tiles.
		if expr, ok := node.Config["expression"].(string); ok && expr != "" {
			id, _ := node.Config["rule_id"].(string)
			msg, _ := node.Config["message"].(string)
			celRules = append(celRules, RuleValidatorRule{ID: id, Expression: expr, Message: msg})
		}

		validator := &RuleValidatorTransformer{
			Engine:   e.ruleEngine,
			TenantID: tenantID.String(),
			CELRules: celRules,
		}
		return validator.Transform(ctx, records)

	case "column_mapper", "mapper":
		mappings := make(map[string]string)
		if rawMap, ok := node.Config["mappings"].(map[string]interface{}); ok {
			for k, v := range rawMap {
				mappings[k] = fmt.Sprintf("%v", v)
			}
		}
		types := make(map[string]string)
		if rawTypes, ok := node.Config["types"].(map[string]interface{}); ok {
			for k, v := range rawTypes {
				types[k] = fmt.Sprintf("%v", v)
			}
		}
		mapper := &ColumnMapper{Mappings: mappings, Types: types}
		return mapper.Transform(ctx, records)

	case "filter":
		field, _ := node.Config["field"].(string)
		op, _ := node.Config["operator"].(string)
		val := node.Config["value"]
		filter := &FilterTransformer{Field: field, Operator: op, Value: val}
		return filter.Transform(ctx, records)

	case "subtype_allowlist", "allowlist_enforcer":
		subtypes, err := e.boDriver.GetSubtypes(ctx, tenantID, "")
		if err != nil {
			return records, nil, nil
		}
		allowlists := make(map[string][]string)
		for _, s := range subtypes {
			allowlists[s.SubtypeCode] = s.FieldAllowlist
		}
		enforcer := &AllowlistEnforcer{Allowlists: allowlists}
		return enforcer.Transform(ctx, records)

	case "graph_synthesizer", "catalog_graph_builder":
		parentField, _ := node.Config["parent_field"].(string)
		if parentField == "" {
			parentField = "table_name"
		}
		childField, _ := node.Config["child_field"].(string)
		if childField == "" {
			childField = "column_name"
		}
		typeField, _ := node.Config["data_type_field"].(string)
		if typeField == "" {
			typeField = "data_type"
		}
		pred, _ := node.Config["edge_predicate"].(string)
		if pred == "" {
			pred = "COLUMN_OF"
		}
		synth := &GraphSynthesizer{
			ParentPathField: parentField,
			ChildNameField:  childField,
			DataTypeField:   typeField,
			EdgePredicate:   pred,
		}
		return synth.Transform(ctx, records)

	case "api_caller", "api_builder_caller", "rest_caller":
		endpointIDStr, _ := node.Config["endpoint_id"].(string)
		if endpointIDStr == "" {
			return records, nil, fmt.Errorf("api_caller transformer: endpoint_id is required in node config; endpoint_url is no longer accepted — register the endpoint in API Studio and reference endpoint_id")
		}
		endpointID, err := uuid.Parse(endpointIDStr)
		if err != nil {
			return records, nil, fmt.Errorf("api_caller transformer: invalid endpoint_id %q: %w", endpointIDStr, err)
		}
		requestTemplate, _ := node.Config["request_template"].(map[string]interface{})
		targetField, _ := node.Config["target_field"].(string)
		mergeOutput, _ := node.Config["merge_output"].(bool)
		caller := NewAPICallerTransformer(e.apiRepo, nil, e.secrets)
		caller.APIEndpointID = endpointID
		caller.RequestTemplate = requestTemplate
		caller.TargetField = targetField
		caller.MergeOutput = mergeOutput
		return caller.Transform(ctx, records)

	case "workflow_caller", "flow_builder_invoker":
		wfID, _ := node.Config["workflow_id"].(string)
		wfName, _ := node.Config["workflow_name"].(string)
		mode, _ := node.Config["mode"].(string)
		caller := &WorkflowCallerTransformer{
			WorkflowID:     wfID,
			WorkflowName:   wfName,
			Mode:           mode,
			TemporalClient: e.temporalClient,
		}
		return caller.Transform(ctx, records)

	case "bo_crud", "business_object_crud":
		table, _ := node.Config["table"].(string)
		if table == "" {
			table = "oms.account"
		}
		op, _ := node.Config["operation"].(string)
		if op == "" {
			op = "INSERT"
		}
		crud := &BOCrudTransformer{
			Driver:    e.boDriver,
			TenantID:  tenantID,
			Table:     table,
			Operation: op,
		}
		return crud.Transform(ctx, records)

	case "bloomberg_field_mapper", "bloomberg_fields":
		prefix, _ := node.Config["category_prefix"].(string)
		mapper := &BloombergFieldsMapper{CategoryPrefix: prefix}
		return mapper.Transform(ctx, records)

	case "host_runtime_calc", "calc_engine":
		entityField, _ := node.Config["entity_field"].(string)
		if entityField == "" {
			entityField = "id"
		}

		var calcNodes []*boresolver.CalcNode
		if rawNodes, ok := node.Config["nodes"].([]interface{}); ok {
			for _, rn := range rawNodes {
				m, ok := rn.(map[string]interface{})
				if !ok {
					continue
				}
				termKey, _ := m["term_key"].(string)
				formula, _ := m["formula"].(string)
				var deps []string
				if rawDeps, ok := m["dependencies"].([]interface{}); ok {
					for _, d := range rawDeps {
						deps = append(deps, fmt.Sprintf("%v", d))
					}
				}
				calcNodes = append(calcNodes, &boresolver.CalcNode{TermKey: termKey, Formula: formula, Dependencies: deps})
			}
		}

		calc := &HostRuntimeCalcTransformer{
			Nodes:       calcNodes,
			EntityField: entityField,
			TenantID:    tenantID.String(),
		}
		return calc.Transform(ctx, records)

	default:
		return records, nil, nil
	}
}

func (e *PipelineEngine) executeLoader(ctx context.Context, tenantID uuid.UUID, node PipelineNode, records []PipelineRecord, concurrency int, batchSize int) ([]PipelineRecord, error) {
	loaderType, _ := node.Config["loaderType"].(string)
	if loaderType == "" {
		loaderType = node.SubType
	}

	switch loaderType {
	case "bo_loader", "business_object":
		table, _ := node.Config["table"].(string)
		if table == "" {
			table = "oms.account"
		}
		// Parallel chunked ingestion
		chunks := chunkRecords(records, batchSize)
		var wg sync.WaitGroup
		errChan := make(chan error, len(chunks))
		sem := make(chan struct{}, concurrency)

		for _, chunk := range chunks {
			wg.Add(1)
			sem <- struct{}{}
			go func(c []PipelineRecord) {
				defer wg.Done()
				defer func() { <-sem }()
				if _, err := e.boDriver.BulkLoadSTI(ctx, tenantID, table, c); err != nil {
					errChan <- err
				}
			}(chunk)
		}
		wg.Wait()
		close(errChan)

		if len(errChan) > 0 {
			return records, <-errChan
		}
		return records, nil

	case "catalog_loader", "catalog_graph":
		// Separate node records from edge records
		var nodeRecords []PipelineRecord
		var edgeRecords []PipelineRecord

		nodeTypeID, _ := node.Config["node_type_id"].(string)

		for _, r := range records {
			if gType, ok := r["__graph_type"].(string); ok && gType == "edge" {
				edgeRecords = append(edgeRecords, r)
			} else {
				rec := cloneRecord(r)
				if nodeTypeID != "" {
					rec["node_type_id"] = nodeTypeID
				}
				nodeRecords = append(nodeRecords, rec)
			}
		}

		if len(nodeRecords) > 0 {
			if _, err := e.catalogDriver.BulkLoadCatalogNodes(ctx, tenantID, nodeRecords); err != nil {
				return records, err
			}
		}
		if len(edgeRecords) > 0 {
			if _, err := e.catalogDriver.BulkLoadCatalogEdges(ctx, tenantID, edgeRecords); err != nil {
				return records, err
			}
		}
		return records, nil

	default:
		return records, nil
	}
}

func chunkRecords(records []PipelineRecord, size int) [][]PipelineRecord {
	if size <= 0 {
		size = 2000
	}
	var chunks [][]PipelineRecord
	for i := 0; i < len(records); i += size {
		end := i + size
		if end > len(records) {
			end = len(records)
		}
		chunks = append(chunks, records[i:end])
	}
	return chunks
}

func cloneRecord(r PipelineRecord) PipelineRecord {
	out := make(PipelineRecord)
	for k, v := range r {
		out[k] = v
	}
	return out
}

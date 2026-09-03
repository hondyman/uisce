package datapipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/boresolver"
)

// PipelineEngine coordinates the compilation and parallel execution of data pipelines
type PipelineEngine struct {
	db            *sqlx.DB
	boDriver      *BODriver
	catalogDriver *CatalogDriver

	runsMut     sync.RWMutex
	activeRuns  map[string]*PipelineExecutionRun
	subscribers map[string][]chan PipelineExecutionRun
}

// NewPipelineEngine creates a new pipeline execution engine
func NewPipelineEngine(db *sqlx.DB) *PipelineEngine {
	boDriver := NewBODriver(db)
	catalogDriver := NewCatalogDriver(db)

	return &PipelineEngine{
		db:            db,
		boDriver:      boDriver,
		catalogDriver: catalogDriver,
		activeRuns:    make(map[string]*PipelineExecutionRun),
		subscribers:   make(map[string][]chan PipelineExecutionRun),
	}
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

func (e *PipelineEngine) notifySubscribers(run PipelineExecutionRun) {
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
}

// GetRun returns the execution run state
func (e *PipelineEngine) GetRun(runID string) (*PipelineExecutionRun, bool) {
	e.runsMut.RLock()
	defer e.runsMut.RUnlock()
	run, ok := e.activeRuns[runID]
	return run, ok
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

// ExecuteRun executes a pipeline DAG in parallel with full telemetry
func (e *PipelineEngine) ExecuteRun(ctx context.Context, tenantID uuid.UUID, def PipelineDefinition, inputRecords []PipelineRecord, isDryRun bool) (*PipelineExecutionRun, error) {
	dag, err := e.CompileDAG(def.DAGJSON)
	if err != nil {
		return nil, err
	}

	runID := uuid.New()
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
	}
	if isDryRun {
		run.Status = "simulated"
	}

	e.runsMut.Lock()
	e.activeRuns[runID.String()] = run
	e.runsMut.Unlock()

	e.notifySubscribers(*run)

	// Topologically order nodes or follow linear sequence
	currentRecords := inputRecords

	// If source node is present and no inputRecords provided, execute source reader
	for _, node := range dag.Nodes {
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
				// In dry-run simulation, mock the load without writing to DB
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
				e.notifySubscribers(*run)
				return run, stepErr
			}
		} else {
			stepMetric.Status = "completed"
		}

		run.StepTelemetry[node.ID] = stepMetric
		currentRecords = stepOut

		if stepMetric.RowsPerSec > run.PeakThroughput {
			run.PeakThroughput = stepMetric.RowsPerSec
		}

		e.notifySubscribers(*run)
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
		// Take first step's records
		for _, s := range run.StepTelemetry {
			if s.NodeType == "source" || s.NodeType == "reader" {
				run.TotalRecordsIn = s.RecordsOut
				break
			}
		}
	}
	run.TotalRecordsOut = int64(len(currentRecords))

	// Include sample preview
	if len(currentRecords) > 10 {
		run.SampleOutput = currentRecords[:10]
	} else {
		run.SampleOutput = currentRecords
	}

	e.notifySubscribers(*run)
	return run, nil
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

	switch subType {
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
		endpoint, _ := node.Config["endpoint_url"].(string)
		method, _ := node.Config["method"].(string)
		targetField, _ := node.Config["target_field"].(string)
		mergeOutput, _ := node.Config["merge_output"].(bool)
		caller := &APICallerTransformer{
			EndpointURL: endpoint,
			Method:      method,
			TargetField: targetField,
			MergeOutput: mergeOutput,
		}
		return caller.Transform(ctx, records)

	case "workflow_caller", "flow_builder_invoker":
		wfID, _ := node.Config["workflow_id"].(string)
		wfName, _ := node.Config["workflow_name"].(string)
		mode, _ := node.Config["mode"].(string)
		caller := &WorkflowCallerTransformer{
			WorkflowID:   wfID,
			WorkflowName: wfName,
			Mode:         mode,
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

		for _, r := range records {
			if gType, ok := r["__graph_type"].(string); ok && gType == "edge" {
				edgeRecords = append(edgeRecords, r)
			} else {
				nodeRecords = append(nodeRecords, r)
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

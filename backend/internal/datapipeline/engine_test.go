package datapipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hondyman/uisce/backend/internal/rules"
)

func TestTransforms_ColumnMapper(t *testing.T) {
	ctx := context.Background()
	mapper := &ColumnMapper{
		Mappings: map[string]string{
			"account_number": "acc_no",
			"balance":        "amt",
		},
		Types: map[string]string{
			"balance": "float64",
		},
	}

	input := []PipelineRecord{
		{"acc_no": "ACC-100", "amt": "54000.50", "subtype_code": "institutional"},
	}

	out, errs, err := mapper.Transform(ctx, input)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}
	if len(errs) > 0 {
		t.Fatalf("Expected 0 errors, got: %v", errs)
	}
	if len(out) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(out))
	}
	if out[0]["account_number"] != "ACC-100" {
		t.Errorf("Expected account_number 'ACC-100', got %v", out[0]["account_number"])
	}
	if out[0]["balance"] != 54000.50 {
		t.Errorf("Expected balance 54000.50, got %v", out[0]["balance"])
	}
	if _, exists := out[0]["acc_no"]; !exists {
		t.Errorf("Source key 'acc_no' should be retained after rename (copy semantics), but it is missing")
	}
	if _, exists := out[0]["amt"]; !exists {
		t.Errorf("Source key 'amt' should be retained after rename (copy semantics), but it is missing")
	}
}

func TestTransforms_ColumnMapper_MoveSemantics(t *testing.T) {
	ctx := context.Background()
	mapper := &ColumnMapper{
		Mappings: map[string]string{
			"account_number": "acc_no",
			"balance":        "amt",
		},
		Move: true,
	}

	input := []PipelineRecord{
		{"acc_no": "ACC-100", "amt": "54000.50", "subtype_code": "institutional"},
	}

	out, errs, err := mapper.Transform(ctx, input)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}
	if len(errs) > 0 {
		t.Fatalf("Expected 0 errors, got: %v", errs)
	}
	if len(out) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(out))
	}
	if out[0]["account_number"] != "ACC-100" {
		t.Errorf("Expected account_number 'ACC-100', got %v", out[0]["account_number"])
	}
	if v := out[0]["balance"]; v != "54000.50" {
		t.Errorf("Expected balance '54000.50', got %v (type %T)", v, v)
	}
	if _, exists := out[0]["acc_no"]; exists {
		t.Errorf("Source key 'acc_no' should be deleted after rename when Move=true, but it is present")
	}
	if _, exists := out[0]["amt"]; exists {
		t.Errorf("Source key 'amt' should be deleted after rename when Move=true, but it is present")
	}
}

func TestTransforms_AllowlistEnforcer(t *testing.T) {
	ctx := context.Background()
	enforcer := &AllowlistEnforcer{
		Allowlists: map[string][]string{
			"institutional": {"account_number", "sponsor_id", "status"},
		},
	}

	input := []PipelineRecord{
		{
			"subtype_code":   "institutional",
			"account_number": "ACC-INST-01",
			"sponsor_id":     "SPON-99",
			"unallowed_col":  "MALICIOUS_DATA",
			"status":         "active",
		},
	}

	out, errs, err := enforcer.Transform(ctx, input)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}
	if len(errs) != 1 {
		t.Errorf("Expected 1 stripped column warning, got %d", len(errs))
	}
	if _, exists := out[0]["unallowed_col"]; exists {
		t.Errorf("Expected unallowed_col to be stripped")
	}
	if out[0]["account_number"] != "ACC-INST-01" {
		t.Errorf("Expected account_number to remain")
	}
}

func TestTransforms_GraphSynthesizer(t *testing.T) {
	ctx := context.Background()
	synth := &GraphSynthesizer{
		ParentPathField: "table_name",
		ChildNameField:  "column_name",
		DataTypeField:   "data_type",
		EdgePredicate:   "COLUMN_OF",
	}

	input := []PipelineRecord{
		{"table_name": "oms.position", "column_name": "quantity", "data_type": "NUMERIC(18,4)"},
	}

	out, _, err := synth.Transform(ctx, input)
	if err != nil {
		t.Fatalf("GraphSynthesizer failed: %v", err)
	}

	// Should emit: 1 parent TABLE node, 1 child ATTRIBUTE node, 1 relationship EDGE
	if len(out) != 3 {
		t.Fatalf("Expected 3 graph elements, got %d", len(out))
	}

	if out[0]["node_name"] != "oms.position" || out[0]["catalog_type"] != "TABLE" {
		t.Errorf("Parent node mismatch: %v", out[0])
	}
	if out[1]["node_name"] != "quantity" || out[1]["catalog_type"] != "ATTRIBUTE" {
		t.Errorf("Child attribute mismatch: %v", out[1])
	}
	if out[2]["edge_type_name"] != "COLUMN_OF" {
		t.Errorf("Edge predicate mismatch: %v", out[2])
	}
}

func TestPipelineEngine_SimulationRun(t *testing.T) {
	ctx := context.Background()
	engine := NewPipelineEngine(nil, nil, nil)

	dag := PipelineDAG{
		Nodes: []PipelineNode{
			{
				ID:      "node-1",
				Type:    "source",
				SubType: "raw_json",
				Label:   "Raw Feed",
				Config: map[string]interface{}{
					"raw_data": []interface{}{
						map[string]interface{}{"raw_acc": "ACC-001", "raw_type": "institutional", "val": 1000},
						map[string]interface{}{"raw_acc": "ACC-002", "raw_type": "institutional", "val": 2000},
					},
				},
			},
			{
				ID:      "node-2",
				Type:    "transform",
				SubType: "column_mapper",
				Label:   "Normalize Fields",
				Config: map[string]interface{}{
					"mappings": map[string]interface{}{
						"account_number": "raw_acc",
						"subtype_code":   "raw_type",
					},
				},
			},
			{
				ID:      "node-3",
				Type:    "loader",
				SubType: "bo_loader",
				Label:   "Uuisce Account STI Loader",
				Config: map[string]interface{}{
					"table": "oms.account",
				},
			},
		},
		Edges: []PipelineEdge{
			{ID: "e1", Source: "node-1", Target: "node-2"},
			{ID: "e2", Source: "node-2", Target: "node-3"},
		},
	}

	dagBytes, _ := json.Marshal(dag)
	def := PipelineDefinition{
		ID:          uuid.New(),
		TenantID:    uuid.New(),
		Name:        "Test Account Pipeline",
		DAGJSON:     dagBytes,
		Concurrency: 4,
		BatchSize:   100,
		ErrorPolicy: "skip_and_log",
	}

	run, err := engine.ExecuteRun(ctx, def.TenantID, def, nil, true)
	if err != nil {
		t.Fatalf("ExecuteRun failed: %v", err)
	}

	if run.Status != "simulated" {
		t.Errorf("Expected status 'simulated', got %s", run.Status)
	}
	if run.TotalRecordsOut != 2 {
		t.Errorf("Expected 2 records out, got %d", run.TotalRecordsOut)
	}
	if len(run.StepTelemetry) != 3 {
		t.Errorf("Expected telemetry for 3 nodes, got %d", len(run.StepTelemetry))
	}
}

func TestHTTPHandlers_RESTEndpoints(t *testing.T) {
	engine := NewPipelineEngine(nil, nil, nil)
	handler := NewDataPipelineHandler(nil, engine)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	// Test 1: Get BO Schema
	req := httptest.NewRequest(http.MethodGet, "/api/v1/data-pipelines/schema/business-objects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GetBOSchema failed with code %d", w.Code)
	}

	// Test 2: Test Single Step
	testStepPayload := TestStepRequest{
		NodeType: "transform",
		SubType:  "column_mapper",
		Config: map[string]interface{}{
			"mappings": map[string]interface{}{"target_col": "src_col"},
		},
		Input: []PipelineRecord{
			{"src_col": "HELLO_WORLD"},
		},
	}
	payloadBytes, _ := json.Marshal(testStepPayload)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/data-pipelines/test-step", bytes.NewBuffer(payloadBytes))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("TestStep failed with code %d", w2.Code)
	}

	var stepResp TestStepResponse
	_ = json.NewDecoder(w2.Body).Decode(&stepResp)
	if !stepResp.Success || len(stepResp.Output) != 1 || stepResp.Output[0]["target_col"] != "HELLO_WORLD" {
		t.Errorf("TestStep output unexpected: %v", stepResp)
	}
}

func TestTransforms_BloombergFieldsMapper(t *testing.T) {
	ctx := context.Background()
	mapper := &BloombergFieldsMapper{}

	input := []PipelineRecord{
		{
			"FieldID":                "DS62 ",
			"FieldMnemonic":          "144A_FLAG",
			"Description":            "Is 144A Eligible",
			"DataLicenseCategory":    "Security Master",
			"Category":               "Descriptive Info",
			"Definition":             "Indicates if the security is eligible for trading exemption under rule 144a. Returns a Y or N.",
			"Equity":                 "Equity",
			"Corp":                   "Corp",
			"Mtge":                   "Mtge",
			"StandardWidth":          4,
			"StandardDecimalPlaces":  0,
			"FieldType":              "Boolean",
			"ProductionDate":         "19980617",
			"CurrentMaximumWidth":    30,
			"HeldSecurities":         "True",
			"HeldSecuritiesOrder":    110,
		},
	}

	out, errs, err := mapper.Transform(ctx, input)
	if err != nil {
		t.Fatalf("BloombergFieldsMapper failed: %v", err)
	}
	if len(errs) > 0 {
		t.Errorf("Unexpected warnings: %v", errs)
	}
	if len(out) != 1 {
		t.Fatalf("Expected 1 catalog node record, got %d", len(out))
	}

	node := out[0]
	if node["catalog_type"] != "BLOOMBERG_FIELD" {
		t.Errorf("Expected catalog_type BLOOMBERG_FIELD, got %v", node["catalog_type"])
	}
	if node["qualified_path"] != "bloomberg.fields/144A_FLAG" {
		t.Errorf("Expected qualified_path bloomberg.fields/144A_FLAG, got %v", node["qualified_path"])
	}

	props, ok := node["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected properties map, got %v", node["properties"])
	}
	if props["mnemonic"] != "144A_FLAG" || props["field_id"] != "DS62" {
		t.Errorf("Properties mismatch: %v", props)
	}
	sectors, ok := props["market_sectors"].(map[string]bool)
	if !ok || !sectors["equity"] || !sectors["corp"] || !sectors["mtge"] || sectors["govt"] {
		t.Errorf("Sectors mismatch: %v", sectors)
	}
}

func TestRuleValidatorTransformer(t *testing.T) {
	ctx := context.Background()
	engine := rules.NewRuleEngine(nil)

	validator := &RuleValidatorTransformer{
		Engine:   engine,
		TenantID: uuid.New().String(),
		CELRules: []RuleValidatorRule{
			{
				ID:         "positive_balance",
				Expression: `input.balance > 0.0`,
				Message:    "balance must be positive",
			},
		},
	}

	input := []PipelineRecord{
		{"account_number": "ACC-1", "balance": 100.0},
		{"account_number": "ACC-2", "balance": -50.0},
	}

	out, errs, err := validator.Transform(ctx, input)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("Expected 1 passing record, got %d: %v", len(out), out)
	}
	if out[0]["account_number"] != "ACC-1" {
		t.Errorf("Expected ACC-1 to pass, got %v", out[0]["account_number"])
	}
	if len(errs) == 0 {
		t.Errorf("Expected an error for the failing record, got none")
	}
}

func TestRuleValidatorTransformer_NoEngine(t *testing.T) {
	ctx := context.Background()
	validator := &RuleValidatorTransformer{}

	input := []PipelineRecord{{"a": 1}, {"a": 2}}
	out, errs, err := validator.Transform(ctx, input)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}
	if len(out) != 2 || len(errs) != 0 {
		t.Errorf("Expected pass-through with nil engine, got out=%v errs=%v", out, errs)
	}
}

func TestEngine_ValidatorNodeDispatch(t *testing.T) {
	ctx := context.Background()
	engine := NewPipelineEngine(nil, rules.NewRuleEngine(nil), nil)
	tenantID := uuid.New()

	node := PipelineNode{
		ID:   "validate-1",
		Type: "validator",
		Config: map[string]interface{}{
			"expression": `input.status == "active"`,
			"message":    "status must be active",
		},
	}

	records := []PipelineRecord{
		{"status": "active"},
		{"status": "inactive"},
	}

	out, errs, err := engine.executeTransform(ctx, tenantID, node, records, 1)
	if err != nil {
		t.Fatalf("executeTransform failed: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("Expected 1 record to pass validator, got %d", len(out))
	}
	if len(errs) != 1 {
		t.Fatalf("Expected 1 validation error, got %d: %v", len(errs), errs)
	}
}

func TestExecuteRunWithRunID_HonorsPreallocatedID(t *testing.T) {
	engine := NewPipelineEngine(nil, nil, nil)

	dag := PipelineDAG{
		Nodes: []PipelineNode{
			{ID: "a", Type: "source", SubType: "raw_json", Label: "Source",
				Config: map[string]interface{}{
					"raw_data": []interface{}{
						map[string]interface{}{"name": "alpha", "value": float64(100)},
					},
				}},
			{ID: "b", Type: "loader", SubType: "catalog_graph", Label: "Sink",
				Config: map[string]interface{}{
					"node_type_id": "68d6d495-0992-4d92-ad2f-7f66dc1e7d78",
				}},
		},
		Edges: []PipelineEdge{
			{ID: "e1", Source: "a", Target: "b"},
		},
	}

	preallocatedID := uuid.New()
	tenantID := uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999")

	run, err := engine.executeRunWithRunID(context.Background(), tenantID, preallocatedID, PipelineDefinition{
		ID:       uuid.New(),
		TenantID: tenantID,
		DAGJSON:  mustMarshal(dag),
	}, nil, true)
	if err != nil {
		t.Fatalf("executeRunWithRunID failed: %v", err)
	}
	if run.RunID != preallocatedID {
		t.Errorf("RunID: expected %s, got %s", preallocatedID.String(), run.RunID.String())
	}
}

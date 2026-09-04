package datapipeline

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

// DataPipelineHandler serves REST endpoints for pipeline management, test runs, and live telemetry
type DataPipelineHandler struct {
	db     *sqlx.DB
	engine *PipelineEngine
	bus    *TelemetryBus
}

// NewDataPipelineHandler creates a new handler instance
func NewDataPipelineHandler(db *sqlx.DB, engine *PipelineEngine) *DataPipelineHandler {
	return &DataPipelineHandler{
		db:     db,
		engine: engine,
	}
}

// SetBus injects the TelemetryBus so the SSE handler can fan out via bus.Listen.
func (h *DataPipelineHandler) SetBus(bus *TelemetryBus) {
	h.bus = bus
}

// RegisterRoutes mounts the data pipeline routes on Chi router
func (h *DataPipelineHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/data-pipelines", func(r chi.Router) {
		r.Get("/", h.ListPipelines)
		r.Post("/", h.CreatePipeline)
		r.Get("/schema/business-objects", h.GetBOSchema)
		r.Get("/schema/catalog-types", h.GetCatalogSchema)
		r.Get("/schema/api-endpoints", h.GetAPIEndpoints)
		r.Get("/schema/workflows", h.GetWorkflows)
		r.Post("/test-step", h.TestStep)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetPipeline)
			r.Put("/", h.UpdatePipeline)
			r.Delete("/", h.DeletePipeline)
			r.Post("/simulate", h.SimulatePipeline)
			r.Post("/run", h.RunPipeline)
		})

		r.Get("/runs/{runId}", h.GetRunStatus)
		r.Get("/runs/{runId}/telemetry", h.StreamTelemetrySSE)
	})
}

func (h *DataPipelineHandler) getTenantID(r *http.Request) uuid.UUID {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims != nil && claims.TenantID != "" {
		if parsed, err := uuid.Parse(claims.TenantID); err == nil {
			return parsed
		}
	}
	if headerTenant := r.Header.Get("X-Tenant-ID"); headerTenant != "" {
		if parsed, err := uuid.Parse(headerTenant); err == nil {
			return parsed
		}
	}
	return uuid.MustParse("00000000-0000-0000-0000-000000000001")
}

func (h *DataPipelineHandler) ListPipelines(w http.ResponseWriter, r *http.Request) {
	tenantID := h.getTenantID(r)

	if h.db == nil {
		// Mock list
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]PipelineDefinition{
			{
				ID:             uuid.New(),
				TenantID:       tenantID,
				Name:           "Institutional Account & Trade Ingestion",
				Description:    "High-speed parallel bulk loader from external feeds to oms.account and oms.trade_order",
				Mode:           "business_object",
				TargetEntity:   "oms.trade_order",
				Concurrency:    8,
				BatchSize:      2000,
				IsActive:       true,
				CreatedAt:      time.Now().Add(-24 * time.Hour),
				LastModifiedAt: time.Now(),
			},
			{
				ID:             uuid.New(),
				TenantID:       tenantID,
				Name:           "Physical Schema to Catalog Graph Ingestor",
				Description:    "Transforms information schema tables into Catalog TABLE and ATTRIBUTE nodes and COLUMN_OF edges",
				Mode:           "catalog_graph",
				TargetEntity:   "catalog_node",
				Concurrency:    4,
				BatchSize:      1000,
				IsActive:       true,
				CreatedAt:      time.Now().Add(-48 * time.Hour),
				LastModifiedAt: time.Now(),
			},
		})
		return
	}

	query := `
		SELECT id, tenant_id, name, description, mode, target_entity, dag_json,
		       concurrency, batch_size, error_policy, is_active, created_by, created_at, last_modified_at
		FROM data_pipeline_definitions
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY last_modified_at DESC
	`
	var list []PipelineDefinition
	err := h.db.SelectContext(r.Context(), &list, query, tenantID)
	if err != nil {
		// Table might not be migrated yet in mock, return empty list
		list = []PipelineDefinition{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (h *DataPipelineHandler) CreatePipeline(w http.ResponseWriter, r *http.Request) {
	tenantID := h.getTenantID(r)

	var def PipelineDefinition
	if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
		http.Error(w, "Invalid payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	def.ID = uuid.New()
	def.TenantID = tenantID
	def.CreatedAt = time.Now().UTC()
	def.LastModifiedAt = time.Now().UTC()
	def.IsActive = true
	if def.Concurrency <= 0 {
		def.Concurrency = 8
	}
	if def.BatchSize <= 0 {
		def.BatchSize = 2000
	}
	if def.ErrorPolicy == "" {
		def.ErrorPolicy = "skip_and_log"
	}

	if h.db != nil {
		query := `
			INSERT INTO data_pipeline_definitions (
				id, tenant_id, name, description, mode, target_entity, dag_json,
				concurrency, batch_size, error_policy, is_active, created_by, created_at, last_modified_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
			)
		`
		_, _ = h.db.ExecContext(r.Context(), query,
			def.ID, def.TenantID, def.Name, def.Description, def.Mode, def.TargetEntity, def.DAGJSON,
			def.Concurrency, def.BatchSize, def.ErrorPolicy, def.IsActive, def.CreatedBy, def.CreatedAt, def.LastModifiedAt,
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(def)
}

func (h *DataPipelineHandler) GetPipeline(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	pID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid pipeline ID", http.StatusBadRequest)
		return
	}
	tenantID := h.getTenantID(r)

	var def PipelineDefinition
	if h.db != nil {
		query := `
			SELECT id, tenant_id, name, description, mode, target_entity, dag_json,
			       concurrency, batch_size, error_policy, is_active, created_by, created_at, last_modified_at
			FROM data_pipeline_definitions
			WHERE id = $1 AND tenant_id = $2
		`
		err = h.db.GetContext(r.Context(), &def, query, pID, tenantID)
		if err != nil {
			http.Error(w, "Pipeline not found", http.StatusNotFound)
			return
		}
	} else {
		def = PipelineDefinition{
			ID:             pID,
			TenantID:       tenantID,
			Name:           "Institutional Trade Ingestion Pipeline",
			Description:    "High-speed parallel execution DAG",
			Mode:           "business_object",
			TargetEntity:   "oms.trade_order",
			Concurrency:    8,
			BatchSize:      2000,
			DAGJSON:        json.RawMessage(`{"nodes":[],"edges":[]}`),
			IsActive:       true,
			CreatedAt:      time.Now(),
			LastModifiedAt: time.Now(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(def)
}

func (h *DataPipelineHandler) UpdatePipeline(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	pID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid pipeline ID", http.StatusBadRequest)
		return
	}
	tenantID := h.getTenantID(r)

	var def PipelineDefinition
	if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
		http.Error(w, "Invalid payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	def.ID = pID
	def.TenantID = tenantID
	def.LastModifiedAt = time.Now().UTC()

	if h.db != nil {
		query := `
			UPDATE data_pipeline_definitions SET
				name = $1, description = $2, mode = $3, target_entity = $4,
				dag_json = $5, concurrency = $6, batch_size = $7, error_policy = $8,
				last_modified_at = NOW()
			WHERE id = $9 AND tenant_id = $10
		`
		_, _ = h.db.ExecContext(r.Context(), query,
			def.Name, def.Description, def.Mode, def.TargetEntity,
			def.DAGJSON, def.Concurrency, def.BatchSize, def.ErrorPolicy,
			def.ID, def.TenantID,
		)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(def)
}

func (h *DataPipelineHandler) DeletePipeline(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	pID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid pipeline ID", http.StatusBadRequest)
		return
	}
	tenantID := h.getTenantID(r)

	if h.db != nil {
		_, _ = h.db.ExecContext(r.Context(),
			`UPDATE data_pipeline_definitions SET is_active = false, last_modified_at = NOW() WHERE id = $1 AND tenant_id = $2`,
			pID, tenantID)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *DataPipelineHandler) TestStep(w http.ResponseWriter, r *http.Request) {
	var req TestStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	tenantID := h.getTenantID(r)
	start := time.Now()

	node := PipelineNode{
		ID:      "test-step",
		Type:    req.NodeType,
		SubType: req.SubType,
		Config:  req.Config,
	}

	out, errs, err := h.engine.executeTransform(r.Context(), tenantID, node, req.Input, 2)
	durationMs := time.Since(start).Milliseconds()

	resp := TestStepResponse{
		Success:      err == nil,
		Output:       out,
		Errors:       errs,
		ExecutionMs:  durationMs,
		RecordsIn:    len(req.Input),
		RecordsOut:   len(out),
		RecordsError: len(errs),
	}
	if err != nil {
		resp.Errors = append(resp.Errors, err.Error())
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *DataPipelineHandler) SimulatePipeline(w http.ResponseWriter, r *http.Request) {
	tenantID := h.getTenantID(r)

	var payload struct {
		Pipeline PipelineDefinition `json:"pipeline"`
		Sample   []PipelineRecord   `json:"sample_data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	run, err := h.engine.ExecuteRun(r.Context(), tenantID, payload.Pipeline, payload.Sample, true)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(run)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}

func (h *DataPipelineHandler) RunPipeline(w http.ResponseWriter, r *http.Request) {
	tenantID := h.getTenantID(r)

	var payload struct {
		Pipeline PipelineDefinition `json:"pipeline"`
		Records  []PipelineRecord   `json:"records,omitempty"`
		Durable  bool               `json:"durable,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	// `?durable=true` query param is equivalent to payload.durable, for
	// callers that prefer to select the mode without touching the body.
	durable := payload.Durable
	if v := r.URL.Query().Get("durable"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			durable = parsed
		}
	}

	if durable {
		workflowID, runID, err := h.engine.ExecuteRunAsWorkflow(r.Context(), tenantID, payload.Pipeline, payload.Records)
		if err != nil {
			http.Error(w, "Failed to start durable pipeline run: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "submitted",
			"workflow_id": workflowID,
			"run_id":      runID.String(),
		})
		return
	}

	// Default: synchronous in-process run (existing behavior, unchanged).
	run, err := h.engine.ExecuteRun(r.Context(), tenantID, payload.Pipeline, payload.Records, false)
	if err != nil {
		http.Error(w, "Execution failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(run)
}

func (h *DataPipelineHandler) GetRunStatus(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runId")
	run, ok := h.engine.GetRun(runID)
	if !ok {
		http.Error(w, "Run not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}

func (h *DataPipelineHandler) StreamTelemetrySSE(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runId")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ch, cleanup := h.engine.SubscribeRun(runID)
	defer cleanup()

	// Send current state immediately if available
	if current, exists := h.engine.GetRun(runID); exists {
		bytes, _ := json.Marshal(current)
		fmt.Fprintf(w, "data: %s\n\n", bytes)
		flusher.Flush()
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case run, ok := <-ch:
			if !ok {
				return
			}
			bytes, _ := json.Marshal(run)
			fmt.Fprintf(w, "data: %s\n\n", bytes)
			flusher.Flush()
			if run.Status == "completed" || run.Status == "failed" {
				return
			}
		}
	}
}

func (h *DataPipelineHandler) GetBOSchema(w http.ResponseWriter, r *http.Request) {
	tenantID := h.getTenantID(r)
	subtypes, err := h.engine.boDriver.GetSubtypes(r.Context(), tenantID, "")
	if err != nil {
		http.Error(w, "Failed to load business object subtypes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tables := []map[string]interface{}{
		{"table": "oms.account", "domain": "OMS", "label": "OMS Accounts", "subtypes": []string{"institutional", "retail_wealth", "sma", "trust_estate", "qualified_retirement", "corporate_treasury"}},
		{"table": "oms.position", "domain": "OMS", "label": "OMS Positions", "subtypes": []string{"settled_long", "short_borrowed", "derivative_exposure", "pledged_collateral", "unsettled_pipeline"}},
		{"table": "oms.security", "domain": "OMS", "label": "OMS Securities", "subtypes": []string{"equity", "sovereign_debt", "corporate_debt", "structured_abs_mbs", "etd_derivative", "otc_derivative"}},
		{"table": "oms.trade_order", "domain": "OMS", "label": "OMS Trade Orders", "subtypes": []string{"block_parent", "dma_execution", "otc_bilateral", "fx_spot_forward", "primary_auction"}},
		{"table": "altinv.alternative_investment", "domain": "AltInv", "label": "Alternative Investments", "subtypes": []string{"private_equity", "venture_capital", "hedge_fund", "real_estate", "direct_investment", "infrastructure", "private_debt"}},
		{"table": "cash_flow.settlement", "domain": "CashFlow", "label": "Cash Flow Settlements", "subtypes": []string{"dividend", "coupon_fixed_income", "capital_call", "lp_distribution", "corporate_action", "expense_fee"}},
		{"table": "master.customer", "domain": "Master", "label": "Master Customers", "subtypes": []string{"institutional_client", "private_wealth", "broker_dealer", "corporate_treasury"}},
		{"table": "master.vendor", "domain": "Master", "label": "Master Vendors", "subtypes": []string{"custodian_prime_broker", "market_data", "fund_admin", "cloud_tech"}},
		{"table": "master.personnel", "domain": "Master", "label": "Master Personnel", "subtypes": []string{"portfolio_manager", "trade_execution", "compliance_officer", "client_advisor"}},
		{"table": "master.sales_ledger", "domain": "Master", "label": "Master Sales Ledger", "subtypes": []string{"aum_management_fee", "trading_commission", "performance_fee", "platform_subscription"}},
	}

	resp := map[string]interface{}{
		"tables":   tables,
		"registry": subtypes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *DataPipelineHandler) GetCatalogSchema(w http.ResponseWriter, r *http.Request) {
	tenantID := h.getTenantID(r)
	isGoldCopy, _ := h.engine.catalogDriver.CheckGoldCopy(r.Context(), tenantID)

	resp := map[string]interface{}{
		"is_gold_copy": isGoldCopy,
		"node_types": []map[string]string{
			{"type": "TABLE", "label": "Physical Table", "icon": "TableChart"},
			{"type": "ATTRIBUTE", "label": "Physical Column / Attribute", "icon": "ViewColumn"},
			{"type": "BUSINESS_OBJECT", "label": "Business Object Entity", "icon": "Business"},
			{"type": "BLOOMBERG_FIELD", "label": "Bloomberg Fields (Market Data Dictionary)", "icon": "LineChart"},
			{"type": "SEMANTIC_TERM", "label": "Semantic Term", "icon": "Lightbulb"},
			{"type": "METRIC", "label": "Calculated Metric", "icon": "Calculate"},
		},
		"edge_types": []map[string]string{
			{"predicate": "ATTRIBUTE_OF", "label": "Attribute Of Entity"},
			{"predicate": "COLUMN_OF", "label": "Column Of Table"},
			{"predicate": "IS_CLASSIFIED_AS", "label": "Is Classified As Semantic Term"},
			{"predicate": "CHILD_OF", "label": "Hierarchical Child Of"},
			{"predicate": "JOINED_WITH", "label": "Relationship Join Key"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *DataPipelineHandler) GetAPIEndpoints(w http.ResponseWriter, r *http.Request) {
	tenantID := h.getTenantID(r)

	type APIEndpointSummary struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Path        string `json:"path"`
		Method      string `json:"method"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}

	var endpoints []APIEndpointSummary
	if h.db != nil {
		_ = h.db.SelectContext(r.Context(), &endpoints, `
			SELECT id::text, name, path, method, description, status 
			FROM api_studio_endpoints 
			WHERE (tenant_id = $1 OR tenant_id = '00000000-0000-0000-0000-000000000001') AND status = 'published'
			ORDER BY name
		`, tenantID)
	}

	if len(endpoints) == 0 {
		endpoints = []APIEndpointSummary{
			{ID: "ep-1", Name: "Verify Customer KYC", Path: "/api/v1/customers/verify-kyc", Method: "POST", Description: "Executes compliance and AML verification on client payload", Status: "published"},
			{ID: "ep-2", Name: "Fetch Market Pricing", Path: "/api/v1/pricing/quote", Method: "GET", Description: "Live market pricing quote lookup by ticker/ISIN", Status: "published"},
			{ID: "ep-3", Name: "Allocate Block Trade", Path: "/api/v1/oms/trades/allocate", Method: "POST", Description: "Calculates proportional share allocation across child accounts", Status: "published"},
			{ID: "ep-4", Name: "Generate Client Statement", Path: "/api/v1/reporting/statement/pdf", Method: "POST", Description: "Generates quarterly wealth management client PDF report", Status: "published"},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(endpoints)
}

func (h *DataPipelineHandler) GetWorkflows(w http.ResponseWriter, r *http.Request) {
	tenantID := h.getTenantID(r)

	type WorkflowSummary struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		TriggerType string `json:"trigger_type"`
		IsActive    bool   `json:"is_active"`
	}

	var workflows []WorkflowSummary
	if h.db != nil {
		_ = h.db.SelectContext(r.Context(), &workflows, `
			SELECT id::text, name, COALESCE(description, '') as description, 'event' as trigger_type, is_active
			FROM pipelines
			WHERE tenant_id = $1 AND is_active = true
			ORDER BY name
		`, tenantID)
	}

	if len(workflows) == 0 {
		workflows = []WorkflowSummary{
			{ID: "wf-1", Name: "Trade Reconciliation & Settlement Approval", Description: "Automated 4-eyes approval flow for large institutional blocks", TriggerType: "flow_builder", IsActive: true},
			{ID: "wf-2", Name: "Account Onboarding & KYC Screening", Description: "Flow Builder pipeline with sanctions screening and document verification", TriggerType: "flow_builder", IsActive: true},
			{ID: "wf-3", Name: "Portfolio Rebalancing & Order Routing", Description: "Runs mean-variance optimization and dispatches execution orders", TriggerType: "flow_builder", IsActive: true},
			{ID: "wf-4", Name: "Alternative Investment Capital Call Workflow", Description: "Multi-tier LP distribution and capital call validation", TriggerType: "flow_builder", IsActive: true},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workflows)
}

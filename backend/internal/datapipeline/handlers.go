package datapipeline

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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
		r.Post("/runs/{runId}/stream-token", h.CreateStreamToken)
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

// claimTenantIDFromRequest returns the tenant from JWT claims, or false if
// the request has no valid JWT. Used for endpoints that must 401 on missing
// auth rather than falling back to the Gold Copy tenant.
func claimTenantIDFromRequest(r *http.Request) (uuid.UUID, bool) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		return uuid.Nil, false
	}
	if claims.TenantID == "" {
		return uuid.Nil, false
	}
	parsed, err := uuid.Parse(claims.TenantID)
	if err != nil {
		return uuid.Nil, false
	}
	return parsed, true
}

func (h *DataPipelineHandler) ListPipelines(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("compact") == "true" {
		h.listPipelinesCompact(w, r)
		return
	}
	tenantID := h.getTenantID(r)

	if h.db == nil {
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
		list = []PipelineDefinition{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (h *DataPipelineHandler) listPipelinesCompact(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := claimTenantIDFromRequest(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if h.db == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}

	type compactPipeline struct {
		ID            uuid.UUID `json:"id" db:"id"`
		Name          string    `json:"name" db:"name"`
		SourceTable   string    `json:"source_table,omitempty" db:"source_table"`
		SinkLabel     string    `json:"sink_label,omitempty" db:"sink_label"`
	}

	query := `
		SELECT id, name,
		       COALESCE(target_entity, '') AS source_table,
		       COALESCE(mode, '') AS sink_label
		FROM data_pipeline_definitions
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY last_modified_at DESC
	`
	var list []compactPipeline
	err := h.db.SelectContext(r.Context(), &list, query, tenantID)
	if err != nil {
		http.Error(w, "query failed: "+err.Error(), http.StatusInternalServerError)
		return
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
	tenantID, ok := claimTenantIDFromRequest(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

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
	tenantID, ok := claimTenantIDFromRequest(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	runID := chi.URLParam(r, "runId")
	run, ok := h.engine.GetRunWithFallback(r.Context(), runID)
	if !ok {
		http.Error(w, "Run not found", http.StatusNotFound)
		return
	}

	if run.TenantID != tenantID {
		http.Error(w, "Run not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}

func (h *DataPipelineHandler) StreamTelemetrySSE(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runId")

	token := r.URL.Query().Get("token")

	if token != "" {
		validatedTenantID, ok := h.validateStreamToken(r.Context(), runID, token)
		if !ok {
			http.Error(w, "unauthorized or token invalid/expired", http.StatusUnauthorized)
			return
		}
		_ = validatedTenantID
	} else {
		if _, ok := claimTenantIDFromRequest(r); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	sendJSON := func(w http.ResponseWriter, data interface{}) {
		bytes, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", bytes)
		flusher.Flush()
	}

	if h.bus == nil {
		http.Error(w, "telemetry bus not configured", http.StatusServiceUnavailable)
		return
	}

	ch, cleanup := h.bus.Listen(runID)
	defer cleanup()

	if current, exists := h.engine.GetRunWithFallback(r.Context(), runID); exists {
		sendJSON(w, current)
		if current.Status == "completed" || current.Status == "failed" {
			return
		}
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case n, ok := <-ch:
			if !ok {
				return
			}
			if n.Run != nil {
				sendJSON(w, n.Run)
				if n.Run.Status == "completed" || n.Run.Status == "failed" {
					return
				}
			} else if n.Step != nil {
				if current, exists := h.engine.GetRunWithFallback(r.Context(), runID); exists {
					sendJSON(w, current)
				}
			}
		}
	}
}

func (h *DataPipelineHandler) CreateStreamToken(w http.ResponseWriter, r *http.Request) {
	runIDStr := chi.URLParam(r, "runId")
	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		http.Error(w, "invalid run_id", http.StatusBadRequest)
		return
	}

	tenantID, ok := claimTenantIDFromRequest(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		TenantID  string `json:"tenant_id"`
		ExpiresIn int    `json:"expires_in"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.TenantID != "" && req.TenantID != tenantID.String() {
		http.Error(w, "tenant_id mismatch", http.StatusForbidden)
		return
	}

	expiresIn := req.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 60
	}
	if expiresIn > 3600 {
		expiresIn = 3600
	}

	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}
	rawTokenHex := hex.EncodeToString(rawToken)
	hash := sha256.Sum256(rawToken)
	tokenHash := fmt.Sprintf("sha256:%x", hash)

	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

	_, err = h.db.ExecContext(r.Context(),
		`INSERT INTO pipeline_stream_tokens (run_id, tenant_id, token_hash, expires_at)
         VALUES ($1, $2, $3, $4)`,
		runID, tenantID, tokenHash, expiresAt)
	if err != nil {
		http.Error(w, "failed to store token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":      rawTokenHex,
		"expires_at": expiresAt.Format(time.RFC3339),
		"run_id":     runID.String(),
	})
}

// validateStreamToken atomically consumes a stream token for a specific run.
// Returns (tenantID, true) if the token is valid, unexpired, and unconsumed.
// Returns (uuid.Nil, false) if the token is invalid, expired, already used,
// or belongs to a different run. The token is consumed on success so it
// cannot be replayed.
func (h *DataPipelineHandler) validateStreamToken(ctx context.Context, runID, rawToken string) (uuid.UUID, bool) {
	if rawToken == "" {
		return uuid.Nil, false
	}

	var tenantID uuid.UUID
	err := h.db.QueryRowContext(ctx,
		`UPDATE pipeline_stream_tokens
         SET used_at = NOW()
         WHERE token_hash = 'sha256:' || encode(sha256(decode($1, 'hex')), 'hex')
           AND run_id = $2
           AND used_at IS NULL
           AND expires_at > NOW()
         RETURNING tenant_id`,
		rawToken, runID,
	).Scan(&tenantID)

	if err == sql.ErrNoRows {
		return uuid.Nil, false
	}
	if err != nil {
		return uuid.Nil, false
	}
	return tenantID, true
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

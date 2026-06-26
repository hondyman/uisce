package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hondyman/semlayer/backend/internal/catalog"
	"go.uber.org/zap"
)

// CatalogIngestionConfig holds configuration for catalog ingestion workers
type CatalogIngestionConfig struct {
	BootstrapServers string        // Kafka bootstrap servers
	GroupID          string        // Consumer group ID
	Topics           []string      // Topics to consume from
	BatchSize        int           // Batch size for catalog writes (default: 100)
	FlushInterval    time.Duration // Flush interval for batch writes (default: 30s)
	MaxRetries       int           // Max retries on failure (default: 3)
	TenantID         string        // Default tenant ID if not in event
}

// AuditIngestionWorker consumes audit events and writes them into the catalog graph
// It routes events to specific ingestors based on event type
type AuditIngestionWorker struct {
	eventChan     <-chan KafkaEventEnvelope
	catalogWriter catalog.Writer
	config        CatalogIngestionConfig
	logger        *zap.Logger
	mu            sync.RWMutex
	running       bool
	stopChan      chan struct{}
	wg            sync.WaitGroup
	eventBuffer   []KafkaEventEnvelope
	nodeBuffer    []catalog.CatalogNode
	edgeBuffer    []catalog.CatalogEdge
	lastFlushTime time.Time
	flushTicker   *time.Ticker
}

// NewAuditIngestionWorker creates a new audit ingestion worker
func NewAuditIngestionWorker(
	eventChan <-chan KafkaEventEnvelope,
	catalogWriter catalog.Writer,
	config CatalogIngestionConfig,
	logger *zap.Logger,
) *AuditIngestionWorker {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.FlushInterval == 0 {
		config.FlushInterval = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	return &AuditIngestionWorker{
		eventChan:     eventChan,
		catalogWriter: catalogWriter,
		config:        config,
		logger:        logger,
		stopChan:      make(chan struct{}),
		eventBuffer:   make([]KafkaEventEnvelope, 0, config.BatchSize),
		nodeBuffer:    make([]catalog.CatalogNode, 0, config.BatchSize*2),
		edgeBuffer:    make([]catalog.CatalogEdge, 0, config.BatchSize*3),
		lastFlushTime: time.Now(),
	}
}

// Start begins consuming and ingesting audit events
func (w *AuditIngestionWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("ingestion worker already running")
	}
	w.running = true
	w.mu.Unlock()

	w.flushTicker = time.NewTicker(w.config.FlushInterval)
	defer w.flushTicker.Stop()

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.run(ctx)
	}()

	w.logger.Info("audit ingestion worker started",
		zap.String("group_id", w.config.GroupID),
		zap.Strings("topics", w.config.Topics),
		zap.Int("batch_size", w.config.BatchSize),
	)

	return nil
}

// run is the main ingestion loop
func (w *AuditIngestionWorker) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("audit ingestion context cancelled")
			_ = w.flush(ctx)
			return

		case <-w.stopChan:
			w.logger.Info("audit ingestion stop signal received")
			_ = w.flush(ctx)
			return

		case <-w.flushTicker.C:
			if time.Since(w.lastFlushTime) >= w.config.FlushInterval {
				if err := w.flush(ctx); err != nil {
					w.logger.Error("flush error", zap.Error(err))
				}
			}

		case evt, ok := <-w.eventChan:
			if !ok {
				w.logger.Info("event channel closed")
				_ = w.flush(ctx)
				return
			}

			// Handle the event (route to appropriate ingestor)
			if err := w.handleEvent(ctx, evt); err != nil {
				w.logger.Error("failed to handle audit event",
					zap.String("event_id", evt.EventID),
					zap.String("event_type", evt.EventType),
					zap.Error(err),
				)
				// Continue processing despite errors
			}

			// Auto-flush if buffers are full
			if len(w.nodeBuffer) >= w.config.BatchSize || len(w.edgeBuffer) >= w.config.BatchSize {
				if err := w.flush(ctx); err != nil {
					w.logger.Error("auto-flush error", zap.Error(err))
				}
			}
		}
	}
}

// handleEvent routes an audit event to the appropriate ingestor
func (w *AuditIngestionWorker) handleEvent(ctx context.Context, evt KafkaEventEnvelope) error {
	switch evt.EventType {
	case "JOB_RUN_COMPLETED":
		var jobRun JobRunCompletedEvent
		if err := json.Unmarshal(evt.Payload, &jobRun); err != nil {
			return fmt.Errorf("failed to parse job run event: %w", err)
		}
		return w.ingestJobRun(ctx, evt, jobRun)

	case "DAG_RUN_COMPLETED":
		var dagRun DAGRunCompletedEvent
		if err := json.Unmarshal(evt.Payload, &dagRun); err != nil {
			return fmt.Errorf("failed to parse dag run event: %w", err)
		}
		return w.ingestDAGRun(ctx, evt, dagRun)

	case "CHANGESET_CREATED":
		var changeset ChangeSetCreatedEvent
		if err := json.Unmarshal(evt.Payload, &changeset); err != nil {
			return fmt.Errorf("failed to parse changeset event: %w", err)
		}
		return w.ingestChangeSet(ctx, evt, changeset)

	case "COMPLIANCE_VIOLATION": // Changed from COMPLIANCE_EVENT to match KE
		var violation ComplianceViolationEvent
		if err := json.Unmarshal(evt.Payload, &violation); err != nil {
			return fmt.Errorf("failed to parse compliance event: %w", err)
		}
		return w.ingestComplianceEvent(ctx, evt, violation)

	case "INCIDENT_CLUSTERED":
		var incident IncidentEvent
		if err := json.Unmarshal(evt.Payload, &incident); err != nil {
			return fmt.Errorf("failed to parse incident event: %w", err)
		}
		return w.ingestIncident(ctx, evt, incident)

	case "SEMANTIC_SNAPSHOT":
		var snapshot SemanticSnapshotEvent
		if err := json.Unmarshal(evt.Payload, &snapshot); err != nil {
			return fmt.Errorf("failed to parse semantic snapshot event: %w", err)
		}
		return w.ingestSemanticSnapshot(ctx, evt, snapshot)

	case "AI_SUGGESTION":
		var suggestion AISuggestionEvent
		if err := json.Unmarshal(evt.Payload, &suggestion); err != nil {
			return fmt.Errorf("failed to parse AI suggestion event: %w", err)
		}
		return w.ingestAISuggestion(ctx, evt, suggestion)

	default:
		w.logger.Warn("unknown event type", zap.String("event_type", evt.EventType))
		return nil
	}
}

// ingestJobRun converts a job run event into catalog nodes and edges
func (w *AuditIngestionWorker) ingestJobRun(ctx context.Context, evt KafkaEventEnvelope, run JobRunCompletedEvent) error {
	nodeID := fmt.Sprintf("job_run:%s", run.RunID)

	// Create node for the job run
	node := catalog.CatalogNode{
		ID:            nodeID,
		NodeType:      "job_run",
		QualifiedPath: fmt.Sprintf("audit/job_run/%s", run.RunID),
		TenantID:      run.TenantID,
		DatasourceID:  run.TenantID, // Use tenant as datasource for now
		Properties: map[string]any{
			"run_id":         run.RunID,
			"job_id":         run.JobID,
			"status":         run.Status,
			"start_ts":       run.StartTS.Unix(),
			"end_ts":         run.EndTS.Unix(),
			"error_message":  run.ErrorMessage,
			"semantic_terms": run.SemanticTerms,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	w.mu.Lock()
	w.nodeBuffer = append(w.nodeBuffer, node)
	w.mu.Unlock()

	// Edge: JOB_RUN -> JOB (runs_job)
	runJobEdge := catalog.CatalogEdge{
		ID:           fmt.Sprintf("edge:runs_job:%s", run.RunID),
		EdgeType:     "runs_job",
		FromNode:     nodeID,
		ToNode:       fmt.Sprintf("job:%s", run.JobID),
		TenantID:     run.TenantID,
		DatasourceID: run.TenantID,
		Properties: map[string]any{
			"executed_at": run.StartTS.Unix(),
		},
		CreatedAt: time.Now(),
	}

	w.mu.Lock()
	w.edgeBuffer = append(w.edgeBuffer, runJobEdge)
	w.mu.Unlock()

	// Create edges to semantic terms (has_semantic_context)
	for _, term := range run.SemanticTerms {
		termEdge := catalog.CatalogEdge{
			ID:           fmt.Sprintf("edge:has_semantic_context:%s:%s", run.RunID, term),
			EdgeType:     "has_semantic_context",
			FromNode:     nodeID,
			ToNode:       fmt.Sprintf("semantic_term:%s", term),
			TenantID:     run.TenantID,
			DatasourceID: run.TenantID,
			CreatedAt:    time.Now(),
		}
		w.mu.Lock()
		w.edgeBuffer = append(w.edgeBuffer, termEdge)
		w.mu.Unlock()
	}

	// Create edge to tenant (has_tenant)
	tenantEdge := catalog.CatalogEdge{
		ID:           fmt.Sprintf("edge:has_tenant:%s", run.RunID),
		EdgeType:     "has_tenant",
		FromNode:     nodeID,
		ToNode:       fmt.Sprintf("tenant:%s", run.TenantID),
		TenantID:     run.TenantID,
		DatasourceID: run.TenantID,
		CreatedAt:    time.Now(),
	}

	w.mu.Lock()
	w.edgeBuffer = append(w.edgeBuffer, tenantEdge)
	w.mu.Unlock()

	return nil
}

// ingestDAGRun converts a DAG run event into catalog nodes and edges
func (w *AuditIngestionWorker) ingestDAGRun(ctx context.Context, evt KafkaEventEnvelope, run DAGRunCompletedEvent) error {
	nodeID := fmt.Sprintf("dag_run:%s", run.DagRunID)

	node := catalog.CatalogNode{
		ID:            nodeID,
		NodeType:      "dag_run",
		QualifiedPath: fmt.Sprintf("audit/dag_run/%s", run.DagRunID),
		TenantID:      run.TenantID,
		DatasourceID:  run.TenantID,
		Properties: map[string]any{
			"run_id":         run.DagRunID,
			"dag_id":         run.DagID,
			"status":         run.Status,
			"start_ts":       run.StartTS.Unix(),
			"end_ts":         run.EndTS.Unix(),
			"semantic_terms": run.SemanticTerms,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	w.mu.Lock()
	w.nodeBuffer = append(w.nodeBuffer, node)
	w.mu.Unlock()

	// Edge: DAG_RUN -> DAG (runs_dag)
	runDAGEdge := catalog.CatalogEdge{
		ID:           fmt.Sprintf("edge:runs_dag:%s", run.DagRunID),
		EdgeType:     "runs_dag",
		FromNode:     nodeID,
		ToNode:       fmt.Sprintf("dag:%s", run.DagID),
		TenantID:     run.TenantID,
		DatasourceID: run.TenantID,
		Properties: map[string]any{
			"executed_at": run.StartTS.Unix(),
		},
		CreatedAt: time.Now(),
	}

	w.mu.Lock()
	w.edgeBuffer = append(w.edgeBuffer, runDAGEdge)
	w.mu.Unlock()

	// Create edges to semantic terms
	for _, term := range run.SemanticTerms {
		termEdge := catalog.CatalogEdge{
			ID:           fmt.Sprintf("edge:has_semantic_context:%s:%s", run.DagRunID, term),
			EdgeType:     "has_semantic_context",
			FromNode:     nodeID,
			ToNode:       fmt.Sprintf("semantic_term:%s", term),
			TenantID:     run.TenantID,
			DatasourceID: run.TenantID,
			CreatedAt:    time.Now(),
		}
		w.mu.Lock()
		w.edgeBuffer = append(w.edgeBuffer, termEdge)
		w.mu.Unlock()
	}

	// Edge to tenant
	tenantEdge := catalog.CatalogEdge{
		ID:           fmt.Sprintf("edge:has_tenant:%s", run.DagRunID),
		EdgeType:     "has_tenant",
		FromNode:     nodeID,
		ToNode:       fmt.Sprintf("tenant:%s", run.TenantID),
		TenantID:     run.TenantID,
		DatasourceID: run.TenantID,
		CreatedAt:    time.Now(),
	}

	w.mu.Lock()
	w.edgeBuffer = append(w.edgeBuffer, tenantEdge)
	w.mu.Unlock()

	return nil
}

// ingestChangeSet converts a changeset event into catalog nodes and edges
func (w *AuditIngestionWorker) ingestChangeSet(ctx context.Context, evt KafkaEventEnvelope, cs ChangeSetCreatedEvent) error {
	nodeID := fmt.Sprintf("changeset_event:%s", cs.ChangesetID)

	node := catalog.CatalogNode{
		ID:            nodeID,
		NodeType:      "changeset_event",
		QualifiedPath: fmt.Sprintf("audit/changeset/%s", cs.ChangesetID),
		TenantID:      cs.TenantID,
		DatasourceID:  cs.TenantID,
		Properties: map[string]any{
			"changeset_id":   cs.ChangesetID,
			"status":         cs.Status,
			"created_at":     cs.CreatedAt.Unix(),
			"description":    cs.Description,
			"impacted_terms": cs.SemanticImpact,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	w.mu.Lock()
	w.nodeBuffer = append(w.nodeBuffer, node)
	w.mu.Unlock()

	// Create edges to impacted entities (has_impact_on)
	// Using SemanticImpact if ImpactedEntities is empty
	for _, entity := range cs.ImpactedEntities {
		impactEdge := catalog.CatalogEdge{
			ID:           fmt.Sprintf("edge:has_impact_on:%s:%s", cs.ChangesetID, entity.ID),
			EdgeType:     "has_impact_on",
			FromNode:     nodeID,
			ToNode:       entity.NodeID,
			TenantID:     cs.TenantID,
			DatasourceID: cs.TenantID,
			Properties:   map[string]any{},
			CreatedAt:    time.Now(),
		}
		w.mu.Lock()
		w.edgeBuffer = append(w.edgeBuffer, impactEdge)
		w.mu.Unlock()
	}

	// Edge to tenant
	tenantEdge := catalog.CatalogEdge{
		ID:           fmt.Sprintf("edge:has_tenant:%s", cs.ChangesetID),
		EdgeType:     "has_tenant",
		FromNode:     nodeID,
		ToNode:       fmt.Sprintf("tenant:%s", cs.TenantID),
		TenantID:     cs.TenantID,
		DatasourceID: cs.TenantID,
		CreatedAt:    time.Now(),
	}

	w.mu.Lock()
	w.edgeBuffer = append(w.edgeBuffer, tenantEdge)
	w.mu.Unlock()

	return nil
}

// ingestComplianceEvent converts a compliance event into catalog nodes and edges
func (w *AuditIngestionWorker) ingestComplianceEvent(ctx context.Context, evt KafkaEventEnvelope, vl ComplianceViolationEvent) error {
	nodeID := fmt.Sprintf("compliance_event:%s", vl.ViolationID)

	node := catalog.CatalogNode{
		ID:            nodeID,
		NodeType:      "compliance_event",
		QualifiedPath: fmt.Sprintf("audit/compliance/%s", vl.ViolationID),
		TenantID:      vl.TenantID,
		DatasourceID:  vl.TenantID,
		Properties: map[string]any{
			"violation_id":   vl.ViolationID,
			"violation_type": vl.ViolationType,
			"severity":       vl.Severity,
			"pii_exposed":    vl.PIIExposed,
			"narrative":      vl.Narrative,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	w.mu.Lock()
	w.nodeBuffer = append(w.nodeBuffer, node)
	w.mu.Unlock()

	// Create edges to impacted entities (has_compliance_context)
	for _, entity := range vl.ImpactedEntities {
		contextEdge := catalog.CatalogEdge{
			ID:           fmt.Sprintf("edge:has_compliance_context:%s:%s", vl.ViolationID, entity.ID),
			EdgeType:     "has_compliance_context",
			FromNode:     nodeID,
			ToNode:       entity.NodeID,
			TenantID:     vl.TenantID,
			DatasourceID: vl.TenantID,
			Properties: map[string]any{
				"violation_type": vl.ViolationType,
				"severity":       vl.Severity,
				"status":         vl.Status,
			},
			CreatedAt: time.Now(),
		}
		w.mu.Lock()
		w.edgeBuffer = append(w.edgeBuffer, contextEdge)
		w.mu.Unlock()
	}

	// Edge to tenant
	tenantEdge := catalog.CatalogEdge{
		ID:           fmt.Sprintf("edge:has_tenant:%s", vl.ViolationID),
		EdgeType:     "has_tenant",
		FromNode:     nodeID,
		ToNode:       fmt.Sprintf("tenant:%s", vl.TenantID),
		TenantID:     vl.TenantID,
		DatasourceID: vl.TenantID,
		CreatedAt:    time.Now(),
	}

	w.mu.Lock()
	w.edgeBuffer = append(w.edgeBuffer, tenantEdge)
	w.mu.Unlock()

	return nil
}

// ingestIncident converts an incident event into catalog nodes and edges
func (w *AuditIngestionWorker) ingestIncident(ctx context.Context, evt KafkaEventEnvelope, incident IncidentEvent) error {
	nodeID := fmt.Sprintf("incident:%s", incident.IncidentID)

	node := catalog.CatalogNode{
		ID:            nodeID,
		NodeType:      "incident",
		QualifiedPath: fmt.Sprintf("audit/incident/%s", incident.IncidentID),
		TenantID:      incident.TenantID,
		DatasourceID:  incident.TenantID,
		Properties: map[string]any{
			"incident_id":     incident.IncidentID,
			"status":          incident.Status,
			"severity":        incident.Severity,
			"title":           incident.Title,
			"description":     incident.Description,
			"detected_at":     incident.DetectedAt.Unix(),
			"resolved_at":     incident.ResolvedAt.Unix(),
			"affected_terms":  incident.AffectedTerms,
			"cause_event_ids": incident.CauseEventIDs,
			"blast_radius":    incident.BlastRadius,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	w.mu.Lock()
	w.nodeBuffer = append(w.nodeBuffer, node)
	w.mu.Unlock()

	// Create edges to cause events (causes)
	for _, causeID := range incident.CauseEventIDs {
		causesEdge := catalog.CatalogEdge{
			ID:           fmt.Sprintf("edge:causes:%s:%s", incident.IncidentID, causeID),
			EdgeType:     "causes",
			FromNode:     nodeID,
			ToNode:       causeID, // Should be a job_run or dag_run node
			TenantID:     incident.TenantID,
			DatasourceID: incident.TenantID,
			CreatedAt:    time.Now(),
		}
		w.mu.Lock()
		w.edgeBuffer = append(w.edgeBuffer, causesEdge)
		w.mu.Unlock()
	}

	// Create edges to affected terms (has_semantic_context)
	for _, term := range incident.AffectedTerms {
		contextEdge := catalog.CatalogEdge{
			ID:           fmt.Sprintf("edge:has_semantic_context:%s:%s", incident.IncidentID, term),
			EdgeType:     "has_semantic_context",
			FromNode:     nodeID,
			ToNode:       fmt.Sprintf("semantic_term:%s", term),
			TenantID:     incident.TenantID,
			DatasourceID: incident.TenantID,
			CreatedAt:    time.Now(),
		}
		w.mu.Lock()
		w.edgeBuffer = append(w.edgeBuffer, contextEdge)
		w.mu.Unlock()
	}

	// Edge to tenant
	tenantEdge := catalog.CatalogEdge{
		ID:           fmt.Sprintf("edge:has_tenant:%s", incident.IncidentID),
		EdgeType:     "has_tenant",
		FromNode:     nodeID,
		ToNode:       fmt.Sprintf("tenant:%s", incident.TenantID),
		TenantID:     incident.TenantID,
		DatasourceID: incident.TenantID,
		CreatedAt:    time.Now(),
	}

	w.mu.Lock()
	w.edgeBuffer = append(w.edgeBuffer, tenantEdge)
	w.mu.Unlock()

	return nil
}

// ingestSemanticSnapshot converts a semantic snapshot event into catalog nodes
func (w *AuditIngestionWorker) ingestSemanticSnapshot(ctx context.Context, evt KafkaEventEnvelope, ss SemanticSnapshotEvent) error {
	nodeID := fmt.Sprintf("semantic_snapshot:%s", ss.SnapshotID)

	node := catalog.CatalogNode{
		ID:            nodeID,
		NodeType:      "semantic_snapshot",
		QualifiedPath: fmt.Sprintf("audit/semantic_snapshot/%s", ss.SnapshotID),
		TenantID:      ss.TenantID,
		DatasourceID:  ss.TenantID,
		Properties: map[string]any{
			"snapshot_id":      ss.SnapshotID,
			"semantic_term_id": ss.SemanticTermID,
			"version":          ss.Version,
			"definition":       ss.Definition,
			// Include region for region-scoped snapshots if provided
			"region": ss.Region,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	w.mu.Lock()
	w.nodeBuffer = append(w.nodeBuffer, node)
	w.mu.Unlock()

	// Edge to semantic term (event_of)
	if ss.SemanticTermID != "" {
		termEdge := catalog.CatalogEdge{
			ID:           fmt.Sprintf("edge:event_of:%s", ss.SnapshotID),
			EdgeType:     "event_of",
			FromNode:     nodeID,
			ToNode:       fmt.Sprintf("semantic_term:%s", ss.SemanticTermID),
			TenantID:     ss.TenantID,
			DatasourceID: ss.TenantID,
			Properties: map[string]any{
				"version":    ss.Version,
				"definition": ss.Definition,
			},
			CreatedAt: time.Now(),
		}

		w.mu.Lock()
		w.edgeBuffer = append(w.edgeBuffer, termEdge)
		w.mu.Unlock()
	}

	// Edge to business term if present
	if ss.BusinessTermID != "" {
		businessEdge := catalog.CatalogEdge{
			ID:           fmt.Sprintf("edge:has_semantic_context:%s:%s", ss.SnapshotID, ss.BusinessTermID),
			EdgeType:     "has_semantic_context",
			FromNode:     nodeID,
			ToNode:       fmt.Sprintf("business_term:%s", ss.BusinessTermID),
			TenantID:     ss.TenantID,
			DatasourceID: ss.TenantID,
			CreatedAt:    time.Now(),
		}

		w.mu.Lock()
		w.edgeBuffer = append(w.edgeBuffer, businessEdge)
		w.mu.Unlock()
	}

	// Edge to tenant
	tenantEdge := catalog.CatalogEdge{
		ID:           fmt.Sprintf("edge:has_tenant:%s", ss.SnapshotID),
		EdgeType:     "has_tenant",
		FromNode:     nodeID,
		ToNode:       fmt.Sprintf("tenant:%s", ss.TenantID),
		TenantID:     ss.TenantID,
		DatasourceID: ss.TenantID,
		CreatedAt:    time.Now(),
	}

	w.mu.Lock()
	w.edgeBuffer = append(w.edgeBuffer, tenantEdge)
	w.mu.Unlock()

	return nil
}

// ingestAISuggestion converts an AI suggestion event into catalog nodes and edges
func (w *AuditIngestionWorker) ingestAISuggestion(ctx context.Context, evt KafkaEventEnvelope, as AISuggestionEvent) error {
	nodeID := fmt.Sprintf("ai_suggestion:%s", as.SuggestionID)

	node := catalog.CatalogNode{
		ID:            nodeID,
		NodeType:      "ai_suggestion",
		QualifiedPath: fmt.Sprintf("audit/ai_suggestion/%s", as.SuggestionID),
		TenantID:      as.TenantID,
		DatasourceID:  as.TenantID,
		Properties: map[string]any{
			"suggestion_id": as.SuggestionID,
			"type":          as.SuggestionType, // or Type
			"narrative":     as.Narrative,
			"confidence":    as.Confidence,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	w.mu.Lock()
	w.nodeBuffer = append(w.nodeBuffer, node)
	w.mu.Unlock()

	// Edge to related audit event (has_ai_narrative)
	if as.RelatedEventID != "" {
		relatedEdge := catalog.CatalogEdge{
			ID:           fmt.Sprintf("edge:has_ai_narrative:%s", as.SuggestionID),
			EdgeType:     "has_ai_narrative",
			FromNode:     as.RelatedEventID,
			ToNode:       nodeID,
			TenantID:     as.TenantID,
			DatasourceID: as.TenantID,
			Properties: map[string]any{
				"confidence":         as.Confidence,
				"suggestion_type":    as.SuggestionType,
				"narrative":          as.Narrative,
				"generated_by":       as.GeneratedBy,
				"related_event_type": as.RelatedEventType,
			},
			CreatedAt: time.Now(),
		}

		w.mu.Lock()
		w.edgeBuffer = append(w.edgeBuffer, relatedEdge)
		w.mu.Unlock()
	}

	// Edge to tenant
	tenantEdge := catalog.CatalogEdge{
		ID:           fmt.Sprintf("edge:has_tenant:%s", as.SuggestionID),
		EdgeType:     "has_tenant",
		FromNode:     nodeID,
		ToNode:       fmt.Sprintf("tenant:%s", as.TenantID),
		TenantID:     as.TenantID,
		DatasourceID: as.TenantID,
		CreatedAt:    time.Now(),
	}

	w.mu.Lock()
	w.edgeBuffer = append(w.edgeBuffer, tenantEdge)
	w.mu.Unlock()

	return nil
}

// flush writes buffered nodes and edges to the catalog
func (w *AuditIngestionWorker) flush(ctx context.Context) error {
	w.mu.Lock()
	nodeCount := len(w.nodeBuffer)
	edgeCount := len(w.edgeBuffer)

	if nodeCount == 0 && edgeCount == 0 {
		w.mu.Unlock()
		return nil
	}

	nodesToWrite := make([]catalog.CatalogNode, nodeCount)
	copy(nodesToWrite, w.nodeBuffer)
	w.nodeBuffer = w.nodeBuffer[:0]

	edgesToWrite := make([]catalog.CatalogEdge, edgeCount)
	copy(edgesToWrite, w.edgeBuffer)
	w.edgeBuffer = w.edgeBuffer[:0]

	w.lastFlushTime = time.Now()
	w.mu.Unlock()

	// Write nodes
	if len(nodesToWrite) > 0 {
		if err := w.catalogWriter.CreateNodes(ctx, nodesToWrite); err != nil {
			w.logger.Error("failed to batch create nodes", zap.Error(err), zap.Int("count", len(nodesToWrite)))
			return err
		}
		w.logger.Debug("flushed audit nodes", zap.Int("count", len(nodesToWrite)))
	}

	// Write edges
	if len(edgesToWrite) > 0 {
		if err := w.catalogWriter.CreateEdges(ctx, edgesToWrite); err != nil {
			w.logger.Error("failed to batch create edges", zap.Error(err), zap.Int("count", len(edgesToWrite)))
			return err
		}
		w.logger.Debug("flushed audit edges", zap.Int("count", len(edgesToWrite)))
	}

	return nil
}

// Stop stops the ingestion worker gracefully
func (w *AuditIngestionWorker) Stop(ctx context.Context) error {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = false
	w.mu.Unlock()

	// Signal stop
	close(w.stopChan)

	// Wait for goroutine to finish (with timeout)
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
		return fmt.Errorf("audit ingestion worker shutdown timeout")
	}
}

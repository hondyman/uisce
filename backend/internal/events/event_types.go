package events

import (
	"time"
)

// EventType represents different types of events in the system
type EventType string

const (
	// API Endpoint Events
	APIEndpointCreated   EventType = "api.endpoint.created"
	APIEndpointUpdated   EventType = "api.endpoint.updated"
	APIEndpointDeleted   EventType = "api.endpoint.deleted"
	APIEndpointActivated EventType = "api.endpoint.activated"

	// Mapping Events
	EntityMappingCreated     EventType = "api.entity_mapping.created"
	EntityMappingDeleted     EventType = "api.entity_mapping.deleted"
	DatasourceMappingCreated EventType = "api.datasource_mapping.created"
	DatasourceMappingDeleted EventType = "api.datasource_mapping.deleted"

	// Catalog Node Events
	CatalogNodeCreated EventType = "catalog.node.created"
	CatalogNodeUpdated EventType = "catalog.node.updated"
	CatalogNodeDeleted EventType = "catalog.node.deleted"

	// Catalog Edge Events
	CatalogEdgeCreated EventType = "catalog.edge.created"
	CatalogEdgeDeleted EventType = "catalog.edge.deleted"

	// UMA Account Events
	UMAAccountCreated EventType = "uma.account.created"
	UMAAccountUpdated EventType = "uma.account.updated"

	// UMA Rebalance Events
	RebalanceRequested        EventType = "uma.rebalance.requested"
	RebalancePlanGenerated    EventType = "uma.rebalance.plan.generated"
	RebalancePlanApproved     EventType = "uma.rebalance.plan.approved"
	RebalanceExecutionStarted EventType = "uma.rebalance.execution.started"
	RebalanceTradeExecuted    EventType = "uma.rebalance.trade.executed"
	RebalanceCompleted        EventType = "uma.rebalance.completed"
	RebalanceFailed           EventType = "uma.rebalance.failed"

	// UMA Sleeve Events
	SleeveAdjusted      EventType = "uma.sleeve.adjusted"
	SleeveDriftDetected EventType = "uma.sleeve.drift.detected"

	// Tax Harvesting Events
	TaxHarvestSimulated EventType = "uma.tax.harvest.simulated"
	TaxHarvestExecuted  EventType = "uma.tax.harvest.executed"

	// Custodial Events
	CustodianSynced EventType = "uma.custodian.synced"
	HoldingsUpdated EventType = "uma.holdings.updated"

	// Gold Copy Events
	GoldCopyConnectionChanged EventType = "gold_copy.connection.changed"

	// Compliance Events
	SemanticTermComplianceUpdated EventType = "compliance.semantic_term.updated"
	BusinessTermComplianceUpdated EventType = "compliance.business_term.updated"
	ComplianceViolationDetected   EventType = "compliance.violation.detected"

	// Phase 3.4: WebSocket Real-Time Events (Incident, RCA, Actions, Propagation)
	EventTypeIncidentDetected    EventType = "incident.detected"
	EventTypeIncidentUpdated     EventType = "incident.updated"
	EventTypeIncidentResolved    EventType = "incident.resolved"
	EventTypeRCAStarted          EventType = "rca.started"
	EventTypeRCACompleted        EventType = "rca.completed"
	EventTypeRCAResultsAvailable EventType = "rca.results"
	EventTypeActionPlanned       EventType = "action.planned"
	EventTypeActionStarted       EventType = "action.started"
	EventTypeActionCompleted     EventType = "action.completed"
	EventTypeActionFailed        EventType = "action.failed"
	EventTypeRegionFailover      EventType = "region.failover"
	EventTypePropagationDetected EventType = "propagation.detected"
	EventTypePropagationBlocked  EventType = "propagation.blocked"
)

// APIEndpointEvent represents an API endpoint lifecycle event
type APIEndpointEvent struct {
	EventID      string                 `json:"event_id"`
	EventType    EventType              `json:"event_type"`
	TenantID     string                 `json:"tenant_id"`
	DatasourceID string                 `json:"datasource_id,omitempty"`
	EndpointID   string                 `json:"endpoint_id"`
	Endpoint     map[string]interface{} `json:"endpoint"`
	Timestamp    time.Time              `json:"timestamp"`
	UserID       *string                `json:"user_id,omitempty"`
	TraceID      string                 `json:"trace_id,omitempty"`
}

// EntityMappingEvent represents a mapping lifecycle event
type EntityMappingEvent struct {
	EventID          string    `json:"event_id"`
	EventType        EventType `json:"event_type"`
	TenantID         string    `json:"tenant_id"`
	APIEndpointID    string    `json:"api_endpoint_id"`
	EntityID         string    `json:"entity_id"`
	RelationshipType string    `json:"relationship_type"`
	Timestamp        time.Time `json:"timestamp"`
	UserID           *string   `json:"user_id,omitempty"`
}

// DatasourceMappingEvent represents a datasource mapping event
type DatasourceMappingEvent struct {
	EventID          string    `json:"event_id"`
	EventType        EventType `json:"event_type"`
	TenantID         string    `json:"tenant_id"`
	APIEndpointID    string    `json:"api_endpoint_id"`
	DatasourceID     string    `json:"datasource_id"`
	RelationshipType string    `json:"relationship_type"`
	Timestamp        time.Time `json:"timestamp"`
	UserID           *string   `json:"user_id,omitempty"`
}

// CatalogNodeEvent represents a catalog node event
type CatalogNodeEvent struct {
	EventID   string                 `json:"event_id"`
	EventType EventType              `json:"event_type"`
	TenantID  string                 `json:"tenant_id"`
	NodeID    string                 `json:"node_id"`
	NodeType  string                 `json:"node_type"`
	Node      map[string]interface{} `json:"node"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    *string                `json:"user_id,omitempty"`
}

// CatalogEdgeEvent represents a catalog edge event
type CatalogEdgeEvent struct {
	EventID          string    `json:"event_id"`
	EventType        EventType `json:"event_type"`
	TenantID         string    `json:"tenant_id"`
	EdgeID           string    `json:"edge_id"`
	SourceNodeID     string    `json:"source_node_id"`
	TargetNodeID     string    `json:"target_node_id"`
	RelationshipType string    `json:"relationship_type"`
	Timestamp        time.Time `json:"timestamp"`
	UserID           *string   `json:"user_id,omitempty"`
}

// EventMetadata contains metadata for event tracking
type EventMetadata struct {
	CorrelationID string            `json:"correlation_id"`
	CausalID      string            `json:"causal_id,omitempty"`
	Version       int               `json:"version"`
	Tags          map[string]string `json:"tags,omitempty"`
	Idempotency   string            `json:"idempotency_key,omitempty"`
}

// DomainEvent is the base interface for all domain events
type DomainEvent interface {
	GetEventID() string
	GetEventType() EventType
	GetTenantID() string
	GetTimestamp() time.Time
	GetUserID() *string
}

// Implement DomainEvent for APIEndpointEvent
func (e *APIEndpointEvent) GetEventID() string      { return e.EventID }
func (e *APIEndpointEvent) GetEventType() EventType { return e.EventType }
func (e *APIEndpointEvent) GetTenantID() string     { return e.TenantID }
func (e *APIEndpointEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *APIEndpointEvent) GetUserID() *string      { return e.UserID }

// Implement DomainEvent for EntityMappingEvent
func (e *EntityMappingEvent) GetEventID() string      { return e.EventID }
func (e *EntityMappingEvent) GetEventType() EventType { return e.EventType }
func (e *EntityMappingEvent) GetTenantID() string     { return e.TenantID }
func (e *EntityMappingEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *EntityMappingEvent) GetUserID() *string      { return e.UserID }

// Implement DomainEvent for DatasourceMappingEvent
func (e *DatasourceMappingEvent) GetEventID() string      { return e.EventID }
func (e *DatasourceMappingEvent) GetEventType() EventType { return e.EventType }
func (e *DatasourceMappingEvent) GetTenantID() string     { return e.TenantID }
func (e *DatasourceMappingEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *DatasourceMappingEvent) GetUserID() *string      { return e.UserID }

// Implement DomainEvent for CatalogNodeEvent
func (e *CatalogNodeEvent) GetEventID() string      { return e.EventID }
func (e *CatalogNodeEvent) GetEventType() EventType { return e.EventType }
func (e *CatalogNodeEvent) GetTenantID() string     { return e.TenantID }
func (e *CatalogNodeEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *CatalogNodeEvent) GetUserID() *string      { return e.UserID }

// Implement DomainEvent for CatalogEdgeEvent
func (e *CatalogEdgeEvent) GetEventID() string      { return e.EventID }
func (e *CatalogEdgeEvent) GetEventType() EventType { return e.EventType }
func (e *CatalogEdgeEvent) GetTenantID() string     { return e.TenantID }
func (e *CatalogEdgeEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *CatalogEdgeEvent) GetUserID() *string      { return e.UserID }

// ============================================================================
// UMA EVENTS
// ============================================================================

// UMARebalanceRequestedEvent emitted when a rebalance is requested
type UMARebalanceRequestedEvent struct {
	EventID      string                 `json:"event_id"`
	EventType    EventType              `json:"event_type"`
	TenantID     string                 `json:"tenant_id"`
	DatasourceID string                 `json:"datasource_id"`
	RequestID    string                 `json:"request_id"`
	UMAAccountID string                 `json:"uma_account_id"`
	RequestType  string                 `json:"request_type"` // drift, manual, scheduled
	Reason       string                 `json:"reason"`
	InitiatedBy  string                 `json:"initiated_by"`
	Timestamp    time.Time              `json:"timestamp"`
	UserID       *string                `json:"user_id,omitempty"`
	TraceID      string                 `json:"trace_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

func (e *UMARebalanceRequestedEvent) GetEventID() string      { return e.EventID }
func (e *UMARebalanceRequestedEvent) GetEventType() EventType { return e.EventType }
func (e *UMARebalanceRequestedEvent) GetTenantID() string     { return e.TenantID }
func (e *UMARebalanceRequestedEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *UMARebalanceRequestedEvent) GetUserID() *string      { return e.UserID }

// UMARebalancePlanGeneratedEvent emitted when a rebalance plan is generated
type UMARebalancePlanGeneratedEvent struct {
	EventID        string                 `json:"event_id"`
	EventType      EventType              `json:"event_type"`
	TenantID       string                 `json:"tenant_id"`
	DatasourceID   string                 `json:"datasource_id"`
	RequestID      string                 `json:"request_id"`
	PlanID         string                 `json:"plan_id"`
	UMAAccountID   string                 `json:"uma_account_id"`
	TradeCount     int                    `json:"trade_count"`
	TotalTaxImpact float64                `json:"total_tax_impact"`
	TotalCost      float64                `json:"total_cost"`
	Timestamp      time.Time              `json:"timestamp"`
	UserID         *string                `json:"user_id,omitempty"`
	TraceID        string                 `json:"trace_id,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

func (e *UMARebalancePlanGeneratedEvent) GetEventID() string      { return e.EventID }
func (e *UMARebalancePlanGeneratedEvent) GetEventType() EventType { return e.EventType }
func (e *UMARebalancePlanGeneratedEvent) GetTenantID() string     { return e.TenantID }
func (e *UMARebalancePlanGeneratedEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *UMARebalancePlanGeneratedEvent) GetUserID() *string      { return e.UserID }

// UMARebalancePlanApprovedEvent emitted when a plan is approved
type UMARebalancePlanApprovedEvent struct {
	EventID      string                 `json:"event_id"`
	EventType    EventType              `json:"event_type"`
	TenantID     string                 `json:"tenant_id"`
	DatasourceID string                 `json:"datasource_id"`
	PlanID       string                 `json:"plan_id"`
	UMAAccountID string                 `json:"uma_account_id"`
	ApprovedBy   string                 `json:"approved_by"`
	Timestamp    time.Time              `json:"timestamp"`
	UserID       *string                `json:"user_id,omitempty"`
	TraceID      string                 `json:"trace_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

func (e *UMARebalancePlanApprovedEvent) GetEventID() string      { return e.EventID }
func (e *UMARebalancePlanApprovedEvent) GetEventType() EventType { return e.EventType }
func (e *UMARebalancePlanApprovedEvent) GetTenantID() string     { return e.TenantID }
func (e *UMARebalancePlanApprovedEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *UMARebalancePlanApprovedEvent) GetUserID() *string      { return e.UserID }

// UMARebalanceExecutionStartedEvent emitted when execution begins
type UMARebalanceExecutionStartedEvent struct {
	EventID      string                 `json:"event_id"`
	EventType    EventType              `json:"event_type"`
	TenantID     string                 `json:"tenant_id"`
	DatasourceID string                 `json:"datasource_id"`
	PlanID       string                 `json:"plan_id"`
	UMAAccountID string                 `json:"uma_account_id"`
	Timestamp    time.Time              `json:"timestamp"`
	UserID       *string                `json:"user_id,omitempty"`
	TraceID      string                 `json:"trace_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

func (e *UMARebalanceExecutionStartedEvent) GetEventID() string      { return e.EventID }
func (e *UMARebalanceExecutionStartedEvent) GetEventType() EventType { return e.EventType }
func (e *UMARebalanceExecutionStartedEvent) GetTenantID() string     { return e.TenantID }
func (e *UMARebalanceExecutionStartedEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *UMARebalanceExecutionStartedEvent) GetUserID() *string      { return e.UserID }

// UMARebalanceCompletedEvent emitted when rebalance completes
type UMARebalanceCompletedEvent struct {
	EventID             string                 `json:"event_id"`
	EventType           EventType              `json:"event_type"`
	TenantID            string                 `json:"tenant_id"`
	DatasourceID        string                 `json:"datasource_id"`
	PlanID              string                 `json:"plan_id"`
	UMAAccountID        string                 `json:"uma_account_id"`
	CompletedTradeCount int                    `json:"completed_trade_count"`
	FailedTradeCount    int                    `json:"failed_trade_count"`
	TotalExecutionCost  float64                `json:"total_execution_cost"`
	ActualTaxImpact     float64                `json:"actual_tax_impact"`
	Timestamp           time.Time              `json:"timestamp"`
	UserID              *string                `json:"user_id,omitempty"`
	TraceID             string                 `json:"trace_id,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

func (e *UMARebalanceCompletedEvent) GetEventID() string      { return e.EventID }
func (e *UMARebalanceCompletedEvent) GetEventType() EventType { return e.EventType }
func (e *UMARebalanceCompletedEvent) GetTenantID() string     { return e.TenantID }
func (e *UMARebalanceCompletedEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *UMARebalanceCompletedEvent) GetUserID() *string      { return e.UserID }

// SleeveDriftDetectedEvent emitted when drift exceeds threshold
type SleeveDriftDetectedEvent struct {
	EventID           string                 `json:"event_id"`
	EventType         EventType              `json:"event_type"`
	TenantID          string                 `json:"tenant_id"`
	DatasourceID      string                 `json:"datasource_id"`
	UMAAccountID      string                 `json:"uma_account_id"`
	SleeveID          string                 `json:"sleeve_id"`
	SleeveType        string                 `json:"sleeve_type"`
	TargetAllocation  float64                `json:"target_allocation"`
	CurrentAllocation float64                `json:"current_allocation"`
	DriftPercent      float64                `json:"drift_percent"`
	Timestamp         time.Time              `json:"timestamp"`
	UserID            *string                `json:"user_id,omitempty"`
	TraceID           string                 `json:"trace_id,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

func (e *SleeveDriftDetectedEvent) GetEventID() string      { return e.EventID }
func (e *SleeveDriftDetectedEvent) GetEventType() EventType { return e.EventType }
func (e *SleeveDriftDetectedEvent) GetTenantID() string     { return e.TenantID }
func (e *SleeveDriftDetectedEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *SleeveDriftDetectedEvent) GetUserID() *string      { return e.UserID }

// TaxHarvestSimulatedEvent emitted when tax harvest is simulated
type TaxHarvestSimulatedEvent struct {
	EventID         string                 `json:"event_id"`
	EventType       EventType              `json:"event_type"`
	TenantID        string                 `json:"tenant_id"`
	DatasourceID    string                 `json:"datasource_id"`
	PlanID          string                 `json:"plan_id"`
	UMAAccountID    string                 `json:"uma_account_id"`
	LossesHarvested float64                `json:"losses_harvested"`
	TaxSavingsEst   float64                `json:"tax_savings_est"`
	Timestamp       time.Time              `json:"timestamp"`
	UserID          *string                `json:"user_id,omitempty"`
	TraceID         string                 `json:"trace_id,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

func (e *TaxHarvestSimulatedEvent) GetEventID() string      { return e.EventID }
func (e *TaxHarvestSimulatedEvent) GetEventType() EventType { return e.EventType }
func (e *TaxHarvestSimulatedEvent) GetTenantID() string     { return e.TenantID }
func (e *TaxHarvestSimulatedEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *TaxHarvestSimulatedEvent) GetUserID() *string      { return e.UserID }

// GoldCopyConnectionEvent emitted when a connection in Gold Copy tenant is changed
type GoldCopyConnectionEvent struct {
	EventID        string                 `json:"event_id"`
	EventType      EventType              `json:"event_type"`
	TenantID       string                 `json:"tenant_id"`
	ConnectionID   string                 `json:"connection_id"`
	Action         string                 `json:"action"` // INSERT, UPDATE, DELETE
	ConnectionData map[string]interface{} `json:"connection_data,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
	UserID         *string                `json:"user_id,omitempty"`
}

func (e *GoldCopyConnectionEvent) GetEventID() string      { return e.EventID }
func (e *GoldCopyConnectionEvent) GetEventType() EventType { return e.EventType }
func (e *GoldCopyConnectionEvent) GetTenantID() string     { return e.TenantID }
func (e *GoldCopyConnectionEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *GoldCopyConnectionEvent) GetUserID() *string      { return e.UserID }

// GoldCopyEntityEvent generic event for gold copy entity changes
type GoldCopyEntityEvent struct {
	EventID    string                 `json:"event_id"`
	EventType  EventType              `json:"event_type"`
	TenantID   string                 `json:"tenant_id"`
	EntityType string                 `json:"entity_type"` // connection, instance, product, datasource
	EntityID   string                 `json:"entity_id"`
	Action     string                 `json:"action"` // INSERT, UPDATE, DELETE
	Data       map[string]interface{} `json:"data,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	UserID     *string                `json:"user_id,omitempty"`
}

func (e *GoldCopyEntityEvent) GetEventID() string      { return e.EventID }
func (e *GoldCopyEntityEvent) GetEventType() EventType { return e.EventType }
func (e *GoldCopyEntityEvent) GetTenantID() string     { return e.TenantID }
func (e *GoldCopyEntityEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *GoldCopyEntityEvent) GetUserID() *string      { return e.UserID }

const (
	GoldCopyEntityChanged EventType = "gold_copy.entity.changed"
)

// SemanticTermComplianceUpdatedEvent emitted when a semantic term inherits compliance flags
type SemanticTermComplianceUpdatedEvent struct {
	EventID              string    `json:"event_id"`
	EventType            EventType `json:"event_type"`
	TenantID             string    `json:"tenant_id"`
	SemanticTermID       string    `json:"semantic_term_id"`
	BusinessTermID       string    `json:"business_term_id"`
	InheritedPIIFlag     bool      `json:"inherited_pii_flag"`
	InheritedResidency   string    `json:"inherited_residency"`
	InheritedSensitivity string    `json:"inherited_sensitivity"`
	Timestamp            time.Time `json:"timestamp"`
	UserID               *string   `json:"user_id,omitempty"`
}

func (e *SemanticTermComplianceUpdatedEvent) GetEventID() string      { return e.EventID }
func (e *SemanticTermComplianceUpdatedEvent) GetEventType() EventType { return e.EventType }
func (e *SemanticTermComplianceUpdatedEvent) GetTenantID() string     { return e.TenantID }
func (e *SemanticTermComplianceUpdatedEvent) GetTimestamp() time.Time { return e.Timestamp }

type BusinessTermComplianceUpdatedEvent struct {
	EventID         string    `json:"event_id"`
	EventType       EventType `json:"event_type"`
	BusinessTermID  string    `json:"business_term_id"`
	PIIFlag         bool      `json:"pii_flag"`
	Residency       string    `json:"residency"`
	Sensitivity     string    `json:"sensitivity"`
	SemanticTermIDs []string  `json:"semantic_term_ids"`
	Timestamp       time.Time `json:"timestamp"`
	UserID          *string   `json:"user_id,omitempty"`
}

func (e *BusinessTermComplianceUpdatedEvent) GetEventID() string      { return e.EventID }
func (e *BusinessTermComplianceUpdatedEvent) GetEventType() EventType { return e.EventType }
func (e *BusinessTermComplianceUpdatedEvent) GetTimestamp() time.Time { return e.Timestamp }

type ComplianceViolationDetectedEvent struct {
	EventID       string    `json:"event_id"`
	EventType     EventType `json:"event_type"`
	JobID         string    `json:"job_id,omitempty"`
	DAGID         string    `json:"dag_id,omitempty"`
	TenantID      string    `json:"tenant_id"`
	ViolationType string    `json:"violation_type"`
	Details       string    `json:"details"`
	Timestamp     time.Time `json:"timestamp"`
}

func (e *ComplianceViolationDetectedEvent) GetEventID() string      { return e.EventID }
func (e *ComplianceViolationDetectedEvent) GetEventType() EventType { return e.EventType }
func (e *ComplianceViolationDetectedEvent) GetTenantID() string     { return e.TenantID }
func (e *ComplianceViolationDetectedEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *SemanticTermComplianceUpdatedEvent) GetUserID() *string    { return e.UserID }

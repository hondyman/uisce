# Advanced Northwind BO Features Implementation Guide

This guide covers the remaining three advanced features needed to complete the Northwind Business Object system:

1. **GraphQL API** - Query language for flexible BO retrieval
2. **Bulk Import/Export** - CSV/JSON batch operations
3. **Workflow Engine** - Business process automation

---

## 📊 Feature Summary

| Feature | Status | Priority | Effort | Dependencies |
|---------|--------|----------|--------|--------------|
| REST API | ✅ COMPLETE | P0 | Done | - |
| EventPublisher | ✅ COMPLETE | P0 | Done | RabbitMQ |
| **GraphQL API** | ⏳ NOT STARTED | P1 | 2-3 days | gqlgen |
| **Bulk Import/Export** | ⏳ NOT STARTED | P1 | 1-2 days | CSV/JSON parsers |
| **Workflow Engine** | ⏳ NOT STARTED | P2 | 3-5 days | State machine lib |

---

## 🔷 Part 1: GraphQL API Implementation

### Overview
GraphQL provides a flexible query interface for fetching Business Objects and instances with precise field selection.

### Implementation Architecture

```
Frontend (Queries/Mutations)
        ↓
GraphQL Schema (Type definitions)
        ↓
GraphQL Resolvers (Query handlers)
        ↓
Business Logic Layer (Services)
        ↓
Database (PostgreSQL)
```

### Step 1: Add gqlgen dependency

```bash
cd backend
go get github.com/99designs/gqlgen
go install github.com/99designs/gqlgen/cmd/gqlgen@latest
```

### Step 2: Define GraphQL Schema

Create `backend/graph/schema.graphqls`:

```graphql
# Business Object Type
type BusinessObject {
  id: String!
  tenantId: String!
  key: String!
  name: String!
  description: String
  fields: [FieldDefinition!]!
  subtypes: [SubtypeDefinition!]!
  createdAt: Time!
  createdBy: String!
}

# Field Definition Type
type FieldDefinition {
  id: String!
  key: String!
  name: String!
  businessName: String!
  technicalName: String!
  fieldType: String!
  isRequired: Boolean!
  isMultiValue: Boolean!
}

# Subtype Definition Type
type SubtypeDefinition {
  id: String!
  key: String!
  name: String!
  fields: [FieldDefinition!]!
}

# Business Object Instance
type BusinessObjectInstance {
  id: String!
  tenantId: String!
  businessObjectKey: String!
  coreFieldValues: JSON!
  customFieldValues: JSON!
  createdAt: Time!
  createdBy: String!
  lastModifiedAt: Time!
  lastModifiedBy: String!
}

# Query Root
type Query {
  # Fetch single BO by key
  businessObject(key: String!): BusinessObject
  
  # List all BOs for tenant
  businessObjects(
    page: Int = 1
    pageSize: Int = 50
  ): [BusinessObject!]!
  
  # Get single instance
  instance(id: String!): BusinessObjectInstance
  
  # List instances for a BO
  instances(
    boKey: String!
    page: Int = 1
    pageSize: Int = 50
    filter: InstanceFilter
  ): InstanceConnection!
  
  # Search instances across fields
  searchInstances(
    boKey: String!
    query: String!
  ): [BusinessObjectInstance!]!
}

# Mutation Root
type Mutation {
  # Create new BO
  createBusinessObject(input: CreateBOInput!): BusinessObject!
  
  # Update BO
  updateBusinessObject(key: String!, input: UpdateBOInput!): BusinessObject!
  
  # Delete BO
  deleteBusinessObject(key: String!): Boolean!
  
  # Clone BO
  cloneBusinessObject(sourceKey: String!, targetKey: String!): BusinessObject!
  
  # Create instance
  createInstance(
    boKey: String!
    input: CreateInstanceInput!
  ): BusinessObjectInstance!
  
  # Update instance
  updateInstance(
    id: String!
    input: UpdateInstanceInput!
  ): BusinessObjectInstance!
  
  # Delete instance
  deleteInstance(id: String!): Boolean!
  
  # Bulk create instances
  bulkCreateInstances(
    boKey: String!
    inputs: [CreateInstanceInput!]!
  ): [BusinessObjectInstance!]!
}

# Input types
input CreateBOInput {
  key: String!
  name: String!
  description: String
  fields: [FieldInput!]!
}

input UpdateBOInput {
  name: String
  description: String
}

input CreateInstanceInput {
  coreFields: JSON!
  customFields: JSON
}

input UpdateInstanceInput {
  coreFields: JSON
  customFields: JSON
}

input InstanceFilter {
  field: String!
  operator: FilterOperator!
  value: String!
}

enum FilterOperator {
  EQ
  NEQ
  GT
  LT
  CONTAINS
  STARTS_WITH
  ENDS_WITH
}

type InstanceConnection {
  edges: [InstanceEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type InstanceEdge {
  node: BusinessObjectInstance!
  cursor: String!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}

# Scalars
scalar Time
scalar JSON
```

### Step 3: Initialize gqlgen

```bash
cd backend
go run github.com/99designs/gqlgen init
```

This creates:
- `graph/generated.go` - Generated code
- `graph/model/` - Go models
- `graph/resolver.go` - Resolver template

### Step 4: Implement Resolvers

Create `backend/graph/resolver/business_object_resolver.go`:

```go
package resolver

import (
	"context"

	"github.com/eganpj/semlayer/backend/internal/models"
	"github.com/eganpj/semlayer/backend/internal/services"
	graphmodel "github.com/eganpj/semlayer/backend/graph/model"
)

type BusinessObjectResolver struct {
	boService *services.BusinessObjectService
}

// Query Resolvers
func (r *queryResolver) BusinessObject(ctx context.Context, key string) (*graphmodel.BusinessObject, error) {
	tenantID := ctx.Value("tenantID").(string)
	
	bo, err := r.boService.GetBusinessObject(ctx, tenantID, key)
	if err != nil {
		return nil, err
	}
	
	return r.mapBOToGraphQL(bo), nil
}

func (r *queryResolver) BusinessObjects(ctx context.Context, page *int, pageSize *int) ([]*graphmodel.BusinessObject, error) {
	tenantID := ctx.Value("tenantID").(string)
	
	pageNum := 1
	pageSz := 50
	
	if page != nil {
		pageNum = *page
	}
	if pageSize != nil {
		pageSz = *pageSize
	}
	
	offset := (pageNum - 1) * pageSz
	bos, _, err := r.boService.ListBusinessObjects(ctx, tenantID, offset, pageSz)
	if err != nil {
		return nil, err
	}
	
	var result []*graphmodel.BusinessObject
	for _, bo := range bos {
		result = append(result, r.mapBOToGraphQL(bo))
	}
	
	return result, nil
}

// Mutation Resolvers
func (r *mutationResolver) CreateBusinessObject(ctx context.Context, input graphmodel.CreateBOInput) (*graphmodel.BusinessObject, error) {
	tenantID := ctx.Value("tenantID").(string)
	userID := ctx.Value("userID").(string)
	
	req := models.CreateBusinessObjectRequest{
		Key:         input.Key,
		Name:        input.Name,
		Description: input.Description,
	}
	
	bo, err := r.boService.CreateBusinessObject(ctx, tenantID, req, userID)
	if err != nil {
		return nil, err
	}
	
	return r.mapBOToGraphQL(bo), nil
}

// Helper functions
func (r *queryResolver) mapBOToGraphQL(bo *models.BusinessObjectDefinition) *graphmodel.BusinessObject {
	result := &graphmodel.BusinessObject{
		ID:          bo.ID,
		TenantID:    bo.TenantID,
		Key:         bo.Key,
		Name:        bo.Name,
		Description: bo.Description,
		CreatedAt:   bo.CreatedAt,
		CreatedBy:   bo.CreatedBy,
	}
	
	// Map fields
	for _, field := range bo.Fields {
		result.Fields = append(result.Fields, &graphmodel.FieldDefinition{
			ID:            field.ID,
			Key:           field.Key,
			Name:          field.Name,
			BusinessName:  field.BusinessName,
			TechnicalName: field.TechnicalName,
			FieldType:     field.FieldType,
			IsRequired:    field.IsRequired,
			IsMultiValue:  field.IsMultiValue,
		})
	}
	
	return result
}
```

### Step 5: Register GraphQL Handler

In `backend/internal/handlers/graphql_handler.go`:

```go
package handlers

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/eganpj/semlayer/backend/graph/generated"
	"github.com/eganpj/semlayer/backend/graph/resolver"
	"github.com/eganpj/semlayer/backend/internal/services"
	"github.com/go-chi/chi/v5"
)

func RegisterGraphQLRoutes(router *chi.Mux, boService *services.BusinessObjectService) {
	// GraphQL schema
	schema := generated.NewExecutableSchema(
		generated.Config{
			Resolvers: &resolver.Resolver{
				BOService: boService,
			},
		},
	)
	
	// GraphQL endpoint
	router.Handle("/graphql", &gqlHandler{schema})
	
	// GraphQL Playground (dev)
	router.Handle("/graphql/playground", playground.Handler("GraphQL", "/graphql"))
}

type gqlHandler struct {
	schema graphql.ExecutableSchema
}

func (h *gqlHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Implementation using graphql.NewExecutor
	// Handle authorization
	// Execute query
}
```

### Usage Examples

**Query: Fetch all customers with their fields**
```graphql
query {
  businessObjects(page: 1, pageSize: 10) {
    id
    key
    name
    fields {
      key
      name
      businessName
      fieldType
    }
  }
}
```

**Mutation: Create customer instance**
```graphql
mutation {
  createInstance(
    boKey: "customer"
    input: {
      coreFields: { name: "Acme Corp", email: "info@acme.com" }
    }
  ) {
    id
    coreFieldValues
    createdAt
  }
}
```

**Query: Search instances**
```graphql
query {
  instances(
    boKey: "customer"
    filter: { field: "name", operator: CONTAINS, value: "Acme" }
  ) {
    edges {
      node {
        id
        coreFieldValues
      }
    }
    pageInfo {
      hasNextPage
      totalCount
    }
  }
}
```

---

## 📦 Part 2: Bulk Import/Export Implementation

### Overview
Enable bulk operations for efficient data migration and reporting.

### Step 1: Create Import Service

Create `backend/internal/services/bulk_import_service.go`:

```go
package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"github.com/eganpj/semlayer/backend/internal/models"
)

type BulkImportService struct {
	boService *BusinessObjectService
}

// ImportResult tracks import progress
type ImportResult struct {
	Total       int                              `json:"total"`
	Success     int                              `json:"success"`
	Failed      int                              `json:"failed"`
	Errors      map[int]string                  `json:"errors,omitempty"`
	Instances   []*models.BusinessObjectInstance `json:"instances,omitempty"`
}

// ImportCSV imports instances from CSV
func (s *BulkImportService) ImportCSV(
	ctx context.Context,
	tenantID, boKey, userID string,
	reader io.Reader,
	config ImportConfig,
) (*ImportResult, error) {
	csvReader := csv.NewReader(reader)
	
	// Read header
	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}
	
	result := &ImportResult{
		Errors: make(map[int]string),
	}
	
	rowNum := 1
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Failed++
			result.Errors[rowNum] = err.Error()
			continue
		}
		
		rowNum++
		
		// Parse row into field map
		coreFields := make(map[string]interface{})
		for i, value := range record {
			if i < len(header) {
				coreFields[header[i]] = value
			}
		}
		
		// Create instance
		instance := &models.BusinessObjectInstance{
			BusinessObjectKey: boKey,
			CoreFieldValues:   coreFields,
		}
		
		created, err := s.boService.CreateInstance(ctx, tenantID, userID, instance)
		if err != nil {
			result.Failed++
			result.Errors[rowNum] = err.Error()
			continue
		}
		
		result.Success++
		result.Instances = append(result.Instances, created)
	}
	
	result.Total = result.Success + result.Failed
	return result, nil
}

// ImportJSON imports instances from JSON
func (s *BulkImportService) ImportJSON(
	ctx context.Context,
	tenantID, boKey, userID string,
	reader io.Reader,
) (*ImportResult, error) {
	var items []map[string]interface{}
	if err := json.NewDecoder(reader).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	result := &ImportResult{
		Total:  len(items),
		Errors: make(map[int]string),
	}
	
	for i, item := range items {
		instance := &models.BusinessObjectInstance{
			BusinessObjectKey: boKey,
			CoreFieldValues:   item,
		}
		
		created, err := s.boService.CreateInstance(ctx, tenantID, userID, instance)
		if err != nil {
			result.Failed++
			result.Errors[i] = err.Error()
			continue
		}
		
		result.Success++
		result.Instances = append(result.Instances, created)
	}
	
	return result, nil
}

// ImportConfig configures import behavior
type ImportConfig struct {
	SkipErrors     bool   `json:"skipErrors"`
	DuplicateKey   string `json:"duplicateKey"`  // Field to check for duplicates
	UpdateOnMatch  bool   `json:"updateOnMatch"` // Update if duplicate found
}
```

### Step 2: Create Export Service

Create `backend/internal/services/bulk_export_service.go`:

```go
package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"io"

	"github.com/eganpj/semlayer/backend/internal/models"
)

type BulkExportService struct {
	boService *BusinessObjectService
}

// ExportCSV exports instances to CSV
func (s *BulkExportService) ExportCSV(
	ctx context.Context,
	tenantID, boKey string,
	writer io.Writer,
) error {
	instances, _, err := s.boService.ListInstances(ctx, tenantID, boKey, 0, 10000)
	if err != nil {
		return err
	}
	
	if len(instances) == 0 {
		return nil
	}
	
	csvWriter := csv.NewWriter(writer)
	
	// Write header from first instance
	first := instances[0]
	var headers []string
	for key := range first.CoreFieldValues {
		headers = append(headers, key)
	}
	csvWriter.Write(headers)
	
	// Write rows
	for _, instance := range instances {
		row := make([]string, len(headers))
		for i, header := range headers {
			if val, ok := instance.CoreFieldValues[header]; ok {
				row[i] = toString(val)
			}
		}
		csvWriter.Write(row)
	}
	
	csvWriter.Flush()
	return csvWriter.Error()
}

// ExportJSON exports instances to JSON
func (s *BulkExportService) ExportJSON(
	ctx context.Context,
	tenantID, boKey string,
	writer io.Writer,
) error {
	instances, _, err := s.boService.ListInstances(ctx, tenantID, boKey, 0, 10000)
	if err != nil {
		return err
	}
	
	return json.NewEncoder(writer).Encode(instances)
}

func toString(val interface{}) string {
	switch v := val.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%v", v)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}
```

### Step 3: Create Bulk Import/Export Handlers

In `backend/internal/handlers/bulk_handler.go`:

```go
package handlers

import (
	"fmt"
	"net/http"

	"github.com/eganpj/semlayer/backend/internal/services"
	"github.com/go-chi/chi/v5"
)

type BulkHandler struct {
	importService *services.BulkImportService
	exportService *services.BulkExportService
}

// POST /api/bo/{boKey}/import - Import instances
func (h *BulkHandler) ImportInstances(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	userID := r.Header.Get("X-User-ID")
	boKey := chi.URLParam(r, "boKey")
	
	format := r.URL.Query().Get("format") // "csv" or "json"
	if format == "" {
		format = "json"
	}
	
	var result *services.ImportResult
	var err error
	
	if format == "csv" {
		result, err = h.importService.ImportCSV(r.Context(), tenantID, boKey, userID, r.Body, services.ImportConfig{})
	} else {
		result, err = h.importService.ImportJSON(r.Context(), tenantID, boKey, userID, r.Body)
	}
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GET /api/bo/{boKey}/export - Export instances
func (h *BulkHandler) ExportInstances(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	boKey := chi.URLParam(r, "boKey")
	
	format := r.URL.Query().Get("format") // "csv" or "json"
	if format == "" {
		format = "json"
	}
	
	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", boKey))
		h.exportService.ExportCSV(r.Context(), tenantID, boKey, w)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", boKey))
		h.exportService.ExportJSON(r.Context(), tenantID, boKey, w)
	}
}
```

---

## ⚙️ Part 3: Workflow Engine Implementation

### Overview
Enable complex business processes with state machines and event-driven triggers.

### Architecture

```
Workflow Definition (Config)
         ↓
Instance Created Event
         ↓
Workflow Engine (State Machine)
         ↓
Transitions (Validate → Approve → Publish)
         ↓
Event Published to RabbitMQ
         ↓
Next Service (Notification, Audit)
```

### Step 1: Define Workflow Models

Create `backend/internal/models/workflow.go`:

```go
package models

import (
	"time"
)

// WorkflowDefinition defines a business process
type WorkflowDefinition struct {
	ID              string            `db:"id" json:"id"`
	TenantID        string            `db:"tenant_id" json:"tenantId"`
	Name            string            `db:"name" json:"name"`
	Description     string            `db:"description" json:"description"`
	BusinessObjectKey string          `db:"business_object_key" json:"businessObjectKey"`
	Triggers        []string          `db:"triggers" json:"triggers"` // Events that start workflow
	States          []WorkflowState   `json:"states"`
	InitialState    string            `db:"initial_state" json:"initialState"`
	CreatedAt       time.Time         `db:"created_at" json:"createdAt"`
}

// WorkflowState represents a state in the workflow
type WorkflowState struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Type        string               `json:"type"` // start, process, decision, end
	Actions     []WorkflowAction     `json:"actions"`
	Transitions []WorkflowTransition `json:"transitions"`
	IsTerminal  bool                 `json:"isTerminal"`
}

// WorkflowAction is an action taken in a state
type WorkflowAction struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // notify, validate, transform, publish_event
	Config   map[string]interface{} `json:"config"`
}

// WorkflowTransition defines how to move between states
type WorkflowTransition struct {
	ID        string                 `json:"id"`
	FromState string                 `json:"fromState"`
	ToState   string                 `json:"toState"`
	Condition string                 `json:"condition"` // CEL or simple condition
	Label     string                 `json:"label"`
}

// WorkflowInstance tracks execution of a workflow
type WorkflowInstance struct {
	ID                   string            `db:"id" json:"id"`
	TenantID             string            `db:"tenant_id" json:"tenantId"`
	WorkflowDefinitionID string            `db:"workflow_definition_id" json:"workflowDefinitionId"`
	BusinessObjectInstanceID string        `db:"business_object_instance_id" json:"businessObjectInstanceId"`
	CurrentState         string            `db:"current_state" json:"currentState"`
	Status               string            `db:"status" json:"status"` // running, completed, failed
	StartedAt            time.Time         `db:"started_at" json:"startedAt"`
	CompletedAt          *time.Time        `db:"completed_at" json:"completedAt"`
	History              []WorkflowStep    `json:"history"`
}

// WorkflowStep tracks a single step in workflow execution
type WorkflowStep struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	FromState  string                 `json:"fromState"`
	ToState    string                 `json:"toState"`
	ActionName string                 `json:"actionName"`
	Result     map[string]interface{} `json:"result"`
	Error      string                 `json:"error,omitempty"`
}
```

### Step 2: Create Workflow Engine

Create `backend/internal/services/workflow_engine.go`:

```go
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/eganpj/semlayer/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type WorkflowEngine struct {
	db             *sqlx.DB
	eventPublisher *EventPublisher
}

// ExecuteWorkflow starts a workflow instance
func (w *WorkflowEngine) ExecuteWorkflow(
	ctx context.Context,
	workflowID, boInstanceID, tenantID, userID string,
) (*models.WorkflowInstance, error) {
	// Load workflow definition
	workflowDef := &models.WorkflowDefinition{}
	err := w.db.GetContext(ctx, workflowDef, 
		"SELECT * FROM workflows WHERE id = $1 AND tenant_id = $2",
		workflowID, tenantID)
	if err != nil {
		return nil, err
	}
	
	// Create workflow instance
	instance := &models.WorkflowInstance{
		ID:                    uuid.New().String(),
		TenantID:              tenantID,
		WorkflowDefinitionID:  workflowID,
		BusinessObjectInstanceID: boInstanceID,
		CurrentState:          workflowDef.InitialState,
		Status:                "running",
		StartedAt:             time.Now(),
	}
	
	// Save instance
	_, err = w.db.ExecContext(ctx,
		"INSERT INTO workflow_instances (id, workflow_definition_id, business_object_instance_id, current_state, status, started_at) VALUES ($1, $2, $3, $4, $5, $6)",
		instance.ID, workflowID, boInstanceID, instance.CurrentState, "running", instance.StartedAt,
	)
	if err != nil {
		return nil, err
	}
	
	// Execute initial state
	w.executeState(ctx, instance, workflowDef, userID)
	
	return instance, nil
}

// TransitionWorkflow moves workflow to next state
func (w *WorkflowEngine) TransitionWorkflow(
	ctx context.Context,
	instanceID, newState, tenantID, userID string,
) (*models.WorkflowInstance, error) {
	// Load current instance
	instance := &models.WorkflowInstance{}
	err := w.db.GetContext(ctx, instance,
		"SELECT * FROM workflow_instances WHERE id = $1 AND tenant_id = $2",
		instanceID, tenantID)
	if err != nil {
		return nil, err
	}
	
	// Update state
	instance.CurrentState = newState
	_, err = w.db.ExecContext(ctx,
		"UPDATE workflow_instances SET current_state = $1 WHERE id = $2",
		newState, instanceID)
	if err != nil {
		return nil, err
	}
	
	// Publish transition event
	if w.eventPublisher != nil {
		w.eventPublisher.PublishWorkflowEvent(ctx,
			services.EventWorkflowProgress,
			instanceID,
			tenantID,
			userID,
			map[string]interface{}{
				"from_state": instance.CurrentState,
				"to_state":   newState,
			},
		)
	}
	
	return instance, nil
}

func (w *WorkflowEngine) executeState(
	ctx context.Context,
	instance *models.WorkflowInstance,
	workflowDef *models.WorkflowDefinition,
	userID string,
) error {
	// Find current state definition
	var currentState *models.WorkflowState
	for _, state := range workflowDef.States {
		if state.Name == instance.CurrentState {
			currentState = &state
			break
		}
	}
	
	if currentState == nil {
		return fmt.Errorf("state not found: %s", instance.CurrentState)
	}
	
	// Execute actions in state
	for _, action := range currentState.Actions {
		result, err := w.executeAction(ctx, action, instance, userID)
		if err != nil {
			instance.Status = "failed"
			return err
		}
		
		// Log step
		step := &models.WorkflowStep{
			ID:         uuid.New().String(),
			Timestamp:  time.Now(),
			FromState:  instance.CurrentState,
			ActionName: action.Type,
			Result:     result,
		}
		
		instance.History = append(instance.History, *step)
	}
	
	// Check if terminal state
	if currentState.IsTerminal {
		now := time.Now()
		instance.CompletedAt = &now
		instance.Status = "completed"
	}
	
	return nil
}

func (w *WorkflowEngine) executeAction(
	ctx context.Context,
	action models.WorkflowAction,
	instance *models.WorkflowInstance,
	userID string,
) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	switch action.Type {
	case "notify":
		// Send notification
		result["notified"] = true
	case "validate":
		// Validate instance data
		result["valid"] = true
	case "transform":
		// Transform data
		result["transformed"] = true
	case "publish_event":
		// Publish custom event
		result["published"] = true
	}
	
	return result, nil
}
```

---

## 🚀 Integration Path

### Phase 1: GraphQL (Week 1)
- Install and configure gqlgen
- Define GraphQL schema
- Implement resolvers
- Deploy on /graphql endpoint

### Phase 2: Bulk Operations (Week 2)
- Implement CSV import/export
- Implement JSON import/export
- Add validation and error handling
- Deploy on /api/bo/{boKey}/import-export

### Phase 3: Workflows (Week 3)
- Design workflow DSL
- Implement WorkflowEngine
- Create workflow management UI
- Publish events to RabbitMQ

---

## 📝 Next Steps

1. **Install dependencies** (when ready to implement)
2. **Follow implementation guides** section by section
3. **Test each feature** with curl/GraphQL playground/UI
4. **Update documentation** with examples
5. **Deploy to staging** for integration testing

See individual sections above for detailed implementation guides.

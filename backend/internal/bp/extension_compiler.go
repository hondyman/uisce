package bp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/tenant"
)

var (
	ErrExtensionAnchorNotFound = errors.New("extension anchor not found in base workflow")
	ErrBaseDefinitionNotFound  = errors.New("base process definition not found")
	ErrInvalidSourceType       = errors.New("invalid source_type: must be CORE, EXTENDED, or CLONED")
)

// ExtensionOp represents the operation type on an extension point
type ExtensionOp string

const (
	OpInsertAfter    ExtensionOp = "INSERT_AFTER"
	OpAppendBranch   ExtensionOp = "APPEND_BRANCH"
	OpConfigOverride ExtensionOp = "CONFIG_OVERRIDE"
)

// ExtensionSpec represents an individual extension delta attached to a base workflow anchor
type ExtensionSpec struct {
	AnchorID        string                 `json:"anchor_id"`
	Operation       ExtensionOp            `json:"operation"`
	Steps           []WorkflowStep         `json:"steps,omitempty"`
	BranchCondition *string                `json:"branch_condition,omitempty"`
	ConfigOverrides map[string]interface{} `json:"config_overrides,omitempty"`
}

// WorkflowStep is the standard step shape serialized into steps_json
type WorkflowStep struct {
	StepID        string                 `json:"step_id"`
	StepName      string                 `json:"step_name"`
	StepType      string                 `json:"step_type"`
	StepOrder     int                    `json:"step_order"`
	DurationHours int                    `json:"duration_hours,omitempty"`
	AssigneeRole  string                 `json:"assignee_role,omitempty"`
	Config        map[string]interface{} `json:"config,omitempty"`
}

// ProcessDefinition represents a row in public.bp_process_definition
type ProcessDefinition struct {
	ID               uuid.UUID       `db:"id" json:"id"`
	ProcessID        string          `db:"process_id" json:"process_id"`
	TenantID         string          `db:"tenant_id" json:"tenant_id"`
	Version          int             `db:"version" json:"version"`
	Name             string          `db:"name" json:"name"`
	StepsJSON        json.RawMessage `db:"steps_json" json:"steps_json"`
	GraphJSON        json.RawMessage `db:"graph_json" json:"graph_json"`
	SourceType       string          `db:"source_type" json:"source_type"` // 'CORE', 'EXTENDED', 'CLONED'
	BaseDefinitionID *uuid.UUID      `db:"base_definition_id" json:"base_definition_id,omitempty"`
	BaseVersion      *int            `db:"base_version" json:"base_version,omitempty"`
	ExtensionsJSON   json.RawMessage `db:"extensions_json" json:"extensions_json,omitempty"`
	CreatedAt        time.Time       `db:"created_at" json:"created_at"`
}

// CompileProcessRequest is the payload to compile and persist a workflow definition
type CompileProcessRequest struct {
	TenantID         string          `json:"tenant_id"`
	ProcessID        string          `json:"process_id"`
	Name             string          `json:"name"`
	SourceType       string          `json:"source_type"` // 'CORE', 'EXTENDED', 'CLONED'
	BaseDefinitionID *uuid.UUID      `json:"base_definition_id,omitempty"`
	BaseVersion      *int            `json:"base_version,omitempty"`
	Extensions       []ExtensionSpec `json:"extensions,omitempty"`
	Steps            []WorkflowStep  `json:"steps,omitempty"` // Provided directly if CORE or CLONED
	GraphJSON        json.RawMessage `json:"graph_json,omitempty"`
	TriggerType      string          `json:"trigger_type,omitempty"` // 'event', 'manual', 'schedule'
	TriggerTopic     string          `json:"trigger_topic,omitempty"`
	TriggerEventType string          `json:"trigger_event_type,omitempty"`
	TriggerOps       json.RawMessage `json:"trigger_ops,omitempty"`
}

// WorkflowCompiler compiles, extends, and persists process definitions
type WorkflowCompiler struct {
	db               *sql.DB
	subscriptionRepo *TriggerSubscriptionStore
}

// NewWorkflowCompiler creates a new WorkflowCompiler instance
func NewWorkflowCompiler(db *sql.DB, subs *TriggerSubscriptionStore) *WorkflowCompiler {
	if subs == nil && db != nil {
		subs = NewTriggerSubscriptionStore(db)
	}
	return &WorkflowCompiler{
		db:               db,
		subscriptionRepo: subs,
	}
}

// CompileAndPublish resolves extensions (if any), materializes the workflow definition,
// persists the new version under tenant RLS transaction isolation, and maintains trigger subscriptions.
func (c *WorkflowCompiler) CompileAndPublish(ctx context.Context, req CompileProcessRequest) (*ProcessDefinition, error) {
	if c.db == nil {
		return nil, fmt.Errorf("workflow compiler: db is nil")
	}
	if req.TenantID == "" || req.ProcessID == "" {
		return nil, fmt.Errorf("tenant_id and process_id are required")
	}
	if req.SourceType == "" {
		req.SourceType = "CORE"
	}
	if req.SourceType != "CORE" && req.SourceType != "EXTENDED" && req.SourceType != "CLONED" {
		return nil, ErrInvalidSourceType
	}

	var materializedSteps []WorkflowStep
	var resolvedBaseID *uuid.UUID
	var resolvedBaseVersion *int
	var extensionsJSON []byte

	switch req.SourceType {
	case "CORE", "CLONED":
		materializedSteps = req.Steps
		// Re-number consecutively
		for i := range materializedSteps {
			materializedSteps[i].StepOrder = i + 1
		}
	case "EXTENDED":
		if req.BaseDefinitionID == nil {
			return nil, fmt.Errorf("base_definition_id required for EXTENDED source_type")
		}

		// Published-core semantics:
		// Currently loads the latest CORE version for base_definition_id.
		// Lifecycle governance status will later refine this to: WHERE status = 'published'.
		baseDef, err := c.loadBaseDefinition(ctx, req.TenantID, *req.BaseDefinitionID, req.BaseVersion)
		if err != nil {
			return nil, err
		}

		resolvedBaseID = &baseDef.ID
		resolvedBaseVersion = &baseDef.Version

		var baseSteps []WorkflowStep
		if len(baseDef.StepsJSON) > 0 {
			if err := json.Unmarshal(baseDef.StepsJSON, &baseSteps); err != nil {
				return nil, fmt.Errorf("unmarshal base steps: %w", err)
			}
		}

		// Materialize extensions onto base steps
		mergedSteps, err := c.MaterializeExtensions(baseSteps, req.Extensions)
		if err != nil {
			return nil, err
		}
		materializedSteps = mergedSteps

		if len(req.Extensions) > 0 {
			extBytes, err := json.Marshal(req.Extensions)
			if err != nil {
				return nil, fmt.Errorf("marshal extensions: %w", err)
			}
			extensionsJSON = extBytes
		}
	}

	stepsBytes, err := json.Marshal(materializedSteps)
	if err != nil {
		return nil, fmt.Errorf("marshal materialized steps: %w", err)
	}

	graphBytes := req.GraphJSON
	if len(graphBytes) == 0 {
		graphBytes = json.RawMessage(`{}`)
	}

	// Persist inside scoped transaction setting tenant RLS context
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin compile tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := tenant.SetRLSContext(ctx, tx, req.TenantID); err != nil {
		return nil, fmt.Errorf("set rls context: %w", err)
	}

	// Compute next version for this (tenant_id, process_id)
	var nextVersion int
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(version), 0) + 1
		FROM public.bp_process_definition
		WHERE process_id = $1
	`, req.ProcessID).Scan(&nextVersion)
	if err != nil {
		return nil, fmt.Errorf("compute next version: %w", err)
	}

	defID := uuid.New()
	var extJSONParam *string
	if len(extensionsJSON) > 0 {
		s := string(extensionsJSON)
		extJSONParam = &s
	}

	query := `
		INSERT INTO public.bp_process_definition (
			id, process_id, tenant_id, version, name,
			steps_json, graph_json, source_type,
			base_definition_id, base_version, extensions_json
		) VALUES (
			$1, $2, $3, $4, $5,
			$6::jsonb, $7::jsonb, $8,
			$9, $10, $11::jsonb
		)
		RETURNING id, process_id, tenant_id, version, name, steps_json, graph_json, source_type,
		          base_definition_id, base_version, extensions_json, created_at
	`

	var persisted ProcessDefinition
	var baseIDNull sql.NullString
	var baseVerNull sql.NullInt64
	var extJSONNull sql.NullString

	var stepsRaw, graphRaw []byte

	err = tx.QueryRowContext(ctx, query,
		defID,
		req.ProcessID,
		req.TenantID,
		nextVersion,
		req.Name,
		string(stepsBytes),
		string(graphBytes),
		req.SourceType,
		resolvedBaseID,
		resolvedBaseVersion,
		extJSONParam,
	).Scan(
		&persisted.ID,
		&persisted.ProcessID,
		&persisted.TenantID,
		&persisted.Version,
		&persisted.Name,
		&stepsRaw,
		&graphRaw,
		&persisted.SourceType,
		&baseIDNull,
		&baseVerNull,
		&extJSONNull,
		&persisted.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert process definition: %w", err)
	}

	persisted.StepsJSON = stepsRaw
	persisted.GraphJSON = graphRaw

	if baseIDNull.Valid {
		parsed, _ := uuid.Parse(baseIDNull.String)
		persisted.BaseDefinitionID = &parsed
	}
	if baseVerNull.Valid {
		v := int(baseVerNull.Int64)
		persisted.BaseVersion = &v
	}
	if extJSONNull.Valid {
		persisted.ExtensionsJSON = json.RawMessage(extJSONNull.String)
	}

	// Maintain trigger subscriptions
	tenantUUID, _ := uuid.Parse(req.TenantID)
	if req.TriggerType == "event" && req.TriggerTopic != "" && tenantUUID != uuid.Nil {
		var evtType *string
		if req.TriggerEventType != "" {
			evtType = &req.TriggerEventType
		}
		sub := TriggerSubscription{
			TenantID:    tenantUUID,
			ProcessID:   req.ProcessID,
			TriggerType: "event",
			Topic:       req.TriggerTopic,
			EventType:   evtType,
			Ops:         req.TriggerOps,
			Version:     nextVersion,
		}
		if err := c.subscriptionRepo.UpsertSubscription(ctx, sub); err != nil {
			return nil, fmt.Errorf("upsert trigger subscription: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit compile tx: %w", err)
	}

	return &persisted, nil
}

// MaterializeExtensions applies extension operations onto base steps in a deterministic, idempotent manner.
func (c *WorkflowCompiler) MaterializeExtensions(baseSteps []WorkflowStep, extensions []ExtensionSpec) ([]WorkflowStep, error) {
	// 1. Build index map of base step IDs
	stepMap := make(map[string]int)
	for i, s := range baseSteps {
		stepMap[s.StepID] = i
	}

	// 2. Validate all extension anchors exist
	for _, ext := range extensions {
		if _, exists := stepMap[ext.AnchorID]; !exists {
			return nil, fmt.Errorf("%w: anchor %q not found in base workflow", ErrExtensionAnchorNotFound, ext.AnchorID)
		}
	}

	// 3. Make working copy of base steps
	result := make([]WorkflowStep, len(baseSteps))
	copy(result, baseSteps)

	// Deep clone configs
	for i := range result {
		if result[i].Config != nil {
			clone := make(map[string]interface{})
			for k, v := range result[i].Config {
				clone[k] = v
			}
			result[i].Config = clone
		}
	}

	// 4. Apply CONFIG_OVERRIDE first
	for _, ext := range extensions {
		if ext.Operation == OpConfigOverride {
			idx := -1
			for i, s := range result {
				if s.StepID == ext.AnchorID {
					idx = i
					break
				}
			}
			if idx >= 0 {
				if result[idx].Config == nil {
					result[idx].Config = make(map[string]interface{})
				}
				for k, v := range ext.ConfigOverrides {
					result[idx].Config[k] = v
				}
			}
		}
	}

	// 5. Apply INSERT_AFTER and APPEND_BRANCH
	for _, ext := range extensions {
		if ext.Operation == OpInsertAfter || ext.Operation == OpAppendBranch {
			idx := -1
			for i, s := range result {
				if s.StepID == ext.AnchorID {
					idx = i
					break
				}
			}
			if idx < 0 {
				continue
			}

			// Prepare steps to insert
			stepsToInsert := make([]WorkflowStep, len(ext.Steps))
			for i, s := range ext.Steps {
				sCopy := s
				if ext.Operation == OpAppendBranch && ext.BranchCondition != nil {
					if sCopy.Config == nil {
						sCopy.Config = make(map[string]interface{})
					}
					sCopy.Config["branch_condition"] = *ext.BranchCondition
				}
				stepsToInsert[i] = sCopy
			}

			// Splice immediately after anchor step
			insertPos := idx + 1
			newResult := make([]WorkflowStep, 0, len(result)+len(stepsToInsert))
			newResult = append(newResult, result[:insertPos]...)
			newResult = append(newResult, stepsToInsert...)
			newResult = append(newResult, result[insertPos:]...)
			result = newResult
		}
	}

	// 6. Re-index step orders deterministically 1..N
	for i := range result {
		result[i].StepOrder = i + 1
	}

	return result, nil
}

func (c *WorkflowCompiler) loadBaseDefinition(ctx context.Context, tenantID string, baseID uuid.UUID, version *int) (*ProcessDefinition, error) {
	// Read base definition inside a read-only transaction with tenant RLS set so core definitions are readable
	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin load base def tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := tenant.SetRLSContext(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("set rls context in load base def: %w", err)
	}

	var query string
	var args []interface{}

	if version != nil && *version > 0 {
		query = `
			SELECT id, process_id, tenant_id, version, name, steps_json, graph_json, source_type
			FROM public.bp_process_definition
			WHERE (id = $1 OR process_id = $1::text) AND version = $2
			LIMIT 1
		`
		args = []interface{}{baseID, *version}
	} else {
		query = `
			SELECT id, process_id, tenant_id, version, name, steps_json, graph_json, source_type
			FROM public.bp_process_definition
			WHERE id = $1 OR process_id = $1::text
			ORDER BY version DESC
			LIMIT 1
		`
		args = []interface{}{baseID}
	}

	var def ProcessDefinition
	err = tx.QueryRowContext(ctx, query, args...).Scan(
		&def.ID,
		&def.ProcessID,
		&def.TenantID,
		&def.Version,
		&def.Name,
		&def.StepsJSON,
		&def.GraphJSON,
		&def.SourceType,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: id=%s", ErrBaseDefinitionNotFound, baseID)
		}
		return nil, fmt.Errorf("query base definition: %w", err)
	}

	return &def, nil
}

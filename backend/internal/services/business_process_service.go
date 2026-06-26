package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/bo"
	hasuraclient "github.com/hondyman/semlayer/libs/hasura-client"
	"go.uber.org/zap"
)

// BusinessProcessService manages business process execution
type BusinessProcessService struct {
	client *hasuraclient.HasuraClient
	logger *zap.Logger
}

// NewBusinessProcessService creates a new BusinessProcessService
func NewBusinessProcessService(client *hasuraclient.HasuraClient) *BusinessProcessService {
	logger, _ := zap.NewProduction()
	return &BusinessProcessService{
		client: client,
		logger: logger,
	}
}

// StartProcess initiates a new business process instance
func (s *BusinessProcessService) StartProcess(ctx context.Context, processKey, entityType, entityID, createdBy string, initialData map[string]interface{}) (*bo.ProcessInstance, error) {
	// Get process definition
	process, err := s.GetProcessByKey(ctx, processKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get process: %w", err)
	}

	if process == nil {
		return nil, fmt.Errorf("process not found: %s", processKey)
	}

	// Get first step
	steps, err := s.GetProcessSteps(ctx, process.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get steps: %w", err)
	}

	var firstStepID *string
	if len(steps) > 0 {
		firstStepID = &steps[0].ID
	}

	// Create instance
	instance := &bo.ProcessInstance{
		ID:            uuid.New().String(),
		TenantID:      "default-tenant",
		ProcessID:     process.ID,
		EntityType:    entityType,
		EntityID:      entityID,
		CurrentStepID: firstStepID,
		Status:        "in_progress",
		StartedAt:     time.Now(),
		CreatedBy:     createdBy,
	}

	if initialData != nil {
		dataJSON, _ := json.Marshal(initialData)
		instance.Data = dataJSON
	}

	// Insert instance
	mutation := `
		mutation InsertProcessInstance($object: process_instances_insert_input!) {
			insert_process_instances_one(object: $object) {
				id
				status
				started_at
			}
		}
	`

	object := map[string]interface{}{
		"id":              instance.ID,
		"tenant_id":       instance.TenantID,
		"process_id":      instance.ProcessID,
		"entity_type":     instance.EntityType,
		"entity_id":       instance.EntityID,
		"current_step_id": instance.CurrentStepID,
		"status":          instance.Status,
		"started_at":      instance.StartedAt,
		"data":            string(instance.Data),
		"created_by":      instance.CreatedBy,
	}

	_, err = s.client.Mutate(mutation, map[string]interface{}{"object": object})
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	// Record first step history
	if firstStepID != nil {
		s.recordStepHistory(ctx, instance.ID, *firstStepID, "started", createdBy, "", nil)
	}

	s.logger.Info("Process started",
		zap.String("process", processKey),
		zap.String("instance", instance.ID),
		zap.String("entity", entityID))

	return instance, nil
}

// AdvanceProcess moves the process to the next step
func (s *BusinessProcessService) AdvanceProcess(ctx context.Context, instanceID, action, actor, comments string, data map[string]interface{}) error {
	// Get current instance
	instance, err := s.GetInstance(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	if instance.Status == "completed" || instance.Status == "cancelled" {
		return fmt.Errorf("process already %s", instance.Status)
	}

	// Get all steps
	steps, err := s.GetProcessSteps(ctx, instance.ProcessID)
	if err != nil {
		return fmt.Errorf("failed to get steps: %w", err)
	}

	// Find current step index
	currentIdx := -1
	for i, step := range steps {
		if instance.CurrentStepID != nil && step.ID == *instance.CurrentStepID {
			currentIdx = i
			break
		}
	}

	// Record action on current step
	if instance.CurrentStepID != nil {
		s.recordStepHistory(ctx, instanceID, *instance.CurrentStepID, action, actor, comments, data)
	}

	// Handle action
	switch action {
	case "approved", "completed", "skipped":
		// Move to next step
		if currentIdx < len(steps)-1 {
			nextStep := steps[currentIdx+1]
			s.updateInstanceStep(ctx, instanceID, nextStep.ID, "in_progress")
			s.recordStepHistory(ctx, instanceID, nextStep.ID, "started", actor, "", nil)
		} else {
			// Last step - complete process
			s.completeProcess(ctx, instanceID)
		}

	case "rejected":
		s.updateInstanceStatus(ctx, instanceID, "rejected")

	case "cancelled":
		s.updateInstanceStatus(ctx, instanceID, "cancelled")

	default:
		// Just update data, stay on current step
		if data != nil {
			s.updateInstanceData(ctx, instanceID, data)
		}
	}

	return nil
}

// CompleteProcess marks a process as completed
func (s *BusinessProcessService) CompleteProcess(ctx context.Context, instanceID string) error {
	return s.completeProcess(ctx, instanceID)
}

// GetProcessByKey fetches a process definition by key
func (s *BusinessProcessService) GetProcessByKey(ctx context.Context, key string) (*bo.BusinessProcess, error) {
	query := `
		query GetProcessByKey($key: String!) {
			business_processes(where: { key: { _eq: $key } }, limit: 1) {
				id
				tenant_id
				key
				name
				display_name
				description
				category
				status
				version
				is_system
			}
		}
	`

	result, err := s.client.Query(query, map[string]interface{}{"key": key})
	if err != nil {
		return nil, err
	}

	processes, ok := result["business_processes"].([]interface{})
	if !ok || len(processes) == 0 {
		return nil, nil
	}

	data := processes[0].(map[string]interface{})
	return &bo.BusinessProcess{
		ID:          data["id"].(string),
		TenantID:    getString(data, "tenant_id"),
		Key:         getString(data, "key"),
		Name:        getString(data, "name"),
		DisplayName: getString(data, "display_name"),
		Description: getString(data, "description"),
		Category:    getString(data, "category"),
		Status:      getString(data, "status"),
		IsSystem:    getBool(data, "is_system"),
	}, nil
}

// GetProcessSteps fetches all steps for a process
func (s *BusinessProcessService) GetProcessSteps(ctx context.Context, processID string) ([]*bo.ProcessStep, error) {
	query := `
		query GetProcessSteps($process_id: String!) {
			process_steps(
				where: { process_id: { _eq: $process_id } }
				order_by: { sequence: asc }
			) {
				id
				tenant_id
				process_id
				key
				name
				display_name
				step_type
				sequence
				config
				is_required
			}
		}
	`

	result, err := s.client.Query(query, map[string]interface{}{"process_id": processID})
	if err != nil {
		return nil, err
	}

	items, ok := result["process_steps"].([]interface{})
	if !ok {
		return []*bo.ProcessStep{}, nil
	}

	steps := make([]*bo.ProcessStep, 0, len(items))
	for _, item := range items {
		data := item.(map[string]interface{})
		step := &bo.ProcessStep{
			ID:          data["id"].(string),
			TenantID:    getString(data, "tenant_id"),
			ProcessID:   getString(data, "process_id"),
			Key:         getString(data, "key"),
			Name:        getString(data, "name"),
			DisplayName: getString(data, "display_name"),
			StepType:    getString(data, "step_type"),
			Sequence:    int(data["sequence"].(float64)),
			IsRequired:  getBool(data, "is_required"),
		}
		if cfg, ok := data["config"].(map[string]interface{}); ok {
			step.Config, _ = json.Marshal(cfg)
		}
		steps = append(steps, step)
	}

	return steps, nil
}

// GetInstance fetches a process instance by ID
func (s *BusinessProcessService) GetInstance(ctx context.Context, instanceID string) (*bo.ProcessInstance, error) {
	query := `
		query GetInstance($id: String!) {
			process_instances_by_pk(id: $id) {
				id
				tenant_id
				process_id
				entity_type
				entity_id
				current_step_id
				status
				started_at
				completed_at
				data
				created_by
			}
		}
	`

	result, err := s.client.Query(query, map[string]interface{}{"id": instanceID})
	if err != nil {
		return nil, err
	}

	data, ok := result["process_instances_by_pk"].(map[string]interface{})
	if !ok || data == nil {
		return nil, fmt.Errorf("instance not found: %s", instanceID)
	}

	instance := &bo.ProcessInstance{
		ID:         data["id"].(string),
		TenantID:   getString(data, "tenant_id"),
		ProcessID:  getString(data, "process_id"),
		EntityType: getString(data, "entity_type"),
		EntityID:   getString(data, "entity_id"),
		Status:     getString(data, "status"),
		CreatedBy:  getString(data, "created_by"),
	}

	if stepID, ok := data["current_step_id"].(string); ok {
		instance.CurrentStepID = &stepID
	}

	return instance, nil
}

// GetInstanceHistory fetches the step history for an instance
func (s *BusinessProcessService) GetInstanceHistory(ctx context.Context, instanceID string) ([]*bo.StepHistory, error) {
	query := `
		query GetInstanceHistory($instance_id: String!) {
			step_history(
				where: { instance_id: { _eq: $instance_id } }
				order_by: { created_at: asc }
			) {
				id
				instance_id
				step_id
				action
				actor
				comments
				data
				created_at
			}
		}
	`

	result, err := s.client.Query(query, map[string]interface{}{"instance_id": instanceID})
	if err != nil {
		return nil, err
	}

	items, ok := result["step_history"].([]interface{})
	if !ok {
		return []*bo.StepHistory{}, nil
	}

	history := make([]*bo.StepHistory, 0, len(items))
	for _, item := range items {
		data := item.(map[string]interface{})
		h := &bo.StepHistory{
			ID:         data["id"].(string),
			InstanceID: getString(data, "instance_id"),
			StepID:     getString(data, "step_id"),
			Action:     getString(data, "action"),
			Actor:      getString(data, "actor"),
			Comments:   getString(data, "comments"),
		}
		history = append(history, h)
	}

	return history, nil
}

// ListProcesses fetches all available business processes
func (s *BusinessProcessService) ListProcesses(ctx context.Context, category string) ([]*bo.BusinessProcess, error) {
	query := `
		query ListProcesses($category: String) {
			business_processes(
				where: { category: { _eq: $category }, status: { _eq: "active" } }
				order_by: { name: asc }
			) {
				id
				key
				name
				display_name
				description
				category
				is_system
			}
		}
	`

	vars := map[string]interface{}{}
	if category != "" {
		vars["category"] = category
	}

	result, err := s.client.Query(query, vars)
	if err != nil {
		return nil, err
	}

	items, ok := result["business_processes"].([]interface{})
	if !ok {
		return []*bo.BusinessProcess{}, nil
	}

	processes := make([]*bo.BusinessProcess, 0, len(items))
	for _, item := range items {
		data := item.(map[string]interface{})
		p := &bo.BusinessProcess{
			ID:          data["id"].(string),
			Key:         getString(data, "key"),
			Name:        getString(data, "name"),
			DisplayName: getString(data, "display_name"),
			Description: getString(data, "description"),
			Category:    getString(data, "category"),
			IsSystem:    getBool(data, "is_system"),
		}
		processes = append(processes, p)
	}

	return processes, nil
}

// ListInstancesForEntity fetches all process instances for an entity
func (s *BusinessProcessService) ListInstancesForEntity(ctx context.Context, entityType, entityID string) ([]*bo.ProcessInstance, error) {
	query := `
		query ListInstancesForEntity($entity_type: String!, $entity_id: String!) {
			process_instances(
				where: { entity_type: { _eq: $entity_type }, entity_id: { _eq: $entity_id } }
				order_by: { started_at: desc }
			) {
				id
				process_id
				status
				started_at
				completed_at
				created_by
			}
		}
	`

	result, err := s.client.Query(query, map[string]interface{}{
		"entity_type": entityType,
		"entity_id":   entityID,
	})
	if err != nil {
		return nil, err
	}

	items, ok := result["process_instances"].([]interface{})
	if !ok {
		return []*bo.ProcessInstance{}, nil
	}

	instances := make([]*bo.ProcessInstance, 0, len(items))
	for _, item := range items {
		data := item.(map[string]interface{})
		inst := &bo.ProcessInstance{
			ID:         data["id"].(string),
			ProcessID:  getString(data, "process_id"),
			EntityType: entityType,
			EntityID:   entityID,
			Status:     getString(data, "status"),
			CreatedBy:  getString(data, "created_by"),
		}
		instances = append(instances, inst)
	}

	return instances, nil
}

// Helper methods

func (s *BusinessProcessService) completeProcess(ctx context.Context, instanceID string) error {
	mutation := `
		mutation CompleteProcess($id: String!, $completed_at: timestamptz!) {
			update_process_instances_by_pk(
				pk_columns: { id: $id }
				_set: { status: "completed", completed_at: $completed_at }
			) {
				id
				status
			}
		}
	`
	_, err := s.client.Mutate(mutation, map[string]interface{}{
		"id":           instanceID,
		"completed_at": time.Now(),
	})
	return err
}

func (s *BusinessProcessService) updateInstanceStep(ctx context.Context, instanceID, stepID, status string) error {
	mutation := `
		mutation UpdateInstanceStep($id: String!, $step_id: String!, $status: String!) {
			update_process_instances_by_pk(
				pk_columns: { id: $id }
				_set: { current_step_id: $step_id, status: $status }
			) {
				id
			}
		}
	`

	_, err := s.client.Mutate(mutation, map[string]interface{}{
		"id":      instanceID,
		"step_id": stepID,
		"status":  status,
	})
	return err
}

func (s *BusinessProcessService) updateInstanceStatus(ctx context.Context, instanceID, status string) error {
	mutation := `
		mutation UpdateInstanceStatus($id: String!, $status: String!) {
			update_process_instances_by_pk(
				pk_columns: { id: $id }
				_set: { status: $status }
			) {
				id
			}
		}
	`

	_, err := s.client.Mutate(mutation, map[string]interface{}{
		"id":     instanceID,
		"status": status,
	})
	return err
}

func (s *BusinessProcessService) updateInstanceData(ctx context.Context, instanceID string, data map[string]interface{}) error {
	dataJSON, _ := json.Marshal(data)

	mutation := `
		mutation UpdateInstanceData($id: String!, $data: jsonb!) {
			update_process_instances_by_pk(
				pk_columns: { id: $id }
				_set: { data: $data }
			) {
				id
			}
		}
	`

	_, err := s.client.Mutate(mutation, map[string]interface{}{
		"id":   instanceID,
		"data": string(dataJSON),
	})
	return err
}

func (s *BusinessProcessService) recordStepHistory(ctx context.Context, instanceID, stepID, action, actor, comments string, data map[string]interface{}) error {
	var dataJSON json.RawMessage
	if data != nil {
		dataJSON, _ = json.Marshal(data)
	}

	mutation := `
		mutation InsertStepHistory($object: step_history_insert_input!) {
			insert_step_history_one(object: $object) {
				id
			}
		}
	`

	object := map[string]interface{}{
		"id":          uuid.New().String(),
		"instance_id": instanceID,
		"step_id":     stepID,
		"action":      action,
		"actor":       actor,
		"comments":    comments,
		"data":        string(dataJSON),
		"created_at":  time.Now(),
	}

	_, err := s.client.Mutate(mutation, map[string]interface{}{"object": object})
	return err
}

// Helper functions
func getString(data map[string]interface{}, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

func getBool(data map[string]interface{}, key string) bool {
	if v, ok := data[key].(bool); ok {
		return v
	}
	return false
}

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/bo"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// BusinessProcessService manages business process execution
type BusinessProcessService struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewBusinessProcessService creates a new BusinessProcessService
func NewBusinessProcessService(db *sqlx.DB) *BusinessProcessService {
	logger, _ := zap.NewProduction()
	return &BusinessProcessService{db: db, logger: logger}
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

	// Insert instance via direct SQL
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO process_instances (
			id, tenant_id, process_id, entity_type, entity_id,
			current_step_id, status, started_at, data, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, instance.ID, instance.TenantID, instance.ProcessID, instance.EntityType, instance.EntityID,
		instance.CurrentStepID, instance.Status, instance.StartedAt,
		string(instance.Data), instance.CreatedBy)

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
	instance, err := s.GetInstance(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	if instance.Status == "completed" || instance.Status == "cancelled" {
		return fmt.Errorf("process already %s", instance.Status)
	}

	steps, err := s.GetProcessSteps(ctx, instance.ProcessID)
	if err != nil {
		return fmt.Errorf("failed to get steps: %w", err)
	}

	currentIdx := -1
	for i, step := range steps {
		if instance.CurrentStepID != nil && step.ID == *instance.CurrentStepID {
			currentIdx = i
			break
		}
	}

	if instance.CurrentStepID != nil {
		s.recordStepHistory(ctx, instanceID, *instance.CurrentStepID, action, actor, comments, data)
	}

	switch action {
	case "approved", "completed", "skipped":
		if currentIdx < len(steps)-1 {
			nextStep := steps[currentIdx+1]
			s.updateInstanceStep(ctx, instanceID, nextStep.ID, "in_progress")
			s.recordStepHistory(ctx, instanceID, nextStep.ID, "started", actor, "", nil)
		} else {
			s.completeProcess(ctx, instanceID)
		}

	case "rejected":
		s.updateInstanceStatus(ctx, instanceID, "rejected")

	case "cancelled":
		s.updateInstanceStatus(ctx, instanceID, "cancelled")

	default:
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
	var p struct {
		ID          string `db:"id"`
		TenantID    string `db:"tenant_id"`
		Key         string `db:"key"`
		Name        string `db:"name"`
		DisplayName string `db:"display_name"`
		Description string `db:"description"`
		Category    string `db:"category"`
		Status      string `db:"status"`
		IsSystem    bool   `db:"is_system"`
	}

	err := s.db.GetContext(ctx, &p, `
		SELECT id, COALESCE(tenant_id::text,'') as tenant_id, key, name,
		       COALESCE(display_name,'') as display_name, COALESCE(description,'') as description,
		       COALESCE(category,'') as category, COALESCE(status,'') as status,
		       COALESCE(is_system, false) as is_system
		FROM business_processes WHERE key = $1 LIMIT 1
	`, key)

	if err != nil {
		return nil, nil // not found
	}

	return &bo.BusinessProcess{
		ID:          p.ID,
		TenantID:    p.TenantID,
		Key:         p.Key,
		Name:        p.Name,
		DisplayName: p.DisplayName,
		Description: p.Description,
		Category:    p.Category,
		Status:      p.Status,
		IsSystem:    p.IsSystem,
	}, nil
}

// GetProcessSteps fetches all steps for a process
func (s *BusinessProcessService) GetProcessSteps(ctx context.Context, processID string) ([]*bo.ProcessStep, error) {
	type row struct {
		ID          string `db:"id"`
		TenantID    string `db:"tenant_id"`
		ProcessID   string `db:"process_id"`
		Key         string `db:"key"`
		Name        string `db:"name"`
		DisplayName string `db:"display_name"`
		StepType    string `db:"step_type"`
		Sequence    int    `db:"sequence"`
		Config      string `db:"config"`
		IsRequired  bool   `db:"is_required"`
	}

	var rows []row
	err := s.db.SelectContext(ctx, &rows, `
		SELECT id, COALESCE(tenant_id::text,'') as tenant_id, process_id,
		       key, name, COALESCE(display_name,'') as display_name,
		       COALESCE(step_type,'') as step_type, sequence,
		       COALESCE(config::text,'{}') as config,
		       COALESCE(is_required, false) as is_required
		FROM process_steps
		WHERE process_id = $1
		ORDER BY sequence
	`, processID)

	if err != nil {
		return []*bo.ProcessStep{}, nil
	}

	steps := make([]*bo.ProcessStep, 0, len(rows))
	for _, r := range rows {
		step := &bo.ProcessStep{
			ID:          r.ID,
			TenantID:    r.TenantID,
			ProcessID:   r.ProcessID,
			Key:         r.Key,
			Name:        r.Name,
			DisplayName: r.DisplayName,
			StepType:    r.StepType,
			Sequence:    r.Sequence,
			IsRequired:  r.IsRequired,
		}
		step.Config = json.RawMessage(r.Config)
		steps = append(steps, step)
	}
	return steps, nil
}

// GetInstance fetches a process instance by ID
func (s *BusinessProcessService) GetInstance(ctx context.Context, instanceID string) (*bo.ProcessInstance, error) {
	var r struct {
		ID            string  `db:"id"`
		TenantID      string  `db:"tenant_id"`
		ProcessID     string  `db:"process_id"`
		EntityType    string  `db:"entity_type"`
		EntityID      string  `db:"entity_id"`
		CurrentStepID *string `db:"current_step_id"`
		Status        string  `db:"status"`
		CreatedBy     string  `db:"created_by"`
	}

	err := s.db.GetContext(ctx, &r, `
		SELECT id, COALESCE(tenant_id::text,'') as tenant_id, process_id,
		       entity_type, entity_id, current_step_id::text, status, COALESCE(created_by,'') as created_by
		FROM process_instances WHERE id = $1
	`, instanceID)

	if err != nil {
		return nil, fmt.Errorf("instance not found: %s", instanceID)
	}

	return &bo.ProcessInstance{
		ID:            r.ID,
		TenantID:      r.TenantID,
		ProcessID:     r.ProcessID,
		EntityType:    r.EntityType,
		EntityID:      r.EntityID,
		CurrentStepID: r.CurrentStepID,
		Status:        r.Status,
		CreatedBy:     r.CreatedBy,
	}, nil
}

// GetInstanceHistory fetches the step history for an instance
func (s *BusinessProcessService) GetInstanceHistory(ctx context.Context, instanceID string) ([]*bo.StepHistory, error) {
	type row struct {
		ID         string `db:"id"`
		InstanceID string `db:"instance_id"`
		StepID     string `db:"step_id"`
		Action     string `db:"action"`
		Actor      string `db:"actor"`
		Comments   string `db:"comments"`
	}

	var rows []row
	err := s.db.SelectContext(ctx, &rows, `
		SELECT id, instance_id, step_id, action, actor, COALESCE(comments,'') as comments
		FROM step_history
		WHERE instance_id = $1
		ORDER BY created_at ASC
	`, instanceID)

	if err != nil {
		return []*bo.StepHistory{}, nil
	}

	history := make([]*bo.StepHistory, 0, len(rows))
	for _, r := range rows {
		history = append(history, &bo.StepHistory{
			ID:         r.ID,
			InstanceID: r.InstanceID,
			StepID:     r.StepID,
			Action:     r.Action,
			Actor:      r.Actor,
			Comments:   r.Comments,
		})
	}
	return history, nil
}

// ListProcesses fetches all available business processes
func (s *BusinessProcessService) ListProcesses(ctx context.Context, category string) ([]*bo.BusinessProcess, error) {
	type row struct {
		ID          string `db:"id"`
		Key         string `db:"key"`
		Name        string `db:"name"`
		DisplayName string `db:"display_name"`
		Description string `db:"description"`
		Category    string `db:"category"`
		IsSystem    bool   `db:"is_system"`
	}

	var rows []row
	var err error
	if category != "" {
		err = s.db.SelectContext(ctx, &rows, `
			SELECT id, key, name, COALESCE(display_name,'') as display_name,
			       COALESCE(description,'') as description,
			       COALESCE(category,'') as category, COALESCE(is_system, false) as is_system
			FROM business_processes
			WHERE category = $1 AND status = 'active'
			ORDER BY name
		`, category)
	} else {
		err = s.db.SelectContext(ctx, &rows, `
			SELECT id, key, name, COALESCE(display_name,'') as display_name,
			       COALESCE(description,'') as description,
			       COALESCE(category,'') as category, COALESCE(is_system, false) as is_system
			FROM business_processes
			WHERE status = 'active'
			ORDER BY name
		`)
	}

	if err != nil {
		return []*bo.BusinessProcess{}, nil
	}

	processes := make([]*bo.BusinessProcess, 0, len(rows))
	for _, r := range rows {
		processes = append(processes, &bo.BusinessProcess{
			ID:          r.ID,
			Key:         r.Key,
			Name:        r.Name,
			DisplayName: r.DisplayName,
			Description: r.Description,
			Category:    r.Category,
			IsSystem:    r.IsSystem,
		})
	}
	return processes, nil
}

// ListInstancesForEntity fetches all process instances for an entity
func (s *BusinessProcessService) ListInstancesForEntity(ctx context.Context, entityType, entityID string) ([]*bo.ProcessInstance, error) {
	type row struct {
		ID        string `db:"id"`
		ProcessID string `db:"process_id"`
		Status    string `db:"status"`
		CreatedBy string `db:"created_by"`
	}

	var rows []row
	err := s.db.SelectContext(ctx, &rows, `
		SELECT id, process_id, status, COALESCE(created_by,'') as created_by
		FROM process_instances
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY started_at DESC
	`, entityType, entityID)

	if err != nil {
		return []*bo.ProcessInstance{}, nil
	}

	instances := make([]*bo.ProcessInstance, 0, len(rows))
	for _, r := range rows {
		instances = append(instances, &bo.ProcessInstance{
			ID:         r.ID,
			ProcessID:  r.ProcessID,
			EntityType: entityType,
			EntityID:   entityID,
			Status:     r.Status,
			CreatedBy:  r.CreatedBy,
		})
	}
	return instances, nil
}

// Helper methods

func (s *BusinessProcessService) completeProcess(ctx context.Context, instanceID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE process_instances SET status = 'completed', completed_at = NOW()
		WHERE id = $1
	`, instanceID)
	return err
}

func (s *BusinessProcessService) updateInstanceStep(ctx context.Context, instanceID, stepID, status string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE process_instances SET current_step_id = $2, status = $3
		WHERE id = $1
	`, instanceID, stepID, status)
	return err
}

func (s *BusinessProcessService) updateInstanceStatus(ctx context.Context, instanceID, status string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE process_instances SET status = $2 WHERE id = $1
	`, instanceID, status)
	return err
}

func (s *BusinessProcessService) updateInstanceData(ctx context.Context, instanceID string, data map[string]interface{}) error {
	dataJSON, _ := json.Marshal(data)
	_, err := s.db.ExecContext(ctx, `
		UPDATE process_instances SET data = $2 WHERE id = $1
	`, instanceID, string(dataJSON))
	return err
}

func (s *BusinessProcessService) recordStepHistory(ctx context.Context, instanceID, stepID, action, actor, comments string, data map[string]interface{}) error {
	var dataJSON json.RawMessage
	if data != nil {
		dataJSON, _ = json.Marshal(data)
	} else {
		dataJSON = json.RawMessage("{}")
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO step_history (id, instance_id, step_id, action, actor, comments, data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`, uuid.New().String(), instanceID, stepID, action, actor, comments, string(dataJSON))
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

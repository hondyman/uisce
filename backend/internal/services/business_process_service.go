package services

import (
	"context"
	"fmt"

	"github.com/hondyman/uisce/backend/internal/bo"
	"go.uber.org/zap"
)

type BusinessProcessService struct {
	logger *zap.Logger
}

func NewBusinessProcessService() *BusinessProcessService {
	logger, _ := zap.NewProduction()
	return &BusinessProcessService{
		logger: logger,
	}
}

func (s *BusinessProcessService) StartProcess(ctx context.Context, processKey, entityType, entityID, createdBy string, initialData map[string]interface{}) (*bo.ProcessInstance, error) {
	return nil, fmt.Errorf("StartProcess: Hasura removed from BusinessProcessService")
}

func (s *BusinessProcessService) AdvanceProcess(ctx context.Context, instanceID, action, actor, comments string, data map[string]interface{}) error {
	return fmt.Errorf("AdvanceProcess: Hasura removed from BusinessProcessService")
}

func (s *BusinessProcessService) CompleteProcess(ctx context.Context, instanceID string) error {
	return fmt.Errorf("CompleteProcess: Hasura removed from BusinessProcessService")
}

func (s *BusinessProcessService) GetProcessByKey(ctx context.Context, key string) (*bo.BusinessProcess, error) {
	return nil, fmt.Errorf("GetProcessByKey: Hasura removed from BusinessProcessService")
}

func (s *BusinessProcessService) GetProcessSteps(ctx context.Context, processID string) ([]*bo.ProcessStep, error) {
	return nil, fmt.Errorf("GetProcessSteps: Hasura removed from BusinessProcessService")
}

func (s *BusinessProcessService) GetInstance(ctx context.Context, instanceID string) (*bo.ProcessInstance, error) {
	return nil, fmt.Errorf("GetInstance: Hasura removed from BusinessProcessService")
}

func (s *BusinessProcessService) GetInstanceHistory(ctx context.Context, instanceID string) ([]*bo.StepHistory, error) {
	return nil, fmt.Errorf("GetInstanceHistory: Hasura removed from BusinessProcessService")
}

func (s *BusinessProcessService) ListProcesses(ctx context.Context, category string) ([]*bo.BusinessProcess, error) {
	return nil, fmt.Errorf("ListProcesses: Hasura removed from BusinessProcessService")
}

func (s *BusinessProcessService) ListInstancesForEntity(ctx context.Context, entityType, entityID string) ([]*bo.ProcessInstance, error) {
	return nil, fmt.Errorf("ListInstancesForEntity: Hasura removed from BusinessProcessService")
}

func (s *BusinessProcessService) completeProcess(ctx context.Context, instanceID string) error {
	return fmt.Errorf("completeProcess: Hasura removed from BusinessProcessService")
}

func (s *BusinessProcessService) updateInstanceStep(ctx context.Context, instanceID, stepID, status string) error {
	return fmt.Errorf("updateInstanceStep: Hasura removed from BusinessProcessService")
}

func (s *BusinessProcessService) updateInstanceStatus(ctx context.Context, instanceID, status string) error {
	return fmt.Errorf("updateInstanceStatus: Hasura removed from BusinessProcessService")
}

func (s *BusinessProcessService) updateInstanceData(ctx context.Context, instanceID string, data map[string]interface{}) error {
	return fmt.Errorf("updateInstanceData: Hasura removed from BusinessProcessService")
}

func (s *BusinessProcessService) recordStepHistory(ctx context.Context, instanceID, stepID, action, actor, comments string, data map[string]interface{}) error {
	return fmt.Errorf("recordStepHistory: Hasura removed from BusinessProcessService")
}

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

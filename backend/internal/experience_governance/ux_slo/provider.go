package uxslo

import (
	"context"

	"github.com/google/uuid"
)

type SLOProvider struct {
	// DB or Repository would be here
}

func NewSLOProvider() *SLOProvider {
	return &SLOProvider{}
}

func (p *SLOProvider) EvaluateContracts(ctx context.Context, pageID uuid.UUID) ([]SLOStatus, error) {
	// Mock logic for MVP
	// In real world, query metrics DB (e.g. Prometheus/M3/ClickHouse)

	return []SLOStatus{
		{
			ContractID: uuid.New(),
			Status:     "passing",
			Current:    250,
			Target:     300,
			Gap:        -50,
		},
	}, nil
}

func (p *SLOProvider) CreateContract(ctx context.Context, contract *UXContract) error {
	// Mock save
	return nil
}

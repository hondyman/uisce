package services

import (
	"os"

	"go.uber.org/zap"
)

type BusinessObjectFieldService struct {
	logger      *zap.Logger
	isAdminCore bool
}

func NewBusinessObjectFieldService() *BusinessObjectFieldService {
	logger, _ := zap.NewProduction()
	isAdminCore := os.Getenv("ADMIN_CORE") == "true"
	return &BusinessObjectFieldService{
		logger:      logger,
		isAdminCore: isAdminCore,
	}
}

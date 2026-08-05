package handlers

import (
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/services"
)

type QueryHandler struct{}

func NewQueryHandler(qs *services.QueryService, securityDeps *security.HandlerDependencies) *QueryHandler {
	return &QueryHandler{}
}

type SavedQueryHandler struct{}

func NewSavedQueryHandler(qs *services.QueryService, securityDeps *security.HandlerDependencies) *SavedQueryHandler {
	return &SavedQueryHandler{}
}

type SaveExtensionRequest struct{}

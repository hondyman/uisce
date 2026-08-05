package handlers

import "net/http"

type QueryHandler struct{}

func NewQueryHandler(qs interface{}, securityDeps SecurityContextDeps) *QueryHandler {
	return &QueryHandler{}
}

func (h *QueryHandler) HandleExecuteQuery(w http.ResponseWriter, r *http.Request) {}
func (h *QueryHandler) HandleCompileQuery(w http.ResponseWriter, r *http.Request) {}
func (h *QueryHandler) HandleExportQuery(w http.ResponseWriter, r *http.Request) {}
func (h *QueryHandler) HandleListHistory(w http.ResponseWriter, r *http.Request) {}

type SavedQueryHandler struct{}

func NewSavedQueryHandler(qs interface{}, securityDeps SecurityContextDeps) *SavedQueryHandler {
	return &SavedQueryHandler{}
}

func (h *SavedQueryHandler) HandleListSavedQueries(w http.ResponseWriter, r *http.Request) {}
func (h *SavedQueryHandler) HandleCreateSavedQuery(w http.ResponseWriter, r *http.Request) {}
func (h *SavedQueryHandler) HandleGetDuplicates(w http.ResponseWriter, r *http.Request) {}
func (h *SavedQueryHandler) HandleGetSavedQuery(w http.ResponseWriter, r *http.Request) {}
func (h *SavedQueryHandler) HandleUpdateSavedQuery(w http.ResponseWriter, r *http.Request) {}
func (h *SavedQueryHandler) HandleDeleteSavedQuery(w http.ResponseWriter, r *http.Request) {}
func (h *SavedQueryHandler) HandleCloneSavedQuery(w http.ResponseWriter, r *http.Request) {}
func (h *SavedQueryHandler) HandleShareQuery(w http.ResponseWriter, r *http.Request) {}
func (h *SavedQueryHandler) HandleGetPreview(w http.ResponseWriter, r *http.Request) {}
func (h *SavedQueryHandler) HandleGetDiff(w http.ResponseWriter, r *http.Request) {}

type SaveExtensionRequest struct {
	ModelObject interface{}
}

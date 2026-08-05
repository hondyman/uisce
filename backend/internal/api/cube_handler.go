package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/cube"
	"github.com/hondyman/uisce/backend/internal/services"
)

type CubeHandler struct {
	cubeClient    *cube.Client
	cubeGenerator *services.CubeGenerator
}

func NewCubeHandler(cubeClient *cube.Client, cubeGenerator *services.CubeGenerator) *CubeHandler {
	return &CubeHandler{
		cubeClient:    cubeClient,
		cubeGenerator: cubeGenerator,
	}
}

type CubeQueryRequest struct {
	Measures       []string             `json:"measures"`
	Dimensions     []string             `json:"dimensions"`
	Filters        []cube.Filter        `json:"filters"`
	TimeDimensions []cube.TimeDimension `json:"timeDimensions"`
	Order          map[string]string    `json:"order"`
	Limit          int                  `json:"limit"`
	Timezone       string               `json:"timezone"`
}

type CubeQueryResponse struct {
	Data       []map[string]interface{} `json:"data"`
	Annotation *cube.Annotation         `json:"annotation,omitempty"`
	Query      *cube.Query              `json:"query,omitempty"`
	Count      int                      `json:"count"`
}

func (h *CubeHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/cube", func(r chi.Router) {
		r.Post("/query", h.ExecuteQuery)
		r.Get("/meta", h.GetMeta)
		r.Get("/pre-aggregations", h.GetPreAggregations)
		r.Post("/dry-run", h.DryRun)
		r.Post("/generate", h.GenerateCubeSchema)
		r.Post("/generate/{boID}", h.GenerateCubeFromBO)
		r.Get("/preview", h.PreviewCubeSchema)
	})
}

func (h *CubeHandler) ExecuteQuery(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (h *CubeHandler) GetMeta(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (h *CubeHandler) GetPreAggregations(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (h *CubeHandler) DryRun(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (h *CubeHandler) GenerateCubeSchema(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (h *CubeHandler) GenerateCubeFromBO(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (h *CubeHandler) PreviewCubeSchema(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

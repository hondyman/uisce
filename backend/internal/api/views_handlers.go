package api

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/services"
)

// registerViewsRoutes registers view-related handlers previously located in api.go.
//
//lint:ignore U1000 retained as an example and for future integration
func (s *Server) registerViewsRoutes(r chi.Router, viewService *services.ViewService) {
	// Suggestions for a specific view
	r.Get("/views/{name}/suggestions", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if strings.TrimSpace(name) == "" {
			writeJSONError(w, http.StatusBadRequest, "View name is required", "missing_name", nil)
			return
		}
		suggestions, err := viewService.GetSuggestedQueries(r.Context(), name)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to get suggestions", "suggestions_failed", err.Error())
			return
		}
		respond(w, r, suggestions, nil)
	})

	// List views with optional filters and pagination
	r.Get("/views", func(w http.ResponseWriter, r *http.Request) {
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		datasourceID := strings.TrimSpace(r.URL.Query().Get("datasource_id"))
		q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
		page := parseIntDefault(r.URL.Query().Get("page"), 1)
		if page < 1 {
			page = 1
		}
		pageSize := parseIntDefault(r.URL.Query().Get("page_size"), 50)
		if pageSize <= 0 || pageSize > 500 {
			pageSize = 50
		}
		offset := (page - 1) * pageSize

		where := make(map[string]interface{})
		andConditions := []map[string]interface{}{}
		if tenantID != "" {
			andConditions = append(andConditions, map[string]interface{}{"tenant_id": map[string]string{"_eq": tenantID}})
		}
		if datasourceID != "" {
			andConditions = append(andConditions, map[string]interface{}{"tenant_datasource_id": map[string]string{"_eq": datasourceID}})
		}
		if q != "" {
			searchPattern := q
			andConditions = append(andConditions, map[string]interface{}{
				"_or": []map[string]interface{}{
					{"name": map[string]string{"_ilike": "%" + searchPattern + "%"}},
					{"view": map[string]interface{}{"_contains": map[string]string{"title": searchPattern}}},
					{"view": map[string]interface{}{"_contains": map[string]string{"description": searchPattern}}},
				},
			})
		}
		if len(andConditions) > 0 {
			where["_and"] = andConditions
		}

		type viewItem struct {
			ID          string    `json:"id,omitempty"`
			Name        string    `json:"name"`
			Title       string    `json:"title,omitempty"`
			Description string    `json:"description,omitempty"`
			CubeCount   int       `json:"cube_count"`
			FolderCount int       `json:"folder_count"`
			ModifiedAt  time.Time `json:"modified_at,omitempty"`
			ETag        string    `json:"etag"`
		}
		var items []viewItem
		var total int

		gqlResponse, err := executeGraphQLQuery(map[string]interface{}{"limit": pageSize, "offset": offset, "where": where}, `
			query GetViews($limit: Int, $offset: Int, $where: views_bool_exp) {
				views(limit: $limit, offset: $offset, where: $where, order_by: {name: asc}) {
						id
						name
						view
						updated_at
				}
				views_aggregate(where: $where) {
					aggregate { count }
				}
			}
		`)

		if err == nil && gqlResponse != nil {
			var responseData struct {
				Views []struct {
					ID        string          `json:"id"`
					Name      string          `json:"name"`
					View      json.RawMessage `json:"view"`
					UpdatedAt time.Time       `json:"updated_at"`
				} `json:"views"`
				ViewsAggregate struct {
					Aggregate struct {
						Count int `json:"count"`
					} `json:"aggregate"`
				} `json:"views_aggregate"`
			}
			dataBytes, _ := json.Marshal(gqlResponse["data"])
			if json.Unmarshal(dataBytes, &responseData) == nil {
				total = responseData.ViewsAggregate.Aggregate.Count
				for _, item := range responseData.Views {
					var v map[string]any
					if err := json.Unmarshal(item.View, &v); err != nil {
						continue
					}
					title := ""
					if t, ok := v["title"].(string); ok {
						title = t
					}
					desc := ""
					if d, ok := v["description"].(string); ok {
						desc = d
					}
					cubes, _ := v["cubes"].([]any)
					folders, _ := v["folders"].([]any)
					etag := fmt.Sprintf("W/\"%x\"", sha1.Sum(item.View))
					items = append(items, viewItem{ID: item.ID, Name: item.Name, Title: title, Description: desc, CubeCount: len(cubes), FolderCount: len(folders), ModifiedAt: item.UpdatedAt, ETag: etag})
				}
			}
		} else {
			logging.GetLogger().Sugar().Warnf("GraphQL query for views failed, falling back to direct DB query: %v", err)
			query := "SELECT id, name, view, updated_at FROM public.views WHERE tenant_id = $1 AND tenant_datasource_id = $2 ORDER BY name"
			rows, qerr := s.DB.QueryContext(r.Context(), query, tenantID, datasourceID)
			if qerr == nil {
				defer rows.Close()
				for rows.Next() {
					var id, name string
					var viewRaw []byte
					var updatedAt time.Time
					if err := rows.Scan(&id, &name, &viewRaw, &updatedAt); err != nil {
						continue
					}
					var v map[string]any
					if json.Unmarshal(viewRaw, &v) != nil {
						continue
					}
					title, _ := v["title"].(string)
					desc, _ := v["description"].(string)
					cubes, _ := v["cubes"].([]any)
					folders, _ := v["folders"].([]any)
					etag := fmt.Sprintf("W/\"%x\"", sha1.Sum(viewRaw))
					items = append(items, viewItem{ID: id, Name: name, Title: title, Description: desc, CubeCount: len(cubes), FolderCount: len(folders), ModifiedAt: updatedAt, ETag: etag})
				}
			}
		}

		respond(w, r, map[string]any{"views": items, "total": total}, nil)
	})
}

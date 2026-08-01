package api

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/services"
)

// registerViewsRoutes registers view-related handlers previously located in api.go.
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

		baseQuery := "FROM public.views WHERE tenant_id = $1 AND tenant_datasource_id = $2"
		countQuery := "SELECT COUNT(*) " + baseQuery
		selectQuery := "SELECT id, name, view, updated_at " + baseQuery

		args := []interface{}{tenantID, datasourceID}
		argIdx := 3

		if q != "" {
			searchClause := fmt.Sprintf(" AND (LOWER(name) ILIKE $%d OR view::text ILIKE $%d)", argIdx, argIdx)
			baseQuery += searchClause
			countQuery += searchClause
			selectQuery += searchClause
			args = append(args, "%"+q+"%")
			argIdx++
		}

		countQuery += " AND tenant_id = $1 AND tenant_datasource_id = $2"
		err := s.DB.QueryRowContext(r.Context(), countQuery, args...).Scan(&total)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to count views", "count_error", err.Error())
			return
		}

		selectQuery += fmt.Sprintf(" ORDER BY name LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
		args = append(args, pageSize, offset)

		rows, err := s.DB.QueryContext(r.Context(), selectQuery, args...)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to query views", "query_error", err.Error())
			return
		}
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

		if items == nil {
			items = []viewItem{}
		}

		respond(w, r, map[string]any{"views": items, "total": total}, nil)
	})
}

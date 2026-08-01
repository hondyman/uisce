package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

func RegisterSemanticTagsRoutes(r chi.Router, db *sqlx.DB) {
	r.Route("/semantic/tags", func(r chi.Router) {
		r.Get("/", handleListSemanticTags(db))
		r.Post("/suggest", handleSuggestSemanticTags(db))
	})
}

type semanticTag struct {
	ID          string `json:"id"`
	TagKey      string `json:"tagKey"`
	TagLabel    string `json:"tagLabel"`
	TagCategory string `json:"tagCategory"`
	Description string `json:"description,omitempty"`
	ColorCode   string `json:"colorCode,omitempty"`
	IconName    string `json:"iconName,omitempty"`
	TermID      string `json:"termId,omitempty"`
	TermName    string `json:"termName,omitempty"`
}

type tagSuggestionInput struct {
	NodeName     string   `json:"nodeName"`
	DisplayName  string   `json:"displayName"`
	Description  string   `json:"description"`
	DataType     string   `json:"dataType"`
	Domain       string   `json:"domain"`
	Expression   string   `json:"expression"`
	ExistingTags []string `json:"existingTags"`
}

func handleListSemanticTags(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "X-Tenant-ID header is required", "missing_tenant", nil)
			return
		}

		var tags []semanticTag
		query := `
			SELECT 
				stt.tag_id as id,
				stt.tag_key as tag_key,
				stt.tag_label as tag_label,
				COALESCE(stt.tag_category, st.governance_level) as tag_category,
				stt.color_code as color_code,
				st.definition as description,
				stt.semantic_term_id as term_id,
				st.name as term_name
			FROM public.semantic_term_tags stt
			JOIN public.semantic_terms st ON st.id = stt.semantic_term_id
			WHERE stt.tenant_id = $1
			ORDER BY stt.tag_category, stt.tag_label
		`
		err := db.SelectContext(r.Context(), &tags, query, tenantID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to list semantic tags", "list_error", err.Error())
			return
		}

		if tags == nil {
			tags = []semanticTag{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": tags,
		})
	}
}

func handleSuggestSemanticTags(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "X-Tenant-ID header is required", "missing_tenant", nil)
			return
		}

		var input tagSuggestionInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var suggestions []semanticTag
		var reasons []string

		baseQuery := `
			SELECT 
				stt.tag_id as id,
				stt.tag_key as tag_key,
				stt.tag_label as tag_label,
				COALESCE(stt.tag_category, st.governance_level) as tag_category,
				stt.color_code as color_code,
				st.definition as description,
				stt.semantic_term_id as term_id,
				st.name as term_name
			FROM public.semantic_term_tags stt
			JOIN public.semantic_terms st ON st.id = stt.semantic_term_id
			WHERE stt.tenant_id = $1
		`

		args := []interface{}{tenantID}
		argIdx := 2

		if input.Domain != "" {
			baseQuery += fmt.Sprintf(" AND (stt.tag_category ILIKE $%d OR st.governance_level ILIKE $%d)", argIdx, argIdx)
			args = append(args, "%"+input.Domain+"%")
			argIdx++
		}

		if input.DataType != "" {
			baseQuery += fmt.Sprintf(" AND st.data_type ILIKE $%d", argIdx)
			args = append(args, "%"+input.DataType+"%")
			argIdx++
		}

		if len(input.ExistingTags) > 0 {
			baseQuery += " AND stt.tag_key NOT IN ("
			for i, t := range input.ExistingTags {
				if i > 0 {
					baseQuery += ", "
				}
				baseQuery += fmt.Sprintf("$%d", argIdx)
				args = append(args, t)
				argIdx++
			}
			baseQuery += ")"
		}

		baseQuery += " ORDER BY stt.tag_category, stt.tag_label LIMIT 10"

		err := db.SelectContext(r.Context(), &suggestions, baseQuery, args...)
		if err != nil {
			reasons = append(reasons, "database query failed: "+err.Error())
			suggestions = []semanticTag{}
		}

		if len(suggestions) == 0 {
			reasons = append(reasons, "no matching tags found for the given criteria")
			suggestions = []semanticTag{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"suggestions": suggestions,
				"reasons":     reasons,
			},
		})
	}
}

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ProcessTemplateHandlers manages process template API operations
type ProcessTemplateHandlers struct {
	db *sqlx.DB
}

// NewProcessTemplateHandlers creates a new handler
func NewProcessTemplateHandlers(db *sqlx.DB) *ProcessTemplateHandlers {
	return &ProcessTemplateHandlers{db: db}
}

// ===========================================================================
// DATA STRUCTURES
// ===========================================================================

// ProcessTemplate represents a reusable workflow template
type ProcessTemplate struct {
	ID                        string                 `json:"id" db:"id"`
	TemplateKey               string                 `json:"template_key" db:"template_key"`
	Name                      string                 `json:"name" db:"name"`
	Description               string                 `json:"description" db:"description"`
	Category                  string                 `json:"category" db:"category"`
	Tags                      []string               `json:"tags" db:"tags"`
	IconName                  string                 `json:"icon_name" db:"icon_name"`
	DifficultyLevel           string                 `json:"difficulty_level" db:"difficulty_level"`
	EstimatedSetupTimeMinutes int                    `json:"estimated_setup_time_minutes" db:"estimated_setup_time_minutes"`
	IsOfficial                bool                   `json:"is_official" db:"is_official"`
	IsFeatured                bool                   `json:"is_featured" db:"is_featured"`
	TemplateDefinition        map[string]interface{} `json:"template_definition" db:"template_definition"`
	CustomizationGuide        string                 `json:"customization_guide" db:"customization_guide"`
	ExampleUseCases           []string               `json:"example_use_cases" db:"example_use_cases"`
	AuthorName                string                 `json:"author_name" db:"author_name"`
	AuthorOrganization        string                 `json:"author_organization" db:"author_organization"`
	Version                   string                 `json:"version" db:"version"`
	CompatibleWithVersion     string                 `json:"compatible_with_version" db:"compatible_with_version"`
	UsageCount                int                    `json:"usage_count" db:"usage_count"`
	CloneCount                int                    `json:"clone_count" db:"clone_count"`
	FavoriteCount             int                    `json:"favorite_count" db:"favorite_count"`
	RatingAverage             float64                `json:"rating_average" db:"rating_average"`
	RatingCount               int                    `json:"rating_count" db:"rating_count"`
	SearchKeywords            string                 `json:"search_keywords" db:"search_keywords"`
	DocumentationURL          string                 `json:"documentation_url" db:"documentation_url"`
	DemoVideoURL              string                 `json:"demo_video_url" db:"demo_video_url"`
	ScreenshotURL             string                 `json:"screenshot_url" db:"screenshot_url"`
	CreatedAt                 time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt                 time.Time              `json:"updated_at" db:"updated_at"`
	PublishedAt               *time.Time             `json:"published_at" db:"published_at"`
}

// TemplateClone tracks when a user clones a template
type TemplateClone struct {
	ID                    string     `json:"id" db:"id"`
	TemplateID            string     `json:"template_id" db:"template_id"`
	TenantID              string     `json:"tenant_id" db:"tenant_id"`
	DatasourceID          string     `json:"datasource_id" db:"datasource_id"`
	ProcessID             *string    `json:"process_id" db:"process_id"`
	ClonedBy              string     `json:"cloned_by" db:"cloned_by"`
	WasCustomized         bool       `json:"was_customized" db:"was_customized"`
	CustomizationNotes    string     `json:"customization_notes" db:"customization_notes"`
	ClonedAt              time.Time  `json:"cloned_at" db:"cloned_at"`
	LastUsedAt            *time.Time `json:"last_used_at" db:"last_used_at"`
	TimeToFirstUseMinutes *int       `json:"time_to_first_use_minutes" db:"time_to_first_use_minutes"`
	UsageCount            int        `json:"usage_count" db:"usage_count"`
	// Joined fields from template
	TemplateName string `json:"template_name" db:"template_name"`
	TemplateKey  string `json:"template_key" db:"template_key"`
}

// TemplateRating represents a user rating/review
type TemplateRating struct {
	ID               string    `json:"id" db:"id"`
	TemplateID       string    `json:"template_id" db:"template_id"`
	TenantID         string    `json:"tenant_id" db:"tenant_id"`
	DatasourceID     string    `json:"datasource_id" db:"datasource_id"`
	Rating           int       `json:"rating" db:"rating"`
	ReviewText       string    `json:"review_text" db:"review_text"`
	ReviewTitle      string    `json:"review_title" db:"review_title"`
	ReviewerName     string    `json:"reviewer_name" db:"reviewer_name"`
	ReviewerRole     string    `json:"reviewer_role" db:"reviewer_role"`
	HelpfulCount     int       `json:"helpful_count" db:"helpful_count"`
	NotHelpfulCount  int       `json:"not_helpful_count" db:"not_helpful_count"`
	IsVerifiedUser   bool      `json:"is_verified_user" db:"is_verified_user"`
	IsModerated      bool      `json:"is_moderated" db:"is_moderated"`
	ModerationStatus string    `json:"moderation_status" db:"moderation_status"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// TemplateCategory represents category metadata
type TemplateCategory struct {
	ID            string    `json:"id" db:"id"`
	CategoryKey   string    `json:"category_key" db:"category_key"`
	DisplayName   string    `json:"display_name" db:"display_name"`
	Description   string    `json:"description" db:"description"`
	IconName      string    `json:"icon_name" db:"icon_name"`
	SortOrder     int       `json:"sort_order" db:"sort_order"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	TemplateCount int       `json:"template_count" db:"template_count"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// ===========================================================================
// API HANDLERS
// ===========================================================================

// RegisterRoutes registers all template routes
func (h *ProcessTemplateHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/templates", func(r chi.Router) {
		// Browse templates
		r.Get("/", h.GetTemplates)
		r.Get("/{key}", h.GetTemplate)
		r.Get("/category/{category}", h.GetTemplatesByCategory)
		r.Get("/featured", h.GetFeaturedTemplates)

		// Clone template
		r.Post("/clone/{id}", h.CloneTemplate)

		// User's cloned templates
		r.Get("/clones", h.GetUserClones)
		r.Get("/clones/{id}", h.GetClone)
		r.Put("/clones/{id}/notes", h.UpdateCloneNotes)
		r.Delete("/clones/{id}", h.DeleteClone)

		// Ratings and reviews
		r.Post("/{id}/rate", h.RateTemplate)
		r.Get("/{id}/ratings", h.GetTemplateRatings)
		r.Put("/ratings/{ratingId}/helpful", h.MarkRatingHelpful)

		// Categories
		r.Get("/categories", h.GetCategories)
		r.Get("/categories/{key}/templates", h.GetTemplatesByCategory)

		// Analytics
		r.Get("/{id}/stats", h.GetTemplateStats)
	})
}

// GetTemplates retrieves templates with filtering and search
func (h *ProcessTemplateHandlers) GetTemplates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Query parameters
	category := r.URL.Query().Get("category")
	search := r.URL.Query().Get("search")
	difficulty := r.URL.Query().Get("difficulty")
	sortBy := r.URL.Query().Get("sort_by") // rating, usage, recent, name
	if sortBy == "" {
		sortBy = "rating"
	}

	// Build query
	query := `
		SELECT 
			id, template_key, name, description, category, tags, icon_name,
			difficulty_level, estimated_setup_time_minutes, is_official, is_featured,
			usage_count, clone_count, favorite_count, rating_average, rating_count,
			screenshot_url, version, created_at, published_at
		FROM process_templates
		WHERE published_at IS NOT NULL
	`
	args := []interface{}{}
	argNum := 1

	if category != "" && category != "all" {
		query += fmt.Sprintf(" AND category = $%d", argNum)
		args = append(args, category)
		argNum++
	}

	if search != "" {
		query += fmt.Sprintf(" AND (to_tsvector('english', name || ' ' || description || ' ' || COALESCE(search_keywords, '')) @@ plainto_tsquery('english', $%d) OR name ILIKE $%d OR template_key ILIKE $%d)", argNum, argNum+1, argNum+2)
		args = append(args, search, "%"+search+"%", "%"+search+"%")
		argNum += 3
	}

	if difficulty != "" {
		query += fmt.Sprintf(" AND difficulty_level = $%d", argNum)
		args = append(args, difficulty)
		argNum++
	}

	// Sorting
	switch sortBy {
	case "rating":
		query += " ORDER BY rating_average DESC, rating_count DESC, name ASC"
	case "usage":
		query += " ORDER BY usage_count DESC, clone_count DESC, name ASC"
	case "recent":
		query += " ORDER BY published_at DESC, created_at DESC"
	case "name":
		query += " ORDER BY name ASC"
	default:
		query += " ORDER BY rating_average DESC, name ASC"
	}

	query += " LIMIT 100"

	var templates []ProcessTemplate
	err := h.db.SelectContext(ctx, &templates, query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch templates: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, templates)
}

// GetTemplate retrieves a single template by key
func (h *ProcessTemplateHandlers) GetTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := chi.URLParam(r, "key")

	var template ProcessTemplate
	err := h.db.GetContext(ctx, &template, `
		SELECT * FROM process_templates 
		WHERE template_key = $1 AND published_at IS NOT NULL
	`, key)

	if err != nil {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	// Increment view count (usage_count)
	_, _ = h.db.ExecContext(ctx, `
		UPDATE process_templates 
		SET usage_count = usage_count + 1 
		WHERE id = $1
	`, template.ID)

	respondJSON(w, http.StatusOK, template)
}

// GetTemplatesByCategory retrieves templates for a specific category
func (h *ProcessTemplateHandlers) GetTemplatesByCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	category := chi.URLParam(r, "category")

	var templates []ProcessTemplate
	err := h.db.SelectContext(ctx, &templates, `
		SELECT 
			id, template_key, name, description, category, tags, icon_name,
			difficulty_level, estimated_setup_time_minutes, is_official, is_featured,
			usage_count, clone_count, rating_average, rating_count, screenshot_url
		FROM process_templates
		WHERE category = $1 AND published_at IS NOT NULL
		ORDER BY rating_average DESC, usage_count DESC
		LIMIT 50
	`, category)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch templates: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, templates)
}

// GetFeaturedTemplates retrieves featured templates
func (h *ProcessTemplateHandlers) GetFeaturedTemplates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var templates []ProcessTemplate
	err := h.db.SelectContext(ctx, &templates, `
		SELECT 
			id, template_key, name, description, category, tags, icon_name,
			difficulty_level, estimated_setup_time_minutes, is_official, is_featured,
			usage_count, clone_count, rating_average, rating_count, screenshot_url
		FROM process_templates
		WHERE is_featured = true AND published_at IS NOT NULL
		ORDER BY rating_average DESC, usage_count DESC
		LIMIT 10
	`)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch featured templates: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, templates)
}

// CloneTemplate clones a template for a tenant
func (h *ProcessTemplateHandlers) CloneTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	templateID := chi.URLParam(r, "id")

	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "tenant_id and datasource_id are required", http.StatusBadRequest)
		return
	}

	// Parse request body for customization
	var req struct {
		ProcessName        string `json:"process_name"`
		CustomizationNotes string `json:"customization_notes"`
		ClonedBy           string `json:"cloned_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get template
	var template ProcessTemplate
	err := h.db.GetContext(ctx, &template, "SELECT * FROM process_templates WHERE id = $1", templateID)
	if err != nil {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	// Create process from template definition
	// In production, this would call the BP Builder API to create the process
	// For now, we'll just track the clone
	processID := uuid.New().String()

	// Record clone
	cloneID := uuid.New().String()
	_, err = h.db.ExecContext(ctx, `
		INSERT INTO template_clones 
		(id, template_id, tenant_id, datasource_id, process_id, cloned_by, customization_notes, was_customized)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, cloneID, templateID, tenantID, datasourceID, processID, req.ClonedBy, req.CustomizationNotes, req.ProcessName != template.Name)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to record clone: %v", err), http.StatusInternalServerError)
		return
	}

	// Update template clone count
	_, _ = h.db.ExecContext(ctx, `
		UPDATE process_templates 
		SET clone_count = clone_count + 1 
		WHERE id = $1
	`, templateID)

	// Return the template definition with customizations
	result := map[string]interface{}{
		"clone_id":            cloneID,
		"process_id":          processID,
		"template_definition": template.TemplateDefinition,
		"template_name":       template.Name,
		"process_name":        req.ProcessName,
		"customization_guide": template.CustomizationGuide,
	}

	respondJSON(w, http.StatusCreated, result)
}

// GetUserClones retrieves templates cloned by a tenant
func (h *ProcessTemplateHandlers) GetUserClones(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "tenant_id and datasource_id are required", http.StatusBadRequest)
		return
	}

	var clones []TemplateClone
	err := h.db.SelectContext(ctx, &clones, `
		SELECT 
			tc.*, 
			pt.name as template_name, 
			pt.template_key
		FROM template_clones tc
		JOIN process_templates pt ON tc.template_id = pt.id
		WHERE tc.tenant_id = $1 AND tc.datasource_id = $2
		ORDER BY tc.cloned_at DESC
		LIMIT 100
	`, tenantID, datasourceID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch clones: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, clones)
}

// GetClone retrieves a single clone
func (h *ProcessTemplateHandlers) GetClone(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cloneID := chi.URLParam(r, "id")

	var clone TemplateClone
	err := h.db.GetContext(ctx, &clone, `
		SELECT 
			tc.*, 
			pt.name as template_name, 
			pt.template_key
		FROM template_clones tc
		JOIN process_templates pt ON tc.template_id = pt.id
		WHERE tc.id = $1
	`, cloneID)

	if err != nil {
		http.Error(w, "Clone not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, clone)
}

// UpdateCloneNotes updates customization notes for a clone
func (h *ProcessTemplateHandlers) UpdateCloneNotes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cloneID := chi.URLParam(r, "id")

	var req struct {
		CustomizationNotes string `json:"customization_notes"`
		WasCustomized      bool   `json:"was_customized"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err := h.db.ExecContext(ctx, `
		UPDATE template_clones 
		SET customization_notes = $1, was_customized = $2 
		WHERE id = $3
	`, req.CustomizationNotes, req.WasCustomized, cloneID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update clone: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Clone updated successfully"})
}

// DeleteClone deletes a clone record
func (h *ProcessTemplateHandlers) DeleteClone(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cloneID := chi.URLParam(r, "id")

	_, err := h.db.ExecContext(ctx, "DELETE FROM template_clones WHERE id = $1", cloneID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete clone: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Clone deleted successfully"})
}

// RateTemplate submits a rating/review for a template
func (h *ProcessTemplateHandlers) RateTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	templateID := chi.URLParam(r, "id")

	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "tenant_id and datasource_id are required", http.StatusBadRequest)
		return
	}

	var req struct {
		Rating       int    `json:"rating"`
		ReviewText   string `json:"review_text"`
		ReviewTitle  string `json:"review_title"`
		ReviewerName string `json:"reviewer_name"`
		ReviewerRole string `json:"reviewer_role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Rating < 1 || req.Rating > 5 {
		http.Error(w, "Rating must be between 1 and 5", http.StatusBadRequest)
		return
	}

	// Check if user has cloned this template (verified user)
	var cloneCount int
	_ = h.db.GetContext(ctx, &cloneCount, `
		SELECT COUNT(*) FROM template_clones 
		WHERE template_id = $1 AND tenant_id = $2 AND datasource_id = $3
	`, templateID, tenantID, datasourceID)

	isVerified := cloneCount > 0

	// Insert or update rating
	ratingID := uuid.New().String()
	_, err := h.db.ExecContext(ctx, `
		INSERT INTO template_ratings 
		(id, template_id, tenant_id, datasource_id, rating, review_text, review_title, reviewer_name, reviewer_role, is_verified_user, moderation_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (template_id, tenant_id, datasource_id) 
		DO UPDATE SET 
			rating = EXCLUDED.rating,
			review_text = EXCLUDED.review_text,
			review_title = EXCLUDED.review_title,
			updated_at = NOW()
	`, ratingID, templateID, tenantID, datasourceID, req.Rating, req.ReviewText, req.ReviewTitle, req.ReviewerName, req.ReviewerRole, isVerified, "approved")

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to submit rating: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]string{"message": "Rating submitted successfully", "rating_id": ratingID})
}

// GetTemplateRatings retrieves ratings for a template
func (h *ProcessTemplateHandlers) GetTemplateRatings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	templateID := chi.URLParam(r, "id")

	sortBy := r.URL.Query().Get("sort_by") // recent, rating, helpful
	if sortBy == "" {
		sortBy = "recent"
	}

	query := `
		SELECT * FROM template_ratings
		WHERE template_id = $1 AND moderation_status = 'approved'
	`

	switch sortBy {
	case "rating":
		query += " ORDER BY rating DESC, created_at DESC"
	case "helpful":
		query += " ORDER BY helpful_count DESC, created_at DESC"
	default:
		query += " ORDER BY created_at DESC"
	}

	query += " LIMIT 50"

	var ratings []TemplateRating
	err := h.db.SelectContext(ctx, &ratings, query, templateID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch ratings: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, ratings)
}

// MarkRatingHelpful marks a rating as helpful or not helpful
func (h *ProcessTemplateHandlers) MarkRatingHelpful(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ratingID := chi.URLParam(r, "ratingId")

	var req struct {
		Helpful bool `json:"helpful"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	column := "helpful_count"
	if !req.Helpful {
		column = "not_helpful_count"
	}

	_, err := h.db.ExecContext(ctx, fmt.Sprintf(`
		UPDATE template_ratings 
		SET %s = %s + 1 
		WHERE id = $1
	`, column, column), ratingID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update rating: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Rating updated successfully"})
}

// GetCategories retrieves all template categories
func (h *ProcessTemplateHandlers) GetCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var categories []TemplateCategory
	err := h.db.SelectContext(ctx, &categories, `
		SELECT 
			c.*,
			COUNT(t.id) as template_count
		FROM template_categories c
		LEFT JOIN process_templates t ON c.category_key = t.category AND t.published_at IS NOT NULL
		WHERE c.is_active = true
		GROUP BY c.id
		ORDER BY c.sort_order ASC, c.display_name ASC
	`)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch categories: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, categories)
}

// GetTemplateStats retrieves usage statistics for a template
func (h *ProcessTemplateHandlers) GetTemplateStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	templateID := chi.URLParam(r, "id")

	var stats struct {
		TotalClones       int     `json:"total_clones" db:"total_clones"`
		TotalUsage        int     `json:"total_usage" db:"total_usage"`
		UniqueUsers       int     `json:"unique_users" db:"unique_users"`
		CustomizationRate float64 `json:"customization_rate" db:"customization_rate"`
		AverageRating     float64 `json:"average_rating" db:"average_rating"`
		TotalRatings      int     `json:"total_ratings" db:"total_ratings"`
		Last30DaysClones  int     `json:"last_30_days_clones" db:"last_30_days_clones"`
		AverageSetupTime  float64 `json:"average_setup_time" db:"average_setup_time"`
	}

	err := h.db.GetContext(ctx, &stats, `
		SELECT 
			COUNT(*) as total_clones,
			SUM(usage_count) as total_usage,
			COUNT(DISTINCT tenant_id) as unique_users,
			ROUND(100.0 * SUM(CASE WHEN was_customized THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0), 2) as customization_rate,
			COALESCE((SELECT rating_average FROM process_templates WHERE id = $1), 0) as average_rating,
			COALESCE((SELECT rating_count FROM process_templates WHERE id = $1), 0) as total_ratings,
			SUM(CASE WHEN cloned_at > NOW() - INTERVAL '30 days' THEN 1 ELSE 0 END) as last_30_days_clones,
			AVG(time_to_first_use_minutes) as average_setup_time
		FROM template_clones
		WHERE template_id = $1
	`, templateID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch stats: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

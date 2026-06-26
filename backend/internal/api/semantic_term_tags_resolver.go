package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/services"

	"github.com/google/uuid"
)

// TagResolver handles GraphQL resolver logic for tag operations
type TagResolver struct {
	db                   *sql.DB
	tagSuggestionService *services.TagSuggestionService
}

// NewTagResolver creates a new TagResolver instance
func NewTagResolver(db *sql.DB) *TagResolver {
	return &TagResolver{
		db:                   db,
		tagSuggestionService: services.NewTagSuggestionService(DBAdapter{db: db}),
	}
}

// DBAdapter wraps *sql.DB to implement TagSuggestionDB interface
type DBAdapter struct {
	db *sql.DB
}

func (d DBAdapter) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return d.db.QueryContext(ctx, query, args...)
}

// ============================================================================
// QUERIES
// ============================================================================

// SemanticTags returns all available tags
func (r *TagResolver) SemanticTags(ctx context.Context) ([]*models.Tag, error) {
	query := `
		SELECT id, tag_key, tag_label, tag_category, description, color_code, 
		       icon_name, auto_suggest, sort_order, is_active, created_at, updated_at
		FROM semantic_term_tags
		WHERE is_active = true
		ORDER BY sort_order ASC, tag_label ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tags: %w", err)
	}
	defer rows.Close()

	var tags []*models.Tag
	for rows.Next() {
		tag := &models.Tag{}
		if err := rows.Scan(
			&tag.ID, &tag.TagKey, &tag.TagLabel, &tag.TagCategory,
			&tag.Description, &tag.ColorCode, &tag.IconName,
			&tag.AutoSuggest, &tag.SortOrder, &tag.IsActive,
			&tag.CreatedAt, &tag.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tags: %w", err)
	}

	return tags, nil
}

// TagsByCategory returns tags filtered by category
func (r *TagResolver) TagsByCategory(ctx context.Context, category string) ([]*models.Tag, error) {
	query := `
		SELECT id, tag_key, tag_label, tag_category, description, color_code, 
		       icon_name, auto_suggest, sort_order, is_active, created_at, updated_at
		FROM semantic_term_tags
		WHERE tag_category = $1 AND is_active = true
		ORDER BY sort_order ASC, tag_label ASC
	`

	rows, err := r.db.QueryContext(ctx, query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to query tags by category: %w", err)
	}
	defer rows.Close()

	var tags []*models.Tag
	for rows.Next() {
		tag := &models.Tag{}
		if err := rows.Scan(
			&tag.ID, &tag.TagKey, &tag.TagLabel, &tag.TagCategory,
			&tag.Description, &tag.ColorCode, &tag.IconName,
			&tag.AutoSuggest, &tag.SortOrder, &tag.IsActive,
			&tag.CreatedAt, &tag.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

// SemanticTermTags returns current tags for a semantic term
func (r *TagResolver) SemanticTermTags(ctx context.Context, termID string) ([]*models.Tag, error) {
	query := `
		SELECT stt.id, stt.tag_key, stt.tag_label, stt.tag_category, 
		       stt.description, stt.color_code, stt.icon_name, stt.auto_suggest, 
		       stt.sort_order, stt.is_active, stt.created_at, stt.updated_at
		FROM semantic_term_tags stt
		JOIN (
			SELECT jsonb_array_elements(tags)->>'tag_key' as tag_key
			FROM catalog_node
			WHERE id = $1
		) cn ON stt.tag_key = cn.tag_key
		WHERE stt.is_active = true
		ORDER BY stt.sort_order ASC, stt.tag_label ASC
	`

	rows, err := r.db.QueryContext(ctx, query, termID)
	if err != nil {
		return nil, fmt.Errorf("failed to query semantic term tags: %w", err)
	}
	defer rows.Close()

	var tags []*models.Tag
	for rows.Next() {
		tag := &models.Tag{}
		if err := rows.Scan(
			&tag.ID, &tag.TagKey, &tag.TagLabel, &tag.TagCategory,
			&tag.Description, &tag.ColorCode, &tag.IconName,
			&tag.AutoSuggest, &tag.SortOrder, &tag.IsActive,
			&tag.CreatedAt, &tag.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

// TagCategories returns all tag categories with their tags
func (r *TagResolver) TagCategories(ctx context.Context) ([]*models.TagCategory, error) {
	query := `
		SELECT DISTINCT tag_category FROM semantic_term_tags
		WHERE is_active = true
		ORDER BY tag_category ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tag categories: %w", err)
	}
	defer rows.Close()

	var categories []*models.TagCategory
	categoryMap := make(map[string]*models.TagCategory)

	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}

		tags, err := r.TagsByCategory(ctx, category)
		if err != nil {
			return nil, err
		}

		cat := &models.TagCategory{
			CategoryName: category,
			DisplayName:  formatCategoryName(category),
			Tags:         tags,
		}
		categoryMap[category] = cat
		categories = append(categories, cat)
	}

	return categories, rows.Err()
}

// SuggestSemanticTermTags returns tag suggestions for a semantic term
func (r *TagResolver) SuggestSemanticTermTags(
	ctx context.Context,
	input *models.TagSuggestionRequest,
) (*models.TagSuggestionResponse, error) {
	// Use the tag suggestion service to get suggestions
	suggestions, err := r.tagSuggestionService.SuggestTagsForSemanticTerm(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tag suggestions: %w", err)
	}

	return suggestions, nil
}

// ============================================================================
// MUTATIONS
// ============================================================================

// AddTagToSemanticTerm adds a single tag to a semantic term
func (r *TagResolver) AddTagToSemanticTerm(
	ctx context.Context,
	termID string,
	tagKey string,
) error {
	// Get current tags
	var tagsJSON sql.NullString
	query := `SELECT tags FROM catalog_node WHERE id = $1`
	if err := r.db.QueryRowContext(ctx, query, termID).Scan(&tagsJSON); err != nil {
		return fmt.Errorf("failed to get current tags: %w", err)
	}

	// Parse current tags
	var tags []map[string]interface{}
	if tagsJSON.Valid {
		if err := json.Unmarshal([]byte(tagsJSON.String), &tags); err != nil {
			return fmt.Errorf("failed to parse current tags: %w", err)
		}
	}

	// Check if tag already exists
	for _, t := range tags {
		if tk, ok := t["tag_key"].(string); ok && tk == tagKey {
			return nil // Already exists
		}
	}

	// Get tag details
	tagQuery := `
		SELECT tag_label, tag_category FROM semantic_term_tags WHERE tag_key = $1
	`
	var tagLabel, tagCategory string
	if err := r.db.QueryRowContext(ctx, tagQuery, tagKey).Scan(&tagLabel, &tagCategory); err != nil {
		return fmt.Errorf("failed to get tag details: %w", err)
	}

	// Add new tag
	newTag := map[string]interface{}{
		"tag_key":      tagKey,
		"tag_label":    tagLabel,
		"tag_category": tagCategory,
		"added_at":     time.Now(),
	}
	tags = append(tags, newTag)

	// Update database
	tagsData, err := json.Marshal(tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	updateQuery := `UPDATE catalog_node SET tags = $1, updated_at = $2 WHERE id = $3`
	if _, err := r.db.ExecContext(ctx, updateQuery, string(tagsData), time.Now(), termID); err != nil {
		return fmt.Errorf("failed to update semantic term tags: %w", err)
	}

	return nil
}

// RemoveTagFromSemanticTerm removes a tag from a semantic term
func (r *TagResolver) RemoveTagFromSemanticTerm(
	ctx context.Context,
	termID string,
	tagKey string,
) error {
	query := `SELECT tags FROM catalog_node WHERE id = $1`
	var tagsJSON sql.NullString
	if err := r.db.QueryRowContext(ctx, query, termID).Scan(&tagsJSON); err != nil {
		return fmt.Errorf("failed to get current tags: %w", err)
	}

	var tags []map[string]interface{}
	if tagsJSON.Valid {
		if err := json.Unmarshal([]byte(tagsJSON.String), &tags); err != nil {
			return fmt.Errorf("failed to parse current tags: %w", err)
		}
	}

	// Remove tag
	filteredTags := make([]map[string]interface{}, 0)
	for _, t := range tags {
		if tk, ok := t["tag_key"].(string); !ok || tk != tagKey {
			filteredTags = append(filteredTags, t)
		}
	}

	// Update database
	tagsData, err := json.Marshal(filteredTags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	updateQuery := `UPDATE catalog_node SET tags = $1, updated_at = $2 WHERE id = $3`
	if _, err := r.db.ExecContext(ctx, updateQuery, string(tagsData), time.Now(), termID); err != nil {
		return fmt.Errorf("failed to update semantic term tags: %w", err)
	}

	return nil
}

// UpdateSemanticTermTags replaces all tags for a semantic term
func (r *TagResolver) UpdateSemanticTermTags(
	ctx context.Context,
	termID string,
	tagKeys []string,
) error {
	// Fetch tag details for all keys
	tags := make([]map[string]interface{}, 0)
	for _, tagKey := range tagKeys {
		query := `
			SELECT tag_key, tag_label, tag_category FROM semantic_term_tags 
			WHERE tag_key = $1 AND is_active = true
		`
		var key, label, category string
		if err := r.db.QueryRowContext(ctx, query, tagKey).Scan(&key, &label, &category); err != nil {
			if err == sql.ErrNoRows {
				continue // Skip invalid tags
			}
			return fmt.Errorf("failed to get tag details: %w", err)
		}

		tag := map[string]interface{}{
			"tag_key":      key,
			"tag_label":    label,
			"tag_category": category,
			"added_at":     time.Now(),
		}
		tags = append(tags, tag)
	}

	// Update database
	tagsData, err := json.Marshal(tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	updateQuery := `UPDATE catalog_node SET tags = $1, updated_at = $2 WHERE id = $3`
	if _, err := r.db.ExecContext(ctx, updateQuery, string(tagsData), time.Now(), termID); err != nil {
		return fmt.Errorf("failed to update semantic term tags: %w", err)
	}

	return nil
}

// CreateSemanticTag creates a new tag
func (r *TagResolver) CreateSemanticTag(ctx context.Context, input *models.TagInput) (*models.Tag, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}
	id := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO semantic_term_tags 
		(id, tag_key, tag_label, tag_category, description, color_code, 
		 icon_name, auto_suggest, sort_order, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, tag_key, tag_label, tag_category, description, color_code, 
		          icon_name, auto_suggest, sort_order, is_active, created_at, updated_at
	`

	tag := &models.Tag{}
	err := r.db.QueryRowContext(ctx, query,
		id, input.TagKey, input.TagLabel, input.TagCategory, input.Description,
		input.ColorCode, input.IconName, input.AutoSuggest, input.SortOrder,
		true, now, now,
	).Scan(
		&tag.ID, &tag.TagKey, &tag.TagLabel, &tag.TagCategory,
		&tag.Description, &tag.ColorCode, &tag.IconName,
		&tag.AutoSuggest, &tag.SortOrder, &tag.IsActive,
		&tag.CreatedAt, &tag.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}

	return tag, nil
}

// UpdateSemanticTag updates an existing tag
func (r *TagResolver) UpdateSemanticTag(
	ctx context.Context,
	tagKey string,
	input *models.TagInput,
) (*models.Tag, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}
	now := time.Now()

	query := `
		UPDATE semantic_term_tags 
		SET tag_label = $1, description = $2, color_code = $3, 
		    icon_name = $4, auto_suggest = $5, sort_order = $6, updated_at = $7
		WHERE tag_key = $8
		RETURNING id, tag_key, tag_label, tag_category, description, color_code, 
		          icon_name, auto_suggest, sort_order, is_active, created_at, updated_at
	`

	tag := &models.Tag{}
	err := r.db.QueryRowContext(ctx, query,
		input.TagLabel, input.Description, input.ColorCode, input.IconName,
		input.AutoSuggest, input.SortOrder, now, tagKey,
	).Scan(
		&tag.ID, &tag.TagKey, &tag.TagLabel, &tag.TagCategory,
		&tag.Description, &tag.ColorCode, &tag.IconName,
		&tag.AutoSuggest, &tag.SortOrder, &tag.IsActive,
		&tag.CreatedAt, &tag.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update tag: %w", err)
	}

	return tag, nil
}

// DeleteSemanticTag soft-deletes a tag
func (r *TagResolver) DeleteSemanticTag(ctx context.Context, tagKey string) (bool, error) {
	query := `UPDATE semantic_term_tags SET is_active = false, updated_at = $1 WHERE tag_key = $2`
	result, err := r.db.ExecContext(ctx, query, time.Now(), tagKey)
	if err != nil {
		return false, fmt.Errorf("failed to delete tag: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

// AcceptTagSuggestion records acceptance or rejection of a tag suggestion
func (r *TagResolver) AcceptTagSuggestion(
	ctx context.Context,
	termID string,
	tagKey string,
	isAccepted bool,
) error {
	id := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO semantic_term_tag_suggestions 
		(id, semantic_term_id, tag_key, is_accepted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (semantic_term_id, tag_key) DO UPDATE
		SET is_accepted = $4, updated_at = $6
	`

	if _, err := r.db.ExecContext(ctx, query, id, termID, tagKey, isAccepted, now, now); err != nil {
		return fmt.Errorf("failed to record suggestion acceptance: %w", err)
	}

	if isAccepted {
		// Also add the tag if it was accepted
		return r.AddTagToSemanticTerm(ctx, termID, tagKey)
	}

	return nil
}

// ApplyTagSuggestions applies multiple suggested tags to a semantic term
func (r *TagResolver) ApplyTagSuggestions(
	ctx context.Context,
	termID string,
	suggestedTags []string,
) error {
	for _, tagKey := range suggestedTags {
		if err := r.AddTagToSemanticTerm(ctx, termID, tagKey); err != nil {
			return fmt.Errorf("failed to apply tag suggestion: %w", err)
		}

		// Record acceptance
		if err := r.AcceptTagSuggestion(ctx, termID, tagKey, true); err != nil {
			return fmt.Errorf("failed to record tag suggestion: %w", err)
		}
	}

	return nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func formatCategoryName(category string) string {
	switch category {
	case "business_area":
		return "Business Area"
	case "data_type":
		return "Data Type"
	case "domain":
		return "Domain"
	case "usage_pattern":
		return "Usage Pattern"
	case "sensitivity":
		return "Sensitivity"
	case "governance":
		return "Governance"
	default:
		return category
	}
}

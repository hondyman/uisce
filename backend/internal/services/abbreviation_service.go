package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/pkg/llm"
	"github.com/jmoiron/sqlx"
)

// AbbreviationService handles abbreviation lookups and management
type AbbreviationService struct {
	db          *sqlx.DB
	llmProvider llm.LLMProvider
}

// getGoldCopyTenantID retrieves the tenant_id of the gold copy tenant
func (s *AbbreviationService) getGoldCopyTenantID(ctx context.Context) (string, error) {
	var tenantID string
	query := `SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1`
	err := s.db.GetContext(ctx, &tenantID, query)
	if err != nil {
		return "", fmt.Errorf("failed to get gold copy tenant: %w", err)
	}
	return tenantID, nil
}

// NewAbbreviationService creates a new abbreviation service
func NewAbbreviationService(db *sqlx.DB, llmProvider llm.LLMProvider) *AbbreviationService {
	return &AbbreviationService{
		db:          db,
		llmProvider: llmProvider,
	}
}

// GetLLMProvider returns the LLM provider for external use
func (s *AbbreviationService) GetLLMProvider() llm.LLMProvider {
	return s.llmProvider
}

// Abbreviation represents an abbreviation entry
type Abbreviation struct {
	ID           int        `json:"id" db:"id"`
	Abbreviation string     `json:"abbreviation" db:"abbreviation"`
	FullWord     string     `json:"full_word" db:"full_word"`
	Notes        string     `json:"notes" db:"notes"`
	TenantID     string     `json:"tenant_id" db:"tenant_id"`
	IsCore       bool       `json:"is_core" db:"-"` // Computed: true if tenant is gold_copy
	CreatedAt    *time.Time `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

// ...

// GetAbbreviationsParams defines parameters for listing abbreviations
type GetAbbreviationsParams struct {
	Limit  int
	Offset int
	Search string
}

// GetAbbreviationsResponse contains paginated abbreviations
type GetAbbreviationsResponse struct {
	Items      []Abbreviation `json:"items"`
	TotalCount int            `json:"total_count"`
	Limit      int            `json:"limit"`
	Offset     int            `json:"offset"`
}

// GetAbbreviations retrieves abbreviations with pagination and search
// Returns both core (uisce) abbreviations and tenant-specific abbreviations
func (s *AbbreviationService) GetAbbreviations(ctx context.Context, params GetAbbreviationsParams) (*GetAbbreviationsResponse, error) {
	// Defaults
	if params.Limit <= 0 {
		params.Limit = 50
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	// Extract tenant_id from context
	tenantIDVal := ctx.Value("tenant_id")
	tenantIDStr, ok := tenantIDVal.(string)
	if !ok || tenantIDStr == "" {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	// Get gold copy tenant ID
	goldCopyTenantID, err := s.getGoldCopyTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gold copy tenant: %w", err)
	}

	// Query to get abbreviations for both the gold copy tenant (core) and the current tenant
	queryBase := fmt.Sprintf(`FROM sml.abbreviation_lookup WHERE tenant_id IN ('%s', $1)`, goldCopyTenantID)
	countQuery := `SELECT COUNT(*) ` + queryBase
	selectQuery := `SELECT id, abbreviation, full_word, notes, tenant_id, created_at, updated_at ` + queryBase

	// Filter
	args := []interface{}{tenantIDStr}
	argIdx := 2

	if params.Search != "" {
		filter := fmt.Sprintf(` AND (abbreviation ILIKE $%d OR full_word ILIKE $%d OR notes ILIKE $%d)`, argIdx, argIdx, argIdx)
		countQuery += filter
		selectQuery += filter
		args = append(args, "%"+params.Search+"%")
		argIdx++
	}

	// Count total
	var totalCount int
	if err := s.db.GetContext(ctx, &totalCount, countQuery, args...); err != nil {
		return nil, fmt.Errorf("failed to count abbreviations: %w", err)
	}

	// Fetch items with pagination
	selectQuery += fmt.Sprintf(` ORDER BY abbreviation ASC LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	args = append(args, params.Limit, params.Offset)

	var items []Abbreviation
	if err := s.db.SelectContext(ctx, &items, selectQuery, args...); err != nil {
		return nil, fmt.Errorf("failed to fetch abbreviations: %w", err)
	}

	// Get gold copy tenant ID for IsCore computation
	goldCopyTenantID, err = s.getGoldCopyTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gold copy tenant: %w", err)
	}

	// Compute IsCore for each abbreviation
	for i := range items {
		items[i].IsCore = items[i].TenantID == goldCopyTenantID
	}

	// Ensure empty slice instead of nil
	if items == nil {
		items = []Abbreviation{}
	}

	return &GetAbbreviationsResponse{
		Items:      items,
		TotalCount: totalCount,
		Limit:      params.Limit,
		Offset:     params.Offset,
	}, nil
}

// GetAllAbbreviations retrieves all abbreviations (Internal/Legacy)
// Returns both core (gold copy) and tenant-specific abbreviations
func (s *AbbreviationService) GetAllAbbreviations(ctx context.Context) ([]Abbreviation, error) {
	// Extract tenant_id from context
	tenantIDVal := ctx.Value("tenant_id")
	tenantIDStr, ok := tenantIDVal.(string)
	if !ok || tenantIDStr == "" {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	// Get gold copy tenant ID
	goldCopyTenantID, err := s.getGoldCopyTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gold copy tenant: %w", err)
	}

	var abbreviations []Abbreviation
	// Query to get all abbreviations for both core and current tenant
	query := fmt.Sprintf(`SELECT id, abbreviation, full_word, notes, tenant_id, created_at, updated_at 
	          FROM sml.abbreviation_lookup 
	          WHERE tenant_id IN ('%s', $1)
	          ORDER BY abbreviation ASC`, goldCopyTenantID)

	err = s.db.SelectContext(ctx, &abbreviations, query, tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch abbreviations: %w", err)
	}

	// Compute IsCore field
	for i := range abbreviations {
		abbreviations[i].IsCore = abbreviations[i].TenantID == goldCopyTenantID
	}

	return abbreviations, nil
}

// AddAbbreviation adds a new abbreviation
func (s *AbbreviationService) AddAbbreviation(ctx context.Context, abbr, full, notes string) error {
	// Extract tenant ID from context
	tenantIDVal := ctx.Value("tenant_id")
	tenantIDStr, ok := tenantIDVal.(string)
	if !ok || tenantIDStr == "" {
		return fmt.Errorf("tenant_id not found in context")
	}

	query := `
		INSERT INTO sml.abbreviation_lookup (abbreviation, full_word, notes, tenant_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (tenant_id, abbreviation) DO UPDATE
		SET full_word = EXCLUDED.full_word,
			notes = EXCLUDED.notes
	`
	_, err := s.db.ExecContext(ctx, query, strings.ToUpper(abbr), strings.ToUpper(full), notes, tenantIDStr)
	if err != nil {
		return fmt.Errorf("failed to add abbreviation: %w", err)
	}
	return nil
}

// UpdateAbbreviation updates an existing abbreviation
// Only the owning tenant or gold copy tenant can update an abbreviation
func (s *AbbreviationService) UpdateAbbreviation(ctx context.Context, id int, abbr, full, notes string) error {
	// Extract tenant ID from context
	tenantIDVal := ctx.Value("tenant_id")
	tenantIDStr, ok := tenantIDVal.(string)
	if !ok || tenantIDStr == "" {
		return fmt.Errorf("tenant_id not found in context")
	}

	// Get gold copy tenant ID
	goldCopyTenantID, err := s.getGoldCopyTenantID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gold copy tenant: %w", err)
	}

	// Check ownership - only allow update if tenant owns the abbreviation or is gold copy
	var ownerTenantID string
	checkQuery := `SELECT tenant_id FROM sml.abbreviation_lookup WHERE id = $1`
	err = s.db.GetContext(ctx, &ownerTenantID, checkQuery, id)
	if err != nil {
		if err == fmt.Errorf("sql: no rows in result set") {
			return fmt.Errorf("abbreviation not found")
		}
		return fmt.Errorf("failed to check abbreviation ownership: %w", err)
	}

	// Only allow update if:
	// 1. Current tenant is gold copy (can update anything), OR
	// 2. Current tenant owns the abbreviation
	if tenantIDStr != goldCopyTenantID && ownerTenantID != tenantIDStr {
		return fmt.Errorf("permission denied: cannot update abbreviation owned by another tenant")
	}

	query := `
		UPDATE sml.abbreviation_lookup 
		SET abbreviation = $1, 
			full_word = $2, 
			notes = $3
		WHERE id = $4
	`

	result, err := s.db.ExecContext(ctx, query, strings.ToUpper(abbr), strings.ToUpper(full), notes, id)
	if err != nil {
		return fmt.Errorf("failed to update abbreviation: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("abbreviation not found")
	}

	return nil
}

// DeleteAbbreviation deletes an abbreviation
// Only the owning tenant or gold copy tenant can delete an abbreviation
func (s *AbbreviationService) DeleteAbbreviation(ctx context.Context, id int) error {
	// Extract tenant ID from context
	tenantIDVal := ctx.Value("tenant_id")
	tenantIDStr, ok := tenantIDVal.(string)
	if !ok || tenantIDStr == "" {
		return fmt.Errorf("tenant_id not found in context")
	}

	// Get gold copy tenant ID
	goldCopyTenantID, err := s.getGoldCopyTenantID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gold copy tenant: %w", err)
	}

	// Check ownership - only allow delete if tenant owns the abbreviation or is gold copy
	var ownerTenantID string
	checkQuery := `SELECT tenant_id FROM sml.abbreviation_lookup WHERE id = $1`
	err = s.db.GetContext(ctx, &ownerTenantID, checkQuery, id)
	if err != nil {
		if err == fmt.Errorf("sql: no rows in result set") {
			return fmt.Errorf("abbreviation not found")
		}
		return fmt.Errorf("failed to check abbreviation ownership: %w", err)
	}

	// Only allow delete if:
	// 1. Current tenant is gold copy (can delete anything), OR
	// 2. Current tenant owns the abbreviation
	if tenantIDStr != goldCopyTenantID && ownerTenantID != tenantIDStr {
		return fmt.Errorf("permission denied: cannot delete abbreviation owned by another tenant")
	}

	query := `DELETE FROM sml.abbreviation_lookup WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete abbreviation: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("abbreviation not found")
	}

	return nil
}

// ExpandAbbreviations expands abbreviations in a term (e.g., column name)
func (s *AbbreviationService) ExpandAbbreviations(ctx context.Context, term string) (map[string]interface{}, error) {
	// Get all abbreviations map
	abbrevs, err := s.GetAllAbbreviations(ctx)
	if err != nil {
		return nil, err
	}

	abbrMap := make(map[string]string)
	for _, a := range abbrevs {
		abbrMap[a.Abbreviation] = a.FullWord
	}

	// Split term by underscores
	parts := strings.Split(strings.ToUpper(term), "_")
	var expandedParts []string
	var variations []string

	for _, part := range parts {
		if expanded, ok := abbrMap[part]; ok {
			expandedParts = append(expandedParts, expanded)
		} else {
			expandedParts = append(expandedParts, part)
		}
	}

	fullExpansion := strings.Join(expandedParts, " ")
	variations = append(variations, strings.Join(expandedParts, "_"))
	variations = append(variations, strings.Join(expandedParts, ""))

	return map[string]interface{}{
		"column_name": term,
		"expansions":  fullExpansion,
		"variations":  variations,
	}, nil
}

// ValidateSemanticTerms checks for abbreviation violations
func (s *AbbreviationService) ValidateSemanticTerms(ctx context.Context, terms []string) (map[string]interface{}, error) {
	abbrevs, err := s.GetAllAbbreviations(ctx)
	if err != nil {
		return nil, err
	}

	abbrMap := make(map[string]string)
	for _, a := range abbrevs {
		abbrMap[a.Abbreviation] = a.FullWord
	}

	violations := make(map[string][]string)
	validCount := 0

	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}

		parts := strings.Split(strings.ToUpper(term), "_")
		var termViolations []string

		for _, part := range parts {
			if _, ok := abbrMap[part]; ok {
				termViolations = append(termViolations, part)
			}
		}

		if len(termViolations) > 0 {
			violations[term] = termViolations
		} else {
			validCount++
		}
	}

	return map[string]interface{}{
		"violations":  violations,
		"valid_terms": validCount,
		"total_terms": len(terms),
	}, nil
}

// ScanForAbbreviations scans the database for potential abbreviations in column names
func (s *AbbreviationService) ScanForAbbreviations(ctx context.Context) ([]string, error) {
	// 1. Get all column names from information_schema
	query := `
		SELECT column_name 
		FROM information_schema.columns 
		WHERE table_schema NOT IN ('information_schema', 'pg_catalog', 'temporal_spec')
	`
	var columnNames []string
	err := s.db.SelectContext(ctx, &columnNames, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch column names: %w", err)
	}

	// 2. Get existing abbreviations
	existingAbbrevs, err := s.GetAllAbbreviations(ctx)
	if err != nil {
		return nil, err
	}
	knownAbbrevs := make(map[string]bool)
	for _, a := range existingAbbrevs {
		knownAbbrevs[a.Abbreviation] = true
	}

	// 3. Tokenize and count frequency
	tokenCounts := make(map[string]int)
	for _, col := range columnNames {
		parts := strings.Split(strings.ToUpper(col), "_")
		for _, part := range parts {
			if knownAbbrevs[part] {
				continue
			}
			if len(part) < 2 {
				continue
			}
			tokenCounts[part]++
		}
	}

	// 4. Filter by frequency
	var candidates []string
	for token, count := range tokenCounts {
		if count > 1 {
			candidates = append(candidates, token)
		}
	}

	sort.Strings(candidates)
	return candidates, nil
}

// SuggestExpansions uses LLM to suggest expansions for abbreviations
func (s *AbbreviationService) SuggestExpansions(ctx context.Context, candidates []string) (map[string]string, error) {
	if s.llmProvider == nil {
		return nil, fmt.Errorf("LLM provider not configured")
	}

	if len(candidates) == 0 {
		return map[string]string{}, nil
	}

	batchSize := 50
	allSuggestions := make(map[string]string)

	for i := 0; i < len(candidates); i += batchSize {
		end := i + batchSize
		if end > len(candidates) {
			end = len(candidates)
		}
		batch := candidates[i:end]

		prompt := fmt.Sprintf(`
You are a data architect expert in Wealth Management and Financial domains.
Suggest full word expansions for the following potential abbreviations found in database column names:
%s

Return ONLY a valid JSON object where keys are the abbreviations and values are the suggested full words (in UPPERCASE).
If you are unsure or it looks like a full word already, exclude it from the JSON.
Example format: {"ACCT": "ACCOUNT", "VAL": "VALUE"}
`, strings.Join(batch, ", "))

		response, err := s.llmProvider.GenerateResponse(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("LLM generation failed: %w", err)
		}

		cleanResponse := strings.TrimSpace(response)
		if strings.HasPrefix(cleanResponse, "```json") {
			cleanResponse = strings.TrimPrefix(cleanResponse, "```json")
			cleanResponse = strings.TrimSuffix(cleanResponse, "```")
		} else if strings.HasPrefix(cleanResponse, "```") {
			cleanResponse = strings.TrimPrefix(cleanResponse, "```")
			cleanResponse = strings.TrimSuffix(cleanResponse, "```")
		}

		var batchSuggestions map[string]string
		if err := json.Unmarshal([]byte(cleanResponse), &batchSuggestions); err != nil {
			return nil, fmt.Errorf("failed to parse LLM response: %w", err)
		}

		for k, v := range batchSuggestions {
			allSuggestions[k] = v
		}
	}

	return allSuggestions, nil
}

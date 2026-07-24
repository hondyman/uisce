package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AbbreviationService handles abbreviation lookup and expansion
type AbbreviationService struct {
	db          *sql.DB
	logger      *zap.Logger
	cache       map[string]string
	cacheMutex  sync.RWMutex
	lastUpdated time.Time
	cacheExpiry time.Duration
}

// AbbreviationEntry represents an abbreviation record from the database
type AbbreviationEntry struct {
	ID           int    `json:"id"`
	Abbreviation string `json:"abbreviation"`
	FullWord     string `json:"full_word"`
	Notes        string `json:"notes"`
}

// NewAbbreviationService creates a new abbreviation service
func NewAbbreviationService(db *sql.DB, logger *zap.Logger) *AbbreviationService {
	return &AbbreviationService{
		db:          db,
		logger:      logger,
		cache:       make(map[string]string),
		cacheExpiry: 1 * time.Hour, // Cache for 1 hour
	}
}

// LoadAbbreviations loads abbreviations from the database into cache
func (s *AbbreviationService) LoadAbbreviations(ctx context.Context) error {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	// Check if cache is still fresh
	if time.Since(s.lastUpdated) < s.cacheExpiry && len(s.cache) > 0 {
		return nil
	}

	query := `SELECT abbreviation, full_word FROM sml.abbreviation_lookup ORDER BY abbreviation`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		s.logger.Error("Failed to query abbreviations", zap.Error(err))
		return fmt.Errorf("failed to query abbreviations: %w", err)
	}
	defer rows.Close()

	newCache := make(map[string]string)
	for rows.Next() {
		var abbrev, expansion string
		if err := rows.Scan(&abbrev, &expansion); err != nil {
			s.logger.Error("Failed to scan abbreviation row", zap.Error(err))
			continue
		}
		// Store in uppercase for consistent lookup
		newCache[strings.ToUpper(abbrev)] = strings.ToUpper(expansion)
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("Error iterating abbreviation rows", zap.Error(err))
		return fmt.Errorf("error iterating abbreviation rows: %w", err)
	}

	s.cache = newCache
	s.lastUpdated = time.Now()
	s.logger.Info("Loaded abbreviations into cache", zap.Int("count", len(s.cache)))

	return nil
}

// GetAllAbbreviations returns all abbreviations from the database
func (s *AbbreviationService) GetAllAbbreviations(ctx context.Context) ([]AbbreviationEntry, error) {
	query := `SELECT id, abbreviation, full_word, COALESCE(notes, '') as notes 
			  FROM sml.abbreviation_lookup 
			  ORDER BY abbreviation`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		s.logger.Error("Failed to query all abbreviations", zap.Error(err))
		return nil, fmt.Errorf("failed to query abbreviations: %w", err)
	}
	defer rows.Close()

	var abbreviations []AbbreviationEntry
	for rows.Next() {
		var entry AbbreviationEntry
		if err := rows.Scan(&entry.ID, &entry.Abbreviation, &entry.FullWord, &entry.Notes); err != nil {
			s.logger.Error("Failed to scan abbreviation entry", zap.Error(err))
			continue
		}
		abbreviations = append(abbreviations, entry)
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("Error iterating abbreviation rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating abbreviation rows: %w", err)
	}

	return abbreviations, nil
}

// GetAbbreviationMap returns the current abbreviation cache as a map
func (s *AbbreviationService) GetAbbreviationMap(ctx context.Context) (map[string]string, error) {
	if err := s.LoadAbbreviations(ctx); err != nil {
		return nil, err
	}

	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	// Return a copy of the cache
	result := make(map[string]string, len(s.cache))
	for k, v := range s.cache {
		result[k] = v
	}

	return result, nil
}

// ExpandAbbreviations creates variations of a column name with expanded abbreviations
func (s *AbbreviationService) ExpandAbbreviations(ctx context.Context, columnName string) ([]string, error) {
	if err := s.LoadAbbreviations(ctx); err != nil {
		return []string{columnName}, err
	}

	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	normalized := strings.ToUpper(columnName)
	variations := []string{normalized}

	// Split on common separators
	separators := []string{"_", "-", ".", " "}
	var tokens []string

	for _, sep := range separators {
		if strings.Contains(normalized, sep) {
			tokens = strings.Split(normalized, sep)
			break
		}
	}

	if len(tokens) == 0 {
		tokens = []string{normalized}
	}

	// Check if any token is an abbreviation
	hasExpansion := false
	expandedTokenSets := make([][]string, len(tokens))

	for i, token := range tokens {
		tokenVariations := []string{token}
		if expansion, exists := s.cache[token]; exists {
			tokenVariations = append(tokenVariations, expansion)
			hasExpansion = true
		}
		expandedTokenSets[i] = tokenVariations
	}

	// Generate combinations if we have expansions
	if hasExpansion {
		combinations := s.generateCombinations(expandedTokenSets)
		for _, combo := range combinations {
			variations = append(variations, strings.Join(combo, "_"))
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	result := []string{}
	for _, v := range variations {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}

	return result, nil
}

// generateCombinations generates all possible combinations from token variations
func (s *AbbreviationService) generateCombinations(tokenSets [][]string) [][]string {
	if len(tokenSets) == 0 {
		return [][]string{}
	}
	if len(tokenSets) == 1 {
		result := make([][]string, len(tokenSets[0]))
		for i, token := range tokenSets[0] {
			result[i] = []string{token}
		}
		return result
	}

	var result [][]string
	restCombos := s.generateCombinations(tokenSets[1:])

	for _, token := range tokenSets[0] {
		for _, restCombo := range restCombos {
			combo := make([]string, len(restCombo)+1)
			combo[0] = token
			copy(combo[1:], restCombo)
			result = append(result, combo)
		}
	}

	return result
}

// AddAbbreviation adds a new abbreviation to the database
func (s *AbbreviationService) AddAbbreviation(ctx context.Context, abbreviation, fullWord, notes string) error {
	query := `INSERT INTO abbreviations (abbreviation, full_word, notes, created_at, updated_at) 
			  VALUES ($1, $2, $3, NOW(), NOW()) 
			  ON CONFLICT (abbreviation) 
			  DO UPDATE SET full_word = EXCLUDED.full_word, notes = EXCLUDED.notes, updated_at = NOW()`

	_, err := s.db.ExecContext(ctx, query, abbreviation, fullWord, notes)
	if err != nil {
		s.logger.Error("Failed to add abbreviation", zap.Error(err))
		return fmt.Errorf("failed to add abbreviation: %w", err)
	}

	// Invalidate cache to force reload
	s.cacheMutex.Lock()
	s.lastUpdated = time.Time{}
	s.cacheMutex.Unlock()

	s.logger.Info("Added abbreviation", zap.String("abbreviation", abbreviation), zap.String("full_word", fullWord))
	return nil
}

// UpdateAbbreviation updates an existing abbreviation in the database
func (s *AbbreviationService) UpdateAbbreviation(ctx context.Context, id int, abbreviation, fullWord, notes string) error {
	query := `UPDATE abbreviations 
			  SET abbreviation = $1, full_word = $2, notes = $3, updated_at = NOW() 
			  WHERE id = $4`

	result, err := s.db.ExecContext(ctx, query, abbreviation, fullWord, notes, id)
	if err != nil {
		s.logger.Error("Failed to update abbreviation", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("failed to update abbreviation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("abbreviation with id %d not found", id)
	}

	// Invalidate cache to force reload
	s.cacheMutex.Lock()
	s.lastUpdated = time.Time{}
	s.cacheMutex.Unlock()

	s.logger.Info("Updated abbreviation", zap.Int("id", id), zap.String("abbreviation", abbreviation), zap.String("full_word", fullWord))
	return nil
}

// DeleteAbbreviation removes an abbreviation from the database
func (s *AbbreviationService) DeleteAbbreviation(ctx context.Context, id int) error {
	query := `DELETE FROM sml.abbreviation_lookup WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		s.logger.Error("Failed to delete abbreviation", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("failed to delete abbreviation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("abbreviation with id %d not found", id)
	}

	// Invalidate cache to force reload
	s.cacheMutex.Lock()
	s.lastUpdated = time.Time{}
	s.cacheMutex.Unlock()

	s.logger.Info("Deleted abbreviation", zap.Int("id", id))
	return nil
}

// ValidateSemanticTerms checks if semantic terms contain abbreviations and suggests expansions
func (s *AbbreviationService) ValidateSemanticTerms(ctx context.Context, termNames []string) (map[string][]string, error) {
	if err := s.LoadAbbreviations(ctx); err != nil {
		return nil, err
	}

	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	violations := make(map[string][]string)

	for _, termName := range termNames {
		// Split term name and check for abbreviations
		normalized := strings.ToUpper(termName)
		separators := []string{"_", "-", ".", " "}
		var tokens []string

		for _, sep := range separators {
			if strings.Contains(normalized, sep) {
				tokens = strings.Split(normalized, sep)
				break
			}
		}

		if len(tokens) == 0 {
			tokens = []string{normalized}
		}

		var foundAbbreviations []string
		for _, token := range tokens {
			if expansion, exists := s.cache[token]; exists {
				foundAbbreviations = append(foundAbbreviations, fmt.Sprintf("%s -> %s", token, expansion))
			}
		}

		if len(foundAbbreviations) > 0 {
			violations[termName] = foundAbbreviations
		}
	}

	return violations, nil
}

// GetExpandedAbbreviations returns a formatted string of expanded abbreviations for display
func (s *AbbreviationService) GetExpandedAbbreviations(ctx context.Context, columnName string) (string, error) {
	if err := s.LoadAbbreviations(ctx); err != nil {
		return "", err
	}

	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	normalized := strings.ToUpper(columnName)
	tokens := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == '_' || r == '-' || r == '.' || r == ' '
	})

	var expansions []string
	for _, token := range tokens {
		if expansion, exists := s.cache[token]; exists {
			expansions = append(expansions, fmt.Sprintf("%s→%s", token, expansion))
		}
	}

	return strings.Join(expansions, ", "), nil
}

// ExpandToHumanReadable expands abbreviations and returns a human-readable title-cased string
func (s *AbbreviationService) ExpandToHumanReadable(ctx context.Context, term string) (string, error) {
	if err := s.LoadAbbreviations(ctx); err != nil {
		return term, err
	}

	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	normalized := strings.ToUpper(term)
	tokens := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == '_' || r == '-' || r == '.' || r == ' '
	})

	var expandedTokens []string
	for _, token := range tokens {
		if expansion, exists := s.cache[token]; exists {
			expandedTokens = append(expandedTokens, expansion)
		} else {
			expandedTokens = append(expandedTokens, token)
		}
	}

	joined := strings.Join(expandedTokens, " ")
	return strings.Title(strings.ToLower(joined)), nil
}

// AbbreviateToShort takes an expanded title and abbreviates words using the abbreviations table.
// Returns result in Title Case (e.g., "Customer Address" -> "Cust Addr")
func (s *AbbreviationService) AbbreviateToShort(ctx context.Context, title string) (string, error) {
	if err := s.LoadAbbreviations(ctx); err != nil {
		return title, err
	}

	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	// Build reverse lookup: full_word -> abbreviation
	reverseCache := make(map[string]string)
	for abbrev, fullWord := range s.cache {
		reverseCache[strings.ToUpper(fullWord)] = abbrev
	}

	// Split on spaces
	words := strings.Fields(title)
	var abbreviatedWords []string

	for _, word := range words {
		upperWord := strings.ToUpper(word)
		if abbrev, exists := reverseCache[upperWord]; exists {
			// Convert abbreviation to Title Case
			abbreviatedWords = append(abbreviatedWords, toTitleCase(abbrev))
		} else {
			// Keep original word but ensure Title Case
			abbreviatedWords = append(abbreviatedWords, toTitleCase(word))
		}
	}

	return strings.Join(abbreviatedWords, " "), nil
}

// toTitleCase converts a string to Title Case (first letter uppercase, rest lowercase)
func toTitleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + strings.ToLower(s[1:])
}

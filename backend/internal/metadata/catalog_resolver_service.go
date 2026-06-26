package metadata

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// CatalogResolverService handles the resolution of catalog references to canonical keys.
type CatalogResolverService struct {
	DB     *sqlx.DB
	Logger *zap.Logger
}

// NewCatalogResolverService creates a new CatalogResolverService.
func NewCatalogResolverService(db *sqlx.DB) *CatalogResolverService {
	return &CatalogResolverService{
		DB:     db,
		Logger: logging.GetLogger(),
	}
}

// CanonicalKeyFormat is the regex for validating canonical keys: <domain>:<entity>:<version>
var CanonicalKeyFormat = regexp.MustCompile(`^[a-z0-9_]+:[a-z0-9_]+:v[0-9]+$`)

// ResolveReference resolves a given reference (alias or key) to a canonical key.
func (s *CatalogResolverService) ResolveReference(ctx context.Context, tenantID uuid.UUID, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("empty reference")
	}

	// 1. Check if it's already a valid canonical key
	if CanonicalKeyFormat.MatchString(ref) {
		return ref, nil
	}

	// 2. Check if it's an alias in the database
	var canonicalKey string
	query := `SELECT canonical_key FROM catalog_aliases WHERE tenant_id = $1 AND alias = $2`
	err := s.DB.GetContext(ctx, &canonicalKey, query, tenantID, ref)
	if err == nil {
		s.Logger.Info("resolved catalog alias",
			zap.String("tenant_id", tenantID.String()),
			zap.String("alias", ref),
			zap.String("canonical_key", canonicalKey),
		)
		return canonicalKey, nil
	}

	s.Logger.Warn("failed to resolve catalog reference",
		zap.String("tenant_id", tenantID.String()),
		zap.String("reference", ref),
		zap.Error(err),
	)

	// 3. If not found, return error
	return "", fmt.Errorf("unresolvable reference: %s", ref)
}

// GetCanonicalKey constructs a canonical key from parts.
func (s *CatalogResolverService) GetCanonicalKey(domain, entity string, version int) string {
	return fmt.Sprintf("%s:%s:v%d", strings.ToLower(domain), strings.ToLower(entity), version)
}

// ValidateDAG checks for cycles and missing dependencies in a list of calculations.
func (s *CatalogResolverService) ValidateDAG(calculations []models.Calculation) error {
	// Map of calculation Name -> Calculation
	calcMap := make(map[string]models.Calculation)
	for _, c := range calculations {
		calcMap[c.Name] = c
	}

	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	var validate func(currentName string) error
	validate = func(currentName string) error {
		visited[currentName] = true
		recursionStack[currentName] = true

		calc, exists := calcMap[currentName]
		if !exists {
			// If it's not in the list, we assume it's an external dependency (e.g. a raw column)
			// which is fine for this check.
			recursionStack[currentName] = false
			return nil
		}

		// Extract dependencies from formula
		// This is a simplified regex to find potential references.
		// In a real implementation, we'd use a proper parser.
		// Matches words that look like identifiers or keys.
		refs := extractReferences(calc.Formula)
		for _, ref := range refs {
			if ref == currentName {
				continue // Ignore self-reference in formula if it's just the name appearing
			}
			
			if !visited[ref] {
				if err := validate(ref); err != nil {
					return err
				}
			} else if recursionStack[ref] {
				s.Logger.Error("dag cycle detected",
					zap.String("node", currentName),
					zap.String("dependency", ref),
				)
				return fmt.Errorf("cycle detected: %s -> %s", currentName, ref)
			}
		}

		recursionStack[currentName] = false
		return nil
	}

	for _, c := range calculations {
		if !visited[c.Name] {
			if err := validate(c.Name); err != nil {
				return err
			}
		}
	}

	return nil
}

// extractReferences is a helper to find potential dependencies in a formula.
func extractReferences(formula string) []string {
	// Matches words: [a-zA-Z_][a-zA-Z0-9_:]*
	// This will catch "revenue", "finance:metric:revenue:v1", etc.
	re := regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_:]*`)
	return re.FindAllString(formula, -1)
}

// RebuildIndex refreshes the in-memory cache of aliases.
// For this milestone, we'll just log that we're rebuilding.
// In a real implementation, this would populate a concurrent map or Redis.
func (s *CatalogResolverService) RebuildIndex(ctx context.Context, tenantID uuid.UUID) error {
	s.Logger.Info("rebuilding catalog index", zap.String("tenant_id", tenantID.String()))

	// Fetch all aliases
	var aliases []models.CatalogAlias
	query := `SELECT * FROM catalog_aliases WHERE tenant_id = $1`
	err := s.DB.SelectContext(ctx, &aliases, query, tenantID)
	if err != nil {
		s.Logger.Error("failed to fetch aliases for index rebuild",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to fetch aliases: %w", err)
	}

	s.Logger.Info("rebuilt catalog index",
		zap.String("tenant_id", tenantID.String()),
		zap.Int("alias_count", len(aliases)),
	)
	return nil
}

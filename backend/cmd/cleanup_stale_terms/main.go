// cleanup_stale_terms is a one-off migration that regenerates every
// semantic_term catalog node whose name is still a raw catalog qualified
// path (e.g. "/Public/Customers/Customer Id") left over from before the
// GenerateSemanticTerms rewrite. For each one it derives the same
// two-layer (semantic term + business term) structure the live endpoint
// now produces, reuses an existing correctly-named term when the concept
// already has one (merging tenant-wide, not per-datasource), repoints the
// column's MAPS_TO edge, and deletes the stale node.
//
// This intentionally reimplements (rather than imports) the small naming
// helpers from internal/api/glossary_handler.go, since those are
// unexported methods on GlossaryHandler; keeping this script self-contained
// avoids reaching into another package's internals for a one-off tool.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/services"
	"github.com/hondyman/uisce/backend/pkg/llm"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const tenantID = "99e99e99-99e9-49e9-89e9-99e99e99e999"

var camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

func tokenizeColumnName(name string) []string {
	spaced := camelBoundary.ReplaceAllString(name, "$1 $2")
	spaced = strings.NewReplacer("_", " ", ".", " ", "-", " ", "/", " ").Replace(spaced)
	var tokens []string
	for _, t := range strings.Fields(spaced) {
		if t != "" {
			tokens = append(tokens, t)
		}
	}
	return tokens
}

var commonShortWords = map[string]bool{
	"ID": true, "NO": true, "OF": true, "IN": true, "ON": true, "AT": true,
	"IS": true, "OR": true, "TO": true, "BY": true, "AN": true, "UP": true,
	"DUE": true, "NEW": true, "OLD": true, "KEY": true, "PIN": true,
}

func looksLikeAbbreviation(token string) bool {
	upper := strings.ToUpper(token)
	if commonShortWords[upper] {
		return false
	}
	if token == strings.ToUpper(token) && len(token) <= 6 {
		return true
	}
	return false
}

func titleCase(tokens []string) string {
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if t == "" {
			continue
		}
		if len(t) == 1 {
			out = append(out, strings.ToUpper(t))
			continue
		}
		out = append(out, strings.ToUpper(t[:1])+strings.ToLower(t[1:]))
	}
	return strings.Join(out, " ")
}

func pascalCase(tokens []string) string {
	var b strings.Builder
	for _, t := range tokens {
		if t == "" {
			continue
		}
		if t == strings.ToUpper(t) && len(t) <= 3 {
			b.WriteString(t)
			continue
		}
		b.WriteString(strings.ToUpper(t[:1]) + strings.ToLower(t[1:]))
	}
	return b.String()
}

var idSuffixWords = map[string]string{
	"IDENTIFIER": "ID",
	"CODE":       "CD",
}

type derivedNames struct {
	SemanticName string
	BusinessName string
}

func deriveTermNames(ctx context.Context, abbrevSvc *services.AbbreviationService, rawName string) derivedNames {
	tokens := tokenizeColumnName(rawName)
	if len(tokens) == 0 {
		return derivedNames{}
	}

	svcCtx := context.WithValue(ctx, "tenant_id", tenantID)
	abbrevs, err := abbrevSvc.GetAllAbbreviations(svcCtx)
	if err != nil {
		log.Printf("abbreviation lookup failed for %q, falling back to raw tokens: %v", rawName, err)
		return derivedNames{SemanticName: pascalCase(tokens), BusinessName: titleCase(tokens)}
	}
	abbrMap := make(map[string]string, len(abbrevs))
	for _, a := range abbrevs {
		abbrMap[strings.ToUpper(a.Abbreviation)] = a.FullWord
	}

	resolved := make([]string, len(tokens))
	var unresolvedIdx []int
	var unresolvedTokens []string
	for i, tok := range tokens {
		upper := strings.ToUpper(tok)
		if full, ok := abbrMap[upper]; ok {
			resolved[i] = full
		} else if looksLikeAbbreviation(tok) {
			resolved[i] = tok
			unresolvedIdx = append(unresolvedIdx, i)
			unresolvedTokens = append(unresolvedTokens, upper)
		} else {
			resolved[i] = tok
		}
	}

	if len(unresolvedTokens) > 0 {
		suggestions, err := abbrevSvc.SuggestExpansionsInContext(svcCtx, unresolvedTokens, rawName)
		if err != nil {
			log.Printf("LLM disambiguation failed for %v (%q): %v", unresolvedTokens, rawName, err)
		} else {
			for _, idx := range unresolvedIdx {
				upper := strings.ToUpper(tokens[idx])
				if full, ok := suggestions[upper]; ok && full != "" {
					resolved[idx] = full
					if addErr := abbrevSvc.AddAbbreviation(svcCtx, upper, full, "auto-learned via stale term cleanup"); addErr != nil {
						log.Printf("failed to persist learned abbreviation %s=%s: %v", upper, full, addErr)
					}
				}
			}
		}
	}

	semanticTokens := make([]string, len(resolved))
	copy(semanticTokens, resolved)
	if len(semanticTokens) > 0 {
		last := strings.ToUpper(semanticTokens[len(semanticTokens)-1])
		if abbr, ok := idSuffixWords[last]; ok {
			semanticTokens[len(semanticTokens)-1] = abbr
		}
	}

	return derivedNames{SemanticName: pascalCase(semanticTokens), BusinessName: titleCase(resolved)}
}

func resolveOrCreateNodeType(db *sql.DB, typeName string) (string, error) {
	var id string
	err := db.QueryRow(`SELECT id FROM catalog_node_type WHERE catalog_type_name = $1 LIMIT 1`, typeName).Scan(&id)
	if err == sql.ErrNoRows {
		err = db.QueryRow(
			`INSERT INTO catalog_node_type (tenant_id, catalog_type_name, created_at, updated_at) VALUES ($1, $2, NOW(), NOW()) RETURNING id`,
			tenantID, typeName,
		).Scan(&id)
	}
	return id, err
}

func resolveOrCreateEdgeType(db *sql.DB, typeName string) (string, error) {
	var id string
	err := db.QueryRow(`SELECT id FROM catalog_edge_type WHERE edge_type_name = $1 LIMIT 1`, typeName).Scan(&id)
	if err == sql.ErrNoRows {
		err = db.QueryRow(
			`INSERT INTO catalog_edge_type (tenant_id, edge_type_name, created_at, updated_at) VALUES ($1, $2, NOW(), NOW()) RETURNING id`,
			tenantID, typeName,
		).Scan(&id)
	}
	return id, err
}

func findOrCreateTermNode(db *sql.DB, datasourceID, nodeTypeID, qualifiedPrefix, name, definition string) (id string, reused bool, err error) {
	qualifiedPath := fmt.Sprintf("%s/%s", qualifiedPrefix, name)
	err = db.QueryRow(
		`SELECT id FROM catalog_node WHERE node_type_id = $1 AND tenant_id = $2 AND (qualified_path = $3 OR lower(node_name) = lower($4)) LIMIT 1`,
		nodeTypeID, tenantID, qualifiedPath, name,
	).Scan(&id)
	if err == nil && id != "" {
		return id, true, nil
	}
	properties := "{}"
	if definition != "" {
		if b, mErr := json.Marshal(map[string]string{"description": definition}); mErr == nil {
			properties = string(b)
		}
	}
	var dsArg interface{}
	if datasourceID != "" {
		dsArg = datasourceID
	}
	err = db.QueryRow(
		`INSERT INTO catalog_node (node_name, node_type_id, tenant_id, tenant_datasource_id, properties, qualified_path, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5::jsonb, $6, NOW(), NOW()) RETURNING id`,
		name, nodeTypeID, tenantID, dsArg, properties, qualifiedPath,
	).Scan(&id)
	if err != nil {
		return "", false, err
	}
	return id, false, nil
}

func ensureEdge(db *sql.DB, datasourceID, subjectID, objectID, edgeTypeID string) error {
	var existingID string
	err := db.QueryRow(
		`SELECT id FROM catalog_edge WHERE source_node_id = $1 AND target_node_id = $2 AND edge_type_id = $3 LIMIT 1`,
		subjectID, objectID, edgeTypeID,
	).Scan(&existingID)
	if err == nil && existingID != "" {
		return nil
	}
	var dsArg interface{}
	if datasourceID != "" {
		dsArg = datasourceID
	}
	_, err = db.Exec(
		`INSERT INTO catalog_edge (id, tenant_id, tenant_datasource_id, source_node_id, target_node_id, properties, edge_type_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, '{}'::jsonb, $6, NOW(), NOW())`,
		uuid.New().String(), tenantID, dsArg, subjectID, objectID, edgeTypeID,
	)
	return err
}

func withPublicSearchPath(dsn string) string {
	sep := "&"
	if !strings.Contains(dsn, "?") {
		sep = "?"
	}
	return dsn + sep + "options=" + url.QueryEscape("-c search_path=public")
}

func main() {
	limit := 0
	if len(os.Args) > 1 {
		fmt.Sscanf(os.Args[1], "%d", &limit)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:postgres@100.84.50.65:5432/alpha?sslmode=disable"
	}
	db, err := sql.Open("postgres", withPublicSearchPath(dbURL))
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	sqlxDB := sqlx.NewDb(db, "postgres")
	llmProvider := llm.NewGeminiProvider("", "")
	abbrevSvc := services.NewAbbreviationService(sqlxDB, llmProvider)

	semanticNodeTypeID, err := resolveOrCreateNodeType(db, "semantic_term")
	if err != nil {
		log.Fatalf("resolve semantic_term node type: %v", err)
	}
	businessNodeTypeID, err := resolveOrCreateNodeType(db, "business_term")
	if err != nil {
		log.Fatalf("resolve business_term node type: %v", err)
	}
	hasSemanticEdgeTypeID, err := resolveOrCreateEdgeType(db, "has_semantic_context")
	if err != nil {
		log.Fatalf("resolve has_semantic_context edge type: %v", err)
	}
	mapsToEdgeTypeID, err := resolveOrCreateEdgeType(db, "MAPS_TO")
	if err != nil {
		log.Fatalf("resolve MAPS_TO edge type: %v", err)
	}

	rows, err := db.Query(`
		SELECT cn.id, ce.id, ce.target_node_id, col.node_name, col.tenant_datasource_id
		FROM catalog_node cn
		JOIN catalog_edge ce ON ce.source_node_id = cn.id
		JOIN catalog_edge_type cet ON cet.id = ce.edge_type_id AND cet.edge_type_name = 'MAPS_TO'
		JOIN catalog_node col ON col.id = ce.target_node_id
		WHERE cn.node_type_id = $1
		  AND (cn.node_name LIKE '/%' OR cn.node_name LIKE '%/%')
		ORDER BY cn.node_name
	`, semanticNodeTypeID)
	if err != nil {
		log.Fatalf("query stale terms: %v", err)
	}

	type row struct {
		staleTermID, staleEdgeID, colID, colName string
		datasourceID                             sql.NullString
	}
	var toProcess []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.staleTermID, &r.staleEdgeID, &r.colID, &r.colName, &r.datasourceID); err != nil {
			log.Fatalf("scan: %v", err)
		}
		toProcess = append(toProcess, r)
	}
	rows.Close()

	if limit > 0 && limit < len(toProcess) {
		toProcess = toProcess[:limit]
	}
	log.Printf("processing %d stale semantic terms", len(toProcess))

	ctx := context.Background()
	ok, failed := 0, 0
	for i, r := range toProcess {
		ds := ""
		if r.datasourceID.Valid {
			ds = r.datasourceID.String
		}
		names := deriveTermNames(ctx, abbrevSvc, r.colName)
		if names.SemanticName == "" {
			log.Printf("[%d/%d] SKIP %s: could not derive a name from column %q", i+1, len(toProcess), r.staleTermID, r.colName)
			failed++
			continue
		}

		semanticID, semReused, err := findOrCreateTermNode(db, ds, semanticNodeTypeID, "semantic_term", names.SemanticName, "")
		if err != nil {
			log.Printf("[%d/%d] FAIL resolve semantic term %q: %v", i+1, len(toProcess), names.SemanticName, err)
			failed++
			continue
		}

		if err := ensureEdge(db, ds, semanticID, r.colID, mapsToEdgeTypeID); err != nil {
			log.Printf("[%d/%d] FAIL link column %s to %s: %v", i+1, len(toProcess), r.colID, names.SemanticName, err)
			failed++
			continue
		}

		businessName := names.BusinessName
		if businessName == "" {
			businessName = names.SemanticName
		}
		var definition string
		if def, defErr := abbrevSvc.GenerateStandardDefinition(context.WithValue(ctx, "tenant_id", tenantID), businessName, r.colName); defErr == nil {
			definition = def.Definition
		}
		businessID, bizReused, err := findOrCreateTermNode(db, ds, businessNodeTypeID, "business_term", businessName, definition)
		if err != nil {
			log.Printf("[%d/%d] FAIL resolve business term %q: %v", i+1, len(toProcess), businessName, err)
			failed++
			continue
		}
		if err := ensureEdge(db, ds, businessID, semanticID, hasSemanticEdgeTypeID); err != nil {
			log.Printf("[%d/%d] WARN link business term %s to semantic term %s: %v", i+1, len(toProcess), businessID, semanticID, err)
		}

		// Delete the old stale edge and, if it was superseded (not just
		// reused), the old stale node too.
		if _, err := db.Exec(`DELETE FROM catalog_edge WHERE id = $1`, r.staleEdgeID); err != nil {
			log.Printf("[%d/%d] WARN delete stale edge %s: %v", i+1, len(toProcess), r.staleEdgeID, err)
		}
		if semanticID != r.staleTermID {
			if _, err := db.Exec(`DELETE FROM catalog_node WHERE id = $1`, r.staleTermID); err != nil {
				log.Printf("[%d/%d] WARN delete stale term %s: %v", i+1, len(toProcess), r.staleTermID, err)
			}
		}

		log.Printf("[%d/%d] OK %q -> semantic=%q(reused=%v) business=%q(reused=%v)",
			i+1, len(toProcess), r.colName, names.SemanticName, semReused, businessName, bizReused)
		ok++
	}

	log.Printf("done: %d ok, %d failed", ok, failed)
}

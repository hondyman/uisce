package api

import (
	"context"
	"log"
	"strings"
)

// Term reuse. A semantic term names a meaning, not a spelling: TenantId, TenantIdentifier and Tenant_ID are one
// concept. The generator used to reuse an existing term only when the derived name matched EXACTLY (ignoring
// case), so once the abbreviation dictionary expanded "id" to "Identifier" every column named *_id got a new
// *Identifier twin next to the existing *Id term, splitting one concept over two terms (27 pairs in CRIMS) and
// leaving rules written against one twin blind to the other's columns.

// builtinSynonyms are used only when the dictionary has no entry for a token, so the two most common
// spellings of the same word still compare equal without a dictionary.
var builtinSynonyms = map[string]string{"ID": "IDENTIFIER", "CD": "CODE"}

// canonicalTermKey returns the same key for names that mean the same thing: the name is split into words
// (camelCase and separators), each word is expanded with the abbreviation dictionary, and the result is
// lower-cased and joined. abbr is keyed by upper-cased abbreviation, as buildAbbreviationMap returns it.
func canonicalTermKey(name string, abbr map[string]string) string {
	var b strings.Builder
	for _, tok := range tokenizeColumnName(name) {
		word := tok
		if full, ok := abbr[strings.ToUpper(tok)]; ok && full != "" {
			word = full
		} else if syn, ok := builtinSynonyms[strings.ToUpper(tok)]; ok {
			word = syn
		}
		b.WriteString(strings.ToLower(strings.Join(strings.Fields(word), "")))
	}
	return b.String()
}

// termIndex maps a canonical key to the name of the existing term to reuse. The first name added for a key wins,
// so callers add the best term first (most mapped columns, then oldest).
type termIndex struct{ byKey map[string]string }

func newTermIndex() *termIndex { return &termIndex{byKey: map[string]string{}} }

func (ix *termIndex) add(name string, abbr map[string]string) {
	if ix == nil || name == "" {
		return
	}
	key := canonicalTermKey(name, abbr)
	if key == "" {
		return
	}
	if _, exists := ix.byKey[key]; !exists {
		ix.byKey[key] = name
	}
}

// reuse returns the existing term that means the same as name, if there is one.
func (ix *termIndex) reuse(name string, abbr map[string]string) (string, bool) {
	if ix == nil {
		return "", false
	}
	existing, ok := ix.byKey[canonicalTermKey(name, abbr)]
	return existing, ok
}

// loadTermIndex indexes the semantic terms visible to the tenant (its own and the gold copy's), best first:
// the term with the most mapped columns wins, then the oldest. A load failure yields an empty index, which
// is the old behaviour (exact-name reuse only) rather than a failed generation.
func (s *GlossaryService) loadTermIndex(ctx context.Context, tenantID string, abbr map[string]string) *termIndex {
	ix := newTermIndex()
	if s == nil || s.db == nil {
		return ix
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT n.node_name
		FROM catalog_node n
		JOIN catalog_node_type t ON t.id = n.node_type_id AND t.catalog_type_name = 'semantic_term'
		LEFT JOIN (
			SELECT e.source_node_id, count(*) AS mapped
			FROM catalog_edge e
			JOIN catalog_edge_type et ON et.id = e.edge_type_id AND et.edge_type_name = 'MAPS_TO'
			GROUP BY e.source_node_id
		) m ON m.source_node_id = n.id
		WHERE n.tenant_id = $1 OR n.tenant_id = public.uisce_gold_copy_tenant_id()
		ORDER BY COALESCE(m.mapped, 0) DESC, n.created_at ASC, n.node_name`, tenantID)
	if err != nil {
		log.Printf("[term reuse] could not load existing terms, reusing exact names only: %v", err)
		return ix
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			ix.add(name, abbr)
		}
	}
	return ix
}

// reuseExistingTerm maps a derived name onto an existing term with the same meaning, per generation batch: the
// abbreviation map and term index are loaded once into the batch cache. Without a cache they are loaded for
// this call only. Returns name unchanged when there is nothing to reuse.
func (s *GlossaryService) reuseExistingTerm(ctx context.Context, tenantID, name string, cache *termCache) string {
	if name == "" {
		return name
	}
	abbr, ix := s.batchTermIndex(ctx, tenantID, cache)
	if existing, ok := ix.reuse(name, abbr); ok {
		return existing
	}
	return name
}

func (s *GlossaryService) batchTermIndex(ctx context.Context, tenantID string, cache *termCache) (map[string]string, *termIndex) {
	load := func() (map[string]string, *termIndex) {
		var lookup abbreviationLookup
		if s.abbrevSvc != nil {
			lookup = abbreviationSvcAdapter{real: s.abbrevSvc}
		}
		abbr := buildAbbreviationMap(ctx, lookup, tenantID)
		return abbr, s.loadTermIndex(ctx, tenantID, abbr)
	}
	if cache == nil {
		return load()
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.index == nil {
		cache.abbr, cache.index = load()
	}
	return cache.abbr, cache.index
}

// noteNewTerm makes a term created in this batch reusable by the rest of the batch.
func (c *termCache) noteNewTerm(name string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.index != nil {
		c.index.add(name, c.abbr)
	}
}

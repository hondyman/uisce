package vocabulary

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"
)

type Resolver struct {
	db *sqlx.DB
}

func NewResolver(db *sqlx.DB) *Resolver {
	return &Resolver{db: db}
}

func (r *Resolver) DB() *sqlx.DB {
	return r.db
}

type CanonicalTerm struct {
	TermID         string  `json:"term_id"`
	TermName       string  `json:"term_name"`
	TermKey        string  `json:"term_key,omitempty"`
	CanonicalKey   string  `json:"canonical_key,omitempty"`
	NodeID         string  `json:"node_id"`
	MatchedVia     string  `json:"matched_via"`
	MatchedToken   string  `json:"matched_token"`
	SemanticTerm   *string `json:"semantic_term,omitempty"`
	SemanticTermID *string `json:"semantic_term_id,omitempty"`
	Scope          string  `json:"scope,omitempty"`
}

// ResolveTerm resolves a raw token to zero or more canonical business terms.
//
// Strategies run in order and short-circuit on the first one that produces
// a result: alias edges, synonym edges, then a direct business_term name
// match. A fourth strategy — embedding/fuzzy similarity — previously lived
// here but queried a `business_terms` table that does not exist on live
// deployments; it silently swallowed the resulting error and always
// contributed nothing. It has been removed rather than left as dead code.
// Fuzzy/embedding resolution is tracked as its own piece of work
// (docs/TERM_RESOLUTION_DESIGN.md §4, §8.2 step 4 — `catalog_node_embedding`)
// and should be reintroduced once that table exists, not resurrected here.
//
// Every strategy now records a hit/miss/error outcome via ResolutionAttempts
// so a strategy that silently stops matching (edge type never populated,
// query broken by a schema change, etc.) shows up as a metric instead of
// requiring another schema audit to discover.
func (r *Resolver) ResolveTerm(ctx context.Context, tenantID, rawToken string) ([]CanonicalTerm, error) {
	if rawToken == "" {
		return nil, nil
	}
	normalized := strings.ToLower(strings.TrimSpace(rawToken))
	if normalized == "" {
		return nil, nil
	}

	var results []CanonicalTerm

	err := r.resolveViaAlias(ctx, tenantID, normalized, &results)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		err = r.resolveViaSynonym(ctx, tenantID, normalized, &results)
		if err != nil {
			return nil, err
		}
	}

	if len(results) == 0 {
		err = r.resolveViaBusinessTermName(ctx, tenantID, normalized, &results)
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// resolveViaAlias matches against ALIAS_OF edges from a `synonym` node to a
// business_term node.
//
// NOTE on edge typing convention: this query matches on the denormalized
// `catalog_edge.relationship_type` text column, not the normalized
// `edge_type_id` FK to `catalog_edge_types`. Live data on alpha shows these
// two columns are not kept in sync for other edge types (e.g. MAPS_TO has
// 234 rows typed via edge_type_id vs. 14 via relationship_type — see
// docs/TERM_RESOLUTION_DESIGN.md §0). As of this fix, ALIAS_OF has zero rows
// under either column, so this is not yet an observed discrepancy for this
// edge type specifically — but a future write path that only sets
// edge_type_id (the convention the design doc adopts as authoritative, §0)
// would be invisible to this query as written. That reconciliation is a
// schema/registration decision (§8.2 step 3: registering ALIAS_OF with
// source/target type constraints), not part of this fix — left as-is here
// so this PR stays a bug fix, not a rewrite.
func (r *Resolver) resolveViaAlias(ctx context.Context, tenantID, token string, out *[]CanonicalTerm) error {
	const strategy = "alias"
	query := `
		SELECT DISTINCT
			bn.id               AS term_id,
			bn.node_name        AS term_name,
			cn.node_name        AS matched_token,
			'ALIAS_OF'          AS matched_via,
			bn.tenant_id        AS tenant_id
		FROM catalog_node alias_n
		JOIN catalog_edge ce
		  ON ce.source_node_id = alias_n.id
		 AND ce.relationship_type = 'ALIAS_OF'
		JOIN catalog_node bn
		  ON bn.id = ce.target_node_id
		WHERE alias_n.tenant_id = $1
		  AND alias_n.node_type_id IN (
		      SELECT id FROM catalog_node_type WHERE catalog_type_name = 'synonym'
		  )
		  AND LOWER(alias_n.node_name) = $2
		LIMIT 5
	`
	rows, err := r.db.QueryxContext(ctx, query, tenantID, token)
	if err != nil {
		recordError(tenantID, strategy)
		return err
	}
	defer rows.Close()

	matched := 0
	for rows.Next() {
		var t CanonicalTerm
		if err := rows.Scan(&t.TermID, &t.TermName, &t.MatchedToken, &t.MatchedVia, &t.NodeID); err != nil {
			recordError(tenantID, strategy)
			return err
		}
		t.MatchedToken = token
		r.enrichWithSemanticTerm(ctx, tenantID, &t)
		*out = append(*out, t)
		matched++
	}
	if err := rows.Err(); err != nil {
		recordError(tenantID, strategy)
		return err
	}

	if matched > 0 {
		recordHit(tenantID, strategy)
	} else {
		recordMiss(tenantID, strategy)
	}
	return nil
}

func (r *Resolver) resolveViaSynonym(ctx context.Context, tenantID, token string, out *[]CanonicalTerm) error {
	const strategy = "synonym"
	query := `
		SELECT DISTINCT
			bn.id               AS term_id,
			bn.node_name        AS term_name,
			syn_n.node_name     AS matched_token,
			'HAS_SYNONYM'       AS matched_via,
			bn.tenant_id        AS tenant_id
		FROM catalog_node syn_n
		JOIN catalog_edge ce
		  ON ce.source_node_id = syn_n.id
		 AND ce.relationship_type = 'HAS_SYNONYM'
		JOIN catalog_node bn
		  ON bn.id = ce.target_node_id
		WHERE syn_n.tenant_id = $1
		  AND syn_n.node_type_id IN (
		      SELECT id FROM catalog_node_type WHERE catalog_type_name = 'synonym'
		  )
		  AND LOWER(syn_n.node_name) = $2
		LIMIT 5
	`
	rows, err := r.db.QueryxContext(ctx, query, tenantID, token)
	if err != nil {
		recordError(tenantID, strategy)
		return err
	}
	defer rows.Close()

	matched := 0
	for rows.Next() {
		var t CanonicalTerm
		if err := rows.Scan(&t.TermID, &t.TermName, &t.MatchedToken, &t.MatchedVia, &t.NodeID); err != nil {
			recordError(tenantID, strategy)
			return err
		}
		t.MatchedToken = token
		r.enrichWithSemanticTerm(ctx, tenantID, &t)
		*out = append(*out, t)
		matched++
	}
	if err := rows.Err(); err != nil {
		recordError(tenantID, strategy)
		return err
	}

	if matched > 0 {
		recordHit(tenantID, strategy)
	} else {
		recordMiss(tenantID, strategy)
	}
	return nil
}

func (r *Resolver) resolveViaBusinessTermName(ctx context.Context, tenantID, token string, out *[]CanonicalTerm) error {
	const strategy = "business_term_name"
	query := `
		SELECT DISTINCT
			cn.id               AS term_id,
			cn.node_name        AS term_name,
			cn.node_name        AS matched_token,
			'DIRECT'            AS matched_via,
			cn.tenant_id        AS tenant_id
		FROM catalog_node cn
		JOIN catalog_node_type cnt ON cnt.id = cn.node_type_id
		WHERE cn.tenant_id = $1
		  AND cnt.catalog_type_name = 'business_term'
		  AND LOWER(cn.node_name) = $2
		LIMIT 5
	`
	rows, err := r.db.QueryxContext(ctx, query, tenantID, token)
	if err != nil {
		recordError(tenantID, strategy)
		return err
	}
	defer rows.Close()

	matched := 0
	for rows.Next() {
		var t CanonicalTerm
		if err := rows.Scan(&t.TermID, &t.TermName, &t.MatchedToken, &t.MatchedVia, &t.NodeID); err != nil {
			recordError(tenantID, strategy)
			return err
		}
		t.MatchedToken = token
		r.enrichWithSemanticTerm(ctx, tenantID, &t)
		*out = append(*out, t)
		matched++
	}
	if err := rows.Err(); err != nil {
		recordError(tenantID, strategy)
		return err
	}

	if matched > 0 {
		recordHit(tenantID, strategy)
	} else {
		recordMiss(tenantID, strategy)
	}
	return nil
}

// enrichWithSemanticTerm attaches the semantic term bound to a resolved
// business term, when one exists.
//
// NOTE: live data (docs/TERM_RESOLUTION_DESIGN.md §0) shows the real bridge
// edge is HAS_BUSINESS_TERM, directed semantic_term -> business_term — the
// opposite direction and a different name than this query previously used
// (MAPS_TO_SEMANTIC_TERM, business_term -> semantic_term), which is why
// enrichment never actually populated SemanticTerm/SemanticTermID in
// production even on the one strategy (business_term_name) that does match.
// Fixed to query the real edge in the real direction.
//
// A no-rows result here is an expected, legitimate outcome (most business
// terms may have no bound semantic term yet) and is not an error. Any other
// error is now surfaced via the error metric instead of being swallowed —
// previously this function ignored every error indiscriminately, including
// real query/connection failures, not just "no match."
func (r *Resolver) enrichWithSemanticTerm(ctx context.Context, tenantID string, t *CanonicalTerm) {
	const strategy = "enrich_semantic_term"
	query := `
		SELECT st.node_name, st.id
		FROM catalog_node st
		JOIN catalog_edge ce
		  ON ce.source_node_id = st.id
		 AND ce.relationship_type = 'HAS_BUSINESS_TERM'
		JOIN catalog_node bt
		  ON bt.id = ce.target_node_id
		WHERE bt.id = $1
		  AND bt.tenant_id = $2
		LIMIT 1
	`
	var semName, semID string
	err := r.db.QueryRowxContext(ctx, query, t.NodeID, tenantID).Scan(&semName, &semID)
	switch {
	case err == nil:
		t.SemanticTerm = &semName
		t.SemanticTermID = &semID
	case errors.Is(err, sql.ErrNoRows):
		// Expected: this business term has no bound semantic term yet.
	default:
		recordError(tenantID, strategy)
	}
}

func (r *Resolver) ResolveTerms(ctx context.Context, tenantID string, tokens []string) (map[string][]CanonicalTerm, error) {
	results := make(map[string][]CanonicalTerm)
	for _, token := range tokens {
		terms, err := r.ResolveTerm(ctx, tenantID, token)
		if err != nil {
			return nil, err
		}
		if len(terms) > 0 {
			results[token] = terms
		}
	}
	return results, nil
}

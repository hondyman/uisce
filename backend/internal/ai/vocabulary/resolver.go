package vocabulary

import (
	"context"
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
	TermID         string   `json:"term_id"`
	TermName       string   `json:"term_name"`
	TermKey        string   `json:"term_key,omitempty"`
	CanonicalKey   string   `json:"canonical_key,omitempty"`
	NodeID         string   `json:"node_id"`
	MatchedVia     string   `json:"matched_via"`
	MatchedToken   string   `json:"matched_token"`
	SemanticTerm   *string `json:"semantic_term,omitempty"`
	SemanticTermID *string `json:"semantic_term_id,omitempty"`
	Scope          string   `json:"scope,omitempty"`
}

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

	if len(results) == 0 {
		err = r.resolveViaEmbedding(ctx, tenantID, normalized, &results)
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

func (r *Resolver) resolveViaAlias(ctx context.Context, tenantID, token string, out *[]CanonicalTerm) error {
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
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var t CanonicalTerm
		if err := rows.Scan(&t.TermID, &t.TermName, &t.MatchedToken, &t.MatchedVia, &t.NodeID); err != nil {
			return err
		}
		t.MatchedToken = token
		r.enrichWithSemanticTerm(ctx, tenantID, &t)
		*out = append(*out, t)
	}
	return rows.Err()
}

func (r *Resolver) resolveViaSynonym(ctx context.Context, tenantID, token string, out *[]CanonicalTerm) error {
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
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var t CanonicalTerm
		if err := rows.Scan(&t.TermID, &t.TermName, &t.MatchedToken, &t.MatchedVia, &t.NodeID); err != nil {
			return err
		}
		t.MatchedToken = token
		r.enrichWithSemanticTerm(ctx, tenantID, &t)
		*out = append(*out, t)
	}
	return rows.Err()
}

func (r *Resolver) resolveViaBusinessTermName(ctx context.Context, tenantID, token string, out *[]CanonicalTerm) error {
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
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var t CanonicalTerm
		if err := rows.Scan(&t.TermID, &t.TermName, &t.MatchedToken, &t.MatchedVia, &t.NodeID); err != nil {
			return err
		}
		t.MatchedToken = token
		r.enrichWithSemanticTerm(ctx, tenantID, &t)
		*out = append(*out, t)
	}
	return rows.Err()
}

func (r *Resolver) resolveViaEmbedding(ctx context.Context, tenantID, token string, out *[]CanonicalTerm) error {
	query := `
		SELECT id, term, canonical_key, scope
		FROM business_terms
		WHERE tenant_id = $1
		  AND embedding IS NOT NULL
		  AND vector_dims(embedding) = 1536
		ORDER BY embedding <=> (
		    -- Generate a placeholder embedding for fuzzy fallback
		    -- In production this would be replaced by a real embedding call
		    -- or a pre-computed average vector for the token
		    COALESCE(
		        (SELECT embedding FROM business_terms
		         WHERE tenant_id = $1 AND LOWER(term) = $2 LIMIT 1),
		        (SELECT embedding FROM business_terms
		         WHERE tenant_id = $1 LIMIT 1)
		    )
		)
		LIMIT 3
	`
	rows, err := r.db.QueryxContext(ctx, query, tenantID, token)
	if err != nil {
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var termID, term, canonicalKey, scope string
		if err := rows.Scan(&termID, &term, &canonicalKey, &scope); err != nil {
			continue
		}
		t := CanonicalTerm{
			TermID:       termID,
			TermName:     term,
			CanonicalKey: canonicalKey,
			MatchedVia:   "EMBEDDING_SIMILARITY",
			MatchedToken: token,
			Scope:        scope,
		}
		r.enrichWithSemanticTerm(ctx, tenantID, &t)
		*out = append(*out, t)
	}
	return rows.Err()
}

func (r *Resolver) enrichWithSemanticTerm(ctx context.Context, tenantID string, t *CanonicalTerm) {
	query := `
		SELECT st.node_name, st.id
		FROM catalog_node bt
		JOIN catalog_edge ce
		  ON ce.source_node_id = bt.id
		 AND ce.relationship_type = 'MAPS_TO_SEMANTIC_TERM'
		JOIN catalog_node st
		  ON st.id = ce.target_node_id
		WHERE bt.id = $1
		  AND bt.tenant_id = $2
		LIMIT 1
	`
	var semName, semID string
	err := r.db.QueryRowxContext(ctx, query, t.NodeID, tenantID).Scan(&semName, &semID)
	if err == nil {
		t.SemanticTerm = &semName
		t.SemanticTermID = &semID
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

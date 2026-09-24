package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/services"
)

const syncThreshold = 25

type GlossaryService struct {
	db        *sql.DB
	abbrevSvc *services.AbbreviationService
	jobStore  *InMemoryJobStore
	appCtx   context.Context
}

func NewGlossaryService(ctx context.Context, db *sql.DB, abbrevSvc *services.AbbreviationService, jobStore *InMemoryJobStore) *GlossaryService {
	return &GlossaryService{
		db:        db,
		abbrevSvc: abbrevSvc,
		jobStore:  jobStore,
		appCtx:   ctx,
	}
}

// DB exposes the underlying database connection for use by handlers that need
// to resolve column metadata (qualified_path, datasource_id) not available in
// the service layer.
func (s *GlossaryService) DB() *sql.DB {
	return s.db
}

func (s *GlossaryService) deriveTermNames(ctx context.Context, tenantID, rawName string, tableSchemaContext string, siblingColumnNames []string) derivedTermNames {
	tokens := tokenizeColumnName(rawName)
	if len(tokens) == 0 {
		return derivedTermNames{}
	}
	if s.abbrevSvc == nil {
		return derivedTermNames{SemanticName: pascalCase(tokens), BusinessName: titleCase(tokens)}
	}

	svcCtx := context.WithValue(ctx, "tenant_id", tenantID)
	contextSensitive := false

	abbrevs, err := s.abbrevSvc.GetAllAbbreviations(svcCtx)
	if err != nil {
		log.Printf("[deriveTermNames] abbreviation lookup failed, falling back to raw tokens: %v", err)
		return derivedTermNames{SemanticName: pascalCase(tokens), BusinessName: titleCase(tokens)}
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
		log.Printf("[deriveTermNames] calling SuggestExpansionsInContext for %d unresolved tokens: %v", len(unresolvedTokens), unresolvedTokens)
		suggestions, err := s.abbrevSvc.SuggestExpansionsInContext(svcCtx, unresolvedTokens, rawName, tableSchemaContext, siblingColumnNames)
		if err != nil {
			log.Printf("[deriveTermNames] LLM disambiguation failed for %v: %v", unresolvedTokens, err)
		} else {
			contextSensitive = true // LLM expansion prompt included table context
			for _, idx := range unresolvedIdx {
				upper := strings.ToUpper(tokens[idx])
				if full, ok := suggestions[upper]; ok && sanitizeExpansion(full) != "" {
					full = sanitizeExpansion(full)
					resolved[idx] = full
					if addErr := s.abbrevSvc.AddAbbreviation(svcCtx, upper, full, "auto-learned via semantic term generation"); addErr != nil {
						log.Printf("[deriveTermNames] failed to persist learned abbreviation %s=%s: %v", upper, full, addErr)
					}
				}
			}
		}
	}

	tableName := extractTableNameFromContext(tableSchemaContext)
	if len(resolved) == 1 && strings.EqualFold(resolved[0], rawName) && isGenericWord(resolved[0]) && tableSchemaContext != "" {
		if tableName != "" && !strings.EqualFold(tableName, rawName) {
			log.Printf("[deriveTermNames] bare generic word %q detected — invoking LLM qualification with table %q", rawName, tableName)
			if qualified, qualErr := s.abbrevSvc.QualifyGenericWord(svcCtx, resolved[0], tableName, siblingColumnNames); qualErr == nil && qualified != "" {
				log.Printf("[deriveTermNames] LLM qualified %q → semanticName=%q, businessName=%q, base=%q", rawName, qualified, titleCase(pascalCaseToWords(qualified)), strings.Title(strings.ToLower(resolved[0])))
				return derivedTermNames{
					SemanticName:     qualified,
					BusinessName:     titleCase(pascalCaseToWords(qualified)),
					BaseGenericTerm:  strings.Title(strings.ToLower(resolved[0])),
					ContextSensitive: true, // LLM qualification prompt included table name
				}
			} else if qualErr != nil {
				log.Printf("[deriveTermNames] LLM qualification failed for %q: %v", rawName, qualErr)
			}
		}
	}

	result := deriveTermNamesDeterministic(resolved, rawName, tableSchemaContext)
	result.ContextSensitive = result.ContextSensitive || contextSensitive
	return result
}

func (s *GlossaryService) resolveOrCreateNodeType(tenantID, typeName string) (string, error) {
	var id string
	err := s.db.QueryRow(`SELECT id FROM catalog_node_type WHERE catalog_type_name = $1 LIMIT 1`, typeName).Scan(&id)
	if err == sql.ErrNoRows {
		err = s.db.QueryRow(
			`INSERT INTO catalog_node_type (tenant_id, catalog_type_name, created_at, updated_at) VALUES ($1, $2, NOW(), NOW()) RETURNING id`,
			tenantID, typeName,
		).Scan(&id)
	}
	return id, err
}

func (s *GlossaryService) resolveOrCreateEdgeType(tenantID, typeName string) (string, error) {
	var id string
	err := s.db.QueryRow(`SELECT id FROM catalog_edge_type WHERE edge_type_name = $1 LIMIT 1`, typeName).Scan(&id)
	if err == sql.ErrNoRows {
		err = s.db.QueryRow(
			`INSERT INTO catalog_edge_type (tenant_id, edge_type_name, created_at, updated_at) VALUES ($1, $2, NOW(), NOW()) RETURNING id`,
			tenantID, typeName,
		).Scan(&id)
	}
	return id, err
}

// findOrCreateTermNode implements a two-phase find-or-create with distinct guarantees.
//
// Phase 1 — advisory pre-check (SELECT by node_type_id + node_name):
//   Handles semantic dedup: two callers passing semantically equivalent names
//   (e.g. "AccountId" vs "account_id" after tokenization) but different
//   qualified paths collapse to one existing node. This is a plain read with no
//   row lock; concurrent callers can both pass it. That's fine — its job is the
//   common (non-racing) case.
//
// Phase 2 — INSERT with ON CONFLICT (tenant_id, qualified_path) DO NOTHING:
//   Closes the concurrent-insert race. Two callers who both miss Phase 1 and
//   race to INSERT the same qualified_path: one wins, the other gets zero rows
//   back from RETURNING and falls through to the re-SELECT keyed only on
//   (tenant_id, qualified_path) — the row that won the unique-index constraint.
//
// Neither phase is redundant. The pre-check handles the cross-path name match;
// ON CONFLICT handles the concurrent-insert race.
func (s *GlossaryService) findOrCreateTermNode(ctx context.Context, tenantID, datasourceID, nodeTypeID, qualifiedPrefix, name, definition string) (id string, reused bool, err error) {
	if datasourceID == "none" {
		datasourceID = ""
	}
	qualifiedPath := fmt.Sprintf("%s/%s", qualifiedPrefix, name)

	// Phase 1: advisory pre-check — handles cross-path name collisions
	err = s.db.QueryRowContext(ctx,
		`SELECT id FROM catalog_node WHERE node_type_id = $1 AND tenant_id = $2 AND (qualified_path = $3 OR lower(node_name) = lower($4)) LIMIT 1`,
		nodeTypeID, tenantID, qualifiedPath, name,
	).Scan(&id)
	if err == nil && id != "" {
		return id, true, nil
	}

	// Phase 2: atomic insert — closes the concurrent-insert race
	properties := "{}"
	if definition != "" {
		propsBytes, marshalErr := json.Marshal(map[string]string{"description": definition})
		if marshalErr == nil {
			properties = string(propsBytes)
		}
	}
	err = s.db.QueryRowContext(ctx,
		`INSERT INTO catalog_node (node_name, node_type_id, tenant_id, tenant_datasource_id, properties, qualified_path, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5::jsonb, $6, NOW(), NOW())
		 ON CONFLICT (tenant_id, qualified_path) DO NOTHING
		 RETURNING id`,
		name, nodeTypeID, tenantID, datasourceID, properties, qualifiedPath,
	).Scan(&id)
	if err == nil && id != "" {
		// Row newly inserted
		return id, false, nil
	}
	if err != sql.ErrNoRows {
		// Unexpected error
		return "", false, err
	}

	// Conflict: ON CONFLICT DO NOTHING fired; re-SELECT by the unique key
	// to retrieve the existing row's id. Predicate is narrowed to
	// (tenant_id, qualified_path) because that's what caused the conflict.
	err = s.db.QueryRowContext(ctx,
		`SELECT id FROM catalog_node WHERE tenant_id = $1 AND qualified_path = $2 LIMIT 1`,
		tenantID, qualifiedPath,
	).Scan(&id)
	if err != nil {
		return "", false, err
	}
	return id, true, nil
}

func (s *GlossaryService) ensureEdge(tenantID, datasourceID, subjectID, objectID, edgeTypeID string) error {
	if datasourceID == "none" {
		datasourceID = ""
	}
	var existingID string
	err := s.db.QueryRow(
		`SELECT id FROM catalog_edge WHERE source_node_id = $1 AND target_node_id = $2 AND edge_type_id = $3 LIMIT 1`,
		subjectID, objectID, edgeTypeID,
	).Scan(&existingID)
	if err == nil && existingID != "" {
		return nil
	}
	_, err = s.db.Exec(
		`INSERT INTO catalog_edge (id, tenant_id, tenant_datasource_id, source_node_id, target_node_id, properties, edge_type_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, '{}'::jsonb, $6, NOW(), NOW())`,
		uuid.New().String(), tenantID, datasourceID, subjectID, objectID, edgeTypeID,
	)
	return err
}

func (s *GlossaryService) generateSingleTerm(ctx context.Context, tenantID, defaultDatasourceID string, item generateTermItem, cache *termCache, rejections rejectionSet) (*generateTermResult, error) {
	if len(item.ColumnIDs) == 0 {
		return nil, fmt.Errorf("column_ids is required")
	}

	datasourceID := defaultDatasourceID
	if datasourceID == "" || datasourceID == "none" {
		datasourceID = ""
		_ = s.db.QueryRow(
			`SELECT tenant_datasource_id::text FROM catalog_node WHERE id = $1 AND tenant_datasource_id IS NOT NULL`,
			item.ColumnIDs[0],
		).Scan(&datasourceID)
	}

	var columnNodeName, qualifiedPath string
	rowErr := s.db.QueryRow(`
		SELECT node_name, COALESCE(qualified_path, '')
		FROM catalog_node
		WHERE id = $1 AND tenant_id = $2
	`, item.ColumnIDs[0], tenantID).Scan(&columnNodeName, &qualifiedPath)
	if rowErr != nil || columnNodeName == "" {
		return nil, fmt.Errorf("could not resolve a name for this term: column not found (tenant=%s): %v", tenantID, rowErr)
	}

	var tableSchemaContext string
	var siblingColumnNames []string
	if qualifiedPath != "" {
		var schemaName, tableName string
		if strings.HasPrefix(qualifiedPath, "/") {
			parts := strings.Split(strings.TrimPrefix(qualifiedPath, "/"), "/")
			if len(parts) >= 3 {
				schemaName = parts[len(parts)-3]
				tableName = parts[len(parts)-2]
				tableSchemaContext = fmt.Sprintf("table %q in schema %q", tableName, schemaName)
			}
		} else {
			parts := strings.Split(qualifiedPath, ".")
			if len(parts) >= 3 {
				schemaName = parts[len(parts)-3]
				tableName = parts[len(parts)-2]
				tableSchemaContext = fmt.Sprintf("table %q in schema %q", tableName, schemaName)
			}
		}

		if tableName != "" && schemaName != "" {
			var rows *sql.Rows
			var err error
			if strings.HasPrefix(qualifiedPath, "/") {
				tablePrefix := "/" + schemaName + "/" + tableName + "/"
				rows, err = s.db.Query(`
					SELECT node_name FROM catalog_node
					WHERE tenant_id = $1
					  AND qualified_path LIKE $2
					  AND qualified_path != $3
					LIMIT 40
				`, tenantID, tablePrefix+"%", qualifiedPath)
			} else {
				tablePrefix := schemaName + "." + tableName + "."
				rows, err = s.db.Query(`
					SELECT node_name FROM catalog_node
					WHERE tenant_id = $1
					  AND qualified_path LIKE $2
					  AND qualified_path != $3
					LIMIT 40
				`, tenantID, tablePrefix+"%", qualifiedPath)
			}
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var siblingName string
					if err := rows.Scan(&siblingName); err == nil {
						siblingColumnNames = append(siblingColumnNames, siblingName)
					}
				}
			}
		}
	}

	names := s.deriveTermNames(ctx, tenantID, columnNodeName, tableSchemaContext, siblingColumnNames)
	semanticName := names.SemanticName

	// If the user provided an explicit name (dirty / edited cell), use it verbatim.
	// Rejection logic does NOT override user edits — rejections must only ever record
	// derived candidate names, not user-typed values.
	// See: amendment 3 — ✕ is disabled when dirtyColumns.has(id).
	if item.Name != "" && !strings.Contains(item.Name, "/") {
		semanticName = item.Name
	} else if rejections != nil {
		// No user edit: apply rejection filtering to the ranked candidate list.
		// Use the deterministic path's resolved tokens for candidate generation.
		candidates := deriveTermNamesCandidates(
			resolveTokensWithAbbreviations(ctx, s.abbrevSvc, tenantID, tokenizeColumnName(columnNodeName)),
			columnNodeName,
			tableSchemaContext,
		)
		if len(candidates) > 0 {
			chosen, _ := pickFirstNonRejected(candidates, rejections, datasourceID, qualifiedPath)
			if chosen != "" {
				semanticName = chosen
			}
		}
	}

	// Cache the deterministic derivation output for this column. This is
	// best-effort — failure is logged but does not fail the generation.
	// The cache is read on wizard reopen and by preview-semantic-terms.
	if item.Name == "" || strings.Contains(item.Name, "/") {
		s.upsertSuggestion(ctx, tenantID, datasourceID, qualifiedPath, columnNodeName, names)
	}

	if semanticName == "" {
		return nil, fmt.Errorf("could not derive a semantic term name")
	}
	// A derived name is mapped onto an existing term with the same meaning (TenantIdentifier -> TenantId) so one
	// concept is not split over twins. A name the user typed is kept verbatim.
	if item.Name == "" || strings.Contains(item.Name, "/") {
		if existing := s.reuseExistingTerm(ctx, tenantID, semanticName, cache); existing != semanticName {
			log.Printf("[GenerateSemanticTerms] reusing existing term %q for derived name %q", existing, semanticName)
			semanticName = existing
			names.BusinessName = titleCase(pascalCaseToWords(existing))
		}
	}
	businessName := names.BusinessName
	if businessName == "" {
		businessName = semanticName
	}

	semanticNodeTypeID, err := s.resolveOrCreateNodeType(tenantID, "semantic_term")
	if err != nil {
		log.Printf("[GenerateSemanticTerms] failed to resolve semantic_term node type: %v", err)
		return nil, fmt.Errorf("failed to resolve semantic term type: %w", err)
	}

	var semanticTermID string
	semanticReused := false
	if cache != nil {
		if cachedID, ok := cache.load(termCacheKey{nodeTypeID: semanticNodeTypeID, name: semanticName}); ok {
			semanticTermID = cachedID
			semanticReused = true
		}
	}
	if semanticTermID == "" {
		var err error
		semanticTermID, semanticReused, err = s.findOrCreateTermNode(ctx, tenantID, datasourceID, semanticNodeTypeID, "semantic_term", semanticName, "")
		if err != nil {
			log.Printf("[GenerateSemanticTerms] failed to resolve semantic term %q: %v", semanticName, err)
			return nil, fmt.Errorf("failed to resolve semantic term: %w", err)
		}
		if cache != nil {
			cache.store(termCacheKey{nodeTypeID: semanticNodeTypeID, name: semanticName}, semanticTermID)
			if !semanticReused {
				cache.noteNewTerm(semanticName)
			}
		}
	}

	mapsToEdgeTypeID, err := s.resolveOrCreateEdgeType(tenantID, "MAPS_TO")
	if err != nil {
		log.Printf("[GenerateSemanticTerms] failed to resolve MAPS_TO edge type: %v", err)
		return nil, fmt.Errorf("failed to resolve MAPS_TO edge type: %w", err)
	}

	linked := 0
	for _, colID := range item.ColumnIDs {
		if colID == "" {
			continue
		}
		if err := s.ensureEdge(tenantID, datasourceID, semanticTermID, colID, mapsToEdgeTypeID); err != nil {
			log.Printf("[GenerateSemanticTerms] failed to link column %s to term %s: %v", colID, semanticTermID, err)
			continue
		}
		linked++
	}

	if names.BaseGenericTerm != "" && names.BaseGenericTerm != semanticName {
		var baseTermID string
		err := s.db.QueryRow(`
			SELECT id FROM catalog_node
			WHERE tenant_id = $1 AND node_name = $2
			  AND node_type_id = $3
			LIMIT 1
		`, tenantID, names.BaseGenericTerm, semanticNodeTypeID).Scan(&baseTermID)
		if err == nil && baseTermID != "" && baseTermID != semanticTermID {
			specializationEdgeTypeID, edgeErr := s.resolveOrCreateEdgeType(tenantID, "IS_SPECIALIZATION_OF")
			if edgeErr == nil {
				if linkErr := s.ensureEdge(tenantID, datasourceID, semanticTermID, baseTermID, specializationEdgeTypeID); linkErr != nil {
					log.Printf("[GenerateSemanticTerms] failed to create IS_SPECIALIZATION_OF edge from %s to %s: %v", semanticName, names.BaseGenericTerm, linkErr)
				} else {
					log.Printf("[GenerateSemanticTerms] created IS_SPECIALIZATION_OF edge: %s → %s", semanticName, names.BaseGenericTerm)
				}
			}
		}
	}

	businessNodeTypeID, err := s.resolveOrCreateNodeType(tenantID, "business_term")
	if err != nil {
		log.Printf("[GenerateSemanticTerms] failed to resolve business_term node type: %v", err)
		return nil, fmt.Errorf("failed to resolve business term type: %w", err)
	}

	definitionSource := ""
	var definition string
	if s.abbrevSvc != nil {
		svcCtx := context.WithValue(ctx, "tenant_id", tenantID)
		if def, defErr := s.abbrevSvc.GenerateStandardDefinition(svcCtx, businessName, columnNodeName); defErr != nil {
			log.Printf("[GenerateSemanticTerms] definition generation failed for %q: %v", businessName, defErr)
		} else {
			definition = def.Definition
			definitionSource = def.Source
		}
	}
	var businessTermID string
	businessReused := false
	if cache != nil {
		if cachedID, ok := cache.load(termCacheKey{nodeTypeID: businessNodeTypeID, name: businessName}); ok {
			businessTermID = cachedID
			businessReused = true
		}
	}
	if businessTermID == "" {
		var err error
		businessTermID, businessReused, err = s.findOrCreateTermNode(ctx, tenantID, datasourceID, businessNodeTypeID, "business_term", businessName, definition)
		if err != nil {
			log.Printf("[GenerateSemanticTerms] failed to resolve business term %q: %v", businessName, err)
			return nil, fmt.Errorf("failed to resolve business term: %w", err)
		}
		if cache != nil {
			cache.store(termCacheKey{nodeTypeID: businessNodeTypeID, name: businessName}, businessTermID)
		}
	}

	hasSemanticEdgeTypeID, err := s.resolveOrCreateEdgeType(tenantID, "has_semantic_context")
	if err != nil {
		log.Printf("[GenerateSemanticTerms] failed to resolve has_semantic_context edge type: %v", err)
		return nil, fmt.Errorf("failed to resolve has_semantic_context edge type: %w", err)
	}
	if err := s.ensureEdge(tenantID, datasourceID, businessTermID, semanticTermID, hasSemanticEdgeTypeID); err != nil {
		log.Printf("[GenerateSemanticTerms] failed to link business term %s to semantic term %s: %v", businessTermID, semanticTermID, err)
	}

	return &generateTermResult{
		ID:                 semanticTermID,
		Name:               semanticName,
		ReusedExisting:     semanticReused,
		BusinessTermID:     businessTermID,
		BusinessTermName:   businessName,
		BusinessTermReused: businessReused,
		DefinitionSource:   definitionSource,
		ColumnsLinked:      linked,
		ColumnsTotal:       len(item.ColumnIDs),
	}, nil
}

func (s *GlossaryService) generateTerms(ctx context.Context, tenantID, datasourceID string, items []generateTermItem) generateTermsResponse {
	if len(items) <= syncThreshold {
		return s.generateTermsSync(ctx, tenantID, datasourceID, items)
	}
	jobID, err := s.startBulkJob(ctx, tenantID, datasourceID, items)
	if err != nil {
		return generateTermsResponse{
			Success: false,
			Results: []generateTermResult{{Name: "", Error: "failed to start bulk job: " + err.Error()}},
		}
	}
	return generateTermsResponse{JobID: jobID}
}

func (s *GlossaryService) generateTermsSync(ctx context.Context, tenantID, datasourceID string, items []generateTermItem) generateTermsResponse {
	rejections, _ := s.loadRejections(ctx, tenantID)

	results := make([]generateTermResult, 0, len(items))
	createdTerms := 0
	reusedTerms := 0
	columnsLinked := 0

	for _, item := range items {
		res, err := s.generateSingleTerm(ctx, tenantID, datasourceID, item, nil, rejections)
		if err != nil {
			results = append(results, generateTermResult{
				Name:  item.Name,
				Error: err.Error(),
			})
			continue
		}
		if res.ReusedExisting {
			reusedTerms++
		} else {
			createdTerms++
		}
		columnsLinked += res.ColumnsLinked
		results = append(results, *res)
	}

	return generateTermsResponse{
		Success:        true,
		TotalRequested: len(items),
		CreatedTerms:   createdTerms,
		ReusedTerms:    reusedTerms,
		ColumnsLinked:  columnsLinked,
		Results:        results,
	}
}

func (s *GlossaryService) startBulkJob(ctx context.Context, tenantID, datasourceID string, items []generateTermItem) (string, error) {
	job := &Job{
		ID:           "",
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		Status:       JobStatusRunning,
		Total:        len(items),
		Done:         0,
		Failed:       0,
		Results:      make([]generateTermResult, len(items)),
		Errors:       nil,
		StartedAt:    time.Now(),
		FinishedAt:   time.Now(),
	}
	if err := s.jobStore.Create(job); err != nil {
		return "", fmt.Errorf("creating job: %w", err)
	}

	go func() {
		s.runBulk(s.appCtx, tenantID, datasourceID, items, job, s.jobStore)
	}()

	return job.ID, nil
}

func (s *GlossaryService) getJob(tenantID, jobID string) (Job, bool) {
	return s.jobStore.Get(tenantID, jobID)
}

const previewCap = 2000

type abbreviationSvcAdapter struct {
	real *services.AbbreviationService
}

func (a abbreviationSvcAdapter) GetAllAbbreviations(ctx context.Context) ([]map[string]string, error) {
	entries, err := a.real.GetAllAbbreviations(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]string, len(entries))
	for i, e := range entries {
		result[i] = map[string]string{
			"abbreviation": e.Abbreviation,
			"full_word":    e.FullWord,
		}
	}
	return result, nil
}

func (s *GlossaryService) PreviewSemanticTerms(ctx context.Context, tenantID string, columnIDs []string) ([]PreviewResult, error) {
	if len(columnIDs) == 0 {
		return nil, fmt.Errorf("column_ids is required")
	}
	if len(columnIDs) > previewCap {
		return nil, fmt.Errorf("column_ids capped at %d (got %d)", previewCap, len(columnIDs))
	}

	query := `
		SELECT id, node_name, COALESCE(qualified_path, node_name), COALESCE(tenant_datasource_id, '00000000-0000-0000-0000-000000000000'::uuid)
		FROM catalog_node
		WHERE id = ANY($1) AND tenant_id = $2
	`
	rows, err := s.db.QueryContext(ctx, query, columnIDs, tenantID)
	if err != nil {
		return nil, fmt.Errorf("querying catalog nodes: %w", err)
	}
	defer rows.Close()

	nodeMap := make(map[string]struct{ nodeName, qualifiedPath, datasourceID string }, len(columnIDs))
	for rows.Next() {
		var id, nodeName, qualifiedPath, datasourceID string
		if err := rows.Scan(&id, &nodeName, &qualifiedPath, &datasourceID); err != nil {
			return nil, fmt.Errorf("scanning catalog node: %w", err)
		}
		nodeMap[id] = struct{ nodeName, qualifiedPath, datasourceID string }{nodeName, qualifiedPath, datasourceID}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating catalog nodes: %w", err)
	}

	// Load rejection set once per request — keyed by (datasourceID, qualifiedPath, rejectedName).
	rejections, err := s.loadRejections(ctx, tenantID)
	if err != nil {
		log.Printf("[PreviewSemanticTerms] failed to load rejections: %v", err)
		rejections = rejectionSet{}
	}

	// Load the abbreviation map once per request, not once per column (see buildAbbreviationMap).
	var abbrevLookup abbreviationLookup
	if s.abbrevSvc != nil {
		abbrevLookup = abbreviationSvcAdapter{real: s.abbrevSvc}
	}
	abbrMap := buildAbbreviationMap(ctx, abbrevLookup, tenantID)
	// Existing terms by meaning, loaded once: a proposed name that means the same as an existing term is shown as
	// that term, so the wizard does not offer TenantIdentifier when TenantId already exists.
	existingTerms := s.loadTermIndex(ctx, tenantID, abbrMap)

	results := make([]PreviewResult, 0, len(columnIDs))
	for _, colID := range columnIDs {
		node, ok := nodeMap[colID]
		if !ok {
			continue
		}

		var tableSchemaContext string
		if node.qualifiedPath != "" {
			tableSchemaContext = buildTableSchemaContext(node.qualifiedPath)
		}

		candidates := deriveTermNamesPreviewCandidatesWithMap(abbrMap, node.nodeName, tableSchemaContext)

		semanticName, source := pickFirstNonRejected(candidates, rejections, node.datasourceID, node.qualifiedPath)

		// Derive business name from the primary candidate's businessName (derived from first candidate).
		var businessName string
		if len(candidates) > 0 {
			businessName = titleCase(pascalCaseToWords(candidates[0].Name))
		}
		if existing, ok := existingTerms.reuse(semanticName, abbrMap); ok && existing != semanticName {
			semanticName, source = existing, "existing_term"
			businessName = titleCase(pascalCaseToWords(existing))
		}

		results = append(results, PreviewResult{
			ColumnID:     colID,
			SemanticName: semanticName,
			BusinessName: businessName,
			Source:       source,
		})
	}

	return results, nil
}

func buildTableSchemaContext(qualifiedPath string) string {
	var schemaName, tableName string
	if strings.HasPrefix(qualifiedPath, "/") {
		parts := strings.Split(strings.TrimPrefix(qualifiedPath, "/"), "/")
		if len(parts) >= 3 {
			schemaName = parts[len(parts)-3]
			tableName = parts[len(parts)-2]
		}
	} else {
		parts := strings.Split(qualifiedPath, ".")
		if len(parts) >= 3 {
			schemaName = parts[len(parts)-3]
			tableName = parts[len(parts)-2]
		}
	}
	if tableName != "" && schemaName != "" {
		return fmt.Sprintf("table %q in schema %q", tableName, schemaName)
	}
	return ""
}

// resolveTokensWithAbbreviations resolves tokens using the abbreviation table only
// (no LLM). Used by generateSingleTerm to build the candidate list for rejection
// filtering without calling the LLM a second time.
func resolveTokensWithAbbreviations(ctx context.Context, abbrevSvc *services.AbbreviationService, tenantID string, tokens []string) []string {
	if abbrevSvc == nil {
		return tokens
	}
	svcCtx := context.WithValue(ctx, "tenant_id", tenantID)
	abbrevs, err := abbrevSvc.GetAllAbbreviations(svcCtx)
	if err != nil || len(abbrevs) == 0 {
		return tokens
	}
	abbrMap := make(map[string]string, len(abbrevs))
	for _, a := range abbrevs {
		abbrMap[strings.ToUpper(a.Abbreviation)] = a.FullWord
	}
	resolved := make([]string, len(tokens))
	for i, tok := range tokens {
		upper := strings.ToUpper(tok)
		if full, ok := abbrMap[upper]; ok {
			resolved[i] = full
		} else {
			resolved[i] = tok
		}
	}
	return resolved
}

// loadRejections loads all rejection records for a tenant and returns them as a
// rejectionSet keyed by datasourceID + "\x00" + qualifiedPath + "\x00" + rejectedName.
// The set is built once per PreviewSemanticTerms or generateTermsSync call.
func (s *GlossaryService) loadRejections(ctx context.Context, tenantID string) (rejectionSet, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT COALESCE(datasource_id, '00000000-0000-0000-0000-000000000000'::uuid)::text,
		       qualified_path, rejected_name, COALESCE(preferred_name, '')
		FROM sml.semantic_term_rejections
		WHERE tenant_id = $1
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("loading rejections: %w", err)
	}
	defer rows.Close()

	set := make(rejectionSet)
	for rows.Next() {
		var dsID, qPath, name, preferred string
		if err := rows.Scan(&dsID, &qPath, &name, &preferred); err != nil {
			return nil, fmt.Errorf("scanning rejection row: %w", err)
		}
		set[makeRejectionKey(dsID, qPath, name)] = preferred
	}
	return set, rows.Err()
}

// RecordRejection persists a rejection with an optional preferred_name override.
// Uses INSERT ... ON CONFLICT DO UPDATE so re-runs refresh the preferred_name.
func (s *GlossaryService) RecordRejection(ctx context.Context, tenantID, datasourceID, qualifiedPath, rejectedName, preferredName string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sml.semantic_term_rejections (tenant_id, datasource_id, qualified_path, rejected_name, preferred_name)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''))
		ON CONFLICT (tenant_id, datasource_id, qualified_path, rejected_name) DO UPDATE
		   SET preferred_name = EXCLUDED.preferred_name
	`, tenantID, datasourceID, qualifiedPath, rejectedName, preferredName)
	return err
}

// UnrecordRejection removes a rejection. Idempotent: returns 204 whether the row
// existed or not.
func (s *GlossaryService) UnrecordRejection(ctx context.Context, tenantID, datasourceID, qualifiedPath, rejectedName string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM sml.semantic_term_rejections
		WHERE tenant_id = $1
		  AND datasource_id = $2
		  AND qualified_path = $3
		  AND rejected_name = $4
	`, tenantID, datasourceID, qualifiedPath, rejectedName)
	return err
}

// cachedSuggestion is a row from sml.glossary_term_suggestions.
type cachedSuggestion struct {
	SemanticName    string
	BusinessName    string
	BaseGenericTerm string
	DerivedVia      string
}

// loadSuggestion reads the cached suggestion for a column, if one exists.
// Returns nil on cache miss (column not yet processed).
func (s *GlossaryService) loadSuggestion(ctx context.Context, tenantID, qualifiedPath string) *cachedSuggestion {
	var cs cachedSuggestion
	err := s.db.QueryRowContext(ctx, `
		SELECT semantic_name, business_name, COALESCE(base_generic_term, ''), derived_via
		FROM sml.glossary_term_suggestions
		WHERE tenant_id = $1 AND qualified_path = $2
	`, tenantID, qualifiedPath).Scan(&cs.SemanticName, &cs.BusinessName, &cs.BaseGenericTerm, &cs.DerivedVia)
	if err != nil {
		return nil
	}
	return &cs
}

// upsertSuggestion persists the deterministic derivation output for a column.
// Best-effort: failure is log-warned but does not fail the generation.
func (s *GlossaryService) upsertSuggestion(ctx context.Context, tenantID, datasourceID, qualifiedPath, columnName string, derived derivedTermNames) {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sml.glossary_term_suggestions
			(tenant_id, datasource_id, qualified_path, column_node_name, semantic_name, business_name, base_generic_term, derived_via, computed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
		ON CONFLICT (tenant_id, qualified_path) DO UPDATE
		   SET semantic_name = EXCLUDED.semantic_name,
		       business_name = EXCLUDED.business_name,
		       base_generic_term = EXCLUDED.base_generic_term,
		       derived_via = EXCLUDED.derived_via,
		       computed_at = now()
	`, tenantID, datasourceID, qualifiedPath, columnName, derived.SemanticName, derived.BusinessName, derived.BaseGenericTerm, derived.source)
	if err != nil {
		log.Printf("[upsertSuggestion] cache write failed for %s: %v", qualifiedPath, err)
	}
}

// cachedDefinition is a row from sml.glossary_term_definition_cache.
type cachedDefinition struct {
	Definition      string
	DefinitionSource string
}

// loadDefinition reads a cached definition by cache key. Returns nil on miss.
func (s *GlossaryService) loadDefinition(ctx context.Context, cacheKey string) *cachedDefinition {
	var cd cachedDefinition
	err := s.db.QueryRowContext(ctx, `
		SELECT definition, definition_source
		FROM sml.glossary_term_definition_cache
		WHERE cache_key = $1
	`, cacheKey).Scan(&cd.Definition, &cd.DefinitionSource)
	if err != nil {
		return nil
	}
	return &cd
}

// upsertDefinition persists a definition to the cache. Best-effort: failure is
// log-warned but does not fail the generation. ON CONFLICT DO UPDATE refreshes
// the definition if the cache key already exists.
func (s *GlossaryService) upsertDefinition(ctx context.Context, cacheKey, semanticName, definition, source string) {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sml.glossary_term_definition_cache
			(cache_key, semantic_name, definition, definition_source, created_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (cache_key) DO UPDATE
		   SET definition = EXCLUDED.definition,
		       definition_source = EXCLUDED.definition_source,
		       created_at = now()
	`, cacheKey, semanticName, definition, source)
	if err != nil {
		log.Printf("[upsertDefinition] cache write failed for %s: %v", semanticName, err)
	}
}

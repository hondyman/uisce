package api

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

type termCacheKey struct {
	nodeTypeID string
	name      string
}

type termCache struct {
	mu  sync.Mutex
	mem map[termCacheKey]string

	// existing terms by meaning, loaded once per batch (see term_reuse.go)
	index *termIndex
	abbr  map[string]string
}

func newTermCache() *termCache {
	return &termCache{mem: make(map[termCacheKey]string)}
}

func (c *termCache) load(key termCacheKey) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	id, ok := c.mem[key]
	return id, ok
}

func (c *termCache) store(key termCacheKey, id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mem[key] = id
}

// preGroupedItem is a group of items that share the same derived name and
// context sensitivity, collapsed into a single generateTermItem for dispatch.
// The representative carries the merged column_ids; all other fields come from
// the first item in the group.
type preGroupedItem struct {
	item      generateTermItem
	origIndex int // index of the representative in the original items slice
}

// runBulk processes a bulk glossary generation job with pre-grouped dispatch.
// Items are grouped by their deterministic derivation key BEFORE hitting the
// worker pool. This eliminates the TOCTOU race (two workers sharing CustomerId
// both miss and both call Gemini) and collapses per-item Gemini calls to
// one-call-per-derived-name.
//
// Grouping key: sha256(tenantID || semanticName || contextPart)
// - contextPart = tableSchemaContext when ContextSensitive=true (address-line,
//   bare-generic, abbreviation expansion — any rule that consulted table context)
// - contextPart = "" when ContextSensitive=false (pure token assembly)
//
// This is race-safe: by the time the pool dispatches, every member of a group
// provably has the same LLM inputs (same tokens, same abbreviations, same or
// no context), so sharing one call is definitionally correct.
func (s *GlossaryService) runBulk(ctx context.Context, tenantID, datasourceID string, items []generateTermItem, job *Job, store JobStore) {
	cache := newTermCache()
	rejections, _ := s.loadRejections(ctx, tenantID)

	// Load abbreviations once for the whole batch — needed for group key computation.
	// This is a pure read-only lookup; no LLM calls.
	var abbrMap map[string]string
	if s.abbrevSvc != nil {
		svcCtx := context.WithValue(ctx, "tenant_id", tenantID)
		abbrevs, err := s.abbrevSvc.GetAllAbbreviations(svcCtx)
		if err != nil {
			log.Printf("[runBulk] abbreviation lookup failed, grouping will use raw names: %v", err)
		} else {
			abbrMap = make(map[string]string, len(abbrevs))
			for _, a := range abbrevs {
				abbrMap[strings.ToUpper(a.Abbreviation)] = a.FullWord
			}
		}
	}

	// Phase 1: Pre-group items by their deterministic derivation key.
	// Items with the same key share identical LLM inputs and can safely
	// share one LLM call. User-named items (item.Name != "") are never
	// grouped — rejections must only ever record derived candidate names,
	// not user-typed values.
	groups := make(map[string]*preGroupedItem)
	groupOrder := make([]string, 0) // preserve insertion order for deterministic results

	for i, item := range items {
		// User-named items get their own singleton group — no grouping.
		if item.Name != "" && !strings.Contains(item.Name, "/") {
			key := fmt.Sprintf("user_%d", i)
			groups[key] = &preGroupedItem{item: item, origIndex: i}
			groupOrder = append(groupOrder, key)
			continue
		}

		// Resolve column metadata for group key computation.
		// We need the raw column name and table context.
		var colName, qPath string
		if len(item.ColumnIDs) > 0 {
			_ = s.db.QueryRow(`
				SELECT node_name, COALESCE(qualified_path, '')
				FROM catalog_node
				WHERE id = $1 AND tenant_id = $2
			`, item.ColumnIDs[0], tenantID).Scan(&colName, &qPath)
		}

		// Build table schema context from qualified path (same logic as generateSingleTerm).
		tableSchemaContext := ""
		if qPath != "" {
			parts := strings.Split(qPath, "/")
			if len(parts) >= 2 {
				tableSchemaContext = fmt.Sprintf("table %q in schema %q", parts[len(parts)-2], parts[0])
			}
		}

		gk := computeGroupKeyForItem(tenantID, colName, tableSchemaContext, abbrMap)
		if gk == "" {
			gk = fmt.Sprintf("fallback_%d", i)
		}

		if existing, ok := groups[gk]; ok {
			// Merge column_ids into the existing group's representative.
			existing.item.ColumnIDs = append(existing.item.ColumnIDs, item.ColumnIDs...)
		} else {
			groups[gk] = &preGroupedItem{item: item, origIndex: i}
			groupOrder = append(groupOrder, gk)
		}
	}

	// Phase 2: Dispatch one worker call per group. Each group gets a merged
	// generateTermItem with all column_ids. generateSingleTerm already handles
	// multiple column_ids (it loops over item.ColumnIDs to create MAPS_TO edges).
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(8)

	prog := &Progress{}
	prog.SetTotal(len(items))

	results := make([]generateTermResult, len(items))

	for _, gk := range groupOrder {
		group := groups[gk]
		item := group.item
		origIdx := group.origIndex
		gk := gk // capture for closure
		_ = gk

		g.Go(func() error {
			res, err := s.generateSingleTerm(ctx, tenantID, datasourceID, item, cache, rejections)
			if err != nil {
				store.Update(job.ID, func(j *Job) {
					if len(j.Errors) < maxErrorsPerJob {
						j.Errors = append(j.Errors, err.Error())
					}
					j.Failed++
				})
				prog.IncrementFailed()
				return nil
			}

			results[origIdx] = *res
			store.Update(job.ID, func(j *Job) {
				j.Done += len(item.ColumnIDs)
			})
			prog.IncrementN(len(item.ColumnIDs))
			return nil
		})
	}

	_ = g.Wait()

	store.Update(job.ID, func(j *Job) {
		j.Results = results
		if j.Status == JobStatusRunning {
			if j.Failed == j.Total {
				j.Status = JobStatusFailed
			} else {
				j.Status = JobStatusCompleted
			}
			j.FinishedAt = time.Now()
		}
	})
}

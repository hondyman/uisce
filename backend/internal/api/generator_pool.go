package api

import (
	"context"
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

func (s *GlossaryService) runBulk(ctx context.Context, tenantID, datasourceID string, items []generateTermItem, job *Job, store JobStore) {
	cache := newTermCache()
	rejections, _ := s.loadRejections(ctx, tenantID)

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(8)

	prog := &Progress{}
	prog.SetTotal(len(items))

	results := make([]generateTermResult, len(items))

	for i, item := range items {
		i, item := i, item
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

			results[i] = *res
			store.Update(job.ID, func(j *Job) {
				j.Done++
			})
			prog.Increment()
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

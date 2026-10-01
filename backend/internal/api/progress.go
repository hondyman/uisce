package api

import (
	"crypto/rand"
	"fmt"
	"sort"
	"sync"
	"time"
)

const (
	JobStatusRunning    = "running"
	JobStatusCompleted = "completed"
	JobStatusFailed    = "failed"
	jobTTL             = 1 * time.Hour
	maxErrorsPerJob    = 100
	maxCompletedJobs   = 100
)

type Job struct {
	mu           sync.Mutex
	ID           string               `json:"id"`
	TenantID     string               `json:"tenant_id"`
	DatasourceID string               `json:"datasource_id"`
	Status       string               `json:"status"`
	Total        int                  `json:"total"`
	Done         int                  `json:"done"`
	Failed       int                  `json:"failed"`
	Results      []generateTermResult `json:"results"`
	Errors       []string             `json:"errors"`
	StartedAt    time.Time            `json:"started_at"`
	FinishedAt   time.Time            `json:"finished_at"`
}

func (j *Job) snapshot() Job {
	j.mu.Lock()
	defer j.mu.Unlock()
	return Job{
		ID:           j.ID,
		TenantID:     j.TenantID,
		DatasourceID: j.DatasourceID,
		Status:       j.Status,
		Total:        j.Total,
		Done:         j.Done,
		Failed:       j.Failed,
		Results:      append([]generateTermResult(nil), j.Results...),
		Errors:       append([]string(nil), j.Errors...),
		StartedAt:    j.StartedAt,
		FinishedAt:   j.FinishedAt,
	}
}

type Progress struct {
	mu    sync.Mutex
	done  int
	total int
	failed int
}

func (p *Progress) Increment() {
	p.mu.Lock()
	p.done++
	p.mu.Unlock()
}

// IncrementN advances the done counter by n (the number of items in a group).
func (p *Progress) IncrementN(n int) {
	p.mu.Lock()
	p.done += n
	p.mu.Unlock()
}

func (p *Progress) IncrementFailed() {
	p.mu.Lock()
	p.done++
	p.failed++
	p.mu.Unlock()
}

func (p *Progress) Snapshot() (done, total, failed int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.done, p.total, p.failed
}

func (p *Progress) SetTotal(n int) {
	p.mu.Lock()
	p.total = n
	p.mu.Unlock()
}

type JobStore interface {
	Create(job *Job) error
	Get(tenantID, jobID string) (Job, bool)
	Update(jobID string, fn func(*Job))
}

type InMemoryJobStore struct {
	mu    sync.RWMutex
	jobs  map[string]*Job
	stopC chan struct{}
}

func NewInMemoryJobStore() *InMemoryJobStore {
	store := &InMemoryJobStore{
		jobs:  make(map[string]*Job),
		stopC: make(chan struct{}),
	}
	go store.Janitor()
	return store
}

func sortByFinished(ja []jobAge) {
	sort.Slice(ja, func(i, j int) bool { return ja[i].age < ja[j].age })
}

type jobAge struct {
	id  string
	age time.Duration
}

// evict is the actual eviction logic. Janitor calls it on each tick; tests
// call it directly to avoid waiting for the ticker.
func (s *InMemoryJobStore) evict() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	var toDelete []string
	var completed []jobAge
	for id, job := range s.jobs {
		job.mu.Lock()
		age := now.Sub(job.FinishedAt)
		stillRunning := job.Status == JobStatusRunning
		job.mu.Unlock()
		if !stillRunning && age > jobTTL {
			toDelete = append(toDelete, id)
		} else if !stillRunning {
			completed = append(completed, jobAge{id: id, age: age})
		}
	}
	if len(completed) > maxCompletedJobs {
		sortByFinished(completed)
		excess := len(completed) - maxCompletedJobs
		for i := 0; i < excess; i++ {
			toDelete = append(toDelete, completed[i].id)
		}
	}
	for _, id := range toDelete {
		delete(s.jobs, id)
	}
}

// Janitor runs the eviction sweep on a ticker. Exposed for testing.
func (s *InMemoryJobStore) Janitor() {
	ticker := time.NewTicker(jobTTL / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.evict()
		case <-s.stopC:
			return
		}
	}
}

func (s *InMemoryJobStore) Stop() {
	close(s.stopC)
}

func (s *InMemoryJobStore) Create(job *Job) error {
	if job.ID == "" {
		id, err := newJobID()
		if err != nil {
			return fmt.Errorf("generating job ID: %w", err)
		}
		job.ID = id
	}
	job.mu.Lock()
	defer job.mu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
	return nil
}

func (s *InMemoryJobStore) Get(tenantID, jobID string) (Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return Job{}, false
	}
	if job.TenantID != tenantID {
		return Job{}, false
	}
	return job.snapshot(), true
}

func (s *InMemoryJobStore) Update(jobID string, fn func(*Job)) {
	s.mu.RLock()
	job, ok := s.jobs[jobID]
	s.mu.RUnlock()
	if !ok {
		return
	}
	job.mu.Lock()
	fn(job)
	job.mu.Unlock()
}

func newJobID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

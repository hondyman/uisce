package api

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewJobID(t *testing.T) {
	id1, err := newJobID()
	if err != nil {
		t.Fatalf("newJobID() error = %v", err)
	}
	if id1 == "" {
		t.Fatal("newJobID() returned empty string")
	}
	if len(id1) != 36 {
		t.Errorf("newJobID() len = %d; want 36 (UUID format)", len(id1))
	}
	id2, err := newJobID()
	if err != nil {
		t.Fatalf("newJobID() error = %v", err)
	}
	if id1 == id2 {
		t.Error("newJobID() generated duplicate IDs")
	}
	if !strings.Contains(id1, "-") {
		t.Errorf("newJobID() = %q; want UUID format with hyphens", id1)
	}
}

func TestProgress_Increment(t *testing.T) {
	p := &Progress{}
	p.SetTotal(10)
	p.Increment()
	done, total, failed := p.Snapshot()
	if done != 1 {
		t.Errorf("Increment: done = %d; want 1", done)
	}
	if total != 10 {
		t.Errorf("Increment: total = %d; want 10", total)
	}
	if failed != 0 {
		t.Errorf("Increment: failed = %d; want 0", failed)
	}
}

func TestProgress_IncrementFailed(t *testing.T) {
	p := &Progress{}
	p.SetTotal(10)
	p.IncrementFailed()
	done, _, failed := p.Snapshot()
	if done != 1 {
		t.Errorf("IncrementFailed: done = %d; want 1", done)
	}
	if failed != 1 {
		t.Errorf("IncrementFailed: failed = %d; want 1", failed)
	}
}

func TestInMemoryJobStore_Create(t *testing.T) {
	store := NewInMemoryJobStore()
	defer store.Stop()

	job := &Job{
		TenantID:     "tenant-1",
		DatasourceID: "ds-1",
		Status:       JobStatusRunning,
		Total:        5,
		StartedAt:    time.Now(),
		FinishedAt:   time.Now(),
	}
	err := store.Create(job)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if job.ID == "" {
		t.Error("Create() left job.ID empty")
	}
}

func TestInMemoryJobStore_Get(t *testing.T) {
	store := NewInMemoryJobStore()
	defer store.Stop()

	job := &Job{
		TenantID:     "tenant-1",
		DatasourceID: "ds-1",
		Status:       JobStatusRunning,
		Total:        5,
		StartedAt:    time.Now(),
		FinishedAt:   time.Now(),
	}
	store.Create(job)

	got, ok := store.Get("tenant-1", job.ID)
	if !ok {
		t.Fatal("Get() returned ok=false for existing job")
	}
	if got.ID != job.ID {
		t.Errorf("Get() ID = %q; want %q", got.ID, job.ID)
	}

	_, ok = store.Get("wrong-tenant", job.ID)
	if ok {
		t.Error("Get() returned ok=true for wrong tenant")
	}

	_, ok = store.Get("tenant-1", "nonexistent-id")
	if ok {
		t.Error("Get() returned ok=true for nonexistent job ID")
	}
}

func TestInMemoryJobStore_Update(t *testing.T) {
	store := NewInMemoryJobStore()
	defer store.Stop()

	job := &Job{
		TenantID:     "tenant-1",
		DatasourceID: "ds-1",
		Status:       JobStatusRunning,
		Total:        5,
		Done:         0,
		StartedAt:    time.Now(),
		FinishedAt:   time.Now(),
	}
	store.Create(job)

	store.Update(job.ID, func(j *Job) {
		j.Done = 3
	})

	got, _ := store.Get("tenant-1", job.ID)
	if got.Done != 3 {
		t.Errorf("Update: Done = %d; want 3", got.Done)
	}
}

func TestInMemoryJobStore_Update_FinishedAt(t *testing.T) {
	store := NewInMemoryJobStore()
	defer store.Stop()

	job := &Job{
		TenantID:     "tenant-1",
		DatasourceID: "ds-1",
		Status:       JobStatusRunning,
		Total:        5,
		Done:         0,
		StartedAt:    time.Now(),
		FinishedAt:   time.Now(),
	}
	store.Create(job)

	store.Update(job.ID, func(j *Job) {
		j.Done = 3
	})

	got, _ := store.Get("tenant-1", job.ID)
	if got.Done != 3 {
		t.Errorf("Update: Done = %d; want 3", got.Done)
	}
	if got.Status != JobStatusRunning {
		t.Errorf("Update: Status = %q; want %q (auto-complete moved to runBulk)", got.Status, JobStatusRunning)
	}
}

func TestInMemoryJobStore_GetUpdateRace(t *testing.T) {
	store := NewInMemoryJobStore()
	defer store.Stop()

	job := &Job{
		TenantID:     "tenant-1",
		DatasourceID: "ds-1",
		Status:       JobStatusRunning,
		Total:        100,
		Done:         0,
		StartedAt:    time.Now(),
		FinishedAt:   time.Now(),
	}
	store.Create(job)

	var wg sync.WaitGroup
	const goroutines = 8

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				store.Update(job.ID, func(j *Job) {
					j.Done++
				})
			}
		}()
	}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				store.Get("tenant-1", job.ID)
			}
		}()
	}

	wg.Wait()
}

package api

import (
	"testing"
	"time"
)

func TestDefaultProfilerGetStatusAndResults(t *testing.T) {
	srv := &Server{}
	job := &ProfileJob{
		ID:        "job-test-1",
		Status:    "pending",
		CreatedAt: time.Now(),
		Results:   []string{"a", "b"},
	}
	srv.ProfileJobs.Store(job.ID, job)

	svc := newDefaultProfilerService(srv)

	gotIface, err := svc.GetProfileStatus(job.ID)
	if err != nil {
		t.Fatalf("unexpected error from GetProfileStatus: %v", err)
	}
	got, ok := gotIface.(*ProfileJob)
	if !ok {
		t.Fatalf("expected *ProfileJob from GetProfileStatus, got %T", gotIface)
	}
	if got.ID != job.ID {
		t.Fatalf("expected job id %s, got %s", job.ID, got.ID)
	}

	res, err := svc.GetProfileResults(job.ID)
	if err != nil {
		t.Fatalf("unexpected error from GetProfileResults: %v", err)
	}
	if res == nil {
		t.Fatalf("expected non-nil results")
	}
}

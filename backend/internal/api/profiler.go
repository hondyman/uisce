package api

import (
	"context"
	"fmt"
)

// ProfilerService defines the operations the API expects from the profiler subsystem.
type ProfilerService interface {
	StartProfile(ctx context.Context, job interface{}) error
	GetProfileStatus(jobID string) (interface{}, error)
	GetProfileResults(jobID string) (interface{}, error)
}

// defaultProfilerService is a thin wrapper around Server methods to keep compatibility.
type defaultProfilerService struct {
	srv *Server
}

func (d *defaultProfilerService) StartProfile(ctx context.Context, job interface{}) error {
	// Expecting *ProfileJob; if not, return an error
	pj, ok := job.(*ProfileJob)
	if !ok {
		return fmt.Errorf("invalid job type")
	}
	d.srv.ProfileJobs.Store(pj.ID, pj)
	go d.srv.runProfile(pj.ID)
	return nil
}

func (d *defaultProfilerService) GetProfileStatus(jobID string) (interface{}, error) {
	jobInterface, exists := d.srv.ProfileJobs.Load(jobID)
	if !exists {
		return nil, fmt.Errorf("job not found")
	}
	job := jobInterface.(*ProfileJob)
	return job, nil
}

func (d *defaultProfilerService) GetProfileResults(jobID string) (interface{}, error) {
	jobInterface, exists := d.srv.ProfileJobs.Load(jobID)
	if !exists {
		return nil, fmt.Errorf("job not found")
	}
	job := jobInterface.(*ProfileJob)
	return job.Results, nil
}

// helper to create a default ProfilerService bound to the Server
func newDefaultProfilerService(s *Server) ProfilerService {
	return &defaultProfilerService{srv: s}
}

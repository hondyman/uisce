package api

import (
	"database/sql"
	"testing"
)

// Compile-time interface checks
var _ ProfilerService = (*defaultProfilerService)(nil)
var _ SessionService = (*dbSessionService)(nil)

// Small runtime smoke tests to ensure construction functions return non-nil values
func TestDefaultServicesConstruct(t *testing.T) {
	srv := &Server{DB: &sql.DB{}}
	p := newDefaultProfilerService(srv)
	if p == nil {
		t.Fatal("newDefaultProfilerService returned nil")
	}

	s := NewDBSessionService(srv.DB)
	if s == nil {
		t.Fatal("NewDBSessionService returned nil")
	}
}

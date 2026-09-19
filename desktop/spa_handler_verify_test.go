//go:build verify

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSPAHandler_VerifyReportEndpoint(t *testing.T) {
	mockFS := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte("<!DOCTYPE html><html><head><title>App</title></head><body>Root</body></html>"),
		},
	}

	handler := NewSPAHandler(mockFS)

	reportJSON := `{"step":"test_step","windowId":"win_test","success":true}`
	reqReport := httptest.NewRequest(http.MethodPost, "/api/desk-verify/report", strings.NewReader(reportJSON))
	recReport := httptest.NewRecorder()
	handler.ServeHTTP(recReport, reqReport)

	if recReport.Code != http.StatusOK {
		t.Fatalf("expected 200 for verify report, got %d", recReport.Code)
	}

	select {
	case rep := <-GetVerifyReportsChannel():
		if rep.Step != "test_step" || rep.WindowID != "win_test" || !rep.Success {
			t.Fatalf("unexpected report received: %+v", rep)
		}
	default:
		t.Fatalf("expected report on GetVerifyReportsChannel, got none")
	}
}

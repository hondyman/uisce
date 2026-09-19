//go:build verify

package main

import (
	"encoding/json"
	"net/http"
)

// VerifyReport represents an incoming verification assertion from a running window.
type VerifyReport struct {
	Step     string `json:"step"`
	WindowID string `json:"windowId"`
	Details  string `json:"details,omitempty"`
	URL      string `json:"url,omitempty"`
	Token    string `json:"token,omitempty"`
	Payload  string `json:"payload,omitempty"`
	Success  bool   `json:"success"`
}

var globalVerifyReports = make(chan VerifyReport, 50)

func initVerifyHandler(h *SPAHandler) {
	h.customHandler = func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path == "/api/desk-verify/report" && r.Method == http.MethodPost {
			var rep VerifyReport
			if err := json.NewDecoder(r.Body).Decode(&rep); err == nil {
				select {
				case globalVerifyReports <- rep:
				default:
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return true
		}
		return false
	}
}

func GetVerifyReportsChannel() <-chan VerifyReport {
	return globalVerifyReports
}

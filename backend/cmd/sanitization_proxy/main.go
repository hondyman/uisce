package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/google/uuid"
)

type SanitizeReq struct {
	TenantID  string `json:"tenant_id"`
	ClientID  string `json:"client_id"`
	Text      string `json:"text"`
	RequestID string `json:"request_id"`
}

type SanitizeResp struct {
	SanitizedText string `json:"sanitized_text"`
	PIIMapID      string `json:"pii_map_id"`
}

var (
	// Basic regex patterns for PII detection
	reSSN   = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	reAcct  = regexp.MustCompile(`\b\d{3}-\d{3}-\d{3}\b`)
	// Naive name masking - in production use NER models
	reNames = regexp.MustCompile(`\b([A-Z][a-z]+ [A-Z][a-z]+)\b`)
)

func sanitize(s string) string {
	s = reSSN.ReplaceAllString(s, "[SSN]")
	s = reAcct.ReplaceAllString(s, "[ACCT]")
	s = reNames.ReplaceAllString(s, "[NAME]")
	return s
}

func handleSanitize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SanitizeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	// In a real implementation, we would store the mapping of masked tokens to original values
	// securely, keyed by PII Map ID, to allow for controlled desanitization.
	piiMapID := uuid.New().String()

	out := SanitizeResp{
		SanitizedText: sanitize(req.Text),
		PIIMapID:      piiMapID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/sanitize", handleSanitize)

	addr := os.Getenv("SAN_LISTEN_ADDR")
	if addr == "" {
		addr = ":8081"
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	log.Printf("sanitization proxy listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// Proxy AI endpoints to the configured AI service (local dev friendly)
func registerAIProxyRoutesV2(r chi.Router) {
	aiService := os.Getenv("AI_SERVICE_URL")
	if aiService == "" {
		aiService = "http://localhost:8088"
	}

	r.Route("/ai", func(rr chi.Router) {
		rr.Post("/generate-layout", proxyJSONV2(aiService))
		rr.Post("/field-recommendations", proxyJSONV2(aiService))
		rr.Post("/mark-adopted", proxyJSONV2(aiService))
		rr.Get("/layouts", proxyJSONV2(aiService))

		// MVP Stub for intelligent validation (avoids external dependency)
		rr.Post("/generate-validation", stubGenerateValidation)
	})
}

type ValidationGenRequest struct {
	Prompt string `json:"prompt"`
	Entity string `json:"entity"`
}

type ValidationGenResponse struct {
	Script      string `json:"script"`
	Explanation string `json:"explanation"`
}

func stubGenerateValidation(w http.ResponseWriter, r *http.Request) {
	var req ValidationGenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	prompt := strings.ToLower(req.Prompt)
	var script string
	var explanation string

	// Simple heuristic generation for MVP
	if strings.Contains(prompt, "age") || strings.Contains(prompt, "old") {
		script = `// Age validation (ASL)
// Must be at least 18
input.age >= 18`
		explanation = "Generated a check for age >= 18 based on your prompt."
	} else if strings.Contains(prompt, "email") {
		script = `// Email validation (ASL)
// Check for valid format
input.email.contains("@") && input.email.contains(".")`
		explanation = "Generated a basic email format check."
	} else if strings.Contains(prompt, "required") || strings.Contains(prompt, "empty") {
		script = `// Required field check (ASL)
input.name != ""`
		explanation = "Generated a non-empty check for common fields."
	} else {
		script = `// Generic validation rule (ASL)
// Example:
// input.status == "Active"`
		explanation = "Generated a template validation script. Customize the logic as needed."
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ValidationGenResponse{
		Script:      script,
		Explanation: explanation,
	})
}

// proxyJSON forwards JSON requests to the specified target
func proxyJSON(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if target == "" {
			http.Error(w, "no proxy target", http.StatusServiceUnavailable)
			return
		}
		url := target + r.URL.Path
		if r.URL.RawQuery != "" {
			url += "?" + r.URL.RawQuery
		}
		req, err := http.NewRequestWithContext(r.Context(), r.Method, url, r.Body)
		if err != nil {
			http.Error(w, "failed to create upstream request", http.StatusBadGateway)
			return
		}
		req.Header.Set("Content-Type", r.Header.Get("Content-Type"))
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "upstream error", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}

// proxyJSONV2 is a compatibility wrapper while we migrate callers.
func proxyJSONV2(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { // direct pass-through
		proxyJSON(target)(w, r)
	}
}

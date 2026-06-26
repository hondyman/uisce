package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

// A tiny reverse proxy used during local development. It listens on :29080 by
// default and forwards requests to service-specific backends. Targets are
// configurable via environment variables so this can be used across setups.

func newProxy(target string) *httputil.ReverseProxy {
	u, err := url.Parse(target)
	if err != nil {
		log.Fatalf("invalid proxy target %s: %v", target, err)
	}
	p := httputil.NewSingleHostReverseProxy(u)
	// set a short timeout-friendly transport
	p.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	// adjust request host to backend
	originalDirector := p.Director
	p.Director = func(req *http.Request) {
		originalDirector(req)
		// preserve original path (SingleHostReverseProxy replaced it already)
	}
	return p
}

func main() {
	// flags/env for targets
	listen := flag.String("listen", getEnv("PROXY_LISTEN", ":29080"), "listen address")
	ruleEngine := flag.String("rule", getEnv("RULE_ENGINE_URL", "http://localhost:8083"), "rule-engine base URL")
	backend := flag.String("backend", getEnv("BACKEND_URL", "http://localhost:8080"), "backend/api gateway URL")
	hasura := flag.String("hasura", getEnv("HASURA_URL", "http://100.84.126.19:8085"), "hasura base URL (REMOTE ONLY)")
	flag.Parse()

	log.Printf("proxy starting on %s; rule=%s backend=%s hasura=%s", *listen, *ruleEngine, *backend, *hasura)

	ruleProxy := newProxy(*ruleEngine)
	backendProxy := newProxy(*backend)
	hasuraProxy := newProxy(*hasura)

	mux := http.NewServeMux()

	// route /api/validation-rules (and variants) -> rule-engine
	mux.HandleFunc("/api/validation-rules", func(w http.ResponseWriter, r *http.Request) { ruleProxy.ServeHTTP(w, r) })
	mux.HandleFunc("/api/validation-rules/", func(w http.ResponseWriter, r *http.Request) { ruleProxy.ServeHTTP(w, r) })

	// GraphQL -> Hasura
	mux.HandleFunc("/v1/graphql", func(w http.ResponseWriter, r *http.Request) { hasuraProxy.ServeHTTP(w, r) })
	mux.HandleFunc("/v1/graphql/", func(w http.ResponseWriter, r *http.Request) { hasuraProxy.ServeHTTP(w, r) })

	// fallback for /api -> backend
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		// If path looks like marketplace or other special routes you can add cases here
		backendProxy.ServeHTTP(w, r)
	})

	// simple health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","proxy":true}`))
	})

	srv := &http.Server{
		Addr:         *listen,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// Local developer proxy command. Run with:
//   go run ./backend/local/cmd/proxy
// It listens on :29080 by default and proxies specific paths to local services.

func newProxy(target string) *httputil.ReverseProxy {
	u, err := url.Parse(target)
	if err != nil {
		log.Fatalf("invalid proxy target %s: %v", target, err)
	}
	p := httputil.NewSingleHostReverseProxy(u)
	p.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	// Ensure Host header and URL host are set to upstream host. This helps
	// some backends that rely on Host and also enables proper WebSocket
	// upgrade handling when the proxy forwards requests.
	originalDirector := p.Director
	p.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = u.Host
		// Ensure scheme/host are set correctly on URL
		req.URL.Scheme = u.Scheme
		req.URL.Host = u.Host
	}
	return p
}

func main() {
	listen := flag.String("listen", getEnv("PROXY_LISTEN", ":29080"), "listen address")
	ruleEngine := flag.String("rule", getEnv("RULE_ENGINE_URL", "http://localhost:8083"), "rule-engine base URL")
	backend := flag.String("backend", getEnv("BACKEND_URL", "http://localhost:8080"), "backend/api gateway URL")
	hasura := flag.String("hasura", getEnv("HASURA_URL", "http://localhost:8080"), "hasura base URL")
	flag.Parse()

	log.Printf("proxy starting on %s; rule=%s backend=%s hasura=%s", *listen, *ruleEngine, *backend, *hasura)

	ruleProxy := newProxy(*ruleEngine)
	backendProxy := newProxy(*backend)
	hasuraProxy := newProxy(*hasura)

	mux := http.NewServeMux()

	// route /api/validation-rules -> rule-engine
	mux.HandleFunc("/api/validation-rules", func(w http.ResponseWriter, r *http.Request) { ruleProxy.ServeHTTP(w, r) })
	mux.HandleFunc("/api/validation-rules/", func(w http.ResponseWriter, r *http.Request) { ruleProxy.ServeHTTP(w, r) })

	// GraphQL -> Hasura
	mux.HandleFunc("/v1/graphql", func(w http.ResponseWriter, r *http.Request) { hasuraProxy.ServeHTTP(w, r) })
	mux.HandleFunc("/v1/graphql/", func(w http.ResponseWriter, r *http.Request) { hasuraProxy.ServeHTTP(w, r) })

	// Add more specific routes if needed in the future
	// Debug endpoint: echo headers and basic info so we can validate proxy-to-backend path
	mux.HandleFunc("/api/debug/headers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"path":   r.URL.Path,
			"method": r.Method,
			"headers": map[string]string{
				"X-Tenant-ID":          jwtmiddleware.GetClaimsFromContext(r).TenantID,
				"X-Tenant-Datasource-ID": r.Header.Get("X-Tenant-Datasource-ID"),
				"Host":                 r.Host,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// For all other /api routes, forward to backend api gateway
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		backendProxy.ServeHTTP(w, r)
	})

	// health endpoint
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

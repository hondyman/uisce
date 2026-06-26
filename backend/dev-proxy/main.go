package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

func main() {
	backend := os.Getenv("BACKEND_URL")
	if backend == "" {
		backend = "http://localhost:3002"
	}
	target, err := url.Parse(backend)
	if err != nil {
		log.Fatalf("invalid BACKEND_URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		// Default director will set scheme/host/path based on target.
		// Keep the request path as-is (so /api/views forwards to backend /api/views).
		originalDirector(req)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (strings.HasPrefix(origin, "http://localhost:517") || strings.HasPrefix(origin, "http://127.0.0.1:517")) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, X-User-Id, X-Tenant-Id, X-Datasource-Id")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, PATCH, DELETE")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		proxy.ServeHTTP(w, r)
	})

	addr := ":8001"
	log.Printf("Dev proxy starting on %s -> %s", addr, backend)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("proxy failed: %v", err)
	}
}

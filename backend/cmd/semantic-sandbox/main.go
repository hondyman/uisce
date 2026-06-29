package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/hondyman/semlayer/backend/internal/pagestudio"
)

func main() {
	port := flag.String("port", "8081", "Port to run sandbox on")
	dataDir := flag.String("data", "./semantic", "Path to semantic data directory")
	flag.Parse()

	r := chi.NewRouter()

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Tenant-ID", "X-Env", "X-User-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("sandbox ok"))
	})

	// Page Studio Mocks
	r.Get("/api/page-studio/pages", func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, filepath.Join(*dataDir, "pages", "pages.json"))
	})

	r.Get("/api/page-studio/pages/slug/{slug}", func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		// In a real implementation we'd parse the pages.json and find the specific page.
		// For MVP, if we are loading from specific files, we might need a better structure.
		// Current semctl pull writes all pages to one file.
		pagesPath := filepath.Join(*dataDir, "pages", "pages.json")

		data, err := os.ReadFile(pagesPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var pages []pagestudio.CorePage
		if err := json.Unmarshal(data, &pages); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, p := range pages {
			if p.Slug == slug {
				json.NewEncoder(w).Encode(p)
				return
			}
		}
		http.Error(w, "page not found", http.StatusNotFound)
	})

	// API Studio Mocks
	r.Get("/api/api-studio/endpoints", func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, filepath.Join(*dataDir, "apis", "endpoints.json"))
	})

	fmt.Printf("Starting Semantic Sandbox on :%s serving %s\n", *port, *dataDir)
	if err := http.ListenAndServe(":"+*port, r); err != nil {
		fmt.Printf("Server failed: %v\n", err)
		os.Exit(1)
	}
}

func serveFile(w http.ResponseWriter, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			w.Write([]byte("[]"))
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

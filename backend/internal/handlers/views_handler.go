package handlers

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/viewmodel"
)

// ViewsHandler serves generated view metadata and definitions from the runtime directory.
type ViewsHandler struct {
	ViewsDir    string
	ResolvedDir string
}

func NewViewsHandler(viewsDir string, resolvedDir string) *ViewsHandler {
	return &ViewsHandler{ViewsDir: viewsDir, ResolvedDir: resolvedDir}
}

// RegisterRoutes mounts views routes
func (h *ViewsHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/views", func(r chi.Router) {
		r.Get("/", h.ListViews)
		r.Get("/{name}", h.GetView)
		r.Get("/{name}/download", h.DownloadView)
	})
}

// ListViews returns a lightweight catalog of available views with basic metadata.
func (h *ViewsHandler) ListViews(w http.ResponseWriter, r *http.Request) {
	// Choose source dir
	dir := h.ViewsDir
	if src := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("source"))); src == "resolved" && h.ResolvedDir != "" {
		dir = h.ResolvedDir
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"views": []any{}, "total": 0, "page": 1, "page_size": 0})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": fmt.Sprintf("failed to read views dir: %v", err)})
		return
	}

	// Query params
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	page := parseIntDefault(r.URL.Query().Get("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := parseIntDefault(r.URL.Query().Get("page_size"), 50)
	if pageSize <= 0 || pageSize > 500 {
		pageSize = 50
	}

	type viewItem struct {
		Name        string    `json:"name"`
		Title       string    `json:"title,omitempty"`
		Description string    `json:"description,omitempty"`
		CubeCount   int       `json:"cube_count"`
		FolderCount int       `json:"folder_count"`
		ModifiedAt  time.Time `json:"modified_at"`
		ETag        string    `json:"etag"`
	}

	var allItems []viewItem
	var etagHasher = sha1.New()
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		fi, fiErr := e.Info()
		if fiErr != nil {
			// skip entries we can't stat (avoid nil pointer panics)
			continue
		}
		fp := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(fp)
		if err != nil {
			continue
		}
		var v viewmodel.View
		if err := json.Unmarshal(b, &v); err != nil {
			continue
		}
		// compute file etag
		fe := fileETag(b, fi)
		// filter
		if q != "" {
			if !strings.Contains(strings.ToLower(v.Name), q) && !strings.Contains(strings.ToLower(v.Title), q) && !strings.Contains(strings.ToLower(v.Description), q) {
				continue
			}
		}
		item := viewItem{
			Name:        v.Name,
			Title:       v.Title,
			Description: v.Description,
			CubeCount:   len(v.Cubes),
			FolderCount: len(v.Folders),
			ModifiedAt:  fi.ModTime(),
			ETag:        fe,
		}
		allItems = append(allItems, item)
		// include in list etag summary input
		etagHasher.Write([]byte(e.Name()))
		etagHasher.Write([]byte(fi.ModTime().UTC().Format(time.RFC3339Nano)))
	}

	sort.Slice(allItems, func(i, j int) bool { return strings.ToLower(allItems[i].Name) < strings.ToLower(allItems[j].Name) })

	total := len(allItems)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	pageItems := allItems[start:end]

	// ETag for the collection (weak)
	listETag := "W/\"" + hex.EncodeToString(etagHasher.Sum(nil)) + "\""
	w.Header().Set("ETag", listETag)
	if inm := r.Header.Get("If-None-Match"); inm != "" && inm == listETag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"views":     pageItems,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetView returns the full JSON definition for a given view name.
func (h *ViewsHandler) GetView(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "missing view name"})
		return
	}
	// Basic path sanitization
	if strings.Contains(name, string(os.PathSeparator)) || strings.Contains(name, "..") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid view name"})
		return
	}
	dir := h.ViewsDir
	if src := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("source"))); src == "resolved" && h.ResolvedDir != "" {
		dir = h.ResolvedDir
	}
	fp := filepath.Join(dir, name+".json")
	b, err := os.ReadFile(fp)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "view not found"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": fmt.Sprintf("failed to read view: %v", err)})
		return
	}
	// ETag support
	fi, _ := os.Stat(fp)
	etag := fileETag(b, fi)
	w.Header().Set("ETag", etag)
	if inm := r.Header.Get("If-None-Match"); inm != "" && inm == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Stream raw JSON content (already JSON); unmarshal and return as object to ensure valid JSON
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": fmt.Sprintf("invalid view json: %v", err)})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"view": v})
}

// DownloadView streams the raw JSON with Content-Disposition for download.
func (h *ViewsHandler) DownloadView(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "missing view name"})
		return
	}
	if strings.Contains(name, string(os.PathSeparator)) || strings.Contains(name, "..") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid view name"})
		return
	}
	dir := h.ViewsDir
	if src := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("source"))); src == "resolved" && h.ResolvedDir != "" {
		dir = h.ResolvedDir
	}
	fp := filepath.Join(dir, name+".json")
	b, err := os.ReadFile(fp)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "view not found"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": fmt.Sprintf("failed to read view: %v", err)})
		return
	}
	fi, _ := os.Stat(fp)
	etag := fileETag(b, fi)
	w.Header().Set("ETag", etag)
	if inm := r.Header.Get("If-None-Match"); inm != "" && inm == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.json\"", name))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func fileETag(b []byte, fi os.FileInfo) string {
	h := sha1.New()
	h.Write(b)
	if fi != nil {
		h.Write([]byte(fi.ModTime().UTC().Format(time.RFC3339Nano)))
		h.Write([]byte(strconv.FormatInt(fi.Size(), 10)))
	}
	return "\"" + hex.EncodeToString(h.Sum(nil)) + "\""
}

// errorsIs abstracts fs errors for Go <1.20 compatibility in some environments.
func errorsIs(err error, target error) bool {
	return err != nil && (err == target || isPathError(err, target))
}

func isPathError(err error, target error) bool {
	if pe, ok := err.(*fs.PathError); ok {
		return pe.Err == target
	}
	return false
}

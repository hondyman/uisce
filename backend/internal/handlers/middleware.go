package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func ParseJSON(r *http.Request, dst interface{}) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

func SendJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func SendError(w http.ResponseWriter, code int, message string) {
	SendJSON(w, code, map[string]string{"error": message})
}

func SendErrorf(w http.ResponseWriter, code int, format string, args ...interface{}) {
	SendJSON(w, code, map[string]string{"error": http.StatusText(code) + ": " + fmt.Sprintf(format, args...)})
}

func Paginate(r *http.Request) (limit, offset int) {
	limit = 50
	offset = 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	return limit, offset
}

func PaginateWithMax(r *http.Request, maxLimit int) (limit, offset int) {
	limit, offset = Paginate(r)
	if limit > maxLimit {
		limit = maxLimit
	}
	return limit, offset
}

func GetQueryInt(r *http.Request, param string, defaultVal int) int {
	if v := r.URL.Query().Get(param); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

func GetQueryBool(r *http.Request, param string) bool {
	return r.URL.Query().Get(param) == "true" || r.URL.Query().Get(param) == "1"
}

func GetQueryString(r *http.Request, param string) string {
	return r.URL.Query().Get(param)
}

func RequireJSONContentType(w http.ResponseWriter, r *http.Request) bool {
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return false
	}
	return true
}

func RequireAuth(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("Authorization") == "" {
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return false
	}
	return true
}

type Claims struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
}

func GetClaims(r *http.Request) *Claims {
	return nil
}

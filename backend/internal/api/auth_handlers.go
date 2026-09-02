package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/auth"
)

// RegisterAuthRoutes registers the routes that survive the local
// password/session auth removal (Keycloak is the sole identity provider —
// see AuthContextMiddleware). User preferences are unrelated to
// authentication and keep working off the caller's already-validated
// identity (auth.GetUserFromContext).
func (s *Server) RegisterAuthRoutes(r chi.Router) {
	r.Route("/users/{userId}/preferences", func(r chi.Router) {
		r.Get("/", s.getUserPreferences)
		r.Put("/", s.updateUserPreferences)
	})
}

// getUserPreferences returns light-weight user preferences (language) stored on the users table
func (s *Server) getUserPreferences(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		respondWithError(w, http.StatusBadRequest, "User ID is required")
		return
	}

	var language sql.NullString
	err := s.DB.QueryRowContext(r.Context(), `SELECT language FROM public.users WHERE id = $1`, userID).Scan(&language)
	if err != nil {
		if err == sql.ErrNoRows {
			respond(w, r, map[string]string{"language": "en"}, nil)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch preferences")
		return
	}

	lang := "en"
	if language.Valid && language.String != "" {
		lang = language.String
	}
	respond(w, r, map[string]string{"language": lang}, nil)
}

// updateUserPreferences updates language preference in the users table. Users may only update their own preferences.
func (s *Server) updateUserPreferences(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		respondWithError(w, http.StatusBadRequest, "User ID is required")
		return
	}

	// Ensure authenticated user matches the requested user to prevent privilege escalation.
	if u, ok := auth.GetUserFromContext(r.Context()); ok {
		if u.ID != userID {
			respondWithError(w, http.StatusForbidden, "Not authorized to update another user's preferences")
			return
		}
	}

	var payload struct {
		Language string `json:"language"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Basic validation: allow only short locale codes like en, es, fr
	if payload.Language == "" {
		payload.Language = "en"
	}

	_, err := s.DB.ExecContext(r.Context(), `UPDATE public.users SET language = $1, updated_at = NOW() WHERE id = $2`, payload.Language, userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update preferences")
		return
	}

	respond(w, r, map[string]string{"language": payload.Language}, nil)
}

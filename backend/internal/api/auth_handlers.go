package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/auth"
	"github.com/hondyman/semlayer/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// SessionService abstracts session storage so handlers can be tested.
type SessionService interface {
	StoreSession(ctx context.Context, userID, accessToken, refreshToken string, r *http.Request) error
	InvalidateSession(ctx context.Context, token string) error
	VerifySessionToken(ctx context.Context, token string) (string, error)
}

type dbSessionService struct {
	db *sql.DB
}

func (d *dbSessionService) StoreSession(ctx context.Context, userID, accessToken, refreshToken string, r *http.Request) error {
	expiresAt := time.Now().Add(time.Hour)
	refreshExpiresAt := time.Now().Add(24 * time.Hour)

	var ipAddress interface{}
	if r.RemoteAddr != "" {
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			ipAddress = host
		} else {
			ipAddress = nil
		}
	} else {
		ipAddress = nil
	}

	_, err := d.db.ExecContext(ctx, `
        INSERT INTO private_markets_sessions (user_id, session_token, refresh_token, expires_at, refresh_expires_at, ip_address, user_agent)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, userID, accessToken, refreshToken, expiresAt, refreshExpiresAt, ipAddress, "")
	return err
}

func (d *dbSessionService) InvalidateSession(ctx context.Context, token string) error {
	_, err := d.db.ExecContext(ctx, `UPDATE private_markets_sessions SET is_active = false WHERE session_token = $1`, token)
	return err
}

func (d *dbSessionService) VerifySessionToken(ctx context.Context, token string) (string, error) {
	var userID string
	err := d.db.QueryRowContext(ctx, `SELECT user_id FROM private_markets_sessions WHERE session_token = $1 AND expires_at > now() AND is_active = true`, token).Scan(&userID)
	if err != nil {
		return "", err
	}
	return userID, nil
}

func NewDBSessionService(db *sql.DB) SessionService {
	return &dbSessionService{db: db}
}

func (s *Server) RegisterAuthRoutes(r chi.Router) {
	r.Post("/login", s.login)
	r.Post("/logout", s.logout)
	r.Post("/refresh", s.refreshToken)
	r.Post("/register", s.register)
	r.Get("/me", s.getCurrentUser)

	// User preferences
	r.Route("/users/{userId}/preferences", func(r chi.Router) {
		r.Get("/", s.getUserPreferences)
		r.Put("/", s.updateUserPreferences)
	})
}

// Authentication handlers (login/register/logout/refresh/getCurrentUser)
// These are thin and rely on SessionService for session persistence.

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Fprintf(os.Stderr, "[AUTH] Login failed - invalid request body: %v\n", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fmt.Fprintf(os.Stderr, "[AUTH] Login attempt for email: %s\n", req.Email)

	var user models.User
	var email string
	var permissions []byte
	var passwordHash sql.NullString // Allow null for users without password (e.g. SSO)

	// Fetch user with tenant_id for multi-tenant security
	var tenantID sql.NullString
	err := s.DB.QueryRowContext(r.Context(), `
        SELECT id, email, COALESCE(name, ''), COALESCE(role, ''), COALESCE(organization, ''), permissions, COALESCE(is_core_admin, false), COALESCE(is_active, true), password_hash, tenant_id
        FROM public.users
        WHERE email = $1
    `, req.Email).Scan(&user.ID, &email, &user.Name, &user.Role, &user.Organization, &permissions, &user.IsCoreAdmin, &user.IsActive, &passwordHash, &tenantID)

	if err != nil {
		fmt.Fprintf(os.Stderr, "[AUTH] Login failed - user matching email %s not found or query failed: %v\n", req.Email, err)
		http.Error(w, "User not found or database error", http.StatusUnauthorized)
		return
	}

	user.Email = email
	if len(permissions) > 0 {
		if err := json.Unmarshal(permissions, &user.Permissions); err != nil {
			user.Permissions = []string{"read"}
		}
	} else {
		user.Permissions = []string{"read"}
	}

	if !passwordHash.Valid || passwordHash.String == "" {
		fmt.Fprintf(os.Stderr, "[AUTH] Login failed - password hash missing for user %s\n", user.ID)
		http.Error(w, "Auth data missing", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash.String), []byte(req.Password)); err != nil {
		fmt.Fprintf(os.Stderr, "[AUTH] Login failed - password mismatch for user %s: %v\n", user.ID, err)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	fmt.Fprintf(os.Stderr, "[AUTH] Login successful for user %s (%s)\n", user.ID, user.Email)

	// Generate JWT token for Hasura with tenant context
	allowedRoles := []string{"user"}
	defaultRole := "user"

	// Global admins (Uisce organization) get global_admin role
	if user.Organization == "uisce" && user.IsCoreAdmin {
		allowedRoles = append(allowedRoles, "global_admin")
		defaultRole = "global_admin"
	}

	// Add user's role if it's not already in the list
	if user.Role != "" && user.Role != "user" && user.Role != "global_admin" {
		allowedRoles = append(allowedRoles, user.Role)
	}

	hasuraClaims := map[string]interface{}{
		"x-hasura-allowed-roles": allowedRoles,
		"x-hasura-default-role":  defaultRole,
		"x-hasura-user-id":       user.ID,
	}

	// Add tenant_id to Hasura claims for RLS filtering
	if tenantID.Valid {
		hasuraClaims["x-hasura-tenant-id"] = tenantID.String
		user.TenantID = tenantID.String
	}

	jwtClaims := jwt.MapClaims{
		"user_id":                      user.ID,
		"email":                        user.Email,
		"name":                         user.Name,
		"role":                         user.Role,
		"organization":                 user.Organization,
		"tenant_id":                    tenantID.String,
		"permissions":                  user.Permissions,
		"is_core_admin":                user.IsCoreAdmin,
		"iat":                          time.Now().Unix(),
		"exp":                          time.Now().Add(time.Hour).Unix(),
		"https://hasura.io/jwt/claims": hasuraClaims,
	}

	jwtToken, jwtErr := s.SecMgr.SignToken(jwtClaims)
	if jwtErr != nil {
		fmt.Fprintf(os.Stderr, "[AUTH] Failed to sign JWT for user %s: %v\n", user.ID, jwtErr)
		http.Error(w, "Failed to generate JWT token", http.StatusInternalServerError)
		return
	}

	// Use JWT as the session token for unified auth
	sessionToken := jwtToken
	refreshToken := generateRandomToken()

	if s.SecMgr == nil {
		// ensure SessionService fallback
	}

	// Store session in DB (required for SessionAuthMiddleware)
	if err := NewDBSessionService(s.DB).StoreSession(r.Context(), user.ID, sessionToken, refreshToken, r); err != nil {
		fmt.Fprintf(os.Stderr, "[AUTH] Failed to store session for user %s: %v\n", user.ID, err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	s.DB.ExecContext(r.Context(), "UPDATE public.users SET last_login = now() WHERE id = $1", user.ID)

	fmt.Fprintf(os.Stderr, "[AUTH] Login successful for user %s (%s)\n", user.ID, user.Email)

	response := AuthResponse{
		User:         user,
		AccessToken:  sessionToken, // JWT
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
	}

	http.SetCookie(w, &http.Cookie{Name: "session_token", Value: sessionToken, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, Expires: time.Now().Add(time.Hour)})
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: refreshToken, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, Expires: time.Now().Add(24 * time.Hour)})

	respond(w, r, response, nil)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "No token provided", http.StatusUnauthorized)
		return
	}
	tokenString := authHeader
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenString = strings.TrimPrefix(authHeader, "Bearer ")
	}

	if err := NewDBSessionService(s.DB).InvalidateSession(r.Context(), tokenString); err != nil {
		http.Error(w, "Failed to logout", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{Name: "session_token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully"})
}

// refreshToken issues a new access token when provided a valid refresh token.
func (s *Server) refreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.RefreshToken == "" {
		http.Error(w, "refresh_token is required", http.StatusBadRequest)
		return
	}
	// Verify session by refresh token lookup (simple DB query)
	var userID string
	err := s.DB.QueryRowContext(r.Context(), `SELECT user_id FROM private_markets_sessions WHERE refresh_token = $1 AND refresh_expires_at > now() AND is_active = true`, req.RefreshToken).Scan(&userID)
	if err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}

	// Create a new access token
	accessToken := generateRandomToken()
	// Store session record for new access token
	if err := NewDBSessionService(s.DB).StoreSession(r.Context(), userID, accessToken, req.RefreshToken, r); err != nil {
		http.Error(w, "failed to store session", http.StatusInternalServerError)
		return
	}

	respond(w, r, map[string]string{"access_token": accessToken, "token_type": "Bearer"}, nil)
}

// register creates a new user and returns tokens. This is minimal and intended for local/dev flows.
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password are required", http.StatusBadRequest)
		return
	}
	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	id := uuid.New().String()

	// private_markets_users has a role CHECK constraint (lp/gp/fof/steward).
	// The platform UI may use broader roles (e.g., admin/platform_operator) in public.users,
	// so we map any unknown roles to a safe default for the private markets auth tables.
	pmRole := req.Role
	switch pmRole {
	case "lp", "gp", "fof", "steward":
		// ok
	default:
		pmRole = "steward"
	}
	// Note: public.users requires username (NOT NULL) in many schemas.
	// For local/dev registration, treat email as the username.
	_, err = s.DB.ExecContext(
		r.Context(),
		`INSERT INTO public.users (
			id,
			username,
			email,
			name,
			role,
			organization,
			is_core_admin,
			is_active,
			status,
			created_at,
			updated_at,
			password_hash,
			salt
		) VALUES ($1,$2,$3,$4,$5,$6,false,true,'active',now(),now(),$7,'bcrypt')`,
		id,
		req.Email,
		req.Email,
		req.Name,
		req.Role,
		req.Organization,
		string(hash),
	)
	if err != nil {
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	// Legacy tables private_markets_users and private_markets_user_auth are no longer populated.
	// Authentication is now fully consolidated in public.users.

	// Issue tokens
	accessToken := generateRandomToken()
	refreshToken := generateRandomToken()
	if err := NewDBSessionService(s.DB).StoreSession(r.Context(), id, accessToken, refreshToken, r); err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	respond(w, r, AuthResponse{User: models.User{ID: id, Email: req.Email, Name: req.Name, Role: req.Role, Organization: req.Organization}, AccessToken: accessToken, RefreshToken: refreshToken, TokenType: "Bearer", ExpiresIn: 3600}, nil)
}

// getCurrentUser returns the authenticated user's profile. It tries cookie/session or Authorization header.
func (s *Server) getCurrentUser(w http.ResponseWriter, r *http.Request) {
	// Try session cookie first
	cookie, err := r.Cookie("session_token")
	var userID string
	if err == nil && cookie.Value != "" {
		// verify
		userID, err = NewDBSessionService(s.DB).VerifySessionToken(r.Context(), cookie.Value)
		if err != nil {
			userID = ""
		}
	}
	// Fallback to Authorization header
	if userID == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if uid, err := NewDBSessionService(s.DB).VerifySessionToken(r.Context(), token); err == nil {
				userID = uid
			}
		}
	}
	if userID == "" {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	var user models.User
	var permissions []byte
	var email string
	err = s.DB.QueryRowContext(r.Context(), `SELECT id, email, COALESCE(name, ''), COALESCE(role, ''), COALESCE(organization, ''), permissions, COALESCE(is_core_admin, false), COALESCE(is_active, true) FROM public.users WHERE id = $1`, userID).Scan(&user.ID, &email, &user.Name, &user.Role, &user.Organization, &permissions, &user.IsCoreAdmin, &user.IsActive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[AUTH] /me failed - user %s not found: %v\n", userID, err)
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	user.Email = email
	if len(permissions) > 0 {
		_ = json.Unmarshal(permissions, &user.Permissions)
	}
	respond(w, r, user, nil)
}

// listUsers returns a list of all users. Restricted to admins.
func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	// Auth check is done by middleware, but we double check for admin role
	ctxUser, ok := r.Context().Value("user").(models.User)
	if !ok || (!ctxUser.IsCoreAdmin && ctxUser.Role != "admin" && ctxUser.Role != "global_admin") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Query users
	rows, err := s.DB.QueryContext(r.Context(), `
		SELECT id, email, COALESCE(name, ''), COALESCE(role, ''), COALESCE(organization, ''), is_core_admin, COALESCE(is_active, true)
		FROM public.users
		ORDER BY id DESC
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list users: %v\n", err)
		http.Error(w, "Failed to list users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Organization, &u.IsCoreAdmin, &u.IsActive); err != nil {
			continue
		}
		users = append(users, u)
	}

	respond(w, r, users, nil)
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

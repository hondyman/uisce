package api

import (
	"context"
	"encoding/json"
	"net/http"

	"calendar-service/internal/database"
	"calendar-service/internal/middleware"
	"calendar-service/internal/services"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type TeamHandler struct {
	dbClient     *database.Client
	auditService *services.AuditServiceImpl
	logger       *logrus.Entry
}

func NewTeamHandler(db *database.Client, as *services.AuditServiceImpl, logger *logrus.Entry) *TeamHandler {
	return &TeamHandler{
		dbClient:     db,
		auditService: as,
		logger:       logger.WithField("handler", "team"),
	}
}

type CreateTeamRequest struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	Slug             string `json:"slug"`
	SubscriptionTier string `json:"subscription_tier"`
}

func (h *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := middleware.ExtractTenantIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := middleware.ExtractUserIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	teamID := uuid.New()

	query := `
		INSERT INTO teams (id, name, description, slug, tenant_id, owner_id, subscription_tier)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, slug, created_at
	`

	var team struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Slug      string `json:"slug"`
		CreatedAt string `json:"created_at"`
	}

	err = h.dbClient.Pool().QueryRow(ctx, query,
		teamID, req.Name, req.Description, req.Slug, tenantID, userID, req.SubscriptionTier,
	).Scan(&team.ID, &team.Name, &team.Slug, &team.CreatedAt)

	if err != nil {
		http.Error(w, "Failed to create team", http.StatusInternalServerError)
		return
	}

	h.addTeamMember(ctx, teamID.String(), userID, "owner")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"team": team,
	})
}

func (h *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]

	query := `
		SELECT id, name, description, slug, avatar_url, settings, subscription_tier, billing_email, created_at, updated_at
		FROM teams
		WHERE id = $1
	`

	var team map[string]interface{}
	var settings []byte

	err := h.dbClient.Pool().QueryRow(r.Context(), query, teamID).Scan(
		&team["id"], &team["name"], &team["description"], &team["slug"],
		&team["avatar_url"], &settings, &team["subscription_tier"],
		&team["billing_email"], &team["created_at"], &team["updated_at"],
	)

	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(team)
}

func (h *TeamHandler) ListTeams(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := middleware.ExtractTenantIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := `
		SELECT id, name, description, slug, created_at, subscription_tier
		FROM teams
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := h.dbClient.Pool().Query(ctx, query, tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list teams")
		http.Error(w, "Failed to list teams", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var teams []map[string]interface{}
	for rows.Next() {
		var id, name, description, slug, subscriptionTier, createdAt string
		if err := rows.Scan(&id, &name, &description, &slug, &createdAt, &subscriptionTier); err != nil {
			continue
		}
		teams = append(teams, map[string]interface{}{
			"id":                id,
			"name":              name,
			"description":       description,
			"slug":              slug,
			"created_at":        createdAt,
			"subscription_tier": subscriptionTier,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(teams)
}

func (h *TeamHandler) InviteTeamMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]

	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `
		INSERT INTO team_invitations (id, team_id, email, role, expires_at)
		VALUES ($1, $2, $3, $4, NOW() + INTERVAL '7 days')
		RETURNING id, email, role, expires_at
	`

	var id, expiresAt string
	err := h.dbClient.Pool().QueryRow(r.Context(), query,
		uuid.New(), teamID, req.Email, req.Role,
	).Scan(&id, &req.Email, &req.Role, &expiresAt)

	if err != nil {
		h.logger.WithError(err).Error("Failed to invite team member")
		http.Error(w, "Failed to invite member", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "invitation_sent",
	})
}

func (h *TeamHandler) CreateSharedCalendar(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]

	userID, _ := middleware.ExtractUserIDFromContextStrict(r.Context())

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Color       string `json:"color"`
		Visibility  string `json:"visibility"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `
		INSERT INTO shared_calendars (id, team_id, owner_id, name, description, color, visibility)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, created_at
	`

	var id, name, createdAt string
	err := h.dbClient.Pool().QueryRow(r.Context(), query,
		uuid.New(), teamID, userID, req.Name, req.Description, req.Color, req.Visibility,
	).Scan(&id, &name, &createdAt)

	if err != nil {
		h.logger.WithError(err).Error("Failed to create shared calendar")
		http.Error(w, "Failed to create shared calendar", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "created",
	})
}

func (h *TeamHandler) addTeamMember(ctx context.Context, teamID, userID, role string) error {
	query := `
		INSERT INTO team_members (id, team_id, user_id, role)
		VALUES ($1, $2, $3, $4)
	`

	_, err := h.dbClient.Pool().Exec(ctx, query, uuid.New(), teamID, userID, role)
	return err
}

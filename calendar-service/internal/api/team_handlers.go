package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"calendar-service/internal/hasura"
	"calendar-service/internal/middleware"
	"calendar-service/internal/services"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// TeamHandler handles team/workspace API endpoints
type TeamHandler struct {
	hasuraClient *hasura.Client
	auditService *services.AuditServiceImpl
	logger       *logrus.Entry
}

// NewTeamHandler creates a new team handler
func NewTeamHandler(hc *hasura.Client, as *services.AuditServiceImpl, logger *logrus.Entry) *TeamHandler {
	return &TeamHandler{
		hasuraClient: hc,
		auditService: as,
		logger:       logger.WithField("handler", "team"),
	}
}

// CreateTeamRequest represents team creation request
type CreateTeamRequest struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	Slug             string `json:"slug"`
	SubscriptionTier string `json:"subscription_tier"`
}

// CreateTeam creates a new team
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

	mutation := `
    mutation CreateTeam($object: teams_insert_input!) {
        insert_teams_one(object: $object) {
            id name slug created_at
        }
    }
    `

	object := map[string]interface{}{
		"name":              req.Name,
		"description":       req.Description,
		"slug":              req.Slug,
		"tenant_id":         tenantID,
		"owner_id":          userID,
		"subscription_tier": req.SubscriptionTier,
	}

	var result struct {
		InsertOne struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Slug      string `json:"slug"`
			CreatedAt string `json:"created_at"`
		} `json:"insert_teams_one"`
	}

	if err := h.hasuraClient.Mutate(ctx, mutation, map[string]interface{}{"object": object}, &result); err != nil {
		http.Error(w, "Failed to create team", http.StatusInternalServerError)
		return
	}

	// Add creator as team owner
	h.addTeamMember(ctx, result.InsertOne.ID, userID, "owner")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"team": result.InsertOne,
	})
}

// GetTeam returns a specific team
func (h *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"] // fixed from team_id to id to match common pattern or check router.go

	query := `
    query GetTeam($id: uuid!) {
        teams_by_pk(id: $id) {
            id name description slug avatar_url settings
            subscription_tier billing_email created_at updated_at
        }
    }
    `

	var result struct {
		Team map[string]interface{} `json:"teams_by_pk"`
	}

	if err := h.hasuraClient.QueryRaw(r.Context(), query, map[string]interface{}{
		"id": teamID,
	}, &result); err != nil {
		http.Error(w, "Failed to get team", http.StatusInternalServerError)
		return
	}

	if result.Team == nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result.Team)
}

// ListTeams returns a list of teams for the current tenant
func (h *TeamHandler) ListTeams(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := middleware.ExtractTenantIDFromContextStrict(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := `
    query ListTeams($tenant_id: uuid!) {
        teams(where: {tenant_id: {_eq: $tenant_id}}, order_by: {created_at: desc}) {
            id name description slug created_at subscription_tier
        }
    }
    `

	var result struct {
		Teams []map[string]interface{} `json:"teams"`
	}

	if err := h.hasuraClient.QueryRaw(ctx, query, map[string]interface{}{
		"tenant_id": tenantID,
	}, &result); err != nil {
		h.logger.WithError(err).Error("Failed to list teams")
		http.Error(w, "Failed to list teams", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result.Teams)
}

// InviteTeamMember invites a user to join a team
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

	// Create invitation
	mutation := `
    mutation CreateTeamInvitation($object: team_invitations_insert_input!) {
        insert_team_invitations_one(object: $object) {
            id email role status expires_at
        }
    }
    `

	object := map[string]interface{}{
		"team_id":    teamID,
		"email":      req.Email,
		"role":       req.Role,
		"expires_at": time.Now().Add(7 * 24 * time.Hour),
	}

	if err := h.hasuraClient.Mutate(r.Context(), mutation, map[string]interface{}{"object": object}, &struct{}{}); err != nil {
		h.logger.WithError(err).Error("Failed to invite team member")
		http.Error(w, "Failed to invite member", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "invitation_sent",
	})
}

// CreateSharedCalendar creates a shared team calendar
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

	mutation := `
    mutation CreateSharedCalendar($object: shared_calendars_insert_input!) {
        insert_shared_calendars_one(object: $object) {
            id name created_at
        }
    }
    `

	object := map[string]interface{}{
		"team_id":     teamID,
		"owner_id":    userID,
		"name":        req.Name,
		"description": req.Description,
		"color":       req.Color,
		"visibility":  req.Visibility,
	}

	if err := h.hasuraClient.Mutate(r.Context(), mutation, map[string]interface{}{"object": object}, &struct{}{}); err != nil {
		h.logger.WithError(err).Error("Failed to create shared calendar")
		http.Error(w, "Failed to create shared calendar", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "created",
	})
}

// Helper functions
func (h *TeamHandler) addTeamMember(ctx context.Context, teamID, userID, role string) error {
	mutation := `
    mutation AddTeamMember($object: team_members_insert_input!) {
        insert_team_members_one(object: $object) {
            id
        }
    }
    `

	object := map[string]interface{}{
		"team_id": teamID,
		"user_id": userID,
		"role":    role,
	}

	return h.hasuraClient.Mutate(ctx, mutation, map[string]interface{}{"object": object}, &struct{}{})
}

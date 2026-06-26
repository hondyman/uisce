package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/models"
)

// CollaborationHandler handles API requests for comments and approvals.
type CollaborationHandler struct {
	service *services.CollaborationService
}

// NewCollaborationHandler creates a new CollaborationHandler.
func NewCollaborationHandler(service *services.CollaborationService) *CollaborationHandler {
	return &CollaborationHandler{service: service}
}

// RegisterRoutes registers the routes for CollaborationHandler.
func (h *CollaborationHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/collaboration", func(r chi.Router) {
		r.Get("/comments", h.HandleListComments)
		r.Post("/comments", h.HandleAddComment)
		r.Get("/approvals/{asset_id}", h.HandleGetApprovalStatus)
		r.Post("/approvals/request", h.HandleRequestApproval)
		r.Post("/approvals/update", h.HandleUpdateApprovalStatus)
		r.Post("/access/request", h.HandleRequestSemanticModelAccess)
		r.Get("/access/requests", h.HandleListAccessRequests)
		r.Post("/access/requests/{id}/approve", h.HandleApproveAccessRequest)
		r.Post("/access/requests/{id}/reject", h.HandleRejectAccessRequest)
		r.Get("/claims/effective", h.HandleGetEffectiveClaims)
		r.Post("/claims/simulate", h.HandleSimulateClaims)
		r.Get("/steward/domains", h.HandleGetStewardDomains)
		r.Get("/roles", h.HandleListAllRoles)
		r.Get("/roles/claims", h.HandleListRoleClaims)
		r.Post("/roles/claims", h.HandleUpdateRoleClaim)
		r.Post("/claims/direct", h.HandleGrantDirectClaim)
		r.Delete("/claims/direct/{id}", h.HandleRevokeDirectClaim)
		r.Post("/claims/renewal/{id}", h.HandleRequestClaimRenewal)
		r.Get("/claims/lifecycle", h.HandleGetClaimLifecycleSnapshot)
		r.Get("/policies", h.HandleListAccessPolicies)
		r.Get("/lineage/{node_id}/aware", h.HandleGetClaimAwareLineage)
		r.Post("/policies/simulate", h.HandleSimulatePolicyChange)
		r.Get("/notifications", h.HandleListNotifications)
		r.Post("/notifications/{id}/read", h.HandleMarkNotificationAsRead)
		r.Get("/audit/logs", h.HandleListAccessAuditLogs)
		r.Get("/claims/simulations", h.HandleListClaimSimulations)
		r.Post("/access/denied", h.HandleLogAccessDeniedAttempt)
		r.Post("/alerts/evaluate", h.HandleEvaluateAlert)
		r.Get("/alerts/suppressed", h.HandleListSuppressedAlerts)
		r.Get("/alerts/escalated", h.HandleListEscalatedAlerts)
		r.Post("/alerts/{id}/override", h.HandleOverrideAlertStatus)
		r.Get("/suggestions/claims", h.HandleListClaimSuggestions)
		r.Get("/bundles/claims", h.HandleListClaimBundles)
		r.Get("/drift/claims", h.HandleDetectClaimDrift)
		r.Get("/heatmap", h.HandleGetGovernanceHeatmap)
		r.Get("/conflicts/claims", h.HandleListClaimConflicts)
		r.Post("/conflicts/claims/{id}/resolve", h.HandleResolveClaimConflict)
		r.Get("/notifications/rules", h.HandleListNotificationRules)
		r.Post("/notifications/rules", h.HandleUpdateNotificationRule)
		r.Get("/notifications/preview", h.HandlePreviewRecipients)
	})
}

// HandleListComments retrieves comments for a given asset.
func (h *CollaborationHandler) HandleListComments(w http.ResponseWriter, r *http.Request) {
	assetIDStr := r.URL.Query().Get("asset_id")
	assetID, err := uuid.Parse(assetIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid asset_id"})
		return
	}

	comments, err := h.service.ListComments(r.Context(), assetID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to list comments"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

// HandleAddComment creates a new comment.
func (h *CollaborationHandler) HandleAddComment(w http.ResponseWriter, r *http.Request) {
	var comment models.Comment
	if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid comment payload"})
		return
	}
	// In a real app, AuthorUserID would come from auth context.
	comment.AuthorUserID = "current_user"

	newComment, err := h.service.AddComment(r.Context(), comment)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to add comment"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newComment)
}

// HandleGetApprovalStatus retrieves the approval status for an asset.
func (h *CollaborationHandler) HandleGetApprovalStatus(w http.ResponseWriter, r *http.Request) {
	assetID, err := uuid.Parse(chi.URLParam(r, "asset_id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid asset_id"})
		return
	}
	status, err := h.service.GetApprovalStatus(r.Context(), assetID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Approval status not found"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// HandleRequestApproval submits an asset for review.
func (h *CollaborationHandler) HandleRequestApproval(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AssetID   uuid.UUID `json:"asset_id" binding:"required"`
		AssetType string    `json:"asset_type" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})
		return
	}
	// In a real app, UserID would come from auth context.
	userID := "current_user"
	approval, err := h.service.RequestApproval(r.Context(), req.AssetID, req.AssetType, userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to request approval"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(approval)
}

// HandleUpdateApprovalStatus approves or rejects an asset.
func (h *CollaborationHandler) HandleUpdateApprovalStatus(w http.ResponseWriter, r *http.Request) {
	// This would be a single handler for approve/reject in a real app.
	// For simplicity, we'll assume the service handles the logic.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "decision recorded"})
}

// HandleRequestSemanticModelAccess submits a request for model access.
func (h *CollaborationHandler) HandleRequestSemanticModelAccess(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ModelID    uuid.UUID `json:"model_id" binding:"required"`
		Permission string    `json:"permission" binding:"required"`
		Reason     string    `json:"reason" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})
		return
	}
	// In a real app, UserID would come from auth context.
	userID := "current_user"
	request, err := h.service.RequestSemanticModelAccess(r.Context(), userID, req.ModelID, req.Permission, req.Reason)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create access request"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(request)
}

// HandleListAccessRequests lists access requests for a user or reviewer.
func (h *CollaborationHandler) HandleListAccessRequests(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	reviewerID := r.URL.Query().Get("reviewer_id")

	requests, err := h.service.ListAccessRequests(r.Context(), userID, reviewerID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to list access requests"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(requests)
}

// HandleApproveAccessRequest approves an access request.
func (h *CollaborationHandler) HandleApproveAccessRequest(w http.ResponseWriter, r *http.Request) {
	requestID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request ID"})
		return
	}
	// In a real app, reviewerID would come from auth context.
	reviewerID := "current_reviewer"
	err = h.service.ApproveAccessRequest(r.Context(), requestID, reviewerID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to approve request"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
}

// HandleRejectAccessRequest rejects an access request.
func (h *CollaborationHandler) HandleRejectAccessRequest(w http.ResponseWriter, r *http.Request) {
	requestID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request ID"})
		return
	}
	var req struct {
		Notes string `json:"notes" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload, 'notes' is required"})
		return
	}
	// In a real app, reviewerID would come from auth context.
	reviewerID := "current_reviewer"
	err = h.service.RejectAccessRequest(r.Context(), requestID, reviewerID, req.Notes)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to reject request"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "rejected"})
}

// HandleGetEffectiveClaims retrieves all effective claims for a user.
func (h *CollaborationHandler) HandleGetEffectiveClaims(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "user_id is required"})
		return
	}
	claims, err := h.service.GetEffectiveClaimsForUser(r.Context(), userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get effective claims"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(claims)
}

// HandleSimulateClaims runs a claim simulation.
func (h *CollaborationHandler) HandleSimulateClaims(w http.ResponseWriter, r *http.Request) {
	var req models.ClaimSimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})
		return
	}
	// In a real app, simulatedBy would come from auth context.
	simulatedBy := "current_admin"

	result, err := h.service.SimulateClaims(r.Context(), req, simulatedBy)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to run simulation"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleGetStewardDomains retrieves the domains a user is a steward for.
func (h *CollaborationHandler) HandleGetStewardDomains(w http.ResponseWriter, r *http.Request) {
	// In a real app, UserID would come from auth context.
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "user_id is required"})
		return
	}

	domains, err := h.service.GetStewardDomainsForUser(r.Context(), userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get steward domains"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domains)
}

// HandleListAllRoles lists all defined roles in the system.
func (h *CollaborationHandler) HandleListAllRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.service.ListAllRoles(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to list roles"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roles)
}

// HandleListRoleClaims lists all role-to-model permission mappings.
func (h *CollaborationHandler) HandleListRoleClaims(w http.ResponseWriter, r *http.Request) {
	claims, err := h.service.ListRoleClaims(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to list role claims"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(claims)
}

// HandleUpdateRoleClaim updates permissions for a role on a model.
func (h *CollaborationHandler) HandleUpdateRoleClaim(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role        string   `json:"role" binding:"required"`
		ModelID     string   `json:"model_id" binding:"required"`
		Permissions []string `json:"permissions"` // Empty array means revoke
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})
		return
	}
	// In a real app, actorID would come from auth context.
	actorID := "current_admin"
	claim, err := h.service.UpdateRoleClaim(r.Context(), req.Role, req.ModelID, req.Permissions, actorID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update role claim"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(claim)
}

// HandleGrantDirectClaim grants a direct permission to a user.
func (h *CollaborationHandler) HandleGrantDirectClaim(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID     string `json:"user_id" binding:"required"`
		ModelID    string `json:"model_id" binding:"required"`
		Permission string `json:"permission" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})
		return
	}
	actorID := "current_admin"
	claim, err := h.service.GrantDirectClaim(r.Context(), req.UserID, req.ModelID, req.Permission, actorID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to grant claim"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(claim)
}

// HandleRevokeDirectClaim revokes a direct permission from a user.
func (h *CollaborationHandler) HandleRevokeDirectClaim(w http.ResponseWriter, r *http.Request) {
	claimID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid claim ID"})
		return
	}
	actorID := "current_admin"
	err = h.service.RevokeDirectClaim(r.Context(), claimID, actorID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to revoke claim"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "revoked"})
}

// HandleRequestClaimRenewal submits a renewal request for a claim.
func (h *CollaborationHandler) HandleRequestClaimRenewal(w http.ResponseWriter, r *http.Request) {
	claimID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid claim ID"})
		return
	}
	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload, 'reason' is required"})
		return
	}
	// In a real app, UserID would come from auth context.
	userID := "current_user"
	err = h.service.RequestClaimRenewal(r.Context(), claimID, userID, req.Reason)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to request claim renewal"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "renewal_requested"})
}

// HandleGetClaimLifecycleSnapshot retrieves the snapshot for the claim lifecycle dashboard.
func (h *CollaborationHandler) HandleGetClaimLifecycleSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.service.GetClaimLifecycleSnapshot(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get claim lifecycle snapshot"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}

// HandleListAccessPolicies lists all access control policies.
func (h *CollaborationHandler) HandleListAccessPolicies(w http.ResponseWriter, r *http.Request) {
	policies, err := h.service.ListAccessPolicies(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to list policies"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(policies)
}

// HandleGetClaimAwareLineage retrieves lineage details including effective access claims.
func (h *CollaborationHandler) HandleGetClaimAwareLineage(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "node_id")
	// In a real app, userID would come from auth context.
	userID := "current_user"

	lineage, err := h.service.GetClaimAwareLineage(r.Context(), nodeID, userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get claim-aware lineage"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lineage)
}

// HandleSimulatePolicyChange simulates the impact of a policy update.
func (h *CollaborationHandler) HandleSimulatePolicyChange(w http.ResponseWriter, r *http.Request) {
	var policy models.AccessControlPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid policy payload"})
		return
	}
	// In a real app, actorID would come from auth context.
	actorID := "current_admin"

	impact, err := h.service.SimulatePolicyChange(r.Context(), policy, actorID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to simulate policy change"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(impact)
}

// HandleListNotifications lists user notifications.
func (h *CollaborationHandler) HandleListNotifications(w http.ResponseWriter, r *http.Request) {
	// In a real app, UserID would come from auth context.
	userID := "current_user"
	notifications, err := h.service.ListNotificationsForUser(r.Context(), userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to list notifications"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

// HandleMarkNotificationAsRead marks a notification as read.
func (h *CollaborationHandler) HandleMarkNotificationAsRead(w http.ResponseWriter, r *http.Request) {
	notificationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid notification ID"})
		return
	}
	// In a real app, UserID would come from auth context.
	userID := "current_user"
	err = h.service.MarkNotificationAsRead(r.Context(), notificationID, userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update notification"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "read"})
}

// HandleListAccessAuditLogs retrieves access audit logs.
func (h *CollaborationHandler) HandleListAccessAuditLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := h.service.ListAccessAuditLogs(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve audit logs"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// HandleListClaimSimulations lists past claim simulations.
func (h *CollaborationHandler) HandleListClaimSimulations(w http.ResponseWriter, r *http.Request) {
	sims, err := h.service.ListClaimSimulations(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to list simulations"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sims)
}

// HandleLogAccessDeniedAttempt logs a rejected access attempt.
func (h *CollaborationHandler) HandleLogAccessDeniedAttempt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID    string `json:"user_id" binding:"required"`
		AssetType string `json:"asset_type" binding:"required"`
		AssetID   string `json:"asset_id" binding:"required"`
		Reason    string `json:"reason" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})
		return
	}
	err := h.service.LogAccessDeniedAttempt(r.Context(), req.UserID, req.AssetType, req.AssetID, req.Reason)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to log attempt"})
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// --- Alert Suppression & Escalation Handlers ---

// HandleEvaluateAlert evaluates a specific security alert.
func (h *CollaborationHandler) HandleEvaluateAlert(w http.ResponseWriter, r *http.Request) {
	var event models.SemanticChangeEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})
		return
	}
	res, err := h.service.EvaluateAlert(r.Context(), event)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to evaluate alert"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// HandleListSuppressedAlerts lists currently suppressed alerts.
func (h *CollaborationHandler) HandleListSuppressedAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.service.ListAlertsByStatus(r.Context(), "suppressed")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to list suppressed alerts"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// HandleListEscalatedAlerts lists alerts that have been escalated.
func (h *CollaborationHandler) HandleListEscalatedAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.service.ListAlertsByStatus(r.Context(), "escalated")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to list escalated alerts"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// HandleOverrideAlertStatus manually overrides an alert status.
func (h *CollaborationHandler) HandleOverrideAlertStatus(w http.ResponseWriter, r *http.Request) {
	alertID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid alert ID"})
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})
		return
	}
	// In a real app, actorID would come from auth context.
	actorID := "current_steward"
	err = h.service.OverrideAlertStatus(r.Context(), alertID, req.Status, actorID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to override alert"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "overridden"})
}

// --- Advanced Governance Handlers ---

// HandleListClaimSuggestions lists AI-driven claim optimization suggestions.
func (h *CollaborationHandler) HandleListClaimSuggestions(w http.ResponseWriter, r *http.Request) {
	// In a real app, reviewerID would come from auth context.
	reviewerID := "current_admin"
	suggestions, err := h.service.ListClaimSuggestions(r.Context(), reviewerID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to list suggestions"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}

// HandleListClaimBundles lists pre-defined permission bundles.
func (h *CollaborationHandler) HandleListClaimBundles(w http.ResponseWriter, r *http.Request) {
	bundles, err := h.service.ListClaimBundles(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to list bundles"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bundles)
}

// HandleDetectClaimDrift analyzes potential drift in effective permissions.
func (h *CollaborationHandler) HandleDetectClaimDrift(w http.ResponseWriter, r *http.Request) {
	driftData, err := h.service.DetectClaimDrift(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to detect drift"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(driftData)
}

// HandleGetGovernanceHeatmap retrieves high-level governance health metrics.
func (h *CollaborationHandler) HandleGetGovernanceHeatmap(w http.ResponseWriter, r *http.Request) {
	heatmap, err := h.service.GetGovernanceHeatmap(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get heatmap"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(heatmap)
}

// HandleListClaimConflicts lists conflicting effective permissions.
func (h *CollaborationHandler) HandleListClaimConflicts(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "user_id is required"})
		return
	}
	conflicts, err := h.service.DetectClaimConflicts(r.Context(), userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to list conflicts"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conflicts)
}

// HandleResolveClaimConflict applies a resolution strategy to a conflict.
func (h *CollaborationHandler) HandleResolveClaimConflict(w http.ResponseWriter, r *http.Request) {
	conflictID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid conflict ID"})
		return
	}
	var req struct {
		Strategy string `json:"strategy" binding:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})
		return
	}
	// In a real app, actorID would come from auth context.
	actorID := "current_admin"
	err = h.service.ResolveClaimConflict(r.Context(), conflictID, req.Strategy, actorID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to resolve conflict"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "resolved"})
}

// HandleListNotificationRules lists active notification routing rules.
func (h *CollaborationHandler) HandleListNotificationRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.service.ListNotificationRules(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to list notification rules"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

// HandleUpdateNotificationRule updates or creates a notification rule.
func (h *CollaborationHandler) HandleUpdateNotificationRule(w http.ResponseWriter, r *http.Request) {
	var rule models.NotificationRoutingRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid rule payload"})
		return
	}
	// In a real app, actorID would come from auth context.
	actorID := "current_admin"
	res, err := h.service.UpdateNotificationRule(r.Context(), rule, actorID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update notification rule"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// HandlePreviewRecipients previews who would receive a notification.
func (h *CollaborationHandler) HandlePreviewRecipients(w http.ResponseWriter, r *http.Request) {
	assetIDStr := r.URL.Query().Get("asset_id")
	assetID, err := uuid.Parse(assetIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid asset_id"})
		return
	}
	recipients, err := h.service.PreviewRecipients(r.Context(), assetID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to preview recipients"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipients)
}

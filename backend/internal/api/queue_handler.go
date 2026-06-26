package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/auth"
	"github.com/hondyman/semlayer/backend/internal/queue"
)

type QueueHandler struct {
	queueService *queue.QueueService
}

func NewQueueHandler(s *queue.QueueService) *QueueHandler {
	return &QueueHandler{queueService: s}
}

// GET /api/my-approvals
func (h *QueueHandler) GetMyApprovals(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Use real roles from context
	roles := user.Roles
	if len(roles) == 0 {
		// Fallback or empty
		roles = []string{}
	}

	var allTasks []queue.QueuedTask
	totalCount := int64(0)

	for _, role := range roles {
		limit := 100
		offset := 0
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			fmt.Sscanf(lStr, "%d", &limit)
		}
		if oStr := r.URL.Query().Get("offset"); oStr != "" {
			fmt.Sscanf(oStr, "%d", &offset)
		}

		tasks, count, _ := h.queueService.GetQueuedTasks(
			r.Context(),
			user.TenantID,
			user.ID, // Pass viewing user
			role,
			queue.QueueRequest{
				Role:   role,
				Status: r.URL.Query().Get("status"),
				SortBy: r.URL.Query().Get("sort"),
				Limit:  limit,
				Offset: offset,
			},
		)

		allTasks = append(allTasks, tasks...)
		totalCount += count
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks": allTasks,
		"count": totalCount,
	})
}

// POST /api/instances/{instanceId}/assign-to-me
func (h *QueueHandler) AssignToMe(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	instanceID := chi.URLParam(r, "instanceId")

	if err := h.queueService.AssignTaskToUser(r.Context(), user.TenantID, instanceID, user.ID); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(map[string]string{"status": "assigned"})
}

// POST /api/instances/{instanceId}/unassign
func (h *QueueHandler) Unassign(w http.ResponseWriter, r *http.Request) {
	instanceID := chi.URLParam(r, "instanceId")

	if err := h.queueService.UnassignTask(r.Context(), instanceID); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(map[string]string{"status": "unassigned"})
}

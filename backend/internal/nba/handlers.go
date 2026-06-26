package nba

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// NBAHandler handles HTTP requests for NBA system
type NBAHandler struct {
	db             *sqlx.DB
	signalDetector *SignalDetector
}

// NewNBAHandler creates a new NBA handler
func NewNBAHandler(db *sqlx.DB) *NBAHandler {
	return &NBAHandler{
		db:             db,
		signalDetector: NewSignalDetector(db),
	}
}

// GetRecommendations returns NBA recommendations for an advisor
// GET /api/nba/recommendations
func (h *NBAHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	queryData := r.URL.Query()
	advisorID := queryData.Get("advisor_id")
	status := queryData.Get("status")
	limitStr := queryData.Get("limit")

	if status == "" {
		status = "PENDING"
	}
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query GetNBARecommendations($tenantId: uuid!, $advisorId: uuid!, $status: String!, $limit: Int!) {
	//   nba_recommendations(
	//     where: {
	//       tenant_id: {_eq: $tenantId},
	//       advisor_id: {_eq: $advisorId},
	//       status: {_eq: $status}
	//     },
	//     order_by: [{overall_score: desc}, {recommended_at: desc}],
	//     limit: $limit
	//   ) {
	//     recommendation_id
	//     client_id
	//     advisor_id
	//     action_id
	//     confidence_score
	//     urgency_score
	//     expected_value
	//     success_probability
	//     overall_score
	//     reasoning
	//     supporting_data
	//     status
	//     recommended_at
	//     expires_at
	//   }
	// }
	query := `
		SELECT 
			recommendation_id,
			client_id,
			advisor_id,
			action_id,
			confidence_score,
			urgency_score,
			expected_value,
			success_probability,
			overall_score,
			reasoning,
			supporting_data,
			status,
			recommended_at,
			expires_at
		FROM nba_recommendations
		WHERE tenant_id = $1
		AND advisor_id = $2
		AND status = $3
		ORDER BY overall_score DESC, recommended_at DESC
		LIMIT $4
	`

	type Recommendation struct {
		RecommendationID   uuid.UUID  `json:"recommendation_id" db:"recommendation_id"`
		ClientID           uuid.UUID  `json:"client_id" db:"client_id"`
		AdvisorID          uuid.UUID  `json:"advisor_id" db:"advisor_id"`
		ActionID           uuid.UUID  `json:"action_id" db:"action_id"`
		ConfidenceScore    float64    `json:"confidence_score" db:"confidence_score"`
		UrgencyScore       float64    `json:"urgency_score" db:"urgency_score"`
		ExpectedValue      float64    `json:"expected_value" db:"expected_value"`
		SuccessProbability float64    `json:"success_probability" db:"success_probability"`
		OverallScore       float64    `json:"overall_score" db:"overall_score"`
		Reasoning          string     `json:"reasoning" db:"reasoning"`
		SupportingData     string     `json:"supporting_data" db:"supporting_data"`
		Status             string     `json:"status" db:"status"`
		RecommendedAt      time.Time  `json:"recommended_at" db:"recommended_at"`
		ExpiresAt          *time.Time `json:"expires_at" db:"expires_at"`
	}

	var recommendations []Recommendation
	err := h.db.SelectContext(r.Context(), &recommendations, query, tenantID, advisorID, status, limit)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch recommendations"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"recommendations": recommendations,
		"total":           len(recommendations),
	})
}

// ExecuteRecommendation marks a recommendation as being executed
// POST /api/nba/recommendations/{id}/execute
func (h *NBAHandler) ExecuteRecommendation(w http.ResponseWriter, r *http.Request) {
	recommendationID := chi.URLParam(r, "id")

	// TODO(hasura-migration): Replace SQL UPDATE with Hasura GraphQL mutation
	// ... (GraphQL mutation comment omitted for brevity)
	query := `
		UPDATE nba_recommendations
		SET status = 'EXECUTING',
		    executed_at = NOW(),
		    updated_at = NOW()
		WHERE recommendation_id = $1
		AND status = 'PENDING'
		RETURNING recommendation_id
	`

	var id uuid.UUID
	err := h.db.GetContext(r.Context(), &id, query, recommendationID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Recommendation not found or already executed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":           "Recommendation marked as executing",
		"recommendation_id": id,
	})
}

// DismissRecommendation marks a recommendation as dismissed
// POST /api/nba/recommendations/{id}/dismiss
func (h *NBAHandler) DismissRecommendation(w http.ResponseWriter, r *http.Request) {
	recommendationID := chi.URLParam(r, "id")

	var req struct {
		Reason string `json:"reason"`
		Notes  string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// TODO(hasura-migration): Replace SQL UPDATE with Hasura GraphQL mutation
	// Example GraphQL mutation:
	// mutation DismissRecommendation($id: uuid!, $reason: String!, $notes: String) {
	//   update_nba_recommendations(
	//     where: {
	//       recommendation_id: {_eq: $id},
	//       status: {_in: ["PENDING", "VIEWED"]}
	//     },
	//     _set: {
	//       status: "DISMISSED",
	//       dismissed_at: "now()",
	//       dismissal_reason: $reason,
	//       dismissal_notes: $notes,
	//       updated_at: "now()"
	//     }
	//   ) {
	//     affected_rows
	//     returning {
	//       recommendation_id
	//     }
	//   }
	// }
	query := `
		UPDATE nba_recommendations
		SET status = 'DISMISSED',
		    dismissed_at = NOW(),
		    dismissal_reason = $2,
		    dismissal_notes = $3,
		    updated_at = NOW()
		WHERE recommendation_id = $1
		AND status IN ('PENDING', 'VIEWED')
		RETURNING recommendation_id
	`

	var id uuid.UUID
	err := h.db.GetContext(r.Context(), &id, query, recommendationID, req.Reason, req.Notes)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Recommendation not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":           "Recommendation dismissed",
		"recommendation_id": id,
	})
}

// GetActionCatalog returns available NBA actions
// GET /api/nba/actions
func (h *NBAHandler) GetActionCatalog(w http.ResponseWriter, r *http.Request) {
	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query GetActionCatalog {
	//   nba_action_catalog(
	//     where: {active: {_eq: true}},
	//     order_by: {action_name: asc}
	//   ) {
	//     action_id
	//     action_code
	//     action_name
	//     action_category
	//     description
	//     default_channel
	//     estimated_duration_minutes
	//     estimated_revenue_impact
	//     template_content
	//   }
	// }
	query := `
		SELECT 
			action_id,
			action_code,
			action_name,
			action_category,
			description,
			default_channel,
			estimated_duration_minutes,
			estimated_revenue_impact,
			template_content
		FROM nba_action_catalog
		WHERE active = true
		ORDER BY action_name
	`

	type Action struct {
		ActionID                 uuid.UUID `json:"action_id" db:"action_id"`
		ActionCode               string    `json:"action_code" db:"action_code"`
		ActionName               string    `json:"action_name" db:"action_name"`
		ActionCategory           string    `json:"action_category" db:"action_category"`
		Description              string    `json:"description" db:"description"`
		DefaultChannel           string    `json:"default_channel" db:"default_channel"`
		EstimatedDurationMinutes int       `json:"estimated_duration_minutes" db:"estimated_duration_minutes"`
		EstimatedRevenueImpact   float64   `json:"estimated_revenue_impact" db:"estimated_revenue_impact"`
		TemplateContent          string    `json:"template_content" db:"template_content"`
	}

	var actions []Action
	err := h.db.SelectContext(r.Context(), &actions, query)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch actions"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"actions": actions,
		"total":   len(actions),
	})
}

// GetNBAStats returns NBA statistics for advisor dashboard
// GET /api/nba/stats
func (h *NBAHandler) GetNBAStats(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	advisorID := r.URL.Query().Get("advisor_id")

	type Stats struct {
		PendingCount      int     `json:"pending_count" db:"pending_count"`
		CriticalCount     int     `json:"critical_count" db:"critical_count"`
		TotalPotentialRev float64 `json:"total_potential_revenue" db:"total_potential_revenue"`
		AvgSuccessRate    float64 `json:"avg_success_rate" db:"avg_success_rate"`
		CompletedToday    int     `json:"completed_today" db:"completed_today"`
	}

	// TODO(hasura-migration): Replace SQL aggregation query with Hasura GraphQL query
	// Note: Hasura aggregations require aggregate functions; complex FILTER logic may need multiple queries
	// Example GraphQL query:
	// query GetNBAStats($tenantId: uuid!, $advisorId: uuid!) {
	//   pending: nba_recommendations_aggregate(
	//     where: {
	//       tenant_id: {_eq: $tenantId},
	//       advisor_id: {_eq: $advisorId},
	//       status: {_eq: "PENDING"}
	//     }
	//   ) {
	//     aggregate {
	//       count
	//       sum { expected_value }
	//       avg { success_probability }
	//     }
	//   }
	//   critical: nba_recommendations_aggregate(
	//     where: {
	//       tenant_id: {_eq: $tenantId},
	//       advisor_id: {_eq: $advisorId},
	//       status: {_eq: "PENDING"},
	//       urgency_score: {_gt: 0.8}
	//     }
	//   ) {
	//     aggregate { count }
	//   }
	//   completed_today: nba_recommendations_aggregate(
	//     where: {
	//       tenant_id: {_eq: $tenantId},
	//       advisor_id: {_eq: $advisorId},
	//       status: {_eq: "COMPLETED"},
	//       completed_at: {_gte: "today"}
	//     }
	//   ) {
	//     aggregate { count }
	//   }
	// }
	query := `
		SELECT 
			COUNT(*) FILTER (WHERE status = 'PENDING') as pending_count,
			COUNT(*) FILTER (WHERE status = 'PENDING' AND urgency_score > 0.8) as critical_count,
			COALESCE(SUM(expected_value) FILTER (WHERE status = 'PENDING'), 0) as total_potential_revenue,
			COALESCE(AVG(success_probability) FILTER (WHERE status = 'PENDING'), 0) as avg_success_rate,
			COUNT(*) FILTER (WHERE status = 'COMPLETED' AND DATE(completed_at) = CURRENT_DATE) as completed_today
		FROM nba_recommendations
		WHERE tenant_id = $1
		AND advisor_id = $2
	`

	var stats Stats
	err := h.db.GetContext(r.Context(), &stats, query, tenantID, advisorID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch stats"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// RegisterRoutes registers all NBA routes
func (h *NBAHandler) RegisterRoutes(r chi.Router) {
	r.Route("/nba", func(r chi.Router) {
		r.Get("/recommendations", h.GetRecommendations)
		r.Post("/recommendations/{id}/execute", h.ExecuteRecommendation)
		r.Post("/recommendations/{id}/dismiss", h.DismissRecommendation)
		r.Get("/actions", h.GetActionCatalog)
		r.Get("/stats", h.GetNBAStats)
	})
}

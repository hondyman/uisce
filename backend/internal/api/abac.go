package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ABACPolicy represents an access control policy
type ABACPolicy struct {
	ID               string                 `json:"id" db:"id"`
	TenantID         string                 `json:"tenant_id" db:"tenant_id"`
	DatasourceID     string                 `json:"datasource_id" db:"datasource_id"`
	Name             string                 `json:"name" db:"name"`
	Description      string                 `json:"description" db:"description"`
	Effect           string                 `json:"effect" db:"effect"`     // "allow" or "deny"
	Priority         int                    `json:"priority" db:"priority"` // Higher = evaluated first
	Enabled          bool                   `json:"enabled" db:"enabled"`
	SubjectRules     map[string]interface{} `json:"subject_rules" db:"subject_rules"`         // JSON
	ActionRules      map[string]interface{} `json:"action_rules" db:"action_rules"`           // JSON
	ResourceRules    map[string]interface{} `json:"resource_rules" db:"resource_rules"`       // JSON
	EnvironmentRules map[string]interface{} `json:"environment_rules" db:"environment_rules"` // JSON
	CreatedBy        string                 `json:"created_by" db:"created_by"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
}

// ABACEvaluationRequest is sent by React components
type ABACEvaluationRequest struct {
	Subject      string            `json:"subject"`  // user ID or role
	Action       string            `json:"action"`   // action to evaluate
	Resource     string            `json:"resource"` // resource type
	Context      map[string]string `json:"context"`  // additional context
	TenantID     string            `json:"tenant_id"`
	DatasourceID string            `json:"datasource_id"`
}

// ABACEvaluationResult is returned to React components
type ABACEvaluationResult struct {
	Decision  string    `json:"decision"` // "allow" or "deny"
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
	PolicyID  string    `json:"policy_id"`
}

// ABACDelegation represents a temporary role delegation
type ABACDelegation struct {
	ID           string    `json:"id" db:"id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	DatasourceID string    `json:"datasource_id" db:"datasource_id"`
	FromUserID   string    `json:"from_user_id" db:"from_user_id"`
	ToUserID     string    `json:"to_user_id" db:"to_user_id"`
	PolicyID     string    `json:"policy_id" db:"policy_id"`
	ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// AuditLogEntry represents an ABAC decision log
type AuditLogEntry struct {
	ID           string    `json:"id" db:"id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	DatasourceID string    `json:"datasource_id" db:"datasource_id"`
	Actor        string    `json:"actor" db:"actor"`
	Action       string    `json:"action" db:"action"`
	Resource     string    `json:"resource" db:"resource"`
	Decision     string    `json:"decision" db:"decision"` // "allow" or "deny"
	Reason       string    `json:"reason" db:"reason"`
	IPAddress    string    `json:"ip_address" db:"ip_address"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
}

// RegisterABACRoutes registers all ABAC endpoints
func RegisterABACRoutes(r chi.Router, db *sql.DB) {
	abacAPI := &ABACAPI{db: db}

	r.Route("/api/abac", func(r chi.Router) {
		// Policy endpoints
		r.Post("/policies", abacAPI.createPolicy)
		r.Get("/policies", abacAPI.listPolicies)
		r.Route("/policies/{id}", func(r chi.Router) {
			r.Put("/", abacAPI.updatePolicy)
			r.Delete("/", abacAPI.deletePolicy)
		})

		// Evaluation endpoint
		r.Post("/evaluate", abacAPI.evaluatePolicy)

		// Delegation endpoints
		r.Post("/delegations", abacAPI.createDelegation)
		r.Get("/delegations", abacAPI.listDelegations)
		r.Delete("/delegations/{id}", abacAPI.revokeDelegation)

		// Audit endpoints
		r.Get("/audit", abacAPI.listAuditLogs)
	})

	log.Println("ABAC routes registered")
}

type ABACAPI struct {
	db *sql.DB
}

// createPolicy creates a new ABAC policy
func (a *ABACAPI) createPolicy(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing tenant scope"})
		return
	}

	var req struct {
		Name             string                 `json:"name"`
		Description      string                 `json:"description"`
		Effect           string                 `json:"effect"`
		Priority         int                    `json:"priority"`
		Enabled          bool                   `json:"enabled"`
		SubjectRules     map[string]interface{} `json:"subject_rules"`
		ActionRules      map[string]interface{} `json:"action_rules"`
		ResourceRules    map[string]interface{} `json:"resource_rules"`
		EnvironmentRules map[string]interface{} `json:"environment_rules"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Validate required fields
	if req.Name == "" || req.Effect == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Name and Effect are required"})
		return
	}

	if req.Effect != "allow" && req.Effect != "deny" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Effect must be 'allow' or 'deny'"})
		return
	}

	policyID := uuid.New().String()
	userID := r.Header.Get("X-User-ID") // Set by auth middleware or should be in header

	subjectRulesJSON, _ := json.Marshal(req.SubjectRules)
	actionRulesJSON, _ := json.Marshal(req.ActionRules)
	resourceRulesJSON, _ := json.Marshal(req.ResourceRules)
	environmentRulesJSON, _ := json.Marshal(req.EnvironmentRules)

	query := `
		INSERT INTO abac_policies 
		(id, tenant_id, datasource_id, name, description, effect, priority, enabled, 
		 subject_rules, action_rules, resource_rules, environment_rules, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW())
	`

	_, err := a.db.ExecContext(r.Context(), query, policyID, tenantID, datasourceID, req.Name, req.Description,
		req.Effect, req.Priority, req.Enabled,
		string(subjectRulesJSON), string(actionRulesJSON),
		string(resourceRulesJSON), string(environmentRulesJSON), userID)

	if err != nil {
		log.Printf("Error creating policy: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create policy"})
		return
	}

	// Log to audit trail
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}
	a.logAuditEvent(tenantID, datasourceID, userID, "create_policy",
		fmt.Sprintf("policy:%s", policyID), "allow", "", clientIP)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"id":      policyID,
		"message": "Policy created successfully",
	})
}

// listPolicies lists all policies for the tenant
func (a *ABACAPI) listPolicies(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing tenant scope"})
		return
	}

	query := `
		SELECT id, tenant_id, datasource_id, name, description, effect, priority, enabled,
			   subject_rules, action_rules, resource_rules, environment_rules,
			   created_by, created_at, updated_at
		FROM abac_policies
		WHERE tenant_id = $1 AND datasource_id = $2 AND enabled = true
		ORDER BY priority DESC, created_at DESC
	`

	rows, err := a.db.QueryContext(r.Context(), query, tenantID, datasourceID)
	if err != nil {
		log.Printf("Error querying policies: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch policies"})
		return
	}
	defer rows.Close()

	policies := []ABACPolicy{}
	for rows.Next() {
		var p ABACPolicy
		var subjectJSON, actionJSON, resourceJSON, environmentJSON string

		err := rows.Scan(&p.ID, &p.TenantID, &p.DatasourceID, &p.Name, &p.Description,
			&p.Effect, &p.Priority, &p.Enabled, &subjectJSON, &actionJSON,
			&resourceJSON, &environmentJSON, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)

		if err != nil {
			log.Printf("Error scanning policy: %v", err)
			continue
		}

		json.Unmarshal([]byte(subjectJSON), &p.SubjectRules)
		json.Unmarshal([]byte(actionJSON), &p.ActionRules)
		json.Unmarshal([]byte(resourceJSON), &p.ResourceRules)
		json.Unmarshal([]byte(environmentJSON), &p.EnvironmentRules)

		policies = append(policies, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"policies": policies})
}

// updatePolicy updates an existing policy
func (a *ABACAPI) updatePolicy(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	policyID := chi.URLParam(r, "id")

	if tenantID == "" || datasourceID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing tenant scope"})
		return
	}

	var req struct {
		Name             string                 `json:"name"`
		Description      string                 `json:"description"`
		Effect           string                 `json:"effect"`
		Priority         int                    `json:"priority"`
		Enabled          bool                   `json:"enabled"`
		SubjectRules     map[string]interface{} `json:"subject_rules"`
		ActionRules      map[string]interface{} `json:"action_rules"`
		ResourceRules    map[string]interface{} `json:"resource_rules"`
		EnvironmentRules map[string]interface{} `json:"environment_rules"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	userID := r.Header.Get("X-User-ID")

	subjectRulesJSON, _ := json.Marshal(req.SubjectRules)
	actionRulesJSON, _ := json.Marshal(req.ActionRules)
	resourceRulesJSON, _ := json.Marshal(req.ResourceRules)
	environmentRulesJSON, _ := json.Marshal(req.EnvironmentRules)

	query := `
		UPDATE abac_policies
		SET name = $1, description = $2, effect = $3, priority = $4, enabled = $5,
			subject_rules = $6, action_rules = $7, resource_rules = $8,
			environment_rules = $9, updated_at = NOW()
		WHERE id = $10 AND tenant_id = $11 AND datasource_id = $12
	`

	result, err := a.db.ExecContext(r.Context(), query, req.Name, req.Description, req.Effect, req.Priority,
		req.Enabled, string(subjectRulesJSON), string(actionRulesJSON),
		string(resourceRulesJSON), string(environmentRulesJSON), policyID, tenantID, datasourceID)

	if err != nil {
		log.Printf("Error updating policy: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update policy"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Policy not found"})
		return
	}

	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}
	a.logAuditEvent(tenantID, datasourceID, userID, "update_policy",
		fmt.Sprintf("policy:%s", policyID), "allow", "", clientIP)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Policy updated successfully"})
}

// deletePolicy deletes a policy
func (a *ABACAPI) deletePolicy(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	policyID := chi.URLParam(r, "id")

	if tenantID == "" || datasourceID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing tenant scope"})
		return
	}

	userID := r.Header.Get("X-User-ID")

	query := `
		DELETE FROM abac_policies
		WHERE id = $1 AND tenant_id = $2 AND datasource_id = $3
	`

	result, err := a.db.ExecContext(r.Context(), query, policyID, tenantID, datasourceID)
	if err != nil {
		log.Printf("Error deleting policy: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete policy"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Policy not found"})
		return
	}

	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}
	a.logAuditEvent(tenantID, datasourceID, userID, "delete_policy",
		fmt.Sprintf("policy:%s", policyID), "allow", "", clientIP)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Policy deleted successfully"})
}

// evaluatePolicy evaluates an access control decision
func (a *ABACAPI) evaluatePolicy(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing tenant scope"})
		return
	}

	var req ABACEvaluationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Query policies ordered by priority
	query := `
		SELECT id, effect FROM abac_policies
		WHERE tenant_id = $1 AND datasource_id = $2 AND enabled = true
		ORDER BY priority DESC
	`

	rows, err := a.db.QueryContext(r.Context(), query, tenantID, datasourceID)
	if err != nil {
		log.Printf("Error evaluating policy: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Evaluation failed"})
		return
	}
	defer rows.Close()

	// Default decision: deny
	decision := "deny"
	policyID := ""
	reason := "No matching policy"

	// Check policies in priority order
	for rows.Next() {
		var id, effect string
		if err := rows.Scan(&id, &effect); err != nil {
			continue
		}

		// Simple evaluation: if policy matches request, use the effect
		// In production, implement full policy matching logic
		decision = effect
		policyID = id
		reason = fmt.Sprintf("Matched policy %s", id)
		break
	}

	// Log the evaluation decision
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}
	a.logAuditEvent(tenantID, datasourceID, req.Subject, req.Action,
		req.Resource, decision, reason, clientIP)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ABACEvaluationResult{
		Decision:  decision,
		Reason:    reason,
		Timestamp: time.Now(),
		PolicyID:  policyID,
	})
}

// createDelegation creates a temporary role delegation
func (a *ABACAPI) createDelegation(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing tenant scope"})
		return
	}

	var req struct {
		FromUserID string    `json:"from_user_id"`
		ToUserID   string    `json:"to_user_id"`
		PolicyID   string    `json:"policy_id"`
		ExpiresAt  time.Time `json:"expires_at"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	delegationID := uuid.New().String()

	query := `
		INSERT INTO abac_delegations (id, tenant_id, datasource_id, from_user_id, to_user_id, policy_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`

	_, err := a.db.ExecContext(r.Context(), query, delegationID, tenantID, datasourceID, req.FromUserID, req.ToUserID, req.PolicyID, req.ExpiresAt)
	if err != nil {
		log.Printf("Error creating delegation: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create delegation"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"id":      delegationID,
		"message": "Delegation created successfully",
	})
}

// listDelegations lists active delegations
func (a *ABACAPI) listDelegations(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing tenant scope"})
		return
	}

	query := `
		SELECT id, tenant_id, datasource_id, from_user_id, to_user_id, policy_id, expires_at, created_at
		FROM abac_delegations
		WHERE tenant_id = $1 AND datasource_id = $2 AND expires_at > NOW()
		ORDER BY created_at DESC
	`

	rows, err := a.db.QueryContext(r.Context(), query, tenantID, datasourceID)
	if err != nil {
		log.Printf("Error querying delegations: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch delegations"})
		return
	}
	defer rows.Close()

	delegations := []ABACDelegation{}
	for rows.Next() {
		var d ABACDelegation
		if err := rows.Scan(&d.ID, &d.TenantID, &d.DatasourceID, &d.FromUserID, &d.ToUserID,
			&d.PolicyID, &d.ExpiresAt, &d.CreatedAt); err != nil {
			continue
		}
		delegations = append(delegations, d)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"delegations": delegations})
}

// revokeDelegation revokes a delegation
func (a *ABACAPI) revokeDelegation(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	delegationID := chi.URLParam(r, "id")

	if tenantID == "" || datasourceID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing tenant scope"})
		return
	}

	query := `
		DELETE FROM abac_delegations
		WHERE id = $1 AND tenant_id = $2 AND datasource_id = $3
	`

	result, err := a.db.ExecContext(r.Context(), query, delegationID, tenantID, datasourceID)
	if err != nil {
		log.Printf("Error revoking delegation: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to revoke delegation"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Delegation not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Delegation revoked successfully"})
}

// listAuditLogs lists audit log entries
func (a *ABACAPI) listAuditLogs(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing tenant scope"})
		return
	}

	// Parse filters
	days := r.URL.Query().Get("days")
	if days == "" {
		days = "30"
	}
	action := r.URL.Query().Get("action")
	result := r.URL.Query().Get("result")

	query := `
		SELECT id, tenant_id, datasource_id, actor, action, resource, decision, reason, ip_address, timestamp
		FROM audit_log
		WHERE tenant_id = $1 AND datasource_id = $2
		AND timestamp > NOW() - INTERVAL '1 day' * $3::integer
	`

	params := []interface{}{tenantID, datasourceID, days}
	paramCount := 3

	if action != "" {
		paramCount++
		query += fmt.Sprintf(" AND action = $%d", paramCount)
		params = append(params, action)
	}

	if result != "" {
		paramCount++
		query += fmt.Sprintf(" AND decision = $%d", paramCount)
		params = append(params, result)
	}

	query += " ORDER BY timestamp DESC LIMIT 1000"

	rows, err := a.db.QueryContext(r.Context(), query, params...)
	if err != nil {
		log.Printf("Error querying audit logs: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch audit logs"})
		return
	}
	defer rows.Close()

	logs := []AuditLogEntry{}
	for rows.Next() {
		var entry AuditLogEntry
		if err := rows.Scan(&entry.ID, &entry.TenantID, &entry.DatasourceID, &entry.Actor,
			&entry.Action, &entry.Resource, &entry.Decision, &entry.Reason, &entry.IPAddress, &entry.Timestamp); err != nil {
			continue
		}
		logs = append(logs, entry)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"audit_logs": logs})
}

// logAuditEvent logs an access control decision to the audit trail
func (a *ABACAPI) logAuditEvent(tenantID, datasourceID, actor, action, resource, decision, reason, ipAddress string) {
	query := `
		INSERT INTO audit_log (id, tenant_id, datasource_id, actor, action, resource, decision, reason, ip_address, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`

	auditID := uuid.New().String()
	_, err := a.db.Exec(query, auditID, tenantID, datasourceID, actor, action, resource, decision, reason, ipAddress)
	if err != nil {
		log.Printf("Error logging audit event: %v", err)
	}
}

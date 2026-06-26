package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/infrastructure"
	"github.com/hondyman/semlayer/backend/internal/metadata"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// WriteHandler handles generic writes driven by the metadata graph
type WriteHandler struct {
	GraphService *metadata.GraphService
	DB           *sqlx.DB
	Ignite       *infrastructure.IgniteClient
	AbacService  *services.AbacService
}

// NewWriteHandler creates a new WriteHandler
func NewWriteHandler(gs *metadata.GraphService, db *sqlx.DB, ignite *infrastructure.IgniteClient, abac *services.AbacService) *WriteHandler {
	return &WriteHandler{
		GraphService: gs,
		DB:           db,
		Ignite:       ignite,
		AbacService:  abac,
	}
}

// HandleGenericWrite processes POST /api/object/{ObjectType}
func (h *WriteHandler) HandleGenericWrite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	objectType := chi.URLParam(r, "ObjectType")
	// Header based tenant isolation (simplified for this implementation)
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = "default" // Fallback or Error
	}

	// 1. Parse Payload
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// 2. Lookup Metadata (Single Source of Truth)
	node, err := h.GraphService.GetNodeByName(ctx, tenantID, objectType)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching metadata: %v", err), http.StatusInternalServerError)
		return
	}
	if node == nil {
		http.Error(w, fmt.Sprintf("Object type '%s' not found in metadata", objectType), http.StatusNotFound)
		return
	}

	// 2.5 Security: ABAC Enforcement
	// Subject: User (from context/header - simplified here)
	// Action: "write"
	// Resource: ObjectType + ID (if present)
	// Environment: IP, Time, Tenant
	if h.AbacService != nil {
		// Mock subject/env for now, in real world extract from JWT/Request
		subjectAttrs := map[string]any{
			"user_id": r.Header.Get("X-User-ID"),
			"role":    r.Header.Get("X-User-Role"),
		}
		if subjectAttrs["user_id"] == "" {
			subjectAttrs["user_id"] = "anonymous"
		}

		resourceAttrs := map[string]any{
			"type":      objectType,
			"tenant_id": tenantID,
		}
		if idVal, ok := payload["id"].(string); ok {
			resourceAttrs["id"] = idVal
		}

		envAttrs := map[string]any{
			"ip":        r.RemoteAddr, // Note: might need stripping port
			"timestamp": time.Now().Unix(),
		}

		allowed, reason, err := h.AbacService.EvaluateAccess(ctx, subjectAttrs, "write", resourceAttrs, envAttrs)
		if err != nil {
			// Fail secure on error
			fmt.Printf("ABAC Evaluation Error: %v\n", err)
			http.Error(w, "Access Denied (System Error)", http.StatusForbidden)
			return
		}
		if !allowed {
			fmt.Printf("ABAC Deny: %s\n", reason)
			http.Error(w, fmt.Sprintf("Access Denied: %s", reason), http.StatusForbidden)
			return
		}
	}

	// 3. Validation Logic (Metadata-First)
	validationRules, err := h.GraphService.GetValidationRules(ctx, node.ID)
	if err != nil {
		http.Error(w, "Error fetching validation rules", http.StatusInternalServerError)
		return
	}

	for _, rule := range validationRules {
		if strings.HasPrefix(rule, "regex:") {
			parts := strings.SplitN(rule, ":", 3)
			if len(parts) == 3 {
				field := parts[1]
				pattern := parts[2]
				if val, ok := payload[field].(string); ok {
					matched, _ := regexp.MatchString(pattern, val)
					if !matched {
						http.Error(w, fmt.Sprintf("Validation failed for field '%s': does not match pattern", field), http.StatusBadRequest)
						return
					}
				}
			}
		}
	}

	// 4. Persistence: Schema-less JSONB Store
	// Pattern: Ignite (Cache) -> Postgres (JSONB Persistent Store) -> Debezium -> StarRocks

	// Determine ID
	var objectID string
	if idVal, ok := payload["id"].(string); ok && idVal != "" {
		objectID = idVal
	} else {
		// New object, generate generic ID if not provided
		// In a real system you might want standard UUIDs
		// For now, let's assume client MUST provide ID or we gen one
		objectID = fmt.Sprintf("%s-%d", objectType, time.Now().UnixNano())
		payload["id"] = objectID
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "Failed to marshal payload", http.StatusInternalServerError)
		return
	}

	// 5. Write-Through: Ignite Cache
	if h.Ignite != nil {
		err := h.Ignite.Put(objectType, objectID, payload)
		if err != nil {
			fmt.Printf("Warning: Ignite Put failed: %v\n", err)
		}
	}

	// 6. Persistence: Postgres "persistent_store" table
	// We use ON CONFLICT to handle both Insert and Update (Upsert)
	query := `
		INSERT INTO persistent_store (id, object_type, data, tenant_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE 
		SET data = EXCLUDED.data, 
		    object_type = EXCLUDED.object_type,
		    updated_at = CURRENT_TIMESTAMP
	`
	_, err = h.DB.ExecContext(ctx, query, objectID, objectType, payloadJSON, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database persist error: %v", err), http.StatusInternalServerError)
		return
	}

	// 7. Invalidate/Refresh Cube Cache
	go func() {
		cubeURL := "http://cube:4000/cubejs-api/v1/pre-aggregations/refresh"
		req, _ := http.NewRequest("POST", cubeURL, nil)
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Warning: Failed to refresh Cube cache: %v\n", err)
			return
		}
		defer resp.Body.Close()
	}()

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status":"success"}`))
}

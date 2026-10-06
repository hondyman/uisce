package surveillance

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

// SurveillanceFindingRecord models a row from compliance.compliance_surveillance_finding
type SurveillanceFindingRecord struct {
	ID                  uuid.UUID       `json:"id"`
	TenantID            uuid.UUID       `json:"tenant_id"`
	DetectorType        string          `json:"detector_type"`
	Severity            string          `json:"severity"`
	Status              string          `json:"status"`
	DedupKey            string          `json:"dedup_key"`
	Title               string          `json:"title"`
	Description         string          `json:"description"`
	EntityID            *uuid.UUID      `json:"entity_id,omitempty"`
	EntityType          string          `json:"entity_type"`
	Metadata            json.RawMessage `json:"metadata"`
	DetectedAt          time.Time       `json:"detected_at"`
	ActivityWindowStart time.Time       `json:"activity_window_start"`
	ActivityWindowEnd   time.Time       `json:"activity_window_end"`
	AssignedTo          *string         `json:"assigned_to,omitempty"`
	ResolutionNotes     *string         `json:"resolution_notes,omitempty"`
	ResolvedBy          *string         `json:"resolved_by,omitempty"`
	ResolvedAt          *time.Time      `json:"resolved_at,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// SurveillanceEventRecord models a row from compliance.compliance_surveillance_event
type SurveillanceEventRecord struct {
	ID        uuid.UUID       `json:"id"`
	FindingID uuid.UUID       `json:"finding_id"`
	TenantID  uuid.UUID       `json:"tenant_id"`
	EventType string          `json:"event_type"`
	Actor     string          `json:"actor"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

// Handler handles HTTP requests for post-trade surveillance findings queue.
type Handler struct {
	db *sql.DB
}

// NewHandler creates a new surveillance Handler.
func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

// RegisterRoutes registers routes on the provided chi Router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/compliance/surveillance/findings", func(r chi.Router) {
		r.Get("/", h.HandleListFindings)
		r.Get("/unaddressed", h.HandleListUnaddressedFindings)
		r.Get("/{id}", h.HandleGetFinding)
		r.Post("/{id}/status", h.HandleUpdateFindingStatus)
	})
}

func (h *Handler) getTenantID(r *http.Request) (uuid.UUID, error) {
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil {
		if claims.TenantID != "" {
			if u, err := uuid.Parse(claims.TenantID); err == nil {
				return u, nil
			}
		}
	}
	if tenantHdr := r.Header.Get("X-Tenant-ID"); tenantHdr != "" {
		return uuid.Parse(tenantHdr)
	}
	return uuid.MustParse("99e99e99-99e9-49e9-89e9-99e99e99e999"), nil
}

func (h *Handler) getActor(r *http.Request) string {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims != nil && claims.Email != "" {
		return claims.Email
	}
	if userHdr := r.Header.Get("X-User-Email"); userHdr != "" {
		return userHdr
	}
	return "compliance_officer"
}

// HandleListUnaddressedFindings returns active open findings from the operational view
func (h *Handler) HandleListUnaddressedFindings(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		http.Error(w, "invalid tenant id", http.StatusUnauthorized)
		return
	}

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT finding_id, tenant_id, detector_type, severity, status, dedup_key,
		       title, description, entity_id, entity_type, metadata,
		       detected_at, activity_window_start, activity_window_end, assigned_to,
		       created_at, updated_at, age_hours
		FROM compliance.v_unaddressed_surveillance_findings
		WHERE tenant_id = $1
		ORDER BY detected_at DESC
	`, tenantID)
	if err != nil {
		http.Error(w, "query unaddressed findings failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var findings []map[string]any
	for rows.Next() {
		var (
			id, tID                                                  uuid.UUID
			detectorType, severity, status, dedupKey, title, desc    string
			entityID                                                 sql.NullString
			entityType                                               string
			metaJSON                                                 []byte
			detectedAt, windowStart, windowEnd, createdAt, updatedAt time.Time
			assignedTo                                               sql.NullString
			ageHours                                                 float64
		)
		err := rows.Scan(
			&id, &tID, &detectorType, &severity, &status, &dedupKey,
			&title, &desc, &entityID, &entityType, &metaJSON,
			&detectedAt, &windowStart, &windowEnd, &assignedTo,
			&createdAt, &updatedAt, &ageHours,
		)
		if err == nil {
			item := map[string]any{
				"id":                    id,
				"tenant_id":             tID,
				"detector_type":         detectorType,
				"severity":              severity,
				"status":                status,
				"dedup_key":             dedupKey,
				"title":                 title,
				"description":           desc,
				"entity_type":           entityType,
				"metadata":              json.RawMessage(metaJSON),
				"detected_at":           detectedAt,
				"activity_window_start": windowStart,
				"activity_window_end":   windowEnd,
				"created_at":            createdAt,
				"updated_at":            updatedAt,
				"age_hours":             ageHours,
			}
			if entityID.Valid {
				item["entity_id"] = entityID.String
			}
			if assignedTo.Valid {
				item["assigned_to"] = assignedTo.String
			}
			findings = append(findings, item)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":        findings,
		"total_count": len(findings),
	})
}

// HandleListFindings returns paginated findings with filters
func (h *Handler) HandleListFindings(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		http.Error(w, "invalid tenant id", http.StatusUnauthorized)
		return
	}

	q := r.URL.Query()
	detectorType := q.Get("detector_type")
	severity := q.Get("severity")
	status := q.Get("status")

	query := `
		SELECT id, tenant_id, detector_type, severity, status, dedup_key,
		       title, description, entity_id, entity_type, metadata,
		       detected_at, activity_window_start, activity_window_end, assigned_to,
		       resolution_notes, resolved_by, resolved_at, created_at, updated_at
		FROM compliance.compliance_surveillance_finding
		WHERE tenant_id = $1
	`
	args := []any{tenantID}
	argIdx := 2

	if detectorType != "" {
		query += " AND detector_type = $" + strconv.Itoa(argIdx)
		args = append(args, detectorType)
		argIdx++
	}
	if severity != "" {
		query += " AND severity = $" + strconv.Itoa(argIdx)
		args = append(args, severity)
		argIdx++
	}
	if status != "" {
		query += " AND status = $" + strconv.Itoa(argIdx)
		args = append(args, status)
		argIdx++
	}
	query += " ORDER BY detected_at DESC LIMIT 100"

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		http.Error(w, "query findings failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var findings []SurveillanceFindingRecord
	for rows.Next() {
		var f SurveillanceFindingRecord
		var entityID, assignedTo, resNotes, resBy sql.NullString
		var resAt sql.NullTime
		err := rows.Scan(
			&f.ID, &f.TenantID, &f.DetectorType, &f.Severity, &f.Status, &f.DedupKey,
			&f.Title, &f.Description, &entityID, &f.EntityType, &f.Metadata,
			&f.DetectedAt, &f.ActivityWindowStart, &f.ActivityWindowEnd, &assignedTo,
			&resNotes, &resBy, &resAt, &f.CreatedAt, &f.UpdatedAt,
		)
		if err == nil {
			if entityID.Valid {
				u, _ := uuid.Parse(entityID.String)
				f.EntityID = &u
			}
			if assignedTo.Valid {
				f.AssignedTo = &assignedTo.String
			}
			if resNotes.Valid {
				f.ResolutionNotes = &resNotes.String
			}
			if resBy.Valid {
				f.ResolvedBy = &resBy.String
			}
			if resAt.Valid {
				f.ResolvedAt = &resAt.Time
			}
			findings = append(findings, f)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":        findings,
		"total_count": len(findings),
	})
}

// HandleGetFinding returns single finding with its append-only event trail
func (h *Handler) HandleGetFinding(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		http.Error(w, "invalid tenant id", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	findingID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid finding id", http.StatusBadRequest)
		return
	}

	var f SurveillanceFindingRecord
	var entityID, assignedTo, resNotes, resBy sql.NullString
	var resAt sql.NullTime
	err = h.db.QueryRowContext(r.Context(), `
		SELECT id, tenant_id, detector_type, severity, status, dedup_key,
		       title, description, entity_id, entity_type, metadata,
		       detected_at, activity_window_start, activity_window_end, assigned_to,
		       resolution_notes, resolved_by, resolved_at, created_at, updated_at
		FROM compliance.compliance_surveillance_finding
		WHERE id = $1 AND tenant_id = $2
	`, findingID, tenantID).Scan(
		&f.ID, &f.TenantID, &f.DetectorType, &f.Severity, &f.Status, &f.DedupKey,
		&f.Title, &f.Description, &entityID, &f.EntityType, &f.Metadata,
		&f.DetectedAt, &f.ActivityWindowStart, &f.ActivityWindowEnd, &assignedTo,
		&resNotes, &resBy, &resAt, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		http.Error(w, "finding not found: "+err.Error(), http.StatusNotFound)
		return
	}
	if entityID.Valid {
		u, _ := uuid.Parse(entityID.String)
		f.EntityID = &u
	}
	if assignedTo.Valid {
		f.AssignedTo = &assignedTo.String
	}
	if resNotes.Valid {
		f.ResolutionNotes = &resNotes.String
	}
	if resBy.Valid {
		f.ResolvedBy = &resBy.String
	}
	if resAt.Valid {
		f.ResolvedAt = &resAt.Time
	}

	// Fetch event ledger
	eventRows, err := h.db.QueryContext(r.Context(), `
		SELECT id, finding_id, tenant_id, event_type, actor, payload, created_at
		FROM compliance.compliance_surveillance_event
		WHERE finding_id = $1
		ORDER BY created_at ASC
	`, findingID)
	var events []SurveillanceEventRecord
	if err == nil {
		defer eventRows.Close()
		for eventRows.Next() {
			var ev SurveillanceEventRecord
			if err := eventRows.Scan(&ev.ID, &ev.FindingID, &ev.TenantID, &ev.EventType, &ev.Actor, &ev.Payload, &ev.CreatedAt); err == nil {
				events = append(events, ev)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"finding": f,
		"events":  events,
	})
}

// HandleUpdateFindingStatus executes state-machine status transition
func (h *Handler) HandleUpdateFindingStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		http.Error(w, "invalid tenant id", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	findingID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid finding id", http.StatusBadRequest)
		return
	}

	var payload struct {
		Status          string `json:"status"`
		ResolutionNotes string `json:"resolution_notes"`
		AssignedTo      string `json:"assigned_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	actor := h.getActor(r)

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "transaction begin failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	if payload.Status == "DISMISSED" || payload.Status == "REMEDIATED" || payload.Status == "CLOSED" {
		_, err = tx.ExecContext(r.Context(), `
			UPDATE compliance.compliance_surveillance_finding
			SET status = $1,
			    resolution_notes = $2,
			    resolved_by = $3,
			    resolved_at = now(),
			    updated_at = now()
			WHERE id = $4 AND tenant_id = $5
		`, payload.Status, payload.ResolutionNotes, actor, findingID, tenantID)
	} else {
		_, err = tx.ExecContext(r.Context(), `
			UPDATE compliance.compliance_surveillance_finding
			SET status = $1,
			    assigned_to = COALESCE(NULLIF($2, ''), assigned_to),
			    updated_at = now()
			WHERE id = $3 AND tenant_id = $4
		`, payload.Status, payload.AssignedTo, findingID, tenantID)
	}
	if err != nil {
		http.Error(w, "status transition failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Append to event ledger
	eventPayload, _ := json.Marshal(map[string]any{
		"new_status":       payload.Status,
		"resolution_notes": payload.ResolutionNotes,
		"assigned_to":      payload.AssignedTo,
	})
	_, err = tx.ExecContext(r.Context(), `
		INSERT INTO compliance.compliance_surveillance_event (
			finding_id, tenant_id, event_type, actor, payload
		) VALUES ($1, $2, 'STATUS_CHANGED', $3, $4::jsonb)
	`, findingID, tenantID, actor, string(eventPayload))
	if err != nil {
		http.Error(w, "failed to append surveillance event: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "transaction commit failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":     "UPDATED",
		"finding_id": findingID,
		"new_status": payload.Status,
	})
}

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/tenant"
)

// bindingFieldInput is one field from the binding wizard.
type bindingFieldInput struct {
	FieldName          string `json:"fieldName"`
	FieldRole          string `json:"fieldRole"`
	DataType           string `json:"dataType"`
	TermNodeID         string `json:"termNodeId"`
	BindingRequirement string `json:"bindingRequirement"`
	EligibilitySource  string `json:"eligibilitySource"`
	OrdinalPosition    int    `json:"ordinalPosition"`
	IsExposed          *bool  `json:"isExposed"`
}

var (
	fieldRoles         = set("KEY", "DIMENSION", "MEASURE", "TIME_DIMENSION", "ATTRIBUTE", "CALCULATION")
	bindingRequirement = set("REQUIRED", "OPTIONAL", "BACKEND_SPECIFIC", "CALCULATED", "INTERNAL")
	eligibilitySources = set("DIRECT", "RELATED", "CALCULATED", "MANUAL", "INTERNAL")
)

func set(v ...string) map[string]bool {
	m := make(map[string]bool, len(v))
	for _, s := range v {
		m[s] = true
	}
	return m
}

func pick(v, def string, allowed map[string]bool) (string, error) {
	v = strings.ToUpper(strings.TrimSpace(v))
	if v == "" {
		return def, nil
	}
	if !allowed[v] {
		return "", fmt.Errorf("%q is not allowed", v)
	}
	return v, nil
}

// UpsertBindingFields saves the fields the binding wizard picked for a
// binding: each becomes a business object field (name + semantic term - what
// rules and the semantic field map read) and, where the term MAPS_TO a column
// of the binding's driving table, a RESOLVED field binding to that column.
// Re-saving updates fields in place. is_required (value required) is never
// set here - binding_requirement is a storage-mapping property.
func (h *BusinessObjectHandler) UpsertBindingFields(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil || secCtx == nil || secCtx.TenantID == "" {
		http.Error(w, "authentication with a tenant is required", http.StatusUnauthorized)
		return
	}
	boID, bindingID := chi.URLParam(r, "id"), chi.URLParam(r, "bindingId")
	if _, err := uuid.Parse(boID); err != nil {
		http.Error(w, "business object id must be a uuid", http.StatusBadRequest)
		return
	}
	if _, err := uuid.Parse(bindingID); err != nil {
		http.Error(w, "binding id must be a uuid", http.StatusBadRequest)
		return
	}
	var req struct {
		Fields []bindingFieldInput `json:"fields"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Fields) == 0 || len(req.Fields) > 1000 {
		http.Error(w, "send 1 to 1000 fields", http.StatusBadRequest)
		return
	}
	if h.db == nil {
		http.Error(w, "database unavailable", http.StatusInternalServerError)
		return
	}

	tx, err := h.db.BeginTxx(ctx, nil)
	if err != nil {
		http.Error(w, "database unavailable", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback() //nolint:errcheck
	if err := tenant.SetRLSContext(ctx, tx, secCtx.TenantID); err != nil {
		http.Error(w, "database unavailable", http.StatusInternalServerError)
		return
	}

	// The binding must belong to this business object in the caller's tenant.
	var drivingNodeID string
	if err := tx.GetContext(ctx, &drivingNodeID, `
		SELECT b.driving_node_id::text FROM public.business_object_binding b
		JOIN public.business_objects bo ON bo.id = b.bo_id AND bo.tenant_id = b.tenant_id
		WHERE b.bo_binding_id = $1::uuid AND b.bo_id = $2::uuid AND b.tenant_id = $3::uuid`,
		bindingID, boID, secCtx.TenantID); err != nil {
		http.Error(w, "binding not found", http.StatusNotFound)
		return
	}

	type result struct {
		FieldID string `json:"fieldId"`
		Name    string `json:"fieldName"`
		Status  string `json:"bindingStatus"`
		Column  string `json:"column,omitempty"`
	}
	out := make([]result, 0, len(req.Fields))
	for i, f := range req.Fields {
		name := strings.TrimSpace(f.FieldName)
		if name == "" || len(name) > 128 {
			http.Error(w, fmt.Sprintf("field %d: fieldName must be 1-128 characters", i+1), http.StatusBadRequest)
			return
		}
		role, err1 := pick(f.FieldRole, "DIMENSION", fieldRoles)
		requirement, err2 := pick(f.BindingRequirement, "OPTIONAL", bindingRequirement)
		source, err3 := pick(f.EligibilitySource, "DIRECT", eligibilitySources)
		if err := firstErr(err1, err2, err3); err != nil {
			http.Error(w, fmt.Sprintf("field %s: %v", name, err), http.StatusBadRequest)
			return
		}
		var term any
		if f.TermNodeID != "" {
			if _, err := uuid.Parse(f.TermNodeID); err != nil {
				http.Error(w, fmt.Sprintf("field %s: termNodeId must be a uuid", name), http.StatusBadRequest)
				return
			}
			term = f.TermNodeID
		}
		exposed := true
		if f.IsExposed != nil {
			exposed = *f.IsExposed
		}

		var fieldID string
		if err := tx.GetContext(ctx, &fieldID, `
			INSERT INTO public.business_object_fields
				(tenant_id, bo_id, field_name, term_node_id, field_role, binding_requirement, eligibility_source,
				 is_exposed, display_order, data_type, display_name, technical_name)
			VALUES ($1::uuid, $2::uuid, $3, $4::uuid, $5, $6, $7, $8, $9, NULLIF($10, ''), $3, $3)
			ON CONFLICT (tenant_id, bo_id, field_name) DO UPDATE SET
				term_node_id = EXCLUDED.term_node_id, field_role = EXCLUDED.field_role,
				binding_requirement = EXCLUDED.binding_requirement, eligibility_source = EXCLUDED.eligibility_source,
				is_exposed = EXCLUDED.is_exposed, display_order = EXCLUDED.display_order,
				data_type = COALESCE(EXCLUDED.data_type, business_object_fields.data_type), updated_at = now()
			RETURNING id::text`,
			secCtx.TenantID, boID, name, term, role, requirement, source, exposed, f.OrdinalPosition, f.DataType); err != nil {
			http.Error(w, fmt.Sprintf("field %s could not be saved: %v", name, err), http.StatusBadRequest)
			return
		}

		// The column of the driving table the term MAPS_TO, if any.
		status, column := "UNRESOLVED", ""
		var colID, colName string
		if term != nil {
			if err := tx.QueryRowxContext(ctx, `
				SELECT col.id::text, col.node_name FROM public.catalog_edge ce
				JOIN public.catalog_edge_type et ON et.id = ce.edge_type_id AND et.edge_type_name = 'MAPS_TO'
				JOIN public.catalog_node col ON col.id = ce.target_node_id
				WHERE ce.source_node_id = $1::uuid AND col.parent_id = $2::uuid
				ORDER BY col.node_name LIMIT 1`, term, drivingNodeID).Scan(&colID, &colName); err == nil {
				status, column = "RESOLVED", colName
			}
		}
		if status == "RESOLVED" {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO public.field_bindings
					(tenant_id, bo_id, binding_id, field_id, source_node_id, source_type, transformation_type, binding_status)
				VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5::uuid, 'COLUMN', 'NONE', 'RESOLVED')
				ON CONFLICT (tenant_id, binding_id, field_id) DO UPDATE SET
					source_node_id = EXCLUDED.source_node_id, binding_status = 'RESOLVED', updated_at = now()`,
				secCtx.TenantID, boID, bindingID, fieldID, colID); err != nil {
				http.Error(w, fmt.Sprintf("field %s: binding could not be saved: %v", name, err), http.StatusBadRequest)
				return
			}
		}
		out = append(out, result{FieldID: fieldID, Name: name, Status: status, Column: column})
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "fields could not be saved", http.StatusInternalServerError)
		return
	}
	resolved := 0
	for _, f := range out {
		if f.Status == "RESOLVED" {
			resolved++
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"fields": out, "saved": len(out), "resolved": resolved})
}

func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

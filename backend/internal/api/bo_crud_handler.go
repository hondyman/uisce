package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/cbo"
	"github.com/hondyman/uisce/backend/internal/security"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

type BOCRUDHandler struct {
	db              *sqlx.DB
	trigger         *TriggerEngine
	entitlementRepo *cbo.DBEntitlementRepository
	// entitlements is the fallback authorization check for writes when no
	// semantic.bo_entitlement_policies row exists for a BO (the common
	// case — that table is empty platform-wide as of this writing). Without
	// it, evaluateBOEntitlements silently allows every write once no cbo
	// policy is configured, which is a real, live bypass around whatever a
	// role's bp_field_permissions actually grant — this direct CRUD path
	// doesn't go through metadata.BusinessObjectService.requireAccess at
	// all. See SetEntitlementsService.
	entitlements      *security.EntitlementsService
	groupRoleResolver *security.GroupRoleResolver
}

func NewBOCRUDHandler(db *sqlx.DB, trigger *TriggerEngine) *BOCRUDHandler {
	return &BOCRUDHandler{db: db, trigger: trigger, entitlementRepo: cbo.NewDBEntitlementRepository(db)}
}

// SetEntitlementsService wires the bp_field_permissions-backed write gate
// used when no cbo entitlement policy is configured for a BO.
func (h *BOCRUDHandler) SetEntitlementsService(ent *security.EntitlementsService, groupRoles *security.GroupRoleResolver) {
	h.entitlements = ent
	h.groupRoleResolver = groupRoles
}

// boCRUDIdentifierPattern restricts entitlement-policy-supplied column names
// to safe SQL identifiers before they're interpolated into a WHERE clause —
// mirrors cbo_adapter.go's filterIdentifierPattern.
var boCRUDIdentifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// boEntitlementContext is the result of evaluating this BO's entitlement
// policies (see cbo.EntitlementPolicy / semantic.bo_entitlement_policies)
// for the current caller: an optional row-scoping condition and the set of
// fields to strip from responses. It's the same policy table and semantics
// cbo.Planner enforces for API Studio's REST/GraphQL runtime — evaluated
// here directly (BO CRUD doesn't route through the CBO planner) so record
// access is entitled consistently regardless of whether it's reached
// through a page, this direct CRUD API, or Studio-published endpoints.
type boEntitlementContext struct {
	rowFilterColumn string
	rowFilterValue  string
	maskedFields    map[string][]string
	// directMasks is populated by the bp_field_permissions fallback (see
	// fallbackEntitlementsGate) when no cbo policy exists for this BO: field
	// name -> "HIDE" (delete the field) or "MASK" (replace its value).
	// Unlike maskedFields (a role allowlist checked later against the
	// caller's roles), this is already resolved for the specific caller, so
	// applyDirectFieldMasks applies it unconditionally.
	directMasks map[string]string
}

// evaluateBOEntitlements fetches cbo policies for boKey and enforces the
// role gate, returning cbo.ErrEntitlementDenied if the caller doesn't
// satisfy it. requireWrite distinguishes a mutation (create/update/delete)
// from a read (get/list). When no cbo policy exists for the BO — the
// common case, see the entitlements field's comment — both reads and
// writes fall back to bp_field_permissions via security.EntitlementsService
// (fallbackEntitlementsGate): a role explicitly denied or scoped to
// read-only can't write through this path just because nobody configured a
// cbo policy, and masked/hidden fields are stripped from what's returned
// even though this handler is a separate code path from
// metadata.BusinessObjectService.QueryBORecords.
func (h *BOCRUDHandler) evaluateBOEntitlements(ctx context.Context, r *http.Request, tenantID uuid.UUID, boKey string, requireWrite bool) (*boEntitlementContext, error) {
	ec := &boEntitlementContext{}
	if h.entitlementRepo == nil {
		return h.fallbackEntitlementsGate(ctx, r, tenantID, boKey, requireWrite, ec)
	}

	policies, err := h.entitlementRepo.GetPoliciesForBO(ctx, &tenantID, boKey)
	if err != nil || len(policies) == 0 {
		return h.fallbackEntitlementsGate(ctx, r, tenantID, boKey, requireWrite, ec)
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	callerRoles := make(map[string]bool)
	var organizationID, userID string
	if claims != nil {
		for _, role := range claims.Roles {
			callerRoles[role] = true
		}
		organizationID = claims.OrganizationID
		userID = claims.UserID
	}

	ec.maskedFields = make(map[string][]string)
	for _, policy := range policies {
		if len(policy.RequiredRoles) > 0 {
			satisfied := false
			for _, role := range policy.RequiredRoles {
				if callerRoles[role] {
					satisfied = true
					break
				}
			}
			if !satisfied {
				return nil, cbo.ErrEntitlementDenied
			}
		}

		if policy.FilterColumn != "" && boCRUDIdentifierPattern.MatchString(policy.FilterColumn) {
			var claimValue string
			switch policy.RowFilterClaim {
			case "tenant_id":
				claimValue = tenantID.String()
			case "organization_id":
				claimValue = organizationID
			case "user_id":
				claimValue = userID
			}
			if claimValue != "" {
				// Only one row-filter binding is applied if multiple
				// policies declare one; combining independent policies'
				// row filters with AND is not yet supported.
				ec.rowFilterColumn = policy.FilterColumn
				ec.rowFilterValue = claimValue
			}
		}

		for field, roles := range policy.MaskedFields {
			ec.maskedFields[field] = append(ec.maskedFields[field], roles...)
		}
	}

	return ec, nil
}

// fallbackEntitlementsGate is the bp_field_permissions-backed check used
// whenever no cbo entitlement policy governs a BO (the common case — see
// the entitlements field's comment). It denies the whole request if the BO
// is explicitly hidden, denies a write if the caller's role is scoped to
// read-only (or explicitly none), and always populates ec.directMasks so
// callers strip/mask fields the caller isn't entitled to see — this is
// what keeps a read through this direct CRUD path consistent with what
// metadata.BusinessObjectService.QueryBORecords already enforces.
//
// Any resolution failure (no entitlements service wired, BO not found in
// business_objects, no JWT claims) allows the request through unchanged —
// this is a targeted patch for the concrete case of an explicit grant
// existing, not a new fail-closed default for the platform; failing open
// here matches this path's pre-existing behavior for every other case.
func (h *BOCRUDHandler) fallbackEntitlementsGate(ctx context.Context, r *http.Request, tenantID uuid.UUID, boKey string, requireWrite bool, ec *boEntitlementContext) (*boEntitlementContext, error) {
	if h.entitlements == nil {
		return ec, nil
	}
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil || claims.UserID == "" {
		return ec, nil
	}

	var boID string
	err := h.db.GetContext(ctx, &boID, `
		SELECT id FROM business_objects
		WHERE (bo_key = $1 OR id::text = $1) AND tenant_id = $2
		LIMIT 1
	`, boKey, tenantID)
	if err != nil {
		return ec, nil
	}

	secCtx := &security.Context{
		UserID:   claims.UserID,
		TenantID: tenantID.String(),
		Roles:    claims.Roles,
	}
	for _, role := range claims.Roles {
		if role == "global_admin" || role == "global_ops" {
			secCtx.IsGlobalAdmin = true
			break
		}
	}
	if h.groupRoleResolver != nil && len(claims.IdpGroups) > 0 {
		if groupRoles, gerr := h.groupRoleResolver.ResolveRoles(ctx, claims.UserID, secCtx.TenantID, claims.IdpGroups); gerr == nil {
			secCtx.Roles = append(secCtx.Roles, groupRoles...)
		}
	}

	entitlements, err := h.entitlements.ForUser(ctx, secCtx)
	if err != nil || entitlements == nil {
		return ec, nil
	}

	if _, hidden := entitlements.HiddenBOs[boID]; hidden {
		return nil, cbo.ErrEntitlementDenied
	}

	if requireWrite {
		key := security.EntitlementKey{ResourceID: boID, FieldName: "*"}
		switch entitlements.Entitlements[key] {
		case security.PermissionRead, security.PermissionNone:
			return nil, cbo.ErrEntitlementDenied
		}
	}

	ec.directMasks = make(map[string]string)
	for key, perm := range entitlements.Entitlements {
		if key.ResourceID != boID || key.FieldName == "*" {
			continue
		}
		switch perm {
		case security.PermissionNone:
			ec.directMasks[key.FieldName] = "HIDE"
		case security.PermissionMask:
			ec.directMasks[key.FieldName] = "MASK"
		}
	}

	return ec, nil
}

// applyDirectFieldMasks applies ec.directMasks — already resolved for the
// specific caller by fallbackEntitlementsGate — deleting HIDE fields and
// replacing MASK fields with a placeholder.
func applyDirectFieldMasks(records []map[string]interface{}, directMasks map[string]string) {
	if len(directMasks) == 0 {
		return
	}
	for _, record := range records {
		for field, mode := range directMasks {
			if _, ok := record[field]; !ok {
				continue
			}
			switch mode {
			case "HIDE":
				delete(record, field)
			case "MASK":
				record[field] = "[MASKED]"
			}
		}
	}
}

// maskBORecordFields strips fields from each record when the caller holds
// none of the roles the field's entitlement policy requires.
func maskBORecordFields(records []map[string]interface{}, maskedFields map[string][]string, r *http.Request) {
	if len(maskedFields) == 0 || len(records) == 0 {
		return
	}
	claims := jwtmiddleware.GetClaimsFromContext(r)
	roleSet := make(map[string]bool)
	if claims != nil {
		for _, role := range claims.Roles {
			roleSet[role] = true
		}
	}
	for field, allowedRoles := range maskedFields {
		allowed := false
		for _, role := range allowedRoles {
			if roleSet[role] {
				allowed = true
				break
			}
		}
		if allowed {
			continue
		}
		for _, record := range records {
			delete(record, field)
		}
	}
}

// resolveWritableColumns returns the set of real column names for drivingTable,
// used to allowlist client-supplied JSON keys before they are interpolated into
// SQL as identifiers (column names can't be bind-parameterized).
func (h *BOCRUDHandler) resolveWritableColumns(ctx context.Context, drivingTable string) (map[string]bool, error) {
	schema := "public"
	table := drivingTable
	if idx := strings.Index(drivingTable, "."); idx >= 0 {
		schema = drivingTable[:idx]
		table = drivingTable[idx+1:]
	}
	var cols []string
	err := h.db.SelectContext(ctx, &cols, `
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2;
	`, schema, table)
	if err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("no columns found for table '%s'", drivingTable)
	}
	set := make(map[string]bool, len(cols))
	for _, c := range cols {
		set[c] = true
	}
	return set, nil
}

// emitBORowEvent fires a best-effort trigger evaluation after a committed write.
// Failures are logged, never surfaced to the caller — trigger evaluation must not
// roll back or fail an otherwise-successful BO mutation.
func (h *BOCRUDHandler) emitBORowEvent(triggerKey string, tenantID uuid.UUID, userID, boKey, recordID string, eventData map[string]interface{}) {
	if h.trigger == nil {
		return
	}
	go func() {
		_, err := h.trigger.EvaluateTriggers(context.Background(), &TriggerContext{
			TenantID:     tenantID.String(),
			UserID:       userID,
			TriggerKey:   triggerKey,
			TargetEntity: boKey,
			EntityID:     recordID,
			EventData:    eventData,
			RequestedAt:  time.Now(),
		})
		if err != nil {
			log.Printf("[WARN] BO row event %q for %s/%s failed: %v", triggerKey, boKey, recordID, err)
		}
	}()
}

func (h *BOCRUDHandler) RegisterRoutes(r chi.Router) {
	r.Route("/bo", func(r chi.Router) {
		r.Get("/{boKey}/records", h.HandleListBORecords)
		r.Post("/{boKey}/records", h.HandleCreateBORecord)
		r.Get("/{boKey}/records/{recordId}", h.HandleGetBORecord)
		r.Put("/{boKey}/records/{recordId}", h.HandleUpdateBORecord)
		r.Delete("/{boKey}/records/{recordId}", h.HandleDeleteBORecord)
		r.Get("/{boKey}/topology-summary", h.HandleGetBOTopologySummary)
		r.Get("/{boKey}/records/{recordId}/relationships/{relKey}", h.HandleListRelatedRecords)
		r.Post("/{boKey}/records/{recordId}/relationships/{relKey}", h.HandleCreateRelatedRecord)
		r.Put("/{boKey}/records/{recordId}/relationships/{relKey}/{childId}", h.HandleUpdateRelatedRecord)
		r.Delete("/{boKey}/records/{recordId}/relationships/{relKey}/{childId}", h.HandleDeleteRelatedRecord)
	})
}

type boBindingMetadata struct {
	DrivingTable string `db:"driving_table"`
	KeyColumn    string `db:"key_column"`
}

func (h *BOCRUDHandler) resolveBOMetadata(ctx context.Context, boKey string, tenantID uuid.UUID) (*boBindingMetadata, error) {
	var boMeta boBindingMetadata

	// 1. Try public.business_objects + business_object_bindings
	metaQuery := `
		SELECT COALESCE(bob.driving_table, bo.driver_table_name, '') AS driving_table,
		       COALESCE(bob.key_column, 'id') AS key_column
		FROM public.business_objects bo
		LEFT JOIN public.business_object_bindings bob ON bob.bo_id = bo.id AND bob.is_default = TRUE
		WHERE (bo.key = $1 OR bo.id::text = $1) AND (bo.tenant_id = $2 OR bo.is_gold_copy = TRUE)
		ORDER BY CASE WHEN bo.tenant_id = $2 THEN 0 ELSE 1 END
		LIMIT 1;
	`
	err := h.db.GetContext(ctx, &boMeta, metaQuery, boKey, tenantID)
	if err == nil && boMeta.DrivingTable != "" {
		return &boMeta, nil
	}

	// 2. Try catalog_node (BUSINESS_OBJECT)
	catalogQuery := `
		SELECT COALESCE(metadata->>'driving_table', metadata->>'table_name', replace(qualified_path, '/', '.')) AS driving_table,
		       COALESCE(metadata->>'key_column', metadata->>'primary_key', 'id') AS key_column
		FROM public.catalog_node
		WHERE node_type = 'BUSINESS_OBJECT'
		  AND (node_key = $1 OR qualified_path = $1 OR id::text = $1)
		  AND (tenant_id = $2 OR is_gold_copy = TRUE)
		ORDER BY CASE WHEN tenant_id = $2 THEN 0 ELSE 1 END
		LIMIT 1;
	`
	err = h.db.GetContext(ctx, &boMeta, catalogQuery, boKey, tenantID)
	if err == nil && boMeta.DrivingTable != "" {
		return &boMeta, nil
	}

	// 3. Fallback: Check if boKey is a direct qualified table like "oms.account", "master.customer", etc.
	if strings.Contains(boKey, ".") {
		return &boBindingMetadata{
			DrivingTable: boKey,
			KeyColumn:    "id",
		}, nil
	}

	// 4. Default schema fallback if table exists
	for _, schema := range []string{"oms", "master", "altinv", "cash_flow", "public"} {
		qualified := fmt.Sprintf("%s.%s", schema, boKey)
		var exists bool
		checkQuery := `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables 
				WHERE table_schema = $1 AND table_name = $2
			);
		`
		if err := h.db.GetContext(ctx, &exists, checkQuery, schema, boKey); err == nil && exists {
			return &boBindingMetadata{
				DrivingTable: qualified,
				KeyColumn:    "id",
			}, nil
		}
	}

	return nil, fmt.Errorf("business object definition not found for key '%s'", boKey)
}

// resolveDiscriminatorColumn probes whether the driving table has a "subtype_code" column —
// the convention actually used across every STI subtype BO in the live catalog
// (business_objects.sti_discriminator_column is always "subtype_code" for compound
// "{root}/{subtypeCode}" bo_key rows, e.g. "oms.account/sma"). Returns ("", false) for BOs
// without subtypes so callers can skip subtype scoping entirely (no behavior change).
func (h *BOCRUDHandler) resolveDiscriminatorColumn(ctx context.Context, drivingTable string) (string, bool) {
	schema := "public"
	table := drivingTable
	if idx := strings.Index(drivingTable, "."); idx >= 0 {
		schema = drivingTable[:idx]
		table = drivingTable[idx+1:]
	}
	var exists bool
	checkQuery := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = $1 AND table_name = $2 AND column_name = 'subtype_code'
		);
	`
	if err := h.db.GetContext(ctx, &exists, checkQuery, schema, table); err == nil && exists {
		return "subtype_code", true
	}
	return "", false
}

func extractTenantUUIDFromRequest(r *http.Request) uuid.UUID {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims != nil && claims.TenantID != "" {
		if id, err := uuid.Parse(claims.TenantID); err == nil {
			return id
		}
	}
	tenantHeader := r.Header.Get("X-Tenant-ID")
	if tenantHeader != "" {
		if id, err := uuid.Parse(tenantHeader); err == nil {
			return id
		}
	}
	return uuid.MustParse("00000000-0000-0000-0000-000000000001")
}

// HandleUpdateBORecord commits validated OLTP mutations with Cardinal Rule 7 tenant scoping
func (h *BOCRUDHandler) HandleUpdateBORecord(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)
	boKey := chi.URLParam(r, "boKey")
	recordID := chi.URLParam(r, "recordId")

	if boKey == "" || recordID == "" {
		http.Error(w, "boKey and recordId are required", http.StatusBadRequest)
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	boMeta, err := h.resolveBOMetadata(r.Context(), boKey, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving BO contract: %v", err), http.StatusNotFound)
		return
	}

	ec, err := h.evaluateBOEntitlements(r.Context(), r, tenantID, boKey, true)
	if err != nil {
		if errors.Is(err, cbo.ErrEntitlementDenied) {
			http.Error(w, "Forbidden: caller does not satisfy entitlement policy", http.StatusForbidden)
			return
		}
		http.Error(w, fmt.Sprintf("failed evaluating entitlements: %v", err), http.StatusInternalServerError)
		return
	}

	writableCols, err := h.resolveWritableColumns(r.Context(), boMeta.DrivingTable)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving table schema: %v", err), http.StatusInternalServerError)
		return
	}

	setClauses := make([]string, 0)
	args := []interface{}{tenantID, recordID}
	argIdx := 3

	for fieldKey, val := range payload {
		// Rule 7 Defense: Protect tenant, surrogate primary keys, and audit timestamps from mutation
		lower := strings.ToLower(fieldKey)
		if lower == "id" || lower == "tenant_id" || lower == "created_at" || lower == "created_by" {
			continue
		}
		if !writableCols[fieldKey] {
			http.Error(w, fmt.Sprintf("unknown attribute '%s'", fieldKey), http.StatusBadRequest)
			return
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", fieldKey, argIdx))
		args = append(args, val)
		argIdx++
	}

	if len(setClauses) == 0 {
		http.Error(w, "no writable attributes provided", http.StatusBadRequest)
		return
	}

	extraWhere := ""
	if subtype := r.URL.Query().Get("subtype"); subtype != "" {
		if col, ok := h.resolveDiscriminatorColumn(r.Context(), boMeta.DrivingTable); ok {
			// Defends against updating a record that doesn't belong to this subtype.
			extraWhere += fmt.Sprintf(" AND %s = $%d", col, argIdx)
			args = append(args, subtype)
			argIdx++
		}
	}
	if ec.rowFilterColumn != "" {
		extraWhere += fmt.Sprintf(" AND %s = $%d", ec.rowFilterColumn, argIdx)
		args = append(args, ec.rowFilterValue)
		argIdx++
	}

	updateSQL := fmt.Sprintf(`
		UPDATE %s
		SET %s, updated_at = NOW()
		WHERE tenant_id = $1 AND %s = $2%s
		RETURNING *;
	`, boMeta.DrivingTable, strings.Join(setClauses, ", "), boMeta.KeyColumn, extraWhere)

	rows, err := h.db.QueryxContext(r.Context(), updateSQL, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("database mutation error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := make(map[string]interface{})
	if rows.Next() {
		if err := rows.MapScan(result); err != nil {
			http.Error(w, "failed mapping updated record: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "record not found or tenant access violation (Rule 7)", http.StatusNotFound)
		return
	}

	// Clean byte arrays or UUIDs for JSON serialization
	cleanScanResult(result)
	maskBORecordFields([]map[string]interface{}{result}, ec.maskedFields, r)
	applyDirectFieldMasks([]map[string]interface{}{result}, ec.directMasks)

	h.emitBORowEvent("row_update", tenantID, actorFromRequest(r), boKey, recordID, result)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// HandleCreateBORecord creates a new record in the driving table
func (h *BOCRUDHandler) HandleCreateBORecord(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)
	boKey := chi.URLParam(r, "boKey")

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	boMeta, err := h.resolveBOMetadata(r.Context(), boKey, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving BO contract: %v", err), http.StatusNotFound)
		return
	}

	ec, err := h.evaluateBOEntitlements(r.Context(), r, tenantID, boKey, true)
	if err != nil {
		if errors.Is(err, cbo.ErrEntitlementDenied) {
			http.Error(w, "Forbidden: caller does not satisfy entitlement policy", http.StatusForbidden)
			return
		}
		http.Error(w, fmt.Sprintf("failed evaluating entitlements: %v", err), http.StatusInternalServerError)
		return
	}

	subtype := r.URL.Query().Get("subtype")
	if subtype != "" {
		if col, ok := h.resolveDiscriminatorColumn(r.Context(), boMeta.DrivingTable); ok {
			// Force the discriminator value server-side, overriding whatever the client sent.
			payload[col] = subtype
		}
	}
	if ec.rowFilterColumn != "" {
		// Force the entitled scoping value server-side, same as subtype above —
		// a caller must not be able to create a record outside their own scope.
		payload[ec.rowFilterColumn] = ec.rowFilterValue
	}

	writableCols, err := h.resolveWritableColumns(r.Context(), boMeta.DrivingTable)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving table schema: %v", err), http.StatusInternalServerError)
		return
	}

	columns := []string{"tenant_id"}
	placeholders := []string{"$1"}
	args := []interface{}{tenantID}
	argIdx := 2

	for fieldKey, val := range payload {
		lower := strings.ToLower(fieldKey)
		if lower == "tenant_id" || lower == "created_at" || lower == "updated_at" {
			continue
		}
		if !writableCols[fieldKey] {
			http.Error(w, fmt.Sprintf("unknown attribute '%s'", fieldKey), http.StatusBadRequest)
			return
		}
		columns = append(columns, fieldKey)
		placeholders = append(placeholders, fmt.Sprintf("$%d", argIdx))
		args = append(args, val)
		argIdx++
	}

	insertSQL := fmt.Sprintf(`
		INSERT INTO %s (%s)
		VALUES (%s)
		RETURNING *;
	`, boMeta.DrivingTable, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	rows, err := h.db.QueryxContext(r.Context(), insertSQL, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed creating record: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := make(map[string]interface{})
	if rows.Next() {
		if err := rows.MapScan(result); err != nil {
			http.Error(w, "failed mapping created record", http.StatusInternalServerError)
			return
		}
	}

	cleanScanResult(result)
	maskBORecordFields([]map[string]interface{}{result}, ec.maskedFields, r)
	applyDirectFieldMasks([]map[string]interface{}{result}, ec.directMasks)

	newRecordID := fmt.Sprintf("%v", result[boMeta.KeyColumn])
	h.emitBORowEvent("row_insert", tenantID, actorFromRequest(r), boKey, newRecordID, result)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(result)
}

// HandleGetBORecord hydrates a single record by ID
func (h *BOCRUDHandler) HandleGetBORecord(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)
	boKey := chi.URLParam(r, "boKey")
	recordID := chi.URLParam(r, "recordId")

	boMeta, err := h.resolveBOMetadata(r.Context(), boKey, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving BO contract: %v", err), http.StatusNotFound)
		return
	}

	ec, err := h.evaluateBOEntitlements(r.Context(), r, tenantID, boKey, false)
	if err != nil {
		if errors.Is(err, cbo.ErrEntitlementDenied) {
			http.Error(w, "Forbidden: caller does not satisfy entitlement policy", http.StatusForbidden)
			return
		}
		http.Error(w, fmt.Sprintf("failed evaluating entitlements: %v", err), http.StatusInternalServerError)
		return
	}

	args := []interface{}{tenantID, recordID}
	extraWhere := ""
	argIdx := 3
	if subtype := r.URL.Query().Get("subtype"); subtype != "" {
		if col, ok := h.resolveDiscriminatorColumn(r.Context(), boMeta.DrivingTable); ok {
			extraWhere += fmt.Sprintf(" AND %s = $%d", col, argIdx)
			args = append(args, subtype)
			argIdx++
		}
	}
	if ec.rowFilterColumn != "" {
		extraWhere += fmt.Sprintf(" AND %s = $%d", ec.rowFilterColumn, argIdx)
		args = append(args, ec.rowFilterValue)
		argIdx++
	}

	selectSQL := fmt.Sprintf(`
		SELECT * FROM %s
		WHERE tenant_id = $1 AND %s = $2%s
		LIMIT 1;
	`, boMeta.DrivingTable, boMeta.KeyColumn, extraWhere)

	rows, err := h.db.QueryxContext(r.Context(), selectSQL, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed querying record: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := make(map[string]interface{})
	if rows.Next() {
		if err := rows.MapScan(result); err != nil {
			http.Error(w, "failed mapping record", http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "record not found", http.StatusNotFound)
		return
	}

	cleanScanResult(result)
	maskBORecordFields([]map[string]interface{}{result}, ec.maskedFields, r)
	applyDirectFieldMasks([]map[string]interface{}{result}, ec.directMasks)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// HandleListBORecords provides paginated / infinite-scroll chunk loading
func (h *BOCRUDHandler) HandleListBORecords(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)
	boKey := chi.URLParam(r, "boKey")

	boMeta, err := h.resolveBOMetadata(r.Context(), boKey, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving BO contract: %v", err), http.StatusNotFound)
		return
	}

	ec, err := h.evaluateBOEntitlements(r.Context(), r, tenantID, boKey, false)
	if err != nil {
		if errors.Is(err, cbo.ErrEntitlementDenied) {
			http.Error(w, "Forbidden: caller does not satisfy entitlement policy", http.StatusForbidden)
			return
		}
		http.Error(w, fmt.Sprintf("failed evaluating entitlements: %v", err), http.StatusInternalServerError)
		return
	}

	limit := 30
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	parentId := r.URL.Query().Get("parentId")
	whereClauses := []string{"tenant_id = $1"}
	args := []interface{}{tenantID}
	argIdx := 2

	if parentId != "" {
		// Common foreign key columns
		whereClauses = append(whereClauses, fmt.Sprintf("(account_id = $%d OR parent_id = $%d OR sponsor_id = $%d)", argIdx, argIdx, argIdx))
		args = append(args, parentId)
		argIdx++
	}

	if subtype := r.URL.Query().Get("subtype"); subtype != "" {
		if col, ok := h.resolveDiscriminatorColumn(r.Context(), boMeta.DrivingTable); ok {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", col, argIdx))
			args = append(args, subtype)
			argIdx++
		}
	}
	if ec.rowFilterColumn != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", ec.rowFilterColumn, argIdx))
		args = append(args, ec.rowFilterValue)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT * FROM %s
		WHERE %s
		ORDER BY %s DESC
		LIMIT $%d OFFSET $%d;
	`, boMeta.DrivingTable, strings.Join(whereClauses, " AND "), boMeta.KeyColumn, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := h.db.QueryxContext(r.Context(), query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed listing records: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	records := make([]map[string]interface{}, 0)
	for rows.Next() {
		item := make(map[string]interface{})
		if err := rows.MapScan(item); err == nil {
			cleanScanResult(item)
			records = append(records, item)
		}
	}
	maskBORecordFields(records, ec.maskedFields, r)
	applyDirectFieldMasks(records, ec.directMasks)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"records": records,
		"count":   len(records),
		"limit":   limit,
		"offset":  offset,
	})
}

// HandleDeleteBORecord deletes or soft-deletes a record
func (h *BOCRUDHandler) HandleDeleteBORecord(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)
	boKey := chi.URLParam(r, "boKey")
	recordID := chi.URLParam(r, "recordId")

	boMeta, err := h.resolveBOMetadata(r.Context(), boKey, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving BO contract: %v", err), http.StatusNotFound)
		return
	}

	ec, err := h.evaluateBOEntitlements(r.Context(), r, tenantID, boKey, true)
	if err != nil {
		if errors.Is(err, cbo.ErrEntitlementDenied) {
			http.Error(w, "Forbidden: caller does not satisfy entitlement policy", http.StatusForbidden)
			return
		}
		http.Error(w, fmt.Sprintf("failed evaluating entitlements: %v", err), http.StatusInternalServerError)
		return
	}

	deleteArgs := []interface{}{tenantID, recordID}
	extraWhere := ""
	if ec.rowFilterColumn != "" {
		extraWhere = fmt.Sprintf(" AND %s = $3", ec.rowFilterColumn)
		deleteArgs = append(deleteArgs, ec.rowFilterValue)
	}

	deleteSQL := fmt.Sprintf(`DELETE FROM %s WHERE tenant_id = $1 AND %s = $2%s`, boMeta.DrivingTable, boMeta.KeyColumn, extraWhere)
	res, err := h.db.ExecContext(r.Context(), deleteSQL, deleteArgs...)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed deleting record: %v", err), http.StatusInternalServerError)
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		http.Error(w, "record not found or unauthorized", http.StatusNotFound)
		return
	}

	h.emitBORowEvent("row_delete", tenantID, actorFromRequest(r), boKey, recordID, nil)

	w.WriteHeader(http.StatusNoContent)
}

type TopologySubtype struct {
	SubtypeCode         string `json:"subtypeCode"`
	DisplayName         string `json:"displayName"`
	IsSatelliteTable    bool   `json:"isSatelliteTable"`
	SatelliteTable      string `json:"satelliteTable,omitempty"`
	AssignedFieldsCount int    `json:"assignedFieldsCount"`
}

type TopologyRelationship struct {
	RelKey          string `json:"relKey"`
	RelName         string `json:"relName"`
	TargetBOKey     string `json:"targetBoKey"`
	TargetBOName    string `json:"targetBoName"`
	Cardinality     string `json:"cardinality"`
	IsSubtypeScoped bool   `json:"isSubtypeScoped"`
}

// HandleGetBOTopologySummary inspects the catalog graph and subtype registry
func (h *BOCRUDHandler) HandleGetBOTopologySummary(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)
	boKey := chi.URLParam(r, "boKey")

	// 1. Discover Subtypes from oms.subtype_registry
	var subtypes []TopologySubtype
	subtypesQuery := `
		SELECT subtype_code AS "subtypeCode",
		       subtype_name AS "displayName",
		       false AS "isSatelliteTable",
		       '' AS "satelliteTable",
		       COALESCE(jsonb_array_length(field_allowlist), 0) AS "assignedFieldsCount"
		FROM oms.subtype_registry
		WHERE root_object = $1 AND (tenant_id = $2 OR tenant_id = '00000000-0000-0000-0000-000000000001')
		ORDER BY subtype_code;
	`
	_ = h.db.SelectContext(r.Context(), &subtypes, subtypesQuery, boKey, tenantID)

	// Fallback mock/defaults if empty
	if len(subtypes) == 0 {
		if boKey == "account" || boKey == "oms.account" {
			subtypes = []TopologySubtype{
				{SubtypeCode: "institutional", DisplayName: "Institutional Account", AssignedFieldsCount: 14},
				{SubtypeCode: "retail_wealth", DisplayName: "Retail Wealth", AssignedFieldsCount: 12},
				{SubtypeCode: "sma", DisplayName: "Separately Managed Account", AssignedFieldsCount: 10},
			}
		}
	}

	// 2. Discover Relationships from catalog_edge or graph conventions
	relationships := []TopologyRelationship{
		{
			RelKey:       "mandate_info",
			RelName:      "Account Mandate Info",
			TargetBOKey:  "mandate",
			TargetBOName: "Mandate",
			Cardinality:  "1:1",
		},
		{
			RelKey:       "positions",
			RelName:      "Account Positions",
			TargetBOKey:  "position",
			TargetBOName: "Position",
			Cardinality:  "1:N",
		},
		{
			RelKey:       "trade_orders",
			RelName:      "Trade Orders",
			TargetBOKey:  "trade_order",
			TargetBOName: "Trade Order",
			Cardinality:  "1:N",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"rootBoKey":     boKey,
		"subtypes":      subtypes,
		"relationships": relationships,
	})
}

func cleanScanResult(m map[string]interface{}) {
	for k, v := range m {
		if b, ok := v.([]byte); ok {
			m[k] = string(b)
		}
	}
}

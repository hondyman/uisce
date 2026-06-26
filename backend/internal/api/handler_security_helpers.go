package api

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/audit"
	"github.com/hondyman/semlayer/backend/internal/identity"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/internal/webhooks"
)

// authorizeRequest enforces tenant scope, actor identity, and optional RBAC permissions.
func authorizeRequest(w http.ResponseWriter, r *http.Request, secMgr *services.SecurityManager, permission string) (string, string, string, bool) {
	ctx := r.Context()
	tenantContext, err := extractTenantContext(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Tenant scope required: "+err.Error(), "missing_tenant_scope", nil)
		return "", "", "", false
	}

	actorID, _ := identity.ActorIDFromContext(ctx)
	if actorID == "" {
		if raw := ctx.Value("user_id"); raw != nil {
			switch v := raw.(type) {
			case uuid.UUID:
				actorID = v.String()
			case string:
				actorID = v
			}
		}
	}
	if actorID == "" {
		actorID = r.Header.Get("X-User-ID")
	}
	if actorID == "" {
		actorID = r.Header.Get("X-Actor-ID")
	}
	if actorID == "" {
		writeJSONError(w, http.StatusUnauthorized, "Authenticated user context required", "missing_actor", nil)
		return "", "", "", false
	}

	if permission != "" {
		if secMgr == nil {
			logging.GetLogger().Sugar().Warnw("security manager missing; skipping permission enforcement", "permission", permission)
		} else if !secMgr.HasPermission(actorID, permission) {
			writeJSONError(w, http.StatusForbidden, "Insufficient permissions for requested operation", "forbidden", nil)
			return "", "", "", false
		}
	}

	return actorID, tenantContext.TenantID, tenantContext.DatasourceID, true
}

// logAuditAccess records read/list style operations if the audit service is configured.
func logAuditAccess(ctx context.Context, auditSvc *audit.Service, actorID, tenantID, objectID, objectType, action string, details map[string]interface{}) {
	if auditSvc == nil {
		return
	}
	if details == nil {
		details = map[string]interface{}{}
	}
	objectName := objectID
	if objectName == "" {
		objectName = objectType
	}
	if err := auditSvc.LogDataAccess(ctx, actorID, tenantID, objectID, objectType, objectName, action, details); err != nil {
		logging.GetLogger().Sugar().Warnw("failed to log access audit", "error", err, "object_type", objectType, "action", action)
	}
}

// logAuditModification records create/update/delete actions through the audit service.
func logAuditModification(ctx context.Context, auditSvc *audit.Service, actorID, tenantID, objectID, objectType, action string, oldData, newData interface{}) {
	if auditSvc == nil {
		return
	}
	objectName := objectID
	if objectName == "" {
		objectName = objectType
	}
	if err := auditSvc.LogDataModification(ctx, actorID, tenantID, objectID, objectType, objectName, action, oldData, newData); err != nil {
		logging.GetLogger().Sugar().Warnw("failed to log modification audit", "error", err, "object_type", objectType, "action", action)
	}
}

// dispatchWebhookEvent sends webhook notifications when a webhook service is available.
func dispatchWebhookEvent(ctx context.Context, webhookSvc *webhooks.Service, eventType string, payload map[string]interface{}, attributes map[string]string) {
	if webhookSvc == nil {
		return
	}
	if payload == nil {
		payload = map[string]interface{}{}
	}
	if attributes == nil {
		attributes = map[string]string{}
	}
	evt := webhooks.Event{
		ID:         uuid.New(),
		Type:       eventType,
		Payload:    payload,
		Attributes: attributes,
	}
	if err := webhookSvc.DispatchEvent(ctx, evt); err != nil {
		logging.GetLogger().Sugar().Errorw("failed to dispatch webhook event", "error", err, "event_type", eventType)
	}
}

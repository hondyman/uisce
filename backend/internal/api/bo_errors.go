package api

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/metadata"
	"github.com/hondyman/uisce/backend/internal/msgcat"
)

// Business-object record errors are catalog messages (set 9000, seeded in
// db/migrations/20261028_001_message_catalog_bo.up.sql): the client gets
// the text in its language, a code and a correlation ID; the cause is
// logged, never sent.
const setBO = 9000

func boNotFound(boKey string) *msgcat.Error {
	return msgcat.New(setBO, 1, boKey).WithStatus(http.StatusNotFound)
}

func boRecordNotFound(boKey string) *msgcat.Error {
	return msgcat.New(setBO, 2, boKey).WithStatus(http.StatusNotFound)
}

func boUnknownField(field string) *msgcat.Error { return msgcat.New(setBO, 3, field) }

func boNoFields() *msgcat.Error { return msgcat.New(setBO, 4) }

func boRuleRejected(rules string) *msgcat.Error {
	return msgcat.New(setBO, 5, rules).WithStatus(http.StatusUnprocessableEntity)
}

func boRequiredMissing(fields string) *msgcat.Error {
	return msgcat.New(setBO, 6, fields).WithStatus(http.StatusUnprocessableEntity)
}

func boTooManyRecords(max int) *msgcat.Error {
	return msgcat.New(setBO, 7, max).WithStatus(http.StatusRequestEntityTooLarge)
}

func boBadMode() *msgcat.Error { return msgcat.New(setBO, 8) }

func boUpsertNeedsKeys() *msgcat.Error { return msgcat.New(setBO, 9) }

func boBadKeyField(field string) *msgcat.Error { return msgcat.New(setBO, 10, field) }

func boRecordsUnavailable(boKey string) *msgcat.Error {
	return msgcat.New(setBO, 11, boKey).WithStatus(http.StatusServiceUnavailable)
}

func boRelationshipNotFound(relKey, boKey string) *msgcat.Error {
	return msgcat.New(setBO, 12, relKey, boKey).WithStatus(http.StatusNotFound)
}

func boNotChanged() *msgcat.Error {
	return msgcat.New(setBO, 13).WithStatus(http.StatusNotFound)
}

func boEnforcementUnavailable() *msgcat.Error {
	return msgcat.New(setBO, 14).WithStatus(http.StatusServiceUnavailable)
}

func boMissingKeyField(field string) *msgcat.Error { return msgcat.New(setBO, 15, field) }

func boNotRelated(boKey string) *msgcat.Error {
	return msgcat.New(setBO, 16, boKey).WithStatus(http.StatusUnprocessableEntity)
}

// tenantError turns a tenant-resolution failure into its catalog message.
func tenantError(err error) error {
	var te *tenantResolutionError
	if !errors.As(err, &te) {
		return err
	}
	switch te.status {
	case http.StatusForbidden:
		return msgcat.NotPermitted("access to this organization").Wrap(err)
	case http.StatusBadRequest:
		return msgcat.InvalidInput("organization identifier").Wrap(err)
	default:
		return msgcat.Unauthenticated().Wrap(err)
	}
}

// writeError maps an enforced write's error: a rule rejection or missing
// required fields name what failed; anything else is internal.
func writeError(err error) error {
	var rej *metadata.RuleRejectionError
	var req *metadata.RequiredFieldsError
	switch {
	case errors.As(err, &rej):
		return boRuleRejected(joinComma(rej.Rules)).WithDetail("rules", rej.Rules).Wrap(err)
	case errors.As(err, &req):
		return boRequiredMissing(joinComma(req.Fields)).WithDetail("missing", req.Fields).Wrap(err)
	case errors.Is(err, metadata.ErrNoRowWritten):
		return boNotChanged().Wrap(err)
	case errors.Is(err, errNoRuleEnforcer):
		return boEnforcementUnavailable().Wrap(err)
	}
	return err
}

func joinComma(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ", "
		}
		out += v
	}
	return out
}

// fail answers with err rendered from the message catalog. Without a
// catalog (tests), msgcat still answers with the code and never the cause.
func (h *BOCRUDHandler) fail(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, err error) {
	tenant := ""
	if tenantID != uuid.Nil {
		tenant = tenantID.String()
	}
	h.catalog.WriteError(w, r, tenant, err)
}

// rowError renders one bulk row's failure. A cause that is not a catalog
// message (a database error, say) is logged against ref and answered as
// the internal-error message, so no row ever carries a raw error.
func (h *BOCRUDHandler) rowError(r *http.Request, tenantID uuid.UUID, ref string, index int, err error) (code, message string) {
	var me *msgcat.Error
	if !errors.As(writeError(err), &me) {
		logging.GetLogger().Sugar().Errorw("bulk row failed", "correlation_id", ref, "tenant", tenantID.String(), "index", index, "error", err)
		me = msgcat.Internal(ref)
	}
	out := h.catalog.Render(r.Context(), tenantID.String(), msgcat.Preferences(r.Header.Get("Accept-Language")), me, ref)
	return out.Code, out.Message
}

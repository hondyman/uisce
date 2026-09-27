package mastering

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/analytics"
)

// PlatformCatalog reads the business object's field -> column map from the
// platform catalog (the MAPS_TO edges of its canonical binding).
type PlatformCatalog struct{ DB *sqlx.DB }

// AttrFields maps each BO field to the anchor column it masters. The BO is
// the tenant's own, else the gold copy's.
func (c PlatformCatalog) AttrFields(ctx context.Context, tenantID, boKey, anchorTable, binding string) (map[string]string, error) {
	var boID string
	err := c.DB.GetContext(ctx, &boID, `SELECT id::text FROM public.business_objects
		WHERE bo_key = $2 AND (tenant_id::text = $1 OR tenant_id = public.uisce_gold_copy_tenant_id())
		ORDER BY (tenant_id::text = $1) DESC LIMIT 1`, tenantID, boKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, msgNoProfile(boKey)
	}
	if err != nil {
		return nil, err
	}
	if binding != "" {
		// A named binding's resolved field bindings: the fields actually
		// mapped to that binding's table (a field it doesn't map is absent,
		// never borrowed from another binding).
		var bindingID string
		err := c.DB.GetContext(ctx, &bindingID, `SELECT bo_binding_id::text FROM public.business_object_binding
			WHERE bo_id::text = $1 AND binding_name = $2 AND is_active IS NOT FALSE
			ORDER BY (tenant_id::text = $3) DESC LIMIT 1`, boID, binding, tenantID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, msgNoBOBinding(boKey, binding)
		}
		if err != nil {
			return nil, err
		}
		return analytics.ResolveSemanticFieldMapForBinding(ctx, c.DB, bindingID)
	}
	return analytics.ResolveSemanticFieldMap(ctx, c.DB, boID, "/"+strings.Replace(anchorTable, ".", "/", 1))
}

// GoldCopyTenant returns the gold-copy tenant id from the platform.
func (c PlatformCatalog) GoldCopyTenant(ctx context.Context) (string, error) {
	var id string
	err := c.DB.GetContext(ctx, &id, `SELECT public.uisce_gold_copy_tenant_id()::text`)
	return id, err
}

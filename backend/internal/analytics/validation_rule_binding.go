package analytics

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Binding scope for validation rules.
//
// A rule's field references are semantic terms, resolved to physical columns per binding at
// evaluation time, so one rule can run against every binding of a BO. Some rules should not:
// MDM sourcing rules, for example, only make sense on the MDM binding. A rule can therefore
// carry BindingIDs (business_object_binding.bo_binding_id). Empty means "every binding".

type bindingCtxKey struct{}

// WithBinding returns a context that names the binding a write or evaluation is going through.
// Without it, the active binding is inferred from the BO's driving table.
func WithBinding(ctx context.Context, bindingID string) context.Context {
	return context.WithValue(ctx, bindingCtxKey{}, bindingID)
}

// BindingFromContext returns the binding named by WithBinding, or "".
func BindingFromContext(ctx context.Context) string {
	id, _ := ctx.Value(bindingCtxKey{}).(string)
	return id
}

// RuleScopeApplies reports whether a rule scoped to scope runs for the active binding.
// An empty scope always applies. A scoped rule with no known active binding is reported as
// undetermined rather than skipped: the engine's rule is that a rule that cannot be evaluated is
// never a silent pass, and silently skipping a scoped rule is the same failure.
func RuleScopeApplies(scope []string, activeBindingID string) (applies, undetermined bool) {
	if len(scope) == 0 {
		return true, false
	}
	if activeBindingID == "" {
		return false, true
	}
	for _, id := range scope {
		if id == activeBindingID {
			return true, false
		}
	}
	return false, false
}

// ActiveBinding is the binding a write is going through and its driving table's path.
type ActiveBinding struct {
	ID          string `db:"id"`
	DrivingPath string `db:"driving_path"`
}

// ResolveActiveBinding finds the active binding of a BO: the one named on ctx (WithBinding) if
// any, otherwise the active binding whose driving table is drivingTable. It returns (nil, nil)
// when no binding row matches, which is the case for BOs that have not been bound yet.
func ResolveActiveBinding(ctx context.Context, db *sqlx.DB, tenantID, boID, drivingTable string) (*ActiveBinding, error) {
	const base = `
		SELECT b.bo_binding_id::text AS id, n.qualified_path AS driving_path
		FROM public.business_object_binding b
		JOIN public.catalog_node n ON n.id = b.driving_node_id
		WHERE b.tenant_id = $1::uuid AND b.bo_id = $2::uuid AND b.is_active`
	var ab ActiveBinding
	var err error
	if id := BindingFromContext(ctx); id != "" {
		err = db.GetContext(ctx, &ab, base+` AND b.bo_binding_id = $3::uuid`, tenantID, boID, id)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("binding %s is not an active binding of BO %s", id, boID)
		}
	} else {
		err = db.GetContext(ctx, &ab, base+` AND n.qualified_path = $3 ORDER BY b.is_core DESC, b.bo_binding_id LIMIT 1`, tenantID, boID, drivingTable)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
	}
	if err != nil {
		return nil, fmt.Errorf("resolve active binding for BO %s: %w", boID, err)
	}
	return &ab, nil
}

// validateBindingScope rejects a rule whose binding scope names something that is not a binding of
// its BO in this tenant. A typo'd or foreign binding id would otherwise produce a rule that
// silently never applies.
func (s *ValidationRuleService) validateBindingScope(ctx context.Context, tenantID, boName string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			return fmt.Errorf("binding %s is listed more than once", id)
		}
		seen[id] = true
	}
	var found []string
	err := s.db.SelectContext(ctx, &found, `
		SELECT b.bo_binding_id::text
		FROM public.business_object_binding b
		JOIN public.business_objects bo ON bo.id = b.bo_id
		WHERE b.tenant_id = $1::uuid AND (bo.bo_key = $2 OR bo.bo_name = $2)
		  AND bo.tenant_id = $1::uuid AND b.bo_binding_id::text = ANY($3)`, tenantID, boName, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("validate binding scope: %w", err)
	}
	ok := map[string]bool{}
	for _, id := range found {
		ok[id] = true
	}
	for _, id := range ids {
		if !ok[id] {
			return fmt.Errorf("binding %s is not a binding of BO %q in this tenant", id, boName)
		}
	}
	return nil
}

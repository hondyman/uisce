package security

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
)

// A placeholder or garbage datasource id is "not available", never a
// database error, and is rejected before any query runs.
func TestDBDatasourceResolver_NonUUIDIsNotAvailable(t *testing.T) {
	r := NewDBDatasourceResolver(sqlx.NewDb(&sql.DB{}, "postgres"))
	for _, id := range []string{"none", "ds-123", "11111111-1111-1111-1111-11111111111"} {
		_, err := r.Resolve(context.Background(), id)
		if !errors.Is(err, ErrDatasourceNotAvailable) {
			t.Errorf("Resolve(%q) = %v, want ErrDatasourceNotAvailable", id, err)
		}
	}
}

type fixedResolver struct{ tenant string }

func (f fixedResolver) Resolve(_ context.Context, id string) (*ResolvedDatasource, error) {
	return &ResolvedDatasource{TenantID: f.tenant, InstanceID: "i", ProductID: "p", DatasourceID: id}, nil
}

// Another tenant's datasource is not available to the caller; a global
// admin may still use it.
func TestBuildContext_OtherTenantsDatasourceIsNotAvailable(t *testing.T) {
	req := BuildContextRequest{DatasourceID: "ds", Region: "us-east-1"}
	_, err := BuildContext(context.Background(), AuthInfo{UserID: "u", TenantIDs: []string{"mine"}}, req, fixedResolver{"theirs"})
	if !errors.Is(err, ErrDatasourceNotAvailable) {
		t.Fatalf("err = %v, want ErrDatasourceNotAvailable", err)
	}
	if _, err := BuildContext(context.Background(), AuthInfo{UserID: "u", Roles: []string{"global_admin"}}, req, fixedResolver{"theirs"}); err != nil {
		t.Fatalf("global admin: %v", err)
	}
}

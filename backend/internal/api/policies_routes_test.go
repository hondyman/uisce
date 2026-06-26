package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	coremodels "github.com/hondyman/semlayer/backend/models"
)

type fakeCollabSvc struct{}

func (f *fakeCollabSvc) ListAccessPolicies(ctx context.Context) ([]coremodels.AccessControlPolicy, error) {
	return []coremodels.AccessControlPolicy{{PolicyID: "p1"}}, nil
}
func (f *fakeCollabSvc) GetAccessPolicyByID(ctx context.Context, id uuid.UUID) (*coremodels.AccessControlPolicy, error) {
	return &coremodels.AccessControlPolicy{PolicyID: "by-id"}, nil
}
func (f *fakeCollabSvc) GetAccessPolicyBySlug(ctx context.Context, slug string) (*coremodels.AccessControlPolicy, error) {
	return &coremodels.AccessControlPolicy{PolicyID: slug}, nil
}
func (f *fakeCollabSvc) DeleteAccessPolicy(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeCollabSvc) CreateAccessPolicy(ctx context.Context, policy *coremodels.AccessControlPolicy) (*coremodels.AccessControlPolicy, error) {
	return policy, nil
}
func (f *fakeCollabSvc) UpdateAccessPolicy(ctx context.Context, policy *coremodels.AccessControlPolicy) (*coremodels.AccessControlPolicy, error) {
	return policy, nil
}

func TestRegisterPolicyRoutes_ListPolicies(t *testing.T) {
	r := chi.NewRouter()
	srv := &Server{}
	svc := &fakeCollabSvc{}
	routes := NewRoutes()
	routes.RegisterPolicyRoutes(r, srv, svc)

	req := httptest.NewRequest(http.MethodGet, "/policies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d, body=%s", w.Code, w.Body.String())
	}
}

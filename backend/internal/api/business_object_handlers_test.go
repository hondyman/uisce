package api_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	httpapi "github.com/hondyman/semlayer/backend/internal/api"
	catalogmeta "github.com/hondyman/semlayer/backend/internal/metadata"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/stretchr/testify/require"
)

// fakeService implements BOService for testing
type fakeService struct {
	parent                  *models.BusinessObjectDefinition
	list                    []*models.BusinessObjectDefinition
	ListBusinessObjectsFunc func(ctx context.Context, secCtx *security.Context) ([]*models.BusinessObjectDefinition, error)
}

func (f *fakeService) GetBusinessObject(ctx context.Context, secCtx *security.Context, boKey string) (*models.BusinessObjectDefinition, error) {
	return f.parent, nil
}
func (f *fakeService) ListBusinessObjects(ctx context.Context, secCtx *security.Context) ([]*models.BusinessObjectDefinition, error) {
	if f.ListBusinessObjectsFunc != nil {
		return f.ListBusinessObjectsFunc(ctx, secCtx)
	}
	return f.list, nil
}
func (f *fakeService) CreateBusinessObject(ctx context.Context, secCtx *security.Context, req models.CreateBusinessObjectRequest, userID string) (*models.BusinessObjectDefinition, error) {
	return nil, nil
}
func (f *fakeService) UpdateBusinessObject(ctx context.Context, secCtx *security.Context, boKey string, req models.UpdateBusinessObjectRequest, userID string) (*models.BusinessObjectDefinition, error) {
	return nil, nil
}
func (f *fakeService) DeleteBusinessObject(ctx context.Context, secCtx *security.Context, boKey, userID string) error {
	return nil
}
func (f *fakeService) RenameSubtype(ctx context.Context, secCtx *security.Context, boKey, subtypeKey, newName, userID string) (*models.BusinessObjectDefinition, error) {
	return nil, nil
}
func (f *fakeService) DeleteSubtype(ctx context.Context, secCtx *security.Context, boKey, subtypeKey, userID string) (*models.BusinessObjectDefinition, error) {
	return nil, nil
}

func (f *fakeService) GetBusinessObjectRelationships(ctx context.Context, secCtx *security.Context, boID string) (*catalogmeta.BORelationshipsResponse, error) {
	return &catalogmeta.BORelationshipsResponse{}, nil
}

func (f *fakeService) ListBusinessObjectsComposed(ctx context.Context, secCtx *security.Context) ([]*models.BusinessObjectDefinition, error) {
	return f.ListBusinessObjects(ctx, secCtx)
}

// withAuth and withValidHeaders are used from test_helpers_test.go

func TestGetBusinessObjectHandler_AttachesChildren(t *testing.T) {
	parent := &models.BusinessObjectDefinition{ID: "parent1", Key: "parent_key", Name: "Parent", Subtypes: map[string]models.SubtypeDefinition{}}
	child := &models.BusinessObjectDefinition{ID: "child1", Key: "child_key", Name: "Child", ParentID: sql.NullString{String: "parent1", Valid: true}, CustomFields: []models.FieldDefinition{{Key: "f1", Name: "Field 1"}}}

	svc := &fakeService{parent: parent, list: []*models.BusinessObjectDefinition{child}}
	h := httpapi.NewBusinessObjectHandler(svc, &mockResolver{})
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/business-objects/parent1", nil)
	req = withValidHeaders(req, "ten", "ds1")
	req = withAuth(req, "ten")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Result().StatusCode)
	var bo models.BusinessObjectDefinition
	err := json.NewDecoder(w.Body).Decode(&bo)
	require.NoError(t, err)
	// should attach child
	require.NotNil(t, bo.Subtypes)
	require.Equal(t, 1, len(bo.Subtypes))
	_, exists := bo.Subtypes["child_key"]
	require.True(t, exists)
}

func TestListBusinessObjects_UsesNewDatasourceHeader(t *testing.T) {
	captured := ""
	f := &fakeService{list: []*models.BusinessObjectDefinition{&models.BusinessObjectDefinition{ID: "b1", Key: "b1", Name: "BO1"}}}
	// override ListBusinessObjects to capture datasource
	f.ListBusinessObjectsFunc = func(ctx context.Context, secCtx *security.Context) ([]*models.BusinessObjectDefinition, error) {
		captured = secCtx.DatasourceID
		return f.list, nil
	}
	h := httpapi.NewBusinessObjectHandler(f, &mockResolver{})
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/business-objects", nil)
	req = withValidHeaders(req, "ten", "ds-123")
	req = withAuth(req, "ten")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Result().StatusCode)
	require.Equal(t, "ds-123", captured)
}

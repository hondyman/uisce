package api_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	httpapi "github.com/hondyman/uisce/backend/internal/api"
	catalogmeta "github.com/hondyman/uisce/backend/internal/metadata"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/security"
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

func (f *fakeService) QueryBORecords(ctx context.Context, secCtx *security.Context, boIDOrKey string, req models.BORecordQueryRequest) (*models.BORecordQueryResponse, error) {
	return &models.BORecordQueryResponse{}, nil
}

func (f *fakeService) CreateBORecord(ctx context.Context, secCtx *security.Context, boIDOrKey string, req models.BOCrudRecordRequest, userID string) (map[string]interface{}, error) {
	return map[string]interface{}{"id": "1"}, nil
}

func (f *fakeService) UpdateBORecord(ctx context.Context, secCtx *security.Context, boIDOrKey string, recordID string, req models.BOCrudRecordRequest, userID string) (map[string]interface{}, error) {
	return map[string]interface{}{"id": recordID}, nil
}

func (f *fakeService) DeleteBORecord(ctx context.Context, secCtx *security.Context, boIDOrKey string, recordID string, userID string) error {
	return nil
}

func (f *fakeService) IntrospectTable(ctx context.Context, secCtx *security.Context, tableName string) (*models.TableIntrospectionResponse, error) {
	return &models.TableIntrospectionResponse{}, nil
}


func (f *fakeService) GetBODelta(ctx context.Context, secCtx *security.Context, boIDOrKey string) (*models.BODeltaResponse, error) {
	return &models.BODeltaResponse{}, nil
}

func (f *fakeService) SynthesizeBOWithAI(ctx context.Context, secCtx *security.Context, req models.BOAISynthesizeRequest) (*models.BOAISynthesizeResponse, error) {
	return &models.BOAISynthesizeResponse{}, nil
}

func (f *fakeService) TranslateNLToQueryDef(ctx context.Context, secCtx *security.Context, req models.BOAINLQRequest) (*models.BOAINLQResponse, error) {
	return &models.BOAINLQResponse{}, nil
}

func (f *fakeService) ExplainDeltaWithAI(ctx context.Context, secCtx *security.Context, req models.BOAIExplainDeltaRequest) (*models.BOAIExplainDeltaResponse, error) {
	return &models.BOAIExplainDeltaResponse{}, nil
}

func (f *fakeService) DetectAnomaliesWithAI(ctx context.Context, secCtx *security.Context, req models.BOAIAnomalyDetectRequest) (*models.BOAIAnomalyDetectResponse, error) {
	return &models.BOAIAnomalyDetectResponse{}, nil
}

func (f *fakeService) GetBOWorkflowStatus(ctx context.Context, secCtx *security.Context, boIDOrKey string) (*models.BOWorkflowStatusResponse, error) {
	return &models.BOWorkflowStatusResponse{}, nil
}

func (f *fakeService) ExecuteWorkflowAction(ctx context.Context, secCtx *security.Context, boIDOrKey string, req models.BOWorkflowActionRequest, userID string) (*models.BOWorkflowStatusResponse, error) {
	return &models.BOWorkflowStatusResponse{}, nil
}

func (f *fakeService) DiscoverBindingScope(ctx context.Context, secCtx *security.Context, boIDOrKey string, drivingNodeID string) (*models.BOScopeDiscoveryResponse, error) {
	return &models.BOScopeDiscoveryResponse{}, nil
}

func (f *fakeService) ValidatePublishGate(ctx context.Context, secCtx *security.Context, boIDOrKey string) (*models.BOPublishGateValidationResponse, error) {
	return &models.BOPublishGateValidationResponse{CanPublish: true}, nil
}

func (f *fakeService) GetMultiBackendConfiguration(ctx context.Context, secCtx *security.Context, boIDOrKey string) (*models.BOMultiBackendConfiguration, error) {
	return &models.BOMultiBackendConfiguration{}, nil
}

func (f *fakeService) PerformGraphRAGContext(ctx context.Context, secCtx *security.Context, req models.GraphRAGContextRequest) (*models.GraphRAGContextResponse, error) {
	return &models.GraphRAGContextResponse{}, nil
}

func (f *fakeService) SimulateLineageImpact(ctx context.Context, secCtx *security.Context, req models.BOLineageImpactSimulationRequest) (*models.BOLineageImpactSimulationResponse, error) {
	return &models.BOLineageImpactSimulationResponse{}, nil
}

func (f *fakeService) GenerateBOArtifacts(ctx context.Context, secCtx *security.Context, boIDOrKey string) (*models.BOArtifactGenerationResponse, error) {
	return &models.BOArtifactGenerationResponse{}, nil
}

func (f *fakeService) EvaluateQueryCost(ctx context.Context, secCtx *security.Context, req models.BOQueryCostEvaluationRequest) (*models.BOQueryCostEvaluationResponse, error) {
	return &models.BOQueryCostEvaluationResponse{ComplexityScore: 20, CostBand: models.CostBandLow}, nil
}

func (f *fakeService) DetectSchemaDrift(ctx context.Context, secCtx *security.Context, boIDOrKey string) (*models.BODataQualitySentinelResponse, error) {
	return &models.BODataQualitySentinelResponse{}, nil
}

func (f *fakeService) ApplyDriftRepairPatch(ctx context.Context, secCtx *security.Context, req models.BODriftRepairPatchRequest, userID string) (*models.BODriftRepairPatchResponse, error) {
	return &models.BODriftRepairPatchResponse{Status: "APPLIED"}, nil
}

func (f *fakeService) RunLakehouseCompaction(ctx context.Context, secCtx *security.Context, boIDOrKey string) (*models.LakehouseMaintenanceReport, error) {
	return &models.LakehouseMaintenanceReport{Status: "COMPLETED"}, nil
}






// withAuth and withValidHeaders are used from test_helpers_test.go

func TestGetBusinessObjectHandler_AttachesChildren(t *testing.T) {
	parent := &models.BusinessObjectDefinition{ID: "parent1", Key: "parent_key", Name: "Parent", Subtypes: map[string]models.SubtypeDefinition{}}
	child := &models.BusinessObjectDefinition{ID: "child1", Key: "child_key", Name: "Child", ParentID: sql.NullString{String: "parent1", Valid: true}, CustomFields: []models.FieldDefinition{{Key: "f1", Name: "Field 1"}}}

	svc := &fakeService{parent: parent, list: []*models.BusinessObjectDefinition{child}}
	h := httpapi.NewBusinessObjectHandler(svc, &mockResolver{}, nil, nil)
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
	h := httpapi.NewBusinessObjectHandler(f, &mockResolver{}, nil, nil)
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

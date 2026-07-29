package lineage

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLineageRepository struct {
	findUpstreamFn   func(ctx context.Context, rootID string, depth int) (*Graph, error)
	findDownstreamFn func(ctx context.Context, rootID string, depth int) (*Graph, error)

	upsertNodeFn func(ctx context.Context, node LineageNode) error
	upsertEdgeFn func(ctx context.Context, edge LineageEdge) error

	mu         sync.Mutex
	upsertNode LineageNode
	upsertEdge LineageEdge
}

func (m *mockLineageRepository) UpsertNode(ctx context.Context, node LineageNode) error {
	if m.upsertNodeFn != nil {
		return m.upsertNodeFn(ctx, node)
	}
	m.mu.Lock()
	m.upsertNode = node
	m.mu.Unlock()
	return nil
}
func (m *mockLineageRepository) UpsertEdge(ctx context.Context, edge LineageEdge) error {
	if m.upsertEdgeFn != nil {
		return m.upsertEdgeFn(ctx, edge)
	}
	m.mu.Lock()
	m.upsertEdge = edge
	m.mu.Unlock()
	return nil
}
func (m *mockLineageRepository) DeleteNode(ctx context.Context, id string) error {
	return nil
}
func (m *mockLineageRepository) DeleteEdge(ctx context.Context, fromID, toID, edgeType string) error {
	return nil
}
func (m *mockLineageRepository) FindDownstreamGraph(ctx context.Context, rootID string, depth int) (*Graph, error) {
	if m.findDownstreamFn != nil {
		return m.findDownstreamFn(ctx, rootID, depth)
	}
	return &Graph{}, nil
}
func (m *mockLineageRepository) FindUpstreamGraph(ctx context.Context, rootID string, depth int) (*Graph, error) {
	if m.findUpstreamFn != nil {
		return m.findUpstreamFn(ctx, rootID, depth)
	}
	return &Graph{}, nil
}
func (m *mockLineageRepository) FindBiDirectionalGraph(ctx context.Context, rootID string, depth int) (*Graph, error) {
	return &Graph{}, nil
}
func (m *mockLineageRepository) FindGraphByDatasource(ctx context.Context, datasourceID string) (*Graph, error) {
	return &Graph{}, nil
}
func (m *mockLineageRepository) SyncDatasource(ctx context.Context, datasourceID string) error {
	return nil
}

func TestNewImpactSimulator(t *testing.T) {
	svc := NewLineageService(&mockLineageRepository{})
	sim := NewImpactSimulator(svc)
	assert.NotNil(t, sim)
	assert.Equal(t, svc, sim.svc)
}

func TestSimulateImpact_DefaultsToDepth5(t *testing.T) {
	var capturedDepth int
	repo := &mockLineageRepository{
		findUpstreamFn: func(ctx context.Context, rootID string, depth int) (*Graph, error) {
			capturedDepth = depth
			return &Graph{}, nil
		},
		findDownstreamFn: func(ctx context.Context, rootID string, depth int) (*Graph, error) {
			capturedDepth = depth
			return &Graph{}, nil
		},
	}
	svc := NewLineageService(repo)
	sim := NewImpactSimulator(svc)

	report, err := sim.SimulateImpact(context.Background(), "tenant-1", "node-1", "drop_column", 0)
	require.NoError(t, err)
	assert.Equal(t, 5, capturedDepth)
	assert.NotNil(t, report)
	assert.Equal(t, "node-1", report.TargetNode)
	assert.Equal(t, "drop_column", report.Action)
	assert.Equal(t, 0, report.BlastRadiusSummary.TotalImpactedArtifacts)
}

func TestSimulateImpact_UpstreamError_Continues(t *testing.T) {
	repo := &mockLineageRepository{
		findUpstreamFn: func(ctx context.Context, rootID string, depth int) (*Graph, error) {
			return nil, errors.New("upstream error")
		},
		findDownstreamFn: func(ctx context.Context, rootID string, depth int) (*Graph, error) {
			return &Graph{Nodes: []LineageNode{{ID: "n1", Type: NodePage, Name: "dash"}}}, nil
		},
	}
	svc := NewLineageService(repo)
	sim := NewImpactSimulator(svc)

	report, err := sim.SimulateImpact(context.Background(), "tenant-1", "node-1", "drop_column", 3)
	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Len(t, report.UpstreamNodes, 0)
	assert.Len(t, report.DownstreamNodes, 1)
}

func TestSimulateImpact_DownstreamError_Continues(t *testing.T) {
	repo := &mockLineageRepository{
		findUpstreamFn: func(ctx context.Context, rootID string, depth int) (*Graph, error) {
			return &Graph{Nodes: []LineageNode{{ID: "n1", Type: NodeBO, Name: "order"}}}, nil
		},
		findDownstreamFn: func(ctx context.Context, rootID string, depth int) (*Graph, error) {
			return nil, errors.New("downstream error")
		},
	}
	svc := NewLineageService(repo)
	sim := NewImpactSimulator(svc)

	report, err := sim.SimulateImpact(context.Background(), "tenant-1", "node-1", "drop_column", 3)
	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Len(t, report.UpstreamNodes, 1)
	assert.Len(t, report.DownstreamNodes, 0)
}

func TestSimulateImpact_ScoresCorrectly(t *testing.T) {
	tenant := "tenant-1"
	repo := &mockLineageRepository{
		findUpstreamFn: func(ctx context.Context, rootID string, depth int) (*Graph, error) {
			return &Graph{Nodes: []LineageNode{
				{ID: "up1", Type: NodeTable, Name: "orders", TenantID: &tenant},
			}}, nil
		},
		findDownstreamFn: func(ctx context.Context, rootID string, depth int) (*Graph, error) {
			return &Graph{Nodes: []LineageNode{
				{ID: "d1", Type: NodePage, Name: "Dashboard"},
				{ID: "d2", Type: NodePage, Name: "AdminPanel"},
				{ID: "d3", Type: NodeAPIEndpoint, Name: "/api/users"},
				{ID: "d4", Type: NodeASOOpt, Name: "RevenueReport"},
				{ID: "d5", Type: NodeBO, Name: "OrderBO"},
			}}, nil
		},
	}
	svc := NewLineageService(repo)
	sim := NewImpactSimulator(svc)

	report, err := sim.SimulateImpact(context.Background(), tenant, "node-1", "drop_column", 3)
	require.NoError(t, err)

	assert.Equal(t, 5, report.BlastRadiusSummary.TotalImpactedArtifacts)
	domains := make([]ConsumerDomain, len(report.ImpactedConsumers))
	for i, c := range report.ImpactedConsumers {
		domains[i] = c.Domain
	}
	assert.Contains(t, domains, DomainReactDashboards)
	assert.Contains(t, domains, DomainDownstreamTenantQueries)
	assert.Contains(t, domains, DomainRegulatoryExporters)
	assert.Contains(t, domains, DomainBusinessObjects)
}

func TestComputeBlastRadiusSeverity(t *testing.T) {
	tests := []struct {
		score    float64
		expected BlastRadiusSeverity
	}{
		{0.0, BlastSeverityLow},
		{0.99, BlastSeverityLow},
		{1.0, BlastSeverityMedium},
		{3.9, BlastSeverityMedium},
		{4.0, BlastSeverityHigh},
		{7.9, BlastSeverityHigh},
		{8.0, BlastSeverityCritical},
		{100.0, BlastSeverityCritical},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, ComputeBlastRadiusSeverity(tt.score), "score=%v", tt.score)
	}
}

func TestWeightFor(t *testing.T) {
	assert.Equal(t, 10.0, WeightFor(DomainRegulatoryExporters))
	assert.Equal(t, 8.0, WeightFor(DomainReactDashboards))
	assert.Equal(t, 6.0, WeightFor(DomainDownstreamTenantQueries))
	assert.Equal(t, 5.0, WeightFor(DomainInternalAPIs))
	assert.Equal(t, 4.0, WeightFor(DomainBusinessObjects))
	assert.Equal(t, 1.0, WeightFor(DomainUnknown))
	assert.Equal(t, 1.0, WeightFor(ConsumerDomain("garbage")))
}

func TestCategorizeByConsumerDomain(t *testing.T) {
	nodes := []LineageNode{
		{ID: "n1", Type: NodePage, Name: "Dash1"},
		{ID: "n2", Type: NodePage, Name: "Dash2"},
		{ID: "n3", Type: NodeAPIEndpoint, Name: "API1"},
		{ID: "n4", Type: NodeASOOpt, Name: "ASO1"},
		{ID: "n5", Type: NodeBO, Name: "BO1"},
		{ID: "n6", Type: NodeTable, Name: "Table1"},
	}
	consumers := categorizeByConsumerDomain(nodes)

	domainMap := make(map[ConsumerDomain][]string)
	for _, c := range consumers {
		domainMap[c.Domain] = c.Artifacts
	}

	assert.Equal(t, []string{"Dash1", "Dash2"}, domainMap[DomainReactDashboards])
	assert.Equal(t, []string{"API1"}, domainMap[DomainDownstreamTenantQueries])
	assert.Equal(t, []string{"ASO1"}, domainMap[DomainRegulatoryExporters])
	assert.Equal(t, []string{"BO1"}, domainMap[DomainBusinessObjects])
	assert.Equal(t, []string{"Table1"}, domainMap[DomainUnknown])
}

func TestInferConsumerDomain(t *testing.T) {
	tests := []struct {
		node     LineageNode
		expected ConsumerDomain
	}{
		{LineageNode{Type: NodePage}, DomainReactDashboards},
		{LineageNode{Type: NodeAPIEndpoint}, DomainDownstreamTenantQueries},
		{LineageNode{Type: NodeASOOpt}, DomainRegulatoryExporters},
		{LineageNode{Type: NodeBO}, DomainBusinessObjects},
		{LineageNode{Type: NodeTable}, DomainUnknown},
		{LineageNode{Type: NodePage, Metadata: mustMarshal(map[string]interface{}{"consumer_domain": "REGULATORY_EXPORTERS"})}, DomainRegulatoryExporters},
		{LineageNode{Type: NodePage, Metadata: mustMarshal(map[string]interface{}{"consumer_domain": "BUSINESS_OBJECTS"})}, DomainBusinessObjects},
		{LineageNode{Type: NodeBO, Metadata: mustMarshal(map[string]interface{}{"consumer_domain": "INTERNAL_APIS"})}, DomainInternalAPIs},
		{LineageNode{Type: NodePage, Metadata: mustMarshal(map[string]interface{}{"consumer_domain": "GARBAGE"})}, DomainReactDashboards},
		{LineageNode{Type: NodeTable, Metadata: mustMarshal(map[string]interface{}{"consumer_domain": "DOWNSTREAM_TENANT_QUERIES"})}, DomainDownstreamTenantQueries},
		{LineageNode{Type: NodeTable, Metadata: []byte("not json")}, DomainUnknown},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, inferConsumerDomain(tt.node), "node=%+v", tt.node)
	}
}

func TestRiskDescription(t *testing.T) {
	assert.Equal(t, "UI components will render null values or fail to load.", riskDescription(DomainReactDashboards))
	assert.Equal(t, "Regulatory report generation will fail schema validation.", riskDescription(DomainRegulatoryExporters))
	assert.Equal(t, "Compilation AST will throw unbound semantic term errors.", riskDescription(DomainDownstreamTenantQueries))
	assert.Equal(t, "Internal API contracts will be violated, potentially breaking callers.", riskDescription(DomainInternalAPIs))
	assert.Equal(t, "Business Object definitions will reference invalid fields.", riskDescription(DomainBusinessObjects))
	assert.Equal(t, "Downstream consumers may experience degraded functionality.", riskDescription(DomainUnknown))
	assert.Equal(t, "Downstream consumers may experience degraded functionality.", riskDescription(ConsumerDomain("unknown")))
}

func TestImpactReport_Structure(t *testing.T) {
	tenant := "tenant-1"
	now := time.Now()
	report := &ImpactReport{
		NodeID: "test-node",
		AffectedBOs: []LineageNode{{ID: "bo1", Type: NodeBO, Name: "Order", TenantID: &tenant, CreatedAt: now}},
		AffectedPreAggs: []LineageNode{{ID: "pa1", Type: NodePreAgg, Name: "Agg1", TenantID: &tenant}},
		AffectedEntitlements: []LineageNode{{ID: "e1", Type: NodeEntitlement, Name: "Ent1"}},
		AffectedASOOptimizations: []LineageNode{{ID: "aso1", Type: NodeASOOpt, Name: "ASO1"}},
		AffectedPages: []LineageNode{{ID: "p1", Type: NodePage, Name: "Page1"}},
		AffectedAPIEndpoints: []LineageNode{{ID: "api1", Type: NodeAPIEndpoint, Name: "API1"}},
		AffectedTenants: []string{"tenant-1", "tenant-2"},
	}

	data, err := json.Marshal(report)
	require.NoError(t, err)

	var unmarshaled ImpactReport
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, "test-node", unmarshaled.NodeID)
	assert.Len(t, unmarshaled.AffectedBOs, 1)
	assert.Len(t, unmarshaled.AffectedTenants, 2)
}

func TestBlastRadiusReport_Structure(t *testing.T) {
	report := &BlastRadiusReport{
		TargetNode: "node-1",
		Action:     "drop_column",
		BlastRadiusSummary: BlastRadiusSummary{
			TotalImpactedArtifacts: 10,
			Severity:                BlastSeverityCritical,
			WeightedScore:           12.5,
		},
		UpstreamNodes:   []LineageNode{{ID: "up1", Type: NodeTable, Name: "orders"}},
		DownstreamNodes: []LineageNode{{ID: "dn1", Type: NodePage, Name: "Dash"}},
		ImpactedConsumers: []ImpactedConsumer{
			{Domain: DomainReactDashboards, Artifacts: []string{"Dash"}, Risk: "UI components will render null values or fail to load."},
		},
	}

	data, err := json.Marshal(report)
	require.NoError(t, err)

	var unmarshaled BlastRadiusReport
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, "node-1", unmarshaled.TargetNode)
	assert.Equal(t, "drop_column", unmarshaled.Action)
	assert.Equal(t, 10, unmarshaled.BlastRadiusSummary.TotalImpactedArtifacts)
	assert.Equal(t, BlastSeverityCritical, unmarshaled.BlastRadiusSummary.Severity)
	assert.Equal(t, 12.5, unmarshaled.BlastRadiusSummary.WeightedScore)
	assert.Len(t, unmarshaled.UpstreamNodes, 1)
	assert.Len(t, unmarshaled.DownstreamNodes, 1)
	assert.Len(t, unmarshaled.ImpactedConsumers, 1)
}

func TestLineageService_IngestBusinessObject(t *testing.T) {
	repo := &mockLineageRepository{}
	svc := NewLineageService(repo)

	deps := []uuid.UUID{uuid.New(), uuid.New()}
	meta := map[string]interface{}{"source": "test", "region": "us-east"}
	err := svc.IngestBusinessObject(context.Background(), uuid.New(), "OrderBO", "prod", "tenant-1", deps, meta)
	require.NoError(t, err)

	repo.mu.Lock()
	upserted := repo.upsertNode
	repo.mu.Unlock()

	assert.Equal(t, "OrderBO", upserted.Name)
	assert.Equal(t, NodeBO, upserted.Type)
	assert.Equal(t, "prod", upserted.Env)
	assert.NotNil(t, upserted.TenantID)
	assert.Equal(t, "tenant-1", *upserted.TenantID)
}

func TestLineageService_ImpactOfNode(t *testing.T) {
	tenant := "tenant-1"
	repo := &mockLineageRepository{
		findDownstreamFn: func(ctx context.Context, rootID string, depth int) (*Graph, error) {
			return &Graph{Nodes: []LineageNode{
				{ID: "bo1", Type: NodeBO, Name: "Order", TenantID: &tenant},
				{ID: "page1", Type: NodePage, Name: "Dashboard", TenantID: &tenant},
				{ID: "api1", Type: NodeAPIEndpoint, Name: "/api/orders", TenantID: &tenant},
				{ID: "tenant1", Type: NodeTenant, Name: "TenantA", TenantID: &tenant},
			}}, nil
		},
	}
	svc := NewLineageService(repo)

	report, err := svc.ImpactOfNode(context.Background(), "bo-orders", 3)
	require.NoError(t, err)

	assert.Equal(t, "bo-orders", report.NodeID)
	assert.Len(t, report.AffectedBOs, 1)
	assert.Len(t, report.AffectedPages, 1)
	assert.Len(t, report.AffectedAPIEndpoints, 1)
	assert.Len(t, report.AffectedTenants, 1)
	assert.Contains(t, report.AffectedTenants, "tenant-1")
}

func TestSimulateImpact_EmptyGraphs(t *testing.T) {
	repo := &mockLineageRepository{
		findUpstreamFn:   func(ctx context.Context, rootID string, depth int) (*Graph, error) { return &Graph{}, nil },
		findDownstreamFn: func(ctx context.Context, rootID string, depth int) (*Graph, error) { return &Graph{}, nil },
	}
	svc := NewLineageService(repo)
	sim := NewImpactSimulator(svc)

	report, err := sim.SimulateImpact(context.Background(), "tenant-1", "orphan-node", "drop_table", 2)
	require.NoError(t, err)
	assert.Equal(t, 0, report.BlastRadiusSummary.TotalImpactedArtifacts)
	assert.Equal(t, BlastSeverityLow, report.BlastRadiusSummary.Severity)
	assert.Empty(t, report.UpstreamNodes)
	assert.Empty(t, report.DownstreamNodes)
	assert.Empty(t, report.ImpactedConsumers)
}

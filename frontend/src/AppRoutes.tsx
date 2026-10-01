import React from "react";
import { Routes, Route, Navigate, useParams } from "react-router-dom";
import useBlockableNavigate from './components/RouteBlocker/useBlockableNavigate';
import { useLocale } from "./i18n/useLocale";
import { MicroBundleCatalogExample } from "./MicroBundleCatalogExample";
import { JITRequestPanelExample } from "./JITRequestPanelExample";
import { AccessExplanationExample } from "./AccessExplanationExample";
import ConversationalQueryPage from "./pages/ConversationalQueryPage";
import ManagementPage from "./features/fabric/pages/preaggregations/ManagementPage";
import BundleExplorer from "./components/BundleExplorer";
import CalculationsLibraryPage from "./features/fabric/pages/CalculationsLibraryPage";
import ProtectedRoute from "./components/ProtectedRoute";
import CalculatedFieldBuilderPage from "./pages/CalculatedFieldBuilderPage";
import IPWhitelistManagementPage from "./features/fabric/pages/IPWhitelistManagementPage";
import DashboardPage from "./features/fabric/pages/DashboardPage";
import AuditLogsPage from "./features/fabric/pages/AuditLogsPage";
import SettingsPage from "./features/fabric/pages/SettingsPage";
import TenantsManagementPage from "./features/fabric/pages/TenantsManagementPage";
import ViewsCatalogPage from "./features/views/pages/ViewsCatalogPage";
import ViewDetailsPage from "./features/views/pages/ViewDetailsPage";
import BundleListPage from "./pages/bundles/BundleListPage";
import BundleEditor from "./pages/bundles/BundleEditor";
import RoleListPage from "./pages/roles/RoleListPage";
import RoleEditorPage from "./pages/roles/RoleEditorPage";
import DomainsManagementPage from "./features/core/pages/DomainsManagementPage";
import SemanticMapperPage from "./features/core/pages/SemanticMapperPage";
import { TenantDetailPageV2 } from "./features/tenants/pages/TenantDetailPageV2";
import { SemanticCatalogPage } from "./pages/SemanticCatalogPage";

import GlossaryExplorer from "./features/glossary/GlossaryExplorer";
import BusinessTermsExplorer from "./features/glossary/BusinessTermsExplorer";
import ApiInventoryPage from "./features/catalog/pages/ApiInventoryPage";
import { AbbreviationsPage } from "./pages/core/AbbreviationsPage";
import { CatalogNodeTypesPage } from "./pages/catalog/CatalogNodeTypesPage";
import { CatalogEdgeTypesPage } from "./pages/catalog/CatalogEdgeTypesPage";
import { NodeTypeDetailPage } from "./pages/catalog/NodeTypeDetailPage";
import { EdgeTypeDetailPage } from "./pages/catalog/EdgeTypeDetailPage";
import { AIBusinessTermSuggestionsPage } from "./pages/catalog/AIBusinessTermSuggestionsPage";
import { BusinessTermDetailPage } from "./pages/catalog/BusinessTermDetailPage";
import CustomComponentPage from "./pages/CustomComponentPage";
import AdvancedRuleBuilderPage from "./pages/AdvancedRuleBuilderPage";
import SystemValidationsPage from "./pages/SystemValidationsPage";
import UisceBuilder from "./features/uisce-builder/UisceBuilder";
import { InvestmentValidationPage } from "./pages/InvestmentValidationPage";
import ApprovalWorkflowDashboard from "./pages/ApprovalWorkflowDashboard";
import { WorkflowDesignerPage } from "./features/workflow/pages/WorkflowDesignerPage";
import { DynamicDataProductPage } from "./pages/DynamicDataProductPage";

import { NotificationCenterPage } from "./features/workflow/pages/NotificationCenterPage";
import { NotificationTemplateEditorPage } from "./features/workflow/pages/NotificationTemplateEditorPage";
import { NotificationPreferencesPage } from "./features/workflow/pages/NotificationPreferencesPage";
import { SLADashboardPage } from "./features/workflow/pages/SLADashboardPage";
import { RegulatorDashboardPage } from "./features/workflow/pages/RegulatorDashboardPage";
import { ProcessCatalogPage } from "./features/workflow/pages/ProcessCatalogPage";
import { AuditExplorerPage } from "./features/workflow/pages/AuditExplorerPage";
import AuditExplorer from "./components/audit/AuditExplorer";
import TemporalOpsPage from "./features/admin/pages/TemporalOpsPage";
import SeedingPage from "./features/admin/pages/SeedingPage";
const QueryLibrary = React.lazy(() => import("./features/query-builder/pages/QueryLibrary"));
const SavedQueryEditor = React.lazy(() => import("./features/query-builder/pages/SavedQueryEditor"));
const SqlStudioPage = React.lazy(() => import("./pages/analytical/SqlStudioPage"));
const SemanticCatalogDetailPage = React.lazy(() => import("./pages/analytical/SemanticCatalogDetailPage"));
const PipelinesPage = React.lazy(() => import("./pages/analytical/PipelinesPage"));
const ReportsPage = React.lazy(() => import("./pages/analytical/ReportsPage"));
// Metrics Console imports
const MetricsConsolePage = React.lazy(() => import("./pages/MetricsConsolePage"));
const MetricDetailPage = React.lazy(() => import("./pages/MetricDetailPage"));
const MetricCreatePage = React.lazy(() => import("./pages/MetricCreatePage"));
const MetricEditPage = React.lazy(() => import("./pages/MetricEditPage"));
const MetricCalcConsole = React.lazy(() => import("./pages/metrics/MetricCalcConsole"));
const SemanticEnrichmentWizard = React.lazy(() => import("./pages/SemanticEnrichment/SemanticEnrichmentWizard"));
const NLQPage = React.lazy(() => import("./pages/nlq/NLQPage"));
const LLMConfigPage = React.lazy(() => import("./pages/admin/LLMConfigPage"));
const MessageCatalogPage = React.lazy(() => import("./features/message-catalog/MessageCatalogPage"));

const Feed = React.lazy(() => import("./features/feed/components/Feed").then(m => ({ default: m.Feed })));
const ApprovalInboxPage = React.lazy(() => import("./features/wealth/pages/ApprovalInboxPage").then(m => ({ default: m.ApprovalInboxPage })));
const GenUIApprovalInboxPage = React.lazy(() => import("./features/workflow/pages/GenUIApprovalInboxPage").then(m => ({ default: m.GenUIApprovalInboxPage })));
const GenUIProposalDemoPage = React.lazy(() => import("./features/workflow/pages/GenUIProposalDemoPage").then(m => ({ default: m.GenUIProposalDemoPage })));
const GenUIChatPage = React.lazy(() => import("./pages/GenUIChatPage"));
const FactorAnalysisPage = React.lazy(() => import("./features/analytics/pages/FactorAnalysisPage").then(m => ({ default: m.FactorAnalysisPage })));
const AdvisorDashboard = React.lazy(() => import("./pages/AdvisorDashboard"));
const DirectIndexingPage = React.lazy(() => import("./pages/investment/DirectIndexingPage"));
const ValuesProfileEditor = React.lazy(() => import("./pages/investment/ValuesProfileEditor"));
// Crypto Platform
const CryptoDashboard = React.lazy(() => import("./features/crypto/CryptoDashboard"));
const CryptoPortfolioCenter = React.lazy(() => import("./features/crypto/CryptoPortfolioCenter"));
// Secrets Management
const SecretsConfigPage = React.lazy(() => import("./features/secrets").then(m => ({ default: m.SecretsConfigPage })));
const SecretsAuditPage = React.lazy(() => import("./features/secrets").then(m => ({ default: m.SecretsAuditPage })));
const SecretsMonitoringPage = React.lazy(() => import("./features/secrets").then(m => ({ default: m.SecretsMonitoringPage })));
// Reporting
const WorldClassReportBuilder = React.lazy(() => import("./features/reporting/components/SelfServiceReportBuilder").then(m => ({ default: m.WorldClassReportBuilder })));

// BP Framework Console
const BPConsolePage = React.lazy(() => import("./features/bp-console/pages/BPConsolePage"));
const ReportLibrary = React.lazy(() => import("./features/reporting/components/ReportLibrary").then(m => ({ default: m.ReportLibrary })));
const ReportBuilderPage = React.lazy(() => import("./pages/ReportBuilderPage"));
const DataExplorer = React.lazy(() => import('./components/reporting/DataExplorer').then(m => ({ default: m.DataExplorer })));
const SemanticModelManager = React.lazy(() => import('./features/semantic/components/SemanticModelManager').then(m => ({ default: m.SemanticModelManager })));

const EntityManagerPage = React.lazy(() => import("./features/admin/pages/EntityManagerPage"));
const EntityDetailsPage = React.lazy(() => import("./pages/EntityDetailsPage"));
const BusinessObjectsPage = React.lazy(() => import("./pages/BusinessObjectsPage"));
const BusinessObjectDetailsPage = React.lazy(() => import("./pages/BusinessObjectDetailsPage"));
const SemanticHealthDashboard = React.lazy(() => import("./pages/SemanticHealthDashboard"));
const SchemaExplorerPage = React.lazy(() => import("./features/schema-explorer/pages/SchemaExplorer"));
const PageRuntimeRenderer = React.lazy(() => import("./pages/PageRuntimeRenderer"));
const WorkflowStudioPage = React.lazy(() => import("./pages/WorkflowStudioPage"));
const BusinessRuleEditorPage = React.lazy(() => import("./pages/BusinessRuleEditorPage"));
const SecurityRoutes = React.lazy(() => import("./features/security/routes").then(m => ({ default: m.SecurityRoutes })));


// RBAC Management Pages
const RoleManagerPage = React.lazy(() => import("./features/admin/pages/RoleManagerPage"));
const UserManagementPage = React.lazy(() => import("./features/admin/pages/UserManagementPage").then(m => ({ default: m.UserManagementPage })));
const UserRoleAssignmentPage = React.lazy(() => import("./features/admin/pages/UserRoleAssignmentPage"));
const TenantUserAssignmentPage = React.lazy(() => import("./features/admin/pages/TenantUserAssignmentPage"));
const DelegationManagerPage = React.lazy(() => import("./features/admin/pages/DelegationManagerPage"));
const FieldPermissionEditorPage = React.lazy(() => import("./features/admin/pages/FieldPermissionEditorPage"));

// Self-Service Studio — Phase D (Security & Access Mesh spec PART 5)
const ProfilesDashboard = React.lazy(() => import("./admin-v2").then(m => ({ default: m.ProfilesDashboard })));
const ProfileCustomizer = React.lazy(() => import("./admin-v2").then(m => ({ default: m.ProfileCustomizer })));
const EntitlementMatrix = React.lazy(() => import("./admin-v2").then(m => ({ default: m.EntitlementMatrix })));

const TeamManagerPage = React.lazy(() => import("./features/admin/pages/TeamManagerPage"));

// ASO Pages
const OptimizationCenter = React.lazy(() => import("./pages/OptimizationCenter").then(m => ({ default: m.OptimizationCenter })));
const ASOOptimizationDetail = React.lazy(() => import("./components/aso/ASOOptimizationDetail").then(m => ({ default: m.ASOOptimizationDetail })));

// Profile-key → ProfileCustomizer route wrapper. Pulls the URL param
// out of react-router so the component doesn't need its own router hook.
const ProfileCustomizerRoute: React.FC = () => {
  const { profileKey } = useParams<{ profileKey: string }>();
  // The system blueprint check happens server-side; we render the
  // customizer read-only for system profiles by setting isSystem=true.
  return <ProfileCustomizer profileKey={profileKey || ""} isSystem={profileKey?.startsWith("platform_") || false} />;
};

const EntitlementMatrixRoute: React.FC = () => {
  const { profileKey } = useParams<{ profileKey: string }>();
  return <EntitlementMatrix profileKey={profileKey || ""} isCustom={!profileKey?.startsWith("platform_")} />;
};
const ObservabilityDashboard = React.lazy(() => import("./pages/ObservabilityDashboard"));
const SLODashboard = React.lazy(() => import("./pages/SLODashboard"));
const ChangeReviewPage = React.lazy(() => import("./pages/ChangeReviewPage"));
const IncidentPage = React.lazy(() => import("./pages/scheduler/IncidentPage"));
const APIStudioPage = React.lazy(() => import('./pages/api-studio/APIStudioPage'));
const PageStudioListPage = React.lazy(() => import('./pages/page-studio/PageStudioListPage'));
const PageStudioDetailsPage = React.lazy(() => import('./pages/page-studio/PageStudioDetailsPage'));
const MenuDesignerPage = React.lazy(() => import('./pages/menu-designer/MenuDesignerPage'));
const StandaloneWindowWrapper = React.lazy(() => import('./components/desktop').then(m => ({ default: m.StandaloneWindowWrapper })));
const UniversalWorkspaceHub = React.lazy(() => import('./components/docking/UniversalWorkspaceHub').then(m => ({ default: m.UniversalWorkspaceHub })));
const RuntimePage = React.lazy(() => import('./pages/PageRuntimeRenderer'));

// Code-split workstation components for standalone / detached popout routes
const PageBrowser = React.lazy(() => import('./pages/PageBrowser'));
// MDM routes are served by core Page Studio pages (studioRoutes.ts).
const StudioPageContent = React.lazy(() => import('./pages/PageBrowser').then((m) => ({ default: m.PageContent })));
const StandalonePageRenderer = React.lazy<React.ComponentType<{ slug?: string; recordId?: string }>>(() =>
  import('./pages/PageBrowser').then((m) => ({ default: m.StandalonePageRenderer }))
);
const FixedIncomeDashboard = React.lazy(() => import('./components/FixedIncomeDashboard'));
const AIPortfolioRebalancer = React.lazy(() => import('./components/AIPortfolioRebalancer'));
const ScenarioAnalysisPro = React.lazy(() => import('./components/ScenarioAnalysisPro'));

// Intelligence & Governance (New)
import IntelligenceDashboard from "./pages/intelligence/IntelligenceDashboard";
import IndexAdvisorPage from "./pages/intelligence/IndexAdvisorPage";
import StorageTieringPage from "./pages/intelligence/StorageTieringPage";
import DataQualityMonitorPage from "./pages/intelligence/DataQualityMonitorPage";
import GovernanceConsolePage from "./pages/governance/GovernanceConsolePage";
import GlobalNLQueryPage from "./pages/GlobalNLQueryPage";

import SimulationWorkspace from "./pages/simulation/SimulationWorkspace";
import ScenarioDetail from "./pages/simulation/ScenarioDetail";
import ScenarioComparison from "./pages/simulation/ScenarioComparison";
import RebalancingWizard from "./pages/simulation/RebalancingWizard";
import { STUDIO_ROUTES } from './pages/page-studio/studioRoutes';

export function AppRoutes() {
  return (
    <Routes>
      <Route path="*" element={<ProtectedApp />} />
    </Routes>
  );
}

function ProtectedApp() {
  const navigate = useBlockableNavigate();
  const locale = useLocale();

  const handleBundleSave = () => {
    void navigate(`/${locale}/fabric/bundles`);
  };

  const handleBundleCancel = () => {
    void navigate(`/${locale}/fabric/bundles`);
  };

  const handleRoleSave = () => {
    void navigate(`/${locale}/fabric/roles`);
  };

  const handleRoleCancel = () => {
    void navigate(`/${locale}/fabric/roles`);
  };

  return (
    <>
      <React.Suspense fallback={<div style={{ padding: '32px', color: '#94a3b8', background: '#050d1a', height: '100%' }}>Loading view...</div>}>
        <Routes>
        {/* ═══════════════════════════════════════════════════════════════════
            PLATFORM - Organization, security, and setup
            ═══════════════════════════════════════════════════════════════════ */}
        <Route path="tenants" element={<ProtectedRoute><TenantDetailPageV2 /></ProtectedRoute>} />
        <Route path="tenants/:tenantId" element={<ProtectedRoute><TenantDetailPageV2 /></ProtectedRoute>} />
        <Route path="admin/rbac/roles" element={<ProtectedRoute><RoleManagerPage /></ProtectedRoute>} />
        <Route path="admin/rbac/users" element={<ProtectedRoute><UserManagementPage /></ProtectedRoute>} />
        <Route path="admin/rbac/teams" element={<ProtectedRoute><TeamManagerPage /></ProtectedRoute>} />
        <Route path="admin/rbac/delegations" element={<ProtectedRoute><DelegationManagerPage /></ProtectedRoute>} />
        <Route path="admin/rbac/field-permissions" element={<ProtectedRoute><FieldPermissionEditorPage /></ProtectedRoute>} />
        <Route path="admin/rbac/user-roles" element={<ProtectedRoute><UserRoleAssignmentPage /></ProtectedRoute>} />
        <Route path="admin/rbac/user-tenants" element={<ProtectedRoute><TenantUserAssignmentPage /></ProtectedRoute>} />
        <Route path="fabric/ip-whitelist" element={<ProtectedRoute><IPWhitelistManagementPage /></ProtectedRoute>} />
        <Route path="secrets/config" element={<ProtectedRoute><SecretsConfigPage tenantId="default" /></ProtectedRoute>} />
        <Route path="secrets/audit" element={<ProtectedRoute><SecretsAuditPage tenantId="default" /></ProtectedRoute>} />
        <Route path="secrets/monitoring" element={<ProtectedRoute><SecretsMonitoringPage tenantId="default" /></ProtectedRoute>} />

        <Route path="audit" element={<ProtectedRoute><AuditExplorer tenantId="default" tenantName="Default" /></ProtectedRoute>} />
        <Route path="admin/llm" element={<ProtectedRoute><LLMConfigPage /></ProtectedRoute>} />
        <Route path="admin/message-catalog" element={<ProtectedRoute><MessageCatalogPage /></ProtectedRoute>} />
        <Route path="admin/seeding" element={<ProtectedRoute><SeedingPage /></ProtectedRoute>} />
        <Route path="admin/temporal-ops" element={<ProtectedRoute><TemporalOpsPage /></ProtectedRoute>} />
        <Route path="fabric/tenants" element={<ProtectedRoute><TenantsManagementPage /></ProtectedRoute>} />
        <Route path="security/*" element={<ProtectedRoute><SecurityRoutes /></ProtectedRoute>} />

        {/* ═══════════════════════════════════════════════════════════════════
            CATALOG - Discovery and lineage
            ═══════════════════════════════════════════════════════════════════ */}
        <Route path="core/glossary" element={<ProtectedRoute><GlossaryExplorer /></ProtectedRoute>} />
        <Route path="core/semantic-terms" element={<ProtectedRoute><GlossaryExplorer /></ProtectedRoute>} />
        <Route path="catalog/semantic-terms" element={<ProtectedRoute><GlossaryExplorer /></ProtectedRoute>} />
        <Route path="core/business-terms" element={<ProtectedRoute><BusinessTermsExplorer /></ProtectedRoute>} />
        <Route path="catalog/business-terms" element={<ProtectedRoute><BusinessTermsExplorer /></ProtectedRoute>} />
        <Route path="core/abbreviations" element={<ProtectedRoute><AbbreviationsPage /></ProtectedRoute>} />
        <Route path="core/domains" element={<ProtectedRoute><DomainsManagementPage /></ProtectedRoute>} />
        <Route path="schema-explorer" element={<ProtectedRoute><SchemaExplorerPage /></ProtectedRoute>} />
        <Route path="catalog/api-inventory" element={<ProtectedRoute><ApiInventoryPage /></ProtectedRoute>} />

        <Route path="catalog/node-types" element={<ProtectedRoute><CatalogNodeTypesPage /></ProtectedRoute>} />
        <Route path="catalog/node-types/:id" element={<ProtectedRoute><NodeTypeDetailPage /></ProtectedRoute>} />
        <Route path="catalog/edge-types" element={<ProtectedRoute><CatalogEdgeTypesPage /></ProtectedRoute>} />
        <Route path="catalog/edge-types/:id" element={<ProtectedRoute><EdgeTypeDetailPage /></ProtectedRoute>} />

        <Route path="core/semantic-mapper" element={<ProtectedRoute><SemanticMapperPage /></ProtectedRoute>} />

        <Route path="catalog/ai-suggestions" element={<ProtectedRoute><AIBusinessTermSuggestionsPage /></ProtectedRoute>} />
        <Route path="catalog/business-terms/:id" element={<ProtectedRoute><BusinessTermDetailPage /></ProtectedRoute>} />

        {/* ═══════════════════════════════════════════════════════════════════
            BUILD - Semantic layer
            ═══════════════════════════════════════════════════════════════════ */}
        <Route path="business-objects" element={<ProtectedRoute><BusinessObjectsPage /></ProtectedRoute>} />
        {/* The old standalone "new BO" page read fake driving tables and posted to an
            unrouted save endpoint; new business objects are created by the binding
            wizard on the list page, which ?new=1 opens. */}
        <Route path="business-objects/new" element={<Navigate to={`/${locale}/business-objects?new=1`} replace />} />
        <Route path="business-objects/:id" element={<ProtectedRoute><BusinessObjectDetailsPage /></ProtectedRoute>} />
        <Route path="semantic-health" element={<ProtectedRoute><SemanticHealthDashboard /></ProtectedRoute>} />
        <Route path="views" element={<ProtectedRoute><ViewsCatalogPage /></ProtectedRoute>} />
        <Route path="views/:id" element={<ProtectedRoute><ViewDetailsPage /></ProtectedRoute>} />
        <Route path="fabric/bundles" element={<ProtectedRoute><BundleListPage /></ProtectedRoute>} />
        <Route path="fabric/bundles/create" element={<ProtectedRoute><BundleEditor onSave={handleBundleSave} onCancel={handleBundleCancel} /></ProtectedRoute>} />
        <Route path="fabric/bundles/:bundleId/edit" element={<ProtectedRoute><BundleEditor onSave={handleBundleSave} onCancel={handleBundleCancel} /></ProtectedRoute>} />
        {/* Validation rules: one editor, one engine (internal/rules/vm). */}
        {STUDIO_ROUTES.map(({ path, slug }) => (
          <Route key={path} path={path} element={<ProtectedRoute><StudioPageContent slug={slug} /></ProtectedRoute>} />
        ))}
        <Route path="core/validation-rules" element={<ProtectedRoute><AdvancedRuleBuilderPage /></ProtectedRoute>} />
        <Route path="core/validation-rules/editor" element={<ProtectedRoute><AdvancedRuleBuilderPage /></ProtectedRoute>} />
        {/* System-wide validation-rule-nodes view (every BO, one page) -
            a sibling of the two routes above, not a replacement: those
            two are the older catalog_validation_rules-era engine and the
            per-rule node editor respectively; this is the new node
            system's tenant-wide list. */}
        <Route path="core/validation-rules/all" element={<ProtectedRoute><SystemValidationsPage /></ProtectedRoute>} />
        <Route path="core/calculated-fields" element={<ProtectedRoute><CalculatedFieldBuilderPage /></ProtectedRoute>} />
        <Route path="core/flow-builder" element={<ProtectedRoute><UisceBuilder /></ProtectedRoute>} />
        <Route path="core/validation" element={<ProtectedRoute><InvestmentValidationPage /></ProtectedRoute>} />
        <Route path="query-builder/editor/:id?" element={<ProtectedRoute><SavedQueryEditor /></ProtectedRoute>} />
        <Route path="sql-studio" element={<ProtectedRoute><SqlStudioPage /></ProtectedRoute>} />
        <Route path="semantic-catalog" element={<ProtectedRoute><SemanticCatalogDetailPage /></ProtectedRoute>} />
        <Route path="pipelines" element={<ProtectedRoute><PipelinesPage /></ProtectedRoute>} />
        <Route path="reports" element={<ProtectedRoute><ReportsPage /></ProtectedRoute>} />

        {/* ═══════════════════════════════════════════════════════════════════
            STUDIO - Low-code tools
            ═══════════════════════════════════════════════════════════════════ */}
        <Route path="api-studio" element={<ProtectedRoute><APIStudioPage /></ProtectedRoute>} />
        <Route path="page-studio" element={<ProtectedRoute><PageStudioListPage /></ProtectedRoute>} />
        <Route path="page-studio/:id" element={<ProtectedRoute><PageStudioDetailsPage /></ProtectedRoute>} />
        <Route path="menu-designer" element={<ProtectedRoute><MenuDesignerPage /></ProtectedRoute>} />
        <Route path="pages" element={<ProtectedRoute><PageBrowser /></ProtectedRoute>} />
        <Route path="pages/:slug" element={<ProtectedRoute><PageBrowser /></ProtectedRoute>} />
        <Route path="pages/:slug/:recordId" element={<ProtectedRoute><PageBrowser /></ProtectedRoute>} />

        {/* ═══════════════════════════════════════════════════════════════════
            STANDALONE / DETACHED MULTI-MONITOR VIEWS (Wails v3 & Popouts)
            ═══════════════════════════════════════════════════════════════════ */}
        <Route
          path="view/page/:slug"
          element={
            <StandaloneWindowWrapper title="Page View">
              <ProtectedRoute>
                <StandalonePageRenderer />
              </ProtectedRoute>
            </StandaloneWindowWrapper>
          }
        />
        <Route
          path="view/page/:slug/:recordId"
          element={
            <StandaloneWindowWrapper title="Page Detail">
              <ProtectedRoute>
                <StandalonePageRenderer />
              </ProtectedRoute>
            </StandaloneWindowWrapper>
          }
        />
        <Route
          path="view/rebalancer"
          element={
            <StandaloneWindowWrapper title="AI Portfolio Rebalancer">
              <ProtectedRoute>
                <AIPortfolioRebalancer />
              </ProtectedRoute>
            </StandaloneWindowWrapper>
          }
        />
        <Route
          path="view/scenario"
          element={
            <StandaloneWindowWrapper title="Scenario Analysis Pro">
              <ProtectedRoute>
                <ScenarioAnalysisPro />
              </ProtectedRoute>
            </StandaloneWindowWrapper>
          }
        />
        <Route
          path="view/fixed-income"
          element={
            <StandaloneWindowWrapper title="Fixed Income Analytics">
              <ProtectedRoute>
                <FixedIncomeDashboard />
              </ProtectedRoute>
            </StandaloneWindowWrapper>
          }
        />
        <Route
          path="workspace"
          element={
            <ProtectedRoute>
              <UniversalWorkspaceHub />
            </ProtectedRoute>
          }
        />

        <Route path="app/data-product/:pageKey" element={<ProtectedRoute><DynamicDataProductPage /></ProtectedRoute>} />
        <Route path="client-portal/workflow-studio" element={<ProtectedRoute><WorkflowStudioPage /></ProtectedRoute>} />
        <Route path="client-portal/rules-editor" element={<ProtectedRoute><BusinessRuleEditorPage /></ProtectedRoute>} />
        <Route path="fabric/custom-components" element={<ProtectedRoute><CustomComponentPage /></ProtectedRoute>} />
        <Route path="core/workflow-designer" element={<ProtectedRoute><WorkflowDesignerPage /></ProtectedRoute>} />

        {/* ═══════════════════════════════════════════════════════════════════
            OPERATIONS - Scheduling and workflows
            ═══════════════════════════════════════════════════════════════════ */}
        {/* D1(a): Scheduler Intelligence console = Schedules console on /api/schedules. */}
        <Route path="scheduler-intelligence" element={<Navigate to={`/${locale}/automation/schedules`} replace />} />
        <Route path="scheduler-intelligence/*" element={<Navigate to={`/${locale}/automation/schedules`} replace />} />
        {/* S1/S7 job scheduler pages retired; same console. */}
        <Route path="scheduler/*" element={<Navigate to={`/${locale}/automation/schedules`} replace />} />
        
        <Route path="bp-console" element={<ProtectedRoute><BPConsolePage /></ProtectedRoute>} />
        <Route path="bp-console/:tab" element={<ProtectedRoute><BPConsolePage /></ProtectedRoute>} />
        <Route path="core/process-catalog" element={<ProtectedRoute><ProcessCatalogPage /></ProtectedRoute>} />
        <Route path="bp-console/instances" element={<ProtectedRoute><BPConsolePage /></ProtectedRoute>} />
        <Route path="bp-console/queues" element={<ProtectedRoute><BPConsolePage /></ProtectedRoute>} />
        
        <Route path="governance/changesets" element={<ProtectedRoute><GovernanceConsolePage /></ProtectedRoute>} />
        <Route path="core/approval-workflows" element={<ProtectedRoute><ApprovalWorkflowDashboard /></ProtectedRoute>} />
        <Route path="core/notifications" element={<ProtectedRoute><NotificationCenterPage /></ProtectedRoute>} />
        <Route path="core/notifications/templates" element={<ProtectedRoute><NotificationTemplateEditorPage /></ProtectedRoute>} />
        <Route path="core/notifications/preferences" element={<ProtectedRoute><NotificationPreferencesPage /></ProtectedRoute>} />

        {/* ═══════════════════════════════════════════════════════════════════
            INTELLIGENCE - Optimization and observability
            ═══════════════════════════════════════════════════════════════════ */}
        <Route path="intelligence" element={<ProtectedRoute><IntelligenceDashboard /></ProtectedRoute>} />
        <Route path="intelligence/index-advisor" element={<ProtectedRoute><IndexAdvisorPage /></ProtectedRoute>} />
        <Route path="intelligence/storage" element={<ProtectedRoute><StorageTieringPage /></ProtectedRoute>} />
        <Route path="intelligence/data-quality" element={<ProtectedRoute><DataQualityMonitorPage /></ProtectedRoute>} />
        <Route path="optimization" element={<ProtectedRoute><OptimizationCenter scope="global" /></ProtectedRoute>} />
        <Route path="optimization/:optimizationId" element={<ProtectedRoute><ASOOptimizationDetail /></ProtectedRoute>} />
        <Route path="observability" element={<ProtectedRoute><ObservabilityDashboard /></ProtectedRoute>} />
        <Route path="observability/slos" element={<ProtectedRoute><SLODashboard /></ProtectedRoute>} />
        <Route path="nlq" element={<ProtectedRoute><NLQPage /></ProtectedRoute>} />
        <Route path="global-intelligence" element={<ProtectedRoute><GlobalNLQueryPage /></ProtectedRoute>} />
        
        <Route path="simulation" element={<ProtectedRoute><SimulationWorkspace /></ProtectedRoute>} />
        <Route path="simulation/rebalance" element={<ProtectedRoute><RebalancingWizard /></ProtectedRoute>} />
        <Route path="simulation/:id" element={<ProtectedRoute><ScenarioDetail /></ProtectedRoute>} />
        <Route path="simulation/compare" element={<ProtectedRoute><ScenarioComparison /></ProtectedRoute>} />

        {/* ═══════════════════════════════════════════════════════════════════
            CONSUME - Reports and analytics
            ═══════════════════════════════════════════════════════════════════ */}
        <Route path="reports/library" element={<ProtectedRoute><ReportLibrary /></ProtectedRoute>} />
        <Route path="reports/builder" element={<ProtectedRoute><ReportBuilderPage /></ProtectedRoute>} />
        <Route path="reports/queries" element={<ProtectedRoute><QueryLibrary /></ProtectedRoute>} />
        <Route path="reports/:reportId/edit" element={<ProtectedRoute><ReportBuilderPage /></ProtectedRoute>} />
        <Route path="reports/models" element={<ProtectedRoute><SemanticModelManager /></ProtectedRoute>} />
        
        <Route path="analytics/factors" element={<ProtectedRoute><FactorAnalysisPage /></ProtectedRoute>} />
        <Route path="analytics/factors/:portfolioID?" element={<ProtectedRoute><FactorAnalysisPage /></ProtectedRoute>} />
        <Route path="fixed-income" element={<ProtectedRoute><FixedIncomeDashboard /></ProtectedRoute>} />
        <Route path="private-markets" element={<ProtectedRoute><AIPortfolioRebalancer /></ProtectedRoute>} />
        <Route path="analytics/scenario-analysis" element={<ProtectedRoute><ScenarioAnalysisPro /></ProtectedRoute>} />
        <Route path="analytics/rebalancer" element={<ProtectedRoute><AIPortfolioRebalancer /></ProtectedRoute>} />
        <Route path="analytics/advisor-dashboard" element={<ProtectedRoute><AdvisorDashboard /></ProtectedRoute>} />
        <Route path="crypto/portfolio" element={<ProtectedRoute><CryptoPortfolioCenter clientId={""} /></ProtectedRoute>} />
        <Route path="wealth/feed" element={<ProtectedRoute><Feed /></ProtectedRoute>} />

        {/* Legacy / Utilities / Misc */}
        <Route path="bundles" element={<ProtectedRoute><MicroBundleCatalogExample /></ProtectedRoute>} />
        <Route path="bundle-explorer" element={<ProtectedRoute><BundleExplorer /></ProtectedRoute>} />
        <Route path="jit-request" element={<ProtectedRoute><JITRequestPanelExample /></ProtectedRoute>} />
        <Route path="access-explanation" element={<ProtectedRoute><AccessExplanationExample /></ProtectedRoute>} />
        <Route path="fabric/preaggregations" element={<ProtectedRoute><ManagementPage tenantId="default" datasourceId="default" /></ProtectedRoute>} />
        <Route path="fabric/calculations" element={<ProtectedRoute><CalculationsLibraryPage tenantId="default" datasourceId="default" /></ProtectedRoute>} />
        <Route path="fabric/dashboard" element={<ProtectedRoute><DashboardPage /></ProtectedRoute>} />
        <Route path="fabric/audit-logs" element={<ProtectedRoute><AuditLogsPage /></ProtectedRoute>} />
        <Route path="fabric/settings" element={<ProtectedRoute><SettingsPage /></ProtectedRoute>} />
        <Route path="core/audit-explorer" element={<ProtectedRoute><AuditExplorerPage /></ProtectedRoute>} />
        <Route path="core/approval-inbox" element={<ProtectedRoute><ApprovalInboxPage /></ProtectedRoute>} />
        <Route path="core/sla-dashboard" element={<ProtectedRoute><SLADashboardPage /></ProtectedRoute>} />

        {/* GenUI Routes */}
        <Route path="core/genui-chat" element={<ProtectedRoute><GenUIChatPage /></ProtectedRoute>} />
        <Route path="core/genui-proposal" element={<ProtectedRoute><GenUIProposalDemoPage /></ProtectedRoute>} />
        <Route path="core/genui-inbox" element={<ProtectedRoute><GenUIApprovalInboxPage /></ProtectedRoute>} />
        {/* Compatibility Redirects */}

        <Route path="rbac" element={<Navigate to={`/${locale}/admin/rbac/roles`} replace />} />
        <Route path="glossary" element={<Navigate to={`/${locale}/core/glossary`} replace />} />
        <Route path="abbreviations" element={<Navigate to={`/${locale}/core/abbreviations`} replace />} />
        <Route path="validation-rules" element={<Navigate to={`/${locale}/core/validation-rules`} replace />} />
        <Route path="flow-builder" element={<Navigate to={`/${locale}/core/flow-builder`} replace />} />
        <Route path="api-designer" element={<Navigate to={`/${locale}/api-studio`} replace />} />
        <Route path="page-designer" element={<Navigate to={`/${locale}/page-studio`} replace />} />
        <Route path="change-review" element={<Navigate to={`/${locale}/governance/changesets`} replace />} />
        <Route path="change-reviews" element={<Navigate to={`/${locale}/governance/changesets`} replace />} />
        <Route path="change-reviews/:id" element={<ProtectedRoute><ChangeReviewPage /></ProtectedRoute>} />
        <Route path="incidents/:id" element={<ProtectedRoute><IncidentPage /></ProtectedRoute>} />
        <Route path="scheduler" element={<Navigate to={`/${locale}/automation/schedules`} replace />} />
        <Route path="aso" element={<Navigate to={`/${locale}/optimization`} replace />} />
        
        <Route path="" element={<ProtectedRoute><BundleExplorer /></ProtectedRoute>} />

        {/* Self-Service Studio — Phase D (Spec PART 5) */}
        <Route path="admin/entitlements" element={<ProtectedRoute><ProfilesDashboard /></ProtectedRoute>} />
        <Route path="admin/entitlements/profiles/:profileKey" element={<ProtectedRoute><ProfileCustomizerRoute /></ProtectedRoute>} />
        <Route path="admin/entitlements/profiles/:profileKey/components" element={<ProtectedRoute><EntitlementMatrixRoute /></ProtectedRoute>} />
      </Routes>
      </React.Suspense>
    </>
  );
}

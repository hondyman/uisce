import React from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import useBlockableNavigate from './components/RouteBlocker/useBlockableNavigate';
import { RouteBlockerProvider } from './components/RouteBlocker/RouteBlocker';
import { MicroBundleCatalogExample } from "./MicroBundleCatalogExample";
import { JITRequestPanelExample } from "./JITRequestPanelExample";
import { AccessExplanationExample } from "./AccessExplanationExample";
import ConversationalQueryPage from "./pages/ConversationalQueryPage";
import ManagementPage from "./features/fabric/pages/preaggregations/ManagementPage";
import FixedIncomeDashboard from "./components/FixedIncomeDashboard";
import BundleExplorer from "./components/BundleExplorer";
import CalculationsLibraryPage from "./features/fabric/pages/CalculationsLibraryPage";
import LoginPage from "./pages/AuthPage";
import AuthCallbackPage from "./pages/AuthCallbackPage";
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
import TenantsPage from "./features/tenants/pages/TenantsPage";
import { TenantDetailPageV2 } from "./features/tenants/pages/TenantDetailPageV2";
import { TenantListPage } from "./features/tenants/pages/TenantListPage";
import { SemanticCatalogPage } from "./pages/SemanticCatalogPage";
import { CatalogSetupPage } from "./pages/catalog/CatalogSetupPage";
import { CatalogSetupTestPage } from "./pages/catalog/CatalogSetupTestPage";
import { BusinessGlossaryPage } from "./pages/glossary/BusinessGlossaryPage";
import { AbbreviationsPage } from "./pages/core/AbbreviationsPage";
import { CatalogNodeTypesPage } from "./pages/catalog/CatalogNodeTypesPage";
import { CatalogEdgeTypesPage } from "./pages/catalog/CatalogEdgeTypesPage";
import { NodeTypeDetailPage } from "./pages/catalog/NodeTypeDetailPage";
import { EdgeTypeDetailPage } from "./pages/catalog/EdgeTypeDetailPage";
import { AIBusinessTermSuggestionsPage } from "./pages/catalog/AIBusinessTermSuggestionsPage";
import { BusinessTermDetailPage } from "./pages/catalog/BusinessTermDetailPage";
import DynamicUIGeneratorPage from "./pages/DynamicUIGeneratorPage";
import CustomComponentPage from "./pages/CustomComponentPage";
import ComponentMarketplacePage from "./pages/marketplace/ComponentMarketplacePage";
import Marketplace from "./pages/marketplace/Marketplace";
import { ValidationRulesBuilderPage } from "./pages/ValidationRulesBuilderPage";
import { UisceBuilderPage } from "./pages/UisceBuilderPage";
import UisceBuilder from "./features/uisce-builder/UisceBuilder";
import { InvestmentValidationPage } from "./pages/InvestmentValidationPage";
import ApprovalWorkflowDashboard from "./pages/ApprovalWorkflowDashboard";
import { WorkflowDesignerPage } from "./features/workflow/pages/WorkflowDesignerPage";

import { NotificationCenterPage } from "./features/workflow/pages/NotificationCenterPage";
import { NotificationTemplateEditorPage } from "./features/workflow/pages/NotificationTemplateEditorPage";
import { NotificationPreferencesPage } from "./features/workflow/pages/NotificationPreferencesPage";
import { SLADashboardPage } from "./features/workflow/pages/SLADashboardPage";
import { RegulatorDashboardPage } from "./features/workflow/pages/RegulatorDashboardPage";
import { ProcessCatalogPage } from "./features/workflow/pages/ProcessCatalogPage";
import { BusinessObjectExplorerPage } from "./features/workflow/pages/BusinessObjectExplorerPage";
import { AuditExplorerPage } from "./features/workflow/pages/AuditExplorerPage";
import AuditExplorer from "./components/audit/AuditExplorer";
import TemporalOpsPage from "./features/admin/pages/TemporalOpsPage";
import SeedingPage from "./features/admin/pages/SeedingPage";
import ScenarioAnalysisPro from "./components/ScenarioAnalysisPro";
import AIPortfolioRebalancer from "./components/AIPortfolioRebalancer";
// Metrics Console imports
import MetricsConsolePage from "./pages/MetricsConsolePage";
import MetricDetailPage from "./pages/MetricDetailPage";
import MetricCreatePage from "./pages/MetricCreatePage";
import MetricEditPage from "./pages/MetricEditPage";
import MetricCalcConsole from "./pages/metrics/MetricCalcConsole";
import SemanticEnrichmentWizard from "./pages/SemanticEnrichment/SemanticEnrichmentWizard";
import NLQPage from "./pages/nlq/NLQPage";
import LLMConfigPage from "./pages/admin/LLMConfigPage";

import { Feed } from "./features/feed/components/Feed";
import { ApprovalInboxPage } from "./features/wealth/pages/ApprovalInboxPage";
import { GenUIApprovalInboxPage } from "./features/workflow/pages/GenUIApprovalInboxPage";
import { GenUIProposalDemoPage } from "./features/workflow/pages/GenUIProposalDemoPage";
import GenUIChatPage from "./pages/GenUIChatPage";
import { FactorAnalysisPage } from "./features/analytics/pages/FactorAnalysisPage";
import AdvisorDashboard from "./pages/AdvisorDashboard";
import DirectIndexingPage from "./pages/investment/DirectIndexingPage";
import ValuesProfileEditor from "./pages/investment/ValuesProfileEditor";
// Crypto Platform
import CryptoDashboard from "./features/crypto/CryptoDashboard";
import CryptoPortfolioCenter from "./features/crypto/CryptoPortfolioCenter";
// Scheduler imports
import {
  SchedulerDashboardPage,
  JobsListPage,
  JobEditorPage,
  ExecutionsListPage,
  ExecutionDetailPage,
  DependencyVisualizerPage,
  BusinessCalendarsPage,
  CalendarEditorPage,
  NotificationTemplatesPage,
  NotificationTemplateEditorPage as SchedulerNotificationTemplateEditorPage,
  ComplianceDashboardPage,
} from "./features/scheduler";
// Scheduler Intelligence Console
import SchedulerConsolePage from "./pages/scheduler/SchedulerConsolePage";
// Secrets Management
import {
  SecretsConfigPage,
  SecretsAuditPage,
  SecretsMonitoringPage,
} from "./features/secrets";
// Reporting
import { WorldClassReportBuilder } from "./features/reporting/components/SelfServiceReportBuilder";

// BP Framework Console
import BPConsolePage from "./features/bp-console/pages/BPConsolePage";
import { ReportLibrary } from "./features/reporting/components/ReportLibrary";
import ReportBuilderPage from "./pages/ReportBuilderPage";
import { DataExplorer } from './components/reporting/DataExplorer';
import { QueryLibraryDashboard } from './components/reporting/QueryLibraryDashboard';
import { SemanticModelManager } from './features/semantic/components/SemanticModelManager';
import { ExpressionLibrary } from "./features/expressions/components/ExpressionLibrary";
import MetadataExplorer from "./components/MetadataExplorer";


import EntityManagerPage from "./features/admin/pages/EntityManagerPage";
import EntityDetailsPage from "./pages/EntityDetailsPage";
import BusinessObjectsPage from "./pages/BusinessObjectsPage";
import BusinessObjectDetailsPage from "./pages/BusinessObjectDetailsPage";
import SchemaExplorerPage from "./features/schema-explorer/pages/SchemaExplorer";
import PageRuntimeRenderer from "./pages/PageRuntimeRenderer";
import WorkflowStudioPage from "./pages/WorkflowStudioPage";
import BusinessRuleEditorPage from "./pages/BusinessRuleEditorPage";
import { SecurityRoutes } from "./features/security/routes";
import AuditHistoryPage from "./features/audit/pages/AuditHistoryPage";

// RBAC Management Pages
import RoleManagerPage from "./features/admin/pages/RoleManagerPage";
import { UserManagementPage } from "./features/admin/pages/UserManagementPage";
import UserRoleAssignmentPage from "./features/admin/pages/UserRoleAssignmentPage";
import DelegationManagerPage from "./features/admin/pages/DelegationManagerPage";
import FieldPermissionEditorPage from "./features/admin/pages/FieldPermissionEditorPage";

import TeamManagerPage from "./features/admin/pages/TeamManagerPage";

// ASO Pages
import { OptimizationCenter } from "./pages/OptimizationCenter";
import { ASOOptimizationDetail } from "./components/aso/ASOOptimizationDetail";
import { LineageExplorerPage } from "./pages/LineageExplorerPage";
import ObservabilityDashboard from "./pages/ObservabilityDashboard";
import SLODashboard from "./pages/SLODashboard";
import ChangeReviewPage from "./pages/ChangeReviewPage";
import IncidentPage from "./pages/scheduler/IncidentPage";
import APIStudioPage from './pages/api-studio/APIStudioPage';
import PageStudioPage from './pages/page-studio/PageStudioPage';
import RuntimePage from './pages/page-studio/RuntimePage';

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

export function AppRoutes() {
  return (
    <RouteBlockerProvider>
      <Routes>
        <Route path="/api-studio" element={<APIStudioPage />} />
        <Route path="/page-studio" element={<PageStudioPage />} />
        <Route path="/app/:slug" element={<RuntimePage />} />
        <Route path="/change-review" element={<ChangeReviewPage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/auth/callback" element={<AuthCallbackPage />} />
        <Route path="/*" element={<ProtectedApp />} />
      </Routes>
    </RouteBlockerProvider>
  );
}

function ProtectedApp() {
  const navigate = useBlockableNavigate();

  const handleBundleSave = () => {
    void navigate('/fabric/bundles');
  };

  const handleBundleCancel = () => {
    void navigate('/fabric/bundles');
  };

  const handleRoleSave = () => {
    void navigate('/fabric/roles');
  };

  const handleRoleCancel = () => {
    void navigate('/fabric/roles');
  };

  return (
    <>

      <Routes>
        {/* ═══════════════════════════════════════════════════════════════════
            PLATFORM - Organization, security, and setup
            ═══════════════════════════════════════════════════════════════════ */}
        <Route path="/tenants" element={<ProtectedRoute><TenantListPage /></ProtectedRoute>} />
        <Route path="/tenants/:tenantId" element={<ProtectedRoute><TenantDetailPageV2 /></ProtectedRoute>} />
        <Route path="/admin/rbac/roles" element={<ProtectedRoute><RoleManagerPage /></ProtectedRoute>} />
        <Route path="/admin/rbac/users" element={<ProtectedRoute><UserManagementPage /></ProtectedRoute>} />
        <Route path="/admin/rbac/teams" element={<ProtectedRoute><TeamManagerPage /></ProtectedRoute>} />
        <Route path="/admin/rbac/delegations" element={<ProtectedRoute><DelegationManagerPage /></ProtectedRoute>} />
        <Route path="/admin/rbac/field-permissions" element={<ProtectedRoute><FieldPermissionEditorPage /></ProtectedRoute>} />
        <Route path="/admin/rbac/user-roles" element={<ProtectedRoute><UserRoleAssignmentPage /></ProtectedRoute>} />
        <Route path="/fabric/ip-whitelist" element={<ProtectedRoute><IPWhitelistManagementPage /></ProtectedRoute>} />
        <Route path="/secrets/config" element={<ProtectedRoute><SecretsConfigPage tenantId="default" /></ProtectedRoute>} />
        <Route path="/secrets/audit" element={<ProtectedRoute><SecretsAuditPage tenantId="default" /></ProtectedRoute>} />
        <Route path="/secrets/monitoring" element={<ProtectedRoute><SecretsMonitoringPage tenantId="default" /></ProtectedRoute>} />
        <Route path="/setup/audit" element={<ProtectedRoute><AuditHistoryPage /></ProtectedRoute>} />
        <Route path="/audit" element={<ProtectedRoute><AuditExplorer tenantId="default" tenantName="Default" /></ProtectedRoute>} />
        <Route path="/admin/llm" element={<ProtectedRoute><LLMConfigPage /></ProtectedRoute>} />
        <Route path="/admin/seeding" element={<ProtectedRoute><SeedingPage /></ProtectedRoute>} />
        <Route path="/admin/temporal-ops" element={<ProtectedRoute><TemporalOpsPage /></ProtectedRoute>} />
        <Route path="/fabric/tenants" element={<ProtectedRoute><TenantsManagementPage /></ProtectedRoute>} />
        <Route path="/security/*" element={<ProtectedRoute><SecurityRoutes /></ProtectedRoute>} />

        {/* ═══════════════════════════════════════════════════════════════════
            CATALOG - Discovery and lineage
            ═══════════════════════════════════════════════════════════════════ */}
        <Route path="/core/glossary" element={<ProtectedRoute><BusinessGlossaryPage /></ProtectedRoute>} />
        <Route path="/core/abbreviations" element={<ProtectedRoute><AbbreviationsPage /></ProtectedRoute>} />
        <Route path="/core/domains" element={<ProtectedRoute><DomainsManagementPage /></ProtectedRoute>} />
        <Route path="/schema-explorer" element={<ProtectedRoute><SchemaExplorerPage /></ProtectedRoute>} />
        <Route path="/metadata-explorer" element={<ProtectedRoute><MetadataExplorer /></ProtectedRoute>} />
        <Route path="/catalog/node-types" element={<ProtectedRoute><CatalogNodeTypesPage /></ProtectedRoute>} />
        <Route path="/catalog/node-types/:id" element={<ProtectedRoute><NodeTypeDetailPage /></ProtectedRoute>} />
        <Route path="/catalog/edge-types" element={<ProtectedRoute><CatalogEdgeTypesPage /></ProtectedRoute>} />
        <Route path="/catalog/edge-types/:id" element={<ProtectedRoute><EdgeTypeDetailPage /></ProtectedRoute>} />
        <Route path="/lineage" element={<ProtectedRoute><LineageExplorerPage /></ProtectedRoute>} />
        <Route path="/lineage/:nodeId" element={<ProtectedRoute><LineageExplorerPage /></ProtectedRoute>} />
        <Route path="/core/semantic-mapper" element={<ProtectedRoute><SemanticMapperPage /></ProtectedRoute>} />
        <Route path="/core/catalog-setup" element={<ProtectedRoute><CatalogSetupPage /></ProtectedRoute>} />
        <Route path="/core/catalog-setup-test" element={<ProtectedRoute><CatalogSetupTestPage /></ProtectedRoute>} />
        <Route path="/catalog/ai-suggestions" element={<ProtectedRoute><AIBusinessTermSuggestionsPage /></ProtectedRoute>} />
        <Route path="/catalog/business-terms/:id" element={<ProtectedRoute><BusinessTermDetailPage /></ProtectedRoute>} />

        {/* ═══════════════════════════════════════════════════════════════════
            BUILD - Semantic layer
            ═══════════════════════════════════════════════════════════════════ */}
        <Route path="/business-objects" element={<ProtectedRoute><BusinessObjectsPage /></ProtectedRoute>} />
        <Route path="/business-objects/:id" element={<ProtectedRoute><BusinessObjectDetailsPage /></ProtectedRoute>} />
        <Route path="/views" element={<ProtectedRoute><ViewsCatalogPage /></ProtectedRoute>} />
        <Route path="/views/:id" element={<ProtectedRoute><ViewDetailsPage /></ProtectedRoute>} />
        <Route path="/fabric/bundles" element={<ProtectedRoute><BundleListPage /></ProtectedRoute>} />
        <Route path="/fabric/bundles/create" element={<ProtectedRoute><BundleEditor onSave={handleBundleSave} onCancel={handleBundleCancel} /></ProtectedRoute>} />
        <Route path="/fabric/bundles/:bundleId/edit" element={<ProtectedRoute><BundleEditor onSave={handleBundleSave} onCancel={handleBundleCancel} /></ProtectedRoute>} />
        <Route path="/core/validation-rules" element={<ProtectedRoute><ValidationRulesBuilderPage /></ProtectedRoute>} />
        <Route path="/core/calculated-fields" element={<ProtectedRoute><CalculatedFieldBuilderPage /></ProtectedRoute>} />
        <Route path="/reports/expressions" element={<ProtectedRoute><ExpressionLibrary /></ProtectedRoute>} />
        <Route path="/core/flow-builder" element={<ProtectedRoute><UisceBuilder /></ProtectedRoute>} />
        <Route path="/core/uisce-builder" element={<ProtectedRoute><UisceBuilderPage /></ProtectedRoute>} />
        <Route path="/core/validation" element={<ProtectedRoute><InvestmentValidationPage /></ProtectedRoute>} />
        <Route path="/marketplace" element={<ProtectedRoute><Marketplace /></ProtectedRoute>} />
        <Route path="/marketplace/components" element={<ProtectedRoute><ComponentMarketplacePage /></ProtectedRoute>} />

        {/* ═══════════════════════════════════════════════════════════════════
            STUDIO - Low-code tools
            ═══════════════════════════════════════════════════════════════════ */}
        <Route path="/api-studio" element={<ProtectedRoute><APIStudioPage /></ProtectedRoute>} />
        <Route path="/page-studio" element={<ProtectedRoute><PageStudioPage /></ProtectedRoute>} />
        <Route path="/dynamic-ui" element={<ProtectedRoute><DynamicUIGeneratorPage /></ProtectedRoute>} />
        <Route path="/page-designer" element={<ProtectedRoute><DynamicUIGeneratorPage /></ProtectedRoute>} />
        <Route path="/client-portal/workflow-studio" element={<ProtectedRoute><WorkflowStudioPage /></ProtectedRoute>} />
        <Route path="/client-portal/rules-editor" element={<ProtectedRoute><BusinessRuleEditorPage /></ProtectedRoute>} />
        <Route path="/fabric/custom-components" element={<ProtectedRoute><CustomComponentPage /></ProtectedRoute>} />
        <Route path="/core/workflow-designer" element={<ProtectedRoute><WorkflowDesignerPage /></ProtectedRoute>} />

        {/* ═══════════════════════════════════════════════════════════════════
            OPERATIONS - Scheduling and workflows
            ═══════════════════════════════════════════════════════════════════ */}
        <Route path="/scheduler-intelligence" element={<ProtectedRoute><SchedulerConsolePage /></ProtectedRoute>} />
        <Route path="/scheduler/jobs" element={<ProtectedRoute><JobsListPage /></ProtectedRoute>} />
        <Route path="/scheduler/jobs/new" element={<ProtectedRoute><JobEditorPage /></ProtectedRoute>} />
        <Route path="/scheduler/jobs/:jobId" element={<ProtectedRoute><JobEditorPage /></ProtectedRoute>} />
        <Route path="/scheduler/jobs/:jobId/edit" element={<ProtectedRoute><JobEditorPage /></ProtectedRoute>} />
        <Route path="/scheduler/executions" element={<ProtectedRoute><ExecutionsListPage /></ProtectedRoute>} />
        <Route path="/scheduler/executions/:executionId" element={<ProtectedRoute><ExecutionDetailPage /></ProtectedRoute>} />
        <Route path="/scheduler/calendars" element={<ProtectedRoute><BusinessCalendarsPage /></ProtectedRoute>} />
        <Route path="/scheduler/calendars/new" element={<ProtectedRoute><CalendarEditorPage /></ProtectedRoute>} />
        <Route path="/scheduler/calendars/:calendarId/edit" element={<ProtectedRoute><CalendarEditorPage /></ProtectedRoute>} />
        
        <Route path="/bp-console" element={<ProtectedRoute><BPConsolePage /></ProtectedRoute>} />
        <Route path="/bp-console/:tab" element={<ProtectedRoute><BPConsolePage /></ProtectedRoute>} />
        <Route path="/core/process-catalog" element={<ProtectedRoute><ProcessCatalogPage /></ProtectedRoute>} />
        <Route path="/bp-console/instances" element={<ProtectedRoute><BPConsolePage /></ProtectedRoute>} />
        <Route path="/bp-console/queues" element={<ProtectedRoute><BPConsolePage /></ProtectedRoute>} />
        
        <Route path="/governance/changesets" element={<ProtectedRoute><GovernanceConsolePage /></ProtectedRoute>} />
        <Route path="/core/approval-workflows" element={<ProtectedRoute><ApprovalWorkflowDashboard /></ProtectedRoute>} />
        <Route path="/core/notifications" element={<ProtectedRoute><NotificationCenterPage /></ProtectedRoute>} />
        <Route path="/core/notifications/templates" element={<ProtectedRoute><NotificationTemplateEditorPage /></ProtectedRoute>} />
        <Route path="/core/notifications/preferences" element={<ProtectedRoute><NotificationPreferencesPage /></ProtectedRoute>} />

        {/* ═══════════════════════════════════════════════════════════════════
            INTELLIGENCE - Optimization and observability
            ═══════════════════════════════════════════════════════════════════ */}
        <Route path="/intelligence" element={<ProtectedRoute><IntelligenceDashboard /></ProtectedRoute>} />
        <Route path="/intelligence/index-advisor" element={<ProtectedRoute><IndexAdvisorPage /></ProtectedRoute>} />
        <Route path="/intelligence/storage" element={<ProtectedRoute><StorageTieringPage /></ProtectedRoute>} />
        <Route path="/intelligence/data-quality" element={<ProtectedRoute><DataQualityMonitorPage /></ProtectedRoute>} />
        <Route path="/optimization" element={<ProtectedRoute><OptimizationCenter scope="global" /></ProtectedRoute>} />
        <Route path="/optimization/:optimizationId" element={<ProtectedRoute><ASOOptimizationDetail /></ProtectedRoute>} />
        <Route path="/observability" element={<ProtectedRoute><ObservabilityDashboard /></ProtectedRoute>} />
        <Route path="/observability/slos" element={<ProtectedRoute><SLODashboard /></ProtectedRoute>} />
        <Route path="/nlq" element={<ProtectedRoute><NLQPage /></ProtectedRoute>} />
        <Route path="/global-intelligence" element={<ProtectedRoute><GlobalNLQueryPage /></ProtectedRoute>} />
        
        <Route path="/simulation" element={<ProtectedRoute><SimulationWorkspace /></ProtectedRoute>} />
        <Route path="/simulation/rebalance" element={<ProtectedRoute><RebalancingWizard /></ProtectedRoute>} />
        <Route path="/simulation/:id" element={<ProtectedRoute><ScenarioDetail /></ProtectedRoute>} />
        <Route path="/simulation/compare" element={<ProtectedRoute><ScenarioComparison /></ProtectedRoute>} />

        {/* ═══════════════════════════════════════════════════════════════════
            CONSUME - Reports and analytics
            ═══════════════════════════════════════════════════════════════════ */}
        <Route path="/reports/library" element={<ProtectedRoute><ReportLibrary /></ProtectedRoute>} />
        <Route path="/reports/builder" element={<ProtectedRoute><ReportBuilderPage /></ProtectedRoute>} />
        <Route path="/reports/queries" element={<ProtectedRoute><QueryLibraryDashboard /></ProtectedRoute>} />
        <Route path="/reports/:reportId/edit" element={<ProtectedRoute><ReportBuilderPage /></ProtectedRoute>} />
        <Route path="/reports/models" element={<ProtectedRoute><SemanticModelManager /></ProtectedRoute>} />
        
        <Route path="/analytics/factors" element={<ProtectedRoute><FactorAnalysisPage /></ProtectedRoute>} />
        <Route path="/analytics/factors/:portfolioID?" element={<ProtectedRoute><FactorAnalysisPage /></ProtectedRoute>} />
        <Route path="/fixed-income" element={<ProtectedRoute><FixedIncomeDashboard /></ProtectedRoute>} />
        <Route path="/private-markets" element={<ProtectedRoute><AIPortfolioRebalancer /></ProtectedRoute>} />
        <Route path="/analytics/scenario-analysis" element={<ProtectedRoute><ScenarioAnalysisPro /></ProtectedRoute>} />
        <Route path="/analytics/rebalancer" element={<ProtectedRoute><AIPortfolioRebalancer /></ProtectedRoute>} />
        <Route path="/analytics/advisor-dashboard" element={<ProtectedRoute><AdvisorDashboard /></ProtectedRoute>} />
        <Route path="/crypto/portfolio" element={<ProtectedRoute><CryptoPortfolioCenter clientId={""} /></ProtectedRoute>} />
        <Route path="/wealth/feed" element={<ProtectedRoute><Feed /></ProtectedRoute>} />

        {/* Legacy / Utilities / Misc */}
        <Route path="/bundles" element={<ProtectedRoute><MicroBundleCatalogExample /></ProtectedRoute>} />
        <Route path="/bundle-explorer" element={<ProtectedRoute><BundleExplorer /></ProtectedRoute>} />
        <Route path="/jit-request" element={<ProtectedRoute><JITRequestPanelExample /></ProtectedRoute>} />
        <Route path="/access-explanation" element={<ProtectedRoute><AccessExplanationExample /></ProtectedRoute>} />
        <Route path="/fabric/preaggregations" element={<ProtectedRoute><ManagementPage tenantId="default" datasourceId="default" /></ProtectedRoute>} />
        <Route path="/fabric/calculations" element={<ProtectedRoute><CalculationsLibraryPage tenantId="default" datasourceId="default" /></ProtectedRoute>} />
        <Route path="/fabric/dashboard" element={<ProtectedRoute><DashboardPage /></ProtectedRoute>} />
        <Route path="/fabric/audit-logs" element={<ProtectedRoute><AuditLogsPage /></ProtectedRoute>} />
        <Route path="/fabric/settings" element={<ProtectedRoute><SettingsPage /></ProtectedRoute>} />
        <Route path="/core/audit-explorer" element={<ProtectedRoute><AuditExplorerPage /></ProtectedRoute>} />
        <Route path="/core/approval-inbox" element={<ProtectedRoute><ApprovalInboxPage /></ProtectedRoute>} />
        <Route path="/core/sla-dashboard" element={<ProtectedRoute><SLADashboardPage /></ProtectedRoute>} />
        <Route path="/core/business-objects" element={<ProtectedRoute><BusinessObjectExplorerPage /></ProtectedRoute>} />
        
        {/* GenUI Routes */}
        <Route path="/core/genui-chat" element={<ProtectedRoute><GenUIChatPage /></ProtectedRoute>} />
        <Route path="/core/genui-proposal" element={<ProtectedRoute><GenUIProposalDemoPage /></ProtectedRoute>} />
        <Route path="/core/genui-inbox" element={<ProtectedRoute><GenUIApprovalInboxPage /></ProtectedRoute>} />
        {/* Compatibility Redirects */}
        <Route path="/audit" element={<Navigate to="/setup/audit" replace />} />
        <Route path="/rbac" element={<Navigate to="/admin/rbac/roles" replace />} />
        <Route path="/glossary" element={<Navigate to="/core/glossary" replace />} />
        <Route path="/abbreviations" element={<Navigate to="/core/abbreviations" replace />} />
        <Route path="/validation-rules" element={<Navigate to="/core/validation-rules" replace />} />
        <Route path="/flow-builder" element={<Navigate to="/core/flow-builder" replace />} />
        <Route path="/api-designer" element={<Navigate to="/api-studio" replace />} />
        <Route path="/page-designer" element={<Navigate to="/page-studio" replace />} />
        <Route path="/change-review" element={<Navigate to="/governance/changesets" replace />} />
        <Route path="/change-reviews" element={<Navigate to="/governance/changesets" replace />} />
        <Route path="/change-reviews/:id" element={<ProtectedRoute><ChangeReviewPage /></ProtectedRoute>} />
        <Route path="/incidents/:id" element={<ProtectedRoute><IncidentPage /></ProtectedRoute>} />
        <Route path="/scheduler" element={<Navigate to="/scheduler-intelligence" replace />} />
        <Route path="/aso" element={<Navigate to="/optimization" replace />} />
        
        <Route path="/" element={<ProtectedRoute><BundleExplorer /></ProtectedRoute>} />
      </Routes>
    </>
  );
}

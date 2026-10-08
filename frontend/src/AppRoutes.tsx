import React from "react";
import { Routes, Route, Navigate, useParams } from "react-router-dom";
import useBlockableNavigate from './components/RouteBlocker/useBlockableNavigate';
import { useLocale } from "./i18n/useLocale";
import BundleExplorer from "./components/BundleExplorer";
import ProtectedRoute from "./components/ProtectedRoute";
import AuditLogsPage from "./features/fabric/pages/AuditLogsPage";
import ViewDetailsPage from "./features/views/pages/ViewDetailsPage";
import BundleEditor from "./pages/bundles/BundleEditor";
import { TenantDetailPageV2 } from "./features/tenants/pages/TenantDetailPageV2";

import { NodeTypeDetailPage } from "./pages/catalog/NodeTypeDetailPage";
import { EdgeTypeDetailPage } from "./pages/catalog/EdgeTypeDetailPage";
import { BusinessTermDetailPage } from "./pages/catalog/BusinessTermDetailPage";
import WorkbenchPage from "./features/custom-attributes/pages/WorkbenchPage";
import CustomComponentPage from "./pages/CustomComponentPage";
import SystemValidationsPage from "./pages/SystemValidationsPage";
import { AuditExplorerPage } from "./features/workflow/pages/AuditExplorerPage";

const SavedQueryEditor = React.lazy(() => import("./features/query-builder/pages/SavedQueryEditor"));
const CubeQueryBuilder = React.lazy(() => import("./features/query-builder/pages/BusinessObjectQueryBuilder"));
const SqlStudioPage = React.lazy(() => import("./pages/analytical/SqlStudioPage"));
const SemanticCatalogDetailPage = React.lazy(() => import("./pages/analytical/SemanticCatalogDetailPage"));
const PipelinesPage = React.lazy(() => import("./pages/analytical/PipelinesPage"));
const ReportsPage = React.lazy(() => import("./pages/analytical/ReportsPage"));

const Feed = React.lazy(() => import("./features/feed/components/Feed").then(m => ({ default: m.Feed })));
const GenUIApprovalInboxPage = React.lazy(() => import("./features/workflow/pages/GenUIApprovalInboxPage").then(m => ({ default: m.GenUIApprovalInboxPage })));
const GenUIProposalDemoPage = React.lazy(() => import("./features/workflow/pages/GenUIProposalDemoPage").then(m => ({ default: m.GenUIProposalDemoPage })));
const GenUIChatPage = React.lazy(() => import("./pages/GenUIChatPage"));
const FactorAnalysisPage = React.lazy(() => import("./features/analytics/pages/FactorAnalysisPage").then(m => ({ default: m.FactorAnalysisPage })));
// Crypto Platform
const CryptoPortfolioCenter = React.lazy(() => import("./features/crypto/CryptoPortfolioCenter"));
// Secrets Management
const SecretsAuditPage = React.lazy(() => import("./features/secrets").then(m => ({ default: m.SecretsAuditPage })));
const SecretsMonitoringPage = React.lazy(() => import("./features/secrets").then(m => ({ default: m.SecretsMonitoringPage })));
// Reporting
const ReportBuilderPage = React.lazy(() => import("./pages/ReportBuilderPage"));

const BusinessObjectDetailsPage = React.lazy(() => import("./pages/BusinessObjectDetailsPage"));
const SemanticHealthDashboard = React.lazy(() => import("./pages/SemanticHealthDashboard"));

// Self-Service Studio — Phase D (Security & Access Mesh spec PART 5)
const ProfilesDashboard = React.lazy(() => import("./admin-v2").then(m => ({ default: m.ProfilesDashboard })));
const ProfileCustomizer = React.lazy(() => import("./admin-v2").then(m => ({ default: m.ProfileCustomizer })));
const EntitlementMatrix = React.lazy(() => import("./admin-v2").then(m => ({ default: m.EntitlementMatrix })));

// ASO Pages
const ASOOptimizationDetail = React.lazy(() => import("./components/aso/ASOOptimizationDetail").then(m => ({ default: m.ASOOptimizationDetail })));

// Profile-key → ProfileCustomizer route wrapper. Pulls the URL param
// out of react-router so the component doesn't need its own router hook.
const ProfileCustomizerRoute: React.FC = () => {
  const { profileKey } = useParams<{ profileKey: string }>();
  return <ProfileCustomizer profileKey={profileKey || ""} isSystem={profileKey?.startsWith("platform_") || false} />;
};

const EntitlementMatrixRoute: React.FC = () => {
  const { profileKey } = useParams<{ profileKey: string }>();
  return <EntitlementMatrix profileKey={profileKey || ""} isCustom={!profileKey?.startsWith("platform_")} />;
};
const ChangeReviewPage = React.lazy(() => import("./pages/ChangeReviewPage"));
const IncidentPage = React.lazy(() => import("./pages/scheduler/IncidentPage"));
const APIStudioPage = React.lazy(() => import('./pages/api-studio/APIStudioPage'));
const PageStudioListPage = React.lazy(() => import('./pages/page-studio/PageStudioListPage'));
const PageStudioDetailsPage = React.lazy(() => import('./pages/page-studio/PageStudioDetailsPage'));
const MenuDesignerPage = React.lazy(() => import('./pages/menu-designer/MenuDesignerPage'));
const StandaloneWindowWrapper = React.lazy(() => import('./components/desktop').then(m => ({ default: m.StandaloneWindowWrapper })));
const UniversalWorkspaceHub = React.lazy(() => import('./components/docking/UniversalWorkspaceHub').then(m => ({ default: m.UniversalWorkspaceHub })));

// Code-split workstation components for standalone / detached popout routes
const PageBrowser = React.lazy(() => import('./pages/PageBrowser'));
// MDM routes are served by core Page Studio pages (route_aliases table, served via /route-aliases).
const StudioPageContent = React.lazy(() => import('./pages/PageBrowser').then((m) => ({ default: m.PageContent })));
const StandalonePageRenderer = React.lazy<React.ComponentType<{ slug?: string; recordId?: string }>>(() =>
  import('./pages/PageBrowser').then((m) => ({ default: m.StandalonePageRenderer }))
);
const FixedIncomeDashboard = React.lazy(() => import('./components/FixedIncomeDashboard'));
const ComplianceDecisionBlotter = React.lazy(() => import('./components/Compliance/ComplianceDecisionBlotter'));
const RegulatoryChangeQueue = React.lazy(() =>
  import('./components/Compliance/RegulatoryChangeQueue').then((m) => ({ default: m.RegulatoryChangeQueue }))
);
const SurveillanceFindingsQueue = React.lazy(() =>
  import('./components/Compliance/SurveillanceFindingsQueue').then((m) => ({ default: m.SurveillanceFindingsQueue }))
);
const AIPortfolioRebalancer = React.lazy(() => import('./components/AIPortfolioRebalancer'));
const ScenarioAnalysisPro = React.lazy(() => import('./components/ScenarioAnalysisPro'));

import SimulationWorkspace from "./pages/simulation/SimulationWorkspace";
import ScenarioDetail from "./pages/simulation/ScenarioDetail";
import ScenarioComparison from "./pages/simulation/ScenarioComparison";
import RebalancingWizard from "./pages/simulation/RebalancingWizard";
import { useRouteAliases } from './hooks/useRouteAliases';

export function AppRoutes() {
  return (
    <Routes>
      <Route path="*" element={<ProtectedApp />} />
    </Routes>
  );
}

function SlugPage() {
  const { slug } = useParams<{ slug: string }>();
  return <StudioPageContent slug={slug ?? ''} />;
}

function ProtectedApp() {
  const { aliases, loaded: aliasesLoaded } = useRouteAliases();
  const navigate = useBlockableNavigate();
  const locale = useLocale();

  const handleBundleSave = () => {
    void navigate(`/${locale}/fabric/bundles`);
  };

  const handleBundleCancel = () => {
    void navigate(`/${locale}/fabric/bundles`);
  };

  return (
    <>
      <React.Suspense fallback={<div style={{ padding: '32px', color: '#94a3b8', background: '#050d1a', height: '100%' }}>Loading view...</div>}>
        {aliasesLoaded ? (
        <Routes>
        {/* ═══════════════════════════════════════════════════════════════════
            PLATFORM - Organization, security, and setup
            ═══════════════════════════════════════════════════════════════════ */}
        <Route path="tenants/:tenantId" element={<ProtectedRoute><TenantDetailPageV2 /></ProtectedRoute>} />
        <Route path="secrets/audit" element={<ProtectedRoute><SecretsAuditPage tenantId="default" /></ProtectedRoute>} />
        <Route path="secrets/monitoring" element={<ProtectedRoute><SecretsMonitoringPage tenantId="default" /></ProtectedRoute>} />

        {/* ═══════════════════════════════════════════════════════════════════
            CATALOG - Discovery and lineage
            ═══════════════════════════════════════════════════════════════════ */}
        <Route path="catalog/custom-fields/:entityType" element={<ProtectedRoute><WorkbenchPage /></ProtectedRoute>} />

        <Route path="catalog/node-types/:id" element={<ProtectedRoute><NodeTypeDetailPage /></ProtectedRoute>} />
        <Route path="catalog/edge-types/:id" element={<ProtectedRoute><EdgeTypeDetailPage /></ProtectedRoute>} />

        <Route path="catalog/business-terms/:id" element={<ProtectedRoute><BusinessTermDetailPage /></ProtectedRoute>} />

        {/* ═══════════════════════════════════════════════════════════════════
            BUILD - Semantic layer
            ═══════════════════════════════════════════════════════════════════ */}
        {/* The old standalone "new BO" page read fake driving tables and posted to an
            unrouted save endpoint; new business objects are created by the binding
            wizard on the list page, which ?new=1 opens. */}
        <Route path="business-objects/new" element={<Navigate to={`/${locale}/business-objects?new=1`} replace />} />
        <Route path="business-objects/:id" element={<ProtectedRoute><BusinessObjectDetailsPage /></ProtectedRoute>} />
        <Route path="semantic-health" element={<ProtectedRoute><SemanticHealthDashboard /></ProtectedRoute>} />
        <Route path="views/:id" element={<ProtectedRoute><ViewDetailsPage /></ProtectedRoute>} />
        <Route path="fabric/bundles/create" element={<ProtectedRoute><BundleEditor onSave={handleBundleSave} onCancel={handleBundleCancel} /></ProtectedRoute>} />
        <Route path="fabric/bundles/:bundleId/edit" element={<ProtectedRoute><BundleEditor onSave={handleBundleSave} onCancel={handleBundleCancel} /></ProtectedRoute>} />
        {/* Validation rules: one editor, one engine (internal/rules/vm). */}
        {/* Every Page Designer page is reachable at /p/:slug; legacy URLs come from route_aliases (DB). */}
        <Route path="p/:slug" element={<ProtectedRoute><SlugPage /></ProtectedRoute>} />
        {aliases.map(({ path, pageKey }) => (
          <Route key={path} path={path.replace(/^\//, '')} element={<ProtectedRoute><StudioPageContent slug={pageKey} /></ProtectedRoute>} />
        ))}
        {/* System-wide validation-rule-nodes view (every BO, one page) */}
        <Route path="core/validation-rules/all" element={<ProtectedRoute><SystemValidationsPage /></ProtectedRoute>} />
        <Route path="query-builder/editor/:id?" element={<ProtectedRoute><SavedQueryEditor /></ProtectedRoute>} />
        {/* Cube-pinned query builder (subject.kind='cube'). See CubeQueryBuilder
            mount guard in BusinessObjectQueryBuilder.tsx — it reads :cubeId from
            the URL, calls cubes.get to populate state, and surfaces an
            undeployed-state banner before preview returns empty rows. */}
        <Route path="query-builder/cube/:cubeId" element={<ProtectedRoute><CubeQueryBuilder /></ProtectedRoute>} />
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
          path="view/compliance-blotter"
          element={
            <StandaloneWindowWrapper title="Pre-Trade Compliance Blotter">
              <ProtectedRoute>
                <ComplianceDecisionBlotter />
              </ProtectedRoute>
            </StandaloneWindowWrapper>
          }
        />
        <Route
          path="view/regulatory-queue"
          element={
            <StandaloneWindowWrapper title="Regulatory Change Queue">
              <ProtectedRoute>
                <RegulatoryChangeQueue />
              </ProtectedRoute>
            </StandaloneWindowWrapper>
          }
        />
        <Route
          path="view/surveillance-findings"
          element={
            <StandaloneWindowWrapper title="Surveillance Findings Queue">
              <ProtectedRoute>
                <SurveillanceFindingsQueue />
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

        <Route path="fabric/custom-components" element={<ProtectedRoute><CustomComponentPage /></ProtectedRoute>} />

        {/* ═══════════════════════════════════════════════════════════════════
            OPERATIONS - Scheduling and workflows
            ═══════════════════════════════════════════════════════════════════ */}
        {/* D1(a): Scheduler Intelligence console = Schedules console on /api/schedules. */}
        <Route path="scheduler-intelligence" element={<Navigate to={`/${locale}/automation/schedules`} replace />} />
        <Route path="scheduler-intelligence/*" element={<Navigate to={`/${locale}/automation/schedules`} replace />} />
        {/* S1/S7 job scheduler pages retired; same console. */}
        <Route path="scheduler/*" element={<Navigate to={`/${locale}/automation/schedules`} replace />} />

        {/* ═══════════════════════════════════════════════════════════════════
            INTELLIGENCE - Optimization and observability
            ═══════════════════════════════════════════════════════════════════ */}
        <Route path="optimization/:optimizationId" element={<ProtectedRoute><ASOOptimizationDetail /></ProtectedRoute>} />
        
        <Route path="simulation" element={<ProtectedRoute><SimulationWorkspace /></ProtectedRoute>} />
        <Route path="simulation/rebalance" element={<ProtectedRoute><RebalancingWizard /></ProtectedRoute>} />
        <Route path="simulation/:id" element={<ProtectedRoute><ScenarioDetail /></ProtectedRoute>} />
        <Route path="simulation/compare" element={<ProtectedRoute><ScenarioComparison /></ProtectedRoute>} />

        {/* ═══════════════════════════════════════════════════════════════════
            CONSUME - Reports and analytics
            ═══════════════════════════════════════════════════════════════════ */}
        <Route path="reports/:reportId/edit" element={<ProtectedRoute><ReportBuilderPage /></ProtectedRoute>} />
        
        <Route path="analytics/factors" element={<ProtectedRoute><FactorAnalysisPage /></ProtectedRoute>} />
        <Route path="analytics/factors/:portfolioID?" element={<ProtectedRoute><FactorAnalysisPage /></ProtectedRoute>} />
        <Route path="fixed-income" element={<ProtectedRoute><FixedIncomeDashboard /></ProtectedRoute>} />
        <Route path="private-markets" element={<ProtectedRoute><AIPortfolioRebalancer /></ProtectedRoute>} />
        <Route path="analytics/rebalancer" element={<ProtectedRoute><AIPortfolioRebalancer /></ProtectedRoute>} />
        <Route path="crypto/portfolio" element={<ProtectedRoute><CryptoPortfolioCenter clientId={""} /></ProtectedRoute>} />
        <Route path="wealth/feed" element={<ProtectedRoute><Feed /></ProtectedRoute>} />

        {/* Legacy / Utilities / Misc */}
        <Route path="fabric/audit-logs" element={<ProtectedRoute><AuditLogsPage /></ProtectedRoute>} />
        <Route path="core/audit-explorer" element={<ProtectedRoute><AuditExplorerPage /></ProtectedRoute>} />

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
        ) : (
          <div style={{ padding: '32px', color: '#94a3b8', background: '#050d1a', height: '100%' }}>Loading view...</div>
        )}
      </React.Suspense>
    </>
  );
}

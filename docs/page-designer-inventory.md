# Page Designer inventory

Generated from `frontend/src/AppRoutes.tsx` and `frontend/src/pages/page-studio/studioRoutes.ts`. 
Each hand-built route is a React page that is not a Page Studio page. The rule (memory: `every-page-in-page-designer`) is that every screen is a Page Studio page: a blueprint, operations, a gold-copy core page, a route, and a menu placement.

## Counts

- Page Studio routes today: **21** (see below).
- Hand-built routes to convert: **102** (after excluding 29 tooling/runtime routes listed separately).
- Redirect routes (no page; keep as aliases): **18**.

`?` means the component source could not be located automatically (inline or multi-line lazy import), so its size is unknown.

Effort is a rough proxy: the lines of code the page reaches through relative imports (its transitive closure). S is under 1,000 lines, M is 1,000 to 5,000, L is over 5,000. A thin route file can wrap a large component, so the own-file count alone is misleading. The real cost depends on data operations and domain widgets, which this does not measure.

## Already in Page Designer (served by `STUDIO_ROUTES`)

| Route | Page slug |
|---|---|
| `/data/mastering` | `mastering-console` |
| `/data/staging-bindings` | `staging-bindings` |
| `/data/pipelines` | `data-pipelines` |
| `/data/pipelines/:id` | `data-pipeline-editor` |
| `/data/mdm/source-hierarchy` | `mdm-source-hierarchy` |
| `/data/mdm/match-rules` | `mdm-match-rules` |
| `/data/mdm/vendors` | `mdm-vendors` |
| `/automation/schedules` | `schedules` |
| `/data/mdm/source-scoring` | `mdm-source-scoring` |
| `/governance/validation-rules` | `validation-rules` |
| `/data/validation-rules` | `validation-rules` |
| `/data/lakehouse-streaming` | `lakehouse-streaming` |
| `/infrastructure/streaming` | `lakehouse-streaming` |
| `/lakehouse-streaming` | `lakehouse-streaming` |
| `/system/lakehouse` | `system-lakehouse` |
| `/build/cubes` | `cubes-catalog` |
| `/core/domains` | `core-domains` |
| `/fabric/settings` | `fabric-settings` |
| `/admin/rbac/roles` | `rbac-roles` |
| `/build/cubes/new` | `cube-designer` |
| `/build/cubes/:id` | `cube-designer` |

## Needs a decision first (tooling and runtime)

These are authoring tools or generic runtimes. A designer cannot be built inside itself, so each needs an explicit decision: keep as a tool route, or host it as a Page Studio page.

| Route | Component | Source | Why it is listed here |
|---|---|---|---|
| `/core/flow-builder` | `UisceBuilder` | `frontend/src/features/uisce-builder/UisceBuilder.tsx` | authoring tool |
| `/query-builder` | `BusinessObjectQueryBuilder` | `frontend/src/features/query-builder/pages/BusinessObjectQueryBuilder.tsx` | authoring tool |
| `/query-builder/editor/:id?` | `SavedQueryEditor` | `frontend/src/features/query-builder/pages/SavedQueryEditor.tsx` | authoring tool |
| `/sql-studio` | `SqlStudioPage` | `frontend/src/pages/analytical/SqlStudioPage.tsx` | authoring tool |
| `/api-studio` | `APIStudioPage` | `frontend/src/pages/api-studio/APIStudioPage.tsx` | authoring tool |
| `/page-studio` | `PageStudioListPage` | `frontend/src/pages/page-studio/PageStudioListPage.tsx` | authoring tool |
| `/page-studio/:id` | `PageStudioDetailsPage` | `frontend/src/pages/page-studio/PageStudioDetailsPage.tsx` | authoring tool |
| `/menu-designer` | `MenuDesignerPage` | `frontend/src/pages/menu-designer/MenuDesignerPage.tsx` | authoring tool |
| `/pages` | `PageBrowser` | `frontend/src/pages/PageBrowser.tsx` | generic page runtime |
| `/pages/:slug` | `PageBrowser` | `frontend/src/pages/PageBrowser.tsx` | generic page runtime |
| `/pages/:slug/:recordId` | `PageBrowser` | `frontend/src/pages/PageBrowser.tsx` | generic page runtime |
| `/view/page/:slug` | `StandaloneWindowWrapper` | `frontend/src/components/desktop/index.ts` | generic page runtime |
| `/view/page/:slug/:recordId` | `StandaloneWindowWrapper` | `frontend/src/components/desktop/index.ts` | generic page runtime |
| `/view/rebalancer` | `StandaloneWindowWrapper` | `frontend/src/components/desktop/index.ts` | generic page runtime |
| `/view/scenario` | `StandaloneWindowWrapper` | `frontend/src/components/desktop/index.ts` | generic page runtime |
| `/view/fixed-income` | `StandaloneWindowWrapper` | `frontend/src/components/desktop/index.ts` | generic page runtime |
| `/view/compliance-blotter` | `StandaloneWindowWrapper` | `frontend/src/components/desktop/index.ts` | generic page runtime |
| `/view/regulatory-queue` | `StandaloneWindowWrapper` | `frontend/src/components/desktop/index.ts` | generic page runtime |
| `/view/surveillance-findings` | `StandaloneWindowWrapper` | `frontend/src/components/desktop/index.ts` | generic page runtime |
| `/workspace` | `UniversalWorkspaceHub` | `frontend/src/components/docking/UniversalWorkspaceHub.tsx` | generic page runtime |
| `/app/data-product/:pageKey` | `DynamicDataProductPage` | `frontend/src/pages/DynamicDataProductPage.tsx` | generic page runtime |
| `/client-portal/workflow-studio` | `WorkflowStudioPage` | `frontend/src/pages/WorkflowStudioPage.tsx` | authoring tool |
| `/client-portal/rules-editor` | `BusinessRuleEditorPage` | `frontend/src/pages/BusinessRuleEditorPage.tsx` | authoring tool |
| `/fabric/custom-components` | `CustomComponentPage` | `frontend/src/pages/CustomComponentPage.tsx` | authoring tool |
| `/core/workflow-designer` | `WorkflowDesignerPage` | `frontend/src/features/workflow/pages/WorkflowDesignerPage.tsx` | authoring tool |
| `/bp-console` | `BPConsolePage` | `frontend/src/features/bp-console/pages/BPConsolePage.tsx` | authoring tool |
| `/bp-console/:tab` | `BPConsolePage` | `frontend/src/features/bp-console/pages/BPConsolePage.tsx` | authoring tool |
| `/bp-console/instances` | `BPConsolePage` | `frontend/src/features/bp-console/pages/BPConsolePage.tsx` | authoring tool |
| `/bp-console/queues` | `BPConsolePage` | `frontend/src/features/bp-console/pages/BPConsolePage.tsx` | authoring tool |

## Hand-built routes by area

Effort split: S 23, M 11, L 64.

| Area | Count |
|---|---|
| `core` | 18 |
| `admin` | 13 |
| `catalog` | 11 |
| `fabric` | 9 |
| `reports` | 6 |
| `analytics` | 5 |
| `intelligence` | 4 |
| `simulation` | 4 |
| `compliance` | 3 |
| `secrets` | 3 |
| `observability` | 2 |
| `optimization` | 2 |
| `tenants` | 2 |
| `views` | 2 |
| `access-explanation` | 1 |
| `audit` | 1 |
| `bundle-explorer` | 1 |
| `bundles` | 1 |
| `business-objects` | 1 |
| `crypto` | 1 |
| `fixed-income` | 1 |
| `global-intelligence` | 1 |
| `governance` | 1 |
| `jit-request` | 1 |
| `nlq` | 1 |
| `pipelines` | 1 |
| `private-markets` | 1 |
| `schema-explorer` | 1 |
| `security` | 1 |
| `semantic-catalog` | 1 |
| `semantic-health` | 1 |
| `wealth` | 1 |

### `core`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/core/abbreviations` | `AbbreviationsPage` | `frontend/src/pages/core/AbbreviationsPage.tsx` | 24 | 6983 | API | L | Wave 3 |
| `/core/approval-inbox` | `ApprovalInboxPage` | `frontend/src/features/wealth/pages/ApprovalInboxPage.tsx` | 139 | 5394 | API | L | Wave 3 |
| `/core/approval-workflows` | `ApprovalWorkflowDashboard` | `frontend/src/pages/ApprovalWorkflowDashboard.tsx` | 473 | 7305 | API | L | Wave 3 |
| `/core/audit-explorer` | `AuditExplorerPage` | `frontend/src/features/workflow/pages/AuditExplorerPage.tsx` | 240 | 240 | **no API found** | S | Wave 1 |
| `/core/business-terms` | `BusinessTermsExplorer` | `frontend/src/features/glossary/BusinessTermsExplorer.tsx` | 825 | 9332 | API | L | Wave 3 |
| `/core/calculated-fields` | `CalculatedFieldBuilderPage` | `frontend/src/pages/CalculatedFieldBuilderPage.tsx` | 25 | 347 | API | S | Wave 1 |
| `/core/glossary` | `GlossaryExplorer` | `frontend/src/features/glossary/GlossaryExplorer.tsx` | 1431 | 13462 | API | L | Wave 3 |
| `/core/notifications` | `NotificationCenterPage` | `frontend/src/features/workflow/pages/NotificationCenterPage.tsx` | 190 | 7966 | API | L | Wave 3 |
| `/core/notifications/preferences` | `NotificationPreferencesPage` | `frontend/src/features/workflow/pages/NotificationPreferencesPage.tsx` | 39 | 7520 | API | L | Wave 3 |
| `/core/notifications/templates` | `NotificationTemplateEditorPage` | `frontend/src/features/workflow/pages/NotificationTemplateEditorPage.tsx` | 29 | 7414 | API | L | Wave 3 |
| `/core/process-catalog` | `ProcessCatalogPage` | `frontend/src/features/workflow/pages/ProcessCatalogPage.tsx` | 331 | 331 | **no API found** | S | Wave 1 |
| `/core/semantic-mapper` | `SemanticMapperPage` | `frontend/src/features/core/pages/SemanticMapperPage.tsx` | 11 | 9894 | API | L | Wave 3 |
| `/core/semantic-terms` | `GlossaryExplorer` | `frontend/src/features/glossary/GlossaryExplorer.tsx` | 1431 | 13462 | API | L | Wave 3 |
| `/core/sla-dashboard` | `SLADashboardPage` | `frontend/src/features/workflow/pages/SLADashboardPage.tsx` | 272 | 272 | **no API found** | S | Wave 1 |
| `/core/validation` | `InvestmentValidationPage` | `frontend/src/pages/InvestmentValidationPage.tsx` | 534 | 7536 | API | L | Wave 3 |
| `/core/validation-rules` | `AdvancedRuleBuilderPage` | `frontend/src/pages/AdvancedRuleBuilderPage.tsx` | 929 | 9233 | API | L | Wave 3 |
| `/core/validation-rules/all` | `SystemValidationsPage` | `frontend/src/pages/SystemValidationsPage.tsx` | 296 | 7007 | API | L | Wave 3 |
| `/core/validation-rules/editor` | `AdvancedRuleBuilderPage` | `frontend/src/pages/AdvancedRuleBuilderPage.tsx` | 929 | 9233 | API | L | Wave 3 |

### `admin`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/admin/entitlements` | `ProfilesDashboard` | `frontend/src/admin-v2/index.ts` | 7 | 8119 | API | L | Wave 3 |
| `/admin/entitlements/profiles/:profileKey` | `ProfileCustomizerRoute` | `None` | None | None | unknown (source not located) | ? | ? |
| `/admin/entitlements/profiles/:profileKey/components` | `EntitlementMatrixRoute` | `None` | None | None | unknown (source not located) | ? | ? |
| `/admin/llm` | `LLMConfigPage` | `frontend/src/pages/admin/LLMConfigPage.tsx` | 264 | 7089 | API | L | Wave 3 |
| `/admin/message-catalog` | `MessageCatalogPage` | `frontend/src/features/message-catalog/MessageCatalogPage.tsx` | 207 | 6061 | API | L | Wave 3 |
| `/admin/rbac/delegations` | `DelegationManagerPage` | `frontend/src/features/admin/pages/DelegationManagerPage.tsx` | 26 | 6725 | API | L | Wave 3 |
| `/admin/rbac/field-permissions` | `FieldPermissionEditorPage` | `frontend/src/features/admin/pages/FieldPermissionEditorPage.tsx` | 26 | 6129 | API | L | Wave 3 |
| `/admin/rbac/teams` | `TeamManagerPage` | `frontend/src/features/admin/pages/TeamManagerPage.tsx` | 28 | 6567 | API | L | Wave 3 |
| `/admin/rbac/user-roles` | `UserRoleAssignmentPage` | `frontend/src/features/admin/pages/UserRoleAssignmentPage.tsx` | 28 | 6380 | API | L | Wave 3 |
| `/admin/rbac/user-tenants` | `TenantUserAssignmentPage` | `frontend/src/features/admin/pages/TenantUserAssignmentPage.tsx` | 541 | 7505 | API | L | Wave 3 |
| `/admin/rbac/users` | `UserManagementPage` | `frontend/src/features/admin/pages/UserManagementPage.tsx` | 663 | 7374 | API | L | Wave 3 |
| `/admin/seeding` | `SeedingPage` | `frontend/src/features/admin/pages/SeedingPage.tsx` | 141 | 6935 | API | L | Wave 3 |
| `/admin/temporal-ops` | `TemporalOpsPage` | `frontend/src/features/admin/pages/TemporalOpsPage.tsx` | 309 | 1579 | API | M | Wave 2 |

### `catalog`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/catalog/ai-suggestions` | `AIBusinessTermSuggestionsPage` | `frontend/src/pages/catalog/AIBusinessTermSuggestionsPage.tsx` | 147 | 2846 | API | M | Wave 2 |
| `/catalog/api-inventory` | `ApiInventoryPage` | `frontend/src/features/catalog/pages/ApiInventoryPage.tsx` | 2123 | 8834 | API | L | Wave 3 |
| `/catalog/business-terms` | `BusinessTermsExplorer` | `frontend/src/features/glossary/BusinessTermsExplorer.tsx` | 825 | 9332 | API | L | Wave 3 |
| `/catalog/business-terms/:id` | `BusinessTermDetailPage` | `frontend/src/pages/catalog/BusinessTermDetailPage.tsx` | 271 | 6138 | API | L | Wave 3 |
| `/catalog/custom-fields` | `EntityPickerPage` | `frontend/src/features/custom-attributes/pages/EntityPickerPage.tsx` | 108 | 5971 | API | L | Wave 3 |
| `/catalog/custom-fields/:entityType` | `WorkbenchPage` | `frontend/src/features/custom-attributes/pages/WorkbenchPage.tsx` | 224 | 6783 | API | L | Wave 3 |
| `/catalog/edge-types` | `CatalogEdgeTypesPage` | `frontend/src/pages/catalog/CatalogEdgeTypesPage.tsx` | 468 | 8823 | API | L | Wave 3 |
| `/catalog/edge-types/:id` | `EdgeTypeDetailPage` | `frontend/src/pages/catalog/EdgeTypeDetailPage.tsx` | 698 | 8943 | API | L | Wave 3 |
| `/catalog/node-types` | `CatalogNodeTypesPage` | `frontend/src/pages/catalog/CatalogNodeTypesPage.tsx` | 428 | 7447 | API | L | Wave 3 |
| `/catalog/node-types/:id` | `NodeTypeDetailPage` | `frontend/src/pages/catalog/NodeTypeDetailPage.tsx` | 667 | 8912 | API | L | Wave 3 |
| `/catalog/semantic-terms` | `GlossaryExplorer` | `frontend/src/features/glossary/GlossaryExplorer.tsx` | 1431 | 13462 | API | L | Wave 3 |

### `fabric`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/fabric/audit-logs` | `AuditLogsPage` | `frontend/src/features/fabric/pages/AuditLogsPage.tsx` | 296 | 6620 | API | L | Wave 3 |
| `/fabric/bundles` | `BundleListPage` | `frontend/src/pages/bundles/BundleListPage.tsx` | 199 | 7041 | API | L | Wave 3 |
| `/fabric/bundles/:bundleId/edit` | `BundleEditor` | `frontend/src/pages/bundles/BundleEditor.tsx` | 273 | 5781 | API | L | Wave 3 |
| `/fabric/bundles/create` | `BundleEditor` | `frontend/src/pages/bundles/BundleEditor.tsx` | 273 | 5781 | API | L | Wave 3 |
| `/fabric/calculations` | `CalculationsLibraryPage` | `frontend/src/features/fabric/pages/CalculationsLibraryPage.tsx` | 1165 | 8938 | API | L | Wave 3 |
| `/fabric/dashboard` | `DashboardPage` | `frontend/src/features/fabric/pages/DashboardPage.tsx` | 111 | 6593 | API | L | Wave 3 |
| `/fabric/ip-whitelist` | `IPWhitelistManagementPage` | `frontend/src/features/fabric/pages/IPWhitelistManagementPage.tsx` | 102 | 7758 | API | L | Wave 3 |
| `/fabric/preaggregations` | `ManagementPage` | `frontend/src/features/fabric/pages/preaggregations/ManagementPage.tsx` | 264 | 464 | API | S | Wave 1 |
| `/fabric/tenants` | `TenantsManagementPage` | `frontend/src/features/fabric/pages/TenantsManagementPage.tsx` | 1043 | 8538 | API | L | Wave 3 |

### `reports`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/reports` | `ReportsPage` | `frontend/src/pages/analytical/ReportsPage.tsx` | 575 | 7680 | API | L | Wave 3 |
| `/reports/:reportId/edit` | `ReportBuilderPage` | `frontend/src/pages/ReportBuilderPage.tsx` | 61 | 18675 | API | L | Wave 3 |
| `/reports/builder` | `ReportBuilderPage` | `frontend/src/pages/ReportBuilderPage.tsx` | 61 | 18675 | API | L | Wave 3 |
| `/reports/library` | `ReportLibrary` | `frontend/src/features/reporting/components/ReportLibrary.tsx` | 2085 | 9952 | API | L | Wave 3 |
| `/reports/models` | `SemanticModelManager` | `frontend/src/features/semantic/components/SemanticModelManager.tsx` | 426 | 933 | API | S | Wave 1 |
| `/reports/queries` | `QueryLibrary` | `frontend/src/features/query-builder/pages/QueryLibrary.tsx` | 349 | 7238 | API | L | Wave 3 |

### `analytics`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/analytics/advisor-dashboard` | `AdvisorDashboard` | `frontend/src/pages/AdvisorDashboard.tsx` | 252 | 316 | **no API found** | S | Wave 1 |
| `/analytics/factors` | `FactorAnalysisPage` | `frontend/src/features/analytics/pages/FactorAnalysisPage.tsx` | 77 | 172 | API | S | Wave 1 |
| `/analytics/factors/:portfolioID?` | `FactorAnalysisPage` | `frontend/src/features/analytics/pages/FactorAnalysisPage.tsx` | 77 | 172 | API | S | Wave 1 |
| `/analytics/rebalancer` | `AIPortfolioRebalancer` | `frontend/src/components/AIPortfolioRebalancer.tsx` | 448 | 1712 | API | M | Wave 2 |
| `/analytics/scenario-analysis` | `ScenarioAnalysisPro` | `frontend/src/components/ScenarioAnalysisPro.tsx` | 306 | 7961 | API | L | Wave 3 |

### `intelligence`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/intelligence` | `IntelligenceDashboard` | `frontend/src/pages/intelligence/IntelligenceDashboard.tsx` | 282 | 639 | **no API found** | S | Wave 1 |
| `/intelligence/data-quality` | `DataQualityMonitorPage` | `frontend/src/pages/intelligence/DataQualityMonitorPage.tsx` | 302 | 659 | **no API found** | S | Wave 1 |
| `/intelligence/index-advisor` | `IndexAdvisorPage` | `frontend/src/pages/intelligence/IndexAdvisorPage.tsx` | 219 | 576 | **no API found** | S | Wave 1 |
| `/intelligence/storage` | `StorageTieringPage` | `frontend/src/pages/intelligence/StorageTieringPage.tsx` | 361 | 773 | API | S | Wave 1 |

### `simulation`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/simulation` | `SimulationWorkspace` | `frontend/src/pages/simulation/SimulationWorkspace.tsx` | 164 | 242 | API | S | Wave 1 |
| `/simulation/:id` | `ScenarioDetail` | `frontend/src/pages/simulation/ScenarioDetail.tsx` | 196 | 274 | API | S | Wave 1 |
| `/simulation/compare` | `ScenarioComparison` | `frontend/src/pages/simulation/ScenarioComparison.tsx` | 16 | 16 | **no API found** | S | Wave 1 |
| `/simulation/rebalance` | `RebalancingWizard` | `frontend/src/pages/simulation/RebalancingWizard.tsx` | 405 | 483 | API | S | Wave 1 |

### `compliance`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/compliance/blotter` | `ComplianceDecisionBlotter` | `frontend/src/components/Compliance/ComplianceDecisionBlotter.tsx` | 814 | 2006 | API | M | Wave 2 |
| `/compliance/regulatory` | `RegulatoryChangeQueue` | `None` | None | None | unknown (source not located) | ? | ? |
| `/compliance/surveillance` | `SurveillanceFindingsQueue` | `None` | None | None | unknown (source not located) | ? | ? |

### `secrets`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/secrets/audit` | `SecretsAuditPage` | `frontend/src/features/secrets/index.ts` | 4 | 6590 | API | L | Wave 3 |
| `/secrets/config` | `SecretsConfigPage` | `frontend/src/features/secrets/index.ts` | 4 | 6590 | API | L | Wave 3 |
| `/secrets/monitoring` | `SecretsMonitoringPage` | `frontend/src/features/secrets/index.ts` | 4 | 6590 | API | L | Wave 3 |

### `observability`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/observability` | `ObservabilityDashboard` | `frontend/src/pages/ObservabilityDashboard.tsx` | 390 | 390 | API | S | Wave 1 |
| `/observability/slos` | `SLODashboard` | `frontend/src/pages/SLODashboard.tsx` | 83 | 3013 | API | M | Wave 2 |

### `optimization`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/optimization` | `OptimizationCenter` | `frontend/src/pages/OptimizationCenter.tsx` | 494 | 2463 | API | M | Wave 2 |
| `/optimization/:optimizationId` | `ASOOptimizationDetail` | `frontend/src/components/aso/ASOOptimizationDetail.tsx` | 353 | 2534 | API | M | Wave 2 |

### `tenants`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/tenants` | `TenantDetailPageV2` | `frontend/src/features/tenants/pages/TenantDetailPageV2.tsx` | 1680 | 16775 | API | L | Wave 3 |
| `/tenants/:tenantId` | `TenantDetailPageV2` | `frontend/src/features/tenants/pages/TenantDetailPageV2.tsx` | 1680 | 16775 | API | L | Wave 3 |

### `views`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/views` | `ViewsCatalogPage` | `frontend/src/features/views/pages/ViewsCatalogPage.tsx` | 652 | 8686 | API | L | Wave 3 |
| `/views/:id` | `ViewDetailsPage` | `frontend/src/features/views/pages/ViewDetailsPage.tsx` | 179 | 11667 | API | L | Wave 3 |

### `access-explanation`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/access-explanation` | `AccessExplanationExample` | `frontend/src/AccessExplanationExample.tsx` | 5 | 18 | **no API found** | S | Wave 1 |

### `audit`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/audit` | `AuditExplorer` | `frontend/src/components/audit/AuditExplorer.tsx` | 575 | 7057 | API | L | Wave 3 |

### `bundle-explorer`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/bundle-explorer` | `BundleExplorer` | `frontend/src/components/BundleExplorer.tsx` | 262 | 1829 | API | M | Wave 2 |

### `bundles`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/bundles` | `MicroBundleCatalogExample` | `frontend/src/MicroBundleCatalogExample.tsx` | 88 | 5956 | API | L | Wave 3 |

### `business-objects`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/business-objects/:id` | `BusinessObjectDetailsPage` | `frontend/src/pages/BusinessObjectDetailsPage.tsx` | 2088 | 22630 | API | L | Wave 3 |

### `crypto`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/crypto/portfolio` | `CryptoPortfolioCenter` | `frontend/src/features/crypto/CryptoPortfolioCenter.tsx` | 475 | 475 | API | S | Wave 1 |

### `fixed-income`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/fixed-income` | `FixedIncomeDashboard` | `frontend/src/components/FixedIncomeDashboard.tsx` | 449 | 3039 | API | M | Wave 2 |

### `global-intelligence`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/global-intelligence` | `GlobalNLQueryPage` | `frontend/src/pages/GlobalNLQueryPage.tsx` | 252 | 7069 | API | L | Wave 3 |

### `governance`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/governance/changesets` | `GovernanceConsolePage` | `frontend/src/pages/governance/GovernanceConsolePage.tsx` | 430 | 430 | **no API found** | S | Wave 1 |

### `jit-request`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/jit-request` | `JITRequestPanelExample` | `frontend/src/JITRequestPanelExample.tsx` | 16 | 125 | **no API found** | S | Wave 1 |

### `nlq`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/nlq` | `NLQPage` | `frontend/src/pages/nlq/NLQPage.tsx` | 284 | 8310 | API | L | Wave 3 |

### `pipelines`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/pipelines` | `PipelinesPage` | `frontend/src/pages/analytical/PipelinesPage.tsx` | 546 | 7651 | API | L | Wave 3 |

### `private-markets`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/private-markets` | `AIPortfolioRebalancer` | `frontend/src/components/AIPortfolioRebalancer.tsx` | 448 | 1712 | API | M | Wave 2 |

### `schema-explorer`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/schema-explorer` | `SchemaExplorerPage` | `frontend/src/features/schema-explorer/pages/SchemaExplorer.tsx` | 1263 | 8129 | API | L | Wave 3 |

### `security`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/security/*` | `SecurityRoutes` | `frontend/src/features/security/routes.tsx` | 26 | 7423 | API | L | Wave 3 |

### `semantic-catalog`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/semantic-catalog` | `SemanticCatalogDetailPage` | `frontend/src/pages/analytical/SemanticCatalogDetailPage.tsx` | 1050 | 8408 | API | L | Wave 3 |

### `semantic-health`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/semantic-health` | `SemanticHealthDashboard` | `frontend/src/pages/SemanticHealthDashboard.tsx` | 442 | 442 | API | S | Wave 1 |

### `wealth`

| Route | Component | Source | Own lines | Transitive lines | Data | Effort | Wave |
|---|---|---|---|---|---|---|---|
| `/wealth/feed` | `Feed` | `frontend/src/features/feed/components/Feed.tsx` | 40 | 2007 | API | M | Wave 2 |

## Static screens: no API found in the page's imports

These routes reach no API call through their imports. They are most likely static or mock UI (hardcoded rows, local state). Converting them to Page Studio would copy the mock data, so each needs a decision: build the real data source, or retire the screen.

| Route | Component | Transitive lines |
|---|---|---|
| `/access-explanation` | `AccessExplanationExample` | 18 |
| `/analytics/advisor-dashboard` | `AdvisorDashboard` | 316 |
| `/core/audit-explorer` | `AuditExplorerPage` | 240 |
| `/core/process-catalog` | `ProcessCatalogPage` | 331 |
| `/core/sla-dashboard` | `SLADashboardPage` | 272 |
| `/governance/changesets` | `GovernanceConsolePage` | 430 |
| `/intelligence` | `IntelligenceDashboard` | 639 |
| `/intelligence/data-quality` | `DataQualityMonitorPage` | 659 |
| `/intelligence/index-advisor` | `IndexAdvisorPage` | 576 |
| `/jit-request` | `JITRequestPanelExample` | 125 |
| `/simulation/compare` | `ScenarioComparison` | 16 |

## Redirects (keep; not pages)

| Route | Target |
|---|---|
| `/business-objects/new` | (see `AppRoutes.tsx:278`) |
| `/scheduler-intelligence` | (see `AppRoutes.tsx:446`) |
| `/scheduler-intelligence/*` | (see `AppRoutes.tsx:447`) |
| `/scheduler/*` | (see `AppRoutes.tsx:449`) |
| `/core/genui-inbox` | (see `AppRoutes.tsx:517`) |
| `/rbac` | (see `AppRoutes.tsx:520`) |
| `/glossary` | (see `AppRoutes.tsx:521`) |
| `/abbreviations` | (see `AppRoutes.tsx:522`) |
| `/validation-rules` | (see `AppRoutes.tsx:523`) |
| `/flow-builder` | (see `AppRoutes.tsx:524`) |
| `/api-designer` | (see `AppRoutes.tsx:525`) |
| `/page-designer` | (see `AppRoutes.tsx:526`) |
| `/change-review` | (see `AppRoutes.tsx:527`) |
| `/change-reviews` | (see `AppRoutes.tsx:528`) |
| `/change-reviews/:id` | (see `AppRoutes.tsx:529`) |
| `/incidents/:id` | (see `AppRoutes.tsx:530`) |
| `/scheduler` | (see `AppRoutes.tsx:531`) |
| `/aso` | (see `AppRoutes.tsx:532`) |

## Proposed waves

- **Wave 1 (S):** smaller real screens with an API. Before converting, exclude the static screens above. Lowest risk; proves the conversion recipe (blueprint + operations + gold-copy seed + route + menu).
- **Wave 2 (M):** multi-section pages with their own data operations. Start with `catalog/*` and `core/*`, which share Glossary and Governance data.
- **Wave 3 (L):** large canvases, wizards and analytics. Convert only after the domain-widget question is settled (memory: rich domain UI is a registered component inside a studio page; confirm before adding more).
- **Decision:** the tooling and runtime routes above.

Each wave lands in its own PR. A converted route is removed from `AppRoutes.tsx` in the same PR, and its menu placement is seeded in the gold copy.

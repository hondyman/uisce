# Frontend Routing Map

**Source files (read-only reference):**
- `frontend/src/routes/localeShell.tsx` — top-level, no `/:locale/` prefix
- `frontend/src/AppRoutes.tsx` — locale-prefixed app routes (under `/:locale/*`)
- `frontend/src/components/Navigation.tsx` — global sidebar menu config
- `frontend/src/components/MainNavigation.tsx` — global sidebar component
- `frontend/src/components/MobileResponsiveNavigation.tsx` — mobile variant
- `frontend/src/pages/page-studio/studioRoutes.ts` — page-studio-served routes (`STUDIO_ROUTES`)

**Counts (origin/main):** 8 top-level + 144 app routes + 15 legacy `<Navigate>` redirects = **167 routes total**.

---

## 1. Top-level (no locale prefix) — `frontend/src/routes/localeShell.tsx`

| Path | Element | Notes |
|---|---|---|
| `/login` | `<LoginPage>` | auth flow |
| `/auth/callback` | `<AuthCallbackPage>` | OAuth callback |
| `/api-studio` | `<APIStudioPage>` | unprefixed canonical |
| `/page-studio` | `<PageStudioListPage>` | unprefixed canonical |
| `/app/:slug` | `<RuntimePage>` (`PageRuntimeRenderer`) | runtime page embed |
| `/change-review` | `<ChangeReviewPage>` | unprefixed canonical |
| `/` | `<RootRedirect>` | redirects to preferred locale |
| `/:locale/*` | `<LocaleLayout>` | single splat → `<AppRoutes>` → `<ProtectedApp>` → nested `<Routes>` |

---

## 2. App routes (under `/:locale/`) — `frontend/src/AppRoutes.tsx`

### 2.1 Tenants

| Path | Element |
|---|---|
| `tenants` | `<TenantDetailPageV2>` |
| `tenants/:tenantId` | `<TenantDetailPageV2>` |

### 2.2 Admin — RBAC

| Path | Element |
|---|---|
| `admin/rbac/roles` | `<RoleManagerPage>` |
| `admin/rbac/users` | `<UserManagementPage>` |
| `admin/rbac/teams` | `<TeamManagerPage>` |
| `admin/rbac/delegations` | `<DelegationManagerPage>` |
| `admin/rbac/field-permissions` | `<FieldPermissionEditorPage>` |
| `admin/rbac/user-roles` | `<UserRoleAssignmentPage>` |
| `admin/rbac/user-tenants` | `<TenantUserAssignmentPage>` |

### 2.3 Admin — Misc

| Path | Element |
|---|---|
| `admin/llm` | `<LLMConfigPage>` |
| `admin/message-catalog` | `<MessageCatalogPage>` |
| `admin/seeding` | `<SeedingPage>` |
| `admin/temporal-ops` | `<TemporalOpsPage>` |
| `admin/entitlements` | `<ProfilesDashboard>` |
| `admin/entitlements/profiles/:profileKey` | `<ProfileCustomizerRoute>` |
| `admin/entitlements/profiles/:profileKey/components` | `<EntitlementMatrixRoute>` |

### 2.4 Fabric

| Path | Element |
|---|---|
| `fabric/ip-whitelist` | `<IPWhitelistManagementPage>` |
| `fabric/tenants` | `<TenantsManagementPage>` |
| `fabric/bundles` | `<BundleListPage>` |
| `fabric/bundles/create` | `<BundleEditor>` |
| `fabric/bundles/:bundleId/edit` | `<BundleEditor>` |
| `fabric/dashboard` | `<DashboardPage>` |
| `fabric/audit-logs` | `<AuditLogsPage>` |
| `fabric/settings` | `<SettingsPage>` |
| `fabric/preaggregations` | `<ManagementPage>` (tenantId=`default`, datasourceId=`default`) |
| `fabric/calculations` | `<CalculationsLibraryPage>` (tenantId=`default`, datasourceId=`default`) |

### 2.5 Secrets

| Path | Element |
|---|---|
| `secrets/config` | `<SecretsConfigPage>` (tenantId=`default`) |
| `secrets/audit` | `<SecretsAuditPage>` (tenantId=`default`) |
| `secrets/monitoring` | `<SecretsMonitoringPage>` (tenantId=`default`) |

### 2.6 Audit

| Path | Element |
|---|---|
| `audit` | `<AuditExplorer>` (tenantId=`default`) |
| `core/audit-explorer` | `<AuditExplorerPage>` |

### 2.7 Security

| Path | Element |
|---|---|
| `security/*` | `<SecurityRoutes>` (nested) |

### 2.8 Core — Glossary / Catalog

| Path | Element |
|---|---|
| `core/glossary` | `<GlossaryExplorer>` |
| `core/semantic-terms` | `<GlossaryExplorer>` |
| `catalog/semantic-terms` | `<GlossaryExplorer>` |
| `core/business-terms` | `<BusinessTermsExplorer>` |
| `catalog/business-terms` | `<BusinessTermsExplorer>` |
| `catalog/business-terms/:id` | `<BusinessTermDetailPage>` |
| `core/abbreviations` | `<AbbreviationsPage>` |
| `core/domains` | `<DomainsManagementPage>` |
| `core/semantic-mapper` | `<SemanticMapperPage>` |

### 2.9 Core — Catalog / Schema

| Path | Element |
|---|---|
| `schema-explorer` | `<SchemaExplorerPage>` |
| `catalog/api-inventory` | `<ApiInventoryPage>` |
| `catalog/custom-fields` | `<EntityPickerPage>` |
| `catalog/custom-fields/:entityType` | `<WorkbenchPage>` |
| `catalog/node-types` | `<CatalogNodeTypesPage>` |
| `catalog/node-types/:id` | `<NodeTypeDetailPage>` |
| `catalog/edge-types` | `<CatalogEdgeTypesPage>` |
| `catalog/edge-types/:id` | `<EdgeTypeDetailPage>` |
| `catalog/ai-suggestions` | `<AIBusinessTermSuggestionsPage>` |

### 2.10 Business Objects & Semantic Health

| Path | Element |
|---|---|
| `business-objects` | `<BusinessObjectsPage>` |
| `business-objects/new` | `<Navigate>` redirect to `business-objects?new=1` |
| `business-objects/:id` | `<BusinessObjectDetailsPage>` |
| `semantic-health` | `<SemanticHealthDashboard>` |

### 2.11 Views

| Path | Element |
|---|---|
| `views` | `<ViewsCatalogPage>` |
| `views/:id` | `<ViewDetailsPage>` |

### 2.12 Workflow / Studio

| Path | Element |
|---|---|
| `core/flow-builder` | `<UisceBuilder>` |
| `core/validation` | `<InvestmentValidationPage>` |
| `core/validation-rules` | `<AdvancedRuleBuilderPage>` |
| `core/validation-rules/editor` | `<AdvancedRuleBuilderPage>` |
| `core/validation-rules/all` | `<SystemValidationsPage>` |
| `core/calculated-fields` | `<CalculatedFieldBuilderPage>` |
| `core/approval-inbox` | `<ApprovalInboxPage>` |
| `core/sla-dashboard` | `<SLADashboardPage>` |
| `core/genui-chat` | `<GenUIChatPage>` |
| `core/genui-proposal` | `<GenUIProposalDemoPage>` |
| `core/genui-inbox` | `<GenUIApprovalInboxPage>` |

### 2.13 Query / SQL / Pipelines / Reports

| Path | Element |
|---|---|
| `query-builder/editor/:id?` | `<SavedQueryEditor>` |
| `sql-studio` | `<SqlStudioPage>` |
| `semantic-catalog` | `<SemanticCatalogDetailPage>` |
| `pipelines` | `<PipelinesPage>` |
| `reports` | `<ReportsPage>` |
| `reports/library` | `<ReportLibrary>` |
| `reports/builder` | `<ReportBuilderPage>` |
| `reports/queries` | `<QueryLibrary>` |
| `reports/:reportId/edit` | `<ReportBuilderPage>` |
| `reports/models` | `<SemanticModelManager>` |

### 2.14 Studio pages

| Path | Element |
|---|---|
| `api-studio` | `<APIStudioPage>` |
| `page-studio` | `<PageStudioListPage>` |
| `page-studio/:id` | `<PageStudioDetailsPage>` |
| `menu-designer` | `<MenuDesignerPage>` |
| `pages` | `<PageBrowser>` |
| `pages/:slug` | `<PageBrowser>` |
| `pages/:slug/:recordId` | `<PageBrowser>` |

### 2.15 Runtime view pages

| Path | Element |
|---|---|
| `view/page/:slug` | `<RuntimePage>` |
| `view/page/:slug/:recordId` | `<RuntimePage>` |
| `view/rebalancer` | `<RuntimePage>` |
| `view/scenario` | `<RuntimePage>` |
| `view/fixed-income` | `<RuntimePage>` |

### 2.16 Intelligence / Optimization / Observability

| Path | Element |
|---|---|
| `intelligence` | `<IntelligenceDashboard>` |
| `intelligence/index-advisor` | `<IndexAdvisorPage>` |
| `intelligence/storage` | `<StorageTieringPage>` |
| `intelligence/data-quality` | `<DataQualityMonitorPage>` |
| `optimization` | `<OptimizationCenter scope="global">` |
| `optimization/:optimizationId` | `<ASOOptimizationDetail>` |
| `observability` | `<ObservabilityDashboard>` |
| `observability/slos` | `<SLODashboard>` |
| `nlq` | `<NLQPage>` |
| `global-intelligence` | `<GlobalNLQueryPage>` |

### 2.17 Simulation

| Path | Element |
|---|---|
| `simulation` | `<SimulationWorkspace>` |
| `simulation/rebalance` | `<RebalancingWizard>` |
| `simulation/:id` | `<ScenarioDetail>` |
| `simulation/compare` | `<ScenarioComparison>` |

### 2.18 Analytics / Wealth / Fixed-Income / Crypto

| Path | Element |
|---|---|
| `analytics/factors` | `<FactorAnalysisPage>` |
| `analytics/factors/:portfolioID?` | `<FactorAnalysisPage>` |
| `analytics/scenario-analysis` | `<ScenarioAnalysisPro>` |
| `analytics/rebalancer` | `<AIPortfolioRebalancer>` |
| `analytics/advisor-dashboard` | `<AdvisorDashboard>` |
| `fixed-income` | `<FixedIncomeDashboard>` |
| `private-markets` | `<AIPortfolioRebalancer>` |
| `crypto/portfolio` | `<CryptoPortfolioCenter clientId="">` |
| `wealth/feed` | `<Feed>` |

### 2.19 Bundles / Examples

| Path | Element |
|---|---|
| `bundles` | `<MicroBundleCatalogExample>` |
| `bundle-explorer` | `<BundleExplorer>` |
| `jit-request` | `<JITRequestPanelExample>` |
| `access-explanation` | `<AccessExplanationExample>` |

### 2.20 Change Review / Incidents

| Path | Element |
|---|---|
| `change-reviews/:id` | `<ChangeReviewPage>` |
| `incidents/:id` | `<IncidentPage>` |

### 2.21 Studio-served routes (`STUDIO_ROUTES`)

These are routes served by Page Studio pages rather than hand-built components.
The list is the **only authoritative map** between Page Studio slugs and the
URL paths they appear at — `AppRoutes.tsx` reads from it.

| App path | Page Studio slug |
|---|---|
| `data/mastering` | `mastering-console` |
| `data/staging-bindings` | `staging-bindings` |
| `data/pipelines` | `data-pipelines` |
| `data/pipelines/:id` | `data-pipeline-editor` |
| `data/mdm/source-hierarchy` | `mdm-source-hierarchy` |
| `data/mdm/match-rules` | `mdm-match-rules` |
| `data/mdm/vendors` | `mdm-vendors` |
| `automation/schedules` | `schedules` |
| `data/mdm/source-scoring` | `mdm-source-scoring` |
| `governance/validation-rules` | `validation-rules` |
| `data/validation-rules` | `validation-rules` |
| `data/lakehouse-streaming` | `lakehouse-streaming` |
| `infrastructure/streaming` | `lakehouse-streaming` |
| `lakehouse-streaming` | `lakehouse-streaming` |
| `system/lakehouse` | `system-lakehouse` |

### 2.22 Legacy redirects (no component — preserve URLs)

| From | To |
|---|---|
| `rbac` | `admin/rbac/roles` |
| `glossary` | `core/glossary` |
| `abbreviations` | `core/abbreviations` |
| `validation-rules` | `core/validation-rules` |
| `flow-builder` | `core/flow-builder` |
| `api-designer` | `api-studio` |
| `page-designer` | `page-studio` |
| `change-review` | `governance/changesets` |
| `change-reviews` | `governance/changesets` |
| `scheduler` | `automation/schedules` |
| `aso` | `optimization` |

### 2.23 Catch-all (locally-scoped empty path)

| Path | Element |
|---|---|
| `""` (locale root) | `<BundleExplorer>` |

---

## 3. Sidebar / menu configuration — `frontend/src/components/Navigation.tsx`

Sidebar groups, with their menu items. The path column is the `to:` field — i.e.
the URL the user lands on when the item is clicked.

| Group | Item id | Path | Notes |
|---|---|---|---|
| Analytics | metrics | `/metrics` | **dangle** — see Notes §4.3 |
| Analytics | nlq | `/nlq` | |
| Analytics | fixed-income | `/fixed-income` | |
| Analytics | scenario | `/analytics/scenario-analysis` | |
| Analytics | rebalancer | `/analytics/rebalancer` | |
| Analytics | preaggs | `/fabric/preaggregations` | |
| Analytics | calc | `/fabric/calculations` | |
| Scheduler | dashboard | `/scheduler` | legacy URL; the route is a redirect to `automation/schedules` |
| Admin | llm | `/admin/llm` | |
| Admin | menu-designer | `/menu-designer` | **the editor, not a link** — see Notes §4.2 |

(Remaining sidebar items duplicate route entries already in §2; only the items
where the path is interesting or doesn't match a registered route are listed
here.)

---

## 4. Notes — gotchas and known desyncs

### 4.1 Duplicate `MainNavigation` export

Two files export a symbol named `MainNavigation`:

- `frontend/src/components/MainNavigation.tsx:564` —
  `export const MainNavigation: React.FC<...> = () => {...}`
  The desktop / primary sidebar component.
- `frontend/src/components/MobileResponsiveNavigation.tsx:62` —
  `export const MainNavigation: React.FC<...> = ({ onToggleTheme }) => {...}`
  The mobile variant, but **with the same export name**.

A barrel re-export at any layer will pick one or the other. Tests that import
`MainNavigation` from a non-relative path may get the wrong component depending
on the import order. Files that need the mobile variant should import
`MobileMainNavigation` or import the source file directly. (Looking at this
file naming — the mobile variant probably should be exported as
`MobileMainNavigation`, leaving the unqualified `MainNavigation` for the
desktop one; that rename is a one-line change but is out of scope for a docs
commit.)

### 4.2 `/menu-designer` is a different menu

`/menu-designer` is the URL of the **Menu Designer** tool — a tree-editor UI
for the *per-tenant navigation menu* stored in `navigation_menu_nodes`. The
tool itself is reachable from the global sidebar (line 197 and 300 of
`MainNavigation.tsx`), but its purpose is to edit a dynamic, per-tenant
structure, not the global sidebar you see today.

In other words: **the sidebar that lists "Menu Designer" is a *hand-coded* menu;
"Menu Designer" the route is the *editor for a different menu altogether*.** A
reader who hits `/menu-designer` is editing tenant-scoped navigation, not the
global navigation.

### 4.3 Route ↔ menu desync (the `/metrics` link)

`Navigation.tsx:25` declares:

```
{ id: 'metrics', label: 'Metrics Console', ..., to: '/metrics', icon: '📊' },
```

…but no `<Route path="metrics" …>` exists in `AppRoutes.tsx`. Clicking
"Metrics Console" in the global sidebar lands on the catch-all `<Route path=""`
element, which renders `<BundleExplorer>`. The metrics console exists
(`MetricsConsolePage.tsx`, `MetricDetailPage.tsx`, `MetricCreatePage.tsx`,
`MetricEditPage.tsx`, `MetricCalcConsole.tsx`) but is not wired into the
locale-prefixed routes. **Two ways to fix this, neither done here:**

- register the routes (`metrics`, `metrics/create`, `metrics/:id`,
  `metrics/:id/edit`, `metrics/:id/calc`) in `AppRoutes.tsx`
- or remove the `metrics` menu item until the routes exist

This is the same class of issue as the duplicate `MainNavigation` export
above: **two sources of truth (the menu config and the route table) can drift
because nothing reconciles them.** A small linter or test that asserts
"every `to:` field in `Navigation.tsx` resolves to a registered route" would
catch both at PR time.

### 4.4 Surface area that no CI job, test, or queue item owns

167 routes, of which a large fraction are hand-built domain pages
(`FixedIncomeDashboard`, `CryptoPortfolioCenter`, `RebalancingWizard`,
`ScenarioAnalysisPro`, `FactorAnalysisPage`, `AIPortfolioRebalancer`,
`AdvisorDashboard`, `Feed`, `BundleExplorer`, `ManagementPage`,
`CalculationsLibraryPage`, etc.) that do not appear in the test suite under
any of the `__tests__` directories. Most routes under `view/*` are served
by `PageRuntimeRenderer` and inherit the Page Studio governance, but the
hand-built pages do not.

This is the same question `#379` answered for Go modules — *who verifies
this?* — applied to the frontend. The map above is the input; a follow-up
audit ("which routes have test coverage, which are reachable from the menu,
which are one refactor from dead?") is the output. Filed as a separate
issue. Not urgent, but the drift risk on the dedicated domain pages
(`fabric/`, `simulation/`, `crypto/`, `wealth/`, `intelligence/`) is the
same drift risk that bit `rebalancing/worker`: a hand-built surface that
becomes load-bearing without anyone noticing.

### 4.5 Notes that are *not* desyncs (verified, in case of doubt)

- All routes under `admin/entitlements/*` are present in the table.
- The legacy `<Navigate>` redirects all resolve to live routes as of
  origin/main.
- `STUDIO_ROUTES` are the only place where a Page-Studio-served path is
  registered; nothing else writes to the route table.
- The `LocaleShell` route table is intentionally small: it is the
  pre-locale gate. The locale prefix is the boundary at which the
  rest of the routes live.

---

## 5. How to use this document

- To find a route → use §1 (top-level) and §2 (locale-prefixed) for the
  URL; the URL is the source of truth, the menu item is decoration.
- To find a menu item → use §3, then cross-check against §2 to confirm
  the link resolves.
- To add a new route → register it in `AppRoutes.tsx` (locale-prefixed)
  or `LocaleShell` (top-level). If it must also be reachable from the
  global sidebar, add a menu entry in `Navigation.tsx`. Both changes
  should land in the same commit; a linter that asserts
  "every menu `to:` resolves" is the cheapest enforcement.
- To deprecate a route → add a `<Navigate>` redirect in §2.22 first,
  then remove the route and the menu item in a later commit once the
  redirect is verified to be unused.
- To deprecate a menu item → if the route is going away, §2.22 first
  (the route stays, just redirects); otherwise remove both atomically.
- To find a route by component name → grep `frontend/src/AppRoutes.tsx`
  for the component; the table above was extracted from that file and
  is regenerated on every diff that touches it.

---

## 6. What this document is *not*

- Not a TypeScript API surface. The route table contains only the
  `path` and the `element` component name; it does not enumerate
  route props, query parameters, or hook usage. For those, read the
  component file.
- Not a permission map. The `<ProtectedRoute>` wrapper is uniform on
  every locale-prefixed route; capability gating happens inside the
  component, not at the route. Capability keys are defined in
  `frontend/src/api/capabilities.ts` and resolved by
  `useCapabilities()`. The route table does not show which
  capabilities a given component requires.
- Not a runtime trace. The page load time, lazy boundaries, and
  bundle splits are visible only at runtime; this document is
  a static map.
- Not a permission to add or remove routes. Changes to either
  `Navigation.tsx` or `AppRoutes.tsx` go through PR review; this
  map is a cold-reader reference, not a registry of approved
  state. Regenerate on every PR that touches either file.

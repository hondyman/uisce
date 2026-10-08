import type { CorePageDefinition } from '../../../../types/pageStudio';
import { masteringConsoleBlueprint } from './masteringConsole';
import { stagingBindingsBlueprint } from './stagingBindings';
import { dataPipelinesBlueprint, dataPipelineEditorBlueprint } from './dataPipelines';
import { matchRulesBlueprint, sourceHierarchyBlueprint, vendorRegistryBlueprint } from './mdmConfig';
import { schedulesBlueprint } from './schedules';
import { sourceScoringBlueprint } from './sourceScoring';
import { validationRulesBlueprint } from './validationRules';
import { lakehouseStreamingBlueprint } from './lakehouseStreaming';
import { systemLakehouseBlueprint } from './systemLakehouse';
import { cubesCatalogBlueprint } from './cubesCatalog';
import { cubeDesignerBlueprint } from './cubeDesigner';
import { lakehouseStatusBlueprint } from './lakehouseStatus';
import { abbreviationsBlueprint, catalogEdgeTypesBlueprint, catalogNodeTypesBlueprint } from './catalogTypes';
import {
  delegationsBlueprint,
  fieldPermissionsBlueprint,
  ipWhitelistBlueprint,
  messageCatalogBlueprint,
  rolesBlueprint,
  seedingBlueprint,
  teamsBlueprint,
  userRolesBlueprint,
  usersBlueprint,
  userTenantsBlueprint,
} from './platformPages';
import {
  aiSuggestionsBlueprint,
  apiInventoryBlueprint,
  businessTermsBlueprint,
  customFieldsBlueprint,
  domainsBlueprint,
  glossaryBlueprint,
  schemaExplorerBlueprint,
  semanticMapperBlueprint,
} from './catalogPages';
import {
  approvalInboxBlueprint,
  approvalWorkflowsBlueprint,
  bpConsoleBlueprint,
  bpInstancesBlueprint,
  bpQueuesBlueprint,
  businessRulesBlueprint,
  calculatedFieldsBlueprint,
  flowBuilderBlueprint,
  governanceChangesetsBlueprint,
  notificationCenterBlueprint,
  notificationPreferencesBlueprint,
  notificationTemplatesBlueprint,
  processCatalogBlueprint,
  slaDashboardBlueprint,
  validationBlueprint,
  validationRulesBlueprint as workflowValidationRulesBlueprint,
  workflowDesignerBlueprint,
  workflowStudioBlueprint,
} from './workflowPages';
import {
  advisorDashboardBlueprint,
  asoCenterBlueprint,
  dataQualityBlueprint,
  globalIntelligenceBlueprint,
  indexAdvisorBlueprint,
  intelligenceDashboardBlueprint,
  nlqBlueprint,
  observabilityDashboardBlueprint,
  portfolioMasterBlueprint,
  scenarioAnalysisBlueprint,
  sloDashboardBlueprint,
  storageTieringBlueprint,
} from './analyticsPages';
import {
  bundleExplorerBlueprint,
  businessObjectsBlueprint,
  calculationsLibraryBlueprint,
  fabricDashboardBlueprint,
  fabricSettingsBlueprint,
  preaggregationsBlueprint,
  queryLibraryBlueprint,
  reportBuilderBlueprint,
  reportLibraryBlueprint,
  semanticModelsBlueprint,
  viewsCatalogBlueprint,
} from './fabricPages';
import {
  accessExplanationBlueprint,
  accessRulesBlueprint,
  auditLogBlueprint,
  entitlementsBlueprint,
  jitRequestsBlueprint,
  llmConfigBlueprint,
  secretsConfigBlueprint,
  temporalOpsBlueprint,
  tenantsManagementBlueprint,
} from './securityPages';

/**
 * Pages the studio can start from: complete, working pages built entirely
 * from the page model - editable like any other page once created.
 */
export interface PageBlueprint {
  id: string;
  name: string;
  description: string;
  build: () => Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'>;
}

export const PAGE_BLUEPRINTS: PageBlueprint[] = [
  {
    id: 'mastering-console',
    name: 'Mastering console',
    description: 'Golden records, prices, runs, exceptions, duplicate review and steward overrides for every mastered entity.',
    build: masteringConsoleBlueprint,
  },
  {
    id: 'staging-bindings',
    name: 'Staging bindings',
    description: 'Map vendor staging tables to business object fields for mastering; changes need a second administrator.',
    build: stagingBindingsBlueprint,
  },
  {
    id: 'data-pipelines',
    name: 'Data pipelines',
    description: 'The list of data pipelines that load files and business objects into staging for mastering.',
    build: dataPipelinesBlueprint,
  },
  {
    id: 'data-pipeline-editor',
    name: 'Data pipeline editor',
    description: 'The visual pipeline editor (canvas, preview, assistant, runs), served at /data/pipelines/:id.',
    build: dataPipelineEditorBlueprint,
  },
  {
    id: 'mdm-source-hierarchy',
    name: 'Source hierarchy',
    description: 'Per mastered entity: which source wins for each field group or price type. Maker-checker.',
    build: sourceHierarchyBlueprint,
  },
  {
    id: 'mdm-match-rules',
    name: 'Match rules',
    description: 'Per record entity: exact and fuzzy match keys with auto-match and review thresholds. Maker-checker.',
    build: matchRulesBlueprint,
  },
  {
    id: 'mdm-vendors',
    name: 'Vendor registry',
    description: 'The one list of data sources mastering ranks and matches on. Maker-checker.',
    build: vendorRegistryBlueprint,
  },
  {
    id: 'schedules',
    name: 'Schedules',
    description: 'The platform scheduler: every schedule and its run history, with the schedule editor.',
    build: schedulesBlueprint,
  },
  {
    id: 'mdm-source-scoring',
    name: 'Source scoring & displacement',
    description: 'Vendor quality scoring, substitution rates, override endorsements, value-for-money efficient frontier, and displacement readiness simulation.',
    build: sourceScoringBlueprint,
  },
  {
    id: 'validation-rules',
    name: 'Validation Rules & Rule Studio',
    description: 'Centralized single-store validation catalog, portable AST bundles, and live evaluation engine.',
    build: validationRulesBlueprint,
  },
  {
    id: 'lakehouse-streaming',
    name: 'Lakehouse & CDC Stream Ingestion',
    description: 'Apache Iceberg REST catalog management, Debezium CDC ingestion pipelines, Layer 2 tenant assertion, and streaming gatekeeper inspection.',
    build: lakehouseStreamingBlueprint,
  },
  {
    id: 'system-lakehouse',
    name: 'Tenant lakehouse',
    description: "System: each tenant's one Iceberg warehouse and its per-tenant audit retention, with provisioning and the audit trail.",
    build: systemLakehouseBlueprint,
  },
  {
    id: 'lakehouse-status',
    name: 'Lakehouse status',
    description: 'Read-only panel: cluster health, resource groups, per-tenant wiring, and cross-check warnings.',
    build: lakehouseStatusBlueprint,
  },
  {
    id: 'cubes-catalog',
    name: 'Cubes',
    description: 'Aggregation contracts: dimensions, governed metrics, grains, and materialization. Built in Page Studio.',
    build: cubesCatalogBlueprint,
  },
  {
    id: 'cube-designer',
    name: 'Cube designer',
    description: 'Composed cube editor at /build/cubes/new and /build/cubes/:id (overview, axes, federation, materialization).',
    build: cubeDesignerBlueprint,
  },
  {
    id: 'catalog-node-types',
    name: 'Node Types',
    description: 'Metadata structures and node types in the catalog graph.',
    build: catalogNodeTypesBlueprint,
  },
  {
    id: 'catalog-edge-types',
    name: 'Edge Types',
    description: 'Relationship types and connections in the catalog graph.',
    build: catalogEdgeTypesBlueprint,
  },
  {
    id: 'core-abbreviations',
    name: 'Abbreviations',
    description: 'Standard business abbreviations and their full semantic expansions.',
    build: abbreviationsBlueprint,
  },
  {
    id: 'platform-teams',
    name: 'Teams',
    description: 'Manage organizational teams, membership, and resource assignments.',
    build: teamsBlueprint,
  },
  {
    id: 'platform-users',
    name: 'Users',
    description: 'User accounts, profile management, and credentials.',
    build: usersBlueprint,
  },
  {
    id: 'platform-user-roles',
    name: 'User Roles',
    description: 'Assign and manage RBAC role bindings for users.',
    build: userRolesBlueprint,
  },
  {
    id: 'platform-user-tenants',
    name: 'User Tenants',
    description: 'Multi-tenant memberships and tenant access assignment.',
    build: userTenantsBlueprint,
  },
  {
    id: 'platform-delegations',
    name: 'Delegations',
    description: 'Time-bound and approval-scoped user access delegations.',
    build: delegationsBlueprint,
  },
  {
    id: 'platform-field-permissions',
    name: 'Field Permissions',
    description: 'Fine-grained attribute and column-level permission matrix.',
    build: fieldPermissionsBlueprint,
  },
  {
    id: 'platform-roles',
    name: 'Roles & Permissions',
    description: 'Define RBAC roles, permission sets, and capability grants.',
    build: rolesBlueprint,
  },
  {
    id: 'platform-ip-whitelist',
    name: 'IP Whitelist',
    description: 'Configure network IP allowlists and CIDR ranges per tenant.',
    build: ipWhitelistBlueprint,
  },
  {
    id: 'platform-message-catalog',
    name: 'Message Catalog',
    description: 'Unified error code, localization, and message catalog editor.',
    build: messageCatalogBlueprint,
  },
  {
    id: 'platform-seeding',
    name: 'System Seeding',
    description: 'Seed catalog metadata, demo business objects, and rule fixtures.',
    build: seedingBlueprint,
  },
  {
    id: 'catalog-api-inventory',
    name: 'API Inventory',
    description: 'Catalog and inspect endpoints, payloads, schema contracts, and authentication.',
    build: apiInventoryBlueprint,
  },
  {
    id: 'catalog-glossary',
    name: 'Glossary & Semantic Terms',
    description: 'Explore business glossary, semantic terms, graph relationships, and classifications.',
    build: glossaryBlueprint,
  },
  {
    id: 'catalog-business-terms',
    name: 'Business Terms',
    description: 'Governed business taxonomy and terminology explorer.',
    build: businessTermsBlueprint,
  },
  {
    id: 'catalog-custom-fields',
    name: 'Custom Fields',
    description: 'Manage dynamic custom attributes and entity extensions.',
    build: customFieldsBlueprint,
  },
  {
    id: 'core-domains',
    name: 'Data Domains',
    description: 'Business data domains, ownership boundaries, and semantic scope.',
    build: domainsBlueprint,
  },
  {
    id: 'catalog-schema-explorer',
    name: 'Schema Explorer',
    description: 'Deep introspection of underlying database tables, views, and columns.',
    build: schemaExplorerBlueprint,
  },
  {
    id: 'core-semantic-mapper',
    name: 'Semantic Mapper',
    description: 'Map physical columns and upstream assets to canonical business concepts.',
    build: semanticMapperBlueprint,
  },
  {
    id: 'catalog-ai-suggestions',
    name: 'AI Term Suggestions',
    description: 'AI-assisted terminology recommendations, match confidence, and review workflow.',
    build: aiSuggestionsBlueprint,
  },
  {
    id: 'core-process-catalog',
    name: 'Process Catalog',
    description: 'Catalog of business processes, choreography, and execution history.',
    build: processCatalogBlueprint,
  },
  {
    id: 'core-approval-inbox',
    name: 'Approval Inbox',
    description: 'Unified queue for pending business and operational approvals.',
    build: approvalInboxBlueprint,
  },
  {
    id: 'core-approval-workflows',
    name: 'Approval Workflows',
    description: 'Monitor multi-stage approval chains, escalation paths, and turnaround metrics.',
    build: approvalWorkflowsBlueprint,
  },
  {
    id: 'client-workflow-studio',
    name: 'Process Designer',
    description: 'Visual process modeling, workflow state machine authoring, and orchestration.',
    build: workflowStudioBlueprint,
  },
  {
    id: 'core-workflow-designer',
    name: 'Workflow Designer',
    description: 'Step-by-step workflow canvas and node choreography.',
    build: workflowDesignerBlueprint,
  },
  {
    id: 'client-rules-editor',
    name: 'Business Rules',
    description: 'Rule authoring, trigger matrices, and condition evaluation.',
    build: businessRulesBlueprint,
  },
  {
    id: 'core-notifications',
    name: 'Notification Center',
    description: 'System alerts, real-time event feeds, and delivery logs.',
    build: notificationCenterBlueprint,
  },
  {
    id: 'core-notification-templates',
    name: 'Notification Templates',
    description: 'Configurable email, SMS, and in-app message templates.',
    build: notificationTemplatesBlueprint,
  },
  {
    id: 'core-notification-preferences',
    name: 'Notification Preferences',
    description: 'User and tenant alert subscriptions, channels, and thresholds.',
    build: notificationPreferencesBlueprint,
  },
  {
    id: 'core-sla-dashboard',
    name: 'SLA Dashboard',
    description: 'Service level agreements, breach forecasting, and execution velocity.',
    build: slaDashboardBlueprint,
  },
  {
    id: 'core-flow-builder',
    name: 'Flow Builder',
    description: 'Dataflow graph visual designer and execution engine.',
    build: flowBuilderBlueprint,
  },
  {
    id: 'core-validation-rules',
    name: 'Validation Rules',
    description: 'Data validation rules, constraint expressions, and error conditions.',
    build: workflowValidationRulesBlueprint,
  },
  {
    id: 'core-calculated-fields',
    name: 'Calculated Fields',
    description: 'Expression engine for computed attributes and formula fields.',
    build: calculatedFieldsBlueprint,
  },
  {
    id: 'core-validation',
    name: 'Run Validations',
    description: 'On-demand portfolio and investment validation engine run.',
    build: validationBlueprint,
  },
  {
    id: 'bp-console',
    name: 'BP Console',
    description: 'Execution monitoring, live process instances, and work queues.',
    build: bpConsoleBlueprint,
  },
  {
    id: 'bp-console-instances',
    name: 'Instance Explorer',
    description: 'Explore live and archived BP execution instances.',
    build: bpInstancesBlueprint,
  },
  {
    id: 'bp-console-queues',
    name: 'Work Queues',
    description: 'Monitor task queues and workload distribution across queues.',
    build: bpQueuesBlueprint,
  },
  {
    id: 'governance-changesets',
    name: 'ChangeSets',
    description: 'Audit and approve schema, metadata, and configuration changesets.',
    build: governanceChangesetsBlueprint,
  },
  // Analytics & Intelligence
  {
    id: 'intelligence-dashboard',
    name: 'Intelligence Dashboard',
    description: 'Autonomous optimization insights, workload signals, and recommendations.',
    build: intelligenceDashboardBlueprint,
  },
  {
    id: 'intelligence-index-advisor',
    name: 'Index Advisor',
    description: 'AI-driven index synthesis, workload pattern analysis, and query tuning.',
    build: indexAdvisorBlueprint,
  },
  {
    id: 'intelligence-storage',
    name: 'Storage Tiering',
    description: 'Storage efficiency, cold tier archival, and cost optimization telemetry.',
    build: storageTieringBlueprint,
  },
  {
    id: 'intelligence-data-quality',
    name: 'Data Quality',
    description: 'Anomaly detection, freshness monitoring, and schema drift metrics.',
    build: dataQualityBlueprint,
  },
  {
    id: 'optimization-aso',
    name: 'ASO Center',
    description: 'Autonomous Storage & Query Optimization control center.',
    build: asoCenterBlueprint,
  },
  {
    id: 'observability-dashboard',
    name: 'Metrics Dashboard',
    description: 'Real-time telemetry, latency percentiles, and database metrics.',
    build: observabilityDashboardBlueprint,
  },
  {
    id: 'observability-slos',
    name: 'SLO Dashboard',
    description: 'Service level objectives, error budget burn rates, and alerts.',
    build: sloDashboardBlueprint,
  },
  {
    id: 'nlq-page',
    name: 'Natural Language Query',
    description: 'Natural language query exploration over semantic models.',
    build: nlqBlueprint,
  },
  {
    id: 'global-intelligence',
    name: 'Global Intelligence',
    description: 'Unified cross-domain enterprise search and intelligence assistant.',
    build: globalIntelligenceBlueprint,
  },
  {
    id: 'analytics-scenario-analysis',
    name: 'Scenario Analysis',
    description: 'Interactive stress testing, macroeconomic shocks, and portfolio what-if modeling.',
    build: scenarioAnalysisBlueprint,
  },
  {
    id: 'analytics-portfolio-master',
    name: 'Portfolio Master',
    description: 'Comprehensive multi-asset portfolio positioning, exposure, and attribution.',
    build: portfolioMasterBlueprint,
  },
  {
    id: 'analytics-advisor-dashboard',
    name: 'Advisor Dashboard',
    description: 'Client wealth dashboard, household views, and personalized action recommendations.',
    build: advisorDashboardBlueprint,
  },
  // Fabric & Reporting
  {
    id: 'fabric-calculations',
    name: 'Calculations Library',
    description: 'Central registry of metrics, business formulas, and reusable calculations.',
    build: calculationsLibraryBlueprint,
  },
  {
    id: 'fabric-preaggregations',
    name: 'Preaggregations',
    description: 'Pre-computed rollup cubes, partition refreshes, and acceleration telemetry.',
    build: preaggregationsBlueprint,
  },
  {
    id: 'fabric-dashboard',
    name: 'Fabric Dashboard',
    description: 'Unified semantic fabric overview, node topology, and performance monitors.',
    build: fabricDashboardBlueprint,
  },
  {
    id: 'fabric-settings',
    name: 'Fabric Settings',
    description: 'System configurations, caching policies, and query engine routing rules.',
    build: fabricSettingsBlueprint,
  },
  {
    id: 'reports-library',
    name: 'Report Library',
    description: 'Enterprise catalog of published executive reports, templates, and analytics.',
    build: reportLibraryBlueprint,
  },
  {
    id: 'reports-builder',
    name: 'Report Builder',
    description: 'Interactive visual designer for building reports and parameterized views.',
    build: reportBuilderBlueprint,
  },
  {
    id: 'reports-queries',
    name: 'Query Builder',
    description: 'Visual query builder and saved query management repository.',
    build: queryLibraryBlueprint,
  },
  {
    id: 'reports-models',
    name: 'Semantic Models',
    description: 'Manage analytical semantic models, dimensions, and measures.',
    build: semanticModelsBlueprint,
  },
  {
    id: 'bundle-explorer',
    name: 'Bundle Explorer',
    description: 'Inspect micro-bundles, metadata packages, and cross-tenant exports.',
    build: bundleExplorerBlueprint,
  },
  {
    id: 'views',
    name: 'Views Catalog',
    description: 'Catalog of materialized, logical, and dynamic semantic views.',
    build: viewsCatalogBlueprint,
  },
  {
    id: 'business-objects',
    name: 'Business Objects',
    description: 'Enterprise business entities, driving tables, and bitemporal mappings.',
    build: businessObjectsBlueprint,
  },
  // Security & Admin
  {
    id: 'admin-llm',
    name: 'LLM Configuration',
    description: 'Manage LLM models, API keys, temperature settings, and tenant tokens.',
    build: llmConfigBlueprint,
  },
  {
    id: 'admin-temporal-ops',
    name: 'Temporal Operations',
    description: 'Temporal workflow orchestration, task queue telemetry, and failure recoveries.',
    build: temporalOpsBlueprint,
  },
  {
    id: 'admin-entitlements',
    name: 'Entitlement Management',
    description: 'Fine-grained attribute-based access control, profile permissions, and matrices.',
    build: entitlementsBlueprint,
  },
  {
    id: 'admin-audit',
    name: 'Audit Log',
    description: 'Tamper-evident audit trail for system events, authorization decisions, and changes.',
    build: auditLogBlueprint,
  },
  {
    id: 'tenants-management',
    name: 'Manage Resources',
    description: 'Multi-tenant provisioning, database quotas, and isolation policies.',
    build: tenantsManagementBlueprint,
  },
  {
    id: 'security-access-rules',
    name: 'Access Rules',
    description: 'Security policies, zero-trust constraints, and dynamic authorization rules.',
    build: accessRulesBlueprint,
  },
  {
    id: 'access-explanation',
    name: 'Access Explanation',
    description: 'Inspect effective user entitlements, RBAC/ABAC rationale, and grant lineage.',
    build: accessExplanationBlueprint,
  },
  {
    id: 'jit-requests',
    name: 'JIT Elevation Requests',
    description: 'Just-in-Time privilege escalation requests, approvals, and expiration timers.',
    build: jitRequestsBlueprint,
  },
  {
    id: 'secrets-config',
    name: 'Secrets',
    description: 'Secure credentials, API tokens, encryption key rotation, and KMS config.',
    build: secretsConfigBlueprint,
  },
];



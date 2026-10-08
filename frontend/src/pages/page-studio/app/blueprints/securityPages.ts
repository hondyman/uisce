import type { ComponentDefinition, CorePageDefinition, PageLayout } from '../../../../types/pageStudio';
import '../../../../features/security/studio';

type Spec = Record<string, { type: 'Row' | 'Column'; children: string[]; style?: Record<string, string> }>;
const layout = (root: string, spec: Spec): PageLayout => ({
  root,
  nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])),
});

function makeSingleDCBlueprint(opts: {
  name: string;
  slug: string;
  description: string;
  icon: string;
  dcId: string;
  title: string;
  subtitle: string;
}): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const components: Record<string, ComponentDefinition> = {
    header: {
      id: 'header',
      type: 'PageHeader',
      props: {
        icon: opts.icon,
        title: opts.title,
        subtitle: opts.subtitle,
      },
    },
    dc: {
      id: 'dc',
      type: 'DomainComponent',
      props: {
        component: opts.dcId,
        inputs: {},
      },
    },
  };

  const l = layout('root', {
    root: { type: 'Column', children: ['dc'], style: { gap: '16px' } },
  });

  const fb = layout('fb_root', {
    fb_root: { type: 'Row', children: ['header'], style: { alignItems: 'center', width: '100%' } },
  });

  return {
    name: opts.name,
    slug: opts.slug,
    description: opts.description,
    version: 1,
    isCore: true,
    status: 'published',
    layout: l,
    components,
    dataSources: [],
    tabs: [],
    presentationEvents: [],
    filterBar: fb,
    appModel: {
      chrome: 'none',
      surface: { padding: 3, maxWidth: 1600 },
      queries: [],
      variables: [],
    },
  };
}

export const llmConfigBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'LLM Configuration',
    slug: 'admin-llm',
    description: 'Manage LLM models, API keys, temperature settings, and tenant tokens.',
    icon: 'Cpu',
    dcId: 'security.LLMConfig',
    title: 'LLM Configuration',
    subtitle: 'Manage LLM models, API keys, temperature settings, and tenant tokens.',
  });

export const temporalOpsBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Temporal Operations',
    slug: 'admin-temporal-ops',
    description: 'Temporal workflow orchestration, task queue telemetry, and failure recoveries.',
    icon: 'Clock',
    dcId: 'security.TemporalOps',
    title: 'Temporal Operations',
    subtitle: 'Temporal workflow orchestration, task queue telemetry, and failure recoveries.',
  });

export const entitlementsBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Entitlement Management',
    slug: 'admin-entitlements',
    description: 'Fine-grained attribute-based access control, profile permissions, and matrices.',
    icon: 'Key',
    dcId: 'security.EntitlementMatrix',
    title: 'Entitlement Management',
    subtitle: 'Fine-grained attribute-based access control, profile permissions, and matrices.',
  });

export const auditLogBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Audit Log',
    slug: 'admin-audit',
    description: 'Tamper-evident audit trail for system events, authorization decisions, and changes.',
    icon: 'ShieldCheck',
    dcId: 'security.AuditLog',
    title: 'Audit Log Explorer',
    subtitle: 'Tamper-evident audit trail for system events, authorization decisions, and changes.',
  });

export const tenantsManagementBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Manage Resources',
    slug: 'tenants-management',
    description: 'Multi-tenant provisioning, database quotas, and isolation policies.',
    icon: 'Server',
    dcId: 'security.TenantsManagement',
    title: 'Manage Resources & Tenants',
    subtitle: 'Multi-tenant provisioning, database quotas, and isolation policies.',
  });

export const accessRulesBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Access Rules',
    slug: 'security-access-rules',
    description: 'Security policies, zero-trust constraints, and dynamic authorization rules.',
    icon: 'Lock',
    dcId: 'security.AccessRules',
    title: 'Access Rules',
    subtitle: 'Security policies, zero-trust constraints, and dynamic authorization rules.',
  });

export const accessExplanationBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Access Explanation',
    slug: 'access-explanation',
    description: 'Inspect effective user entitlements, RBAC/ABAC rationale, and grant lineage.',
    icon: 'HelpCircle',
    dcId: 'security.AccessExplanation',
    title: 'Access Explanation',
    subtitle: 'Inspect effective user entitlements, RBAC/ABAC rationale, and grant lineage.',
  });

export const jitRequestsBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'JIT Elevation Requests',
    slug: 'jit-requests',
    description: 'Just-in-Time privilege escalation requests, approvals, and expiration timers.',
    icon: 'Zap',
    dcId: 'security.JITRequests',
    title: 'JIT Elevation Requests',
    subtitle: 'Just-in-Time privilege escalation requests, approvals, and expiration timers.',
  });

export const secretsConfigBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Secrets',
    slug: 'secrets-config',
    description: 'Secure credentials, API tokens, encryption key rotation, and KMS config.',
    icon: 'KeyRound',
    dcId: 'security.Secrets',
    title: 'Secrets Management',
    subtitle: 'Secure credentials, API tokens, encryption key rotation, and KMS config.',
  });

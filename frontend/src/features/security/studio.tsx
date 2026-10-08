import React from 'react';
import { registerDomainComponents } from '../../studio-core/components/registry';

import LLMConfigPage from '../../pages/admin/LLMConfigPage';
import TemporalOpsPage from '../admin/pages/TemporalOpsPage';
import { EntitlementMatrix } from '../../admin-v2';
import AuditExplorer from '../../components/audit/AuditExplorer';
import TenantsManagementPage from '../fabric/pages/TenantsManagementPage';
import AccessRulesDashboard from './pages/AccessRulesDashboard';
import { AccessExplanationExample } from '../../AccessExplanationExample';
import { JITRequestPanelExample } from '../../JITRequestPanelExample';
import { SecretsConfigPage } from '../secrets';

const DOMAIN = 'security';

registerDomainComponents([
  {
    id: 'security.LLMConfig',
    domain: DOMAIN,
    label: 'LLM Configuration',
    description: 'Manage LLM models, API keys, temperature settings, and tenant tokens.',
    inputs: [],
    events: [],
    render: () => <LLMConfigPage />,
  },
  {
    id: 'security.TemporalOps',
    domain: DOMAIN,
    label: 'Temporal Operations',
    description: 'Temporal workflow orchestration, task queue telemetry, and failure recoveries.',
    inputs: [],
    events: [],
    render: () => <TemporalOpsPage />,
  },
  {
    id: 'security.EntitlementMatrix',
    domain: DOMAIN,
    label: 'Entitlement Management',
    description: 'Fine-grained attribute-based access control, profile permissions, and matrices.',
    inputs: [],
    events: [],
    render: () => <EntitlementMatrix profileKey="" isCustom={true} />,
  },
  {
    id: 'security.AuditLog',
    domain: DOMAIN,
    label: 'Audit Log Explorer',
    description: 'Tamper-evident audit trail for system events, authorization decisions, and changes.',
    inputs: [],
    events: [],
    render: () => <AuditExplorer tenantId="default" tenantName="Default" />,
  },
  {
    id: 'security.TenantsManagement',
    domain: DOMAIN,
    label: 'Manage Resources & Tenants',
    description: 'Multi-tenant provisioning, database quotas, and isolation policies.',
    inputs: [],
    events: [],
    render: () => <TenantsManagementPage />,
  },
  {
    id: 'security.AccessRules',
    domain: DOMAIN,
    label: 'Access Rules',
    description: 'Security policies, zero-trust constraints, and dynamic authorization rules.',
    inputs: [],
    events: [],
    render: () => <AccessRulesDashboard />,
  },
  {
    id: 'security.AccessExplanation',
    domain: DOMAIN,
    label: 'Access Explanation',
    description: 'Inspect effective user entitlements, RBAC/ABAC rationale, and grant lineage.',
    inputs: [],
    events: [],
    render: () => <AccessExplanationExample />,
  },
  {
    id: 'security.JITRequests',
    domain: DOMAIN,
    label: 'JIT Elevation Requests',
    description: 'Just-in-Time privilege escalation requests, approvals, and expiration timers.',
    inputs: [],
    events: [],
    render: () => <JITRequestPanelExample />,
  },
  {
    id: 'security.Secrets',
    domain: DOMAIN,
    label: 'Secrets Management',
    description: 'Secure credentials, API tokens, encryption key rotation, and KMS config.',
    inputs: [],
    events: [],
    render: () => <SecretsConfigPage tenantId="default" />,
  },
]);

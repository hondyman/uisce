import React from 'react';
import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import { registerDomainComponents } from '../../studio-core/components/registry';
import {
  RuleLibraryExplorer,
  RuleActivationMatrix,
  ComplianceDecisionBlotter,
  SurveillanceFindingsQueue,
  RegulatoryChangeQueue,
  ComplianceCalendar,
  ComplianceLimitDashboard,
} from '../../components/Compliance';

/**
 * Compliance Page Studio Operations:
 * Exposes compliance queries and mutations to the Page Studio runtime.
 */

const operations: OperationDef[] = [
  {
    id: 'compliance.listRules',
    domain: 'compliance',
    kind: 'query',
    label: 'List Core Compliance Rules',
    description: 'Lists all 94 Gold-Copy core library compliance rules.',
    params: [
      { name: 'jurisdiction', type: 'string', required: false },
      { name: 'ruleset_code', type: 'string', required: false },
      { name: 'domain', type: 'string', required: false },
    ],
    fields: [
      { name: 'rules' },
      { name: 'total', type: 'number' },
    ],
    run: async (params) => {
      const qs = new URLSearchParams();
      if (params.jurisdiction) qs.set('jurisdiction', String(params.jurisdiction));
      if (params.ruleset_code) qs.set('ruleset_code', String(params.ruleset_code));
      if (params.domain) qs.set('domain', String(params.domain));
      const res = await fetch(`/api/compliance/library/rules?${qs.toString()}`);
      if (!res.ok) throw new Error('Failed to fetch compliance rules');
      const rules = await res.json();
      return { rules, total: rules.length };
    },
  },
  {
    id: 'compliance.listEvaluations',
    domain: 'compliance',
    kind: 'query',
    label: 'List Pre-Trade Compliance Evaluations',
    description: 'Lists paginated pre-trade evaluation decisions and hashes from the blotter.',
    params: [
      { name: 'tenant_id', type: 'string', required: false },
      { name: 'action_taken', type: 'string', required: false },
      { name: 'page', type: 'number', required: false },
    ],
    fields: [
      { name: 'evaluations' },
      { name: 'total', type: 'number' },
    ],
    run: async (params) => {
      const qs = new URLSearchParams();
      if (params.tenant_id) qs.set('tenant_id', String(params.tenant_id));
      if (params.action_taken) qs.set('action_taken', String(params.action_taken));
      if (params.page) qs.set('page', String(params.page));
      const res = await fetch(`/api/compliance/evaluations?${qs.toString()}`);
      if (!res.ok) throw new Error('Failed to fetch compliance evaluations');
      return res.json();
    },
  },
  {
    id: 'compliance.listFindings',
    domain: 'compliance',
    kind: 'query',
    label: 'List Post-Trade Surveillance Findings',
    description: 'Lists surveillance findings, filing deadlines, and open exceptions.',
    params: [
      { name: 'tenant_id', type: 'string', required: false },
      { name: 'status', type: 'string', required: false },
    ],
    fields: [
      { name: 'findings' },
      { name: 'total', type: 'number' },
    ],
    run: async (params) => {
      const qs = new URLSearchParams();
      if (params.tenant_id) qs.set('tenant_id', String(params.tenant_id));
      if (params.status) qs.set('status', String(params.status));
      const res = await fetch(`/api/compliance/surveillance/findings?${qs.toString()}`);
      if (!res.ok) throw new Error('Failed to fetch surveillance findings');
      return res.json();
    },
  },
];

registerOperations(operations);

registerDomainComponents([
  {
    id: 'compliance.RuleLibraryExplorer',
    domain: 'compliance',
    label: 'Compliance Rule Library Explorer',
    description: 'Gold-Copy 94-rule catalog with AST predicates, parameters, citations, and test cases.',
    inputs: [],
    events: ['onSelectRule'],
    render: () => <RuleLibraryExplorer />,
  },
  {
    id: 'compliance.RuleActivationMatrix',
    domain: 'compliance',
    label: 'Tenant Rule Activation Matrix',
    description: 'Manage tenant rule activations, threshold overrides, and companion version repinning.',
    inputs: [
      { name: 'tenant_id', label: 'Tenant ID', type: 'string', required: false },
    ],
    events: ['onUpdateRule'],
    render: ({ inputs }) => (
      <RuleActivationMatrix tenantId={inputs.tenant_id ? String(inputs.tenant_id) : undefined} />
    ),
  },
  {
    id: 'compliance.DecisionBlotter',
    domain: 'compliance',
    label: 'Pre-Trade Decision Blotter',
    description: 'Real-time WebSocket stream of pre-trade evaluation decisions, explainability bundles, and RFC 8785 hashes.',
    inputs: [],
    events: ['onBroadcastOrder'],
    render: () => <ComplianceDecisionBlotter />,
  },
  {
    id: 'compliance.SurveillanceQueue',
    domain: 'compliance',
    label: 'Surveillance Findings Queue',
    description: 'Post-trade exception management, statutory filing deadlines (13D, 13G, SSR, DTR5), and triage workflow.',
    inputs: [],
    events: ['onStatusChange'],
    render: () => <SurveillanceFindingsQueue />,
  },
  {
    id: 'compliance.RegulatoryQueue',
    domain: 'compliance',
    label: 'Regulatory Change & Steward Queue',
    description: 'Regulatory change inbox, impact assessment, steward review, and release approvals.',
    inputs: [],
    events: ['onApprove', 'onReject'],
    render: () => <RegulatoryChangeQueue />,
  },
  {
    id: 'compliance.Calendar',
    domain: 'compliance',
    label: 'Compliance Deadlines Calendar',
    description: 'Statutory filing countdowns, dealing disclosure cutoffs, and regulatory obligations.',
    inputs: [],
    events: ['onSelectEvent'],
    render: () => <ComplianceCalendar />,
  },
  {
    id: 'compliance.LimitDashboard',
    domain: 'compliance',
    label: 'Limit Utilization & Headroom Dashboard',
    description: 'Real-time threshold proximity gauges, headroom capacity buffers, and sparkline trends.',
    inputs: [],
    events: ['onInspectLimit'],
    render: () => <ComplianceLimitDashboard />,
  },
]);

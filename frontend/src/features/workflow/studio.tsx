import React from 'react';
import { registerDomainComponents } from '../../studio-core/components/registry';

import { ProcessCatalogPage } from './pages/ProcessCatalogPage';
import { ApprovalInboxPage } from './pages/ApprovalInboxPage';
import { WorkflowDesignerPage } from './pages/WorkflowDesignerPage';
import { NotificationCenterPage } from './pages/NotificationCenterPage';
import { NotificationTemplateEditorPage } from './pages/NotificationTemplateEditorPage';
import { NotificationPreferencesPage } from './pages/NotificationPreferencesPage';
import { SLADashboardPage } from './pages/SLADashboardPage';
import ApprovalWorkflowDashboard from '../../pages/ApprovalWorkflowDashboard';
import WorkflowStudioPage from '../../pages/WorkflowStudioPage';
import BusinessRuleEditorPage from '../../pages/BusinessRuleEditorPage';
import AdvancedRuleBuilderPage from '../../pages/AdvancedRuleBuilderPage';
import CalculatedFieldBuilderPage from '../../pages/CalculatedFieldBuilderPage';
import { InvestmentValidationPage } from '../../pages/InvestmentValidationPage';
import UisceBuilder from '../uisce-builder/UisceBuilder';
import BPConsolePage from '../bp-console/pages/BPConsolePage';
import GovernanceConsolePage from '../../pages/governance/GovernanceConsolePage';

const DOMAIN = 'workflow';

registerDomainComponents([
  {
    id: 'workflow.ProcessCatalog',
    domain: DOMAIN,
    label: 'Process Catalog',
    description: 'Catalog of business processes, choreography, and execution history.',
    inputs: [],
    events: [],
    render: () => <ProcessCatalogPage />,
  },
  {
    id: 'workflow.ApprovalInbox',
    domain: DOMAIN,
    label: 'Approval Inbox',
    description: 'Unified queue for pending business and operational approvals.',
    inputs: [],
    events: [],
    render: () => <ApprovalInboxPage />,
  },
  {
    id: 'workflow.ApprovalWorkflows',
    domain: DOMAIN,
    label: 'Approval Workflows Dashboard',
    description: 'Monitor multi-stage approval chains, escalation paths, and turnaround metrics.',
    inputs: [],
    events: [],
    render: () => <ApprovalWorkflowDashboard />,
  },
  {
    id: 'workflow.WorkflowStudio',
    domain: DOMAIN,
    label: 'Process Designer (Studio)',
    description: 'Visual process modeling, workflow state machine authoring, and orchestration.',
    inputs: [],
    events: [],
    render: () => <WorkflowStudioPage />,
  },
  {
    id: 'workflow.WorkflowDesigner',
    domain: DOMAIN,
    label: 'Workflow Designer',
    description: 'Step-by-step workflow canvas and node choreography.',
    inputs: [],
    events: [],
    render: () => <WorkflowDesignerPage />,
  },
  {
    id: 'workflow.BusinessRules',
    domain: DOMAIN,
    label: 'Business Rules Editor',
    description: 'Rule authoring, trigger matrices, and condition evaluation.',
    inputs: [],
    events: [],
    render: () => <BusinessRuleEditorPage />,
  },
  {
    id: 'workflow.NotificationCenter',
    domain: DOMAIN,
    label: 'Notification Center',
    description: 'System alerts, real-time event feeds, and delivery logs.',
    inputs: [],
    events: [],
    render: () => <NotificationCenterPage />,
  },
  {
    id: 'workflow.NotificationTemplates',
    domain: DOMAIN,
    label: 'Notification Templates',
    description: 'Configurable email, SMS, and in-app message templates.',
    inputs: [],
    events: [],
    render: () => <NotificationTemplateEditorPage />,
  },
  {
    id: 'workflow.NotificationPreferences',
    domain: DOMAIN,
    label: 'Notification Preferences',
    description: 'User and tenant alert subscriptions, channels, and thresholds.',
    inputs: [],
    events: [],
    render: () => <NotificationPreferencesPage />,
  },
  {
    id: 'workflow.SLADashboard',
    domain: DOMAIN,
    label: 'SLA Dashboard',
    description: 'Service level agreements, breach forecasting, and execution velocity.',
    inputs: [],
    events: [],
    render: () => <SLADashboardPage />,
  },
  {
    id: 'workflow.FlowBuilder',
    domain: DOMAIN,
    label: 'Flow Builder (Uisce)',
    description: 'Dataflow graph visual designer and execution engine.',
    inputs: [],
    events: [],
    render: () => <UisceBuilder />,
  },
  {
    id: 'workflow.ValidationRules',
    domain: DOMAIN,
    label: 'Validation Rules',
    description: 'Data validation rules, constraint expressions, and error conditions.',
    inputs: [],
    events: [],
    render: () => <AdvancedRuleBuilderPage />,
  },
  {
    id: 'workflow.CalculatedFields',
    domain: DOMAIN,
    label: 'Calculated Fields',
    description: 'Expression engine for computed attributes and formula fields.',
    inputs: [],
    events: [],
    render: () => <CalculatedFieldBuilderPage />,
  },
  {
    id: 'workflow.InvestmentValidation',
    domain: DOMAIN,
    label: 'Run Validations',
    description: 'On-demand portfolio and investment validation engine run.',
    inputs: [],
    events: [],
    render: () => <InvestmentValidationPage />,
  },
  {
    id: 'workflow.BPConsole',
    domain: DOMAIN,
    label: 'Business Process Console',
    description: 'Execution monitoring, live process instances, and work queues.',
    inputs: [],
    events: [],
    render: () => <BPConsolePage />,
  },
  {
    id: 'workflow.GovernanceChangeSets',
    domain: DOMAIN,
    label: 'Governance ChangeSets',
    description: 'Audit and approve schema, metadata, and configuration changesets.',
    inputs: [],
    events: [],
    render: () => <GovernanceConsolePage />,
  },
]);

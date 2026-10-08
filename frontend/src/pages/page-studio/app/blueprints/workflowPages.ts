import type { ComponentDefinition, CorePageDefinition, PageLayout } from '../../../../types/pageStudio';
import '../../../../features/workflow/studio';

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

export const processCatalogBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Process Catalog',
    slug: 'core-process-catalog',
    description: 'Catalog of business processes, choreography, and execution history.',
    icon: 'Layers',
    dcId: 'workflow.ProcessCatalog',
    title: 'Process Catalog',
    subtitle: 'Catalog of business processes, choreography, and execution history.',
  });

export const approvalInboxBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Approval Inbox',
    slug: 'core-approval-inbox',
    description: 'Unified queue for pending business and operational approvals.',
    icon: 'CheckSquare',
    dcId: 'workflow.ApprovalInbox',
    title: 'Approval Inbox',
    subtitle: 'Unified queue for pending business and operational approvals.',
  });

export const approvalWorkflowsBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Approval Workflows',
    slug: 'core-approval-workflows',
    description: 'Monitor multi-stage approval chains, escalation paths, and turnaround metrics.',
    icon: 'GitPullRequest',
    dcId: 'workflow.ApprovalWorkflows',
    title: 'Approval Workflows',
    subtitle: 'Monitor multi-stage approval chains, escalation paths, and turnaround metrics.',
  });

export const workflowStudioBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Process Designer',
    slug: 'client-workflow-studio',
    description: 'Visual process modeling, workflow state machine authoring, and orchestration.',
    icon: 'Workflow',
    dcId: 'workflow.WorkflowStudio',
    title: 'Process Designer',
    subtitle: 'Visual process modeling, workflow state machine authoring, and orchestration.',
  });

export const workflowDesignerBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Workflow Designer',
    slug: 'core-workflow-designer',
    description: 'Step-by-step workflow canvas and node choreography.',
    icon: 'GitMerge',
    dcId: 'workflow.WorkflowDesigner',
    title: 'Workflow Designer',
    subtitle: 'Step-by-step workflow canvas and node choreography.',
  });

export const businessRulesBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Business Rules',
    slug: 'client-rules-editor',
    description: 'Rule authoring, trigger matrices, and condition evaluation.',
    icon: 'Sliders',
    dcId: 'workflow.BusinessRules',
    title: 'Business Rules',
    subtitle: 'Rule authoring, trigger matrices, and condition evaluation.',
  });

export const notificationCenterBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Notification Center',
    slug: 'core-notifications',
    description: 'System alerts, real-time event feeds, and delivery logs.',
    icon: 'Bell',
    dcId: 'workflow.NotificationCenter',
    title: 'Notification Center',
    subtitle: 'System alerts, real-time event feeds, and delivery logs.',
  });

export const notificationTemplatesBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Notification Templates',
    slug: 'core-notification-templates',
    description: 'Configurable email, SMS, and in-app message templates.',
    icon: 'Mail',
    dcId: 'workflow.NotificationTemplates',
    title: 'Notification Templates',
    subtitle: 'Configurable email, SMS, and in-app message templates.',
  });

export const notificationPreferencesBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Notification Preferences',
    slug: 'core-notification-preferences',
    description: 'User and tenant alert subscriptions, channels, and thresholds.',
    icon: 'Settings',
    dcId: 'workflow.NotificationPreferences',
    title: 'Notification Preferences',
    subtitle: 'User and tenant alert subscriptions, channels, and thresholds.',
  });

export const slaDashboardBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'SLA Dashboard',
    slug: 'core-sla-dashboard',
    description: 'Service level agreements, breach forecasting, and execution velocity.',
    icon: 'Clock',
    dcId: 'workflow.SLADashboard',
    title: 'SLA Dashboard',
    subtitle: 'Service level agreements, breach forecasting, and execution velocity.',
  });

export const flowBuilderBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Flow Builder',
    slug: 'core-flow-builder',
    description: 'Dataflow graph visual designer and execution engine.',
    icon: 'Share2',
    dcId: 'workflow.FlowBuilder',
    title: 'Flow Builder',
    subtitle: 'Dataflow graph visual designer and execution engine.',
  });

export const validationRulesBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Validation Rules',
    slug: 'core-validation-rules',
    description: 'Data validation rules, constraint expressions, and error conditions.',
    icon: 'CheckCircle',
    dcId: 'workflow.ValidationRules',
    title: 'Validation Rules',
    subtitle: 'Data validation rules, constraint expressions, and error conditions.',
  });

export const calculatedFieldsBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Calculated Fields',
    slug: 'core-calculated-fields',
    description: 'Expression engine for computed attributes and formula fields.',
    icon: 'Calculator',
    dcId: 'workflow.CalculatedFields',
    title: 'Calculated Fields',
    subtitle: 'Expression engine for computed attributes and formula fields.',
  });

export const validationBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Run Validations',
    slug: 'core-validation',
    description: 'On-demand portfolio and investment validation engine run.',
    icon: 'PlayCircle',
    dcId: 'workflow.InvestmentValidation',
    title: 'Run Validations',
    subtitle: 'On-demand portfolio and investment validation engine run.',
  });

export const bpConsoleBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'BP Console',
    slug: 'bp-console',
    description: 'Execution monitoring, live process instances, and work queues.',
    icon: 'Activity',
    dcId: 'workflow.BPConsole',
    title: 'BP Console',
    subtitle: 'Execution monitoring, live process instances, and work queues.',
  });

export const bpInstancesBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Instance Explorer',
    slug: 'bp-console-instances',
    description: 'Explore live and archived BP execution instances.',
    icon: 'List',
    dcId: 'workflow.BPConsole',
    title: 'Instance Explorer',
    subtitle: 'Explore live and archived BP execution instances.',
  });

export const bpQueuesBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Work Queues',
    slug: 'bp-console-queues',
    description: 'Monitor task queues and workload distribution across queues.',
    icon: 'Inbox',
    dcId: 'workflow.BPConsole',
    title: 'Work Queues',
    subtitle: 'Monitor task queues and workload distribution across queues.',
  });

export const governanceChangesetsBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'ChangeSets',
    slug: 'governance-changesets',
    description: 'Audit and approve schema, metadata, and configuration changesets.',
    icon: 'Shield',
    dcId: 'workflow.GovernanceChangeSets',
    title: 'Governance ChangeSets',
    subtitle: 'Audit and approve schema, metadata, and configuration changesets.',
  });

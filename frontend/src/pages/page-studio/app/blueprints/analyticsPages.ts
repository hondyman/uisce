import type { ComponentDefinition, CorePageDefinition, PageLayout } from '../../../../types/pageStudio';
import '../../../../features/analytics/studio';

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

export const intelligenceDashboardBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Intelligence Dashboard',
    slug: 'intelligence-dashboard',
    description: 'Autonomous optimization insights, workload signals, and recommendations.',
    icon: 'Brain',
    dcId: 'analytics.IntelligenceDashboard',
    title: 'Intelligence Dashboard',
    subtitle: 'Autonomous optimization insights, workload signals, and recommendations.',
  });

export const indexAdvisorBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Index Advisor',
    slug: 'intelligence-index-advisor',
    description: 'AI-driven index synthesis, workload pattern analysis, and query tuning.',
    icon: 'Sparkles',
    dcId: 'analytics.IndexAdvisor',
    title: 'Index Advisor',
    subtitle: 'AI-driven index synthesis, workload pattern analysis, and query tuning.',
  });

export const storageTieringBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Storage Tiering',
    slug: 'intelligence-storage',
    description: 'Storage efficiency, cold tier archival, and cost optimization telemetry.',
    icon: 'HardDrive',
    dcId: 'analytics.StorageTiering',
    title: 'Storage Tiering',
    subtitle: 'Storage efficiency, cold tier archival, and cost optimization telemetry.',
  });

export const dataQualityBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Data Quality',
    slug: 'intelligence-data-quality',
    description: 'Anomaly detection, freshness monitoring, and schema drift metrics.',
    icon: 'Activity',
    dcId: 'analytics.DataQuality',
    title: 'Data Quality Monitor',
    subtitle: 'Anomaly detection, freshness monitoring, and schema drift metrics.',
  });

export const asoCenterBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'ASO Center',
    slug: 'optimization-aso',
    description: 'Autonomous Storage & Query Optimization control center.',
    icon: 'Zap',
    dcId: 'analytics.OptimizationCenter',
    title: 'ASO Optimization Center',
    subtitle: 'Autonomous Storage & Query Optimization control center.',
  });

export const observabilityDashboardBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Metrics Dashboard',
    slug: 'observability-dashboard',
    description: 'Real-time telemetry, latency percentiles, and database metrics.',
    icon: 'BarChart2',
    dcId: 'analytics.Observability',
    title: 'Observability Dashboard',
    subtitle: 'Real-time telemetry, latency percentiles, and database metrics.',
  });

export const sloDashboardBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'SLO Dashboard',
    slug: 'observability-slos',
    description: 'Service level objectives, error budget burn rates, and alerts.',
    icon: 'Target',
    dcId: 'analytics.SLODashboard',
    title: 'SLO Dashboard',
    subtitle: 'Service level objectives, error budget burn rates, and alerts.',
  });

export const nlqBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Natural Language Query',
    slug: 'nlq-page',
    description: 'Natural language query exploration over semantic models.',
    icon: 'MessageSquare',
    dcId: 'analytics.NLQ',
    title: 'Natural Language Query (NLQ)',
    subtitle: 'Natural language query exploration over semantic models.',
  });

export const globalIntelligenceBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Global Intelligence',
    slug: 'global-intelligence',
    description: 'Unified cross-domain enterprise search and intelligence assistant.',
    icon: 'Globe',
    dcId: 'analytics.GlobalNLQ',
    title: 'Global Intelligence NLQ',
    subtitle: 'Unified cross-domain enterprise search and intelligence assistant.',
  });

export const scenarioAnalysisBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Scenario Analysis',
    slug: 'analytics-scenario-analysis',
    description: 'Interactive stress testing, macroeconomic shocks, and portfolio what-if modeling.',
    icon: 'TrendingUp',
    dcId: 'analytics.ScenarioAnalysis',
    title: 'Scenario Analysis Pro',
    subtitle: 'Interactive stress testing, macroeconomic shocks, and portfolio what-if modeling.',
  });

export const portfolioMasterBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Portfolio Master',
    slug: 'analytics-portfolio-master',
    description: 'Comprehensive multi-asset portfolio positioning, exposure, and attribution.',
    icon: 'Briefcase',
    dcId: 'analytics.PortfolioMaster',
    title: 'Portfolio Master',
    subtitle: 'Comprehensive multi-asset portfolio positioning, exposure, and attribution.',
  });

export const advisorDashboardBlueprint = () =>
  makeSingleDCBlueprint({
    name: 'Advisor Dashboard',
    slug: 'analytics-advisor-dashboard',
    description: 'Client wealth dashboard, household views, and personalized action recommendations.',
    icon: 'UserCheck',
    dcId: 'analytics.AdvisorDashboard',
    title: 'Advisor Dashboard',
    subtitle: 'Client wealth dashboard, household views, and personalized action recommendations.',
  });

import React from 'react';
import { registerDomainComponents } from '../../studio-core/components/registry';

import IntelligenceDashboard from '../../pages/intelligence/IntelligenceDashboard';
import IndexAdvisorPage from '../../pages/intelligence/IndexAdvisorPage';
import StorageTieringPage from '../../pages/intelligence/StorageTieringPage';
import DataQualityMonitorPage from '../../pages/intelligence/DataQualityMonitorPage';
import OptimizationCenter from '../../pages/OptimizationCenter';
import ObservabilityDashboard from '../../pages/ObservabilityDashboard';
import SLODashboard from '../../pages/SLODashboard';
import NLQPage from '../../pages/nlq/NLQPage';
import GlobalNLQueryPage from '../../pages/GlobalNLQueryPage';
import ScenarioAnalysisPro from '../../components/ScenarioAnalysisPro';
import PortfolioMasterDashboard from '../../components/PortfolioMasterDashboard';
import AdvisorDashboard from '../../pages/AdvisorDashboard';

const DOMAIN = 'analytics';

registerDomainComponents([
  {
    id: 'analytics.IntelligenceDashboard',
    domain: DOMAIN,
    label: 'Intelligence Dashboard',
    description: 'Autonomous optimization insights, workload signals, and recommendations.',
    inputs: [],
    events: [],
    render: () => <IntelligenceDashboard />,
  },
  {
    id: 'analytics.IndexAdvisor',
    domain: DOMAIN,
    label: 'Index Advisor',
    description: 'AI-driven index synthesis, workload pattern analysis, and query tuning.',
    inputs: [],
    events: [],
    render: () => <IndexAdvisorPage />,
  },
  {
    id: 'analytics.StorageTiering',
    domain: DOMAIN,
    label: 'Storage Tiering',
    description: 'Storage efficiency, cold tier archival, and cost optimization telemetry.',
    inputs: [],
    events: [],
    render: () => <StorageTieringPage />,
  },
  {
    id: 'analytics.DataQuality',
    domain: DOMAIN,
    label: 'Data Quality Monitor',
    description: 'Anomaly detection, freshness monitoring, and schema drift metrics.',
    inputs: [],
    events: [],
    render: () => <DataQualityMonitorPage />,
  },
  {
    id: 'analytics.OptimizationCenter',
    domain: DOMAIN,
    label: 'ASO Optimization Center',
    description: 'Autonomous Storage & Query Optimization control center.',
    inputs: [],
    events: [],
    render: () => <OptimizationCenter scope="global" />,
  },
  {
    id: 'analytics.Observability',
    domain: DOMAIN,
    label: 'Observability Dashboard',
    description: 'Real-time telemetry, latency percentiles, and database metrics.',
    inputs: [],
    events: [],
    render: () => <ObservabilityDashboard />,
  },
  {
    id: 'analytics.SLODashboard',
    domain: DOMAIN,
    label: 'SLO Dashboard',
    description: 'Service level objectives, error budget burn rates, and alerts.',
    inputs: [],
    events: [],
    render: () => <SLODashboard />,
  },
  {
    id: 'analytics.NLQ',
    domain: DOMAIN,
    label: 'Natural Language Query (NLQ)',
    description: 'Natural language query exploration over semantic models.',
    inputs: [],
    events: [],
    render: () => <NLQPage />,
  },
  {
    id: 'analytics.GlobalNLQ',
    domain: DOMAIN,
    label: 'Global Intelligence NLQ',
    description: 'Unified cross-domain enterprise search and intelligence assistant.',
    inputs: [],
    events: [],
    render: () => <GlobalNLQueryPage />,
  },
  {
    id: 'analytics.ScenarioAnalysis',
    domain: DOMAIN,
    label: 'Scenario Analysis Pro',
    description: 'Interactive stress testing, macroeconomic shocks, and portfolio what-if modeling.',
    inputs: [],
    events: [],
    render: () => <ScenarioAnalysisPro />,
  },
  {
    id: 'analytics.PortfolioMaster',
    domain: DOMAIN,
    label: 'Portfolio Master',
    description: 'Comprehensive multi-asset portfolio positioning, exposure, and attribution.',
    inputs: [],
    events: [],
    render: () => <PortfolioMasterDashboard />,
  },
  {
    id: 'analytics.AdvisorDashboard',
    domain: DOMAIN,
    label: 'Advisor Dashboard',
    description: 'Client wealth dashboard, household views, and personalized action recommendations.',
    inputs: [],
    events: [],
    render: () => <AdvisorDashboard />,
  },
]);

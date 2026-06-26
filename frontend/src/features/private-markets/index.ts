// Main Explorer Components
export { PrivateMarketsExplorer } from './PrivateMarketsExplorer';
export { LPPrivateMarketsDashboard } from './LPPrivateMarketsDashboard';
export { GPPrivateMarketsDashboard } from './GPPrivateMarketsDashboard';
export { TemplateReviewDashboard } from './TemplateReviewDashboard';

// Context and Hooks
export { ExplorerProvider, useExplorer } from './ExplorerContext';
export type { User, Bundle } from './ExplorerContext';

// Shared Components
export { FundSelector } from './components/FundSelector';
export { IRRCurveChart } from './components/IRRCurveChart';
export { JCurvePlot } from './components/JCurvePlot';
export { MultipleOverlayPanel } from './components/MultipleOverlayPanel';
export { BenchmarkComparison } from './components/BenchmarkComparison';
export { LiquidityPanel } from './components/LiquidityPanel';
export { DeploymentPacingChart } from './components/DeploymentPacingChart';

// Bundle Configurations
import lpBundleData from './bundles/lp_private_markets_bundle.json';
import gpBundleData from './bundles/gp_private_markets_bundle.json';
import fofBundleData from './bundles/fof_private_markets_bundle.json';

export const lpBundle = lpBundleData;
export const gpBundle = gpBundleData;
export const fofBundle = fofBundleData;

// Types
export interface Fund {
  id: string;
  name: string;
  vintage: number;
  manager: string;
  strategy: string;
  geography: string;
  status: 'active' | 'liquidated' | 'realizing';
}

export interface Template {
  id: string;
  name: string;
  domain: string;
  category: string;
  subcategory: string;
  version: string;
  status: 'draft' | 'reviewed' | 'golden';
  owner: string;
  description: string;
  tags: string[];
  governance: {
    status: string;
    steward_group: string;
    schema_hash: string;
    sla: {
      refresh_frequency: string;
      max_latency: string;
    };
  };
  created_at: string;
  updated_at: string;
  review_comments: Array<{
    id: string;
    author: string;
    comment: string;
    type: 'comment' | 'approval' | 'rejection' | 'change_request';
    created_at: string;
  }>;
}

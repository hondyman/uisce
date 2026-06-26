/// Dynamic import of the JSON bundle
import { devLog, devError } from '../utils/devLogger';

import { BundleLibrary } from '../types/bundles';

let bundleData: BundleLibrary | null = null;

const loadBundle = async (): Promise<void> => {
  if (bundleData) return; // Already loaded

  try {
    const response = await fetch('/complete_multi_dialect_unified_super_bundle.json');
    if (!response.ok) {
      throw new Error(`Failed to load bundle: ${response.status}`);
    }
  bundleData = await response.json();
  // Log bundle load in dev logger
  devLog('Bundle data loaded:', bundleData);
  } catch (error) {
  devError('Error loading bundle:', error);
    // Fallback to empty bundle
    bundleData = {
      bundle_id: 'fallback',
      description: 'Fallback bundle',
      version: '0.0.0',
      last_updated: '2025-09-13',
      engines: [],
      total_metrics: 0,
      domains: [],
      metrics: [],
      function_mapping: []
    };
  }
};

export interface Metric {
  node_id: string;
  domain: string;
  category: string;
  description: string;
  neutral_formula: string;
  dax_formula: string;
  sql_server: string;
  oracle: string;
  postgres: string;
  snowflake: string;
  iceberg: string;
  preaggregation: {
    enabled: boolean;
    grain: string;
    rollups: string[];
    snapshot: boolean;
    partition_keys?: string[];
    watermark_field?: string;
  };
  governance: string;
  audience: string[];
}

export interface FunctionMapping {
  class: string;
  dax: string;
  neutral: string;
  sql_server: string;
  oracle: string;
  postgres: string;
  snowflake: string;
  iceberg: string;
  notes: string;
}

export interface Bundle {
  bundle_id: string;
  description: string;
  version: string;
  last_updated: string;
  engines: string[];
  total_metrics: number;
  domains: string[];
  metrics: Metric[];
  function_mapping: FunctionMapping[];
}

// Avoid asserting bundleData before it's loaded; access via helpers below.

/**
 * Get all metrics in the bundle
 */
export async function getAllMetrics(): Promise<Metric[]> {
  await loadBundle();
  return bundleData?.metrics || [];
}

/**
 * Get metrics by domain
 */
export async function getMetricsByDomain(domain: string): Promise<Metric[]> {
  await loadBundle();
  return bundleData?.metrics.filter(metric => metric.domain === domain) || [];
}

/**
 * Get metrics by category
 */
export async function getMetricsByCategory(category: string): Promise<Metric[]> {
  await loadBundle();
  return bundleData?.metrics.filter(metric => metric.category === category) || [];
}

/**
 * Get a specific metric by node_id
 */
export async function getMetricById(nodeId: string): Promise<Metric | undefined> {
  await loadBundle();
  return bundleData?.metrics.find(metric => metric.node_id === nodeId);
}

/**
 * Get function mappings
 */
export async function getFunctionMappings(): Promise<FunctionMapping[]> {
  await loadBundle();
  return bundleData?.function_mapping || [];
}

/**
 * Get function mapping by DAX function
 */
export async function getFunctionMappingByDax(daxFunction: string): Promise<FunctionMapping | undefined> {
  await loadBundle();
  return bundleData?.function_mapping.find(mapping => mapping.dax === daxFunction);
}

/**
 * Get SQL translation for a metric in a specific engine
 */
export async function getSqlForMetric(metricId: string, engine: string): Promise<string | undefined> {
  const metric = await getMetricById(metricId);
  if (!metric) return undefined;

  switch (engine.toLowerCase()) {
    case 'sql_server':
      return metric.sql_server;
    case 'oracle':
      return metric.oracle;
    case 'postgres':
      return metric.postgres;
    case 'snowflake':
      return metric.snowflake;
    case 'iceberg':
      return metric.iceberg;
    default:
      return undefined;
  }
}

/**
 * Get all domains
 */
export async function getDomains(): Promise<string[]> {
  await loadBundle();
  return bundleData?.domains || [];
}

/**
 * Get bundle metadata
 */
export async function getBundleInfo() {
  await loadBundle();
  if (!bundleData) return null;
  return {
    bundle_id: bundleData.bundle_id,
    description: bundleData.description,
    version: bundleData.version,
    last_updated: bundleData.last_updated,
    engines: bundleData.engines,
    total_metrics: bundleData.total_metrics,
    domains: bundleData.domains
  };
}

export default bundleData;

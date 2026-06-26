import bundleData from './complete_multi_dialect_unified_super_bundle.json';

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

const bundle: Bundle = bundleData as Bundle;

/**
 * Get all metrics in the bundle
 */
export function getAllMetrics(): Metric[] {
  return bundle.metrics;
}

/**
 * Get metrics by domain
 */
export function getMetricsByDomain(domain: string): Metric[] {
  return bundle.metrics.filter(metric => metric.domain === domain);
}

/**
 * Get metrics by category
 */
export function getMetricsByCategory(category: string): Metric[] {
  return bundle.metrics.filter(metric => metric.category === category);
}

/**
 * Get a specific metric by node_id
 */
export function getMetricById(nodeId: string): Metric | undefined {
  return bundle.metrics.find(metric => metric.node_id === nodeId);
}

/**
 * Get function mappings
 */
export function getFunctionMappings(): FunctionMapping[] {
  return bundle.function_mapping;
}

/**
 * Get function mapping by DAX function
 */
export function getFunctionMappingByDax(daxFunction: string): FunctionMapping | undefined {
  return bundle.function_mapping.find(mapping => mapping.dax === daxFunction);
}

/**
 * Get SQL translation for a metric in a specific engine
 */
export function getSqlForMetric(metricId: string, engine: string): string | undefined {
  const metric = getMetricById(metricId);
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
export function getDomains(): string[] {
  return bundle.domains;
}

/**
 * Get bundle metadata
 */
export function getBundleInfo() {
  return {
    bundle_id: bundle.bundle_id,
    description: bundle.description,
    version: bundle.version,
    last_updated: bundle.last_updated,
    engines: bundle.engines,
    total_metrics: bundle.total_metrics,
    domains: bundle.domains
  };
}

export default bundle;

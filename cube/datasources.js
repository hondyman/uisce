/**
 * Cube.js Multi-Datasource Configuration
 * 
 * Architecture:
 * ┌─────────────────────────────────────────────────────────────────────┐
 * │                     CUBE DATA SOURCE ROUTING                        │
 * ├─────────────────────────────────────────────────────────────────────┤
 * │                                                                     │
 * │  HOT CUBES (data_source: starrocks)                                │
 * │  └─► StarRocks Native Tables (semantic_hot.*)                      │
 * │      • Real-time positions, current NAV                            │
 * │      • Refresh: every 10 min                                       │
 * │                                                                     │
 * │  COLD CUBES (data_source: starrocks_cold)                          │
 * │  └─► StarRocks External Tables on Parquet/Iceberg                  │
 * │      • Historical trades, archived positions                       │
 * │      • Refresh: daily                                              │
 * │      • Pre-aggs STILL stored in StarRocks native (fast!)           │
 * │                                                                     │
 * │  WHY StarRocks for both?                                           │
 * │  • Single query engine (simpler ops)                               │
 * │  • Can JOIN hot + cold in one query                                │
 * │  • External tables = zero data copy, reads Parquet directly        │
 * │  • Pre-aggs always in native tables = always fast                  │
 * │                                                                     │
 * └─────────────────────────────────────────────────────────────────────┘
 */

const MySQLDriver = require('@cubejs-backend/mysql-driver');

/**
 * Creates a StarRocks driver for a specific database
 * StarRocks uses MySQL protocol on port 9030
 */
function createStarRocksDriver(database, resourceGroup = null) {
  const driver = new MySQLDriver({
    host: process.env.STARROCKS_HOST || 'starrocks-fe',
    port: parseInt(process.env.STARROCKS_PORT || '9030'),
    user: process.env.STARROCKS_USER || 'root',
    password: process.env.STARROCKS_PASSWORD || '',
    database: database,
    // Connection pool settings for multi-tenant workloads
    connectionLimit: parseInt(process.env.STARROCKS_POOL_SIZE || '20'),
    queueLimit: 50,
    connectTimeout: 10000,
  });

  // If resource group specified, set it on each connection
  if (resourceGroup) {
    const originalQuery = driver.query.bind(driver);
    driver.query = async (query, values) => {
      // Set resource group for QoS before each query
      await originalQuery(`SET resource_group = '${resourceGroup}'`, []);
      return originalQuery(query, values);
    };
  }

  return driver;
}

/**
 * Datasource definitions for Cube.js
 * 
 * In your schema files, use:
 *   data_source: starrocks       # For hot/current data
 *   data_source: starrocks_cold  # For historical/archived data
 */
const datasources = {
  // =========================================================================
  // HOT TIER: StarRocks Native Tables
  // =========================================================================
  // Use for: Current positions, real-time valuations, recent transactions
  // Tables: semantic_hot.holdings, semantic_hot.portfolio_nav, etc.
  // =========================================================================
  starrocks: {
    type: 'mysql', // StarRocks uses MySQL protocol
    driver: () => createStarRocksDriver(
      process.env.STARROCKS_HOT_DB || 'semantic_hot',
      'analytics_normal' // Resource group for BI queries
    ),
  },

  // =========================================================================
  // COLD TIER: StarRocks External Tables on Parquet/Iceberg
  // =========================================================================
  // Use for: Historical trades, archived positions, compliance data
  // Tables: semantic_cold.holdings, semantic_cold.transactions, etc.
  // These are EXTERNAL tables pointing to S3/HDFS Parquet files
  // =========================================================================
  starrocks_cold: {
    type: 'mysql',
    driver: () => createStarRocksDriver(
      process.env.STARROCKS_COLD_DB || 'semantic_cold',
      'batch_low' // Lower priority for large historical scans
    ),
  },

  // =========================================================================
  // DEFAULT: Falls back to hot tier
  // =========================================================================
  default: {
    type: 'mysql',
    driver: () => createStarRocksDriver(
      process.env.STARROCKS_HOT_DB || 'semantic_hot',
      'analytics_normal'
    ),
  },
};

/**
 * Driver factory for Cube.js configuration
 * Routes queries to appropriate StarRocks database based on data_source
 */
function driverFactory({ dataSource }) {
  const ds = datasources[dataSource] || datasources.default;
  return ds.driver();
}

/**
 * Database type resolver
 * All datasources use MySQL protocol (StarRocks)
 */
function dbType({ dataSource }) {
  return 'mysql';
}

module.exports = {
  datasources,
  driverFactory,
  dbType,
  createStarRocksDriver,
};

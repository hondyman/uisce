/**
 * Cube.js Multi-DataSource Configuration
 * 
 * Architecture:
 *   - StarRocks (Hot): Real-time queries, pre-aggregations
 *   - StarRocks on Parquet (Cold): Historical analytics
 *   - PostgreSQL (Metadata): Tenant catalog (accessed via API, not Cube)
 *   - PostgreSQL per Tenant: Isolated financial data
 * 
 * NO REDIS REQUIRED - Uses Cube Store for queue/cache coordination
 */

// Data source definitions
module.exports = {
  // StarRocks Hot Store (primary analytics)
  starrocks: {
    type: 'mysql', // StarRocks uses MySQL protocol
    host: process.env.STARROCKS_HOST || 'starrocks-fe',
    port: parseInt(process.env.STARROCKS_PORT || '9030'),
    user: process.env.STARROCKS_USER || 'root',
    password: process.env.STARROCKS_PASSWORD || '',
    database: process.env.STARROCKS_DATABASE || 'cube_hot',
  },

  // StarRocks Pre-aggregation Store
  starrocks_preagg: {
    type: 'mysql',
    host: process.env.STARROCKS_HOST || 'starrocks-fe',
    port: parseInt(process.env.STARROCKS_PORT || '9030'),
    user: process.env.STARROCKS_USER || 'root',
    password: process.env.STARROCKS_PASSWORD || '',
    database: process.env.STARROCKS_PREAGG_DATABASE || 'cube_preagg',
  },

  // StarRocks Cold Store (Parquet external tables)
  starrocks_cold: {
    type: 'mysql',
    host: process.env.STARROCKS_HOST || 'starrocks-fe',
    port: parseInt(process.env.STARROCKS_PORT || '9030'),
    user: process.env.STARROCKS_USER || 'root',
    password: process.env.STARROCKS_PASSWORD || '',
    database: process.env.STARROCKS_COLD_DATABASE || 'cube_cold',
  },

  // Dynamic tenant database (resolved at query time)
  // This is a placeholder - actual connection is resolved via contextToAppId
  tenant_financial: {
    type: 'postgres',
    host: process.env.TENANT_DB_HOST || 'localhost',
    port: parseInt(process.env.TENANT_DB_PORT || '5432'),
    user: process.env.TENANT_DB_USER || 'tenant',
    password: process.env.TENANT_DB_PASSWORD || '',
    database: process.env.TENANT_DB_NAME || 'tenant',
  },
};

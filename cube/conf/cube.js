/**
 * Cube.js Configuration - NO REDIS
 * 
 * Uses Cube Store for:
 *   - Query queue management
 *   - Pre-aggregation coordination
 *   - Result caching
 * 
 * Pre-aggregations stored in StarRocks (not Redis)
 */

const fetch = require('node-fetch');

module.exports = {
  // ============================================================================
  // SCHEDULING (Pre-aggregation refresh)
  // ============================================================================
  
  scheduledRefreshTimer: 60, // Check every 60 seconds
  
  scheduledRefreshContexts: async () => {
    // In production, fetch active tenants from metadata API
    // For now, return static list
    return [
      { securityContext: { tenant_id: 'demo' } },
    ];
  },

  // ============================================================================
  // MULTI-TENANT CONTEXT
  // ============================================================================
  
  // Map security context to app ID (for tenant isolation)
  contextToAppId: ({ securityContext }) => {
    return securityContext?.tenant_id || 'default';
  },

  // Map context to data source (for tenant-specific databases)
  contextToOrchestratorId: ({ securityContext }) => {
    return securityContext?.tenant_id || 'default';
  },

  // Dynamic data source configuration per tenant
  driverFactory: async ({ securityContext, dataSource }) => {
    const tenantId = securityContext?.tenant_id;
    
    // StarRocks sources don't need dynamic resolution
    if (dataSource === 'starrocks' || dataSource === 'starrocks_preagg' || dataSource === 'starrocks_cold') {
      return {
        type: 'mysql',
        host: process.env.STARROCKS_HOST || 'starrocks-fe',
        port: parseInt(process.env.STARROCKS_PORT || '9030'),
        user: process.env.STARROCKS_USER || 'root',
        password: process.env.STARROCKS_PASSWORD || '',
        database: dataSource === 'starrocks_preagg' ? 'cube_preagg' : 
                  dataSource === 'starrocks_cold' ? 'cube_cold' : 'cube_hot',
      };
    }
    
    // Tenant-specific PostgreSQL (isolated financial data)
    if (dataSource === 'tenant_financial' && tenantId) {
      const tenantDb = await getTenantDatabaseConfig(tenantId);
      return {
        type: 'postgres',
        host: tenantDb.host,
        port: tenantDb.port,
        database: tenantDb.database,
        user: tenantDb.user,
        password: tenantDb.password,
        ssl: tenantDb.ssl,
      };
    }
    
    return null; // Use default from datasources.js
  },

  // ============================================================================
  // PRE-AGGREGATION CONFIGURATION
  // ============================================================================
  
  // Store pre-aggregations in StarRocks (not Redis!)
  externalDbType: 'mysql',
  
  preAggregationsSchema: ({ securityContext }) => {
    // Tenant-specific pre-agg schema in StarRocks
    const tenantId = securityContext?.tenant_id || 'default';
    return `preagg_${tenantId.replace(/-/g, '_')}`;
  },

  // ============================================================================
  // CACHING (Cube Store, not Redis)
  // ============================================================================
  
  // Cache driver is automatically Cube Store when CUBEJS_CUBESTORE_HOST is set
  // No additional configuration needed

  // Query result cache TTL
  queryCacheOptions: {
    refreshKeyRenewalThreshold: 900, // 15 minutes
    backgroundRenew: true,
    queueOptions: {
      concurrency: 4,
    },
  },

  // ============================================================================
  // SECURITY
  // ============================================================================
  
  // Check query authorization
  checkAuth: async (req, authorization) => {
    // In production, validate JWT and extract tenant context
    if (process.env.CUBEJS_DEV_MODE === 'true') {
      return {
        tenant_id: req.headers['x-tenant-id'] || 'demo',
        user_id: 'dev-user',
        roles: ['admin'],
      };
    }
    
    // JWT validation would go here
    throw new Error('Authentication required');
  },

  // Query rewriting for row-level security
  queryRewrite: (query, { securityContext }) => {
    const tenantId = securityContext?.tenant_id;
    
    if (tenantId && query.filters) {
      // Add tenant filter to all queries if not already present
      const hasTenantFilter = query.filters.some(f => f.member?.endsWith('.tenant_id'));
      if (!hasTenantFilter) {
        query.filters.push({
          member: `${query.dimensions?.[0]?.split('.')[0] || 'Transactions'}.tenant_id`,
          operator: 'equals',
          values: [tenantId],
        });
      }
    }
    
    return query;
  },

  // ============================================================================
  // TELEMETRY & OBSERVABILITY
  // ============================================================================
  
  telemetry: false,
  
  // Custom logger
  logger: (msg, params) => {
    console.log(JSON.stringify({
      timestamp: new Date().toISOString(),
      message: msg,
      ...params,
    }));
  },
};

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

/**
 * Get tenant database configuration from metadata service
 */
async function getTenantDatabaseConfig(tenantId) {
  const metadataUrl = process.env.METADATA_API_URL || 'http://api-gateway:8080';
  
  try {
    const response = await fetch(`${metadataUrl}/api/v1/tenants/${tenantId}/database`, {
      headers: {
        'Authorization': `Bearer ${process.env.INTERNAL_SERVICE_TOKEN}`,
      },
    });
    
    if (!response.ok) {
      throw new Error(`Failed to get tenant database config: ${response.status}`);
    }
    
    const config = await response.json();
    
    // Get password from secrets manager
    const password = await getSecretValue(config.secret_ref);
    
    return {
      host: config.host,
      port: config.port,
      database: config.database_name,
      user: config.username,
      password: password,
      ssl: config.ssl_mode === 'require' ? { rejectUnauthorized: false } : false,
    };
  } catch (error) {
    console.error(`Failed to get tenant database config for ${tenantId}:`, error);
    throw error;
  }
}

/**
 * Get secret value (from env for dev, from secrets manager in prod)
 */
async function getSecretValue(secretRef) {
  if (process.env.CUBEJS_DEV_MODE === 'true') {
    // In dev, use environment variable
    return process.env.TENANT_DB_PASSWORD || 'tenant-demo-dev';
  }
  
  // In production, fetch from Azure Key Vault / AWS Secrets Manager
  // Implementation depends on cloud provider
  const secretsUrl = process.env.SECRETS_API_URL;
  if (secretsUrl) {
    const response = await fetch(`${secretsUrl}/${secretRef}`);
    const data = await response.json();
    return data.value;
  }
  
  throw new Error(`Secret not found: ${secretRef}`);
}

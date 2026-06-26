/**
 * Cube.js Security Integration Module
 * 
 * This module provides integration between Cube.js and the platform's
 * RBAC/ABAC security service with high-performance caching.
 * 
 * Usage in cube.js:
 *   const security = require('./security-integration');
 *   
 *   module.exports = {
 *     queryRewrite: security.queryRewrite,
 *     securityContext: security.securityContext,
 *     ...
 *   };
 */

const http = require('http');
const https = require('https');
const crypto = require('crypto');

// Configuration
const SECURITY_SERVICE_URL = process.env.CUBE_SECURITY_SERVICE_URL || 'http://localhost:8080/api/cube/security';
const CACHE_TTL_MS = parseInt(process.env.CUBE_SECURITY_CACHE_TTL_MS || '300000', 10); // 5 minutes
const CACHE_MAX_SIZE = parseInt(process.env.CUBE_SECURITY_CACHE_MAX_SIZE || '10000', 10);
const REQUEST_TIMEOUT_MS = parseInt(process.env.CUBE_SECURITY_TIMEOUT_MS || '5000', 10);

// In-memory L1 cache for security decisions
const securityCache = new Map();
let cacheHits = 0;
let cacheMisses = 0;

/**
 * Generate cache key from security context and cubes
 */
function generateCacheKey(securityContext, cubes = []) {
  const keyData = {
    tenant: securityContext.tenant_id,
    datasource: securityContext.datasource_id,
    user: securityContext.user_id,
    roles: (securityContext.roles || []).sort(),
    groups: (securityContext.groups || []).sort(),
    cubes: cubes.sort(),
  };
  const hash = crypto.createHash('sha256');
  hash.update(JSON.stringify(keyData));
  return hash.digest('hex');
}

/**
 * Check L1 cache for security decision
 */
function checkCache(key) {
  const entry = securityCache.get(key);
  if (!entry) {
    return null;
  }
  
  // Check expiration
  if (Date.now() > entry.expiresAt) {
    securityCache.delete(key);
    return null;
  }
  
  cacheHits++;
  entry.hitCount++;
  return entry.decision;
}

/**
 * Store decision in L1 cache
 */
function setCache(key, decision) {
  // Prune cache if too large
  if (securityCache.size >= CACHE_MAX_SIZE) {
    pruneCache();
  }
  
  securityCache.set(key, {
    decision,
    expiresAt: Date.now() + CACHE_TTL_MS,
    hitCount: 0,
    createdAt: Date.now(),
  });
}

/**
 * Remove expired and LRU entries
 */
function pruneCache() {
  const now = Date.now();
  const entries = [];
  
  // Remove expired entries
  for (const [key, value] of securityCache.entries()) {
    if (now > value.expiresAt) {
      securityCache.delete(key);
    } else {
      entries.push({ key, ...value });
    }
  }
  
  // If still too large, remove lowest hit count entries
  if (entries.length >= CACHE_MAX_SIZE * 0.8) {
    entries.sort((a, b) => a.hitCount - b.hitCount);
    const removeCount = Math.floor(entries.length * 0.2);
    for (let i = 0; i < removeCount; i++) {
      securityCache.delete(entries[i].key);
    }
  }
}

/**
 * Make HTTP request to security service
 */
function makeSecurityRequest(path, method, body, tenantId, datasourceId) {
  return new Promise((resolve, reject) => {
    const url = new URL(`${SECURITY_SERVICE_URL}${path}`);
    url.searchParams.set('tenant_id', tenantId);
    url.searchParams.set('datasource_id', datasourceId);
    
    const isHttps = url.protocol === 'https:';
    const transport = isHttps ? https : http;
    
    const options = {
      hostname: url.hostname,
      port: url.port || (isHttps ? 443 : 80),
      path: url.pathname + url.search,
      method,
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': tenantId,
        'X-Tenant-Datasource-ID': datasourceId,
      },
      timeout: REQUEST_TIMEOUT_MS,
    };
    
    const req = transport.request(options, (res) => {
      let data = '';
      res.on('data', chunk => { data += chunk; });
      res.on('end', () => {
        try {
          if (res.statusCode >= 200 && res.statusCode < 300) {
            resolve(JSON.parse(data));
          } else {
            reject(new Error(`Security service returned ${res.statusCode}: ${data}`));
          }
        } catch (err) {
          reject(new Error(`Failed to parse security response: ${err.message}`));
        }
      });
    });
    
    req.on('error', reject);
    req.on('timeout', () => {
      req.destroy();
      reject(new Error('Security service request timed out'));
    });
    
    if (body) {
      req.write(JSON.stringify(body));
    }
    req.end();
  });
}

/**
 * Evaluate security policies for a query
 * Returns security decision with row filters, column masks, and limits
 */
async function evaluateSecurity(securityContext, cubes = []) {
  const tenantId = securityContext.tenant_id;
  const datasourceId = securityContext.datasource_id;
  
  if (!tenantId || !datasourceId) {
    return {
      allowed: false,
      denial_reason: 'Missing tenant_id or datasource_id in security context',
      row_filters: [],
      column_masks: [],
      query_limits: null,
      applied_policies: [],
    };
  }
  
  // Check L1 cache
  const cacheKey = generateCacheKey(securityContext, cubes);
  const cached = checkCache(cacheKey);
  if (cached) {
    return cached;
  }
  
  cacheMisses++;
  
  // Call security service
  try {
    const decision = await makeSecurityRequest(
      '/evaluate',
      'POST',
      { security_context: securityContext, cubes },
      tenantId,
      datasourceId
    );
    
    // Cache the decision
    setCache(cacheKey, decision);
    
    return decision;
  } catch (err) {
    console.error('[CUBE-SECURITY] Failed to evaluate security:', err.message);
    
    // Fail-safe: deny on security service errors
    return {
      allowed: false,
      denial_reason: `Security evaluation failed: ${err.message}`,
      row_filters: [],
      column_masks: [],
      query_limits: null,
      applied_policies: [],
    };
  }
}

/**
 * Generate SQL WHERE clause from row filters
 */
function generateWhereClause(rowFilters, securityContext) {
  if (!rowFilters || rowFilters.length === 0) {
    return null;
  }
  
  const clauses = [];
  
  for (const filter of rowFilters) {
    let values = filter.values;
    
    // Resolve dynamic values
    if (filter.dynamic) {
      values = values.map(v => {
        if (typeof v === 'string') {
          if (v === '${tenant_id}' || v === '$tenant_id') {
            return securityContext.tenant_id;
          }
          if (v === '${user_id}' || v === '$user_id') {
            return securityContext.user_id;
          }
          if (v === '${datasource_id}' || v === '$datasource_id') {
            return securityContext.datasource_id;
          }
          // Check for attribute reference
          const attrMatch = v.match(/^\$\{(\w+)\}$/);
          if (attrMatch && securityContext.attributes) {
            return securityContext.attributes[attrMatch[1]];
          }
        }
        return v;
      }).filter(v => v !== undefined);
    }
    
    // Use raw expression if provided
    if (filter.expression) {
      clauses.push(`(${filter.expression})`);
      continue;
    }
    
    const dim = filter.dimension;
    
    switch (filter.operator) {
      case 'equals':
      case 'eq':
      case '=':
        if (values.length > 0) {
          clauses.push(`${dim} = ${sqlValue(values[0])}`);
        }
        break;
        
      case 'notEquals':
      case 'ne':
      case '!=':
        if (values.length > 0) {
          clauses.push(`${dim} != ${sqlValue(values[0])}`);
        }
        break;
        
      case 'in':
        if (values.length > 0) {
          clauses.push(`${dim} IN (${values.map(sqlValue).join(', ')})`);
        } else {
          clauses.push('1=0'); // Empty IN = no rows
        }
        break;
        
      case 'notIn':
        if (values.length > 0) {
          clauses.push(`${dim} NOT IN (${values.map(sqlValue).join(', ')})`);
        }
        break;
        
      case 'contains':
        if (values.length > 0) {
          clauses.push(`${dim} LIKE '%${escapeSQL(String(values[0]))}%'`);
        }
        break;
        
      case 'gt':
      case '>':
        if (values.length > 0) {
          clauses.push(`${dim} > ${sqlValue(values[0])}`);
        }
        break;
        
      case 'gte':
      case '>=':
        if (values.length > 0) {
          clauses.push(`${dim} >= ${sqlValue(values[0])}`);
        }
        break;
        
      case 'lt':
      case '<':
        if (values.length > 0) {
          clauses.push(`${dim} < ${sqlValue(values[0])}`);
        }
        break;
        
      case 'lte':
      case '<=':
        if (values.length > 0) {
          clauses.push(`${dim} <= ${sqlValue(values[0])}`);
        }
        break;
        
      case 'isNull':
        clauses.push(`${dim} IS NULL`);
        break;
        
      case 'isNotNull':
        clauses.push(`${dim} IS NOT NULL`);
        break;
    }
  }
  
  if (clauses.length === 0) {
    return null;
  }
  
  return clauses.join(' AND ');
}

/**
 * Convert value to SQL literal
 */
function sqlValue(v) {
  if (v === null || v === undefined) {
    return 'NULL';
  }
  if (typeof v === 'number') {
    return String(v);
  }
  if (typeof v === 'boolean') {
    return v ? 'TRUE' : 'FALSE';
  }
  return `'${escapeSQL(String(v))}'`;
}

/**
 * Escape SQL string
 */
function escapeSQL(s) {
  return s.replace(/'/g, "''");
}

/**
 * Enhanced queryRewrite with ABAC integration
 * 
 * This function replaces the basic tenant filter with full ABAC policy evaluation.
 * Row filters from security policies are injected as additional query filters.
 */
async function queryRewrite(query, { securityContext }) {
  const tenantId = securityContext.tenant_id;
  const datasourceId = securityContext.datasource_id;
  
  if (!tenantId || !datasourceId) {
    throw new Error('tenant_id and datasource_id are required in security context');
  }
  
  // Extract cubes from query
  const cubes = extractCubesFromQuery(query);
  
  // Evaluate security policies
  const decision = await evaluateSecurity(securityContext, cubes);
  
  // Deny access if not allowed
  if (!decision.allowed) {
    throw new Error(decision.denial_reason || 'Access denied by security policy');
  }
  
  // Start with mandatory tenant isolation
  const filters = [
    {
      member: 'tenant_id',
      operator: 'equals',
      values: [tenantId],
    },
    {
      member: 'datasource_id',
      operator: 'equals',
      values: [datasourceId],
    },
  ];
  
  // Add row filters from ABAC policies
  if (decision.row_filters && decision.row_filters.length > 0) {
    for (const filter of decision.row_filters) {
      // Skip cube-specific filters for other cubes
      if (filter.cube && cubes.length > 0 && !cubes.includes(filter.cube)) {
        continue;
      }
      
      let values = filter.values;
      
      // Resolve dynamic values
      if (filter.dynamic) {
        values = values.map(v => {
          if (typeof v === 'string') {
            if (v === '${tenant_id}' || v === '$tenant_id') return tenantId;
            if (v === '${user_id}' || v === '$user_id') return securityContext.user_id;
            if (v === '${datasource_id}' || v === '$datasource_id') return datasourceId;
            const attrMatch = v.match(/^\$\{(\w+)\}$/);
            if (attrMatch && securityContext.attributes) {
              return securityContext.attributes[attrMatch[1]];
            }
          }
          return v;
        }).filter(v => v !== undefined);
      }
      
      filters.push({
        member: filter.dimension,
        operator: filter.operator === 'eq' ? 'equals' : filter.operator,
        values: values,
      });
    }
  }
  
  // Apply query limits if defined
  if (decision.query_limits) {
    const limits = decision.query_limits;
    
    // Enforce max rows
    if (limits.max_rows && (!query.limit || query.limit > limits.max_rows)) {
      query.limit = limits.max_rows;
    }
    
    // Enforce cube restrictions
    if (limits.denied_cubes && limits.denied_cubes.length > 0) {
      for (const cube of cubes) {
        if (limits.denied_cubes.includes(cube)) {
          throw new Error(`Access to cube '${cube}' is denied by security policy`);
        }
      }
    }
    
    if (limits.allowed_cubes && limits.allowed_cubes.length > 0) {
      for (const cube of cubes) {
        if (!limits.allowed_cubes.includes(cube)) {
          throw new Error(`Access to cube '${cube}' is not permitted`);
        }
      }
    }
    
    // Enforce measure restrictions
    if (query.measures) {
      for (const measure of query.measures) {
        const measureName = measure.split('.').pop();
        if (limits.denied_measures?.includes(measureName) || limits.denied_measures?.includes(measure)) {
          throw new Error(`Access to measure '${measure}' is denied by security policy`);
        }
        if (limits.allowed_measures?.length > 0 && 
            !limits.allowed_measures.includes(measureName) && 
            !limits.allowed_measures.includes(measure)) {
          throw new Error(`Access to measure '${measure}' is not permitted`);
        }
      }
    }
    
    // Enforce dimension restrictions
    if (query.dimensions) {
      for (const dimension of query.dimensions) {
        const dimName = dimension.split('.').pop();
        if (limits.denied_dimensions?.includes(dimName) || limits.denied_dimensions?.includes(dimension)) {
          throw new Error(`Access to dimension '${dimension}' is denied by security policy`);
        }
        if (limits.allowed_dimensions?.length > 0 &&
            !limits.allowed_dimensions.includes(dimName) &&
            !limits.allowed_dimensions.includes(dimension)) {
          throw new Error(`Access to dimension '${dimension}' is not permitted`);
        }
      }
    }
  }
  
  // Store security metadata for result transformation
  query.__securityDecision = decision;
  
  return {
    ...query,
    filters: [...(query.filters || []), ...filters],
  };
}

/**
 * Extract cube names from query
 */
function extractCubesFromQuery(query) {
  const cubes = new Set();
  
  // From measures
  if (query.measures) {
    for (const m of query.measures) {
      const parts = m.split('.');
      if (parts.length >= 2) {
        cubes.add(parts[0]);
      }
    }
  }
  
  // From dimensions
  if (query.dimensions) {
    for (const d of query.dimensions) {
      const parts = d.split('.');
      if (parts.length >= 2) {
        cubes.add(parts[0]);
      }
    }
  }
  
  // From filters
  if (query.filters) {
    for (const f of query.filters) {
      if (f.member) {
        const parts = f.member.split('.');
        if (parts.length >= 2) {
          cubes.add(parts[0]);
        }
      }
    }
  }
  
  // From time dimensions
  if (query.timeDimensions) {
    for (const td of query.timeDimensions) {
      if (td.dimension) {
        const parts = td.dimension.split('.');
        if (parts.length >= 2) {
          cubes.add(parts[0]);
        }
      }
    }
  }
  
  return Array.from(cubes);
}

/**
 * Enhanced security context generator
 * Builds security context from JWT claims and request headers
 */
function buildSecurityContext(req, authInfo) {
  const context = {
    tenant_id: null,
    datasource_id: null,
    user_id: null,
    roles: [],
    groups: [],
    attributes: {},
    session_id: null,
    ip_address: null,
    request_timestamp: new Date().toISOString(),
  };
  
  // From JWT claims (highest priority)
  if (authInfo && authInfo.u) {
    context.tenant_id = authInfo.tenant_id || authInfo.tid;
    context.datasource_id = authInfo.datasource_id || authInfo.dsid;
    context.user_id = authInfo.sub || authInfo.user_id || authInfo.u;
    context.roles = authInfo.roles || authInfo.role || [];
    context.groups = authInfo.groups || authInfo.grp || [];
    context.session_id = authInfo.sid || authInfo.session_id;
    
    // Copy all other claims as attributes
    for (const [key, value] of Object.entries(authInfo)) {
      if (!['tenant_id', 'datasource_id', 'user_id', 'roles', 'groups', 'sub', 'exp', 'iat', 'iss', 'aud'].includes(key)) {
        context.attributes[key] = value;
      }
    }
  }
  
  // From headers (override if provided)
  if (req && req.headers) {
    if (req.headers['x-tenant-id']) {
      context.tenant_id = req.headers['x-tenant-id'];
    }
    if (req.headers['x-tenant-datasource-id']) {
      context.datasource_id = req.headers['x-tenant-datasource-id'];
    }
    if (req.headers['x-user-id']) {
      context.user_id = req.headers['x-user-id'];
    }
    if (req.headers['x-user-roles']) {
      context.roles = req.headers['x-user-roles'].split(',').map(r => r.trim());
    }
    if (req.headers['x-user-groups']) {
      context.groups = req.headers['x-user-groups'].split(',').map(g => g.trim());
    }
    
    // Get IP address
    context.ip_address = req.headers['x-forwarded-for']?.split(',')[0]?.trim() || 
                         req.headers['x-real-ip'] ||
                         req.connection?.remoteAddress;
  }
  
  // Ensure roles is always an array
  if (typeof context.roles === 'string') {
    context.roles = [context.roles];
  }
  if (typeof context.groups === 'string') {
    context.groups = [context.groups];
  }
  
  return context;
}

/**
 * Apply column masks to query results
 * This should be called in the result transformation phase
 */
function applyColumnMasks(results, securityDecision, securityContext) {
  if (!securityDecision || !securityDecision.column_masks || securityDecision.column_masks.length === 0) {
    return results;
  }
  
  const userRoles = new Set(securityContext.roles || []);
  
  // Build mask map
  const maskMap = new Map();
  for (const mask of securityDecision.column_masks) {
    // Check if user has role that bypasses masking
    if (mask.allowed_roles && mask.allowed_roles.some(r => userRoles.has(r))) {
      continue;
    }
    maskMap.set(mask.member, mask);
  }
  
  if (maskMap.size === 0) {
    return results;
  }
  
  // Apply masks to result data
  if (Array.isArray(results)) {
    return results.map(row => applyMasksToRow(row, maskMap));
  }
  
  if (results && results.data && Array.isArray(results.data)) {
    return {
      ...results,
      data: results.data.map(row => applyMasksToRow(row, maskMap)),
    };
  }
  
  return results;
}

/**
 * Apply masks to a single row
 */
function applyMasksToRow(row, maskMap) {
  const masked = { ...row };
  
  for (const [key, value] of Object.entries(row)) {
    // Check both full member name and short name
    const parts = key.split('.');
    const shortName = parts[parts.length - 1];
    
    const mask = maskMap.get(key) || maskMap.get(shortName);
    if (!mask) continue;
    
    masked[key] = applyMask(value, mask);
  }
  
  return masked;
}

/**
 * Apply a single mask to a value
 */
function applyMask(value, mask) {
  if (value === null || value === undefined) {
    return value;
  }
  
  switch (mask.mask_type) {
    case 'redact':
      return '***REDACTED***';
      
    case 'nullify':
      return null;
      
    case 'hash':
      const hash = crypto.createHash('sha256');
      hash.update(String(value));
      return hash.digest('hex').substring(0, 16);
      
    case 'truncate':
      const str = String(value);
      return str.length > 3 ? str.substring(0, 3) + '...' : str;
      
    case 'partial':
      if (mask.mask_pattern) {
        // Pattern like "XXX-XX-{last4}" for SSN
        const pattern = mask.mask_pattern;
        const strValue = String(value);
        
        return pattern.replace(/\{last(\d+)\}/g, (_, digits) => {
          const n = parseInt(digits, 10);
          return strValue.slice(-n);
        }).replace(/\{first(\d+)\}/g, (_, digits) => {
          const n = parseInt(digits, 10);
          return strValue.slice(0, n);
        });
      }
      // Default partial: show first and last char
      const s = String(value);
      if (s.length <= 2) return '**';
      return s[0] + '*'.repeat(s.length - 2) + s[s.length - 1];
      
    case 'custom':
      // Custom masks should be handled externally
      return '[MASKED]';
      
    default:
      return '***';
  }
}

/**
 * Get cache statistics for monitoring
 */
function getCacheStats() {
  return {
    size: securityCache.size,
    hits: cacheHits,
    misses: cacheMisses,
    hitRate: cacheHits + cacheMisses > 0 
      ? (cacheHits / (cacheHits + cacheMisses) * 100).toFixed(2) + '%'
      : '0%',
    ttlMs: CACHE_TTL_MS,
    maxSize: CACHE_MAX_SIZE,
  };
}

/**
 * Invalidate cache (call when policies change)
 */
function invalidateCache() {
  securityCache.clear();
  console.log('[CUBE-SECURITY] Cache invalidated');
}

/**
 * Export as Cube.js compatible module
 */
module.exports = {
  queryRewrite,
  evaluateSecurity,
  buildSecurityContext,
  applyColumnMasks,
  generateWhereClause,
  getCacheStats,
  invalidateCache,
  extractCubesFromQuery,
};

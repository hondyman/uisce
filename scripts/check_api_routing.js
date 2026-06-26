const fs = require('fs');
const path = require('path');

function readEnv(file) {
  const p = path.resolve(file);
  if (!fs.existsSync(p)) return {};
  const raw = fs.readFileSync(p, 'utf8');
  const out = {};
  for (const line of raw.split(/\n/)) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;
    const idx = trimmed.indexOf('=');
    if (idx === -1) continue;
    const k = trimmed.slice(0, idx);
    const v = trimmed.slice(idx + 1);
    out[k] = v;
  }
  return out;
}

const env = readEnv(path.join(__dirname, '../frontend/.env.local'));
const DEV = true;
const frontendOrigin = 'http://localhost:5173';
const configuredBase = env.VITE_API_BASE_URL || undefined;
const backendTarget = env.VITE_BACKEND_TARGET || undefined;
const useProxy = (env.VITE_USE_PROXY || 'true').toLowerCase() === 'true';

function shouldEnforceTenantScope(url) {
  try {
    const resolved = new URL(url, frontendOrigin);
    const { pathname, hostname } = resolved;
    if (hostname.includes('localhost') && (pathname.includes('/v1/graphql') || pathname.includes('/graphql'))) return false;
    if (!pathname.startsWith('/api')) return false;
    const OPTIONAL = ['/api/tenants','/api/auth','/api/system','/api/health','/api/status','/api/marketplace'];
    return !OPTIONAL.some(p=>pathname.startsWith(p));
  } catch (e) { return true; }
}

function appendScopeToUrl(url, tenantId, datasourceId) {
  try {
    if (backendTarget && DEV && !configuredBase) {
      let final = new URL(url, backendTarget);
      try {
        if (final.origin === frontendOrigin) {
          const base = new URL(backendTarget);
          final = new URL(final.pathname + final.search, base.origin);
        }
      } catch(e){}
      final.searchParams.set('tenant_id', tenantId);
      final.searchParams.set('datasource_id', datasourceId);
      return final.toString();
    }

    if (configuredBase) {
      let final = new URL(url, configuredBase);
      try {
        if (final.origin === frontendOrigin) {
          const base = new URL(configuredBase);
          final = new URL(final.pathname + final.search, base.origin);
        }
      } catch(e){}
      final.searchParams.set('tenant_id', tenantId);
      final.searchParams.set('datasource_id', datasourceId);
      return final.toString();
    }

    const fallback = new URL(url, frontendOrigin);
    fallback.searchParams.set('tenant_id', tenantId);
    fallback.searchParams.set('datasource_id', datasourceId);
    return fallback.toString();
  } catch(e) { return url; }
}

const samples = [
  '/api/views',
  '/api/health',
  'http://localhost:5173/api/views',
  '/api/graphql',
  'http://localhost:5173/api/graphql',
  '/api/semantic-objects/search?q=test'
];

const tenant = '910638ba-a459-4a3f-bb2d-78391b0595f6';
const datasource = '982aef38-418f-46dc-acd0-35fe8f3b97b0';

console.log('ENV:', { configuredBase, backendTarget, useProxy });
for (const s of samples) {
  console.log('\nSample:', s);
  console.log('  enforceScope?', shouldEnforceTenantScope(s));
  const final = appendScopeToUrl(s, tenant, datasource);
  console.log('  final:', final);
}

const express = require('express');
const http = require('http');
const https = require('https');

const app = express();
const PORT = process.env.PORT || 5175;

const API_TARGET = process.env.API_TARGET || 'http://localhost:8001';
const CATALOG_TARGET = process.env.CATALOG_TARGET || 'http://localhost:9088';

// Add JSON body parsing middleware
app.use(express.json());

// Add CORS headers for cross-origin requests from frontend
app.use((req, res, next) => {
  res.header('Access-Control-Allow-Origin', '*');
  res.header('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
  res.header('Access-Control-Allow-Headers', 'Content-Type, Authorization');
  if (req.method === 'OPTIONS') {
    return res.sendStatus(200);
  }
  next();
});

// Log all incoming requests so we can trace routing issues
app.use((req, _res, next) => {
  try {
    console.log('[dev-proxy] incoming:', req.method, req.originalUrl);
  } catch (e) {}
  next();
});

// In-memory storage for saved models (for development)
const savedModels = new Map();

// Forward specific model API endpoints to the gateway (backend)
// These are handled by the backend, not the dev-proxy
app.use('/api/models/generated', (req, _res, next) => {
  console.log('[dev-proxy] Forwarding models/generated to gateway:', req.method, req.originalUrl);
  next('route'); // Skip to the general /api proxy
});

app.use('/api/models/custom', (req, _res, next) => {
  console.log('[dev-proxy] Forwarding models/custom to gateway:', req.method, req.originalUrl);
  next('route'); // Skip to the general /api proxy
});

app.use('/api/models/clone', (req, _res, next) => {
  console.log('[dev-proxy] Forwarding models/clone to gateway:', req.method, req.originalUrl);
  next('route'); // Skip to the general /api proxy
});

// Model save endpoint for simple table names - MUST come before general /api proxy
app.post('/api/models/:tableName', (req, res, next) => {
  const { tableName } = req.params;
  
  // Skip backend-specific endpoints
  if (tableName === 'generated' || tableName === 'custom' || tableName === 'clone') {
    return next('route');
  }
  
  const modelData = req.body;
  
  console.log(`[dev-proxy] Saving model for table: ${tableName}`);
  savedModels.set(tableName, {
    ...modelData,
    savedAt: new Date().toISOString()
  });
  
  res.json({ 
    success: true, 
    message: `Model saved for ${tableName}`,
    tableName 
  });
});

// Model retrieve endpoint for simple table names - MUST come before general /api proxy
app.get('/api/models/:tableName', (req, res, next) => {
  const { tableName } = req.params;
  
  // Skip backend-specific endpoints
  if (tableName === 'generated' || tableName === 'custom' || tableName === 'clone') {
    return next('route');
  }
  
  console.log(`[dev-proxy] Retrieving model for table: ${tableName}`);
  const model = savedModels.get(tableName);
  
  if (model) {
    res.json(model);
  } else {
    res.status(404).json({ 
      error: 'Model not found',
      message: `No saved model found for table: ${tableName}`
    });
  }
});

// Minimal manual proxy implementation for reliable debugging in dev
// But handle /api/catalog specially: forward to CATALOG_TARGET
app.use('/api/catalog', (req, res) => {
  console.log('[dev-proxy] catalog before-proxy:', req.method, req.originalUrl);
  try {
    const targetUrl = new URL(CATALOG_TARGET);
    const isHttps = targetUrl.protocol === 'https:';
    const proxyOptions = {
      hostname: targetUrl.hostname,
      port: targetUrl.port || (isHttps ? 443 : 80),
      path: req.originalUrl, // forward path and query as-is
      method: req.method,
      headers: Object.assign({}, req.headers),
    };

    proxyOptions.headers.host = targetUrl.host;

    const proxyReq = (isHttps ? https : http).request(proxyOptions, (proxyRes) => {
      console.log('[dev-proxy] catalog upstream response:', proxyRes.statusCode, proxyRes.statusMessage, 'for', req.method, req.originalUrl);
      res.writeHead(proxyRes.statusCode, proxyRes.headers);
      proxyRes.pipe(res, { end: true });
    });

    proxyReq.on('error', (err) => {
      console.error('[dev-proxy] manual proxy error for', req.originalUrl, err && err.message ? err.message : err);
      res.statusCode = 502;
      res.end('Bad gateway (dev-proxy manual)');
    });

    req.pipe(proxyReq, { end: true });
  } catch (e) {
    console.error('[dev-proxy] exception in manual proxy', e && e.stack ? e.stack : e);
    res.statusCode = 500;
    res.end('dev-proxy internal error');
  }
});

// Generic /api proxy falls back to API_TARGET
app.use('/api', (req, res) => {
  console.log('[dev-proxy] before-proxy:', req.method, req.originalUrl);
  // If caller is requesting fabric_defn, dump a few helpful headers to trace origin
  try {
    if (req.originalUrl && req.originalUrl.includes('fabric_defn')) {
      // Verbose tracing to help identify the origin of fabric_defn requests
      try {
        const bodySnippet = (req.body && Object.keys(req.body).length) ? req.body : '<no-body-or-unparsed>';
        console.log('[dev-proxy] >>> fabric_defn TRACE {timestamp, method, url, remoteAddress, headers, body}:', {
          timestamp: new Date().toISOString(),
          method: req.method,
          url: req.originalUrl,
          remoteAddress: (req.socket && (req.socket.remoteAddress || req.socket.localAddress)) || '<none>',
          headers: req.headers || {},
          body: bodySnippet
        });
      } catch (innerErr) {
        console.error('[dev-proxy] >>> fabric_defn verbose logging failed:', innerErr && innerErr.message ? innerErr.message : innerErr);
      }
    }
  } catch (e) {
    // swallow outer logging errors
  }
  try {
    const targetUrl = new URL(API_TARGET);
    const isHttps = targetUrl.protocol === 'https:';
    const proxyOptions = {
      hostname: targetUrl.hostname,
      port: targetUrl.port || (isHttps ? 443 : 80),
      path: req.originalUrl, // forward path and query as-is
      method: req.method,
      headers: Object.assign({}, req.headers),
    };

    // ensure host header points to backend host
    proxyOptions.headers.host = targetUrl.host;

    const proxyReq = (isHttps ? https : http).request(proxyOptions, (proxyRes) => {
      console.log('[dev-proxy] upstream response:', proxyRes.statusCode, proxyRes.statusMessage, 'for', req.method, req.originalUrl);
      res.writeHead(proxyRes.statusCode, proxyRes.headers);
      proxyRes.pipe(res, { end: true });
    });

    proxyReq.on('error', (err) => {
      console.error('[dev-proxy] manual proxy error for', req.originalUrl, err && err.message ? err.message : err);
      res.statusCode = 502;
      res.end('Bad gateway (dev-proxy manual)');
    });

    // pipe request body
    req.pipe(proxyReq, { end: true });
  } catch (e) {
    console.error('[dev-proxy] exception in manual proxy', e && e.stack ? e.stack : e);
    res.statusCode = 500;
    res.end('dev-proxy internal error');
  }
});

// Generic error handler to log unexpected errors
app.use((err, req, res, _next) => {
  console.error('[dev-proxy] uncaught error for', req && req.originalUrl ? req.originalUrl : '<unknown>', err && err.stack ? err.stack : err);
  if (!res.headersSent) {
    res.status(500).send('dev-proxy internal error');
  }
});

// Simple health and debug endpoints
app.get('/_ping', (_req, res) => res.send('ok'));
app.get('/_debug', (_req, res) => {
  console.log('[dev-proxy] debug endpoint called');
  res.json({ 
    status: 'working', 
    timestamp: new Date().toISOString(),
    target: API_TARGET 
  });
});

app.listen(PORT, () => {
  console.log(`Dev proxy listening on http://localhost:${PORT}, forwarding /api -> ${API_TARGET}`);
});

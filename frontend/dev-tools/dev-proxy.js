const express = require('express');
const { createProxyMiddleware } = require('http-proxy-middleware');

const app = express();
const PORT = process.env.PORT || 5175;

const API_TARGET = process.env.API_TARGET || 'http://localhost:9090';
const CATALOG_TARGET = process.env.CATALOG_TARGET || 'http://localhost:9090';

// Add a proxy for catalog endpoints specifically so they can be forwarded to a
// different upstream (backend) while leaving other /api routes pointed at API_TARGET.
const catalogProxy = createProxyMiddleware('/api/catalog', {
  target: CATALOG_TARGET,
  changeOrigin: true,
  secure: false,
  logLevel: 'debug',
  onProxyReq: (proxyReq, _req, _res) => {
    try {
      console.error('[dev-proxy] proxying catalog request ->', proxyReq.method, proxyReq.path, 'to', CATALOG_TARGET);
    } catch (e) {}
  },
  onProxyRes: (proxyRes, _req, _res) => {
    try {
      console.error('[dev-proxy] catalog upstream response:', proxyRes.statusCode, proxyRes.statusMessage, 'for', _req.method, _req.originalUrl);
    } catch (e) {}
  }
});

// Add a proxy with logging hooks so we can diagnose routing / upstream errors in dev
const apiProxy = createProxyMiddleware('/api', {
  target: API_TARGET,
  changeOrigin: true,
  secure: false,
  logLevel: 'debug',
  onProxyReq: (proxyReq, _req, _res) => {
    try {
      console.error('[dev-proxy] proxying request ->', proxyReq.method, proxyReq.path, 'to', API_TARGET);
    } catch (e) {
      // ignore logging errors
    }
  },
  onProxyRes: (proxyRes, _req, _res) => {
    try {
      console.error('[dev-proxy] upstream response:', proxyRes.statusCode, proxyRes.statusMessage, 'for', _req.method, _req.originalUrl);
    } catch (e) {
      // ignore
    }
  },
  onError: (err, _req, _res) => {
    console.error('[dev-proxy] proxy error for', _req && _req.originalUrl ? _req.originalUrl : '<unknown>', err && err.message ? err.message : err);
    try {
      _res.writeHead(502, { 'Content-Type': 'text/plain' });
      _res.end('Bad gateway (dev-proxy)');
    } catch (e) {
      // ignore
    }
  }
});

app.use(apiProxy);

// Ensure catalog proxy is registered before the generic /api proxy so it wins
app.use(catalogProxy);

app.listen(PORT, () => {
  console.error(`Dev proxy listening on http://localhost:${PORT}, forwarding /api -> ${API_TARGET}`);
});

// Simple health
app.get('/_ping', (_req, res) => res.send('ok'));

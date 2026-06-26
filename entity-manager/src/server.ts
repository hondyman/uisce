import express from 'express';
import cors from 'cors';
import helmet from 'helmet';
import compression from 'compression';
import { createServer } from 'http';
import { config } from 'dotenv';
import { getEnv } from '../internal/pkg/env/getEnv.js';

// Load environment variables
config();

import { logger } from './utils/logger.js';
import { setupRoutes } from './api/routes.js';
import { connectDatabase } from './services/database.js';
import { connectRedis } from './services/redis.js';
import { initTemporal } from './services/temporal.js';
import { initKafka } from './services/kafka.js';
import { ApprovalWorkflowEngine } from './approval/ApprovalWorkflowEngine.js';
import { jwtMiddleware, injectTenantFromClaims } from '../../libs/jwt-middleware-node.js';

const app = express();
const PORT = Number(getEnv('PORT', 'VITE_PORT', '4000'));

// Middleware
app.use(helmet());
app.use(cors());
app.use(compression());
app.use(express.json({ limit: '10mb' }));
app.use(express.urlencoded({ extended: true }));

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    timestamp: new Date().toISOString(),
    service: 'entity-manager'
  });
});

// Readiness endpoint: verifies DB + Redis connectivity
import { getPool } from './services/database.js';
import { getRedisClient } from './services/redis.js';

app.get('/ready', async (req, res) => {
  try {
    // Database check
    try {
      const pool = getPool();
      await pool.query('SELECT 1');
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      throw new Error(`DB not ready: ${msg}`);
    }

    // Redis check
    try {
      const client = getRedisClient();
      if (typeof client.ping === 'function') {
        await client.ping();
      } else {
        await client.sendCommand(['PING']);
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      throw new Error(`Redis not ready: ${msg}`);
    }

    res.status(200).json({ status: 'ready' });
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err);
    res.status(503).json({ status: 'not_ready', error: msg });
  }
});

// JWT Middleware - validates JWT on all routes except /health and /ready
const publicPaths = ['/health', '/ready', '/docs', '/api/docs', '/api-docs'];
app.use(jwtMiddleware(publicPaths));
app.use(injectTenantFromClaims());

// API routes
setupRoutes(app);

// Error handling middleware
app.use((err: Error, req: express.Request, res: express.Response, next: express.NextFunction) => {
  logger.error('Unhandled error:', err);
  res.status(500).json({
    error: 'Internal server error',
    message: getEnv('NODE_ENV', 'VITE_NODE_ENV') === 'development' ? err.message : 'Something went wrong'
  });
});

// 404 handler
app.use((req, res) => {
  res.status(404).json({
    error: 'Not found',
    message: `Route ${req.method} ${req.path} not found`
  });
});

async function startServer() {
  try {
    // Initialize connections
    await connectDatabase();
    await connectRedis();
    await initTemporal();
    try {
      await initKafka();
    } catch (error) {
      logger.error('❌ Kafka connection failed:', error);
      if (getEnv('NODE_ENV', 'VITE_NODE_ENV') === 'production') {
        throw error;
      }
      logger.warn('Continuing without Kafka (non-production)');
    }

    try {
      await ApprovalWorkflowEngine.getInstance().setupEventListeners();
    } catch (e) {
      logger.warn('Approval event listeners not started:', e);
    }

    // Start server
    const server = createServer(app);
    server.listen(PORT, () => {
      logger.info(`🚀 Entity Manager Service running on port ${PORT}`);
      logger.info(`📊 Health check: http://localhost:${PORT}/health`);
      logger.info(`📚 API Documentation: http://localhost:${PORT}/api/docs`);
    });

  } catch (error) {
    logger.error('Failed to start server:', error);
    process.exit(1);
  }
}

// Graceful shutdown
process.on('SIGTERM', () => {
  logger.info('SIGTERM received, shutting down gracefully');
  process.exit(0);
});

process.on('SIGINT', () => {
  logger.info('SIGINT received, shutting down gracefully');
  process.exit(0);
});

startServer();
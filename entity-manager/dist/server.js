import express from 'express';
import cors from 'cors';
import helmet from 'helmet';
import compression from 'compression';
import { createServer } from 'http';
import { config } from 'dotenv';
import { getEnv } from '../internal/pkg/env/getEnv.js';
config();
import { logger } from './utils/logger.js';
import { setupRoutes } from './api/routes.js';
import { connectDatabase } from './services/database.js';
import { connectRedis } from './services/redis.js';
import { initTemporal } from './services/temporal.js';
import { initKafka } from './services/kafka.js';
import { ApprovalWorkflowEngine } from './approval/ApprovalWorkflowEngine.js';
const app = express();
const PORT = Number(getEnv('PORT', 'VITE_PORT', '4000'));
app.use(helmet());
app.use(cors());
app.use(compression());
app.use(express.json({ limit: '10mb' }));
app.use(express.urlencoded({ extended: true }));
app.get('/health', (req, res) => {
    res.json({
        status: 'healthy',
        timestamp: new Date().toISOString(),
        service: 'entity-manager'
    });
});
import { getPool } from './services/database.js';
import { getRedisClient } from './services/redis.js';
app.get('/ready', async (req, res) => {
    try {
        try {
            const pool = getPool();
            await pool.query('SELECT 1');
        }
        catch (e) {
            const msg = e instanceof Error ? e.message : String(e);
            throw new Error(`DB not ready: ${msg}`);
        }
        try {
            const client = getRedisClient();
            if (typeof client.ping === 'function') {
                await client.ping();
            }
            else {
                await client.sendCommand(['PING']);
            }
        }
        catch (e) {
            const msg = e instanceof Error ? e.message : String(e);
            throw new Error(`Redis not ready: ${msg}`);
        }
        res.status(200).json({ status: 'ready' });
    }
    catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        res.status(503).json({ status: 'not_ready', error: msg });
    }
});
setupRoutes(app);
app.use((err, req, res, next) => {
    logger.error('Unhandled error:', err);
    res.status(500).json({
        error: 'Internal server error',
        message: getEnv('NODE_ENV', 'VITE_NODE_ENV') === 'development' ? err.message : 'Something went wrong'
    });
});
app.use((req, res) => {
    res.status(404).json({
        error: 'Not found',
        message: `Route ${req.method} ${req.path} not found`
    });
});
async function startServer() {
    try {
        await connectDatabase();
        await connectRedis();
        await initTemporal();
        try {
            await initKafka();
        }
        catch (error) {
            logger.error('❌ Kafka connection failed:', error);
            if (getEnv('NODE_ENV', 'VITE_NODE_ENV') === 'production') {
                throw error;
            }
            logger.warn('Continuing without Kafka (non-production)');
        }
        try {
            await ApprovalWorkflowEngine.getInstance().setupEventListeners();
        }
        catch (e) {
            logger.warn('Approval event listeners not started:', e);
        }
        const server = createServer(app);
        server.listen(PORT, () => {
            logger.info(`🚀 Entity Manager Service running on port ${PORT}`);
            logger.info(`📊 Health check: http://localhost:${PORT}/health`);
            logger.info(`📚 API Documentation: http://localhost:${PORT}/api/docs`);
        });
    }
    catch (error) {
        logger.error('Failed to start server:', error);
        process.exit(1);
    }
}
process.on('SIGTERM', () => {
    logger.info('SIGTERM received, shutting down gracefully');
    process.exit(0);
});
process.on('SIGINT', () => {
    logger.info('SIGINT received, shutting down gracefully');
    process.exit(0);
});
startServer();
//# sourceMappingURL=server.js.map
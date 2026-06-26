import { Pool } from 'pg';
import { logger } from '../utils/logger.js';
import { getEnv } from '../../internal/pkg/env/getEnv.js';
let pool;
export async function connectDatabase() {
    const databaseUrl = getEnv('DATABASE_URL', 'VITE_DATABASE_URL');
    if (!databaseUrl) {
        throw new Error('DATABASE_URL environment variable is required');
    }
    const retries = Number(process.env.DB_CONNECT_RETRIES ?? 10);
    const baseDelay = Number(process.env.DB_CONNECT_DELAY_MS ?? 2000);
    const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
    for (let attempt = 1; attempt <= retries; attempt++) {
        try {
            pool = new Pool({
                connectionString: databaseUrl,
                max: 20,
                idleTimeoutMillis: 30000,
                connectionTimeoutMillis: 2000,
            });
            const client = await pool.connect();
            await client.query('SELECT NOW()');
            client.release();
            logger.info('✅ Database connected successfully');
            return;
        }
        catch (error) {
            logger.error(`❌ Database connection failed (attempt ${attempt}/${retries}):`, error);
            try {
                if (pool)
                    await pool.end();
            }
            catch (e) {
                logger.debug('Error while closing pool after failed attempt', e);
            }
            if (attempt === retries) {
                logger.error('Exceeded max Database connection attempts');
                throw error;
            }
            const backoff = baseDelay * 2 ** (attempt - 1);
            logger.info(`Retrying Database connection in ${backoff}ms...`);
            await sleep(backoff);
        }
    }
}
export function getPool() {
    if (!pool) {
        throw new Error('Database not connected. Call connectDatabase() first.');
    }
    return pool;
}
export async function disconnectDatabase() {
    if (pool) {
        await pool.end();
        logger.info('Database connection closed');
    }
}
//# sourceMappingURL=database.js.map
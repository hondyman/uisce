import { createClient } from 'redis';
import { logger } from '../utils/logger.js';
import { getEnv } from '../../internal/pkg/env/getEnv.js';

let redisClient: ReturnType<typeof createClient>;

export async function connectRedis(): Promise<void> {
  let redisUrl = getEnv('REDIS_URL', 'VITE_REDIS_URL') || process.env.REDIS_ADDR || 'redis://localhost:6379';
  if (!redisUrl.startsWith('redis://')) {
    redisUrl = `redis://${redisUrl}`;
  }
  logger.info(`Connecting to Redis at ${redisUrl}`);
  const retries = Number(process.env.REDIS_CONNECT_RETRIES ?? 10);
  const baseDelay = Number(process.env.REDIS_CONNECT_DELAY_MS ?? 2000);

  const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

  for (let attempt = 1; attempt <= retries; attempt++) {
    try {
      redisClient = createClient({ url: redisUrl });

      redisClient.on('error', (err) => logger.error('Redis Client Error', err));
      redisClient.on('connect', () => logger.info('✅ Redis connected successfully'));

      await redisClient.connect();
      return;
    } catch (error) {
      logger.error(`❌ Redis connection failed (attempt ${attempt}/${retries}):`, error);
      try {
        if (redisClient) await redisClient.disconnect();
      } catch (e) {
        logger.debug('Error while disconnecting failed client', e);
      }
      if (attempt === retries) {
        logger.error('Exceeded max Redis connection attempts');
        throw error;
      }
      const backoff = baseDelay * 2 ** (attempt - 1);
      logger.info(`Retrying Redis connection in ${backoff}ms...`);
      await sleep(backoff);
    }
  }
} 

export function getRedisClient(): ReturnType<typeof createClient> {
  if (!redisClient) {
    throw new Error('Redis not connected. Call connectRedis() first.');
  }
  return redisClient;
}

export async function disconnectRedis(): Promise<void> {
  if (redisClient) {
    await redisClient.disconnect();
    logger.info('Redis connection closed');
  }
}
import { Connection, Client } from '@temporalio/client';
import { logger } from '../utils/logger.js';
import { getEnv } from '../../internal/pkg/env/getEnv.js';

let temporalConnection: Connection;
let temporalClient: Client;

export async function initTemporal(): Promise<void> {
  try {
    const address = getEnv('TEMPORAL_ADDRESS', 'VITE_TEMPORAL_ADDRESS', 'localhost:7233');

    temporalConnection = await Connection.connect({ address });
    temporalClient = new Client({ connection: temporalConnection });

    logger.info('✅ Temporal connected successfully');
  } catch (error) {
    logger.error('❌ Temporal connection failed:', error);
    // Don't throw error in development if Temporal is not running
    if (getEnv('NODE_ENV', 'VITE_NODE_ENV') === 'production') {
      throw error;
    }
  }
}

export function getTemporalClient(): Client {
  if (!temporalClient) {
    throw new Error('Temporal not connected. Call initTemporal() first.');
  }
  return temporalClient;
}

export async function disconnectTemporal(): Promise<void> {
  if (temporalConnection) {
    temporalConnection.close();
    logger.info('Temporal connection closed');
  }
}
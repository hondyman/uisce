/* RabbitMQ removed — use Kafka (Redpanda) instead via `src/services/kafka.ts` */
import { logger } from '../utils/logger.js';



export async function initRabbitMQ(): Promise<void> {
  throw new Error('RabbitMQ integration has been removed. Use Kafka via services/kafka.ts (initKafka/publishEvent/getKafkaConsumer)');
}

export function getRabbitMQChannel(): never {
  throw new Error('RabbitMQ integration removed.');
}

export async function disconnectRabbitMQ(): Promise<void> {
  logger.warn('disconnectRabbitMQ called but RabbitMQ integration has been removed');
}

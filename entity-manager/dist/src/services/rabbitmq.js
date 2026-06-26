import { logger } from '../utils/logger.js';
export async function initRabbitMQ() {
    throw new Error('RabbitMQ integration has been removed. Use Kafka via services/kafka.ts (initKafka/publishEvent/getKafkaConsumer)');
}
export function getRabbitMQChannel() {
    throw new Error('RabbitMQ integration removed.');
}
export async function disconnectRabbitMQ() {
    logger.warn('disconnectRabbitMQ called but RabbitMQ integration has been removed');
}
//# sourceMappingURL=rabbitmq.js.map
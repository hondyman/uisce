import { Kafka } from 'kafkajs';
import { logger } from '../utils/logger.js';
import { getEnv } from '../../internal/pkg/env/getEnv.js';

let kafka: Kafka;
let producer: ReturnType<Kafka['producer']> | null = null;
let consumer: ReturnType<Kafka['consumer']> | null = null;
let kafkaReady = false; // whether connect completed successfully

const DEFAULT_BROKERS = ['localhost:9092'];

function parseBrokers(): string[] {
  const raw = getEnv('KAFKA_BROKERS', 'VITE_KAFKA_BROKERS');
  if (!raw) return DEFAULT_BROKERS;
  return raw.split(',').map((s) => s.trim());
}

export async function initKafka(): Promise<void> {
  const brokers = parseBrokers();
  kafka = new Kafka({ clientId: 'entity-manager', brokers });

  producer = kafka.producer();
  consumer = kafka.consumer({ groupId: 'entity-manager-group' });

  const retries = Number(process.env.KAFKA_CONNECT_RETRIES ?? 5);
  const baseDelay = Number(process.env.KAFKA_CONNECT_DELAY_MS ?? 1000);

  const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

  for (let attempt = 1; attempt <= retries; attempt++) {
    try {
      await producer.connect();
      await consumer.connect();
      kafkaReady = true;
      logger.info('✅ Kafka connected successfully', { brokers });
      return;
    } catch (err) {
      logger.error(`Kafka connection attempt ${attempt}/${retries} failed`, err);
      if (attempt === retries) throw err;
      const backoff = baseDelay * 2 ** (attempt - 1);
      logger.info(`Retrying Kafka connection in ${backoff}ms...`);
      await sleep(backoff);
    }
  }
}

export async function publishEvent(topic: string, message: any): Promise<void> {
  if (!producer) throw new Error('Kafka producer not initialized. Call initKafka() first.');

  const retries = Number(process.env.KAFKA_PRODUCE_RETRIES ?? 5);
  const baseDelay = Number(process.env.KAFKA_PRODUCE_DELAY_MS ?? 500);
  const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

  for (let attempt = 1; attempt <= retries; attempt++) {
    try {
      await producer.send({
        topic,
        messages: [
          { key: message.eventType ?? undefined, value: JSON.stringify(message) }
        ]
      });
      return;
    } catch (err) {
      logger.error(`Failed to publish event to ${topic} (attempt ${attempt}/${retries})`, err);
      if (attempt === retries) throw err;
      const backoff = baseDelay * 2 ** (attempt - 1);
      logger.info(`Retrying publish in ${backoff}ms...`);
      await sleep(backoff);
    }
  }
}

export function getKafkaConsumer() {
  if (!consumer) throw new Error('Kafka consumer not initialized. Call initKafka() first.');
  return consumer;
}

export function isKafkaReady(): boolean {
  return kafkaReady;
}

export async function disconnectKafka(): Promise<void> {
  kafkaReady = false;
  try {
    if (producer) await producer.disconnect();
    if (consumer) await consumer.disconnect();
    logger.info('Kafka connection closed');
  } catch (err) {
    logger.warn('Error while disconnecting Kafka', err);
  }
}

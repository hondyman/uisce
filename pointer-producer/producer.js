// pointer-producer.js
const AWS = require('aws-sdk');
const { Kafka } = require('kafkajs');
const fs = require('fs');

const s3 = new AWS.S3({
  endpoint: 'http://localhost:9000',
  s3ForcePathStyle: true,
  accessKeyId: 'minioadmin',
  secretAccessKey: 'minioadmin'
});

const kafka = new Kafka({ brokers: ['localhost:9092'] });
const producer = kafka.producer();

async function publishPointer(payload, table, op, id) {
  const key = `payloads/${table}/${id}-${Date.now()}.json`;
  await s3.putObject({ Bucket: 'data', Key: key, Body: JSON.stringify(payload) }).promise();

  const event = { op, table, id, object_path: `s3://data/${key}`, ts: new Date().toISOString() };
  await producer.connect();
  await producer.send({ topic: `events.${table}`, messages: [{ value: JSON.stringify(event) }] });
  await producer.disconnect();
}

module.exports = { publishPointer };

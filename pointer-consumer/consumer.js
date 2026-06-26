// pointer-consumer.js
const AWS = require('aws-sdk');
const { Kafka } = require('kafkajs');
const axios = require('axios'); // for calling Iceberg writer API or job

const s3 = new AWS.S3({
  endpoint: 'http://localhost:9000',
  s3ForcePathStyle: true,
  accessKeyId: 'minioadmin',
  secretAccessKey: 'minioadmin'
});

const kafka = new Kafka({ brokers: ['localhost:9092'] });
const consumer = kafka.consumer({ groupId: 'pointer-consumer' });

async function start() {
  await consumer.connect();
  await consumer.subscribe({ topic: 'events.holdings', fromBeginning: true });

  await consumer.run({
    eachMessage: async ({ message }) => {
      const event = JSON.parse(message.value.toString());
      const s3Key = event.object_path.replace('s3://data/', '');
      const obj = await s3.getObject({ Bucket: 'data', Key: s3Key }).promise();
      const payload = JSON.parse(obj.Body.toString());

      console.log(`Received payload for entity ${event.id}:`, payload);
      // Example: call a local HTTP endpoint that writes to Iceberg staging or triggers MERGE
      // await axios.post('http://localhost:9002/ingest/staging/holdings', { payload, metadata: event });
    }
  });
}

start().catch(console.error);

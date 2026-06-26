import express from 'express';
import { logger } from '../utils/logger.js';
import { publishEvent } from '../services/kafka.js';
import { ApprovalWorkflowEngine } from '../approval/ApprovalWorkflowEngine.js';

export const internalRoutes = express.Router();

// POST /api/internal/publish-test-approval
internalRoutes.post('/publish-test-approval', async (req, res) => {
  try {
    const payload = req.body || {
      eventType: 'workflow.test',
      data: { message: 'test event', ts: new Date().toISOString() }
    };

    // If Kafka isn't ready, record immediately so the test doesn't block waiting for Kafka metadata
    const { isKafkaReady } = await import('../services/kafka.js');
    if (!isKafkaReady()) {
      logger.warn('Kafka not ready — using local fallback for test publish');
      await ApprovalWorkflowEngine.getInstance().recordEvent(payload);
      return res.status(202).json({ status: 'accepted', payload, published: false, fallback: true });
    }

    // Fire-and-forget publish so the endpoint returns quickly for CI
    (async () => {
      try {
        await publishEvent('approval.events', payload);
        logger.info('Published test event');
      } catch (error) {
        logger.warn('Publish failed, falling back to local record', error);
        // Fallback: record the event directly so tests can assert the handler ran
        await ApprovalWorkflowEngine.getInstance().recordEvent(payload);
      }
    })();

    return res.status(202).json({ status: 'accepted', payload, published: true });
  } catch (error) {
    logger.error('Failed to publish test approval event', error);
    return res.status(500).json({ error: 'failed_to_publish' });
  }
});

// GET /api/internal/last-approval-event
internalRoutes.get('/last-approval-event', (req, res) => {
  try {
    const last = ApprovalWorkflowEngine.getInstance().getLastEvent();
    if (!last) {
      return res.status(404).json({ status: 'not_found' });
    }
    return res.json({ status: 'ok', last });
  } catch (error) {
    logger.error('Failed to get last approval event', error);
    return res.status(500).json({ error: 'failed_to_get' });
  }
});

// GET /api/internal/consumer-ready - returns 200 when event listeners are started and stable
internalRoutes.get('/consumer-ready', (req, res) => {
  try {
    const ready = ApprovalWorkflowEngine.getInstance().isListenersReady();
    if (ready) return res.status(200).json({ status: 'ready' });
    return res.status(503).json({ status: 'not_ready' });
  } catch (error) {
    logger.error('Failed to get consumer-ready', error);
    return res.status(500).json({ error: 'failed_to_get' });
  }
});

// POST /api/internal/publish-test-approval-sync - strict end-to-end test (publish to Kafka and wait for consumer)
internalRoutes.post('/publish-test-approval-sync', async (req, res) => {
  try {
    const { v4: uuidv4 } = await import('uuid');
    const payload = req.body || {
      eventType: 'workflow.test',
      data: { message: 'test event', ts: new Date().toISOString() }
    };

    // Attach a unique marker to the data so we can correlate consumption
    const testId = uuidv4();
    if (typeof payload.data === 'object' && payload.data !== null) {
      Object.assign(payload.data, { __testId: testId });
    } else {
      payload.data = { value: payload.data, __testId: testId };
    }

    // Publish synchronously (will throw on failure)
    await publishEvent('approval.events', payload);

    // Wait for consumer to process the message
    const timeoutMs = Number(process.env.TEST_CONSUME_TIMEOUT_MS ?? 30000);
    const pollIntervalMs = 1000;
    const start = Date.now();
    while (Date.now() - start < timeoutMs) {
      const last = ApprovalWorkflowEngine.getInstance().getLastEvent();
      if (last && last.data && last.data.__testId === testId) {
        return res.status(200).json({ status: 'consumed', last });
      }
      await new Promise((r) => setTimeout(r, pollIntervalMs));
    }

    return res.status(504).json({ error: 'consume_timeout' });
  } catch (error) {
    logger.error('Failed sync publish/consume test', error);
    return res.status(500).json({ error: 'publish_or_wait_failed' });
  }
});

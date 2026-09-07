-- Verification Script for Hybrid Integration
-- This script verifies that the necessary tables for the Exporter and Audit Log exist.

-- 1. Verify StarRocks Tables for Exporter Source
EXISTS TABLE trades_stream;
EXISTS TABLE compliance_events;

-- 2. Verify Audit Log Table (Destination for Exporter Logs)
EXISTS TABLE audit_log;

-- 3. Mock Audit Log Insertion (Simulate Exporter Logging)
INSERT INTO audit_log (event_time, trace_id, actor_id, action, resource_type, resource_id, payload)
VALUES (
    now(), 
    generateUUIDv4(), 
    'system-exporter', 
    'ExportIncremental', 
    'Table', 
    'trades_stream', 
    '{"rowCount": 1500, "snapshotID": "abc-123"}'
);

-- 4. Verify Audit Log Entry
SELECT * FROM audit_log WHERE actor_id = 'system-exporter' ORDER BY event_time DESC LIMIT 1;

import React from 'react';
import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import { registerDomainComponents } from '../../studio-core/components/registry';
import { lakehouseApi } from './api';
import { StreamingGatekeeperMonitor } from './components/StreamingGatekeeperMonitor';
import { LakekeeperCatalogManager } from './components/LakekeeperCatalogManager';
import { DLQInspectorPanel } from './components/DLQInspectorPanel';

const operations: OperationDef[] = [
  {
    id: 'lakehouse.overview',
    domain: 'lakehouse_streaming',
    kind: 'query',
    label: 'Get Lakehouse Overview',
    description: 'Fetches CDC stream loader health, Lakekeeper REST catalog status, and DLQ metrics.',
    params: [],
    fields: [
      { name: 'lakekeeper_status', type: 'string' },
      { name: 'total_loaded_today', type: 'number' },
      { name: 'total_dlq_blocked', type: 'number' },
      { name: 'loaders' },
    ],
    run: async () => {
      return lakehouseApi.getOverview();
    },
  },
  {
    id: 'lakehouse.listTables',
    domain: 'lakehouse_streaming',
    kind: 'query',
    label: 'List Iceberg Tables',
    description: 'Lists all registered Iceberg lakehouse tables with partition schemes.',
    params: [],
    fields: [
      { name: 'tables' },
    ],
    run: async () => {
      const tables = await lakehouseApi.listTables();
      return { tables, total: tables.length };
    },
  },
  {
    id: 'lakehouse.listDLQ',
    domain: 'lakehouse_streaming',
    kind: 'query',
    label: 'List DLQ Records',
    description: 'Retrieves rejected stream events and rule block violations.',
    params: [
      { name: 'limit', type: 'number', required: false },
    ],
    fields: [
      { name: 'records' },
    ],
    run: async (params) => {
      const limit = params.limit ? Number(params.limit) : 50;
      const records = await lakehouseApi.listDLQ(limit);
      return { records, total: records.length };
    },
  },
];

registerOperations(operations);

registerDomainComponents([
  {
    id: 'lakehouse.StreamingGatekeeperMonitor',
    domain: 'lakehouse_streaming',
    label: 'Streaming Gatekeeper & Ingestion Monitor',
    description: 'Real-time telemetry on Debezium CDC ingestion, tenant assertion, and in-line rule validation.',
    inputs: [],
    events: [],
    render: () => <StreamingGatekeeperMonitor />,
  },
  {
    id: 'lakehouse.LakekeeperCatalogManager',
    domain: 'lakehouse_streaming',
    label: 'Lakekeeper Iceberg REST Catalog Manager',
    description: 'Apache Iceberg namespace, table schema, partition scheme, and MinIO storage manager.',
    inputs: [],
    events: [],
    render: () => <LakekeeperCatalogManager />,
  },
  {
    id: 'lakehouse.DLQInspectorPanel',
    domain: 'lakehouse_streaming',
    label: 'Dead-Letter Queue (DLQ) Inspector',
    description: 'Inspection and replay console for stream records blocked by tenant or rule assertions.',
    inputs: [],
    events: [],
    render: () => <DLQInspectorPanel />,
  },
]);

import { Node, Edge } from 'reactflow';
import { PipelineNodeData } from '../types/pipeline';

export interface PipelineTemplate {
  id: string;
  name: string;
  description: string;
  mode: 'business_object' | 'catalog_graph' | 'external';
  category: string;
  targetEntity: string;
  concurrency: number;
  batchSize: number;
  nodes: Node<PipelineNodeData>[];
  edges: Edge[];
}

export const PIPELINE_TEMPLATES: PipelineTemplate[] = [
  {
    id: 'full-enterprise-api-workflow-crud',
    name: 'Full Pipeline: API Builder + BO CRUD + Flow Builder Approval',
    description: 'Complete data pipeline invoking API Studio KYC verification, executing Business Object CRUD updates, and triggering Flow Builder approval workflows.',
    mode: 'business_object',
    category: 'Full Enterprise Pipeline',
    targetEntity: 'oms.trade_order',
    concurrency: 8,
    batchSize: 2000,
    nodes: [
      {
        id: 'node-1',
        type: 'pipelineTile',
        position: { x: 50, y: 150 },
        data: {
          label: 'Raw Trade Orders Feed',
          category: 'source',
          subType: 'raw_json',
          icon: 'FileText',
          badge: 'Feed',
          description: 'Streaming CSV/JSON block trades from counterparty broker',
          config: { sourceType: 'raw_json' },
        },
      },
      {
        id: 'node-2',
        type: 'pipelineTile',
        position: { x: 380, y: 150 },
        data: {
          label: 'Column Normalizer & Cast',
          category: 'transform',
          subType: 'column_mapper',
          icon: 'ArrowRightLeft',
          badge: 'Mapper',
          description: 'Maps raw trade fields to canonical OMS schema types',
          config: {
            mappings: {
              account_number: 'ext_acc_num',
              quantity: 'qty',
              price: 'px',
              subtype_code: 'stype',
            },
            types: {
              quantity: 'float',
              price: 'float',
            },
          },
        },
      },
      {
        id: 'node-3',
        type: 'pipelineTile',
        position: { x: 710, y: 150 },
        data: {
          label: 'API Builder: KYC & Sanctions',
          category: 'transform',
          subType: 'api_caller',
          icon: 'Code2',
          badge: 'API Studio',
          description: 'Invokes /api/v1/customers/verify-kyc created in API Designer',
          config: {
            endpoint_url: '/api/v1/customers/verify-kyc',
            method: 'POST',
            merge_output: true,
          },
        },
      },
      {
        id: 'node-4',
        type: 'pipelineTile',
        position: { x: 1040, y: 150 },
        data: {
          label: 'BO CRUD: Update Account',
          category: 'transform',
          subType: 'bo_crud',
          icon: 'Edit3',
          badge: 'BO CRUD',
          description: 'Updates oms.account status and balances directly in STI table',
          config: {
            table: 'oms.account',
            operation: 'UPDATE',
          },
        },
      },
      {
        id: 'node-5',
        type: 'pipelineTile',
        position: { x: 1370, y: 150 },
        data: {
          label: 'Flow Builder: Settlement Approval',
          category: 'transform',
          subType: 'workflow_caller',
          icon: 'Workflow',
          badge: 'Flow Builder',
          description: 'Dispatches Flow Builder 4-eyes approval workflow for block trade',
          config: {
            workflow_name: 'Trade Reconciliation & Settlement Approval',
            workflow_id: 'wf-1',
            mode: 'sync',
          },
        },
      },
      {
        id: 'node-6',
        type: 'pipelineTile',
        position: { x: 1700, y: 150 },
        data: {
          label: 'Uuisce Trade Orders Loader',
          category: 'loader',
          subType: 'bo_loader',
          icon: 'Database',
          badge: 'Mode 1 (STI)',
          description: 'Parallel COPY bulk upsert into oms.trade_order with bitemporal history',
          config: {
            loaderType: 'bo_loader',
            table: 'oms.trade_order',
          },
        },
      },
    ],
    edges: [
      { id: 'e1-2', source: 'node-1', target: 'node-2', animated: true, style: { stroke: '#3b82f6', strokeWidth: 2 } },
      { id: 'e2-3', source: 'node-2', target: 'node-3', animated: true, style: { stroke: '#0ea5e9', strokeWidth: 2 } },
      { id: 'e3-4', source: 'node-3', target: 'node-4', animated: true, style: { stroke: '#f97316', strokeWidth: 2 } },
      { id: 'e4-5', source: 'node-4', target: 'node-5', animated: true, style: { stroke: '#ec4899', strokeWidth: 2 } },
      { id: 'e5-6', source: 'node-5', target: 'node-6', animated: true, style: { stroke: '#10b981', strokeWidth: 2 } },
    ],
  },
  {
    id: 'informatica-trade-account-loader',
    name: 'Informatica Replacement: Trade Order & Account Bulk Loader',
    description: 'High-throughput parallel bulk ingestion pipeline that extracts, validates STI subtype allowlists, and loads millions of institutional trade records into Uuisce.',
    mode: 'business_object',
    category: 'Informatica / Talend Alternative',
    targetEntity: 'oms.trade_order',
    concurrency: 16,
    batchSize: 5000,
    nodes: [
      {
        id: 'node-1',
        type: 'pipelineTile',
        position: { x: 50, y: 150 },
        data: {
          label: 'Trade Stream Ingest Feed',
          category: 'source',
          subType: 'raw_json',
          icon: 'FileText',
          badge: 'Feed',
          description: 'Streaming batch trades from FIX engine drop-copy',
          config: { sourceType: 'raw_json' },
        },
      },
      {
        id: 'node-2',
        type: 'pipelineTile',
        position: { x: 380, y: 150 },
        data: {
          label: 'Column Mapper & Cast',
          category: 'transform',
          subType: 'column_mapper',
          icon: 'ArrowRightLeft',
          badge: 'Mapper',
          description: 'Renames incoming column keys and casts data types',
          config: {
            mappings: {
              account_number: 'ext_acc_num',
              quantity: 'qty',
              price: 'px',
              subtype_code: 'stype',
            },
            types: {
              quantity: 'float',
              price: 'float',
            },
          },
        },
      },
      {
        id: 'node-3',
        type: 'pipelineTile',
        position: { x: 710, y: 150 },
        data: {
          label: 'Subtype Allowlist Enforcer',
          category: 'validator',
          subType: 'subtype_allowlist',
          icon: 'ShieldCheck',
          badge: 'Rule 1 Invariant',
          description: 'Enforces STI subtype allowlist rules defined in oms.subtype_registry',
          config: { root_object: 'account' },
        },
      },
      {
        id: 'node-4',
        type: 'pipelineTile',
        position: { x: 1040, y: 150 },
        data: {
          label: 'Uuisce Trade Orders Loader',
          category: 'loader',
          subType: 'bo_loader',
          icon: 'Database',
          badge: 'Mode 1 (STI)',
          description: 'Parallel bulk upsert into oms.trade_order with bitemporal history',
          config: {
            loaderType: 'bo_loader',
            table: 'oms.trade_order',
          },
        },
      },
    ],
    edges: [
      { id: 'e1-2', source: 'node-1', target: 'node-2', animated: true, style: { stroke: '#3b82f6', strokeWidth: 2 } },
      { id: 'e2-3', source: 'node-2', target: 'node-3', animated: true, style: { stroke: '#8b5cf6', strokeWidth: 2 } },
      { id: 'e3-4', source: 'node-3', target: 'node-4', animated: true, style: { stroke: '#10b981', strokeWidth: 2 } },
    ],
  },
  {
    id: 'catalog-schema-introspector',
    name: 'Catalog Graph: Physical Schema Introspection & Edge Builder',
    description: 'Extracts physical table & column metadata, transforms them into hierarchical catalog_node structures (TABLE, ATTRIBUTE), and constructs relationship edges.',
    mode: 'catalog_graph',
    category: 'Data Catalog Graph Pipeline',
    targetEntity: 'catalog_node',
    concurrency: 8,
    batchSize: 2000,
    nodes: [
      {
        id: 'node-1',
        type: 'pipelineTile',
        position: { x: 50, y: 150 },
        data: {
          label: 'Catalog Graph Reader',
          category: 'source',
          subType: 'catalog_reader',
          icon: 'Share2',
          badge: 'Mode 2 (Graph)',
          description: 'Reads schema metadata definitions from catalog graph',
          config: { sourceType: 'catalog_reader', catalog_type: 'TABLE' },
        },
      },
      {
        id: 'node-2',
        type: 'pipelineTile',
        position: { x: 380, y: 150 },
        data: {
          label: 'Graph Node & Edge Synthesizer',
          category: 'graph_synthesizer',
          subType: 'graph_synthesizer',
          icon: 'Network',
          badge: 'Mode 2 (Graph)',
          description: 'Synthesizes TABLE, ATTRIBUTE nodes and COLUMN_OF edges',
          config: {
            parent_field: 'table_name',
            child_field: 'column_name',
            data_type_field: 'data_type',
            edge_predicate: 'COLUMN_OF',
          },
        },
      },
      {
        id: 'node-3',
        type: 'pipelineTile',
        position: { x: 710, y: 150 },
        data: {
          label: 'Uuisce Catalog Graph Ingestor',
          category: 'loader',
          subType: 'catalog_loader',
          icon: 'Layers',
          badge: 'Mode 2 (Graph)',
          description: 'Parallel upsert into catalog_node and catalog_edge respecting Gold Copy delta rules',
          config: { loaderType: 'catalog_loader' },
        },
      },
    ],
    edges: [
      { id: 'e1-2', source: 'node-1', target: 'node-2', animated: true, style: { stroke: '#3b82f6', strokeWidth: 2 } },
      { id: 'e2-3', source: 'node-2', target: 'node-3', animated: true, style: { stroke: '#06b6d4', strokeWidth: 2 } },
    ],
  },
  {
    id: 'bloomberg-fields-csv-sync',
    name: 'Bloomberg Fields: bb_fields.csv Ingestion & Catalog Sync Pipeline',
    description: 'Ingests Bloomberg Data License fields dictionary (bb_fields.csv), maps mnemonics and market sector eligibility flags into BLOOMBERG_FIELD catalog nodes, and syncs them continuously.',
    mode: 'catalog_graph',
    category: 'Market Data Dictionary Sync',
    targetEntity: 'catalog_node',
    concurrency: 16,
    batchSize: 2000,
    nodes: [
      {
        id: 'node-1',
        type: 'pipelineTile',
        position: { x: 50, y: 150 },
        data: {
          label: 'bb_fields.csv File / Stream Feed',
          category: 'source',
          subType: 'raw_json',
          icon: 'FileText',
          badge: 'CSV Feed',
          description: 'Reads raw Bloomberg Data License fields CSV dictionary feed',
          config: {
            sourceType: 'raw_json',
            raw_data: [
              {
                FieldID: 'DS62 ',
                FieldMnemonic: '144A_FLAG',
                Description: 'Is 144A Eligible',
                DataLicenseCategory: 'Security Master',
                Category: 'Descriptive Info',
                Definition: 'Indicates if the security is eligible for trading exemption under rule 144a. Returns a Y or N.',
                Equity: 'Equity',
                Corp: 'Corp',
                Mtge: 'Mtge',
                StandardWidth: 4,
                StandardDecimalPlaces: 0,
                FieldType: 'Boolean',
                ProductionDate: '19980617',
                CurrentMaximumWidth: 30,
                HeldSecurities: 'True',
                HeldSecuritiesOrder: 110,
              },
              {
                FieldID: 'SP13 ',
                FieldMnemonic: '10Y_ASK_CDS_SPREAD',
                Description: '10 Year Ask Par CDS Spread on Reference Name',
                DataLicenseCategory: 'Derived Data',
                Category: 'Analytics - Risk Measures',
                Definition: 'Returns the 10 year credit default swap ask spread.',
                Pfd: 'Pfd',
                Govt: 'Govt',
                Corp: 'Corp',
                StandardWidth: 10,
                StandardDecimalPlaces: 4,
                FieldType: 'Real',
                ProductionDate: '20030616',
                CurrentMaximumWidth: 30,
                HeldSecurities: 'False',
                HeldSecuritiesOrder: 1,
              },
            ],
          },
        },
      },
      {
        id: 'node-2',
        type: 'pipelineTile',
        position: { x: 420, y: 150 },
        data: {
          label: 'Bloomberg Fields Dictionary Mapper',
          category: 'transform',
          subType: 'bloomberg_field_mapper',
          icon: 'LineChart',
          badge: 'Data License',
          description: 'Maps bb_fields.csv into BLOOMBERG_FIELD catalog nodes with JSON properties & market sector eligibility',
          config: {
            category_prefix: 'bloomberg.fields',
          },
        },
      },
      {
        id: 'node-3',
        type: 'pipelineTile',
        position: { x: 790, y: 150 },
        data: {
          label: 'Catalog Graph Ingestor (Load / Sync)',
          category: 'loader',
          subType: 'catalog_loader',
          icon: 'Layers',
          badge: 'Mode 2 (Graph)',
          description: 'Parallel bulk upsert into catalog_node respecting Gold Copy / tenant delta overlays',
          config: {
            loaderType: 'catalog_loader',
          },
        },
      },
    ],
    edges: [
      { id: 'e1-2', source: 'node-1', target: 'node-2', animated: true, style: { stroke: '#ff6d00', strokeWidth: 2 } },
      { id: 'e2-3', source: 'node-2', target: 'node-3', animated: true, style: { stroke: '#06b6d4', strokeWidth: 2 } },
    ],
  },
];

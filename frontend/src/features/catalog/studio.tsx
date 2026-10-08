import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import { apiFetch } from '../../lib/apiClient';
import { abbreviationApiClient } from '../../utils/abbreviationApi';

const DOMAIN = 'catalog';

const operations: OperationDef[] = [
  {
    id: `${DOMAIN}.listNodeTypes`,
    domain: DOMAIN,
    kind: 'query',
    label: 'List node types',
    description: 'List all graph node types in the catalog.',
    params: [{ name: 'search', type: 'string' }],
    fields: [
      { name: 'id', label: 'ID', type: 'string' },
      { name: 'catalog_type_name', label: 'Type Name', type: 'string' },
      { name: 'type', label: 'Origin', type: 'string' },
      { name: 'description', label: 'Description', type: 'string' },
      { name: 'is_active', label: 'Active', type: 'boolean' },
    ],
    run: async (params) => {
      const q = typeof params?.search === 'string' ? params.search.trim() : '';
      const queryParams = new URLSearchParams();
      if (q) queryParams.append('q', q);
      const res = await apiFetch(`/api/node-types?${queryParams.toString()}`);
      if (!res.ok) throw new Error('Failed to list node types');
      const data = await res.json();
      const rows = Array.isArray(data) ? data : data?.data || [];
      return rows.map((r: any) => ({
        id: r.id,
        catalog_type_name: r.catalog_type_name,
        type: r.type || (r.catalog_type_name?.startsWith('CDM') || ['SemanticTerm', 'Metric', 'Report'].includes(r.catalog_type_name) ? 'core' : 'custom'),
        description: r.description || '',
        is_active: r.is_active !== false,
      }));
    },
  },
  {
    id: `${DOMAIN}.createNodeType`,
    domain: DOMAIN,
    kind: 'mutation',
    label: 'Create node type',
    description: 'Create a custom catalog node type.',
    params: [
      { name: 'catalog_type_name', label: 'Type Name', type: 'string', required: true },
      { name: 'description', label: 'Description', type: 'string' },
    ],
    run: async (params) => {
      const res = await apiFetch('/api/node-types', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          catalog_type_name: params.catalog_type_name,
          description: params.description || '',
          is_active: true,
          type: 'custom',
        }),
      });
      if (!res.ok) throw new Error('Failed to create node type');
      return res.json();
    },
    invalidates: [[DOMAIN]],
  },
  {
    id: `${DOMAIN}.deleteNodeType`,
    domain: DOMAIN,
    kind: 'mutation',
    label: 'Delete node type',
    description: 'Delete a custom catalog node type.',
    params: [{ name: 'id', label: 'ID', type: 'string', required: true }],
    run: async (params) => {
      const res = await apiFetch(`/api/node-types/${params.id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to delete node type');
      return { success: true };
    },
    invalidates: [[DOMAIN]],
  },
  {
    id: `${DOMAIN}.listEdgeTypes`,
    domain: DOMAIN,
    kind: 'query',
    label: 'List edge types',
    description: 'List all graph edge types in the catalog.',
    params: [{ name: 'search', type: 'string' }],
    fields: [
      { name: 'id', label: 'ID', type: 'string' },
      { name: 'edge_type_name', label: 'Edge Type Name', type: 'string' },
      { name: 'type', label: 'Origin', type: 'string' },
      { name: 'description', label: 'Description', type: 'string' },
      { name: 'is_active', label: 'Active', type: 'boolean' },
    ],
    run: async (params) => {
      const q = typeof params?.search === 'string' ? params.search.trim() : '';
      const url = q ? `/api/edge-types?q=${encodeURIComponent(q)}` : '/api/edge-types';
      const res = await apiFetch(url);
      if (!res.ok) throw new Error('Failed to list edge types');
      const rows = await res.json();
      return (Array.isArray(rows) ? rows : []).map((r: any) => ({
        id: r.id,
        edge_type_name: r.edge_type_name,
        type: r.type || (r.is_system ? 'core' : 'custom'),
        description: r.description || '',
        is_active: r.is_active !== false,
      }));
    },
  },
  {
    id: `${DOMAIN}.createEdgeType`,
    domain: DOMAIN,
    kind: 'mutation',
    label: 'Create edge type',
    description: 'Create a custom catalog edge type.',
    params: [
      { name: 'edge_type_name', label: 'Edge Type Name', type: 'string', required: true },
      { name: 'description', label: 'Description', type: 'string' },
    ],
    run: async (params) => {
      const res = await apiFetch('/api/edge-types', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          edge_type_name: params.edge_type_name,
          description: params.description || '',
          is_active: true,
        }),
      });
      if (!res.ok) throw new Error('Failed to create edge type');
      return res.json();
    },
    invalidates: [[DOMAIN]],
  },
  {
    id: `${DOMAIN}.deleteEdgeType`,
    domain: DOMAIN,
    kind: 'mutation',
    label: 'Delete edge type',
    description: 'Delete a custom catalog edge type.',
    params: [{ name: 'id', label: 'ID', type: 'string', required: true }],
    run: async (params) => {
      const res = await apiFetch(`/api/edge-types/${params.id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to delete edge type');
      return { success: true };
    },
    invalidates: [[DOMAIN]],
  },
  {
    id: `${DOMAIN}.listAbbreviations`,
    domain: DOMAIN,
    kind: 'query',
    label: 'List abbreviations',
    description: 'List business abbreviations and their expansions.',
    params: [{ name: 'search', type: 'string' }],
    fields: [
      { name: 'id', label: 'ID', type: 'string' },
      { name: 'abbreviation', label: 'Abbreviation', type: 'string' },
      { name: 'expansion', label: 'Expansion', type: 'string' },
      { name: 'domain', label: 'Domain', type: 'string' },
      { name: 'is_active', label: 'Active', type: 'boolean' },
    ],
    run: async (params) => {
      const q = typeof params?.search === 'string' ? params.search.trim() : '';
      const res = await abbreviationApiClient.getAbbreviations(100, 0, q);
      return (res.items || []).map((r: any) => ({
        id: String(r.id),
        abbreviation: r.abbreviation,
        expansion: r.full_word || r.expansion,
        domain: r.notes || (r.is_core ? 'Core' : 'Custom'),
        is_active: true,
      }));
    },
  },
  {
    id: `${DOMAIN}.createAbbreviation`,
    domain: DOMAIN,
    kind: 'mutation',
    label: 'Create abbreviation',
    description: 'Create a new business abbreviation.',
    params: [
      { name: 'abbreviation', label: 'Abbreviation', type: 'string', required: true },
      { name: 'expansion', label: 'Expansion', type: 'string', required: true },
      { name: 'notes', label: 'Notes', type: 'string' },
    ],
    run: async (params) => {
      await abbreviationApiClient.addAbbreviation(
        String(params.abbreviation),
        String(params.expansion),
        params.notes ? String(params.notes) : undefined,
      );
      return { success: true };
    },
    invalidates: [[DOMAIN]],
  },
];

registerOperations(operations);

import React from 'react';
import { registerDomainComponents } from '../../studio-core/components/registry';
import GlossaryExplorer from '../glossary/GlossaryExplorer';
import BusinessTermsExplorer from '../glossary/BusinessTermsExplorer';
import ApiInventoryPage from './pages/ApiInventoryPage';
import EntityPickerPage from '../custom-attributes/pages/EntityPickerPage';
import DomainsManagementPage from '../core/pages/DomainsManagementPage';
import SchemaExplorerPage from '../schema-explorer/pages/SchemaExplorer';
import SemanticMapperPage from '../core/pages/SemanticMapperPage';
import { AIBusinessTermSuggestionsPage } from '../../pages/catalog/AIBusinessTermSuggestionsPage';

registerDomainComponents([
  {
    id: 'catalog.ApiInventory',
    domain: DOMAIN,
    label: 'API Inventory',
    description: 'Catalog and inspect endpoints, payloads, schema contracts, and authentication.',
    inputs: [],
    events: [],
    render: () => <ApiInventoryPage />,
  },
  {
    id: 'catalog.GlossaryExplorer',
    domain: DOMAIN,
    label: 'Glossary & Semantic Terms Explorer',
    description: 'Explore business glossary, semantic terms, graph relationships, and classifications.',
    inputs: [],
    events: [],
    render: () => <GlossaryExplorer />,
  },
  {
    id: 'catalog.BusinessTerms',
    domain: DOMAIN,
    label: 'Business Terms Explorer',
    description: 'Governed business taxonomy and terminology explorer.',
    inputs: [],
    events: [],
    render: () => <BusinessTermsExplorer />,
  },
  {
    id: 'catalog.CustomFields',
    domain: DOMAIN,
    label: 'Custom Fields & Attributes',
    description: 'Manage dynamic custom attributes and entity extensions.',
    inputs: [],
    events: [],
    render: () => <EntityPickerPage />,
  },
  {
    id: 'catalog.DomainsManagement',
    domain: DOMAIN,
    label: 'Data Domains Management',
    description: 'Business data domains, ownership boundaries, and semantic scope.',
    inputs: [],
    events: [],
    render: () => <DomainsManagementPage />,
  },
  {
    id: 'catalog.SchemaExplorer',
    domain: DOMAIN,
    label: 'Physical Schema Explorer',
    description: 'Deep introspection of underlying database tables, views, and columns.',
    inputs: [],
    events: [],
    render: () => <SchemaExplorerPage />,
  },
  {
    id: 'catalog.SemanticMapper',
    domain: DOMAIN,
    label: 'Intelligent Semantic Mapper',
    description: 'Map physical columns and upstream assets to canonical business concepts.',
    inputs: [],
    events: [],
    render: () => <SemanticMapperPage />,
  },
  {
    id: 'catalog.AISuggestions',
    domain: DOMAIN,
    label: 'AI Business Term Suggestions',
    description: 'AI-assisted terminology recommendations, match confidence, and review workflow.',
    inputs: [],
    events: [],
    render: () => <AIBusinessTermSuggestionsPage />,
  },
]);

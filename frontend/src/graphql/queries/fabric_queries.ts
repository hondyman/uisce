// fabric_queries.ts
// Apollo-ready TypeScript documents and types for querying the semantic layer.
// Assumes Hasura metadata from our previous step (tables, view, relationships, function).

import { gql, type TypedDocumentNode } from '@apollo/client';
import type { JSONValue } from '../../types/json';

// ---------- Shared scalars and enums (GraphQL -> TS) ----------
export type UUID = string;
export type Timestamptz = string;

export type FabricStatus = 'draft' | 'published' | 'archived';
export type JoinRelationship = 'one_to_one' | 'one_to_many' | 'many_to_one' | 'many_to_many';

// ---------- Fragments ----------
export const FABRIC_DEFN_LIST_FIELDS = gql`
  fragment FabricDefnListFields on fabric_defn {
    id
    model_key
    version
    status
    is_current
    title
    created_at
    published_at
  }
`;

export const FABRIC_DEFN_CORE_FIELDS = gql`
  fragment FabricDefnCoreFields on fabric_defn {
    id
    tenant_id
    model_key
    version
    status
    is_current
    title
    description
    source_config
    resolved_config
    created_by
    created_at
    published_at
  }
`;

export const FABRIC_INDEX_ITEM_FIELDS = gql`
  fragment FabricIndexItemFields on fabric_defn_index {
    kind
    name
    type
    relationship
    title
    description
    sql
    version
  }
`;

export const FABRIC_AUDIT_FIELDS = gql`
  fragment FabricAuditFields on fabric_defn_audit {
    audit_id
    action
    at
    actor_id
    before_doc
    after_doc
  }
`;

// ---------- 1) Get current published model by key ----------
export interface GetCurrentModelVariables {
  model_key: string;
}
export interface GetCurrentModelData {
  fabric_defn_current: Array<{
    id: UUID;
    model_key: string;
    version: number;
    title: string | null;
    description: string | null;
    resolved_config: JSONValue;
    published_at: Timestamptz | null;
    created_by: UUID;
  }>;
}
export const GET_CURRENT_MODEL: TypedDocumentNode<
  GetCurrentModelData,
  GetCurrentModelVariables
> = gql`
  query GetCurrentModel($model_key: String!) {
    fabric_defn_current(where: { model_key: { _eq: $model_key } }) {
      id
      model_key
      version
      title
      description
      resolved_config
      published_at
      created_by
    }
  }
` as TypedDocumentNode<GetCurrentModelData, GetCurrentModelVariables>;

// ---------- 2) List definitions (paginated) ----------
export interface ListDefinitionsVariables {
  limit?: number | null;
  offset?: number | null;
}
export interface ListDefinitionsData {
  fabric_defn: Array<{
    id: UUID;
    model_key: string;
    version: number;
    status: FabricStatus;
    is_current: boolean;
    title: string | null;
    created_at: Timestamptz;
    published_at: Timestamptz | null;
  }>;
}
export const LIST_DEFINITIONS: TypedDocumentNode<
  ListDefinitionsData,
  ListDefinitionsVariables
> = gql`
  ${FABRIC_DEFN_LIST_FIELDS}
  query ListDefinitions($limit: Int = 20, $offset: Int = 0) {
    fabric_defn(order_by: [{ created_at: desc }], limit: $limit, offset: $offset) {
      ...FabricDefnListFields
    }
  }
` as TypedDocumentNode<ListDefinitionsData, ListDefinitionsVariables>;

// ---------- 3) Get full definition by ID (with index items) ----------
export interface GetDefinitionByIdVariables {
  id: UUID;
}
export interface GetDefinitionByIdData {
  fabric_defn_by_pk: ({
    index_items: Array<{
      kind: 'dimension' | 'measure' | 'join';
      name: string;
      type: string | null;
      relationship: JoinRelationship | null;
      title: string | null;
      description: string | null;
      sql: string | null;
      version: number;
    }>;
  } & {
    id: UUID;
    tenant_id: UUID;
    model_key: string;
    version: number;
    status: FabricStatus;
    is_current: boolean;
    title: string | null;
    description: string | null;
    source_config: JSONValue;
    resolved_config: JSONValue;
    created_by: UUID;
    created_at: Timestamptz;
    published_at: Timestamptz | null;
  }) | null;
}
export const GET_DEFINITION_BY_ID: TypedDocumentNode<
  GetDefinitionByIdData,
  GetDefinitionByIdVariables
> = gql`
  ${FABRIC_DEFN_CORE_FIELDS}
  ${FABRIC_INDEX_ITEM_FIELDS}
  query GetDefinitionById($id: uuid!) {
    fabric_defn_by_pk(id: $id) {
      ...FabricDefnCoreFields
      index_items {
        ...FabricIndexItemFields
      }
    }
  }
` as TypedDocumentNode<GetDefinitionByIdData, GetDefinitionByIdVariables>;

// ---------- 4) Search index entries ----------
export interface SearchIndexVariables {
  model_key: string;
  kinds?: string[] | null; // e.g., ["dimension","measure","join"]
  q: string; // use %term% from caller if you want contains
}
export interface SearchIndexData {
  fabric_defn_index: Array<{
    kind: 'dimension' | 'measure' | 'join';
    name: string;
    type: string | null;
    relationship: JoinRelationship | null;
    title: string | null;
    description: string | null;
    sql: string | null;
    version: number;
  }>;
}
export const SEARCH_INDEX: TypedDocumentNode<
  SearchIndexData,
  SearchIndexVariables
> = gql`
  ${FABRIC_INDEX_ITEM_FIELDS}
  query SearchIndex($model_key: String!, $kinds: [String!], $q: String!) {
    fabric_defn_index(
      where: {
        model_key: { _eq: $model_key }
        kind: { _in: $kinds }
        name: { _ilike: $q }
      }
      order_by: [{ kind: asc }, { name: asc }]
    ) {
      ...FabricIndexItemFields
    }
  }
` as TypedDocumentNode<SearchIndexData, SearchIndexVariables>;

// ---------- 5) Get audit log for a definition ----------
export interface GetDefinitionAuditVariables {
  defn_id: UUID;
}
export interface GetDefinitionAuditData {
  fabric_defn_audit: Array<{
    audit_id: number;
    action: 'create' | 'update' | 'publish' | 'archive';
    at: Timestamptz;
    actor_id: UUID;
    before_doc: JSONValue | null;
    after_doc: JSONValue | null;
  }>;
}
export const GET_DEFINITION_AUDIT: TypedDocumentNode<
  GetDefinitionAuditData,
  GetDefinitionAuditVariables
> = gql`
  ${FABRIC_AUDIT_FIELDS}
  query GetDefinitionAudit($defn_id: uuid!) {
    fabric_defn_audit(where: { defn_id: { _eq: $defn_id } }, order_by: { at: desc }) {
      ...FabricAuditFields
    }
  }
` as TypedDocumentNode<GetDefinitionAuditData, GetDefinitionAuditVariables>;

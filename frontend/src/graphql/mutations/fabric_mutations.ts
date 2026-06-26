// fabric_mutations.ts
// Apollo-ready TypeScript documents and types for mutations on the semantic layer.

import { gql, type TypedDocumentNode } from '@apollo/client';
import type { UUID, FabricStatus, Timestamptz } from '../queries/fabric_queries';
import type { JSONValue } from '../../types/json';


// ---------- Input helpers (mirror Hasura input shapes you use) ----------
// Hasura exposes insert/update input types in GraphQL, but we define TS aliases
// for client-side type safety without codegen. Tighten as needed.

export interface FabricDefnInsertInput {
  model_key: string;
  version: number;
  title?: string | null;
  description?: string | null;
  source_config: JSONValue;
  resolved_config: JSONValue;
  checksum_sha256?: string | null; // Hasura's bytea scalar is usually serialized as base64 string
}

export interface FabricDefnSetInput {
  title?: string | null;
  description?: string | null;
  source_config?: JSONValue;
  resolved_config?: JSONValue;
  checksum_sha256?: string | null;
}

// ---------- 1) Create a new draft ----------
export interface CreateDraftVariables {
  input: FabricDefnInsertInput; // corresponds to fabric_defn_insert_input
}
export interface CreateDraftData {
  insert_fabric_defn_one: {
    id: UUID;
    tenant_id: UUID;
    model_key: string;
    version: number;
    status: FabricStatus;
    created_by: UUID;
    created_at: Timestamptz;
  } | null;
}
export const CREATE_DRAFT: TypedDocumentNode<
  CreateDraftData,
  CreateDraftVariables
> = gql`
  mutation CreateDraft($input: fabric_defn_insert_input!) {
    insert_fabric_defn_one(object: $input) {
      id
      tenant_id
      model_key
      version
      status
      created_by
      created_at
    }
  }
` as TypedDocumentNode<CreateDraftData, CreateDraftVariables>;

// ---------- 2) Update a draft (only allowed for creator) ----------
export interface UpdateDraftVariables {
  id: UUID;
  patch: FabricDefnSetInput; // corresponds to fabric_defn_set_input
}
export interface UpdateDraftData {
  update_fabric_defn_by_pk: {
    id: UUID;
    version: number;
    title: string | null;
    description: string | null;
    resolved_config: JSONValue;
  } | null;
}
export const UPDATE_DRAFT: TypedDocumentNode<
  UpdateDraftData,
  UpdateDraftVariables
> = gql`
  mutation UpdateDraft($id: uuid!, $patch: fabric_defn_set_input!) {
    update_fabric_defn_by_pk(pk_columns: { id: $id }, _set: $patch) {
      id
      version
      title
      description
      resolved_config
    }
  }
` as TypedDocumentNode<UpdateDraftData, UpdateDraftVariables>;

// ---------- 3) Publish a definition (tracked SQL function) ----------
export interface PublishDefinitionVariables {
  defn_id: UUID;
  actor_id: UUID;
}
export interface PublishDefinitionData {
  publish_fabric_defn: {
    id: UUID;
    model_key: string;
    version: number;
    status: FabricStatus;
    is_current: boolean;
    published_at: Timestamptz | null;
  };
}
export const PUBLISH_DEFINITION: TypedDocumentNode<
  PublishDefinitionData,
  PublishDefinitionVariables
> = gql`
  mutation PublishDefinition($defn_id: uuid!, $actor_id: uuid!) {
    publish_fabric_defn(args: { p_defn_id: $defn_id, p_actor_id: $actor_id }) {
      id
      model_key
      version
      status
      is_current
      published_at
    }
  }
` as TypedDocumentNode<PublishDefinitionData, PublishDefinitionVariables>;

// ---------- 4) Delete a draft ----------
export interface DeleteDraftVariables {
  id: UUID;
}
export interface DeleteDraftData {
  delete_fabric_defn_by_pk: {
    id: UUID;
    model_key: string;
    version: number;
  } | null;
}
export const DELETE_DRAFT: TypedDocumentNode<
  DeleteDraftData,
  DeleteDraftVariables
> = gql`
  mutation DeleteDraft($id: uuid!) {
    delete_fabric_defn_by_pk(id: $id) {
      id
      model_key
      version
    }
  }
` as TypedDocumentNode<DeleteDraftData, DeleteDraftVariables>;

// ---------- (Optional) Archive instead of delete ----------
// Requires you to expose `archive_fabric_defn(p_defn_id uuid, p_actor_id uuid)` from Postgres
// and track it in Hasura metadata as a mutation.
/*
export interface ArchiveDefinitionVariables {
  defn_id: UUID;
  actor_id: UUID;
}
export interface ArchiveDefinitionData {
  archive_fabric_defn: {
    id: UUID;
    model_key: string;
    version: number;
    status: FabricStatus; // 'archived'
  };
}
export const ARCHIVE_DEFINITION: TypedDocumentNode<
  ArchiveDefinitionData,
  ArchiveDefinitionVariables
> = gql`
  mutation ArchiveDefinition($defn_id: uuid!, $actor_id: uuid!) {
    archive_fabric_defn(args: { p_defn_id: $defn_id, p_actor_id: $actor_id }) {
      id
      model_key
      version
      status
    }
  }
` as TypedDocumentNode<ArchiveDefinitionData, ArchiveDefinitionVariables>;
*/

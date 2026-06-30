// fabric_mutations.ts
// Apollo-ready TypeScript documents and types for mutations on the semantic layer.
// 2026-06-30 partial-prune (Phase 0.4): kept only CREATE_DRAFT and its dependencies.
// UPDATE_DRAFT, PUBLISH_DEFINITION, DELETE_DRAFT, and the commented ARCHIVE_DEFINITION
// all had zero external importers — see docs/HASURA_AUDIT.md §E.

import { gql, type TypedDocumentNode } from '@apollo/client';
import type { UUID, FabricStatus, Timestamptz } from '../queries/fabric_queries';

// ---------- Input helpers (mirror Hasura input shapes you use) ----------
export interface FabricDefnInsertInput {
  model_key: string;
  version: number;
  title?: string | null;
  description?: string | null;
  source_config: unknown;
  resolved_config: unknown;
  checksum_sha256?: string | null; // Hasura's bytea scalar is usually serialized as base64 string
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

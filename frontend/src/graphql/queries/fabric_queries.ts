// fabric_queries.ts
// 2026-06-30 partial-prune (Phase 0.4): the file used to contain 4 fragments and 5
// queries (GET_CURRENT_MODEL, LIST_DEFINITIONS, GET_DEFINITION_BY_ID, SEARCH_INDEX,
// GET_DEFINITION_AUDIT) — all confirmed dead via ts-prune + per-export grep.
// The file is now a pure types module used by `fabric_mutations.ts` and
// `joinExtractionService.ts`. See docs/HASURA_AUDIT.md §E.

// ---------- Shared scalars and enums (GraphQL -> TS) ----------
export type UUID = string;
export type Timestamptz = string;

export type FabricStatus = 'draft' | 'published' | 'archived';
export type JoinRelationship = 'one_to_one' | 'one_to_many' | 'many_to_one' | 'many_to_many';

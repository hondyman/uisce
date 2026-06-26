// fabric_apollo_helpers.ts
// Small helpers to improve DX when using Apollo with Hasura.
// - ILIKE pattern builders with correct escaping
// - Debounced search wrapper
// - Apollo error normalization + safe mutation executor

import {
  ApolloError,
  type MutationFunction,
  type OperationVariables,
  type QueryHookOptions,
  type QueryResult,
  useQuery,
} from '@apollo/client';
import { useEffect, useMemo, useState } from 'react';

// Import your typed docs and TS types
import {
  SEARCH_INDEX,
  type SearchIndexData,
  type SearchIndexVariables,
} from '../queries/fabric_queries';

// -------------------------------
// ILIKE helpers
// -------------------------------

/**
 * Escape Postgres LIKE metacharacters. Postgres uses backslash as default escape.
 * Converts % -> \% and _ -> \_ to prevent them from acting as wildcards.
 */
export function escapeLikeLiteral(input: string): string {
  // Also escape backslashes to avoid accidental escapes in the pattern itself.
  return input.replace(/\\/g, '\\\\').replace(/%/g, '\\%').replace(/_/g, '\\_');
}

/**
 * Build a case-insensitive LIKE pattern for "contains".
 * Examples:
 *  - term="" => "%" (match all)
 *  - term="rev_10%" => "%rev\_10\%%"
 */
export function makeIlikeContains(term: string): string {
  const t = term.trim();
  if (!t) return '%';
  return `%${escapeLikeLiteral(t)}%`;
}

/** Build a case-insensitive LIKE pattern for "starts with". */
export function makeIlikeStartsWith(term: string): string {
  const t = term.trim();
  if (!t) return '%';
  return `${escapeLikeLiteral(t)}%`;
}

/** Build a case-insensitive LIKE pattern for "ends with". */
export function makeIlikeEndsWith(term: string): string {
  const t = term.trim();
  if (!t) return '%';
  return `%${escapeLikeLiteral(t)}`;
}

// -------------------------------
// Error normalization
// -------------------------------

export type NormalizedErrorKind = 'network' | 'graphql' | 'client' | 'unknown';

export interface NormalizedError {
  kind: NormalizedErrorKind;
  message: string;
  code?: string | number | null;
  path?: string | string[] | null;
  details?: unknown;
  raw?: unknown; // Keep the original error object for logging if needed
}

/**
 * Normalize Apollo errors into a single, UI-friendly shape.
 * - Prefers the first GraphQL error if present (common for Hasura constraint/permission errors).
 * - Falls back to network error or a generic message.
 */
export function normalizeApolloError(err: unknown): NormalizedError {
  if (err instanceof ApolloError) {
    // GraphQL errors (from server)
    if (err.graphQLErrors?.length) {
      const g = err.graphQLErrors[0] as any;
      return {
        kind: 'graphql',
        message: g.message || 'A GraphQL error occurred',
        code: g.extensions?.code ?? null,
        path: (g.path as string[] | undefined) ?? null,
        details: g.extensions ?? undefined,
        raw: err,
      };
    }
    // Network errors (transport-level)
    if (err.networkError) {
      const ne: any = err.networkError;
      return {
        kind: 'network',
        message: ne.result?.error || ne.message || 'A network error occurred',
        code: ne.statusCode ?? null,
        details: ne.result ?? undefined,
        raw: err,
      };
    }
    // Client-side errors (parsing, cache, etc.)
    if (err.clientErrors?.length) {
      const ce = err.clientErrors[0];
      return {
        kind: 'client',
        message: ce.message || 'A client error occurred',
        code: null,
        details: ce,
        raw: err,
      };
    }
    return {
      kind: 'unknown',
      message: err.message || 'An unknown Apollo error occurred',
      code: null,
      raw: err,
    };
  }
  // Non-Apollo errors
  return {
    kind: 'unknown',
    message: (err as any)?.message || 'An unexpected error occurred',
    code: null,
    raw: err,
  };
}

/**
 * Execute a mutation safely and return a Result-like union.
 * Avoids try/catch sprawl at call sites and centralizes error shape.
 */
export async function execMutation<TData, TVars extends OperationVariables>(
  mutate: MutationFunction<TData, TVars>,
  variables: TVars
): Promise<{ data: TData | null; error: NormalizedError | null }> {
  try {
    const res = await mutate({ variables });
    return { data: res.data as TData, error: null };
  } catch (e) {
    return { data: null, error: normalizeApolloError(e) };
  }
}

// -------------------------------
// Debounced search helper
// -------------------------------

/** Minimal debouncer for primitive values. */
export function useDebouncedValue<T>(value: T, delayMs = 250): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const h = setTimeout(() => setDebounced(value), delayMs);
    return () => clearTimeout(h);
  }, [value, delayMs]);
  return debounced;
}

/**
 * Opinionated search wrapper for fabric_defn_index using ILIKE contains.
 * - Debounces user input
 * - Auto wraps and escapes the search term
 * - Skips querying until minChars is reached (default 1)
 */
export function useSearchIndexQueryWrapped(
  args: {
    model_key: string;
    kinds?: string[] | null; // e.g., ['dimension','measure','join']
    term: string;
    minChars?: number;
    debounceMs?: number;
  },
  options?: Omit<QueryHookOptions<SearchIndexData, SearchIndexVariables>, 'variables' | 'skip'>
): QueryResult<SearchIndexData, SearchIndexVariables> {
  const { model_key, kinds, term, minChars = 1, debounceMs = 250 } = args;

  const debouncedTerm = useDebouncedValue(term, debounceMs);
  const ilike = useMemo(() => makeIlikeContains(debouncedTerm), [debouncedTerm]);

  const skip = useMemo(() => {
    // Skip if model_key is empty or we haven't met minChars (unless empty -> allow "%")
    const trimmed = debouncedTerm.trim();
    const notEnoughChars = trimmed.length < minChars && trimmed.length > 0;
    return !model_key || notEnoughChars;
  }, [debouncedTerm, minChars, model_key]);

  return useQuery<SearchIndexData, SearchIndexVariables>(SEARCH_INDEX, {
    variables: { model_key, kinds: kinds ?? ['dimension', 'measure', 'join'], q: ilike },
    skip,
    notifyOnNetworkStatusChange: true,
    fetchPolicy: options?.fetchPolicy ?? 'cache-first',
    ...options,
  });
}

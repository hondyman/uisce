import { devError } from './devLogger';
import { useState, useEffect, useCallback, useRef } from 'react';
import apiClient from './apiClient';
import { getCachedGoldCopyId } from './goldCopy';

export interface AbbreviationEntry {
  id: number;
  abbreviation: string;
  full_word: string;
  notes: string;
  tenant_id: string;
  is_core: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface GetAbbreviationsResponse {
  items: AbbreviationEntry[];
  total_count: number;
  limit: number;
  offset: number;
}

export interface AbbreviationExpansion {
  column_name: string;
  variations: string[];
  expansions: string;
}

export interface AbbreviationValidation {
  violations: Record<string, string[]>;
  valid_terms: number;
  total_terms: number;
}

export interface ScanResult {
  candidates: string[];
  count: number;
}

export interface SuggestionResult {
  [abbreviation: string]: string;
}

// ============================================================================
// Dev fallback: the backend /abbreviations endpoint depends on the
// sml.abbreviation_lookup table, which is not always available in local dev.
// When the API fails in DEV mode, fall back to a gold-copy (core) map plus
// tenant-scoped custom entries persisted in localStorage so the
// /core/abbreviations page still shows values.
// ============================================================================

const isDevFallbackEnabled = typeof import.meta !== 'undefined' && (import.meta as any).env?.DEV === true;

const GOLD_COPY_ABBREVIATIONS: Record<string, string> = {
  // Geographic
  CNTRY: 'COUNTRY',
  CTRY: 'COUNTRY',
  ST: 'STATE',
  ADDR: 'ADDRESS',
  ZIP: 'ZIPCODE',
  POSTAL: 'POSTALCODE',
  CTY: 'CITY',
  REGN: 'REGION',

  // Financial
  AMT: 'AMOUNT',
  VAL: 'VALUE',
  BAL: 'BALANCE',
  CURR: 'CURRENCY',
  FX: 'FOREIGN_EXCHANGE',
  ACCT: 'ACCOUNT',
  TXN: 'TRANSACTION',
  PMT: 'PAYMENT',

  // Temporal
  DT: 'DATE',
  DTM: 'DATETIME',
  TS: 'TIMESTAMP',
  YR: 'YEAR',
  MON: 'MONTH',
  WK: 'WEEK',
  QTR: 'QUARTER',

  // Business
  CUST: 'CUSTOMER',
  CLNT: 'CLIENT',
  ORD: 'ORDER',
  PROD: 'PRODUCT',
  CATEG: 'CATEGORY',
  DEPT: 'DEPARTMENT',
  DIV: 'DIVISION',
  ORG: 'ORGANIZATION',
  COMP: 'COMPANY',

  // Identity
  ID: 'IDENTIFIER',
  NUM: 'NUMBER',
  NBR: 'NUMBER',
  NO: 'NUMBER',
  CD: 'CODE',
  KEY: 'KEY',
  REF: 'REFERENCE',

  // Measurements
  QTY: 'QUANTITY',
  CNT: 'COUNT',
  PCT: 'PERCENT',
  RATE: 'RATE',
  RATIO: 'RATIO',
  SCORE: 'SCORE',
  RANK: 'RANK',

  // Status/Flags
  FLG: 'FLAG',
  IND: 'INDICATOR',
  STAT: 'STATUS',
  TYP: 'TYPE',

  // Common prefixes/suffixes
  DESC: 'DESCRIPTION',
  NM: 'NAME',
  TTL: 'TOTAL',
  AVG: 'AVERAGE',
  MIN: 'MINIMUM',
  MAX: 'MAXIMUM',
  SUM: 'SUMMARY',
};

function localAbbreviationsKey(tenantId?: string): string {
  return `uisce_abbreviations_v1_custom_${tenantId || 'global'}`;
}

function loadLocalAbbreviations(tenantId?: string): AbbreviationEntry[] {
  try {
    const raw = localStorage.getItem(localAbbreviationsKey(tenantId));
    return raw ? (JSON.parse(raw) as AbbreviationEntry[]) : [];
  } catch (e) {
    devError('Failed to load local abbreviations:', e);
    return [];
  }
}

function saveLocalAbbreviations(items: AbbreviationEntry[], tenantId?: string): void {
  try {
    localStorage.setItem(localAbbreviationsKey(tenantId), JSON.stringify(items));
  } catch (e) {
    devError('Failed to save local abbreviations:', e);
  }
}

function getCoreAbbreviations(): AbbreviationEntry[] {
  const goldCopyId = getCachedGoldCopyId() ?? 'pending-resolution';
  return Object.entries(GOLD_COPY_ABBREVIATIONS).map(([abbreviation, full_word], index) => ({
    id: -1000 - index,
    abbreviation,
    full_word,
    notes: 'Core (gold copy) abbreviation',
    tenant_id: goldCopyId,
    is_core: true,
    created_at: undefined,
    updated_at: undefined,
  }));
}

function getFallbackAbbreviationMap(tenantId?: string): Map<string, string> {
  const all = [...getCoreAbbreviations(), ...loadLocalAbbreviations(tenantId)];
  return new Map(all.map((a) => [a.abbreviation.toUpperCase(), a.full_word.toUpperCase()]));
}

function getFallbackAbbreviations(tenantId?: string): AbbreviationEntry[] {
  return [...getCoreAbbreviations(), ...loadLocalAbbreviations(tenantId)];
}

function shouldUseFallback(): boolean {
  return isDevFallbackEnabled;
}

function filterAndPaginateAbbreviations(
  items: AbbreviationEntry[],
  limit: number,
  offset: number,
  search: string
): GetAbbreviationsResponse {
  let filtered = [...items];
  if (search) {
    const query = search.toUpperCase();
    filtered = filtered.filter(
      (a) =>
        a.abbreviation.toUpperCase().includes(query) ||
        a.full_word.toUpperCase().includes(query) ||
        (a.notes || '').toUpperCase().includes(query)
    );
  }
  filtered.sort((a, b) => a.abbreviation.localeCompare(b.abbreviation));
  const total_count = filtered.length;
  return {
    items: filtered.slice(offset, offset + limit),
    total_count,
    limit,
    offset,
  };
}

function nextLocalId(items: AbbreviationEntry[]): number {
  const numericIds = items.map((i) => Number(i.id) || 0);
  const min = numericIds.length > 0 ? Math.min(...numericIds) : 0;
  return (min <= 0 ? min : 0) - 1;
}

function addLocalAbbreviation(
  abbreviation: string,
  fullWord: string,
  notes: string,
  tenantId?: string
): void {
  const items = loadLocalAbbreviations(tenantId);
  items.push({
    id: nextLocalId(items),
    abbreviation,
    full_word: fullWord,
    notes,
    tenant_id: tenantId || '',
    is_core: false,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  });
  saveLocalAbbreviations(items, tenantId);
}

function updateLocalAbbreviation(
  id: number,
  abbreviation: string,
  fullWord: string,
  notes: string,
  tenantId?: string
): void {
  const items = loadLocalAbbreviations(tenantId);
  const index = items.findIndex((i) => i.id === id);
  if (index === -1) {
    throw new Error('Abbreviation not found');
  }
  items[index] = {
    ...items[index],
    abbreviation,
    full_word: fullWord,
    notes,
    updated_at: new Date().toISOString(),
  };
  saveLocalAbbreviations(items, tenantId);
}

function deleteLocalAbbreviation(id: number, tenantId?: string): void {
  const items = loadLocalAbbreviations(tenantId);
  const next = items.filter((i) => i.id !== id);
  if (next.length === items.length) {
    throw new Error('Abbreviation not found');
  }
  saveLocalAbbreviations(next, tenantId);
}

function getCurrentTenantIdFromStorage(): string | undefined {
  try {
    const raw = localStorage.getItem('selected_tenant');
    if (!raw) return undefined;
    const parsed = JSON.parse(raw);
    return typeof parsed?.id === 'string' ? parsed.id : undefined;
  } catch {
    return undefined;
  }
}

function generateTokenCombinations(tokenSets: string[][]): string[][] {
  if (tokenSets.length === 0) return [];
  if (tokenSets.length === 1) return tokenSets[0].map((token) => [token]);

  const [first, ...rest] = tokenSets;
  const restCombos = generateTokenCombinations(rest);
  const result: string[][] = [];
  for (const token of first) {
    for (const combo of restCombos) {
      result.push([token, ...combo]);
    }
  }
  return result;
}

function expandAbbreviationsFallback(columnName: string): AbbreviationExpansion {
  const normalized = columnName.toUpperCase();
  const tokens = normalized.split(/[_\-\.\s]/);
  const map = getFallbackAbbreviationMap(getCurrentTenantIdFromStorage());

  const expandedTokenSets = tokens.map((token) => {
    const expansion = map.get(token);
    return expansion ? [token, expansion] : [token];
  });

  const hasExpansion = expandedTokenSets.some((set) => set.length > 1);
  const variations: string[] = [normalized];
  if (hasExpansion) {
    const combos = generateTokenCombinations(expandedTokenSets);
    combos.forEach((combo) => variations.push(combo.join('_')));
  }

  const expansions = tokens
    .filter((token) => map.has(token))
    .map((token) => `${token}→${map.get(token)}`)
    .join(', ');

  return {
    column_name: columnName,
    variations: [...new Set(variations)],
    expansions,
  };
}

function validateSemanticTermsFallback(termNames: string[]): AbbreviationValidation {
  const map = getFallbackAbbreviationMap(getCurrentTenantIdFromStorage());
  const violations: Record<string, string[]> = {};
  let valid_terms = 0;

  for (const termName of termNames) {
    const normalized = termName.toUpperCase();
    const tokens = normalized.split(/[_\-\.\s]/);
    const found: string[] = [];

    for (const token of tokens) {
      const expansion = map.get(token);
      if (expansion) {
        found.push(`${token} -> ${expansion}`);
      }
    }

    if (found.length > 0) {
      violations[termName] = found;
    } else {
      valid_terms++;
    }
  }

  return {
    violations,
    valid_terms,
    total_terms: termNames.length,
  };
}

// API client for abbreviations
class AbbreviationApiClient {
  /**
   * Get abbreviations with pagination and search
   */
  async getAbbreviations(limit = 50, offset = 0, search = '', tenantId?: string): Promise<GetAbbreviationsResponse> {
    try {
      const params = new URLSearchParams({
        limit: limit.toString(),
        offset: offset.toString(),
      });
      if (search) {
        params.append('q', search);
      }

      const headers: Record<string, string> = {};
      if (tenantId) {
        headers['X-Tenant-ID'] = tenantId;
      }

      return await apiClient<GetAbbreviationsResponse>(`/abbreviations?${params.toString()}`, {
        headers
      });
    } catch (error) {
      if (shouldUseFallback()) {
        devError('Abbreviations API failed, using dev fallback:', error);
        return filterAndPaginateAbbreviations(getFallbackAbbreviations(tenantId), limit, offset, search);
      }
      throw error;
    }
  }

  /**
   * Add a new abbreviation
   */
  async addAbbreviation(abbreviation: string, fullWord: string, notes?: string, tenantId?: string): Promise<void> {
    try {
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
      };
      if (tenantId) {
        headers['X-Tenant-ID'] = tenantId;
      }

      await apiClient<void>(`/abbreviations`, {
        method: 'POST',
        headers,
        body: JSON.stringify({
          abbreviation,
          full_word: fullWord,
          notes: notes || '',
        }),
      });
    } catch (error) {
      if (shouldUseFallback()) {
        devError('Add abbreviation API failed, using dev fallback:', error);
        addLocalAbbreviation(abbreviation, fullWord, notes || '', tenantId);
        return;
      }
      throw error;
    }
  }

  // ... (skipping expand/validate/scan/suggest as they might not need explicit tenantId or are less critical right now, but keeping context)
  /**
   * Expand abbreviations in a column name
   */
  async expandAbbreviations(columnName: string): Promise<AbbreviationExpansion> {
    try {
      return await apiClient<AbbreviationExpansion>(`/abbreviations/expand`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          column_name: columnName,
        }),
      });
    } catch (error) {
      if (shouldUseFallback()) {
        devError('Expand abbreviations API failed, using dev fallback:', error);
        return expandAbbreviationsFallback(columnName);
      }
      throw error;
    }
  }

  /**
   * Validate semantic terms for abbreviation violations
   */
  async validateSemanticTerms(termNames: string[]): Promise<AbbreviationValidation> {
    try {
      return await apiClient<AbbreviationValidation>(`/abbreviations/validate`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          term_names: termNames,
        }),
      });
    } catch (error) {
      if (shouldUseFallback()) {
        devError('Validate semantic terms API failed, using dev fallback:', error);
        return validateSemanticTermsFallback(termNames);
      }
      throw error;
    }
  }

  /**
   * Scan database for new abbreviation candidates
   */
  async scanForAbbreviations(): Promise<ScanResult> {
    return apiClient<ScanResult>(`/abbreviations/scan`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    });
  }

  /**
   * Suggest expansions for candidates using LLM
   */
  async suggestExpansions(candidates: string[]): Promise<SuggestionResult> {
    return apiClient<SuggestionResult>(`/abbreviations/suggest`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        candidates,
      }),
    });
  }

  /**
   * Update an existing abbreviation
   */
  async updateAbbreviation(id: number, abbreviation: string, fullWord: string, notes?: string, tenantId?: string): Promise<void> {
    try {
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
      };
      if (tenantId) {
        headers['X-Tenant-ID'] = tenantId;
      }

      await apiClient<void>(`/abbreviations/${id}`, {
        method: 'PUT',
        headers,
        body: JSON.stringify({
          abbreviation,
          full_word: fullWord,
          notes: notes || '',
        }),
      });
    } catch (error) {
      if (shouldUseFallback()) {
        devError('Update abbreviation API failed, using dev fallback:', error);
        updateLocalAbbreviation(id, abbreviation, fullWord, notes || '', tenantId);
        return;
      }
      throw error;
    }
  }

  /**
   * Delete an abbreviation
   */
  async deleteAbbreviation(id: number, tenantId?: string): Promise<void> {
    try {
      const headers: Record<string, string> = {};
      if (tenantId) {
        headers['X-Tenant-ID'] = tenantId;
      }

      await apiClient<void>(`/abbreviations/${id}`, {
        method: 'DELETE',
        headers,
      });
    } catch (error) {
      if (shouldUseFallback()) {
        devError('Delete abbreviation API failed, using dev fallback:', error);
        deleteLocalAbbreviation(id, tenantId);
        return;
      }
      throw error;
    }
  }
}

export const abbreviationApiClient = new AbbreviationApiClient();

// React hook for managing abbreviation data
export function useAbbreviations(tenantId?: string) {
  const [abbreviations, setAbbreviations] = useState<AbbreviationEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [totalCount, setTotalCount] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  const [loaded, setLoaded] = useState(false);

  // Pagination state
  const LIMIT = 50;
  const offsetRef = useRef(0);
  const searchRef = useRef('');
  const loadingRef = useRef(false); // To prevent double fetch

  const fetchAbbreviations = useCallback(async (reset = false, search?: string) => {
    if (loadingRef.current) return;

    setLoading(true);
    loadingRef.current = true;
    setError(null);

    try {
      if (reset) {
        offsetRef.current = 0;
        if (search !== undefined) searchRef.current = search;
      }

      // Pass tenantId if present
      const data = await abbreviationApiClient.getAbbreviations(LIMIT, offsetRef.current, searchRef.current, tenantId);

      setAbbreviations(prev => reset ? data.items : [...prev, ...data.items]);
      setTotalCount(data.total_count);

      const newOffset = offsetRef.current + data.items.length;
      offsetRef.current = newOffset;
      setHasMore(newOffset < data.total_count);
      setLoaded(true);

    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch abbreviations');
    } finally {
      setLoading(false);
      loadingRef.current = false;
    }
  }, [tenantId]); // Add tenantId dependency

  const loadMore = useCallback(() => {
    if (!hasMore || loading) return;
    fetchAbbreviations(false);
  }, [hasMore, loading, fetchAbbreviations]);

  const searchAbbreviations = useCallback((query: string) => {
    fetchAbbreviations(true, query);
  }, [fetchAbbreviations]);

  const addAbbreviation = useCallback(async (abbreviation: string, fullWord: string, notes?: string) => {
    setError(null);
    try {
      await abbreviationApiClient.addAbbreviation(abbreviation, fullWord, notes, tenantId);
      await fetchAbbreviations(true); // Refresh list
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add abbreviation');
      return false;
    }
  }, [fetchAbbreviations, tenantId]);

  const updateAbbreviation = useCallback(async (id: number, abbreviation: string, fullWord: string, notes?: string) => {
    setError(null);
    try {
      await abbreviationApiClient.updateAbbreviation(id, abbreviation, fullWord, notes, tenantId);
      await fetchAbbreviations(true); // Refresh list (simple approach: reset to top)
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update abbreviation');
      return false;
    }
  }, [fetchAbbreviations, tenantId]);

  const deleteAbbreviation = useCallback(async (id: number) => {
    setError(null);
    try {
      await abbreviationApiClient.deleteAbbreviation(id, tenantId);
      // Optimistic update could be done here, but for now we reset 
      // or we could filter out the item to keep position
      setAbbreviations(prev => prev.filter(a => a.id !== id));
      setTotalCount(prev => prev - 1);
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete abbreviation');
      return false;
    }
  }, [tenantId]);

  useEffect(() => {
    // Only load if explicitly requested - lazy loading in component
    // fetchAbbreviations();
    // Reset loaded state when tenant changes
    if (tenantId) {
      setLoaded(false);
      setAbbreviations([]);
      offsetRef.current = 0;
    }
  }, [fetchAbbreviations, tenantId]);

  return {
    abbreviations,
    totalCount,
    hasMore,
    loading,
    error,
    loaded,
    loadMore,
    searchAbbreviations,
    fetchAbbreviations, // For manual refresh if needed
    addAbbreviation,
    updateAbbreviation,
    deleteAbbreviation,
  };
}

// React hook for abbreviation expansion
export function useAbbreviationExpansion() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const expandColumn = useCallback(async (columnName: string): Promise<AbbreviationExpansion | null> => {
    setLoading(true);
    setError(null);
    try {
      const result = await abbreviationApiClient.expandAbbreviations(columnName);
      return result;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to expand abbreviations');
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    expandColumn,
    loading,
    error,
  };
}

// React hook for semantic term validation
export function useSemanticTermValidation() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const validateTerms = useCallback(async (termNames: string[]): Promise<AbbreviationValidation | null> => {
    setLoading(true);
    setError(null);
    try {
      const result = await abbreviationApiClient.validateSemanticTerms(termNames);
      return result;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to validate terms');
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    validateTerms,
    loading,
    error,
  };
}

// Cache for abbreviations to avoid repeated API calls
let abbreviationCache: Map<string, string> | null = null;
let cacheTimestamp = 0;
const CACHE_DURATION = 5 * 60 * 1000; // 5 minutes

/**
 * Get abbreviation map with caching
 */
export async function getAbbreviationMap(): Promise<Map<string, string>> {
  const now = Date.now();

  // Return cached data if still fresh
  if (abbreviationCache && (now - cacheTimestamp) < CACHE_DURATION) {
    return abbreviationCache;
  }

  try {
    // Request a large limit for caching purposes, or loop. 
    // Ideally scan/expand logic should be server side, but for now we fetch a lot.
    // 10000 limit should cover most cases for now.
    const response = await abbreviationApiClient.getAbbreviations(10000, 0, '');
    abbreviationCache = new Map(
      response.items.map(abbrev => [abbrev.abbreviation.toUpperCase(), abbrev.full_word.toUpperCase()])
    );
    cacheTimestamp = now;
    return abbreviationCache;
  } catch (error) {
    devError('Failed to fetch abbreviations:', error);
    if (shouldUseFallback()) {
      const fallback = getFallbackAbbreviationMap(getCurrentTenantIdFromStorage());
      abbreviationCache = fallback;
      cacheTimestamp = now;
      return fallback;
    }
    // Return empty map if API fails in production
    return new Map();
  }
}

/**
 * Clear abbreviation cache to force refresh
 */
export function clearAbbreviationCache(): void {
  abbreviationCache = null;
  cacheTimestamp = 0;
}

// React hook for abbreviation wizard
export function useAbbreviationWizard() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const scan = useCallback(async (): Promise<ScanResult | null> => {
    setLoading(true);
    setError(null);
    try {
      const result = await abbreviationApiClient.scanForAbbreviations();
      return result;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to scan for abbreviations');
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  const suggest = useCallback(async (candidates: string[]): Promise<SuggestionResult | null> => {
    setLoading(true);
    setError(null);
    try {
      const result = await abbreviationApiClient.suggestExpansions(candidates);
      return result;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to suggest expansions');
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    scan,
    suggest,
    loading,
    error,
  };
}
import { devError } from './devLogger';
import { useState, useEffect, useCallback, useRef } from 'react';
import apiClient from './apiClient';

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

// API client for abbreviations
class AbbreviationApiClient {
  /**
   * Get abbreviations with pagination and search
   */
  async getAbbreviations(limit = 50, offset = 0, search = '', tenantId?: string): Promise<GetAbbreviationsResponse> {
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

    return apiClient<GetAbbreviationsResponse>(`/abbreviations?${params.toString()}`, {
      headers
    });
  }

  /**
   * Add a new abbreviation
   */
  async addAbbreviation(abbreviation: string, fullWord: string, notes?: string, tenantId?: string): Promise<void> {
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
  }

  // ... (skipping expand/validate/scan/suggest as they might not need explicit tenantId or are less critical right now, but keeping context)
  /**
   * Expand abbreviations in a column name
   */
  async expandAbbreviations(columnName: string): Promise<AbbreviationExpansion> {
    return apiClient<AbbreviationExpansion>(`/abbreviations/expand`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        column_name: columnName,
      }),
    });
  }

  /**
   * Validate semantic terms for abbreviation violations
   */
  async validateSemanticTerms(termNames: string[]): Promise<AbbreviationValidation> {
    return apiClient<AbbreviationValidation>(`/abbreviations/validate`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        term_names: termNames,
      }),
    });
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
  }

  /**
   * Delete an abbreviation
   */
  async deleteAbbreviation(id: number, tenantId?: string): Promise<void> {
    const headers: Record<string, string> = {};
    if (tenantId) {
      headers['X-Tenant-ID'] = tenantId;
    }

    await apiClient<void>(`/abbreviations/${id}`, {
      method: 'DELETE',
      headers,
    });
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
    devError('Failed to fetch abbreviations, using empty map:', error);
    // Return empty map if API fails
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
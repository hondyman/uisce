import { useState, useEffect } from 'react';
import * as ruleService from '../services/ruleService';

export type SemanticTerm = ruleService.SemanticTerm;

export interface UseSemanticTermsReturn {
  terms: SemanticTerm[];
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * useSemanticTerms Hook
 *
 * Loads semantic terms (business-friendly data definitions) for a business object
 * from the backend semantic catalog.
 *
 * Features:
 * - Fetches terms from public.catalog_node semantic catalog
 * - Lazy loads terms on mount
 * - Caches results in component state
 * - Categorizes terms by business meaning
 * - Displays governance status (approved/draft/deprecated)
 * - Provides sample values for UI preview
 *
 * @param businessObject - Business object name (e.g., 'calendar', 'trade', 'portfolio')
 * @returns Hook state with terms array and loading/error flags
 */
export const useSemanticTerms = (businessObject: string): UseSemanticTermsReturn => {
  const [terms, setTerms] = useState<SemanticTerm[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTerms = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await ruleService.getSemanticTerms(businessObject);
      setTerms(data);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch semantic terms';
      setError(message);
      console.error('Error loading semantic terms:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTerms();
  }, [businessObject]);

  return {
    terms,
    loading,
    error,
    refetch: fetchTerms,
  };
};

export default useSemanticTerms;

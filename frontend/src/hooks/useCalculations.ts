/**
 * useCalculations Hook
 * Hook for fetching and managing calculations from the library
 */

import { useState, useEffect } from 'react';
import { fetchAPI } from '../api';

export interface Calculation {
  id: string;
  name: string;
  description?: string;
  expression?: string;
  return_type?: string;
  arguments?: CalculationArgument[];
  category?: string;
  created_at?: string;
  updated_at?: string;
}

export interface CalculationArgument {
  name: string;
  type: string;
  description?: string;
  required?: boolean;
  default_value?: string;
}

interface UseCalculationsResult {
  calculations: Calculation[];
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

export function useCalculations(): UseCalculationsResult {
  const [calculations, setCalculations] = useState<Calculation[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchCalculations = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchAPI<Calculation[]>('/calculations');
      setCalculations(data || []);
    } catch (err) {
      console.error('Failed to fetch calculations:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch calculations');
      setCalculations([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCalculations();
  }, []);

  return {
    calculations,
    loading,
    error,
    refetch: fetchCalculations,
  };
}

export default useCalculations;

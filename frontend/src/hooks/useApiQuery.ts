import { useState, useEffect, useCallback, useRef } from 'react';
import { apiClient } from '../utils/apiClient';

export interface ApiQueryOptions {
  skip?: boolean;
  dependencies?: any[];
}

export interface UseApiQueryResult<T = any> {
  data: T | null;
  loading: boolean;
  error: Error | null;
  refetch: () => Promise<void>;
}

export function useApiQuery<T = any>(
  url: string,
  options: ApiQueryOptions = {},
): UseApiQueryResult<T> {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(!options.skip);
  const [error, setError] = useState<Error | null>(null);

  const depsKey = JSON.stringify(options.dependencies ?? []);
  const skipRef = useRef(options.skip);
  skipRef.current = options.skip;

  const execute = useCallback(async () => {
    if (skipRef.current) return;
    setLoading(true);
    setError(null);
    try {
      const resolvedPath = url.startsWith('/') ? url : `/${url}`;
      const response = await apiClient<T>(resolvedPath);
      setData(response);
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setLoading(false);
    }
  }, [url, depsKey]);

  useEffect(() => {
    execute();
  }, [execute]);

  return { data, loading, error, refetch: execute };
}

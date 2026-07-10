import { useState, useCallback } from 'react';
import { apiClient } from '../utils/apiClient';

export interface UseApiMutationOptions {
  onCompleted?: (data: any) => void;
  onError?: (error: Error) => void;
}

export interface UseApiMutationResult<TData = any, TVariables = any> {
  mutate: (variables: TVariables) => Promise<TData>;
  loading: boolean;
  error: Error | null;
  data: TData | null;
}

export function useApiMutation<TData = any, TVariables = any>(
  path: string,
  method: 'POST' | 'PATCH' | 'DELETE' = 'POST',
  options?: UseApiMutationOptions,
): UseApiMutationResult<TData, TVariables> {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [data, setData] = useState<TData | null>(null);

  const mutate = useCallback(
    async (variables: TVariables): Promise<TData> => {
      setLoading(true);
      setError(null);

      try {
        const resolvedPath = path.startsWith('/') ? path : `/${path}`;

        const response = await apiClient<TData>(resolvedPath, {
          method,
          headers: {
            'Content-Type': 'application/json',
          },
          body: method !== 'DELETE' ? JSON.stringify(variables) : undefined,
          credentials: 'include',
        });

        setData(response);
        options?.onCompleted?.(response);
        return response;
      } catch (err) {
        const errorObj = err instanceof Error ? err : new Error(String(err));
        setError(errorObj);
        options?.onError?.(errorObj);
        throw errorObj;
      } finally {
        setLoading(false);
      }
    },
    [path, method, options],
  );

  return { mutate, loading, error, data };
}

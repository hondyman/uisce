import { useEffect, useRef, useState } from 'react';

/**
 * Polls a REST endpoint on an interval, replacing what used to be a live
 * GraphQL subscription over Apollo/Hasura. This is the simplest safe
 * substitute for real-time updates now that the frontend no longer talks
 * to Hasura directly.
 */
export function usePolling<T>(fetcher: () => Promise<T>, intervalMs = 5000, deps: unknown[] = []) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const fetcherRef = useRef(fetcher);
  fetcherRef.current = fetcher;

  useEffect(() => {
    let cancelled = false;

    const run = async () => {
      try {
        const result = await fetcherRef.current();
        if (!cancelled) {
          setData(result);
          setError(null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err : new Error(String(err)));
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    run();
    const id = setInterval(run, intervalMs);

    return () => {
      cancelled = true;
      clearInterval(id);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  return { data, loading, error };
}

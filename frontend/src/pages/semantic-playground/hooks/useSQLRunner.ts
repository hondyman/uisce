// Custom hook for running SQL queries
import { useState } from "react";
import {
  QueryExecutionResponse,
} from "../types";
import { semanticPlaygroundApi } from "../utils/api";

interface UseSQLRunnerResult {
  results: QueryExecutionResponse | null;
  executionTime: number | null;
  loading: boolean;
  error: string | null;
  runSQL: (sql: string) => Promise<void>;
}

export function useSQLRunner(): UseSQLRunnerResult {
  const [results, setResults] = useState<QueryExecutionResponse | null>(null);
  const [executionTime, setExecutionTime] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const runSQL = async (sql: string) => {
    setLoading(true);
    setError(null);
    const startTime = performance.now();

    try {
      const response = await semanticPlaygroundApi.runSQL({
        sql,
        limit: 1000,
      });
      const endTime = performance.now();
      setResults(response);
      setExecutionTime(response.execution_time_ms);
    } catch (err: any) {
      setError(err.message || "Failed to run SQL query");
      setResults(null);
    } finally {
      setLoading(false);
    }
  };

  return {
    results,
    executionTime,
    loading,
    error,
    runSQL,
  };
}

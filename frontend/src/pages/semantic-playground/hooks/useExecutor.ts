// Custom hook for calling the executor LLM
import { useState } from "react";
import {
  ExecutorRequest,
  ExecutorResponse,
} from "../types";
import { semanticPlaygroundApi } from "../utils/api";

interface UseExecutorResult {
  generatedSQL: string | null;
  warnings: string[];
  loading: boolean;
  error: string | null;
  callExecutor: (request: ExecutorRequest) => Promise<void>;
}

export function useExecutor(): UseExecutorResult {
  const [generatedSQL, setGeneratedSQL] = useState<string | null>(null);
  const [warnings, setWarnings] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const callExecutor = async (request: ExecutorRequest) => {
    setLoading(true);
    setError(null);

    try {
      const response = await semanticPlaygroundApi.callExecutor(request);
      setGeneratedSQL(response.generated_sql);
      setWarnings(response.warnings || []);
    } catch (err: any) {
      setError(err.message || "Failed to call executor LLM");
      setGeneratedSQL(null);
    } finally {
      setLoading(false);
    }
  };

  return {
    generatedSQL,
    warnings,
    loading,
    error,
    callExecutor,
  };
}

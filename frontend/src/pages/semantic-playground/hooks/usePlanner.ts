// Custom hook for calling the planner LLM
import { useState } from "react";
import {
  PlannerRequest,
  PlannerResponse,
  SemanticQuery,
} from "../types";
import { semanticPlaygroundApi } from "../utils/api";

interface UsePlannerResult {
  semanticQuery: SemanticQuery | null;
  explanation: string | null;
  confidence: number | null;
  warnings: string[];
  loading: boolean;
  error: string | null;
  callPlanner: (request: PlannerRequest) => Promise<void>;
}

export function usePlanner(): UsePlannerResult {
  const [semanticQuery, setSemanticQuery] = useState<SemanticQuery | null>(
    null
  );
  const [explanation, setExplanation] = useState<string | null>(null);
  const [confidence, setConfidence] = useState<number | null>(null);
  const [warnings, setWarnings] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const callPlanner = async (request: PlannerRequest) => {
    setLoading(true);
    setError(null);

    try {
      const response = await semanticPlaygroundApi.callPlanner(request);
      setSemanticQuery(response.semantic_query);
      setExplanation(response.explanation || null);
      setConfidence(response.confidence || null);
      setWarnings(response.warnings || []);
    } catch (err: any) {
      setError(err.message || "Failed to call planner LLM");
      setSemanticQuery(null);
    } finally {
      setLoading(false);
    }
  };

  return {
    semanticQuery,
    explanation,
    confidence,
    warnings,
    loading,
    error,
    callPlanner,
  };
}

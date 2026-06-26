// Custom hook for fetching semantic bundles
import { useEffect, useState } from "react";
import {
  SemanticBundle,
  BundleVersion,
} from "../types";
import { semanticPlaygroundApi } from "../utils/api";

interface UseSemanticBundleResult {
  bundle: SemanticBundle | null;
  versions: BundleVersion[];
  loading: boolean;
  error: string | null;
  fetchBundle: (datasource: string, version?: string) => Promise<void>;
  fetchVersions: (datasource: string) => Promise<void>;
}

export function useSemanticBundle(): UseSemanticBundleResult {
  const [bundle, setBundle] = useState<SemanticBundle | null>(null);
  const [versions, setVersions] = useState<BundleVersion[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchBundle = async (datasource: string, version?: string) => {
    setLoading(true);
    setError(null);

    try {
      const bundle = await semanticPlaygroundApi.getBundle(datasource, version);
      setBundle(bundle);
    } catch (err: any) {
      setError(err.message || "Failed to fetch semantic bundle");
      setBundle(null);
    } finally {
      setLoading(false);
    }
  };

  const fetchVersions = async (datasource: string) => {
    try {
      const versions = await semanticPlaygroundApi.getBundleVersions(datasource);
      setVersions(versions);
    } catch (err: any) {
      devError("Failed to fetch bundle versions:", err);
    }
  };

  return {
    bundle,
    versions,
    loading,
    error,
    fetchBundle,
    fetchVersions,
  };
}

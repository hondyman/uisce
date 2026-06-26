// API Client for Semantic Playground

import {
  SemanticBundle,
  PlannerRequest,
  PlannerResponse,
  ExecutorRequest,
  ExecutorResponse,
  QueryExecutionRequest,
  QueryExecutionResponse,
  Datasource,
  BundleVersion,
  LineageNode,
} from "../types";

import { getSelectedRegion } from "../../../lib/region";

const API_BASE_URL = import.meta.env.VITE_API_URL || "http://localhost:8080/api";

interface ApiError {
  message: string;
  code?: string;
  details?: any;
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const error: ApiError = {
      message: `HTTP ${response.status}: ${response.statusText}`,
      code: `HTTP_${response.status}`,
    };

    try {
      error.details = await response.json();
    } catch {
      error.details = await response.text();
    }

    throw error;
  }

  return response.json();
}

export const semanticPlaygroundApi = {
  // Get list of datasources
  async getDatasources(): Promise<Datasource[]> {
    const response = await fetch(`${API_BASE_URL}/semantic/datasources`, {
      headers: {
        "X-Tenant-ID": localStorage.getItem("tenantId") || "default",
        ...(getSelectedRegion() ? { 'X-Tenant-Region': getSelectedRegion() } : {}),
      },
    });
    return handleResponse(response);
  },

  // Get semantic bundle for a datasource + version
  async getBundle(
    datasource: string,
    version?: string
  ): Promise<SemanticBundle> {
    const queryParams = new URLSearchParams({ datasource });
    if (version) queryParams.append("version", version);

    const response = await fetch(
      `${API_BASE_URL}/semantic/bundles/by-id?${queryParams}`,
      {
        headers: {
          "X-Tenant-ID": localStorage.getItem("tenantId") || "default",
          ...(getSelectedRegion() ? { 'X-Tenant-Region': getSelectedRegion() } : {}),
        },
      }
    );
    return handleResponse(response);
  },

  // Get available versions for a datasource
  async getBundleVersions(datasource: string): Promise<BundleVersion[]> {
    const response = await fetch(
      `${API_BASE_URL}/semantic/bundles/${datasource}/versions`,
      {
        headers: {
          "X-Tenant-ID": localStorage.getItem("tenantId") || "default",
          ...(getSelectedRegion() ? { 'X-Tenant-Region': getSelectedRegion() } : {}),
        },
      }
    );
    return handleResponse(response);
  },

  // Call planner LLM: NL -> SemanticQuery
  async callPlanner(request: PlannerRequest): Promise<PlannerResponse> {
    const response = await fetch(`${API_BASE_URL}/semantic/plan`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Tenant-ID": localStorage.getItem("tenantId") || "default",
        ...(getSelectedRegion() ? { 'X-Tenant-Region': getSelectedRegion() } : {}),
      },
      body: JSON.stringify(request),
    });
    return handleResponse(response);
  },

  // Call executor LLM: SemanticQuery -> SQL
  async callExecutor(request: ExecutorRequest): Promise<ExecutorResponse> {
    const response = await fetch(`${API_BASE_URL}/semantic/execute`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Tenant-ID": localStorage.getItem("tenantId") || "default",
        ...(getSelectedRegion() ? { 'X-Tenant-Region': getSelectedRegion() } : {}),
      },
      body: JSON.stringify(request),
    });
    return handleResponse(response);
  },

  // Run SQL query
  async runSQL(request: QueryExecutionRequest): Promise<QueryExecutionResponse> {
    const response = await fetch(`${API_BASE_URL}/sql/run`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Tenant-ID": localStorage.getItem("tenantId") || "default",
        ...(getSelectedRegion() ? { 'X-Tenant-Region': getSelectedRegion() } : {}),
      },
      body: JSON.stringify(request),
    });
    return handleResponse(response);
  },

  // Get field lineage
  async getFieldLineage(fieldId: string): Promise<LineageNode> {
    const response = await fetch(
      `${API_BASE_URL}/semantic/lineage/${fieldId}`,
      {
        headers: {
          "X-Tenant-ID": localStorage.getItem("tenantId") || "default",
          ...(getSelectedRegion() ? { 'X-Tenant-Region': getSelectedRegion() } : {}),
        },
      }
    );
    return handleResponse(response);
  },

  // Diff bundles
  async diffBundles(
    datasource: string,
    fromVersion: string,
    toVersion: string
  ): Promise<any> {
    const queryParams = new URLSearchParams({
      datasource,
      from: fromVersion,
      to: toVersion,
    });

    const response = await fetch(
      `${API_BASE_URL}/semantic/bundles/diff?${queryParams}`,
      {
        headers: {
          "X-Tenant-ID": localStorage.getItem("tenantId") || "default",
          ...(getSelectedRegion() ? { 'X-Tenant-Region': getSelectedRegion() } : {}),
        },
      }
    );
    return handleResponse(response);
  },

  // Explain a semantic query
  async explainQuery(datasource: string, query: any): Promise<string> {
    const response = await fetch(`${API_BASE_URL}/semantic/explain`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Tenant-ID": localStorage.getItem("tenantId") || "default",
        ...(getSelectedRegion() ? { 'X-Tenant-Region': getSelectedRegion() } : {}),
      },
      body: JSON.stringify({ datasource, query }),
    });
    return handleResponse(response);
  },
};

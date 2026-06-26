// API client wrapper - single source of truth for all API calls
import { QueryClient } from "@tanstack/react-query";

const API_BASE = process.env.REACT_APP_API_URL || "http://localhost:8082/api";

export const getAuthToken = () => localStorage.getItem("token");

export async function api<T>(
  url: string,
  options: RequestInit = {}
): Promise<T> {
  const headers = new Headers(options.headers || {});

  // Add content type if not present
  if (!headers.has("Content-Type") && options.body) {
    headers.set("Content-Type", "application/json");
  }

  // Add auth token if available
  const token = getAuthToken();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const response = await fetch(`${API_BASE}${url}`, {
    ...options,
    headers
  });

  if (!response.ok) {
    let message = `API Error: ${response.status} ${response.statusText}`;
    try {
      const text = await response.text();
      if (text) {
        const json = JSON.parse(text);
        message = json.error || json.message || message;
      }
    } catch {
      // Failed to parse error response
    }
    throw new Error(message);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json();
}

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      gcTime: 1000 * 60 * 10, // 10 minutes
      retry: 1
    }
  }
});

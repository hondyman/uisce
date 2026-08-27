// Admin API client — shared by all admin-v2 hooks
import apiClient from "../utils/apiClient";

export async function api<T = unknown>(
  path: string,
  options?: RequestInit
): Promise<T> {
  const response = await apiClient<T>(path, options);

  // If apiClient already parsed the JSON, return it directly.
  if (response && typeof response === "object" && !("ok" in response)) {
    return response as T;
  }

  // If we got a raw Response back (unusual), handle it.
  if (response && typeof response === "object" && "ok" in response) {
    const resp = response as Response;
    if (!resp.ok) {
      const text = await resp.text().catch(() => "");
      throw new Error(`${resp.status} ${resp.statusText}${text ? `: ${text}` : ""}`);
    }
    return resp.json() as Promise<T>;
  }

  return response as T;
}

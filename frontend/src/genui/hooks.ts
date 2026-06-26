import { useQuery, useMutation } from "@tanstack/react-query";
import type { LayoutSchema } from "./schema";

// API base URL (configure via env)
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";

interface IntentRequest {
    query: string;
    tenant_id?: string;
    user_id?: string;
    context?: Record<string, any>;
}

interface IntentResponse {
    intent: {
        type: string;
        objects: string[];
        metrics: string[];
        confidence: number;
    };
    layout: LayoutSchema;
}

/**
 * Hook to generate a layout from natural language query
 */
export function useGenUIIntent() {
    return useMutation<IntentResponse, Error, IntentRequest>({
        mutationFn: async (request) => {
            const response = await fetch(`${API_BASE_URL}/genui/intent`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify(request),
            });

            if (!response.ok) {
                throw new Error(`GenUI API error: ${response.statusText}`);
            }

            return response.json();
        },
    });
}

/**
 * Hook to fetch available layout templates
 */
export function useGenUITemplates() {
    return useQuery<{ templates: TemplateInfo[] }>({
        queryKey: ["genui", "templates"],
        queryFn: async () => {
            const response = await fetch(`${API_BASE_URL}/genui/templates`);

            if (!response.ok) {
                throw new Error(`GenUI API error: ${response.statusText}`);
            }

            return response.json();
        },
    });
}

interface TemplateInfo {
    id: string;
    name: string;
    description: string;
    tags: string[];
}

/**
 * Hook to load a layout by template ID
 */
export function useGenUITemplate(templateId: string) {
    return useQuery<LayoutSchema>({
        queryKey: ["genui", "template", templateId],
        queryFn: async () => {
            const response = await fetch(`${API_BASE_URL}/genui/templates/${templateId}`);

            if (!response.ok) {
                throw new Error(`GenUI API error: ${response.statusText}`);
            }

            return response.json();
        },
        enabled: !!templateId,
    });
}

/**
 * Hook to automatically fetch a layout from a natural language query
 */
export function useGenUIQuery(query: string, tenantId?: string) {
    return useQuery<LayoutSchema, Error>({
        queryKey: ["genui", "query", query, tenantId],
        queryFn: async () => {
            const response = await fetch(`${API_BASE_URL}/genui/intent`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({
                    query,
                    tenant_id: tenantId,
                }),
            });

            if (!response.ok) {
                throw new Error(`GenUI API error: ${response.statusText}`);
            }

            const data: IntentResponse = await response.json();
            return data.layout;
        },
        enabled: !!query,
    });
}

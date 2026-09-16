import { useMutation } from '@tanstack/react-query';
import { useTenant } from '../contexts/TenantContext';
import { apiFetch } from '../lib/apiClient';

export interface GenerateDAGRequest {
    prompt: string;
    existingDefinition?: object;
}

export interface GenerateDAGResponse {
    dagDefinition: object;
    explanation: string;
}

export function useGenerateDAG() {
    const { tenant } = useTenant();

    return useMutation({
        mutationFn: async (request: GenerateDAGRequest): Promise<GenerateDAGResponse> => {
            const res = await apiFetch('/api/ai/generate-dag', {
                method: 'POST',
                credentials: 'include',
                headers: {
                    'Content-Type': 'application/json',
                    ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
                },
                body: JSON.stringify(request),
            });

            return res.json();
        },
    });
}

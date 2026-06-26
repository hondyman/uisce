import { useQuery } from '@tanstack/react-query';
import { useTenant } from '../contexts/TenantContext';

export interface ViewDefinition {
    id: string;
    name: string;
    title: string;
    components: ViewComponent[];
}

export interface ViewComponent {
    id: string;
    dataKey: string;
    componentType: string; // 'ReadOnlyText', 'Table', 'Select', 'TextArea', 'Button'
    label: string;
    order: number;
    properties?: Record<string, any>;
}

export const uiMetadataKeys = {
    viewDefinition: (id: string) => ['ui-metadata', 'view-definition', id] as const,
};

export function useViewDefinition(idOrName: string) {
    const { tenant } = useTenant();

    return useQuery({
        queryKey: uiMetadataKeys.viewDefinition(idOrName),
        queryFn: async () => {
            // Direct call to backend handler
            // Note: Assuming /ui-definitions/{id} is registered at root /ui-definitions or /api/ui-definitions
            // Based on api.go registration: r.Get("/ui-definitions/{id}", ...) inside /api group
            // So path is /api/ui-definitions/{id}
            const url = `/api/ui-definitions/${idOrName}`;

            const res = await fetch(url, {
                headers: {
                    ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
                },
            });

            if (!res.ok) {
                throw new Error('Failed to fetch view definition');
            }

            return res.json() as Promise<ViewDefinition>;
        },
        enabled: !!idOrName,
    });
}

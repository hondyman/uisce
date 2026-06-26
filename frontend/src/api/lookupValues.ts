import { useQuery } from '@tanstack/react-query';

interface LookupValue {
    id: string;
    lookup_type: string;
    value: string;
    label: string;
    description?: string;
    sort_order?: number;
    is_active: boolean;
}

export function useLookupValues(lookupType: string | undefined) {
    return useQuery({
        queryKey: ['lookup-values', lookupType],
        queryFn: async () => {
            if (!lookupType) return [];

            const res = await fetch(`/api/lookup-values?type=${lookupType}`, {
                credentials: 'include',
            });

            if (!res.ok) {
                throw new Error('Failed to fetch lookup values');
            }

            const data: LookupValue[] = await res.json();
            return data;
        },
        enabled: !!lookupType,
    });
}

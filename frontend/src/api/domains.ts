import { fetchAPI } from '../api';

export interface DataDomain {
    id: string;
    name: string;
    slug: string;
    parent_id?: string;
    level: number;
    description?: string;
}

export function listDomains(): Promise<DataDomain[]> {
    return fetchAPI('/data-domains');
}

export function searchDomains(query: string): Promise<DataDomain[]> {
    return fetchAPI(`/data-domains/search?q=${encodeURIComponent(query)}`);
}

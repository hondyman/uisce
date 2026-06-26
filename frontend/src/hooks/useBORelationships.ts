import { useState, useEffect } from 'react';
import { useAccess } from '../contexts/AccessContext';
import { getSelectedRegion } from '../lib/region';
import { devError } from '../utils/devLogger';

export interface RelationshipResult {
    related_object_name: string;
    relationship_type: string;
    description: string;
}

export interface SemanticFieldResult {
    field_name: string;
    semantic_term_name: string;
    edge_type_name: string;
}

export interface AvailableSemanticTerm {
    id: string;
    node_name: string;
    display_name: string;
    description: string;
    dataType: string;
    role: string;
    qualified_path: string;
    source_column?: string;
}

export interface BORelationships {
    relatedObjects: RelationshipResult[];
    semanticFields: SemanticFieldResult[];
    availableTerms: AvailableSemanticTerm[];
}

export const useBORelationships = (boId?: string) => {
    const { currentTenant: tenant, currentDatasource: datasource } = useAccess();
    const [data, setData] = useState<BORelationships>({
        relatedObjects: [],
        semanticFields: [],
        availableTerms: [],
    });
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        if (!boId || !tenant?.id) return;

        const fetchRelationships = async () => {
            setLoading(true);
            setError(null);
            try {
                const token = localStorage.getItem('auth_token');
                const headers: Record<string, string> = {
                    'X-Tenant-ID': tenant.id,
                    'X-Tenant-Datasource-ID': datasource?.id || datasource?.alpha_tenant_instance_id || '',
                    'X-Tenant-Region': getSelectedRegion(),
                    'Content-Type': 'application/json',
                };
                if (token) {
                    headers['Authorization'] = `Bearer ${token}`;
                }

                const response = await fetch(`/api/business-objects/${boId}/relationships`, {
                    headers,
                });

                if (!response.ok) {
                    throw new Error(`Failed to fetch relationships: ${response.statusText}`);
                }

                const json = await response.json();
                setData({
                    relatedObjects: json.relatedObjects || [],
                    semanticFields: json.semanticFields || [],
                    availableTerms: json.availableTerms || [],
                });
            } catch (err) {
                devError('Error fetching BO relationships:', err);
                setError(err instanceof Error ? err.message : String(err));
            } finally {
                setLoading(false);
            }
        };

        fetchRelationships();
    }, [boId, tenant?.id, datasource?.id]);

    return { data, loading, error };
};


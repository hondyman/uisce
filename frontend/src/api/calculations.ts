import { fetchAPI } from '../api';

export interface Calculation {
    id?: string;
    node_id: string;
    name: string;
    title: string;
    description?: string;
    formula: string;
    engine_type: string;
    return_type: string;
    arguments?: Record<string, any>;
    category?: string;
    subcategory?: string;
    is_materialized?: boolean;
    created_at?: string;
    updated_at?: string;
    domain_id?: string;
    execution_type?: string; // realtime, batch
    engine?: string; // internal, cube, spark
    // Frontend compatibility fields
    type?: string; // mapped to engine_type or return_type
    sql?: string; // mapped to formula
    backendEndpoint?: string;
    financial_calc?: {
        type: string;
        formula?: string;
        arguments?: Record<string, string>;
    };
}

export function listCalculations(): Promise<Calculation[]> {
    return fetchAPI('/calculations');
}

export function createCalculation(calc: Calculation): Promise<Calculation> {
    return fetchAPI('/calculations', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(calc),
    });
}

export function getCalculation(name: string): Promise<Calculation> {
    return fetchAPI(`/calculations/${name}`);
}

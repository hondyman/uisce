import { fetchAPI } from '../api';

export interface CalcField {
    id?: string;
    tenant_id?: string;
    tenant_instance_id?: string;
    object_id: string;
    name: string;
    sql_expr: string;
    data_type: string;
    is_measure: boolean;
    realtime?: boolean;
}

export interface PreviewRequest {
    object_id: string;
    sql_expr: string;
    limit?: number;
}

export interface PreviewResponse {
    columns: string[];
    rows: any[][];
}

export async function createCalcField(field: CalcField): Promise<any> {
    return fetchAPI('/calc', {
        method: 'POST',
        body: JSON.stringify(field),
    });
}

export async function previewCalcField(req: PreviewRequest): Promise<PreviewResponse> {
    return fetchAPI('/calc/preview', {
        method: 'POST',
        body: JSON.stringify(req),
    });
}

export async function getCalcFields(objectId: string): Promise<CalcField[]> {
    const response = await fetchAPI<{ data: CalcField[] }>(`/calc?object_id=${encodeURIComponent(objectId)}`);
    return response.data || [];
}

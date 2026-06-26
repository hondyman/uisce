import { fetchAPI } from '../api';

export interface SemanticModelCalculation {
    id: string;
    semantic_model_id: string;
    calculation_id: string;
    argument_mapping: Record<string, string>;
    output_name: string;
    is_public: boolean;
    created_at: string;
    updated_at: string;
    calculation_name?: string;
}

export interface AddCalculationRequest {
    calculation_id: string;
    argument_mapping: Record<string, string>;
    output_name: string;
    is_public: boolean;
}

export async function addCalculation(modelId: string, req: AddCalculationRequest): Promise<SemanticModelCalculation> {
    return fetchAPI<SemanticModelCalculation>(`/fabric/models/${modelId}/calculations`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
    });
}

export async function removeCalculation(modelId: string, calcId: string): Promise<void> {
    await fetchAPI<void>(`/fabric/models/${modelId}/calculations/${calcId}`, {
        method: 'DELETE',
    });
}

export async function getCalculations(modelId: string): Promise<SemanticModelCalculation[]> {
    return fetchAPI<SemanticModelCalculation[]>(`/fabric/models/${modelId}/calculations`);
}

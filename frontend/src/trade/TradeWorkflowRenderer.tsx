import React, { useState, useEffect } from 'react';
import { fetchAPI } from '../api';
import { DynamicForm } from './DynamicForm';
import { WorkflowDefinition, WorkflowStage, TradeInput, WorkflowResult } from './types';

interface TradeWorkflowRendererProps {
    tenantId: string;
    workflowName: string;
}

export const TradeWorkflowRenderer: React.FC<TradeWorkflowRendererProps> = ({ tenantId, workflowName }) => {
    const [workflowDef, setWorkflowDef] = useState<WorkflowDefinition | null>(null);
    const [currentStageIndex, setCurrentStageIndex] = useState(0);
    const [workflowResult, setWorkflowResult] = useState<WorkflowResult | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [workflowData, setWorkflowData] = useState<Record<string, any>>({});

    useEffect(() => {
        const loadWorkflow = async () => {
            try {
                setLoading(true);
                const wd = await fetchAPI<WorkflowDefinition>(`/trade/metadata/workflows?name=${workflowName}`, {
                    headers: { 'x-tenant-id': tenantId }
                });
                setWorkflowDef(wd);
            } catch (err: any) {
                setError(err.message);
            } finally {
                setLoading(false);
            }
        };

        if (tenantId && workflowName) {
            loadWorkflow();
        }
    }, [tenantId, workflowName]);

    const handleStageSubmit = async (data: Record<string, any>) => {
        const updatedData = { ...workflowData, ...data };
        setWorkflowData(updatedData);

        if (!workflowDef) return;

        const isLastStage = currentStageIndex === workflowDef.stages.length - 1;

        if (isLastStage) {
            // Start the trade workflow
            try {
                setLoading(true);
                const input: TradeInput = {
                    tenant_id: tenantId,
                    workflow_name: workflowName,
                    data: updatedData
                };
                const result = await fetchAPI<WorkflowResult>('/trade/start', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(input)
                });
                setWorkflowResult(result);
            } catch (err: any) {
                setError(err.message);
            } finally {
                setLoading(false);
            }
        } else {
            // Move to next stage
            setCurrentStageIndex(prev => prev + 1);
        }
    };

    if (loading) return <div>Loading workflow...</div>;
    if (error) return <div className="text-red-500">Error: {error}</div>;
    if (!workflowDef) return <div>Workflow not found</div>;
    if (workflowResult) return (
        <div className="p-4 border rounded bg-green-50">
            <h3 className="text-lg font-bold text-green-800">Trade Submitted!</h3>
            <p>Workflow ID: {workflowResult.workflow_id}</p>
            <p>Run ID: {workflowResult.run_id}</p>
        </div>
    );

    const currentStage = workflowDef.stages[currentStageIndex];

    return (
        <div className="max-w-2xl mx-auto mt-8">
            <div className="mb-6">
                <h2 className="text-2xl font-bold mb-2">{workflowDef.name}</h2>
                <div className="flex space-x-2">
                    {workflowDef.stages.map((stage, idx) => (
                        <div key={stage.id} className={`flex-1 h-2 rounded ${idx <= currentStageIndex ? 'bg-blue-500' : 'bg-gray-200'}`} />
                    ))}
                </div>
                <p className="mt-2 text-gray-600">Stage: {currentStage.name}</p>
            </div>

            <DynamicForm
                fields={currentStage.config.fields}
                onSubmit={handleStageSubmit}
                submitLabel={currentStageIndex === workflowDef.stages.length - 1 ? 'Submit Trade' : 'Next'}
            />
        </div>
    );
};

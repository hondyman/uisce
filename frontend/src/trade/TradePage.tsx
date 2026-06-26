import React from 'react';
import { TradeWorkflowRenderer } from './TradeWorkflowRenderer';

export const TradePage: React.FC = () => {
    // In a real app, tenantId might come from context or auth
    const tenantId = "default-tenant-id"; 
    const workflowName = "TradeExecution";

    return (
        <div className="container mx-auto p-4">
            <h1 className="text-3xl font-bold mb-8">New Trade</h1>
            <TradeWorkflowRenderer tenantId={tenantId} workflowName={workflowName} />
        </div>
    );
};

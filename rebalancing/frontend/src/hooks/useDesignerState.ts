import { useCallback, useState } from "react";
import { Node, Edge } from "reactflow";
import { BPStepData } from "../bp/StepInspector";

export interface DesignerState {
    bpDefId: string | null;
    bpKey: string;
    bpVersion: number;
    status: "draft" | "in_review" | "approved";
    nodes: Node<BPStepData>[];
    edges: Edge[];
    lastSavedAt: Date | null;
    isSaving: boolean;
    error: string | null;
}

export function useDesignerState(tenantId: string) {
    const [state, setState] = useState<DesignerState>({
        bpDefId: null,
        bpKey: "new_bp",
        bpVersion: 1,
        status: "draft",
        nodes: [],
        edges: [],
        lastSavedAt: null,
        isSaving: false,
        error: null,
    });

    // Load designer from backend
    const load = useCallback(
        async (bpDefId: string) => {
            try {
                const res = await fetch(`/api/bp-designer/${bpDefId}`, {
                    headers: { "X-Tenant-ID": tenantId }
                });
                if (!res.ok) throw new Error("Failed to load");
                const data = await res.json();

                setState((prev) => ({
                    ...prev,
                    bpDefId: data.bpDefId,
                    bpKey: data.bpKey,
                    bpVersion: data.bpVersion,
                    status: data.status,
                    nodes: stepsToNodes(data.steps),
                    edges: [], // In a real app, edges would be persisted or derived from seq/graph
                    lastSavedAt: new Date(data.lastSavedAt),
                }));
            } catch (e) {
                setState((prev) => ({ ...prev, error: String(e) }));
            }
        },
        [tenantId]
    );

    // Save designer to backend
    const save = useCallback(async () => {
        setState((prev) => ({ ...prev, isSaving: true, error: null }));
        try {
            const steps = nodesToSteps(state.nodes);
            const res = await fetch("/api/bp-designer/save", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    "X-Tenant-ID": tenantId
                },
                body: JSON.stringify({
                    tenantId,
                    bpDefId: state.bpDefId,
                    bpKey: state.bpKey,
                    bpVersion: state.bpVersion,
                    status: state.status,
                    steps,
                }),
            });
            if (!res.ok) throw new Error("Failed to save");
            const result = await res.json();
            setState((prev) => ({
                ...prev,
                bpDefId: result.bpDefId,
                lastSavedAt: new Date(), // approximate
                isSaving: false,
            }));
        } catch (e) {
            setState((prev) => ({ ...prev, isSaving: false, error: String(e) }));
        }
    }, [state.nodes, state.bpDefId, state.bpKey, state.bpVersion, state.status, tenantId]);

    // Update nodes in-memory
    const setNodes = useCallback((nodes: Node<BPStepData>[]) => {
        setState((prev) => ({ ...prev, nodes }));
    }, []);

    // Update edges in-memory
    const setEdges = useCallback((edges: Edge[]) => {
        setState((prev) => ({ ...prev, edges }));
    }, []);

    return { state, load, save, setNodes, setEdges };
}

// Simple layout helper
function nodesToSteps(nodes: Node<BPStepData>[]): any[] {
    // Sort by Y position to infer sequence if needed, or trust the array order
    const sorted = [...nodes].sort((a, b) => a.position.y - b.position.y);

    return sorted.map((node, idx) => ({
        seq: idx + 1,
        stepKey: node.data.stepKey,
        type: node.data.type,
        activityName: node.data.type, // simplified
        signalName: "",
        conditionExprType: node.data.conditionExprType,
        conditionExpr: node.data.conditionExpr,
        preValidationRuleIds: node.data.preValidationRuleIds ?? [],
        postValidationRuleIds: node.data.postValidationRuleIds ?? [],
        approvalChain: node.data.approvalChain,
        routingRules: node.data.routingRules,
        delayExprType: "starlark", // default or add to BPStepData
        delayExpr: node.data.delayExpr,
        slaExprType: "starlark",
        slaExpr: node.data.slaExpr,
    }));
}

function stepsToNodes(steps: any[]): Node<BPStepData>[] {
    return steps.map((step) => ({
        id: step.stepKey, // use stepKey as ID or generic ID
        type: "default",
        data: {
            id: step.stepKey,
            stepKey: step.stepKey,
            type: step.type,
            conditionExprType: step.conditionExprType,
            conditionExpr: step.conditionExpr,
            preValidationRuleIds: step.preValidationRuleIds,
            postValidationRuleIds: step.postValidationRuleIds,
            approvalChain: step.approvalChain,
            routingRules: step.routingRules,
            delayExpr: step.delayExpr,
            slaExpr: step.slaExpr,
        },
        position: { x: 250, y: step.seq * 150 },
        style: { background: '#fff', border: '1px solid #777', padding: 10, borderRadius: 5, width: 150 },
    }));
}

import { create } from 'zustand';
import {
    Connection,
    Edge,
    EdgeChange,
    Node,
    NodeChange,
    addEdge,
    OnNodesChange,
    OnEdgesChange,
    OnConnect,
    applyNodeChanges,
    applyEdgeChanges,
} from 'reactflow';
import { TraceResult, DebugStep } from '../api/uisceApi';

export type UisceNode = Node<{
    label: string;
    filterType?: string;
    config: Record<string, any>;
    traceStatus?: 'PASS' | 'FAIL';
    traceError?: string;
}>;

interface UisceState {
    nodes: UisceNode[];
    edges: Edge[];
    selectedNodeId: string | null;
    traceResult: TraceResult | null;
    isDebugging: boolean;

    onNodesChange: OnNodesChange;
    onEdgesChange: OnEdgesChange;
    onConnect: OnConnect;

    setNodes: (nodes: UisceNode[]) => void;
    setEdges: (edges: Edge[]) => void;
    addNode: (node: UisceNode) => void;
    selectNode: (nodeId: string | null) => void;
    updateNodeConfig: (nodeId: string, config: Record<string, any>) => void;

    // BO integration
    selectedBO: any | null; // Using any for now to avoid circular dependency with type defs
    setSelectedBO: (bo: any | null) => void;

    // Debug actions
    setIsDebugging: (isDebugging: boolean) => void;
    applyTraceResult: (result: TraceResult) => void;
    clearTraceResult: () => void;
}

const useUisceStore = create<UisceState>((set, get) => ({
    nodes: [],
    edges: [],
    selectedNodeId: null,
    traceResult: null,
    isDebugging: false,
    selectedBO: null,

    setSelectedBO: (bo) => set({ selectedBO: bo }),

    onNodesChange: (changes: NodeChange[]) => {
        set({
            nodes: applyNodeChanges(changes, get().nodes) as UisceNode[],
        });
    },

    onEdgesChange: (changes: EdgeChange[]) => {
        set({
            edges: applyEdgeChanges(changes, get().edges),
        });
    },

    onConnect: (connection: Connection) => {
        set({
            edges: addEdge(connection, get().edges),
        });
    },

    setNodes: (nodes) => set({ nodes }),
    setEdges: (edges) => set({ edges }),

    addNode: (node) => set({ nodes: [...get().nodes, node] }),

    selectNode: (nodeId) => set({ selectedNodeId: nodeId }),

    updateNodeConfig: (nodeId, config) => {
        set({
            nodes: get().nodes.map((node) => {
                if (node.id === nodeId) {
                    return {
                        ...node,
                        data: { ...node.data, config: { ...node.data.config, ...config } },
                    };
                }
                return node;
            }),
        });
    },

    setIsDebugging: (isDebugging) => set({ isDebugging }),

    applyTraceResult: (result: TraceResult) => {
        // Map trace steps to nodes by filter name
        const stepMap = new Map<string, DebugStep>();
        result.steps.forEach(step => {
            stepMap.set(step.filterName, step);
        });

        set({
            traceResult: result,
            nodes: get().nodes.map((node) => {
                const step = stepMap.get(node.data.label);
                if (step) {
                    return {
                        ...node,
                        data: {
                            ...node.data,
                            traceStatus: step.status,
                            traceError: step.errorDetails,
                        },
                    };
                }
                return node;
            }),
        });
    },

    clearTraceResult: () => {
        set({
            traceResult: null,
            nodes: get().nodes.map((node) => ({
                ...node,
                data: {
                    ...node.data,
                    traceStatus: undefined,
                    traceError: undefined,
                },
            })),
        });
    },
}));

export default useUisceStore;


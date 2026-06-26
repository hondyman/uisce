// src/runtime/actions.ts
import { ActionDefinition } from "../types/pageStudio";

export interface ActionContext {
    pageId: string;
    tenantId: string;
    state: Record<string, any>;
    setState: (updater: (prev: any) => any) => void;
    openModal: (id: string, props?: any) => void;
    closeModal: (id: string) => void;
    refreshComponent: (id: string) => void;
    executeMutation: (sourceId: string, vars: any) => Promise<any>;
    navigate: (pageId: string, params?: Record<string, any>) => void;
}

export async function runActions(
    actions: ActionDefinition[],
    ctx: ActionContext,
    eventPayload: any
) {
    for (const action of actions) {
        const params = resolveTemplates(action.params || {}, eventPayload, ctx.state);

        switch (action.type) {
            case "navigate":
                if (action.targetPageId) {
                    ctx.navigate(action.targetPageId, params);
                }
                break;
            case "mutate":
                if (action.mutationSourceId) {
                    await ctx.executeMutation(action.mutationSourceId, params);
                }
                break;
            case "refresh":
                if (action.targetComponentId) {
                    ctx.refreshComponent(action.targetComponentId);
                }
                break;
            case "openModal":
                if (action.targetComponentId) {
                    ctx.openModal(action.targetComponentId, params);
                }
                break;
            case "closeModal":
                if (action.targetComponentId) {
                    ctx.closeModal(action.targetComponentId);
                }
                break;
            case "setState":
                if (action.stateKey) {
                    ctx.setState((prev) => ({
                        ...prev,
                        [action.stateKey!]: resolveTemplates(action.stateValue, eventPayload, prev),
                    }));
                }
                break;
        }
    }
}

export function resolveTemplates(obj: any, payload: any, state: any): any {
    if (typeof obj === "string") {
        return obj.replace(/\{\{([^}]+)\}\}/g, (_, expr) => {
            const parts = expr.trim().split(".");
            const root = parts[0];
            const path = parts.slice(1);

            let value: any;
            if (root === "row") {
                value = payload?.row;
            } else if (root === "state") {
                value = state;
            } else if (root === "payload") {
                value = payload;
            }

            if (value === undefined) return "";

            return path.reduce((v: any, k: string) => (v && v[k] !== undefined ? v[k] : ""), value);
        });
    }
    if (Array.isArray(obj)) return obj.map((v) => resolveTemplates(v, payload, state));
    if (obj && typeof obj === "object") {
        const out: any = {};
        for (const [k, v] of Object.entries(obj)) {
            out[k] = resolveTemplates(v, payload, state);
        }
        return out;
    }
    return obj;
}

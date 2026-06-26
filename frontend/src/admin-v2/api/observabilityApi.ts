import { api } from "../api";

export interface ETLRun {
    id: string;
    tenant_id: string;
    valuation_date: string;
    status: string; // "STARTED", "SUCCESS", "FAILED"
    start_time: string;
    end_time?: string;
    duration_ms?: number;
    records_processed: number;
    rules_evaluated: number;
    scenarios_evaluated: number;
    wasm_orchestrator_version?: string;
    error_message?: string;
    created_at: string;
    updated_at: string;
}

export interface WasmModuleVersion {
    id: string;
    module_name: string;
    version_tag: string;
    wasm_hash: string;
    size_bytes: number;
    uploaded_by: string;
    is_active: boolean;
    metadata?: any;
    created_at: string;
    updated_at: string;
}

export interface RuleLineage {
    id: string;
    tenant_id: string;
    rule_id: string;
    portfolio_id: string;
    security_id?: string;
    valuation_date: string;
    etl_run_id: string;
    wasm_version_id: string;
    status: string;
    metric_value?: number;
    threshold_value?: number;
    duration_ms: number;
    semantic_terms_used: string[];
    created_at: string;
}

export interface ScenarioLineage {
    id: string;
    tenant_id: string;
    scenario_id: string;
    portfolio_id: string;
    valuation_date: string;
    etl_run_id: string;
    wasm_version_id: string;
    total_base_value: number;
    total_stressed_value: number;
    pnl_amount: number;
    pnl_percent: number;
    duration_ms: number;
    semantic_terms_used: string[];
    created_at: string;
}

export const observabilityApi = {
    // ETL Runs
    listETLRuns: (params?: { tenant_id?: string; status?: string; from?: string; to?: string; limit?: number; offset?: number }) => {
        const query = new URLSearchParams();
        if (params?.tenant_id) query.append("tenant_id", params.tenant_id);
        if (params?.status) query.append("status", params.status);
        if (params?.from) query.append("from", params.from);
        if (params?.to) query.append("to", params.to);
        if (params?.limit) query.append("limit", params.limit.toString());
        if (params?.offset) query.append("offset", params.offset.toString());

        const qs = query.toString();
        return api<{ runs: ETLRun[] }>(`/v1/telemetry/etl-runs${qs ? `?${qs}` : ""}`);
    },

    getETLRun: (id: string) => {
        return api<ETLRun>(`/v1/telemetry/etl-runs/${id}`);
    },

    // Wasm Versions
    listWasmVersions: (moduleName?: string) => {
        const qs = moduleName ? `?module_name=${encodeURIComponent(moduleName)}` : "";
        return api<{ versions: WasmModuleVersion[] }>(`/v1/telemetry/wasm-versions${qs}`);
    },

    activateWasmVersion: (id: string) => {
        return api<{ status: string }>(`/v1/telemetry/wasm-versions/${id}/activate`, {
            method: "POST"
        });
    },

    // Lineage Explorers
    getRuleLineage: (ruleId: string, params?: { from?: string; to?: string; tenant_id?: string; status?: string }) => {
        const query = new URLSearchParams();
        if (params?.from) query.append("from", params.from);
        if (params?.to) query.append("to", params.to);
        if (params?.tenant_id) query.append("tenant_id", params.tenant_id);
        if (params?.status) query.append("status", params.status);

        const qs = query.toString();
        return api<{ lineage: RuleLineage[] }>(`/v1/telemetry/rules/${ruleId}/lineage${qs ? `?${qs}` : ""}`);
    },

    getScenarioLineage: (scenarioId: string, params?: { from?: string; to?: string; tenant_id?: string }) => {
        const query = new URLSearchParams();
        if (params?.from) query.append("from", params.from);
        if (params?.to) query.append("to", params.to);
        if (params?.tenant_id) query.append("tenant_id", params.tenant_id);

        const qs = query.toString();
        return api<{ lineage: ScenarioLineage[] }>(`/v1/telemetry/scenarios/${scenarioId}/lineage${qs ? `?${qs}` : ""}`);
    }
};

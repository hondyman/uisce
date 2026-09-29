"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ValidationClient = void 0;
class ValidationClient {
    baseUrl;
    tenantId;
    authToken;
    timeoutMs;
    constructor(config) {
        this.baseUrl = (config.baseUrl || "http://localhost:8080").replace(/\/$/, "");
        this.tenantId = config.tenantId;
        this.authToken = config.authToken;
        this.timeoutMs = config.timeoutMs || 30000;
    }
    async request(method, path, body, params) {
        let url = `${this.baseUrl}${path}`;
        if (params) {
            const query = Object.entries(params)
                .filter(([_, v]) => v !== undefined && v !== null)
                .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`)
                .join("&");
            if (query) {
                url += `?${query}`;
            }
        }
        const headers = {
            "Content-Type": "application/json",
            "Accept": "application/json",
        };
        if (this.tenantId) {
            headers["X-Tenant-ID"] = this.tenantId;
        }
        if (this.authToken) {
            headers["Authorization"] = `Bearer ${this.authToken}`;
        }
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), this.timeoutMs);
        try {
            const resp = await fetch(url, {
                method,
                headers,
                body: body ? JSON.stringify(body) : undefined,
                signal: controller.signal,
            });
            const respText = await resp.text();
            let data;
            try {
                data = respText ? JSON.parse(respText) : {};
            }
            catch {
                data = { raw: respText };
            }
            if (!resp.ok && resp.status !== 422) {
                throw new Error(`HTTP ${resp.status}: ${respText}`);
            }
            return data;
        }
        finally {
            clearTimeout(timeoutId);
        }
    }
    /**
     * Evaluates validation rules against a single business object record.
     */
    async evaluateRecord(boName, record, options = {}) {
        const payload = {
            tenant_id: this.tenantId,
            bo_name: boName,
            record,
            record_id: options.recordId,
            domain: options.domain,
            timing: options.timing,
        };
        return this.request("POST", "/api/validation-rule-nodes/evaluate-record", payload);
    }
    /**
     * Evaluates a batch of records under an isolated transactional RuleSnapshot.
     */
    async evaluateBatch(boName, records, options = {}) {
        const payload = {
            tenant_id: this.tenantId,
            bo_name: boName,
            records,
            domain: options.domain,
            timing: options.timing,
            as_of: options.asOf,
        };
        return this.request("POST", "/api/validation-rule-nodes/evaluate-batch", payload);
    }
    /**
     * Verifies the cryptographic Glassbox audit chain between sequence numbers.
     */
    async verifyChain(seqFrom = 1, seqTo = 10000) {
        return this.request("GET", "/api/validation-rule-nodes/verify-chain", undefined, {
            from: seqFrom,
            to: seqTo,
        });
    }
}
exports.ValidationClient = ValidationClient;

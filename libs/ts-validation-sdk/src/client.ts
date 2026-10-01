import {
  ClientConfig,
  RecordEvaluationResult,
  BatchEvaluationReport,
  ChainVerificationResult,
} from "./types";

export class ValidationClient {
  private baseUrl: string;
  private tenantId: string;
  private authToken?: string;
  private timeoutMs: number;

  constructor(config: ClientConfig) {
    this.baseUrl = (config.baseUrl || "http://localhost:8080").replace(/\/$/, "");
    this.tenantId = config.tenantId;
    this.authToken = config.authToken;
    this.timeoutMs = config.timeoutMs || 30000;
  }

  private async request<T>(method: string, path: string, body?: unknown, params?: Record<string, string | number>): Promise<T> {
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

    const headers: Record<string, string> = {
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
      let data: unknown;
      try {
        data = respText ? JSON.parse(respText) : {};
      } catch {
        data = { raw: respText };
      }

      if (!resp.ok && resp.status !== 422) {
        throw new Error(`HTTP ${resp.status}: ${respText}`);
      }

      return data as T;
    } finally {
      clearTimeout(timeoutId);
    }
  }

  /**
   * Evaluates validation rules against a single business object record.
   */
  public async evaluateRecord(
    boName: string,
    record: Record<string, unknown>,
    options: { recordId?: string; domain?: string; timing?: string } = {}
  ): Promise<RecordEvaluationResult> {
    const payload = {
      tenant_id: this.tenantId,
      bo_name: boName,
      record,
      record_id: options.recordId,
      domain: options.domain,
      timing: options.timing,
    };
    return this.request<RecordEvaluationResult>("POST", "/api/validation-rule-nodes/evaluate-record", payload);
  }

  /**
   * Evaluates a batch of records under an isolated transactional RuleSnapshot.
   */
  public async evaluateBatch(
    boName: string,
    records: Record<string, unknown>[],
    options: { domain?: string; timing?: string; asOf?: string } = {}
  ): Promise<BatchEvaluationReport> {
    const payload = {
      tenant_id: this.tenantId,
      bo_name: boName,
      records,
      domain: options.domain,
      timing: options.timing,
      as_of: options.asOf,
    };
    return this.request<BatchEvaluationReport>("POST", "/api/validation-rule-nodes/evaluate-batch", payload);
  }

  /**
   * Verifies the cryptographic Glassbox audit chain between sequence numbers.
   */
  public async verifyChain(seqFrom = 1, seqTo = 10000): Promise<ChainVerificationResult> {
    return this.request<ChainVerificationResult>("GET", "/api/validation-rule-nodes/verify-chain", undefined, {
      from: seqFrom,
      to: seqTo,
    });
  }
}

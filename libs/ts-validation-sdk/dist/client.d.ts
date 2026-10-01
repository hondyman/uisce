import { ClientConfig, RecordEvaluationResult, BatchEvaluationReport, ChainVerificationResult } from "./types";
export declare class ValidationClient {
    private baseUrl;
    private tenantId;
    private authToken?;
    private timeoutMs;
    constructor(config: ClientConfig);
    private request;
    /**
     * Evaluates validation rules against a single business object record.
     */
    evaluateRecord(boName: string, record: Record<string, unknown>, options?: {
        recordId?: string;
        domain?: string;
        timing?: string;
    }): Promise<RecordEvaluationResult>;
    /**
     * Evaluates a batch of records under an isolated transactional RuleSnapshot.
     */
    evaluateBatch(boName: string, records: Record<string, unknown>[], options?: {
        domain?: string;
        timing?: string;
        asOf?: string;
    }): Promise<BatchEvaluationReport>;
    /**
     * Verifies the cryptographic Glassbox audit chain between sequence numbers.
     */
    verifyChain(seqFrom?: number, seqTo?: number): Promise<ChainVerificationResult>;
}

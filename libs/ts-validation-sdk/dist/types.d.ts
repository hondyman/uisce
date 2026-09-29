export interface ViolationRecord {
    rule_id: string;
    rule_key: string;
    rule_name: string;
    severity: "BLOCK" | "WARN";
    message: string;
    rule_version: string;
    bo_key?: string;
    record_id?: string;
    fields?: string[];
    context?: Record<string, unknown>;
    write_blocked?: boolean;
    rule_error?: boolean;
}
export interface RuleErrorEntry {
    rule_id: string;
    rule_key: string;
    rule_name: string;
    error: string;
    rule_version: string;
}
export interface RecordEvaluationResult {
    passed: boolean;
    violations: ViolationRecord[];
    rule_errors: RuleErrorEntry[];
    rules_evaluated: number;
    snapshot_id: string;
    context_required?: boolean;
    missing_fields?: string[];
}
export interface BatchRecordResult {
    index: number;
    record_id: string;
    passed: boolean;
    violations: ViolationRecord[];
    rule_errors: RuleErrorEntry[];
    context_required?: boolean;
    missing_fields?: string[];
}
export interface BatchEvaluationReport {
    total_records: number;
    passed_records: number;
    failed_records: number;
    blocked_records: number;
    snapshot_id: string;
    results: BatchRecordResult[];
}
export interface ChainVerificationResult {
    valid: boolean;
    tenant_id: string;
    seq_from: number;
    seq_to: number;
    total_rows: number;
    anchors_checked: number;
    errors?: string[];
}
export interface ClientConfig {
    baseUrl?: string;
    tenantId: string;
    authToken?: string;
    timeoutMs?: number;
}

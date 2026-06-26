import axios from '@/utils/axiosClient';

const API_BASE = '/api/metadata';

export interface EvidenceBundle {
    id: string;
    upgrade_request_id: string;
    old_version: string;
    new_version: string;
    status: 'in_progress' | 'completed' | 'failed' | 'rolled_back';
    stages: StageEvidence[];
    created_at: string;
    completed_at?: string;
}

export interface StageEvidence {
    id: string;
    bundle_id: string;
    stage_name: 'diff' | 'rebase' | 'test' | 'approval' | 'deploy' | 'rollback' | 'audit';
    status: 'pending' | 'running' | 'success' | 'failed' | 'skipped';
    artifacts: Artifact[];
    started_at: string;
    completed_at?: string;
    actor_id?: string;
}

export interface Artifact {
    type: string;
    storage_path: string;
    checksum: string;
    metadata?: Record<string, any>;
    created_at: string;
    size_bytes?: number;
}

export interface ComplianceReport {
    bundle_id: string;
    generated_at: string;
    executive_summary: ExecutiveSummary;
    change_inventory: ChangeRecord[];
    test_summary: TestSummary;
    approval_chain: ApprovalDecision[];
    deployment_log: DeploymentSummary;
    artifacts: Artifact[];
}

export interface ExecutiveSummary {
    status: string;
    risk_level: 'LOW' | 'MEDIUM' | 'HIGH';
    breaking_changes: number;
    additive_changes: number;
    test_pass_rate: number;
    deployment_success: boolean;
    rollbacks_required: number;
}

export interface ChangeRecord {
    path: string;
    type: string;
    severity: 'ADDITIVE' | 'BREAKING' | 'SAFE';
    old_value?: any;
    new_value?: any;
    impact: string;
}

export interface TestSummary {
    total_tests: number;
    passed_tests: number;
    failed_tests: number;
    skipped_tests: number;
    coverage: number;
    execution_time_ms: number;
    failed_test_details?: FailedTest[];
}

export interface FailedTest {
    test_name: string;
    error_message: string;
    related_diff?: string;
}

export interface DeploymentSummary {
    started_at: string;
    completed_at?: string;
    target_tenants: string[];
    successful_deploys: number;
    failed_deploys: number;
    rollback_events?: RollbackEvent[];
}

export interface RollbackEvent {
    tenant_id: string;
    reason: string;
    timestamp: string;
    actor_id: string;
}

export interface ApprovalRequest {
    id: string;
    bundle_id: string;
    requested_by: string;
    requested_at: string;
    required_role: string;
    status: 'pending' | 'approved' | 'rejected' | 'expired';
    approver_id?: string;
    decision?: 'approved' | 'rejected';
    justification?: string;
    decided_at?: string;
}

export interface ApprovalDecision {
    request_id: string;
    approver_id: string;
    decision: 'approved' | 'rejected';
    justification: string;
    decided_at: string;
}

export class EvidenceBundleAPI {
    static async getBundle(bundleId: string): Promise<EvidenceBundle> {
        const response = await axios.get(`${API_BASE}/evidence/bundles/${bundleId}`);
        return response.data;
    }

    static async getComplianceReport(bundleId: string, format: 'json' | 'download' = 'json'): Promise<ComplianceReport> {
        const response = await axios.get(
            `${API_BASE}/evidence/bundles/${bundleId}/compliance-report`,
            { params: { format } }
        );
        return response.data;
    }

    static async downloadComplianceReport(bundleId: string): Promise<void> {
        const response = await axios.get(
            `${API_BASE}/evidence/bundles/${bundleId}/compliance-report`,
            {
                params: { format: 'download' },
                responseType: 'blob'
            }
        );

        const url = window.URL.createObjectURL(new Blob([response.data]));
        const link = document.createElement('a');
        link.href = url;
        link.setAttribute('download', `compliance-report-${bundleId}.json`);
        document.body.appendChild(link);
        link.click();
        link.remove();
    }

    static async getStages(bundleId: string): Promise<StageEvidence[]> {
        const response = await axios.get(`${API_BASE}/evidence/bundles/${bundleId}/stages`);
        return response.data;
    }

    static async getPendingApprovals(role: string): Promise<ApprovalRequest[]> {
        const response = await axios.get(`${API_BASE}/approvals/pending`, {
            params: { role }
        });
        return response.data;
    }

    static async approveUpgrade(requestId: string, approverId: string, justification: string): Promise<void> {
        await axios.post(`${API_BASE}/approvals/${requestId}/approve`, {
            approver_id: approverId,
            justification
        });
    }

    static async rejectUpgrade(requestId: string, approverId: string, justification: string): Promise<void> {
        await axios.post(`${API_BASE}/approvals/${requestId}/reject`, {
            approver_id: approverId,
            justification
        });
    }

    static async getApprovalChain(bundleId: string): Promise<ApprovalDecision[]> {
        const response = await axios.get(`${API_BASE}/evidence/bundles/${bundleId}/approvals`);
        return response.data;
    }

    static async triggerUpgrade(params: {
        old_core_version: string;
        new_core_version: string;
        target_tenants: string[];
        requested_by: string;
    }): Promise<{ message: string; upgrade_id: string }> {
        const response = await axios.post(`${API_BASE}/upgrades`, params);
        return response.data;
    }
}
